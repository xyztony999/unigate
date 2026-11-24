package service

import (
	"context"
	"fmt"
	"strings"

	"unigate/backend/internal/errcode"
	"unigate/backend/internal/store"
)

// handleParty dispatches enrollment operations (PartyService.handle).
func (svc *Service) handleParty(ctx context.Context, sc *store.Scope, op string, r map[string]any, c *Req, entry string) (any, error) {
	ev := store.StrOrEmpty(r["eventId"])
	if strings.HasPrefix(op, "My") {
		ev = store.StrOrEmpty(c.Session["event_id"])
	}
	switch op {
	case "SubmitRegistration":
		e, err := svc.S.Event(ctx, sc, ev)
		if err != nil {
			return nil, err
		}
		if !svc.registrationAvailable(e) {
			return nil, errcode.New(409, "REGISTRATION_CLOSED", "报名时间已关闭")
		}
		nv, err := svc.noticeVersion(e)
		if err != nil {
			return nil, err
		}
		if nv != store.StrOrEmpty(r["acceptedNoticeVersion"]) {
			return nil, errcode.New(409, "NOTICE_CHANGED", "活动告知已更新，请重新确认")
		}
		policy := store.Parse(e["policy"])
		if policy["phoneVerificationEnabled"] == true {
			if err := svc.verifyPhone(ctx, sc, r, c, e); err != nil {
				return nil, err
			}
		}
		fields, _ := r["fields"].(map[string]any)
		v, err := svc.partyCreate(ctx, sc, c, e, fields, "BUILTIN", store.StrOrEmpty(policy["approvalMode"]) == "IMMEDIATE", store.StrOrEmpty(r["acceptedNoticeVersion"]), entry)
		if err != nil {
			return nil, err
		}
		p, err := svc.partyRaw(ctx, sc, ev, store.StrOrEmpty(v["partyId"]))
		if err != nil {
			return nil, err
		}
		access, err := svc.partyAccess(ctx, sc, p, e)
		if err != nil {
			return nil, err
		}
		return store.Map("party", v, "access", access, "operationId", r["operationId"], "replayed", false), nil

	case "PartyExchange":
		q, err := svc.S.CredentialByToken(ctx, sc, svc.S.Crypto.Hash(store.StrOrEmpty(r["queryToken"])), "QUERY")
		if err != nil {
			return nil, err
		}
		if _, err := svc.S.Event(ctx, sc, store.StrOrEmpty(q["event_id"])); err != nil {
			return nil, err
		}
		ns, err := svc.newSession(ctx, sc, "PARTY_QUERY", "ug_party", nil, q["event_id"], q["party_id"], nil)
		if err != nil {
			return nil, err
		}
		if _, has := r["changeToken"]; has {
			if _, err := svc.S.ChangeCredentialByToken(ctx, sc, svc.S.Crypto.Hash(store.StrOrEmpty(r["changeToken"])), store.StrOrEmpty(q["party_id"])); err != nil {
				return nil, err
			}
			cs, err := svc.newSession(ctx, sc, "PARTY_CHANGE", "ug_change", nil, q["event_id"], q["party_id"], nil)
			if err != nil {
				return nil, err
			}
			csRow, _ := cs["row"].(map[string]any)
			if err := svc.S.UpdateSessionCsrf(ctx, sc, svc.S.Crypto.Hash(store.StrOrEmpty(ns["csrf"])), store.StrOrEmpty(csRow["id"])); err != nil {
				return nil, err
			}
		}
		ses, _ := ns["row"].(map[string]any)
		return store.Map("csrfToken", ns["csrf"], "expiresAt", ses["expires_at"], "clientContextId", store.StrOrEmpty(ses["id"])), nil

	case "MyParty", "MyCredential":
		e, err := svc.S.Event(ctx, sc, ev)
		if err != nil {
			return nil, err
		}
		p, err := svc.partyRaw(ctx, sc, ev, store.StrOrEmpty(c.Session["party_id"]))
		if err != nil {
			return nil, err
		}
		if op == "MyParty" {
			return svc.partyState(ctx, sc, p, e)
		}
		return svc.partyCredential(ctx, sc, p, e)

	case "PartyList":
		if err := c.View(ctx, ev); err != nil {
			return nil, err
		}
		if err := c.Action(ctx, ev, "VIEW_DETAIL"); err != nil {
			return nil, err
		}
		role, err := c.EventRole(ctx, ev)
		if err != nil {
			return nil, err
		}
		if role == "CHECKER" {
			return nil, errcode.New(403, "FORBIDDEN", "请使用辅助找票")
		}
		allowed, err := c.Fields(ctx, ev)
		if err != nil {
			return nil, err
		}
		return svc.partyList(ctx, sc, ev, r, allowed)

	case "PartyDetail":
		if err := c.View(ctx, ev); err != nil {
			return nil, err
		}
		if err := c.Action(ctx, ev, "VIEW_DETAIL"); err != nil {
			return nil, err
		}
		role, err := c.EventRole(ctx, ev)
		if err != nil {
			return nil, err
		}
		if role == "CHECKER" {
			return nil, errcode.New(403, "FORBIDDEN", "请使用辅助找票")
		}
		p, err := svc.partyRaw(ctx, sc, ev, store.StrOrEmpty(r["partyId"]))
		if err != nil {
			return nil, err
		}
		allowed, err := c.Fields(ctx, ev)
		if err != nil {
			return nil, err
		}
		return svc.partyView(ctx, sc, p, allowed)

	case "PartyChange", "MyPartyChange", "PartyCancel", "MyPartyCancel":
		return svc.partyChangeCancel(ctx, sc, op, ev, r, c)

	case "ReviewList":
		if err := c.Organizer(ctx, ev); err != nil {
			return nil, err
		}
		if _, err := svc.S.Event(ctx, sc, ev); err != nil {
			return nil, err
		}
		rows, err := svc.S.ListReviews(ctx, sc, ev)
		if err != nil {
			return nil, err
		}
		reviews := []any{}
		for _, x := range rows {
			if _, has := r["kind"]; has && store.StrOrEmpty(r["kind"]) != store.StrOrEmpty(x["kind"]) {
				continue
			}
			if _, has := r["result"]; has && store.StrOrEmpty(r["result"]) != store.StrOrEmpty(x["result"]) {
				continue
			}
			v, err := svc.reviewView(ctx, sc, x)
			if err != nil {
				return nil, err
			}
			reviews = append(reviews, v)
		}
		return store.Page(reviews, r), nil

	case "ReviewDecide":
		return svc.reviewDecide(ctx, sc, ev, r, c)

	case "CredentialRevoke", "CredentialReissue":
		return svc.credentialRevokeReissue(ctx, sc, op, ev, r, c)

	case "GateWalkIn":
		return svc.gateWalkIn(ctx, sc, ev, r, c, entry)

	case "GateIncrease":
		return svc.gateIncrease(ctx, sc, ev, r, c)

	default:
		return nil, fmt.Errorf("unknown party operation %q", op)
	}
}

func (svc *Service) partyChangeCancel(ctx context.Context, sc *store.Scope, op, ev string, r map[string]any, c *Req) (any, error) {
	e, err := svc.S.Event(ctx, sc, ev)
	if err != nil {
		return nil, err
	}
	if err := partyReady(e); err != nil {
		return nil, err
	}
	self := strings.HasPrefix(op, "My")
	if !self {
		if err := c.Organizer(ctx, ev); err != nil {
			return nil, err
		}
	}
	pid := store.StrOrEmpty(r["partyId"])
	if self {
		pid = store.StrOrEmpty(c.Session["party_id"])
	}
	p, err := svc.partyRaw(ctx, sc, ev, pid)
	if err != nil {
		return nil, err
	}
	if err := store.Version(p, r["version"]); err != nil {
		return nil, err
	}
	if p["ever_admitted"] == true {
		return nil, errcode.New(409, "ADMISSION_HISTORY_EXISTS", "已有入场记录，不允许普通变更")
	}
	status := store.StrOrEmpty(p["status"])
	if status != "PENDING" && status != "APPROVED" {
		return nil, errcode.New(409, "PARTY_STATE_CONFLICT", "当前资格不允许变更")
	}
	cancel := strings.HasSuffix(op, "Cancel")
	if self {
		key := "selfEditEnabled"
		if cancel {
			key = "selfCancelEnabled"
		}
		if !svc.canSelf(p, e, key) {
			return nil, errcode.New(403, "SELF_SERVICE_DISABLED", "自助入口已关闭")
		}
	}
	if cancel {
		if store.Num(p["approved_size"]) > 0 {
			if err := svc.allocate(ctx, sc, e, -store.Num(p["approved_size"])); err != nil {
				return nil, err
			}
		}
		if err := svc.S.CancelParty(ctx, sc, store.StrOrEmpty(p["id"])); err != nil {
			return nil, err
		}
		if err := svc.S.RevokeTicketChangeCredentials(ctx, sc, store.StrOrEmpty(p["id"])); err != nil {
			return nil, err
		}
		if err := svc.S.RejectPendingReviews(ctx, sc, store.StrOrEmpty(p["id"])); err != nil {
			return nil, err
		}
	} else {
		f, err := svc.S.Unseal(p["fields_cipher"])
		if err != nil {
			return nil, err
		}
		rule := store.Parse(store.Parse(store.Parse(e["policy"])["fields"])["companionCount"])
		if rule["enabled"] != true {
			return nil, errcode.New(422, "FIELD_DISABLED", "随行人数未启用")
		}
		n := store.Num(r["companionCount"]) + 1
		approved := store.Num(p["approved_size"])
		f["companionCount"] = r["companionCount"]
		if err := svc.S.RejectPendingReviews(ctx, sc, store.StrOrEmpty(p["id"])); err != nil {
			return nil, err
		}
		pending := 0
		newApproved := approved
		if status == "PENDING" {
			if err := svc.insertReview(ctx, sc, p, "INITIAL", n, n); err != nil {
				return nil, err
			}
		} else if n > approved && store.StrOrEmpty(store.Parse(e["policy"])["approvalMode"]) == "REVIEW" {
			pending = n - approved
			if err := svc.insertReview(ctx, sc, p, "INCREASE", n, pending); err != nil {
				return nil, err
			}
		} else {
			if err := svc.allocate(ctx, sc, e, n-approved); err != nil {
				return nil, err
			}
			newApproved = n
		}
		fieldsCipher, err := svc.S.Seal(sc, f)
		if err != nil {
			return nil, err
		}
		if err := svc.S.ChangeParty(ctx, sc, fieldsCipher, n, newApproved, pending, store.StrOrEmpty(p["id"])); err != nil {
			return nil, err
		}
	}
	p, err = svc.partyRaw(ctx, sc, ev, pid)
	if err != nil {
		return nil, err
	}
	if self {
		return svc.partyState(ctx, sc, p, e)
	}
	allowed, err := c.Fields(ctx, ev)
	if err != nil {
		return nil, err
	}
	return svc.partyView(ctx, sc, p, allowed)
}

func (svc *Service) reviewDecide(ctx context.Context, sc *store.Scope, ev string, r map[string]any, c *Req) (any, error) {
	if err := c.Organizer(ctx, ev); err != nil {
		return nil, err
	}
	e, err := svc.S.Event(ctx, sc, ev)
	if err != nil {
		return nil, err
	}
	if err := partyReady(e); err != nil {
		return nil, err
	}
	rev, err := svc.S.ReviewByIDForUpdate(ctx, sc, store.StrOrEmpty(r["reviewId"]), ev)
	if err != nil {
		return nil, err
	}
	if err := store.Version(rev, r["version"]); err != nil {
		return nil, err
	}
	if store.StrOrEmpty(rev["result"]) != "PENDING" {
		return nil, errcode.New(409, "REVIEW_STATE_CONFLICT", "已处理审核")
	}
	p, err := svc.partyRaw(ctx, sc, ev, store.StrOrEmpty(rev["party_id"]))
	if err != nil {
		return nil, err
	}
	approve := store.StrOrEmpty(r["decision"]) == "APPROVE"
	initial := store.StrOrEmpty(rev["kind"]) == "INITIAL"
	if approve {
		delta := store.Num(rev["requested_increase"])
		if initial {
			delta = store.Num(rev["requested_size"])
		}
		if err := svc.allocate(ctx, sc, e, delta); err != nil {
			return nil, err
		}
		if err := svc.S.ApproveParty(ctx, sc, delta, store.StrOrEmpty(p["id"])); err != nil {
			return nil, err
		}
		if initial {
			fresh, err := svc.partyRaw(ctx, sc, ev, store.StrOrEmpty(p["id"]))
			if err != nil {
				return nil, err
			}
			if _, err := svc.issueCapability(ctx, sc, fresh, "TICKET", e); err != nil {
				return nil, err
			}
		}
	} else {
		if err := svc.S.RejectParty(ctx, sc, initial, store.StrOrEmpty(p["id"])); err != nil {
			return nil, err
		}
	}
	reasonCipher, err := svc.S.Seal(sc, store.Map("reason", r["reason"]))
	if err != nil {
		return nil, err
	}
	result := "REJECTED"
	if approve {
		result = "APPROVED"
	}
	if err := svc.S.DecideReview(ctx, sc, result, store.StrOrEmpty(c.ActorID()), reasonCipher, store.StrOrEmpty(rev["id"])); err != nil {
		return nil, err
	}
	fresh, err := svc.partyRaw(ctx, sc, ev, store.StrOrEmpty(p["id"]))
	if err != nil {
		return nil, err
	}
	if err := svc.enqueueEmail(ctx, sc, fresh, e); err != nil {
		return nil, err
	}
	row, err := svc.S.ReviewViewRow(ctx, sc, store.StrOrEmpty(rev["id"]))
	if err != nil {
		return nil, err
	}
	return svc.reviewView(ctx, sc, row)
}

func (svc *Service) credentialRevokeReissue(ctx context.Context, sc *store.Scope, op, ev string, r map[string]any, c *Req) (any, error) {
	if err := c.Organizer(ctx, ev); err != nil {
		return nil, err
	}
	e, err := svc.S.Event(ctx, sc, ev)
	if err != nil {
		return nil, err
	}
	p, err := svc.partyRaw(ctx, sc, ev, store.StrOrEmpty(r["partyId"]))
	if err != nil {
		return nil, err
	}
	if err := store.Version(p, r["version"]); err != nil {
		return nil, err
	}
	if store.StrOrEmpty(p["status"]) != "APPROVED" {
		return nil, errcode.New(409, "PARTY_STATE_CONFLICT", "尚无有效资格")
	}
	if err := svc.S.RevokeTicketCredentials(ctx, sc, store.StrOrEmpty(p["id"])); err != nil {
		return nil, err
	}
	if err := svc.S.BumpPartyVersion(ctx, sc, store.StrOrEmpty(p["id"])); err != nil {
		return nil, err
	}
	p, err = svc.partyRaw(ctx, sc, ev, store.StrOrEmpty(p["id"]))
	if err != nil {
		return nil, err
	}
	allowed, err := c.Fields(ctx, ev)
	if err != nil {
		return nil, err
	}
	if op == "CredentialRevoke" {
		return svc.partyView(ctx, sc, p, allowed)
	}
	if _, err := svc.issueCapability(ctx, sc, p, "TICKET", e); err != nil {
		return nil, err
	}
	pv, err := svc.partyView(ctx, sc, p, allowed)
	if err != nil {
		return nil, err
	}
	cred, err := svc.partyCredential(ctx, sc, p, e)
	if err != nil {
		return nil, err
	}
	out := store.Map("party", pv, "credential", cred, "operationId", r["operationId"], "replayed", false)
	if r["rotateAccess"] == true {
		if err := svc.S.RevokeAccessCredentials(ctx, sc, store.StrOrEmpty(p["id"])); err != nil {
			return nil, err
		}
		if err := svc.S.RevokePartySessions(ctx, sc, store.StrOrEmpty(p["id"])); err != nil {
			return nil, err
		}
		if _, err := svc.issueCapability(ctx, sc, p, "QUERY", e); err != nil {
			return nil, err
		}
		if _, err := svc.issueCapability(ctx, sc, p, "CHANGE", e); err != nil {
			return nil, err
		}
		access, err := svc.partyAccess(ctx, sc, p, e)
		if err != nil {
			return nil, err
		}
		out["access"] = access
	}
	return out, nil
}

func (svc *Service) gateWalkIn(ctx context.Context, sc *store.Scope, ev string, r map[string]any, c *Req, entry string) (any, error) {
	if err := c.Organizer(ctx, ev); err != nil {
		return nil, err
	}
	e, err := svc.S.Event(ctx, sc, ev)
	if err != nil {
		return nil, err
	}
	if store.StrOrEmpty(e["status"]) != "CHECKIN_OPEN" {
		return nil, errcode.New(409, "EVENT_PHASE_CONFLICT", "检票未开放")
	}
	if store.Parse(e["policy"])["phoneVerificationEnabled"] == true {
		if err := svc.verifyPhone(ctx, sc, r, c, e); err != nil {
			return nil, err
		}
	}
	allowed, err := c.Fields(ctx, ev)
	if err != nil {
		return nil, err
	}
	if _, has := r["existingPendingPartyId"]; has {
		p, err := svc.partyRaw(ctx, sc, ev, store.StrOrEmpty(r["existingPendingPartyId"]))
		if err != nil {
			return nil, err
		}
		if err := store.Version(p, r["partyVersion"]); err != nil {
			return nil, err
		}
		if store.StrOrEmpty(p["status"]) != "PENDING" {
			return nil, errcode.New(409, "PARTY_STATE_CONFLICT", "不是待审核报名")
		}
		fields, _ := r["fields"].(map[string]any)
		if err := svc.validateFields(e, fields); err != nil {
			return nil, err
		}
		size := 1 + store.Num(fields["companionCount"])
		if err := svc.allocate(ctx, sc, e, size); err != nil {
			return nil, err
		}
		fieldsCipher, err := svc.S.Seal(sc, fields)
		if err != nil {
			return nil, err
		}
		if err := svc.S.WalkInParty(ctx, sc, fieldsCipher, size, store.StrOrEmpty(p["id"])); err != nil {
			return nil, err
		}
		reasonCipher, err := svc.S.Seal(sc, store.Map("reason", r["reason"]))
		if err != nil {
			return nil, err
		}
		if err := svc.S.ApprovePendingReviews(ctx, sc, store.StrOrEmpty(c.ActorID()), reasonCipher, store.StrOrEmpty(p["id"])); err != nil {
			return nil, err
		}
		p, err = svc.partyRaw(ctx, sc, ev, store.StrOrEmpty(p["id"]))
		if err != nil {
			return nil, err
		}
		if _, err := svc.issueCapability(ctx, sc, p, "TICKET", e); err != nil {
			return nil, err
		}
		return svc.partyView(ctx, sc, p, allowed)
	}
	fields, _ := r["fields"].(map[string]any)
	nv, err := svc.noticeVersion(e)
	if err != nil {
		return nil, err
	}
	return svc.partyCreate(ctx, sc, c, e, fields, "WALK_IN", true, nv, entry)
}

func (svc *Service) gateIncrease(ctx context.Context, sc *store.Scope, ev string, r map[string]any, c *Req) (any, error) {
	if err := c.Organizer(ctx, ev); err != nil {
		return nil, err
	}
	e, err := svc.S.Event(ctx, sc, ev)
	if err != nil {
		return nil, err
	}
	if store.StrOrEmpty(e["status"]) != "CHECKIN_OPEN" {
		return nil, errcode.New(409, "EVENT_PHASE_CONFLICT", "检票未开放")
	}
	p, err := svc.partyRaw(ctx, sc, ev, store.StrOrEmpty(r["partyId"]))
	if err != nil {
		return nil, err
	}
	if err := store.Version(p, r["version"]); err != nil {
		return nil, err
	}
	if store.StrOrEmpty(p["status"]) != "APPROVED" {
		return nil, errcode.New(409, "PARTY_STATE_CONFLICT", "需要有效资格")
	}
	count := store.Num(r["additionalCount"])
	if err := svc.allocate(ctx, sc, e, count); err != nil {
		return nil, err
	}
	if err := svc.S.IncreaseParty(ctx, sc, count, store.StrOrEmpty(p["id"])); err != nil {
		return nil, err
	}
	payloadJSON, err := store.Stringify(store.Map("count", count))
	if err != nil {
		return nil, err
	}
	if err := svc.S.InsertOutbox(ctx, sc, svc.S.ID(), ev, "OnsiteIncrease", store.StrOrEmpty(p["id"]), payloadJSON); err != nil {
		return nil, err
	}
	fresh, err := svc.partyRaw(ctx, sc, ev, store.StrOrEmpty(p["id"]))
	if err != nil {
		return nil, err
	}
	allowed, err := c.Fields(ctx, ev)
	if err != nil {
		return nil, err
	}
	return svc.partyView(ctx, sc, fresh, allowed)
}