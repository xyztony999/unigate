package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"unigate/backend/internal/errcode"
	"unigate/backend/internal/store"
)

// partyRaw locks a party row (PartyService.raw).
func (svc *Service) partyRaw(ctx context.Context, sc *store.Scope, ev, partyID string) (map[string]any, error) {
	return svc.S.PartyRaw(ctx, sc, ev, partyID)
}

// partyProgress maps net_admitted/approved_size to the progress code.
func partyProgress(p map[string]any) string {
	approved := store.Num(p["approved_size"])
	n := store.Num(p["net_admitted"])
	if n == 0 {
		return "NONE"
	}
	if n == approved {
		return "COMPLETE"
	}
	return "PARTIAL"
}

func allFields() map[string]bool {
	return map[string]bool{"name": true, "phone": true, "email": true, "companionCount": true}
}

// partyView renders a party projecting only allowed fields.
func (svc *Service) partyView(ctx context.Context, sc *store.Scope, p map[string]any, allowed map[string]bool) (map[string]any, error) {
	fields, err := svc.S.Unseal(p["fields_cipher"])
	if err != nil {
		return nil, err
	}
	for k := range fields {
		if !allowed[k] {
			delete(fields, k)
		}
	}
	return store.Map(
		"partyId", store.StrOrEmpty(p["id"]),
		"partyReference", p["party_reference"],
		"eventId", store.StrOrEmpty(p["event_id"]),
		"fields", fields,
		"requestedSize", p["requested_size"],
		"approvedSize", p["approved_size"],
		"pendingIncrease", p["pending_increase"],
		"netAdmitted", p["net_admitted"],
		"remainingCount", store.Num(p["approved_size"])-store.Num(p["net_admitted"]),
		"status", svc.value("PartyStatus", p["status"]),
		"progress", svc.value("AdmissionProgress", partyProgress(p)),
		"sourceType", svc.value("SourceType", p["source_type"]),
		"version", p["version"],
		"createdAt", p["created_at"],
		"updatedAt", p["updated_at"],
		"hasAdmissionHistory", p["ever_admitted"],
	), nil
}

// validateFields enforces the event policy field rules and trims/blanks checks.
func (svc *Service) validateFields(e map[string]any, fields map[string]any) error {
	rules := store.Parse(store.Parse(e["policy"])["fields"])
	for key, rv := range rules {
		rule, _ := rv.(map[string]any)
		if _, has := fields[key]; has && rule["enabled"] != true {
			return errcode.New(422, "FIELD_DISABLED", "未启用字段不能提交: "+key)
		}
		if rule["required"] == true {
			if _, has := fields[key]; !has {
				return errcode.New(422, "VALIDATION_FAILED", "缺少必填字段: "+key)
			}
		}
	}
	for key, v := range fields {
		if text, ok := v.(string); ok {
			fields[key] = strings.TrimSpace(text)
		}
	}
	for _, v := range fields {
		if text, ok := v.(string); ok && strings.TrimSpace(text) == "" {
			return errcode.New(422, "FIELD_VALIDATION_FAILED", "字段不能为空白")
		}
	}
	return nil
}

// allocate adjusts the event allocated_count within capacity.
func (svc *Service) allocate(ctx context.Context, sc *store.Scope, e map[string]any, delta int) error {
	changed, err := svc.S.AllocateEvent(ctx, sc, delta, store.StrOrEmpty(e["id"]))
	if err != nil {
		return err
	}
	if changed != 1 {
		return errcode.New(409, "CAPACITY_CONFLICT", "可用名额不足")
	}
	e["allocated_count"] = store.Num(e["allocated_count"]) + delta
	return nil
}

func (svc *Service) searchHMAC(o any) []byte {
	if o == nil {
		return nil
	}
	return svc.S.Crypto.Hash(strings.TrimSpace(store.StrOrEmpty(o)))
}

// partyCreate creates a party, issues capabilities and queues email.
func (svc *Service) partyCreate(ctx context.Context, sc *store.Scope, c *Req, event map[string]any, fields map[string]any, sourceType string, approved bool, noticeVersion, entry string) (map[string]any, error) {
	if err := svc.validateFields(event, fields); err != nil {
		return nil, err
	}
	size := 1 + store.Num(fields["companionCount"])
	if approved {
		if err := svc.allocate(ctx, sc, event, size); err != nil {
			return nil, err
		}
	}
	id := svc.S.ID()
	ref := "UG-" + strings.ToUpper(id[:8])
	fieldsCipher, err := svc.S.Seal(sc, fields)
	if err != nil {
		return nil, err
	}
	evID := store.StrOrEmpty(event["id"])
	approvedSize := 0
	status := "PENDING"
	if approved {
		approvedSize = size
		status = "APPROVED"
	}
	if err := svc.S.InsertParty(ctx, sc, id, evID, ref, fieldsCipher, "event:"+evID+":v1",
		svc.searchHMAC(fields["name"]), svc.searchHMAC(fields["phone"]), svc.searchHMAC(fields["email"]),
		size, approvedSize, status, sourceType, noticeVersion); err != nil {
		return nil, err
	}
	p, err := svc.partyRaw(ctx, sc, evID, id)
	if err != nil {
		return nil, err
	}
	if _, err := svc.issueCapability(ctx, sc, p, "QUERY", event); err != nil {
		return nil, err
	}
	if _, err := svc.issueCapability(ctx, sc, p, "CHANGE", event); err != nil {
		return nil, err
	}
	if approved {
		if _, err := svc.issueCapability(ctx, sc, p, "TICKET", event); err != nil {
			return nil, err
		}
	} else if err := svc.insertReview(ctx, sc, p, "INITIAL", size, size); err != nil {
		return nil, err
	}
	if err := svc.enqueueEmail(ctx, sc, p, event); err != nil {
		return nil, err
	}
	return svc.partyView(ctx, sc, p, allFields())
}

// enqueueEmail queues an EMAIL job when the party provided an email.
func (svc *Service) enqueueEmail(ctx context.Context, sc *store.Scope, p, e map[string]any) error {
	f, err := svc.S.Unseal(p["fields_cipher"])
	if err != nil {
		return err
	}
	if _, has := f["email"]; !has {
		return nil
	}
	payload, err := svc.S.Seal(sc, store.Map("partyId", store.StrOrEmpty(p["id"]), "email", f["email"]))
	if err != nil {
		return err
	}
	due, _ := store.Time(e["personal_due_at"])
	return svc.S.InsertJob(ctx, sc, svc.S.ID(), e["id"], "EMAIL",
		fmt.Sprintf("party:%v:%v", p["id"], p["version"]), "PENDING", payload, 0, svc.S.Now(), nil, nil, svc.S.Now(), nil, due)
}

// issueCapability creates an ACTIVE credential and returns its token.
func (svc *Service) issueCapability(ctx context.Context, sc *store.Scope, p map[string]any, kind string, e map[string]any) (string, error) {
	token := svc.S.Crypto.Token()
	tokenCipher, err := svc.S.Seal(sc, store.Map("token", token))
	if err != nil {
		return "", err
	}
	due, _ := store.Time(e["personal_due_at"])
	if err := svc.S.InsertCredential(ctx, sc, svc.S.ID(), store.StrOrEmpty(p["event_id"]), store.StrOrEmpty(p["id"]), kind,
		svc.S.Crypto.Hash(token), tokenCipher, p["version"], due); err != nil {
		return "", err
	}
	return token, nil
}

// partyAccess returns the query/change tokens bundle.
func (svc *Service) partyAccess(ctx context.Context, sc *store.Scope, p, e map[string]any) (map[string]any, error) {
	out := store.Map("expiresAt", e["personal_due_at"])
	creds, err := svc.S.ActiveAccessCredentials(ctx, sc, store.StrOrEmpty(p["id"]))
	if err != nil {
		return nil, err
	}
	for _, cred := range creds {
		tokenCipher, err := svc.S.Unseal(cred["token_cipher"])
		if err != nil {
			return nil, err
		}
		key := "changeToken"
		if store.StrOrEmpty(cred["kind"]) == "QUERY" {
			key = "queryToken"
		}
		out[key] = tokenCipher["token"]
	}
	return out, nil
}

// partyCredential renders the ticket credential view.
func (svc *Service) partyCredential(ctx context.Context, sc *store.Scope, p, e map[string]any) (map[string]any, error) {
	creds, err := svc.S.TicketCredentials(ctx, sc, store.StrOrEmpty(p["id"]))
	if err != nil {
		return nil, err
	}
	if len(creds) == 0 {
		return store.Map(
			"partyId", store.StrOrEmpty(p["id"]),
			"status", svc.value("CredentialStatus", "REVOKED"),
			"credentialVersion", p["version"],
			"expiresAt", e["personal_due_at"],
		), nil
	}
	row := creds[0]
	out := store.Map(
		"partyId", store.StrOrEmpty(p["id"]),
		"status", svc.value("CredentialStatus", row["status"]),
		"credentialVersion", row["credential_version"],
		"expiresAt", row["expires_at"],
	)
	if store.StrOrEmpty(row["status"]) == "ACTIVE" && store.StrOrEmpty(p["status"]) == "APPROVED" {
		tokenCipher, err := svc.S.Unseal(row["token_cipher"])
		if err != nil {
			return nil, err
		}
		out["ticketToken"] = tokenCipher["token"]
	}
	return out, nil
}

func (svc *Service) insertReview(ctx context.Context, sc *store.Scope, p map[string]any, kind string, requested, delta int) error {
	return svc.S.InsertReview(ctx, sc, svc.S.ID(), store.StrOrEmpty(p["event_id"]), store.StrOrEmpty(p["id"]), kind, requested, delta)
}

// canSelf reports whether a self-service action is currently permitted.
func (svc *Service) canSelf(p, e map[string]any, key string) bool {
	policy := store.Parse(e["policy"])
	deadlineVal := policy["selfServiceEndsAt"]
	if deadlineVal == nil {
		deadlineVal = e["registration_ends_at"]
	}
	deadline, err := store.Time(deadlineVal)
	if err != nil {
		return false
	}
	status := store.StrOrEmpty(p["status"])
	return policy[key] == true &&
		store.StrOrEmpty(e["status"]) == "READY" &&
		p["ever_admitted"] != true &&
		svc.S.Now().Before(deadline) &&
		(status == "APPROVED" || status == "PENDING")
}

// partyState renders the self-service party state view.
func (svc *Service) partyState(ctx context.Context, sc *store.Scope, p, e map[string]any) (map[string]any, error) {
	out, err := svc.partyView(ctx, sc, p, allFields())
	if err != nil {
		return nil, err
	}
	for _, k := range []string{"requestedSize", "sourceType", "createdAt", "updatedAt", "hasAdmissionHistory"} {
		delete(out, k)
	}
	companionRule := store.Parse(store.Parse(store.Parse(e["policy"])["fields"])["companionCount"])
	companion := companionRule["enabled"] == true
	out["canChange"] = companion && svc.canSelf(p, e, "selfEditEnabled")
	out["canCancel"] = svc.canSelf(p, e, "selfCancelEnabled")
	out["expiresAt"] = e["personal_due_at"]
	return out, nil
}

func partyReady(e map[string]any) error {
	if store.StrOrEmpty(e["status"]) != "READY" {
		return errcode.New(409, "EVENT_PHASE_CONFLICT", "现场开始后普通变更已冻结")
	}
	return nil
}

// reviewView renders a review joined with approved_size and decrypted reason.
func (svc *Service) reviewView(ctx context.Context, sc *store.Scope, rev map[string]any) (map[string]any, error) {
	v := store.Map(
		"reviewId", store.StrOrEmpty(rev["id"]),
		"partyId", store.StrOrEmpty(rev["party_id"]),
		"eventId", store.StrOrEmpty(rev["event_id"]),
		"kind", svc.value("ReviewKind", rev["kind"]),
		"requestedSize", rev["requested_size"],
		"approvedSize", rev["approved_size"],
		"requestedIncrease", rev["requested_increase"],
		"result", svc.value("ReviewResult", rev["result"]),
		"version", rev["version"],
		"createdAt", rev["created_at"],
	)
	reason, err := svc.S.Unseal(rev["reason_cipher"])
	if err != nil {
		return nil, err
	}
	for k, val := range reason {
		v[k] = val
	}
	return v, nil
}

// partyList renders the filtered, projected party page with asOf.
func (svc *Service) partyList(ctx context.Context, sc *store.Scope, ev string, r map[string]any, allowed map[string]bool) (map[string]any, error) {
	rows, err := svc.S.ListParties(ctx, sc, ev)
	if err != nil {
		return nil, err
	}
	result := []any{}
	for _, p := range rows {
		if _, has := r["status"]; has && store.StrOrEmpty(r["status"]) != store.StrOrEmpty(p["status"]) {
			continue
		}
		if _, has := r["sourceType"]; has && store.StrOrEmpty(r["sourceType"]) != store.StrOrEmpty(p["source_type"]) {
			continue
		}
		if _, has := r["progress"]; has && store.StrOrEmpty(r["progress"]) != partyProgress(p) {
			continue
		}
		created, _ := store.Time(p["created_at"])
		if _, has := r["createdFrom"]; has {
			from, err := store.Time(r["createdFrom"])
			if err == nil && created.Before(from) {
				continue
			}
		}
		if _, has := r["createdTo"]; has {
			to, err := store.Time(r["createdTo"])
			if err == nil && created.After(to) {
				continue
			}
		}
		if _, has := r["hasPendingIncrease"]; has {
			if r["hasPendingIncrease"] != (store.Num(p["pending_increase"]) > 0) {
				continue
			}
		}
		if _, has := r["query"]; has {
			q := store.StrOrEmpty(r["query"])
			f, err := svc.S.Unseal(p["fields_cipher"])
			if err != nil {
				return nil, err
			}
			match := strings.Contains(store.StrOrEmpty(p["party_reference"]), q)
			for key := range allowed {
				if fv, ok := f[key]; ok && strings.Contains(store.StrOrEmpty(fv), q) {
					match = true
				}
			}
			if !match {
				continue
			}
		}
		v, err := svc.partyView(ctx, sc, p, allowed)
		if err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	page := store.Page(result, r)
	page["asOf"] = svc.S.Now()
	return page, nil
}

// verifyPhone validates the single-use phone verification proof.
func (svc *Service) verifyPhone(ctx context.Context, sc *store.Scope, r map[string]any, c *Req, e map[string]any) error {
	if _, has := r["phoneVerificationProof"]; !has {
		return errcode.New(422, "PHONE_VERIFICATION_REQUIRED", "请先验证手机号码")
	}
	proof, err := svc.S.Crypto.Verify(store.StrOrEmpty(r["phoneVerificationProof"]))
	if err != nil {
		return errcode.New(422, "PHONE_VERIFICATION_REQUIRED", "手机验证已失效")
	}
	exp, perr := store.Time(proof["expiresAt"])
	if store.StrOrEmpty(e["id"]) != store.StrOrEmpty(proof["eventId"]) ||
		store.StrOrEmpty(c.Session["id"]) != store.StrOrEmpty(proof["contextId"]) ||
		perr != nil || !exp.After(svc.S.Now()) {
		return errcode.New(422, "PHONE_VERIFICATION_REQUIRED", "手机验证已失效")
	}
	challenge, err := svc.S.ChallengeByIDEventForUpdate(ctx, sc, store.StrOrEmpty(proof["challengeId"]), store.StrOrEmpty(e["id"]))
	if err != nil {
		return err
	}
	info, err := svc.S.Unseal(challenge["context_cipher"])
	if err != nil {
		return err
	}
	fields, _ := r["fields"].(map[string]any)
	phone := store.StrOrEmpty(fields["phone"])
	normalized := stripPhone(phone)
	if normalized != store.StrOrEmpty(proof["phone"]) || info["proofUsed"] == true {
		return errcode.New(422, "VERIFICATION_REQUIRED", "手机号验证无效或已使用")
	}
	info["proofUsed"] = true
	sealed, err := svc.S.Seal(sc, info)
	if err != nil {
		return err
	}
	return svc.S.UpdateChallengeContext(ctx, sc, sealed, store.StrOrEmpty(challenge["id"]))
}

var phoneStripRe = regexp.MustCompile(`[\s()\-\x0b]`)

func stripPhone(s string) string { return phoneStripRe.ReplaceAllString(s, "") }