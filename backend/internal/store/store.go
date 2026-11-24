// Package store is the only layer that touches SQL (pgx/PostgreSQL, ADR-048).
// It reproduces com.unigate.Store: dynamic row maps, event scoping, optimistic
// version checks, encrypted audit reasons and the pagination envelopes.
package store

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"unigate/backend/internal/errcode"
)

// Store owns the connection pool and the crypto/key vault used to seal rows.
type Store struct {
	Pool   *pgxpool.Pool
	Crypto *Crypto
	Keys   *KeyVault
}

// Scope carries the per-request/per-transaction state that Java kept in
// ThreadLocals (eventScope, readOnlyRequest) plus the active transaction and
// the Set-Cookie headers accumulated for idempotent replay.
type Scope struct {
	Tx       pgx.Tx
	EventID  string // "" means no event scope (master-key sealing)
	ReadOnly bool   // operation is a read (for share instead of for update)
	Snapshot bool   // read-only repeatable-read transaction (Statistics)
	Cookies  []string
}

// querier is satisfied by both the pool and a transaction.
type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func (s *Store) q(sc *Scope) querier {
	if sc != nil && sc.Tx != nil {
		return sc.Tx
	}
	return s.Pool
}

// Rows runs a parameterized query ($N placeholders) and returns dynamic maps.
func (s *Store) Rows(ctx context.Context, sc *Scope, sql string, args ...any) ([]map[string]any, error) {
	rows, err := s.q(sc).Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	fds := rows.FieldDescriptions()
	out := make([]map[string]any, 0)
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		m := make(map[string]any, len(fds))
		for i, fd := range fds {
			m[string(fd.Name)] = normalizeValue(fd.DataTypeOID, vals[i])
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}
	return out, nil
}

// Row returns the first row or 404 RESOURCE_NOT_FOUND (Store.row semantics).
func (s *Store) Row(ctx context.Context, sc *Scope, sql string, args ...any) (map[string]any, error) {
	l, err := s.Rows(ctx, sc, sql, args...)
	if err != nil {
		return nil, err
	}
	if len(l) == 0 {
		return nil, errcode.New(404, "RESOURCE_NOT_FOUND", "未找到资源")
	}
	return l[0], nil
}

// Exec runs a parameterized statement and returns the affected row count.
func (s *Store) Exec(ctx context.Context, sc *Scope, sql string, args ...any) (int64, error) {
	tag, err := s.q(sc).Exec(ctx, sql, args...)
	if err != nil {
		return 0, fmt.Errorf("exec: %w", err)
	}
	return tag.RowsAffected(), nil
}

// QueryInt runs a single integer aggregate query.
func (s *Store) QueryInt(ctx context.Context, sc *Scope, sql string, args ...any) (int64, error) {
	var n int64
	var err error
	if sc != nil && sc.Tx != nil {
		err = sc.Tx.QueryRow(ctx, sql, args...).Scan(&n)
	} else {
		err = s.Pool.QueryRow(ctx, sql, args...).Scan(&n)
	}
	if err != nil {
		return 0, fmt.Errorf("query int: %w", err)
	}
	return n, nil
}

// normalizeValue converts pgx driver values into the canonical row types:
// uuid→string, timestamptz→time.Time(UTC), jsonb→parsed JSON, bytea→[]byte.
func normalizeValue(oid uint32, v any) any {
	if v == nil {
		return nil
	}
	switch oid {
	case pgtype.UUIDOID:
		if u, ok := v.(pgtype.UUID); ok && u.Valid {
			return uuidString(u.Bytes)
		}
	case pgtype.TimestamptzOID, pgtype.TimestampOID:
		if t, ok := v.(time.Time); ok {
			return t.UTC()
		}
	case pgtype.JSONBOID, pgtype.JSONOID:
		switch b := v.(type) {
		case []byte:
			return RawJSON(append([]byte(nil), b...))
		case string:
			return RawJSON([]byte(b))
		}
	}
	switch t := v.(type) {
	case pgtype.UUID:
		if t.Valid {
			return uuidString(t.Bytes)
		}
		return nil
	case time.Time:
		return t.UTC()
	case [16]byte:
		return uuidString(t)
	default:
		return v
	}
}

func uuidString(b [16]byte) string {
	const hex = "0123456789abcdef"
	out := make([]byte, 36)
	pos := 0
	for i, x := range b {
		if i == 4 || i == 6 || i == 8 || i == 10 {
			out[pos] = '-'
			pos++
		}
		out[pos] = hex[x>>4]
		out[pos+1] = hex[x&0xf]
		pos += 2
	}
	return string(out)
}

// ParseJSON decodes JSON preserving number lexemes (json.Number) so digests
// and re-serialization stay faithful to the original Jackson behaviour.
func ParseJSON(b []byte) any {
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.UseNumber()
	var out any
	if err := dec.Decode(&out); err != nil {
		return nil
	}
	return out
}

// ID generates a random UUID string (Store.id).
func (s *Store) ID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Sprintf("crypto/rand unavailable: %v", err))
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return uuidString(b)
}

// Now returns the current UTC time (Store.now).
func (s *Store) Now() time.Time { return time.Now().UTC() }

// RawJSON preserves the exact jsonb text PostgreSQL returned. Java's JDBC
// driver yields a PGobject whose toString()/getValue() is this raw text; keeping
// it lets noticeVersion and source mapping fingerprints stay byte-identical.
type RawJSON []byte

// MarshalJSON writes the preserved jsonb text verbatim (like json.RawMessage).
// Without this, a RawJSON nested in a map/slice would be base64-encoded by
// encoding/json ([]byte default), corrupting sealed envelopes such as the
// import row "raw" object. Java's Jackson serializes PGobject verbatim.
func (r RawJSON) MarshalJSON() ([]byte, error) {
	if len(r) == 0 {
		return []byte("null"), nil
	}
	return []byte(r), nil
}

// Parse coerces a jsonb value (already decoded or raw string) into a map.
func Parse(o any) map[string]any {
	if o == nil {
		return map[string]any{}
	}
	if m, ok := o.(map[string]any); ok {
		return m
	}
	if r, ok := o.(RawJSON); ok {
		if m, ok := ParseJSON([]byte(r)).(map[string]any); ok {
			return m
		}
		return map[string]any{}
	}
	if b, ok := o.([]byte); ok {
		if m, ok := ParseJSON(b).(map[string]any); ok {
			return m
		}
	}
	if str, ok := o.(string); ok {
		if m, ok := ParseJSON([]byte(str)).(map[string]any); ok {
			return m
		}
	}
	return map[string]any{}
}

// List coerces a jsonb value into a slice.
func List(o any) []any {
	if o == nil {
		return []any{}
	}
	if l, ok := o.([]any); ok {
		return l
	}
	if r, ok := o.(RawJSON); ok {
		if l, ok := ParseJSON([]byte(r)).([]any); ok {
			return l
		}
		return []any{}
	}
	if b, ok := o.([]byte); ok {
		if l, ok := ParseJSON(b).([]any); ok {
			return l
		}
	}
	if str, ok := o.(string); ok {
		if l, ok := ParseJSON([]byte(str)).([]any); ok {
			return l
		}
	}
	return []any{}
}

// Stringify serializes a value to JSON (Store.stringify). A RawJSON value is
// returned verbatim, matching PGobject.toString() in the Java implementation.
func Stringify(o any) (string, error) {
	if r, ok := o.(RawJSON); ok {
		return string(r), nil
	}
	b, err := json.Marshal(o)
	if err != nil {
		return "", fmt.Errorf("stringify: %w", err)
	}
	return string(b), nil
}

// JSONString renders a Go string as a JSON string literal without HTML
// escaping, matching Jackson's default string serialization.
func JSONString(s string) (string, error) {
	var buf strings.Builder
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(s); err != nil {
		return "", fmt.Errorf("json string: %w", err)
	}
	return strings.TrimRight(buf.String(), "\n"), nil
}

// PGObjectJSON reproduces Jackson's serialization of a JDBC PGobject(jsonb):
// {"type":"jsonb","value":<raw text>,"null":false}. Used by noticeVersion so the
// consent fingerprint stays byte-identical to the Java backend.
func PGObjectJSON(raw string) (string, error) {
	value, err := JSONString(raw)
	if err != nil {
		return "", err
	}
	return `{"type":"jsonb","value":` + value + `,"null":false}`, nil
}

// Num truncates any numeric representation to int (Store.num).
func Num(o any) int {
	switch n := o.(type) {
	case nil:
		return 0
	case int:
		return n
	case int32:
		return int(n)
	case int64:
		return int(n)
	case float64:
		return int(n)
	case json.Number:
		if i, err := n.Int64(); err == nil {
			return int(i)
		}
		if f, err := n.Float64(); err == nil {
			return int(f)
		}
		return 0
	default:
		return 0
	}
}

// Str renders a value as string, nil stays nil (Store.str).
func Str(o any) any {
	if o == nil {
		return nil
	}
	switch v := o.(type) {
	case string:
		return v
	case time.Time:
		return FormatTime(v)
	case json.Number:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

// StrOrEmpty is Str with "" for nil (Java str() used in contains/equals).
func StrOrEmpty(o any) string {
	if o == nil {
		return ""
	}
	if s, ok := Str(o).(string); ok {
		return s
	}
	return fmt.Sprintf("%v", o)
}

// Time coerces a row/input value to time.Time (Store.time).
func Time(o any) (time.Time, error) {
	switch t := o.(type) {
	case time.Time:
		return t.UTC(), nil
	case string:
		parsed, err := time.Parse(time.RFC3339Nano, t)
		if err != nil {
			return time.Time{}, fmt.Errorf("parse time %q: %w", t, err)
		}
		return parsed.UTC(), nil
	default:
		return time.Time{}, fmt.Errorf("not a time value: %T", o)
	}
}

// FormatTime renders RFC3339 with millisecond precision and a numeric UTC
// offset, matching Jackson's OffsetDateTime serialization ("...T..:..:..sss+00:00").
func FormatTime(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000-07:00")
}

// JSONReady recursively converts row values into their wire representation
// (times become the frozen RFC3339 millisecond string), mirroring Jackson's
// convertValue before response-contract validation and serialization.
func JSONReady(v any) any {
	switch t := v.(type) {
	case time.Time:
		return FormatTime(t)
	case RawJSON:
		return JSONReady(ParseJSON([]byte(t)))
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			out[k] = JSONReady(val)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			out[i] = JSONReady(val)
		}
		return out
	case []map[string]any:
		out := make([]any, len(t))
		for i, val := range t {
			out[i] = JSONReady(val)
		}
		return out
	default:
		return v
	}
}

// Map builds an insertion-ordered map skipping nil values (Store.map).
// Go maps are unordered; JSON key order is normalized by Marshal, and the
// canonical digest sorts keys explicitly, so a plain map is equivalent.
func Map(pairs ...any) map[string]any {
	m := make(map[string]any)
	for i := 0; i+1 < len(pairs); i += 2 {
		if pairs[i+1] != nil {
			m[fmt.Sprintf("%v", pairs[i])] = pairs[i+1]
		}
	}
	return m
}

// Value renders an enum value object {code,label} from the contract pool.
func (s *Store) Value(enums map[string]map[string]string, typ string, code any) map[string]any {
	if code == nil {
		return nil
	}
	c := StrOrEmpty(code)
	label := c
	if pool, ok := enums[typ]; ok {
		if l, ok := pool[c]; ok {
			label = l
		}
	}
	return Map("code", c, "label", label)
}

// Page slices an in-memory list into the frozen page envelope (Store.page).
func Page(all []any, r map[string]any) map[string]any {
	p := 1
	if v, ok := r["page"]; ok {
		p = Num(v)
	}
	if p < 1 {
		p = 1
	}
	size := 20
	if v, ok := r["pageSize"]; ok {
		size = Num(v)
	}
	if size < 1 {
		size = 1
	}
	if size > 100 {
		size = 100
	}
	start := int(math.Min(float64((p-1)*size), float64(len(all))))
	end := int(math.Min(float64(p*size), float64(len(all))))
	return Map(
		"items", all[start:end],
		"total", len(all),
		"page", p,
		"pageSize", size,
	)
}

// SelectorPage implements cursor pagination (base64 offset) for option lists.
func SelectorPage(all []any, size int, request map[string]any) (map[string]any, error) {
	start := 0
	if cur, ok := request["cursor"]; ok {
		raw, err := base64.RawURLEncoding.DecodeString(StrOrEmpty(cur))
		if err != nil {
			return nil, errcode.New(422, "FIELD_VALIDATION_FAILED", "无效分页游标")
		}
		n, err := strconv.Atoi(string(raw))
		if err != nil || n < 0 {
			return nil, errcode.New(422, "FIELD_VALIDATION_FAILED", "无效分页游标")
		}
		start = n
	}
	end := int(math.Min(float64(start+size), float64(len(all))))
	lo := int(math.Min(float64(start), float64(len(all))))
	out := Map("items", all[lo:end])
	if end < len(all) {
		out["nextCursor"] = base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(end)))
	}
	return out, nil
}

// Seal encrypts a JSON value under the scoped event DEK or the master key.
func (s *Store) Seal(sc *Scope, o any) ([]byte, error) {
	text, err := Stringify(o)
	if err != nil {
		return nil, err
	}
	if sc == nil || sc.EventID == "" {
		return s.Crypto.Seal(text)
	}
	return s.Keys.Seal(sc.EventID, text)
}

// Unseal decrypts a bytea column, dispatching on the version byte.
func (s *Store) Unseal(o any) (map[string]any, error) {
	text, err := s.UnsealPlain(o)
	if err != nil {
		return nil, err
	}
	return Parse([]byte(text)), nil
}

// UnsealPlain decrypts a bytea column to its JSON text without parsing, so
// callers that need the original key order (import raw rows) can decode it
// themselves. nil yields "" like Unseal yields an empty map.
func (s *Store) UnsealPlain(o any) (string, error) {
	if o == nil {
		return "", nil
	}
	data, ok := o.([]byte)
	if !ok {
		return "", fmt.Errorf("unseal: not a bytea value: %T", o)
	}
	if len(data) > 0 && data[0] == 2 {
		return s.Keys.Open(data)
	}
	return s.Crypto.Open(data)
}

// RevokeEventKey revokes the DEK and clears the scope.
func (s *Store) RevokeEventKey(sc *Scope, event string) error {
	if err := s.Keys.Revoke(event); err != nil {
		return err
	}
	sc.EventID = ""
	return nil
}

// Event loads the event row under the appropriate lock, sets the event scope
// and enforces the personal-data expiry (Store.event). Snapshot transactions
// take no lock; read operations take FOR SHARE; mutations FOR UPDATE.
func (s *Store) Event(ctx context.Context, sc *Scope, id string) (map[string]any, error) {
	sc.EventID = id
	if err := s.Keys.RequireNotRevoked(id); err != nil {
		return nil, err
	}
	lock := " for update"
	if sc.Snapshot {
		lock = ""
	} else if sc.ReadOnly {
		lock = " for share"
	}
	e, err := s.Row(ctx, sc, "select * from ug_event where id=$1"+lock, id)
	if err != nil {
		return nil, err
	}
	due, err := Time(e["personal_due_at"])
	if err != nil {
		return nil, fmt.Errorf("event due time: %w", err)
	}
	if !due.After(s.Now()) || e["status"] == "ARCHIVED" {
		return nil, errcode.New(410, "DATA_EXPIRED", "活动明细已到期")
	}
	return e, nil
}

// Audit writes an audit row with the encrypted reason (Store.audit).
func (s *Store) Audit(ctx context.Context, sc *Scope, eventID any, action string, actorID any, deviceID any, resource any, operation any, reason any) error {
	var reasonCipher []byte
	if reason != nil {
		sealed, err := s.Seal(sc, Map("reason", reason))
		if err != nil {
			return err
		}
		reasonCipher = sealed
	}
	_, err := s.Exec(ctx, sc,
		"insert into ug_audit(id,event_id,action,actor_id,device_id,resource_id,operation_id,reason_cipher,occurred_at) values($1,$2,$3,$4,$5,$6,$7,$8,$9)",
		s.ID(), nullableUUID(eventID), action, nullableUUID(actorID), deviceID,
		nullableUUID(resource), nullableUUID(operation), reasonCipher, s.Now())
	if err != nil {
		return fmt.Errorf("insert audit: %w", err)
	}
	return nil
}

// nullableUUID converts nil to a SQL NULL and everything else to its string
// form for uuid columns (Store.uuid semantics).
func nullableUUID(o any) any {
	if o == nil {
		return nil
	}
	return StrOrEmpty(o)
}

// Version enforces optimistic locking (Store.version).
func Version(row map[string]any, expected any) error {
	if expected == nil || Num(row["version"]) != Num(expected) {
		return errcode.New(409, "VERSION_CONFLICT", "数据已变化，请刷新后确认")
	}
	return nil
}

// IsLockConflict reports PostgreSQL lock/serialization failures that the
// controller maps to 409 OPERATION_IN_PROGRESS.
func IsLockConflict(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "55P03", "40001", "40P01":
			return true
		}
	}
	return false
}

// IsIntegrityViolation reports unique/foreign-key/check violations that the
// controller maps to 409 DATA_CONFLICT (canonical EVENT_STATE_CONFLICT).
func IsIntegrityViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return strings.HasPrefix(pgErr.Code, "23")
	}
	return false
}