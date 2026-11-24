// Package errcode defines the frozen wire error codes and the canonical
// mapping reproduced from com.unigate.ApiException. It is a leaf package with
// no dependencies so store, service and api can all share it without creating
// import cycles.
package errcode

import (
	"errors"
	"fmt"
)

// Error is the business error carrying the real HTTP status, the canonical
// string code and the user-facing Chinese message.
type Error struct {
	Status  int
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Message }

// New builds an Error applying the canonical code mapping. The status is
// preserved except IDEMPOTENCY_PAYLOAD_MISMATCH which is always 400.
func New(status int, code, message string) *Error {
	c := canonical(status, code)
	if c == "IDEMPOTENCY_PAYLOAD_MISMATCH" {
		status = 400
	}
	return &Error{Status: status, Code: c, Message: message}
}

// Require panics-free assertion helper mirroring ApiException.require: it
// returns an *Error when cond is false, nil otherwise. Callers propagate it.
// The return type is error (not *Error) so that returning nil yields a nil
// interface; a *Error return would wrap a nil pointer in a non-nil interface
// when returned directly from an error-returning function (typed-nil bug).
func Require(cond bool, status int, code, message string) error {
	if cond {
		return nil
	}
	return New(status, code, message)
}

var known = map[string]bool{
	"INVALID_REQUEST": true, "IDEMPOTENCY_PAYLOAD_MISMATCH": true,
	"UNAUTHENTICATED": true, "SESSION_EXPIRED": true, "FORBIDDEN": true,
	"CSRF_INVALID": true, "RESOURCE_NOT_FOUND": true, "OPERATION_NOT_FOUND": true,
	"VERSION_CONFLICT": true, "CAPACITY_EXCEEDED": true,
	"PARTY_REMAINING_EXCEEDED": true, "EVENT_STATE_CONFLICT": true,
	"PUBLIC_MUTATIONS_FROZEN": true, "PREVIEW_STALE": true, "SOURCE_CONFLICT": true,
	"OPERATION_IN_PROGRESS": true, "TOKEN_EXPIRED": true, "ARTIFACT_EXPIRED": true,
	"FIELD_VALIDATION_FAILED": true, "TICKET_UNAVAILABLE": true,
	"CREDENTIAL_INVALID": true, "VERIFICATION_REQUIRED": true,
	"DUPLICATE_REVIEW_REQUIRED": true, "RATE_LIMITED": true, "INTERNAL_ERROR": true,
	"SERVICE_UNAVAILABLE": true, "PROVIDER_UNAVAILABLE": true,
}

func canonical(status int, code string) string {
	switch code {
	case "IDEMPOTENCY_CONFLICT":
		return "IDEMPOTENCY_PAYLOAD_MISMATCH"
	case "CAPACITY_CONFLICT":
		return "CAPACITY_EXCEEDED"
	case "ADMISSION_BALANCE_CONFLICT":
		return "PARTY_REMAINING_EXCEEDED"
	case "CSRF_REJECTED", "ORIGIN_REJECTED":
		return "CSRF_INVALID"
	case "AUTHENTICATION_FAILED", "DEVICE_UNAUTHORIZED", "PAIRING_INVALID":
		return "UNAUTHENTICATED"
	case "VALIDATION_FAILED", "FIELD_DISABLED", "INVALID_FILE", "INVALID_ENCODING", "INVALID_PROVIDER":
		return "FIELD_VALIDATION_FAILED"
	case "PHONE_VERIFICATION_REQUIRED":
		return "VERIFICATION_REQUIRED"
	case "PAIRING_EXPIRED", "CAPABILITY_EXPIRED", "SCAN_EXPIRED", "DATA_EXPIRED", "RESULT_EXPIRED", "BATCH_EXPIRED":
		return "TOKEN_EXPIRED"
	case "REGISTRATION_CLOSED", "EVENT_PHASE_CONFLICT", "PARTY_STATE_CONFLICT",
		"REVIEW_STATE_CONFLICT", "LAST_ADMIN", "LAST_ORGANIZER", "CORRECTION_CONFLICT",
		"ADMISSION_HISTORY_EXISTS", "POLICY_IN_USE", "NOTICE_CHANGED", "CHECKIN_CLOSED", "DATA_CONFLICT":
		return "EVENT_STATE_CONFLICT"
	case "PASSWORD_CHANGE_REQUIRED", "SELF_SERVICE_DISABLED":
		return "FORBIDDEN"
	}
	if known[code] {
		return code
	}
	switch status {
	case 400:
		return "INVALID_REQUEST"
	case 401:
		return "UNAUTHENTICATED"
	case 403:
		return "FORBIDDEN"
	case 404:
		return "RESOURCE_NOT_FOUND"
	case 409:
		return "EVENT_STATE_CONFLICT"
	case 410:
		return "ARTIFACT_EXPIRED"
	case 422:
		return "FIELD_VALIDATION_FAILED"
	case 429:
		return "RATE_LIMITED"
	case 503:
		return "SERVICE_UNAVAILABLE"
	default:
		return "INTERNAL_ERROR"
	}
}

// AsError extracts an *Error from an error chain (unwrapping %w), or nil.
func AsError(err error) *Error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return nil
}

// Wrapf adds context while keeping the *Error discoverable via errors.As.
func Wrapf(err error, format string, args ...any) error {
	return fmt.Errorf(format+": %w", append(args, err)...)
}
