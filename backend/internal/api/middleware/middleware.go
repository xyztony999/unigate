// Package middleware holds the HTTP cross-cutting concerns reproduced from
// ApiController.execute()'s header block, its generic catch (recovery) and
// the Spring multipart size limits surfaced by ApiErrors.
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Size limits mirror spring.servlet.multipart (max-request-size=11MB,
// max-file-size=10MB). The per-file cap is enforced by the api upload parser.
const (
	MaxRequestBytes = 11 << 20
	MaxFileBytes    = 10 << 20
)

type requestIDKey struct{}

// RequestID assigns every request a fresh UUID, exposes it on the response
// (X-Request-ID), the gin context and the request context so handlers and
// logs can correlate (ApiController.execute reqId = s.id()).
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := newID()
		c.Set("requestId", id)
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), requestIDKey{}, id))
		c.Writer.Header().Set("X-Request-ID", id)
		c.Next()
	}
}

// ID returns the request id stored by RequestID ("" when absent).
func ID(c *gin.Context) string {
	if v, ok := c.Get("requestId"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return FromContext(c.Request.Context())
}

// FromContext returns the request id carried by ctx ("" when absent).
func FromContext(ctx context.Context) string {
	if s, ok := ctx.Value(requestIDKey{}).(string); ok {
		return s
	}
	return ""
}

func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand never fails on supported platforms; fall back to a
		// zero UUID rather than dropping the request.
		return "00000000-0000-4000-8000-011111100000"
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// SecurityHeaders reproduces the fixed response headers set at the top of
// ApiController.execute() for every response, including errors.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("Cache-Control", "no-store")
		h.Set("Pragma", "no-cache")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("X-Content-Type-Options", "nosniff")
		c.Next()
	}
}

// Recovery converts a panic into the frozen 500 envelope without crashing
// the process (ApiController.execute generic Exception catch).
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			if rec == http.ErrAbortHandler {
				panic(rec) // net/http sentinel: let the server abort the connection
			}
			slog.ErrorContext(c.Request.Context(), "request panicked",
				"requestId", ID(c), "panic", fmt.Sprint(rec))
			if !c.Writer.Written() {
				AbortWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR",
					"服务暂不可用，请沿原操作编号核对结果")
			}
			c.Abort()
		}()
		c.Next()
	}
}

// BodyLimit caps the request body at the Spring max-request-size so oversized
// uploads fail with the frozen 422 envelope instead of buffering unbounded
// data. The api layer distinguishes multipart (422) from JSON (400) overflow.
func BodyLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxRequestBytes)
		}
		c.Next()
	}
}

// IsOversize reports whether err is the body-limit breach from BodyLimit.
func IsOversize(err error) bool {
	var mb *http.MaxBytesError
	return errors.As(err, &mb)
}

// AbortWithError writes the frozen error envelope with the real HTTP status
// and an empty object data (Java map()), then stops the chain.
func AbortWithError(c *gin.Context, status int, code, message string) {
	c.Writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	c.Writer.WriteHeader(status)
	body := map[string]any{
		"code":      code,
		"message":   message,
		"data":      map[string]any{},
		"requestId": ID(c),
	}
	if err := json.NewEncoder(c.Writer).Encode(body); err != nil {
		slog.ErrorContext(c.Request.Context(), "write error envelope failed",
			"requestId", ID(c), "error", err)
	}
	c.Abort()
}
