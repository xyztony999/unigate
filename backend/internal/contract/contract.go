// Package contract loads the frozen OpenAPI contract (byte-identical copy of
// openapi-unigate.json) and reproduces com.unigate.Contract schema validation
// for both requests and successful responses.
package contract

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"
	"unicode/utf16"

	"unigate/backend/internal/errcode"
)

//go:embed openapi-unigate.json
var raw []byte

// Operation is one entry of x-operation-index.
type Operation struct {
	Name     string   `json:"name"`
	Method   string   `json:"method"`
	Path     string   `json:"path"`
	Auth     string   `json:"auth"`
	Mutation bool     `json:"mutation"`
	Status   int      `json:"status"`
	Roles    []string `json:"roles"`
	Request  string   `json:"request"`
	Data     string   `json:"data"`
	Upload   bool     `json:"upload"`
}

// Contract holds schemas, enum pool and the operation indexes.
type Contract struct {
	Schemas    map[string]map[string]any
	Enums      map[string]map[string]string
	Routes     map[string]*Operation
	Operations map[string]*Operation
}

// Load parses the embedded frozen contract.
func Load() (*Contract, error) {
	var doc struct {
		Components struct {
			Schemas map[string]map[string]any `json:"schemas"`
		} `json:"components"`
		EnumPool map[string]map[string]string `json:"x-enum-pool"`
		Index    []*Operation                 `json:"x-operation-index"`
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("cannot load API contract: %w", err)
	}
	c := &Contract{
		Schemas:    doc.Components.Schemas,
		Enums:      doc.EnumPool,
		Routes:     map[string]*Operation{},
		Operations: map[string]*Operation{},
	}
	for _, op := range doc.Index {
		c.Routes[strings.ToUpper(op.Method)+" "+op.Path] = op
		c.Operations[op.Name] = op
	}
	return c, nil
}

var (
	uuidRe  = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	emailRe = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
)

// Validate checks value against the named schema (Contract.validate).
func (c *Contract) Validate(schema string, value any) error {
	return c.validateNode(c.Schemas[schema], value, schema)
}

func schemaName(ref string) string {
	// "#/components/schemas/" is 21 characters, same substring as Java.
	return strings.TrimPrefix(ref, "#/components/schemas/")
}

func (c *Contract) validateNode(node map[string]any, v any, path string) error {
	if node == nil {
		return nil
	}
	if ref, ok := node["$ref"].(string); ok {
		return c.validateNode(c.Schemas[schemaName(ref)], v, path)
	}
	typ := node["type"]
	if v == nil {
		if l, ok := typ.([]any); ok {
			for _, t := range l {
				if t == "null" {
					return nil
				}
			}
		}
		return errcode.New(422, "VALIDATION_FAILED", "字段不允许为空: "+path)
	}
	t := ""
	switch tt := typ.(type) {
	case string:
		t = tt
	case []any:
		if len(tt) > 0 {
			t, _ = tt[0].(string)
		}
	}
	valid := true
	switch t {
	case "object":
		_, valid = v.(map[string]any)
	case "array":
		_, valid = v.([]any)
	case "string":
		_, valid = v.(string)
	case "integer":
		valid = isInteger(v)
	case "number":
		valid = isNumber(v)
	case "boolean":
		_, valid = v.(bool)
	}
	if !valid {
		return errcode.New(422, "VALIDATION_FAILED", "字段类型错误: "+path)
	}
	if enum, ok := node["enum"].([]any); ok {
		found := false
		for _, e := range enum {
			if equalJSON(e, v) {
				found = true
				break
			}
		}
		if !found {
			return errcode.New(422, "VALIDATION_FAILED", "字段值无效: "+path)
		}
	}
	if s, ok := v.(string); ok {
		n := utf16len(s)
		if ml, ok := node["minLength"]; ok && n < num(ml) {
			return errcode.New(422, "VALIDATION_FAILED", "字段过短: "+path)
		}
		if ml, ok := node["maxLength"]; ok && n > num(ml) {
			return errcode.New(422, "VALIDATION_FAILED", "字段过长: "+path)
		}
		format, _ := node["format"].(string)
		switch format {
		case "uuid":
			if !uuidRe.MatchString(s) {
				return errcode.New(422, "VALIDATION_FAILED", "字段格式无效: "+path)
			}
		case "date-time":
			if _, err := time.Parse(time.RFC3339Nano, s); err != nil {
				return errcode.New(422, "VALIDATION_FAILED", "字段格式无效: "+path)
			}
		case "email":
			if !emailRe.MatchString(s) {
				return errcode.New(422, "VALIDATION_FAILED", "字段格式无效: "+path)
			}
		}
	}
	if isNumber(v) {
		d := toFloat(v)
		if lo, ok := node["minimum"]; ok && isNumber(lo) && d < toFloat(lo) {
			return errcode.New(422, "VALIDATION_FAILED", "字段小于下限: "+path)
		}
		if hi, ok := node["maximum"]; ok && isNumber(hi) && d > toFloat(hi) {
			return errcode.New(422, "VALIDATION_FAILED", "字段大于上限: "+path)
		}
	}
	if m, ok := v.(map[string]any); ok {
		props, _ := node["properties"].(map[string]any)
		if req, ok := node["required"].([]any); ok {
			for _, r := range req {
				key, _ := r.(string)
				if _, present := m[key]; !present {
					return errcode.New(422, "VALIDATION_FAILED", "缺少字段: "+path+"."+key)
				}
			}
		}
		for key, val := range m {
			propSchema, known := props[key]
			if !known {
				if ap, ok := node["additionalProperties"].(bool); ok && !ap {
					return errcode.New(400, "INVALID_REQUEST", "未知字段: "+path+"."+key)
				}
				continue
			}
			ps, _ := propSchema.(map[string]any)
			if err := c.validateNode(ps, val, path+"."+key); err != nil {
				return err
			}
		}
		if deps, ok := node["dependentRequired"].(map[string]any); ok {
			for key, dep := range deps {
				if _, present := m[key]; !present {
					continue
				}
				if l, ok := dep.([]any); ok {
					for _, d := range l {
						dk, _ := d.(string)
						if _, present := m[dk]; !present {
							return errcode.New(422, "VALIDATION_FAILED", "缺少关联字段: "+dk)
						}
					}
				}
			}
		}
	}
	if l, ok := v.([]any); ok {
		if mi, ok := node["maxItems"]; ok && len(l) > num(mi) {
			return errcode.New(422, "VALIDATION_FAILED", "列表过长: "+path)
		}
		if mi, ok := node["minItems"]; ok && len(l) < num(mi) {
			return errcode.New(422, "VALIDATION_FAILED", "列表不能为空: "+path)
		}
		items, _ := node["items"].(map[string]any)
		for _, item := range l {
			if err := c.validateNode(items, item, path+"[]"); err != nil {
				return err
			}
		}
	}
	return nil
}

func utf16len(s string) int { return len(utf16.Encode([]rune(s))) }

func num(o any) int {
	switch n := o.(type) {
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	}
	return 0
}

func isNumber(v any) bool {
	switch v.(type) {
	case json.Number, float64, int, int32, int64:
		return true
	}
	return false
}

func toFloat(v any) float64 {
	switch n := v.(type) {
	case json.Number:
		f, _ := n.Float64()
		return f
	case float64:
		return n
	case int:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	}
	return math.NaN()
}

// isInteger mirrors Java's n.doubleValue() == n.longValue() check.
func isInteger(v any) bool {
	if !isNumber(v) {
		return false
	}
	f := toFloat(v)
	return f == math.Trunc(f) && f >= math.MinInt64 && f < 9223372036854775808.0
}

func equalJSON(a, b any) bool {
	if isNumber(a) && isNumber(b) {
		return toFloat(a) == toFloat(b)
	}
	return a == b
}