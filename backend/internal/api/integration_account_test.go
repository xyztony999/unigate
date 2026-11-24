package api

// PostgreSQL integration tests (HTTP level, frozen wire contract) for account
// administration and event-grant authorization. Port of the account/grant half
// of WorkflowIntegrationTest (schoolReaderFieldsActionsAndRevocationAreEnforcedInQueries
// and the newAccount()/staff() lifecycle), plus the optimistic-version,
// last-admin / last-organizer, organizer-scope and grant-expiry guards.
//
// Run with:
//
//	set -a; . ../.local/go-test.env; set +a
//	go test ./internal/api/ -run 'TestIntegrationAccount|TestIntegrationGrant' -v

import (
	"bytes"
	"context"
	"encoding/base64"
	"net/http"
	"testing"
	"time"
)

// itCodes extracts the code strings from an array of {code,label} enum values.
func itCodes(t *testing.T, m map[string]any, key string) []string {
	t.Helper()
	raw, ok := m[key].([]any)
	if !ok {
		t.Fatalf("field %q: want array, got %v (%T)", key, m[key], m[key])
	}
	out := make([]string, 0, len(raw))
	for _, x := range raw {
		out = append(out, itStr(t, x.(map[string]any), "code"))
	}
	return out
}

// itItems extracts the items array of a page/selector envelope.
func itItems(t *testing.T, m map[string]any) []any {
	t.Helper()
	raw, ok := m["items"].([]any)
	if !ok {
		t.Fatalf("items: want array, got %v (%T)", m["items"], m["items"])
	}
	return raw
}

// itHasCode reports whether a code slice contains want.
func itHasCode(codes []string, want string) bool {
	for _, c := range codes {
		if c == want {
			return true
		}
	}
	return false
}

// itFuture returns an RFC3339Nano timestamp d hours from now (UTC).
func itFuture(d time.Duration) string {
	return time.Now().UTC().Add(d).Format(time.RFC3339Nano)
}

// TestIntegrationAccountLifecycleAndLastAdminGuard covers AccountCreate (worker
// with temporary password + roles), AccountList paging/query, AccountUpdate
// (displayName/roles/enabled with optimistic version), the disabled-account
// session cutoff, and the rule that the last enabled ADMIN can be neither
// disabled nor demoted (409 canonical EVENT_STATE_CONFLICT).
func TestIntegrationAccountLifecycleAndLastAdminGuard(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)

	// Create three workers with distinct roles; each starts enabled at version 0.
	created := []map[string]any{}
	for _, role := range []string{"ORGANIZER", "CHECKER", "READER"} {
		acc := a.write("AccountCreate", map[string]any{
			"username":          "worker-" + role + "-" + e.s.ID(),
			"displayName":       "Worker " + role,
			"temporaryPassword": "Temporary-staff-123!",
			"roles":             []any{role},
		}, http.StatusOK)
		itStr(t, acc, "accountId")
		if !itBool(t, acc, "enabled") {
			t.Fatalf("new account must be enabled: %+v", acc)
		}
		if got := itNum(t, acc, "version"); got != 0 {
			t.Fatalf("new account version = %d, want 0", got)
		}
		if codes := itCodes(t, acc, "roles"); !itHasCode(codes, role) || len(codes) != 1 {
			t.Fatalf("new account roles = %v, want [%s]", codes, role)
		}
		created = append(created, acc)
	}

	// AccountList pages over admin + 3 workers (total 4), stable username order.
	page1 := a.request("AccountList", map[string]any{"page": 1, "pageSize": 2}, http.StatusOK)
	if got := itNum(t, page1, "total"); got != 4 {
		t.Fatalf("AccountList total = %d, want 4", got)
	}
	if got := itNum(t, page1, "page"); got != 1 {
		t.Fatalf("AccountList page = %d, want 1", got)
	}
	if got := itNum(t, page1, "pageSize"); got != 2 {
		t.Fatalf("AccountList pageSize = %d, want 2", got)
	}
	if got := len(itItems(t, page1)); got != 2 {
		t.Fatalf("AccountList page1 items = %d, want 2", got)
	}
	page2 := a.request("AccountList", map[string]any{"page": 2, "pageSize": 2}, http.StatusOK)
	if got := len(itItems(t, page2)); got != 2 {
		t.Fatalf("AccountList page2 items = %d, want 2", got)
	}
	// Pages must not overlap.
	seen := map[string]bool{}
	for _, p := range []map[string]any{page1, page2} {
		for _, item := range itItems(t, p) {
			id := itStr(t, item.(map[string]any), "accountId")
			if seen[id] {
				t.Fatalf("AccountList pages overlap on %s", id)
			}
			seen[id] = true
		}
	}

	// Query narrows by username substring.
	checker := created[1]
	q := a.request("AccountList", map[string]any{"query": itStr(t, checker, "username")}, http.StatusOK)
	if got := itNum(t, q, "total"); got != 1 {
		t.Fatalf("AccountList query total = %d, want 1", got)
	}
	if got := itStr(t, itItems(t, q)[0].(map[string]any), "accountId"); got != itStr(t, checker, "accountId") {
		t.Fatalf("AccountList query returned wrong account: %s", got)
	}

	// AccountUpdate changes displayName/roles/enabled and bumps the version.
	reader := created[2]
	readerID := itStr(t, reader, "accountId")
	updated := a.write("AccountUpdate", map[string]any{
		"accountId":   readerID,
		"version":     reader["version"],
		"displayName": "Renamed reader",
		"enabled":     true,
		"roles":       []any{"READER", "CHECKER"},
	}, http.StatusOK)
	if got := itStr(t, updated, "displayName"); got != "Renamed reader" {
		t.Fatalf("updated displayName = %q, want Renamed reader", got)
	}
	if got := itNum(t, updated, "version"); got != 1 {
		t.Fatalf("updated version = %d, want 1", got)
	}
	if codes := itCodes(t, updated, "roles"); !itHasCode(codes, "READER") || !itHasCode(codes, "CHECKER") {
		t.Fatalf("updated roles = %v, want READER+CHECKER", codes)
	}

	// A stale version is an optimistic-lock conflict (canonical VERSION_CONFLICT).
	a.write("AccountUpdate", map[string]any{
		"accountId": readerID, "version": 0, "displayName": "Stale", "enabled": true, "roles": []any{"READER"},
	}, http.StatusConflict)
	if a.lastCode != "VERSION_CONFLICT" {
		t.Fatalf("stale AccountUpdate code = %q, want VERSION_CONFLICT", a.lastCode)
	}

	// Log the reader in, then disable it: the live session is cut off (401).
	rb := e.browser(t)
	rb.bootstrap()
	rb.write("Login", map[string]any{"username": itStr(t, reader, "username"), "password": "Temporary-staff-123!"}, http.StatusOK)
	rb.write("PasswordChange", map[string]any{"oldPassword": "Temporary-staff-123!", "newPassword": "Changed-staff-123!"}, http.StatusOK)
	rb.bootstrap()
	rb.write("Login", map[string]any{"username": itStr(t, reader, "username"), "password": "Changed-staff-123!"}, http.StatusOK)
	rb.request("SessionMe", map[string]any{}, http.StatusOK)
	current := a.request("AccountList", map[string]any{"query": itStr(t, reader, "username")}, http.StatusOK)
	curVersion := itItems(t, current)[0].(map[string]any)["version"]
	a.write("AccountUpdate", map[string]any{
		"accountId": readerID, "version": curVersion, "displayName": "Disabled reader", "enabled": false, "roles": []any{"READER", "CHECKER"},
	}, http.StatusOK)
	rb.request("SessionMe", map[string]any{}, http.StatusUnauthorized)
	if rb.lastCode != "SESSION_EXPIRED" {
		t.Fatalf("disabled account SessionMe code = %q, want SESSION_EXPIRED", rb.lastCode)
	}

	// The bootstrap admin is the only enabled ADMIN: disabling it is refused.
	self := a.request("SessionMe", map[string]any{}, http.StatusOK)
	adminID := itStr(t, self, "accountId")
	adminRow := itItems(t, a.request("AccountList", map[string]any{"query": itAdminUsername}, http.StatusOK))[0].(map[string]any)
	a.write("AccountUpdate", map[string]any{
		"accountId": adminID, "version": adminRow["version"], "displayName": "Last admin", "enabled": false, "roles": []any{"ADMIN", "ORGANIZER"},
	}, http.StatusConflict)
	if a.lastCode != "EVENT_STATE_CONFLICT" {
		t.Fatalf("disable last admin code = %q, want canonical EVENT_STATE_CONFLICT (LAST_ADMIN)", a.lastCode)
	}
	// Demoting the last admin (enabled but no ADMIN role) is refused too.
	a.write("AccountUpdate", map[string]any{
		"accountId": adminID, "version": adminRow["version"], "displayName": "Demoted admin", "enabled": true, "roles": []any{"ORGANIZER"},
	}, http.StatusConflict)
	if a.lastCode != "EVENT_STATE_CONFLICT" {
		t.Fatalf("demote last admin code = %q, want canonical EVENT_STATE_CONFLICT (LAST_ADMIN)", a.lastCode)
	}

	// With a second enabled ADMIN the guard lifts and the bootstrap admin can be disabled.
	a.write("AccountCreate", map[string]any{
		"username": "second-admin-" + e.s.ID(), "displayName": "Second admin",
		"temporaryPassword": "Temporary-staff-123!", "roles": []any{"ADMIN"},
	}, http.StatusOK)
	adminRow = itItems(t, a.request("AccountList", map[string]any{"query": itAdminUsername}, http.StatusOK))[0].(map[string]any)
	a.write("AccountUpdate", map[string]any{
		"accountId": adminID, "version": adminRow["version"], "displayName": "Retired admin", "enabled": false, "roles": []any{"ADMIN", "ORGANIZER"},
	}, http.StatusOK)
	a.request("SessionMe", map[string]any{}, http.StatusUnauthorized)

	// A non-admin cannot call the account administration operations. The CHECKER
	// worker created above still holds its temporary password.
	cb := e.browser(t)
	cb.bootstrap()
	cb.write("Login", map[string]any{"username": itStr(t, checker, "username"), "password": "Temporary-staff-123!"}, http.StatusOK)
	cb.write("PasswordChange", map[string]any{"oldPassword": "Temporary-staff-123!", "newPassword": "Changed-staff-123!"}, http.StatusOK)
	cb.bootstrap()
	cb.write("Login", map[string]any{"username": itStr(t, checker, "username"), "password": "Changed-staff-123!"}, http.StatusOK)
	cb.request("AccountList", map[string]any{}, http.StatusForbidden)
	if cb.lastCode != "FORBIDDEN" {
		t.Fatalf("non-admin AccountList code = %q, want FORBIDDEN", cb.lastCode)
	}
}

// TestIntegrationAccountForcedPasswordChange asserts the initial-password
// lifecycle: a new worker logs in with mustChangePassword, every protected
// operation is blocked (403 canonical FORBIDDEN) until the change, SessionMe
// stays reachable, the change revokes the old session, and re-login is clean.
func TestIntegrationAccountForcedPasswordChange(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)

	acc := a.write("AccountCreate", map[string]any{
		"username": "forced-" + e.s.ID(), "displayName": "Forced change",
		"temporaryPassword": "Temporary-staff-123!", "roles": []any{"CHECKER"},
	}, http.StatusOK)
	username := itStr(t, acc, "username")

	b := e.browser(t)
	b.bootstrap()
	login := b.write("Login", map[string]any{"username": username, "password": "Temporary-staff-123!"}, http.StatusOK)
	if !itBool(t, login, "mustChangePassword") {
		t.Fatalf("initial login must force a password change: %+v", login)
	}
	// SessionMe is allowed during the forced-change window.
	b.request("SessionMe", map[string]any{}, http.StatusOK)
	// A protected operation is blocked with PASSWORD_CHANGE_REQUIRED -> FORBIDDEN.
	b.request("MyGrantList", map[string]any{}, http.StatusForbidden)
	if b.lastCode != "FORBIDDEN" {
		t.Fatalf("forced-change MyGrantList code = %q, want canonical FORBIDDEN (PASSWORD_CHANGE_REQUIRED)", b.lastCode)
	}
	// Wrong old password is rejected (canonical UNAUTHENTICATED).
	b.write("PasswordChange", map[string]any{"oldPassword": "Wrong-password-123!", "newPassword": "Changed-staff-123!"}, http.StatusUnauthorized)
	if b.lastCode != "UNAUTHENTICATED" {
		t.Fatalf("wrong old password code = %q, want UNAUTHENTICATED", b.lastCode)
	}
	// The correct change succeeds and revokes the current session.
	b.write("PasswordChange", map[string]any{"oldPassword": "Temporary-staff-123!", "newPassword": "Changed-staff-123!"}, http.StatusOK)
	b.request("SessionMe", map[string]any{}, http.StatusUnauthorized)
	if b.lastCode != "SESSION_EXPIRED" {
		t.Fatalf("post-change SessionMe code = %q, want SESSION_EXPIRED", b.lastCode)
	}
	// Re-login with the new password no longer forces a change.
	b.bootstrap()
	relogin := b.write("Login", map[string]any{"username": username, "password": "Changed-staff-123!"}, http.StatusOK)
	if itBool(t, relogin, "mustChangePassword") {
		t.Fatalf("re-login must not force a change: %+v", relogin)
	}
	b.request("MyGrantList", map[string]any{}, http.StatusOK)
}

// TestIntegrationGrantEventOptionsSelector asserts the ADMIN-only thin event
// selector: items carry only eventId/title/startsAt (no roster/capacity), query
// narrows by title, cursor offsets the page, and an invalid cursor is rejected.
func TestIntegrationGrantEventOptionsSelector(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)

	titles := []string{"Alpha Gala", "Beta Gala", "Gamma Gala"}
	for _, title := range titles {
		now := time.Now().UTC()
		a.write("EventCreate", map[string]any{
			"title": title, "startsAt": now.Add(2 * time.Hour).Format(time.RFC3339Nano),
			"endsAt": now.Add(5 * time.Hour).Format(time.RFC3339Nano), "timeZone": "Asia/Shanghai",
			"registrationStartsAt": now.Add(-1 * time.Hour).Format(time.RFC3339Nano),
			"registrationEndsAt":   now.Add(1 * time.Hour).Format(time.RFC3339Nano),
			"capacity":             10,
		}, http.StatusOK)
	}

	all := a.request("GrantEventOptions", map[string]any{}, http.StatusOK)
	items := itItems(t, all)
	if len(items) != 3 {
		t.Fatalf("GrantEventOptions items = %d, want 3", len(items))
	}
	for _, item := range items {
		m := item.(map[string]any)
		if len(m) != 3 {
			t.Fatalf("selector item must carry only eventId/title/startsAt, got %d keys: %+v", len(m), m)
		}
		itStr(t, m, "eventId")
		itStr(t, m, "title")
		itStr(t, m, "startsAt")
	}
	if _, has := all["nextCursor"]; has {
		t.Fatalf("selector with <=100 rows must not emit nextCursor: %+v", all)
	}

	// Query narrows by title substring.
	filtered := a.request("GrantEventOptions", map[string]any{"query": "Beta"}, http.StatusOK)
	fi := itItems(t, filtered)
	if len(fi) != 1 || itStr(t, fi[0].(map[string]any), "title") != "Beta Gala" {
		t.Fatalf("GrantEventOptions query=Beta items = %+v, want only Beta Gala", fi)
	}

	// A valid cursor offsets the slice (base64 raw-url of the start index).
	cursor := base64.RawURLEncoding.EncodeToString([]byte("1"))
	offset := a.request("GrantEventOptions", map[string]any{"cursor": cursor}, http.StatusOK)
	if got := len(itItems(t, offset)); got != 2 {
		t.Fatalf("GrantEventOptions cursor=1 items = %d, want 2", got)
	}
	// An invalid cursor is a field validation failure.
	a.request("GrantEventOptions", map[string]any{"cursor": "!!!not-base64!!!"}, http.StatusUnprocessableEntity)
	if a.lastCode != "FIELD_VALIDATION_FAILED" {
		t.Fatalf("invalid cursor code = %q, want FIELD_VALIDATION_FAILED", a.lastCode)
	}

	// A non-admin cannot use the selector (itNewStaff returns a ready session).
	ob, _ := itNewStaff(t, e, a, "ORGANIZER")
	ob.request("GrantEventOptions", map[string]any{}, http.StatusForbidden)
	if ob.lastCode != "FORBIDDEN" {
		t.Fatalf("non-admin GrantEventOptions code = %q, want FORBIDDEN", ob.lastCode)
	}
}

// TestIntegrationGrantSaveListRevokeAndLastOrganizer covers admin GrantSave for
// CHECKER/READER/ORGANIZER, GrantList/MyGrantList paging, the duplicate-grant
// conflict, the grantId+version update path, and GrantRevoke including the
// last-organizer guard (409 canonical EVENT_STATE_CONFLICT) and the rule that a
// non-admin can never revoke an ORGANIZER.
func TestIntegrationGrantSaveListRevokeAndLastOrganizer(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)
	ev := itPublishedEvent(t, e, a, 10)
	adminID := itStr(t, a.request("SessionMe", map[string]any{}, http.StatusOK), "accountId")

	// EventCreate auto-grants the creator an ORGANIZER grant; capture it so the
	// last-organizer guard can be exercised against the true final organizer.
	autoGrant := itItems(t, a.request("GrantList", map[string]any{"eventId": ev, "page": 1, "pageSize": 100}, http.StatusOK))[0].(map[string]any)
	if got := itStr(t, autoGrant, "accountId"); got != adminID {
		t.Fatalf("creator auto-grant accountId = %s, want admin %s", got, adminID)
	}
	if got := itStr(t, itMap(t, autoGrant, "role"), "code"); got != "ORGANIZER" {
		t.Fatalf("creator auto-grant role = %s, want ORGANIZER", got)
	}

	checker, checkerAcc := itNewStaff(t, e, a, "CHECKER")
	reader, readerAcc := itNewStaff(t, e, a, "READER")
	org1, org1Acc := itNewStaff(t, e, a, "ORGANIZER")
	_, org2Acc := itNewStaff(t, e, a, "ORGANIZER")
	checkerID := itStr(t, checkerAcc, "accountId")
	readerID := itStr(t, readerAcc, "accountId")
	org1ID := itStr(t, org1Acc, "accountId")
	org2ID := itStr(t, org2Acc, "accountId")

	// CHECKER grant: name field + CHECKIN action.
	checkerGrant := a.write("GrantSave", map[string]any{
		"eventId": ev, "accountId": checkerID, "role": "CHECKER",
		"allowedFields": []any{"name"}, "allowedActions": []any{"CHECKIN"},
		"expiresAt": itFuture(24 * time.Hour),
	}, http.StatusOK)
	if got := itStr(t, itMap(t, checkerGrant, "role"), "code"); got != "CHECKER" {
		t.Fatalf("checker grant role = %s, want CHECKER", got)
	}
	if codes := itCodes(t, checkerGrant, "allowedFields"); !itHasCode(codes, "name") {
		t.Fatalf("checker grant allowedFields = %v, want name", codes)
	}
	// A CHECKER grant carries no disclosure (recipient/purpose/retentionNote absent).
	if _, has := checkerGrant["recipient"]; has {
		t.Fatalf("CHECKER grant must not carry recipient disclosure: %+v", checkerGrant)
	}

	// READER grant requires recipient/purpose/retentionNote and view-only actions.
	readerGrant := a.write("GrantSave", map[string]any{
		"eventId": ev, "accountId": readerID, "role": "READER",
		"allowedFields": []any{"name"}, "allowedActions": []any{"VIEW_DETAIL", "VIEW_STATISTICS"},
		"recipient": "School office", "purpose": "Follow-up", "retentionNote": "Delete after review",
		"expiresAt": itFuture(24 * time.Hour),
	}, http.StatusOK)
	if got := itStr(t, readerGrant, "recipient"); got != "School office" {
		t.Fatalf("reader grant recipient = %q, want School office", got)
	}

	// Two ORGANIZER grants so the last-organizer guard can be exercised.
	org1Grant := a.write("GrantSave", map[string]any{
		"eventId": ev, "accountId": org1ID, "role": "ORGANIZER",
		"allowedFields": []any{"name", "phone"}, "allowedActions": []any{"VIEW_DETAIL", "CHECKIN", "EXPORT"},
		"expiresAt": itFuture(24 * time.Hour),
	}, http.StatusOK)
	org2Grant := a.write("GrantSave", map[string]any{
		"eventId": ev, "accountId": org2ID, "role": "ORGANIZER",
		"allowedFields": []any{"name"}, "allowedActions": []any{"VIEW_DETAIL"},
		"expiresAt": itFuture(24 * time.Hour),
	}, http.StatusOK)

	// GrantList (admin) pages over the five grants of this event: the creator's
	// auto ORGANIZER grant plus CHECKER, READER and the two added ORGANIZERs.
	gl := a.request("GrantList", map[string]any{"eventId": ev, "page": 1, "pageSize": 2}, http.StatusOK)
	if got := itNum(t, gl, "total"); got != 5 {
		t.Fatalf("GrantList total = %d, want 5", got)
	}
	if got := len(itItems(t, gl)); got != 2 {
		t.Fatalf("GrantList page1 items = %d, want 2", got)
	}

	// MyGrantList is scoped to the caller's own account.
	mine := checker.request("MyGrantList", map[string]any{"eventId": ev}, http.StatusOK)
	if got := itNum(t, mine, "total"); got != 1 {
		t.Fatalf("checker MyGrantList total = %d, want 1", got)
	}
	if got := itStr(t, itItems(t, mine)[0].(map[string]any), "accountId"); got != checkerID {
		t.Fatalf("MyGrantList leaked another account: %s", got)
	}
	// eventId narrows to zero for an unrelated event.
	other := itPublishedEvent(t, e, a, 5)
	if got := itNum(t, checker.request("MyGrantList", map[string]any{"eventId": other}, http.StatusOK), "total"); got != 0 {
		t.Fatalf("MyGrantList other-event total = %d, want 0", got)
	}

	// A duplicate (event,account,role) create is a conflict.
	a.write("GrantSave", map[string]any{
		"eventId": ev, "accountId": checkerID, "role": "CHECKER",
		"allowedFields": []any{"name"}, "allowedActions": []any{"CHECKIN"},
		"expiresAt": itFuture(24 * time.Hour),
	}, http.StatusConflict)
	if a.lastCode != "VERSION_CONFLICT" {
		t.Fatalf("duplicate GrantSave code = %q, want VERSION_CONFLICT", a.lastCode)
	}

	// The update path requires grantId+version and bumps the version.
	upd := a.write("GrantSave", map[string]any{
		"eventId": ev, "accountId": checkerID, "role": "CHECKER",
		"grantId": itStr(t, checkerGrant, "grantId"), "version": checkerGrant["version"],
		"allowedFields": []any{"name", "phone"}, "allowedActions": []any{"CHECKIN"},
		"expiresAt": itFuture(24 * time.Hour),
	}, http.StatusOK)
	if got := itNum(t, upd, "version"); got != itNum(t, checkerGrant, "version")+1 {
		t.Fatalf("updated grant version = %d, want %d", got, itNum(t, checkerGrant, "version")+1)
	}
	// A stale grant version is an optimistic conflict.
	a.write("GrantSave", map[string]any{
		"eventId": ev, "accountId": checkerID, "role": "CHECKER",
		"grantId": itStr(t, checkerGrant, "grantId"), "version": checkerGrant["version"],
		"allowedFields": []any{"name"}, "allowedActions": []any{"CHECKIN"},
		"expiresAt": itFuture(24 * time.Hour),
	}, http.StatusConflict)
	if a.lastCode != "VERSION_CONFLICT" {
		t.Fatalf("stale GrantSave code = %q, want VERSION_CONFLICT", a.lastCode)
	}
	// The update path cannot switch the target account.
	a.write("GrantSave", map[string]any{
		"eventId": ev, "accountId": readerID, "role": "CHECKER",
		"grantId": itStr(t, checkerGrant, "grantId"), "version": upd["version"],
		"allowedFields": []any{"name"}, "allowedActions": []any{"CHECKIN"},
		"expiresAt": itFuture(24 * time.Hour),
	}, http.StatusUnprocessableEntity)
	if a.lastCode != "FIELD_VALIDATION_FAILED" {
		t.Fatalf("switch-account GrantSave code = %q, want FIELD_VALIDATION_FAILED", a.lastCode)
	}

	// A non-admin organizer cannot revoke an ORGANIZER grant at all.
	org1.write("GrantRevoke", map[string]any{
		"eventId": ev, "grantId": itStr(t, org2Grant, "grantId"), "version": org2Grant["version"],
	}, http.StatusForbidden)
	if org1.lastCode != "FORBIDDEN" {
		t.Fatalf("organizer revoke ORGANIZER code = %q, want FORBIDDEN", org1.lastCode)
	}

	// Three organizers are live (creator auto-grant + org1 + org2). Revoke the
	// two added ones, leaving the creator's auto-grant as the last organizer.
	a.write("GrantRevoke", map[string]any{
		"eventId": ev, "grantId": itStr(t, org2Grant, "grantId"), "version": org2Grant["version"],
	}, http.StatusOK)
	a.write("GrantRevoke", map[string]any{
		"eventId": ev, "grantId": itStr(t, org1Grant, "grantId"), "version": org1Grant["version"],
	}, http.StatusOK)
	// The final organizer cannot be revoked (409 canonical EVENT_STATE_CONFLICT).
	a.write("GrantRevoke", map[string]any{
		"eventId": ev, "grantId": itStr(t, autoGrant, "grantId"), "version": autoGrant["version"],
	}, http.StatusConflict)
	if a.lastCode != "EVENT_STATE_CONFLICT" {
		t.Fatalf("revoke last organizer code = %q, want canonical EVENT_STATE_CONFLICT (LAST_ORGANIZER)", a.lastCode)
	}
	// A stale revoke version is an optimistic conflict (version checked before the guard).
	a.write("GrantRevoke", map[string]any{
		"eventId": ev, "grantId": itStr(t, autoGrant, "grantId"), "version": 999,
	}, http.StatusConflict)
	if a.lastCode != "VERSION_CONFLICT" {
		t.Fatalf("stale GrantRevoke code = %q, want VERSION_CONFLICT", a.lastCode)
	}

	// Revoking the reader grant cuts off its event access immediately.
	a.write("GrantRevoke", map[string]any{
		"eventId": ev, "grantId": itStr(t, readerGrant, "grantId"), "version": readerGrant["version"],
	}, http.StatusOK)
	if got := itNum(t, reader.request("MyGrantList", map[string]any{"eventId": ev}, http.StatusOK), "total"); got != 0 {
		t.Fatalf("revoked reader MyGrantList total = %d, want 0", got)
	}
	reader.request("PartyList", map[string]any{"eventId": ev}, http.StatusNotFound)
	if reader.lastCode != "RESOURCE_NOT_FOUND" {
		t.Fatalf("revoked reader PartyList code = %q, want RESOURCE_NOT_FOUND", reader.lastCode)
	}
}

// TestIntegrationGrantOrganizerScopeGuards asserts that a non-admin organizer
// can only delegate within its own grant: never ORGANIZER, never EXPORT, never
// fields/actions/expiry beyond its own scope, but a within-scope CHECKER succeeds.
func TestIntegrationGrantOrganizerScopeGuards(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)
	ev := itPublishedEvent(t, e, a, 10)

	org, orgAcc := itNewStaff(t, e, a, "ORGANIZER")
	orgID := itStr(t, orgAcc, "accountId")
	_, checkerAcc := itNewStaff(t, e, a, "CHECKER")
	checkerID := itStr(t, checkerAcc, "accountId")

	// The organizer's own grant: name only, VIEW_DETAIL only, 24h.
	a.write("GrantSave", map[string]any{
		"eventId": ev, "accountId": orgID, "role": "ORGANIZER",
		"allowedFields": []any{"name"}, "allowedActions": []any{"VIEW_DETAIL"},
		"expiresAt": itFuture(24 * time.Hour),
	}, http.StatusOK)

	base := map[string]any{
		"eventId": ev, "accountId": checkerID, "role": "CHECKER",
		"allowedFields": []any{"name"}, "allowedActions": []any{"VIEW_DETAIL"},
		"expiresAt": itFuture(1 * time.Hour),
	}
	clone := func(mut map[string]any) map[string]any {
		out := map[string]any{}
		for k, v := range base {
			out[k] = v
		}
		for k, v := range mut {
			out[k] = v
		}
		return out
	}

	// Cannot grant ORGANIZER (even to an account holding that role: itself).
	org.write("GrantSave", clone(map[string]any{"role": "ORGANIZER", "accountId": orgID}), http.StatusForbidden)
	if org.lastCode != "FORBIDDEN" {
		t.Fatalf("organizer grant ORGANIZER code = %q, want FORBIDDEN", org.lastCode)
	}
	// Cannot grant fields beyond its own (phone).
	org.write("GrantSave", clone(map[string]any{"allowedFields": []any{"name", "phone"}}), http.StatusForbidden)
	if org.lastCode != "FORBIDDEN" {
		t.Fatalf("organizer grant beyond-fields code = %q, want FORBIDDEN", org.lastCode)
	}
	// Cannot grant EXPORT.
	org.write("GrantSave", clone(map[string]any{"allowedActions": []any{"EXPORT"}}), http.StatusForbidden)
	if org.lastCode != "FORBIDDEN" {
		t.Fatalf("organizer grant EXPORT code = %q, want FORBIDDEN", org.lastCode)
	}
	// Cannot grant actions beyond its own (CHECKIN).
	org.write("GrantSave", clone(map[string]any{"allowedActions": []any{"VIEW_DETAIL", "CHECKIN"}}), http.StatusForbidden)
	if org.lastCode != "FORBIDDEN" {
		t.Fatalf("organizer grant beyond-actions code = %q, want FORBIDDEN", org.lastCode)
	}
	// Cannot grant expiry beyond its own (48h > 24h).
	org.write("GrantSave", clone(map[string]any{"expiresAt": itFuture(48 * time.Hour)}), http.StatusForbidden)
	if org.lastCode != "FORBIDDEN" {
		t.Fatalf("organizer grant beyond-expiry code = %q, want FORBIDDEN", org.lastCode)
	}
	// A within-scope CHECKER grant succeeds.
	ok := org.write("GrantSave", clone(nil), http.StatusOK)
	if got := itStr(t, itMap(t, ok, "role"), "code"); got != "CHECKER" {
		t.Fatalf("within-scope grant role = %s, want CHECKER", got)
	}
	// The organizer can list and revoke its own event's grants: the creator's
	// auto ORGANIZER grant, its own ORGANIZER grant and the delegated CHECKER.
	if got := itNum(t, org.request("GrantList", map[string]any{"eventId": ev}, http.StatusOK), "total"); got != 3 {
		t.Fatalf("organizer GrantList total = %d, want 3 (creator auto + own ORGANIZER + delegated CHECKER)", got)
	}
	org.write("GrantRevoke", map[string]any{
		"eventId": ev, "grantId": itStr(t, ok, "grantId"), "version": ok["version"],
	}, http.StatusOK)
}

// TestIntegrationGrantReaderProjectionAndExpiry ports
// schoolReaderFieldsActionsAndRevocationAreEnforcedInQueries: a READER sees only
// its granted fields/actions (no export, no statistics without the action, no
// mutations), the disclosure is sealed at rest, and an expired grant loses all
// access. Also covers the GrantSave validation guards (READER disclosure,
// CHECKER export, READER action set, target role, past expiry).
func TestIntegrationGrantReaderProjectionAndExpiry(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)
	ev := itPublishedEvent(t, e, a, 10)

	// Register a party carrying name + phone so projection can be observed.
	p := e.browser(t)
	p.bootstrap()
	pub := p.request("PublicEvent", map[string]any{"eventId": ev}, http.StatusOK)
	sub := p.write("SubmitRegistration", map[string]any{
		"eventId":               ev,
		"fields":                map[string]any{"name": "Visible Parent", "phone": "+61 400 777 555"},
		"acceptedNoticeVersion": itStr(t, pub, "noticeVersion"),
	}, http.StatusOK)
	partyID := itStr(t, itMap(t, sub, "party"), "partyId")

	reader, readerAcc := itNewStaff(t, e, a, "READER")
	readerID := itStr(t, readerAcc, "accountId")

	// Before any grant the reader cannot even list the roster (404).
	reader.request("PartyList", map[string]any{"eventId": ev}, http.StatusNotFound)

	grant := a.write("GrantSave", map[string]any{
		"eventId": ev, "accountId": readerID, "role": "READER",
		"allowedFields": []any{"name"}, "allowedActions": []any{"VIEW_DETAIL"},
		"recipient": "School test reader", "purpose": "Event follow-up", "retentionNote": "Delete after review",
		"expiresAt": itFuture(24 * time.Hour),
	}, http.StatusOK)
	grantID := itStr(t, grant, "grantId")

	// The disclosure is sealed at rest: the raw cipher must not contain plaintext.
	ctx := context.Background()
	row, err := e.s.Row(ctx, nil, "select disclosure_cipher from ug_event_grant where id=$1", grantID)
	if err != nil {
		t.Fatalf("read grant disclosure: %v", err)
	}
	cipher, ok := row["disclosure_cipher"].([]byte)
	if !ok || len(cipher) == 0 {
		t.Fatalf("disclosure_cipher must be non-empty bytea, got %T", row["disclosure_cipher"])
	}
	if bytes.Contains(cipher, []byte("School test reader")) {
		t.Fatal("disclosure must be sealed, found plaintext recipient in cipher")
	}
	// The view unseals it for authorized callers.
	if got := itStr(t, grant, "recipient"); got != "School test reader" {
		t.Fatalf("grant view recipient = %q, want School test reader", got)
	}

	// PartyDetail projects only the granted field (name), never phone.
	detail := reader.request("PartyDetail", map[string]any{"eventId": ev, "partyId": partyID}, http.StatusOK)
	fields := itMap(t, detail, "fields")
	if len(fields) != 1 {
		t.Fatalf("reader PartyDetail fields = %+v, want only name", fields)
	}
	if _, has := fields["name"]; !has {
		t.Fatalf("reader PartyDetail must include name: %+v", fields)
	}
	if _, has := fields["phone"]; has {
		t.Fatalf("reader PartyDetail must hide phone: %+v", fields)
	}
	// Searching by the hidden phone yields nothing.
	if got := itNum(t, reader.request("PartyList", map[string]any{"eventId": ev, "query": "777"}, http.StatusOK), "total"); got != 0 {
		t.Fatalf("reader phone-search total = %d, want 0", got)
	}
	// No VIEW_STATISTICS action -> Statistics forbidden.
	reader.request("Statistics", map[string]any{"eventId": ev}, http.StatusForbidden)
	if reader.lastCode != "FORBIDDEN" {
		t.Fatalf("reader Statistics code = %q, want FORBIDDEN", reader.lastCode)
	}
	// No export right (ExportCreate is organizer-only).
	reader.write("ExportCreate", map[string]any{
		"eventId": ev, "format": "CSV", "columns": []any{"name"}, "purpose": "No export right",
	}, http.StatusForbidden)
	if reader.lastCode != "FORBIDDEN" {
		t.Fatalf("reader ExportCreate code = %q, want FORBIDDEN", reader.lastCode)
	}
	// No mutation right.
	reader.write("PartyCancel", map[string]any{
		"eventId": ev, "partyId": partyID, "version": 0, "reason": "Forbidden",
	}, http.StatusForbidden)
	if reader.lastCode != "FORBIDDEN" {
		t.Fatalf("reader PartyCancel code = %q, want FORBIDDEN", reader.lastCode)
	}

	// GrantSave validation guards.
	_, checkerAcc := itNewStaff(t, e, a, "CHECKER")
	checkerID := itStr(t, checkerAcc, "accountId")
	// READER without disclosure -> field validation failure.
	a.write("GrantSave", map[string]any{
		"eventId": ev, "accountId": readerID, "role": "READER",
		"allowedFields": []any{"name"}, "allowedActions": []any{"VIEW_DETAIL"},
		"expiresAt": itFuture(24 * time.Hour),
	}, http.StatusUnprocessableEntity)
	if a.lastCode != "FIELD_VALIDATION_FAILED" {
		t.Fatalf("READER missing disclosure code = %q, want FIELD_VALIDATION_FAILED", a.lastCode)
	}
	// CHECKER + EXPORT -> field validation failure (检票员不能导出).
	a.write("GrantSave", map[string]any{
		"eventId": ev, "accountId": checkerID, "role": "CHECKER",
		"allowedFields": []any{"name"}, "allowedActions": []any{"CHECKIN", "EXPORT"},
		"expiresAt": itFuture(24 * time.Hour),
	}, http.StatusUnprocessableEntity)
	if a.lastCode != "FIELD_VALIDATION_FAILED" {
		t.Fatalf("CHECKER+EXPORT code = %q, want FIELD_VALIDATION_FAILED", a.lastCode)
	}
	// READER + CHECKIN action -> field validation failure (校方仅可查看).
	a.write("GrantSave", map[string]any{
		"eventId": ev, "accountId": readerID, "role": "READER",
		"allowedFields": []any{"name"}, "allowedActions": []any{"CHECKIN"},
		"recipient": "R", "purpose": "P", "retentionNote": "N",
		"expiresAt": itFuture(24 * time.Hour),
	}, http.StatusUnprocessableEntity)
	if a.lastCode != "FIELD_VALIDATION_FAILED" {
		t.Fatalf("READER+CHECKIN code = %q, want FIELD_VALIDATION_FAILED", a.lastCode)
	}
	// Target account lacking the role -> field validation failure.
	a.write("GrantSave", map[string]any{
		"eventId": ev, "accountId": readerID, "role": "CHECKER",
		"allowedFields": []any{"name"}, "allowedActions": []any{"CHECKIN"},
		"expiresAt": itFuture(24 * time.Hour),
	}, http.StatusUnprocessableEntity)
	if a.lastCode != "FIELD_VALIDATION_FAILED" {
		t.Fatalf("target missing role code = %q, want FIELD_VALIDATION_FAILED", a.lastCode)
	}
	// Past expiry -> field validation failure.
	a.write("GrantSave", map[string]any{
		"eventId": ev, "accountId": checkerID, "role": "CHECKER",
		"allowedFields": []any{"name"}, "allowedActions": []any{"CHECKIN"},
		"expiresAt": itFuture(-1 * time.Hour),
	}, http.StatusUnprocessableEntity)
	if a.lastCode != "FIELD_VALIDATION_FAILED" {
		t.Fatalf("past expiry code = %q, want FIELD_VALIDATION_FAILED", a.lastCode)
	}

	// Expire the reader grant at the storage layer; access vanishes immediately.
	if _, err := e.s.Exec(ctx, nil, "update ug_event_grant set expires_at=$1 where id=$2", time.Now().UTC().Add(-1*time.Hour), grantID); err != nil {
		t.Fatalf("expire grant: %v", err)
	}
	reader.request("PartyDetail", map[string]any{"eventId": ev, "partyId": partyID}, http.StatusNotFound)
	if reader.lastCode != "RESOURCE_NOT_FOUND" {
		t.Fatalf("expired reader PartyDetail code = %q, want RESOURCE_NOT_FOUND", reader.lastCode)
	}
	if got := itNum(t, reader.request("MyGrantList", map[string]any{"eventId": ev}, http.StatusOK), "total"); got != 0 {
		t.Fatalf("expired reader MyGrantList total = %d, want 0", got)
	}
}