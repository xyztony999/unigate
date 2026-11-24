package service

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"unigate/backend/internal/contract"
	"unigate/backend/internal/errcode"
	"unigate/backend/internal/store"
)

// Dispatch routes an operation to its owning service handler and wraps write
// operations with the idempotency / replay / audit envelope, reproducing
// ApiController.execute() + dispatch(). Authentication (Authenticate) and the
// HTTP envelope stay in the api layer; this method returns business data or an
// *errcode.Error. The caller may pass a scope already bound to a transaction;
// otherwise Dispatch opens one (READ COMMITTED, 5s; Statistics gets an
// independent REPEATABLE READ read-only snapshot) and commits it on success.
func (svc *Service) Dispatch(ctx context.Context, sc *store.Scope, opName string, r map[string]any, c *Req) (any, error) {
	op := svc.C.Operations[opName]
	if op == nil {
		return nil, errcode.New(404, "RESOURCE_NOT_FOUND", "未找到接口")
	}
	if sc == nil {
		sc = &store.Scope{}
	}
	c.Sc = sc
	ownTx := sc.Tx == nil
	if ownTx {
		opts := store.TxOptions{Timeout: 5 * time.Second}
		if opName == "Statistics" {
			opts.RepeatableRead = true
			opts.ReadOnly = true
		}
		begun, err := svc.S.Begin(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("begin dispatch tx: %w", err)
		}
		sc = begun
		c.Sc = sc
		defer svc.S.Rollback(ctx, sc)
	}
	sc.ReadOnly = !op.Mutation
	data, err := svc.executeOp(ctx, sc, op, r, c)
	if err != nil {
		return nil, dispatchConflict(err)
	}
	if ownTx {
		if err := svc.S.Commit(ctx, sc); err != nil {
			return nil, dispatchConflict(fmt.Errorf("commit dispatch tx: %w", err))
		}
	}
	return data, nil
}

// dispatchConflict maps PostgreSQL lock/serialization and integrity failures
// to the frozen controller catch blocks (409 OPERATION_IN_PROGRESS /
// DATA_CONFLICT); business errors pass through untouched.
func dispatchConflict(err error) error {
	if store.IsLockConflict(err) {
		return errcode.New(409, "OPERATION_IN_PROGRESS", "操作竞争，请沿用原操作编号核对")
	}
	if store.IsIntegrityViolation(err) {
		return errcode.New(409, "DATA_CONFLICT", "数据已变化或记录冲突，请刷新核对")
	}
	return err
}

// executeOp reproduces the transactional body of ApiController.execute():
// event scoping, the idempotent insert/replay branch, business dispatch,
// response-contract check, audit and the sealed result commit.
func (svc *Service) executeOp(ctx context.Context, sc *store.Scope, op *contract.Operation, r map[string]any, c *Req) (any, error) {
	name := op.Name
	ev := store.StrOrEmpty(r["eventId"])
	if ev == "" && c.Session != nil {
		ev = store.StrOrEmpty(c.Session["event_id"])
	}
	if ev == "" && c.Device != nil && (c.Auth == "GATE" || c.Auth == "DEVICE") {
		ev = store.StrOrEmpty(c.Device["event_id"])
	}
	if ev != "" {
		if name != "CleanupDetail" && name != "CleanupRetry" {
			if _, err := svc.S.Event(ctx, sc, ev); err != nil {
				return nil, err
			}
		}
		if c.Device != nil && (c.Auth == "GATE" || c.Auth == "DEVICE") {
			d, err := svc.S.DeviceActiveForUpdate(ctx, sc, store.StrOrEmpty(c.Device["id"]))
			if err != nil {
				return nil, err
			}
			c.Device = d
		}
		if c.Auth == "STAFF" && name != "MyGrantList" {
			globalAdmin := c.Admin() && containsOpRole(op.Roles, "ADMIN")
			if !globalAdmin {
				role, err := c.EventRole(ctx, ev)
				if err != nil {
					return nil, err
				}
				if !containsOpRole(op.Roles, role) {
					return nil, errcode.New(403, "FORBIDDEN", "当前活动授权不包含此动作")
				}
			}
		}
	}
	if !op.Mutation {
		data, err := svc.dispatchOp(ctx, sc, name, r, c, "")
		if err != nil {
			return nil, err
		}
		return svc.checked(ctx, op, data)
	}

	actor := "CLIENT:" + store.StrOrEmpty(c.Session["id"])
	if c.Auth == "GATE" {
		actor = "GATE"
	} else if c.Account != nil && c.Auth == "STAFF" {
		actor = "STAFF:" + store.StrOrEmpty(c.ActorID())
	}
	digestInput := make(map[string]any, len(r))
	for k, v := range r {
		digestInput[k] = v
	}
	delete(digestInput, "operationId")
	if name == "GateCommit" {
		// Bind the digest to the scanned party and the acting device so a
		// replayed operationId cannot be redirected at another target.
		scan, err := svc.gateScan(store.StrOrEmpty(r["scanContextToken"]), c, false)
		if err != nil {
			return nil, err
		}
		delete(digestInput, "scanContextToken")
		digestInput["partyId"] = scan["partyId"]
		digestInput["deviceId"] = store.StrOrEmpty(c.Device["id"])
	}
	if op.Upload {
		f, ok := r["file"].(UploadFile)
		if !ok {
			return nil, errcode.New(422, "INVALID_FILE", "无法读取文件")
		}
		sum := sha256.Sum256(f.Data)
		digestInput["file"] = hex.EncodeToString(sum[:])
	}
	text, err := store.Stringify(canonicalJSON(digestInput))
	if err != nil {
		return nil, fmt.Errorf("stringify digest input: %w", err)
	}
	digest := svc.S.Crypto.Hash(text)
	eventScope := ev
	if eventScope == "" {
		eventScope = "GLOBAL"
	}
	entry := svc.S.ID()
	var eventID any
	if ev != "" {
		eventID = ev
	}
	inserted, err := svc.S.InsertOperation(ctx, sc, entry, actor, eventScope, eventID, name, store.StrOrEmpty(r["operationId"]), digest)
	if err != nil {
		return nil, fmt.Errorf("insert operation: %w", err)
	}
	if inserted == 0 {
		return svc.replayOperation(ctx, sc, op, r, c, actor, eventScope, ev, digest)
	}
	result, err := svc.dispatchOp(ctx, sc, name, r, c, entry)
	if err != nil {
		return nil, err
	}
	result, err = svc.checked(ctx, op, result)
	if err != nil {
		return nil, err
	}
	resource := resourceId(result)
	var auditEvent any
	if ev != "" {
		auditEvent = ev
	}
	var deviceID any
	if c.Device != nil {
		deviceID = store.StrOrEmpty(c.Device["id"])
	}
	if err := svc.S.Audit(ctx, sc, auditEvent, name, c.ActorID(), deviceID, resource, r["operationId"], r["reason"]); err != nil {
		return nil, fmt.Errorf("audit operation: %w", err)
	}
	expiry, err := svc.privateExpiry(ctx, sc, name, ev)
	if err != nil {
		return nil, err
	}
	cipher, err := svc.S.Seal(sc, store.Map("data", result, "cookies", sc.Cookies))
	if err != nil {
		return nil, fmt.Errorf("seal operation result: %w", err)
	}
	if err := svc.S.CompleteOperation(ctx, sc, resource, cipher, expiry, entry); err != nil {
		return nil, fmt.Errorf("complete operation: %w", err)
	}
	return result, nil
}

// replayOperation serves the stored result of a succeeded operation with the
// same idempotency key (ApiController.execute inserted==0 branch).
func (svc *Service) replayOperation(ctx context.Context, sc *store.Scope, op *contract.Operation, r map[string]any, c *Req, actor, eventScope, ev string, digest []byte) (any, error) {
	name := op.Name
	prior, err := svc.S.OperationByScope(ctx, sc, actor, eventScope, name, store.StrOrEmpty(r["operationId"]))
	if err != nil {
		return nil, fmt.Errorf("load prior operation: %w", err)
	}
	priorDigest, _ := prior["payload_digest"].([]byte)
	if subtle.ConstantTimeCompare(digest, priorDigest) != 1 {
		return nil, errcode.New(409, "IDEMPOTENCY_CONFLICT", "原操作编号对应其他内容，请核对")
	}
	if store.StrOrEmpty(prior["status"]) != "SUCCEEDED" {
		return nil, errcode.New(409, "OPERATION_IN_PROGRESS", "原操作正在处理中，请沿用原编号查询")
	}
	if prior["private_result_expires_at"] != nil {
		exp, err := store.Time(prior["private_result_expires_at"])
		if err != nil {
			return nil, fmt.Errorf("prior result expiry: %w", err)
		}
		if !exp.After(svc.S.Now()) {
			return nil, errcode.New(410, "RESULT_EXPIRED", "原结果已到期，请联系负责人核对")
		}
	}
	bundle, err := svc.S.Unseal(prior["result_cipher"])
	if err != nil {
		return nil, fmt.Errorf("unseal prior result: %w", err)
	}
	for _, cookie := range store.List(bundle["cookies"]) {
		sc.Cookies = append(sc.Cookies, store.StrOrEmpty(cookie))
	}
	result := store.Parse(bundle["data"])
	if c.Auth == "STAFF" && ev != "" && replayProjectionOps[name] {
		fields, err := c.Fields(ctx, ev)
		if err != nil {
			return nil, err
		}
		projectReplay(result, fields)
	}
	if _, ok := result["replayed"]; ok {
		result["replayed"] = true
	}
	if _, ok := result["showAsNewAdmission"]; ok {
		result["showAsNewAdmission"] = false
	}
	return svc.checked(ctx, op, result)
}

// replayProjectionOps are the mutations whose replayed private fields must be
// re-projected against the caller's current grant (ApiController.execute).
var replayProjectionOps = map[string]bool{
	"PartyChange":       true,
	"PartyCancel":       true,
	"CredentialReissue": true,
	"CredentialRevoke":  true,
	"GateWalkIn":        true,
	"GateIncrease":      true,
}

// dispatchOp routes an operation name to its owning service handler
// (ApiController.dispatch). Notification operations fall through to
// handleIntegration, which delegates them to handleNotification.
func (svc *Service) dispatchOp(ctx context.Context, sc *store.Scope, op string, r map[string]any, c *Req, entry string) (any, error) {
	switch op {
	case "SessionBootstrap":
		return svc.Bootstrap(ctx, sc, c.Info)
	case "Login", "Logout", "SessionMe", "PasswordChange",
		"AccountList", "AccountCreate", "AccountUpdate",
		"GrantEventOptions", "MyGrantList", "GrantList", "GrantSave", "GrantRevoke":
		return svc.handleIdentity(ctx, sc, op, r, c, entry)
	case "EventCreate", "EventList", "EventOptions", "PublicEvent", "EventDetail",
		"EventUpdate", "EventTransition", "PolicyGet", "PolicyUpdate":
		return svc.handleEvent(ctx, sc, op, r, c, entry)
	case "SubmitRegistration", "PartyExchange", "MyParty", "MyCredential",
		"PartyList", "PartyDetail", "PartyChange", "MyPartyChange", "PartyCancel", "MyPartyCancel",
		"ReviewList", "ReviewDecide", "CredentialRevoke", "CredentialReissue",
		"GateWalkIn", "GateIncrease":
		return svc.handleParty(ctx, sc, op, r, c, entry)
	case "DeviceRegister", "DeviceClaim", "DeviceDetail", "DeviceList", "DeviceRevoke",
		"TicketResolve", "GateCommit", "GateOperation", "GateLookup", "GateRecords", "GateCorrect":
		return svc.handleGate(ctx, sc, op, r, c, entry)
	case "Statistics", "AuditList", "OperationResult", "MyOperation", "RegistrationResult":
		return svc.reports(ctx, sc, op, r, c)
	default:
		return svc.handleIntegration(ctx, sc, op, r, c, entry)
	}
}

// checked validates a successful response against the frozen contract schema
// inside the request transaction (ApiController.checked). A mismatch is a
// server defect, not a client error.
func (svc *Service) checked(ctx context.Context, op *contract.Operation, data any) (any, error) {
	if _, ok := data.(Binary); ok {
		return data, nil
	}
	if err := svc.C.Validate(op.Data, store.JSONReady(data)); err != nil {
		slog.ErrorContext(ctx, "response contract mismatch", "operation", op.Name, "error", err)
		return nil, fmt.Errorf("response contract mismatch for %s: %w", op.Name, err)
	}
	return data, nil
}

// projectReplay retains only the grant-allowed keys in every {partyId,fields}
// object found in a replayed result (ApiController.projectReplay).
func projectReplay(value map[string]any, fields map[string]bool) {
	if _, ok := value["partyId"]; ok {
		if data, ok := value["fields"].(map[string]any); ok {
			for k := range data {
				if !fields[k] {
					delete(data, k)
				}
			}
		}
	}
	for _, child := range value {
		switch t := child.(type) {
		case map[string]any:
			projectReplay(t, fields)
		case []any:
			for _, item := range t {
				if m, ok := item.(map[string]any); ok {
					projectReplay(m, fields)
				}
			}
		}
	}
}

// privateExpiry caps the sealed-result lifetime of credential-bearing
// operations at 24h or the event's personal-data due time, whichever is
// earlier (ApiController.privateExpiry).
func (svc *Service) privateExpiry(ctx context.Context, sc *store.Scope, op, ev string) (any, error) {
	switch op {
	case "Login", "PartyExchange", "SubmitRegistration", "RecoveryVerify",
		"DeviceRegister", "DeviceClaim", "CredentialReissue", "SmsVerify":
	default:
		return nil, nil
	}
	expiry := svc.S.Now().Add(24 * time.Hour)
	if ev != "" {
		row, err := svc.S.EventDue(ctx, sc, ev)
		if err != nil {
			return nil, fmt.Errorf("event due for private expiry: %w", err)
		}
		due, err := store.Time(row["personal_due_at"])
		if err != nil {
			return nil, fmt.Errorf("event due time: %w", err)
		}
		if due.Before(expiry) {
			expiry = due
		}
	}
	return expiry, nil
}

// canonicalJSON recursively sorts map keys so the payload digest is
// independent of input key order (ApiController.canonical + Store.stringify).
func canonicalJSON(o any) any {
	switch t := o.(type) {
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := make(map[string]any, len(t))
		for _, k := range keys {
			out[k] = canonicalJSON(t[k])
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, v := range t {
			out[i] = canonicalJSON(v)
		}
		return out
	case store.RawJSON:
		return canonicalJSON(store.ParseJSON([]byte(t)))
	default:
		return o
	}
}

// resourceId picks the audit/operation resource id from a result map
// (ApiController.resourceId).
func resourceId(result any) any {
	m, ok := result.(map[string]any)
	if !ok {
		return nil
	}
	for _, k := range []string{
		"partyId", "eventId", "recordId", "reviewId", "batchId", "exportId",
		"jobId", "accountId", "grantId", "connectionId", "sourceId", "correctionId",
	} {
		if m[k] != nil {
			return m[k]
		}
	}
	if p, ok := m["party"].(map[string]any); ok {
		return p["partyId"]
	}
	if d, ok := m["device"].(map[string]any); ok {
		return d["deviceId"]
	}
	return nil
}

func containsOpRole(roles []string, want string) bool {
	for _, r := range roles {
		if r == want {
			return true
		}
	}
	return false
}

// reports serves the read-only reporting operations that Java kept in the
// controller (ApiController.reports).
func (svc *Service) reports(ctx context.Context, sc *store.Scope, op string, r map[string]any, c *Req) (any, error) {
	ev := store.StrOrEmpty(r["eventId"])
	switch op {
	case "MyOperation":
		ev = store.StrOrEmpty(c.Session["event_id"])
	case "RegistrationResult":
		if _, err := svc.S.Event(ctx, sc, ev); err != nil {
			return nil, err
		}
		o, err := svc.S.RegistrationOperation(ctx, sc, "CLIENT:"+store.StrOrEmpty(c.Session["id"]), ev, store.StrOrEmpty(r["operationId"]))
		if err != nil {
			return nil, fmt.Errorf("load registration result: %w", err)
		}
		bundle, err := svc.S.Unseal(o["result_cipher"])
		if err != nil {
			return nil, fmt.Errorf("unseal registration result: %w", err)
		}
		v := store.Parse(bundle["data"])
		v["replayed"] = true
		return v, nil
	case "Statistics":
		return svc.statistics(ctx, sc, ev, c)
	case "AuditList":
		return svc.auditList(ctx, sc, ev, r, c)
	}
	// OperationResult / MyOperation share the lookup below.
	if op == "OperationResult" {
		if err := c.GateHistory(ctx, ev); err != nil {
			return nil, err
		}
	} else if !containsOpRole([]string{"MyPartyChange", "MyPartyCancel"}, store.StrOrEmpty(r["action"])) {
		return nil, errcode.New(403, "FORBIDDEN", "不允许查询此动作")
	}
	found, err := svc.S.OperationsByEventAction(ctx, sc, ev, store.StrOrEmpty(r["operationId"]), store.StrOrEmpty(r["action"]))
	if err != nil {
		return nil, fmt.Errorf("lookup operation: %w", err)
	}
	if len(found) == 0 {
		return nil, errcode.New(404, "OPERATION_NOT_FOUND", "尚未查到原操作，请沿用原编号")
	}
	o := found[0]
	if op == "MyOperation" {
		if store.StrOrEmpty(c.Session["party_id"]) != store.StrOrEmpty(o["resource_id"]) {
			return nil, errcode.New(404, "RESOURCE_NOT_FOUND", "未找到资源")
		}
	} else {
		role, err := c.EventRole(ctx, ev)
		if err != nil {
			return nil, err
		}
		if role != "ORGANIZER" {
			if store.StrOrEmpty(r["action"]) != "GateCommit" {
				return nil, errcode.New(403, "FORBIDDEN", "不能查询此动作")
			}
			a, err := svc.S.AdmissionByOperationEntry(ctx, sc, store.StrOrEmpty(o["id"]))
			if err != nil {
				return nil, fmt.Errorf("load admission for operation: %w", err)
			}
			if store.StrOrEmpty(c.ActorID()) != store.StrOrEmpty(a["actor_id"]) &&
				(c.Device == nil || store.StrOrEmpty(c.Device["id"]) != store.StrOrEmpty(a["device_id"])) {
				return nil, errcode.New(404, "RESOURCE_NOT_FOUND", "未找到资源")
			}
		}
	}
	return store.Map(
		"operationId", r["operationId"],
		"status", svc.value("OperationStatus", o["status"]),
		"action", o["action"],
		"resourceId", store.Str(o["resource_id"]),
		"committedAt", o["committed_at"],
		"resultCode", o["result_code"],
		"replayed", true,
	), nil
}

// statistics aggregates the attendance summary (ApiController.reports
// Statistics branch) inside the read-only snapshot transaction.
func (svc *Service) statistics(ctx context.Context, sc *store.Scope, ev string, c *Req) (any, error) {
	if err := c.View(ctx, ev); err != nil {
		return nil, err
	}
	if err := c.Action(ctx, ev, "VIEW_STATISTICS"); err != nil {
		return nil, err
	}
	e, err := svc.S.Event(ctx, sc, ev)
	if err != nil {
		return nil, err
	}
	stats, err := svc.S.PartyStats(ctx, sc, ev)
	if err != nil {
		return nil, fmt.Errorf("party stats: %w", err)
	}
	gross, err := svc.S.AdmissionGross(ctx, sc, ev)
	if err != nil {
		return nil, fmt.Errorf("admission gross: %w", err)
	}
	corrected, err := svc.S.AdmissionCorrected(ctx, sc, ev)
	if err != nil {
		return nil, fmt.Errorf("admission corrected: %w", err)
	}
	onsite, err := svc.S.OnsiteAdded(ctx, sc, ev)
	if err != nil {
		return nil, fmt.Errorf("onsite added: %w", err)
	}
	allocated := store.Num(e["allocated_count"])
	net := store.Num(stats["net"])
	var rate any
	if allocated != 0 {
		rate = float64(net) / float64(allocated)
	}
	out := store.Map(
		"eventId", ev,
		"asOf", svc.S.Now(),
		"applicationGroups", store.Num(stats["groups"]),
		"pendingGroups", store.Num(stats["pending"]),
		"approvedGroups", store.Num(stats["approved"]),
		"allocatedPeople", allocated,
		"grossAdmissions", gross,
		"correctedPeople", corrected,
		"netAdmissions", net,
		"unusedPlaces", allocated-net,
		"walkInGroups", store.Num(stats["walkins"]),
		"onsiteAddedPlaces", onsite,
	)
	// Java puts admissionRate explicitly (null when nothing is allocated);
	// store.Map skips nil values, which would drop the required frozen field.
	out["admissionRate"] = rate
	return out, nil
}

// auditList pages the event audit trail with the lead/self visibility rule
// (ApiController.reports AuditList branch).
func (svc *Service) auditList(ctx context.Context, sc *store.Scope, ev string, r map[string]any, c *Req) (any, error) {
	if err := c.GateHistory(ctx, ev); err != nil {
		return nil, err
	}
	role, err := c.EventRole(ctx, ev)
	if err != nil {
		return nil, err
	}
	lead := role == "ORGANIZER"
	rows, err := svc.S.AuditList(ctx, sc, ev)
	if err != nil {
		return nil, fmt.Errorf("list audit: %w", err)
	}
	all := []any{}
	for _, a := range rows {
		if !lead && store.StrOrEmpty(c.ActorID()) != store.StrOrEmpty(a["actor_id"]) &&
			(c.Device == nil || store.StrOrEmpty(c.Device["id"]) != store.StrOrEmpty(a["device_id"])) {
			continue
		}
		if v, ok := r["action"]; ok && store.StrOrEmpty(v) != store.StrOrEmpty(a["action"]) {
			continue
		}
		all = append(all, store.Map(
			"auditId", store.StrOrEmpty(a["id"]),
			"eventId", ev,
			"action", a["action"],
			"actorId", store.Str(a["actor_id"]),
			"deviceId", store.Str(a["device_id"]),
			"resourceId", store.Str(a["resource_id"]),
			"occurredAt", a["occurred_at"],
		))
	}
	return store.Page(all, r), nil
}
