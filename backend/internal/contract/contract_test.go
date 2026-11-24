package contract

import (
	"encoding/json"
	"testing"

	"unigate/backend/internal/errcode"
)

func assertValidationFailed(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected validation failure, got nil")
	}
	ec := errcode.AsError(err)
	if ec == nil || ec.Status != 422 || ec.Code != "FIELD_VALIDATION_FAILED" {
		t.Fatalf("err = %v, want 422/FIELD_VALIDATION_FAILED", err)
	}
}

// TestValidateRealContract exercises the embedded frozen contract: enum codes,
// $ref resolution, required fields and additionalProperties:false.
func TestValidateRealContract(t *testing.T) {
	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(c.Schemas) == 0 || len(c.Operations) == 0 {
		t.Fatalf("contract loaded empty: %d schemas, %d operations", len(c.Schemas), len(c.Operations))
	}

	ok := []struct {
		schema string
		value  any
	}{
		{"RoleCode", "ADMIN"},
		{"RoleCode", "PARTICIPANT"},
		{"PartyStatusCode", "PENDING"},
		{"EventStatusCode", "DRAFT"},
		{"RoleValue", map[string]any{"code": "ADMIN", "label": "管理员"}},
	}
	for _, tc := range ok {
		if err := c.Validate(tc.schema, tc.value); err != nil {
			t.Errorf("Validate(%s, %v) = %v, want nil", tc.schema, tc.value, err)
		}
	}

	bad := []struct {
		schema string
		value  any
	}{
		{"RoleCode", "BOGUS"},                                       // enum miss
		{"RoleCode", 123},                                           // wrong type
		{"PartyStatusCode", "NOPE"},                                 // enum miss
		{"RoleValue", map[string]any{"code": "ADMIN"}},              // missing required label
		{"RoleValue", map[string]any{"code": "NOPE", "label": "x"}}, // bad enum via $ref
		{"RoleValue", map[string]any{"code": "ADMIN", "label": ""}}, // label minLength 1
	}
	for _, tc := range bad {
		assertValidationFailed(t, c.Validate(tc.schema, tc.value))
	}

	// additionalProperties:false yields 400 INVALID_REQUEST, not 422.
	err = c.Validate("RoleValue", map[string]any{"code": "ADMIN", "label": "x", "extra": "y"})
	ec := errcode.AsError(err)
	if ec == nil || ec.Status != 400 || ec.Code != "INVALID_REQUEST" {
		t.Fatalf("unknown field err = %v, want 400/INVALID_REQUEST", err)
	}
}

// syntheticContract builds small schemas to deterministically exercise every
// validateNode branch (types, formats, bounds, UTF-16 length, arrays, nullable,
// dependentRequired, $ref).
func syntheticContract() *Contract {
	str := func(extra map[string]any) map[string]any {
		m := map[string]any{"type": "string"}
		for k, v := range extra {
			m[k] = v
		}
		return m
	}
	return &Contract{
		Schemas: map[string]map[string]any{
			"Str":         str(map[string]any{"minLength": 2, "maxLength": 5}),
			"Max1":        str(map[string]any{"maxLength": 1}),
			"Int":         {"type": "integer", "minimum": 0, "maximum": 10},
			"Num":         {"type": "number", "minimum": 0, "maximum": 1},
			"Bool":        {"type": "boolean"},
			"Uuid":        str(map[string]any{"format": "uuid"}),
			"DateTime":    str(map[string]any{"format": "date-time"}),
			"Email":       str(map[string]any{"format": "email"}),
			"Enum":        {"type": "string", "enum": []any{"X", "Y"}},
			"Arr":         {"type": "array", "minItems": 1, "maxItems": 3, "items": map[string]any{"type": "string"}},
			"NullableStr": {"type": []any{"string", "null"}},
			"Obj": {
				"type":                 "object",
				"properties":           map[string]any{"a": map[string]any{"type": "string"}, "b": map[string]any{"type": "integer"}},
				"required":             []any{"a"},
				"additionalProperties": false,
			},
			"Dep": {
				"type":              "object",
				"properties":        map[string]any{"a": map[string]any{"type": "string"}, "b": map[string]any{"type": "string"}},
				"dependentRequired": map[string]any{"b": []any{"a"}},
			},
			"Ref": {"$ref": "#/components/schemas/Str"},
		},
	}
}

func TestValidateBranches(t *testing.T) {
	c := syntheticContract()
	n := func(s string) json.Number { return json.Number(s) }

	ok := []struct {
		schema string
		value  any
	}{
		{"Str", "ab"},
		{"Str", "abcde"},
		{"Max1", "a"},
		{"Max1", "中"}, // 1 UTF-16 unit
		{"Int", n("5")},
		{"Int", 5},
		{"Int", n("0")},
		{"Int", n("10")},
		{"Num", n("0.5")},
		{"Bool", true},
		{"Bool", false},
		{"Uuid", "123e4567-e89b-12d3-a456-426614174000"},
		{"DateTime", "2026-11-24T10:30:45.123+00:00"},
		{"DateTime", "2026-11-24T10:30:45Z"},
		{"Email", "a@b.com"},
		{"Enum", "X"},
		{"Arr", []any{"a"}},
		{"Arr", []any{"a", "b", "c"}},
		{"NullableStr", nil},
		{"NullableStr", "hi"},
		{"Obj", map[string]any{"a": "x"}},
		{"Obj", map[string]any{"a": "x", "b": n("1")}},
		{"Dep", map[string]any{"a": "x"}},
		{"Dep", map[string]any{"a": "x", "b": "y"}},
		{"Ref", "ab"},
		{"DoesNotExist", "anything"}, // unknown schema node is permissive
	}
	for _, tc := range ok {
		if err := c.Validate(tc.schema, tc.value); err != nil {
			t.Errorf("Validate(%s, %#v) = %v, want nil", tc.schema, tc.value, err)
		}
	}

	bad := []struct {
		schema string
		value  any
	}{
		{"Str", "a"},                       // minLength
		{"Str", "abcdef"},                  // maxLength
		{"Str", nil},                       // null not allowed
		{"Str", 123},                       // wrong type
		{"Max1", "中文字"},                    // 3 UTF-16 units > 1
		{"Max1", "😀"},                      // 1 rune but 2 UTF-16 units > 1
		{"Int", n("11")},                   // > maximum
		{"Int", n("-1")},                   // < minimum
		{"Int", n("3.5")},                  // not an integer
		{"Int", "5"},                       // string is not a number
		{"Num", n("1.5")},                  // > maximum
		{"Num", n("-0.1")},                 // < minimum
		{"Bool", "true"},                   // wrong type
		{"Bool", 1},                        // wrong type
		{"Uuid", "not-uuid"},               // bad format
		{"DateTime", "2026-11-24"},         // missing time
		{"DateTime", "nope"},               // bad format
		{"Email", "abc"},                   // no @
		{"Email", "a@b"},                   // no dot in domain
		{"Enum", "Z"},                      // enum miss
		{"Arr", []any{}},                   // minItems
		{"Arr", []any{"a", "b", "c", "d"}}, // maxItems
		{"Arr", []any{"a", 1}},             // item type
		{"NullableStr", 123},               // null allowed but int is not string
		{"Obj", map[string]any{}},          // missing required a
		{"Obj", map[string]any{"a": "x", "b": "notint"}}, // property type
		{"Dep", map[string]any{"b": "y"}},                // dependentRequired: b needs a
		{"Ref", "a"},                                     // $ref resolves to Str minLength 2
	}
	for _, tc := range bad {
		assertValidationFailed(t, c.Validate(tc.schema, tc.value))
	}

	// additionalProperties:false on the synthetic object.
	err := c.Validate("Obj", map[string]any{"a": "x", "zzz": "y"})
	ec := errcode.AsError(err)
	if ec == nil || ec.Status != 400 || ec.Code != "INVALID_REQUEST" {
		t.Fatalf("unknown field err = %v, want 400/INVALID_REQUEST", err)
	}
}
