package service

import (
	"strings"
	"testing"
	"time"

	"unigate/backend/internal/errcode"
)

// TestSafeCellFormulaInjection reproduces IntegrationService.safeCell: a cell
// whose first non-whitespace character is = + - @ is prefixed with a single
// quote. Leading whitespace is Java \s = [ \t\n\x0B\f\r] (ASCII only).
func TestSafeCellFormulaInjection(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{nil, ""},
		{"", ""},
		{"hello", "hello"},
		{"=SUM(A1)", "'=SUM(A1)"},
		{"+1", "'+1"},
		{"-1", "'-1"},
		{"@cmd", "'@cmd"},
		{"-", "'-"},
		{"  =evil", "'  =evil"},
		{"\t=evil", "'\t=evil"},
		{"\n=evil", "'\n=evil"},
		{"\r=evil", "'\r=evil"},
		{"\f=evil", "'\f=evil"},
		{"\x0b=evil", "'\x0b=evil"}, // vertical tab is Java \s
		{" \t\n\x0b\f\r=evil", "' \t\n\x0b\f\r=evil"},
		{" \t+cmd", "' \t+cmd"},
		// Formula char must be first non-whitespace; embedded '=' is safe.
		{"a=b", "a=b"},
		{"  hello", "  hello"},
		// NBSP is NOT Java \s (no UNICODE_CHARACTER_CLASS), so no prefix added.
		{"\u00a0=evil", "\u00a0=evil"},
		// Numbers render via toString then get the same treatment.
		{5, "5"},
		{-5, "'-5"},
	}
	for _, tc := range cases {
		if got := safeCell(tc.in); got != tc.want {
			t.Errorf("safeCell(%#v) = %q, want %q", tc.in, got, tc.want)
		}
	}

	// time.Time routes through javaTimeCell (starts with a digit, never prefixed).
	tm := time.Date(2026, 11, 24, 10, 30, 45, 123000000, time.UTC)
	if got, want := safeCell(tm), "2026-11-24T10:30:45.123Z"; got != want {
		t.Errorf("safeCell(time) = %q, want %q", got, want)
	}
}

func assertInvalidFile(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected header validation failure, got nil")
	}
	ec := errcode.AsError(err)
	if ec == nil || ec.Status != 422 || ec.Code != "FIELD_VALIDATION_FAILED" {
		t.Fatalf("err = %v, want 422/FIELD_VALIDATION_FAILED (canonical INVALID_FILE)", err)
	}
}

func TestValidateHeaders(t *testing.T) {
	ok := [][]string{
		{"name"},
		{"name", "phone", "email"},
		{"姓名", "电话"}, // CJK headers
	}
	for _, h := range ok {
		if err := validateHeaders(h); err != nil {
			t.Errorf("validateHeaders(%v) = %v, want nil", h, err)
		}
	}

	bad := [][]string{
		{},               // empty
		{"name", "name"}, // duplicate
		{"name", ""},     // blank
		{"name", "   "},  // whitespace-only
		{"name", "\t"},   // whitespace-only
	}
	for _, h := range bad {
		assertInvalidFile(t, validateHeaders(h))
	}

	// Column count boundary: 200 ok, 201 rejected.
	cols := func(n int) []string {
		h := make([]string, n)
		for i := range h {
			h[i] = "c" + strings.Repeat("x", i%7) + string(rune('a'+i%26)) + itoa(i)
		}
		return h
	}
	if err := validateHeaders(cols(200)); err != nil {
		t.Errorf("200 columns rejected: %v", err)
	}
	assertInvalidFile(t, validateHeaders(cols(201)))
}

// TestValidateHeadersUTF16Length proves header length is measured in UTF-16 code
// units (Java String.length()), not runes or bytes.
func TestValidateHeadersUTF16Length(t *testing.T) {
	// 200 CJK chars = 200 UTF-16 units (ok); 201 = 201 units (rejected).
	if err := validateHeaders([]string{strings.Repeat("中", 200)}); err != nil {
		t.Errorf("200 CJK units rejected: %v", err)
	}
	assertInvalidFile(t, validateHeaders([]string{strings.Repeat("中", 201)}))

	// Each emoji is 2 UTF-16 units: 100 emoji = 200 units (ok), 101 = 202 (rejected).
	// A rune-counting implementation would wrongly accept 101 emoji.
	if err := validateHeaders([]string{strings.Repeat("😀", 100)}); err != nil {
		t.Errorf("100 emoji (200 units) rejected: %v", err)
	}
	assertInvalidFile(t, validateHeaders([]string{strings.Repeat("😀", 101)}))

	// ASCII boundary.
	if err := validateHeaders([]string{strings.Repeat("a", 200)}); err != nil {
		t.Errorf("200 ASCII units rejected: %v", err)
	}
	assertInvalidFile(t, validateHeaders([]string{strings.Repeat("a", 201)}))
}

// TestJavaTimeCell pins javaTimeCell to OffsetDateTime.toString(): shortest ISO
// local time, fractional seconds in 3/6/9-digit groups, "Z" for UTC.
func TestJavaTimeCell(t *testing.T) {
	d := func(sec int, ns int) time.Time {
		return time.Date(2026, 11, 24, 10, 30, sec, ns, time.UTC)
	}
	cases := []struct {
		in   time.Time
		want string
	}{
		{d(0, 0), "2026-11-24T10:30Z"},                       // whole minute omits seconds
		{d(45, 0), "2026-11-24T10:30:45Z"},                   // seconds, no fraction
		{d(0, 123000000), "2026-11-24T10:30:00.123Z"},        // nano forces ":00"
		{d(45, 123000000), "2026-11-24T10:30:45.123Z"},       // 3-digit group
		{d(45, 123456000), "2026-11-24T10:30:45.123456Z"},    // 6-digit group
		{d(45, 123456789), "2026-11-24T10:30:45.123456789Z"}, // 9-digit group
		{d(45, 100000000), "2026-11-24T10:30:45.100Z"},       // padded to 3
		{d(45, 120000000), "2026-11-24T10:30:45.120Z"},       // padded to 3
		{d(45, 1), "2026-11-24T10:30:45.000000001Z"},         // 9 digits
		{d(45, 10), "2026-11-24T10:30:45.000000010Z"},        // rounded up to 9
		{d(45, 123456780), "2026-11-24T10:30:45.123456780Z"},
		// Non-UTC zone normalized to UTC.
		{time.Date(2026, 11, 24, 18, 30, 45, 0, time.FixedZone("X", 8*3600)), "2026-11-24T10:30:45Z"},
	}
	for _, tc := range cases {
		if got := javaTimeCell(tc.in); got != tc.want {
			t.Errorf("javaTimeCell(%s) = %q, want %q", tc.in.Format(time.RFC3339Nano), got, tc.want)
		}
	}
}

// itoa avoids importing strconv just for unique header generation.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
