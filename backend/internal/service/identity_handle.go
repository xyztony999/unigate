package service

import (
	"context"
	"fmt"
	"strings"

	"unigate/backend/internal/errcode"
	"unigate/backend/internal/store"
)

// handleIdentity dispatches the identity operations (IdentityService.handle).
func (svc *Service) handleIdentity(ctx context.Context, sc *store.Scope, op string, r map[string]any, c *Req, entry string) (any, error) {
	switch op {
	case "Login":
		accounts, err := svc.S.AccountByUsernameEnabled(ctx, sc, store.StrOrEmpty(r["username"]))
		if err != nil {
			return nil, err
		}
		if err := svc.checkLogin(ctx, sc, store.StrOrEmpty(r["username"]), c.ClientIP); err != nil {
			return nil, err
		}
		if len(accounts) == 0 || !svc.Passwords.Matches(store.StrOrEmpty(r["password"]), store.StrOrEmpty(accounts[0]["password_hash"])) {
			if err := svc.failedLogin(ctx, store.StrOrEmpty(r["username"]), c.ClientIP); err != nil {
				return nil, err
			}
			return nil, errcode.New(401, "AUTHENTICATION_FAILED", "用户名或密码错误")
		}
		c.Account = accounts[0]
		ns, err := svc.newSession(ctx, sc, "STAFF", "ug_staff", c.ActorID(), nil, nil, c.Account["auth_version"])
		if err != nil {
			return nil, err
		}
		c.Session, _ = ns["row"].(map[string]any)
		return svc.sessionView(c, store.StrOrEmpty(ns["csrf"])), nil

	case "SessionMe":
		return svc.sessionView(c, svc.csrfFor(store.StrOrEmpty(c.Session["id"]))), nil

	case "Logout":
		if err := svc.S.RevokeSession(ctx, sc, store.StrOrEmpty(c.Session["id"])); err != nil {
			return nil, err
		}
		svc.cookie(sc, "ug_staff", "")
		return store.Map(), nil

	case "PasswordChange":
		if !svc.Passwords.Matches(store.StrOrEmpty(r["oldPassword"]), store.StrOrEmpty(c.Account["password_hash"])) {
			return nil, errcode.New(401, "AUTHENTICATION_FAILED", "当前密码错误")
		}
		hash, err := svc.Passwords.Encode(store.StrOrEmpty(r["newPassword"]))
		if err != nil {
			return nil, fmt.Errorf("hash new password: %w", err)
		}
		id := store.StrOrEmpty(c.Account["id"])
		if err := svc.S.UpdateAccountPassword(ctx, sc, hash, id); err != nil {
			return nil, err
		}
		if err := svc.S.RevokeAccountSessions(ctx, sc, id); err != nil {
			return nil, err
		}
		svc.cookie(sc, "ug_staff", "")
		return store.Map(), nil

	case "AccountList":
		if err := c.RequireAdmin(); err != nil {
			return nil, err
		}
		rows, err := svc.S.ListAccounts(ctx, sc)
		if err != nil {
			return nil, err
		}
		q := store.StrOrEmpty(r["query"])
		all := []any{}
		for _, a := range rows {
			if strings.Contains(store.StrOrEmpty(a["username"]), q) || strings.Contains(store.StrOrEmpty(a["display_name"]), q) {
				all = append(all, svc.accountView(a))
			}
		}
		return store.Page(all, r), nil

	case "AccountCreate":
		if err := c.RequireAdmin(); err != nil {
			return nil, err
		}
		id := svc.S.ID()
		hash, err := svc.Passwords.Encode(store.StrOrEmpty(r["temporaryPassword"]))
		if err != nil {
			return nil, fmt.Errorf("hash temporary password: %w", err)
		}
		rolesJSON, err := store.Stringify(r["roles"])
		if err != nil {
			return nil, err
		}
		if err := svc.S.InsertAccount(ctx, sc, id, store.StrOrEmpty(r["username"]), store.StrOrEmpty(r["displayName"]), hash, rolesJSON); err != nil {
			return nil, err
		}
		row, err := svc.S.AccountByID(ctx, sc, id)
		if err != nil {
			return nil, err
		}
		return svc.accountView(row), nil

	case "AccountUpdate":
		if err := c.RequireAdmin(); err != nil {
			return nil, err
		}
		if err := svc.S.AccountAdvisoryLock(ctx, sc); err != nil {
			return nil, err
		}
		a, err := svc.S.AccountByIDForUpdate(ctx, sc, store.StrOrEmpty(r["accountId"]))
		if err != nil {
			return nil, err
		}
		if err := store.Version(a, r["version"]); err != nil {
			return nil, err
		}
		active := r["enabled"] == true
		roles := store.List(r["roles"])
		if (!active || !containsStr(roles, "ADMIN")) && containsStr(store.List(a["roles"]), "ADMIN") {
			count, err := svc.S.EnabledAdminCount(ctx, sc)
			if err != nil {
				return nil, err
			}
			if count <= 1 {
				return nil, errcode.New(409, "LAST_ADMIN", "不能禁用最后一个管理员")
			}
		}
		rolesJSON, err := store.Stringify(roles)
		if err != nil {
			return nil, err
		}
		if err := svc.S.UpdateAccountMeta(ctx, sc, store.StrOrEmpty(r["displayName"]), rolesJSON, active, store.StrOrEmpty(a["id"])); err != nil {
			return nil, err
		}
		row, err := svc.S.AccountByID(ctx, sc, store.StrOrEmpty(a["id"]))
		if err != nil {
			return nil, err
		}
		return svc.accountView(row), nil

	case "GrantEventOptions":
		if err := c.RequireAdmin(); err != nil {
			return nil, err
		}
		rows, err := svc.S.GrantEventOptions(ctx, sc)
		if err != nil {
			return nil, err
		}
		all := []any{}
		for _, e := range rows {
			if _, has := r["query"]; has && !strings.Contains(store.StrOrEmpty(e["title"]), store.StrOrEmpty(r["query"])) {
				continue
			}
			all = append(all, store.Map("eventId", store.StrOrEmpty(e["id"]), "title", e["title"], "startsAt", e["starts_at"]))
		}
		return store.SelectorPage(all, 100, r)

	case "MyGrantList", "GrantList":
		ev := store.StrOrEmpty(r["eventId"])
		if op == "GrantList" && !c.Admin() {
			if err := c.Organizer(ctx, ev); err != nil {
				return nil, err
			}
		}
		rows, err := svc.S.ListGrantsJoined(ctx, sc)
		if err != nil {
			return nil, err
		}
		list := []any{}
		for _, g := range rows {
			if op == "MyGrantList" {
				if !containsStr(store.List(c.Account["roles"]), store.StrOrEmpty(g["role"])) {
					continue
				}
				if _, has := r["eventId"]; has && store.StrOrEmpty(r["eventId"]) != store.StrOrEmpty(g["event_id"]) {
					continue
				}
			}
			if (op == "MyGrantList" && store.StrOrEmpty(c.ActorID()) == store.StrOrEmpty(g["account_id"])) ||
				(op == "GrantList" && ev == store.StrOrEmpty(g["event_id"])) {
				v, err := svc.grantView(ctx, sc, g)
				if err != nil {
					return nil, err
				}
				list = append(list, v)
			}
		}
		return store.Page(list, r), nil

	case "GrantSave":
		return svc.grantSave(ctx, sc, r, c)

	case "GrantRevoke":
		return svc.grantRevoke(ctx, sc, r, c)

	default:
		return nil, fmt.Errorf("unknown identity operation %q", op)
	}
}

func (svc *Service) grantSave(ctx context.Context, sc *store.Scope, r map[string]any, c *Req) (any, error) {
	ev := store.StrOrEmpty(r["eventId"])
	if _, err := svc.S.Event(ctx, sc, ev); err != nil {
		return nil, err
	}
	target, err := svc.S.AccountByID(ctx, sc, store.StrOrEmpty(r["accountId"]))
	if err != nil {
		return nil, err
	}
	if !containsStr(store.List(target["roles"]), store.StrOrEmpty(r["role"])) {
		return nil, errcode.New(422, "FIELD_VALIDATION_FAILED", "目标账号缺少对应角色")
	}
	actions := store.List(r["allowedActions"])
	for _, x := range actions {
		if !containsStr([]any{"VIEW_DETAIL", "VIEW_STATISTICS", "CHECKIN", "EXPORT"}, store.StrOrEmpty(x)) {
			return nil, errcode.New(422, "FIELD_VALIDATION_FAILED", "无效授权动作")
		}
	}
	fields := store.List(r["allowedFields"])
	if !c.Admin() {
		if err := c.Organizer(ctx, ev); err != nil {
			return nil, err
		}
		if store.StrOrEmpty(r["role"]) == "ORGANIZER" {
			return nil, errcode.New(403, "FORBIDDEN", "只有管理员可授予负责人")
		}
		own, err := c.Fields(ctx, ev)
		if err != nil {
			return nil, err
		}
		for _, f := range fields {
			if !own[store.StrOrEmpty(f)] {
				return nil, errcode.New(403, "FORBIDDEN", "字段超出自身授权")
			}
		}
		if containsStr(actions, "EXPORT") {
			return nil, errcode.New(403, "FORBIDDEN", "导出需管理员单独授权")
		}
	}
	role := store.StrOrEmpty(r["role"])
	if role == "READER" {
		for _, key := range []string{"recipient", "purpose", "retentionNote"} {
			if _, ok := r[key]; !ok {
				return nil, errcode.New(422, "VALIDATION_FAILED", "缺少接收方用途信息")
			}
		}
	}
	if role == "CHECKER" && containsStr(actions, "EXPORT") {
		return nil, errcode.New(422, "VALIDATION_FAILED", "检票员不能导出")
	}
	if role == "READER" {
		for _, x := range actions {
			if !containsStr([]any{"VIEW_DETAIL", "VIEW_STATISTICS"}, store.StrOrEmpty(x)) {
				return nil, errcode.New(422, "VALIDATION_FAILED", "校方仅可查看")
			}
		}
	}
	expires, err := store.Time(r["expiresAt"])
	if err != nil {
		return nil, errcode.New(422, "FIELD_VALIDATION_FAILED", "授权期限必须在未来")
	}
	if !expires.After(svc.S.Now()) {
		return nil, errcode.New(422, "FIELD_VALIDATION_FAILED", "授权期限必须在未来")
	}
	if !c.Admin() {
		own, err := c.Grant(ctx, ev)
		if err != nil {
			return nil, err
		}
		ownExpires, err := store.Time(own["expires_at"])
		if err != nil {
			return nil, fmt.Errorf("grant expires: %w", err)
		}
		if expires.After(ownExpires) {
			return nil, errcode.New(403, "FORBIDDEN", "授权期限超出自身范围")
		}
		ownActions := store.List(own["allowed_actions"])
		for _, x := range actions {
			if !containsStr(ownActions, store.StrOrEmpty(x)) {
				return nil, errcode.New(403, "FORBIDDEN", "动作超出自身授权")
			}
		}
	}
	id := store.StrOrEmpty(r["grantId"])
	if _, has := r["grantId"]; has {
		old, err := svc.S.GrantByIDForUpdate(ctx, sc, id, ev)
		if err != nil {
			return nil, err
		}
		if err := store.Version(old, r["version"]); err != nil {
			return nil, err
		}
		if store.StrOrEmpty(old["account_id"]) != store.StrOrEmpty(r["accountId"]) || store.StrOrEmpty(old["role"]) != role {
			return nil, errcode.New(422, "FIELD_VALIDATION_FAILED", "既有授权不能更换账号或角色")
		}
	} else {
		id = svc.S.ID()
		existing, err := svc.S.GrantExists(ctx, sc, ev, store.StrOrEmpty(r["accountId"]), role)
		if err != nil {
			return nil, err
		}
		if len(existing) > 0 {
			return nil, errcode.New(409, "VERSION_CONFLICT", "授权已存在，请按原版本更新")
		}
	}
	fieldsJSON, err := store.Stringify(fields)
	if err != nil {
		return nil, err
	}
	actionsJSON, err := store.Stringify(actions)
	if err != nil {
		return nil, err
	}
	disclosure, err := svc.S.Seal(sc, store.Map("recipient", r["recipient"], "purpose", r["purpose"], "retentionNote", r["retentionNote"]))
	if err != nil {
		return nil, err
	}
	if err := svc.S.UpsertGrant(ctx, sc, id, ev, store.StrOrEmpty(r["accountId"]), role, fieldsJSON, actionsJSON, disclosure, expires); err != nil {
		return nil, err
	}
	row, err := svc.S.GrantViewRow(ctx, sc, ev, store.StrOrEmpty(r["accountId"]), role)
	if err != nil {
		return nil, err
	}
	return svc.grantView(ctx, sc, row)
}

func (svc *Service) grantRevoke(ctx context.Context, sc *store.Scope, r map[string]any, c *Req) (any, error) {
	ev := store.StrOrEmpty(r["eventId"])
	if _, err := svc.S.Event(ctx, sc, ev); err != nil {
		return nil, err
	}
	if !c.Admin() {
		if err := c.Organizer(ctx, ev); err != nil {
			return nil, err
		}
	}
	g, err := svc.S.GrantByIDForUpdate(ctx, sc, store.StrOrEmpty(r["grantId"]), ev)
	if err != nil {
		return nil, err
	}
	if err := store.Version(g, r["version"]); err != nil {
		return nil, err
	}
	if !c.Admin() && store.StrOrEmpty(g["role"]) == "ORGANIZER" {
		return nil, errcode.New(403, "FORBIDDEN", "不能撤销负责人")
	}
	if store.StrOrEmpty(g["role"]) == "ORGANIZER" {
		count, err := svc.S.ActiveOrganizerCount(ctx, sc, ev)
		if err != nil {
			return nil, err
		}
		if count <= 1 {
			return nil, errcode.New(409, "LAST_ORGANIZER", "不能撤销最后一位负责人")
		}
	}
	if err := svc.S.RevokeGrant(ctx, sc, store.StrOrEmpty(g["id"])); err != nil {
		return nil, err
	}
	return store.Map(), nil
}