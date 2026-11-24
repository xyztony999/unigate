package api

// Port of the core WorkflowIntegrationTest flows (HTTP level): session
// bootstrap -> login -> forced password change revoking sessions -> re-login,
// and the public registration workflow (create event -> publish -> submit ->
// replay/RegistrationResult -> PartyExchange -> MyParty/MyCredential).

import (
	"net/http"
	"testing"
	"time"
)

// TestIntegrationSessionPasswordChangeRevokesSessions ports the
// admin()/readyAdmin() lifecycle: SessionBootstrap, Login with the initial
// password (mustChangePassword=true), PasswordChange revokes the live session
// (old session 401), re-bootstrap and login with the new password.
func TestIntegrationSessionPasswordChangeRevokesSessions(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)

	a := e.browser(t)
	a.bootstrap()
	if a.csrf == "" || a.context == "" {
		t.Fatalf("bootstrap must issue csrfToken and clientContextId: %+v", a.cookies)
	}
	if _, ok := a.cookies["ug_anon"]; !ok {
		t.Fatalf("bootstrap must set the ug_anon cookie: %+v", a.cookies)
	}

	login := a.write("Login", map[string]any{"username": itAdminUsername, "password": itAdminPassword}, http.StatusOK)
	if !itBool(t, login, "mustChangePassword") {
		t.Fatalf("initial login must force a password change: %+v", login)
	}
	if itStr(t, login, "username") != itAdminUsername {
		t.Fatalf("login username = %v", login["username"])
	}
	if _, ok := a.cookies["ug_staff"]; !ok {
		t.Fatalf("login must set the ug_staff cookie: %+v", a.cookies)
	}
	me := a.request("SessionMe", map[string]any{}, http.StatusOK)
	if itStr(t, me, "username") != itAdminUsername {
		t.Fatalf("SessionMe username = %v", me["username"])
	}

	a.write("PasswordChange", map[string]any{"oldPassword": itAdminPassword, "newPassword": itChangedAdminPassword}, http.StatusOK)
	// The changed password revokes every session of the account.
	a.request("SessionMe", map[string]any{}, http.StatusUnauthorized)
	if a.lastCode != "SESSION_EXPIRED" {
		t.Fatalf("revoked session code = %q, want SESSION_EXPIRED", a.lastCode)
	}
	// The old password no longer authenticates.
	stale := e.browser(t)
	stale.bootstrap()
	stale.write("Login", map[string]any{"username": itAdminUsername, "password": itAdminPassword}, http.StatusUnauthorized)

	a.bootstrap()
	relogin := a.write("Login", map[string]any{"username": itAdminUsername, "password": itChangedAdminPassword}, http.StatusOK)
	if itBool(t, relogin, "mustChangePassword") {
		t.Fatalf("password change must clear mustChangePassword: %+v", relogin)
	}
	a.request("SessionMe", map[string]any{}, http.StatusOK)
}

// TestIntegrationPublicRegistrationAndCredential ports the front half of
// onlineGroupWorkflowMaintainsCapacityFactsAndReplay: admin creates and
// publishes an event, an anonymous browser submits a registration (with
// idempotent replay), queries RegistrationResult (own context only), exchanges
// the access tokens and reads MyParty / MyCredential.
func TestIntegrationPublicRegistrationAndCredential(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)

	now := time.Now().UTC()
	created := a.write("EventCreate", map[string]any{
		"title":                "Integration " + e.s.ID(),
		"startsAt":             now.Add(2 * time.Hour).Format(time.RFC3339Nano),
		"endsAt":               now.Add(5 * time.Hour).Format(time.RFC3339Nano),
		"timeZone":             "Asia/Shanghai",
		"registrationStartsAt": now.Add(-1 * time.Hour).Format(time.RFC3339Nano),
		"registrationEndsAt":   now.Add(1 * time.Hour).Format(time.RFC3339Nano),
		"capacity":             3,
	}, http.StatusOK)
	ev := itStr(t, created, "eventId")
	published := a.write("EventTransition", map[string]any{
		"eventId": ev, "version": created["version"], "action": "PUBLISH",
	}, http.StatusOK)
	if code := itMap(t, published, "status")["code"]; code != "READY" {
		t.Fatalf("published event status = %v, want READY", code)
	}

	p := e.browser(t)
	p.bootstrap()
	pub := p.request("PublicEvent", map[string]any{"eventId": ev}, http.StatusOK)
	noticeVersion := itStr(t, pub, "noticeVersion")
	if !itBool(t, pub, "registrationAvailable") {
		t.Fatalf("published event must accept registrations: %+v", pub)
	}

	registration := map[string]any{
		"operationId":           e.s.ID(),
		"eventId":               ev,
		"fields":                map[string]any{"name": "Sample Parent", "companionCount": 2},
		"acceptedNoticeVersion": noticeVersion,
	}
	submitted := p.request("SubmitRegistration", registration, http.StatusOK)
	party := itMap(t, submitted, "party")
	partyID := itStr(t, party, "partyId")
	access := itMap(t, submitted, "access")
	queryToken := itStr(t, access, "queryToken")
	changeToken := itStr(t, access, "changeToken")
	if itBool(t, submitted, "replayed") {
		t.Fatal("first submission must not be a replay")
	}

	// Same operationId + payload replays the sealed result.
	replay := p.request("SubmitRegistration", registration, http.StatusOK)
	if !itBool(t, replay, "replayed") {
		t.Fatalf("replayed submission must flag replayed: %+v", replay)
	}
	if got := itStr(t, itMap(t, replay, "party"), "partyId"); got != partyID {
		t.Fatalf("replay partyId = %s, want %s", got, partyID)
	}

	// RegistrationResult is bound to the submitting anonymous context.
	result := p.request("RegistrationResult", map[string]any{"eventId": ev, "operationId": registration["operationId"]}, http.StatusOK)
	if !itBool(t, result, "replayed") {
		t.Fatalf("RegistrationResult must flag replayed: %+v", result)
	}
	if got := itStr(t, itMap(t, result, "party"), "partyId"); got != partyID {
		t.Fatalf("RegistrationResult partyId = %s, want %s", got, partyID)
	}
	stranger := e.browser(t)
	stranger.bootstrap()
	stranger.request("RegistrationResult", map[string]any{"eventId": ev, "operationId": registration["operationId"]}, http.StatusNotFound)
	if stranger.lastCode != "RESOURCE_NOT_FOUND" {
		t.Fatalf("stranger RegistrationResult code = %q, want RESOURCE_NOT_FOUND", stranger.lastCode)
	}

	// Exchange the access bundle for the private party session, then read the
	// party state and the admission credential.
	p.write("PartyExchange", map[string]any{"queryToken": queryToken, "changeToken": changeToken}, http.StatusOK)
	if _, ok := p.cookies["ug_party"]; !ok {
		t.Fatalf("PartyExchange must set the ug_party cookie: %+v", p.cookies)
	}
	myParty := p.request("MyParty", map[string]any{}, http.StatusOK)
	if got := itStr(t, myParty, "partyId"); got != partyID {
		t.Fatalf("MyParty partyId = %s, want %s", got, partyID)
	}
	if got := itStr(t, itMap(t, myParty, "fields"), "name"); got != "Sample Parent" {
		t.Fatalf("MyParty fields.name = %q", got)
	}
	if got := itNum(t, myParty, "approvedSize"); got != 3 {
		t.Fatalf("MyParty approvedSize = %d, want 3 (1 applicant + 2 companions)", got)
	}
	cred := p.request("MyCredential", map[string]any{}, http.StatusOK)
	if got := itStr(t, cred, "partyId"); got != partyID {
		t.Fatalf("MyCredential partyId = %s, want %s", got, partyID)
	}
	itStr(t, cred, "ticketToken")

	// The event is full: a second anonymous group is rejected with 409.
	extra := map[string]any{
		"eventId":               ev,
		"fields":                map[string]any{},
		"acceptedNoticeVersion": noticeVersion,
	}
	stranger.write("SubmitRegistration", extra, http.StatusConflict)
	// Frozen wire mapping: internal CAPACITY_CONFLICT publishes CAPACITY_EXCEEDED.
	if stranger.lastCode != "CAPACITY_EXCEEDED" {
		t.Fatalf("full-event registration code = %q, want CAPACITY_EXCEEDED", stranger.lastCode)
	}
}
