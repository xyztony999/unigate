package api

// PostgreSQL integration tests (HTTP level, frozen wire contract) for the
// controlled-export / delivery-registration / retention-cleanup chain. Port of
// the export/delivery/cleanup half of IntegrationWorkflowTest
// (exportRequiresIndependentGrantAndRechecksItOnDownload,
// xlsxExportAndDeliveryUseApprovedColumnsAndNeverSendAFile,
// cleanupRemovesFactsAndLeavesHonestBackupFailure) plus the READER ExportCreate
// guard from WorkflowIntegrationTest
// (schoolReaderFieldsActionsAndRevocationAreEnforcedInQueries).
//
// Error assertions use the frozen canonical wire codes (errcode.canonical /
// contract x-errors): the service's internal names such as EXPORT_FORBIDDEN or
// DELIVERY_SCOPE_INVALID are normalized to FORBIDDEN / FIELD_VALIDATION_FAILED
// etc., so the distinctive frozen message is checked to pin the exact guard.
//
// Run with:
//
//	set -a; . ../.local/go-test.env; set +a
//	go test ./internal/api/ -run 'TestIntegrationExport|TestIntegrationDelivery|TestIntegrationCleanup' -v

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"unigate/backend/internal/store"
)

// itExportGrant grants an ORGANIZER event grant with the given fields/actions
// (admin delegation; the creator's auto grant carries no EXPORT). Returns the
// grant view so callers can update it via grantId+version.
func itExportGrant(t *testing.T, e *itEnv, a *itBrowser, ev, accountID string, fields, actions []any) map[string]any {
	t.Helper()
	return a.write("GrantSave", map[string]any{
		"eventId":        ev,
		"accountId":      accountID,
		"role":           "ORGANIZER",
		"allowedFields":  fields,
		"allowedActions": actions,
		"expiresAt":      itFuture(24 * time.Hour),
	}, http.StatusOK)
}

// download performs the binary ExportDownload GET. On success it returns the
// raw artifact bytes and headers (no JSON envelope); on error it asserts the
// frozen error envelope exactly like roundTrip and records lastCode/lastMessage.
func (b *itBrowser) download(ev, exportID string, wantStatus int) (http.Header, []byte) {
	b.t.Helper()
	op := b.env.c.Operations["ExportDownload"]
	if op == nil {
		b.t.Fatal("unknown contract operation ExportDownload")
	}
	u := b.env.server.URL + op.Path + "?eventId=" + url.QueryEscape(ev) + "&exportId=" + url.QueryEscape(exportID)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		b.t.Fatalf("ExportDownload: build request: %v", err)
	}
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
		b.t.Fatalf("ExportDownload: do request: %v", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		b.t.Fatalf("ExportDownload: read body: %v", err)
	}
	if resp.StatusCode != wantStatus {
		b.t.Fatalf("ExportDownload: status = %d, want %d; body: %s", resp.StatusCode, wantStatus, raw)
	}
	if wantStatus >= 200 && wantStatus < 300 {
		// Success is a binary artifact, never the JSON envelope.
		if bytes.HasPrefix(bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF}), []byte("{")) {
			b.t.Fatalf("ExportDownload success must be binary, got JSON envelope: %s", raw)
		}
		return resp.Header, raw
	}
	var envelope struct {
		Code      string         `json:"code"`
		Message   string         `json:"message"`
		Data      map[string]any `json:"data"`
		RequestID string         `json:"requestId"`
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&envelope); err != nil {
		b.t.Fatalf("ExportDownload: decode error envelope (status %d): %v\nbody: %s", resp.StatusCode, err, raw)
	}
	if envelope.Code == "" || envelope.Code == "OK" {
		b.t.Fatalf("ExportDownload: error envelope must carry a string error code, got %q", envelope.Code)
	}
	if envelope.RequestID == "" || envelope.RequestID != resp.Header.Get("X-Request-ID") {
		b.t.Fatalf("ExportDownload: requestId %q missing or != X-Request-ID %q", envelope.RequestID, resp.Header.Get("X-Request-ID"))
	}
	if envelope.Data == nil {
		b.t.Fatal("ExportDownload: error envelope data must be an object")
	}
	b.lastCode = envelope.Code
	b.lastMessage = envelope.Message
	return resp.Header, raw
}

// itCount runs a parameterized aggregate query through the store and returns
// the first column as an integer (test-only SQL, never in the service layer).
func itCount(t *testing.T, e *itEnv, query string, args ...any) int64 {
	t.Helper()
	rows, err := e.s.Rows(context.Background(), nil, query, args...)
	if err != nil {
		t.Fatalf("count query: %v", err)
	}
	if len(rows) == 0 {
		return 0
	}
	for _, v := range rows[0] {
		return int64(store.Num(v))
	}
	return 0
}

// itRow runs a parameterized query and returns the first row.
func itRow(t *testing.T, e *itEnv, query string, args ...any) map[string]any {
	t.Helper()
	rows, err := e.s.Rows(context.Background(), nil, query, args...)
	if err != nil {
		t.Fatalf("row query: %v", err)
	}
	if len(rows) == 0 {
		t.Fatalf("row query returned no rows: %s", query)
	}
	return rows[0]
}

// itScopeCodes extracts the ordered scope codes from a CleanupView.scopes array.
func itScopeCodes(t *testing.T, scopes []any) []string {
	t.Helper()
	out := make([]string, 0, len(scopes))
	for _, s := range scopes {
		out = append(out, itStr(t, itMap(t, s.(map[string]any), "scope"), "code"))
	}
	return out
}

// itRunDueCleanup mirrors LifecycleWorker.expireNext: claim one due PENDING
// cleanup and run it inside a single transaction (the worker is not started in
// the test fixture, so the service entry point is driven directly).
func itRunDueCleanup(t *testing.T, e *itEnv) {
	t.Helper()
	ctx := context.Background()
	sc, err := e.s.Begin(ctx, store.TxOptions{})
	if err != nil {
		t.Fatalf("begin cleanup tx: %v", err)
	}
	defer e.s.Rollback(ctx, sc)
	due, err := e.s.DueCleanup(ctx, sc)
	if err != nil {
		t.Fatalf("claim due cleanup: %v", err)
	}
	if len(due) == 0 {
		t.Fatal("no due PENDING cleanup to run")
	}
	row, err := e.s.CleanupByIDForUpdate(ctx, sc, store.StrOrEmpty(due[0]["id"]), store.StrOrEmpty(due[0]["event_id"]))
	if err != nil {
		t.Fatalf("lock cleanup: %v", err)
	}
	if err := e.svc.RunCleanup(ctx, sc, row); err != nil {
		t.Fatalf("RunCleanup: %v", err)
	}
	if err := e.s.Commit(ctx, sc); err != nil {
		t.Fatalf("commit cleanup tx: %v", err)
	}
}

// itGuard asserts a canonical error code plus the frozen message that pins the
// exact guard (several internal codes normalize to the same wire code).
func itGuard(t *testing.T, b *itBrowser, op, code, message string) {
	t.Helper()
	if b.lastCode != code {
		t.Fatalf("%s code = %q, want canonical %s", op, b.lastCode, code)
	}
	if b.lastMessage != message {
		t.Fatalf("%s message = %q, want %q", op, b.lastMessage, message)
	}
}

// TestIntegrationExportControlledAccessAndDownload ports the controlled-export
// guards: independent EXPORT authorization (READER/CHECKER and the unprivileged
// creator are rejected), column whitelist + field projection, binary CSV/XLSX
// downloads with safeCell formula neutralization, ownership restriction, the
// download-time permission recheck, and the SUCCEEDED/expiry download gates.
func TestIntegrationExportControlledAccessAndDownload(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)
	ev := itPublishedEvent(t, e, a, 10)

	// Three parties: a companion group, one with explicit contact fields, and a
	// formula-injection name to exercise safeCell.
	itRegisteredParty(t, e, ev, "Export Parent", 1)
	func() {
		p := e.browser(t)
		p.bootstrap()
		pub := p.request("PublicEvent", map[string]any{"eventId": ev}, http.StatusOK)
		p.write("SubmitRegistration", map[string]any{
			"eventId":               ev,
			"fields":                map[string]any{"name": "Contact Parent", "phone": "0400 111 222", "email": "parent@example.invalid", "companionCount": 0},
			"acceptedNoticeVersion": itStr(t, pub, "noticeVersion"),
		}, http.StatusOK)
	}()
	itRegisteredParty(t, e, ev, "=SUM(A1:A2)", 0)

	// Cast: org1 full export rights; org2 name-only (field guard + non-owner);
	// reader/checker for the role guard. The admin's auto grant has no EXPORT.
	org1, org1Acc := itNewStaff(t, e, a, "ORGANIZER")
	org1ID := itStr(t, org1Acc, "accountId")
	org1Grant := itExportGrant(t, e, a, ev, org1ID,
		[]any{"name", "phone", "email", "companionCount"}, []any{"VIEW_DETAIL", "EXPORT"})
	org2, org2Acc := itNewStaff(t, e, a, "ORGANIZER")
	itExportGrant(t, e, a, ev, itStr(t, org2Acc, "accountId"),
		[]any{"name"}, []any{"VIEW_DETAIL", "EXPORT"})
	reader, readerAcc := itNewStaff(t, e, a, "READER")
	a.write("GrantSave", map[string]any{
		"eventId": ev, "accountId": itStr(t, readerAcc, "accountId"), "role": "READER",
		"allowedFields": []any{"name"}, "allowedActions": []any{"VIEW_DETAIL"},
		"recipient": "School office", "purpose": "Follow-up", "retentionNote": "Delete after review",
		"expiresAt": itFuture(24 * time.Hour),
	}, http.StatusOK)
	checker, checkerAcc := itNewStaff(t, e, a, "CHECKER")
	a.write("GrantSave", map[string]any{
		"eventId": ev, "accountId": itStr(t, checkerAcc, "accountId"), "role": "CHECKER",
		"allowedFields": []any{"name"}, "allowedActions": []any{"CHECKIN"},
		"expiresAt": itFuture(24 * time.Hour),
	}, http.StatusOK)

	exportArgs := map[string]any{
		"eventId": ev, "format": "CSV",
		"columns": []any{"name", "phone", "approvedSize"},
		"purpose": "Test approved export",
	}

	// READER/CHECKER lack the ORGANIZER role required by ExportCreate.
	reader.write("ExportCreate", exportArgs, http.StatusForbidden)
	itGuard(t, reader, "reader ExportCreate", "FORBIDDEN", "没有操作权限")
	checker.write("ExportCreate", exportArgs, http.StatusForbidden)
	itGuard(t, checker, "checker ExportCreate", "FORBIDDEN", "没有操作权限")
	// The creator's auto ORGANIZER grant carries no EXPORT action.
	a.write("ExportCreate", exportArgs, http.StatusForbidden)
	itGuard(t, a, "unprivileged creator ExportCreate", "FORBIDDEN", "未获得导出权限")
	// org2 has EXPORT but only the name field: a phone column is field-forbidden.
	org2.write("ExportCreate", map[string]any{
		"eventId": ev, "format": "CSV", "columns": []any{"name", "phone"}, "purpose": "Field guard",
	}, http.StatusForbidden)
	itGuard(t, org2, "org2 phone-column ExportCreate", "FORBIDDEN", "当前授权不包含所选字段")
	// Columns outside the frozen whitelist are rejected.
	org1.write("ExportCreate", map[string]any{
		"eventId": ev, "format": "CSV", "columns": []any{"name", "ticketToken"}, "purpose": "Bad column",
	}, http.StatusUnprocessableEntity)
	itGuard(t, org1, "invalid column ExportCreate", "FIELD_VALIDATION_FAILED", "导出列不在允许范围")

	// org1 creates the CSV export: accepted (202) and synchronously SUCCEEDED.
	created := org1.write("ExportCreate", exportArgs, http.StatusAccepted)
	exportID := itStr(t, created, "exportId")
	if code := itMap(t, created, "status")["code"]; code != "SUCCEEDED" {
		t.Fatalf("ExportCreate status = %v, want SUCCEEDED", code)
	}
	if code := itMap(t, created, "format")["code"]; code != "CSV" {
		t.Fatalf("ExportCreate format = %v, want CSV", code)
	}
	if got := itNum(t, created, "rowCount"); got != 3 {
		t.Fatalf("ExportCreate rowCount = %d, want 3", got)
	}
	if got := itNum(t, created, "version"); got != 0 {
		t.Fatalf("ExportCreate version = %d, want 0", got)
	}
	expires := itStr(t, created, "expiresAt")
	if exp, err := time.Parse(time.RFC3339Nano, expires); err != nil || !exp.After(time.Now()) {
		t.Fatalf("ExportCreate expiresAt = %q, want a future timestamp", expires)
	}
	detail := org1.request("ExportDetail", map[string]any{"eventId": ev, "exportId": exportID}, http.StatusOK)
	if code := itMap(t, detail, "status")["code"]; code != "SUCCEEDED" {
		t.Fatalf("ExportDetail status = %v, want SUCCEEDED", code)
	}

	// Binary CSV download: frozen Content-Type/Disposition, BOM+CRLF, projected
	// columns only, and safeCell neutralization of the formula name.
	hdr, body := org1.download(ev, exportID, http.StatusOK)
	if got := hdr.Get("Content-Type"); got != "text/csv;charset=UTF-8" {
		t.Fatalf("CSV Content-Type = %q, want text/csv;charset=UTF-8", got)
	}
	if got := hdr.Get("Content-Disposition"); got != `attachment; filename="unigate.csv"` {
		t.Fatalf("CSV Content-Disposition = %q, want attachment; filename=\"unigate.csv\"", got)
	}
	if !bytes.HasPrefix(body, []byte{0xEF, 0xBB, 0xBF}) {
		t.Fatalf("CSV must start with a UTF-8 BOM: %x", body[:min(3, len(body))])
	}
	text := string(body)
	if !strings.Contains(text, "name,phone,approvedSize\r\n") {
		t.Fatalf("CSV header must project exactly the authorized columns: %q", text)
	}
	if !strings.Contains(text, "Export Parent") || !strings.Contains(text, "0400 111 222") {
		t.Fatalf("CSV must carry the registered name/phone: %q", text)
	}
	if !strings.Contains(text, "'=SUM(A1:A2)") {
		t.Fatalf("CSV must neutralize the formula name with a leading apostrophe: %q", text)
	}
	if strings.Contains(text, "parent@example.invalid") {
		t.Fatalf("CSV must not project the unauthorized email column: %q", text)
	}

	// Ownership: another organizer with EXPORT cannot read or download it.
	org2.request("ExportDetail", map[string]any{"eventId": ev, "exportId": exportID}, http.StatusForbidden)
	itGuard(t, org2, "non-owner ExportDetail", "FORBIDDEN", "只能访问本人生成的导出")
	org2.download(ev, exportID, http.StatusForbidden)
	itGuard(t, org2, "non-owner ExportDownload", "FORBIDDEN", "只能访问本人生成的导出")

	// Download rechecks the grant: removing EXPORT blocks the download.
	noExport := a.write("GrantSave", map[string]any{
		"eventId": ev, "accountId": org1ID, "role": "ORGANIZER",
		"grantId": itStr(t, org1Grant, "grantId"), "version": org1Grant["version"],
		"allowedFields": []any{"name", "phone", "email", "companionCount"}, "allowedActions": []any{"VIEW_DETAIL"},
		"expiresAt": itFuture(24 * time.Hour),
	}, http.StatusOK)
	org1.download(ev, exportID, http.StatusForbidden)
	itGuard(t, org1, "revoked-EXPORT download", "FORBIDDEN", "未获得导出权限")
	// Restoring EXPORT but dropping the phone field rechecks column authorization.
	nameOnly := a.write("GrantSave", map[string]any{
		"eventId": ev, "accountId": org1ID, "role": "ORGANIZER",
		"grantId": itStr(t, org1Grant, "grantId"), "version": noExport["version"],
		"allowedFields": []any{"name"}, "allowedActions": []any{"VIEW_DETAIL", "EXPORT"},
		"expiresAt": itFuture(24 * time.Hour),
	}, http.StatusOK)
	org1.download(ev, exportID, http.StatusForbidden)
	itGuard(t, org1, "field-revoked download", "FORBIDDEN", "当前授权不包含所选字段")
	// Restore the full grant for the remaining gates.
	a.write("GrantSave", map[string]any{
		"eventId": ev, "accountId": org1ID, "role": "ORGANIZER",
		"grantId": itStr(t, org1Grant, "grantId"), "version": nameOnly["version"],
		"allowedFields": []any{"name", "phone", "email", "companionCount"}, "allowedActions": []any{"VIEW_DETAIL", "EXPORT"},
		"expiresAt": itFuture(24 * time.Hour),
	}, http.StatusOK)

	// ExportDownload only serves SUCCEEDED artifacts.
	if _, err := e.s.Exec(context.Background(), nil, "update ug_export set status='PENDING' where id=$1", exportID); err != nil {
		t.Fatalf("set export PENDING: %v", err)
	}
	org1.download(ev, exportID, http.StatusConflict)
	itGuard(t, org1, "non-SUCCEEDED download", "EVENT_STATE_CONFLICT", "导出文件尚未生成")
	if _, err := e.s.Exec(context.Background(), nil, "update ug_export set status='SUCCEEDED' where id=$1", exportID); err != nil {
		t.Fatalf("restore export SUCCEEDED: %v", err)
	}

	// Expired exports report EXPIRED and refuse download.
	if _, err := e.s.Exec(context.Background(), nil, "update ug_export set expires_at=now()-interval '1 second' where id=$1", exportID); err != nil {
		t.Fatalf("expire export: %v", err)
	}
	expiredDetail := org1.request("ExportDetail", map[string]any{"eventId": ev, "exportId": exportID}, http.StatusOK)
	if code := itMap(t, expiredDetail, "status")["code"]; code != "EXPIRED" {
		t.Fatalf("expired ExportDetail status = %v, want EXPIRED", code)
	}
	org1.download(ev, exportID, http.StatusGone)
	itGuard(t, org1, "expired download", "ARTIFACT_EXPIRED", "导出文件已过期")

	// XLSX export with the frozen filter: binary ZIP workbook, safeCell applied.
	xlsx := org1.write("ExportCreate", map[string]any{
		"eventId": ev, "format": "XLSX",
		"columns": []any{"name", "approvedSize"},
		"purpose": "Approved recipient review",
		"filter": map[string]any{
			"status":      "APPROVED",
			"progress":    "NONE",
			"createdFrom": time.Now().UTC().Add(-5 * time.Minute).Format(time.RFC3339Nano),
			"createdTo":   time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339Nano),
		},
	}, http.StatusAccepted)
	xlsxID := itStr(t, xlsx, "exportId")
	if got := itNum(t, xlsx, "rowCount"); got != 3 {
		t.Fatalf("XLSX rowCount = %d, want 3", got)
	}
	xhdr, xbody := org1.download(ev, xlsxID, http.StatusOK)
	if got := xhdr.Get("Content-Type"); got != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Fatalf("XLSX Content-Type = %q, want spreadsheetml.sheet", got)
	}
	if got := xhdr.Get("Content-Disposition"); got != `attachment; filename="unigate.xlsx"` {
		t.Fatalf("XLSX Content-Disposition = %q, want attachment; filename=\"unigate.xlsx\"", got)
	}
	if !bytes.HasPrefix(xbody, []byte("PK\x03\x04")) {
		t.Fatalf("XLSX must be a ZIP workbook, got magic %x", xbody[:min(4, len(xbody))])
	}
	// The sheet part is deflate-compressed; read it back to inspect the cells.
	zr, err := zip.NewReader(bytes.NewReader(xbody), int64(len(xbody)))
	if err != nil {
		t.Fatalf("XLSX must be a readable ZIP: %v", err)
	}
	var sheet []byte
	for _, f := range zr.File {
		if f.Name != "xl/worksheets/sheet1.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open sheet part: %v", err)
		}
		sheet, err = io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("read sheet part: %v", err)
		}
	}
	if sheet == nil {
		t.Fatal("XLSX must contain xl/worksheets/sheet1.xml")
	}
	if !bytes.Contains(sheet, []byte("Export Parent")) {
		t.Fatalf("XLSX sheet must carry the registered names: %s", sheet)
	}
	// The leading apostrophe is XML-escaped (&#39;) inside the sheet part; POI
	// would decode it back to "'=SUM(A1:A2)". Either encoding proves safeCell ran.
	if !bytes.Contains(sheet, []byte("&#39;=SUM(A1:A2)")) && !bytes.Contains(sheet, []byte("'=SUM(A1:A2)")) {
		t.Fatalf("XLSX must neutralize the formula cell with a leading apostrophe: %s", sheet)
	}

	// Download is audited.
	audit := a.request("AuditList", map[string]any{"eventId": ev, "page": 1, "pageSize": 100}, http.StatusOK)
	foundDownload := false
	for _, item := range itItems(t, audit) {
		if item.(map[string]any)["action"] == "ExportDownload" {
			foundDownload = true
		}
	}
	if !foundDownload {
		t.Fatal("AuditList must record ExportDownload")
	}
}

// TestIntegrationDeliveryRecordScopeAndAudit ports the delivery-registration
// chain: recipient/purpose/retention disclosure sealed at rest, delivery scope
// confined to the original export columns, no automatic email/file dispatch,
// ownership and expiry guards, and the delivery audit trail.
func TestIntegrationDeliveryRecordScopeAndAudit(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)
	ev := itPublishedEvent(t, e, a, 10)
	itRegisteredParty(t, e, ev, "Delivery Parent", 0)

	org1, org1Acc := itNewStaff(t, e, a, "ORGANIZER")
	org1ID := itStr(t, org1Acc, "accountId")
	itExportGrant(t, e, a, ev, org1ID,
		[]any{"name", "phone", "email", "companionCount"}, []any{"VIEW_DETAIL", "EXPORT"})
	org2, org2Acc := itNewStaff(t, e, a, "ORGANIZER")
	itExportGrant(t, e, a, ev, itStr(t, org2Acc, "accountId"),
		[]any{"name", "phone", "email", "companionCount"}, []any{"VIEW_DETAIL", "EXPORT"})

	created := org1.write("ExportCreate", map[string]any{
		"eventId": ev, "format": "CSV",
		"columns": []any{"name", "phone", "approvedSize"},
		"purpose": "Approved recipient review",
	}, http.StatusAccepted)
	exportID := itStr(t, created, "exportId")
	org1.download(ev, exportID, http.StatusOK)

	recordArgs := map[string]any{
		"eventId":       ev,
		"exportId":      exportID,
		"recipient":     "Approved school representative",
		"purpose":       "Review attendance",
		"columns":       []any{"name"},
		"retentionNote": "Delete recipient copy after agreed review",
	}
	delivery := org1.write("DeliveryRecord", recordArgs, http.StatusOK)
	deliveryID := itStr(t, delivery, "deliveryId")
	if got := itStr(t, delivery, "recipient"); got != "Approved school representative" {
		t.Fatalf("delivery recipient = %q, want the registered recipient", got)
	}
	if got := itStr(t, delivery, "exportId"); got != exportID {
		t.Fatalf("delivery exportId = %s, want %s", got, exportID)
	}
	if got := itNum(t, delivery, "version"); got != 0 {
		t.Fatalf("delivery version = %d, want 0", got)
	}

	// DeliveryList echoes the unsealed disclosure.
	list := org1.request("DeliveryList", map[string]any{"eventId": ev, "page": 1, "pageSize": 20}, http.StatusOK)
	if got := itNum(t, list, "total"); got != 1 {
		t.Fatalf("DeliveryList total = %d, want 1", got)
	}
	item := itItems(t, list)[0].(map[string]any)
	if got := itStr(t, item, "deliveryId"); got != deliveryID {
		t.Fatalf("DeliveryList deliveryId = %s, want %s", got, deliveryID)
	}
	if got := itStr(t, item, "purpose"); got != "Review attendance" {
		t.Fatalf("DeliveryList purpose = %q, want Review attendance", got)
	}
	if got := itStr(t, item, "retentionNote"); got != "Delete recipient copy after agreed review" {
		t.Fatalf("DeliveryList retentionNote = %q, want the registered note", got)
	}

	// Disclosure is sealed at rest: the plaintext recipient never appears in the
	// raw recipient_cipher column.
	row := itRow(t, e, "select recipient_cipher from ug_delivery where id=$1", deliveryID)
	cipher, ok := row["recipient_cipher"].([]byte)
	if !ok || len(cipher) == 0 {
		t.Fatalf("recipient_cipher must be sealed bytea, got %T", row["recipient_cipher"])
	}
	if bytes.Contains(cipher, []byte("Approved school representative")) {
		t.Fatal("recipient disclosure must be encrypted at rest, found plaintext")
	}

	// Delivery scope must be a non-empty subset of the original export columns.
	org1.write("DeliveryRecord", map[string]any{
		"eventId": ev, "exportId": exportID, "recipient": "R", "purpose": "P",
		"columns": []any{"email"}, "retentionNote": "N",
	}, http.StatusUnprocessableEntity)
	itGuard(t, org1, "out-of-scope DeliveryRecord", "FIELD_VALIDATION_FAILED", "交付范围必须属于原导出列")
	org1.write("DeliveryRecord", map[string]any{
		"eventId": ev, "exportId": exportID, "recipient": "R", "purpose": "P",
		"columns": []any{}, "retentionNote": "N",
	}, http.StatusUnprocessableEntity)
	itGuard(t, org1, "empty-scope DeliveryRecord", "FIELD_VALIDATION_FAILED", "交付范围必须属于原导出列")

	// A different organizer cannot register a delivery against someone's export.
	org2.write("DeliveryRecord", recordArgs, http.StatusForbidden)
	itGuard(t, org2, "non-owner DeliveryRecord", "FORBIDDEN", "只能访问本人生成的导出")

	// Delivery registration never dispatches an email/file job.
	if got := itCount(t, e, "select count(*) from ug_job where event_id=$1 and kind='EMAIL' and business_key like 'delivery:%'", ev); got != 0 {
		t.Fatalf("delivery EMAIL jobs = %d, want 0 (registration must not send a file)", got)
	}

	// An expired source export blocks delivery registration.
	if _, err := e.s.Exec(context.Background(), nil, "update ug_export set expires_at=now()-interval '1 second' where id=$1", exportID); err != nil {
		t.Fatalf("expire export: %v", err)
	}
	org1.write("DeliveryRecord", recordArgs, http.StatusGone)
	itGuard(t, org1, "expired-export DeliveryRecord", "ARTIFACT_EXPIRED", "原导出不可用")

	// The export/delivery chain is audited.
	audit := a.request("AuditList", map[string]any{"eventId": ev, "page": 1, "pageSize": 100}, http.StatusOK)
	actions := map[string]bool{}
	for _, item := range itItems(t, audit) {
		actions[store.StrOrEmpty(item.(map[string]any)["action"])] = true
	}
	for _, want := range []string{"ExportCreate", "ExportDownload", "DeliveryRecord"} {
		if !actions[want] {
			t.Fatalf("AuditList must record %s; got %v", want, actions)
		}
	}
}

// TestIntegrationCleanupRetentionAndHonestBackupFailure ports the retention
// cleanup: the six-scope detail, due cleanup archiving the event and preserving
// anonymous aggregates before deleting linkable facts, managed artifact removal,
// private-query invalidation, the honest BACKUPS failure (never claiming all
// copies deleted), and the FAILED-only retry guard.
func TestIntegrationCleanupRetentionAndHonestBackupFailure(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)
	ev := itPublishedEvent(t, e, a, 10)

	p, _, _ := itRegisteredParty(t, e, ev, "Cleanup Parent", 1)
	itRegisteredParty(t, e, ev, "Other Parent", 0)

	org1, org1Acc := itNewStaff(t, e, a, "ORGANIZER")
	itExportGrant(t, e, a, ev, itStr(t, org1Acc, "accountId"),
		[]any{"name", "phone", "email", "companionCount"}, []any{"VIEW_DETAIL", "EXPORT"})

	// An export artifact and a delivery exist before cleanup.
	created := org1.write("ExportCreate", map[string]any{
		"eventId": ev, "format": "CSV", "columns": []any{"name", "approvedSize"}, "purpose": "Retention fixture",
	}, http.StatusAccepted)
	exportID := itStr(t, created, "exportId")
	artifactKey := itStr(t, itRow(t, e, "select artifact_key from ug_export where id=$1", exportID), "artifact_key")
	storage, err := filepath.Abs(e.cfg.StorageDirectory)
	if err != nil {
		t.Fatalf("storage dir: %v", err)
	}
	artifactPath := filepath.Join(storage, artifactKey)
	if _, err := os.Stat(artifactPath); err != nil {
		t.Fatalf("export artifact must exist before cleanup: %v", err)
	}
	org1.write("DeliveryRecord", map[string]any{
		"eventId": ev, "exportId": exportID, "recipient": "School", "purpose": "Review",
		"columns": []any{"name"}, "retentionNote": "Delete after review",
	}, http.StatusOK)

	// Capture a sealed party cipher to prove the DEK is revoked by cleanup.
	oldCipher, ok := itRow(t, e, "select fields_cipher from ug_party where event_id=$1 limit 1", ev)["fields_cipher"].([]byte)
	if !ok || len(oldCipher) == 0 {
		t.Fatal("party fields_cipher must be sealed bytea")
	}

	// CleanupDetail surfaces the six scopes in the frozen pool order, PENDING.
	detail := org1.request("CleanupDetail", map[string]any{"eventId": ev}, http.StatusOK)
	cleanupID := itStr(t, detail, "cleanupId")
	if code := itMap(t, detail, "status")["code"]; code != "PENDING" {
		t.Fatalf("initial cleanup status = %v, want PENDING", code)
	}
	scopes, ok := detail["scopes"].([]any)
	if !ok || len(scopes) != 6 {
		t.Fatalf("cleanup scopes must have six entries, got %v", detail["scopes"])
	}
	wantOrder := []string{"PRIVATE_ACCESS", "LIVE_ROWS", "TEMP_FILES", "EXPORT_FILES", "TASK_PAYLOADS", "BACKUPS"}
	gotOrder := itScopeCodes(t, scopes)
	for i, want := range wantOrder {
		if gotOrder[i] != want {
			t.Fatalf("cleanup scope order = %v, want %v", gotOrder, wantOrder)
		}
	}
	for _, s := range scopes {
		m := s.(map[string]any)
		if code := itMap(t, m, "status")["code"]; code != "PENDING" {
			t.Fatalf("initial scope %v status = %v, want PENDING", m["scope"], code)
		}
	}
	pendingVersion := itNum(t, detail, "version")

	// A PENDING cleanup is not retryable (only FAILED + due).
	org1.write("CleanupRetry", map[string]any{
		"eventId": ev, "cleanupId": cleanupID, "version": pendingVersion,
	}, http.StatusConflict)
	itGuard(t, org1, "PENDING CleanupRetry", "EVENT_STATE_CONFLICT", "仅能重试已到期的失败清理")

	// Pre-cleanup facts and aggregates.
	if got := itCount(t, e, "select count(*) from ug_party where event_id=$1", ev); got != 2 {
		t.Fatalf("pre-cleanup parties = %d, want 2", got)
	}
	allocated := itNum(t, a.request("Statistics", map[string]any{"eventId": ev}, http.StatusOK), "allocatedPeople")
	if allocated != 3 {
		t.Fatalf("pre-cleanup allocatedPeople = %d, want 3", allocated)
	}

	// Make the cleanup due and run it through the worker entry point.
	if _, err := e.s.Exec(context.Background(), nil, "update ug_cleanup set due_at=now()-interval '1 second' where event_id=$1", ev); err != nil {
		t.Fatalf("set cleanup due: %v", err)
	}
	itRunDueCleanup(t, e)

	// Linkable facts and managed copies are gone.
	if got := itCount(t, e, "select count(*) from ug_party where event_id=$1", ev); got != 0 {
		t.Fatalf("post-cleanup parties = %d, want 0", got)
	}
	if got := itCount(t, e, "select count(*) from ug_delivery where event_id=$1", ev); got != 0 {
		t.Fatalf("post-cleanup deliveries = %d, want 0", got)
	}
	if got := itCount(t, e, "select count(*) from ug_export where event_id=$1", ev); got != 0 {
		t.Fatalf("post-cleanup exports = %d, want 0", got)
	}
	if _, err := os.Stat(artifactPath); !os.IsNotExist(err) {
		t.Fatalf("managed export artifact must be deleted, stat err = %v", err)
	}

	// ARCHIVED precedes physical deletion: anonymous aggregates were snapshotted
	// from the live facts (non-zero) and carry no identifying data.
	event := itRow(t, e, "select status, anonymous_totals from ug_event where id=$1", ev)
	if got := store.StrOrEmpty(event["status"]); got != "ARCHIVED" {
		t.Fatalf("post-cleanup event status = %q, want ARCHIVED", got)
	}
	totals := store.Parse(event["anonymous_totals"])
	if totals == nil {
		t.Fatal("anonymous_totals must be preserved")
	}
	if got := store.Num(totals["application_groups"]); got != 2 {
		t.Fatalf("anonymous application_groups = %d, want the pre-cleanup 2", got)
	}
	if got := store.Num(totals["allocated_people"]); got != 3 {
		t.Fatalf("anonymous allocated_people = %d, want the pre-cleanup 3", got)
	}
	totalsText, err := store.Stringify(event["anonymous_totals"])
	if err != nil {
		t.Fatalf("stringify anonymous_totals: %v", err)
	}
	if strings.Contains(totalsText, "Cleanup Parent") {
		t.Fatal("anonymous aggregates must not retain identifying names")
	}

	// The cleanup record stays FAILED with an honest BACKUPS failure.
	cleanup := itRow(t, e, "select status, failed_count, processed_count, version from ug_cleanup where event_id=$1", ev)
	if got := store.StrOrEmpty(cleanup["status"]); got != "FAILED" {
		t.Fatalf("post-cleanup status = %q, want FAILED (backups unverified)", got)
	}
	if store.Num(cleanup["failed_count"]) < 1 {
		t.Fatalf("post-cleanup failed_count = %v, want >= 1", cleanup["failed_count"])
	}
	if store.Num(cleanup["processed_count"]) <= 0 {
		t.Fatalf("post-cleanup processed_count = %v, want > 0", cleanup["processed_count"])
	}
	after := org1.request("CleanupDetail", map[string]any{"eventId": ev}, http.StatusOK)
	if code := itMap(t, after, "status")["code"]; code != "FAILED" {
		t.Fatalf("CleanupDetail status = %v, want FAILED", code)
	}
	afterScopes := after["scopes"].([]any)
	if got := itScopeCodes(t, afterScopes); strings.Join(got, ",") != strings.Join(wantOrder, ",") {
		t.Fatalf("post-cleanup scope order = %v, want %v", got, wantOrder)
	}
	for _, s := range afterScopes {
		m := s.(map[string]any)
		code := itStr(t, itMap(t, m, "scope"), "code")
		status := itStr(t, itMap(t, m, "status"), "code")
		if code == "BACKUPS" {
			if status != "FAILED" {
				t.Fatalf("BACKUPS scope status = %q, want FAILED", status)
			}
			if got := itStr(t, m, "errorCode"); got != "BACKUP_COVERAGE_UNVERIFIED" {
				t.Fatalf("BACKUPS errorCode = %q, want BACKUP_COVERAGE_UNVERIFIED", got)
			}
		} else if status != "SUCCEEDED" {
			t.Fatalf("scope %s status = %q, want SUCCEEDED", code, status)
		}
	}

	// The independent deletion ledger records backup-pending, not completion.
	ledger := itRow(t, e, "select status from ug_deletion_ledger where event_id=$1", ev)
	if got := store.StrOrEmpty(ledger["status"]); got != "BACKUP_PENDING" {
		t.Fatalf("deletion ledger status = %q, want BACKUP_PENDING", got)
	}

	// The revoked DEK can no longer unseal the captured party cipher.
	if _, err := e.s.Unseal(oldCipher); err == nil {
		t.Fatal("revoked event key must not unseal old party facts")
	}

	// Private party access is invalidated; organizer reads report the expiry.
	p.request("MyParty", map[string]any{}, http.StatusUnauthorized)
	if p.lastCode != "SESSION_EXPIRED" {
		t.Fatalf("post-cleanup MyParty code = %q, want SESSION_EXPIRED", p.lastCode)
	}
	a.request("Statistics", map[string]any{"eventId": ev}, http.StatusGone)
	if a.lastCode != "TOKEN_EXPIRED" {
		t.Fatalf("post-cleanup Statistics code = %q, want TOKEN_EXPIRED", a.lastCode)
	}
	a.request("PartyList", map[string]any{"eventId": ev, "page": 1, "pageSize": 20}, http.StatusGone)
	org1.download(ev, exportID, http.StatusGone)
	org1.request("DeliveryList", map[string]any{"eventId": ev}, http.StatusGone)

	// CleanupRetry reruns a due FAILED cleanup but never claims backup success.
	retryVersion := itNum(t, after, "version")
	job := org1.write("CleanupRetry", map[string]any{
		"eventId": ev, "cleanupId": cleanupID, "version": retryVersion,
	}, http.StatusAccepted)
	if code := itMap(t, job, "kind")["code"]; code != "CLEANUP" {
		t.Fatalf("CleanupRetry job kind = %v, want CLEANUP", code)
	}
	if code := itMap(t, job, "status")["code"]; code != "FAILED" {
		t.Fatalf("CleanupRetry job status = %v, want FAILED", code)
	}
	if got := itStr(t, job, "errorCode"); got != "BACKUP_COVERAGE_UNVERIFIED" {
		t.Fatalf("CleanupRetry errorCode = %q, want BACKUP_COVERAGE_UNVERIFIED", got)
	}
	again := org1.request("CleanupDetail", map[string]any{"eventId": ev}, http.StatusOK)
	if got := itNum(t, again, "version"); got != retryVersion+1 {
		t.Fatalf("retry cleanup version = %d, want %d", got, retryVersion+1)
	}
	if code := itMap(t, again, "status")["code"]; code != "FAILED" {
		t.Fatalf("retried cleanup status = %v, want FAILED", code)
	}
	// Repeated cleanup preserves the original anonymous snapshot.
	totals2 := store.Parse(itRow(t, e, "select anonymous_totals from ug_event where id=$1", ev)["anonymous_totals"])
	if got := store.Num(totals2["application_groups"]); got != 2 {
		t.Fatalf("retried anonymous application_groups = %d, want the preserved 2", got)
	}

	// A stale version is an optimistic conflict.
	org1.write("CleanupRetry", map[string]any{
		"eventId": ev, "cleanupId": cleanupID, "version": retryVersion,
	}, http.StatusConflict)
	if org1.lastCode != "VERSION_CONFLICT" {
		t.Fatalf("stale CleanupRetry code = %q, want VERSION_CONFLICT", org1.lastCode)
	}

	// A FAILED cleanup that is not yet due is not retryable.
	if _, err := e.s.Exec(context.Background(), nil, "update ug_cleanup set due_at=now()+interval '1 day' where event_id=$1", ev); err != nil {
		t.Fatalf("set cleanup not due: %v", err)
	}
	notDueVersion := itNum(t, org1.request("CleanupDetail", map[string]any{"eventId": ev}, http.StatusOK), "version")
	org1.write("CleanupRetry", map[string]any{
		"eventId": ev, "cleanupId": cleanupID, "version": notDueVersion,
	}, http.StatusConflict)
	itGuard(t, org1, "not-due CleanupRetry", "EVENT_STATE_CONFLICT", "仅能重试已到期的失败清理")

	// A non-FAILED cleanup is never retryable even when due.
	if _, err := e.s.Exec(context.Background(), nil, "update ug_cleanup set status='SUCCEEDED', due_at=now()-interval '1 second' where event_id=$1", ev); err != nil {
		t.Fatalf("set cleanup SUCCEEDED: %v", err)
	}
	succeededVersion := itNum(t, org1.request("CleanupDetail", map[string]any{"eventId": ev}, http.StatusOK), "version")
	org1.write("CleanupRetry", map[string]any{
		"eventId": ev, "cleanupId": cleanupID, "version": succeededVersion,
	}, http.StatusConflict)
	itGuard(t, org1, "SUCCEEDED CleanupRetry", "EVENT_STATE_CONFLICT", "仅能重试已到期的失败清理")
}