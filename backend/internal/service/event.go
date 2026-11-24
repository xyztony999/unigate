package service

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"unigate/backend/internal/errcode"
	"unigate/backend/internal/store"
)

// eventDefaults reproduces EventService.defaults().
func eventDefaults() map[string]any {
	return store.Map(
		"fields", store.Map(
			"name", store.Map("enabled", true, "required", false),
			"phone", store.Map("enabled", true, "required", false),
			"email", store.Map("enabled", true, "required", false),
			"companionCount", store.Map("enabled", true, "required", false),
		),
		"approvalMode", "IMMEDIATE",
		"phoneVerificationEnabled", false,
		"selfEditEnabled", false,
		"selfCancelEnabled", false,
		"retentionDays", 30,
		"version", 0,
	)
}

// eventPolicy renders the policy with version and approvalMode value object.
func (svc *Service) eventPolicy(e map[string]any) map[string]any {
	p := store.Map()
	for k, v := range store.Parse(e["policy"]) {
		p[k] = v
	}
	p["version"] = e["policy_version"]
	p["approvalMode"] = svc.value("ApprovalMode", p["approvalMode"])
	return p
}

func (svc *Service) registrationAvailable(e map[string]any) bool {
	if store.StrOrEmpty(e["status"]) != "READY" {
		return false
	}
	starts, err1 := store.Time(e["registration_starts_at"])
	ends, err2 := store.Time(e["registration_ends_at"])
	if err1 != nil || err2 != nil {
		return false
	}
	now := svc.S.Now()
	return !now.Before(starts) && !now.After(ends)
}

// noticeVersion fingerprints the policy for the consent check. Java stringifies
// the raw PGobject(jsonb) as {"type":"jsonb","value":...,"null":false}; we
// reproduce that wrapper over the exact jsonb text so the token is identical.
func (svc *Service) noticeVersion(e map[string]any) (string, error) {
	raw, err := store.Stringify(e["policy"])
	if err != nil {
		return "", err
	}
	wrapped, err := store.PGObjectJSON(raw)
	if err != nil {
		return "", err
	}
	digest := svc.S.Crypto.Hash(store.StrOrEmpty(e["id"]) + wrapped)
	return fmt.Sprintf("v%v-%s", e["policy_version"], hex.EncodeToString(digest)[:24]), nil
}

func (svc *Service) eventView(e map[string]any) map[string]any {
	return store.Map(
		"eventId", store.StrOrEmpty(e["id"]),
		"title", e["title"],
		"venue", e["venue"],
		"startsAt", e["starts_at"],
		"endsAt", e["ends_at"],
		"timeZone", e["time_zone"],
		"registrationStartsAt", e["registration_starts_at"],
		"registrationEndsAt", e["registration_ends_at"],
		"capacity", e["capacity"],
		"allocatedCount", e["allocated_count"],
		"status", svc.value("EventStatus", e["status"]),
		"version", e["version"],
	)
}

// eventDates validates timezone and chronological order.
func eventDates(r map[string]any) error {
	if _, err := time.LoadLocation(store.StrOrEmpty(r["timeZone"])); err != nil {
		return errcode.New(422, "VALIDATION_FAILED", "时区无效")
	}
	rs, e1 := store.Time(r["registrationStartsAt"])
	re, e2 := store.Time(r["registrationEndsAt"])
	st, e3 := store.Time(r["startsAt"])
	en, e4 := store.Time(r["endsAt"])
	if e1 != nil || e2 != nil || e3 != nil || e4 != nil {
		return errcode.New(422, "VALIDATION_FAILED", "活动时间顺序无效")
	}
	if rs.After(re) || re.After(st) || st.After(en) {
		return errcode.New(422, "VALIDATION_FAILED", "活动时间顺序无效")
	}
	return nil
}

// handleEvent dispatches event operations (EventService.handle).
func (svc *Service) handleEvent(ctx context.Context, sc *store.Scope, op string, r map[string]any, c *Req, entry string) (any, error) {
	ev := store.StrOrEmpty(r["eventId"])
	switch op {
	case "EventCreate":
		if err := eventDates(r); err != nil {
			return nil, err
		}
		id := svc.S.ID()
		end, err := store.Time(r["endsAt"])
		if err != nil {
			return nil, errcode.New(422, "VALIDATION_FAILED", "活动时间顺序无效")
		}
		starts, _ := store.Time(r["startsAt"])
		regStarts, _ := store.Time(r["registrationStartsAt"])
		regEnds, _ := store.Time(r["registrationEndsAt"])
		defaultsJSON, err := store.Stringify(eventDefaults())
		if err != nil {
			return nil, err
		}
		if err := svc.S.InsertEvent(ctx, sc, id, store.StrOrEmpty(r["title"]), r["venue"], starts, end, store.StrOrEmpty(r["timeZone"]), regStarts, regEnds, r["capacity"], defaultsJSON, end.AddDate(0, 0, 30), store.StrOrEmpty(c.ActorID())); err != nil {
			return nil, err
		}
		if err := svc.S.InsertOrganizerGrant(ctx, sc, svc.S.ID(), id, store.StrOrEmpty(c.ActorID()), end.AddDate(0, 0, 395)); err != nil {
			return nil, err
		}
		// Seed the six scopes in the frozen CleanupScope pool order. Ranging the
		// enum map would randomize the order (Go map iteration), but the contract
		// requires CleanupView.scopes in pool order (Java iterates the ordered
		// enum keySet); cleanupView returns the stored array verbatim when it has
		// six entries, so the seeding order is what organizers first see.
		scopes := []any{}
		for _, k := range cleanupScopes {
			scopes = append(scopes, store.Map("scope", svc.value("CleanupScope", k), "status", svc.value("JobStatus", "PENDING"), "processedCount", 0, "failedCount", 0))
		}
		scopesJSON, err := store.Stringify(scopes)
		if err != nil {
			return nil, err
		}
		if err := svc.S.InsertCleanup(ctx, sc, svc.S.ID(), id, end.AddDate(0, 0, 30), scopesJSON); err != nil {
			return nil, err
		}
		e, err := svc.S.Event(ctx, sc, id)
		if err != nil {
			return nil, err
		}
		return svc.eventView(e), nil

	case "EventList", "EventOptions":
		rolesJSON, err := store.Stringify(store.List(c.Account["roles"]))
		if err != nil {
			return nil, err
		}
		rows, err := svc.S.ListEventsForAccount(ctx, sc, store.StrOrEmpty(c.ActorID()), rolesJSON)
		if err != nil {
			return nil, err
		}
		all := []any{}
		for _, e := range rows {
			if _, has := r["status"]; has && store.StrOrEmpty(r["status"]) != store.StrOrEmpty(e["status"]) {
				continue
			}
			if _, has := r["query"]; has && !strings.Contains(store.StrOrEmpty(e["title"]), store.StrOrEmpty(r["query"])) {
				continue
			}
			all = append(all, svc.eventView(e))
		}
		if op == "EventOptions" {
			return store.SelectorPage(all, 100, r)
		}
		return store.Page(all, r), nil

	case "PublicEvent":
		e, err := svc.S.Event(ctx, sc, ev)
		if err != nil {
			return nil, err
		}
		if store.StrOrEmpty(e["status"]) == "DRAFT" {
			return nil, errcode.New(404, "RESOURCE_NOT_FOUND", "未找到活动")
		}
		p := store.Parse(e["policy"])
		nv, err := svc.noticeVersion(e)
		if err != nil {
			return nil, err
		}
		notice := fmt.Sprintf("主办方按本活动设置收集报名信息，用于报名、检票和活动通知。个人明细于活动结束后%v天到期清理。请确认本次提交的信息及随行人数。", e["retention_days"])
		return store.Map(
			"eventId", ev,
			"title", e["title"],
			"venue", e["venue"],
			"startsAt", e["starts_at"],
			"endsAt", e["ends_at"],
			"timeZone", e["time_zone"],
			"fields", p["fields"],
			"approvalMode", svc.value("ApprovalMode", p["approvalMode"]),
			"phoneVerificationEnabled", p["phoneVerificationEnabled"],
			"registrationAvailable", svc.registrationAvailable(e),
			"notice", notice,
			"noticeVersion", nv,
			"retentionDays", e["retention_days"],
		), nil

	case "EventDetail":
		if err := c.View(ctx, ev); err != nil {
			return nil, err
		}
		e, err := svc.S.Event(ctx, sc, ev)
		if err != nil {
			return nil, err
		}
		return svc.eventView(e), nil

	case "PolicyGet":
		if err := c.Organizer(ctx, ev); err != nil {
			return nil, err
		}
		e, err := svc.S.Event(ctx, sc, ev)
		if err != nil {
			return nil, err
		}
		return svc.eventPolicy(e), nil

	case "EventUpdate":
		if err := c.Organizer(ctx, ev); err != nil {
			return nil, err
		}
		e, err := svc.S.Event(ctx, sc, ev)
		if err != nil {
			return nil, err
		}
		if err := store.Version(e, r["version"]); err != nil {
			return nil, err
		}
		status := store.StrOrEmpty(e["status"])
		if status != "DRAFT" && status != "READY" {
			return nil, errcode.New(409, "EVENT_PHASE_CONFLICT", "当前阶段不能修改活动")
		}
		if err := eventDates(r); err != nil {
			return nil, err
		}
		if store.Num(r["capacity"]) < store.Num(e["allocated_count"]) {
			return nil, errcode.New(409, "CAPACITY_CONFLICT", "容量不能少于已分配人数")
		}
		end, _ := store.Time(r["endsAt"])
		starts, _ := store.Time(r["startsAt"])
		regStarts, _ := store.Time(r["registrationStartsAt"])
		regEnds, _ := store.Time(r["registrationEndsAt"])
		retention := store.Num(e["retention_days"])
		if err := svc.S.UpdateEvent(ctx, sc, store.StrOrEmpty(r["title"]), r["venue"], starts, end, store.StrOrEmpty(r["timeZone"]), regStarts, regEnds, r["capacity"], end.AddDate(0, 0, retention), ev); err != nil {
			return nil, err
		}
		e, err = svc.S.Event(ctx, sc, ev)
		if err != nil {
			return nil, err
		}
		return svc.eventView(e), nil

	case "PolicyUpdate":
		return svc.policyUpdate(ctx, sc, ev, r, c)

	case "EventTransition":
		if err := c.Organizer(ctx, ev); err != nil {
			return nil, err
		}
		e, err := svc.S.Event(ctx, sc, ev)
		if err != nil {
			return nil, err
		}
		if err := store.Version(e, r["version"]); err != nil {
			return nil, err
		}
		status := store.StrOrEmpty(e["status"])
		action := store.StrOrEmpty(r["action"])
		var next string
		switch action {
		case "PUBLISH":
			if status != "DRAFT" {
				return nil, errcode.New(409, "EVENT_PHASE_CONFLICT", "不是草稿阶段")
			}
			next = "READY"
		case "OPEN":
			if status != "READY" {
				return nil, errcode.New(409, "EVENT_PHASE_CONFLICT", "不是准备阶段")
			}
			next = "CHECKIN_OPEN"
		case "PAUSE":
			if status != "CHECKIN_OPEN" {
				return nil, errcode.New(409, "EVENT_PHASE_CONFLICT", "不是检票开放阶段")
			}
			next = "CHECKIN_PAUSED"
		case "RESUME":
			if status != "CHECKIN_PAUSED" {
				return nil, errcode.New(409, "EVENT_PHASE_CONFLICT", "不是检票暂停阶段")
			}
			next = "CHECKIN_OPEN"
		case "END":
			if status != "CHECKIN_OPEN" && status != "CHECKIN_PAUSED" {
				return nil, errcode.New(409, "EVENT_PHASE_CONFLICT", "当前不能结束")
			}
			next = "ENDED"
		default:
			return nil, errcode.New(422, "VALIDATION_FAILED", "未知动作")
		}
		if err := svc.S.UpdateEventStatus(ctx, sc, next, ev); err != nil {
			return nil, err
		}
		e, err = svc.S.Event(ctx, sc, ev)
		if err != nil {
			return nil, err
		}
		return svc.eventView(e), nil

	default:
		return nil, fmt.Errorf("unknown event operation %q", op)
	}
}

func (svc *Service) policyUpdate(ctx context.Context, sc *store.Scope, ev string, r map[string]any, c *Req) (any, error) {
	if err := c.Organizer(ctx, ev); err != nil {
		return nil, err
	}
	e, err := svc.S.Event(ctx, sc, ev)
	if err != nil {
		return nil, err
	}
	status := store.StrOrEmpty(e["status"])
	if status != "DRAFT" && status != "READY" {
		return nil, errcode.New(409, "EVENT_PHASE_CONFLICT", "当前阶段不能修改规则")
	}
	p, _ := r["policy"].(map[string]any)
	if store.Num(p["version"]) != store.Num(e["policy_version"]) {
		return nil, errcode.New(409, "VERSION_CONFLICT", "规则已更新")
	}
	fields, _ := p["fields"].(map[string]any)
	for _, f := range fields {
		rule, _ := f.(map[string]any)
		if rule["required"] == true && rule["enabled"] != true {
			return nil, errcode.New(422, "VALIDATION_FAILED", "必填字段须启用")
		}
	}
	if p["phoneVerificationEnabled"] == true {
		phone, _ := fields["phone"].(map[string]any)
		if phone["enabled"] != true || phone["required"] != true {
			return nil, errcode.New(422, "VALIDATION_FAILED", "短信验证要求手机号必填")
		}
	}
	if _, has := p["selfServiceEndsAt"]; has {
		selfEnd, err := store.Time(p["selfServiceEndsAt"])
		regEnd, _ := store.Time(e["registration_ends_at"])
		if err != nil || selfEnd.After(regEnd) {
			return nil, errcode.New(422, "VALIDATION_FAILED", "自助截止不可晚于报名截止")
		}
	}
	old := store.Parse(e["policy"])
	count, err := svc.S.PartyCount(ctx, sc, ev)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		for _, key := range []string{"fields", "approvalMode", "phoneVerificationEnabled", "retentionDays"} {
			if !jsonEqual(old[key], p[key]) {
				return nil, errcode.New(409, "POLICY_IN_USE", "已有报名，不能修改已告知规则")
			}
		}
	}
	policyJSON, err := store.Stringify(p)
	if err != nil {
		return nil, err
	}
	end, _ := store.Time(e["ends_at"])
	retention := store.Num(p["retentionDays"])
	if err := svc.S.UpdateEventPolicy(ctx, sc, policyJSON, p["retentionDays"], end.AddDate(0, 0, retention), ev); err != nil {
		return nil, err
	}
	e, err = svc.S.Event(ctx, sc, ev)
	if err != nil {
		return nil, err
	}
	return svc.eventPolicy(e), nil
}

// jsonEqual compares two decoded JSON values structurally (Objects.equals on
// Jackson maps/lists/numbers). Numbers compare by numeric value.
func jsonEqual(a, b any) bool {
	switch av := a.(type) {
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for k, v := range av {
			if !jsonEqual(v, bv[k]) {
				return false
			}
		}
		return true
	case []any:
		bv, ok := b.([]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !jsonEqual(av[i], bv[i]) {
				return false
			}
		}
		return true
	default:
		if isNum(a) && isNum(b) {
			return store.Num(a) == store.Num(b) && numFloat(a) == numFloat(b)
		}
		return a == b
	}
}