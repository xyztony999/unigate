package api

// Concurrency regression tests (HTTP level, frozen wire contract), porting the
// Java WorkflowIntegrationTest concurrency methods:
//
//   - concurrentLastPlaceHasOneDatabaseEffect  -> TestIntegrationLastPlace*
//   - parallelReadsShareActivityWithoutSerializationConflicts
//     -> TestIntegrationRaceParallelReads
//   - independentTabsKeepCsrfAcrossConcurrentSessionReads
//     -> TestIntegrationCsrfMultiTabRotation
//
// plus the concurrent idempotent-replay invariant (same operationId submitted
// in parallel takes effect exactly once; every other caller receives the sealed
// original result without double counting).
//
// itBrowser is not goroutine-safe (cookie/csrf capture mutates it), so every
// concurrent caller gets its own itTab clone sharing the session cookies, and
// itAttempt performs a soft round trip (t.Errorf, never t.Fatalf inside a
// goroutine) returning the real status/code/data for the main goroutine to
// assert. Run with -race:
//
//	set -a; . ../.local/go-test.env; set +a
//	go test ./internal/api/ -run 'TestIntegrationConcurren|TestIntegrationLastPlace|TestIntegrationRace|TestIntegrationCsrf' -v -race

import (
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
)

// itTab clones a browser into an independent "tab" of the same session: the
// cookie values, csrf token and client context are copied, but all subsequent
// capture stays local to the clone so goroutines never share mutable state.
func itTab(b *itBrowser) *itBrowser {
	cookies := make(map[string]string, len(b.cookies))
	for k, v := range b.cookies {
		cookies[k] = v
	}
	return &itBrowser{t: b.t, env: b.env, cookies: cookies, csrf: b.csrf, context: b.context}
}

// itAttemptResult carries the real HTTP status and frozen envelope of one
// concurrent attempt (status 0 means the transport itself failed).
type itAttemptResult struct {
	op      string
	status  int
	code    string
	message string
	data    map[string]any
}

// itAttempt is the goroutine-safe counterpart of itBrowser.request: it sends
// one contract operation with the clone's cookies/csrf/context and validates
// the frozen envelope via t.Errorf (safe from any goroutine, unlike t.Fatalf)
// without mutating the browser or aborting on a business error, so mixed
// 200/409 outcomes can be collected and asserted by the test goroutine.
func itAttempt(t *testing.T, b *itBrowser, name string, args map[string]any) itAttemptResult {
	t.Helper()
	res := itAttemptResult{op: name}
	op := b.env.c.Operations[name]
	if op == nil {
		t.Errorf("unknown contract operation %q", name)
		return res
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
			t.Errorf("%s: marshal args: %v", name, merr)
			return res
		}
		req, err = http.NewRequest(http.MethodPost, b.env.server.URL+op.Path, strings.NewReader(string(body)))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
		}
	}
	if err != nil {
		t.Errorf("%s: build request: %v", name, err)
		return res
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
		t.Errorf("%s: do request: %v", name, err)
		return res
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("%s: read body: %v", name, err)
		return res
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
		t.Errorf("%s: decode envelope (status %d): %v\nbody: %s", name, resp.StatusCode, err, raw)
		return res
	}
	res.status = resp.StatusCode
	res.code = envelope.Code
	res.message = envelope.Message
	res.data = envelope.Data
	if envelope.RequestID == "" || envelope.RequestID != resp.Header.Get("X-Request-ID") {
		t.Errorf("%s: requestId %q missing or != X-Request-ID %q", name, envelope.RequestID, resp.Header.Get("X-Request-ID"))
	}
	if envelope.Data == nil {
		t.Errorf("%s: envelope data must be an object ({} on errors), got null", name)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if envelope.Code != "OK" || envelope.Message != "操作成功" {
			t.Errorf("%s: success envelope code=%q message=%q, want OK/操作成功", name, envelope.Code, envelope.Message)
		}
	} else if envelope.Code == "" || envelope.Code == "OK" {
		t.Errorf("%s: error envelope must carry a string error code, got %q", name, envelope.Code)
	}
	return res
}

// itBurst releases n goroutines at once (Java CountDownLatch start) and waits
// for all of them; fn must only touch its own index of preallocated results.
func itBurst(n int, fn func(i int)) {
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			<-start
			fn(i)
		}(i)
	}
	close(start)
	wg.Wait()
}

// itWantOK asserts a collected attempt succeeded with the frozen envelope.
func itWantOK(t *testing.T, res itAttemptResult) {
	t.Helper()
	if res.status != http.StatusOK || res.code != "OK" {
		t.Fatalf("%s: status=%d code=%q message=%q, want 200/OK", res.op, res.status, res.code, res.message)
	}
}

// TestIntegrationLastPlaceRegistrationRace ports concurrentLastPlaceHasOneDatabaseEffect
// with six racers: capacity 1, parallel SubmitRegistration from independent
// anonymous sessions -> exactly one 200, every loser the canonical 409
// CAPACITY_EXCEEDED, and the database shows one allocated place / one party.
func TestIntegrationLastPlaceRegistrationRace(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)
	ev := itPublishedEvent(t, e, a, 1)

	const racers = 6
	browsers := make([]*itBrowser, racers)
	for i := range browsers {
		browsers[i] = e.browser(t)
		browsers[i].bootstrap()
	}
	pub := browsers[0].request("PublicEvent", map[string]any{"eventId": ev}, http.StatusOK)
	notice := itStr(t, pub, "noticeVersion")
	args := make([]map[string]any, racers)
	for i := range args {
		args[i] = map[string]any{
			"operationId":           e.s.ID(),
			"eventId":               ev,
			"fields":                map[string]any{"name": fmt.Sprintf("Racer %d", i), "companionCount": 0},
			"acceptedNoticeVersion": notice,
		}
	}
	results := make([]itAttemptResult, racers)
	itBurst(racers, func(i int) {
		results[i] = itAttempt(t, browsers[i], "SubmitRegistration", args[i])
	})

	winner := -1
	for i, res := range results {
		if res.status == http.StatusOK {
			if winner != -1 {
				t.Fatalf("two racers won the last place: %+v and %+v", results[winner], res)
			}
			winner = i
			itWantOK(t, res)
			party := itMap(t, res.data, "party")
			if got := itNum(t, party, "approvedSize"); got != 1 {
				t.Fatalf("winner approvedSize = %d, want 1", got)
			}
			if code := itMap(t, party, "status")["code"]; code != "APPROVED" {
				t.Fatalf("winner party status = %v, want APPROVED", code)
			}
			continue
		}
		if res.status != http.StatusConflict || res.code != "CAPACITY_EXCEEDED" {
			t.Fatalf("loser %d: status=%d code=%q, want canonical 409 CAPACITY_EXCEEDED (message %q)", i, res.status, res.code, res.message)
		}
		if len(res.data) != 0 {
			t.Fatalf("loser %d: error envelope data must be {}, got %+v", i, res.data)
		}
	}
	if winner == -1 {
		t.Fatal("no racer won the last place; every submission was rejected")
	}

	// Database effect: exactly one allocated place and one party row; the
	// losers' idempotency rows rolled back with their transactions.
	if got := itCount(t, e, "select allocated_count from ug_event where id=$1", ev); got != 1 {
		t.Fatalf("allocated_count = %d, want 1", got)
	}
	if got := itCount(t, e, "select count(*) from ug_party where event_id=$1", ev); got != 1 {
		t.Fatalf("party rows = %d, want 1", got)
	}
	if got := itCount(t, e, "select count(*) from ug_operation where event_id=$1 and action='SubmitRegistration'", ev); got != 1 {
		t.Fatalf("SubmitRegistration operation rows = %d, want 1 (losers rolled back)", got)
	}
	stats := itGateStats(t, a, ev)
	if got := itNum(t, stats, "allocatedPeople"); got != 1 {
		t.Fatalf("allocatedPeople = %d, want 1 (0 <= allocated <= capacity)", got)
	}
	if got := itNum(t, stats, "netAdmissions"); got != 0 {
		t.Fatalf("netAdmissions = %d, want 0 before check-in", got)
	}
}

// TestIntegrationLastPlaceGateCommitRace races two kiosks committing the final
// remaining place of a one-person group: exactly one SUCCEEDED admission, the
// other a canonical 409 balance/version guard, and a fresh re-scan afterwards
// pins PARTY_REMAINING_EXCEEDED. Net/gross admissions never exceed the group.
func TestIntegrationLastPlaceGateCommitRace(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)
	ev := itPublishedEvent(t, e, a, 2)
	_, _, ticket := itRegisteredParty(t, e, ev, "Last Place Parent", 0)

	terminals := make([]*itBrowser, 2)
	for i := range terminals {
		terminals[i], _ = itClaimedDevice(t, e, a, ev, "KIOSK")
	}
	itTransition(t, a, ev, "OPEN", "CHECKIN_OPEN")
	scans := make([]map[string]any, 2)
	args := make([]map[string]any, 2)
	for i := range terminals {
		scans[i] = terminals[i].request("TicketResolve", map[string]any{"eventId": ev, "ticketToken": ticket}, http.StatusOK)
		if got := itNum(t, scans[i], "remainingCount"); got != 1 {
			t.Fatalf("scan %d remainingCount = %d, want the single last place", i, got)
		}
		args[i] = map[string]any{
			"operationId":          e.s.ID(),
			"eventId":              ev,
			"scanContextToken":     itStr(t, scans[i], "scanContextToken"),
			"expectedPartyVersion": scans[i]["partyVersion"],
			"count":                1,
		}
	}
	results := make([]itAttemptResult, 2)
	itBurst(2, func(i int) {
		results[i] = itAttempt(t, terminals[i], "GateCommit", args[i])
	})

	winner := -1
	for i, res := range results {
		if res.status == http.StatusOK {
			if winner != -1 {
				t.Fatalf("both kiosks admitted the last place: %+v / %+v", results[0], results[1])
			}
			winner = i
			itWantOK(t, res)
			if code := itMap(t, res.data, "status")["code"]; code != "SUCCEEDED" {
				t.Fatalf("winner GateCommit status = %v, want SUCCEEDED", code)
			}
			if got := itNum(t, res.data, "committedCount"); got != 1 {
				t.Fatalf("winner committedCount = %d, want 1", got)
			}
			if got := itNum(t, res.data, "remainingCount"); got != 0 {
				t.Fatalf("winner remainingCount = %d, want 0", got)
			}
			if itBool(t, res.data, "replayed") || !itBool(t, res.data, "showAsNewAdmission") {
				t.Fatalf("winner must be a new admission: %+v", res.data)
			}
			continue
		}
		// The loser scanned before the winner committed, so it hits either the
		// optimistic version guard or the balance guard — both frozen 409s.
		if res.status != http.StatusConflict || (res.code != "PARTY_REMAINING_EXCEEDED" && res.code != "VERSION_CONFLICT") {
			t.Fatalf("loser %d: status=%d code=%q, want canonical 409 PARTY_REMAINING_EXCEEDED/VERSION_CONFLICT", i, res.status, res.code)
		}
	}
	if winner == -1 {
		t.Fatal("neither kiosk admitted the last place")
	}

	// A fresh scan after the race sees the committed balance and pins the
	// canonical PARTY_REMAINING_EXCEEDED for any further commit.
	loser := terminals[1-winner]
	rescan := loser.request("TicketResolve", map[string]any{"eventId": ev, "ticketToken": ticket}, http.StatusOK)
	if got := itNum(t, rescan, "netAdmitted"); got != 1 {
		t.Fatalf("rescan netAdmitted = %d, want 1", got)
	}
	if got := itNum(t, rescan, "remainingCount"); got != 0 {
		t.Fatalf("rescan remainingCount = %d, want 0", got)
	}
	loser.write("GateCommit", map[string]any{
		"eventId":              ev,
		"scanContextToken":     itStr(t, rescan, "scanContextToken"),
		"expectedPartyVersion": rescan["partyVersion"],
		"count":                1,
	}, http.StatusConflict)
	if loser.lastCode != "PARTY_REMAINING_EXCEEDED" {
		t.Fatalf("over-commit code = %q, want canonical PARTY_REMAINING_EXCEEDED", loser.lastCode)
	}

	if got := itCount(t, e, "select count(*) from ug_admission where event_id=$1", ev); got != 1 {
		t.Fatalf("admission rows = %d, want exactly 1", got)
	}
	stats := itGateStats(t, a, ev)
	if got := itNum(t, stats, "netAdmissions"); got != 1 {
		t.Fatalf("netAdmissions = %d, want 1 (0 <= net <= approvedSize)", got)
	}
	if got := itNum(t, stats, "grossAdmissions"); got != 1 {
		t.Fatalf("grossAdmissions = %d, want 1 (no double count)", got)
	}
	records := a.request("GateRecords", map[string]any{"eventId": ev}, http.StatusOK)
	if got := itNum(t, records, "total"); got != 1 {
		t.Fatalf("GateRecords total = %d, want 1", got)
	}
}

// TestIntegrationRaceParallelReads ports parallelReadsShareActivityWithoutSerializationConflicts
// and extends it: 24 parallel Statistics/PartyList/MyParty reads must all
// return 200 with one consistent snapshot, then 8 Statistics readers race a
// checker committing admissions and every captured snapshot must satisfy the
// admission invariants (no dirty reads, no serialization deadlock).
func TestIntegrationRaceParallelReads(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)
	ev := itPublishedEvent(t, e, a, 50)
	_, _, _ = itRegisteredParty(t, e, ev, "Read Parent A", 0)
	pB, partyB, ticketB := itRegisteredParty(t, e, ev, "Read Parent B", 1)
	_, _, _ = itRegisteredParty(t, e, ev, "Read Parent C", 0)
	partyBID := itStr(t, partyB, "partyId")

	// Phase 1: 24 concurrent reads over a quiescent event -> identical snapshot.
	const readers = 24
	tabs := make([]*itBrowser, readers)
	ops := make([]string, readers)
	args := make([]map[string]any, readers)
	for i := 0; i < readers; i++ {
		switch i % 3 {
		case 0:
			tabs[i], ops[i], args[i] = itTab(a), "Statistics", map[string]any{"eventId": ev}
		case 1:
			tabs[i], ops[i], args[i] = itTab(a), "PartyList", map[string]any{"eventId": ev, "page": 1, "pageSize": 20}
		default:
			tabs[i], ops[i], args[i] = itTab(pB), "MyParty", map[string]any{}
		}
	}
	results := make([]itAttemptResult, readers)
	itBurst(readers, func(i int) {
		results[i] = itAttempt(t, tabs[i], ops[i], args[i])
	})
	for i, res := range results {
		itWantOK(t, res)
		switch ops[i] {
		case "Statistics":
			if got := itNum(t, res.data, "allocatedPeople"); got != 4 {
				t.Fatalf("read %d Statistics allocatedPeople = %d, want the consistent 4", i, got)
			}
			if got := itNum(t, res.data, "netAdmissions"); got != 0 {
				t.Fatalf("read %d Statistics netAdmissions = %d, want 0", i, got)
			}
			if got := itNum(t, res.data, "applicationGroups"); got != 3 {
				t.Fatalf("read %d Statistics applicationGroups = %d, want 3", i, got)
			}
			if got := itNum(t, res.data, "unusedPlaces"); got != 4 {
				t.Fatalf("read %d Statistics unusedPlaces = %d, want 4", i, got)
			}
		case "PartyList":
			if got := itNum(t, res.data, "total"); got != 3 {
				t.Fatalf("read %d PartyList total = %d, want the consistent 3", i, got)
			}
		case "MyParty":
			if got := itStr(t, res.data, "partyId"); got != partyBID {
				t.Fatalf("read %d MyParty partyId = %s, want %s", i, got, partyBID)
			}
			if got := itNum(t, res.data, "approvedSize"); got != 2 {
				t.Fatalf("read %d MyParty approvedSize = %d, want 2", i, got)
			}
		}
	}

	// Phase 2: readers keep sampling Statistics while a checker admits the
	// two-person group in two commits. Every snapshot must be internally
	// consistent (REPEATABLE READ, never a half-applied admission).
	terminal, _ := itClaimedDevice(t, e, a, ev, "STAFF")
	writer := itTab(a)
	writer.cookies["ug_device"] = terminal.cookies["ug_device"]
	itTransition(t, a, ev, "OPEN", "CHECKIN_OPEN")

	const samplers = 8
	const samples = 10
	samplerTabs := make([]*itBrowser, samplers)
	for i := range samplerTabs {
		samplerTabs[i] = itTab(a)
	}
	snapshots := make([][]itAttemptResult, samplers)
	for i := range snapshots {
		snapshots[i] = make([]itAttemptResult, samples)
	}
	writerSteps := make([]itAttemptResult, 4)
	itBurst(samplers+1, func(i int) {
		if i < samplers {
			for j := 0; j < samples; j++ {
				snapshots[i][j] = itAttempt(t, samplerTabs[i], "Statistics", map[string]any{"eventId": ev})
			}
			return
		}
		scan := itAttempt(t, writer, "TicketResolve", map[string]any{"eventId": ev, "ticketToken": ticketB})
		writerSteps[0] = scan
		if scan.status != http.StatusOK {
			return
		}
		writerSteps[1] = itAttempt(t, writer, "GateCommit", map[string]any{
			"operationId":          e.s.ID(),
			"eventId":              ev,
			"scanContextToken":     scan.data["scanContextToken"],
			"expectedPartyVersion": scan.data["partyVersion"],
			"count":                1,
		})
		scan2 := itAttempt(t, writer, "TicketResolve", map[string]any{"eventId": ev, "ticketToken": ticketB})
		writerSteps[2] = scan2
		if scan2.status != http.StatusOK {
			return
		}
		writerSteps[3] = itAttempt(t, writer, "GateCommit", map[string]any{
			"operationId":          e.s.ID(),
			"eventId":              ev,
			"scanContextToken":     scan2.data["scanContextToken"],
			"expectedPartyVersion": scan2.data["partyVersion"],
			"count":                1,
		})
	})
	for _, step := range writerSteps {
		itWantOK(t, step)
	}
	for i := range snapshots {
		for j, res := range snapshots[i] {
			itWantOK(t, res)
			allocated := itNum(t, res.data, "allocatedPeople")
			net := itNum(t, res.data, "netAdmissions")
			gross := itNum(t, res.data, "grossAdmissions")
			corrected := itNum(t, res.data, "correctedPeople")
			if allocated != 4 {
				t.Fatalf("snapshot %d/%d allocatedPeople = %d, want stable 4 (commits never allocate)", i, j, allocated)
			}
			if net < 0 || net > 2 || net > allocated {
				t.Fatalf("snapshot %d/%d dirty read: netAdmissions=%d outside [0,2] or > allocated=%d", i, j, net, allocated)
			}
			if gross != net || corrected != 0 {
				t.Fatalf("snapshot %d/%d inconsistent: net=%d gross=%d corrected=%d", i, j, net, gross, corrected)
			}
			if got := itNum(t, res.data, "unusedPlaces"); got != allocated-net {
				t.Fatalf("snapshot %d/%d unusedPlaces = %d, want allocated-net = %d", i, j, got, allocated-net)
			}
		}
	}
	stats := itGateStats(t, a, ev)
	if got := itNum(t, stats, "netAdmissions"); got != 2 {
		t.Fatalf("final netAdmissions = %d, want 2", got)
	}
	if got := itNum(t, stats, "grossAdmissions"); got != 2 {
		t.Fatalf("final grossAdmissions = %d, want 2", got)
	}
	records := a.request("GateRecords", map[string]any{"eventId": ev, "partyId": partyBID}, http.StatusOK)
	if got := itNum(t, records, "total"); got != 2 {
		t.Fatalf("GateRecords total = %d, want 2", got)
	}
}

// TestIntegrationCsrfMultiTabRotation ports independentTabsKeepCsrfAcrossConcurrentSessionReads:
// parallel SessionMe/SessionBootstrap reads from many tabs of the same staff
// and anonymous sessions never rotate the CSRF token, the original tabs keep
// writing afterwards, a pre-upgrade stored random digest stays accepted, and
// PasswordChange revokes every session of the account.
func TestIntegrationCsrfMultiTabRotation(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)
	originalStaffCsrf := a.csrf
	anon := e.browser(t)
	anon.bootstrap()
	originalAnonCsrf := anon.csrf

	const tabs = 12
	staffTabs := make([]*itBrowser, tabs)
	anonTabs := make([]*itBrowser, tabs)
	for i := 0; i < tabs; i++ {
		staffTabs[i] = itTab(a)
		anonTabs[i] = itTab(anon)
	}
	results := make([]itAttemptResult, 2*tabs)
	itBurst(2*tabs, func(i int) {
		if i < tabs {
			results[i] = itAttempt(t, staffTabs[i], "SessionMe", map[string]any{})
		} else {
			results[i] = itAttempt(t, anonTabs[i-tabs], "SessionBootstrap", map[string]any{})
		}
	})
	for i, res := range results {
		itWantOK(t, res)
		want := originalStaffCsrf
		if i >= tabs {
			want = originalAnonCsrf
		}
		if got := itStr(t, res.data, "csrfToken"); got != want {
			t.Fatalf("concurrent read %d rotated csrf: got %q, want the stable %q", i, got, want)
		}
	}
	// The original tabs still hold a valid form token after the read storm.
	a.request("AccountList", map[string]any{}, http.StatusOK)
	anon.write("Login", map[string]any{"username": "missing-" + e.s.ID(), "password": "Wrong-password-123!"}, http.StatusUnauthorized)
	if anon.lastCode != "UNAUTHENTICATED" {
		t.Fatalf("anon login code = %q, want UNAUTHENTICATED (csrf accepted, credentials rejected)", anon.lastCode)
	}

	// A pre-upgrade tab with a stored random CSRF digest also remains valid
	// (Java updates ug_session.csrf_digest directly), and the deterministic
	// token keeps working alongside it.
	oldRandomToken := e.s.Crypto.Token()
	if _, err := e.s.Exec(context.Background(), nil,
		"update ug_session set csrf_digest=$1 where token_digest=$2",
		e.s.Crypto.Hash(oldRandomToken), e.s.Crypto.Hash(a.cookies["ug_staff"])); err != nil {
		t.Fatalf("seed legacy csrf digest: %v", err)
	}
	a.csrf = oldRandomToken
	a.request("AccountList", map[string]any{}, http.StatusOK)
	a.csrf = originalStaffCsrf
	a.request("AccountList", map[string]any{}, http.StatusOK)

	// PasswordChange revokes every session of the account, not just the
	// changing tab: a second logged-in tab is cut off immediately.
	account := a.write("AccountCreate", map[string]any{
		"username":          "csrf-tab-" + e.s.ID(),
		"displayName":       "CSRF Tab Staff",
		"temporaryPassword": "Temporary-staff-123!",
		"roles":             []any{"CHECKER"},
	}, http.StatusOK)
	username := itStr(t, account, "username")
	tab1 := e.browser(t)
	tab1.bootstrap()
	tab1.write("Login", map[string]any{"username": username, "password": "Temporary-staff-123!"}, http.StatusOK)
	tab2 := e.browser(t)
	tab2.bootstrap()
	tab2.write("Login", map[string]any{"username": username, "password": "Temporary-staff-123!"}, http.StatusOK)
	tab1.request("SessionMe", map[string]any{}, http.StatusOK)
	tab2.request("SessionMe", map[string]any{}, http.StatusOK)
	tab1.write("PasswordChange", map[string]any{"oldPassword": "Temporary-staff-123!", "newPassword": "Changed-staff-123!"}, http.StatusOK)
	tab1.request("SessionMe", map[string]any{}, http.StatusUnauthorized)
	if tab1.lastCode != "SESSION_EXPIRED" {
		t.Fatalf("changing tab SessionMe code = %q, want SESSION_EXPIRED", tab1.lastCode)
	}
	tab2.request("SessionMe", map[string]any{}, http.StatusUnauthorized)
	if tab2.lastCode != "SESSION_EXPIRED" {
		t.Fatalf("other tab SessionMe code = %q, want SESSION_EXPIRED (all sessions revoked)", tab2.lastCode)
	}
	if got := itCount(t, e, "select count(*) from ug_session where account_id=$1 and revoked_at is not null", itStr(t, account, "accountId")); got != 2 {
		t.Fatalf("revoked sessions = %d, want 2 (every tab)", got)
	}
	// The new password logs in again with mustChangePassword cleared.
	tab2.bootstrap()
	tab2.write("Login", map[string]any{"username": username, "password": "Changed-staff-123!"}, http.StatusOK)
	me := tab2.request("SessionMe", map[string]any{}, http.StatusOK)
	if itBool(t, me, "mustChangePassword") {
		t.Fatal("mustChangePassword must be false after the change")
	}
}

// TestIntegrationConcurrentIdempotentRegistration submits the same
// operationId from four tabs of one anonymous session in parallel: exactly
// one execution creates the party, the other three replay the sealed result,
// and nothing is double counted.
func TestIntegrationConcurrentIdempotentRegistration(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)
	ev := itPublishedEvent(t, e, a, 3)
	anon := e.browser(t)
	anon.bootstrap()
	pub := anon.request("PublicEvent", map[string]any{"eventId": ev}, http.StatusOK)
	operationID := e.s.ID()
	args := map[string]any{
		"operationId":           operationID,
		"eventId":               ev,
		"fields":                map[string]any{"name": "Idempotent Parent", "companionCount": 1},
		"acceptedNoticeVersion": itStr(t, pub, "noticeVersion"),
	}
	const tabs = 4
	clones := make([]*itBrowser, tabs)
	for i := range clones {
		clones[i] = itTab(anon)
	}
	results := make([]itAttemptResult, tabs)
	itBurst(tabs, func(i int) {
		results[i] = itAttempt(t, clones[i], "SubmitRegistration", args)
	})

	fresh := 0
	var partyID string
	for _, res := range results {
		itWantOK(t, res)
		party := itMap(t, res.data, "party")
		id := itStr(t, party, "partyId")
		if partyID == "" {
			partyID = id
		} else if id != partyID {
			t.Fatalf("replay returned a different party %s, want the sealed %s", id, partyID)
		}
		if got := itNum(t, party, "approvedSize"); got != 2 {
			t.Fatalf("party approvedSize = %d, want 2", got)
		}
		if itBool(t, res.data, "replayed") {
			continue
		}
		fresh++
	}
	if fresh != 1 {
		t.Fatalf("fresh executions = %d, want exactly 1 (others replay)", fresh)
	}
	if got := itCount(t, e, "select count(*) from ug_party where event_id=$1", ev); got != 1 {
		t.Fatalf("party rows = %d, want 1 (no duplicate registration)", got)
	}
	if got := itCount(t, e, "select allocated_count from ug_event where id=$1", ev); got != 2 {
		t.Fatalf("allocated_count = %d, want 2 (no double allocation)", got)
	}
	if got := itCount(t, e, "select count(*) from ug_operation where operation_id=$1", operationID); got != 1 {
		t.Fatalf("operation rows for operationId = %d, want 1", got)
	}
	// The original anonymous context can still query the sealed result.
	queried := anon.request("RegistrationResult", map[string]any{"eventId": ev, "operationId": operationID}, http.StatusOK)
	if !itBool(t, queried, "replayed") {
		t.Fatal("RegistrationResult must flag replayed")
	}
	if got := itStr(t, itMap(t, queried, "party"), "partyId"); got != partyID {
		t.Fatalf("RegistrationResult partyId = %s, want %s", got, partyID)
	}
	stats := itGateStats(t, a, ev)
	if got := itNum(t, stats, "allocatedPeople"); got != 2 {
		t.Fatalf("allocatedPeople = %d, want 2", got)
	}
	if got := itNum(t, stats, "applicationGroups"); got != 1 {
		t.Fatalf("applicationGroups = %d, want 1", got)
	}
}

// TestIntegrationConcurrentIdempotentGateCommit fires the same GateCommit
// operationId from four tabs of one claimed device in parallel: exactly one
// new admission, three sealed replays with the same recordId, and the
// admission counters stay at one.
func TestIntegrationConcurrentIdempotentGateCommit(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)
	ev := itPublishedEvent(t, e, a, 5)
	_, _, ticket := itRegisteredParty(t, e, ev, "Idempotent Gate Parent", 1)
	terminal, _ := itClaimedDevice(t, e, a, ev, "KIOSK")
	itTransition(t, a, ev, "OPEN", "CHECKIN_OPEN")
	scan := terminal.request("TicketResolve", map[string]any{"eventId": ev, "ticketToken": ticket}, http.StatusOK)
	operationID := e.s.ID()
	args := map[string]any{
		"operationId":          operationID,
		"eventId":              ev,
		"scanContextToken":     itStr(t, scan, "scanContextToken"),
		"expectedPartyVersion": scan["partyVersion"],
		"count":                1,
	}
	const tabs = 4
	clones := make([]*itBrowser, tabs)
	for i := range clones {
		clones[i] = itTab(terminal)
	}
	results := make([]itAttemptResult, tabs)
	itBurst(tabs, func(i int) {
		results[i] = itAttempt(t, clones[i], "GateCommit", args)
	})

	fresh := 0
	var recordID string
	for _, res := range results {
		itWantOK(t, res)
		if code := itMap(t, res.data, "status")["code"]; code != "SUCCEEDED" {
			t.Fatalf("GateCommit status = %v, want SUCCEEDED", code)
		}
		id := itStr(t, res.data, "recordId")
		if recordID == "" {
			recordID = id
		} else if id != recordID {
			t.Fatalf("replay returned a different recordId %s, want the sealed %s", id, recordID)
		}
		if got := itNum(t, res.data, "committedCount"); got != 1 {
			t.Fatalf("committedCount = %d, want the sealed 1", got)
		}
		if got := itNum(t, res.data, "remainingCount"); got != 1 {
			t.Fatalf("remainingCount = %d, want the sealed 1 (approvedSize 2 - 1)", got)
		}
		if itBool(t, res.data, "replayed") {
			if itBool(t, res.data, "showAsNewAdmission") {
				t.Fatalf("replay must not show as new admission: %+v", res.data)
			}
			continue
		}
		if !itBool(t, res.data, "showAsNewAdmission") {
			t.Fatalf("the single execution must show as new admission: %+v", res.data)
		}
		fresh++
	}
	if fresh != 1 {
		t.Fatalf("fresh admissions = %d, want exactly 1 (others replay)", fresh)
	}
	if got := itCount(t, e, "select count(*) from ug_admission where event_id=$1", ev); got != 1 {
		t.Fatalf("admission rows = %d, want 1 (no double count)", got)
	}
	if got := itCount(t, e, "select count(*) from ug_operation where operation_id=$1", operationID); got != 1 {
		t.Fatalf("operation rows for operationId = %d, want 1", got)
	}
	stats := itGateStats(t, a, ev)
	if got := itNum(t, stats, "netAdmissions"); got != 1 {
		t.Fatalf("netAdmissions = %d, want 1", got)
	}
	if got := itNum(t, stats, "grossAdmissions"); got != 1 {
		t.Fatalf("grossAdmissions = %d, want 1", got)
	}
	records := a.request("GateRecords", map[string]any{"eventId": ev}, http.StatusOK)
	if got := itNum(t, records, "total"); got != 1 {
		t.Fatalf("GateRecords total = %d, want 1", got)
	}
	// A fresh resolve confirms the balance was applied exactly once.
	rescan := terminal.request("TicketResolve", map[string]any{"eventId": ev, "ticketToken": ticket}, http.StatusOK)
	if got := itNum(t, rescan, "netAdmitted"); got != 1 {
		t.Fatalf("rescan netAdmitted = %d, want 1", got)
	}
	if got := itNum(t, rescan, "remainingCount"); got != 1 {
		t.Fatalf("rescan remainingCount = %d, want 1", got)
	}
}
