package com.unigate;

import static com.unigate.Store.*;

import java.time.*;
import java.util.*;
import org.springframework.stereotype.Service;

@Service
public class EventService {
  final Store s;

  public EventService(Store s) {
    this.s = s;
  }

  public Map<String, Object> defaults() {
    return map(
        "fields",
        map(
            "name",
            map("enabled", true, "required", false),
            "phone",
            map("enabled", true, "required", false),
            "email",
            map("enabled", true, "required", false),
            "companionCount",
            map("enabled", true, "required", false)),
        "approvalMode",
        "IMMEDIATE",
        "phoneVerificationEnabled",
        false,
        "selfEditEnabled",
        false,
        "selfCancelEnabled",
        false,
        "retentionDays",
        30,
        "version",
        0);
  }

  public Map<String, Object> policy(Map e) {
    Map p = new LinkedHashMap(s.parse(e.get("policy")));
    p.put("version", e.get("policy_version"));
    p.put("approvalMode", s.value("ApprovalMode", p.get("approvalMode")));
    return p;
  }

  public boolean registrationAvailable(Map e) {
    return "READY".equals(e.get("status"))
        && !s.now().isBefore(time(e.get("registration_starts_at")))
        && !s.now().isAfter(time(e.get("registration_ends_at")));
  }

  public String noticeVersion(Map e) {
    return "v"
        + e.get("policy_version")
        + "-"
        + HexFormat.of()
            .formatHex(s.crypto.hash(str(e.get("id")) + s.stringify(e.get("policy"))))
            .substring(0, 24);
  }

  public Map<String, Object> view(Map e) {
    return map(
        "eventId",
        str(e.get("id")),
        "title",
        e.get("title"),
        "venue",
        e.get("venue"),
        "startsAt",
        e.get("starts_at"),
        "endsAt",
        e.get("ends_at"),
        "timeZone",
        e.get("time_zone"),
        "registrationStartsAt",
        e.get("registration_starts_at"),
        "registrationEndsAt",
        e.get("registration_ends_at"),
        "capacity",
        e.get("capacity"),
        "allocatedCount",
        e.get("allocated_count"),
        "status",
        s.value("EventStatus", e.get("status")),
        "version",
        e.get("version"));
  }

  void dates(Map r) {
    try {
      ZoneId.of(str(r.get("timeZone")));
      ApiException.require(
          !time(r.get("registrationStartsAt")).isAfter(time(r.get("registrationEndsAt")))
              && !time(r.get("registrationEndsAt")).isAfter(time(r.get("startsAt")))
              && !time(r.get("startsAt")).isAfter(time(r.get("endsAt"))),
          422,
          "VALIDATION_FAILED",
          "活动时间顺序无效");
    } catch (DateTimeException e) {
      throw new ApiException(422, "VALIDATION_FAILED", "时区无效");
    }
  }

  public Object handle(String op, Map<String, Object> r, Context c, UUID entry) {
    String ev = str(r.get("eventId"));
    switch (op) {
      case "EventCreate":
        {
          dates(r);
          String id = s.id();
          OffsetDateTime end = time(r.get("endsAt"));
          s.execute(
              "insert into"
                  + " ug_event(id,title,venue,starts_at,ends_at,time_zone,registration_starts_at,registration_ends_at,capacity,status,policy,personal_due_at,created_by)"
                  + " values(?,?,?,?,?,?,?,?,?,'DRAFT',?::jsonb,?,?)",
              uuid(id),
              r.get("title"),
              r.get("venue"),
              time(r.get("startsAt")),
              end,
              r.get("timeZone"),
              time(r.get("registrationStartsAt")),
              time(r.get("registrationEndsAt")),
              r.get("capacity"),
              s.stringify(defaults()),
              end.plusDays(30),
              uuid(c.actorId()));
          s.execute(
              "insert into"
                  + " ug_event_grant(id,event_id,account_id,role,allowed_fields,allowed_actions,expires_at)"
                  + " values(?,?,?,'ORGANIZER',?::jsonb,'[\"VIEW_DETAIL\",\"VIEW_STATISTICS\",\"CHECKIN\"]'::jsonb,?)",
              uuid(s.id()),
              uuid(id),
              uuid(c.actorId()),
              "[\"name\",\"phone\",\"email\",\"companionCount\"]",
              end.plusDays(395));
          s.execute(
              "insert into ug_cleanup(id,event_id,due_at,status,scope_results)"
                  + " values(?,?,?,'PENDING',?::jsonb)",
              uuid(s.id()),
              uuid(id),
              end.plusDays(30),
              s.stringify(
                  Contract.enums.get("CleanupScope").keySet().stream()
                      .map(
                          k ->
                              map(
                                  "scope",
                                  s.value("CleanupScope", k),
                                  "status",
                                  s.value("JobStatus", "PENDING"),
                                  "processedCount",
                                  0,
                                  "failedCount",
                                  0))
                      .toList()));
          return view(s.event(id));
        }
      case "EventList", "EventOptions":
        {
          List<Object> all = new ArrayList<>();
          for (var e :
              s.rows(
                  "select distinct e.* from ug_event e join ug_event_grant g on g.event_id=e.id"
                      + " where g.account_id=? and g.revoked_at is null and g.expires_at>now() and"
                      + " e.personal_due_at>now() and ?::jsonb @> jsonb_build_array(g.role)"
                      + " order by e.starts_at desc,e.id",
                  uuid(c.actorId()),
                  s.stringify(s.list(c.account.get("roles"))))) {
            if (r.containsKey("status") && !r.get("status").equals(e.get("status"))) continue;
            if (r.containsKey("query") && !str(e.get("title")).contains(str(r.get("query"))))
              continue;
            all.add(view(e));
          }
          return "EventOptions".equals(op) ? s.selectorPage(all, 100, r) : s.page(all, r);
        }
      case "PublicEvent":
        {
          var e = s.event(ev);
          ApiException.require(
              !"DRAFT".equals(e.get("status")), 404, "RESOURCE_NOT_FOUND", "未找到活动");
          Map p = s.parse(e.get("policy"));
          return map(
              "eventId",
              ev,
              "title",
              e.get("title"),
              "venue",
              e.get("venue"),
              "startsAt",
              e.get("starts_at"),
              "endsAt",
              e.get("ends_at"),
              "timeZone",
              e.get("time_zone"),
              "fields",
              p.get("fields"),
              "approvalMode",
              s.value("ApprovalMode", p.get("approvalMode")),
              "phoneVerificationEnabled",
              p.get("phoneVerificationEnabled"),
              "registrationAvailable",
              registrationAvailable(e),
              "notice",
              "主办方按本活动设置收集报名信息，用于报名、检票和活动通知。个人明细于活动结束后"
                  + e.get("retention_days")
                  + "天到期清理。请确认本次提交的信息及随行人数。",
              "noticeVersion",
              noticeVersion(e),
              "retentionDays",
              e.get("retention_days"));
        }
      case "EventDetail":
        {
          c.view(ev);
          return view(s.event(ev));
        }
      case "PolicyGet":
        {
          c.organizer(ev);
          return policy(s.event(ev));
        }
      case "EventUpdate":
        {
          c.organizer(ev);
          var e = s.event(ev);
          s.version(e, r.get("version"));
          ApiException.require(
              List.of("DRAFT", "READY").contains(e.get("status")),
              409,
              "EVENT_PHASE_CONFLICT",
              "当前阶段不能修改活动");
          dates(r);
          ApiException.require(
              num(r.get("capacity")) >= num(e.get("allocated_count")),
              409,
              "CAPACITY_CONFLICT",
              "容量不能少于已分配人数");
          s.execute(
              "update ug_event set"
                  + " title=?,venue=?,starts_at=?,ends_at=?,time_zone=?,registration_starts_at=?,registration_ends_at=?,capacity=?,personal_due_at=?,version=version+1"
                  + " where id=?",
              r.get("title"),
              r.get("venue"),
              time(r.get("startsAt")),
              time(r.get("endsAt")),
              r.get("timeZone"),
              time(r.get("registrationStartsAt")),
              time(r.get("registrationEndsAt")),
              r.get("capacity"),
              time(r.get("endsAt")).plusDays(num(e.get("retention_days"))),
              uuid(ev));
          return view(s.event(ev));
        }
      case "PolicyUpdate":
        {
          c.organizer(ev);
          var e = s.event(ev);
          ApiException.require(
              List.of("DRAFT", "READY").contains(e.get("status")),
              409,
              "EVENT_PHASE_CONFLICT",
              "当前阶段不能修改规则");
          Map<String, Object> p = (Map) r.get("policy");
          ApiException.require(
              num(p.get("version")) == num(e.get("policy_version")),
              409,
              "VERSION_CONFLICT",
              "规则已更新");
          Map fields = (Map) p.get("fields");
          for (Object f : fields.values()) {
            Map rule = (Map) f;
            ApiException.require(
                !Boolean.TRUE.equals(rule.get("required"))
                    || Boolean.TRUE.equals(rule.get("enabled")),
                422,
                "VALIDATION_FAILED",
                "必填字段须启用");
          }
          if (Boolean.TRUE.equals(p.get("phoneVerificationEnabled"))) {
            Map phone = (Map) fields.get("phone");
            ApiException.require(
                Boolean.TRUE.equals(phone.get("enabled"))
                    && Boolean.TRUE.equals(phone.get("required")),
                422,
                "VALIDATION_FAILED",
                "短信验证要求手机号必填");
          }
          if (p.containsKey("selfServiceEndsAt"))
            ApiException.require(
                !time(p.get("selfServiceEndsAt")).isAfter(time(e.get("registration_ends_at"))),
                422,
                "VALIDATION_FAILED",
                "自助截止不可晚于报名截止");
          Map old = s.parse(e.get("policy"));
          int count =
              s.db.queryForObject(
                  "select count(*) from ug_party where event_id=?", Integer.class, uuid(ev));
          if (count > 0)
            for (String key :
                List.of("fields", "approvalMode", "phoneVerificationEnabled", "retentionDays"))
              ApiException.require(
                  Objects.equals(old.get(key), p.get(key)), 409, "POLICY_IN_USE", "已有报名，不能修改已告知规则");
          s.execute(
              "update ug_event set"
                  + " policy=?::jsonb,policy_version=policy_version+1,retention_days=?,personal_due_at=?,version=version+1"
                  + " where id=?",
              s.stringify(p),
              p.get("retentionDays"),
              time(e.get("ends_at")).plusDays(num(p.get("retentionDays"))),
              uuid(ev));
          return policy(s.event(ev));
        }
      case "EventTransition":
        {
          c.organizer(ev);
          var e = s.event(ev);
          s.version(e, r.get("version"));
          String status = str(e.get("status"));
          String action = str(r.get("action"));
          String next =
              switch (action) {
                case "PUBLISH" -> {
                  ApiException.require(
                      "DRAFT".equals(status), 409, "EVENT_PHASE_CONFLICT", "不是草稿阶段");
                  yield "READY";
                }
                case "OPEN" -> {
                  ApiException.require(
                      "READY".equals(status), 409, "EVENT_PHASE_CONFLICT", "不是准备阶段");
                  yield "CHECKIN_OPEN";
                }
                case "PAUSE" -> {
                  ApiException.require(
                      "CHECKIN_OPEN".equals(status), 409, "EVENT_PHASE_CONFLICT", "不是检票开放阶段");
                  yield "CHECKIN_PAUSED";
                }
                case "RESUME" -> {
                  ApiException.require(
                      "CHECKIN_PAUSED".equals(status), 409, "EVENT_PHASE_CONFLICT", "不是检票暂停阶段");
                  yield "CHECKIN_OPEN";
                }
                case "END" -> {
                  ApiException.require(
                      List.of("CHECKIN_OPEN", "CHECKIN_PAUSED").contains(status),
                      409,
                      "EVENT_PHASE_CONFLICT",
                      "当前不能结束");
                  yield "ENDED";
                }
                default -> throw new ApiException(422, "VALIDATION_FAILED", "未知动作");
              };
          s.execute("update ug_event set status=?,version=version+1 where id=?", next, uuid(ev));
          return view(s.event(ev));
        }
      default:
        throw new IllegalArgumentException(op);
    }
  }
}
