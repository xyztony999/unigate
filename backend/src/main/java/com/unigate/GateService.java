package com.unigate;

import static com.unigate.Store.*;

import java.security.MessageDigest;
import java.time.*;
import java.util.*;
import org.springframework.stereotype.Service;

@Service
public class GateService {
  final Store s;
  final PartyService parties;
  final IdentityService identity;

  public GateService(Store s, PartyService p, IdentityService i) {
    this.s = s;
    parties = p;
    identity = i;
  }

  public Map<String, Object> scan(String token, Context c, boolean enforceExpiry) {
    Map p = s.crypto.verify(token, s.json);
    ApiException.require(
        "SCAN".equals(p.get("kind")) && str(c.device.get("id")).equals(p.get("deviceId")),
        403,
        "FORBIDDEN",
        "扫码上下文不属于当前终端");
    if (enforceExpiry)
      ApiException.require(time(p.get("expiresAt")).isAfter(s.now()), 410, "SCAN_EXPIRED", "请重新扫码");
    return p;
  }

  void open(Map e) {
    ApiException.require("CHECKIN_OPEN".equals(e.get("status")), 409, "CHECKIN_CLOSED", "当前未开放检票");
    ApiException.require(time(e.get("ends_at")).isAfter(s.now()), 409, "CHECKIN_CLOSED", "活动已结束");
  }

  Map<String, Object> deviceView(Map d) {
    return map(
        "deviceId",
        str(d.get("id")),
        "eventId",
        str(d.get("event_id")),
        "name",
        d.get("name"),
        "mode",
        s.value("DeviceMode", d.get("mode")),
        "status",
        s.value("DeviceStatus", d.get("status")),
        "expiresAt",
        d.get("expires_at"),
        "lastSeenAt",
        d.get("last_seen_at"),
        "version",
        d.get("version"));
  }

  Map<String, Object> deviceSession(Map d) {
    String csrf =
        s.crypto.signed(
            map(
                "kind",
                "CSRF",
                "deviceId",
                str(d.get("id")),
                "expiresAt",
                time(d.get("expires_at")).toString()),
            s.json);
    return map(
        "device",
        deviceView(d),
        "csrfToken",
        csrf,
        "expiresAt",
        d.get("expires_at"),
        "eventStatus",
        s.value("EventStatus", s.event(str(d.get("event_id"))).get("status")));
  }

  public Object handle(String op, Map<String, Object> r, Context c, UUID entry) {
    String ev = str(r.get("eventId"));
    switch (op) {
      case "DeviceRegister":
        {
          c.organizer(ev);
          Map e = s.event(ev);
          ApiException.require(
              !List.of("ENDED", "ARCHIVED").contains(e.get("status")),
              409,
              "EVENT_PHASE_CONFLICT",
              "活动已结束");
          String id = s.id(), pair = s.crypto.token();
          OffsetDateTime expiry = s.now().plusHours(12);
          if (time(e.get("ends_at")).isBefore(expiry)) expiry = time(e.get("ends_at"));
          ApiException.require(expiry.isAfter(s.now()), 409, "EVENT_PHASE_CONFLICT", "活动已过期");
          s.execute(
              "insert into"
                  + " ug_device(id,event_id,name,mode,status,pairing_digest,pairing_expires_at,expires_at,created_by)"
                  + " values(?,?,?,?,'UNCLAIMED',?,?,?,?)",
              uuid(id),
              uuid(ev),
              r.get("name"),
              r.get("mode"),
              s.crypto.hash(pair),
              s.now().plusMinutes(5),
              expiry,
              uuid(c.actorId()));
          return map(
              "device",
              deviceView(s.row("select * from ug_device where id=?", uuid(id))),
              "pairingCode",
              pair,
              "pairingExpiresAt",
              s.now().plusMinutes(5));
        }
      case "DeviceClaim":
        {
          Map d = s.row("select * from ug_device where id=?", uuid(r.get("deviceId")));
          s.event(str(d.get("event_id")));
          ApiException.require(
              "UNCLAIMED".equals(d.get("status"))
                  && time(d.get("pairing_expires_at")).isAfter(s.now())
                  && num(d.get("pairing_failures")) < 5,
              410,
              "PAIRING_EXPIRED",
              "终端领取码已失效");
          if (!MessageDigest.isEqual(
              (byte[]) d.get("pairing_digest"), s.crypto.hash(str(r.get("pairingCode"))))) {
            identity.attempts.failedPairing(d.get("id"));
            throw new ApiException(401, "PAIRING_INVALID", "终端领取码无效");
          }
          d = s.row("select * from ug_device where id=? for update", d.get("id"));
          ApiException.require(
              "UNCLAIMED".equals(d.get("status")), 410, "PAIRING_EXPIRED", "领取码已失效");
          String token = s.crypto.token();
          s.execute(
              "update ug_device set"
                  + " status='ACTIVE',session_digest=?,pairing_digest=null,version=version+1,last_seen_at=now()"
                  + " where id=?",
              s.crypto.hash(token),
              d.get("id"));
          identity.cookie("ug_device", token);
          return deviceSession(s.row("select * from ug_device where id=?", d.get("id")));
        }
      case "DeviceDetail":
        {
          return deviceSession(c.device);
        }
      case "DeviceList":
        {
          c.organizer(ev);
          s.event(ev);
          return s.page(
              s.rows("select * from ug_device where event_id=? order by name,id", uuid(ev)).stream()
                  .map(this::deviceView)
                  .toList(),
              r);
        }
      case "DeviceRevoke":
        {
          c.organizer(ev);
          s.event(ev);
          Map d =
              s.row(
                  "select * from ug_device where id=? and event_id=? for update",
                  uuid(r.get("deviceId")),
                  uuid(ev));
          s.version(d, r.get("version"));
          s.execute(
              "update ug_device set"
                  + " status='REVOKED',session_digest=null,pairing_digest=null,version=version+1"
                  + " where id=?",
              d.get("id"));
          return map();
        }
      case "TicketResolve":
        {
          Map e = s.event(ev);
          open(e);
          var cred =
              s.row(
                  "select * from ug_credential where token_digest=? and event_id=? and"
                      + " kind='TICKET' and status='ACTIVE' and expires_at>now()",
                  s.crypto.hash(str(r.get("ticketToken"))),
                  uuid(ev));
          Map p = parties.raw(ev, str(cred.get("party_id")));
          ApiException.require(
              "APPROVED".equals(p.get("status")), 409, "PARTY_STATE_CONFLICT", "无有效入场资格");
          String token =
              s.crypto.signed(
                  map(
                      "kind",
                      "SCAN",
                      "deviceId",
                      str(c.device.get("id")),
                      "partyId",
                      str(p.get("id")),
                      "eventId",
                      ev,
                      "credentialId",
                      str(cred.get("id")),
                      "credentialVersion",
                      cred.get("credential_version"),
                      "expiresAt",
                      s.now().plusMinutes(5).toString()),
                  s.json);
          return map(
              "scanContextToken",
              token,
              "eventId",
              ev,
              "approvedSize",
              p.get("approved_size"),
              "netAdmitted",
              p.get("net_admitted"),
              "remainingCount",
              num(p.get("approved_size")) - num(p.get("net_admitted")),
              "partyVersion",
              p.get("version"),
              "progress",
              s.value("AdmissionProgress", parties.progress(p)),
              "expiresAt",
              s.now().plusMinutes(5));
        }
      case "GateCommit":
        {
          Map scan = scan(str(r.get("scanContextToken")), c, true);
          ApiException.require(ev.equals(scan.get("eventId")), 404, "RESOURCE_NOT_FOUND", "未找到资源");
          Map e = s.event(ev);
          open(e);
          Map p = parties.raw(ev, str(scan.get("partyId")));
          s.version(p, r.get("expectedPartyVersion"));
          s.row(
              "select * from ug_credential where id=? and event_id=? and party_id=? and"
                  + " kind='TICKET' and status='ACTIVE' and credential_version=? and"
                  + " expires_at>now()",
              uuid(scan.get("credentialId")),
              uuid(ev),
              p.get("id"),
              scan.get("credentialVersion"));
          int count = num(r.get("count"));
          ApiException.require(
              "APPROVED".equals(p.get("status"))
                  && num(p.get("net_admitted")) + count <= num(p.get("approved_size")),
              409,
              "ADMISSION_BALANCE_CONFLICT",
              "可入场名额不足");
          s.execute(
              "update ug_party set"
                  + " net_admitted=net_admitted+?,ever_admitted=true,version=version+1,updated_at=now()"
                  + " where id=?",
              count,
              p.get("id"));
          String id = s.id();
          OffsetDateTime now = s.now();
          s.execute(
              "insert into"
                  + " ug_admission(id,event_id,party_id,operation_entry_id,device_id,actor_id,admitted_count,committed_at)"
                  + " values(?,?,?,?,?,?,?,?)",
              uuid(id),
              uuid(ev),
              p.get("id"),
              entry,
              c.device.get("id"),
              uuid(c.actorId()),
              count,
              now);
          return map(
              "operationId",
              r.get("operationId"),
              "recordId",
              id,
              "status",
              s.value("OperationStatus", "SUCCEEDED"),
              "requestedCount",
              count,
              "committedCount",
              count,
              "remainingCount",
              num(p.get("approved_size")) - num(p.get("net_admitted")) - count,
              "committedAt",
              now,
              "replayed",
              false,
              "showAsNewAdmission",
              true);
        }
      case "GateOperation":
        {
          ev = str(c.device.get("event_id"));
          s.event(ev);
          Map scan = null;
          if ("KIOSK".equals(c.device.get("mode"))) {
            ApiException.require(r.containsKey("scanContextToken"), 403, "FORBIDDEN", "缺少原组上下文");
            scan = scan(str(r.get("scanContextToken")), c, false);
          }
          var ops =
              s.rows(
                  "select o.*,a.device_id,a.party_id,a.actor_id from ug_operation o left join"
                      + " ug_admission a on a.operation_entry_id=o.id where o.operation_id=? and"
                      + " o.actor_scope='GATE' and o.event_id=? and o.action='GateCommit'",
                  uuid(r.get("operationId")),
                  uuid(ev));
          if (ops.isEmpty())
            throw new ApiException(404, "OPERATION_NOT_FOUND", "尚未查到原操作，请沿用原操作编号核对");
          Map row = ops.getFirst();
          if (scan != null)
            ApiException.require(
                str(row.get("party_id")).equals(scan.get("partyId"))
                    && str(row.get("device_id")).equals(str(c.device.get("id"))),
                404,
                "RESOURCE_NOT_FOUND",
                "未找到资源");
          else if (!"ORGANIZER".equals(c.eventRole(ev)))
            ApiException.require(
                c.actorId().equals(str(row.get("actor_id")))
                    || str(c.device.get("id")).equals(str(row.get("device_id"))),
                404,
                "RESOURCE_NOT_FOUND",
                "未找到资源");
          Map result = s.parse(s.unseal(row.get("result_cipher")).get("data"));
          result.put("replayed", true);
          result.put("showAsNewAdmission", false);
          return result;
        }
      case "GateLookup":
        {
          ApiException.require(
              "STAFF".equals(c.device.get("mode")), 403, "FORBIDDEN", "自助终端不能搜索名单");
          s.event(ev);
          Set<String> allowed = c.fields(ev);
          List<Object> candidates = new ArrayList<>();
          String query = str(r.get("query"));
          for (var p :
              s.rows(
                  "select * from ug_party where event_id=? and status='APPROVED' order by"
                      + " created_at desc,id desc",
                  uuid(ev))) {
            Map f = s.unseal(p.get("fields_cipher"));
            boolean match = str(p.get("party_reference")).contains(query);
            for (String key : allowed)
              if (f.get(key) != null && str(f.get(key)).contains(query)) match = true;
            if (!match) continue;
            String name = allowed.contains("name") ? str(f.get("name")) : null,
                phone = allowed.contains("phone") ? str(f.get("phone")) : null;
            candidates.add(
                map(
                    "partyId",
                    str(p.get("id")),
                    "partyReference",
                    p.get("party_reference"),
                    "displayNameMasked",
                    name == null ? null : name.substring(0, 1) + "**",
                    "phoneSuffix",
                    phone == null ? null : phone.substring(Math.max(0, phone.length() - 4)),
                    "approvedSize",
                    p.get("approved_size"),
                    "netAdmitted",
                    p.get("net_admitted"),
                    "remainingCount",
                    num(p.get("approved_size")) - num(p.get("net_admitted")),
                    "progress",
                    s.value("AdmissionProgress", parties.progress(p))));
            if (candidates.size() == 10) break;
          }
          return map("items", candidates, "total", candidates.size(), "page", 1, "pageSize", 10);
        }
      case "GateRecords":
        {
          c.gateHistory(ev);
          boolean lead = "ORGANIZER".equals(c.eventRole(ev));
          List<Object> result = new ArrayList<>();
          for (var a :
              s.rows(
                  "select a.*,o.operation_id from ug_admission a join ug_operation o on"
                      + " o.id=a.operation_entry_id where a.event_id=? order by a.committed_at"
                      + " desc,a.id desc",
                  uuid(ev))) {
            if (!lead
                && !c.actorId().equals(str(a.get("actor_id")))
                && (c.device == null || !str(c.device.get("id")).equals(str(a.get("device_id")))))
              continue;
            if (r.containsKey("deviceId") && !r.get("deviceId").equals(str(a.get("device_id"))))
              continue;
            if (r.containsKey("partyId") && !r.get("partyId").equals(str(a.get("party_id"))))
              continue;
            result.add(
                map(
                    "recordId",
                    str(a.get("id")),
                    "partyId",
                    str(a.get("party_id")),
                    "eventId",
                    ev,
                    "operationId",
                    str(a.get("operation_id")),
                    "deviceId",
                    str(a.get("device_id")),
                    "actorId",
                    str(a.get("actor_id")),
                    "admittedCount",
                    a.get("admitted_count"),
                    "correctedCount",
                    a.get("corrected_count"),
                    "committedAt",
                    a.get("committed_at")));
          }
          return s.page(result, r);
        }
      case "GateCorrect":
        {
          c.organizer(ev);
          Map e = s.event(ev);
          ApiException.require(
              List.of("CHECKIN_OPEN", "CHECKIN_PAUSED").contains(e.get("status")),
              409,
              "EVENT_PHASE_CONFLICT",
              "当前不能更正");
          Map a =
              s.row(
                  "select * from ug_admission where id=? and event_id=? for update",
                  uuid(r.get("recordId")),
                  uuid(ev));
          Map p = parties.raw(ev, str(a.get("party_id")));
          s.version(p, r.get("partyVersion"));
          int count = num(r.get("correctedCount"));
          ApiException.require(
              num(a.get("corrected_count")) + count <= num(a.get("admitted_count"))
                  && num(p.get("net_admitted")) >= count,
              409,
              "CORRECTION_CONFLICT",
              "更正人数超过原记录");
          s.execute(
              "update ug_admission set corrected_count=corrected_count+? where id=?",
              count,
              a.get("id"));
          s.execute(
              "update ug_party set net_admitted=net_admitted-?,version=version+1,updated_at=now()"
                  + " where id=?",
              count,
              p.get("id"));
          String id = s.id();
          s.execute(
              "insert into"
                  + " ug_admission_correction(id,event_id,party_id,original_record_id,corrected_count,reason_cipher,actor_id,operation_entry_id,created_at)"
                  + " values(?,?,?,?,?,?,?,?,now())",
              uuid(id),
              uuid(ev),
              p.get("id"),
              a.get("id"),
              count,
              s.seal(map("reason", r.get("reason"))),
              uuid(c.actorId()),
              entry);
          return map(
              "correctionId",
              id,
              "originalRecordId",
              r.get("recordId"),
              "partyId",
              str(p.get("id")),
              "correctedCount",
              count,
              "reason",
              r.get("reason"),
              "createdAt",
              s.now(),
              "operationId",
              r.get("operationId"));
        }
      default:
        throw new IllegalArgumentException(op);
    }
  }
}
