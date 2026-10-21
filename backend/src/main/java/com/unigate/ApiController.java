package com.unigate;

import static com.unigate.Store.*;

import jakarta.servlet.http.*;
import java.security.MessageDigest;
import java.util.*;
import org.springframework.dao.*;
import org.springframework.http.*;
import org.springframework.transaction.PlatformTransactionManager;
import org.springframework.transaction.support.TransactionTemplate;
import org.springframework.web.bind.annotation.*;
import org.springframework.web.multipart.*;

@RestController
public class ApiController {
  final Store s;
  final Contract contract;
  final IdentityService identity;
  final EventService events;
  final PartyService parties;
  final GateService gate;
  final IntegrationService integration;
  final TransactionTemplate tx;
  final TransactionTemplate statisticsTx;

  public ApiController(
      Store s,
      Contract contract,
      IdentityService i,
      EventService e,
      PartyService p,
      GateService g,
      IntegrationService integration,
      PlatformTransactionManager tm) {
    this.s = s;
    this.contract = contract;
    identity = i;
    events = e;
    parties = p;
    gate = g;
    this.integration = integration;
    tx = new TransactionTemplate(tm);
    tx.setIsolationLevel(
        org.springframework.transaction.TransactionDefinition.ISOLATION_READ_COMMITTED);
    tx.setTimeout(5);
    statisticsTx = new TransactionTemplate(tm);
    statisticsTx.setPropagationBehavior(
        org.springframework.transaction.TransactionDefinition.PROPAGATION_REQUIRES_NEW);
    statisticsTx.setIsolationLevel(
        org.springframework.transaction.TransactionDefinition.ISOLATION_REPEATABLE_READ);
    statisticsTx.setReadOnly(true);
    statisticsTx.setTimeout(5);
  }

  @RequestMapping(
      value = "/api/v1/**",
      method = {RequestMethod.GET, RequestMethod.POST})
  public ResponseEntity<?> execute(HttpServletRequest request, HttpServletResponse response)
      throws Exception {
    s.eventScope.remove();
    identity.response.set(response);
    String reqId = s.id();
    response.setHeader("Cache-Control", "no-store");
    response.setHeader("Pragma", "no-cache");
    response.setHeader("Referrer-Policy", "no-referrer");
    response.setHeader("X-Content-Type-Options", "nosniff");
    response.setHeader("X-Request-ID", reqId);
    try {
      Map<String, Object> op =
          contract.routes.get(request.getMethod() + " " + request.getRequestURI());
      if (op == null) throw new ApiException(404, "RESOURCE_NOT_FOUND", "未找到接口");
      s.readOnlyRequest.set(!Boolean.TRUE.equals(op.get("mutation")));
      Map<String, Object> r = new LinkedHashMap<>();
      boolean upload = Boolean.TRUE.equals(op.get("upload"));
      if (upload) {
        for (var param : request.getParameterMap().entrySet())
          r.put(param.getKey(), param.getValue()[0]);
        if (request instanceof MultipartHttpServletRequest multi)
          r.put("file", multi.getFile("file"));
        Map copy = new LinkedHashMap(r);
        Object file = copy.remove("file");
        contract.validate(
            str(op.get("request")),
            new LinkedHashMap(copy) {
              {
                put("file", file instanceof MultipartFile m ? m.getOriginalFilename() : null);
              }
            });
      } else if ("GET".equals(request.getMethod())) {
        for (var param : request.getParameterMap().entrySet())
          r.put(param.getKey(), param.getValue()[0]);
        contract.validate(str(op.get("request")), r);
      } else {
        r = s.json.readValue(request.getInputStream(), Map.class);
        contract.validate(str(op.get("request")), r);
      }
      Map<String, Object> input = r;
      boolean statistics = "Statistics".equals(op.get("name"));
      identity.touchStaffSession(request);
      Object data =
          (statistics ? statisticsTx : tx)
              .execute(
                  status -> {
                    if ("SessionBootstrap".equals(op.get("name")))
                      return checked(op, identity.bootstrap(request));
                    Context c = identity.authenticate(op, input, request);
                    String ev = str(input.get("eventId"));
                    if (ev == null && c.session != null && c.session.get("event_id") != null)
                      ev = str(c.session.get("event_id"));
                    if (ev == null && c.device != null && Set.of("GATE", "DEVICE").contains(c.auth))
                      ev = str(c.device.get("event_id"));
                    if (ev != null) {
                      if (!Set.of("CleanupDetail", "CleanupRetry").contains(str(op.get("name"))))
                        s.event(ev);
                      if (c.device != null && Set.of("GATE", "DEVICE").contains(c.auth)) {
                        c.device =
                            s.row(
                                "select * from ug_device where id=? and status='ACTIVE' and"
                                    + " expires_at>now() for update",
                                c.device.get("id"));
                      }
                      if ("STAFF".equals(c.auth) && !"MyGrantList".equals(op.get("name"))) {
                        boolean globalAdmin =
                            c.admin() && ((List<?>) op.get("roles")).contains("ADMIN");
                        if (!globalAdmin) {
                          String role = c.eventRole(ev);
                          ApiException.require(
                              ((List<?>) op.get("roles")).contains(role),
                              403,
                              "FORBIDDEN",
                              "当前活动授权不包含此动作");
                        }
                      }
                    }
                    if (!Boolean.TRUE.equals(op.get("mutation")))
                      return checked(op, dispatch(str(op.get("name")), input, c, null));
                    String name = str(op.get("name"));
                    String actor =
                        "GATE".equals(c.auth)
                            ? "GATE"
                            : c.account != null && "STAFF".equals(c.auth)
                                ? "STAFF:" + c.actorId()
                                : "CLIENT:" + c.session.get("id");
                    Map<String, Object> digestInput = new TreeMap<>(input);
                    digestInput.remove("operationId");
                    if ("GateCommit".equals(name)) {
                      Map scan = gate.scan(str(input.get("scanContextToken")), c, false);
                      digestInput.remove("scanContextToken");
                      digestInput.put("partyId", scan.get("partyId"));
                      digestInput.put("deviceId", str(c.device.get("id")));
                    }
                    if (upload) {
                      MultipartFile file = (MultipartFile) input.get("file");
                      try {
                        digestInput.put(
                            "file",
                            HexFormat.of()
                                .formatHex(
                                    MessageDigest.getInstance("SHA-256").digest(file.getBytes())));
                      } catch (Exception ex) {
                        throw new ApiException(422, "INVALID_FILE", "无法读取文件");
                      }
                    }
                    byte[] digest = s.crypto.hash(s.stringify(canonical(digestInput)));
                    String eventScope = ev == null ? "GLOBAL" : ev;
                    UUID entry = uuid(s.id());
                    int inserted =
                        s.execute(
                            "insert into"
                                + " ug_operation(id,actor_scope,event_scope,event_id,action,operation_id,payload_digest,status,created_at)"
                                + " values(?,?,?,?,?,?,?,'IN_PROGRESS',now()) on"
                                + " conflict(actor_scope,event_scope,action,operation_id) do"
                                + " nothing",
                            entry,
                            actor,
                            eventScope,
                            uuid(ev),
                            name,
                            uuid(input.get("operationId")),
                            digest);
                    if (inserted == 0) {
                      Map prior =
                          s.row(
                              "select * from ug_operation where actor_scope=? and event_scope=? and"
                                  + " action=? and operation_id=?",
                              actor,
                              eventScope,
                              name,
                              uuid(input.get("operationId")));
                      ApiException.require(
                          MessageDigest.isEqual(digest, (byte[]) prior.get("payload_digest")),
                          409,
                          "IDEMPOTENCY_CONFLICT",
                          "原操作编号对应其他内容，请核对");
                      ApiException.require(
                          "SUCCEEDED".equals(prior.get("status")),
                          409,
                          "OPERATION_IN_PROGRESS",
                          "原操作正在处理中，请沿用原编号查询");
                      ApiException.require(
                          prior.get("private_result_expires_at") == null
                              || time(prior.get("private_result_expires_at")).isAfter(s.now()),
                          410,
                          "RESULT_EXPIRED",
                          "原结果已到期，请联系负责人核对");
                      Map bundle = s.unseal(prior.get("result_cipher"));
                      if (bundle.get("cookies") instanceof List cookies)
                        for (Object cookie : cookies)
                          response.addHeader("Set-Cookie", cookie.toString());
                      Map result = s.parse(bundle.get("data"));
                      if ("STAFF".equals(c.auth)
                          && ev != null
                          && Set.of(
                                  "PartyChange",
                                  "PartyCancel",
                                  "CredentialReissue",
                                  "CredentialRevoke",
                                  "GateWalkIn",
                                  "GateIncrease")
                              .contains(name)) projectReplay(result, c.fields(ev));
                      if (result.containsKey("replayed")) result.put("replayed", true);
                      if (result.containsKey("showAsNewAdmission"))
                        result.put("showAsNewAdmission", false);
                      return checked(op, result);
                    }
                    Object result = checked(op, dispatch(name, input, c, entry));
                    Object resource = resourceId(result);
                    s.audit(
                        c, ev, name, resource, input.get("operationId"), str(input.get("reason")));
                    s.execute(
                        "update ug_operation set"
                            + " status='SUCCEEDED',result_code='OK',resource_id=?,result_cipher=?,private_result_expires_at=?,committed_at=now()"
                            + " where id=?",
                        uuid(resource),
                        s.seal(
                            map(
                                "data",
                                result,
                                "cookies",
                                new ArrayList<>(response.getHeaders("Set-Cookie")))),
                        privateExpiry(name, ev),
                        entry);
                    return result;
                  });
      if (data instanceof IntegrationService.Binary b)
        return ResponseEntity.ok()
            .contentType(MediaType.parseMediaType(b.contentType()))
            .header("Content-Disposition", "attachment; filename=\"" + b.filename() + "\"")
            .body(b.data());
      return ResponseEntity.status(num(op.get("status")))
          .body(map("code", "OK", "message", "操作成功", "data", data, "requestId", reqId));
    } catch (ApiException e) {
      return ResponseEntity.status(e.status)
          .body(map("code", e.code, "message", e.getMessage(), "data", map(), "requestId", reqId));
    } catch (CannotAcquireLockException | CannotSerializeTransactionException e) {
      return ResponseEntity.status(409)
          .body(
              map(
                  "code",
                  "OPERATION_IN_PROGRESS",
                  "message",
                  "操作竞争，请沿用原操作编号核对",
                  "data",
                  map(),
                  "requestId",
                  reqId));
    } catch (DataIntegrityViolationException e) {
      return ResponseEntity.status(409)
          .body(
              map(
                  "code",
                  "DATA_CONFLICT",
                  "message",
                  "数据已变化或记录冲突，请刷新核对",
                  "data",
                  map(),
                  "requestId",
                  reqId));
    } catch (com.fasterxml.jackson.core.JsonProcessingException | IllegalArgumentException e) {
      return ResponseEntity.badRequest()
          .body(
              map(
                  "code",
                  "INVALID_REQUEST",
                  "message",
                  "请求格式无效",
                  "data",
                  map(),
                  "requestId",
                  reqId));
    } catch (Exception e) {
      org.slf4j.LoggerFactory.getLogger(getClass())
          .error("Request {} failed ({})", reqId, e.getClass().getSimpleName());
      return ResponseEntity.status(500)
          .body(
              map(
                  "code",
                  "INTERNAL_ERROR",
                  "message",
                  "服务暂不可用，请沿原操作编号核对结果",
                  "data",
                  map(),
                  "requestId",
                  reqId));
    } finally {
      identity.response.remove();
      s.eventScope.remove();
      s.readOnlyRequest.remove();
    }
  }

  Object checked(Map<String, Object> op, Object data) {
    if (data instanceof IntegrationService.Binary) return data;
    try {
      contract.validate(str(op.get("data")), s.json.convertValue(data, Object.class));
      return data;
    } catch (ApiException e) {
      throw new IllegalStateException(
          "Response contract mismatch for " + op.get("name") + ": " + e.getMessage());
    }
  }

  void projectReplay(Map<String, Object> value, Set<String> fields) {
    if (value.containsKey("partyId") && value.get("fields") instanceof Map data) {
      data.keySet().retainAll(fields);
    }
    for (Object child : value.values()) {
      if (child instanceof Map map) projectReplay(map, fields);
      else if (child instanceof List list)
        for (Object item : list) if (item instanceof Map map) projectReplay(map, fields);
    }
  }

  java.time.OffsetDateTime privateExpiry(String op, String ev) {
    if (!Set.of(
            "Login",
            "PartyExchange",
            "SubmitRegistration",
            "RecoveryVerify",
            "DeviceRegister",
            "DeviceClaim",
            "CredentialReissue",
            "SmsVerify")
        .contains(op)) return null;
    var expiry = s.now().plusHours(24);
    if (ev != null) {
      var due =
          time(
              s.row("select personal_due_at from ug_event where id=?", uuid(ev))
                  .get("personal_due_at"));
      if (due.isBefore(expiry)) expiry = due;
    }
    return expiry;
  }

  static Object canonical(Object o) {
    if (o instanceof Map m) {
      Map sorted = new TreeMap();
      m.forEach((k, v) -> sorted.put(k, canonical(v)));
      return sorted;
    }
    if (o instanceof List l) return l.stream().map(ApiController::canonical).toList();
    return o;
  }

  Object resourceId(Object result) {
    if (result instanceof Map m) {
      for (String k :
          List.of(
              "partyId",
              "eventId",
              "recordId",
              "reviewId",
              "batchId",
              "exportId",
              "jobId",
              "accountId",
              "grantId",
              "connectionId",
              "sourceId",
              "correctionId")) if (m.get(k) != null) return m.get(k);
      if (m.get("party") instanceof Map p) return p.get("partyId");
      if (m.get("device") instanceof Map p) return p.get("deviceId");
    }
    return null;
  }

  Object dispatch(String op, Map<String, Object> r, Context c, UUID entry) {
    if (Set.of(
            "Login",
            "Logout",
            "SessionMe",
            "PasswordChange",
            "AccountList",
            "AccountCreate",
            "AccountUpdate",
            "GrantEventOptions",
            "MyGrantList",
            "GrantList",
            "GrantSave",
            "GrantRevoke")
        .contains(op)) return identity.handle(op, r, c, entry);
    if (Set.of(
            "EventCreate",
            "EventList",
            "EventOptions",
            "PublicEvent",
            "EventDetail",
            "EventUpdate",
            "EventTransition",
            "PolicyGet",
            "PolicyUpdate")
        .contains(op)) return events.handle(op, r, c, entry);
    if (Set.of(
            "SubmitRegistration",
            "PartyExchange",
            "MyParty",
            "MyCredential",
            "PartyList",
            "PartyDetail",
            "PartyChange",
            "MyPartyChange",
            "PartyCancel",
            "MyPartyCancel",
            "ReviewList",
            "ReviewDecide",
            "CredentialRevoke",
            "CredentialReissue",
            "GateWalkIn",
            "GateIncrease")
        .contains(op)) return parties.handle(op, r, c, entry);
    if (Set.of(
            "DeviceRegister",
            "DeviceClaim",
            "DeviceDetail",
            "DeviceList",
            "DeviceRevoke",
            "TicketResolve",
            "GateCommit",
            "GateOperation",
            "GateLookup",
            "GateRecords",
            "GateCorrect")
        .contains(op)) return gate.handle(op, r, c, entry);
    if (Set.of("Statistics", "AuditList", "OperationResult", "MyOperation", "RegistrationResult")
        .contains(op)) return reports(op, r, c);
    return integration.handle(op, r, c, entry);
  }

  Object reports(String op, Map<String, Object> r, Context c) {
    String ev = str(r.get("eventId"));
    if ("MyOperation".equals(op)) ev = str(c.session.get("event_id"));
    if ("RegistrationResult".equals(op)) {
      s.event(ev);
      Map o =
          s.row(
              "select * from ug_operation where actor_scope=? and event_scope=? and"
                  + " action='SubmitRegistration' and operation_id=? and status='SUCCEEDED' and"
                  + " private_result_expires_at>now()",
              "CLIENT:" + c.session.get("id"),
              ev,
              uuid(r.get("operationId")));
      Map v = s.parse(s.unseal(o.get("result_cipher")).get("data"));
      v.put("replayed", true);
      return v;
    }
    if ("Statistics".equals(op)) {
      c.view(ev);
      c.action(ev, "VIEW_STATISTICS");
      Map e = s.event(ev);
      Map stats =
          s.row(
              "select count(*) groups,count(*) filter(where status='PENDING') pending,count(*)"
                  + " filter(where status='APPROVED') approved,coalesce(sum(net_admitted),0)"
                  + " net,count(*) filter(where source_type='WALK_IN') walkins from ug_party where"
                  + " event_id=?",
              uuid(ev));
      int
          gross =
              s.db.queryForObject(
                  "select coalesce(sum(admitted_count),0) from ug_admission where event_id=?",
                  Integer.class,
                  uuid(ev)),
          corrected =
              s.db.queryForObject(
                  "select coalesce(sum(corrected_count),0) from ug_admission where event_id=?",
                  Integer.class,
                  uuid(ev)),
          allocated = num(e.get("allocated_count")),
          net = num(stats.get("net"));
      Map<String, Object> statistics =
          map(
              "eventId",
              ev,
              "asOf",
              s.now(),
              "applicationGroups",
              num(stats.get("groups")),
              "pendingGroups",
              num(stats.get("pending")),
              "approvedGroups",
              num(stats.get("approved")),
              "allocatedPeople",
              allocated,
              "grossAdmissions",
              gross,
              "correctedPeople",
              corrected,
              "netAdmissions",
              net,
              "unusedPlaces",
              allocated - net,
              "walkInGroups",
              num(stats.get("walkins")),
              "onsiteAddedPlaces",
              onsiteAdded(ev),
              "admissionRate",
              allocated == 0 ? 0.0 : ((double) net / allocated));
      if (allocated == 0) statistics.put("admissionRate", null);
      return statistics;
    }
    if ("AuditList".equals(op)) {
      c.gateHistory(ev);
      boolean lead = "ORGANIZER".equals(c.eventRole(ev));
      List<Object> all = new ArrayList<>();
      for (Map a :
          s.rows(
              "select * from ug_audit where event_id=? order by occurred_at desc,id desc",
              uuid(ev))) {
        if (!lead
            && !c.actorId().equals(str(a.get("actor_id")))
            && (c.device == null || !str(c.device.get("id")).equals(str(a.get("device_id")))))
          continue;
        if (r.containsKey("action") && !r.get("action").equals(a.get("action"))) continue;
        all.add(
            map(
                "auditId",
                str(a.get("id")),
                "eventId",
                ev,
                "action",
                a.get("action"),
                "actorId",
                str(a.get("actor_id")),
                "deviceId",
                str(a.get("device_id")),
                "resourceId",
                str(a.get("resource_id")),
                "occurredAt",
                a.get("occurred_at")));
      }
      return s.page(all, r);
    }
    if ("OperationResult".equals(op)) c.gateHistory(ev);
    else
      ApiException.require(
          Set.of("MyPartyChange", "MyPartyCancel").contains(r.get("action")),
          403,
          "FORBIDDEN",
          "不允许查询此动作");
    List<Map<String, Object>> found =
        s.rows(
            "select * from ug_operation where event_id=? and operation_id=? and action=?",
            uuid(ev),
            uuid(r.get("operationId")),
            r.get("action"));
    ApiException.require(!found.isEmpty(), 404, "OPERATION_NOT_FOUND", "尚未查到原操作，请沿用原编号");
    Map o = found.getFirst();
    if ("MyOperation".equals(op))
      ApiException.require(
          str(c.session.get("party_id")).equals(str(o.get("resource_id"))),
          404,
          "RESOURCE_NOT_FOUND",
          "未找到资源");
    else if (!"ORGANIZER".equals(c.eventRole(ev))) {
      ApiException.require(
          Set.of("GateCommit").contains(r.get("action")), 403, "FORBIDDEN", "不能查询此动作");
      Map a = s.row("select * from ug_admission where operation_entry_id=?", o.get("id"));
      ApiException.require(
          c.actorId().equals(str(a.get("actor_id")))
              || (c.device != null && str(c.device.get("id")).equals(str(a.get("device_id")))),
          404,
          "RESOURCE_NOT_FOUND",
          "未找到资源");
    }
    return map(
        "operationId",
        r.get("operationId"),
        "status",
        s.value("OperationStatus", o.get("status")),
        "action",
        o.get("action"),
        "resourceId",
        str(o.get("resource_id")),
        "committedAt",
        o.get("committed_at"),
        "resultCode",
        o.get("result_code"),
        "replayed",
        true);
  }

  int onsiteAdded(String ev) {
    return s.db.queryForObject(
        "select coalesce(sum((payload->>'count')::integer),0) from ug_outbox where event_id=? and"
            + " event_type='OnsiteIncrease'",
        Integer.class,
        uuid(ev));
  }
}
