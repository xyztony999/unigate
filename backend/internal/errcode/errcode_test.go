package errcode

import (
	"errors"
	"fmt"
	"testing"
)

// TestCanonicalMapping reproduces com.unigate.ApiException.canonical. Each row
// pins the wire code (and the IDEMPOTENCY_PAYLOAD_MISMATCH status override).
func TestCanonicalMapping(t *testing.T) {
	cases := []struct {
		status     int
		code       string
		wantCode   string
		wantStatus int
	}{
		// Named aliases.
		{409, "IDEMPOTENCY_CONFLICT", "IDEMPOTENCY_PAYLOAD_MISMATCH", 400},
		{409, "CAPACITY_CONFLICT", "CAPACITY_EXCEEDED", 409},
		{409, "ADMISSION_BALANCE_CONFLICT", "PARTY_REMAINING_EXCEEDED", 409},
		{403, "CSRF_REJECTED", "CSRF_INVALID", 403},
		{403, "ORIGIN_REJECTED", "CSRF_INVALID", 403},
		{401, "AUTHENTICATION_FAILED", "UNAUTHENTICATED", 401},
		{401, "DEVICE_UNAUTHORIZED", "UNAUTHENTICATED", 401},
		{401, "PAIRING_INVALID", "UNAUTHENTICATED", 401},
		{422, "VALIDATION_FAILED", "FIELD_VALIDATION_FAILED", 422},
		{422, "FIELD_DISABLED", "FIELD_VALIDATION_FAILED", 422},
		{422, "INVALID_FILE", "FIELD_VALIDATION_FAILED", 422},
		{422, "INVALID_ENCODING", "FIELD_VALIDATION_FAILED", 422},
		{422, "INVALID_PROVIDER", "FIELD_VALIDATION_FAILED", 422},
		{403, "PHONE_VERIFICATION_REQUIRED", "VERIFICATION_REQUIRED", 403},
		{410, "PAIRING_EXPIRED", "TOKEN_EXPIRED", 410},
		{410, "CAPABILITY_EXPIRED", "TOKEN_EXPIRED", 410},
		{410, "SCAN_EXPIRED", "TOKEN_EXPIRED", 410},
		{410, "DATA_EXPIRED", "TOKEN_EXPIRED", 410},
		{410, "RESULT_EXPIRED", "TOKEN_EXPIRED", 410},
		{410, "BATCH_EXPIRED", "TOKEN_EXPIRED", 410},
		{409, "REGISTRATION_CLOSED", "EVENT_STATE_CONFLICT", 409},
		{409, "EVENT_PHASE_CONFLICT", "EVENT_STATE_CONFLICT", 409},
		{409, "PARTY_STATE_CONFLICT", "EVENT_STATE_CONFLICT", 409},
		{409, "REVIEW_STATE_CONFLICT", "EVENT_STATE_CONFLICT", 409},
		{409, "LAST_ADMIN", "EVENT_STATE_CONFLICT", 409},
		{409, "LAST_ORGANIZER", "EVENT_STATE_CONFLICT", 409},
		{409, "CORRECTION_CONFLICT", "EVENT_STATE_CONFLICT", 409},
		{409, "ADMISSION_HISTORY_EXISTS", "EVENT_STATE_CONFLICT", 409},
		{409, "POLICY_IN_USE", "EVENT_STATE_CONFLICT", 409},
		{409, "NOTICE_CHANGED", "EVENT_STATE_CONFLICT", 409},
		{409, "CHECKIN_CLOSED", "EVENT_STATE_CONFLICT", 409},
		{409, "DATA_CONFLICT", "EVENT_STATE_CONFLICT", 409},
		{403, "PASSWORD_CHANGE_REQUIRED", "FORBIDDEN", 403},
		{403, "SELF_SERVICE_DISABLED", "FORBIDDEN", 403},

		// Known codes pass through unchanged.
		{401, "UNAUTHENTICATED", "UNAUTHENTICATED", 401},
		{401, "SESSION_EXPIRED", "SESSION_EXPIRED", 401},
		{404, "RESOURCE_NOT_FOUND", "RESOURCE_NOT_FOUND", 404},
		{404, "OPERATION_NOT_FOUND", "OPERATION_NOT_FOUND", 404},
		{409, "VERSION_CONFLICT", "VERSION_CONFLICT", 409},
		{409, "OPERATION_IN_PROGRESS", "OPERATION_IN_PROGRESS", 409},
		{410, "TOKEN_EXPIRED", "TOKEN_EXPIRED", 410},
		{410, "ARTIFACT_EXPIRED", "ARTIFACT_EXPIRED", 410},
		{422, "FIELD_VALIDATION_FAILED", "FIELD_VALIDATION_FAILED", 422},
		{429, "RATE_LIMITED", "RATE_LIMITED", 429},
		{503, "SERVICE_UNAVAILABLE", "SERVICE_UNAVAILABLE", 503},
		{503, "PROVIDER_UNAVAILABLE", "PROVIDER_UNAVAILABLE", 503},
		{400, "IDEMPOTENCY_PAYLOAD_MISMATCH", "IDEMPOTENCY_PAYLOAD_MISMATCH", 400},
		// Direct known code still forces 400 regardless of the supplied status.
		{409, "IDEMPOTENCY_PAYLOAD_MISMATCH", "IDEMPOTENCY_PAYLOAD_MISMATCH", 400},

		// Unknown codes fall back by status.
		{400, "SOMETHING_ELSE", "INVALID_REQUEST", 400},
		{401, "SOMETHING_ELSE", "UNAUTHENTICATED", 401},
		{403, "SOMETHING_ELSE", "FORBIDDEN", 403},
		{404, "SOMETHING_ELSE", "RESOURCE_NOT_FOUND", 404},
		{409, "SOMETHING_ELSE", "EVENT_STATE_CONFLICT", 409},
		{410, "SOMETHING_ELSE", "ARTIFACT_EXPIRED", 410},
		{422, "SOMETHING_ELSE", "FIELD_VALIDATION_FAILED", 422},
		{429, "SOMETHING_ELSE", "RATE_LIMITED", 429},
		{503, "SOMETHING_ELSE", "SERVICE_UNAVAILABLE", 503},
		{500, "SOMETHING_ELSE", "INTERNAL_ERROR", 500},
		{502, "SOMETHING_ELSE", "INTERNAL_ERROR", 502},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%d/%s", tc.status, tc.code), func(t *testing.T) {
			e := New(tc.status, tc.code, "消息")
			if e.Code != tc.wantCode {
				t.Fatalf("Code = %s, want %s", e.Code, tc.wantCode)
			}
			if e.Status != tc.wantStatus {
				t.Fatalf("Status = %d, want %d", e.Status, tc.wantStatus)
			}
			if e.Message != "消息" || e.Error() != "消息" {
				t.Fatalf("Message = %q, want 消息", e.Message)
			}
		})
	}
}

func TestRequire(t *testing.T) {
	if e := Require(true, 422, "VALIDATION_FAILED", "ok"); e != nil {
		t.Fatalf("Require(true) = %v, want nil", e)
	}
	e := Require(false, 422, "VALIDATION_FAILED", "bad")
	ec := AsError(e)
	if ec == nil || ec.Code != "FIELD_VALIDATION_FAILED" || ec.Status != 422 {
		t.Fatalf("Require(false) = %v, want 422/FIELD_VALIDATION_FAILED", e)
	}
}

// TestRequireTypedNilRegression guards the typed-nil interface bug: returning
// Require(true) directly from an error-returning function must yield a nil
// error, not a non-nil interface wrapping a nil *Error.
func TestRequireTypedNilRegression(t *testing.T) {
	guard := func(cond bool) error { return Require(cond, 422, "INVALID_FILE", "x") }
	if err := guard(true); err != nil {
		t.Fatalf("Require(true) returned as error = %v (typed-nil), want nil", err)
	}
	if err := guard(false); err == nil {
		t.Fatalf("Require(false) returned as error = nil, want non-nil")
	}
}

func TestAsError(t *testing.T) {
	base := New(410, "CAPABILITY_EXPIRED", "凭证已失效")
	if got := AsError(base); got != base {
		t.Fatalf("AsError direct = %v, want same pointer", got)
	}
	wrapped := fmt.Errorf("context: %w", base)
	got := AsError(wrapped)
	if got == nil || got.Status != 410 || got.Code != "TOKEN_EXPIRED" {
		t.Fatalf("AsError unwrapped = %v, want 410/TOKEN_EXPIRED", got)
	}
	deep := Wrapf(wrapped, "outer %d", 1)
	if got := AsError(deep); got == nil || got.Code != "TOKEN_EXPIRED" {
		t.Fatalf("AsError deep = %v, want TOKEN_EXPIRED", got)
	}
	if got := AsError(errors.New("plain")); got != nil {
		t.Fatalf("AsError plain = %v, want nil", got)
	}
	if got := AsError(nil); got != nil {
		t.Fatalf("AsError nil = %v, want nil", got)
	}
}
