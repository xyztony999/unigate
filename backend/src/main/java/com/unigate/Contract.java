package com.unigate;

import com.fasterxml.jackson.databind.*;
import java.time.*;
import java.util.*;
import org.springframework.stereotype.Component;

@Component
public class Contract {
  public static Map<String, Map<String, Object>> schemas;
  public static Map<String, Map<String, String>> enums;
  public final Map<String, Map<String, Object>> routes = new HashMap<>();
  public final Map<String, Map<String, Object>> operations = new HashMap<>();

  public Contract(ObjectMapper json) {
    try {
      Map o = json.readValue(getClass().getResourceAsStream("/openapi-unigate.json"), Map.class);
      schemas = (Map) ((Map) o.get("components")).get("schemas");
      enums = (Map) o.get("x-enum-pool");
      for (Object item : (List) o.get("x-operation-index")) {
        Map<String, Object> op = (Map) item;
        routes.put(op.get("method").toString().toUpperCase() + " " + op.get("path"), op);
        operations.put(op.get("name").toString(), op);
      }
    } catch (Exception e) {
      throw new IllegalStateException("Cannot load API contract", e);
    }
  }

  public void validate(String schema, Object value) {
    validateNode(schemas.get(schema), value, schema);
  }

  public void validateNode(Map<String, Object> node, Object v, String path) {
    if (node.containsKey("$ref")) {
      validateNode(schemas.get(node.get("$ref").toString().substring(21)), v, path);
      return;
    }
    Object type = node.get("type");
    if (v == null) {
      if (type instanceof List l && l.contains("null")) return;
      throw new ApiException(422, "VALIDATION_FAILED", "字段不允许为空: " + path);
    }
    String t = type instanceof List l ? l.getFirst().toString() : String.valueOf(type);
    boolean valid =
        switch (t) {
          case "object" -> v instanceof Map;
          case "array" -> v instanceof List;
          case "string" -> v instanceof String;
          case "integer" -> v instanceof Number n && n.doubleValue() == n.longValue();
          case "number" -> v instanceof Number;
          case "boolean" -> v instanceof Boolean;
          default -> true;
        };
    ApiException.require(valid, 422, "VALIDATION_FAILED", "字段类型错误: " + path);
    if (node.get("enum") instanceof List l)
      ApiException.require(l.contains(v), 422, "VALIDATION_FAILED", "字段值无效: " + path);
    if (v instanceof String s) {
      if (node.containsKey("minLength"))
        ApiException.require(
            s.length() >= Store.num(node.get("minLength")),
            422,
            "VALIDATION_FAILED",
            "字段过短: " + path);
      if (node.containsKey("maxLength"))
        ApiException.require(
            s.length() <= Store.num(node.get("maxLength")),
            422,
            "VALIDATION_FAILED",
            "字段过长: " + path);
      try {
        switch (String.valueOf(node.get("format"))) {
          case "uuid" -> {
            if (!UUID.fromString(s).toString().equalsIgnoreCase(s))
              throw new IllegalArgumentException();
          }
          case "date-time" -> OffsetDateTime.parse(s);
          case "email" -> {
            if (!s.matches("[^\\s@]+@[^\\s@]+\\.[^\\s@]+")) throw new IllegalArgumentException();
          }
        }
      } catch (Exception e) {
        throw new ApiException(422, "VALIDATION_FAILED", "字段格式无效: " + path);
      }
    }
    if (v instanceof Number n) {
      if (node.get("minimum") instanceof Number lo)
        ApiException.require(
            n.doubleValue() >= lo.doubleValue(), 422, "VALIDATION_FAILED", "字段小于下限: " + path);
      if (node.get("maximum") instanceof Number hi)
        ApiException.require(
            n.doubleValue() <= hi.doubleValue(), 422, "VALIDATION_FAILED", "字段大于上限: " + path);
    }
    if (v instanceof Map m) {
      Map<String, Map<String, Object>> props = (Map) node.getOrDefault("properties", Map.of());
      for (Object req : (List) node.getOrDefault("required", List.of()))
        ApiException.require(
            m.containsKey(req), 422, "VALIDATION_FAILED", "缺少字段: " + path + "." + req);
      for (Object key : m.keySet()) {
        if (!props.containsKey(key)) {
          ApiException.require(
              !Boolean.FALSE.equals(node.get("additionalProperties")),
              400,
              "INVALID_REQUEST",
              "未知字段: " + path + "." + key);
          continue;
        }
        validateNode(props.get(key), m.get(key), path + "." + key);
      }
      if (node.get("dependentRequired") instanceof Map deps)
        for (Object key : deps.keySet())
          if (m.containsKey(key))
            for (Object dep : (List) deps.get(key))
              ApiException.require(m.containsKey(dep), 422, "VALIDATION_FAILED", "缺少关联字段: " + dep);
    }
    if (v instanceof List l) {
      if (node.containsKey("maxItems"))
        ApiException.require(
            l.size() <= Store.num(node.get("maxItems")), 422, "VALIDATION_FAILED", "列表过长: " + path);
      if (node.containsKey("minItems"))
        ApiException.require(
            l.size() >= Store.num(node.get("minItems")),
            422,
            "VALIDATION_FAILED",
            "列表不能为空: " + path);
      for (Object item : l) validateNode((Map) node.get("items"), item, path + "[]");
    }
  }
}
