package api

// Port of the import half of IntegrationWorkflowTest
// (selectedImportIsAllOrNothingAndDoesNotOverAllocate,
// policyDriftRequiresACompletelyNewPreview,
// formulaRowsAreExcludedButOtherRowsRemainCommittable) at the HTTP level:
// multipart BatchUpload -> BatchDetail -> BatchPreview -> BatchRows ->
// DuplicateResolve -> atomic BatchCommit, with error rows excluded from
// selection, whole-batch rollback on conflict and the frozen intake guards.

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
	"time"
)

// timeNowPlus renders an RFC3339 timestamp h hours from now (test fixtures).
func timeNowPlus(h int) string {
	return time.Now().UTC().Add(time.Duration(h) * time.Hour).Format(time.RFC3339Nano)
}

// upload performs a multipart contract operation (BatchUpload) reusing the
// frozen envelope assertions of the shared browser.
func (b *itBrowser) upload(name string, fields map[string]string, filename string, content []byte, wantStatus int) map[string]any {
	b.t.Helper()
	op := b.env.c.Operations[name]
	if op == nil {
		b.t.Fatalf("unknown contract operation %q", name)
	}
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		if err := w.WriteField(k, v); err != nil {
			b.t.Fatalf("%s: write field %s: %v", name, k, err)
		}
	}
	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		b.t.Fatalf("%s: create form file: %v", name, err)
	}
	if _, err := fw.Write(content); err != nil {
		b.t.Fatalf("%s: write file: %v", name, err)
	}
	if err := w.Close(); err != nil {
		b.t.Fatalf("%s: close multipart: %v", name, err)
	}
	req, err := http.NewRequest(http.MethodPost, b.env.server.URL+op.Path, &buf)
	if err != nil {
		b.t.Fatalf("%s: build request: %v", name, err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	return b.roundTrip(name, req, wantStatus)
}

// itUploadBatch uploads a CSV fixture (Java preview()'s BatchUpload step).
func itUploadBatch(t *testing.T, e *itEnv, a *itBrowser, ev, csv string) map[string]any {
	t.Helper()
	return a.upload("BatchUpload", map[string]string{
		"operationId": e.s.ID(),
		"eventId":     ev,
		"sourceLabel": "CSV fixture",
	}, "fixture.csv", []byte(csv), http.StatusOK)
}

// itPreviewBatch runs BatchPreview against the batch's current version.
func itPreviewBatch(t *testing.T, a *itBrowser, ev string, batch map[string]any, mapping map[string]any) map[string]any {
	t.Helper()
	return a.write("BatchPreview", map[string]any{
		"eventId":     ev,
		"batchId":     batch["batchId"],
		"version":     batch["version"],
		"mapping":     mapping,
		"eligibility": "ELIGIBLE",
	}, http.StatusOK)
}

// itRowIDs lists the batch rows in line order (Java rowIds()).
func itRowIDs(t *testing.T, a *itBrowser, ev string, batch map[string]any) []any {
	t.Helper()
	rows := a.request("BatchRows", map[string]any{"eventId": ev, "batchId": batch["batchId"], "page": 1, "pageSize": 100}, http.StatusOK)
	ids := []any{}
	for _, item := range rows["items"].([]any) {
		ids = append(ids, itStr(t, item.(map[string]any), "rowId"))
	}
	return ids
}

// itRowCount asserts the event has exactly n parties (whole-batch rollback).
func itRowCount(t *testing.T, a *itBrowser, ev string, want int64) {
	t.Helper()
	list := a.request("PartyList", map[string]any{"eventId": ev, "page": 1, "pageSize": 100}, http.StatusOK)
	if got := itNum(t, list, "total"); got != want {
		t.Fatalf("party count = %d, want %d (batch must be all-or-nothing)", got, want)
	}
}

// itPolicyInput reads PolicyGet and renders the frozen EventPolicyInput shape.
func itPolicyInput(t *testing.T, a *itBrowser, ev string) map[string]any {
	t.Helper()
	p := a.request("PolicyGet", map[string]any{"eventId": ev}, http.StatusOK)
	return map[string]any{
		"fields":                   p["fields"],
		"approvalMode":             itMap(t, p, "approvalMode")["code"],
		"phoneVerificationEnabled": p["phoneVerificationEnabled"],
		"selfEditEnabled":          p["selfEditEnabled"],
		"selfCancelEnabled":        p["selfCancelEnabled"],
		"retentionDays":            p["retentionDays"],
		"version":                  p["version"],
	}
}

// TestIntegrationImportBatchLifecycleAndAtomicCommit ports
// selectedImportIsAllOrNothingAndDoesNotOverAllocate: upload -> detail ->
// preview -> rows, an over-capacity selection rolls back completely, a partial
// selection commits atomically, replay is sealed, and re-uploading the same
// file surfaces ALREADY_LINKED rows that cannot be committed again.
func TestIntegrationImportBatchLifecycleAndAtomicCommit(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)
	ev := itPublishedEvent(t, e, a, 3)

	csv := "name,phone,total\nParent A,138-0000-0001,2\nParent B,13800000002,2\n"
	batch := itUploadBatch(t, e, a, ev, csv)
	batchID := itStr(t, batch, "batchId")
	if code := itMap(t, batch, "status")["code"]; code != "UPLOADED" {
		t.Fatalf("uploaded batch status = %v, want UPLOADED", code)
	}
	if got := itNum(t, batch, "previewVersion"); got != 0 {
		t.Fatalf("fresh batch previewVersion = %d, want 0", got)
	}
	// Rows start unvalidated (ERROR) until a preview normalizes them.
	if got := itNum(t, batch, "errorRows"); got != 2 {
		t.Fatalf("fresh batch errorRows = %d, want 2", got)
	}
	columns := batch["availableColumns"].([]any)
	if len(columns) != 3 {
		t.Fatalf("availableColumns = %+v, want the 3 header columns", columns)
	}
	for i, want := range []string{"name", "phone", "total"} {
		if got := itStr(t, columns[i].(map[string]any), "code"); got != want {
			t.Fatalf("availableColumns[%d] = %s, want %s (header order)", i, got, want)
		}
	}

	detail := a.request("BatchDetail", map[string]any{"eventId": ev, "batchId": batchID}, http.StatusOK)
	if itStr(t, detail, "batchId") != batchID || itNum(t, detail, "version") != itNum(t, batch, "version") {
		t.Fatalf("BatchDetail must echo the uploaded batch: %+v", detail)
	}

	preview := itPreviewBatch(t, a, ev, batch, map[string]any{
		"nameColumn": "name", "phoneColumn": "phone", "countColumn": "total", "countMode": "TOTAL",
	})
	if got := itNum(t, preview, "validRows"); got != 2 {
		t.Fatalf("preview validRows = %d, want 2", got)
	}
	if got := itNum(t, preview, "errorRows"); got != 0 {
		t.Fatalf("preview errorRows = %d, want 0", got)
	}
	if got := itNum(t, preview, "previewVersion"); got != 1 {
		t.Fatalf("previewVersion = %d, want 1", got)
	}
	if code := itMap(t, preview, "status")["code"]; code != "PREVIEWED" {
		t.Fatalf("previewed batch status = %v, want PREVIEWED", code)
	}
	if got := itStr(t, itMap(t, preview, "mapping"), "nameColumn"); got != "name" {
		t.Fatalf("stored mapping nameColumn = %q", got)
	}
	if code := itMap(t, preview, "eligibility")["code"]; code != "ELIGIBLE" {
		t.Fatalf("eligibility = %v, want ELIGIBLE", code)
	}

	rows := a.request("BatchRows", map[string]any{"eventId": ev, "batchId": batchID}, http.StatusOK)
	if got := itNum(t, rows, "total"); got != 2 {
		t.Fatalf("BatchRows total = %d, want 2", got)
	}
	if got := itNum(t, rows, "page"); got != 1 {
		t.Fatalf("BatchRows page = %d, want 1", got)
	}
	first := rows["items"].([]any)[0].(map[string]any)
	if got := itNum(t, first, "lineNumber"); got != 1 {
		t.Fatalf("first row lineNumber = %d, want 1", got)
	}
	fields := itMap(t, first, "fields")
	if got := itStr(t, fields, "name"); got != "Parent A" {
		t.Fatalf("mapped name = %q", got)
	}
	if got := itStr(t, fields, "phone"); got != "13800000001" {
		t.Fatalf("normalized phone = %q, want 13800000001", got)
	}
	if got := itNum(t, fields, "companionCount"); got != 1 {
		t.Fatalf("TOTAL 2 must map to companionCount 1, got %d", got)
	}
	if got := itNum(t, first, "requestedSize"); got != 2 {
		t.Fatalf("requestedSize = %d, want 2", got)
	}
	if code := itMap(t, first, "status")["code"]; code != "VALID" {
		t.Fatalf("row status = %v, want VALID", code)
	}
	if errs := first["errors"].([]any); len(errs) != 0 {
		t.Fatalf("valid row errors = %+v", errs)
	}
	// Status filter and pagination freeze the page envelope.
	filtered := a.request("BatchRows", map[string]any{"eventId": ev, "batchId": batchID, "status": "ERROR"}, http.StatusOK)
	if got := itNum(t, filtered, "total"); got != 0 {
		t.Fatalf("ERROR-filtered total = %d, want 0", got)
	}
	page2 := a.request("BatchRows", map[string]any{"eventId": ev, "batchId": batchID, "page": 2, "pageSize": 1}, http.StatusOK)
	if got := itNum(t, page2, "total"); got != 2 {
		t.Fatalf("paged total = %d, want 2", got)
	}
	if items := page2["items"].([]any); len(items) != 1 {
		t.Fatalf("page 2 items = %d, want 1", len(items))
	}

	selected := itRowIDs(t, a, ev, preview)
	commit := map[string]any{
		"operationId":    e.s.ID(),
		"eventId":        ev,
		"batchId":        batchID,
		"previewVersion": preview["previewVersion"],
		"selectedRowIds": selected,
	}
	// 2 + 2 people exceed capacity 3: the whole batch rolls back.
	a.request("BatchCommit", commit, http.StatusConflict)
	if a.lastCode != "CAPACITY_EXCEEDED" {
		t.Fatalf("over-capacity commit code = %q, want CAPACITY_EXCEEDED", a.lastCode)
	}
	itRowCount(t, a, ev, 0)
	if got := itNum(t, itGateStats(t, a, ev), "allocatedPeople"); got != 0 {
		t.Fatalf("rolled-back batch allocatedPeople = %d, want 0", got)
	}

	// Selecting only the first row commits atomically.
	commit["operationId"] = e.s.ID()
	commit["selectedRowIds"] = []any{selected[0]}
	result := a.request("BatchCommit", commit, http.StatusOK)
	if got := itNum(t, result, "createdCount"); got != 1 {
		t.Fatalf("createdCount = %d, want 1", got)
	}
	if got := itNum(t, result, "linkedCount"); got != 0 {
		t.Fatalf("linkedCount = %d, want 0", got)
	}
	if got := itNum(t, result, "allocatedPeople"); got != 2 {
		t.Fatalf("allocatedPeople = %d, want 2", got)
	}
	if itBool(t, result, "replayed") {
		t.Fatal("first commit must not be a replay")
	}
	itRowCount(t, a, ev, 1)
	if got := itNum(t, itGateStats(t, a, ev), "allocatedPeople"); got != 2 {
		t.Fatalf("allocatedPeople = %d, want 2", got)
	}
	detail = a.request("BatchDetail", map[string]any{"eventId": ev, "batchId": batchID}, http.StatusOK)
	if code := itMap(t, detail, "status")["code"]; code != "COMMITTED" {
		t.Fatalf("committed batch status = %v, want COMMITTED", code)
	}
	rows = a.request("BatchRows", map[string]any{"eventId": ev, "batchId": batchID}, http.StatusOK)
	committedRow := rows["items"].([]any)[0].(map[string]any)
	if code := itMap(t, committedRow, "status")["code"]; code != "COMMITTED" {
		t.Fatalf("committed row status = %v, want COMMITTED", code)
	}
	if committedRow["linkedPartyId"] == nil {
		t.Fatal("committed row must carry linkedPartyId")
	}

	// Replaying the sealed commit does not double-allocate.
	replay := a.request("BatchCommit", commit, http.StatusOK)
	if !itBool(t, replay, "replayed") {
		t.Fatalf("replayed commit flags = %+v", replay)
	}
	if got := itNum(t, replay, "createdCount"); got != 1 {
		t.Fatalf("replay createdCount = %d, want the sealed 1", got)
	}
	itRowCount(t, a, ev, 1)
	if got := itNum(t, itGateStats(t, a, ev), "allocatedPeople"); got != 2 {
		t.Fatalf("replay allocatedPeople = %d, want 2", got)
	}

	// A committed batch cannot be previewed again.
	a.write("BatchPreview", map[string]any{
		"eventId": ev, "batchId": batchID, "version": detail["version"],
		"mapping":     map[string]any{"nameColumn": "name", "countMode": "COMPANIONS"},
		"eligibility": "ELIGIBLE",
	}, http.StatusConflict)
	if a.lastCode != "EVENT_STATE_CONFLICT" {
		t.Fatalf("committed-batch preview code = %q, want canonical EVENT_STATE_CONFLICT", a.lastCode)
	}
	if !strings.Contains(a.lastMessage, "批次已提交") {
		t.Fatalf("committed-batch preview message = %q", a.lastMessage)
	}

	// Re-uploading the identical file links the same source records: the
	// committed line surfaces ALREADY_LINKED and cannot be committed again,
	// while the never-committed line stays VALID.
	batch2 := itUploadBatch(t, e, a, ev, csv)
	preview2 := itPreviewBatch(t, a, ev, batch2, map[string]any{
		"nameColumn": "name", "phoneColumn": "phone", "countColumn": "total", "countMode": "TOTAL",
	})
	if got := itNum(t, preview2, "validRows"); got != 1 {
		t.Fatalf("re-upload validRows = %d, want 1 (only the uncommitted line)", got)
	}
	linked := a.request("BatchRows", map[string]any{"eventId": ev, "batchId": itStr(t, batch2, "batchId"), "status": "ALREADY_LINKED"}, http.StatusOK)
	if got := itNum(t, linked, "total"); got != 1 {
		t.Fatalf("ALREADY_LINKED total = %d, want 1", got)
	}
	linkedID := itStr(t, linked["items"].([]any)[0].(map[string]any), "rowId")
	a.write("BatchCommit", map[string]any{
		"eventId": ev, "batchId": itStr(t, batch2, "batchId"), "previewVersion": preview2["previewVersion"],
		"selectedRowIds": []any{linkedID},
	}, http.StatusConflict)
	if a.lastCode != "EVENT_STATE_CONFLICT" {
		t.Fatalf("already-linked commit code = %q, want canonical EVENT_STATE_CONFLICT", a.lastCode)
	}
	if !strings.Contains(a.lastMessage, "选中行包含错误") {
		t.Fatalf("already-linked commit message = %q", a.lastMessage)
	}
	itRowCount(t, a, ev, 1)
}

// TestIntegrationImportErrorRowsAndDuplicateResolution ports the error-row
// exclusion (formulaRowsAreExcludedButOtherRowsRemainCommittable, via a CSV
// invalid count) and the duplicate review flow: suspect rows are uncommittable
// until KEEP_SEPARATE / LINK_EXISTING resolves them, and only unlinked eligible
// rows allocate places.
func TestIntegrationImportErrorRowsAndDuplicateResolution(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)
	ev := itPublishedEvent(t, e, a, 10)

	// TOTAL 0 is invalid: the row stays ERROR and unselectable.
	batch := itUploadBatch(t, e, a, ev, "name,phone,total\nParent A,13800000001,1\nBad Row,13800000002,0\n")
	preview := itPreviewBatch(t, a, ev, batch, map[string]any{
		"nameColumn": "name", "phoneColumn": "phone", "countColumn": "total", "countMode": "TOTAL",
	})
	if got := itNum(t, preview, "validRows"); got != 1 {
		t.Fatalf("validRows = %d, want 1", got)
	}
	if got := itNum(t, preview, "errorRows"); got != 1 {
		t.Fatalf("errorRows = %d, want 1", got)
	}
	errRows := a.request("BatchRows", map[string]any{"eventId": ev, "batchId": itStr(t, batch, "batchId"), "status": "ERROR"}, http.StatusOK)
	if got := itNum(t, errRows, "total"); got != 1 {
		t.Fatalf("ERROR rows total = %d, want 1", got)
	}
	errRow := errRows["items"].([]any)[0].(map[string]any)
	if got := itNum(t, errRow, "lineNumber"); got != 2 {
		t.Fatalf("error row lineNumber = %d, want 2", got)
	}
	errs := errRow["errors"].([]any)
	if len(errs) == 0 {
		t.Fatal("error row must carry errors")
	}
	if code := errs[0].(map[string]any)["code"]; code != "INVALID_FIELD" {
		t.Fatalf("error code = %v, want INVALID_FIELD", code)
	}

	all := itRowIDs(t, a, ev, batch)
	a.write("BatchCommit", map[string]any{
		"eventId": ev, "batchId": itStr(t, batch, "batchId"), "previewVersion": preview["previewVersion"],
		"selectedRowIds": all,
	}, http.StatusConflict)
	if a.lastCode != "EVENT_STATE_CONFLICT" {
		t.Fatalf("error-row commit code = %q, want canonical EVENT_STATE_CONFLICT", a.lastCode)
	}
	if !strings.Contains(a.lastMessage, "选中行包含错误") {
		t.Fatalf("error-row commit message = %q", a.lastMessage)
	}
	itRowCount(t, a, ev, 0)

	// The valid row alone commits.
	validRows := a.request("BatchRows", map[string]any{"eventId": ev, "batchId": itStr(t, batch, "batchId"), "status": "VALID"}, http.StatusOK)
	validID := itStr(t, validRows["items"].([]any)[0].(map[string]any), "rowId")
	result := a.write("BatchCommit", map[string]any{
		"eventId": ev, "batchId": itStr(t, batch, "batchId"), "previewVersion": preview["previewVersion"],
		"selectedRowIds": []any{validID},
	}, http.StatusOK)
	if got := itNum(t, result, "createdCount"); got != 1 {
		t.Fatalf("createdCount = %d, want 1", got)
	}
	itRowCount(t, a, ev, 1)
	parentA := a.request("PartyList", map[string]any{"eventId": ev, "page": 1, "pageSize": 10}, http.StatusOK)
	parentAID := itStr(t, parentA["items"].([]any)[0].(map[string]any), "partyId")

	// Both new rows share Parent A's phone: one matches an existing party, the
	// other duplicates the first within the file. Both become suspects.
	batch2 := itUploadBatch(t, e, a, ev, "name,phone,total\nParent C,13800000001,1\nParent D,13800000001,1\n")
	preview2 := itPreviewBatch(t, a, ev, batch2, map[string]any{
		"nameColumn": "name", "phoneColumn": "phone", "countColumn": "total", "countMode": "TOTAL",
	})
	if got := itNum(t, preview2, "suspectRows"); got != 2 {
		t.Fatalf("suspectRows = %d, want 2", got)
	}
	if got := itNum(t, preview2, "validRows"); got != 0 {
		t.Fatalf("suspect preview validRows = %d, want 0", got)
	}
	ids2 := itRowIDs(t, a, ev, batch2)
	batch2ID := itStr(t, batch2, "batchId")

	// A stale preview version blocks the commit.
	a.write("BatchCommit", map[string]any{
		"eventId": ev, "batchId": batch2ID, "previewVersion": 99, "selectedRowIds": ids2,
	}, http.StatusConflict)
	if a.lastCode != "EVENT_STATE_CONFLICT" {
		t.Fatalf("stale-preview commit code = %q, want canonical EVENT_STATE_CONFLICT", a.lastCode)
	}
	itRowCount(t, a, ev, 1)

	// KEEP_SEPARATE clears the in-file duplicate; LINK_EXISTING binds Parent C
	// to the existing party without allocating new places.
	resolved := a.write("DuplicateResolve", map[string]any{
		"eventId": ev, "batchId": batch2ID, "rowId": ids2[1],
		"previewVersion": preview2["previewVersion"], "resolution": "KEEP_SEPARATE", "reason": "Different family",
	}, http.StatusOK)
	if got := itNum(t, resolved, "suspectRows"); got != 1 {
		t.Fatalf("suspectRows after KEEP_SEPARATE = %d, want 1", got)
	}
	pv := itNum(t, resolved, "previewVersion")
	resolved = a.write("DuplicateResolve", map[string]any{
		"eventId": ev, "batchId": batch2ID, "rowId": ids2[0],
		"previewVersion": pv, "resolution": "LINK_EXISTING", "existingPartyId": parentAID, "reason": "Same family",
	}, http.StatusOK)
	if got := itNum(t, resolved, "suspectRows"); got != 0 {
		t.Fatalf("suspectRows after LINK_EXISTING = %d, want 0", got)
	}
	pv = itNum(t, resolved, "previewVersion")
	rows2 := a.request("BatchRows", map[string]any{"eventId": ev, "batchId": batch2ID}, http.StatusOK)
	linkedRow := rows2["items"].([]any)[0].(map[string]any)
	if got := itStr(t, linkedRow, "linkedPartyId"); got != parentAID {
		t.Fatalf("LINK_EXISTING linkedPartyId = %s, want %s", got, parentAID)
	}

	// A resolved (VALID) row no longer needs duplicate review.
	a.write("DuplicateResolve", map[string]any{
		"eventId": ev, "batchId": batch2ID, "rowId": ids2[1],
		"previewVersion": pv, "resolution": "EXCLUDE", "reason": "Again",
	}, http.StatusConflict)
	if a.lastCode != "EVENT_STATE_CONFLICT" {
		t.Fatalf("re-resolve code = %q, want canonical EVENT_STATE_CONFLICT", a.lastCode)
	}

	// Commit both: one created (1 place), one linked (0 places).
	result = a.write("BatchCommit", map[string]any{
		"eventId": ev, "batchId": batch2ID, "previewVersion": pv, "selectedRowIds": ids2,
	}, http.StatusOK)
	if got := itNum(t, result, "createdCount"); got != 1 {
		t.Fatalf("createdCount = %d, want 1", got)
	}
	if got := itNum(t, result, "linkedCount"); got != 1 {
		t.Fatalf("linkedCount = %d, want 1", got)
	}
	if got := itNum(t, result, "allocatedPeople"); got != 1 {
		t.Fatalf("allocatedPeople = %d, want 1 (linked rows allocate nothing)", got)
	}
	itRowCount(t, a, ev, 2)
	if got := itNum(t, itGateStats(t, a, ev), "allocatedPeople"); got != 2 {
		t.Fatalf("allocatedPeople = %d, want 2 (1 + 1)", got)
	}
}

// TestIntegrationImportPolicyDriftAndPhaseGuards ports
// policyDriftRequiresACompletelyNewPreview and the intake state guards: a policy
// change invalidates the preview, only READY events accept commits, check-in
// freezes intake, unknown rows and columns are rejected, and COMPANIONS mapping
// keeps the raw count.
func TestIntegrationImportPolicyDriftAndPhaseGuards(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)
	ev := itPublishedEvent(t, e, a, 10)

	batch := itUploadBatch(t, e, a, ev, "name\nSolo Parent\n")
	preview := itPreviewBatch(t, a, ev, batch, map[string]any{"nameColumn": "name", "countMode": "COMPANIONS"})
	if got := itNum(t, preview, "validRows"); got != 1 {
		t.Fatalf("validRows = %d, want 1", got)
	}

	// A policy change bumps policy_version: the old preview must not commit.
	policy := itPolicyInput(t, a, ev)
	policy["selfEditEnabled"] = !policy["selfEditEnabled"].(bool)
	a.write("PolicyUpdate", map[string]any{"eventId": ev, "policy": policy}, http.StatusOK)
	a.write("BatchCommit", map[string]any{
		"eventId": ev, "batchId": itStr(t, batch, "batchId"), "previewVersion": preview["previewVersion"],
		"selectedRowIds": itRowIDs(t, a, ev, batch),
	}, http.StatusConflict)
	if a.lastCode != "EVENT_STATE_CONFLICT" {
		t.Fatalf("policy-drift commit code = %q, want canonical EVENT_STATE_CONFLICT", a.lastCode)
	}
	if !strings.Contains(a.lastMessage, "活动字段规则已变化") {
		t.Fatalf("policy-drift message = %q", a.lastMessage)
	}
	itRowCount(t, a, ev, 0)

	// A completely new preview against the current batch version recovers.
	detail := a.request("BatchDetail", map[string]any{"eventId": ev, "batchId": itStr(t, batch, "batchId")}, http.StatusOK)
	preview2 := itPreviewBatch(t, a, ev, detail, map[string]any{"nameColumn": "name", "countMode": "COMPANIONS"})
	result := a.write("BatchCommit", map[string]any{
		"eventId": ev, "batchId": itStr(t, batch, "batchId"), "previewVersion": preview2["previewVersion"],
		"selectedRowIds": itRowIDs(t, a, ev, batch),
	}, http.StatusOK)
	if got := itNum(t, result, "createdCount"); got != 1 {
		t.Fatalf("createdCount after re-preview = %d, want 1", got)
	}

	// Unknown columns and unknown rows are rejected.
	batch2 := itUploadBatch(t, e, a, ev, "name,total\nCompanion Parent,2\n")
	a.write("BatchPreview", map[string]any{
		"eventId": ev, "batchId": itStr(t, batch2, "batchId"), "version": batch2["version"],
		"mapping":     map[string]any{"nameColumn": "missing", "countMode": "COMPANIONS"},
		"eligibility": "ELIGIBLE",
	}, http.StatusUnprocessableEntity)
	if a.lastCode != "FIELD_VALIDATION_FAILED" {
		t.Fatalf("bad-column preview code = %q, want canonical FIELD_VALIDATION_FAILED", a.lastCode)
	}
	preview3 := itPreviewBatch(t, a, ev, batch2, map[string]any{"nameColumn": "name", "countColumn": "total", "countMode": "COMPANIONS"})
	// COMPANIONS keeps the raw count: 2 companions -> requestedSize 3.
	rows := a.request("BatchRows", map[string]any{"eventId": ev, "batchId": itStr(t, batch2, "batchId")}, http.StatusOK)
	row := rows["items"].([]any)[0].(map[string]any)
	if got := itNum(t, itMap(t, row, "fields"), "companionCount"); got != 2 {
		t.Fatalf("COMPANIONS companionCount = %d, want 2", got)
	}
	if got := itNum(t, row, "requestedSize"); got != 3 {
		t.Fatalf("COMPANIONS requestedSize = %d, want 3", got)
	}
	a.write("BatchCommit", map[string]any{
		"eventId": ev, "batchId": itStr(t, batch2, "batchId"), "previewVersion": preview3["previewVersion"],
		"selectedRowIds": []any{e.s.ID()},
	}, http.StatusConflict)
	if a.lastCode != "EVENT_STATE_CONFLICT" {
		t.Fatalf("unknown-row commit code = %q, want canonical EVENT_STATE_CONFLICT", a.lastCode)
	}
	if !strings.Contains(a.lastMessage, "存在未知预览行") {
		t.Fatalf("unknown-row message = %q", a.lastMessage)
	}

	// Check-in freezes every intake mutation.
	itTransition(t, a, ev, "OPEN", "CHECKIN_OPEN")
	a.upload("BatchUpload", map[string]string{
		"operationId": e.s.ID(), "eventId": ev, "sourceLabel": "Late",
	}, "late.csv", []byte("name\nLate Parent\n"), http.StatusConflict)
	if a.lastCode != "EVENT_STATE_CONFLICT" {
		t.Fatalf("upload after OPEN code = %q, want canonical EVENT_STATE_CONFLICT", a.lastCode)
	}
	a.write("BatchPreview", map[string]any{
		"eventId": ev, "batchId": itStr(t, batch2, "batchId"), "version": batch2["version"],
		"mapping":     map[string]any{"nameColumn": "name", "countMode": "COMPANIONS"},
		"eligibility": "ELIGIBLE",
	}, http.StatusConflict)
	a.write("BatchCommit", map[string]any{
		"eventId": ev, "batchId": itStr(t, batch2, "batchId"), "previewVersion": preview3["previewVersion"],
		"selectedRowIds": itRowIDs(t, a, ev, batch2),
	}, http.StatusConflict)
	if a.lastCode != "EVENT_STATE_CONFLICT" {
		t.Fatalf("commit after OPEN code = %q, want canonical EVENT_STATE_CONFLICT", a.lastCode)
	}
	itRowCount(t, a, ev, 1)

	// A DRAFT event accepts upload/preview but not commit; publishing unblocks it.
	draft := a.write("EventCreate", map[string]any{
		"title":                "Draft " + e.s.ID(),
		"startsAt":             timeNowPlus(2),
		"endsAt":               timeNowPlus(5),
		"timeZone":             "Asia/Shanghai",
		"registrationStartsAt": timeNowPlus(-1),
		"registrationEndsAt":   timeNowPlus(1),
		"capacity":             5,
	}, http.StatusOK)
	draftID := itStr(t, draft, "eventId")
	batch3 := itUploadBatch(t, e, a, draftID, "name\nDraft Parent\n")
	preview4 := itPreviewBatch(t, a, draftID, batch3, map[string]any{"nameColumn": "name", "countMode": "COMPANIONS"})
	a.write("BatchCommit", map[string]any{
		"eventId": draftID, "batchId": itStr(t, batch3, "batchId"), "previewVersion": preview4["previewVersion"],
		"selectedRowIds": itRowIDs(t, a, draftID, batch3),
	}, http.StatusConflict)
	if a.lastCode != "EVENT_STATE_CONFLICT" {
		t.Fatalf("DRAFT commit code = %q, want canonical EVENT_STATE_CONFLICT", a.lastCode)
	}
	if !strings.Contains(a.lastMessage, "仅已发布准备中的活动可提交名单") {
		t.Fatalf("DRAFT commit message = %q", a.lastMessage)
	}
	itTransition(t, a, draftID, "PUBLISH", "READY")
	result = a.write("BatchCommit", map[string]any{
		"eventId": draftID, "batchId": itStr(t, batch3, "batchId"), "previewVersion": preview4["previewVersion"],
		"selectedRowIds": itRowIDs(t, a, draftID, batch3),
	}, http.StatusOK)
	if got := itNum(t, result, "createdCount"); got != 1 {
		t.Fatalf("published DRAFT createdCount = %d, want 1", got)
	}
}
