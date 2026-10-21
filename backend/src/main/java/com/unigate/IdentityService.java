package com.unigate;

import static com.unigate.Store.*;

import jakarta.servlet.http.*;
import java.security.MessageDigest;
import java.time.*;
import java.util.*;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.ApplicationRunner;
import org.springframework.context.annotation.Bean;
import org.springframework.http.ResponseCookie;
import org.springframework.stereotype.Service;

@Service
public class IdentityService {
  final Store s;
  final SecurityAttempts attempts;
  final PasswordHasher passwords = new PasswordHasher();
  final boolean secure;
  final String allowedOrigin;
  final ThreadLocal<HttpServletResponse> response = new ThreadLocal<>();

  public IdentityService(
      Store s,
      SecurityAttempts attempts,
      @Value("${unigate.cookie-secure}") boolean secure,
      @Value("${unigate.allowed-origin}") String origin) {
    this.s = s;
    this.attempts = attempts;
    this.secure = secure;
    this.allowedOrigin = origin;
  }

  @Bean
  ApplicationRunner bootstrapAccount(
      @Value("${unigate.bootstrap-username}") String user,
      @Value("${unigate.bootstrap-password}") String pass) {
    return args -> {
      if (s.db.queryForObject("select count(*) from ug_account", Integer.class) == 0) {
        ApiException.require(
            pass.length() >= 12,
            500,
            "CONFIGURATION_REQUIRED",
            "Set UNIGATE_BOOTSTRAP_PASSWORD with at least 12 characters");
        s.execute(
            "insert into ug_account(id,username,display_name,password_hash,roles)"
                + " values(?,?,?,?,?::jsonb)",
            uuid(s.id()),
            user,
            "系统管理员",
            passwords.encode(pass),
            "[\"ADMIN\",\"ORGANIZER\"]");
      }
    };
  }

  void cookie(String name, String token) {
    response
        .get()
        .addHeader(
            "Set-Cookie",
            ResponseCookie.from(name, token)
                .httpOnly(true)
                .secure(secure)
                .sameSite("ug_staff".equals(name) ? "Lax" : "Strict")
                .path("/")
                .maxAge(43200)
                .build()
                .toString());
  }

  String getCookie(HttpServletRequest request, String name) {
    if (request.getCookies() != null)
      for (var c : request.getCookies()) if (name.equals(c.getName())) return c.getValue();
    return null;
  }

  Map<String, Object> session(HttpServletRequest req, String name) {
    String token = getCookie(req, name);
    if (token == null) return null;
    var l =
        s.rows(
            "select * from ug_session where token_digest=? and revoked_at is null and"
                + " expires_at>now()",
            s.crypto.hash(token));
    return l.isEmpty() ? null : l.getFirst();
  }

  Map<String, Object> newSession(
      String kind,
      String cookieName,
      String account,
      String event,
      String party,
      Object authVersion) {
    String token = s.crypto.token(), id = s.id(), csrf = csrfFor(id);
    OffsetDateTime expires = s.now().plusHours(12);
    if (event != null) {
      var e = s.event(event);
      if (time(e.get("personal_due_at")).isBefore(expires))
        expires = time(e.get("personal_due_at"));
    }
    s.execute(
        "insert into"
            + " ug_session(id,token_digest,kind,account_id,event_id,party_id,auth_version,csrf_digest,created_at,last_seen_at,expires_at)"
            + " values(?,?,?,?,?,?,?,?,?,?,?)",
        uuid(id),
        s.crypto.hash(token),
        kind,
        uuid(account),
        uuid(event),
        uuid(party),
        authVersion,
        s.crypto.hash(csrf),
        s.now(),
        s.now(),
        expires);
    cookie(cookieName, token);
    return map("row", s.row("select * from ug_session where id=?", uuid(id)), "csrf", csrf);
  }

  public Object bootstrap(HttpServletRequest req) {
    Map<String, Object> row = session(req, "ug_anon");
    String csrf;
    if (row == null) {
      var created = newSession("ANONYMOUS", "ug_anon", null, null, null, null);
      row = (Map) created.get("row");
      csrf = str(created.get("csrf"));
    } else {
      csrf = csrfFor(str(row.get("id")));
    }
    return map(
        "csrfToken",
        csrf,
        "clientContextId",
        str(row.get("id")),
        "expiresAt",
        row.get("expires_at"));
  }

  public Context authenticate(
      Map<String, Object> op, Map<String, Object> r, HttpServletRequest req) {
    Context c = new Context(s);
    c.clientIp = req.getRemoteAddr();
    c.auth = str(op.get("auth"));
    Map staff = session(req, "ug_staff");
    if (staff != null && time(staff.get("last_seen_at")).isBefore(s.now().minusMinutes(30)))
      staff = null;
    if (staff != null) {
      var a =
          s.rows(
              "select * from ug_account where id=? and enabled=true and auth_version=?",
              staff.get("account_id"),
              staff.get("auth_version"));
      if (!a.isEmpty()) {
        c.account = a.getFirst();
        c.session = staff;
      }
    }
    String deviceToken = getCookie(req, "ug_device");
    if (deviceToken != null) {
      var ds =
          s.rows(
              "select * from ug_device where session_digest=? and status='ACTIVE' and"
                  + " expires_at>now()",
              s.crypto.hash(deviceToken));
      if (!ds.isEmpty()) c.device = ds.getFirst();
    }
    switch (c.auth) {
      case "ANON" -> {
        if ("ConnectionCallback".equals(op.get("name"))) {
          ApiException.require(c.account != null, 401, "SESSION_EXPIRED", "授权发起会话已失效");
          break;
        }
        c.session = session(req, "ug_anon");
        if (!"SessionBootstrap".equals(op.get("name")))
          ApiException.require(c.session != null, 401, "SESSION_EXPIRED", "请重新打开页面");
      }
      case "STAFF" -> {
        ApiException.require(c.account != null, 401, "SESSION_EXPIRED", "请先登录");
        if (Boolean.TRUE.equals(c.account.get("must_change_password")))
          ApiException.require(
              List.of("SessionMe", "PasswordChange", "Logout").contains(op.get("name")),
              403,
              "PASSWORD_CHANGE_REQUIRED",
              "请先修改初始密码");
        List roles = (List) op.get("roles");
        ApiException.require(
            s.list(c.account.get("roles")).stream().anyMatch(roles::contains),
            403,
            "FORBIDDEN",
            "没有操作权限");
      }
      case "PARTY", "CHANGE" -> {
        c.session = session(req, "CHANGE".equals(c.auth) ? "ug_change" : "ug_party");
        ApiException.require(c.session != null, 401, "SESSION_EXPIRED", "私密访问已失效");
        s.event(str(c.session.get("event_id")));
      }
      case "DEVICE", "GATE" -> {
        ApiException.require(c.device != null, 401, "DEVICE_UNAUTHORIZED", "请先授权终端");
        if ("STAFF".equals(c.device.get("mode")))
          ApiException.require(c.account != null, 401, "SESSION_EXPIRED", "工作人员终端需要登录");
        if (r.containsKey("eventId"))
          ApiException.require(
              str(c.device.get("event_id")).equals(r.get("eventId")),
              404,
              "RESOURCE_NOT_FOUND",
              "未找到资源");
        if ("STAFF".equals(c.device.get("mode"))) {
          ApiException.require(
              List.of("ORGANIZER", "CHECKER").contains(c.eventRole(str(c.device.get("event_id")))),
              403,
              "FORBIDDEN",
              "没有检票权限");
          c.action(str(c.device.get("event_id")), "CHECKIN");
        }
      }
      default -> throw new IllegalStateException();
    }
    if ("post".equals(op.get("method"))) {
      String origin = req.getHeader("Origin");
      String serverOrigin =
          req.getScheme()
              + "://"
              + req.getServerName()
              + ((req.getServerPort() == 80 || req.getServerPort() == 443)
                  ? ""
                  : ":" + req.getServerPort());
      ApiException.require(
          origin != null && (origin.equals(allowedOrigin) || origin.equals(serverOrigin)),
          403,
          "ORIGIN_REJECTED",
          "来源验证失败");
      String csrf = req.getHeader("X-CSRF-Token");
      boolean csrfOk = validCsrf(c.session, csrf);
      if (!csrfOk && csrf != null && Set.of("PARTY", "CHANGE").contains(c.auth)) {
        Map anon = session(req, "ug_anon");
        csrfOk = validCsrf(anon, csrf);
      }
      if (!csrfOk && csrf != null && c.device != null) {
        Map payload = s.crypto.verify(csrf, s.json);
        csrfOk =
            str(c.device.get("id")).equals(payload.get("deviceId"))
                && "CSRF".equals(payload.get("kind"))
                && time(payload.get("expiresAt")).isAfter(s.now());
      }
      ApiException.require(csrfOk, 403, "CSRF_REJECTED", "页面验证已失效，请刷新");
      if ("ANON".equals(c.auth)) {
        ApiException.require(
            str(c.session.get("id")).equals(req.getHeader("X-Client-Context-ID")),
            401,
            "SESSION_EXPIRED",
            "原匿名上下文已失效");
      }
    }
    if ("RegistrationResult".equals(op.get("name")))
      ApiException.require(
          str(c.session.get("id")).equals(req.getHeader("X-Client-Context-ID")),
          401,
          "SESSION_EXPIRED",
          "原匿名上下文已失效");
    return c;
  }

  public void touchStaffSession(HttpServletRequest request) {
    String token = getCookie(request, "ug_staff");
    if (token != null)
      s.execute(
          "update ug_session set last_seen_at=now()"
              + " where token_digest=? and kind='STAFF' and revoked_at is null and expires_at>now()"
              + " and last_seen_at>now()-interval '30 minutes'",
          s.crypto.hash(token));
  }

  String csrfFor(String sessionId) {
    return Base64.getUrlEncoder().withoutPadding()
        .encodeToString(s.crypto.hash("csrf:v1:" + sessionId));
  }

  boolean validCsrf(Map session, String csrf) {
    return session != null && csrf != null
        && (MessageDigest.isEqual(s.crypto.hash(csrf), (byte[]) session.get("csrf_digest"))
            || MessageDigest.isEqual(csrf.getBytes(java.nio.charset.StandardCharsets.US_ASCII),
                csrfFor(str(session.get("id"))).getBytes(java.nio.charset.StandardCharsets.US_ASCII)));
  }

  void identityCookieClear() {
    cookie("ug_staff", "");
  }

  public Object handle(String op, Map<String, Object> r, Context c, UUID entry) {
    switch (op) {
      case "Login":
        {
          var a =
              s.rows(
                  "select * from ug_account where username=? and enabled=true", r.get("username"));
          attempts.checkLogin(str(r.get("username")), c.clientIp);
          if (a.isEmpty()
              || !passwords.matches(
                  str(r.get("password")), str(a.getFirst().get("password_hash")))) {
            attempts.failedLogin(str(r.get("username")), c.clientIp);
            throw new ApiException(401, "AUTHENTICATION_FAILED", "用户名或密码错误");
          }
          c.account = a.getFirst();
          var ns =
              newSession(
                  "STAFF", "ug_staff", c.actorId(), null, null, c.account.get("auth_version"));
          c.session = (Map) ns.get("row");
          return sessionView(c, str(ns.get("csrf")));
        }
      case "SessionMe":
        {
          return sessionView(c, csrfFor(str(c.session.get("id"))));
        }
      case "Logout":
        {
          s.execute("update ug_session set revoked_at=now() where id=?", c.session.get("id"));
          cookie("ug_staff", "");
          return map();
        }
      case "PasswordChange":
        {
          ApiException.require(
              passwords.matches(str(r.get("oldPassword")), str(c.account.get("password_hash"))),
              401,
              "AUTHENTICATION_FAILED",
              "当前密码错误");
          s.execute(
              "update ug_account set"
                  + " password_hash=?,must_change_password=false,auth_version=auth_version+1,version=version+1"
                  + " where id=?",
              passwords.encode(str(r.get("newPassword"))),
              c.account.get("id"));
          s.execute(
              "update ug_session set revoked_at=now() where account_id=?", c.account.get("id"));
          identityCookieClear();
          return map();
        }
      case "AccountList":
        {
          c.requireAdmin();
          List<Object> all = new ArrayList<>();
          for (var a : s.rows("select * from ug_account order by username,id")) {
            String q = str(r.getOrDefault("query", ""));
            if (str(a.get("username")).contains(q) || str(a.get("display_name")).contains(q))
              all.add(accountView(a));
          }
          return s.page(all, r);
        }
      case "AccountCreate":
        {
          c.requireAdmin();
          String id = s.id();
          s.execute(
              "insert into ug_account(id,username,display_name,password_hash,roles)"
                  + " values(?,?,?,?,?::jsonb)",
              uuid(id),
              r.get("username"),
              r.get("displayName"),
              passwords.encode(str(r.get("temporaryPassword"))),
              s.stringify(r.get("roles")));
          return accountView(s.row("select * from ug_account where id=?", uuid(id)));
        }
      case "AccountUpdate":
        {
          c.requireAdmin();
          s.db.queryForObject("select pg_advisory_xact_lock(14312111)", Object.class);
          var a = s.row("select * from ug_account where id=? for update", uuid(r.get("accountId")));
          s.version(a, r.get("version"));
          boolean active = Boolean.TRUE.equals(r.get("enabled"));
          List roles = (List) r.get("roles");
          if ((!active || !roles.contains("ADMIN")) && s.list(a.get("roles")).contains("ADMIN")) {
            int count =
                s.db.queryForObject(
                    "select count(*) from ug_account where enabled=true and roles @>"
                        + " '[\"ADMIN\"]'::jsonb",
                    Integer.class);
            ApiException.require(count > 1, 409, "LAST_ADMIN", "不能禁用最后一个管理员");
          }
          s.execute(
              "update ug_account set"
                  + " display_name=?,roles=?::jsonb,enabled=?,auth_version=auth_version+1,version=version+1"
                  + " where id=?",
              r.get("displayName"),
              s.stringify(roles),
              active,
              a.get("id"));
          return accountView(s.row("select * from ug_account where id=?", a.get("id")));
        }
      case "GrantEventOptions":
        {
          c.requireAdmin();
          return s.selectorPage(
              s
                  .rows(
                      "select id,title,starts_at from ug_event where personal_due_at>now() order by"
                          + " starts_at desc,id")
                  .stream()
                  .filter(
                      e ->
                          !r.containsKey("query")
                              || str(e.get("title")).contains(str(r.get("query"))))
                  .map(
                      e ->
                          map(
                              "eventId",
                              str(e.get("id")),
                              "title",
                              e.get("title"),
                              "startsAt",
                              e.get("starts_at")))
                  .toList(),
              100,
              r);
        }
      case "MyGrantList", "GrantList":
        {
          String ev = str(r.get("eventId"));
          if ("GrantList".equals(op) && !c.admin()) c.organizer(ev);
          List<Object> list = new ArrayList<>();
          for (var g :
              s.rows(
                  "select g.*,a.display_name,a.username from ug_event_grant g join ug_account a on"
                      + " a.id=g.account_id where g.revoked_at is null and g.expires_at>now() order"
                      + " by g.event_id,g.account_id")) {
            if ("MyGrantList".equals(op)
                && (!s.list(c.account.get("roles")).contains(g.get("role"))
                    || (r.containsKey("eventId")
                        && !r.get("eventId").equals(str(g.get("event_id")))))) continue;
            if (("MyGrantList".equals(op) && c.actorId().equals(str(g.get("account_id"))))
                || ("GrantList".equals(op) && ev.equals(str(g.get("event_id")))))
              list.add(grantView(g));
          }
          return s.page(list, r);
        }
      case "GrantSave":
        {
          String ev = str(r.get("eventId"));
          var e = s.event(ev);
          Map target =
              s.row(
                  "select * from ug_account where id=? and enabled=true", uuid(r.get("accountId")));
          ApiException.require(
              s.list(target.get("roles")).contains(r.get("role")),
              422,
              "FIELD_VALIDATION_FAILED",
              "目标账号缺少对应角色");
          ApiException.require(
              ((List) r.get("allowedActions"))
                  .stream()
                      .allMatch(
                          x ->
                              Set.of("VIEW_DETAIL", "VIEW_STATISTICS", "CHECKIN", "EXPORT")
                                  .contains(x)),
              422,
              "FIELD_VALIDATION_FAILED",
              "无效授权动作");
          if (!c.admin()) {
            c.organizer(ev);
            ApiException.require(
                !"ORGANIZER".equals(r.get("role")), 403, "FORBIDDEN", "只有管理员可授予负责人");
            ApiException.require(
                c.fields(ev).containsAll((List) r.get("allowedFields")),
                403,
                "FORBIDDEN",
                "字段超出自身授权");
            ApiException.require(
                !((List) r.get("allowedActions")).contains("EXPORT"),
                403,
                "FORBIDDEN",
                "导出需管理员单独授权");
          }
          if ("READER".equals(r.get("role")))
            for (String key : List.of("recipient", "purpose", "retentionNote"))
              ApiException.require(r.containsKey(key), 422, "VALIDATION_FAILED", "缺少接收方用途信息");
          ApiException.require(
              !"CHECKER".equals(r.get("role"))
                  || !((List) r.get("allowedActions")).contains("EXPORT"),
              422,
              "VALIDATION_FAILED",
              "检票员不能导出");
          ApiException.require(
              !"READER".equals(r.get("role"))
                  || ((List) r.get("allowedActions"))
                      .stream().allMatch(x -> Set.of("VIEW_DETAIL", "VIEW_STATISTICS").contains(x)),
              422,
              "VALIDATION_FAILED",
              "校方仅可查看");
          ApiException.require(
              time(r.get("expiresAt")).isAfter(s.now()),
              422,
              "FIELD_VALIDATION_FAILED",
              "授权期限必须在未来");
          if (!c.admin()) {
            Map own = c.grant(ev);
            ApiException.require(
                !time(r.get("expiresAt")).isAfter(time(own.get("expires_at"))),
                403,
                "FORBIDDEN",
                "授权期限超出自身范围");
            ApiException.require(
                s.list(own.get("allowed_actions")).containsAll((List) r.get("allowedActions")),
                403,
                "FORBIDDEN",
                "动作超出自身授权");
          }
          String id = str(r.getOrDefault("grantId", s.id()));
          if (r.containsKey("grantId")) {
            var old =
                s.row(
                    "select * from ug_event_grant where id=? and event_id=? for update",
                    uuid(id),
                    uuid(ev));
            s.version(old, r.get("version"));
            ApiException.require(
                str(old.get("account_id")).equals(r.get("accountId"))
                    && old.get("role").equals(r.get("role")),
                422,
                "FIELD_VALIDATION_FAILED",
                "既有授权不能更换账号或角色");
          } else
            ApiException.require(
                s.rows(
                        "select id from ug_event_grant where event_id=? and account_id=? and"
                            + " role=?",
                        uuid(ev),
                        uuid(r.get("accountId")),
                        r.get("role"))
                    .isEmpty(),
                409,
                "VERSION_CONFLICT",
                "授权已存在，请按原版本更新");
          s.execute(
              "insert into"
                  + " ug_event_grant(id,event_id,account_id,role,allowed_fields,allowed_actions,disclosure_cipher,expires_at)"
                  + " values(?,?,?,?,?::jsonb,?::jsonb,?,?) on conflict(event_id,account_id,role)"
                  + " do update set"
                  + " allowed_fields=excluded.allowed_fields,allowed_actions=excluded.allowed_actions,disclosure_cipher=excluded.disclosure_cipher,expires_at=excluded.expires_at,revoked_at=null,version=ug_event_grant.version+1",
              uuid(id),
              uuid(ev),
              uuid(r.get("accountId")),
              r.get("role"),
              s.stringify(r.get("allowedFields")),
              s.stringify(r.get("allowedActions")),
              s.seal(
                  map(
                      "recipient",
                      r.get("recipient"),
                      "purpose",
                      r.get("purpose"),
                      "retentionNote",
                      r.get("retentionNote"))),
              time(r.get("expiresAt")));
          return grantView(
              s.row(
                  "select g.*,a.display_name,a.username from ug_event_grant g join ug_account a on"
                      + " a.id=g.account_id where event_id=? and account_id=? and role=?",
                  uuid(ev),
                  uuid(r.get("accountId")),
                  r.get("role")));
        }
      case "GrantRevoke":
        {
          String ev = str(r.get("eventId"));
          s.event(ev);
          if (!c.admin()) c.organizer(ev);
          var g =
              s.row(
                  "select * from ug_event_grant where id=? and event_id=? for update",
                  uuid(r.get("grantId")),
                  uuid(ev));
          s.version(g, r.get("version"));
          if (!c.admin())
            ApiException.require(!"ORGANIZER".equals(g.get("role")), 403, "FORBIDDEN", "不能撤销负责人");
          if ("ORGANIZER".equals(g.get("role")))
            ApiException.require(
                s.db.queryForObject(
                        "select count(*) from ug_event_grant where event_id=? and role='ORGANIZER'"
                            + " and revoked_at is null and expires_at>now()",
                        Integer.class,
                        uuid(ev))
                    > 1,
                409,
                "LAST_ORGANIZER",
                "不能撤销最后一位负责人");
          s.execute(
              "update ug_event_grant set revoked_at=now(),version=version+1 where id=?",
              g.get("id"));
          return map();
        }
      default:
        throw new IllegalArgumentException(op);
    }
  }

  public Map<String, Object> sessionView(Context c, String csrf) {
    return map(
        "accountId",
        c.actorId(),
        "username",
        c.account.get("username"),
        "displayName",
        c.account.get("display_name"),
        "roles",
        s.list(c.account.get("roles")).stream().map(x -> s.value("Role", x)).toList(),
        "csrfToken",
        csrf,
        "expiresAt",
        c.session.get("expires_at"),
        "mustChangePassword",
        c.account.get("must_change_password"));
  }

  Map<String, Object> accountView(Map a) {
    return map(
        "accountId",
        str(a.get("id")),
        "username",
        a.get("username"),
        "displayName",
        a.get("display_name"),
        "roles",
        s.list(a.get("roles")).stream().map(x -> s.value("Role", x)).toList(),
        "enabled",
        a.get("enabled"),
        "version",
        a.get("version"));
  }

  public Map<String, Object> grantView(Map g) {
    Map m =
        map(
            "grantId",
            str(g.get("id")),
            "accountId",
            str(g.get("account_id")),
            "eventId",
            str(g.get("event_id")),
            "role",
            s.value("Role", g.get("role")),
            "allowedFields",
            s.list(g.get("allowed_fields")).stream().map(x -> s.value("PartyField", x)).toList(),
            "allowedActions",
            s.list(g.get("allowed_actions")),
            "expiresAt",
            g.get("expires_at"),
            "version",
            g.get("version"),
            "accountDisplayName",
            g.get("display_name"),
            "username",
            g.get("username"));
    m.putAll(s.unseal(g.get("disclosure_cipher")));
    return m;
  }
}
