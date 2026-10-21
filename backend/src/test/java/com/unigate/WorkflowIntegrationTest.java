package com.unigate;

import static com.unigate.Store.*;
import static org.junit.jupiter.api.Assertions.*;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.*;

import com.fasterxml.jackson.databind.ObjectMapper;
import jakarta.servlet.http.Cookie;
import java.time.*;
import java.util.*;
import java.util.concurrent.*;
import org.junit.jupiter.api.*;
import org.junit.jupiter.api.condition.EnabledIfEnvironmentVariable;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.DynamicPropertyRegistry;
import org.springframework.test.context.DynamicPropertySource;
import org.springframework.test.web.servlet.*;

@SpringBootTest(
    properties = {
      "unigate.master-key=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
      "unigate.cookie-secure=false",
      "unigate.bootstrap-password=Integration-test-123!",
      "unigate.bootstrap-username=integration-admin",
      "unigate.notifications.email-enabled=false",
      "spring.flyway.schemas=unigate_test",
      "spring.flyway.default-schema=unigate_test"
    })
@AutoConfigureMockMvc
@EnabledIfEnvironmentVariable(named = "UNIGATE_TEST_DB_URL", matches = ".+")
class WorkflowIntegrationTest {
  @DynamicPropertySource
  static void db(DynamicPropertyRegistry r) {
    r.add(
        "spring.datasource.url",
        () ->
            System.getenv("UNIGATE_TEST_DB_URL")
                + (System.getenv("UNIGATE_TEST_DB_URL").contains("?") ? "&" : "?")
                + "currentSchema=unigate_test");
    r.add(
        "spring.datasource.username",
        () -> System.getenv().getOrDefault("UNIGATE_TEST_DB_USER", "unigate"));
    r.add(
        "spring.datasource.password",
        () -> System.getenv().getOrDefault("UNIGATE_TEST_DB_PASSWORD", ""));
  }

  @Autowired MockMvc mvc;
  @Autowired Store store;
  @Autowired ObjectMapper json;
  @Autowired Contract contract;

  class Browser {
    Map<String, Cookie> cookies = new HashMap<>();
    String csrf, context;

    Map<String, Object> request(String name, Map<String, Object> args, int expected)
        throws Exception {
      Map op = contract.operations.get(name);
      var builder =
          "get".equals(op.get("method"))
              ? get(op.get("path").toString())
              : post(op.get("path").toString());
      if ("get".equals(op.get("method")))
        for (var e : args.entrySet()) builder.param(e.getKey(), e.getValue().toString());
      else builder.contentType("application/json").content(json.writeValueAsBytes(args));
      builder.header("Origin", "http://localhost:5173");
      if (csrf != null) builder.header("X-CSRF-Token", csrf);
      if (context != null) builder.header("X-Client-Context-ID", context);
      if (!cookies.isEmpty()) builder.cookie(cookies.values().toArray(Cookie[]::new));
      MvcResult result = mvc.perform(builder).andReturn();
      Map<String, Object> response =
          json.readValue(result.getResponse().getContentAsByteArray(), Map.class);
      assertEquals(expected, result.getResponse().getStatus(), name + ": " + response);
      for (Cookie cookie : result.getResponse().getCookies()) cookies.put(cookie.getName(), cookie);
      Map data = (Map) response.get("data");
      if (data.get("csrfToken") != null) csrf = data.get("csrfToken").toString();
      if (data.get("clientContextId") != null) context = data.get("clientContextId").toString();
      return data;
    }

    void bootstrap() throws Exception {
      request("SessionBootstrap", map(), 200);
    }

    Map write(String op, Map args, int status) throws Exception {
      Map r = new LinkedHashMap(args);
      r.putIfAbsent("operationId", UUID.randomUUID().toString());
      return request(op, r, status);
    }
  }

  @BeforeEach
  void resetOnlyIsolatedTestAccount() {
    store.execute(
        "update ug_account set"
            + " password_hash=?,must_change_password=true,auth_version=auth_version+1 where"
            + " username='integration-admin'",
        new org.springframework.security.crypto.bcrypt.BCryptPasswordEncoder(12)
            .encode("Integration-test-123!"));
  }

  Browser admin() throws Exception {
    Browser b = new Browser();
    b.bootstrap();
    b.write(
        "Login", map("username", "integration-admin", "password", "Integration-test-123!"), 200);
    return b;
  }

  Browser readyAdmin() throws Exception {
    Browser a = admin();
    a.write(
        "PasswordChange",
        map("oldPassword", "Integration-test-123!", "newPassword", "Changed-integration-123!"),
        200);
    a.request("SessionMe", map(), 401);
    a.bootstrap();
    a.write(
        "Login", map("username", "integration-admin", "password", "Changed-integration-123!"), 200);
    return a;
  }

  Map<String, Object> event(Browser a, int capacity) throws Exception {
    OffsetDateTime now = OffsetDateTime.now(ZoneOffset.UTC);
    return a.write(
        "EventCreate",
        map(
            "title",
            "Concurrent " + UUID.randomUUID(),
            "startsAt",
            now.plusHours(2).toString(),
            "endsAt",
            now.plusHours(5).toString(),
            "timeZone",
            "Asia/Shanghai",
            "registrationStartsAt",
            now.minusHours(1).toString(),
            "registrationEndsAt",
            now.plusHours(1).toString(),
            "capacity",
            capacity),
        200);
  }

  Map<String, Object> publish(Browser a, String ev) throws Exception {
    Map e = a.request("EventDetail", map("eventId", ev), 200);
    return a.write(
        "EventTransition",
        map("eventId", ev, "version", e.get("version"), "action", "PUBLISH"),
        200);
  }

  Map<String, Object> registration(Browser b, String ev, int companions) throws Exception {
    Map pub = b.request("PublicEvent", map("eventId", ev), 200);
    return b.write(
        "SubmitRegistration",
        map(
            "eventId",
            ev,
            "fields",
            map("companionCount", companions),
            "acceptedNoticeVersion",
            pub.get("noticeVersion")),
        200);
  }

  Map<String, Object> newAccount(Browser admin, String role) throws Exception {
    return admin.write(
        "AccountCreate",
        map(
            "username",
            "staff-" + UUID.randomUUID(),
            "displayName",
            "Test " + role,
            "temporaryPassword",
            "Temporary-staff-123!",
            "roles",
            List.of(role)),
        200);
  }

  Browser staff(Map<String, Object> account) throws Exception {
    Browser browser = new Browser();
    browser.bootstrap();
    browser.write(
        "Login", map("username", account.get("username"), "password", "Temporary-staff-123!"), 200);
    browser.write(
        "PasswordChange",
        map("oldPassword", "Temporary-staff-123!", "newPassword", "Changed-staff-123!"),
        200);
    browser.bootstrap();
    browser.write(
        "Login", map("username", account.get("username"), "password", "Changed-staff-123!"), 200);
    return browser;
  }

  @Test
  void schoolReaderFieldsActionsAndRevocationAreEnforcedInQueries() throws Exception {
    Browser a = readyAdmin();
    String ev = event(a, 10).get("eventId").toString();
    publish(a, ev);
    Browser participant = new Browser();
    participant.bootstrap();
    Map pub = participant.request("PublicEvent", map("eventId", ev), 200);
    Map sub =
        participant.write(
            "SubmitRegistration",
            map(
                "eventId",
                ev,
                "fields",
                map("name", "Visible Parent", "phone", "+61 400 777 555"),
                "acceptedNoticeVersion",
                pub.get("noticeVersion")),
            200);
    String party = ((Map) sub.get("party")).get("partyId").toString();
    Map<String, Object> account = newAccount(a, "READER");
    Browser reader = staff(account);
    reader.request("PartyList", map("eventId", ev), 404);
    Map<String, Object> grant =
        a.write(
            "GrantSave",
            map(
                "eventId",
                ev,
                "accountId",
                account.get("accountId"),
                "role",
                "READER",
                "allowedFields",
                List.of("name"),
                "allowedActions",
                List.of("VIEW_DETAIL"),
                "recipient",
                "School test reader",
                "purpose",
                "Event follow-up",
                "retentionNote",
                "Delete after review",
                "expiresAt",
                OffsetDateTime.now(ZoneOffset.UTC).plusDays(1).toString()),
            200);
    Map detail = reader.request("PartyDetail", map("eventId", ev, "partyId", party), 200);
    assertEquals(Set.of("name"), ((Map) detail.get("fields")).keySet());
    assertEquals(
        0, num(reader.request("PartyList", map("eventId", ev, "query", "777"), 200).get("total")));
    reader.request("Statistics", map("eventId", ev), 403);
    reader.write(
        "ExportCreate",
        map(
            "eventId",
            ev,
            "format",
            "CSV",
            "columns",
            List.of("name"),
            "purpose",
            "No export right"),
        403);
    reader.write(
        "PartyCancel",
        map("eventId", ev, "partyId", party, "version", 0, "reason", "Forbidden"),
        403);
    assertEquals(1, num(reader.request("MyGrantList", map("eventId", ev), 200).get("total")));
    a.request("GrantList", map("eventId", ev), 200);
    a.request("GrantEventOptions", map("query", "Concurrent"), 200);
    a.write(
        "GrantRevoke",
        map("eventId", ev, "grantId", grant.get("grantId"), "version", grant.get("version")),
        200);
    reader.request("PartyDetail", map("eventId", ev, "partyId", party), 404);
    assertEquals(0, num(reader.request("MyGrantList", map("eventId", ev), 200).get("total")));
    Map self = a.request("SessionMe", map(), 200);
    Map accounts = a.request("AccountList", map("query", account.get("username")), 200);
    Map current = (Map) ((List) accounts.get("items")).getFirst();
    a.write(
        "AccountUpdate",
        map(
            "accountId",
            account.get("accountId"),
            "version",
            current.get("version"),
            "displayName",
            "Disabled reader",
            "enabled",
            false,
            "roles",
            List.of("READER")),
        200);
    reader.request("SessionMe", map(), 401);
    a.write(
        "AccountUpdate",
        map(
            "accountId",
            self.get("accountId"),
            "version",
            store
                .row("select version from ug_account where id=?", uuid(self.get("accountId")))
                .get("version"),
            "displayName",
            "Last admin",
            "enabled",
            false,
            "roles",
            List.of("ADMIN", "ORGANIZER")),
        409);
  }

  @Test
  void selfServiceChangesAndCancellationNeverSurviveTheirGuards() throws Exception {
    Browser a = readyAdmin();
    String ev = event(a, 10).get("eventId").toString();
    Map<String, Object> policy = a.request("PolicyGet", map("eventId", ev), 200);
    policy.put("approvalMode", "IMMEDIATE");
    policy.put("selfEditEnabled", true);
    policy.put("selfCancelEnabled", true);
    a.write("PolicyUpdate", map("eventId", ev, "policy", policy), 200);
    publish(a, ev);
    Browser p = new Browser();
    p.bootstrap();
    Map sub = registration(p, ev, 1);
    Map access = (Map) sub.get("access");
    p.write(
        "PartyExchange",
        map("queryToken", access.get("queryToken"), "changeToken", access.get("changeToken")),
        200);
    Map state = p.request("MyParty", map(), 200);
    assertEquals(true, state.get("canChange"));
    Map changed =
        p.write("MyPartyChange", map("version", state.get("version"), "companionCount", 2), 200);
    assertEquals(3, num(changed.get("approvedSize")));
    p.write("MyPartyChange", map("version", state.get("version"), "companionCount", 3), 409);
    p.write("MyPartyCancel", map("version", changed.get("version")), 200);
    assertEquals(0, num(a.request("Statistics", map("eventId", ev), 200).get("allocatedPeople")));
    Map cred = p.request("MyCredential", map(), 200);
    assertFalse(cred.containsKey("ticketToken"));
    Browser q = new Browser();
    q.bootstrap();
    Map second = registration(q, ev, 0);
    Map secondParty = (Map) second.get("party");
    Map reissue =
        a.write(
            "CredentialReissue",
            map(
                "eventId",
                ev,
                "partyId",
                secondParty.get("partyId"),
                "version",
                secondParty.get("version"),
                "reason",
                "Lost page",
                "rotateAccess",
                true),
            200);
    q.write(
        "PartyExchange", map("queryToken", ((Map) second.get("access")).get("queryToken")), 404);
    q.write(
        "PartyExchange", map("queryToken", ((Map) reissue.get("access")).get("queryToken")), 200);
    a.write(
        "CredentialRevoke",
        map(
            "eventId",
            ev,
            "partyId",
            secondParty.get("partyId"),
            "version",
            ((Map) reissue.get("party")).get("version"),
            "reason",
            "Lost ticket"),
        200);
    assertFalse(q.request("MyCredential", map(), 200).containsKey("ticketToken"));
    Map updated =
        a.request("PartyDetail", map("eventId", ev, "partyId", secondParty.get("partyId")), 200);
    a.write(
        "PartyCancel",
        map(
            "eventId",
            ev,
            "partyId",
            secondParty.get("partyId"),
            "version",
            updated.get("version"),
            "reason",
            "Organizer cancellation"),
        200);
  }

  int postStatus(Browser b, String operation, Map body) throws Exception {
    return mvc.perform(
            post(contract.operations.get(operation).get("path").toString())
                .contentType("application/json")
                .content(json.writeValueAsBytes(body))
                .header("Origin", "http://localhost:5173")
                .header("X-CSRF-Token", b.csrf)
                .header("X-Client-Context-ID", b.context)
                .cookie(b.cookies.values().toArray(Cookie[]::new)))
        .andReturn()
        .getResponse()
        .getStatus();
  }

  @Test
  void longUnicodePasswordsRoundTripWithoutTruncationAndLegacyAccountsStillWork() throws Exception {
    Browser a = readyAdmin();
    String original = "密".repeat(256), replacement = "新".repeat(256);
    String username = "long-password-" + UUID.randomUUID();
    a.write(
        "AccountCreate",
        map(
            "username",
            username,
            "displayName",
            "Unicode password test",
            "temporaryPassword",
            original,
            "roles",
            List.of("CHECKER")),
        200);
    Browser b = new Browser();
    b.bootstrap();
    b.write("Login", map("username", username, "password", original), 200);
    b.write("PasswordChange", map("oldPassword", original, "newPassword", replacement), 200);
    b.bootstrap();
    b.write("Login", map("username", username, "password", replacement), 200);
    assertEquals(username, b.request("SessionMe", map(), 200).get("username"));
    Browser attacker = new Browser();
    attacker.bootstrap();
    attacker.write("Login", map("username", username, "password", "新".repeat(24)), 401);
    attacker.write(
        "Login", map("username", username, "password", replacement.substring(0, 255) + "错"), 401);
  }

  @Test
  void onsiteChangesAreSeparateFromAdmissionAndStaffDevicesNeedAnAccount() throws Exception {
    Browser a = readyAdmin();
    String ev = event(a, 10).get("eventId").toString();
    publish(a, ev);
    Browser p = new Browser();
    p.bootstrap();
    Map registration = registration(p, ev, 0);
    Map access = (Map) registration.get("access");
    p.write("PartyExchange", map("queryToken", access.get("queryToken")), 200);
    Map credential = p.request("MyCredential", map(), 200);
    String party = ((Map) registration.get("party")).get("partyId").toString();
    Map dev =
        a.write(
            "DeviceRegister",
            map("eventId", ev, "name", "Staff test device", "mode", "STAFF"),
            200);
    Browser terminal = new Browser();
    terminal.bootstrap();
    terminal.write(
        "DeviceClaim",
        map(
            "deviceId",
            ((Map) dev.get("device")).get("deviceId"),
            "pairingCode",
            dev.get("pairingCode")),
        200);
    terminal.request(
        "TicketResolve", map("eventId", ev, "ticketToken", credential.get("ticketToken")), 401);
    Map account = newAccount(a, "CHECKER");
    a.write(
        "GrantSave",
        map(
            "eventId",
            ev,
            "accountId",
            account.get("accountId"),
            "role",
            "CHECKER",
            "allowedFields",
            List.of("name"),
            "allowedActions",
            List.of("CHECKIN"),
            "expiresAt",
            OffsetDateTime.now(ZoneOffset.UTC).plusDays(1).toString()),
        200);
    Browser checker = staff(account);
    checker.cookies.put("ug_device", terminal.cookies.get("ug_device"));
    Map e = a.request("EventDetail", map("eventId", ev), 200);
    a.write(
        "EventTransition", map("eventId", ev, "version", e.get("version"), "action", "OPEN"), 200);
    Map walkIn =
        a.write(
            "GateWalkIn",
            map(
                "eventId",
                ev,
                "fields",
                map("name", "Onsite Parent"),
                "reason",
                "Unregistered visitor"),
            200);
    assertEquals(0, num(walkIn.get("netAdmitted")));
    a.write(
        "GateIncrease",
        map(
            "eventId",
            ev,
            "partyId",
            party,
            "version",
            ((Map) registration.get("party")).get("version"),
            "additionalCount",
            2,
            "reason",
            "Additional family"),
        200);
    Map stats = a.request("Statistics", map("eventId", ev), 200);
    assertEquals(4, num(stats.get("allocatedPeople")));
    assertEquals(0, num(stats.get("netAdmissions")));
    assertEquals(2, num(stats.get("onsiteAddedPlaces")));
    checker.request("DeviceDetail", map(), 200);
    checker.request("GateLookup", map("eventId", ev, "query", "Onsite"), 200);
    Map scan =
        checker.request(
            "TicketResolve", map("eventId", ev, "ticketToken", credential.get("ticketToken")), 200);
    Map operation =
        map(
            "operationId",
            UUID.randomUUID().toString(),
            "eventId",
            ev,
            "scanContextToken",
            scan.get("scanContextToken"),
            "expectedPartyVersion",
            scan.get("partyVersion"),
            "count",
            1);
    checker.request("GateCommit", operation, 200);
    assertEquals(1, num(checker.request("GateRecords", map("eventId", ev), 200).get("total")));
    checker.request(
        "OperationResult",
        map("eventId", ev, "operationId", operation.get("operationId"), "action", "GateCommit"),
        200);
    checker.request("AuditList", map("eventId", ev), 200);
    checker.write(
        "GateIncrease",
        map(
            "eventId",
            ev,
            "partyId",
            party,
            "version",
            1,
            "additionalCount",
            1,
            "reason",
            "Forbidden"),
        403);
    e = a.request("EventDetail", map("eventId", ev), 200);
    a.write(
        "EventTransition", map("eventId", ev, "version", e.get("version"), "action", "PAUSE"), 200);
    checker.request(
        "TicketResolve", map("eventId", ev, "ticketToken", credential.get("ticketToken")), 409);
    e = a.request("EventDetail", map("eventId", ev), 200);
    a.write(
        "EventTransition",
        map("eventId", ev, "version", e.get("version"), "action", "RESUME"),
        200);
    Map devices = a.request("DeviceList", map("eventId", ev), 200);
    Map device = (Map) ((List) devices.get("items")).getFirst();
    a.write(
        "DeviceRevoke",
        map(
            "eventId",
            ev,
            "deviceId",
            device.get("deviceId"),
            "version",
            device.get("version"),
            "reason",
            "Station closed"),
        200);
    checker.request("DeviceDetail", map(), 401);
    e = a.request("EventDetail", map("eventId", ev), 200);
    a.write(
        "EventTransition", map("eventId", ev, "version", e.get("version"), "action", "END"), 200);
    a.write("GateWalkIn", map("eventId", ev, "fields", map(), "reason", "Too late"), 409);
  }

  @Test
  void csrfContextAndFailedLoginCountersSurviveRejectedTransactions() throws Exception {
    Browser admin = readyAdmin();
    String csrf = admin.csrf;
    admin.csrf = null;
    admin.request("AccountList", map(), 403);
    admin.csrf = csrf;
    Browser anonymous = new Browser();
    anonymous.bootstrap();
    String originalContext = anonymous.context;
    anonymous.context = UUID.randomUUID().toString();
    anonymous.write(
        "Login",
        map("username", "unknown-" + UUID.randomUUID(), "password", "Wrong-password-123!"),
        401);
    anonymous.context = originalContext;
    String target = "limited-" + UUID.randomUUID();
    for (int i = 0; i < 5; i++)
      anonymous.write("Login", map("username", target, "password", "Wrong-password-123!"), 401);
    anonymous.write("Login", map("username", target, "password", "Wrong-password-123!"), 429);
    assertEquals(
        5,
        store.db.queryForObject(
            "select count(*) from ug_challenge where kind='LOGIN_LIMIT' and actor_scope like"
                + " 'LOGIN:%' and expires_at>now() and code_digest is not null and actor_scope=?",
            Integer.class,
            "LOGIN:" + HexFormat.of().formatHex(store.crypto.hash(target + ":127.0.0.1"))));
    Map account = newAccount(admin, "CHECKER");
    Browser checker = staff(account);
    Map current = checker.request("SessionMe", map(), 200);
    store.execute(
        "update ug_session set last_seen_at=now()-interval '31 minutes' where account_id=?",
        uuid(current.get("accountId")));
    checker.request("SessionMe", map(), 401);
  }

  @Test
  void independentTabsKeepCsrfAcrossConcurrentSessionReads() throws Exception {
    Browser a = readyAdmin();
    String originalStaffCsrf = a.csrf;
    Browser anon = new Browser();
    anon.bootstrap();
    String originalAnonCsrf = anon.csrf;
    try (var pool = Executors.newFixedThreadPool(6)) {
      List<Future<?>> reads = new ArrayList<>();
      for (int i = 0; i < 12; i++) {
        reads.add(pool.submit(() -> a.request("SessionMe", map(), 200)));
        reads.add(pool.submit(() -> anon.request("SessionBootstrap", map(), 200)));
      }
      for (var read : reads) read.get(15, TimeUnit.SECONDS);
    }
    assertEquals(originalStaffCsrf, a.csrf);
    assertEquals(originalAnonCsrf, anon.csrf);
    a.csrf = originalStaffCsrf;
    a.request("AccountList", map(), 200);
    anon.csrf = originalAnonCsrf;
    anon.write("Login", map("username", "missing-" + UUID.randomUUID(), "password", "Wrong-password-123!"), 401);
    // A pre-upgrade tab with its stored random CSRF token also remains valid.
    String oldRandomToken = store.crypto.token();
    store.execute("update ug_session set csrf_digest=? where token_digest=?",
        store.crypto.hash(oldRandomToken), store.crypto.hash(a.cookies.get("ug_staff").getValue()));
    a.request("SessionMe", map(), 200);
    a.csrf = oldRandomToken;
    a.request("AccountList", map(), 200);
  }

  @Test
  void parallelReadsShareActivityWithoutSerializationConflicts() throws Exception {
    Browser a = readyAdmin();
    String eventId = event(a, 50).get("eventId").toString();
    publish(a, eventId);
    CountDownLatch start = new CountDownLatch(1);
    try (var pool = Executors.newFixedThreadPool(8)) {
      List<Future<Map<String, Object>>> calls = new ArrayList<>();
      for (int i = 0; i < 16; i++) {
        String operation = i % 2 == 0 ? "PolicyGet" : "Statistics";
        calls.add(
            pool.submit(
                () -> {
                  start.await();
                  return a.request(operation, map("eventId", eventId), 200);
                }));
      }
      start.countDown();
      for (var call : calls) assertNotNull(call.get(15, TimeUnit.SECONDS));
    }
  }

  @Test
  void concurrentLastPlaceHasOneDatabaseEffect() throws Exception {
    Browser a = readyAdmin();
    Map e = event(a, 1);
    String ev = e.get("eventId").toString();
    publish(a, ev);
    Browser one = new Browser(), two = new Browser();
    one.bootstrap();
    two.bootstrap();
    Map pub = one.request("PublicEvent", map("eventId", ev), 200);
    Map first =
        map(
            "operationId",
            UUID.randomUUID().toString(),
            "eventId",
            ev,
            "fields",
            map(),
            "acceptedNoticeVersion",
            pub.get("noticeVersion"));
    Map second = new LinkedHashMap(first);
    second.put("operationId", UUID.randomUUID().toString());
    CountDownLatch start = new CountDownLatch(1);
    try (var pool = Executors.newFixedThreadPool(2)) {
      Future<Integer> f1 =
          pool.submit(
              () -> {
                start.await();
                return postStatus(one, "SubmitRegistration", first);
              });
      Future<Integer> f2 =
          pool.submit(
              () -> {
                start.await();
                return postStatus(two, "SubmitRegistration", second);
              });
      start.countDown();
      List<Integer> codes = new ArrayList<>(List.of(f1.get(), f2.get()));
      Collections.sort(codes);
      assertEquals(List.of(200, 409), codes);
    }
    assertEquals(
        1,
        store.db.queryForObject(
            "select allocated_count from ug_event where id=?", Integer.class, uuid(ev)));
    assertEquals(
        1,
        store.db.queryForObject(
            "select count(*) from ug_party where event_id=?", Integer.class, uuid(ev)));
  }

  @Test
  void reviewAndDeltaApprovalNeverAllocateUnapprovedPeople() throws Exception {
    Browser a = readyAdmin();
    Map e = event(a, 3);
    String ev = e.get("eventId").toString();
    Map p = a.request("PolicyGet", map("eventId", ev), 200);
    p.put("approvalMode", "REVIEW");
    a.write("PolicyUpdate", map("eventId", ev, "policy", p), 200);
    publish(a, ev);
    Browser b = new Browser();
    b.bootstrap();
    Map submission = registration(b, ev, 1);
    String party = ((Map) submission.get("party")).get("partyId").toString();
    assertEquals(0, num(a.request("Statistics", map("eventId", ev), 200).get("allocatedPeople")));
    Map review =
        (Map)
            ((List)
                    a.request(
                            "ReviewList",
                            map("eventId", ev, "page", 1, "pageSize", 20, "result", "PENDING"),
                            200)
                        .get("items"))
                .getFirst();
    a.write(
        "ReviewDecide",
        map(
            "eventId",
            ev,
            "reviewId",
            review.get("reviewId"),
            "version",
            review.get("version"),
            "decision",
            "APPROVE",
            "reason",
            "Accepted"),
        200);
    Map partyView = a.request("PartyDetail", map("eventId", ev, "partyId", party), 200);
    a.write(
        "PartyChange",
        map(
            "eventId",
            ev,
            "partyId",
            party,
            "version",
            partyView.get("version"),
            "companionCount",
            2,
            "reason",
            "One more parent"),
        200);
    partyView = a.request("PartyDetail", map("eventId", ev, "partyId", party), 200);
    assertEquals(2, num(partyView.get("approvedSize")));
    assertEquals(1, num(partyView.get("pendingIncrease")));
    assertEquals(2, num(a.request("Statistics", map("eventId", ev), 200).get("allocatedPeople")));
    review =
        (Map)
            ((List)
                    a.request(
                            "ReviewList",
                            map("eventId", ev, "page", 1, "pageSize", 20, "result", "PENDING"),
                            200)
                        .get("items"))
                .getFirst();
    a.write(
        "ReviewDecide",
        map(
            "eventId",
            ev,
            "reviewId",
            review.get("reviewId"),
            "version",
            review.get("version"),
            "decision",
            "APPROVE",
            "reason",
            "Extra approved"),
        200);
    assertEquals(3, num(a.request("Statistics", map("eventId", ev), 200).get("allocatedPeople")));
  }

  @Test
  void onlineGroupWorkflowMaintainsCapacityFactsAndReplay() throws Exception {
    Browser a = admin();
    if (Boolean.TRUE.equals(a.request("SessionMe", map(), 200).get("mustChangePassword"))) {
      a.write(
          "PasswordChange",
          map("oldPassword", "Integration-test-123!", "newPassword", "Changed-integration-123!"),
          200);
      a.bootstrap();
      a.write(
          "Login",
          map("username", "integration-admin", "password", "Changed-integration-123!"),
          200);
    }
    OffsetDateTime now = OffsetDateTime.now(ZoneOffset.UTC);
    Map e =
        a.write(
            "EventCreate",
            map(
                "title",
                "Integration " + UUID.randomUUID(),
                "startsAt",
                now.plusHours(2).toString(),
                "endsAt",
                now.plusHours(5).toString(),
                "timeZone",
                "Asia/Shanghai",
                "registrationStartsAt",
                now.minusHours(1).toString(),
                "registrationEndsAt",
                now.plusHours(1).toString(),
                "capacity",
                3),
            200);
    String ev = e.get("eventId").toString();
    e =
        a.write(
            "EventTransition",
            map("eventId", ev, "version", e.get("version"), "action", "PUBLISH"),
            200);
    Browser p = new Browser();
    p.bootstrap();
    Map pub = p.request("PublicEvent", map("eventId", ev), 200);
    Map registration =
        map(
            "operationId",
            UUID.randomUUID().toString(),
            "eventId",
            ev,
            "fields",
            map("name", "Sample Parent", "companionCount", 2),
            "acceptedNoticeVersion",
            pub.get("noticeVersion"));
    Map submitted = p.request("SubmitRegistration", registration, 200);
    String party = ((Map) submitted.get("party")).get("partyId").toString();
    Map replay = p.request("SubmitRegistration", registration, 200);
    assertEquals(true, replay.get("replayed"));
    assertEquals(party, ((Map) replay.get("party")).get("partyId"));
    Browser stranger = new Browser();
    stranger.bootstrap();
    stranger.request(
        "RegistrationResult",
        map("eventId", ev, "operationId", registration.get("operationId")),
        404);
    Map access = (Map) submitted.get("access");
    p.write(
        "PartyExchange",
        map("queryToken", access.get("queryToken"), "changeToken", access.get("changeToken")),
        200);
    Map cred = p.request("MyCredential", map(), 200);
    assertNotNull(cred.get("ticketToken"));
    Map extra =
        map("eventId", ev, "fields", map(), "acceptedNoticeVersion", pub.get("noticeVersion"));
    stranger.write("SubmitRegistration", extra, 409);
    e = a.request("EventDetail", map("eventId", ev), 200);
    a.write(
        "EventTransition", map("eventId", ev, "version", e.get("version"), "action", "OPEN"), 200);
    p.write("MyPartyCancel", map("version", ((Map) submitted.get("party")).get("version")), 409);
    Map device =
        a.write(
            "DeviceRegister",
            map("eventId", ev, "name", "Integration kiosk", "mode", "KIOSK"),
            200);
    Browser k = new Browser();
    k.bootstrap();
    k.write(
        "DeviceClaim",
        map(
            "deviceId",
            ((Map) device.get("device")).get("deviceId"),
            "pairingCode",
            device.get("pairingCode")),
        200);
    k.request("GateLookup", map("eventId", ev, "query", "Sample"), 403);
    Map scan =
        k.request("TicketResolve", map("eventId", ev, "ticketToken", cred.get("ticketToken")), 200);
    Map first =
        map(
            "operationId",
            UUID.randomUUID().toString(),
            "eventId",
            ev,
            "scanContextToken",
            scan.get("scanContextToken"),
            "expectedPartyVersion",
            scan.get("partyVersion"),
            "count",
            2);
    Map result = k.request("GateCommit", first, 200);
    assertEquals(2, num(result.get("committedCount")));
    assertEquals(true, result.get("showAsNewAdmission"));
    Map same = k.request("GateCommit", first, 200);
    assertEquals(false, same.get("showAsNewAdmission"));
    assertEquals(true, same.get("replayed"));
    Map changed = new LinkedHashMap(first);
    changed.put("count", 1);
    k.request("GateCommit", changed, 400);
    Map recovered =
        k.request(
            "GateOperation",
            map(
                "operationId",
                first.get("operationId"),
                "scanContextToken",
                scan.get("scanContextToken")),
            200);
    assertEquals(false, recovered.get("showAsNewAdmission"));
    Map next =
        k.request("TicketResolve", map("eventId", ev, "ticketToken", cred.get("ticketToken")), 200);
    k.write(
        "GateCommit",
        map(
            "eventId",
            ev,
            "scanContextToken",
            next.get("scanContextToken"),
            "expectedPartyVersion",
            next.get("partyVersion"),
            "count",
            1),
        200);
    Map exhausted =
        k.request("TicketResolve", map("eventId", ev, "ticketToken", cred.get("ticketToken")), 200);
    k.write(
        "GateCommit",
        map(
            "eventId",
            ev,
            "scanContextToken",
            exhausted.get("scanContextToken"),
            "expectedPartyVersion",
            exhausted.get("partyVersion"),
            "count",
            1),
        409);
    Map stats = a.request("Statistics", map("eventId", ev), 200);
    assertEquals(3, num(stats.get("netAdmissions")));
    assertEquals(3, num(stats.get("allocatedPeople")));
    assertEquals(
        2,
        store.db.queryForObject(
            "select count(*) from ug_admission where event_id=?", Integer.class, uuid(ev)));
    assertFalse(
        new String(
                (byte[])
                    store
                        .row("select fields_cipher from ug_party where id=?", uuid(party))
                        .get("fields_cipher"))
            .contains("Sample Parent"));
    Map partyView = a.request("PartyDetail", map("eventId", ev, "partyId", party), 200);
    a.write(
        "GateCorrect",
        map(
            "eventId",
            ev,
            "recordId",
            result.get("recordId"),
            "partyVersion",
            partyView.get("version"),
            "correctedCount",
            2,
            "reason",
            "Incorrect headcount"),
        200);
    assertEquals(1, num(a.request("Statistics", map("eventId", ev), 200).get("netAdmissions")));
    assertEquals(
        true,
        a.request("PartyDetail", map("eventId", ev, "partyId", party), 200)
            .get("hasAdmissionHistory"));
  }
}
