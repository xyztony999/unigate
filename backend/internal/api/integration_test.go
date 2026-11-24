package api

// PostgreSQL integration tests (HTTP level, frozen wire contract).
//
// Port of the Java PasswordResetTest / WorkflowIntegrationTest skeletons:
// guarded by UNIGATE_TEST_DB_URL exactly like @EnabledIfEnvironmentVariable,
// one shared server per package run, per-test truncation of the ug_* tables
// plus a reset of the isolated integration-admin account (Java @BeforeEach).
//
// Run with:
//
//	set -a; . ../.local/go-test.env; set +a
//	go test ./internal/api/ -run Integration -v

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"unigate/backend/internal/config"
	"unigate/backend/internal/contract"
	"unigate/backend/internal/service"
	"unigate/backend/internal/store"
)

const (
	itAdminUsername = "integration-admin"
	itAdminPassword = "Integration-test-123!"
	// Fixed non-production test credentials only (same values as the Java tests).
	itMasterKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
	itOrigin    = "http://localhost:5173"
)

// itEnv is the shared integration fixture: real PG pool + migrated schema +
// contract + service + router behind an httptest server.
type itEnv struct {
	s      *store.Store
	c      *contract.Contract
	cfg    *config.Config
	svc    *service.Service
	server *httptest.Server
}

var (
	itOnce sync.Once
	it     *itEnv
	itErr  error
)

// integrationEnv skips unless UNIGATE_TEST_DB_URL is set (Java
// @EnabledIfEnvironmentVariable(named = "UNIGATE_TEST_DB_URL", matches = ".+")).
func integrationEnv(t *testing.T) *itEnv {
	t.Helper()
	if os.Getenv("UNIGATE_TEST_DB_URL") == "" {
		t.Skip("UNIGATE_TEST_DB_URL not set; skipping PostgreSQL integration test")
	}
	itOnce.Do(func() { it, itErr = buildIntegrationEnv() })
	if itErr != nil {
		t.Fatalf("integration setup: %v", itErr)
	}
	return it
}

func itGetenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// buildIntegrationEnv mirrors main.openStore + the Java @SpringBootTest
// properties: NormalizeDBURL (JDBC -> pgx), pool, Migrate, KeyVault, contract,
// service, router, httptest server, bootstrap account when count==0.
func buildIntegrationEnv() (*itEnv, error) {
	ctx := context.Background()
	dsn, err := config.NormalizeDBURL(os.Getenv("UNIGATE_TEST_DB_URL"))
	if err != nil {
		return nil, fmt.Errorf("normalize test db url: %w", err)
	}
	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse test db url: %w", err)
	}
	poolCfg.ConnConfig.User = itGetenv("UNIGATE_TEST_DB_USER", "unigate")
	poolCfg.ConnConfig.Password = os.Getenv("UNIGATE_TEST_DB_PASSWORD")
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create test pool: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect test database: %w", err)
	}
	crypto, err := store.NewCrypto(itGetenv("UNIGATE_MASTER_KEY", itMasterKey))
	if err != nil {
		pool.Close()
		return nil, err
	}
	keyDir := os.Getenv("UNIGATE_KEY_DIRECTORY")
	if keyDir == "" {
		keyDir, err = os.MkdirTemp("", "unigate-it-keys-")
		if err != nil {
			pool.Close()
			return nil, err
		}
	}
	keys, err := store.NewKeyVault(crypto, keyDir)
	if err != nil {
		pool.Close()
		return nil, err
	}
	s := &store.Store{Pool: pool, Crypto: crypto, Keys: keys}
	if err := s.Migrate(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("migrate test database: %w", err)
	}
	c, err := contract.Load()
	if err != nil {
		pool.Close()
		return nil, err
	}
	cfg := &config.Config{
		MasterKey:         itGetenv("UNIGATE_MASTER_KEY", itMasterKey),
		CookieSecure:      itGetenv("UNIGATE_COOKIE_SECURE", "false") == "true",
		AllowedOrigin:     itGetenv("UNIGATE_ALLOWED_ORIGIN", itOrigin),
		BootstrapUsername: itGetenv("UNIGATE_BOOTSTRAP_USERNAME", itAdminUsername),
		BootstrapPassword: itGetenv("UNIGATE_BOOTSTRAP_PASSWORD", itAdminPassword),
		KeyDirectory:      keyDir,
		StorageDirectory:  itGetenv("UNIGATE_STORAGE_DIRECTORY", os.TempDir()),
	}
	svc := service.New(s, c, cfg)
	if err := svc.BootstrapAccount(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("bootstrap account: %w", err)
	}
	h, err := NewRouter(svc, cfg)
	if err != nil {
		pool.Close()
		return nil, err
	}
	return &itEnv{s: s, c: c, cfg: cfg, svc: svc, server: httptest.NewServer(h)}, nil
}

// reset isolates tests: truncate every ug_* table (except the migration
// ledger), recreate the bootstrap admin and force the known initial password
// state, mirroring WorkflowIntegrationTest.resetOnlyIsolatedTestAccount.
func (e *itEnv) reset(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	rows, err := e.s.Rows(ctx, nil,
		"select table_name from information_schema.tables where table_schema=current_schema() and table_name like 'ug_%' and table_name <> 'ug_schema_migration'")
	if err != nil {
		t.Fatalf("list test tables: %v", err)
	}
	names := make([]string, 0, len(rows))
	for _, r := range rows {
		// Identifiers come from information_schema, not from test input.
		names = append(names, `"`+store.StrOrEmpty(r["table_name"])+`"`)
	}
	if len(names) > 0 {
		if _, err := e.s.Exec(ctx, nil, "truncate table "+strings.Join(names, ",")+" cascade"); err != nil {
			t.Fatalf("truncate test tables: %v", err)
		}
	}
	if err := e.svc.BootstrapAccount(ctx); err != nil {
		t.Fatalf("re-bootstrap admin: %v", err)
	}
	hash, err := e.svc.Passwords.Encode(itAdminPassword)
	if err != nil {
		t.Fatalf("hash test password: %v", err)
	}
	if _, err := e.s.Exec(ctx, nil,
		"update ug_account set password_hash=$1,must_change_password=true,auth_version=auth_version+1 where username=$2",
		hash, itAdminUsername); err != nil {
		t.Fatalf("reset admin password state: %v", err)
	}
}

// itBrowser reproduces the Java Browser: cookie jar, csrfToken and
// clientContextId captured from every response envelope, Origin on every
// request, X-CSRF-Token / X-Client-Context-ID once bootstrapped. ANON writes
// carry the anon csrf+context; after Login the same fields carry the staff
// csrf alongside the ug_staff cookie; after PartyExchange the party csrf.
type itBrowser struct {
	t           *testing.T
	env         *itEnv
	cookies     map[string]string
	csrf        string
	context     string
	lastCode    string // envelope code of the most recent request
	lastMessage string // envelope message of the most recent request
}

func (e *itEnv) browser(t *testing.T) *itBrowser {
	return &itBrowser{t: t, env: e, cookies: map[string]string{}}
}

func (b *itBrowser) bootstrap() {
	b.t.Helper()
	b.request("SessionBootstrap", map[string]any{}, http.StatusOK)
}

// write adds a fresh operationId when absent (Java Browser.write).
func (b *itBrowser) write(op string, args map[string]any, status int) map[string]any {
	b.t.Helper()
	r := make(map[string]any, len(args)+1)
	for k, v := range args {
		r[k] = v
	}
	if _, ok := r["operationId"]; !ok {
		r["operationId"] = b.env.s.ID()
	}
	return b.request(op, r, status)
}

// request performs one contract operation and asserts the frozen envelope:
// real HTTP status, {code,message,data,requestId}; success code OK, errors a
// string code with data{}. Returns the data object.
func (b *itBrowser) request(name string, args map[string]any, wantStatus int) map[string]any {
	b.t.Helper()
	op := b.env.c.Operations[name]
	if op == nil {
		b.t.Fatalf("unknown contract operation %q", name)
	}
	var req *http.Request
	var err error
	if strings.EqualFold(op.Method, http.MethodGet) {
		q := url.Values{}
		for k, v := range args {
			q.Set(k, fmt.Sprintf("%v", v))
		}
		u := b.env.server.URL + op.Path
		if len(q) > 0 {
			u += "?" + q.Encode()
		}
		req, err = http.NewRequest(http.MethodGet, u, nil)
	} else {
		body, merr := json.Marshal(args)
		if merr != nil {
			b.t.Fatalf("%s: marshal args: %v", name, merr)
		}
		req, err = http.NewRequest(http.MethodPost, b.env.server.URL+op.Path, strings.NewReader(string(body)))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
		}
	}
	if err != nil {
		b.t.Fatalf("%s: build request: %v", name, err)
	}
	return b.roundTrip(name, req, wantStatus)
}

// roundTrip sends a prepared request (JSON, query or multipart) and asserts the
// frozen envelope, capturing cookies/csrf/context exactly like request.
func (b *itBrowser) roundTrip(name string, req *http.Request, wantStatus int) map[string]any {
	b.t.Helper()
	req.Header.Set("Origin", b.env.cfg.AllowedOrigin)
	if b.csrf != "" {
		req.Header.Set("X-CSRF-Token", b.csrf)
	}
	if b.context != "" {
		req.Header.Set("X-Client-Context-ID", b.context)
	}
	for n, v := range b.cookies {
		req.AddCookie(&http.Cookie{Name: n, Value: v})
	}
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		b.t.Fatalf("%s: do request: %v", name, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		b.t.Fatalf("%s: read body: %v", name, err)
	}
	var envelope struct {
		Code      string         `json:"code"`
		Message   string         `json:"message"`
		Data      map[string]any `json:"data"`
		RequestID string         `json:"requestId"`
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	if err := dec.Decode(&envelope); err != nil {
		b.t.Fatalf("%s: decode envelope (status %d): %v\nbody: %s", name, resp.StatusCode, err, raw)
	}
	if resp.StatusCode != wantStatus {
		b.t.Fatalf("%s: status = %d, want %d; envelope: %+v", name, resp.StatusCode, wantStatus, envelope)
	}
	if envelope.RequestID == "" || envelope.RequestID != resp.Header.Get("X-Request-ID") {
		b.t.Fatalf("%s: requestId %q missing or != X-Request-ID %q", name, envelope.RequestID, resp.Header.Get("X-Request-ID"))
	}
	if envelope.Data == nil {
		b.t.Fatalf("%s: envelope data must be an object ({} on errors), got null", name)
	}
	if wantStatus >= 200 && wantStatus < 300 {
		if envelope.Code != "OK" || envelope.Message != "操作成功" {
			b.t.Fatalf("%s: success envelope code=%q message=%q, want OK/操作成功", name, envelope.Code, envelope.Message)
		}
	} else if envelope.Code == "" || envelope.Code == "OK" {
		b.t.Fatalf("%s: error envelope must carry a string error code, got %q", name, envelope.Code)
	}
	b.lastCode = envelope.Code
	b.lastMessage = envelope.Message
	for _, ck := range resp.Cookies() {
		b.cookies[ck.Name] = ck.Value
	}
	if v, ok := envelope.Data["csrfToken"].(string); ok && v != "" {
		b.csrf = v
	}
	if v, ok := envelope.Data["clientContextId"].(string); ok && v != "" {
		b.context = v
	}
	return envelope.Data
}

// admin logs in with the fixed initial password (mustChangePassword=true).
func (e *itEnv) admin(t *testing.T) *itBrowser {
	t.Helper()
	b := e.browser(t)
	b.bootstrap()
	b.write("Login", map[string]any{"username": itAdminUsername, "password": itAdminPassword}, http.StatusOK)
	return b
}

// readyAdmin completes the forced initial password change and re-logs in
// (Java WorkflowIntegrationTest.readyAdmin).
func (e *itEnv) readyAdmin(t *testing.T) *itBrowser {
	t.Helper()
	a := e.admin(t)
	a.write("PasswordChange", map[string]any{"oldPassword": itAdminPassword, "newPassword": itChangedAdminPassword}, http.StatusOK)
	a.request("SessionMe", map[string]any{}, http.StatusUnauthorized)
	a.bootstrap()
	a.write("Login", map[string]any{"username": itAdminUsername, "password": itChangedAdminPassword}, http.StatusOK)
	return a
}

const itChangedAdminPassword = "Changed-integration-123!"

// itStr extracts a string field, failing the test when absent/mistyped.
func itStr(t *testing.T, m map[string]any, key string) string {
	t.Helper()
	v, ok := m[key].(string)
	if !ok || v == "" {
		t.Fatalf("field %q: want non-empty string, got %v (%T)", key, m[key], m[key])
	}
	return v
}

// itMap extracts an object field.
func itMap(t *testing.T, m map[string]any, key string) map[string]any {
	t.Helper()
	v, ok := m[key].(map[string]any)
	if !ok {
		t.Fatalf("field %q: want object, got %v (%T)", key, m[key], m[key])
	}
	return v
}

// itNum extracts an integer field (json.Number or float64).
func itNum(t *testing.T, m map[string]any, key string) int64 {
	t.Helper()
	switch n := m[key].(type) {
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			t.Fatalf("field %q: not an integer: %v", key, n)
		}
		return i
	case float64:
		return int64(n)
	default:
		t.Fatalf("field %q: want number, got %v (%T)", key, m[key], m[key])
		return 0
	}
}

// itBool extracts a boolean field.
func itBool(t *testing.T, m map[string]any, key string) bool {
	t.Helper()
	v, ok := m[key].(bool)
	if !ok {
		t.Fatalf("field %q: want bool, got %v (%T)", key, m[key], m[key])
	}
	return v
}
