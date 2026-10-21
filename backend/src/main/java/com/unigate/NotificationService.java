package com.unigate;

import static com.unigate.Store.*;

import java.net.*;
import java.net.http.*;
import java.security.*;
import java.time.*;
import java.util.*;
import org.springframework.core.env.Environment;
import org.springframework.mail.SimpleMailMessage;
import org.springframework.mail.javamail.JavaMailSenderImpl;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Service;
import org.springframework.transaction.PlatformTransactionManager;
import org.springframework.transaction.TransactionDefinition;
import org.springframework.transaction.support.TransactionTemplate;

/** Opt-in delivery worker. Queue payloads are encrypted and never returned to clients or logs. */
@Service
public class NotificationService {
  private final Store s;
  private final PartyService parties;
  private final Environment env;
  private final TransactionTemplate securityTx;
  private final SecureRandom random = new SecureRandom();

  public NotificationService(
      Store s, PartyService parties, Environment env, PlatformTransactionManager tx) {
    this.s = s;
    this.parties = parties;
    this.env = env;
    securityTx = new TransactionTemplate(tx);
    securityTx.setPropagationBehavior(TransactionDefinition.PROPAGATION_REQUIRES_NEW);
  }

  private boolean emailEnabled() {
    return env.getProperty("unigate.notifications.email-enabled", Boolean.class, false)
        && !env.getProperty("spring.mail.host", "").isBlank();
  }

  public void requireEmail() {
    ApiException.require(emailEnabled(), 503, "PROVIDER_UNAVAILABLE", "邮件服务尚未启用");
  }

  private boolean smsEnabled() {
    return env.getProperty("unigate.notifications.sms-enabled", Boolean.class, false)
        && !env.getProperty("unigate.notifications.sms-endpoint", "").isBlank();
  }

  public Object handle(String op, Map<String, Object> r, Context c) {
    return op.endsWith("Start") ? start(op, r, c) : verify(op, r, c);
  }

  private Object start(String op, Map<String, Object> r, Context c) {
    boolean sms = op.equals("SmsStart");
    String eventId = str(r.get("eventId"));
    var event = s.event(eventId);
    String contact = str(r.get(sms ? "phone" : "contact")).strip();
    String channel = sms ? "SMS" : str(r.get("channel"));
    contact = normalizeContact(contact, channel);
    if (sms) {
      ApiException.require(
          Boolean.TRUE.equals(s.parse(event.get("policy")).get("phoneVerificationEnabled")),
          403,
          "VERIFICATION_DISABLED",
          "此活动未启用手机验证");
      ApiException.require(smsEnabled(), 503, "PROVIDER_UNAVAILABLE", "短信验证服务尚未配置");
    }
    String kind = sms ? "SMS" : "RECOVERY",
        scope = kind + ":" + HexFormat.of().formatHex(s.crypto.hash(contact)),
        context = str(c.session.get("id"));
    var recent =
        s.rows(
            "select * from ug_challenge where expires_at>now()-interval '10 minutes' and kind in"
                + " ('SMS','RECOVERY')");
    int clientCount = 0, ipCount = 0;
    for (var old : recent) {
      var info = s.unseal(old.get("context_cipher"));
      if (context.equals(info.get("contextId"))) clientCount++;
      if (c.clientIp != null && c.clientIp.equals(info.get("ip"))) ipCount++;
    }
    ApiException.require(clientCount < 10 && ipCount < 30, 429, "RATE_LIMITED", "验证申请过多，请稍后重试");
    ApiException.require(
        s.rows(
                "select id from ug_challenge where actor_scope=? and expires_at>now()+interval '4"
                    + " minutes'",
                scope)
            .isEmpty(),
        429,
        "RATE_LIMITED",
        "请等待60秒后重试");
    String id = s.id(), code = String.format(Locale.ROOT, "%06d", random.nextInt(1000000));
    var expires = min(s.now().plusMinutes(5), time(event.get("personal_due_at")));
    Object partyId = null;
    if (!sms) {
      var candidates =
          s.rows(
              "select * from ug_party where event_id=? and party_reference=?",
              event.get("id"),
              r.get("partyReference"));
      if (!candidates.isEmpty()) {
        var candidate = candidates.getFirst();
        var fields = s.unseal(candidate.get("fields_cipher"));
        String field = channel.equals("EMAIL") ? "email" : "phone";
        if (fields.get(field) != null
            && contact.equals(normalizeContact(str(fields.get(field)), channel)))
          partyId = candidate.get("id");
      }
    }
    s.execute(
        "insert into"
            + " ug_challenge(id,kind,event_id,party_id,actor_scope,target_cipher,code_digest,context_cipher,expires_at)"
            + " values(?,?,?,?,?,?,?,?,?)",
        uuid(id),
        kind,
        event.get("id"),
        partyId,
        scope,
        s.seal(map("contact", contact, "channel", channel)),
        s.crypto.hash(id + ":" + code),
        s.seal(map("contextId", context, "ip", c.clientIp)),
        expires);
    if ((sms || partyId != null) && (channel.equals("EMAIL") ? emailEnabled() : smsEnabled())) {
      s.execute(
          "insert into"
              + " ug_job(id,event_id,kind,business_key,status,payload_cipher,next_run_at,created_at,expires_at)"
              + " values(?,?,'EMAIL',?,'PENDING',?,now(),now(),?)",
          uuid(s.id()),
          event.get("id"),
          "challenge:" + id,
          s.seal(map("challengeId", id, "contact", contact, "channel", channel, "code", code)),
          expires);
    }
    return map("challengeId", id, "expiresAt", expires, "retryAfterSeconds", 60);
  }

  private Object verify(String op, Map<String, Object> r, Context c) {
    String id = str(r.get("challengeId")), context = str(c.session.get("id"));
    boolean sms = op.equals("SmsVerify");
    String kind = sms ? "SMS" : "RECOVERY";
    // A rejected outer business operation must not roll back the brute-force counter.
    boolean accepted =
        Boolean.TRUE.equals(
            securityTx.execute(
                status -> {
                  var candidates =
                      s.rows("select * from ug_challenge where id=? for update", uuid(id));
                  if (candidates.isEmpty()) return false;
                  var row = candidates.getFirst();
                  var info = s.unseal(row.get("context_cipher"));
                  if (!kind.equals(row.get("kind"))
                      || !context.equals(info.get("contextId"))
                      || row.get("consumed_at") != null
                      || !time(row.get("expires_at")).isAfter(s.now())
                      || num(row.get("attempts")) >= 5) return false;
                  if (sms && !str(row.get("event_id")).equals(str(r.get("eventId")))) return false;
                  s.execute(
                      "update ug_challenge set attempts=attempts+1 where id=?", row.get("id"));
                  return MessageDigest.isEqual(
                      (byte[]) row.get("code_digest"), s.crypto.hash(id + ":" + r.get("code")));
                }));
    ApiException.require(accepted, 422, "CHALLENGE_INVALID", "验证码无效、已过期或尝试次数过多");
    var row = s.row("select * from ug_challenge where id=? for update", uuid(id));
    ApiException.require(row.get("consumed_at") == null, 422, "CHALLENGE_INVALID", "验证码已使用");
    var event = s.event(str(row.get("event_id")));
    if (sms) {
      var expires = min(s.now().plusMinutes(10), time(event.get("personal_due_at")));
      var proof =
          map(
              "kind",
              "PHONE_PROOF",
              "challengeId",
              id,
              "eventId",
              str(event.get("id")),
              "contextId",
              context,
              "phone",
              s.unseal(row.get("target_cipher")).get("contact"),
              "expiresAt",
              expires);
      s.execute(
          "update ug_challenge set consumed_at=?,context_cipher=? where id=?",
          s.now(),
          s.seal(map("contextId", context, "proofUsed", false, "proofExpiresAt", expires)),
          row.get("id"));
      return map("verificationProof", s.crypto.signed(proof, s.json), "expiresAt", expires);
    }
    ApiException.require(row.get("party_id") != null, 422, "CHALLENGE_INVALID", "验证码无效");
    var party = parties.raw(str(event.get("id")), str(row.get("party_id")));
    s.execute("update ug_challenge set consumed_at=? where id=?", s.now(), row.get("id"));
    s.execute(
        "update ug_credential set status='REVOKED' where party_id=? and kind in ('QUERY','CHANGE')"
            + " and status='ACTIVE'",
        party.get("id"));
    s.execute("update ug_session set revoked_at=? where party_id=?", s.now(), party.get("id"));
    parties.issueCapability(party, "QUERY", event);
    parties.issueCapability(party, "CHANGE", event);
    return parties.access(party, event);
  }

  private record Work(
      Map<String, Object> job,
      String recipient,
      String subject,
      String text,
      String channel,
      String code,
      OffsetDateTime deadline,
      String error) {}

  @Scheduled(fixedDelayString = "${unigate.notifications.poll-ms:3000}")
  public void deliverNext() {
    try {
      securityTx.executeWithoutResult(
          status -> {
            s.execute(
                "update ug_job set"
                    + " status='EXPIRED',payload_cipher=null,lease_until=null,completed_at=now(),version=version+1"
                    + " where kind='EMAIL' and status in ('PENDING','RUNNING') and"
                    + " expires_at<=now() and (lease_until is null or lease_until<=now())");
            s.execute(
                "update ug_job set"
                    + " status='FAILED',lease_until=null,error_code='PROVIDER_UNAVAILABLE',completed_at=now(),version=version+1"
                    + " where kind='EMAIL' and status='RUNNING' and lease_until<=now() and"
                    + " attempts>=5");
          });
      Work work = securityTx.execute(status -> claim());
      if (work == null) return;
      boolean success = false;
      String error = work.error();
      try {
        ApiException.require(error == null, 503, "PROVIDER_UNAVAILABLE", "通知暂时不可用");
        ApiException.require(work.deadline().isAfter(s.now()), 410, "CHALLENGE_INVALID", "通知已过期");
        if ("SMS".equals(work.channel())) sendSms(work.recipient(), work.code());
        else sendEmail(work.recipient(), work.subject(), work.text());
        success = true;
      } catch (RuntimeException ex) {
        error = ex instanceof ApiException a ? a.code : "PROVIDER_UNAVAILABLE";
      }
      final boolean delivered = success;
      final String failure = error;
      securityTx.executeWithoutResult(status -> finish(work, delivered, failure));
    } finally {
      s.eventScope.remove();
    }
  }

  private Work claim() {
    var rows =
        s.rows(
            "select j.* from ug_job j join ug_event e on j.event_id=e.id where j.kind='EMAIL' and"
                + " ((j.status='PENDING' and j.next_run_at<=now()) or (j.status='RUNNING' and"
                + " j.lease_until<=now())) and j.attempts<5 and j.expires_at>now() and"
                + " e.personal_due_at>now() and e.status<>'ARCHIVED' order by j.next_run_at,j.id"
                + " for update of e skip locked limit 1");
    if (rows.isEmpty()) return null;
    var current = s.row("select * from ug_job where id=? for update", rows.getFirst().get("id"));
    boolean ready =
        "PENDING".equals(current.get("status"))
            && !time(current.get("next_run_at")).isAfter(s.now());
    boolean abandoned =
        "RUNNING".equals(current.get("status"))
            && current.get("lease_until") != null
            && !time(current.get("lease_until")).isAfter(s.now());
    if (num(current.get("attempts")) >= 5
        || (!ready && !abandoned)
        || !time(current.get("expires_at")).isAfter(s.now())) return null;
    s.execute(
        "update ug_job set status='RUNNING',attempts=attempts+1,lease_until=?,version=version+1"
            + " where id=?",
        s.now().plusSeconds(60),
        current.get("id"));
    var job = s.row("select * from ug_job where id=?", current.get("id"));
    var deadline = time(job.get("expires_at"));
    try {
      var event = s.event(str(job.get("event_id")));
      deadline = min(deadline, time(event.get("personal_due_at")));
      var payload = s.unseal(job.get("payload_cipher"));
      String channel = str(payload.getOrDefault("channel", "EMAIL"));
      if (payload.containsKey("challengeId")) {
        var challenge =
            s.row("select * from ug_challenge where id=?", uuid(payload.get("challengeId")));
        ApiException.require(
            challenge.get("consumed_at") == null
                && time(challenge.get("expires_at")).isAfter(s.now()),
            410,
            "CHALLENGE_INVALID",
            "挑战已失效");
        return new Work(
            job,
            str(payload.get("contact")),
            "UniGate 验证码",
            "您的验证码为 " + payload.get("code") + "，5分钟内有效。",
            channel,
            str(payload.get("code")),
            deadline,
            null);
      }
      var party = parties.raw(str(event.get("id")), str(payload.get("partyId")));
      var fields = s.unseal(party.get("fields_cipher"));
      String recipient = str(fields.get("email"));
      ApiException.require(recipient != null, 422, "EMAIL_REQUIRED", "未提供邮箱");
      var access = parties.access(party, event);
      String base =
          env.getProperty("unigate.public-base-url", env.getProperty("unigate.allowed-origin", ""));
      ApiException.require(
          base.startsWith("https://")
              || env.getProperty("unigate.notifications.allow-local-test", Boolean.class, false),
          503,
          "PROVIDER_UNAVAILABLE",
          "领票地址未配置");
      String text =
          "活动："
              + event.get("title")
              + "\n报名参考号："
              + party.get("party_reference")
              + "\n查看本组报名："
              + partyLink(base, str(access.get("queryToken")), str(access.get("changeToken")));
      return new Work(job, recipient, "UniGate 活动报名凭证", text, "EMAIL", null, deadline, null);
    } catch (RuntimeException ex) {
      return new Work(
          job,
          null,
          null,
          null,
          null,
          null,
          deadline,
          ex instanceof ApiException a ? a.code : "PROVIDER_UNAVAILABLE");
    }
  }

  private void finish(Work work, boolean success, String error) {
    var eventRows =
        s.rows(
            "select status,personal_due_at from ug_event where id=? for update",
            work.job().get("event_id"));
    if (eventRows.isEmpty()) return;
    var existing = s.rows("select * from ug_job where id=? for update", work.job().get("id"));
    if (existing.isEmpty()) return;
    var job = existing.getFirst();
    if (!"RUNNING".equals(job.get("status"))
        || num(job.get("version")) != num(work.job().get("version"))) return;
    boolean expired =
        !work.deadline().isAfter(s.now()) || "ARCHIVED".equals(eventRows.getFirst().get("status"));
    String result =
        expired
            ? "EXPIRED"
            : success ? "SUCCEEDED" : num(job.get("attempts")) >= 5 ? "FAILED" : "PENDING";
    long delay = 1L << Math.min(3, Math.max(0, num(job.get("attempts")) - 1));
    s.execute(
        "update ug_job set"
            + " status=?,error_code=?,lease_until=null,next_run_at=?,completed_at=?,payload_cipher=case"
            + " when ? then null else payload_cipher end,version=version+1 where id=? and"
            + " version=?",
        result,
        success ? null : error,
        s.now().plusMinutes(delay),
        Set.of("SUCCEEDED", "FAILED", "EXPIRED").contains(result) ? s.now() : null,
        Set.of("SUCCEEDED", "EXPIRED").contains(result),
        job.get("id"),
        job.get("version"));
  }

  private void sendEmail(String recipient, String subject, String text) {
    requireEmail();
    var sender = new JavaMailSenderImpl();
    sender.setDefaultEncoding("UTF-8");
    sender.setHost(env.getProperty("spring.mail.host"));
    sender.setPort(env.getProperty("spring.mail.port", Integer.class, 25));
    sender.setUsername(env.getProperty("spring.mail.username", ""));
    sender.setPassword(env.getProperty("spring.mail.password", ""));
    var props = sender.getJavaMailProperties();
    props.put("mail.smtp.auth", env.getProperty("spring.mail.properties.mail.smtp.auth", "false"));
    props.put(
        "mail.smtp.starttls.enable",
        env.getProperty("spring.mail.properties.mail.smtp.starttls.enable", "true"));
    props.put("mail.smtp.connectiontimeout", "8000");
    props.put("mail.smtp.timeout", "8000");
    props.put("mail.smtp.writetimeout", "8000");
    String from = env.getProperty("unigate.notifications.from", "");
    ApiException.require(!from.isBlank(), 503, "PROVIDER_UNAVAILABLE", "发信地址未配置");
    var message = new SimpleMailMessage();
    message.setFrom(from);
    message.setTo(recipient);
    message.setSubject(subject);
    message.setText(text);
    sender.send(message);
  }

  private void sendSms(String phone, String code) {
    ApiException.require(smsEnabled(), 503, "PROVIDER_UNAVAILABLE", "短信服务未启用");
    try {
      URI endpoint = URI.create(env.getProperty("unigate.notifications.sms-endpoint"));
      Set<String> allowed =
          new HashSet<>(
              Arrays.asList(
                  env.getProperty("unigate.notifications.sms-allowed-hosts", "").split(",")));
      ApiException.require(
          "https".equals(endpoint.getScheme())
              && endpoint.getUserInfo() == null
              && allowed.contains(endpoint.getHost()),
          503,
          "PROVIDER_UNAVAILABLE",
          "短信网关不在部署授权范围内");
      var request =
          HttpRequest.newBuilder(endpoint)
              .timeout(Duration.ofSeconds(10))
              .header("Content-Type", "application/json")
              .header(
                  "Authorization",
                  "Bearer " + env.getProperty("unigate.notifications.sms-token", ""))
              .POST(
                  HttpRequest.BodyPublishers.ofString(
                      s.stringify(map("phone", phone, "code", code, "expiresInSeconds", 300))))
              .build();
      var result =
          HttpClient.newBuilder()
              .followRedirects(HttpClient.Redirect.NEVER)
              .connectTimeout(Duration.ofSeconds(5))
              .build()
              .send(request, HttpResponse.BodyHandlers.discarding());
      ApiException.require(
          result.statusCode() >= 200 && result.statusCode() < 300,
          503,
          "PROVIDER_UNAVAILABLE",
          "短信网关拒绝请求");
    } catch (ApiException ex) {
      throw ex;
    } catch (InterruptedException ex) {
      Thread.currentThread().interrupt();
      throw new ApiException(503, "PROVIDER_UNAVAILABLE", "短信请求中断");
    } catch (Exception ex) {
      throw new ApiException(503, "PROVIDER_UNAVAILABLE", "无法连接短信网关");
    }
  }

  static String partyLink(String base, String query, String change) {
    return base.replaceAll("/+$", "")
        + "/me/party#queryToken="
        + encode(query)
        + "&changeToken="
        + encode(change);
  }

  static String normalizeContact(String contact, String channel) {
    String value = contact.strip();
    if ("EMAIL".equals(channel)) {
      int at = value.lastIndexOf('@');
      return at < 0
          ? value
          : value.substring(0, at + 1) + value.substring(at + 1).toLowerCase(Locale.ROOT);
    }
    return value.replaceAll("[\\s()\\-]", "");
  }

  private static OffsetDateTime min(OffsetDateTime a, OffsetDateTime b) {
    return a.isBefore(b) ? a : b;
  }

  private static String encode(String value) {
    return URLEncoder.encode(value, java.nio.charset.StandardCharsets.UTF_8);
  }
}
