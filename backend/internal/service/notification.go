package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"
	"time"

	"unigate/backend/internal/errcode"
	"unigate/backend/internal/store"
)

// This file reproduces com.unigate.NotificationService: opt-in challenge
// acceptance/verification and the EMAIL job delivery worker. Queue payloads
// are encrypted and never returned to clients or logs. A delivery failure
// never revokes eligibility; it only updates the job state.

func (svc *Service) emailEnabled() bool {
	return svc.Cfg.Notifications.EmailEnabled && strings.TrimSpace(svc.Cfg.Notifications.MailHost) != ""
}

func (svc *Service) requireEmail() error {
	if err := errcode.Require(svc.emailEnabled(), 503, "PROVIDER_UNAVAILABLE", "邮件服务尚未启用"); err != nil {
		return err
	}
	return nil
}

func (svc *Service) smsEnabled() bool {
	return svc.Cfg.Notifications.SMSEnabled && strings.TrimSpace(svc.Cfg.Notifications.SMSEndpoint) != ""
}

// handleNotification dispatches the challenge operations (NotificationService.handle).
func (svc *Service) handleNotification(ctx context.Context, sc *store.Scope, op string, r map[string]any, c *Req) (any, error) {
	if strings.HasSuffix(op, "Start") {
		return svc.notificationStart(ctx, sc, op, r, c)
	}
	return svc.notificationVerify(ctx, sc, op, r, c)
}

func (svc *Service) notificationStart(ctx context.Context, sc *store.Scope, op string, r map[string]any, c *Req) (any, error) {
	sms := op == "SmsStart"
	event, err := svc.S.Event(ctx, sc, store.StrOrEmpty(r["eventId"]))
	if err != nil {
		return nil, err
	}
	contactKey := "contact"
	if sms {
		contactKey = "phone"
	}
	contact := strings.TrimSpace(store.StrOrEmpty(r[contactKey]))
	channel := store.StrOrEmpty(r["channel"])
	if sms {
		channel = "SMS"
	}
	contact = normalizeContact(contact, channel)
	if sms {
		if err := errcode.Require(store.Parse(event["policy"])["phoneVerificationEnabled"] == true,
			403, "VERIFICATION_DISABLED", "此活动未启用手机验证"); err != nil {
			return nil, err
		}
		if err := errcode.Require(svc.smsEnabled(), 503, "PROVIDER_UNAVAILABLE", "短信验证服务尚未配置"); err != nil {
			return nil, err
		}
	}
	kind := "RECOVERY"
	if sms {
		kind = "SMS"
	}
	scope := kind + ":" + hex.EncodeToString(svc.S.Crypto.Hash(contact))
	contextID := store.StrOrEmpty(c.Session["id"])

	recent, err := svc.S.RecentChallenges(ctx, sc)
	if err != nil {
		return nil, err
	}
	clientCount, ipCount := 0, 0
	for _, old := range recent {
		info, err := svc.S.Unseal(old["context_cipher"])
		if err != nil {
			return nil, err
		}
		if contextID == store.StrOrEmpty(info["contextId"]) {
			clientCount++
		}
		if c.ClientIP != "" && c.ClientIP == store.StrOrEmpty(info["ip"]) {
			ipCount++
		}
	}
	if err := errcode.Require(clientCount < 10 && ipCount < 30, 429, "RATE_LIMITED", "验证申请过多，请稍后重试"); err != nil {
		return nil, err
	}
	active, err := svc.S.ActiveScopeChallenges(ctx, sc, scope)
	if err != nil {
		return nil, err
	}
	if err := errcode.Require(len(active) == 0, 429, "RATE_LIMITED", "请等待60秒后重试"); err != nil {
		return nil, err
	}

	id := svc.S.ID()
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return nil, fmt.Errorf("generate challenge code: %w", err)
	}
	code := fmt.Sprintf("%06d", n.Int64())
	due, err := store.Time(event["personal_due_at"])
	if err != nil {
		return nil, fmt.Errorf("event due time: %w", err)
	}
	now := svc.S.Now()
	expires := minTime(now.Add(5*time.Minute), due)

	var partyID any
	if !sms {
		candidates, err := svc.S.PartiesByReference(ctx, sc, store.StrOrEmpty(event["id"]), store.StrOrEmpty(r["partyReference"]))
		if err != nil {
			return nil, err
		}
		if len(candidates) > 0 {
			candidate := candidates[0]
			fields, err := svc.S.Unseal(candidate["fields_cipher"])
			if err != nil {
				return nil, err
			}
			field := "phone"
			if channel == "EMAIL" {
				field = "email"
			}
			if fields[field] != nil && contact == normalizeContact(store.StrOrEmpty(fields[field]), channel) {
				partyID = candidate["id"]
			}
		}
	}

	targetCipher, err := svc.S.Seal(sc, store.Map("contact", contact, "channel", channel))
	if err != nil {
		return nil, err
	}
	var ipValue any
	if c.ClientIP != "" {
		ipValue = c.ClientIP
	}
	contextCipher, err := svc.S.Seal(sc, store.Map("contextId", contextID, "ip", ipValue))
	if err != nil {
		return nil, err
	}
	if err := svc.S.InsertChallenge(ctx, sc, id, kind, event["id"], partyID, nil, scope,
		targetCipher, svc.S.Crypto.Hash(id+":"+code), contextCipher, expires); err != nil {
		return nil, err
	}

	providerReady := svc.smsEnabled()
	if channel == "EMAIL" {
		providerReady = svc.emailEnabled()
	}
	if (sms || partyID != nil) && providerReady {
		payload, err := svc.S.Seal(sc, store.Map("challengeId", id, "contact", contact, "channel", channel, "code", code))
		if err != nil {
			return nil, err
		}
		if err := svc.S.InsertJob(ctx, sc, svc.S.ID(), event["id"], "EMAIL", "challenge:"+id, "PENDING",
			payload, 0, now, nil, nil, now, nil, expires); err != nil {
			return nil, err
		}
	}
	return store.Map("challengeId", id, "expiresAt", expires, "retryAfterSeconds", 60), nil
}

func (svc *Service) notificationVerify(ctx context.Context, sc *store.Scope, op string, r map[string]any, c *Req) (any, error) {
	id := store.StrOrEmpty(r["challengeId"])
	contextID := store.StrOrEmpty(c.Session["id"])
	sms := op == "SmsVerify"
	kind := "RECOVERY"
	if sms {
		kind = "SMS"
	}
	// A rejected outer business operation must not roll back the brute-force
	// counter: the attempt check/increment commits independently (REQUIRES_NEW).
	accepted := false
	if err := svc.independentTx(ctx, func(isc *store.Scope) error {
		candidates, err := svc.S.ChallengeByIDForUpdate(ctx, isc, id)
		if err != nil {
			return err
		}
		if len(candidates) == 0 {
			return nil
		}
		row := candidates[0]
		info, err := svc.S.Unseal(row["context_cipher"])
		if err != nil {
			return err
		}
		expires, err := store.Time(row["expires_at"])
		if err != nil {
			return fmt.Errorf("challenge expiry: %w", err)
		}
		if kind != store.StrOrEmpty(row["kind"]) ||
			contextID != store.StrOrEmpty(info["contextId"]) ||
			row["consumed_at"] != nil ||
			!expires.After(svc.S.Now()) ||
			store.Num(row["attempts"]) >= 5 {
			return nil
		}
		if sms && store.StrOrEmpty(row["event_id"]) != store.StrOrEmpty(r["eventId"]) {
			return nil
		}
		if err := svc.S.IncrementChallengeAttempts(ctx, isc, store.StrOrEmpty(row["id"])); err != nil {
			return err
		}
		digest, _ := row["code_digest"].([]byte)
		accepted = subtle.ConstantTimeCompare(digest, svc.S.Crypto.Hash(id+":"+store.StrOrEmpty(r["code"]))) == 1
		return nil
	}); err != nil {
		return nil, err
	}
	if err := errcode.Require(accepted, 422, "CHALLENGE_INVALID", "验证码无效、已过期或尝试次数过多"); err != nil {
		return nil, err
	}

	rows, err := svc.S.ChallengeByIDForUpdate(ctx, sc, id)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, errcode.New(404, "RESOURCE_NOT_FOUND", "未找到资源")
	}
	row := rows[0]
	if err := errcode.Require(row["consumed_at"] == nil, 422, "CHALLENGE_INVALID", "验证码已使用"); err != nil {
		return nil, err
	}
	event, err := svc.S.Event(ctx, sc, store.StrOrEmpty(row["event_id"]))
	if err != nil {
		return nil, err
	}
	now := svc.S.Now()
	if sms {
		due, err := store.Time(event["personal_due_at"])
		if err != nil {
			return nil, fmt.Errorf("event due time: %w", err)
		}
		expires := minTime(now.Add(10*time.Minute), due)
		target, err := svc.S.Unseal(row["target_cipher"])
		if err != nil {
			return nil, err
		}
		proof := store.Map(
			"kind", "PHONE_PROOF",
			"challengeId", id,
			"eventId", store.StrOrEmpty(event["id"]),
			"contextId", contextID,
			"phone", target["contact"],
			"expiresAt", store.FormatTime(expires),
		)
		proofContext, err := svc.S.Seal(sc, store.Map("contextId", contextID, "proofUsed", false, "proofExpiresAt", expires))
		if err != nil {
			return nil, err
		}
		if err := svc.S.ConsumeChallengeWithContext(ctx, sc, now, proofContext, store.StrOrEmpty(row["id"])); err != nil {
			return nil, err
		}
		signed, err := svc.S.Crypto.Signed(proof)
		if err != nil {
			return nil, err
		}
		return store.Map("verificationProof", signed, "expiresAt", expires), nil
	}

	if err := errcode.Require(row["party_id"] != nil, 422, "CHALLENGE_INVALID", "验证码无效"); err != nil {
		return nil, err
	}
	party, err := svc.partyRaw(ctx, sc, store.StrOrEmpty(event["id"]), store.StrOrEmpty(row["party_id"]))
	if err != nil {
		return nil, err
	}
	if err := svc.S.ConsumeChallenge(ctx, sc, now, store.StrOrEmpty(row["id"])); err != nil {
		return nil, err
	}
	if err := svc.S.RevokeAccessCredentials(ctx, sc, store.StrOrEmpty(party["id"])); err != nil {
		return nil, err
	}
	if err := svc.S.RevokePartySessions(ctx, sc, store.StrOrEmpty(party["id"])); err != nil {
		return nil, err
	}
	if _, err := svc.issueCapability(ctx, sc, party, "QUERY", event); err != nil {
		return nil, err
	}
	if _, err := svc.issueCapability(ctx, sc, party, "CHANGE", event); err != nil {
		return nil, err
	}
	return svc.partyAccess(ctx, sc, party, event)
}

// notificationWork is the claimed delivery unit (NotificationService.Work).
// errCode carries a claim-stage failure; the payload never leaves the worker.
type notificationWork struct {
	job       map[string]any
	recipient string
	subject   string
	text      string
	channel   string
	code      string
	deadline  time.Time
	errCode   string
}

// deliverNext processes one EMAIL job tick (NotificationService.deliverNext).
// Each database phase commits independently; the provider call runs outside
// any transaction or event lock.
func (svc *Service) deliverNext(ctx context.Context) error {
	if err := svc.independentTx(ctx, func(sc *store.Scope) error {
		if err := svc.S.ExpireJobs(ctx, sc, "EMAIL"); err != nil {
			return err
		}
		return svc.S.FailExhaustedJobs(ctx, sc, "EMAIL")
	}); err != nil {
		return err
	}
	var work *notificationWork
	if err := svc.independentTx(ctx, func(sc *store.Scope) error {
		w, err := svc.claimEmailWork(ctx, sc)
		if err != nil {
			return err
		}
		work = w
		return nil
	}); err != nil {
		return err
	}
	if work == nil {
		return nil
	}

	var sendErr error
	switch {
	case work.errCode != "":
		sendErr = errcode.New(503, "PROVIDER_UNAVAILABLE", "通知暂时不可用")
	case !work.deadline.After(svc.S.Now()):
		sendErr = errcode.New(410, "CHALLENGE_INVALID", "通知已过期")
	case work.channel == "SMS":
		sendErr = svc.sendSms(ctx, work.recipient, work.code)
	default:
		sendErr = svc.sendEmail(ctx, work.recipient, work.subject, work.text)
	}
	success := sendErr == nil
	failure := work.errCode
	if !success {
		failure = providerErrorCode(sendErr)
	}
	return svc.independentTx(ctx, func(sc *store.Scope) error {
		return svc.finishEmailWork(ctx, sc, work, success, failure)
	})
}

// claimEmailWork leases the next ready EMAIL job and prepares its delivery.
func (svc *Service) claimEmailWork(ctx context.Context, sc *store.Scope) (*notificationWork, error) {
	ready, err := svc.S.ClaimEmailJob(ctx, sc)
	if err != nil {
		return nil, err
	}
	if len(ready) == 0 {
		return nil, nil
	}
	current, err := svc.S.JobByIDForUpdate(ctx, sc, store.StrOrEmpty(ready[0]["id"]))
	if err != nil {
		return nil, err
	}
	now := svc.S.Now()
	pending := false
	if store.StrOrEmpty(current["status"]) == "PENDING" {
		nextRun, err := store.Time(current["next_run_at"])
		if err != nil {
			return nil, fmt.Errorf("job next run: %w", err)
		}
		pending = !nextRun.After(now)
	}
	abandoned := false
	if store.StrOrEmpty(current["status"]) == "RUNNING" && current["lease_until"] != nil {
		lease, err := store.Time(current["lease_until"])
		if err != nil {
			return nil, fmt.Errorf("job lease: %w", err)
		}
		abandoned = !lease.After(now)
	}
	expires, err := store.Time(current["expires_at"])
	if err != nil {
		return nil, fmt.Errorf("job expiry: %w", err)
	}
	if store.Num(current["attempts"]) >= 5 || (!pending && !abandoned) || !expires.After(now) {
		return nil, nil
	}
	if err := svc.S.LeaseJob(ctx, sc, now.Add(60*time.Second), store.StrOrEmpty(current["id"])); err != nil {
		return nil, err
	}
	job, err := svc.S.JobByID(ctx, sc, store.StrOrEmpty(current["id"]))
	if err != nil {
		return nil, err
	}
	work := &notificationWork{job: job, deadline: expires}
	if err := svc.buildEmailWork(ctx, sc, work); err != nil {
		work.errCode = providerErrorCode(err)
	}
	return work, nil
}

// buildEmailWork resolves the recipient/body of a claimed job; any failure is
// captured as the work error code instead of aborting the claim.
func (svc *Service) buildEmailWork(ctx context.Context, sc *store.Scope, work *notificationWork) error {
	event, err := svc.S.Event(ctx, sc, store.StrOrEmpty(work.job["event_id"]))
	if err != nil {
		return err
	}
	due, err := store.Time(event["personal_due_at"])
	if err != nil {
		return fmt.Errorf("event due time: %w", err)
	}
	work.deadline = minTime(work.deadline, due)
	payload, err := svc.S.Unseal(work.job["payload_cipher"])
	if err != nil {
		return err
	}
	channel := "EMAIL"
	if v, ok := payload["channel"]; ok && v != nil {
		channel = store.StrOrEmpty(v)
	}
	if challengeID, ok := payload["challengeId"]; ok {
		challenge, err := svc.S.ChallengeByID(ctx, sc, store.StrOrEmpty(challengeID))
		if err != nil {
			return err
		}
		challengeExpires, err := store.Time(challenge["expires_at"])
		if err != nil {
			return fmt.Errorf("challenge expiry: %w", err)
		}
		if err := errcode.Require(challenge["consumed_at"] == nil && challengeExpires.After(svc.S.Now()),
			410, "CHALLENGE_INVALID", "挑战已失效"); err != nil {
			return err
		}
		work.recipient = store.StrOrEmpty(payload["contact"])
		work.subject = "UniGate 验证码"
		work.code = store.StrOrEmpty(payload["code"])
		work.text = "您的验证码为 " + work.code + "，5分钟内有效。"
		work.channel = channel
		return nil
	}
	party, err := svc.partyRaw(ctx, sc, store.StrOrEmpty(event["id"]), store.StrOrEmpty(payload["partyId"]))
	if err != nil {
		return err
	}
	fields, err := svc.S.Unseal(party["fields_cipher"])
	if err != nil {
		return err
	}
	if fields["email"] == nil {
		return errcode.New(422, "EMAIL_REQUIRED", "未提供邮箱")
	}
	access, err := svc.partyAccess(ctx, sc, party, event)
	if err != nil {
		return err
	}
	base := svc.Cfg.PublicBaseURL
	if base == "" {
		base = svc.Cfg.AllowedOrigin
	}
	if err := errcode.Require(strings.HasPrefix(base, "https://") || svc.Cfg.Notifications.AllowLocalTest,
		503, "PROVIDER_UNAVAILABLE", "领票地址未配置"); err != nil {
		return err
	}
	work.recipient = store.StrOrEmpty(fields["email"])
	work.subject = "UniGate 活动报名凭证"
	work.text = "活动：" + store.StrOrEmpty(event["title"]) +
		"\n报名参考号：" + store.StrOrEmpty(party["party_reference"]) +
		"\n查看本组报名：" + partyLink(base, store.StrOrEmpty(access["queryToken"]), store.StrOrEmpty(access["changeToken"]))
	work.channel = "EMAIL"
	return nil
}

// finishEmailWork records the delivery outcome with the optimistic version
// guard. Delivery failures requeue or fail the job only — never eligibility.
func (svc *Service) finishEmailWork(ctx context.Context, sc *store.Scope, work *notificationWork, success bool, failure string) error {
	eventRows, err := svc.S.EventStatusDue(ctx, sc, store.StrOrEmpty(work.job["event_id"]))
	if err != nil {
		return err
	}
	if len(eventRows) == 0 {
		return nil
	}
	jobs, err := svc.S.JobByIDForUpdateRows(ctx, sc, store.StrOrEmpty(work.job["id"]))
	if err != nil {
		return err
	}
	if len(jobs) == 0 {
		return nil
	}
	job := jobs[0]
	if store.StrOrEmpty(job["status"]) != "RUNNING" || store.Num(job["version"]) != store.Num(work.job["version"]) {
		return nil
	}
	now := svc.S.Now()
	expired := !work.deadline.After(now) || store.StrOrEmpty(eventRows[0]["status"]) == "ARCHIVED"
	result := "PENDING"
	switch {
	case expired:
		result = "EXPIRED"
	case success:
		result = "SUCCEEDED"
	case store.Num(job["attempts"]) >= 5:
		result = "FAILED"
	}
	delay := time.Duration(1<<min(3, max(0, store.Num(job["attempts"])-1))) * time.Minute
	var errorCode any
	if !success {
		errorCode = failure
	}
	var completedAt any
	if result == "SUCCEEDED" || result == "FAILED" || result == "EXPIRED" {
		completedAt = now
	}
	clearPayload := result == "SUCCEEDED" || result == "EXPIRED"
	return svc.S.FinishJob(ctx, sc, result, errorCode, now.Add(delay), completedAt, clearPayload,
		store.StrOrEmpty(job["id"]), job["version"])
}

// sendEmail delivers via SMTP with credentials from configuration only.
func (svc *Service) sendEmail(ctx context.Context, recipient, subject, text string) error {
	if err := svc.requireEmail(); err != nil {
		return err
	}
	n := svc.Cfg.Notifications
	if strings.TrimSpace(n.From) == "" {
		return errcode.New(503, "PROVIDER_UNAVAILABLE", "发信地址未配置")
	}
	addr := net.JoinHostPort(n.MailHost, strconv.Itoa(n.MailPort))
	dialer := net.Dialer{Timeout: 8 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer conn.Close()
	client, err := smtp.NewClient(conn, n.MailHost)
	if err != nil {
		return fmt.Errorf("smtp handshake: %w", err)
	}
	defer client.Quit()
	// Java mail.smtp.connectiontimeout/timeout/writetimeout are 8s each.
	deadline := func() error {
		if err := conn.SetDeadline(time.Now().Add(8 * time.Second)); err != nil {
			return fmt.Errorf("smtp deadline: %w", err)
		}
		return nil
	}
	if err := deadline(); err != nil {
		return err
	}
	if n.SMTPStartTLS {
		if err := client.StartTLS(&tls.Config{ServerName: n.MailHost}); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}
	if n.SMTPAuth {
		if err := deadline(); err != nil {
			return err
		}
		if err := client.Auth(smtp.PlainAuth("", n.MailUsername, n.MailPassword, n.MailHost)); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := deadline(); err != nil {
		return err
	}
	if err := client.Mail(n.From); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(recipient); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}
	if err := deadline(); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	msg := "From: " + n.From + "\r\n" +
		"To: " + recipient + "\r\n" +
		"Subject: =?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(subject)) + "?=\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"Content-Transfer-Encoding: 8bit\r\n" +
		"\r\n" + text
	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("smtp write message: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp finish message: %w", err)
	}
	return nil
}

// sendSms posts the code to the configured gateway (allow-listed HTTPS host).
func (svc *Service) sendSms(ctx context.Context, phone, code string) error {
	if err := errcode.Require(svc.smsEnabled(), 503, "PROVIDER_UNAVAILABLE", "短信服务未启用"); err != nil {
		return err
	}
	n := svc.Cfg.Notifications
	endpoint, err := url.Parse(n.SMSEndpoint)
	if err != nil {
		return errcode.New(503, "PROVIDER_UNAVAILABLE", "无法连接短信网关")
	}
	allowed := false
	for _, h := range n.SMSAllowedHosts {
		if h == endpoint.Hostname() {
			allowed = true
			break
		}
	}
	if err := errcode.Require(endpoint.Scheme == "https" && endpoint.User == nil && allowed,
		503, "PROVIDER_UNAVAILABLE", "短信网关不在部署授权范围内"); err != nil {
		return err
	}
	body, err := store.Stringify(store.Map("phone", phone, "code", code, "expiresInSeconds", 300))
	if err != nil {
		return err
	}
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, n.SMSEndpoint, strings.NewReader(body))
	if err != nil {
		return errcode.New(503, "PROVIDER_UNAVAILABLE", "无法连接短信网关")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+n.SMSToken)
	resp, err := svc.HTTP.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return errcode.New(503, "PROVIDER_UNAVAILABLE", "短信请求中断")
		}
		return errcode.New(503, "PROVIDER_UNAVAILABLE", "无法连接短信网关")
	}
	defer resp.Body.Close()
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return errcode.New(503, "PROVIDER_UNAVAILABLE", "无法连接短信网关")
	}
	if err := errcode.Require(resp.StatusCode >= 200 && resp.StatusCode < 300,
		503, "PROVIDER_UNAVAILABLE", "短信网关拒绝请求"); err != nil {
		return err
	}
	return nil
}

// partyLink builds the party self-service URL with encoded capability tokens.
func partyLink(base, query, change string) string {
	return strings.TrimRight(base, "/") + "/me/party#queryToken=" + url.QueryEscape(query) + "&changeToken=" + url.QueryEscape(change)
}

// normalizeContact lowercases the email domain or strips phone formatting.
func normalizeContact(contact, channel string) string {
	value := strings.TrimSpace(contact)
	if channel == "EMAIL" {
		at := strings.LastIndex(value, "@")
		if at < 0 {
			return value
		}
		return value[:at+1] + strings.ToLower(value[at+1:])
	}
	return stripPhone(value)
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

// providerErrorCode maps a failure to its canonical business code, defaulting
// to PROVIDER_UNAVAILABLE for transport/unknown errors (ApiException handling).
func providerErrorCode(err error) string {
	if e := errcode.AsError(err); e != nil {
		return e.Code
	}
	return "PROVIDER_UNAVAILABLE"
}

// pollInterval converts a configured poll period with a safe fallback.
func pollInterval(ms, fallback int) time.Duration {
	if ms <= 0 {
		ms = fallback
	}
	return time.Duration(ms) * time.Millisecond
}

// NotificationWorker drives deliverNext on a fixed delay. Run blocks until ctx
// is cancelled; main owns the goroutine (no bare go func in the service layer).
type NotificationWorker struct {
	svc *Service
}

// NewNotificationWorker wires the delivery worker.
func NewNotificationWorker(svc *Service) *NotificationWorker {
	return &NotificationWorker{svc: svc}
}

// Run polls until ctx is cancelled (Spring @Scheduled fixedDelay equivalent).
func (w *NotificationWorker) Run(ctx context.Context) {
	runPollLoop(ctx, "notification worker", pollInterval(w.svc.Cfg.NotificationsPollMS, 3000), w.svc.deliverNext)
}

// DeliverNext processes a single EMAIL job tick. Run polls it on a fixed delay;
// integration tests drive it directly like the Java deliverNext() call.
func (w *NotificationWorker) DeliverNext(ctx context.Context) error {
	return w.svc.deliverNext(ctx)
}
