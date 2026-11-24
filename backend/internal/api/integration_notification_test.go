package api

// PostgreSQL integration tests (HTTP level, frozen wire contract) for phone
// verification (SMS), party-entry recovery and EMAIL notification delivery.
// Port of the notification half of IntegrationWorkflowTest
// (smsChallengeProofBindsPhoneEventContextExpiryAndSingleUse,
// challengeFailuresSurviveOuterRollbackAndLockAfterFive,
// recoveryStartDoesNotRevealMatchAndChallengeIsActuallyEmailed,
// recoveryRotatesOnlyPrivateAccessAndConsumesChallenge,
// smtpAcceptanceScrubsQueueAndDoesNotHoldEventTransaction and
// notificationFailureUsesFiveAttemptsAndExponentialBackoff).
//
// SMS travels through the injectable service.HTTPDoer (itProviderMock) and
// EMAIL through a loopback SMTP sink (the Java SmtpCapture protocol); no test
// ever reaches the real network. The delivery worker goroutine is not started
// in the shared fixture, so tests drive one deterministic tick at a time via
// NotificationWorker.DeliverNext (mirroring itPullTick / itRunDueCleanup).
//
// Error assertions use the frozen canonical wire codes: CHALLENGE_INVALID /
// EMAIL_REQUIRED normalize to FIELD_VALIDATION_FAILED and VERIFICATION_DISABLED
// to FORBIDDEN, so the distinctive frozen message pins the exact guard (itGuard).
//
// Run with:
//
//	set -a; . ../.local/go-test.env; set +a
//	go test ./internal/api/ -run 'TestIntegrationSms|TestIntegrationRecovery|TestIntegrationNotification|TestIntegrationEmail' -v

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"unigate/backend/internal/service"
	"unigate/backend/internal/store"
)

// Fixed non-production notification credentials only (never real providers).
const (
	itSmsEndpoint = "https://sms.example.invalid/send"
	itSmsHost     = "sms.example.invalid"
	itSmsToken    = "integration-test-sms-token"
	itMailFrom    = "unigate-it@example.invalid"
	itMailUser    = "integration-test-mail-user"
	itMailPass    = "integration-test-mail-password"
)

// itSmtpMessage is one captured SMTP delivery: envelope plus DATA body.
type itSmtpMessage struct {
	from string
	rcpt []string
	auth string // decoded AUTH PLAIN payload ("\x00user\x00password"), empty when unauthenticated
	data string
}

// itSmtpSink is a test-only loopback SMTP sink (Java SmtpCapture). It answers
// the minimal ESMTP dialogue (220/250/354/221, AUTH PLAIN, 8BITMIME), captures
// every DATA body and has no forwarding or external network capability.
type itSmtpSink struct {
	ln       net.Listener
	mu       sync.Mutex
	messages []itSmtpMessage
	onData   func()
}

func newItSmtpSink(t *testing.T, onData func()) *itSmtpSink {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen loopback SMTP sink: %v", err)
	}
	s := &itSmtpSink{ln: ln, onData: onData}
	go s.acceptLoop()
	t.Cleanup(func() { ln.Close() })
	return s
}

func (s *itSmtpSink) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.handle(conn)
	}
}

func (s *itSmtpSink) handle(conn net.Conn) {
	defer conn.Close()
	// Bound every connection so a stalled client cannot leak the goroutine.
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	reader := bufio.NewReader(conn)
	write := func(text string) { _, _ = conn.Write([]byte(text)) }
	write("220 localhost test SMTP\r\n")
	var msg itSmtpMessage
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		upper := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(upper, "EHLO"), strings.HasPrefix(upper, "HELO"):
			write("250-localhost\r\n250-AUTH PLAIN\r\n250 8BITMIME\r\n")
		case strings.HasPrefix(upper, "AUTH PLAIN"):
			raw, derr := base64.StdEncoding.DecodeString(strings.TrimSpace(line[len("AUTH PLAIN"):]))
			if derr != nil {
				write("535 Authentication failed\r\n")
				continue
			}
			msg.auth = string(raw)
			write("235 Authentication successful\r\n")
		case strings.HasPrefix(upper, "MAIL FROM:"):
			msg.from = itSmtpAddress(line)
			msg.rcpt = nil
			write("250 OK\r\n")
		case strings.HasPrefix(upper, "RCPT TO:"):
			msg.rcpt = append(msg.rcpt, itSmtpAddress(line))
			write("250 OK\r\n")
		case upper == "DATA":
			write("354 Send message\r\n")
			var data strings.Builder
			for {
				dl, derr := reader.ReadString('\n')
				if derr != nil {
					return
				}
				dl = strings.TrimRight(dl, "\r\n")
				if dl == "." {
					break
				}
				// Reverse SMTP dot-stuffing.
				dl = strings.TrimPrefix(dl, ".")
				data.WriteString(dl)
				data.WriteString("\r\n")
			}
			msg.data = data.String()
			if s.onData != nil {
				s.onData()
			}
			s.mu.Lock()
			s.messages = append(s.messages, msg)
			s.mu.Unlock()
			msg = itSmtpMessage{auth: msg.auth}
			write("250 Accepted\r\n")
		case upper == "QUIT":
			write("221 Bye\r\n")
			return
		case upper == "RSET":
			msg = itSmtpMessage{auth: msg.auth}
			write("250 OK\r\n")
		default:
			write("250 OK\r\n")
		}
	}
}

// itSmtpAddress extracts the angle-bracket address of MAIL FROM/RCPT TO.
func itSmtpAddress(line string) string {
	if i := strings.Index(line, "<"); i >= 0 {
		if j := strings.Index(line[i:], ">"); j >= 0 {
			return line[i+1 : i+j]
		}
	}
	if _, rest, ok := strings.Cut(line, ":"); ok {
		return strings.TrimSpace(rest)
	}
	return line
}

// waitMessage polls until the index-th delivery was captured.
func (s *itSmtpSink) waitMessage(t *testing.T, index int) itSmtpMessage {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		s.mu.Lock()
		if len(s.messages) > index {
			m := s.messages[index]
			s.mu.Unlock()
			return m
		}
		s.mu.Unlock()
		if time.Now().After(deadline) {
			t.Fatalf("SMTP sink captured no message %d within 10s", index)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func (s *itSmtpSink) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.messages)
}

// itSmtpHeader reads one header from the captured DATA block.
func itSmtpHeader(t *testing.T, data, name string) string {
	t.Helper()
	for _, line := range strings.Split(data, "\r\n") {
		if strings.HasPrefix(line, name+": ") {
			return strings.TrimPrefix(line, name+": ")
		}
	}
	t.Fatalf("SMTP message has no %s header:\n%s", name, data)
	return ""
}

// itSmtpSubject decodes the RFC2047 base64 subject written by sendEmail.
func itSmtpSubject(t *testing.T, data string) string {
	t.Helper()
	v := itSmtpHeader(t, data, "Subject")
	if strings.HasPrefix(v, "=?UTF-8?B?") && strings.HasSuffix(v, "?=") {
		raw, err := base64.StdEncoding.DecodeString(v[len("=?UTF-8?B?") : len(v)-2])
		if err != nil {
			t.Fatalf("decode SMTP subject: %v", err)
		}
		return string(raw)
	}
	return v
}

// itSmtpBody returns the text after the header/body separator.
func itSmtpBody(t *testing.T, data string) string {
	t.Helper()
	_, body, ok := strings.Cut(data, "\r\n\r\n")
	if !ok {
		t.Fatalf("SMTP message has no body:\n%s", data)
	}
	return body
}

// useSmsConfig enables the SMS gateway with fixed test credentials and the
// allow-listed HTTPS host; the real client is restored by useProviderHTTP.
func (e *itEnv) useSmsConfig(t *testing.T) {
	t.Helper()
	prev := e.cfg.Notifications
	e.cfg.Notifications.SMSEnabled = true
	e.cfg.Notifications.SMSEndpoint = itSmsEndpoint
	e.cfg.Notifications.SMSAllowedHosts = []string{itSmsHost}
	e.cfg.Notifications.SMSToken = itSmsToken
	t.Cleanup(func() { e.cfg.Notifications = prev })
}

// useSmtpSink points sendEmail at the loopback sink (no STARTTLS, optional
// AUTH PLAIN) and restores the notification config when the test ends.
func (e *itEnv) useSmtpSink(t *testing.T, onData func()) *itSmtpSink {
	t.Helper()
	sink := newItSmtpSink(t, onData)
	host, portText, err := net.SplitHostPort(sink.ln.Addr().String())
	if err != nil {
		t.Fatalf("sink address: %v", err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatalf("sink port: %v", err)
	}
	prev := e.cfg.Notifications
	e.cfg.Notifications.EmailEnabled = true
	e.cfg.Notifications.MailHost = host
	e.cfg.Notifications.MailPort = port
	e.cfg.Notifications.From = itMailFrom
	e.cfg.Notifications.SMTPAuth = false
	e.cfg.Notifications.SMTPStartTLS = false
	t.Cleanup(func() { e.cfg.Notifications = prev })
	return sink
}

// itNotificationTick drives one NotificationWorker.deliverNext tick directly
// (the worker goroutine is not started in the shared fixture).
func itNotificationTick(t *testing.T, e *itEnv) {
	t.Helper()
	if err := service.NewNotificationWorker(e.svc).DeliverNext(context.Background()); err != nil {
		t.Fatalf("notification worker tick: %v", err)
	}
}

// itExpireOtherEmailJobs isolates one EMAIL job like the Java tests do before
// driving deliverNext (registration/challenge jobs must not race the tick).
func itExpireOtherEmailJobs(t *testing.T, e *itEnv, keep string) {
	t.Helper()
	if _, err := e.s.Exec(context.Background(), nil,
		"update ug_job set status='EXPIRED' where kind='EMAIL' and id<>$1 and status in ('PENDING','RUNNING')", keep); err != nil {
		t.Fatalf("expire other email jobs: %v", err)
	}
}

// itChallengeJob loads the delivery job queued for a challenge and unseals its
// 6-digit code, asserting the ciphertext never contains the plaintext code.
func itChallengeJob(t *testing.T, e *itEnv, challengeID string) (map[string]any, string) {
	t.Helper()
	job := itRow(t, e, "select * from ug_job where business_key=$1", "challenge:"+challengeID)
	payload, err := e.s.Unseal(job["payload_cipher"])
	if err != nil {
		t.Fatalf("unseal challenge payload: %v", err)
	}
	code := store.StrOrEmpty(payload["code"])
	if len(code) != 6 {
		t.Fatalf("challenge code %q: want 6 digits", code)
	}
	if cipher, ok := job["payload_cipher"].([]byte); ok && bytes.Contains(cipher, []byte(code)) {
		t.Fatal("plaintext challenge code leaks inside the payload ciphertext")
	}
	return job, code
}

// itSmsCall returns the last mock SMS gateway call for a phone number.
func itSmsCall(t *testing.T, m *itProviderMock, phone string) itProviderCall {
	t.Helper()
	var found *itProviderCall
	for _, call := range m.recorded() {
		var body struct {
			Phone string `json:"phone"`
		}
		if err := json.Unmarshal([]byte(call.body), &body); err != nil {
			t.Fatalf("SMS mock body %q: %v", call.body, err)
		}
		if body.Phone == phone {
			c := call
			found = &c
		}
	}
	if found == nil {
		t.Fatalf("no SMS gateway call captured for phone %s", phone)
	}
	return *found
}

// itSmsCode extracts and shape-checks the code of one captured SMS call.
func itSmsCode(t *testing.T, call itProviderCall) string {
	t.Helper()
	var body struct {
		Code             string `json:"code"`
		ExpiresInSeconds int    `json:"expiresInSeconds"`
	}
	if err := json.Unmarshal([]byte(call.body), &body); err != nil {
		t.Fatalf("SMS mock body %q: %v", call.body, err)
	}
	if len(body.Code) != 6 {
		t.Fatalf("SMS code %q: want 6 digits", body.Code)
	}
	if body.ExpiresInSeconds != 300 {
		t.Fatalf("SMS expiresInSeconds = %d, want 300", body.ExpiresInSeconds)
	}
	return body.Code
}

// itWrongCode returns a 6-digit code guaranteed different from the real one.
func itWrongCode(code string) string {
	if code == "000000" {
		return "111111"
	}
	return "000000"
}

// itEnablePhoneVerification turns on phoneVerificationEnabled; the policy guard
// requires the phone field to be enabled and mandatory at the same time.
func itEnablePhoneVerification(t *testing.T, a *itBrowser, ev string) {
	t.Helper()
	policy := a.request("PolicyGet", map[string]any{"eventId": ev}, http.StatusOK)
	fields := itMap(t, policy, "fields")
	phone := itMap(t, fields, "phone")
	phone["enabled"] = true
	phone["required"] = true
	itSelfPolicy(t, a, ev, map[string]any{"phoneVerificationEnabled": true, "fields": fields})
}

// itRegisterFields submits an anonymous registration with explicit fields and
// returns the browser plus the full SubmissionView (party + access bundle).
func itRegisterFields(t *testing.T, e *itEnv, ev string, fields map[string]any) (*itBrowser, map[string]any) {
	t.Helper()
	p := e.browser(t)
	p.bootstrap()
	pub := p.request("PublicEvent", map[string]any{"eventId": ev}, http.StatusOK)
	sub := p.write("SubmitRegistration", map[string]any{
		"eventId":               ev,
		"fields":                fields,
		"acceptedNoticeVersion": itStr(t, pub, "noticeVersion"),
	}, http.StatusOK)
	return p, sub
}

// TestIntegrationSmsStartVerifyAndProof ports
// smsChallengeProofBindsPhoneEventContextExpiryAndSingleUse: SmsStart is a 202
// acceptance with a 60-second resend window, the code travels only through the
// mocked SMS gateway, the proof binds phone/event/client context, and
// SubmitRegistration consumes it exactly once.
func TestIntegrationSmsStartVerifyAndProof(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)
	ev := itPublishedEvent(t, e, a, 10)
	itEnablePhoneVerification(t, a, ev)
	plain := itPublishedEvent(t, e, a, 10)
	e.useSmsConfig(t)
	mock := e.useProviderHTTP(t, func(*http.Request) (*http.Response, error) {
		return itProviderResponse(http.StatusOK, `{}`), nil
	})

	phone := "+8613800001234"
	g := e.browser(t)
	g.bootstrap()

	// A event without phone verification rejects the challenge outright.
	g.write("SmsStart", map[string]any{"eventId": plain, "phone": phone}, http.StatusForbidden)
	itGuard(t, g, "SmsStart", "FORBIDDEN", "此活动未启用手机验证")

	// 202 acceptance: challengeId/expiresAt/retryAfterSeconds, nothing else.
	started := g.write("SmsStart", map[string]any{"eventId": ev, "phone": phone}, http.StatusAccepted)
	challengeID := itStr(t, started, "challengeId")
	itStr(t, started, "expiresAt")
	if got := itNum(t, started, "retryAfterSeconds"); got != 60 {
		t.Fatalf("retryAfterSeconds = %d, want 60", got)
	}
	if len(started) != 3 {
		t.Fatalf("ChallengeView must carry exactly challengeId/expiresAt/retryAfterSeconds: %+v", started)
	}

	// The 60-second per-contact resend window.
	g.write("SmsStart", map[string]any{"eventId": ev, "phone": phone}, http.StatusTooManyRequests)
	itGuard(t, g, "SmsStart", "RATE_LIMITED", "请等待60秒后重试")

	// The queued job seals the code; the worker delivers it via the mock gateway.
	job, code := itChallengeJob(t, e, challengeID)
	itExpireOtherEmailJobs(t, e, store.StrOrEmpty(job["id"]))
	itNotificationTick(t, e)
	call := itSmsCall(t, mock, phone)
	if call.method != http.MethodPost || call.url.String() != itSmsEndpoint {
		t.Fatalf("SMS call %s %s, want POST %s", call.method, call.url, itSmsEndpoint)
	}
	if got := call.header.Get("Authorization"); got != "Bearer "+itSmsToken {
		t.Fatalf("SMS Authorization = %q, want the configured bearer token", got)
	}
	if got := call.header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("SMS Content-Type = %q, want application/json", got)
	}
	if got := itSmsCode(t, call); got != code {
		t.Fatalf("delivered SMS code %q != sealed challenge code %q", got, code)
	}
	delivered := itRow(t, e, "select * from ug_job where id=$1", job["id"])
	if store.StrOrEmpty(delivered["status"]) != "SUCCEEDED" || delivered["payload_cipher"] != nil {
		t.Fatalf("SMS job after delivery: status=%v payload=%v, want SUCCEEDED with scrubbed payload",
			delivered["status"], delivered["payload_cipher"])
	}

	// A different client context cannot verify, and the attempt counter stays 0.
	stranger := e.browser(t)
	stranger.bootstrap()
	stranger.write("SmsVerify", map[string]any{"eventId": ev, "challengeId": challengeID, "code": code}, http.StatusUnprocessableEntity)
	itGuard(t, stranger, "SmsVerify", "FIELD_VALIDATION_FAILED", "验证码无效、已过期或尝试次数过多")
	if got := itCount(t, e, "select attempts from ug_challenge where id=$1", challengeID); got != 0 {
		t.Fatalf("attempts after foreign-context verify = %d, want 0", got)
	}
	// A mismatched eventId is rejected without consuming an attempt either.
	g.write("SmsVerify", map[string]any{"eventId": plain, "challengeId": challengeID, "code": code}, http.StatusUnprocessableEntity)
	itGuard(t, g, "SmsVerify", "FIELD_VALIDATION_FAILED", "验证码无效、已过期或尝试次数过多")
	if got := itCount(t, e, "select attempts from ug_challenge where id=$1", challengeID); got != 0 {
		t.Fatalf("attempts after foreign-event verify = %d, want 0", got)
	}

	// The original context verifies and receives the signed single-use proof.
	verified := g.write("SmsVerify", map[string]any{"eventId": ev, "challengeId": challengeID, "code": code}, http.StatusOK)
	proof := itStr(t, verified, "verificationProof")
	itStr(t, verified, "expiresAt")
	if row := itRow(t, e, "select consumed_at from ug_challenge where id=$1", challengeID); row["consumed_at"] == nil {
		t.Fatal("verified challenge must be consumed")
	}

	pub := g.request("PublicEvent", map[string]any{"eventId": ev}, http.StatusOK)
	nv := itStr(t, pub, "noticeVersion")
	fields := map[string]any{"name": "Sms Parent", "phone": phone, "companionCount": 0}
	// Missing proof.
	g.write("SubmitRegistration", map[string]any{
		"eventId": ev, "fields": fields, "acceptedNoticeVersion": nv,
	}, http.StatusUnprocessableEntity)
	itGuard(t, g, "SubmitRegistration", "VERIFICATION_REQUIRED", "请先验证手机号码")
	// Proof bound to another phone number.
	g.write("SubmitRegistration", map[string]any{
		"eventId": ev, "acceptedNoticeVersion": nv, "phoneVerificationProof": proof,
		"fields": map[string]any{"name": "Sms Parent", "phone": "+8613900000000", "companionCount": 0},
	}, http.StatusUnprocessableEntity)
	itGuard(t, g, "SubmitRegistration", "VERIFICATION_REQUIRED", "手机号验证无效或已使用")
	// Proof bound to another client context.
	other := e.browser(t)
	other.bootstrap()
	other.write("SubmitRegistration", map[string]any{
		"eventId": ev, "fields": fields, "acceptedNoticeVersion": nv, "phoneVerificationProof": proof,
	}, http.StatusUnprocessableEntity)
	itGuard(t, other, "SubmitRegistration", "VERIFICATION_REQUIRED", "手机验证已失效")

	// The matching registration succeeds and consumes the proof exactly once.
	sub := g.write("SubmitRegistration", map[string]any{
		"eventId": ev, "fields": fields, "acceptedNoticeVersion": nv, "phoneVerificationProof": proof,
	}, http.StatusOK)
	if got := itStr(t, itMap(t, sub, "party"), "partyId"); got == "" {
		t.Fatal("verified registration must create a party")
	}
	g.write("SubmitRegistration", map[string]any{
		"eventId": ev, "fields": fields, "acceptedNoticeVersion": nv, "phoneVerificationProof": proof,
	}, http.StatusUnprocessableEntity)
	itGuard(t, g, "SubmitRegistration", "VERIFICATION_REQUIRED", "手机号验证无效或已使用")
}

// TestIntegrationSmsBruteForceAndExpiry ports
// challengeFailuresSurviveOuterRollbackAndLockAfterFive plus the provider and
// expiry guards: five wrong codes lock the challenge even though every outer
// request transaction rolls back, the sixth (correct) code stays rejected, an
// expired challenge is rejected without burning an attempt, and an unconfigured
// SMS provider is a 503 acceptance guard.
func TestIntegrationSmsBruteForceAndExpiry(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)
	ev := itPublishedEvent(t, e, a, 10)
	itEnablePhoneVerification(t, a, ev)

	g := e.browser(t)
	g.bootstrap()
	phone := "+8613800005678"

	// Provider not configured: acceptance is refused before any challenge.
	g.write("SmsStart", map[string]any{"eventId": ev, "phone": phone}, http.StatusServiceUnavailable)
	itGuard(t, g, "SmsStart", "PROVIDER_UNAVAILABLE", "短信验证服务尚未配置")
	if got := itCount(t, e, "select count(*) from ug_challenge where event_id=$1", ev); got != 0 {
		t.Fatalf("challenges after unconfigured SmsStart = %d, want 0", got)
	}

	e.useSmsConfig(t)
	e.useProviderHTTP(t, func(*http.Request) (*http.Response, error) {
		return itProviderResponse(http.StatusOK, `{}`), nil
	})
	started := g.write("SmsStart", map[string]any{"eventId": ev, "phone": phone}, http.StatusAccepted)
	challengeID := itStr(t, started, "challengeId")
	_, code := itChallengeJob(t, e, challengeID)

	// Five wrong codes: each 422 rolls the outer request transaction back, yet
	// the brute-force counter commits independently (REQUIRES_NEW).
	wrong := itWrongCode(code)
	for i := 1; i <= 5; i++ {
		g.write("SmsVerify", map[string]any{"eventId": ev, "challengeId": challengeID, "code": wrong}, http.StatusUnprocessableEntity)
		itGuard(t, g, "SmsVerify", "FIELD_VALIDATION_FAILED", "验证码无效、已过期或尝试次数过多")
		if got := itCount(t, e, "select attempts from ug_challenge where id=$1", challengeID); got != int64(i) {
			t.Fatalf("attempts after wrong code %d = %d, want %d (counter must survive outer rollback)", i, got, i)
		}
	}
	// The correct code is locked out after five failures.
	g.write("SmsVerify", map[string]any{"eventId": ev, "challengeId": challengeID, "code": code}, http.StatusUnprocessableEntity)
	itGuard(t, g, "SmsVerify", "FIELD_VALIDATION_FAILED", "验证码无效、已过期或尝试次数过多")
	if got := itCount(t, e, "select attempts from ug_challenge where id=$1", challengeID); got != 5 {
		t.Fatalf("attempts after lockout = %d, want 5", got)
	}
	if row := itRow(t, e, "select consumed_at from ug_challenge where id=$1", challengeID); row["consumed_at"] != nil {
		t.Fatal("locked-out challenge must not be consumed")
	}

	// An expired challenge is rejected without burning an attempt.
	late := g.write("SmsStart", map[string]any{"eventId": ev, "phone": phone + "1"}, http.StatusAccepted)
	lateID := itStr(t, late, "challengeId")
	_, lateCode := itChallengeJob(t, e, lateID)
	if _, err := e.s.Exec(context.Background(), nil,
		"update ug_challenge set expires_at=now()-interval '1 second' where id=$1", lateID); err != nil {
		t.Fatalf("expire challenge: %v", err)
	}
	g.write("SmsVerify", map[string]any{"eventId": ev, "challengeId": lateID, "code": lateCode}, http.StatusUnprocessableEntity)
	itGuard(t, g, "SmsVerify", "FIELD_VALIDATION_FAILED", "验证码无效、已过期或尝试次数过多")
	if got := itCount(t, e, "select attempts from ug_challenge where id=$1", lateID); got != 0 {
		t.Fatalf("attempts after expired verify = %d, want 0", got)
	}
	// Unknown challenge ids are rejected the same way.
	g.write("SmsVerify", map[string]any{"eventId": ev, "challengeId": e.s.ID(), "code": code}, http.StatusUnprocessableEntity)
	itGuard(t, g, "SmsVerify", "FIELD_VALIDATION_FAILED", "验证码无效、已过期或尝试次数过多")
}

// TestIntegrationRecoveryStartMatchAndDelivery ports
// recoveryStartDoesNotRevealMatchAndChallengeIsActuallyEmailed: matching and
// non-matching RecoveryStart requests return the identical 202 envelope, only
// a matched party with a configured provider queues a job, the sealed payload
// never leaks the plaintext code, and the worker actually delivers the code to
// the loopback SMTP sink.
func TestIntegrationRecoveryStartMatchAndDelivery(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)
	ev := itPublishedEvent(t, e, a, 10)
	_, sub := itRegisterFields(t, e, ev, map[string]any{
		"name": "Recovery Parent", "email": "Parent@example.invalid", "companionCount": 0,
	})
	party := itMap(t, sub, "party")
	partyID := itStr(t, party, "partyId")
	reference := itStr(t, party, "partyReference")
	sink := e.useSmtpSink(t, nil)

	g := e.browser(t)
	g.bootstrap()
	missing := g.write("RecoveryStart", map[string]any{
		"eventId": ev, "partyReference": "UG-DOES-NOT-EXIST", "channel": "EMAIL", "contact": "Absent@example.invalid",
	}, http.StatusAccepted)
	// The contact domain is case-insensitive; the local part is preserved.
	matched := g.write("RecoveryStart", map[string]any{
		"eventId": ev, "partyReference": reference, "channel": "EMAIL", "contact": "Parent@EXAMPLE.INVALID",
	}, http.StatusAccepted)
	if len(missing) != 3 || len(matched) != 3 {
		t.Fatalf("RecoveryStart must not reveal match: missing=%+v matched=%+v", missing, matched)
	}
	for _, key := range []string{"challengeId", "expiresAt", "retryAfterSeconds"} {
		if _, ok := missing[key]; !ok {
			t.Fatalf("missing envelope lacks %q", key)
		}
		if _, ok := matched[key]; !ok {
			t.Fatalf("matched envelope lacks %q", key)
		}
	}
	if got := itNum(t, missing, "retryAfterSeconds"); got != 60 {
		t.Fatalf("retryAfterSeconds = %d, want 60", got)
	}
	missingID := itStr(t, missing, "challengeId")
	matchedID := itStr(t, matched, "challengeId")

	// Only the matched challenge binds the party and queues a delivery job.
	if got := itCount(t, e, "select count(*) from ug_job where business_key=$1", "challenge:"+missingID); got != 0 {
		t.Fatalf("non-matching RecoveryStart queued %d jobs, want 0", got)
	}
	if row := itRow(t, e, "select party_id from ug_challenge where id=$1", missingID); row["party_id"] != nil {
		t.Fatal("non-matching challenge must not bind a party")
	}
	if row := itRow(t, e, "select party_id from ug_challenge where id=$1", matchedID); store.StrOrEmpty(row["party_id"]) != partyID {
		t.Fatalf("matched challenge party_id = %v, want %s", row["party_id"], partyID)
	}
	job, code := itChallengeJob(t, e, matchedID)
	itExpireOtherEmailJobs(t, e, store.StrOrEmpty(job["id"]))
	itNotificationTick(t, e)

	msg := sink.waitMessage(t, 0)
	if got := itSmtpSubject(t, msg.data); got != "UniGate 验证码" {
		t.Fatalf("challenge email subject = %q, want UniGate 验证码", got)
	}
	if body := itSmtpBody(t, msg.data); !strings.Contains(body, code) {
		t.Fatalf("challenge email body does not contain the code:\n%s", body)
	}
	if got := itSmtpHeader(t, msg.data, "To"); got != "Parent@example.invalid" {
		t.Fatalf("challenge email To = %q, want the normalized contact", got)
	}
	if len(msg.rcpt) != 1 || msg.rcpt[0] != "Parent@example.invalid" {
		t.Fatalf("SMTP envelope recipients = %v, want the normalized contact", msg.rcpt)
	}
	if msg.from != itMailFrom {
		t.Fatalf("SMTP envelope from = %q, want the configured %s", msg.from, itMailFrom)
	}
	if got := sink.count(); got != 1 {
		t.Fatalf("SMTP sink captured %d deliveries, want exactly 1 (no delivery for the unmatched request)", got)
	}
	delivered := itRow(t, e, "select * from ug_job where id=$1", job["id"])
	if store.StrOrEmpty(delivered["status"]) != "SUCCEEDED" || delivered["payload_cipher"] != nil {
		t.Fatalf("challenge job after delivery: status=%v payload=%v, want SUCCEEDED with scrubbed payload",
			delivered["status"], delivered["payload_cipher"])
	}
}

// TestIntegrationRecoveryVerifyRotatesCredentials ports
// recoveryRotatesOnlyPrivateAccessAndConsumesChallenge: a correct recovery code
// rotates only the private QUERY/CHANGE capabilities (never the ticket or the
// allocation), revokes the old party sessions, consumes the challenge once,
// and wrong/expired/unknown codes are rejected with the frozen guard.
func TestIntegrationRecoveryVerifyRotatesCredentials(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)
	ev := itPublishedEvent(t, e, a, 10)
	p, sub := itRegisterFields(t, e, ev, map[string]any{
		"name": "Recovery Parent", "email": "parent@example.invalid", "companionCount": 1,
	})
	party := itMap(t, sub, "party")
	partyID := itStr(t, party, "partyId")
	reference := itStr(t, party, "partyReference")
	oldAccess := itMap(t, sub, "access")
	p.write("PartyExchange", map[string]any{
		"queryToken":  itStr(t, oldAccess, "queryToken"),
		"changeToken": itStr(t, oldAccess, "changeToken"),
	}, http.StatusOK)
	p.request("MyParty", map[string]any{}, http.StatusOK)
	oldCreds := make([]string, 0, 2)
	rows, err := e.s.Rows(context.Background(), nil,
		"select id from ug_credential where party_id=$1 and kind in ('QUERY','CHANGE') and status='ACTIVE'", partyID)
	if err != nil {
		t.Fatalf("list old access credentials: %v", err)
	}
	for _, row := range rows {
		oldCreds = append(oldCreds, store.StrOrEmpty(row["id"]))
	}
	if len(oldCreds) != 2 {
		t.Fatalf("expected 2 active QUERY/CHANGE credentials before recovery, got %d", len(oldCreds))
	}

	// Email must be configured for the matched challenge to queue a job; the
	// delivery itself is covered by TestIntegrationRecoveryStartMatchAndDelivery.
	e.useSmtpSink(t, nil)
	g := e.browser(t)
	g.bootstrap()
	matched := g.write("RecoveryStart", map[string]any{
		"eventId": ev, "partyReference": reference, "channel": "EMAIL", "contact": "parent@example.invalid",
	}, http.StatusAccepted)
	challengeID := itStr(t, matched, "challengeId")
	job, code := itChallengeJob(t, e, challengeID)
	itExpireOtherEmailJobs(t, e, store.StrOrEmpty(job["id"]))

	// A wrong code burns one attempt but rotates nothing.
	g.write("RecoveryVerify", map[string]any{"challengeId": challengeID, "code": itWrongCode(code)}, http.StatusUnprocessableEntity)
	itGuard(t, g, "RecoveryVerify", "FIELD_VALIDATION_FAILED", "验证码无效、已过期或尝试次数过多")
	if got := itCount(t, e, "select attempts from ug_challenge where id=$1", challengeID); got != 1 {
		t.Fatalf("attempts after wrong recovery code = %d, want 1", got)
	}
	if got := itCount(t, e, "select count(*) from ug_credential where party_id=$1 and kind in ('QUERY','CHANGE') and status='ACTIVE'", partyID); got != 2 {
		t.Fatalf("wrong code must not rotate credentials, active = %d", got)
	}

	// The correct code returns a fresh access bundle.
	access := g.write("RecoveryVerify", map[string]any{"challengeId": challengeID, "code": code}, http.StatusOK)
	newQuery := itStr(t, access, "queryToken")
	newChange := itStr(t, access, "changeToken")
	itStr(t, access, "expiresAt")
	if newQuery == itStr(t, oldAccess, "queryToken") || newChange == itStr(t, oldAccess, "changeToken") {
		t.Fatal("recovery must mint new private tokens")
	}
	for _, id := range oldCreds {
		if got := store.StrOrEmpty(itRow(t, e, "select status from ug_credential where id=$1", id)["status"]); got != "REVOKED" {
			t.Fatalf("old access credential %s status = %s, want REVOKED", id, got)
		}
	}
	if got := itCount(t, e, "select count(*) from ug_credential where party_id=$1 and kind='TICKET' and status='ACTIVE'", partyID); got != 1 {
		t.Fatalf("recovery must keep the ticket credential, active tickets = %d", got)
	}
	if got := itCount(t, e, "select allocated_count from ug_event where id=$1", ev); got != 2 {
		t.Fatalf("recovery must not change the allocation, allocated_count = %d", got)
	}
	if row := itRow(t, e, "select consumed_at from ug_challenge where id=$1", challengeID); row["consumed_at"] == nil {
		t.Fatal("recovery challenge must be consumed")
	}

	// Replaying the consumed challenge is rejected (still the anon CSRF here).
	g.write("RecoveryVerify", map[string]any{"challengeId": challengeID, "code": code}, http.StatusUnprocessableEntity)
	itGuard(t, g, "RecoveryVerify", "FIELD_VALIDATION_FAILED", "验证码无效、已过期或尝试次数过多")
	// Unknown challenge ids are rejected the same way.
	g.write("RecoveryVerify", map[string]any{"challengeId": e.s.ID(), "code": code}, http.StatusUnprocessableEntity)
	itGuard(t, g, "RecoveryVerify", "FIELD_VALIDATION_FAILED", "验证码无效、已过期或尝试次数过多")

	// The old party sessions are revoked: the original browser loses access.
	p.request("MyParty", map[string]any{}, http.StatusUnauthorized)
	itGuard(t, p, "MyParty", "SESSION_EXPIRED", "私密访问已失效")
	// The recovered browser exchanges the new tokens and regains access.
	g.write("PartyExchange", map[string]any{"queryToken": newQuery, "changeToken": newChange}, http.StatusOK)
	mine := g.request("MyParty", map[string]any{}, http.StatusOK)
	if got := itStr(t, mine, "partyId"); got != partyID {
		t.Fatalf("recovered MyParty partyId = %s, want %s", got, partyID)
	}

	// An expired recovery challenge is rejected without burning an attempt
	// (a fresh anon client: after PartyExchange g carries the party CSRF).
	_, sub2 := itRegisterFields(t, e, ev, map[string]any{
		"name": "Second Parent", "email": "second@example.invalid", "companionCount": 0,
	})
	party2 := itMap(t, sub2, "party")
	g2 := e.browser(t)
	g2.bootstrap()
	late := g2.write("RecoveryStart", map[string]any{
		"eventId": ev, "partyReference": itStr(t, party2, "partyReference"), "channel": "EMAIL", "contact": "second@example.invalid",
	}, http.StatusAccepted)
	lateID := itStr(t, late, "challengeId")
	_, lateCode := itChallengeJob(t, e, lateID)
	if _, err := e.s.Exec(context.Background(), nil,
		"update ug_challenge set expires_at=now()-interval '1 second' where id=$1", lateID); err != nil {
		t.Fatalf("expire recovery challenge: %v", err)
	}
	g2.write("RecoveryVerify", map[string]any{"challengeId": lateID, "code": lateCode}, http.StatusUnprocessableEntity)
	itGuard(t, g2, "RecoveryVerify", "FIELD_VALIDATION_FAILED", "验证码无效、已过期或尝试次数过多")
	if got := itCount(t, e, "select attempts from ug_challenge where id=$1", lateID); got != 0 {
		t.Fatalf("attempts after expired recovery verify = %d, want 0", got)
	}
	if got := itCount(t, e, "select count(*) from ug_credential where party_id=$1 and kind in ('QUERY','CHANGE') and status='ACTIVE'", itStr(t, party2, "partyId")); got != 2 {
		t.Fatalf("expired recovery must not rotate the second party's credentials, active = %d", got)
	}
}

// TestIntegrationEmailResendDeliveryAndFailureIsolation ports
// smtpAcceptanceScrubsQueueAndDoesNotHoldEventTransaction and
// notificationFailureUsesFiveAttemptsAndExponentialBackoff: EmailResend is a
// 202 job acceptance, the worker delivers the credential email through the
// loopback sink with credentials taken from configuration (AUTH PLAIN), the
// event row is not locked during SMTP IO, success scrubs the sealed payload,
// and delivery failures only requeue/fail the job with exponential backoff —
// never touching eligibility.
func TestIntegrationEmailResendDeliveryAndFailureIsolation(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	a := e.readyAdmin(t)
	ev := itPublishedEvent(t, e, a, 10)
	_, sub := itRegisterFields(t, e, ev, map[string]any{
		"name": "SMTP Parent", "email": "Parent@example.invalid", "companionCount": 0,
	})
	party := itMap(t, sub, "party")
	partyID := itStr(t, party, "partyId")
	reference := itStr(t, party, "partyReference")
	_, subNoEmail := itRegisterFields(t, e, ev, map[string]any{
		"name": "No Email Parent", "companionCount": 0,
	})
	noEmailID := itStr(t, itMap(t, subNoEmail, "party"), "partyId")

	// Staff without the ORGANIZER role cannot resend.
	checker, _ := itNewStaff(t, e, a, "CHECKER")
	checker.write("EmailResend", map[string]any{
		"eventId": ev, "partyId": partyID, "reason": "No right",
	}, http.StatusForbidden)
	itGuard(t, checker, "EmailResend", "FORBIDDEN", "没有操作权限")
	// Email provider disabled: 503 before any job is queued.
	a.write("EmailResend", map[string]any{
		"eventId": ev, "partyId": partyID, "reason": "Provider off",
	}, http.StatusServiceUnavailable)
	itGuard(t, a, "EmailResend", "PROVIDER_UNAVAILABLE", "邮件服务尚未启用")

	// During DATA the event row must be lockable: provider IO runs outside any
	// database transaction or event lock (Java unlocked AtomicBoolean).
	var unlocked atomic.Bool
	sink := e.useSmtpSink(t, func() {
		if _, err := e.s.Exec(context.Background(), nil,
			"select id from ug_event where id=$1 for update nowait", ev); err == nil {
			unlocked.Store(true)
		}
	})
	// Credentials come from configuration only: AUTH PLAIN with fixed test values.
	e.cfg.Notifications.SMTPAuth = true
	e.cfg.Notifications.MailUsername = itMailUser
	e.cfg.Notifications.MailPassword = itMailPass
	prevBase := e.cfg.PublicBaseURL
	e.cfg.PublicBaseURL = "https://unigate.example"
	t.Cleanup(func() { e.cfg.PublicBaseURL = prevBase })

	// A party without an email is a 422 acceptance guard.
	a.write("EmailResend", map[string]any{
		"eventId": ev, "partyId": noEmailID, "reason": "No email",
	}, http.StatusUnprocessableEntity)
	itGuard(t, a, "EmailResend", "FIELD_VALIDATION_FAILED", "此组未提供邮箱")

	// 202 job acceptance, then one deterministic delivery tick.
	resend := a.write("EmailResend", map[string]any{
		"eventId": ev, "partyId": partyID, "reason": "Participant requested another email",
	}, http.StatusAccepted)
	jobID := itStr(t, resend, "jobId")
	if code := itMap(t, resend, "status")["code"]; code != "PENDING" {
		t.Fatalf("resent job status = %v, want PENDING", code)
	}
	if code := itMap(t, resend, "kind")["code"]; code != "EMAIL" {
		t.Fatalf("resent job kind = %v, want EMAIL", code)
	}
	itExpireOtherEmailJobs(t, e, jobID)
	itNotificationTick(t, e)

	msg := sink.waitMessage(t, 0)
	if got := itSmtpSubject(t, msg.data); got != "UniGate 活动报名凭证" {
		t.Fatalf("credential email subject = %q, want UniGate 活动报名凭证", got)
	}
	body := itSmtpBody(t, msg.data)
	if !strings.Contains(body, "https://unigate.example/me/party#queryToken=") {
		t.Fatalf("credential email lacks the configured self-service link:\n%s", body)
	}
	if !strings.Contains(body, reference) {
		t.Fatalf("credential email lacks the party reference:\n%s", body)
	}
	if got := itSmtpHeader(t, msg.data, "To"); got != "Parent@example.invalid" {
		t.Fatalf("credential email To = %q, want the original party email", got)
	}
	if msg.auth != "\x00"+itMailUser+"\x00"+itMailPass {
		t.Fatalf("SMTP AUTH PLAIN payload = %q, want the configured credentials", msg.auth)
	}
	if !unlocked.Load() {
		t.Fatal("event row was locked during SMTP delivery (provider IO must run outside the event transaction)")
	}
	delivered := itRow(t, e, "select * from ug_job where id=$1", jobID)
	if store.StrOrEmpty(delivered["status"]) != "SUCCEEDED" {
		t.Fatalf("delivered job status = %v, want SUCCEEDED", delivered["status"])
	}
	if delivered["payload_cipher"] != nil || delivered["lease_until"] != nil {
		t.Fatal("successful delivery must scrub the payload and clear the lease")
	}
	if got := store.Num(delivered["attempts"]); got != 1 {
		t.Fatalf("delivered job attempts = %d, want 1", got)
	}
	if got := itCount(t, e, "select allocated_count from ug_event where id=$1", ev); got != 2 {
		t.Fatalf("delivery must not change the allocation, allocated_count = %d", got)
	}

	// Delivery failure: point SMTP at a closed loopback port. Five attempts
	// requeue with exponential backoff, then FAILED — eligibility untouched.
	closed, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen closed port: %v", err)
	}
	_, closedPortText, _ := net.SplitHostPort(closed.Addr().String())
	closedPort, _ := strconv.Atoi(closedPortText)
	closed.Close()
	e.cfg.Notifications.MailPort = closedPort
	resend2 := a.write("EmailResend", map[string]any{
		"eventId": ev, "partyId": partyID, "reason": "Failure isolation",
	}, http.StatusAccepted)
	jobID2 := itStr(t, resend2, "jobId")
	itExpireOtherEmailJobs(t, e, jobID2)
	for attempt := 1; attempt <= 5; attempt++ {
		itNotificationTick(t, e)
		job := itRow(t, e, "select * from ug_job where id=$1", jobID2)
		if got := store.Num(job["attempts"]); got != attempt {
			t.Fatalf("attempt %d: job attempts = %d", attempt, got)
		}
		wantStatus := "PENDING"
		if attempt == 5 {
			wantStatus = "FAILED"
		}
		if got := store.StrOrEmpty(job["status"]); got != wantStatus {
			t.Fatalf("attempt %d: job status = %s, want %s", attempt, got, wantStatus)
		}
		if job["lease_until"] != nil {
			t.Fatalf("attempt %d: lease must be cleared", attempt)
		}
		if got := store.StrOrEmpty(job["error_code"]); got != "PROVIDER_UNAVAILABLE" {
			t.Fatalf("attempt %d: error_code = %s, want PROVIDER_UNAVAILABLE", attempt, got)
		}
		if attempt < 5 {
			nextRun, err := store.Time(job["next_run_at"])
			if err != nil {
				t.Fatalf("attempt %d: next_run_at: %v", attempt, err)
			}
			backoff := time.Duration(1<<uint(attempt-1)) * time.Minute
			if nextRun.Before(time.Now().Add(backoff - 5*time.Second)) {
				t.Fatalf("attempt %d: next_run_at %s backoff shorter than %s", attempt, nextRun, backoff)
			}
			if _, err := e.s.Exec(context.Background(), nil,
				"update ug_job set next_run_at=now() where id=$1", jobID2); err != nil {
				t.Fatalf("reset backoff: %v", err)
			}
		}
	}
	// A further tick does nothing once the job is FAILED.
	itNotificationTick(t, e)
	if got := itCount(t, e, "select attempts from ug_job where id=$1", jobID2); got != 5 {
		t.Fatalf("attempts after exhaustion = %d, want 5", got)
	}
	// Eligibility survives every delivery failure.
	if got := store.StrOrEmpty(itRow(t, e, "select status from ug_party where id=$1", partyID)["status"]); got != "APPROVED" {
		t.Fatalf("party status after delivery failures = %s, want APPROVED", got)
	}
	if got := itCount(t, e, "select allocated_count from ug_event where id=$1", ev); got != 2 {
		t.Fatalf("allocation after delivery failures = %d, want 2", got)
	}
	if got := itCount(t, e, "select count(*) from ug_credential where party_id=$1 and kind='TICKET' and status='ACTIVE'", partyID); got != 1 {
		t.Fatalf("delivery failures must not revoke the ticket, active = %d", got)
	}
}