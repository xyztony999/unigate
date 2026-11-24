package service

import (
	"context"
	"fmt"
	"time"

	"unigate/backend/internal/errcode"
	"unigate/backend/internal/store"
)

// This file reproduces com.unigate.IntegrationService: platform connections,
// event sources, import batches, jobs, controlled exports and lifecycle
// cleanup. Integration boundaries preserve the event transaction and never
// grant eligibility from remote fields.

// UploadFile is the multipart upload handed over by the api layer
// (MultipartFile in Java). Data holds the raw bytes, Filename the original
// client filename used to pick the parser.
type UploadFile struct {
	Data     []byte
	Filename string
}

// Binary is a rendered artifact returned to the api layer for streaming
// (IntegrationService.Binary).
type Binary struct {
	Data        []byte
	ContentType string
	Filename    string
}

// handleIntegration dispatches integration operations (IntegrationService.handle).
// entry is the idempotent operation entry created by the dispatcher; mutation
// results echo r["operationId"] and replayed=false like the other services.
func (svc *Service) handleIntegration(ctx context.Context, sc *store.Scope, op string, r map[string]any, c *Req, entry string) (any, error) {
	switch op {
	case "ConnectionList":
		return svc.connectionList(ctx, sc, r, c)
	case "ConnectionSave":
		return svc.connectionSave(ctx, sc, r, c)
	case "ConnectionRevoke":
		return svc.connectionRevoke(ctx, sc, r, c)
	case "ConnectionAuthorize":
		return svc.connectionAuthorize(ctx, sc, r, c)
	case "ConnectionCallback":
		return svc.connectionCallback(ctx, sc, r, c)
	case "ResourceList", "ResourceFields":
		if err := svc.connectionAccess(ctx, sc, r, c); err != nil {
			return nil, err
		}
		connection, err := svc.S.ConnectionByID(ctx, sc, store.StrOrEmpty(r["connectionId"]))
		if err != nil {
			return nil, err
		}
		if op == "ResourceList" {
			items, err := svc.providerList(ctx, connection)
			if err != nil {
				return nil, err
			}
			return store.Map("items", items), nil
		}
		columns, err := svc.providerFields(ctx, connection, store.StrOrEmpty(r["resourceId"]))
		if err != nil {
			return nil, err
		}
		return store.Map("resourceId", r["resourceId"], "availableColumns", columns), nil
	case "SourceSave":
		return svc.sourceSave(ctx, sc, r, c)
	case "SourceList":
		return svc.sourceList(ctx, sc, r, c)
	case "SourcePull":
		return svc.sourcePull(ctx, sc, r, c)
	case "SmsStart", "SmsVerify", "RecoveryStart", "RecoveryVerify":
		return svc.handleNotification(ctx, sc, op, r, c)
	case "BatchUpload":
		return svc.batchUpload(ctx, sc, r, c)
	case "BatchDetail":
		if _, err := svc.intEvent(ctx, sc, r, c); err != nil {
			return nil, err
		}
		b, err := svc.batchLocked(ctx, sc, r)
		if err != nil {
			return nil, err
		}
		return svc.batchView(ctx, sc, b)
	case "BatchPreview":
		return svc.batchPreview(ctx, sc, r, c)
	case "BatchRows":
		return svc.batchRowsView(ctx, sc, r, c)
	case "DuplicateResolve":
		return svc.duplicateResolve(ctx, sc, r, c)
	case "BatchCommit":
		return svc.batchCommit(ctx, sc, r, c, entry)
	case "JobList":
		return svc.jobList(ctx, sc, r, c)
	case "JobDetail":
		j, err := svc.jobLocked(ctx, sc, r, c)
		if err != nil {
			return nil, err
		}
		return svc.jobView(j), nil
	case "JobRetry":
		return svc.jobRetry(ctx, sc, r, c)
	case "EmailResend":
		return svc.emailResend(ctx, sc, r, c)
	case "ExportCreate":
		return svc.exportCreate(ctx, sc, r, c)
	case "ExportDetail":
		if _, err := svc.intEvent(ctx, sc, r, c); err != nil {
			return nil, err
		}
		x, err := svc.exportRow(ctx, sc, r, c)
		if err != nil {
			return nil, err
		}
		return svc.exportView(x), nil
	case "ExportDownload":
		return svc.exportDownload(ctx, sc, r, c)
	case "DeliveryList":
		return svc.deliveryList(ctx, sc, r, c)
	case "DeliveryRecord":
		return svc.deliveryRecord(ctx, sc, r, c)
	case "CleanupDetail":
		return svc.cleanupDetail(ctx, sc, r, c)
	case "CleanupRetry":
		return svc.cleanupRetry(ctx, sc, r, c)
	default:
		return nil, errcode.New(404, "RESOURCE_NOT_FOUND", "接口不存在")
	}
}

// intAdmin requires the system administrator (IntegrationService.admin).
func (svc *Service) intAdmin(c *Req) error {
	return errcode.Require(c.Admin(), 403, "FORBIDDEN", "需要管理员权限")
}

// intEvent requires the organizer role and loads the event under lock with
// the personal-data expiry check (IntegrationService.event).
func (svc *Service) intEvent(ctx context.Context, sc *store.Scope, r map[string]any, c *Req) (map[string]any, error) {
	id := store.StrOrEmpty(r["eventId"])
	if err := c.Organizer(ctx, id); err != nil {
		return nil, err
	}
	return svc.S.Event(ctx, sc, id)
}

// intakeOpen freezes roster intake once check-in starts.
func (svc *Service) intakeOpen(e map[string]any) error {
	status := store.StrOrEmpty(e["status"])
	return errcode.Require(status == "DRAFT" || status == "READY",
		409, "EVENT_STATE_CONFLICT", "检票开始后名单导入已冻结")
}

// connectionAccess lets admins address any connection; organizers only ones
// assigned to their event.
func (svc *Service) connectionAccess(ctx context.Context, sc *store.Scope, r map[string]any, c *Req) error {
	if c.Admin() {
		_, err := svc.S.ConnectionByID(ctx, sc, store.StrOrEmpty(r["connectionId"]))
		return err
	}
	if err := c.Organizer(ctx, store.StrOrEmpty(r["eventId"])); err != nil {
		return err
	}
	_, err := svc.S.ConnectionAllowed(ctx, sc, store.StrOrEmpty(r["connectionId"]), store.StrOrEmpty(r["eventId"]))
	return err
}

func (svc *Service) connectionList(ctx context.Context, sc *store.Scope, r map[string]any, c *Req) (any, error) {
	if err := svc.intAdmin(c); err != nil {
		return nil, err
	}
	all, err := svc.S.ListConnections(ctx, sc)
	if err != nil {
		return nil, err
	}
	out := []any{}
	for _, x := range all {
		if _, has := r["provider"]; has && store.StrOrEmpty(r["provider"]) != store.StrOrEmpty(x["provider"]) {
			continue
		}
		v, err := svc.connectionView(ctx, sc, x)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return store.Page(out, r), nil
}

func (svc *Service) connectionView(ctx context.Context, sc *store.Scope, x map[string]any) (map[string]any, error) {
	events, err := svc.S.ConnectionEvents(ctx, sc, store.StrOrEmpty(x["id"]))
	if err != nil {
		return nil, err
	}
	ids := []any{}
	for _, y := range events {
		ids = append(ids, y["event_id"])
	}
	out := store.Map(
		"connectionId", x["id"],
		"provider", svc.value("Provider", x["provider"]),
		"accountKind", svc.value("AccountKind", x["account_kind"]),
		"displayName", x["display_name"],
		"status", svc.value("ConnectionStatus", x["status"]),
		"version", x["version"],
		"expiresAt", x["expires_at"],
		"allowedEventIds", ids,
	)
	if x["credentials_cipher"] != nil {
		out["credentialMask"] = "••••••••"
	}
	return out, nil
}

func (svc *Service) connectionSave(ctx context.Context, sc *store.Scope, r map[string]any, c *Req) (any, error) {
	if err := svc.intAdmin(c); err != nil {
		return nil, err
	}
	id := store.StrOrEmpty(r["connectionId"])
	secret := store.Map("clientSecret", r["clientSecret"])
	var secretCipher []byte
	if len(secret) > 0 {
		sealed, err := svc.S.Seal(sc, secret)
		if err != nil {
			return nil, err
		}
		secretCipher = sealed
	}
	if id == "" {
		id = svc.S.ID()
		if err := svc.S.InsertConnection(ctx, sc, id, store.StrOrEmpty(r["provider"]), store.StrOrEmpty(r["accountKind"]),
			store.StrOrEmpty(r["displayName"]), r["clientId"], secretCipher, store.StrOrEmpty(c.ActorID())); err != nil {
			return nil, err
		}
	} else {
		old, err := svc.S.ConnectionByIDForUpdate(ctx, sc, id)
		if err != nil {
			return nil, err
		}
		if err := store.Version(old, r["version"]); err != nil {
			return nil, err
		}
		if err := errcode.Require(store.StrOrEmpty(r["provider"]) == store.StrOrEmpty(old["provider"]),
			409, "PROVIDER_CONFLICT", "既有连接不能更换提供方"); err != nil {
			return nil, err
		}
		if err := svc.S.UpdateConnection(ctx, sc, store.StrOrEmpty(r["accountKind"]), store.StrOrEmpty(r["displayName"]),
			r["clientId"], secretCipher, id); err != nil {
			return nil, err
		}
	}
	requested := map[string]bool{}
	for _, e := range store.List(r["allowedEventIds"]) {
		requested[store.StrOrEmpty(e)] = true
	}
	existing, err := svc.S.ConnectionEvents(ctx, sc, id)
	if err != nil {
		return nil, err
	}
	for _, row := range existing {
		ev := store.StrOrEmpty(row["event_id"])
		if requested[ev] {
			continue
		}
		refs, err := svc.S.SourcesByConnectionEvent(ctx, sc, id, ev)
		if err != nil {
			return nil, err
		}
		if err := errcode.Require(len(refs) == 0, 409, "SOURCE_IN_USE", "请先解除活动来源对该连接的引用"); err != nil {
			return nil, err
		}
		if err := svc.S.DeleteConnectionEvent(ctx, sc, id, ev); err != nil {
			return nil, err
		}
	}
	for ev := range requested {
		if _, err := svc.S.EventExists(ctx, sc, ev); err != nil {
			return nil, err
		}
		if err := svc.S.InsertConnectionEvent(ctx, sc, id, ev); err != nil {
			return nil, err
		}
	}
	row, err := svc.S.ConnectionByID(ctx, sc, id)
	if err != nil {
		return nil, err
	}
	return svc.connectionView(ctx, sc, row)
}

func (svc *Service) connectionRevoke(ctx context.Context, sc *store.Scope, r map[string]any, c *Req) (any, error) {
	if err := svc.intAdmin(c); err != nil {
		return nil, err
	}
	x, err := svc.S.ConnectionByIDForUpdate(ctx, sc, store.StrOrEmpty(r["connectionId"]))
	if err != nil {
		return nil, err
	}
	if err := store.Version(x, r["version"]); err != nil {
		return nil, err
	}
	if err := svc.S.RevokeConnection(ctx, sc, store.StrOrEmpty(x["id"])); err != nil {
		return nil, err
	}
	row, err := svc.S.ConnectionByID(ctx, sc, store.StrOrEmpty(x["id"]))
	if err != nil {
		return nil, err
	}
	return svc.connectionView(ctx, sc, row)
}

func (svc *Service) connectionAuthorize(ctx context.Context, sc *store.Scope, r map[string]any, c *Req) (any, error) {
	if err := svc.intAdmin(c); err != nil {
		return nil, err
	}
	connection, err := svc.S.ConnectionByIDForUpdate(ctx, sc, store.StrOrEmpty(r["connectionId"]))
	if err != nil {
		return nil, err
	}
	if err := store.Version(connection, r["version"]); err != nil {
		return nil, err
	}
	state := svc.S.Crypto.Token()
	url, err := svc.providerAuthorizeURL(connection, state)
	if err != nil {
		return nil, err
	}
	expires := svc.S.Now().Add(5 * time.Minute)
	var sessionID any
	if c.Session != nil {
		sessionID = c.Session["id"]
	}
	contextCipher, err := svc.S.Seal(sc, store.Map("sessionId", sessionID, "connectionVersion", connection["version"]))
	if err != nil {
		return nil, err
	}
	if err := svc.S.InsertOAuthChallenge(ctx, sc, svc.S.ID(), store.StrOrEmpty(connection["id"]),
		store.StrOrEmpty(c.ActorID()), svc.S.Crypto.Hash(state), contextCipher, expires); err != nil {
		return nil, err
	}
	return store.Map("authorizationUrl", url, "expiresAt", expires), nil
}

func (svc *Service) connectionCallback(ctx context.Context, sc *store.Scope, r map[string]any, c *Req) (any, error) {
	if err := svc.intAdmin(c); err != nil {
		return nil, err
	}
	// The state challenge is consumed in its own committed transaction
	// (stateTx REQUIRES_NEW) so a failed exchange cannot replay the state.
	var challenge map[string]any
	var challengeContext map[string]any
	if err := svc.independentTx(ctx, func(inner *store.Scope) error {
		found, err := svc.S.OAuthChallengeByDigest(ctx, inner, svc.S.Crypto.Hash(store.StrOrEmpty(r["state"])))
		if err != nil {
			return err
		}
		context, err := svc.S.Unseal(found["context_cipher"])
		if err != nil {
			return err
		}
		expires, err := store.Time(found["expires_at"])
		if err != nil {
			return fmt.Errorf("oauth challenge expiry: %w", err)
		}
		sessionOK := c.Session != nil && store.StrOrEmpty(c.Session["id"]) == store.StrOrEmpty(context["sessionId"])
		if err := errcode.Require(found["consumed_at"] == nil && expires.After(svc.S.Now()) &&
			store.StrOrEmpty(c.ActorID()) == store.StrOrEmpty(found["actor_scope"]) && sessionOK,
			403, "OAUTH_STATE_INVALID", "授权状态无效，请重新发起授权"); err != nil {
			return err
		}
		if err := svc.S.ConsumeChallenge(ctx, inner, svc.S.Now(), store.StrOrEmpty(found["id"])); err != nil {
			return err
		}
		challenge = found
		challengeContext = context
		return nil
	}); err != nil {
		return nil, err
	}
	connection, err := svc.S.ConnectionByIDForUpdate(ctx, sc, store.StrOrEmpty(challenge["connection_id"]))
	if err != nil {
		return nil, err
	}
	if err := store.Version(connection, challengeContext["connectionVersion"]); err != nil {
		return nil, err
	}
	exchanged, err := svc.providerExchange(ctx, connection, store.StrOrEmpty(r["code"]))
	if err != nil {
		return nil, err
	}
	credentials, err := svc.S.Unseal(connection["credentials_cipher"])
	if err != nil {
		return nil, err
	}
	for k, v := range exchanged {
		credentials[k] = v
	}
	expiry := svc.S.Now().Add(time.Duration(store.Num(exchanged["expiresIn"])) * time.Second)
	delete(credentials, "expiresIn")
	cipher, err := svc.S.Seal(sc, credentials)
	if err != nil {
		return nil, err
	}
	if err := svc.S.AuthorizeConnection(ctx, sc, cipher, expiry, store.StrOrEmpty(connection["id"])); err != nil {
		return nil, err
	}
	row, err := svc.S.ConnectionByID(ctx, sc, store.StrOrEmpty(connection["id"]))
	if err != nil {
		return nil, err
	}
	return svc.connectionView(ctx, sc, row)
}

// mappingView renders the stored mapping with the countMode enum.
func (svc *Service) mappingView(raw any) map[string]any {
	m := store.Parse(raw)
	out := make(map[string]any, len(m)+1)
	for k, v := range m {
		out[k] = v
	}
	out["countMode"] = svc.value("CountMode", m["countMode"])
	return out
}

func (svc *Service) sourceView(x map[string]any) map[string]any {
	return store.Map(
		"sourceId", x["id"],
		"eventId", x["event_id"],
		"connectionId", x["connection_id"],
		"resourceId", x["resource_id"],
		"provider", svc.value("Provider", x["provider"]),
		"mapping", svc.mappingView(x["mapping"]),
		"eligibility", svc.value("SourceEligibility", x["eligibility"]),
		"version", x["version"],
	)
}

func (svc *Service) sourceSave(ctx context.Context, sc *store.Scope, r map[string]any, c *Req) (any, error) {
	e, err := svc.intEvent(ctx, sc, r, c)
	if err != nil {
		return nil, err
	}
	if err := svc.intakeOpen(e); err != nil {
		return nil, err
	}
	con, err := svc.S.ConnectionAllowed(ctx, sc, store.StrOrEmpty(r["connectionId"]), store.StrOrEmpty(r["eventId"]))
	if err != nil {
		return nil, err
	}
	provider := store.StrOrEmpty(con["provider"])
	if err := errcode.Require(provider == "FEISHU" || provider == "TENCENT",
		422, "INVALID_PROVIDER", "来源仅支持飞书或腾讯文档"); err != nil {
		return nil, err
	}
	mappingJSON, err := store.Stringify(r["mapping"])
	if err != nil {
		return nil, err
	}
	id := store.StrOrEmpty(r["sourceId"])
	if id == "" {
		id = svc.S.ID()
		if err := svc.S.InsertSource(ctx, sc, id, store.StrOrEmpty(r["eventId"]), store.StrOrEmpty(r["connectionId"]),
			store.StrOrEmpty(r["resourceId"]), mappingJSON, store.StrOrEmpty(r["eligibility"])); err != nil {
			return nil, err
		}
	} else {
		old, err := svc.S.SourceByIDForUpdate(ctx, sc, id, store.StrOrEmpty(r["eventId"]))
		if err != nil {
			return nil, err
		}
		if err := store.Version(old, r["version"]); err != nil {
			return nil, err
		}
		if err := svc.S.UpdateSource(ctx, sc, store.StrOrEmpty(r["connectionId"]), store.StrOrEmpty(r["resourceId"]),
			mappingJSON, store.StrOrEmpty(r["eligibility"]), id); err != nil {
			return nil, err
		}
	}
	row, err := svc.S.SourceByIDForUpdate(ctx, sc, id, store.StrOrEmpty(r["eventId"]))
	if err != nil {
		return nil, err
	}
	return svc.sourceView(row), nil
}

func (svc *Service) sourceList(ctx context.Context, sc *store.Scope, r map[string]any, c *Req) (any, error) {
	if _, err := svc.intEvent(ctx, sc, r, c); err != nil {
		return nil, err
	}
	rows, err := svc.S.ListSources(ctx, sc, store.StrOrEmpty(r["eventId"]))
	if err != nil {
		return nil, err
	}
	items := []any{}
	for _, x := range rows {
		items = append(items, svc.sourceView(x))
	}
	connections, err := svc.S.AvailableConnections(ctx, sc, store.StrOrEmpty(r["eventId"]))
	if err != nil {
		return nil, err
	}
	available := []any{}
	for _, x := range connections {
		available = append(available, store.Map(
			"connectionId", x["id"],
			"displayName", x["display_name"],
			"provider", svc.value("Provider", x["provider"]),
			"accountKind", svc.value("AccountKind", x["account_kind"]),
			"status", svc.value("ConnectionStatus", x["status"]),
		))
	}
	return store.Map("items", items, "availableConnections", available), nil
}

// sourcePull queues a PULL job; the worker performs the provider call outside
// any database transaction (SourcePullWorker).
func (svc *Service) sourcePull(ctx context.Context, sc *store.Scope, r map[string]any, c *Req) (any, error) {
	e, err := svc.intEvent(ctx, sc, r, c)
	if err != nil {
		return nil, err
	}
	if err := svc.intakeOpen(e); err != nil {
		return nil, err
	}
	source, err := svc.S.SourceByIDForUpdate(ctx, sc, store.StrOrEmpty(r["sourceId"]), store.StrOrEmpty(r["eventId"]))
	if err != nil {
		return nil, err
	}
	if err := store.Version(source, r["version"]); err != nil {
		return nil, err
	}
	if _, err := svc.S.ConnectionAllowed(ctx, sc, store.StrOrEmpty(source["connection_id"]), store.StrOrEmpty(e["id"])); err != nil {
		return nil, err
	}
	due, err := store.Time(e["personal_due_at"])
	if err != nil {
		return nil, fmt.Errorf("event due time: %w", err)
	}
	expiry := minTime(svc.S.Now().Add(24*time.Hour), due)
	j, err := svc.newJob(ctx, sc, store.StrOrEmpty(e["id"]), "PULL", "pull:"+store.StrOrEmpty(r["operationId"]), "PENDING",
		store.Map("sourceId", source["id"], "sourceVersion", source["version"], "actorId", c.ActorID()), nil, nil, expiry)
	if err != nil {
		return nil, err
	}
	return svc.jobView(j), nil
}

// newJob inserts a job row with the Java attempt/completion defaults and
// returns the stored row (IntegrationService.newJob).
func (svc *Service) newJob(ctx context.Context, sc *store.Scope, eventID, kind, key, status string, payload map[string]any, errorCode any, ref any, expires any) (map[string]any, error) {
	id := svc.S.ID()
	var payloadCipher []byte
	if len(payload) > 0 {
		sealed, err := svc.S.Seal(sc, payload)
		if err != nil {
			return nil, err
		}
		payloadCipher = sealed
	}
	attempts := 1
	if status == "PENDING" {
		attempts = 0
	}
	now := svc.S.Now()
	var completedAt any
	if status == "FAILED" || status == "SUCCEEDED" {
		completedAt = now
	}
	if err := svc.S.InsertJob(ctx, sc, id, eventID, kind, key, status, payloadCipher, attempts, now, ref, errorCode, now, completedAt, expires); err != nil {
		return nil, err
	}
	return svc.S.JobByID(ctx, sc, id)
}

func (svc *Service) jobView(j map[string]any) map[string]any {
	return store.Map(
		"jobId", j["id"],
		"eventId", j["event_id"],
		"kind", svc.value("JobKind", j["kind"]),
		"status", svc.value("JobStatus", j["status"]),
		"attempts", j["attempts"],
		"createdAt", j["created_at"],
		"completedAt", j["completed_at"],
		"resultRef", j["result_ref"],
		"errorCode", j["error_code"],
		"expiresAt", j["expires_at"],
		"version", j["version"],
	)
}

// jobLocked loads a job under lock; organizers only see their event's jobs.
func (svc *Service) jobLocked(ctx context.Context, sc *store.Scope, r map[string]any, c *Req) (map[string]any, error) {
	j, err := svc.S.JobByIDForUpdate(ctx, sc, store.StrOrEmpty(r["jobId"]))
	if err != nil {
		return nil, err
	}
	if !c.Admin() {
		if err := c.Organizer(ctx, store.StrOrEmpty(r["eventId"])); err != nil {
			return nil, err
		}
		if err := errcode.Require(store.StrOrEmpty(j["event_id"]) == store.StrOrEmpty(r["eventId"]),
			404, "RESOURCE_NOT_FOUND", "未找到任务"); err != nil {
			return nil, err
		}
	}
	return j, nil
}

func (svc *Service) jobList(ctx context.Context, sc *store.Scope, r map[string]any, c *Req) (any, error) {
	var all []map[string]any
	var err error
	_, hasEvent := r["eventId"]
	if c.Admin() && !hasEvent {
		all, err = svc.S.ListJobs(ctx, sc)
	} else {
		if !c.Admin() {
			if err := c.Organizer(ctx, store.StrOrEmpty(r["eventId"])); err != nil {
				return nil, err
			}
		}
		all, err = svc.S.JobsByEvent(ctx, sc, store.StrOrEmpty(r["eventId"]))
	}
	if err != nil {
		return nil, err
	}
	out := []any{}
	for _, x := range all {
		if _, has := r["kind"]; has && store.StrOrEmpty(r["kind"]) != store.StrOrEmpty(x["kind"]) {
			continue
		}
		out = append(out, svc.jobView(x))
	}
	return store.Page(out, r), nil
}

func (svc *Service) jobRetry(ctx context.Context, sc *store.Scope, r map[string]any, c *Req) (any, error) {
	j, err := svc.jobLocked(ctx, sc, r, c)
	if err != nil {
		return nil, err
	}
	if err := store.Version(j, r["version"]); err != nil {
		return nil, err
	}
	if err := errcode.Require(store.StrOrEmpty(j["status"]) == "FAILED",
		409, "JOB_STATE_CONFLICT", "只能重试失败任务"); err != nil {
		return nil, err
	}
	if j["expires_at"] != nil {
		expires, err := store.Time(j["expires_at"])
		if err != nil {
			return nil, fmt.Errorf("job expiry: %w", err)
		}
		if err := errcode.Require(expires.After(svc.S.Now()), 410, "JOB_EXPIRED", "任务已过期"); err != nil {
			return nil, err
		}
	}
	kind := store.StrOrEmpty(j["kind"])
	switch kind {
	case "PULL":
		e, err := svc.S.Event(ctx, sc, store.StrOrEmpty(j["event_id"]))
		if err != nil {
			return nil, err
		}
		if err := svc.intakeOpen(e); err != nil {
			return nil, err
		}
		payload, err := svc.S.Unseal(j["payload_cipher"])
		if err != nil {
			return nil, err
		}
		source, err := svc.S.SourceByIDEventForUpdate(ctx, sc, store.StrOrEmpty(payload["sourceId"]), store.StrOrEmpty(j["event_id"]))
		if err != nil {
			return nil, err
		}
		if err := errcode.Require(store.Num(source["version"]) == store.Num(payload["sourceVersion"]),
			409, "PREVIEW_CONFLICT", "来源配置已变化，请重新发起拉取"); err != nil {
			return nil, err
		}
	case "CLEANUP":
		return nil, errcode.New(409, "USE_CLEANUP_RETRY", "请从清理记录重试")
	case "EXPORT":
		if err := c.Organizer(ctx, store.StrOrEmpty(j["event_id"])); err != nil {
			return nil, err
		}
		return nil, errcode.New(409, "NEW_EXPORT_REQUIRED", "请重新生成并确认导出范围")
	default:
		if err := svc.requireEmail(); err != nil {
			return nil, err
		}
	}
	if err := svc.S.ResetJob(ctx, sc, svc.S.Now(), store.StrOrEmpty(j["id"])); err != nil {
		return nil, err
	}
	row, err := svc.S.JobByID(ctx, sc, store.StrOrEmpty(j["id"]))
	if err != nil {
		return nil, err
	}
	return svc.jobView(row), nil
}

func (svc *Service) emailResend(ctx context.Context, sc *store.Scope, r map[string]any, c *Req) (any, error) {
	e, err := svc.intEvent(ctx, sc, r, c)
	if err != nil {
		return nil, err
	}
	p, err := svc.partyRaw(ctx, sc, store.StrOrEmpty(e["id"]), store.StrOrEmpty(r["partyId"]))
	if err != nil {
		return nil, err
	}
	if err := errcode.Require(store.StrOrEmpty(p["status"]) == "APPROVED",
		409, "PARTY_NOT_APPROVED", "此组当前不可领票"); err != nil {
		return nil, err
	}
	fields, err := svc.S.Unseal(p["fields_cipher"])
	if err != nil {
		return nil, err
	}
	if err := errcode.Require(fields["email"] != nil, 422, "EMAIL_REQUIRED", "此组未提供邮箱"); err != nil {
		return nil, err
	}
	if err := svc.requireEmail(); err != nil {
		return nil, err
	}
	due, err := store.Time(e["personal_due_at"])
	if err != nil {
		return nil, fmt.Errorf("event due time: %w", err)
	}
	j, err := svc.newJob(ctx, sc, store.StrOrEmpty(e["id"]), "EMAIL", "resend:"+store.StrOrEmpty(r["operationId"]), "PENDING",
		store.Map("partyId", p["id"]), nil, p["id"], due)
	if err != nil {
		return nil, err
	}
	return svc.jobView(j), nil
}

// batchLocked loads a batch under lock and enforces its staging expiry
// (IntegrationService.batch).
func (svc *Service) batchLocked(ctx context.Context, sc *store.Scope, r map[string]any) (map[string]any, error) {
	b, err := svc.S.BatchByIDForUpdate(ctx, sc, store.StrOrEmpty(r["batchId"]), store.StrOrEmpty(r["eventId"]))
	if err != nil {
		return nil, err
	}
	expires, err := store.Time(b["expires_at"])
	if err != nil {
		return nil, fmt.Errorf("batch expiry: %w", err)
	}
	if err := errcode.Require(expires.After(svc.S.Now()), 410, "BATCH_EXPIRED", "暂存名单已过期"); err != nil {
		return nil, err
	}
	return b, nil
}

// previewVersion guards mutations against a stale preview.
func (svc *Service) previewVersion(b map[string]any, r map[string]any) error {
	return errcode.Require(store.StrOrEmpty(b["status"]) == "PREVIEWED" &&
		store.Num(b["preview_version"]) == store.Num(r["previewVersion"]),
		409, "PREVIEW_CONFLICT", "请重新预览并核对名单")
}

// sourceScope derives the stable link scope of a batch: file identity for
// uploads, connection/resource identity for source pulls.
func (svc *Service) sourceScope(ctx context.Context, sc *store.Scope, b map[string]any) (string, error) {
	identity := store.StrOrEmpty(b["file_digest"]) + ":" + store.StrOrEmpty(b["sheet_ref"])
	if b["source_id"] != nil {
		source, err := svc.S.SourceConnectionRef(ctx, sc, store.StrOrEmpty(b["source_id"]))
		if err != nil {
			return "", err
		}
		identity = store.StrOrEmpty(source["connection_id"]) + ":" + store.StrOrEmpty(source["resource_id"])
	}
	return fmt.Sprintf("%x", svc.S.Crypto.Hash(store.StrOrEmpty(b["event_id"])+":"+identity)), nil
}
