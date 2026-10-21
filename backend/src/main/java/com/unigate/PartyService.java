package com.unigate;

import static com.unigate.Store.*;

import java.time.*;
import java.util.*;
import org.springframework.stereotype.Service;

@Service
public class PartyService {
  final Store s;
  final EventService events;
  final IdentityService identity;

  public PartyService(Store s, EventService events, IdentityService identity) {
    this.s = s;
    this.events = events;
    this.identity = identity;
  }

  public Map<String, Object> raw(String eventId, String partyId) {
    return s.row(
        "select * from ug_party where event_id=? and id=? for update",
        uuid(eventId),
        uuid(partyId));
  }

  public Map<String, Object> readRaw(String eventId, String partyId) {
    return raw(eventId, partyId);
  }

  public String progress(Map p) {
    int approved = num(p.get("approved_size")), n = num(p.get("net_admitted"));
    return n == 0 ? "NONE" : n == approved ? "COMPLETE" : "PARTIAL";
  }

  public Map<String, Object> view(Map p, Set<String> allowed) {
    Map<String, Object> fields = s.unseal(p.get("fields_cipher"));
    fields.keySet().retainAll(allowed);
    return map(
        "partyId",
        str(p.get("id")),
        "partyReference",
        p.get("party_reference"),
        "eventId",
        str(p.get("event_id")),
        "fields",
        fields,
        "requestedSize",
        p.get("requested_size"),
        "approvedSize",
        p.get("approved_size"),
        "pendingIncrease",
        p.get("pending_increase"),
        "netAdmitted",
        p.get("net_admitted"),
        "remainingCount",
        num(p.get("approved_size")) - num(p.get("net_admitted")),
        "status",
        s.value("PartyStatus", p.get("status")),
        "progress",
        s.value("AdmissionProgress", progress(p)),
        "sourceType",
        s.value("SourceType", p.get("source_type")),
        "version",
        p.get("version"),
        "createdAt",
        p.get("created_at"),
        "updatedAt",
        p.get("updated_at"),
        "hasAdmissionHistory",
        p.get("ever_admitted"));
  }

  public Set<String> allFields() {
    return Set.of("name", "phone", "email", "companionCount");
  }

  public void validateFields(Map e, Map fields) {
    Map rules = (Map) s.parse(e.get("policy")).get("fields");
    for (Object key : rules.keySet()) {
      Map rule = (Map) rules.get(key);
      ApiException.require(
          !fields.containsKey(key) || Boolean.TRUE.equals(rule.get("enabled")),
          422,
          "FIELD_DISABLED",
          "未启用字段不能提交: " + key);
      ApiException.require(
          !Boolean.TRUE.equals(rule.get("required")) || fields.containsKey(key),
          422,
          "VALIDATION_FAILED",
          "缺少必填字段: " + key);
    }
    for (Object key : new ArrayList<>(fields.keySet()))
      if (fields.get(key) instanceof String text) fields.put(key, text.trim());
    for (Object value : fields.values())
      if (value instanceof String text)
        ApiException.require(!text.isBlank(), 422, "FIELD_VALIDATION_FAILED", "字段不能为空白");
  }

  public void allocate(Map e, int delta) {
    int changed =
        s.execute(
            "update ug_event set allocated_count=allocated_count+?,version=version+1 where id=? and"
                + " allocated_count+? between 0 and capacity",
            delta,
            e.get("id"),
            delta);
    ApiException.require(changed == 1, 409, "CAPACITY_CONFLICT", "可用名额不足");
    e.put("allocated_count", num(e.get("allocated_count")) + delta);
  }

  public Map<String, Object> create(
      Context ctx,
      Map event,
      Map fields,
      String sourceType,
      boolean approved,
      String noticeVersion,
      UUID entry) {
    validateFields(event, fields);
    int size = 1 + num(fields.get("companionCount"));
    if (approved) allocate(event, size);
    String id = s.id(), ref = "UG-" + id.substring(0, 8).toUpperCase(Locale.ROOT);
    s.execute(
        "insert into"
            + " ug_party(id,event_id,party_reference,fields_cipher,data_key_ref,name_search_hmac,phone_search_hmac,email_search_hmac,requested_size,approved_size,status,source_type,accepted_notice_version)"
            + " values(?,?,?,?,?,?,?,?,?,?,?,?,?)",
        uuid(id),
        event.get("id"),
        ref,
        s.seal(fields),
        "event:" + event.get("id") + ":v1",
        hmac(fields.get("name")),
        hmac(fields.get("phone")),
        hmac(fields.get("email")),
        size,
        approved ? size : 0,
        approved ? "APPROVED" : "PENDING",
        sourceType,
        noticeVersion);
    Map p = raw(str(event.get("id")), id);
    issueCapability(p, "QUERY", event);
    issueCapability(p, "CHANGE", event);
    if (approved) issueCapability(p, "TICKET", event);
    else review(p, "INITIAL", size, size);
    enqueueEmail(p, event);
    return view(p, allFields());
  }

  byte[] hmac(Object o) {
    return o == null ? null : s.crypto.hash(o.toString().trim());
  }

  public void enqueueEmail(Map p, Map e) {
    Map f = s.unseal(p.get("fields_cipher"));
    if (!f.containsKey("email")) return;
    String id = s.id();
    s.execute(
        "insert into"
            + " ug_job(id,event_id,kind,business_key,status,payload_cipher,next_run_at,created_at,expires_at)"
            + " values(?,?,'EMAIL',?,'PENDING',?,now(),now(),?) on conflict(kind,business_key) do"
            + " nothing",
        uuid(id),
        e.get("id"),
        "party:" + p.get("id") + ":" + p.get("version"),
        s.seal(map("partyId", str(p.get("id")), "email", f.get("email"))),
        e.get("personal_due_at"));
  }

  String issueCapability(Map p, String kind, Map e) {
    String token = s.crypto.token();
    s.execute(
        "insert into"
            + " ug_credential(id,event_id,party_id,kind,token_digest,token_cipher,credential_version,status,expires_at)"
            + " values(?,?,?,?,?,?,?,'ACTIVE',?)",
        uuid(s.id()),
        p.get("event_id"),
        p.get("id"),
        kind,
        s.crypto.hash(token),
        s.seal(map("token", token)),
        p.get("version"),
        e.get("personal_due_at"));
    return token;
  }

  public Map<String, Object> access(Map p, Map e) {
    Map out = map("expiresAt", e.get("personal_due_at"));
    for (var cred :
        s.rows(
            "select * from ug_credential where party_id=? and kind in ('QUERY','CHANGE') and"
                + " status='ACTIVE' and expires_at>now()",
            p.get("id")))
      out.put(
          "QUERY".equals(cred.get("kind")) ? "queryToken" : "changeToken",
          s.unseal(cred.get("token_cipher")).get("token"));
    return out;
  }

  Map<String, Object> credential(Map p, Map e) {
    var creds =
        s.rows(
            "select * from ug_credential where party_id=? and kind='TICKET' order by created_at"
                + " desc,id desc",
            p.get("id"));
    if (creds.isEmpty())
      return map(
          "partyId",
          str(p.get("id")),
          "status",
          s.value("CredentialStatus", "REVOKED"),
          "credentialVersion",
          p.get("version"),
          "expiresAt",
          e.get("personal_due_at"));
    Map row = creds.getFirst();
    Map out =
        map(
            "partyId",
            str(p.get("id")),
            "status",
            s.value("CredentialStatus", row.get("status")),
            "credentialVersion",
            row.get("credential_version"),
            "expiresAt",
            row.get("expires_at"));
    if ("ACTIVE".equals(row.get("status")) && "APPROVED".equals(p.get("status")))
      out.put("ticketToken", s.unseal(row.get("token_cipher")).get("token"));
    return out;
  }

  void review(Map p, String kind, int requested, int delta) {
    s.execute(
        "insert into"
            + " ug_party_review(id,event_id,party_id,kind,requested_size,requested_increase,result,created_at)"
            + " values(?,?,?,?,?,?,'PENDING',now())",
        uuid(s.id()),
        p.get("event_id"),
        p.get("id"),
        kind,
        requested,
        delta);
  }

  boolean canSelf(Map p, Map e, String key) {
    Map policy = s.parse(e.get("policy"));
    Object deadline = policy.getOrDefault("selfServiceEndsAt", e.get("registration_ends_at"));
    return Boolean.TRUE.equals(policy.get(key))
        && "READY".equals(e.get("status"))
        && !Boolean.TRUE.equals(p.get("ever_admitted"))
        && s.now().isBefore(time(deadline))
        && List.of("APPROVED", "PENDING").contains(p.get("status"));
  }

  Map<String, Object> state(Map p, Map e) {
    Map out = view(p, allFields());
    out.keySet()
        .removeAll(
            Set.of("requestedSize", "sourceType", "createdAt", "updatedAt", "hasAdmissionHistory"));
    boolean companion =
        Boolean.TRUE.equals(
            ((Map) ((Map) s.parse(e.get("policy")).get("fields")).get("companionCount"))
                .get("enabled"));
    out.put("canChange", companion && canSelf(p, e, "selfEditEnabled"));
    out.put("canCancel", canSelf(p, e, "selfCancelEnabled"));
    out.put("expiresAt", e.get("personal_due_at"));
    return out;
  }

  void ready(Map e) {
    ApiException.require(
        "READY".equals(e.get("status")), 409, "EVENT_PHASE_CONFLICT", "现场开始后普通变更已冻结");
  }

  public Object handle(String op, Map<String, Object> r, Context c, UUID entry) {
    String ev = str(r.get("eventId"));
    if ((op.startsWith("My") && !op.equals("MyOperation")) || op.equals("MyOperation"))
      ev = str(c.session.get("event_id"));
    switch (op) {
      case "SubmitRegistration":
        {
          Map e = s.event(ev);
          ApiException.require(
              events.registrationAvailable(e), 409, "REGISTRATION_CLOSED", "报名时间已关闭");
          ApiException.require(
              events.noticeVersion(e).equals(r.get("acceptedNoticeVersion")),
              409,
              "NOTICE_CHANGED",
              "活动告知已更新，请重新确认");
          Map policy = s.parse(e.get("policy"));
          if (Boolean.TRUE.equals(policy.get("phoneVerificationEnabled"))) verifyPhone(r, c, e);
          Map v =
              create(
                  c,
                  e,
                  (Map) r.get("fields"),
                  "BUILTIN",
                  "IMMEDIATE".equals(policy.get("approvalMode")),
                  str(r.get("acceptedNoticeVersion")),
                  entry);
          Map p = raw(ev, str(v.get("partyId")));
          return map(
              "party",
              v,
              "access",
              access(p, e),
              "operationId",
              r.get("operationId"),
              "replayed",
              false);
        }
      case "PartyExchange":
        {
          var q =
              s.row(
                  "select * from ug_credential where token_digest=? and kind='QUERY' and"
                      + " status='ACTIVE' and expires_at>now()",
                  s.crypto.hash(str(r.get("queryToken"))));
          Map e = s.event(str(q.get("event_id")));
          var ns =
              identity.newSession(
                  "PARTY_QUERY",
                  "ug_party",
                  null,
                  str(q.get("event_id")),
                  str(q.get("party_id")),
                  null);
          if (r.containsKey("changeToken")) {
            var change =
                s.row(
                    "select * from ug_credential where token_digest=? and kind='CHANGE' and"
                        + " party_id=? and status='ACTIVE' and expires_at>now()",
                    s.crypto.hash(str(r.get("changeToken"))),
                    q.get("party_id"));
            var cs =
                identity.newSession(
                    "PARTY_CHANGE",
                    "ug_change",
                    null,
                    str(q.get("event_id")),
                    str(q.get("party_id")),
                    null);
            s.execute(
                "update ug_session set csrf_digest=? where id=?",
                s.crypto.hash(str(ns.get("csrf"))),
                ((Map) cs.get("row")).get("id"));
          }
          Map ses = (Map) ns.get("row");
          return map(
              "csrfToken",
              ns.get("csrf"),
              "expiresAt",
              ses.get("expires_at"),
              "clientContextId",
              str(ses.get("id")));
        }
      case "MyParty", "MyCredential":
        {
          var e = s.event(ev);
          Map p = raw(ev, str(c.session.get("party_id")));
          return "MyParty".equals(op) ? state(p, e) : credential(p, e);
        }
      case "PartyList":
        {
          c.view(ev);
          c.action(ev, "VIEW_DETAIL");
          ApiException.require(!"CHECKER".equals(c.eventRole(ev)), 403, "FORBIDDEN", "请使用辅助找票");
          return list(ev, r, c.fields(ev));
        }
      case "PartyDetail":
        {
          c.view(ev);
          c.action(ev, "VIEW_DETAIL");
          ApiException.require(!"CHECKER".equals(c.eventRole(ev)), 403, "FORBIDDEN", "请使用辅助找票");
          return view(raw(ev, str(r.get("partyId"))), c.fields(ev));
        }
      case "PartyChange", "MyPartyChange", "PartyCancel", "MyPartyCancel":
        {
          var e = s.event(ev);
          ready(e);
          boolean self = op.startsWith("My");
          if (!self) c.organizer(ev);
          String pid = self ? str(c.session.get("party_id")) : str(r.get("partyId"));
          Map p = raw(ev, pid);
          s.version(p, r.get("version"));
          ApiException.require(
              !Boolean.TRUE.equals(p.get("ever_admitted")),
              409,
              "ADMISSION_HISTORY_EXISTS",
              "已有入场记录，不允许普通变更");
          ApiException.require(
              List.of("PENDING", "APPROVED").contains(p.get("status")),
              409,
              "PARTY_STATE_CONFLICT",
              "当前资格不允许变更");
          boolean cancel = op.endsWith("Cancel");
          if (self)
            ApiException.require(
                canSelf(p, e, cancel ? "selfCancelEnabled" : "selfEditEnabled"),
                403,
                "SELF_SERVICE_DISABLED",
                "自助入口已关闭");
          if (cancel) {
            if (num(p.get("approved_size")) > 0) allocate(e, -num(p.get("approved_size")));
            s.execute(
                "update ug_party set"
                    + " status='CANCELLED',approved_size=0,pending_increase=0,version=version+1,updated_at=now()"
                    + " where id=?",
                p.get("id"));
            s.execute(
                "update ug_credential set status='REVOKED' where party_id=? and kind in"
                    + " ('TICKET','CHANGE') and status='ACTIVE'",
                p.get("id"));
            s.execute(
                "update ug_party_review set result='REJECTED',decided_at=now(),version=version+1"
                    + " where party_id=? and result='PENDING'",
                p.get("id"));
          } else {
            Map f = s.unseal(p.get("fields_cipher"));
            Map rule = (Map) ((Map) s.parse(e.get("policy")).get("fields")).get("companionCount");
            ApiException.require(
                Boolean.TRUE.equals(rule.get("enabled")), 422, "FIELD_DISABLED", "随行人数未启用");
            int n = num(r.get("companionCount")) + 1, approved = num(p.get("approved_size"));
            f.put("companionCount", r.get("companionCount"));
            s.execute(
                "update ug_party_review set result='REJECTED',decided_at=now(),version=version+1"
                    + " where party_id=? and result='PENDING'",
                p.get("id"));
            int pending = 0, newApproved = approved;
            if ("PENDING".equals(p.get("status"))) {
              review(p, "INITIAL", n, n);
            } else if (n > approved
                && "REVIEW".equals(s.parse(e.get("policy")).get("approvalMode"))) {
              pending = n - approved;
              review(p, "INCREASE", n, pending);
            } else {
              allocate(e, n - approved);
              newApproved = n;
            }
            s.execute(
                "update ug_party set"
                    + " fields_cipher=?,requested_size=?,approved_size=?,pending_increase=?,version=version+1,updated_at=now()"
                    + " where id=?",
                s.seal(f),
                n,
                newApproved,
                pending,
                p.get("id"));
          }
          p = raw(ev, pid);
          return self ? state(p, e) : view(p, c.fields(ev));
        }
      case "ReviewList":
        {
          c.organizer(ev);
          s.event(ev);
          var reviews =
              s
                  .rows(
                      "select r.*,p.approved_size from ug_party_review r join ug_party p on"
                          + " p.id=r.party_id where r.event_id=? order by r.created_at desc,r.id"
                          + " desc",
                      uuid(ev))
                  .stream()
                  .filter(
                      x ->
                          (!r.containsKey("kind") || r.get("kind").equals(x.get("kind")))
                              && (!r.containsKey("result")
                                  || r.get("result").equals(x.get("result"))))
                  .map(this::reviewView)
                  .toList();
          return s.page(reviews, r);
        }
      case "ReviewDecide":
        {
          c.organizer(ev);
          Map e = s.event(ev);
          ready(e);
          Map rev =
              s.row(
                  "select * from ug_party_review where id=? and event_id=? for update",
                  uuid(r.get("reviewId")),
                  uuid(ev));
          s.version(rev, r.get("version"));
          ApiException.require(
              "PENDING".equals(rev.get("result")), 409, "REVIEW_STATE_CONFLICT", "已处理审核");
          Map p = raw(ev, str(rev.get("party_id")));
          boolean approve = "APPROVE".equals(r.get("decision"));
          boolean initial = "INITIAL".equals(rev.get("kind"));
          if (approve) {
            int delta =
                initial ? num(rev.get("requested_size")) : num(rev.get("requested_increase"));
            allocate(e, delta);
            s.execute(
                "update ug_party set"
                    + " status='APPROVED',approved_size=approved_size+?,pending_increase=0,version=version+1,updated_at=now()"
                    + " where id=?",
                delta,
                p.get("id"));
            if (initial) issueCapability(raw(ev, str(p.get("id"))), "TICKET", e);
          } else {
            s.execute(
                "update ug_party set status=?,pending_increase=0,requested_size=case when ? then"
                    + " requested_size else approved_size end,version=version+1,updated_at=now()"
                    + " where id=?",
                initial ? "REJECTED" : "APPROVED",
                initial,
                p.get("id"));
          }
          s.execute(
              "update ug_party_review set"
                  + " result=?,actor_id=?,reason_cipher=?,decided_at=now(),version=version+1 where"
                  + " id=?",
              approve ? "APPROVED" : "REJECTED",
              uuid(c.actorId()),
              s.seal(map("reason", r.get("reason"))),
              rev.get("id"));
          enqueueEmail(raw(ev, str(p.get("id"))), e);
          return reviewView(
              s.row(
                  "select r.*,p.approved_size from ug_party_review r join ug_party p on"
                      + " p.id=r.party_id where r.id=?",
                  rev.get("id")));
        }
      case "CredentialRevoke", "CredentialReissue":
        {
          c.organizer(ev);
          Map e = s.event(ev);
          Map p = raw(ev, str(r.get("partyId")));
          s.version(p, r.get("version"));
          ApiException.require(
              "APPROVED".equals(p.get("status")), 409, "PARTY_STATE_CONFLICT", "尚无有效资格");
          s.execute(
              "update ug_credential set status='REVOKED' where party_id=? and kind='TICKET' and"
                  + " status='ACTIVE'",
              p.get("id"));
          s.execute(
              "update ug_party set version=version+1,updated_at=now() where id=?", p.get("id"));
          p = raw(ev, str(p.get("id")));
          if ("CredentialRevoke".equals(op)) return view(p, c.fields(ev));
          issueCapability(p, "TICKET", e);
          Map out =
              map(
                  "party",
                  view(p, c.fields(ev)),
                  "credential",
                  credential(p, e),
                  "operationId",
                  r.get("operationId"),
                  "replayed",
                  false);
          if (Boolean.TRUE.equals(r.get("rotateAccess"))) {
            s.execute(
                "update ug_credential set status='REVOKED' where party_id=? and kind in"
                    + " ('QUERY','CHANGE') and status='ACTIVE'",
                p.get("id"));
            s.execute("update ug_session set revoked_at=now() where party_id=?", p.get("id"));
            issueCapability(p, "QUERY", e);
            issueCapability(p, "CHANGE", e);
            out.put("access", access(p, e));
          }
          return out;
        }
      case "GateWalkIn":
        {
          c.organizer(ev);
          Map e = s.event(ev);
          ApiException.require(
              "CHECKIN_OPEN".equals(e.get("status")), 409, "EVENT_PHASE_CONFLICT", "检票未开放");
          if (Boolean.TRUE.equals(s.parse(e.get("policy")).get("phoneVerificationEnabled")))
            verifyPhone(r, c, e);
          if (r.containsKey("existingPendingPartyId")) {
            Map p = raw(ev, str(r.get("existingPendingPartyId")));
            s.version(p, r.get("partyVersion"));
            ApiException.require(
                "PENDING".equals(p.get("status")), 409, "PARTY_STATE_CONFLICT", "不是待审核报名");
            validateFields(e, (Map) r.get("fields"));
            int size = 1 + num(((Map) r.get("fields")).get("companionCount"));
            allocate(e, size);
            s.execute(
                "update ug_party set"
                    + " fields_cipher=?,requested_size=?,approved_size=?,status='APPROVED',version=version+1,updated_at=now()"
                    + " where id=?",
                s.seal(r.get("fields")),
                size,
                size,
                p.get("id"));
            s.execute(
                "update ug_party_review set"
                    + " result='APPROVED',actor_id=?,reason_cipher=?,decided_at=now(),version=version+1"
                    + " where party_id=? and result='PENDING'",
                uuid(c.actorId()),
                s.seal(map("reason", r.get("reason"))),
                p.get("id"));
            p = raw(ev, str(p.get("id")));
            issueCapability(p, "TICKET", e);
            return view(p, c.fields(ev));
          }
          return create(
              c, e, (Map) r.get("fields"), "WALK_IN", true, events.noticeVersion(e), entry);
        }
      case "GateIncrease":
        {
          c.organizer(ev);
          Map e = s.event(ev);
          ApiException.require(
              "CHECKIN_OPEN".equals(e.get("status")), 409, "EVENT_PHASE_CONFLICT", "检票未开放");
          Map p = raw(ev, str(r.get("partyId")));
          s.version(p, r.get("version"));
          ApiException.require(
              "APPROVED".equals(p.get("status")), 409, "PARTY_STATE_CONFLICT", "需要有效资格");
          int count = num(r.get("additionalCount"));
          allocate(e, count);
          s.execute(
              "update ug_party set"
                  + " approved_size=approved_size+?,requested_size=requested_size+?,version=version+1,updated_at=now()"
                  + " where id=?",
              count,
              count,
              p.get("id"));
          s.execute(
              "insert into ug_outbox(id,event_id,event_type,resource_id,payload,created_at)"
                  + " values(?,?,'OnsiteIncrease',?,?::jsonb,now())",
              uuid(s.id()),
              uuid(ev),
              p.get("id"),
              s.stringify(map("count", count)));
          return view(raw(ev, str(p.get("id"))), c.fields(ev));
        }
      default:
        throw new IllegalArgumentException(op);
    }
  }

  void verifyPhone(Map r, Context c, Map e) {
    ApiException.require(
        r.containsKey("phoneVerificationProof"), 422, "PHONE_VERIFICATION_REQUIRED", "请先验证手机号码");
    Map proof = s.crypto.verify(str(r.get("phoneVerificationProof")), s.json);
    ApiException.require(
        str(e.get("id")).equals(proof.get("eventId"))
            && str(c.session.get("id")).equals(proof.get("contextId"))
            && time(proof.get("expiresAt")).isAfter(s.now()),
        422,
        "PHONE_VERIFICATION_REQUIRED",
        "手机验证已失效");
    Map challenge =
        s.row(
            "select * from ug_challenge where id=? and event_id=? for update",
            uuid(proof.get("challengeId")),
            e.get("id"));
    Map info = s.unseal(challenge.get("context_cipher"));
    String phone = str(((Map) r.get("fields")).get("phone"));
    String normalized = phone == null ? null : phone.replaceAll("[\\s()\\-]", "");
    ApiException.require(
        Objects.equals(normalized, proof.get("phone"))
            && !Boolean.TRUE.equals(info.get("proofUsed")),
        422,
        "VERIFICATION_REQUIRED",
        "手机号验证无效或已使用");
    info.put("proofUsed", true);
    s.execute(
        "update ug_challenge set context_cipher=? where id=?", s.seal(info), challenge.get("id"));
  }

  public Map<String, Object> list(String ev, Map<String, Object> r, Set<String> allowed) {
    List<Object> result = new ArrayList<>();
    for (var p :
        s.rows(
            "select * from ug_party where event_id=? order by created_at desc,id desc", uuid(ev))) {
      if (r.containsKey("status") && !r.get("status").equals(p.get("status"))) continue;
      if (r.containsKey("sourceType") && !r.get("sourceType").equals(p.get("source_type")))
        continue;
      if (r.containsKey("progress") && !r.get("progress").equals(progress(p))) continue;
      if (r.containsKey("createdFrom")
          && time(p.get("created_at")).isBefore(time(r.get("createdFrom")))) continue;
      if (r.containsKey("createdTo") && time(p.get("created_at")).isAfter(time(r.get("createdTo"))))
        continue;
      if (r.containsKey("hasPendingIncrease")
          && !Objects.equals(r.get("hasPendingIncrease"), num(p.get("pending_increase")) > 0))
        continue;
      if (r.containsKey("query")) {
        String q = str(r.get("query"));
        Map f = s.unseal(p.get("fields_cipher"));
        boolean match = str(p.get("party_reference")).contains(q);
        for (String key : allowed)
          if (f.containsKey(key) && str(f.get(key)).contains(q)) match = true;
        if (!match) continue;
      }
      result.add(view(p, allowed));
    }
    Map page = s.page(result, r);
    page.put("asOf", s.now());
    return page;
  }

  Map<String, Object> reviewView(Map rev) {
    Map v =
        map(
            "reviewId",
            str(rev.get("id")),
            "partyId",
            str(rev.get("party_id")),
            "eventId",
            str(rev.get("event_id")),
            "kind",
            s.value("ReviewKind", rev.get("kind")),
            "requestedSize",
            rev.get("requested_size"),
            "approvedSize",
            rev.get("approved_size"),
            "requestedIncrease",
            rev.get("requested_increase"),
            "result",
            s.value("ReviewResult", rev.get("result")),
            "version",
            rev.get("version"),
            "createdAt",
            rev.get("created_at"));
    v.putAll(s.unseal(rev.get("reason_cipher")));
    return v;
  }
}
