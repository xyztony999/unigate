package com.unigate;

import static com.unigate.Store.*;
import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

import com.fasterxml.jackson.databind.ObjectMapper;
import java.io.*;
import java.net.http.*;
import java.nio.charset.StandardCharsets;
import java.nio.file.Path;
import java.util.*;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;
import org.mockito.ArgumentCaptor;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.mock.env.MockEnvironment;

class ProviderAdapterTest {
  @TempDir Path directory;

  Store store() {
    var crypto = new Crypto("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=");
    return new Store(
        mock(JdbcTemplate.class),
        new ObjectMapper(),
        crypto,
        new EventKeyVault(crypto, directory.toString()));
  }

  HttpResponse<InputStream> response(int status, String text) {
    return new HttpResponse<>() {
      public int statusCode() {
        return status;
      }

      public InputStream body() {
        return new ByteArrayInputStream(text.getBytes(StandardCharsets.UTF_8));
      }

      public HttpRequest request() {
        return null;
      }

      public Optional<HttpResponse<InputStream>> previousResponse() {
        return Optional.empty();
      }

      public HttpHeaders headers() {
        return HttpHeaders.of(Map.of(), (a, b) -> true);
      }

      public Optional<javax.net.ssl.SSLSession> sslSession() {
        return Optional.empty();
      }

      public java.net.URI uri() {
        return java.net.URI.create("https://example.invalid");
      }

      public HttpClient.Version version() {
        return HttpClient.Version.HTTP_1_1;
      }
    };
  }

  @Test
  void feishuUsesOfficialReadApiAndStableProviderIds() throws Exception {
    String id = UUID.randomUUID().toString();
    var env =
        new MockEnvironment()
            .withProperty(
                "unigate.providers.connections." + id + ".access-token", "deployment-test-token")
            .withProperty("unigate.providers.connections." + id + ".resources", "app123/table123");
    var http = mock(HttpClient.class);
    when(http.send(any(HttpRequest.class), any(HttpResponse.BodyHandler.class)))
        .thenReturn(
            response(
                200,
                "{\"code\":0,\"data\":{\"items\":[{\"record_id\":\"rec_stable\",\"fields\":{\"姓名\":[{\"text\":\"家长\"}]}}],\"has_more\":false}}"));
    var adapter = new ProviderAdapter(store(), env, http);
    var rows =
        adapter.pull(
            map("id", id, "provider", "FEISHU", "status", "UNAUTHORIZED"), "app123/table123");
    assertEquals("rec_stable", rows.getFirst().id());
    assertEquals("家长", rows.getFirst().fields().get("姓名"));
    var captor = ArgumentCaptor.forClass(HttpRequest.class);
    verify(http).send(captor.capture(), any(HttpResponse.BodyHandler.class));
    assertEquals("open.feishu.cn", captor.getValue().uri().getHost());
    assertTrue(captor.getValue().uri().getPath().endsWith("/records"));
    assertEquals(
        "Bearer deployment-test-token",
        captor.getValue().headers().firstValue("Authorization").orElseThrow());
  }

  @Test
  void tencentRequiresStableIdColumnAndUsesHeaderAuthentication() throws Exception {
    String id = UUID.randomUUID().toString(), prefix = "unigate.providers.connections." + id + ".";
    var env =
        new MockEnvironment()
            .withProperty(prefix + "access-token", "test-token")
            .withProperty(prefix + "resources", "file123/sheet123")
            .withProperty(prefix + "open-id", "test-open-id")
            .withProperty(prefix + "record-id-column", "responseId");
    var http = mock(HttpClient.class);
    when(http.send(any(HttpRequest.class), any(HttpResponse.BodyHandler.class)))
        .thenReturn(
            response(
                200,
                "{\"ret\":0,\"data\":{\"gridData\":{\"rows\":[{\"values\":[{\"cellValue\":{\"text\":\"responseId\"}},{\"cellValue\":{\"text\":\"name\"}}]}]}}}"),
            response(
                200,
                "{\"ret\":0,\"data\":{\"gridData\":{\"rows\":[{\"values\":[{\"cellValue\":{\"text\":\"answer-42\"}},{\"cellValue\":{\"text\":\"Parent\"}}]}]}}}"));
    var rows =
        new ProviderAdapter(store(), env, http)
            .pull(
                map(
                    "id",
                    id,
                    "provider",
                    "TENCENT",
                    "status",
                    "UNAUTHORIZED",
                    "client_id",
                    "test-app"),
                "file123/sheet123");
    assertEquals("answer-42", rows.getFirst().id());
    var requests = ArgumentCaptor.forClass(HttpRequest.class);
    verify(http, times(2)).send(requests.capture(), any(HttpResponse.BodyHandler.class));
    for (var request : requests.getAllValues()) {
      assertEquals("docs.qq.com", request.uri().getHost());
      assertEquals("test-open-id", request.headers().firstValue("Open-Id").orElseThrow());
      assertEquals("test-app", request.headers().firstValue("Client-Id").orElseThrow());
    }
  }

  @Test
  void resourceCannotEscapeDeploymentAllowlist() {
    String id = UUID.randomUUID().toString();
    var http = mock(HttpClient.class);
    var env =
        new MockEnvironment()
            .withProperty("unigate.providers.connections." + id + ".resources", "app123/table123");
    assertThrows(
        ApiException.class,
        () ->
            new ProviderAdapter(store(), env, http)
                .fields(map("id", id, "provider", "FEISHU"), "https://internal.invalid/path"));
    verifyNoInteractions(http);
  }

  @Test
  void providerRedirectDoesNotBecomeSuccess() throws Exception {
    String id = UUID.randomUUID().toString(), prefix = "unigate.providers.connections." + id + ".";
    var env =
        new MockEnvironment()
            .withProperty(prefix + "access-token", "test-token")
            .withProperty(prefix + "resources", "app/table");
    var http = mock(HttpClient.class);
    when(http.send(any(HttpRequest.class), any(HttpResponse.BodyHandler.class)))
        .thenReturn(response(302, ""));
    assertThrows(
        ApiException.class,
        () ->
            new ProviderAdapter(store(), env, http)
                .fields(
                    map("id", id, "provider", "FEISHU", "status", "UNAUTHORIZED"), "app/table"));
  }

  @Test
  void feishuOAuthUsesRegisteredRedirectAndKeepsSecretInPostBody() throws Exception {
    Store s = store();
    String id = UUID.randomUUID().toString(), prefix = "unigate.providers.connections." + id + ".";
    String redirect = "https://unigate.example/api/v1/integration/connection/callback";
    var env =
        new MockEnvironment()
            .withProperty(prefix + "redirect-uri", redirect)
            .withProperty(prefix + "scope", "bitable:app:readonly");
    var connection =
        map(
            "id",
            id,
            "provider",
            "FEISHU",
            "client_id",
            "test-app",
            "credentials_cipher",
            s.seal(map("clientSecret", "test-client-secret")));
    var http = mock(HttpClient.class);
    when(http.send(any(HttpRequest.class), any(HttpResponse.BodyHandler.class)))
        .thenReturn(
            response(
                200,
                "{\"access_token\":\"provider-token\",\"refresh_token\":\"provider-refresh\",\"expires_in\":3600}"));
    var adapter = new ProviderAdapter(s, env, http);
    var authorization = java.net.URI.create(adapter.authorizeUrl(connection, "one-time-state"));
    assertEquals("accounts.feishu.cn", authorization.getHost());
    assertTrue(authorization.getRawQuery().contains("state=one-time-state"));
    assertFalse(authorization.toString().contains("test-client-secret"));
    var tokens = adapter.exchange(connection, "authorization-code");
    assertEquals("provider-token", tokens.get("accessToken"));
    assertEquals(3600, num(tokens.get("expiresIn")));
    var requests = ArgumentCaptor.forClass(HttpRequest.class);
    verify(http).send(requests.capture(), any(HttpResponse.BodyHandler.class));
    var request = requests.getValue();
    assertEquals("POST", request.method());
    assertEquals("open.feishu.cn", request.uri().getHost());
    assertEquals("/open-apis/authen/v2/oauth/token", request.uri().getPath());
    var subscriber = HttpResponse.BodySubscribers.ofString(StandardCharsets.UTF_8);
    request
        .bodyPublisher()
        .orElseThrow()
        .subscribe(
            new java.util.concurrent.Flow.Subscriber<java.nio.ByteBuffer>() {
              public void onSubscribe(java.util.concurrent.Flow.Subscription subscription) {
                subscriber.onSubscribe(subscription);
              }

              public void onNext(java.nio.ByteBuffer buffer) {
                subscriber.onNext(List.of(buffer));
              }

              public void onError(Throwable error) {
                subscriber.onError(error);
              }

              public void onComplete() {
                subscriber.onComplete();
              }
            });
    var body = s.parse(subscriber.getBody().toCompletableFuture().get());
    assertEquals(redirect, body.get("redirect_uri"));
    assertEquals("test-client-secret", body.get("client_secret"));
    assertEquals("authorization-code", body.get("code"));
  }

  @Test
  void tencentOAuthUsesOfficialAuthorizationAndReturnsUserIdentity() throws Exception {
    Store s = store();
    String id = UUID.randomUUID().toString(), prefix = "unigate.providers.connections." + id + ".";
    var env =
        new MockEnvironment()
            .withProperty(
                prefix + "redirect-uri",
                "https://unigate.example/api/v1/integration/connection/callback");
    var connection =
        map(
            "id",
            id,
            "provider",
            "TENCENT",
            "client_id",
            "test-app",
            "credentials_cipher",
            s.seal(map("clientSecret", "test-client-secret")));
    var http = mock(HttpClient.class);
    when(http.send(any(HttpRequest.class), any(HttpResponse.BodyHandler.class)))
        .thenReturn(
            response(
                200,
                "{\"access_token\":\"provider-token\",\"expires_in\":3600,\"user_id\":\"stable-provider-user\"}"));
    var adapter = new ProviderAdapter(s, env, http);
    var authorization = java.net.URI.create(adapter.authorizeUrl(connection, "one-time-state"));
    assertEquals("docs.qq.com", authorization.getHost());
    assertTrue(authorization.getQuery().contains("scope=all"));
    assertTrue(authorization.getQuery().contains("new_login=1"));
    assertEquals(
        "stable-provider-user", adapter.exchange(connection, "authorization-code").get("openId"));
    var requests = ArgumentCaptor.forClass(HttpRequest.class);
    verify(http).send(requests.capture(), any(HttpResponse.BodyHandler.class));
    assertEquals("GET", requests.getValue().method());
    assertEquals("/oauth/v2/token", requests.getValue().uri().getPath());
    assertEquals("docs.qq.com", requests.getValue().uri().getHost());
  }

  @Test
  void feishuResourceMetadataFollowsOpaquePaginationWithoutReadingAnswers() throws Exception {
    Store s = store();
    String id = UUID.randomUUID().toString(), prefix = "unigate.providers.connections." + id + ".";
    var env =
        new MockEnvironment()
            .withProperty(prefix + "resources", "app/table")
            .withProperty(prefix + "access-token", "test-token");
    var http = mock(HttpClient.class);
    when(http.send(any(HttpRequest.class), any(HttpResponse.BodyHandler.class)))
        .thenReturn(
            response(
                200,
                "{\"code\":0,\"data\":{\"items\":[{\"field_name\":\"Name\"}],\"has_more\":true,\"page_token\":\"next+token\"}}"),
            response(
                200,
                "{\"code\":0,\"data\":{\"items\":[{\"field_name\":\"Total\"}],\"has_more\":false}}"));
    var resources =
        new ProviderAdapter(s, env, http)
            .list(map("id", id, "provider", "FEISHU", "status", "UNAUTHORIZED"));
    assertEquals(1, resources.size());
    assertEquals(true, resources.getFirst().get("readable"));
    assertEquals("BITABLE", resources.getFirst().get("kind"));
    var requests = ArgumentCaptor.forClass(HttpRequest.class);
    verify(http, times(2)).send(requests.capture(), any(HttpResponse.BodyHandler.class));
    assertTrue(
        requests.getAllValues().stream().allMatch(x -> x.uri().getPath().endsWith("/fields")));
    assertTrue(
        requests.getAllValues().get(1).uri().getRawQuery().contains("page_token=next%2Btoken"));
  }

  @Test
  void oauthRejectsUnregisteredOrNonHttpsRedirectBeforeNetwork() {
    Store s = store();
    String id = UUID.randomUUID().toString();
    var env =
        new MockEnvironment()
            .withProperty(
                "unigate.providers.connections." + id + ".redirect-uri",
                "http://internal.invalid/callback");
    var http = mock(HttpClient.class);
    var adapter = new ProviderAdapter(s, env, http);
    assertThrows(
        ApiException.class,
        () ->
            adapter.authorizeUrl(
                map(
                    "id",
                    id,
                    "provider",
                    "TENCENT",
                    "client_id",
                    "test-app",
                    "credentials_cipher",
                    s.seal(map("clientSecret", "test-secret"))),
                "state"));
    verifyNoInteractions(http);
  }
}
