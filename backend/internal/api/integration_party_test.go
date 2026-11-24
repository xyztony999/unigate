package api

// PostgreSQL integration tests (HTTP level, frozen wire contract) for the
// self-service party lifecycle, organizer credential reissue/revoke and staff
// logout. Port of the party/credential half of WorkflowIntegrationTest
// (selfServiceChangesAndCancellationNeverSurviveTheirGuards and the
// MyPartyCancel-after-OPEN guard from onlineGroupWorkflowMaintainsCapacityFactsAndReplay),
// plus the REVIEW-mode increase-delta approval and the Logout session/cookie
// semantics of IdentityService.
//
// Run with:
//
//	set -a; . ../.local/go-test.env; set +a
//	go test ./internal/api/ -run 'TestIntegrationParty|TestIntegrationCredential|TestIntegrationLogout' -v

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

// itSelfPolicy reads the current policy and applies a partial update through
// PolicyUpdate (Java PolicyGet+PolicyUpdate). approvalMode is unwrapped from
// its {code,label} value object; a nil mutation value removes the key (used to
// clear selfServiceEndsAt back to the registration-deadline fallback).
func itSelfPolicy(t *testing.T, a *itBrowser, ev string, mut map[string]any) map[string]any {
	t.Helper()
	policy := a.request("PolicyGet", map[string]any{"eventId": ev}, http.StatusOK)
	input := map[string]any{
		"fields":                   policy["fields"],
		"approvalMode":             itStr(t, itMap(t, policy, "approvalMode"), "code"),
		"phoneVerificationEnabled": policy["phoneVerificationEnabled"],
		"selfEditEnabled":          policy["selfEditEnabled"],
		"selfCancelEnabled":        policy["selfCancelEnabled"],
		"retentionDays":            policy["retentionDays"],
		"version":                  policy["version"],
	}
	if v, has := policy["selfServiceEndsAt"]; has {
		input["selfServiceEndsAt"] = v
	}
	for k, v := range mut {
		if v == nil {
			delete(input, k)
		} else {
			input[k] = v
		}
	}
	return a.write("PolicyUpdate", map[string]any{"eventId": ev, "policy": input}, http.StatusOK)
}

// itAllocated reads Statistics.allocatedPeople for an event.
func itAllocated(t *testing.T, a *itBrowser, ev string) int64 {
	t.Helper()
	return itNum(t, a.request("Statistics", map[string]any{"eventId": ev}, http.StatusOK), "allocatedPeople")
}

// TestIntegrationPartySelfChangeCancelAndReplay ports the self-service
// lifecycle: the default-off switches, PartyExchange capability cookies
// (ug_party/ug_change), MyPartyChange with optimistic version and capacity
// guard, synchronous decrease, MyPartyCancel releasing the places and
// invalidating the ticket/change credentials, and the original-operationId
// replay (sealed result, payload-mismatch conflict) even after cancellation.
func TestIntegrationPartySelfChangeCancelAndReplay(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)
	ev := itPublishedEvent(t, e, a, 10)

	p, party, ticket := itRegisteredParty(t, e, ev, "Self Parent", 1)
	partyID := itStr(t, party, "partyId")
	if got := itNum(t, party, "approvedSize"); got != 2 {
		t.Fatalf("registration approvedSize = %d, want 2 (1+1 companion)", got)
	}
	if ticket == "" {
		t.Fatal("IMMEDIATE registration must issue a ticket token")
	}
	// PartyExchange captured both capability cookies and the party csrf.
	if p.cookies["ug_party"] == "" || p.cookies["ug_change"] == "" {
		t.Fatalf("PartyExchange must set ug_party and ug_change cookies: %+v", p.cookies)
	}

	// The default policy keeps both self-service switches off.
	state := p.request("MyParty", map[string]any{}, http.StatusOK)
	if itBool(t, state, "canChange") || itBool(t, state, "canCancel") {
		t.Fatalf("default policy must disable self-service: %+v", state)
	}
	p.write("MyPartyChange", map[string]any{"version": state["version"], "companionCount": 2}, http.StatusForbidden)
	if p.lastCode != "FORBIDDEN" {
		t.Fatalf("switch-off MyPartyChange code = %q, want canonical FORBIDDEN (SELF_SERVICE_DISABLED)", p.lastCode)
	}
	p.write("MyPartyCancel", map[string]any{"version": state["version"]}, http.StatusForbidden)
	if p.lastCode != "FORBIDDEN" {
		t.Fatalf("switch-off MyPartyCancel code = %q, want canonical FORBIDDEN (SELF_SERVICE_DISABLED)", p.lastCode)
	}

	// Enable the switches (allowed after registration: only the notified rules
	// are locked once parties exist).
	itSelfPolicy(t, a, ev, map[string]any{"selfEditEnabled": true, "selfCancelEnabled": true})
	state = p.request("MyParty", map[string]any{}, http.StatusOK)
	if !itBool(t, state, "canChange") || !itBool(t, state, "canCancel") {
		t.Fatalf("enabled policy must allow self-service: %+v", state)
	}

	// Increase 1 -> 2 companions allocates the delta immediately (IMMEDIATE).
	changeArgs := map[string]any{
		"operationId":    e.s.ID(),
		"version":        state["version"],
		"companionCount": 2,
	}
	changed := p.request("MyPartyChange", changeArgs, http.StatusOK)
	if got := itNum(t, changed, "approvedSize"); got != 3 {
		t.Fatalf("changed approvedSize = %d, want 3", got)
	}
	if got := itNum(t, changed, "version"); got != itNum(t, state, "version")+1 {
		t.Fatalf("changed version = %d, want %d", got, itNum(t, state, "version")+1)
	}
	if got := itNum(t, itMap(t, changed, "fields"), "companionCount"); got != 2 {
		t.Fatalf("changed fields.companionCount = %d, want 2", got)
	}
	if got := itAllocated(t, a, ev); got != 3 {
		t.Fatalf("allocatedPeople after increase = %d, want 3", got)
	}

	// A stale version is an optimistic-lock conflict.
	p.write("MyPartyChange", map[string]any{"version": state["version"], "companionCount": 3}, http.StatusConflict)
	if p.lastCode != "VERSION_CONFLICT" {
		t.Fatalf("stale MyPartyChange code = %q, want VERSION_CONFLICT", p.lastCode)
	}
	// Growing beyond the event capacity is refused without side effects.
	p.write("MyPartyChange", map[string]any{"version": changed["version"], "companionCount": 20}, http.StatusConflict)
	if p.lastCode != "CAPACITY_EXCEEDED" {
		t.Fatalf("over-capacity MyPartyChange code = %q, want CAPACITY_EXCEEDED", p.lastCode)
	}
	if got := itAllocated(t, a, ev); got != 3 {
		t.Fatalf("capacity conflict must not allocate: allocatedPeople = %d, want 3", got)
	}

	// The original operationId replays the sealed state without re-allocating.
	replayed := p.request("MyPartyChange", changeArgs, http.StatusOK)
	if got := itNum(t, replayed, "version"); got != itNum(t, changed, "version") {
		t.Fatalf("replay version = %d, want the sealed %d", got, itNum(t, changed, "version"))
	}
	if got := itNum(t, replayed, "approvedSize"); got != 3 {
		t.Fatalf("replay approvedSize = %d, want the sealed 3", got)
	}
	if got := itAllocated(t, a, ev); got != 3 {
		t.Fatalf("replay must not re-allocate: allocatedPeople = %d, want 3", got)
	}
	// Same operationId with different content is an idempotency conflict.
	mismatch := map[string]any{}
	for k, v := range changeArgs {
		mismatch[k] = v
	}
	mismatch["companionCount"] = 3
	p.request("MyPartyChange", mismatch, http.StatusBadRequest)
	if p.lastCode != "IDEMPOTENCY_PAYLOAD_MISMATCH" {
		t.Fatalf("changed-payload replay code = %q, want IDEMPOTENCY_PAYLOAD_MISMATCH", p.lastCode)
	}

	// Decreasing releases the delta synchronously even without review.
	decreased := p.write("MyPartyChange", map[string]any{"version": changed["version"], "companionCount": 0}, http.StatusOK)
	if got := itNum(t, decreased, "approvedSize"); got != 1 {
		t.Fatalf("decreased approvedSize = %d, want 1", got)
	}
	if got := itAllocated(t, a, ev); got != 1 {
		t.Fatalf("allocatedPeople after decrease = %d, want 1", got)
	}

	// Cancellation releases every approved place and seals the ticket.
	cancelled := p.write("MyPartyCancel", map[string]any{"version": decreased["version"]}, http.StatusOK)
	if code := itMap(t, cancelled, "status")["code"]; code != "CANCELLED" {
		t.Fatalf("cancelled status = %v, want CANCELLED", code)
	}
	if got := itNum(t, cancelled, "approvedSize"); got != 0 {
		t.Fatalf("cancelled approvedSize = %d, want 0", got)
	}
	if itBool(t, cancelled, "canChange") || itBool(t, cancelled, "canCancel") {
		t.Fatalf("cancelled party must not offer self-service: %+v", cancelled)
	}
	if got := itAllocated(t, a, ev); got != 0 {
		t.Fatalf("allocatedPeople after cancel = %d, want 0", got)
	}
	cred := p.request("MyCredential", map[string]any{}, http.StatusOK)
	if _, has := cred["ticketToken"]; has {
		t.Fatalf("cancelled party must not expose a ticket token: %+v", cred)
	}
	if code := itMap(t, cred, "status")["code"]; code != "REVOKED" {
		t.Fatalf("cancelled credential status = %v, want REVOKED", code)
	}
	// Both the ticket and the change capability are revoked at rest.
	ctx := context.Background()
	active, err := e.s.QueryInt(ctx, nil,
		"select count(*) from ug_credential where party_id=$1 and kind in ('TICKET','CHANGE') and status='ACTIVE'", partyID)
	if err != nil {
		t.Fatalf("count active credentials: %v", err)
	}
	if active != 0 {
		t.Fatalf("cancel must revoke TICKET/CHANGE credentials, %d still ACTIVE", active)
	}

	// A cancelled party can neither change nor cancel again (state guard).
	p.write("MyPartyChange", map[string]any{"version": cancelled["version"], "companionCount": 1}, http.StatusConflict)
	if p.lastCode != "EVENT_STATE_CONFLICT" {
		t.Fatalf("post-cancel MyPartyChange code = %q, want canonical EVENT_STATE_CONFLICT (PARTY_STATE_CONFLICT)", p.lastCode)
	}
	p.write("MyPartyCancel", map[string]any{"version": cancelled["version"]}, http.StatusConflict)
	if p.lastCode != "EVENT_STATE_CONFLICT" {
		t.Fatalf("post-cancel MyPartyCancel code = %q, want canonical EVENT_STATE_CONFLICT (PARTY_STATE_CONFLICT)", p.lastCode)
	}

	// Replaying the original change after cancellation still serves the sealed
	// historical state and must not resurrect the allocation.
	late := p.request("MyPartyChange", changeArgs, http.StatusOK)
	if got := itNum(t, late, "approvedSize"); got != 3 {
		t.Fatalf("late replay approvedSize = %d, want the sealed 3", got)
	}
	if got := itAllocated(t, a, ev); got != 0 {
		t.Fatalf("late replay must not re-allocate: allocatedPeople = %d, want 0", got)
	}
}

// TestIntegrationPartySelfServiceGuards asserts the switch/deadline/admission
// guards around self-service: a past selfServiceEndsAt closes the window
// (falling back to registrationEndsAt when cleared), ever_admitted blocks both
// change and cancel even while READY, and CHECKIN_OPEN freezes every ordinary
// self-service mutation (canonical EVENT_STATE_CONFLICT).
func TestIntegrationPartySelfServiceGuards(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)
	ev := itPublishedEvent(t, e, a, 10)

	r, rParty, rTicket := itRegisteredParty(t, e, ev, "Guard Parent", 0)
	s, sParty, _ := itRegisteredParty(t, e, ev, "History Parent", 0)
	rID := itStr(t, rParty, "partyId")
	sID := itStr(t, sParty, "partyId")
	itSelfPolicy(t, a, ev, map[string]any{"selfEditEnabled": true, "selfCancelEnabled": true})
	kiosk, _ := itClaimedDevice(t, e, a, ev, "KIOSK")

	// Simulate a prior admission fact at the storage layer while the event is
	// still READY, isolating the hasAdmissionHistory guard from the phase guard.
	ctx := context.Background()
	if _, err := e.s.Exec(ctx, nil, "update ug_party set ever_admitted=true where id=$1", sID); err != nil {
		t.Fatalf("mark admission history: %v", err)
	}
	sState := s.request("MyParty", map[string]any{}, http.StatusOK)
	if itBool(t, sState, "canChange") || itBool(t, sState, "canCancel") {
		t.Fatalf("admission history must disable self-service buttons: %+v", sState)
	}
	s.write("MyPartyChange", map[string]any{"version": sState["version"], "companionCount": 1}, http.StatusConflict)
	if s.lastCode != "EVENT_STATE_CONFLICT" {
		t.Fatalf("history MyPartyChange code = %q, want canonical EVENT_STATE_CONFLICT (ADMISSION_HISTORY_EXISTS)", s.lastCode)
	}
	s.write("MyPartyCancel", map[string]any{"version": sState["version"]}, http.StatusConflict)
	if s.lastCode != "EVENT_STATE_CONFLICT" {
		t.Fatalf("history MyPartyCancel code = %q, want canonical EVENT_STATE_CONFLICT (ADMISSION_HISTORY_EXISTS)", s.lastCode)
	}
	// The untouched sibling keeps its self-service rights.
	rState := r.request("MyParty", map[string]any{}, http.StatusOK)
	if !itBool(t, rState, "canChange") || !itBool(t, rState, "canCancel") {
		t.Fatalf("sibling party must keep self-service: %+v", rState)
	}

	// A past self-service deadline closes the window for everyone.
	itSelfPolicy(t, a, ev, map[string]any{"selfServiceEndsAt": itFuture(-1 * time.Hour)})
	rState = r.request("MyParty", map[string]any{}, http.StatusOK)
	if itBool(t, rState, "canChange") || itBool(t, rState, "canCancel") {
		t.Fatalf("expired deadline must disable self-service: %+v", rState)
	}
	r.write("MyPartyChange", map[string]any{"version": rState["version"], "companionCount": 1}, http.StatusForbidden)
	if r.lastCode != "FORBIDDEN" {
		t.Fatalf("deadline MyPartyChange code = %q, want canonical FORBIDDEN (SELF_SERVICE_DISABLED)", r.lastCode)
	}
	r.write("MyPartyCancel", map[string]any{"version": rState["version"]}, http.StatusForbidden)
	if r.lastCode != "FORBIDDEN" {
		t.Fatalf("deadline MyPartyCancel code = %q, want canonical FORBIDDEN (SELF_SERVICE_DISABLED)", r.lastCode)
	}
	// Clearing the deadline falls back to the (still future) registration end.
	itSelfPolicy(t, a, ev, map[string]any{"selfServiceEndsAt": nil})
	rState = r.request("MyParty", map[string]any{}, http.StatusOK)
	if !itBool(t, rState, "canChange") || !itBool(t, rState, "canCancel") {
		t.Fatalf("cleared deadline must restore self-service: %+v", rState)
	}

	// CHECKIN_OPEN freezes ordinary self-service changes and cancellations.
	itTransition(t, a, ev, "OPEN", "CHECKIN_OPEN")
	r.write("MyPartyChange", map[string]any{"version": rState["version"], "companionCount": 1}, http.StatusConflict)
	if r.lastCode != "EVENT_STATE_CONFLICT" {
		t.Fatalf("CHECKIN_OPEN MyPartyChange code = %q, want canonical EVENT_STATE_CONFLICT (EVENT_PHASE_CONFLICT)", r.lastCode)
	}
	r.write("MyPartyCancel", map[string]any{"version": rState["version"]}, http.StatusConflict)
	if r.lastCode != "EVENT_STATE_CONFLICT" {
		t.Fatalf("CHECKIN_OPEN MyPartyCancel code = %q, want canonical EVENT_STATE_CONFLICT (EVENT_PHASE_CONFLICT)", r.lastCode)
	}
	rState = r.request("MyParty", map[string]any{}, http.StatusOK)
	if itBool(t, rState, "canChange") || itBool(t, rState, "canCancel") {
		t.Fatalf("CHECKIN_OPEN must hide self-service buttons: %+v", rState)
	}

	// A real admission keeps hasAdmissionHistory true and cancel blocked.
	scan := kiosk.request("TicketResolve", map[string]any{"eventId": ev, "ticketToken": rTicket}, http.StatusOK)
	kiosk.write("GateCommit", map[string]any{
		"eventId":              ev,
		"scanContextToken":     itStr(t, scan, "scanContextToken"),
		"expectedPartyVersion": scan["partyVersion"],
		"count":                1,
	}, http.StatusOK)
	detail := a.request("PartyDetail", map[string]any{"eventId": ev, "partyId": rID}, http.StatusOK)
	if !itBool(t, detail, "hasAdmissionHistory") {
		t.Fatalf("admitted party must carry hasAdmissionHistory: %+v", detail)
	}
	r.write("MyPartyCancel", map[string]any{"version": detail["version"]}, http.StatusConflict)
	if r.lastCode != "EVENT_STATE_CONFLICT" {
		t.Fatalf("admitted MyPartyCancel code = %q, want canonical EVENT_STATE_CONFLICT", r.lastCode)
	}
}

// TestIntegrationPartyReviewIncreaseDelta asserts the REVIEW-mode self-service
// increase: only the delta goes to a pending INCREASE review (the approved size
// and allocation stay put), approval allocates exactly the delta, a decrease
// releases synchronously, and cancellation rejects the pending increase and
// releases the approved places in one transaction.
func TestIntegrationPartyReviewIncreaseDelta(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)
	ev := itPublishedEvent(t, e, a, 5)
	itSelfPolicy(t, a, ev, map[string]any{
		"approvalMode":      "REVIEW",
		"selfEditEnabled":   true,
		"selfCancelEnabled": true,
	})

	p, party, token := itRegisteredParty(t, e, ev, "Review Parent", 0)
	partyID := itStr(t, party, "partyId")
	if code := itMap(t, party, "status")["code"]; code != "PENDING" {
		t.Fatalf("REVIEW registration status = %v, want PENDING", code)
	}
	if token != "" {
		t.Fatal("pending review must not issue a ticket")
	}
	if got := itAllocated(t, a, ev); got != 0 {
		t.Fatalf("pending review must not allocate: allocatedPeople = %d", got)
	}

	// Approve the initial review: the group becomes APPROVED with a ticket.
	initial := itItems(t, a.request("ReviewList", map[string]any{
		"eventId": ev, "page": 1, "pageSize": 20, "result": "PENDING",
	}, http.StatusOK))[0].(map[string]any)
	if code := itMap(t, initial, "kind")["code"]; code != "INITIAL" {
		t.Fatalf("first review kind = %v, want INITIAL", code)
	}
	a.write("ReviewDecide", map[string]any{
		"eventId": ev, "reviewId": itStr(t, initial, "reviewId"), "version": initial["version"],
		"decision": "APPROVE", "reason": "Accepted",
	}, http.StatusOK)
	if got := itStr(t, p.request("MyCredential", map[string]any{}, http.StatusOK), "ticketToken"); got == "" {
		t.Fatal("approved initial review must issue a ticket")
	}
	state := p.request("MyParty", map[string]any{}, http.StatusOK)
	if got := itNum(t, state, "approvedSize"); got != 1 {
		t.Fatalf("approved size = %d, want 1", got)
	}
	if !itBool(t, state, "canChange") {
		t.Fatalf("REVIEW self-edit must stay enabled: %+v", state)
	}

	// Increasing 0 -> 2 companions queues only the delta for review.
	increased := p.write("MyPartyChange", map[string]any{"version": state["version"], "companionCount": 2}, http.StatusOK)
	if got := itNum(t, increased, "approvedSize"); got != 1 {
		t.Fatalf("pending increase must keep approvedSize = 1, got %d", got)
	}
	if got := itNum(t, increased, "pendingIncrease"); got != 2 {
		t.Fatalf("pendingIncrease = %d, want 2", got)
	}
	if got := itAllocated(t, a, ev); got != 1 {
		t.Fatalf("pending increase must not allocate: allocatedPeople = %d, want 1", got)
	}
	// A stale version against the bumped party is an optimistic conflict.
	p.write("MyPartyChange", map[string]any{"version": state["version"], "companionCount": 3}, http.StatusConflict)
	if p.lastCode != "VERSION_CONFLICT" {
		t.Fatalf("stale increase code = %q, want VERSION_CONFLICT", p.lastCode)
	}

	// The INCREASE review carries exactly the delta; approving allocates it.
	incReview := itItems(t, a.request("ReviewList", map[string]any{
		"eventId": ev, "page": 1, "pageSize": 20, "kind": "INCREASE", "result": "PENDING",
	}, http.StatusOK))[0].(map[string]any)
	if got := itNum(t, incReview, "requestedIncrease"); got != 2 {
		t.Fatalf("INCREASE requestedIncrease = %d, want 2", got)
	}
	if got := itStr(t, incReview, "partyId"); got != partyID {
		t.Fatalf("INCREASE review partyId = %s, want %s", got, partyID)
	}
	a.write("ReviewDecide", map[string]any{
		"eventId": ev, "reviewId": itStr(t, incReview, "reviewId"), "version": incReview["version"],
		"decision": "APPROVE", "reason": "Delta accepted",
	}, http.StatusOK)
	state = p.request("MyParty", map[string]any{}, http.StatusOK)
	if got := itNum(t, state, "approvedSize"); got != 3 {
		t.Fatalf("approved size after delta approval = %d, want 3", got)
	}
	if got := itNum(t, state, "pendingIncrease"); got != 0 {
		t.Fatalf("pendingIncrease after approval = %d, want 0", got)
	}
	if got := itAllocated(t, a, ev); got != 3 {
		t.Fatalf("allocatedPeople after delta approval = %d, want 3", got)
	}

	// A decrease releases synchronously even in REVIEW mode.
	decreased := p.write("MyPartyChange", map[string]any{"version": state["version"], "companionCount": 0}, http.StatusOK)
	if got := itNum(t, decreased, "approvedSize"); got != 1 {
		t.Fatalf("decreased approvedSize = %d, want 1", got)
	}
	if got := itNum(t, decreased, "pendingIncrease"); got != 0 {
		t.Fatalf("decrease must clear pendingIncrease, got %d", got)
	}
	if got := itAllocated(t, a, ev); got != 1 {
		t.Fatalf("allocatedPeople after decrease = %d, want 1", got)
	}

	// Queue another increase, then cancel: the pending review is rejected and
	// the approved place released atomically.
	queued := p.write("MyPartyChange", map[string]any{"version": decreased["version"], "companionCount": 1}, http.StatusOK)
	if got := itNum(t, queued, "pendingIncrease"); got != 1 {
		t.Fatalf("queued pendingIncrease = %d, want 1", got)
	}
	cancelled := p.write("MyPartyCancel", map[string]any{"version": queued["version"]}, http.StatusOK)
	if code := itMap(t, cancelled, "status")["code"]; code != "CANCELLED" {
		t.Fatalf("cancelled status = %v, want CANCELLED", code)
	}
	if got := itNum(t, cancelled, "approvedSize"); got != 0 {
		t.Fatalf("cancelled approvedSize = %d, want 0", got)
	}
	if got := itNum(t, cancelled, "pendingIncrease"); got != 0 {
		t.Fatalf("cancelled pendingIncrease = %d, want 0", got)
	}
	if got := itAllocated(t, a, ev); got != 0 {
		t.Fatalf("allocatedPeople after cancel = %d, want 0", got)
	}
	rejected := a.request("ReviewList", map[string]any{
		"eventId": ev, "page": 1, "pageSize": 20, "kind": "INCREASE", "result": "REJECTED",
	}, http.StatusOK)
	if got := itNum(t, rejected, "total"); got < 1 {
		t.Fatalf("cancel must reject the pending INCREASE review, rejected total = %d", got)
	}
	if _, has := p.request("MyCredential", map[string]any{}, http.StatusOK)["ticketToken"]; has {
		t.Fatal("cancelled party must not expose a ticket token")
	}
}

// TestIntegrationCredentialReissueRevoke ports the organizer credential
// lifecycle: reissue invalidates the old ticket and returns a credential (not
// an access bundle) unless rotateAccess is set, rotation revokes the private
// capabilities and party sessions, revoke keeps the allocation, and both
// operations are organizer-only with optimistic versions and sealed replay.
func TestIntegrationCredentialReissueRevoke(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)
	ev := itPublishedEvent(t, e, a, 10)

	// Register manually to keep the original access bundle for the rotation
	// checks (itRegisteredParty discards it).
	p := e.browser(t)
	p.bootstrap()
	pub := p.request("PublicEvent", map[string]any{"eventId": ev}, http.StatusOK)
	sub := p.write("SubmitRegistration", map[string]any{
		"eventId":               ev,
		"fields":                map[string]any{"name": "Credential Parent", "companionCount": 0},
		"acceptedNoticeVersion": itStr(t, pub, "noticeVersion"),
	}, http.StatusOK)
	party := itMap(t, sub, "party")
	partyID := itStr(t, party, "partyId")
	access := itMap(t, sub, "access")
	p.write("PartyExchange", map[string]any{
		"queryToken":  itStr(t, access, "queryToken"),
		"changeToken": itStr(t, access, "changeToken"),
	}, http.StatusOK)
	t1 := itStr(t, p.request("MyCredential", map[string]any{}, http.StatusOK), "ticketToken")

	// A granted CHECKER is not the organizer: both operations are forbidden.
	checker, checkerAcc := itNewStaff(t, e, a, "CHECKER")
	a.write("GrantSave", map[string]any{
		"eventId": ev, "accountId": itStr(t, checkerAcc, "accountId"), "role": "CHECKER",
		"allowedFields": []any{"name"}, "allowedActions": []any{"CHECKIN"},
		"expiresAt": itFuture(24 * time.Hour),
	}, http.StatusOK)
	checker.write("CredentialReissue", map[string]any{
		"eventId": ev, "partyId": partyID, "version": party["version"], "reason": "No right", "rotateAccess": false,
	}, http.StatusForbidden)
	if checker.lastCode != "FORBIDDEN" {
		t.Fatalf("checker CredentialReissue code = %q, want FORBIDDEN", checker.lastCode)
	}
	checker.write("CredentialRevoke", map[string]any{
		"eventId": ev, "partyId": partyID, "version": party["version"], "reason": "No right",
	}, http.StatusForbidden)
	if checker.lastCode != "FORBIDDEN" {
		t.Fatalf("checker CredentialRevoke code = %q, want FORBIDDEN", checker.lastCode)
	}
	// An ORGANIZER-role account without an event grant cannot even see it.
	org, _ := itNewStaff(t, e, a, "ORGANIZER")
	org.write("CredentialReissue", map[string]any{
		"eventId": ev, "partyId": partyID, "version": party["version"], "reason": "No grant", "rotateAccess": false,
	}, http.StatusNotFound)
	if org.lastCode != "RESOURCE_NOT_FOUND" {
		t.Fatalf("ungranted organizer CredentialReissue code = %q, want RESOURCE_NOT_FOUND", org.lastCode)
	}

	// Plain reissue: a fresh ticket, the old one sealed, no access bundle.
	reissueArgs := map[string]any{
		"operationId": e.s.ID(),
		"eventId":     ev, "partyId": partyID, "version": party["version"],
		"reason": "Lost page", "rotateAccess": false,
	}
	reissue := a.request("CredentialReissue", reissueArgs, http.StatusOK)
	pv := itMap(t, reissue, "party")
	if got := itNum(t, pv, "version"); got != itNum(t, party, "version")+1 {
		t.Fatalf("reissue party version = %d, want %d", got, itNum(t, party, "version")+1)
	}
	if code := itMap(t, pv, "status")["code"]; code != "APPROVED" {
		t.Fatalf("reissue must keep APPROVED status, got %v", code)
	}
	cred := itMap(t, reissue, "credential")
	if code := itMap(t, cred, "status")["code"]; code != "ACTIVE" {
		t.Fatalf("reissued credential status = %v, want ACTIVE", code)
	}
	t2 := itStr(t, cred, "ticketToken")
	if t2 == t1 {
		t.Fatal("reissue must mint a new ticket token")
	}
	if got := itNum(t, cred, "credentialVersion"); got != itNum(t, pv, "version") {
		t.Fatalf("credentialVersion = %d, want party version %d", got, itNum(t, pv, "version"))
	}
	if _, has := reissue["access"]; has {
		t.Fatalf("plain reissue must not return an access bundle: %+v", reissue)
	}
	if itBool(t, reissue, "replayed") {
		t.Fatal("first reissue must not be flagged replayed")
	}
	if got := itAllocated(t, a, ev); got != 1 {
		t.Fatalf("reissue must not add places: allocatedPeople = %d, want 1", got)
	}
	// The party browser sees the new ticket immediately.
	if got := itStr(t, p.request("MyCredential", map[string]any{}, http.StatusOK), "ticketToken"); got != t2 {
		t.Fatalf("MyCredential after reissue = %q, want the new ticket", got)
	}

	// A stale version is an optimistic conflict.
	a.write("CredentialReissue", map[string]any{
		"eventId": ev, "partyId": partyID, "version": party["version"], "reason": "Stale", "rotateAccess": false,
	}, http.StatusConflict)
	if a.lastCode != "VERSION_CONFLICT" {
		t.Fatalf("stale CredentialReissue code = %q, want VERSION_CONFLICT", a.lastCode)
	}
	// The original operationId replays the sealed credential.
	rep := a.request("CredentialReissue", reissueArgs, http.StatusOK)
	if !itBool(t, rep, "replayed") {
		t.Fatal("replayed reissue must flag replayed")
	}
	if got := itStr(t, itMap(t, rep, "credential"), "ticketToken"); got != t2 {
		t.Fatalf("replay ticket = %q, want the sealed %q", got, t2)
	}
	if got := itNum(t, itMap(t, rep, "party"), "version"); got != itNum(t, pv, "version") {
		t.Fatalf("replay party version = %d, want the sealed %d", got, itNum(t, pv, "version"))
	}

	// rotation: new access bundle, old capabilities and party sessions dead.
	rotArgs := map[string]any{
		"operationId": e.s.ID(),
		"eventId":     ev, "partyId": partyID, "version": pv["version"],
		"reason": "Rotate access", "rotateAccess": true,
	}
	rot := a.request("CredentialReissue", rotArgs, http.StatusOK)
	pv2 := itMap(t, rot, "party")
	if got := itNum(t, pv2, "version"); got != itNum(t, pv, "version")+1 {
		t.Fatalf("rotation party version = %d, want %d", got, itNum(t, pv, "version")+1)
	}
	t3 := itStr(t, itMap(t, rot, "credential"), "ticketToken")
	if t3 == t2 {
		t.Fatal("rotation must mint yet another ticket token")
	}
	access2 := itMap(t, rot, "access")
	itStr(t, access2, "queryToken")
	itStr(t, access2, "changeToken")
	// The old party sessions are revoked server-side.
	p.request("MyParty", map[string]any{}, http.StatusUnauthorized)
	if p.lastCode != "SESSION_EXPIRED" {
		t.Fatalf("rotated party session code = %q, want SESSION_EXPIRED", p.lastCode)
	}
	// The old query token can no longer be exchanged (Java asserts 404).
	old := e.browser(t)
	old.bootstrap()
	old.write("PartyExchange", map[string]any{"queryToken": itStr(t, access, "queryToken")}, http.StatusNotFound)
	if old.lastCode != "RESOURCE_NOT_FOUND" {
		t.Fatalf("revoked queryToken exchange code = %q, want RESOURCE_NOT_FOUND", old.lastCode)
	}
	// The rotated bundle exchanges cleanly in a fresh browser.
	nb := e.browser(t)
	nb.bootstrap()
	nb.write("PartyExchange", map[string]any{
		"queryToken":  itStr(t, access2, "queryToken"),
		"changeToken": itStr(t, access2, "changeToken"),
	}, http.StatusOK)
	if nb.cookies["ug_party"] == "" || nb.cookies["ug_change"] == "" {
		t.Fatalf("rotated exchange must set capability cookies: %+v", nb.cookies)
	}
	if got := itStr(t, nb.request("MyCredential", map[string]any{}, http.StatusOK), "ticketToken"); got != t3 {
		t.Fatalf("rotated MyCredential = %q, want the newest ticket", got)
	}

	// At the gate the old tickets are dead and only the newest resolves.
	itTransition(t, a, ev, "OPEN", "CHECKIN_OPEN")
	kiosk, _ := itClaimedDevice(t, e, a, ev, "KIOSK")
	kiosk.request("TicketResolve", map[string]any{"eventId": ev, "ticketToken": t1}, http.StatusNotFound)
	if kiosk.lastCode != "RESOURCE_NOT_FOUND" {
		t.Fatalf("original ticket after reissue code = %q, want RESOURCE_NOT_FOUND", kiosk.lastCode)
	}
	kiosk.request("TicketResolve", map[string]any{"eventId": ev, "ticketToken": t2}, http.StatusNotFound)
	if kiosk.lastCode != "RESOURCE_NOT_FOUND" {
		t.Fatalf("second ticket after rotation code = %q, want RESOURCE_NOT_FOUND", kiosk.lastCode)
	}
	scan := kiosk.request("TicketResolve", map[string]any{"eventId": ev, "ticketToken": t3}, http.StatusOK)
	if got := itNum(t, scan, "approvedSize"); got != 1 {
		t.Fatalf("newest ticket approvedSize = %d, want 1", got)
	}

	// Revoke seals the ticket but keeps the qualification and the allocation.
	revokeArgs := map[string]any{
		"operationId": e.s.ID(),
		"eventId":     ev, "partyId": partyID, "version": pv2["version"], "reason": "Lost ticket",
	}
	revoked := a.request("CredentialRevoke", revokeArgs, http.StatusOK)
	if code := itMap(t, revoked, "status")["code"]; code != "APPROVED" {
		t.Fatalf("revoke must keep APPROVED status, got %v", code)
	}
	if got := itNum(t, revoked, "approvedSize"); got != 1 {
		t.Fatalf("revoke must not release places: approvedSize = %d, want 1", got)
	}
	if got := itNum(t, revoked, "version"); got != itNum(t, pv2, "version")+1 {
		t.Fatalf("revoke version = %d, want %d", got, itNum(t, pv2, "version")+1)
	}
	if got := itAllocated(t, a, ev); got != 1 {
		t.Fatalf("revoke must not release the allocation: allocatedPeople = %d, want 1", got)
	}
	kiosk.request("TicketResolve", map[string]any{"eventId": ev, "ticketToken": t3}, http.StatusNotFound)
	if kiosk.lastCode != "RESOURCE_NOT_FOUND" {
		t.Fatalf("revoked ticket resolve code = %q, want RESOURCE_NOT_FOUND", kiosk.lastCode)
	}
	after := nb.request("MyCredential", map[string]any{}, http.StatusOK)
	if _, has := after["ticketToken"]; has {
		t.Fatalf("revoked party must not expose a ticket token: %+v", after)
	}
	if code := itMap(t, after, "status")["code"]; code != "REVOKED" {
		t.Fatalf("revoked credential status = %v, want REVOKED", code)
	}
	// A stale revoke version is an optimistic conflict.
	a.write("CredentialRevoke", map[string]any{
		"eventId": ev, "partyId": partyID, "version": party["version"], "reason": "Stale",
	}, http.StatusConflict)
	if a.lastCode != "VERSION_CONFLICT" {
		t.Fatalf("stale CredentialRevoke code = %q, want VERSION_CONFLICT", a.lastCode)
	}
	// The original revoke replays its sealed party view without side effects.
	revRep := a.request("CredentialRevoke", revokeArgs, http.StatusOK)
	if got := itNum(t, revRep, "version"); got != itNum(t, revoked, "version") {
		t.Fatalf("revoke replay version = %d, want the sealed %d", got, itNum(t, revoked, "version"))
	}
	if got := itAllocated(t, a, ev); got != 1 {
		t.Fatalf("revoke replay must not change the allocation: %d", got)
	}
}

// TestIntegrationLogout asserts the staff logout semantics: the session is
// revoked server-side (the old cookie dies in any browser), the ug_staff
// cookie is cleared with an empty value, re-login is clean, the original
// operationId replays the sealed empty result (restoring the clearing cookie
// without revoking the new session), and Logout stays reachable during the
// forced initial-password window.
func TestIntegrationLogout(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)

	b, acc := itNewStaff(t, e, a, "CHECKER")
	b.request("SessionMe", map[string]any{}, http.StatusOK)
	oldToken := b.cookies["ug_staff"]
	if oldToken == "" {
		t.Fatal("login must set the ug_staff cookie")
	}

	logoutArgs := map[string]any{"operationId": e.s.ID()}
	out := b.request("Logout", logoutArgs, http.StatusOK)
	if len(out) != 0 {
		t.Fatalf("Logout data must be empty, got %+v", out)
	}
	// The clearing cookie was captured (empty value).
	if v, has := b.cookies["ug_staff"]; !has || v != "" {
		t.Fatalf("Logout must clear ug_staff to an empty value, got %q (present=%v)", v, has)
	}
	// The old session is revoked server-side, not just client-side.
	b.request("SessionMe", map[string]any{}, http.StatusUnauthorized)
	if b.lastCode != "SESSION_EXPIRED" {
		t.Fatalf("post-logout SessionMe code = %q, want SESSION_EXPIRED", b.lastCode)
	}
	ob := e.browser(t)
	ob.cookies["ug_staff"] = oldToken
	ob.request("SessionMe", map[string]any{}, http.StatusUnauthorized)
	if ob.lastCode != "SESSION_EXPIRED" {
		t.Fatalf("old-token SessionMe code = %q, want SESSION_EXPIRED", ob.lastCode)
	}

	// Re-login is clean and issues a fresh session.
	b.bootstrap()
	b.write("Login", map[string]any{"username": acc["username"], "password": "Changed-staff-123!"}, http.StatusOK)
	b.request("SessionMe", map[string]any{}, http.StatusOK)
	newToken := b.cookies["ug_staff"]
	if newToken == "" || newToken == oldToken {
		t.Fatalf("re-login must mint a fresh ug_staff token")
	}

	// Replaying the original logout serves the sealed empty result and
	// re-emits the stored clearing cookie, but does not revoke the new session.
	rep := b.request("Logout", logoutArgs, http.StatusOK)
	if len(rep) != 0 {
		t.Fatalf("replayed Logout data must be empty, got %+v", rep)
	}
	if v := b.cookies["ug_staff"]; v != "" {
		t.Fatalf("replay must restore the clearing cookie, got %q", v)
	}
	keep := e.browser(t)
	keep.cookies["ug_staff"] = newToken
	keep.request("SessionMe", map[string]any{}, http.StatusOK)

	// The raw Set-Cookie carries the frozen clearing attributes.
	b2, _ := itNewStaff(t, e, a, "READER")
	op := e.c.Operations["Logout"]
	body, err := json.Marshal(map[string]any{"operationId": e.s.ID()})
	if err != nil {
		t.Fatalf("marshal logout: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, e.server.URL+op.Path, strings.NewReader(string(body)))
	if err != nil {
		t.Fatalf("build logout request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", e.cfg.AllowedOrigin)
	req.Header.Set("X-CSRF-Token", b2.csrf)
	req.AddCookie(&http.Cookie{Name: "ug_staff", Value: b2.cookies["ug_staff"]})
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		t.Fatalf("raw logout: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("raw logout status = %d, want 200", resp.StatusCode)
	}
	var cleared *http.Cookie
	for _, ck := range resp.Cookies() {
		if ck.Name == "ug_staff" {
			cleared = ck
		}
	}
	if cleared == nil {
		t.Fatal("Logout must send a ug_staff Set-Cookie")
	}
	if cleared.Value != "" || cleared.Path != "/" || !cleared.HttpOnly {
		t.Fatalf("clearing cookie = %+v, want empty value, Path=/, HttpOnly", cleared)
	}

	// Logout is exempt from the forced initial-password block.
	temp := a.write("AccountCreate", map[string]any{
		"username": "logout-temp-" + e.s.ID(), "displayName": "Logout temp",
		"temporaryPassword": "Temporary-staff-123!", "roles": []any{"CHECKER"},
	}, http.StatusOK)
	b3 := e.browser(t)
	b3.bootstrap()
	login := b3.write("Login", map[string]any{"username": temp["username"], "password": "Temporary-staff-123!"}, http.StatusOK)
	if !itBool(t, login, "mustChangePassword") {
		t.Fatalf("temporary login must force a change: %+v", login)
	}
	b3.write("Logout", map[string]any{}, http.StatusOK)
	b3.request("SessionMe", map[string]any{}, http.StatusUnauthorized)
	if b3.lastCode != "SESSION_EXPIRED" {
		t.Fatalf("forced-window logout SessionMe code = %q, want SESSION_EXPIRED", b3.lastCode)
	}
}