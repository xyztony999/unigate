package store

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

// TestFormatTime pins store.FormatTime to Jackson's OffsetDateTime serialization:
// fixed 3-digit milliseconds and a numeric UTC offset "+00:00" (NOT the variable
// 3/6/9-digit "Z" form, which is service.javaTimeCell / OffsetDateTime.toString).
func TestFormatTime(t *testing.T) {
	cases := []struct {
		in   time.Time
		want string
	}{
		{time.Date(2026, 11, 24, 10, 30, 45, 123000000, time.UTC), "2026-11-24T10:30:45.123+00:00"},
		{time.Date(2026, 11, 24, 10, 30, 45, 0, time.UTC), "2026-11-24T10:30:45.000+00:00"},
		// Sub-millisecond nanoseconds are truncated to 3 digits.
		{time.Date(2026, 11, 24, 10, 30, 45, 123456789, time.UTC), "2026-11-24T10:30:45.123+00:00"},
		// Non-UTC zones are normalized to UTC.
		{time.Date(2026, 11, 24, 18, 30, 45, 0, time.FixedZone("X", 8*3600)), "2026-11-24T10:30:45.000+00:00"},
	}
	for _, tc := range cases {
		if got := FormatTime(tc.in); got != tc.want {
			t.Errorf("FormatTime = %q, want %q", got, tc.want)
		}
	}
}

func TestJSONReadyTimeConversion(t *testing.T) {
	t0 := time.Date(2026, 11, 24, 10, 30, 45, 123000000, time.UTC)
	const want = "2026-11-24T10:30:45.123+00:00"

	if got := JSONReady(t0); got != want {
		t.Fatalf("JSONReady(time) = %v, want %q", got, want)
	}

	in := map[string]any{
		"created": t0,
		"list":    []any{t0, "keep"},
		"rows":    []map[string]any{{"at": t0}},
		"n":       json.Number("7"),
		"nil":     nil,
	}
	got := JSONReady(in).(map[string]any)
	if got["created"] != want {
		t.Fatalf("nested time = %v, want %q", got["created"], want)
	}
	list := got["list"].([]any)
	if list[0] != want || list[1] != "keep" {
		t.Fatalf("slice conversion = %#v", list)
	}
	rows := got["rows"].([]any)
	if rows[0].(map[string]any)["at"] != want {
		t.Fatalf("[]map conversion = %#v", rows)
	}
	if got["n"] != json.Number("7") {
		t.Fatalf("json.Number should pass through, got %#v", got["n"])
	}
	if got["nil"] != nil {
		t.Fatalf("nil should pass through, got %#v", got["nil"])
	}

	// RawJSON is parsed (numbers kept as json.Number) then made ready.
	rj := JSONReady(RawJSON(`{"n":1}`)).(map[string]any)
	if rj["n"] != json.Number("1") {
		t.Fatalf("RawJSON conversion = %#v", rj)
	}
}

func TestStrOrEmptyAndMap(t *testing.T) {
	t0 := time.Date(2026, 11, 24, 10, 30, 45, 123000000, time.UTC)
	cases := []struct {
		in   any
		want string
	}{
		{nil, ""},
		{"plain", "plain"},
		{json.Number("42"), "42"},
		{7, "7"},
		{t0, "2026-11-24T10:30:45.123+00:00"},
	}
	for _, tc := range cases {
		if got := StrOrEmpty(tc.in); got != tc.want {
			t.Errorf("StrOrEmpty(%#v) = %q, want %q", tc.in, got, tc.want)
		}
	}
	if Str(nil) != nil {
		t.Fatalf("Str(nil) should stay nil")
	}

	// Map skips nil values and stringifies keys.
	m := Map("a", 1, "b", nil, "c", "x")
	want := map[string]any{"a": 1, "c": "x"}
	if !reflect.DeepEqual(m, want) {
		t.Fatalf("Map = %#v, want %#v", m, want)
	}
}
