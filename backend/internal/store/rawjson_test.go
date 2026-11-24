package store

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestRawJSONNestedMarshalVerbatim pins the fix for the import-row envelope bug:
// a RawJSON value nested inside a map must serialize verbatim (like
// json.RawMessage), not base64-encoded as encoding/json does for []byte by
// default. Java's Jackson serializes PGobject verbatim; the sealed "raw" object
// must stay a JSON object so rawJSONFromEnvelope/rawKeys recover the headers.
func TestRawJSONNestedMarshalVerbatim(t *testing.T) {
	m := map[string]any{"raw": RawJSON(`{"name":"A","total":"2"}`), "fileErrors": []any{}}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, `"raw":{"name":"A","total":"2"}`) {
		t.Fatalf("nested RawJSON not verbatim: %s", s)
	}
	if strings.Contains(s, `"raw":"`) {
		t.Fatalf("nested RawJSON was base64/string-encoded: %s", s)
	}
	st, err := Stringify(m)
	if err != nil {
		t.Fatalf("stringify: %v", err)
	}
	if !strings.Contains(st, `"raw":{"name":"A","total":"2"}`) {
		t.Fatalf("Stringify nested RawJSON not verbatim: %s", st)
	}
	// Empty RawJSON marshals as null, not invalid empty output.
	nb, err := json.Marshal(RawJSON(nil))
	if err != nil || string(nb) != "null" {
		t.Fatalf("empty RawJSON = %q, err %v; want null", nb, err)
	}
}
