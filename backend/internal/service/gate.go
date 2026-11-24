package service

import (
	"context"
	"crypto/subtle"
	"fmt"
	"strings"
	"time"

	"unigate/backend/internal/errcode"
	"unigate/backend/internal/store"
)

// gateScan validates a scan context token belongs to the current device.
func (svc *Service) gateScan(token string, c *Req, enforceExpiry bool) (map[string]any, error) {
	p, err := svc.S.Crypto.Verify(token)
	if err != nil {
		return nil, err
	}
	if store.StrOrEmpty(p["kind"]) != "SCAN" || store.StrOrEmpty(c.Device["id"]) != store.StrOrEmpty(p["deviceId"]) {
		return nil, errcode.New(403, "FORBIDDEN", "扫码上下文不属于当前终端")
	}
	if enforceExpiry {
		exp, perr := store.Time(p["expiresAt"])
		if perr != nil || !exp.After(svc.S.Now()) {
			return nil, errcode.New(410, "SCAN_EXPIRED", "请重新扫码")
		}
	}
	return p, nil
}

func (svc *Service) gateOpen(e map[string]any) error {
	if store.StrOrEmpty(e["status"]) != "CHECKIN_OPEN" {
		return errcode.New(409, "CHECKIN_CLOSED", "当前未开放检票")
	}
	end, err := store.Time(e["ends_at"])
	if err != nil || !end.After(svc.S.Now()) {
		return errcode.New(409, "CHECKIN_CLOSED", "活动已结束")
	}
	return nil
}

func (svc *Service) deviceView(d map[string]any) map[string]any {
	return store.Map(
		"deviceId", store.StrOrEmpty(d["id"]),
		"eventId", store.StrOrEmpty(d["event_id"]),
		"name", d["name"],
		"mode", svc.value("DeviceMode", d["mode"]),
		"status", svc.value("DeviceStatus", d["status"]),
		"expiresAt", d["expires_at"],
		"lastSeenAt", d["last_seen_at"],
		"version", d["version"],
	)
}

func (svc *Service) deviceSession(ctx context.Context, sc *store.Scope, d map[string]any) (map[string]any, error) {
	exp, _ := store.Time(d["expires_at"])
	csrf, err := svc.S.Crypto.Signed(store.Map("kind", "CSRF", "deviceId", store.StrOrEmpty(d["id"]), "expiresAt", store.FormatTime(exp)))
	if err != nil {
		return nil, err
	}
	e, err := svc.S.Event(ctx, sc, store.StrOrEmpty(d["event_id"]))
	if err != nil {
		return nil, err
	}
	return store.Map(
		"device", svc.deviceView(d),
		"csrfToken", csrf,
		"expiresAt", d["expires_at"],
		"eventStatus", svc.value("EventStatus", e["status"]),
	), nil
}

// handleGate dispatches gate/device operations (GateService.handle).
func (svc *Service) handleGate(ctx context.Context, sc *store.Scope, op string, r map[string]any, c *Req, entry string) (any, error) {
	ev := store.StrOrEmpty(r["eventId"])
	switch op {
	case "DeviceRegister":
		if err := c.Organizer(ctx, ev); err != nil {
			return nil, err
		}
		e, err := svc.S.Event(ctx, sc, ev)
		if err != nil {
			return nil, err
		}
		status := store.StrOrEmpty(e["status"])
		if status == "ENDED" || status == "ARCHIVED" {
			return nil, errcode.New(409, "EVENT_PHASE_CONFLICT", "活动已结束")
		}
		id := svc.S.ID()
		pair := svc.S.Crypto.Token()
		expiry := svc.S.Now().Add(12 * time.Hour)
		end, _ := store.Time(e["ends_at"])
		if end.Before(expiry) {
			expiry = end
		}
		if !expiry.After(svc.S.Now()) {
			return nil, errcode.New(409, "EVENT_PHASE_CONFLICT", "活动已过期")
		}
		pairingExpires := svc.S.Now().Add(5 * time.Minute)
		if err := svc.S.InsertDevice(ctx, sc, id, ev, store.StrOrEmpty(r["name"]), store.StrOrEmpty(r["mode"]), svc.S.Crypto.Hash(pair), pairingExpires, expiry, store.StrOrEmpty(c.ActorID())); err != nil {
			return nil, err
		}
		row, err := svc.S.DeviceByID(ctx, sc, id)
		if err != nil {
			return nil, err
		}
		return store.Map("device", svc.deviceView(row), "pairingCode", pair, "pairingExpiresAt", pairingExpires), nil

	case "DeviceClaim":
		d, err := svc.S.DeviceByID(ctx, sc, store.StrOrEmpty(r["deviceId"]))
		if err != nil {
			return nil, err
		}
		if _, err := svc.S.Event(ctx, sc, store.StrOrEmpty(d["event_id"])); err != nil {
			return nil, err
		}
		pairingExpires, _ := store.Time(d["pairing_expires_at"])
		if store.StrOrEmpty(d["status"]) != "UNCLAIMED" || !pairingExpires.After(svc.S.Now()) || store.Num(d["pairing_failures"]) >= 5 {
			return nil, errcode.New(410, "PAIRING_EXPIRED", "终端领取码已失效")
		}
		pairingDigest, _ := d["pairing_digest"].([]byte)
		if subtle.ConstantTimeCompare(pairingDigest, svc.S.Crypto.Hash(store.StrOrEmpty(r["pairingCode"]))) != 1 {
			if err := svc.failedPairing(ctx, store.StrOrEmpty(d["id"])); err != nil {
				return nil, err
			}
			return nil, errcode.New(401, "PAIRING_INVALID", "终端领取码无效")
		}
		d, err = svc.S.DeviceByIDForUpdate(ctx, sc, store.StrOrEmpty(d["id"]))
		if err != nil {
			return nil, err
		}
		if store.StrOrEmpty(d["status"]) != "UNCLAIMED" {
			return nil, errcode.New(410, "PAIRING_EXPIRED", "领取码已失效")
		}
		token := svc.S.Crypto.Token()
		if err := svc.S.ClaimDevice(ctx, sc, svc.S.Crypto.Hash(token), store.StrOrEmpty(d["id"])); err != nil {
			return nil, err
		}
		svc.cookie(sc, "ug_device", token)
		row, err := svc.S.DeviceByID(ctx, sc, store.StrOrEmpty(d["id"]))
		if err != nil {
			return nil, err
		}
		return svc.deviceSession(ctx, sc, row)

	case "DeviceDetail":
		return svc.deviceSession(ctx, sc, c.Device)

	case "DeviceList":
		if err := c.Organizer(ctx, ev); err != nil {
			return nil, err
		}
		if _, err := svc.S.Event(ctx, sc, ev); err != nil {
			return nil, err
		}
		rows, err := svc.S.ListDevices(ctx, sc, ev)
		if err != nil {
			return nil, err
		}
		all := []any{}
		for _, d := range rows {
			all = append(all, svc.deviceView(d))
		}
		return store.Page(all, r), nil

	case "DeviceRevoke":
		if err := c.Organizer(ctx, ev); err != nil {
			return nil, err
		}
		if _, err := svc.S.Event(ctx, sc, ev); err != nil {
			return nil, err
		}
		d, err := svc.S.DeviceByIDEventForUpdate(ctx, sc, store.StrOrEmpty(r["deviceId"]), ev)
		if err != nil {
			return nil, err
		}
		if err := store.Version(d, r["version"]); err != nil {
			return nil, err
		}
		if err := svc.S.RevokeDevice(ctx, sc, store.StrOrEmpty(d["id"])); err != nil {
			return nil, err
		}
		return store.Map(), nil

	case "TicketResolve":
		e, err := svc.S.Event(ctx, sc, ev)
		if err != nil {
			return nil, err
		}
		if err := svc.gateOpen(e); err != nil {
			return nil, err
		}
		cred, err := svc.S.CredentialByTokenEvent(ctx, sc, svc.S.Crypto.Hash(store.StrOrEmpty(r["ticketToken"])), ev)
		if err != nil {
			return nil, err
		}
		p, err := svc.partyRaw(ctx, sc, ev, store.StrOrEmpty(cred["party_id"]))
		if err != nil {
			return nil, err
		}
		if store.StrOrEmpty(p["status"]) != "APPROVED" {
			return nil, errcode.New(409, "PARTY_STATE_CONFLICT", "无有效入场资格")
		}
		expires := svc.S.Now().Add(5 * time.Minute)
		token, err := svc.S.Crypto.Signed(store.Map(
			"kind", "SCAN",
			"deviceId", store.StrOrEmpty(c.Device["id"]),
			"partyId", store.StrOrEmpty(p["id"]),
			"eventId", ev,
			"credentialId", store.StrOrEmpty(cred["id"]),
			"credentialVersion", cred["credential_version"],
			"expiresAt", store.FormatTime(expires),
		))
		if err != nil {
			return nil, err
		}
		return store.Map(
			"scanContextToken", token,
			"eventId", ev,
			"approvedSize", p["approved_size"],
			"netAdmitted", p["net_admitted"],
			"remainingCount", store.Num(p["approved_size"])-store.Num(p["net_admitted"]),
			"partyVersion", p["version"],
			"progress", svc.value("AdmissionProgress", partyProgress(p)),
			"expiresAt", expires,
		), nil

	case "GateCommit":
		return svc.gateCommit(ctx, sc, ev, r, c, entry)

	case "GateOperation":
		return svc.gateOperation(ctx, sc, r, c)

	case "GateLookup":
		return svc.gateLookup(ctx, sc, ev, r, c)

	case "GateRecords":
		return svc.gateRecords(ctx, sc, ev, r, c)

	case "GateCorrect":
		return svc.gateCorrect(ctx, sc, ev, r, c, entry)

	default:
		return nil, fmt.Errorf("unknown gate operation %q", op)
	}
}

func (svc *Service) gateCommit(ctx context.Context, sc *store.Scope, ev string, r map[string]any, c *Req, entry string) (any, error) {
	scan, err := svc.gateScan(store.StrOrEmpty(r["scanContextToken"]), c, true)
	if err != nil {
		return nil, err
	}
	if ev != store.StrOrEmpty(scan["eventId"]) {
		return nil, errcode.New(404, "RESOURCE_NOT_FOUND", "未找到资源")
	}
	e, err := svc.S.Event(ctx, sc, ev)
	if err != nil {
		return nil, err
	}
	if err := svc.gateOpen(e); err != nil {
		return nil, err
	}
	p, err := svc.partyRaw(ctx, sc, ev, store.StrOrEmpty(scan["partyId"]))
	if err != nil {
		return nil, err
	}
	if err := store.Version(p, r["expectedPartyVersion"]); err != nil {
		return nil, err
	}
	if _, err := svc.S.TicketCredentialForCommit(ctx, sc, store.StrOrEmpty(scan["credentialId"]), ev, store.StrOrEmpty(p["id"]), scan["credentialVersion"]); err != nil {
		return nil, err
	}
	count := store.Num(r["count"])
	if store.StrOrEmpty(p["status"]) != "APPROVED" || store.Num(p["net_admitted"])+count > store.Num(p["approved_size"]) {
		return nil, errcode.New(409, "ADMISSION_BALANCE_CONFLICT", "可入场名额不足")
	}
	if err := svc.S.AdmitParty(ctx, sc, count, store.StrOrEmpty(p["id"])); err != nil {
		return nil, err
	}
	id := svc.S.ID()
	now := svc.S.Now()
	if err := svc.S.InsertAdmission(ctx, sc, id, ev, store.StrOrEmpty(p["id"]), entry, store.StrOrEmpty(c.Device["id"]), c.ActorID(), count, now); err != nil {
		return nil, err
	}
	return store.Map(
		"operationId", r["operationId"],
		"recordId", id,
		"status", svc.value("OperationStatus", "SUCCEEDED"),
		"requestedCount", count,
		"committedCount", count,
		"remainingCount", store.Num(p["approved_size"])-store.Num(p["net_admitted"])-count,
		"committedAt", now,
		"replayed", false,
		"showAsNewAdmission", true,
	), nil
}

func (svc *Service) gateOperation(ctx context.Context, sc *store.Scope, r map[string]any, c *Req) (any, error) {
	ev := store.StrOrEmpty(c.Device["event_id"])
	if _, err := svc.S.Event(ctx, sc, ev); err != nil {
		return nil, err
	}
	var scan map[string]any
	if store.StrOrEmpty(c.Device["mode"]) == "KIOSK" {
		if _, has := r["scanContextToken"]; !has {
			return nil, errcode.New(403, "FORBIDDEN", "缺少原组上下文")
		}
		var err error
		scan, err = svc.gateScan(store.StrOrEmpty(r["scanContextToken"]), c, false)
		if err != nil {
			return nil, err
		}
	}
	ops, err := svc.S.GateOperations(ctx, sc, store.StrOrEmpty(r["operationId"]), ev)
	if err != nil {
		return nil, err
	}
	if len(ops) == 0 {
		return nil, errcode.New(404, "OPERATION_NOT_FOUND", "尚未查到原操作，请沿用原操作编号核对")
	}
	row := ops[0]
	if scan != nil {
		if store.StrOrEmpty(row["party_id"]) != store.StrOrEmpty(scan["partyId"]) || store.StrOrEmpty(row["device_id"]) != store.StrOrEmpty(c.Device["id"]) {
			return nil, errcode.New(404, "RESOURCE_NOT_FOUND", "未找到资源")
		}
	} else {
		role, err := c.EventRole(ctx, ev)
		if err != nil {
			return nil, err
		}
		if role != "ORGANIZER" {
			if store.StrOrEmpty(c.ActorID()) != store.StrOrEmpty(row["actor_id"]) && store.StrOrEmpty(c.Device["id"]) != store.StrOrEmpty(row["device_id"]) {
				return nil, errcode.New(404, "RESOURCE_NOT_FOUND", "未找到资源")
			}
		}
	}
	bundle, err := svc.S.Unseal(row["result_cipher"])
	if err != nil {
		return nil, err
	}
	result := store.Parse(bundle["data"])
	result["replayed"] = true
	result["showAsNewAdmission"] = false
	return result, nil
}

func (svc *Service) gateLookup(ctx context.Context, sc *store.Scope, ev string, r map[string]any, c *Req) (any, error) {
	if store.StrOrEmpty(c.Device["mode"]) != "STAFF" {
		return nil, errcode.New(403, "FORBIDDEN", "自助终端不能搜索名单")
	}
	if _, err := svc.S.Event(ctx, sc, ev); err != nil {
		return nil, err
	}
	allowed, err := c.Fields(ctx, ev)
	if err != nil {
		return nil, err
	}
	query := store.StrOrEmpty(r["query"])
	rows, err := svc.S.ListApprovedParties(ctx, sc, ev)
	if err != nil {
		return nil, err
	}
	candidates := []any{}
	for _, p := range rows {
		f, err := svc.S.Unseal(p["fields_cipher"])
		if err != nil {
			return nil, err
		}
		match := strings.Contains(store.StrOrEmpty(p["party_reference"]), query)
		for key := range allowed {
			if fv, ok := f[key]; ok && fv != nil && strings.Contains(store.StrOrEmpty(fv), query) {
				match = true
			}
		}
		if !match {
			continue
		}
		var nameMasked, phoneSuffix any
		if allowed["name"] {
			name := store.StrOrEmpty(f["name"])
			if len(name) > 0 {
				nameMasked = string([]rune(name)[:1]) + "**"
			}
		}
		if allowed["phone"] {
			phone := store.StrOrEmpty(f["phone"])
			runes := []rune(phone)
			start := len(runes) - 4
			if start < 0 {
				start = 0
			}
			phoneSuffix = string(runes[start:])
		}
		candidates = append(candidates, store.Map(
			"partyId", store.StrOrEmpty(p["id"]),
			"partyReference", p["party_reference"],
			"displayNameMasked", nameMasked,
			"phoneSuffix", phoneSuffix,
			"approvedSize", p["approved_size"],
			"netAdmitted", p["net_admitted"],
			"remainingCount", store.Num(p["approved_size"])-store.Num(p["net_admitted"]),
			"progress", svc.value("AdmissionProgress", partyProgress(p)),
		))
		if len(candidates) == 10 {
			break
		}
	}
	return store.Map("items", candidates, "total", len(candidates), "page", 1, "pageSize", 10), nil
}

func (svc *Service) gateRecords(ctx context.Context, sc *store.Scope, ev string, r map[string]any, c *Req) (any, error) {
	if err := c.GateHistory(ctx, ev); err != nil {
		return nil, err
	}
	role, err := c.EventRole(ctx, ev)
	if err != nil {
		return nil, err
	}
	lead := role == "ORGANIZER"
	rows, err := svc.S.ListAdmissions(ctx, sc, ev)
	if err != nil {
		return nil, err
	}
	result := []any{}
	for _, a := range rows {
		if !lead && store.StrOrEmpty(c.ActorID()) != store.StrOrEmpty(a["actor_id"]) &&
			(c.Device == nil || store.StrOrEmpty(c.Device["id"]) != store.StrOrEmpty(a["device_id"])) {
			continue
		}
		if _, has := r["deviceId"]; has && store.StrOrEmpty(r["deviceId"]) != store.StrOrEmpty(a["device_id"]) {
			continue
		}
		if _, has := r["partyId"]; has && store.StrOrEmpty(r["partyId"]) != store.StrOrEmpty(a["party_id"]) {
			continue
		}
		result = append(result, store.Map(
			"recordId", store.StrOrEmpty(a["id"]),
			"partyId", store.StrOrEmpty(a["party_id"]),
			"eventId", ev,
			"operationId", store.StrOrEmpty(a["operation_id"]),
			"deviceId", store.StrOrEmpty(a["device_id"]),
			// Kiosk admissions have no account: actor_id is NULL and the frozen
			// AdmissionView.actorId is an optional uuid, so omit it (Java
			// str(null) -> null -> map() skips) instead of emitting "".
			"actorId", store.Str(a["actor_id"]),
			"admittedCount", a["admitted_count"],
			"correctedCount", a["corrected_count"],
			"committedAt", a["committed_at"],
		))
	}
	return store.Page(result, r), nil
}

func (svc *Service) gateCorrect(ctx context.Context, sc *store.Scope, ev string, r map[string]any, c *Req, entry string) (any, error) {
	if err := c.Organizer(ctx, ev); err != nil {
		return nil, err
	}
	e, err := svc.S.Event(ctx, sc, ev)
	if err != nil {
		return nil, err
	}
	status := store.StrOrEmpty(e["status"])
	if status != "CHECKIN_OPEN" && status != "CHECKIN_PAUSED" {
		return nil, errcode.New(409, "EVENT_PHASE_CONFLICT", "当前不能更正")
	}
	a, err := svc.S.AdmissionByIDForUpdate(ctx, sc, store.StrOrEmpty(r["recordId"]), ev)
	if err != nil {
		return nil, err
	}
	p, err := svc.partyRaw(ctx, sc, ev, store.StrOrEmpty(a["party_id"]))
	if err != nil {
		return nil, err
	}
	if err := store.Version(p, r["partyVersion"]); err != nil {
		return nil, err
	}
	count := store.Num(r["correctedCount"])
	if store.Num(a["corrected_count"])+count > store.Num(a["admitted_count"]) || store.Num(p["net_admitted"]) < count {
		return nil, errcode.New(409, "CORRECTION_CONFLICT", "更正人数超过原记录")
	}
	if err := svc.S.AddCorrectedCount(ctx, sc, count, store.StrOrEmpty(a["id"])); err != nil {
		return nil, err
	}
	if err := svc.S.CorrectParty(ctx, sc, count, store.StrOrEmpty(p["id"])); err != nil {
		return nil, err
	}
	id := svc.S.ID()
	reasonCipher, err := svc.S.Seal(sc, store.Map("reason", r["reason"]))
	if err != nil {
		return nil, err
	}
	if err := svc.S.InsertCorrection(ctx, sc, id, ev, store.StrOrEmpty(p["id"]), store.StrOrEmpty(a["id"]), count, reasonCipher, store.StrOrEmpty(c.ActorID()), entry); err != nil {
		return nil, err
	}
	return store.Map(
		"correctionId", id,
		"originalRecordId", r["recordId"],
		"partyId", store.StrOrEmpty(p["id"]),
		"correctedCount", count,
		"reason", r["reason"],
		"createdAt", svc.S.Now(),
		"operationId", r["operationId"],
	), nil
}