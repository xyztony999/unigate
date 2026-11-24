package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"unigate/backend/internal/contract"
	"unigate/backend/internal/service"
)

// TestRouterRegistersAllOperations checks every frozen contract operation is
// registered exactly once with its method/path, and unknown routes answer
// with the frozen 404 envelope and security headers.
func TestRouterRegistersAllOperations(t *testing.T) {
	c, err := contract.Load()
	if err != nil {
		t.Fatalf("load contract: %v", err)
	}
	svc := &service.Service{C: c}
	h, err := NewRouter(svc, nil)
	if err != nil {
		t.Fatalf("new router: %v", err)
	}
	engine, ok := h.(*gin.Engine)
	if !ok {
		t.Fatalf("router is %T, want *gin.Engine", h)
	}
	routes := engine.Routes()
	if len(routes) != len(c.Operations) {
		t.Fatalf("registered %d routes, contract has %d operations", len(routes), len(c.Operations))
	}
	got := make(map[string]bool, len(routes))
	for _, r := range routes {
		got[r.Method+" "+r.Path] = true
	}
	for _, op := range c.Operations {
		key := strings.ToUpper(op.Method) + " " + op.Path
		if !got[key] {
			t.Errorf("missing route %s (%s)", key, op.Name)
		}
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/does/not/exist", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown route status = %d, want 404", rec.Code)
	}
	var body struct {
		Code      string         `json:"code"`
		Message   string         `json:"message"`
		Data      map[string]any `json:"data"`
		RequestID string         `json:"requestId"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if body.Code != "RESOURCE_NOT_FOUND" || body.Data == nil || body.RequestID == "" {
		t.Fatalf("bad 404 envelope: %+v", body)
	}
	if rec.Header().Get("X-Request-ID") != body.RequestID {
		t.Fatalf("X-Request-ID header %q != envelope requestId %q", rec.Header().Get("X-Request-ID"), body.RequestID)
	}
	for k, want := range map[string]string{
		"Cache-Control":          "no-store",
		"Pragma":                 "no-cache",
		"Referrer-Policy":        "no-referrer",
		"X-Content-Type-Options": "nosniff",
	} {
		if got := rec.Header().Get(k); got != want {
			t.Errorf("header %s = %q, want %q", k, got, want)
		}
	}
}
