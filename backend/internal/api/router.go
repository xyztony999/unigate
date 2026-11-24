// Package api is the HTTP entry layer: it registers the frozen contract
// operations on a gin engine and reproduces ApiController.execute()'s request
// parsing, transaction boundary, authentication call and response envelope.
// Business logic stays in internal/service; SQL stays in internal/store.
package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"unigate/backend/internal/api/middleware"
	"unigate/backend/internal/config"
	"unigate/backend/internal/service"
)

// handler is the thin transport shell shared by every operation route.
type handler struct {
	svc *service.Service
}

// NewRouter builds the http.Handler for all contract operations. main passes
// the assembled Service; cfg is accepted for transport-level settings symmetry
// with the Java controller wiring. Every operation path/method pair from the
// frozen x-operation-index is registered exactly once; unknown paths and
// methods answer with the same 404 envelope as a missing contract route.
func NewRouter(svc *service.Service, cfg *config.Config) (http.Handler, error) {
	if svc == nil || svc.C == nil {
		return nil, fmt.Errorf("api router requires a service with a loaded contract")
	}
	_ = cfg // transport settings currently live in service.Config (cookies/origin)
	gin.SetMode(gin.ReleaseMode)
	e := gin.New()
	// The frontend fetch uses redirect:'error'; never issue redirects.
	e.RedirectTrailingSlash = false
	e.RedirectFixedPath = false
	// Only trust the direct peer address for ClientIP (no proxy header spoofing).
	if err := e.SetTrustedProxies(nil); err != nil {
		return nil, fmt.Errorf("configure trusted proxies: %w", err)
	}
	e.Use(
		middleware.RequestID(),
		middleware.SecurityHeaders(),
		middleware.Recovery(),
		middleware.BodyLimit(),
	)
	h := &handler{svc: svc}
	seen := make(map[string]bool, len(svc.C.Operations))
	for _, op := range svc.C.Operations {
		method := strings.ToUpper(op.Method)
		if method != http.MethodGet && method != http.MethodPost {
			return nil, fmt.Errorf("operation %s: unsupported method %q", op.Name, op.Method)
		}
		key := method + " " + op.Path
		if seen[key] {
			return nil, fmt.Errorf("duplicate contract route %s", key)
		}
		seen[key] = true
		e.Handle(method, op.Path, h.execute(op))
	}
	e.NoRoute(h.notFound)
	e.NoMethod(h.notFound)
	return e, nil
}

// notFound reproduces the contract.routes.get(...) == null branch: a real 404
// with the frozen envelope, not gin's default plain-text page.
func (h *handler) notFound(c *gin.Context) {
	middleware.AbortWithError(c, http.StatusNotFound, "RESOURCE_NOT_FOUND", "未找到接口")
}
