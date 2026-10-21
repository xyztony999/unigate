package com.unigate;

import static com.unigate.Store.*;

import java.io.*;
import java.net.*;
import java.net.http.*;
import java.time.Duration;
import java.util.*;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.core.env.Environment;
import org.springframework.stereotype.Component;

/**
 * Official read-only provider APIs. Resource allowlists and secrets are deployment configuration.
 */
@Component
public class ProviderAdapter {
  private final Store s;
  private final Environment env;
  private final HttpClient http;

  @Autowired
  public ProviderAdapter(Store s, Environment env) {
    this(
        s,
        env,
        HttpClient.newBuilder()
            .connectTimeout(Duration.ofSeconds(8))
            .followRedirects(HttpClient.Redirect.NEVER)
            .build());
  }

  ProviderAdapter(Store s, Environment env, HttpClient http) {
    this.s = s;
    this.env = env;
    this.http = http;
  }

  private String setting(Map<String, Object> connection, String name) {
    return env.getProperty(
        "unigate.providers.connections." + connection.get("id") + "." + name, "");
  }

  private String token(Map<String, Object> connection) {
    String token = str(s.unseal(connection.get("credentials_cipher")).get("accessToken"));
    if (token == null || token.isBlank()) token = setting(connection, "access-token");
    ApiException.require(
        token != null
            && !token.isBlank()
            && !Set.of("REVOKED", "EXPIRED").contains(str(connection.get("status")))
            && (connection.get("expires_at") == null
                || time(connection.get("expires_at")).isAfter(s.now())),
        503,
        "PROVIDER_UNAVAILABLE",
        "连接未配置有效的平台令牌");
    return token;
  }

  public String redirect(Map<String, Object> c) {
    String url = setting(c, "redirect-uri");
    URI uri;
    try {
      uri = URI.create(url);
    } catch (Exception ex) {
      throw new ApiException(503, "PROVIDER_UNAVAILABLE", "授权回调地址未配置");
    }
    ApiException.require(
        "https".equals(uri.getScheme())
            && uri.getUserInfo() == null
            && uri.getFragment() == null
            && uri.getQuery() == null
            && "/api/v1/integration/connection/callback".equals(uri.getPath()),
        503,
        "PROVIDER_UNAVAILABLE",
        "需配置已向提供方登记的HTTPS回调地址");
    return url;
  }

  public String authorizeUrl(Map<String, Object> c, String state) {
    String provider = str(c.get("provider"));
    ApiException.require(
        Set.of("FEISHU", "TENCENT").contains(provider)
            && c.get("client_id") != null
            && s.unseal(c.get("credentials_cipher")).containsKey("clientSecret"),
        503,
        "PROVIDER_UNAVAILABLE",
        "请先配置提供方应用凭据");
    String scope = provider.equals("FEISHU") ? setting(c, "scope") : "all";
    ApiException.require(!scope.isBlank(), 503, "PROVIDER_UNAVAILABLE", "请按已审核权限配置授权范围");
    return (provider.equals("FEISHU")
            ? "https://accounts.feishu.cn/open-apis/authen/v1/authorize"
            : "https://docs.qq.com/oauth/v2/authorize")
        + "?client_id="
        + encode(str(c.get("client_id")))
        + "&redirect_uri="
        + encode(redirect(c))
        + "&response_type=code&scope="
        + encode(scope)
        + "&state="
        + encode(state)
        + (provider.equals("TENCENT") ? "&new_login=1" : "");
  }

  public Map<String, Object> exchange(Map<String, Object> c, String code) {
    try {
      String provider = str(c.get("provider")),
          secret = str(s.unseal(c.get("credentials_cipher")).get("clientSecret"));
      var body =
          map(
              "grant_type",
              "authorization_code",
              "client_id",
              c.get("client_id"),
              "client_secret",
              secret,
              "redirect_uri",
              redirect(c),
              "code",
              code);
      HttpRequest request;
      if (provider.equals("FEISHU")) {
        request =
            HttpRequest.newBuilder(
                    URI.create("https://open.feishu.cn/open-apis/authen/v2/oauth/token"))
                .header("Content-Type", "application/json")
                .timeout(Duration.ofSeconds(15))
                .POST(HttpRequest.BodyPublishers.ofString(s.stringify(body)))
                .build();
      } else {
        String query =
            body.entrySet().stream()
                .map(e -> encode(e.getKey()) + "=" + encode(str(e.getValue())))
                .collect(java.util.stream.Collectors.joining("&"));
        request =
            HttpRequest.newBuilder(URI.create("https://docs.qq.com/oauth/v2/token?" + query))
                .timeout(Duration.ofSeconds(15))
                .GET()
                .build();
      }
      var response = http.send(request, HttpResponse.BodyHandlers.ofInputStream());
      Map<String, Object> result;
      try (var stream = response.body()) {
        byte[] bytes = stream.readNBytes(65537);
        ApiException.require(
            bytes.length <= 65536 && response.statusCode() == 200,
            503,
            "PROVIDER_UNAVAILABLE",
            "提供方令牌交换失败");
        result = s.parse(new String(bytes, java.nio.charset.StandardCharsets.UTF_8));
      }
      ApiException.require(
          result.get("access_token") != null && num(result.get("expires_in")) > 0,
          503,
          "PROVIDER_UNAVAILABLE",
          "提供方授权未成功");
      return map(
          "accessToken",
          result.get("access_token"),
          "refreshToken",
          result.get("refresh_token"),
          "openId",
          result.get("user_id"),
          "expiresIn",
          result.get("expires_in"));
    } catch (ApiException ex) {
      throw ex;
    } catch (InterruptedException ex) {
      Thread.currentThread().interrupt();
      throw new ApiException(503, "PROVIDER_UNAVAILABLE", "授权请求中断，请重新授权");
    } catch (Exception ex) {
      throw new ApiException(503, "PROVIDER_UNAVAILABLE", "提供方授权失败，请重新授权");
    }
  }

  private List<String> resources(Map<String, Object> connection) {
    String raw = setting(connection, "resources");
    ApiException.require(!raw.isBlank(), 503, "PROVIDER_UNAVAILABLE", "尚未配置经过授权的资源清单");
    return Arrays.stream(raw.split(",")).map(String::strip).filter(x -> !x.isBlank()).toList();
  }

  public List<Map<String, Object>> list(Map<String, Object> connection) {
    token(connection);
    List<Map<String, Object>> result = new ArrayList<>();
    for (String resource : resources(connection)) {
      fields(connection, resource);
      result.add(
          map(
              "resourceId",
              resource,
              "title",
              resource,
              "kind",
              "FEISHU".equals(connection.get("provider")) ? "BITABLE" : "SPREADSHEET",
              "readable",
              true));
    }
    return result;
  }

  private String[] resource(Map<String, Object> c, String resource) {
    ApiException.require(
        resources(c).contains(resource), 403, "RESOURCE_FORBIDDEN", "资源不在当前连接授权范围内");
    String[] parts = resource.split("/");
    ApiException.require(
        parts.length == 2 && Arrays.stream(parts).allMatch(x -> x.matches("[A-Za-z0-9_-]{1,150}")),
        422,
        "RESOURCE_ID_INVALID",
        "资源标识格式应为appToken/tableId或fileId/sheetId");
    return parts;
  }

  public List<Map<String, Object>> fields(Map<String, Object> c, String resource) {
    String[] p = resource(c, resource);
    if ("FEISHU".equals(c.get("provider"))) {
      List<Map<String, Object>> out = new ArrayList<>();
      String cursor = "";
      do {
        var result =
            get(
                c,
                "https://open.feishu.cn/open-apis/bitable/v1/apps/"
                    + p[0]
                    + "/tables/"
                    + p[1]
                    + "/fields?page_size=100"
                    + (cursor.isBlank() ? "" : "&page_token=" + encode(cursor)));
        var data = s.parse(result.get("data"));
        for (Object x : s.list(data.get("items"))) {
          var field = s.parse(x);
          String name = str(field.get("field_name"));
          out.add(map("code", name, "label", name));
        }
        cursor = Boolean.TRUE.equals(data.get("has_more")) ? str(data.get("page_token")) : "";
        ApiException.require(out.size() <= 200, 422, "PROVIDER_LIMIT", "来源最多支持200列");
      } while (cursor != null && !cursor.isBlank());
      return out;
    }
    if ("TENCENT".equals(c.get("provider"))) {
      var rows = tencentRows(c, p, "A1:GR1");
      ApiException.require(!rows.isEmpty(), 422, "PROVIDER_DATA_INVALID", "来源缺少表头");
      var headers = rows.getFirst();
      ApiException.require(
          new HashSet<>(headers).size() == headers.size()
              && headers.stream().allMatch(x -> !x.isBlank()),
          422,
          "PROVIDER_DATA_INVALID",
          "来源表头须唯一非空");
      return headers.stream().map(x -> map("code", x, "label", x)).toList();
    }
    throw new ApiException(422, "INVALID_PROVIDER", "此连接不支持名单读取");
  }

  public record RemoteRow(String id, Map<String, Object> fields) {}

  public List<RemoteRow> pull(Map<String, Object> c, String resource) {
    String[] p = resource(c, resource);
    List<RemoteRow> out = new ArrayList<>();
    if ("FEISHU".equals(c.get("provider"))) {
      String cursor = "";
      do {
        var result =
            get(
                c,
                "https://open.feishu.cn/open-apis/bitable/v1/apps/"
                    + p[0]
                    + "/tables/"
                    + p[1]
                    + "/records?page_size=500"
                    + (cursor.isBlank() ? "" : "&page_token=" + encode(cursor)));
        var data = s.parse(result.get("data"));
        for (Object x : s.list(data.get("items"))) {
          var record = s.parse(x);
          Map<String, Object> normalized = new LinkedHashMap<>();
          for (var entry : s.parse(record.get("fields")).entrySet())
            normalized.put(entry.getKey(), plain(entry.getValue()));
          out.add(new RemoteRow(str(record.get("record_id")), normalized));
        }
        cursor = Boolean.TRUE.equals(data.get("has_more")) ? str(data.get("page_token")) : "";
        ApiException.require(out.size() <= 10000, 422, "PROVIDER_LIMIT", "来源最多支持10000行");
      } while (cursor != null && !cursor.isBlank());
    } else if ("TENCENT".equals(c.get("provider"))) {
      List<String> headers = fields(c, resource).stream().map(x -> str(x.get("code"))).toList();
      String keyColumn = setting(c, "record-id-column");
      ApiException.require(
          headers.contains(keyColumn), 422, "SOURCE_ID_REQUIRED", "腾讯名单需显式配置稳定记录ID列，不能用可移动行号识别人员");
      int width = headers.size(), block = Math.min(1000, 10000 / Math.max(width, 1));
      String last = excelColumn(width);
      boolean finished = false;
      for (int start = 2; start <= 10001 && !finished; start += block) {
        int end = Math.min(10002, start + block - 1);
        var data = tencentRows(c, p, "A" + start + ":" + last + end);
        if (data.isEmpty()) break;
        for (var values : data) {
          Map<String, Object> row = new LinkedHashMap<>();
          for (int j = 0; j < headers.size(); j++)
            row.put(headers.get(j), j < values.size() ? values.get(j) : "");
          if (row.values().stream().allMatch(x -> str(x).isBlank())) {
            finished = true;
            continue;
          }
          ApiException.require(!finished, 422, "PROVIDER_DATA_INVALID", "名单中间存在空行，请整理连续范围");
          String id = str(row.get(keyColumn));
          ApiException.require(
              id != null && !id.isBlank(), 422, "SOURCE_ID_REQUIRED", "来源记录ID不能为空");
          out.add(new RemoteRow(id, row));
          ApiException.require(out.size() <= 10000, 422, "PROVIDER_LIMIT", "来源最多支持10000行");
        }
        if (data.size() < end - start + 1) break;
      }
    } else throw new ApiException(422, "INVALID_PROVIDER", "此连接不支持名单读取");
    ApiException.require(
        out.stream().map(RemoteRow::id).distinct().count() == out.size(),
        422,
        "SOURCE_ID_CONFLICT",
        "来源记录ID必须唯一");
    return out;
  }

  private String plain(Object v) {
    if (v == null) return "";
    if (v instanceof Number)
      return new java.math.BigDecimal(v.toString()).stripTrailingZeros().toPlainString();
    if (v instanceof String) return v.toString();
    if (v instanceof List<?> l) {
      StringBuilder out = new StringBuilder();
      for (Object x : l) {
        if (x instanceof Map<?, ?> m && m.containsKey("text")) out.append(m.get("text"));
        else throw new ApiException(422, "PROVIDER_DATA_INVALID", "来源包含不支持的复杂字段，请选择文本或数值字段");
      }
      return out.toString();
    }
    throw new ApiException(422, "PROVIDER_DATA_INVALID", "来源包含不支持的复杂字段");
  }

  private List<List<String>> tencentRows(Map<String, Object> c, String[] p, String range) {
    var result =
        get(
            c,
            "https://docs.qq.com/openapi/spreadsheet/v3/files/" + p[0] + "/" + p[1] + "/" + range);
    var data = s.parse(result.get("data"));
    var grid = s.parse(data.get("gridData"));
    List<List<String>> rows = new ArrayList<>();
    for (Object x : s.list(grid.get("rows"))) {
      List<String> values = new ArrayList<>();
      for (Object y : s.list(s.parse(x).get("values"))) {
        var cell = s.parse(y);
        var value = s.parse(cell.get("cellValue"));
        ApiException.require(
            !value.containsKey("formula"), 422, "PROVIDER_DATA_INVALID", "来源公式单元格不可导入");
        Object text = value.getOrDefault("text", value.getOrDefault("number", ""));
        values.add(plain(text));
      }
      while (!values.isEmpty() && values.getLast().isBlank()) values.removeLast();
      rows.add(values);
    }
    return rows;
  }

  private Map<String, Object> get(Map<String, Object> c, String url) {
    try {
      var builder =
          HttpRequest.newBuilder(URI.create(url))
              .timeout(Duration.ofSeconds(15))
              .GET()
              .header("Accept", "application/json");
      if ("FEISHU".equals(c.get("provider"))) builder.header("Authorization", "Bearer " + token(c));
      else {
        String openId =
            str(
                s.unseal(c.get("credentials_cipher"))
                    .getOrDefault("openId", setting(c, "open-id")));
        ApiException.require(
            !openId.isBlank() && c.get("client_id") != null,
            503,
            "PROVIDER_UNAVAILABLE",
            "腾讯连接尚未配置应用或用户身份");
        builder
            .header("Access-Token", token(c))
            .header("Open-Id", openId)
            .header("Client-Id", str(c.get("client_id")));
      }
      var response = http.send(builder.build(), HttpResponse.BodyHandlers.ofInputStream());
      try (var body = response.body()) {
        byte[] bytes = body.readNBytes(10 * 1024 * 1024 + 1);
        ApiException.require(bytes.length <= 10 * 1024 * 1024, 503, "PROVIDER_LIMIT", "提供方响应超出限制");
        ApiException.require(
            response.statusCode() == 200,
            response.statusCode() == 429 ? 429 : 503,
            "PROVIDER_UNAVAILABLE",
            "提供方读取失败，请检查授权状态");
        var result = s.parse(new String(bytes, java.nio.charset.StandardCharsets.UTF_8));
        Object code = result.getOrDefault("code", result.getOrDefault("ret", 0));
        ApiException.require(num(code) == 0, 503, "PROVIDER_UNAVAILABLE", "提供方拒绝读取，请检查权限");
        return result;
      }
    } catch (ApiException ex) {
      throw ex;
    } catch (InterruptedException ex) {
      Thread.currentThread().interrupt();
      throw new ApiException(503, "PROVIDER_UNAVAILABLE", "提供方请求中断");
    } catch (Exception ex) {
      throw new ApiException(503, "PROVIDER_UNAVAILABLE", "暂时无法连接提供方");
    }
  }

  private static String encode(String x) {
    return URLEncoder.encode(x, java.nio.charset.StandardCharsets.UTF_8);
  }

  private static String excelColumn(int n) {
    String out = "";
    while (n > 0) {
      n--;
      out = (char) ('A' + n % 26) + out;
      n /= 26;
    }
    return out;
  }
}
