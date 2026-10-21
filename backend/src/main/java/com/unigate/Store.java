package com.unigate;

import com.fasterxml.jackson.databind.ObjectMapper;
import java.time.*;
import java.util.*;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Component;

@Component
public class Store {
  public final JdbcTemplate db;
  public final ObjectMapper json;
  public final Crypto crypto;
  public final EventKeyVault keys;
  public final ThreadLocal<String> eventScope = new ThreadLocal<>();
  public final ThreadLocal<Boolean> readOnlyRequest = new ThreadLocal<>();

  public Store(JdbcTemplate db, ObjectMapper json, Crypto crypto, EventKeyVault keys) {
    this.db = db;
    this.json = json;
    this.crypto = crypto;
    this.keys = keys;
  }

  public Map<String, Object> row(String sql, Object... args) {
    var l = rows(sql, args);
    if (l.isEmpty()) throw new ApiException(404, "RESOURCE_NOT_FOUND", "未找到资源");
    return l.getFirst();
  }

  public List<Map<String, Object>> rows(String sql, Object... args) {
    return db.queryForList(sql, args);
  }

  public int execute(String sql, Object... args) {
    return db.update(sql, args);
  }

  public String id() {
    return UUID.randomUUID().toString();
  }

  public OffsetDateTime now() {
    return OffsetDateTime.now(ZoneOffset.UTC);
  }

  public Map<String, Object> parse(Object o) {
    if (o == null) return new LinkedHashMap<>();
    if (o instanceof Map) return (Map<String, Object>) o;
    try {
      return json.readValue(o.toString(), Map.class);
    } catch (Exception e) {
      throw new IllegalStateException(e);
    }
  }

  public List<Object> list(Object o) {
    if (o == null) return List.of();
    if (o instanceof List) return (List<Object>) o;
    try {
      return json.readValue(o.toString(), List.class);
    } catch (Exception e) {
      throw new IllegalStateException(e);
    }
  }

  public String stringify(Object o) {
    try {
      return json.writeValueAsString(o);
    } catch (Exception e) {
      throw new IllegalStateException(e);
    }
  }

  public byte[] seal(Object o) {
    return eventScope.get() == null
        ? crypto.seal(stringify(o))
        : keys.seal(eventScope.get(), stringify(o));
  }

  public Map<String, Object> unseal(Object o) {
    if (o == null) return new LinkedHashMap<>();
    byte[] data = (byte[]) o;
    return parse(data[0] == 2 ? keys.open(data) : crypto.open(data));
  }

  public void revokeEventKey(String event) {
    keys.revoke(event);
    eventScope.remove();
  }

  public static UUID uuid(Object o) {
    return o == null ? null : UUID.fromString(o.toString());
  }

  public static int num(Object o) {
    return o == null ? 0 : ((Number) o).intValue();
  }

  public static String str(Object o) {
    return o == null ? null : o.toString();
  }

  public static OffsetDateTime time(Object o) {
    if (o instanceof OffsetDateTime t) return t;
    if (o instanceof java.sql.Timestamp t) return t.toInstant().atOffset(ZoneOffset.UTC);
    return OffsetDateTime.parse(o.toString());
  }

  public static Map<String, Object> map(Object... pairs) {
    Map<String, Object> m = new LinkedHashMap<>();
    for (int i = 0; i < pairs.length; i += 2)
      if (pairs[i + 1] != null) m.put(pairs[i].toString(), pairs[i + 1]);
    return m;
  }

  public Map<String, Object> value(String type, Object code) {
    if (code == null) return null;
    return map(
        "code",
        code.toString(),
        "label",
        Contract.enums.get(type).getOrDefault(code.toString(), code.toString()));
  }

  public Map<String, Object> page(List<?> all, Map<String, Object> r) {
    int p = Math.max(1, num(r.getOrDefault("page", 1))),
        size = Math.min(100, Math.max(1, num(r.getOrDefault("pageSize", 20))));
    return map(
        "items",
        all.subList(
            (int) Math.min((long) (p - 1) * size, all.size()),
            (int) Math.min((long) p * size, all.size())),
        "total",
        all.size(),
        "page",
        p,
        "pageSize",
        size);
  }

  public Map<String, Object> selectorPage(List<?> all, int size, Map<String, Object> request) {
    int start = 0;
    if (request.containsKey("cursor")) {
      try {
        start =
            Integer.parseInt(
                new String(
                    java.util.Base64.getUrlDecoder().decode(str(request.get("cursor"))),
                    java.nio.charset.StandardCharsets.US_ASCII));
        ApiException.require(start >= 0, 422, "FIELD_VALIDATION_FAILED", "无效分页游标");
      } catch (IllegalArgumentException error) {
        throw new ApiException(422, "FIELD_VALIDATION_FAILED", "无效分页游标");
      }
    }
    int end = (int) Math.min((long) start + size, all.size());
    Map<String, Object> out = map("items", all.subList(Math.min(start, all.size()), end));
    if (end < all.size())
      out.put(
          "nextCursor",
          java.util.Base64.getUrlEncoder()
              .withoutPadding()
              .encodeToString(
                  Integer.toString(end).getBytes(java.nio.charset.StandardCharsets.US_ASCII)));
    return out;
  }

  public Map<String, Object> event(String id) {
    eventScope.set(id);
    keys.requireNotRevoked(id);
    boolean snapshot =
        org.springframework.transaction.support.TransactionSynchronizationManager
            .isCurrentTransactionReadOnly();
    String lock =
        snapshot ? "" : Boolean.TRUE.equals(readOnlyRequest.get()) ? " for share" : " for update";
    var e = row("select * from ug_event where id=?" + lock, uuid(id));
    if (!time(e.get("personal_due_at")).isAfter(now()) || "ARCHIVED".equals(e.get("status")))
      throw new ApiException(410, "DATA_EXPIRED", "活动明细已到期");
    return e;
  }

  public void version(Map<String, Object> row, Object expected) {
    ApiException.require(
        expected != null && num(row.get("version")) == num(expected),
        409,
        "VERSION_CONFLICT",
        "数据已变化，请刷新后确认");
  }

  public void audit(
      Context c, String eventId, String action, Object resource, Object operation, String reason) {
    execute(
        "insert into"
            + " ug_audit(id,event_id,action,actor_id,device_id,resource_id,operation_id,reason_cipher,occurred_at)"
            + " values(?,?,?,?,?,?,?,?,?)",
        uuid(id()),
        uuid(eventId),
        action,
        uuid(c.actorId()),
        c.device == null ? null : c.device.get("id"),
        uuid(resource),
        uuid(operation),
        reason == null ? null : seal(map("reason", reason)),
        now());
  }
}
