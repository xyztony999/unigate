// Package service holds the business logic reproduced from the Java services.
// It is HTTP-agnostic: handlers bind input, call these methods and wrap the
// frozen response envelope. SQL lives only in the store layer.
package service

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"unigate/backend/internal/config"
	"unigate/backend/internal/contract"
	"unigate/backend/internal/errcode"
	"unigate/backend/internal/store"
)

// HTTPDoer is the transport used for provider/SMS calls (injectable for tests).
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// Service aggregates the dependencies shared by all business operations,
// mirroring the Spring-injected Store/Contract/Environment singletons.
type Service struct {
	S         *store.Store
	C         *contract.Contract
	Cfg       *config.Config
	Passwords store.PasswordHasher
	HTTP      HTTPDoer
}

// New wires a Service with a no-redirect HTTP client (provider safety).
func New(s *store.Store, c *contract.Contract, cfg *config.Config) *Service {
	return &Service{
		S: s, C: c, Cfg: cfg, Passwords: store.PasswordHasher{},
		HTTP: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// Req is the per-request security context (com.unigate.Context). It carries the
// resolved account/session/device and the scope used for sealing and cookies.
type Req struct {
	Account  map[string]any
	Session  map[string]any
	Device   map[string]any
	Auth     string
	ClientIP string
	Sc       *store.Scope
	S        *store.Store
	C        *contract.Contract
	// Info keeps the transport-derived inputs so Dispatch can serve
	// SessionBootstrap without a second authenticate pass.
	Info RequestInfo
}

// ActorID returns the account id or nil.
func (r *Req) ActorID() any {
	if r.Account == nil {
		return nil
	}
	return r.Account["id"]
}

// Admin reports whether the account holds the ADMIN role.
func (r *Req) Admin() bool {
	if r.Account == nil {
		return false
	}
	return containsStr(store.List(r.Account["roles"]), "ADMIN")
}

func containsStr(list []any, want string) bool {
	for _, x := range list {
		if store.StrOrEmpty(x) == want {
			return true
		}
	}
	return false
}

// Grant resolves the caller's highest-priority live grant for an event.
func (r *Req) Grant(ctx context.Context, eventID string) (map[string]any, error) {
	if r.Account == nil {
		return nil, errcode.New(401, "SESSION_EXPIRED", "请先登录")
	}
	rows, err := r.S.EventGrantsForAccount(ctx, r.Sc, eventID, store.StrOrEmpty(r.ActorID()))
	if err != nil {
		return nil, err
	}
	roles := store.List(r.Account["roles"])
	for _, g := range rows {
		if containsStr(roles, store.StrOrEmpty(g["role"])) {
			return g, nil
		}
	}
	return nil, errcode.New(404, "RESOURCE_NOT_FOUND", "未找到资源")
}

// EventRole returns the caller's role for an event.
func (r *Req) EventRole(ctx context.Context, eventID string) (string, error) {
	g, err := r.Grant(ctx, eventID)
	if err != nil {
		return "", err
	}
	return store.StrOrEmpty(g["role"]), nil
}

// Organizer requires the ORGANIZER role.
func (r *Req) Organizer(ctx context.Context, eventID string) error {
	role, err := r.EventRole(ctx, eventID)
	if err != nil {
		return err
	}
	if role != "ORGANIZER" {
		return errcode.New(403, "FORBIDDEN", "需要活动负责人权限")
	}
	return nil
}

// View requires view permission and loads the event under lock.
func (r *Req) View(ctx context.Context, eventID string) error {
	g, err := r.Grant(ctx, eventID)
	if err != nil {
		return err
	}
	allowed := store.List(g["allowed_actions"])
	ok := containsStr(allowed, "VIEW_DETAIL") || containsStr(allowed, "VIEW_STATISTICS") || store.StrOrEmpty(g["role"]) == "ORGANIZER"
	if !ok {
		return errcode.New(403, "FORBIDDEN", "没有查看权限")
	}
	_, err = r.S.Event(ctx, r.Sc, eventID)
	return err
}

// Action requires a specific allowed action on the grant.
func (r *Req) Action(ctx context.Context, eventID, action string) error {
	g, err := r.Grant(ctx, eventID)
	if err != nil {
		return err
	}
	if !containsStr(store.List(g["allowed_actions"]), action) {
		return errcode.New(403, "FORBIDDEN", "当前授权不包含该操作")
	}
	return nil
}

// GateHistory authorizes gate history access (checker needs CHECKIN).
func (r *Req) GateHistory(ctx context.Context, eventID string) error {
	role, err := r.EventRole(ctx, eventID)
	if err != nil {
		return err
	}
	if role == "CHECKER" {
		if err := r.Action(ctx, eventID, "CHECKIN"); err != nil {
			return err
		}
	} else if err := r.Organizer(ctx, eventID); err != nil {
		return err
	}
	_, err = r.S.Event(ctx, r.Sc, eventID)
	return err
}

// Fields returns the allowed field set for the caller's grant.
func (r *Req) Fields(ctx context.Context, eventID string) (map[string]bool, error) {
	g, err := r.Grant(ctx, eventID)
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, f := range store.List(g["allowed_fields"]) {
		out[store.StrOrEmpty(f)] = true
	}
	return out, nil
}

// RequireAdmin requires the system ADMIN role.
func (r *Req) RequireAdmin() error {
	if !r.Admin() {
		return errcode.New(403, "FORBIDDEN", "需要系统管理员权限")
	}
	return nil
}

// value renders an enum {code,label} using the contract pool.
func (svc *Service) value(typ string, code any) map[string]any {
	return svc.S.Value(svc.C.Enums, typ, code)
}

func isNum(v any) bool {
	switch v.(type) {
	case int, int32, int64, float64, json.Number:
		return true
	}
	return false
}

func numFloat(v any) float64 {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	case float64:
		return n
	case json.Number:
		f, _ := n.Float64()
		return f
	}
	return 0
}