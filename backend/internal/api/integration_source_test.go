package api

// PostgreSQL integration tests (HTTP level, frozen wire contract) for platform
// connections, event sources and the SourcePull background worker. Port of the
// connection/source/pull half of IntegrationWorkflowTest
// (connectionSecretsNeverReturnAndResourceAccessStaysWithinAssignedEvent,
// sourcePullIsQueuedAndProviderIoRunsOutsideTheDatabaseTransaction,
// oauthStateIsBoundToOriginalAdministratorSessionAndConsumedOnce) plus the
// provider transport semantics of ProviderAdapterTest, driven through the
// injectable service.HTTPDoer so no test ever reaches the real network.
//
// Error assertions use the frozen canonical wire codes (errcode.canonical):
// internal names such as SOURCE_IN_USE / PROVIDER_CONFLICT / OAUTH_STATE_INVALID
// normalize to EVENT_STATE_CONFLICT / FORBIDDEN, so the distinctive frozen
// message pins the exact guard (itGuard).
//
// Run with:
//
//	set -a; . ../.local/go-test.env; set +a
//	go test ./internal/api/ -run 'TestIntegrationConnection|TestIntegrationSource|TestIntegrationPull' -v

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"unigate/backend/internal/service"
	"unigate/backend/internal/store"
)

// Fixed non-production provider credentials only (Java ProviderAdapterTest).
const (
	itClientID            = "cli_it_integration_app"
	itClientSecret        = "integration-test-client-secret-0123456789"
	itProviderAccessToken = "integration-test-provider-access-token"
	itProviderOpenID      = "integration-test-open-id"
	itFeishuResource      = "appItToken/tableItId"
	itTencentResource     = "fileItId/sheetItId"
	itOAuthRedirect       = "https://unigate.example.invalid/api/v1/integration/connection/callback"
)

// itProviderCall records one provider HTTP request for URL/header assertions.
type itProviderCall struct {
	method string
	url    *url.URL
	header http.Header
	body   string
}

// itProviderMock is the injectable service.HTTPDoer: fn scripts the response
// and every request is recorded. It never opens a network connection.
type itProviderMock struct {
	mu    sync.Mutex
	calls []itProviderCall
	fn    func(req *http.Request) (*http.Response, error)
}

func (m *itProviderMock) Do(req *http.Request) (*http.Response, error) {
	body := ""
	if req.Body != nil {
		raw, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		body = string(raw)
	}
	m.mu.Lock()
	m.calls = append(m.calls, itProviderCall{method: req.Method, url: req.URL, header: req.Header.Clone(), body: body})
	m.mu.Unlock()
	return m.fn(req)
}

func (m *itProviderMock) recorded() []itProviderCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]itProviderCall, len(m.calls))
	copy(out, m.calls)
	return out
}

// itProviderResponse builds a canned provider HTTP response.
func itProviderResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}

// itFeishuFieldsJSON renders a Feishu bitable field-metadata page.
func itFeishuFieldsJSON(names ...string) string {
	items := make([]any, 0, len(names))
	for _, n := range names {
		items = append(items, map[string]any{"field_name": n})
	}
	raw, _ := json.Marshal(map[string]any{"code": 0, "data": map[string]any{"items": items, "has_more": false}})
	return string(raw)
}

// itFeishuRecordsJSON renders a Feishu bitable record page; each record is
// {record_id, fields} exactly as the open API returns it.
func itFeishuRecordsJSON(records ...map[string]any) string {
	items := make([]any, 0, len(records))
	for _, r := range records {
		items = append(items, r)
	}
	raw, _ := json.Marshal(map[string]any{"code": 0, "data": map[string]any{"items": items, "has_more": false}})
	return string(raw)
}

// itTencentGridJSON renders a Tencent spreadsheet grid block (ret envelope).
func itTencentGridJSON(rows [][]string) string {
	gridRows := make([]any, 0, len(rows))
	for _, row := range rows {
		values := make([]any, 0, len(row))
		for _, cell := range row {
			values = append(values, map[string]any{"cellValue": map[string]any{"text": cell}})
		}
		gridRows = append(gridRows, map[string]any{"values": values})
	}
	raw, _ := json.Marshal(map[string]any{"ret": 0, "data": map[string]any{"gridData": map[string]any{"rows": gridRows}}})
	return string(raw)
}

// useProviderHTTP swaps svc.HTTP for the scripted mock and restores the real
// client when the test ends (tests run sequentially; the fixture is shared).
func (e *itEnv) useProviderHTTP(t *testing.T, fn func(*http.Request) (*http.Response, error)) *itProviderMock {
	t.Helper()
	m := &itProviderMock{fn: fn}
	prev := e.svc.HTTP
	e.svc.HTTP = m
	t.Cleanup(func() { e.svc.HTTP = prev })
	return m
}

// providerConfig sets unigate.providers.connections.<id>.<name> for the test
// (Spring MockEnvironment.withProperty in the Java tests).
func (e *itEnv) providerConfig(t *testing.T, connectionID, name, value string) {
	t.Helper()
	if e.cfg.Providers.Raw == nil {
		e.cfg.Providers.Raw = map[string]string{}
	}
	key := "unigate.providers.connections." + connectionID + "." + name
	e.cfg.Providers.Raw[key] = value
	t.Cleanup(func() { delete(e.cfg.Providers.Raw, key) })
}

// itConnectionSave creates a FEISHU/TENCENT connection with the fixed test
// application credentials, assigned to the given events.
func itConnectionSave(t *testing.T, e *itEnv, a *itBrowser, provider string, events []any) map[string]any {
	t.Helper()
	return a.write("ConnectionSave", map[string]any{
		"provider":        provider,
		"accountKind":     "PERSONAL",
		"displayName":     "IT " + provider + " connection",
		"allowedEventIds": events,
		"clientId":        itClientID,
		"clientSecret":    itClientSecret,
	}, http.StatusOK)
}

// itStrings extracts a plain string array (allowedEventIds is not enum-valued).
func itStrings(t *testing.T, m map[string]any, key string) []string {
	t.Helper()
	raw, ok := m[key].([]any)
	if !ok {
		t.Fatalf("field %q: want array, got %v (%T)", key, m[key], m[key])
	}
	out := make([]string, 0, len(raw))
	for _, x := range raw {
		s, ok := x.(string)
		if !ok || s == "" {
			t.Fatalf("field %q: want non-empty strings, got %v (%T)", key, x, x)
		}
		out = append(out, s)
	}
	return out
}

// itNoSecret marshals a response view and fails when the fixed secret leaks.
func itNoSecret(t *testing.T, view any, secret string) {
	t.Helper()
	raw, err := json.Marshal(view)
	if err != nil {
		t.Fatalf("marshal view: %v", err)
	}
	if bytes.Contains(raw, []byte(secret)) {
		t.Fatalf("response leaks the provider secret: %s", raw)
	}
}

// itPullTick drives one SourcePullWorker.pullNext tick directly (the worker
// goroutine is not started in the shared fixture, mirroring itRunDueCleanup).
func itPullTick(t *testing.T, e *itEnv) {
	t.Helper()
	if err := service.NewSourcePullWorker(e.svc).PullNext(context.Background()); err != nil {
		t.Fatalf("pull worker tick: %v", err)
	}
}

// TestIntegrationConnectionLifecycleAndSecretSealing ports
// connectionSecretsNeverReturnAndResourceAccessStaysWithinAssignedEvent:
// ConnectionSave seals the client secret (bytea never contains the plaintext),
// ConnectionList/SourceList stay thin, provider changes and stale versions are
// rejected, unassigning a referenced event is blocked, and Revoke clears the
// managed credentials without touching business facts.
func TestIntegrationConnectionLifecycleAndSecretSealing(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)
	ev1 := itPublishedEvent(t, e, a, 10)
	ev2 := itPublishedEvent(t, e, a, 10)

	// Non-ADMIN accounts cannot manage connections at all.
	staff, _ := itNewStaff(t, e, a, "ORGANIZER")
	staff.write("ConnectionSave", map[string]any{
		"provider": "FEISHU", "accountKind": "PERSONAL", "displayName": "Staff connection",
		"allowedEventIds": []any{},
	}, http.StatusForbidden)
	itGuard(t, staff, "ConnectionSave", "FORBIDDEN", "没有操作权限")
	staff.request("ConnectionList", map[string]any{}, http.StatusForbidden)
	itGuard(t, staff, "ConnectionList", "FORBIDDEN", "没有操作权限")

	conn := itConnectionSave(t, e, a, "FEISHU", []any{ev1})
	connID := itStr(t, conn, "connectionId")
	if code := itMap(t, conn, "provider")["code"]; code != "FEISHU" {
		t.Fatalf("provider = %v, want FEISHU", code)
	}
	if code := itMap(t, conn, "accountKind")["code"]; code != "PERSONAL" {
		t.Fatalf("accountKind = %v, want PERSONAL", code)
	}
	if code := itMap(t, conn, "status")["code"]; code != "UNAUTHORIZED" {
		t.Fatalf("fresh connection status = %v, want UNAUTHORIZED", code)
	}
	if got := itStr(t, conn, "credentialMask"); got != "••••••••" {
		t.Fatalf("credentialMask = %q, want the fixed mask", got)
	}
	if ids := itStrings(t, conn, "allowedEventIds"); len(ids) != 1 || ids[0] != ev1 {
		t.Fatalf("allowedEventIds = %v, want [%s]", ids, ev1)
	}
	itNoSecret(t, conn, itClientSecret)

	// The secret is sealed at rest: the bytea column never contains plaintext
	// and unseals back to the exact value (Java s.unseal assertion).
	row := itRow(t, e, "select credentials_cipher,client_id,status from ug_connection where id=$1", connID)
	cipher, ok := row["credentials_cipher"].([]byte)
	if !ok || len(cipher) == 0 {
		t.Fatalf("credentials_cipher must be sealed bytea, got %T", row["credentials_cipher"])
	}
	if bytes.Contains(cipher, []byte(itClientSecret)) {
		t.Fatal("credentials_cipher contains the plaintext secret")
	}
	creds, err := e.s.Unseal(cipher)
	if err != nil {
		t.Fatalf("unseal credentials: %v", err)
	}
	if got := store.StrOrEmpty(creds["clientSecret"]); got != itClientSecret {
		t.Fatalf("unsealed clientSecret = %q, want the fixed test secret", got)
	}
	if got := store.StrOrEmpty(row["client_id"]); got != itClientID {
		t.Fatalf("client_id = %q, want %q", got, itClientID)
	}

	// ConnectionList is a thin masked view (provider filter + paging).
	list := a.request("ConnectionList", map[string]any{"provider": "FEISHU", "pageSize": 100}, http.StatusOK)
	items := itItems(t, list)
	if len(items) != 1 {
		t.Fatalf("ConnectionList items = %d, want 1", len(items))
	}
	itNoSecret(t, list, itClientSecret)
	if itStr(t, items[0].(map[string]any), "connectionId") != connID {
		t.Fatalf("ConnectionList must echo the saved connection: %+v", items[0])
	}
	empty := a.request("ConnectionList", map[string]any{"provider": "TENCENT"}, http.StatusOK)
	if got := itNum(t, empty, "total"); got != 0 {
		t.Fatalf("TENCENT-filtered total = %d, want 0", got)
	}

	// Metadata update keeps the assignment; the provider is immutable.
	renamed := a.write("ConnectionSave", map[string]any{
		"connectionId": connID, "version": conn["version"],
		"provider": "FEISHU", "accountKind": "PERSONAL", "displayName": "Renamed connection",
		"allowedEventIds": []any{ev1},
	}, http.StatusOK)
	if got := itStr(t, renamed, "displayName"); got != "Renamed connection" {
		t.Fatalf("displayName = %q, want Renamed connection", got)
	}
	a.write("ConnectionSave", map[string]any{
		"connectionId": connID, "version": renamed["version"],
		"provider": "TENCENT", "accountKind": "PERSONAL", "displayName": "Switched",
		"allowedEventIds": []any{ev1},
	}, http.StatusConflict)
	itGuard(t, a, "ConnectionSave", "EVENT_STATE_CONFLICT", "既有连接不能更换提供方")

	// Stale versions are rejected everywhere.
	a.write("ConnectionSave", map[string]any{
		"connectionId": connID, "version": 0,
		"provider": "FEISHU", "accountKind": "PERSONAL", "displayName": "Stale",
		"allowedEventIds": []any{ev1},
	}, http.StatusConflict)
	if a.lastCode != "VERSION_CONFLICT" {
		t.Fatalf("stale ConnectionSave code = %q, want VERSION_CONFLICT", a.lastCode)
	}

	// Extend the scope to ev2, then a source on ev2 pins the assignment.
	extended := a.write("ConnectionSave", map[string]any{
		"connectionId": connID, "version": renamed["version"],
		"provider": "FEISHU", "accountKind": "PERSONAL", "displayName": "Renamed connection",
		"allowedEventIds": []any{ev1, ev2},
	}, http.StatusOK)
	if ids := itStrings(t, extended, "allowedEventIds"); len(ids) != 2 {
		t.Fatalf("extended allowedEventIds = %v, want both events", ids)
	}
	source := a.write("SourceSave", map[string]any{
		"eventId": ev2, "connectionId": connID, "resourceId": itFeishuResource,
		"mapping": map[string]any{"nameColumn": "name", "countMode": "COMPANIONS"}, "eligibility": "ELIGIBLE",
	}, http.StatusOK)
	a.write("ConnectionSave", map[string]any{
		"connectionId": connID, "version": extended["version"],
		"provider": "FEISHU", "accountKind": "PERSONAL", "displayName": "Renamed connection",
		"allowedEventIds": []any{ev1},
	}, http.StatusConflict)
	itGuard(t, a, "ConnectionSave", "EVENT_STATE_CONFLICT", "请先解除活动来源对该连接的引用")

	// Unknown event ids are rejected (the whole save rolls back).
	a.write("ConnectionSave", map[string]any{
		"connectionId": connID, "version": extended["version"],
		"provider": "FEISHU", "accountKind": "PERSONAL", "displayName": "Renamed connection",
		"allowedEventIds": []any{ev2, e.s.ID()},
	}, http.StatusNotFound)
	if a.lastCode != "RESOURCE_NOT_FOUND" {
		t.Fatalf("unknown-event ConnectionSave code = %q, want RESOURCE_NOT_FOUND", a.lastCode)
	}

	// Revoke clears the managed credentials; the source survives (historical
	// business eligibility is never cancelled by a disconnect).
	revoked := a.write("ConnectionRevoke", map[string]any{
		"connectionId": connID, "version": extended["version"],
	}, http.StatusOK)
	if code := itMap(t, revoked, "status")["code"]; code != "REVOKED" {
		t.Fatalf("revoked status = %v, want REVOKED", code)
	}
	if _, has := revoked["credentialMask"]; has {
		t.Fatalf("revoked view must drop credentialMask: %+v", revoked)
	}
	after := itRow(t, e, "select credentials_cipher,key_ref,expires_at from ug_connection where id=$1", connID)
	if after["credentials_cipher"] != nil || after["key_ref"] != nil || after["expires_at"] != nil {
		t.Fatalf("revoke must clear credentials/key/expiry: %+v", after)
	}
	listed := a.request("SourceList", map[string]any{"eventId": ev2}, http.StatusOK)
	if got := len(itItems(t, listed)); got != 1 {
		t.Fatalf("sources after revoke = %d, want 1 (facts survive)", got)
	}
	if itStr(t, itItems(t, listed)[0].(map[string]any), "sourceId") != itStr(t, source, "sourceId") {
		t.Fatal("SourceList must keep the original source after revoke")
	}
	a.write("ConnectionRevoke", map[string]any{"connectionId": connID, "version": 0}, http.StatusConflict)
	if a.lastCode != "VERSION_CONFLICT" {
		t.Fatalf("stale revoke code = %q, want VERSION_CONFLICT", a.lastCode)
	}
}

// TestIntegrationConnectionOAuthStateAndCallback ports
// oauthStateIsBoundToOriginalAdministratorSessionAndConsumedOnce plus the
// ProviderAdapterTest redirect allowlist: Authorize builds the official URL
// without leaking the secret, the callback exchanges the code through the
// mocked provider transport, the state is bound to the original admin session
// and consumed exactly once, and unregistered/non-HTTPS redirects fail before
// any network call.
func TestIntegrationConnectionOAuthStateAndCallback(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)

	conn := itConnectionSave(t, e, a, "FEISHU", []any{})
	connID := itStr(t, conn, "connectionId")

	// No redirect configured: capability is honestly unavailable, no network.
	// (The reviewed scope is configured first so the redirect guard is the one
	// that fires, matching providerAuthorizeURL's check order.)
	e.providerConfig(t, connID, "scope", "bitable:app:readonly")
	mock := e.useProviderHTTP(t, func(req *http.Request) (*http.Response, error) {
		t.Errorf("provider HTTP must not be called before the redirect is configured: %s", req.URL)
		return itProviderResponse(200, "{}"), nil
	})
	a.write("ConnectionAuthorize", map[string]any{"connectionId": connID, "version": conn["version"]}, http.StatusServiceUnavailable)
	itGuard(t, a, "ConnectionAuthorize", "PROVIDER_UNAVAILABLE", "授权回调地址未配置")
	if len(mock.recorded()) != 0 {
		t.Fatalf("authorize must not touch the network without a redirect: %+v", mock.recorded())
	}

	// Non-HTTPS / unregistered-path redirects are rejected before the network.
	e.providerConfig(t, connID, "redirect-uri", "http://internal.invalid/callback")
	a.write("ConnectionAuthorize", map[string]any{"connectionId": connID, "version": conn["version"]}, http.StatusServiceUnavailable)
	itGuard(t, a, "ConnectionAuthorize", "PROVIDER_UNAVAILABLE", "需配置已向提供方登记的HTTPS回调地址")
	if len(mock.recorded()) != 0 {
		t.Fatalf("bad redirect must not touch the network: %+v", mock.recorded())
	}

	// Registered HTTPS callback: the authorization URL carries the one-time
	// state and never the application secret.
	e.providerConfig(t, connID, "redirect-uri", itOAuthRedirect)
	auth := a.write("ConnectionAuthorize", map[string]any{"connectionId": connID, "version": conn["version"]}, http.StatusOK)
	authURL := itStr(t, auth, "authorizationUrl")
	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("parse authorizationUrl: %v", err)
	}
	if parsed.Host != "accounts.feishu.cn" || parsed.Path != "/open-apis/authen/v1/authorize" {
		t.Fatalf("authorizationUrl = %s, want the official Feishu endpoint", authURL)
	}
	q := parsed.Query()
	state := q.Get("state")
	if state == "" || q.Get("client_id") != itClientID || q.Get("redirect_uri") != itOAuthRedirect ||
		q.Get("scope") != "bitable:app:readonly" || q.Get("response_type") != "code" {
		t.Fatalf("authorizationUrl query = %v", q)
	}
	itNoSecret(t, auth, itClientSecret)
	if auth["expiresAt"] == nil {
		t.Fatal("AuthorizationView must carry expiresAt")
	}

	// The exchange runs through the mocked transport: POST to the official
	// token endpoint with the secret in the body, never in the URL.
	mock.fn = func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Host != "open.feishu.cn" || req.URL.Path != "/open-apis/authen/v2/oauth/token" {
			t.Errorf("exchange request = %s %s, want POST open.feishu.cn/open-apis/authen/v2/oauth/token", req.Method, req.URL)
		}
		return itProviderResponse(200, `{"access_token":"`+itProviderAccessToken+`","refresh_token":"integration-test-refresh-token","expires_in":3600}`), nil
	}

	// A different session of the same admin account cannot consume the state.
	other := e.browser(t)
	other.bootstrap()
	other.write("Login", map[string]any{"username": itAdminUsername, "password": itChangedAdminPassword}, http.StatusOK)
	other.request("ConnectionCallback", map[string]any{"state": state, "code": "provider-test-code"}, http.StatusForbidden)
	itGuard(t, other, "ConnectionCallback", "FORBIDDEN", "授权状态无效，请重新发起授权")
	if len(mock.recorded()) != 0 {
		t.Fatalf("wrong-session callback must not exchange the code: %+v", mock.recorded())
	}

	// Unknown states are rejected.
	a.request("ConnectionCallback", map[string]any{"state": strings.Repeat("x", 43), "code": "provider-test-code"}, http.StatusNotFound)
	if a.lastCode != "RESOURCE_NOT_FOUND" {
		t.Fatalf("unknown-state callback code = %q, want RESOURCE_NOT_FOUND", a.lastCode)
	}

	// The original session completes the exchange exactly once.
	result := a.request("ConnectionCallback", map[string]any{"state": state, "code": "provider-test-code"}, http.StatusOK)
	if code := itMap(t, result, "status")["code"]; code != "AUTHORIZED" {
		t.Fatalf("callback status = %v, want AUTHORIZED", code)
	}
	itNoSecret(t, result, itProviderAccessToken)
	itNoSecret(t, result, itClientSecret)
	if result["expiresAt"] == nil {
		t.Fatal("authorized connection must carry expiresAt")
	}
	calls := mock.recorded()
	if len(calls) != 1 {
		t.Fatalf("exchange calls = %d, want 1", len(calls))
	}
	if strings.Contains(calls[0].url.String(), itClientSecret) {
		t.Fatal("client secret must stay in the POST body, never the URL")
	}
	var exchangeBody map[string]any
	if err := json.Unmarshal([]byte(calls[0].body), &exchangeBody); err != nil {
		t.Fatalf("exchange body not JSON: %v", err)
	}
	if exchangeBody["client_secret"] != itClientSecret || exchangeBody["code"] != "provider-test-code" ||
		exchangeBody["redirect_uri"] != itOAuthRedirect || exchangeBody["grant_type"] != "authorization_code" {
		t.Fatalf("exchange body = %+v", exchangeBody)
	}

	// The exchanged tokens are sealed at rest and never returned.
	row := itRow(t, e, "select credentials_cipher,status from ug_connection where id=$1", connID)
	if store.StrOrEmpty(row["status"]) != "AUTHORIZED" {
		t.Fatalf("stored status = %v, want AUTHORIZED", row["status"])
	}
	cipher, _ := row["credentials_cipher"].([]byte)
	if bytes.Contains(cipher, []byte(itProviderAccessToken)) {
		t.Fatal("credentials_cipher contains the plaintext access token")
	}
	creds, err := e.s.Unseal(cipher)
	if err != nil {
		t.Fatalf("unseal exchanged credentials: %v", err)
	}
	if store.StrOrEmpty(creds["accessToken"]) != itProviderAccessToken || store.StrOrEmpty(creds["clientSecret"]) != itClientSecret {
		t.Fatalf("exchanged credentials must merge with the application secret: %+v", creds)
	}

	// Replaying the consumed state fails without a second exchange.
	a.request("ConnectionCallback", map[string]any{"state": state, "code": "provider-test-code"}, http.StatusForbidden)
	itGuard(t, a, "ConnectionCallback", "FORBIDDEN", "授权状态无效，请重新发起授权")
	if len(mock.recorded()) != 1 {
		t.Fatalf("replayed state must not exchange again: %d calls", len(mock.recorded()))
	}
}

// itSourceSave binds a source to the connection/resource with a field mapping.
func itSourceSave(t *testing.T, e *itEnv, a *itBrowser, ev, connID, resource string, mapping map[string]any, eligibility string) map[string]any {
	t.Helper()
	return a.write("SourceSave", map[string]any{
		"eventId": ev, "connectionId": connID, "resourceId": resource,
		"mapping": mapping, "eligibility": eligibility,
	}, http.StatusOK)
}

// TestIntegrationSourceResourceListFieldsAndScopeGuard ports the resource half of
// connectionSecretsNeverReturnAndResourceAccessStaysWithinAssignedEvent plus
// ProviderAdapterTest's allowlist/pagination semantics: ResourceList returns
// only the configured authorized resources (metadata only), ResourceFields
// reports the actual provider columns (never guessed headers), and organizers
// cannot address a connection outside their event's allowedEventIds.
func TestIntegrationSourceResourceListFieldsAndScopeGuard(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)
	ev := itPublishedEvent(t, e, a, 10)
	other := itPublishedEvent(t, e, a, 10)

	conn := itConnectionSave(t, e, a, "FEISHU", []any{ev})
	connID := itStr(t, conn, "connectionId")
	e.providerConfig(t, connID, "access-token", itProviderAccessToken)
	e.providerConfig(t, connID, "resources", itFeishuResource)

	// Feishu field metadata follows opaque pagination and reads only /fields.
	mock := e.useProviderHTTP(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.Host != "open.feishu.cn" || !strings.HasSuffix(req.URL.Path, "/fields") {
			t.Errorf("ResourceList/Fields must read the official fields API, got %s", req.URL)
		}
		if got := req.Header.Get("Authorization"); got != "Bearer "+itProviderAccessToken {
			t.Errorf("Authorization = %q, want the configured bearer token", got)
		}
		if req.URL.Query().Get("page_token") == "" {
			return itProviderResponse(200, `{"code":0,"data":{"items":[{"field_name":"name"}],"has_more":true,"page_token":"next+token"}}`), nil
		}
		if got := req.URL.Query().Get("page_token"); got != "next+token" {
			t.Errorf("page_token = %q, want the opaque cursor round-tripped", got)
		}
		return itProviderResponse(200, itFeishuFieldsJSON("total")), nil
	})

	list := a.request("ResourceList", map[string]any{"connectionId": connID}, http.StatusOK)
	items := itItems(t, list)
	if len(items) != 1 {
		t.Fatalf("ResourceList items = %d, want the single configured resource", len(items))
	}
	item := items[0].(map[string]any)
	if itStr(t, item, "resourceId") != itFeishuResource || itStr(t, item, "kind") != "BITABLE" || !itBool(t, item, "readable") {
		t.Fatalf("resource metadata = %+v", item)
	}
	itNoSecret(t, list, itProviderAccessToken)
	// ResourceList probes fields for every configured resource (2 pages here).
	if got := len(mock.recorded()); got != 2 {
		t.Fatalf("ResourceList provider calls = %d, want 2 (paged fields)", got)
	}

	fields := a.request("ResourceFields", map[string]any{"connectionId": connID, "resourceId": itFeishuResource}, http.StatusOK)
	if itStr(t, fields, "resourceId") != itFeishuResource {
		t.Fatalf("ResourceFields resourceId = %v", fields["resourceId"])
	}
	columns := itCodes(t, fields, "availableColumns")
	if len(columns) != 2 || columns[0] != "name" || columns[1] != "total" {
		t.Fatalf("availableColumns = %v, want the actual provider columns [name total]", columns)
	}

	// Resources outside the deployment allowlist never reach the network.
	before := len(mock.recorded())
	a.request("ResourceFields", map[string]any{"connectionId": connID, "resourceId": "https://internal.invalid/path"}, http.StatusForbidden)
	itGuard(t, a, "ResourceFields", "FORBIDDEN", "资源不在当前连接授权范围内")
	// A malformed id inside the allowlist still fails the format guard.
	e.providerConfig(t, connID, "resources", itFeishuResource+",badformat")
	a.request("ResourceFields", map[string]any{"connectionId": connID, "resourceId": "badformat"}, http.StatusUnprocessableEntity)
	itGuard(t, a, "ResourceFields", "FIELD_VALIDATION_FAILED", "资源标识格式应为appToken/tableId或fileId/sheetId")
	e.providerConfig(t, connID, "resources", itFeishuResource)
	if len(mock.recorded()) != before {
		t.Fatalf("allowlist violations must not touch the network: %+v", mock.recorded()[before:])
	}

	// A provider redirect is never treated as success.
	mock.fn = func(req *http.Request) (*http.Response, error) {
		return itProviderResponse(302, ""), nil
	}
	a.request("ResourceFields", map[string]any{"connectionId": connID, "resourceId": itFeishuResource}, http.StatusServiceUnavailable)
	itGuard(t, a, "ResourceFields", "PROVIDER_UNAVAILABLE", "提供方读取失败，请检查授权状态")

	// Organizer scope guard: the connection is assigned to ev only, so the
	// same organizer cannot address it through another event.
	mock.fn = func(req *http.Request) (*http.Response, error) {
		return itProviderResponse(200, itFeishuFieldsJSON("name")), nil
	}
	organizer, orgAccount := itNewStaff(t, e, a, "ORGANIZER")
	itExportGrant(t, e, a, ev, itStr(t, orgAccount, "accountId"), []any{"name"}, []any{"VIEW_DETAIL"})
	itExportGrant(t, e, a, other, itStr(t, orgAccount, "accountId"), []any{"name"}, []any{"VIEW_DETAIL"})
	before = len(mock.recorded())
	organizer.request("ResourceFields", map[string]any{"eventId": other, "connectionId": connID, "resourceId": itFeishuResource}, http.StatusNotFound)
	if organizer.lastCode != "RESOURCE_NOT_FOUND" {
		t.Fatalf("cross-event ResourceFields code = %q, want RESOURCE_NOT_FOUND", organizer.lastCode)
	}
	if len(mock.recorded()) != before {
		t.Fatal("cross-event resource access must not touch the provider")
	}
	ok := organizer.request("ResourceFields", map[string]any{"eventId": ev, "connectionId": connID, "resourceId": itFeishuResource}, http.StatusOK)
	if got := itCodes(t, ok, "availableColumns"); len(got) != 1 || got[0] != "name" {
		t.Fatalf("assigned-event availableColumns = %v, want [name]", got)
	}
	organizer.request("ResourceList", map[string]any{"eventId": ev, "connectionId": connID}, http.StatusOK)

	// Tencent resources use header authentication (Client-Id/Open-Id/Access-Token)
	// and read the real header row, never guessed columns.
	tconn := itConnectionSave(t, e, a, "TENCENT", []any{ev})
	tconnID := itStr(t, tconn, "connectionId")
	e.providerConfig(t, tconnID, "access-token", itProviderAccessToken)
	e.providerConfig(t, tconnID, "resources", itTencentResource)
	e.providerConfig(t, tconnID, "open-id", itProviderOpenID)
	tmock := e.useProviderHTTP(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.Host != "docs.qq.com" {
			t.Errorf("tencent host = %s, want docs.qq.com", req.URL.Host)
		}
		if got := req.Header.Get("Client-Id"); got != itClientID {
			t.Errorf("Client-Id = %q, want %q", got, itClientID)
		}
		if got := req.Header.Get("Open-Id"); got != itProviderOpenID {
			t.Errorf("Open-Id = %q, want %q", got, itProviderOpenID)
		}
		if got := req.Header.Get("Access-Token"); got != itProviderAccessToken {
			t.Errorf("Access-Token = %q, want the configured token", got)
		}
		if !strings.HasSuffix(req.URL.Path, "/A1:GR1") {
			t.Errorf("header read range = %s, want A1:GR1", req.URL.Path)
		}
		return itProviderResponse(200, itTencentGridJSON([][]string{{"responseId", "name", "total"}})), nil
	})
	tfields := a.request("ResourceFields", map[string]any{"connectionId": tconnID, "resourceId": itTencentResource}, http.StatusOK)
	if got := itCodes(t, tfields, "availableColumns"); len(got) != 3 || got[0] != "responseId" || got[2] != "total" {
		t.Fatalf("tencent availableColumns = %v, want the actual header row", got)
	}
	if len(tmock.recorded()) != 1 {
		t.Fatalf("tencent fields calls = %d, want 1", len(tmock.recorded()))
	}
	tlist := a.request("ResourceList", map[string]any{"connectionId": tconnID}, http.StatusOK)
	titem := itItems(t, tlist)[0].(map[string]any)
	if itStr(t, titem, "kind") != "SPREADSHEET" {
		t.Fatalf("tencent resource kind = %v, want SPREADSHEET", titem["kind"])
	}

	// Missing header row / duplicate headers are explicit data failures.
	tmock.fn = func(req *http.Request) (*http.Response, error) {
		return itProviderResponse(200, itTencentGridJSON([][]string{})), nil
	}
	a.request("ResourceFields", map[string]any{"connectionId": tconnID, "resourceId": itTencentResource}, http.StatusUnprocessableEntity)
	itGuard(t, a, "ResourceFields", "FIELD_VALIDATION_FAILED", "来源缺少表头")
	tmock.fn = func(req *http.Request) (*http.Response, error) {
		return itProviderResponse(200, itTencentGridJSON([][]string{{"name", "name"}})), nil
	}
	a.request("ResourceFields", map[string]any{"connectionId": tconnID, "resourceId": itTencentResource}, http.StatusUnprocessableEntity)
	itGuard(t, a, "ResourceFields", "FIELD_VALIDATION_FAILED", "来源表头须唯一非空")
}

// TestIntegrationSourceSaveListAndGuards covers SourceSave binding/validation
// (connection scope, provider whitelist, intake freeze, optimistic versions),
// SourceList's thin view with availableConnections, and the mapping round-trip.
func TestIntegrationSourceSaveListAndGuards(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)
	ev := itPublishedEvent(t, e, a, 10)
	other := itPublishedEvent(t, e, a, 10)

	conn := itConnectionSave(t, e, a, "FEISHU", []any{ev})
	connID := itStr(t, conn, "connectionId")

	// A connection not assigned to the event cannot back a source.
	a.write("SourceSave", map[string]any{
		"eventId": other, "connectionId": connID, "resourceId": itFeishuResource,
		"mapping": map[string]any{"countMode": "COMPANIONS"}, "eligibility": "ELIGIBLE",
	}, http.StatusNotFound)
	if a.lastCode != "RESOURCE_NOT_FOUND" {
		t.Fatalf("unassigned-connection SourceSave code = %q, want RESOURCE_NOT_FOUND", a.lastCode)
	}

	source := itSourceSave(t, e, a, ev, connID, itFeishuResource,
		map[string]any{"nameColumn": "name", "countMode": "COMPANIONS"}, "REVIEW_REQUIRED")
	sourceID := itStr(t, source, "sourceId")
	if itStr(t, source, "eventId") != ev || itStr(t, source, "connectionId") != connID || itStr(t, source, "resourceId") != itFeishuResource {
		t.Fatalf("source binding = %+v", source)
	}
	if code := itMap(t, source, "provider")["code"]; code != "FEISHU" {
		t.Fatalf("source provider = %v, want FEISHU", code)
	}
	if code := itMap(t, source, "eligibility")["code"]; code != "REVIEW_REQUIRED" {
		t.Fatalf("source eligibility = %v, want REVIEW_REQUIRED", code)
	}
	mapping := itMap(t, source, "mapping")
	if itStr(t, mapping, "nameColumn") != "name" {
		t.Fatalf("stored mapping nameColumn = %v", mapping["nameColumn"])
	}
	if code := itMap(t, mapping, "countMode")["code"]; code != "COMPANIONS" {
		t.Fatalf("stored mapping countMode = %v, want COMPANIONS", code)
	}

	// SourceList: items + availableConnections, never any secret.
	listed := a.request("SourceList", map[string]any{"eventId": ev}, http.StatusOK)
	if got := len(itItems(t, listed)); got != 1 {
		t.Fatalf("SourceList items = %d, want 1", got)
	}
	available := listed["availableConnections"].([]any)
	if len(available) != 1 {
		t.Fatalf("availableConnections = %d, want 1", len(available))
	}
	av := available[0].(map[string]any)
	if itStr(t, av, "connectionId") != connID {
		t.Fatalf("availableConnections[0] = %+v", av)
	}
	if _, has := av["credentialMask"]; has {
		t.Fatalf("SourceList availableConnections must stay thin: %+v", av)
	}
	itNoSecret(t, listed, itClientSecret)

	// Update with the current version; stale versions conflict.
	updated := a.write("SourceSave", map[string]any{
		"eventId": ev, "sourceId": sourceID, "version": source["version"],
		"connectionId": connID, "resourceId": "appItToken/tableNewId",
		"mapping": map[string]any{"nameColumn": "name", "countColumn": "total", "countMode": "TOTAL"}, "eligibility": "ELIGIBLE",
	}, http.StatusOK)
	if itStr(t, updated, "resourceId") != "appItToken/tableNewId" {
		t.Fatalf("updated resourceId = %v", updated["resourceId"])
	}
	if code := itMap(t, updated, "eligibility")["code"]; code != "ELIGIBLE" {
		t.Fatalf("updated eligibility = %v, want ELIGIBLE", code)
	}
	a.write("SourceSave", map[string]any{
		"eventId": ev, "sourceId": sourceID, "version": source["version"],
		"connectionId": connID, "resourceId": itFeishuResource,
		"mapping": map[string]any{"countMode": "COMPANIONS"}, "eligibility": "ELIGIBLE",
	}, http.StatusConflict)
	if a.lastCode != "VERSION_CONFLICT" {
		t.Fatalf("stale SourceSave code = %q, want VERSION_CONFLICT", a.lastCode)
	}

	// Non-feishu/tencent connections cannot back a roster source.
	email := a.write("ConnectionSave", map[string]any{
		"provider": "EMAIL", "accountKind": "ORGANIZATION", "displayName": "IT email",
		"allowedEventIds": []any{ev},
	}, http.StatusOK)
	a.write("SourceSave", map[string]any{
		"eventId": ev, "connectionId": itStr(t, email, "connectionId"), "resourceId": itFeishuResource,
		"mapping": map[string]any{"countMode": "COMPANIONS"}, "eligibility": "ELIGIBLE",
	}, http.StatusUnprocessableEntity)
	itGuard(t, a, "SourceSave", "FIELD_VALIDATION_FAILED", "来源仅支持飞书或腾讯文档")

	// Check-in freezes source mutations.
	itTransition(t, a, ev, "OPEN", "CHECKIN_OPEN")
	a.write("SourceSave", map[string]any{
		"eventId": ev, "sourceId": sourceID, "version": updated["version"],
		"connectionId": connID, "resourceId": itFeishuResource,
		"mapping": map[string]any{"countMode": "COMPANIONS"}, "eligibility": "ELIGIBLE",
	}, http.StatusConflict)
	itGuard(t, a, "SourceSave", "EVENT_STATE_CONFLICT", "检票开始后名单导入已冻结")
	a.write("SourcePull", map[string]any{
		"eventId": ev, "sourceId": sourceID, "version": updated["version"],
	}, http.StatusConflict)
	itGuard(t, a, "SourcePull", "EVENT_STATE_CONFLICT", "检票开始后名单导入已冻结")
}

// itJSONNumber renders a json.Number-friendly numeric cell for record fields.
func itJSONNumber(s string) json.Number { return json.Number(s) }

// TestIntegrationPullWorkerFeishuSuccess ports
// sourcePullIsQueuedAndProviderIoRunsOutsideTheDatabaseTransaction: SourcePull
// only queues a PULL job (202, no batch yet); the worker claims it with SKIP
// LOCKED, calls the provider after the claim transaction committed (observed
// from a second connection), and finishes with a sealed batch + import rows
// that continue through BatchPreview/BatchCommit without allocating early.
func TestIntegrationPullWorkerFeishuSuccess(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)
	ev := itPublishedEvent(t, e, a, 10)

	conn := itConnectionSave(t, e, a, "FEISHU", []any{ev})
	connID := itStr(t, conn, "connectionId")
	e.providerConfig(t, connID, "access-token", itProviderAccessToken)
	e.providerConfig(t, connID, "resources", itFeishuResource)
	source := itSourceSave(t, e, a, ev, connID, itFeishuResource,
		map[string]any{"nameColumn": "name", "countColumn": "total", "countMode": "TOTAL"}, "ELIGIBLE")
	sourceID := itStr(t, source, "sourceId")

	// 202 acceptance: a PENDING job, no batch, no allocation.
	job := a.write("SourcePull", map[string]any{
		"eventId": ev, "sourceId": sourceID, "version": source["version"],
	}, http.StatusAccepted)
	jobID := itStr(t, job, "jobId")
	if code := itMap(t, job, "status")["code"]; code != "PENDING" {
		t.Fatalf("queued job status = %v, want PENDING", code)
	}
	if code := itMap(t, job, "kind")["code"]; code != "PULL" {
		t.Fatalf("job kind = %v, want PULL", code)
	}
	if got := itNum(t, job, "attempts"); got != 0 {
		t.Fatalf("queued attempts = %d, want 0", got)
	}
	if job["resultRef"] != nil {
		t.Fatalf("queued job must not carry resultRef: %+v", job)
	}
	if got := itCount(t, e, "select count(*) from ug_import_batch where event_id=$1", ev); got != 0 {
		t.Fatalf("batches before the worker = %d, want 0", got)
	}

	// The mock asserts, from a second pool connection, that the claim already
	// committed (job RUNNING) and no event row lock is held during provider IO.
	var ioChecks []string
	mock := e.useProviderHTTP(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.Host != "open.feishu.cn" || !strings.HasSuffix(req.URL.Path, "/records") {
			t.Errorf("pull must read the official records API, got %s", req.URL)
		}
		if got := req.Header.Get("Authorization"); got != "Bearer "+itProviderAccessToken {
			t.Errorf("Authorization = %q, want the configured bearer token", got)
		}
		ctx := context.Background()
		status := itRow(t, e, "select status,lease_until from ug_job where id=$1", jobID)
		if store.StrOrEmpty(status["status"]) != "RUNNING" || status["lease_until"] == nil {
			ioChecks = append(ioChecks, fmt.Sprintf("claim not committed during provider IO: %+v", status))
		}
		if _, err := e.s.Exec(ctx, nil, "select id from ug_event where id=$1 for update nowait", ev); err != nil {
			ioChecks = append(ioChecks, fmt.Sprintf("event row locked during provider IO: %v", err))
		}
		return itProviderResponse(200, itFeishuRecordsJSON(
			map[string]any{"record_id": "rec_stable_1", "fields": map[string]any{"name": []any{map[string]any{"text": "Remote Parent"}}, "total": itJSONNumber("2")}},
			map[string]any{"record_id": "rec_stable_2", "fields": map[string]any{"name": "Second Parent", "total": itJSONNumber("1.0")}},
		)), nil
	})

	itPullTick(t, e)
	if len(ioChecks) != 0 {
		t.Fatalf("provider IO ran inside the claim transaction: %v", ioChecks)
	}
	if len(mock.recorded()) != 1 {
		t.Fatalf("pull provider calls = %d, want 1", len(mock.recorded()))
	}

	// JobDetail reports SUCCEEDED with the batch resultRef.
	detail := a.request("JobDetail", map[string]any{"jobId": jobID, "eventId": ev}, http.StatusOK)
	if code := itMap(t, detail, "status")["code"]; code != "SUCCEEDED" {
		t.Fatalf("finished job status = %v, want SUCCEEDED", code)
	}
	if got := itNum(t, detail, "attempts"); got != 1 {
		t.Fatalf("finished attempts = %d, want 1", got)
	}
	batchID := itStr(t, detail, "resultRef")
	if detail["completedAt"] == nil {
		t.Fatal("succeeded job must carry completedAt")
	}
	if detail["errorCode"] != nil {
		t.Fatalf("succeeded job errorCode = %v, want null", detail["errorCode"])
	}

	// The batch snapshots the source mapping/eligibility; rows are sealed with
	// stable provider record ids and normalized plain values.
	batch := itRow(t, e, "select source_id,mapping,eligibility,status,source_version from ug_import_batch where id=$1", batchID)
	if store.StrOrEmpty(batch["source_id"]) != sourceID || store.StrOrEmpty(batch["status"]) != "UPLOADED" {
		t.Fatalf("pull batch = %+v", batch)
	}
	if store.StrOrEmpty(batch["eligibility"]) != "ELIGIBLE" {
		t.Fatalf("pull batch eligibility = %v, want the source ELIGIBLE", batch["eligibility"])
	}
	rows, err := e.s.Rows(context.Background(), nil,
		"select source_record_id,line_number,row_cipher,content_digest from ug_import_row where batch_id=$1 order by line_number", batchID)
	if err != nil {
		t.Fatalf("list import rows: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("import rows = %d, want 2", len(rows))
	}
	if store.StrOrEmpty(rows[0]["source_record_id"]) != "rec_stable_1" || store.StrOrEmpty(rows[1]["source_record_id"]) != "rec_stable_2" {
		t.Fatalf("stable record ids = %v / %v", rows[0]["source_record_id"], rows[1]["source_record_id"])
	}
	for _, r := range rows {
		cipher, _ := r["row_cipher"].([]byte)
		if bytes.Contains(cipher, []byte("Remote Parent")) || bytes.Contains(cipher, []byte("Second Parent")) {
			t.Fatal("import row_cipher contains plaintext personal data")
		}
		if len(r["content_digest"].([]byte)) == 0 {
			t.Fatal("import row must carry a content digest")
		}
	}
	plain, err := e.s.Unseal(rows[0]["row_cipher"])
	if err != nil {
		t.Fatalf("unseal import row: %v", err)
	}
	raw := store.Parse(plain["raw"])
	if store.StrOrEmpty(raw["name"]) != "Remote Parent" {
		t.Fatalf("flattened feishu text = %v, want Remote Parent", raw["name"])
	}
	if store.StrOrEmpty(raw["total"]) != "2" {
		t.Fatalf("normalized number = %v, want 2", raw["total"])
	}
	plain2, _ := e.s.Unseal(rows[1]["row_cipher"])
	if store.StrOrEmpty(store.Parse(plain2["raw"])["total"]) != "1" {
		t.Fatalf("1.0 must normalize to 1, got %v", store.Parse(plain2["raw"])["total"])
	}
	// Pulling never allocates places by itself.
	if got := itNum(t, itGateStats(t, a, ev), "allocatedPeople"); got != 0 {
		t.Fatalf("allocatedPeople after pull = %d, want 0", got)
	}

	// resultRef continues the normal import flow: preview -> rows -> commit.
	view := a.request("BatchDetail", map[string]any{"eventId": ev, "batchId": batchID}, http.StatusOK)
	if code := itMap(t, view, "status")["code"]; code != "UPLOADED" {
		t.Fatalf("pull batch view status = %v, want UPLOADED", code)
	}
	if itStr(t, view, "sourceId") != sourceID {
		t.Fatalf("batch view sourceId = %v, want %s", view["sourceId"], sourceID)
	}
	preview := itPreviewBatch(t, a, ev, view, map[string]any{
		"nameColumn": "name", "countColumn": "total", "countMode": "TOTAL",
	})
	if got := itNum(t, preview, "validRows"); got != 2 {
		t.Fatalf("pull preview validRows = %d, want 2", got)
	}
	selected := itRowIDs(t, a, ev, preview)
	commit := a.write("BatchCommit", map[string]any{
		"eventId": ev, "batchId": batchID, "previewVersion": preview["previewVersion"],
		"selectedRowIds": selected,
	}, http.StatusOK)
	if got := itNum(t, commit, "createdCount"); got != 2 {
		t.Fatalf("createdCount = %d, want 2", got)
	}
	// TOTAL 2 + TOTAL 1 = 3 people.
	if got := itNum(t, itGateStats(t, a, ev), "allocatedPeople"); got != 3 {
		t.Fatalf("allocatedPeople after commit = %d, want 3", got)
	}
	itRowCount(t, a, ev, 2)

	// A second pull of the unchanged source links the same records: the rows
	// surface ALREADY_LINKED and cannot allocate again.
	job2 := a.write("SourcePull", map[string]any{
		"eventId": ev, "sourceId": sourceID, "version": source["version"],
	}, http.StatusAccepted)
	itPullTick(t, e)
	detail2 := a.request("JobDetail", map[string]any{"jobId": itStr(t, job2, "jobId"), "eventId": ev}, http.StatusOK)
	if code := itMap(t, detail2, "status")["code"]; code != "SUCCEEDED" {
		t.Fatalf("second pull status = %v, want SUCCEEDED", code)
	}
	batch2ID := itStr(t, detail2, "resultRef")
	view2 := a.request("BatchDetail", map[string]any{"eventId": ev, "batchId": batch2ID}, http.StatusOK)
	preview2 := itPreviewBatch(t, a, ev, view2, map[string]any{
		"nameColumn": "name", "countColumn": "total", "countMode": "TOTAL",
	})
	linked := a.request("BatchRows", map[string]any{"eventId": ev, "batchId": batch2ID, "status": "ALREADY_LINKED"}, http.StatusOK)
	if got := itNum(t, linked, "total"); got != 2 {
		t.Fatalf("ALREADY_LINKED rows = %d, want 2 (stable provider ids)", got)
	}
	if got := itNum(t, preview2, "validRows"); got != 0 {
		t.Fatalf("re-pull validRows = %d, want 0", got)
	}
	itRowCount(t, a, ev, 2)
}

// TestIntegrationPullWorkerTencentSuccess ports the Tencent half of
// ProviderAdapterTest (tencentRequiresStableIdColumnAndUsesHeaderAuthentication):
// the pull reads the real header row, requires the configured stable record-id
// column, authenticates with Client-Id/Open-Id/Access-Token headers, and seals
// the grid rows into the same previewable batch shape.
func TestIntegrationPullWorkerTencentSuccess(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)
	ev := itPublishedEvent(t, e, a, 10)

	conn := itConnectionSave(t, e, a, "TENCENT", []any{ev})
	connID := itStr(t, conn, "connectionId")
	e.providerConfig(t, connID, "access-token", itProviderAccessToken)
	e.providerConfig(t, connID, "resources", itTencentResource)
	e.providerConfig(t, connID, "open-id", itProviderOpenID)
	source := itSourceSave(t, e, a, ev, connID, itTencentResource,
		map[string]any{"nameColumn": "name", "countColumn": "total", "countMode": "TOTAL"}, "ELIGIBLE")
	sourceID := itStr(t, source, "sourceId")

	// Without a stable record-id column the pull fails explicitly (row numbers
	// are movable and must never identify people). The header metadata may be
	// read, but no record block is fetched.
	mock := e.useProviderHTTP(t, func(req *http.Request) (*http.Response, error) {
		if strings.HasSuffix(req.URL.Path, "/A1:GR1") {
			return itProviderResponse(200, itTencentGridJSON([][]string{{"responseId", "name", "total"}})), nil
		}
		t.Errorf("pull must not read records before the id column is configured: %s", req.URL)
		return itProviderResponse(200, "{}"), nil
	})
	job := a.write("SourcePull", map[string]any{
		"eventId": ev, "sourceId": sourceID, "version": source["version"],
	}, http.StatusAccepted)
	jobID := itStr(t, job, "jobId")
	itPullTick(t, e)
	detail := a.request("JobDetail", map[string]any{"jobId": jobID, "eventId": ev}, http.StatusOK)
	if code := itMap(t, detail, "status")["code"]; code != "FAILED" {
		t.Fatalf("missing-id-column job status = %v, want FAILED (business conflict)", code)
	}
	if got := itStr(t, detail, "errorCode"); got != "FIELD_VALIDATION_FAILED" {
		t.Fatalf("errorCode = %q, want canonical FIELD_VALIDATION_FAILED", got)
	}
	if got := len(mock.recorded()); got != 1 {
		t.Fatalf("missing id column must fail after the header only, got %d calls: %+v", got, mock.recorded())
	}

	// Configure the stable id column and pull: header block + data block.
	e.providerConfig(t, connID, "record-id-column", "responseId")
	mock.fn = func(req *http.Request) (*http.Response, error) {
		if req.URL.Host != "docs.qq.com" {
			t.Errorf("tencent host = %s, want docs.qq.com", req.URL.Host)
		}
		if got := req.Header.Get("Client-Id"); got != itClientID {
			t.Errorf("Client-Id = %q, want %q", got, itClientID)
		}
		if got := req.Header.Get("Open-Id"); got != itProviderOpenID {
			t.Errorf("Open-Id = %q, want %q", got, itProviderOpenID)
		}
		if got := req.Header.Get("Access-Token"); got != itProviderAccessToken {
			t.Errorf("Access-Token = %q, want the configured token", got)
		}
		path := req.URL.Path
		switch {
		case strings.HasSuffix(path, "/A1:GR1"):
			return itProviderResponse(200, itTencentGridJSON([][]string{{"responseId", "name", "total"}})), nil
		case strings.HasSuffix(path, "/A2:C1001"):
			return itProviderResponse(200, itTencentGridJSON([][]string{
				{"answer-42", "Tencent Parent", "2"},
				{"answer-43", "Second Parent", "1"},
			})), nil
		default:
			t.Errorf("unexpected tencent range read: %s", path)
			return itProviderResponse(200, itTencentGridJSON([][]string{})), nil
		}
	}
	before := len(mock.recorded())
	job2 := a.write("SourcePull", map[string]any{
		"eventId": ev, "sourceId": sourceID, "version": source["version"],
	}, http.StatusAccepted)
	itPullTick(t, e)
	detail2 := a.request("JobDetail", map[string]any{"jobId": itStr(t, job2, "jobId"), "eventId": ev}, http.StatusOK)
	if code := itMap(t, detail2, "status")["code"]; code != "SUCCEEDED" {
		t.Fatalf("tencent pull status = %v, want SUCCEEDED (detail %+v)", code, detail2)
	}
	batchID := itStr(t, detail2, "resultRef")
	rows, err := e.s.Rows(context.Background(), nil,
		"select source_record_id,row_cipher from ug_import_row where batch_id=$1 order by line_number", batchID)
	if err != nil {
		t.Fatalf("list tencent import rows: %v", err)
	}
	if len(rows) != 2 || store.StrOrEmpty(rows[0]["source_record_id"]) != "answer-42" {
		t.Fatalf("tencent rows = %+v, want the stable responseId ids", rows)
	}
	plain, err := e.s.Unseal(rows[0]["row_cipher"])
	if err != nil {
		t.Fatalf("unseal tencent row: %v", err)
	}
	raw := store.Parse(plain["raw"])
	if store.StrOrEmpty(raw["name"]) != "Tencent Parent" || store.StrOrEmpty(raw["responseId"]) != "answer-42" {
		t.Fatalf("tencent raw row = %+v", raw)
	}
	// Header + one data block, all header-authenticated.
	calls := mock.recorded()[before:]
	if len(calls) != 2 {
		t.Fatalf("tencent pull calls = %d, want 2 (header + data block)", len(calls))
	}

	// The batch previews and commits like any other roster intake.
	view := a.request("BatchDetail", map[string]any{"eventId": ev, "batchId": batchID}, http.StatusOK)
	preview := itPreviewBatch(t, a, ev, view, map[string]any{
		"nameColumn": "name", "countColumn": "total", "countMode": "TOTAL",
	})
	if got := itNum(t, preview, "validRows"); got != 2 {
		t.Fatalf("tencent preview validRows = %d, want 2", got)
	}
	commit := a.write("BatchCommit", map[string]any{
		"eventId": ev, "batchId": batchID, "previewVersion": preview["previewVersion"],
		"selectedRowIds": itRowIDs(t, a, ev, preview),
	}, http.StatusOK)
	if got := itNum(t, commit, "createdCount"); got != 2 {
		t.Fatalf("tencent createdCount = %d, want 2", got)
	}
	if got := itNum(t, itGateStats(t, a, ev), "allocatedPeople"); got != 3 {
		t.Fatalf("tencent allocatedPeople = %d, want 3 (TOTAL 2 + 1)", got)
	}

	// Formula cells are rejected as invalid provider data (never imported).
	mock.fn = func(req *http.Request) (*http.Response, error) {
		if strings.HasSuffix(req.URL.Path, "/A1:GR1") {
			return itProviderResponse(200, itTencentGridJSON([][]string{{"responseId", "name"}})), nil
		}
		return itProviderResponse(200, `{"ret":0,"data":{"gridData":{"rows":[{"values":[{"cellValue":{"text":"answer-9"}},{"cellValue":{"formula":"=1+1"}}]}]}}}`), nil
	}
	job3 := a.write("SourcePull", map[string]any{
		"eventId": ev, "sourceId": sourceID, "version": source["version"],
	}, http.StatusAccepted)
	itPullTick(t, e)
	detail3 := a.request("JobDetail", map[string]any{"jobId": itStr(t, job3, "jobId"), "eventId": ev}, http.StatusOK)
	if code := itMap(t, detail3, "status")["code"]; code != "FAILED" {
		t.Fatalf("formula-cell pull status = %v, want FAILED", code)
	}
	if got := itStr(t, detail3, "errorCode"); got != "FIELD_VALIDATION_FAILED" {
		t.Fatalf("formula errorCode = %q, want FIELD_VALIDATION_FAILED", got)
	}
}

// TestIntegrationPullWorkerRetryFailureAndJobRetry covers the failure ladder:
// transient PROVIDER_UNAVAILABLE retries with backoff while attempts<5, the
// fifth attempt fails permanently, business conflicts fail immediately,
// JobRetry only accepts FAILED jobs (202 requeue, attempts reset), and a
// changed source version blocks the retry with PREVIEW_CONFLICT.
func TestIntegrationPullWorkerRetryFailureAndJobRetry(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)
	ev := itPublishedEvent(t, e, a, 10)

	conn := itConnectionSave(t, e, a, "FEISHU", []any{ev})
	connID := itStr(t, conn, "connectionId")
	e.providerConfig(t, connID, "access-token", itProviderAccessToken)
	e.providerConfig(t, connID, "resources", itFeishuResource)
	source := itSourceSave(t, e, a, ev, connID, itFeishuResource,
		map[string]any{"nameColumn": "name", "countMode": "COMPANIONS"}, "ELIGIBLE")
	sourceID := itStr(t, source, "sourceId")

	// Transport failures are transient: PENDING with backoff, never FAILED.
	mock := e.useProviderHTTP(t, func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("simulated provider transport failure")
	})
	job := a.write("SourcePull", map[string]any{
		"eventId": ev, "sourceId": sourceID, "version": source["version"],
	}, http.StatusAccepted)
	jobID := itStr(t, job, "jobId")
	itPullTick(t, e)
	detail := a.request("JobDetail", map[string]any{"jobId": jobID, "eventId": ev}, http.StatusOK)
	if code := itMap(t, detail, "status")["code"]; code != "PENDING" {
		t.Fatalf("transient failure status = %v, want PENDING (retry)", code)
	}
	if got := itNum(t, detail, "attempts"); got != 1 {
		t.Fatalf("attempts after first failure = %d, want 1", got)
	}
	if got := itStr(t, detail, "errorCode"); got != "PROVIDER_UNAVAILABLE" {
		t.Fatalf("errorCode = %q, want PROVIDER_UNAVAILABLE", got)
	}
	if detail["completedAt"] != nil {
		t.Fatalf("retrying job must not be completed: %+v", detail)
	}
	row := itRow(t, e, "select next_run_at,lease_until from ug_job where id=$1", jobID)
	nextRun, err := store.Time(row["next_run_at"])
	if err != nil {
		t.Fatalf("parse next_run_at: %v", err)
	}
	if delay := time.Until(nextRun); delay < 30*time.Second || delay > 2*time.Minute {
		t.Fatalf("first backoff = %v, want ~1 minute", delay)
	}
	if row["lease_until"] != nil {
		t.Fatal("failed claim must release the lease")
	}

	// A retrying job cannot be retried manually (only FAILED is retryable).
	a.write("JobRetry", map[string]any{"jobId": jobID, "eventId": ev, "version": detail["version"]}, http.StatusConflict)
	itGuard(t, a, "JobRetry", "EVENT_STATE_CONFLICT", "只能重试失败任务")

	// Exhaust the five attempts: each due tick reclaims and fails again.
	for i := 2; i <= 5; i++ {
		if _, err := e.s.Exec(context.Background(), nil,
			"update ug_job set next_run_at=now()-interval '1 second' where id=$1", jobID); err != nil {
			t.Fatalf("make job due: %v", err)
		}
		itPullTick(t, e)
		detail = a.request("JobDetail", map[string]any{"jobId": jobID, "eventId": ev}, http.StatusOK)
		if got := itNum(t, detail, "attempts"); got != int64(i) {
			t.Fatalf("attempts after tick %d = %d", i, got)
		}
	}
	if code := itMap(t, detail, "status")["code"]; code != "FAILED" {
		t.Fatalf("exhausted job status = %v, want FAILED", code)
	}
	if detail["completedAt"] == nil {
		t.Fatal("failed job must carry completedAt")
	}
	if got := len(mock.recorded()); got != 5 {
		t.Fatalf("provider attempts = %d, want 5", got)
	}

	// JobRetry requeues the FAILED job (202, attempts reset, error cleared).
	retried := a.write("JobRetry", map[string]any{"jobId": jobID, "eventId": ev, "version": detail["version"]}, http.StatusAccepted)
	if code := itMap(t, retried, "status")["code"]; code != "PENDING" {
		t.Fatalf("retried status = %v, want PENDING", code)
	}
	if got := itNum(t, retried, "attempts"); got != 0 {
		t.Fatalf("retried attempts = %d, want 0", got)
	}
	if retried["errorCode"] != nil {
		t.Fatalf("retried errorCode = %v, want null", retried["errorCode"])
	}
	mock.fn = func(req *http.Request) (*http.Response, error) {
		return itProviderResponse(200, itFeishuRecordsJSON(
			map[string]any{"record_id": "rec_retry", "fields": map[string]any{"name": "Recovered Parent"}})), nil
	}
	itPullTick(t, e)
	detail = a.request("JobDetail", map[string]any{"jobId": jobID, "eventId": ev}, http.StatusOK)
	if code := itMap(t, detail, "status")["code"]; code != "SUCCEEDED" {
		t.Fatalf("recovered job status = %v, want SUCCEEDED", code)
	}
	if itStr(t, detail, "resultRef") == "" {
		t.Fatal("recovered job must carry the batch resultRef")
	}
	// SUCCEEDED jobs are not retryable.
	a.write("JobRetry", map[string]any{"jobId": jobID, "eventId": ev, "version": detail["version"]}, http.StatusConflict)
	itGuard(t, a, "JobRetry", "EVENT_STATE_CONFLICT", "只能重试失败任务")

	// Business conflicts fail immediately without retry: duplicate stable ids.
	mock.fn = func(req *http.Request) (*http.Response, error) {
		return itProviderResponse(200, itFeishuRecordsJSON(
			map[string]any{"record_id": "rec_dup", "fields": map[string]any{"name": "A"}},
			map[string]any{"record_id": "rec_dup", "fields": map[string]any{"name": "B"}})), nil
	}
	job2 := a.write("SourcePull", map[string]any{
		"eventId": ev, "sourceId": sourceID, "version": source["version"],
	}, http.StatusAccepted)
	job2ID := itStr(t, job2, "jobId")
	itPullTick(t, e)
	detail2 := a.request("JobDetail", map[string]any{"jobId": job2ID, "eventId": ev}, http.StatusOK)
	if code := itMap(t, detail2, "status")["code"]; code != "FAILED" {
		t.Fatalf("duplicate-id job status = %v, want FAILED (no retry)", code)
	}
	if got := itNum(t, detail2, "attempts"); got != 1 {
		t.Fatalf("business-conflict attempts = %d, want 1", got)
	}
	if got := itStr(t, detail2, "errorCode"); got != "FIELD_VALIDATION_FAILED" {
		t.Fatalf("duplicate-id errorCode = %q, want canonical FIELD_VALIDATION_FAILED", got)
	}
	if got := itCount(t, e, "select count(*) from ug_import_batch where event_id=$1 and id<>$2", ev, itStr(t, detail, "resultRef")); got != 0 {
		t.Fatalf("failed pull must not write a batch: %d", got)
	}

	// Empty sources are an explicit business failure too.
	mock.fn = func(req *http.Request) (*http.Response, error) {
		return itProviderResponse(200, itFeishuRecordsJSON()), nil
	}
	job3 := a.write("SourcePull", map[string]any{
		"eventId": ev, "sourceId": sourceID, "version": source["version"],
	}, http.StatusAccepted)
	itPullTick(t, e)
	detail3 := a.request("JobDetail", map[string]any{"jobId": itStr(t, job3, "jobId"), "eventId": ev}, http.StatusOK)
	if code := itMap(t, detail3, "status")["code"]; code != "FAILED" {
		t.Fatalf("empty-source status = %v, want FAILED", code)
	}
	if got := itStr(t, detail3, "errorCode"); got != "FIELD_VALIDATION_FAILED" {
		t.Fatalf("empty-source errorCode = %q, want FIELD_VALIDATION_FAILED", got)
	}

	// Retrying job2 after a source edit is blocked: the payload pins the
	// source version captured at queue time.
	editedSource := a.write("SourceSave", map[string]any{
		"eventId": ev, "sourceId": sourceID, "version": source["version"],
		"connectionId": connID, "resourceId": itFeishuResource,
		"mapping": map[string]any{"nameColumn": "name", "countMode": "TOTAL"}, "eligibility": "ELIGIBLE",
	}, http.StatusOK)
	a.write("JobRetry", map[string]any{"jobId": job2ID, "eventId": ev, "version": detail2["version"]}, http.StatusConflict)
	itGuard(t, a, "JobRetry", "EVENT_STATE_CONFLICT", "来源配置已变化，请重新发起拉取")

	// A revoked connection fails the claim without any provider IO.
	a.write("ConnectionRevoke", map[string]any{"connectionId": connID, "version": conn["version"]}, http.StatusOK)
	before := len(mock.recorded())
	job4 := a.write("SourcePull", map[string]any{
		"eventId": ev, "sourceId": sourceID, "version": editedSource["version"],
	}, http.StatusAccepted)
	itPullTick(t, e)
	detail4 := a.request("JobDetail", map[string]any{"jobId": itStr(t, job4, "jobId"), "eventId": ev}, http.StatusOK)
	if got := itStr(t, detail4, "errorCode"); got != "PROVIDER_UNAVAILABLE" {
		t.Fatalf("revoked-connection errorCode = %q, want PROVIDER_UNAVAILABLE", got)
	}
	if code := itMap(t, detail4, "status")["code"]; code != "PENDING" {
		t.Fatalf("revoked-connection status = %v, want PENDING (transient)", code)
	}
	if len(mock.recorded()) != before {
		t.Fatal("revoked connection must not reach the provider")
	}

	// JobList: admins see every job, organizers only their event's, kind filter.
	all := a.request("JobList", map[string]any{"kind": "PULL", "pageSize": 100}, http.StatusOK)
	if got := itNum(t, all, "total"); got != 4 {
		t.Fatalf("admin PULL jobs = %d, want 4", got)
	}
	organizer, orgAccount := itNewStaff(t, e, a, "ORGANIZER")
	itExportGrant(t, e, a, ev, itStr(t, orgAccount, "accountId"), []any{"name"}, []any{"VIEW_DETAIL"})
	scoped := organizer.request("JobList", map[string]any{"eventId": ev, "kind": "PULL"}, http.StatusOK)
	if got := itNum(t, scoped, "total"); got != 4 {
		t.Fatalf("organizer PULL jobs = %d, want 4", got)
	}
	otherEv := itPublishedEvent(t, e, a, 5)
	itExportGrant(t, e, a, otherEv, itStr(t, orgAccount, "accountId"), []any{"name"}, []any{"VIEW_DETAIL"})
	empty := organizer.request("JobList", map[string]any{"eventId": otherEv}, http.StatusOK)
	if got := itNum(t, empty, "total"); got != 0 {
		t.Fatalf("other-event jobs = %d, want 0", got)
	}
	// Cross-event JobDetail is hidden from the organizer.
	organizer.request("JobDetail", map[string]any{"jobId": jobID, "eventId": otherEv}, http.StatusNotFound)
	if organizer.lastCode != "RESOURCE_NOT_FOUND" {
		t.Fatalf("cross-event JobDetail code = %q, want RESOURCE_NOT_FOUND", organizer.lastCode)
	}
}