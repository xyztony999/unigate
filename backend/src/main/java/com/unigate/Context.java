package com.unigate;

import static com.unigate.Store.*;

import java.util.*;

public class Context {
  public Map<String, Object> account, session, device;
  public final Store s;
  public String csrfToken;
  public String auth;
  public String clientIp;

  public Context(Store s) {
    this.s = s;
  }

  public String actorId() {
    return account == null ? null : str(account.get("id"));
  }

  public boolean admin() {
    return account != null && s.list(account.get("roles")).contains("ADMIN");
  }

  public Map<String, Object> grant(String eventId) {
    if (account == null) throw new ApiException(401, "SESSION_EXPIRED", "请先登录");
    var rows =
        s.rows(
            "select * from ug_event_grant where event_id=? and account_id=? and revoked_at is null"
                + " and expires_at>now() order by case role when 'ORGANIZER' then 0 when 'CHECKER'"
                + " then 1 else 2 end",
            uuid(eventId),
            uuid(actorId()));
    rows = rows.stream().filter(g -> s.list(account.get("roles")).contains(g.get("role"))).toList();
    if (rows.isEmpty()) throw new ApiException(404, "RESOURCE_NOT_FOUND", "未找到资源");
    return rows.getFirst();
  }

  public String eventRole(String id) {
    return str(grant(id).get("role"));
  }

  public void organizer(String id) {
    ApiException.require("ORGANIZER".equals(eventRole(id)), 403, "FORBIDDEN", "需要活动负责人权限");
  }

  public void view(String id) {
    var g = grant(id);
    ApiException.require(
        s.list(g.get("allowed_actions")).stream()
                .anyMatch(x -> Set.of("VIEW_DETAIL", "VIEW_STATISTICS").contains(x))
            || "ORGANIZER".equals(g.get("role")),
        403,
        "FORBIDDEN",
        "没有查看权限");
    s.event(id);
  }

  public void action(String id, String action) {
    Map g = grant(id);
    ApiException.require(
        s.list(g.get("allowed_actions")).contains(action), 403, "FORBIDDEN", "当前授权不包含该操作");
  }

  public void gateHistory(String eventId) {
    if ("CHECKER".equals(eventRole(eventId))) action(eventId, "CHECKIN");
    else organizer(eventId);
    s.event(eventId);
  }

  public Set<String> fields(String id) {
    return new LinkedHashSet<>(
        s.list(grant(id).get("allowed_fields")).stream().map(Object::toString).toList());
  }

  public void requireAdmin() {
    ApiException.require(admin(), 403, "FORBIDDEN", "需要系统管理员权限");
  }
}
