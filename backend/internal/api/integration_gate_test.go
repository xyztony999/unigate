package api

// Port of the gate/device half of WorkflowIntegrationTest
// (onsiteChangesAreSeparateFromAdmissionAndStaffDevicesNeedAnAccount) plus the
// review-phase guard from reviewAndDeltaApprovalNeverAllocateUnapprovedPeople:
// device registration/claim (STAFF and KIOSK), ticket resolve ->分批 GateCommit
// with idempotent replay, corrections preserving the original admission facts,
// walk-in / on-site increase, and the CHECKIN_OPEN / READY phase guards.

import (
	"net/http"
	"testing"
	"time"
)

// itPublishedEvent creates and publishes an event (Java event()+publish()).
func itPublishedEvent(t *testing.T, e *itEnv, a *itBrowser, capacity int) string {
	t.Helper()
	now := time.Now().UTC()
	created := a.write("EventCreate", map[string]any{
		"title":                "Gate " + e.s.ID(),
		"startsAt":             now.Add(2 * time.Hour).Format(time.RFC3339Nano),
		"endsAt":               now.Add(5 * time.Hour).Format(time.RFC3339Nano),
		"timeZone":             "Asia/Shanghai",
		"registrationStartsAt": now.Add(-1 * time.Hour).Format(time.RFC3339Nano),
		"registrationEndsAt":   now.Add(1 * time.Hour).Format(time.RFC3339Nano),
		"capacity":             capacity,
	}, http.StatusOK)
	ev := itStr(t, created, "eventId")
	itTransition(t, a, ev, "PUBLISH", "READY")
	return ev
}

// itTransition reads the current event version and applies a lifecycle action,
// asserting the resulting status code.
func itTransition(t *testing.T, a *itBrowser, ev, action, wantStatus string) map[string]any {
	t.Helper()
	detail := a.request("EventDetail", map[string]any{"eventId": ev}, http.StatusOK)
	out := a.write("EventTransition", map[string]any{
		"eventId": ev, "version": detail["version"], "action": action,
	}, http.StatusOK)
	if code := itMap(t, out, "status")["code"]; code != wantStatus {
		t.Fatalf("EventTransition %s status = %v, want %s", action, code, wantStatus)
	}
	return out
}

// itRegisteredParty submits an anonymous registration, exchanges the access
// bundle and reads the ticket credential (Java registration()+PartyExchange).
func itRegisteredParty(t *testing.T, e *itEnv, ev, name string, companions int) (*itBrowser, map[string]any, string) {
	t.Helper()
	p := e.browser(t)
	p.bootstrap()
	pub := p.request("PublicEvent", map[string]any{"eventId": ev}, http.StatusOK)
	submitted := p.write("SubmitRegistration", map[string]any{
		"eventId":               ev,
		"fields":                map[string]any{"name": name, "companionCount": companions},
		"acceptedNoticeVersion": itStr(t, pub, "noticeVersion"),
	}, http.StatusOK)
	party := itMap(t, submitted, "party")
	access := itMap(t, submitted, "access")
	p.write("PartyExchange", map[string]any{
		"queryToken":  itStr(t, access, "queryToken"),
		"changeToken": itStr(t, access, "changeToken"),
	}, http.StatusOK)
	cred := p.request("MyCredential", map[string]any{}, http.StatusOK)
	// PENDING (review-mode) parties have no ticket yet.
	token, _ := cred["ticketToken"].(string)
	return p, party, token
}

// itNewStaff creates an account and completes the forced initial password
// change lifecycle (Java newAccount()+staff()). Fixed test credentials only.
func itNewStaff(t *testing.T, e *itEnv, a *itBrowser, role string) (*itBrowser, map[string]any) {
	t.Helper()
	account := a.write("AccountCreate", map[string]any{
		"username":          "staff-" + e.s.ID(),
		"displayName":       "Test " + role,
		"temporaryPassword": "Temporary-staff-123!",
		"roles":             []any{role},
	}, http.StatusOK)
	b := e.browser(t)
	b.bootstrap()
	b.write("Login", map[string]any{"username": account["username"], "password": "Temporary-staff-123!"}, http.StatusOK)
	b.write("PasswordChange", map[string]any{"oldPassword": "Temporary-staff-123!", "newPassword": "Changed-staff-123!"}, http.StatusOK)
	b.bootstrap()
	b.write("Login", map[string]any{"username": account["username"], "password": "Changed-staff-123!"}, http.StatusOK)
	return b, account
}

// itClaimedDevice registers a device as the organizer and claims it from a
// fresh terminal browser (ug_device cookie + device csrf).
func itClaimedDevice(t *testing.T, e *itEnv, a *itBrowser, ev, mode string) (*itBrowser, map[string]any) {
	t.Helper()
	reg := a.write("DeviceRegister", map[string]any{
		"eventId": ev, "name": "Gate test device " + mode, "mode": mode,
	}, http.StatusOK)
	device := itMap(t, reg, "device")
	if code := itMap(t, device, "mode")["code"]; code != mode {
		t.Fatalf("registered device mode = %v, want %s", code, mode)
	}
	if code := itMap(t, device, "status")["code"]; code != "UNCLAIMED" {
		t.Fatalf("registered device status = %v, want UNCLAIMED", code)
	}
	terminal := e.browser(t)
	terminal.bootstrap()
	claimed := terminal.write("DeviceClaim", map[string]any{
		"deviceId":    itStr(t, device, "deviceId"),
		"pairingCode": itStr(t, reg, "pairingCode"),
	}, http.StatusOK)
	if _, ok := terminal.cookies["ug_device"]; !ok {
		t.Fatalf("DeviceClaim must set the ug_device cookie: %+v", terminal.cookies)
	}
	if itStr(t, claimed, "csrfToken") == "" {
		t.Fatal("DeviceClaim must issue a device csrfToken")
	}
	return terminal, reg
}

// itGateStats reads Statistics and asserts the admission invariants:
// 0 <= netAdmitted <= allocatedPeople and unusedPlaces = allocated - net.
func itGateStats(t *testing.T, a *itBrowser, ev string) map[string]any {
	t.Helper()
	stats := a.request("Statistics", map[string]any{"eventId": ev}, http.StatusOK)
	allocated := itNum(t, stats, "allocatedPeople")
	net := itNum(t, stats, "netAdmissions")
	gross := itNum(t, stats, "grossAdmissions")
	corrected := itNum(t, stats, "correctedPeople")
	if net < 0 || net > allocated {
		t.Fatalf("invariant violated: netAdmissions=%d allocatedPeople=%d", net, allocated)
	}
	if got := itNum(t, stats, "unusedPlaces"); got != allocated-net {
		t.Fatalf("unusedPlaces = %d, want allocated-net = %d", got, allocated-net)
	}
	if net != gross-corrected {
		t.Fatalf("netAdmissions = %d, want gross-corrected = %d", net, gross-corrected)
	}
	return stats
}

// TestIntegrationGateCheckinFlowAndReplay ports the on-site workflow: a STAFF
// device needs a logged-in checker, walk-in/increase allocate without admitting,
// GateCommit admits in batches with idempotent replay, GateCorrect preserves the
// original record, and PAUSE/RESUME/END/revoke guard every gate operation.
func TestIntegrationGateCheckinFlowAndReplay(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)
	ev := itPublishedEvent(t, e, a, 10)

	_, party, ticketToken := itRegisteredParty(t, e, ev, "Gate Parent", 0)
	partyID := itStr(t, party, "partyId")
	if got := itNum(t, party, "approvedSize"); got != 1 {
		t.Fatalf("registration approvedSize = %d, want 1", got)
	}

	terminal, _ := itClaimedDevice(t, e, a, ev, "STAFF")
	// A STAFF-mode device without a logged-in account cannot resolve tickets.
	terminal.request("TicketResolve", map[string]any{"eventId": ev, "ticketToken": ticketToken}, http.StatusUnauthorized)
	if terminal.lastCode != "SESSION_EXPIRED" {
		t.Fatalf("unauthenticated staff device code = %q, want SESSION_EXPIRED", terminal.lastCode)
	}

	checker, account := itNewStaff(t, e, a, "CHECKER")
	a.write("GrantSave", map[string]any{
		"eventId":        ev,
		"accountId":      itStr(t, account, "accountId"),
		"role":           "CHECKER",
		"allowedFields":  []any{"name"},
		"allowedActions": []any{"CHECKIN"},
		"expiresAt":      time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339Nano),
	}, http.StatusOK)
	checker.cookies["ug_device"] = terminal.cookies["ug_device"]

	// Check-in is closed before OPEN even for an authorized checker.
	checker.request("TicketResolve", map[string]any{"eventId": ev, "ticketToken": ticketToken}, http.StatusConflict)
	if checker.lastCode != "EVENT_STATE_CONFLICT" {
		t.Fatalf("resolve before OPEN code = %q, want canonical EVENT_STATE_CONFLICT", checker.lastCode)
	}
	itTransition(t, a, ev, "OPEN", "CHECKIN_OPEN")

	// Walk-in allocates a place but admits nobody.
	walkIn := a.write("GateWalkIn", map[string]any{
		"eventId": ev,
		"fields":  map[string]any{"name": "Onsite Parent"},
		"reason":  "Unregistered visitor",
	}, http.StatusOK)
	if got := itNum(t, walkIn, "netAdmitted"); got != 0 {
		t.Fatalf("walk-in netAdmitted = %d, want 0", got)
	}
	if got := itNum(t, walkIn, "approvedSize"); got != 1 {
		t.Fatalf("walk-in approvedSize = %d, want 1", got)
	}
	if code := itMap(t, walkIn, "sourceType")["code"]; code != "WALK_IN" {
		t.Fatalf("walk-in sourceType = %v, want WALK_IN", code)
	}
	walkInID := itStr(t, walkIn, "partyId")

	// On-site increase raises the approved size without admitting.
	increased := a.write("GateIncrease", map[string]any{
		"eventId":         ev,
		"partyId":         partyID,
		"version":         party["version"],
		"additionalCount": 2,
		"reason":          "Additional family",
	}, http.StatusOK)
	if got := itNum(t, increased, "approvedSize"); got != 3 {
		t.Fatalf("increased approvedSize = %d, want 3", got)
	}
	if got := itNum(t, increased, "netAdmitted"); got != 0 {
		t.Fatalf("increase must not admit: netAdmitted = %d", got)
	}
	stats := itGateStats(t, a, ev)
	if got := itNum(t, stats, "allocatedPeople"); got != 4 {
		t.Fatalf("allocatedPeople = %d, want 4 (1 registration + 1 walk-in + 2 increase)", got)
	}
	if got := itNum(t, stats, "netAdmissions"); got != 0 {
		t.Fatalf("netAdmissions = %d, want 0 before any commit", got)
	}
	if got := itNum(t, stats, "onsiteAddedPlaces"); got != 2 {
		t.Fatalf("onsiteAddedPlaces = %d, want 2", got)
	}
	if got := itNum(t, stats, "walkInGroups"); got != 1 {
		t.Fatalf("walkInGroups = %d, want 1", got)
	}

	session := checker.request("DeviceDetail", map[string]any{}, http.StatusOK)
	if code := itMap(t, session, "eventStatus")["code"]; code != "CHECKIN_OPEN" {
		t.Fatalf("device session eventStatus = %v, want CHECKIN_OPEN", code)
	}
	lookup := checker.request("GateLookup", map[string]any{"eventId": ev, "query": "Onsite"}, http.StatusOK)
	if got := itNum(t, lookup, "total"); got != 1 {
		t.Fatalf("GateLookup total = %d, want 1", got)
	}
	cand := lookup["items"].([]any)[0].(map[string]any)
	if got := itStr(t, cand, "partyId"); got != walkInID {
		t.Fatalf("GateLookup partyId = %s, want walk-in %s", got, walkInID)
	}
	if got := itStr(t, cand, "displayNameMasked"); got != "O**" {
		t.Fatalf("masked name = %q, want O** (grant allows name only)", got)
	}
	if _, has := cand["phoneSuffix"]; has && cand["phoneSuffix"] != nil {
		t.Fatalf("phone must stay hidden without a phone grant: %+v", cand)
	}

	// Resolve the group code, then admit in two batches (1 + 2 of 3).
	scan := checker.request("TicketResolve", map[string]any{"eventId": ev, "ticketToken": ticketToken}, http.StatusOK)
	if got := itNum(t, scan, "approvedSize"); got != 3 {
		t.Fatalf("resolve approvedSize = %d, want 3", got)
	}
	if got := itNum(t, scan, "remainingCount"); got != 3 {
		t.Fatalf("resolve remainingCount = %d, want 3", got)
	}
	commit := map[string]any{
		"operationId":          e.s.ID(),
		"eventId":              ev,
		"scanContextToken":     itStr(t, scan, "scanContextToken"),
		"expectedPartyVersion": scan["partyVersion"],
		"count":                1,
	}
	first := checker.request("GateCommit", commit, http.StatusOK)
	recordID := itStr(t, first, "recordId")
	if code := itMap(t, first, "status")["code"]; code != "SUCCEEDED" {
		t.Fatalf("GateCommit status = %v, want SUCCEEDED", code)
	}
	if got := itNum(t, first, "committedCount"); got != 1 {
		t.Fatalf("committedCount = %d, want 1", got)
	}
	if got := itNum(t, first, "remainingCount"); got != 2 {
		t.Fatalf("remainingCount = %d, want 2", got)
	}
	if itBool(t, first, "replayed") || !itBool(t, first, "showAsNewAdmission") {
		t.Fatalf("first commit must be a new admission: %+v", first)
	}

	// Same operationId replays the sealed result without double counting, even
	// with a freshly resolved scan token (the digest binds party+device).
	rescan := checker.request("TicketResolve", map[string]any{"eventId": ev, "ticketToken": ticketToken}, http.StatusOK)
	if got := itNum(t, rescan, "netAdmitted"); got != 1 {
		t.Fatalf("netAdmitted after first commit = %d, want 1", got)
	}
	if got := itNum(t, rescan, "remainingCount"); got != 2 {
		t.Fatalf("remainingCount after first commit = %d, want 2", got)
	}
	replayArgs := map[string]any{}
	for k, v := range commit {
		replayArgs[k] = v
	}
	replayArgs["scanContextToken"] = itStr(t, rescan, "scanContextToken")
	replay := checker.request("GateCommit", replayArgs, http.StatusOK)
	if !itBool(t, replay, "replayed") || itBool(t, replay, "showAsNewAdmission") {
		t.Fatalf("replayed commit flags = %+v, want replayed=true showAsNewAdmission=false", replay)
	}
	if got := itStr(t, replay, "recordId"); got != recordID {
		t.Fatalf("replay recordId = %s, want %s", got, recordID)
	}
	if got := itNum(t, replay, "committedCount"); got != 1 {
		t.Fatalf("replay committedCount = %d, want the sealed 1", got)
	}
	records := checker.request("GateRecords", map[string]any{"eventId": ev}, http.StatusOK)
	if got := itNum(t, records, "total"); got != 1 {
		t.Fatalf("GateRecords total after replay = %d, want 1 (no duplicate)", got)
	}
	stats = itGateStats(t, a, ev)
	if got := itNum(t, stats, "netAdmissions"); got != 1 {
		t.Fatalf("netAdmissions after replay = %d, want 1", got)
	}
	if got := itNum(t, stats, "grossAdmissions"); got != 1 {
		t.Fatalf("grossAdmissions after replay = %d, want 1", got)
	}

	// Same operationId with different content is an idempotency conflict.
	conflictArgs := map[string]any{}
	for k, v := range replayArgs {
		conflictArgs[k] = v
	}
	conflictArgs["count"] = 2
	checker.request("GateCommit", conflictArgs, http.StatusBadRequest)
	if checker.lastCode != "IDEMPOTENCY_PAYLOAD_MISMATCH" {
		t.Fatalf("changed-payload replay code = %q, want IDEMPOTENCY_PAYLOAD_MISMATCH", checker.lastCode)
	}

	// The original operation is queryable via GateOperation and OperationResult.
	queried := checker.request("GateOperation", map[string]any{"operationId": commit["operationId"]}, http.StatusOK)
	if !itBool(t, queried, "replayed") || itBool(t, queried, "showAsNewAdmission") {
		t.Fatalf("GateOperation flags = %+v", queried)
	}
	if got := itStr(t, queried, "recordId"); got != recordID {
		t.Fatalf("GateOperation recordId = %s, want %s", got, recordID)
	}
	opResult := checker.request("OperationResult", map[string]any{
		"eventId": ev, "operationId": commit["operationId"], "action": "GateCommit",
	}, http.StatusOK)
	if code := itMap(t, opResult, "status")["code"]; code != "SUCCEEDED" {
		t.Fatalf("OperationResult status = %v, want SUCCEEDED", code)
	}
	if got := itStr(t, opResult, "resourceId"); got != recordID {
		t.Fatalf("OperationResult resourceId = %s, want %s", got, recordID)
	}
	if !itBool(t, opResult, "replayed") {
		t.Fatal("OperationResult must flag replayed")
	}

	// Second batch admits the remaining two; the balance then blocks more.
	scan2 := checker.request("TicketResolve", map[string]any{"eventId": ev, "ticketToken": ticketToken}, http.StatusOK)
	second := checker.write("GateCommit", map[string]any{
		"eventId":              ev,
		"scanContextToken":     itStr(t, scan2, "scanContextToken"),
		"expectedPartyVersion": scan2["partyVersion"],
		"count":                2,
	}, http.StatusOK)
	if got := itNum(t, second, "committedCount"); got != 2 {
		t.Fatalf("second committedCount = %d, want 2", got)
	}
	if got := itNum(t, second, "remainingCount"); got != 0 {
		t.Fatalf("second remainingCount = %d, want 0", got)
	}
	scan3 := checker.request("TicketResolve", map[string]any{"eventId": ev, "ticketToken": ticketToken}, http.StatusOK)
	if got := itNum(t, scan3, "netAdmitted"); got != 3 {
		t.Fatalf("netAdmitted = %d, want 3", got)
	}
	if got := itNum(t, scan3, "remainingCount"); got != 0 {
		t.Fatalf("remainingCount = %d, want approvedSize-netAdmitted = 0", got)
	}
	checker.write("GateCommit", map[string]any{
		"eventId":              ev,
		"scanContextToken":     itStr(t, scan3, "scanContextToken"),
		"expectedPartyVersion": scan3["partyVersion"],
		"count":                1,
	}, http.StatusConflict)
	if checker.lastCode != "PARTY_REMAINING_EXCEEDED" {
		t.Fatalf("over-commit code = %q, want canonical PARTY_REMAINING_EXCEEDED", checker.lastCode)
	}

	// Correction keeps the original admission record and releases no places.
	detail := a.request("PartyDetail", map[string]any{"eventId": ev, "partyId": partyID}, http.StatusOK)
	corrected := a.write("GateCorrect", map[string]any{
		"eventId":        ev,
		"recordId":       recordID,
		"partyVersion":   detail["version"],
		"correctedCount": 1,
		"reason":         "One person left early",
	}, http.StatusOK)
	itStr(t, corrected, "correctionId")
	if got := itStr(t, corrected, "originalRecordId"); got != recordID {
		t.Fatalf("correction originalRecordId = %s, want %s", got, recordID)
	}
	records = a.request("GateRecords", map[string]any{"eventId": ev, "partyId": partyID}, http.StatusOK)
	if got := itNum(t, records, "total"); got != 2 {
		t.Fatalf("GateRecords total = %d, want 2 original records", got)
	}
	var original map[string]any
	for _, item := range records["items"].([]any) {
		m := item.(map[string]any)
		if itStr(t, m, "recordId") == recordID {
			original = m
		}
	}
	if original == nil {
		t.Fatalf("original record %s missing from GateRecords", recordID)
	}
	if got := itNum(t, original, "admittedCount"); got != 1 {
		t.Fatalf("original admittedCount = %d, want the preserved 1", got)
	}
	if got := itNum(t, original, "correctedCount"); got != 1 {
		t.Fatalf("original correctedCount = %d, want 1", got)
	}
	stats = itGateStats(t, a, ev)
	if got := itNum(t, stats, "grossAdmissions"); got != 3 {
		t.Fatalf("grossAdmissions = %d, want 3 (facts preserved)", got)
	}
	if got := itNum(t, stats, "correctedPeople"); got != 1 {
		t.Fatalf("correctedPeople = %d, want 1", got)
	}
	if got := itNum(t, stats, "netAdmissions"); got != 2 {
		t.Fatalf("netAdmissions = %d, want 2", got)
	}
	if got := itNum(t, stats, "allocatedPeople"); got != 4 {
		t.Fatalf("correction must not release places: allocatedPeople = %d, want 4", got)
	}

	// A checker cannot increase places; only the organizer can.
	checker.write("GateIncrease", map[string]any{
		"eventId": ev, "partyId": partyID, "version": 1, "additionalCount": 1, "reason": "Forbidden",
	}, http.StatusForbidden)
	if checker.lastCode != "FORBIDDEN" {
		t.Fatalf("checker GateIncrease code = %q, want FORBIDDEN", checker.lastCode)
	}

	// PAUSE closes resolve/commit/walk-in but still allows corrections.
	itTransition(t, a, ev, "PAUSE", "CHECKIN_PAUSED")
	checker.request("TicketResolve", map[string]any{"eventId": ev, "ticketToken": ticketToken}, http.StatusConflict)
	if checker.lastCode != "EVENT_STATE_CONFLICT" {
		t.Fatalf("paused resolve code = %q, want canonical EVENT_STATE_CONFLICT", checker.lastCode)
	}
	// Replaying the succeeded operation still serves the sealed result (the
	// idempotency branch runs before the phase guard), but a NEW commit is blocked.
	pausedReplay := checker.request("GateCommit", replayArgs, http.StatusOK)
	if !itBool(t, pausedReplay, "replayed") || itBool(t, pausedReplay, "showAsNewAdmission") {
		t.Fatalf("paused replay flags = %+v", pausedReplay)
	}
	checker.write("GateCommit", map[string]any{
		"eventId":              ev,
		"scanContextToken":     itStr(t, scan3, "scanContextToken"),
		"expectedPartyVersion": scan3["partyVersion"],
		"count":                1,
	}, http.StatusConflict)
	if checker.lastCode != "EVENT_STATE_CONFLICT" {
		t.Fatalf("paused new commit code = %q, want canonical EVENT_STATE_CONFLICT", checker.lastCode)
	}
	a.write("GateWalkIn", map[string]any{"eventId": ev, "fields": map[string]any{}, "reason": "Paused"}, http.StatusConflict)
	if a.lastCode != "EVENT_STATE_CONFLICT" {
		t.Fatalf("paused walk-in code = %q, want canonical EVENT_STATE_CONFLICT", a.lastCode)
	}
	detail = a.request("PartyDetail", map[string]any{"eventId": ev, "partyId": partyID}, http.StatusOK)
	a.write("GateCorrect", map[string]any{
		"eventId": ev, "recordId": recordID, "partyVersion": detail["version"], "correctedCount": 1, "reason": "Too much",
	}, http.StatusConflict)
	if a.lastCode != "EVENT_STATE_CONFLICT" {
		t.Fatalf("over-correction code = %q, want canonical EVENT_STATE_CONFLICT (phase guard passed while paused)", a.lastCode)
	}

	itTransition(t, a, ev, "RESUME", "CHECKIN_OPEN")
	checker.request("TicketResolve", map[string]any{"eventId": ev, "ticketToken": ticketToken}, http.StatusOK)

	// Revoking the device cuts off the terminal immediately.
	devices := a.request("DeviceList", map[string]any{"eventId": ev}, http.StatusOK)
	device := devices["items"].([]any)[0].(map[string]any)
	a.write("DeviceRevoke", map[string]any{
		"eventId": ev, "deviceId": itStr(t, device, "deviceId"), "version": device["version"], "reason": "Station closed",
	}, http.StatusOK)
	checker.request("DeviceDetail", map[string]any{}, http.StatusUnauthorized)
	if checker.lastCode != "UNAUTHENTICATED" {
		t.Fatalf("revoked device code = %q, want canonical UNAUTHENTICATED", checker.lastCode)
	}

	// END freezes every on-site mutation.
	itTransition(t, a, ev, "END", "ENDED")
	a.write("GateWalkIn", map[string]any{"eventId": ev, "fields": map[string]any{}, "reason": "Too late"}, http.StatusConflict)
	if a.lastCode != "EVENT_STATE_CONFLICT" {
		t.Fatalf("ended walk-in code = %q, want canonical EVENT_STATE_CONFLICT", a.lastCode)
	}
	a.write("GateIncrease", map[string]any{
		"eventId": ev, "partyId": partyID, "version": detail["version"], "additionalCount": 1, "reason": "Too late",
	}, http.StatusConflict)
	if a.lastCode != "EVENT_STATE_CONFLICT" {
		t.Fatalf("ended increase code = %q, want canonical EVENT_STATE_CONFLICT", a.lastCode)
	}

	audit := a.request("AuditList", map[string]any{"eventId": ev, "page": 1, "pageSize": 100}, http.StatusOK)
	if itNum(t, audit, "total") == 0 {
		t.Fatal("AuditList must record the gate operations")
	}
	foundCommit := false
	for _, item := range audit["items"].([]any) {
		if item.(map[string]any)["action"] == "GateCommit" {
			foundCommit = true
		}
	}
	if !foundCommit {
		t.Fatal("AuditList must contain the GateCommit action")
	}
}

// TestIntegrationGateKioskModeAndReviewGuard covers the self-service KIOSK
// device (no account, no people search, scan-bound operation lookup) and the
// rule that reviews are decided only while the event is READY.
func TestIntegrationGateKioskModeAndReviewGuard(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)
	ev := itPublishedEvent(t, e, a, 5)

	_, _, ticket1 := itRegisteredParty(t, e, ev, "Kiosk Parent", 1)
	_, _, ticket2 := itRegisteredParty(t, e, ev, "Other Parent", 0)

	kiosk, _ := itClaimedDevice(t, e, a, ev, "KIOSK")
	// A kiosk needs no account but cannot search the roster.
	kiosk.request("GateLookup", map[string]any{"eventId": ev, "query": "Parent"}, http.StatusForbidden)
	if kiosk.lastCode != "FORBIDDEN" {
		t.Fatalf("kiosk GateLookup code = %q, want FORBIDDEN", kiosk.lastCode)
	}
	itTransition(t, a, ev, "OPEN", "CHECKIN_OPEN")
	scan1 := kiosk.request("TicketResolve", map[string]any{"eventId": ev, "ticketToken": ticket1}, http.StatusOK)
	if got := itNum(t, scan1, "approvedSize"); got != 2 {
		t.Fatalf("kiosk resolve approvedSize = %d, want 2", got)
	}
	commit := map[string]any{
		"operationId":          e.s.ID(),
		"eventId":              ev,
		"scanContextToken":     itStr(t, scan1, "scanContextToken"),
		"expectedPartyVersion": scan1["partyVersion"],
		"count":                1,
	}
	kiosk.request("GateCommit", commit, http.StatusOK)

	// Kiosk operation lookup requires the original group's scan context.
	kiosk.request("GateOperation", map[string]any{"operationId": commit["operationId"]}, http.StatusForbidden)
	if kiosk.lastCode != "FORBIDDEN" {
		t.Fatalf("kiosk GateOperation without scan code = %q, want FORBIDDEN", kiosk.lastCode)
	}
	fresh1 := kiosk.request("TicketResolve", map[string]any{"eventId": ev, "ticketToken": ticket1}, http.StatusOK)
	queried := kiosk.request("GateOperation", map[string]any{
		"operationId": commit["operationId"], "scanContextToken": itStr(t, fresh1, "scanContextToken"),
	}, http.StatusOK)
	if !itBool(t, queried, "replayed") || itBool(t, queried, "showAsNewAdmission") {
		t.Fatalf("kiosk GateOperation flags = %+v", queried)
	}
	// Another group's scan context must not reveal this operation.
	scan2 := kiosk.request("TicketResolve", map[string]any{"eventId": ev, "ticketToken": ticket2}, http.StatusOK)
	kiosk.request("GateOperation", map[string]any{
		"operationId": commit["operationId"], "scanContextToken": itStr(t, scan2, "scanContextToken"),
	}, http.StatusNotFound)
	if kiosk.lastCode != "RESOURCE_NOT_FOUND" {
		t.Fatalf("cross-party GateOperation code = %q, want RESOURCE_NOT_FOUND", kiosk.lastCode)
	}

	// A scan context minted for another device is rejected.
	kiosk2, _ := itClaimedDevice(t, e, a, ev, "KIOSK")
	otherScan := kiosk2.request("TicketResolve", map[string]any{"eventId": ev, "ticketToken": ticket1}, http.StatusOK)
	crossArgs := map[string]any{}
	for k, v := range commit {
		crossArgs[k] = v
	}
	crossArgs["operationId"] = e.s.ID()
	crossArgs["scanContextToken"] = itStr(t, otherScan, "scanContextToken")
	crossArgs["expectedPartyVersion"] = otherScan["partyVersion"]
	kiosk.request("GateCommit", crossArgs, http.StatusForbidden)
	if kiosk.lastCode != "FORBIDDEN" {
		t.Fatalf("cross-device scan code = %q, want FORBIDDEN", kiosk.lastCode)
	}
	// The owning device commits the remaining place.
	own := map[string]any{}
	for k, v := range crossArgs {
		own[k] = v
	}
	own["operationId"] = e.s.ID()
	last := kiosk2.request("GateCommit", own, http.StatusOK)
	if got := itNum(t, last, "remainingCount"); got != 0 {
		t.Fatalf("remainingCount = %d, want 0", got)
	}
	if got := itNum(t, itGateStats(t, a, ev), "netAdmissions"); got != 2 {
		t.Fatalf("netAdmissions = %d, want 2", got)
	}

	// Reviews are decided only while READY: approve one of two pending groups,
	// open check-in, and the second decision is phase-blocked.
	rev := itPublishedEvent(t, e, a, 3)
	policy := a.request("PolicyGet", map[string]any{"eventId": rev}, http.StatusOK)
	a.write("PolicyUpdate", map[string]any{
		"eventId": rev,
		"policy": map[string]any{
			"fields":                   policy["fields"],
			"approvalMode":             "REVIEW",
			"phoneVerificationEnabled": policy["phoneVerificationEnabled"],
			"selfEditEnabled":          policy["selfEditEnabled"],
			"selfCancelEnabled":        policy["selfCancelEnabled"],
			"retentionDays":            policy["retentionDays"],
			"version":                  policy["version"],
		},
	}, http.StatusOK)
	_, pending1, _ := itRegisteredParty(t, e, rev, "Reviewed Parent", 0)
	_, _, _ = itRegisteredParty(t, e, rev, "Second Parent", 0)
	if code := itMap(t, pending1, "status")["code"]; code != "PENDING" {
		t.Fatalf("REVIEW-mode registration status = %v, want PENDING", code)
	}
	if got := itNum(t, itGateStats(t, a, rev), "allocatedPeople"); got != 0 {
		t.Fatalf("pending reviews must not allocate: allocatedPeople = %d", got)
	}
	reviews := a.request("ReviewList", map[string]any{"eventId": rev, "page": 1, "pageSize": 20, "result": "PENDING"}, http.StatusOK)
	if got := itNum(t, reviews, "total"); got != 2 {
		t.Fatalf("pending review total = %d, want 2", got)
	}
	firstReview := reviews["items"].([]any)[0].(map[string]any)
	a.write("ReviewDecide", map[string]any{
		"eventId": rev, "reviewId": itStr(t, firstReview, "reviewId"), "version": firstReview["version"],
		"decision": "APPROVE", "reason": "Accepted",
	}, http.StatusOK)
	if got := itNum(t, itGateStats(t, a, rev), "allocatedPeople"); got != 1 {
		t.Fatalf("approved review allocatedPeople = %d, want 1", got)
	}
	itTransition(t, a, rev, "OPEN", "CHECKIN_OPEN")
	secondReview := reviews["items"].([]any)[1].(map[string]any)
	a.write("ReviewDecide", map[string]any{
		"eventId": rev, "reviewId": itStr(t, secondReview, "reviewId"), "version": secondReview["version"],
		"decision": "APPROVE", "reason": "Too late",
	}, http.StatusConflict)
	if a.lastCode != "EVENT_STATE_CONFLICT" {
		t.Fatalf("review after OPEN code = %q, want canonical EVENT_STATE_CONFLICT", a.lastCode)
	}
}
