package service

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"unigate/backend/internal/contract"
	"unigate/backend/internal/errcode"
	"unigate/backend/internal/store"
)

// RequestInfo carries the HTTP-derived inputs authenticate needs, keeping the
// service layer free of transport types.
type RequestInfo struct {
	Cookies         map[string]string
	Origin          string
	CSRF            string
	ClientContextID string
	ClientIP        string
	ServerOrigin    string
}

func (r RequestInfo) cookie(name string) string { return r.Cookies[name] }

// cookie appends a Set-Cookie header reproducing Spring ResponseCookie order:
// name=value; Path=/; Max-Age=43200; Secure; HttpOnly; SameSite=...
func (svc *Service) cookie(sc *store.Scope, name, token string) {
	var b strings.Builder
	b.WriteString(name)
	b.WriteByte('=')
	b.WriteString(token)
	b.WriteString("; Path=/")
	b.WriteString("; Max-Age=43200")
	if svc.Cfg.CookieSecure {
		b.WriteString("; Secure")
	}
	b.WriteString("; HttpOnly")
	if name == "ug_staff" {
		b.WriteString("; SameSite=Lax")
	} else {
		b.WriteString("; SameSite=Strict")
	}
	sc.Cookies = append(sc.Cookies, b.String())
}

// session resolves a live session row from a named cookie.
func (svc *Service) session(ctx context.Context, sc *store.Scope, info RequestInfo, name string) (map[string]any, error) {
	token := info.cookie(name)
	if token == "" {
		return nil, nil
	}
	rows, err := svc.S.SessionByTokenDigest(ctx, sc, svc.S.Crypto.Hash(token))
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

// newSession creates a session, sets its cookie and returns {row, csrf}.
func (svc *Service) newSession(ctx context.Context, sc *store.Scope, kind, cookieName string, account, event, party any, authVersion any) (map[string]any, error) {
	token := svc.S.Crypto.Token()
	id := svc.S.ID()
	csrf := svc.csrfFor(id)
	expires := svc.S.Now().Add(12 * time.Hour)
	if event != nil {
		e, err := svc.S.Event(ctx, sc, store.StrOrEmpty(event))
		if err != nil {
			return nil, err
		}
		due, err := store.Time(e["personal_due_at"])
		if err != nil {
			return nil, fmt.Errorf("session event due: %w", err)
		}
		if due.Before(expires) {
			expires = due
		}
	}
	now := svc.S.Now()
	if err := svc.S.InsertSession(ctx, sc, id, svc.S.Crypto.Hash(token), kind, account, event, party, authVersion, svc.S.Crypto.Hash(csrf), now, now, expires); err != nil {
		return nil, err
	}
	svc.cookie(sc, cookieName, token)
	row, err := svc.S.SessionByID(ctx, sc, id)
	if err != nil {
		return nil, err
	}
	return store.Map("row", row, "csrf", csrf), nil
}

// Bootstrap establishes or reuses the anonymous client context.
func (svc *Service) Bootstrap(ctx context.Context, sc *store.Scope, info RequestInfo) (map[string]any, error) {
	row, err := svc.session(ctx, sc, info, "ug_anon")
	if err != nil {
		return nil, err
	}
	var csrf string
	if row == nil {
		created, err := svc.newSession(ctx, sc, "ANONYMOUS", "ug_anon", nil, nil, nil, nil)
		if err != nil {
			return nil, err
		}
		row, _ = created["row"].(map[string]any)
		csrf = store.StrOrEmpty(created["csrf"])
	} else {
		csrf = svc.csrfFor(store.StrOrEmpty(row["id"]))
	}
	return store.Map(
		"csrfToken", csrf,
		"clientContextId", store.StrOrEmpty(row["id"]),
		"expiresAt", row["expires_at"],
	), nil
}

// csrfFor derives the deterministic CSRF token for a session id.
func (svc *Service) csrfFor(sessionID string) string {
	return base64.RawURLEncoding.EncodeToString(svc.S.Crypto.Hash("csrf:v1:" + sessionID))
}

// validCsrf accepts either the stored random digest or the deterministic token.
func (svc *Service) validCsrf(session map[string]any, csrf string) bool {
	if session == nil || csrf == "" {
		return false
	}
	digest, _ := session["csrf_digest"].([]byte)
	if subtle.ConstantTimeCompare(svc.S.Crypto.Hash(csrf), digest) == 1 {
		return true
	}
	expected := svc.csrfFor(store.StrOrEmpty(session["id"]))
	return subtle.ConstantTimeCompare([]byte(csrf), []byte(expected)) == 1
}

// touchStaffSession refreshes the staff session sliding window.
func (svc *Service) touchStaffSession(ctx context.Context, sc *store.Scope, info RequestInfo) error {
	token := info.cookie("ug_staff")
	if token == "" {
		return nil
	}
	return svc.S.TouchStaffSession(ctx, sc, svc.S.Crypto.Hash(token))
}

// TouchStaffSession refreshes the staff session sliding window outside the
// request transaction (ApiController.execute calls identity.touchStaffSession
// before tx.execute). It is the exported entry point for the api layer.
func (svc *Service) TouchStaffSession(ctx context.Context, info RequestInfo) error {
	if err := svc.touchStaffSession(ctx, &store.Scope{}, info); err != nil {
		return fmt.Errorf("touch staff session: %w", err)
	}
	return nil
}

// Authenticate resolves the security context for an operation, reproducing
// IdentityService.authenticate (five roles, Origin/CSRF, client-context binding).
func (svc *Service) Authenticate(ctx context.Context, sc *store.Scope, op *contract.Operation, r map[string]any, info RequestInfo) (*Req, error) {
	c := &Req{Sc: sc, S: svc.S, C: svc.C, ClientIP: info.ClientIP, Auth: op.Auth, Info: info}
	staff, err := svc.session(ctx, sc, info, "ug_staff")
	if err != nil {
		return nil, err
	}
	if staff != nil {
		lastSeen, err := store.Time(staff["last_seen_at"])
		if err != nil {
			return nil, fmt.Errorf("staff last seen: %w", err)
		}
		if lastSeen.Before(svc.S.Now().Add(-30 * time.Minute)) {
			staff = nil
		}
	}
	if staff != nil {
		accounts, err := svc.S.AccountByIDEnabledAuthVersion(ctx, sc, store.StrOrEmpty(staff["account_id"]), staff["auth_version"])
		if err != nil {
			return nil, err
		}
		if len(accounts) > 0 {
			c.Account = accounts[0]
			c.Session = staff
		}
	}
	deviceToken := info.cookie("ug_device")
	if deviceToken != "" {
		ds, err := svc.S.DeviceBySessionDigest(ctx, sc, svc.S.Crypto.Hash(deviceToken))
		if err != nil {
			return nil, err
		}
		if len(ds) > 0 {
			c.Device = ds[0]
		}
	}
	switch c.Auth {
	case "ANON":
		if op.Name == "ConnectionCallback" {
			if c.Account == nil {
				return nil, errcode.New(401, "SESSION_EXPIRED", "授权发起会话已失效")
			}
			break
		}
		c.Session, err = svc.session(ctx, sc, info, "ug_anon")
		if err != nil {
			return nil, err
		}
		if op.Name != "SessionBootstrap" && c.Session == nil {
			return nil, errcode.New(401, "SESSION_EXPIRED", "请重新打开页面")
		}
	case "STAFF":
		if c.Account == nil {
			return nil, errcode.New(401, "SESSION_EXPIRED", "请先登录")
		}
		if c.Account["must_change_password"] == true {
			if !containsStr([]any{"SessionMe", "PasswordChange", "Logout"}, op.Name) {
				return nil, errcode.New(403, "PASSWORD_CHANGE_REQUIRED", "请先修改初始密码")
			}
		}
		if !anyRoleMatches(store.List(c.Account["roles"]), op.Roles) {
			return nil, errcode.New(403, "FORBIDDEN", "没有操作权限")
		}
	case "PARTY", "CHANGE":
		name := "ug_party"
		if c.Auth == "CHANGE" {
			name = "ug_change"
		}
		c.Session, err = svc.session(ctx, sc, info, name)
		if err != nil {
			return nil, err
		}
		if c.Session == nil {
			return nil, errcode.New(401, "SESSION_EXPIRED", "私密访问已失效")
		}
		if _, err := svc.S.Event(ctx, sc, store.StrOrEmpty(c.Session["event_id"])); err != nil {
			return nil, err
		}
	case "DEVICE", "GATE":
		if c.Device == nil {
			return nil, errcode.New(401, "DEVICE_UNAUTHORIZED", "请先授权终端")
		}
		if store.StrOrEmpty(c.Device["mode"]) == "STAFF" && c.Account == nil {
			return nil, errcode.New(401, "SESSION_EXPIRED", "工作人员终端需要登录")
		}
		if _, ok := r["eventId"]; ok {
			if store.StrOrEmpty(c.Device["event_id"]) != store.StrOrEmpty(r["eventId"]) {
				return nil, errcode.New(404, "RESOURCE_NOT_FOUND", "未找到资源")
			}
		}
		if store.StrOrEmpty(c.Device["mode"]) == "STAFF" {
			role, err := c.EventRole(ctx, store.StrOrEmpty(c.Device["event_id"]))
			if err != nil {
				return nil, err
			}
			if role != "ORGANIZER" && role != "CHECKER" {
				return nil, errcode.New(403, "FORBIDDEN", "没有检票权限")
			}
			if err := c.Action(ctx, store.StrOrEmpty(c.Device["event_id"]), "CHECKIN"); err != nil {
				return nil, err
			}
		}
	default:
		return nil, fmt.Errorf("unknown auth role %q", c.Auth)
	}
	if strings.EqualFold(op.Method, "post") {
		if info.Origin == "" || (info.Origin != svc.Cfg.AllowedOrigin && info.Origin != info.ServerOrigin) {
			return nil, errcode.New(403, "ORIGIN_REJECTED", "来源验证失败")
		}
		csrf := info.CSRF
		csrfOk := svc.validCsrf(c.Session, csrf)
		if !csrfOk && csrf != "" && (c.Auth == "PARTY" || c.Auth == "CHANGE") {
			anon, err := svc.session(ctx, sc, info, "ug_anon")
			if err != nil {
				return nil, err
			}
			csrfOk = svc.validCsrf(anon, csrf)
		}
		if !csrfOk && csrf != "" && c.Device != nil {
			payload, err := svc.S.Crypto.Verify(csrf)
			if err == nil {
				exp, perr := store.Time(payload["expiresAt"])
				csrfOk = store.StrOrEmpty(c.Device["id"]) == store.StrOrEmpty(payload["deviceId"]) &&
					payload["kind"] == "CSRF" && perr == nil && exp.After(svc.S.Now())
			}
		}
		if !csrfOk {
			return nil, errcode.New(403, "CSRF_REJECTED", "页面验证已失效，请刷新")
		}
		if c.Auth == "ANON" {
			if store.StrOrEmpty(c.Session["id"]) != info.ClientContextID {
				return nil, errcode.New(401, "SESSION_EXPIRED", "原匿名上下文已失效")
			}
		}
	}
	if op.Name == "RegistrationResult" {
		if store.StrOrEmpty(c.Session["id"]) != info.ClientContextID {
			return nil, errcode.New(401, "SESSION_EXPIRED", "原匿名上下文已失效")
		}
	}
	return c, nil
}

func anyRoleMatches(accountRoles []any, opRoles []string) bool {
	for _, ar := range accountRoles {
		for _, or := range opRoles {
			if store.StrOrEmpty(ar) == or {
				return true
			}
		}
	}
	return false
}

// BootstrapAccount creates the initial admin when no accounts exist.
func (svc *Service) BootstrapAccount(ctx context.Context) error {
	sc := &store.Scope{}
	count, err := svc.S.AccountCount(ctx, sc)
	if err != nil {
		return err
	}
	if count != 0 {
		return nil
	}
	if len(svc.Cfg.BootstrapPassword) < 12 {
		return errcode.New(500, "CONFIGURATION_REQUIRED", "Set UNIGATE_BOOTSTRAP_PASSWORD with at least 12 characters")
	}
	hash, err := svc.Passwords.Encode(svc.Cfg.BootstrapPassword)
	if err != nil {
		return fmt.Errorf("hash bootstrap password: %w", err)
	}
	return svc.S.InsertAccount(ctx, sc, svc.S.ID(), svc.Cfg.BootstrapUsername, "系统管理员", hash, "[\"ADMIN\",\"ORGANIZER\"]")
}

// ResetToTemporaryPassword issues a random temporary password, forces a change,
// bumps auth_version and revokes all sessions (password reset feature).
func (svc *Service) ResetToTemporaryPassword(ctx context.Context, username string) (string, error) {
	sc := &store.Scope{}
	accounts, err := svc.S.AccountByUsernameEnabled(ctx, sc, username)
	if err != nil {
		return "", err
	}
	if len(accounts) == 0 {
		return "", errcode.New(404, "RESOURCE_NOT_FOUND", "账号不存在或已禁用")
	}
	temporary := svc.S.Crypto.Token()
	hash, err := svc.Passwords.Encode(temporary)
	if err != nil {
		return "", fmt.Errorf("hash temporary password: %w", err)
	}
	id := store.StrOrEmpty(accounts[0]["id"])
	if err := svc.S.UpdateAccountReset(ctx, sc, hash, id); err != nil {
		return "", err
	}
	if err := svc.S.RevokeAccountSessions(ctx, sc, id); err != nil {
		return "", err
	}
	return temporary, nil
}

// loginScope hashes username:ip for the login-limit challenge scope.
func (svc *Service) loginScope(username, ip string) string {
	return "LOGIN:" + hex.EncodeToString(svc.S.Crypto.Hash(username+":"+ip))
}

// sessionView renders the staff session response.
func (svc *Service) sessionView(c *Req, csrf string) map[string]any {
	roles := []any{}
	for _, x := range store.List(c.Account["roles"]) {
		roles = append(roles, svc.value("Role", x))
	}
	return store.Map(
		"accountId", c.ActorID(),
		"username", c.Account["username"],
		"displayName", c.Account["display_name"],
		"roles", roles,
		"csrfToken", csrf,
		"expiresAt", c.Session["expires_at"],
		"mustChangePassword", c.Account["must_change_password"],
	)
}

func (svc *Service) accountView(a map[string]any) map[string]any {
	roles := []any{}
	for _, x := range store.List(a["roles"]) {
		roles = append(roles, svc.value("Role", x))
	}
	return store.Map(
		"accountId", store.StrOrEmpty(a["id"]),
		"username", a["username"],
		"displayName", a["display_name"],
		"roles", roles,
		"enabled", a["enabled"],
		"version", a["version"],
	)
}

func (svc *Service) grantView(ctx context.Context, sc *store.Scope, g map[string]any) (map[string]any, error) {
	fields := []any{}
	for _, x := range store.List(g["allowed_fields"]) {
		fields = append(fields, svc.value("PartyField", x))
	}
	m := store.Map(
		"grantId", store.StrOrEmpty(g["id"]),
		"accountId", store.StrOrEmpty(g["account_id"]),
		"eventId", store.StrOrEmpty(g["event_id"]),
		"role", svc.value("Role", g["role"]),
		"allowedFields", fields,
		"allowedActions", store.List(g["allowed_actions"]),
		"expiresAt", g["expires_at"],
		"version", g["version"],
		"accountDisplayName", g["display_name"],
		"username", g["username"],
	)
	disclosure, err := svc.S.Unseal(g["disclosure_cipher"])
	if err != nil {
		return nil, err
	}
	for k, v := range disclosure {
		m[k] = v
	}
	return m, nil
}
