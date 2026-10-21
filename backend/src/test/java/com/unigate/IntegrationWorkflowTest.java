package com.unigate;

import static com.unigate.Store.*;
import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

import java.nio.charset.StandardCharsets;
import java.time.*;
import java.util.*;
import org.junit.jupiter.api.*;
import org.junit.jupiter.api.condition.EnabledIfEnvironmentVariable;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.mock.web.MockMultipartFile;
import org.springframework.test.context.DynamicPropertyRegistry;
import org.springframework.test.context.DynamicPropertySource;
import org.springframework.transaction.PlatformTransactionManager;
import org.springframework.transaction.support.TransactionTemplate;

@SpringBootTest(
    properties = {
      "unigate.master-key=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
      "unigate.cookie-secure=false",
      "unigate.bootstrap-password=Integration-test-123!",
      "unigate.bootstrap-username=integration-admin",
      "unigate.notifications.email-enabled=false",
      "unigate.notifications.poll-ms=3600000",
      "unigate.lifecycle.poll-ms=3600000",
      "unigate.sources.poll-ms=3600000",
      "spring.flyway.schemas=unigate_import_test",
      "spring.flyway.default-schema=unigate_import_test"
    })
@EnabledIfEnvironmentVariable(named = "UNIGATE_TEST_DB_URL", matches = ".+")
class IntegrationWorkflowTest {
  @DynamicPropertySource
  static void db(DynamicPropertyRegistry r) {
    r.add(
        "spring.datasource.url",
        () ->
            System.getenv("UNIGATE_TEST_DB_URL")
                + (System.getenv("UNIGATE_TEST_DB_URL").contains("?") ? "&" : "?")
                + "currentSchema=unigate_import_test");
    r.add(
        "spring.datasource.username",
        () -> System.getenv().getOrDefault("UNIGATE_TEST_DB_USER", "unigate"));
    r.add(
        "spring.datasource.password",
        () -> System.getenv().getOrDefault("UNIGATE_TEST_DB_PASSWORD", ""));
  }

  @Autowired Store s;
  @Autowired IntegrationService integration;
  @Autowired EventService events;
  @Autowired PartyService parties;
  @Autowired NotificationService notifications;
  @Autowired Contract contract;
  @Autowired PlatformTransactionManager manager;
  Context c;
  TransactionTemplate tx;

  Context anonymous() {
    Context guest = new Context(s);
    guest.session = map("id", uuid(s.id()));
    guest.clientIp = "192.0.2." + (Math.abs(s.id().hashCode()) % 200 + 1);
    return guest;
  }

  Map<String, Object> notifyCall(
      NotificationService service, String operation, Map<String, Object> request, Context guest) {
    s.eventScope.remove();
    try {
      return tx.execute(status -> (Map<String, Object>) service.handle(operation, request, guest));
    } finally {
      s.eventScope.remove();
    }
  }

  NotificationService smtpService(int port) {
    var env =
        new org.springframework.mock.env.MockEnvironment()
            .withProperty("unigate.notifications.email-enabled", "true")
            .withProperty("unigate.notifications.from", "noreply@example.invalid")
            .withProperty("unigate.public-base-url", "https://unigate.example")
            .withProperty("spring.mail.host", "127.0.0.1")
            .withProperty("spring.mail.port", Integer.toString(port))
            .withProperty("spring.mail.properties.mail.smtp.starttls.enable", "false");
    return new NotificationService(s, parties, env, manager);
  }

  Map<String, Object> integrationCall(
      IntegrationService service, String operation, Map<String, Object> request, Context caller) {
    request.putIfAbsent("operationId", s.id());
    s.eventScope.remove();
    try {
      return tx.execute(
          status -> (Map<String, Object>) service.handle(operation, request, caller, null));
    } finally {
      s.eventScope.remove();
    }
  }

  @BeforeEach
  void setup() {
    tx = new TransactionTemplate(manager);
    c = new Context(s);
    c.account = s.row("select * from ug_account where username='integration-admin'");
  }

  @AfterEach
  void clear() {
    s.eventScope.remove();
  }

  Map<String, Object> event(int capacity) {
    return tx.execute(
        status -> {
          var now = s.now();
          var created =
              (Map<String, Object>)
                  events.handle(
                      "EventCreate",
                      map(
                          "title",
                          "Import fixture " + s.id(),
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
                      c,
                      null);
          return (Map<String, Object>)
              events.handle(
                  "EventTransition",
                  map(
                      "eventId",
                      created.get("eventId"),
                      "version",
                      created.get("version"),
                      "action",
                      "PUBLISH"),
                  c,
                  null);
        });
  }

  Map<String, Object> call(String op, Map<String, Object> r) {
    r.putIfAbsent("operationId", s.id());
    s.eventScope.remove();
    try {
      return tx.execute(status -> (Map<String, Object>) integration.handle(op, r, c, null));
    } finally {
      s.eventScope.remove();
    }
  }

  Map<String, Object> preview(Object event) {
    var batch =
        call(
            "BatchUpload",
            map(
                "eventId",
                event,
                "sourceLabel",
                "CSV fixture",
                "file",
                new MockMultipartFile(
                    "file",
                    "fixture.csv",
                    "text/csv",
                    "name,phone,total\nParent A,13800000001,2\nParent B,13800000002,2\n"
                        .getBytes(StandardCharsets.UTF_8))));
    return call(
        "BatchPreview",
        map(
            "eventId",
            event,
            "batchId",
            batch.get("batchId"),
            "version",
            batch.get("version"),
            "mapping",
            map(
                "nameColumn",
                "name",
                "phoneColumn",
                "phone",
                "countColumn",
                "total",
                "countMode",
                "TOTAL"),
            "eligibility",
            "ELIGIBLE"));
  }

  List<String> rowIds(Object batch) {
    return s
        .rows("select id from ug_import_row where batch_id=? order by line_number", uuid(batch))
        .stream()
        .map(x -> str(x.get("id")))
        .toList();
  }

  @Test
  void selectedImportIsAllOrNothingAndDoesNotOverAllocate() {
    var e = event(3);
    Object event = e.get("eventId");
    var batch = preview(event);
    assertEquals(2, num(batch.get("validRows")));
    var selected = rowIds(batch.get("batchId"));
    var request =
        map(
            "eventId",
            event,
            "batchId",
            batch.get("batchId"),
            "previewVersion",
            batch.get("previewVersion"),
            "selectedRowIds",
            selected);
    var failure = assertThrows(ApiException.class, () -> call("BatchCommit", request));
    assertEquals(409, failure.status);
    assertEquals(
        0,
        s.db.queryForObject(
            "select count(*) from ug_party where event_id=?", Integer.class, uuid(event)));
    assertEquals(
        0,
        num(
            s.row("select allocated_count from ug_event where id=?", uuid(event))
                .get("allocated_count")));
    request.put("selectedRowIds", List.of(selected.getFirst()));
    request.put("operationId", s.id());
    var result = call("BatchCommit", request);
    assertEquals(1, num(result.get("createdCount")));
    assertEquals(2, num(result.get("allocatedPeople")));
    assertEquals(
        2,
        num(
            s.row("select allocated_count from ug_event where id=?", uuid(event))
                .get("allocated_count")));
    assertEquals(
        1,
        s.db.queryForObject(
            "select count(*) from ug_source_link where event_id=?", Integer.class, uuid(event)));
  }

  @Test
  void policyDriftRequiresACompletelyNewPreview() {
    Object event = event(10).get("eventId");
    var batch = preview(event);
    s.execute("update ug_event set policy_version=policy_version+1 where id=?", uuid(event));
    assertThrows(
        ApiException.class,
        () ->
            call(
                "BatchCommit",
                map(
                    "eventId",
                    event,
                    "batchId",
                    batch.get("batchId"),
                    "previewVersion",
                    batch.get("previewVersion"),
                    "selectedRowIds",
                    rowIds(batch.get("batchId")))));
    assertEquals(
        0,
        s.db.queryForObject(
            "select count(*) from ug_party where event_id=?", Integer.class, uuid(event)));
  }

  @Test
  void exportRequiresIndependentGrantAndRechecksItOnDownload() {
    Object event = event(10).get("eventId");
    var batch = preview(event);
    call(
        "BatchCommit",
        map(
            "eventId",
            event,
            "batchId",
            batch.get("batchId"),
            "previewVersion",
            batch.get("previewVersion"),
            "selectedRowIds",
            rowIds(batch.get("batchId"))));
    var args =
        map(
            "eventId",
            event,
            "format",
            "CSV",
            "columns",
            List.of("name", "phone", "approvedSize"),
            "purpose",
            "Test approved export");
    assertThrows(ApiException.class, () -> call("ExportCreate", args));
    s.execute(
        "update ug_event_grant set allowed_actions=allowed_actions || '[\"EXPORT\"]'::jsonb where"
            + " event_id=? and account_id=?",
        uuid(event),
        c.account.get("id"));
    var export = call("ExportCreate", args);
    assertEquals("SUCCEEDED", ((Map<?, ?>) export.get("status")).get("code"));
    var request = map("eventId", event, "exportId", export.get("exportId"));
    var file =
        tx.execute(
            status ->
                (IntegrationService.Binary) integration.handle("ExportDownload", request, c, null));
    assertTrue(new String(file.data(), StandardCharsets.UTF_8).contains("Parent A"));
    s.execute(
        "update ug_event_grant set allowed_fields='[]'::jsonb where event_id=? and account_id=?",
        uuid(event),
        c.account.get("id"));
    assertThrows(
        ApiException.class,
        () -> tx.execute(status -> integration.handle("ExportDownload", request, c, null)));
  }

  @Test
  void cleanupRemovesFactsAndLeavesHonestBackupFailure() {
    Object event = event(10).get("eventId");
    var batch = preview(event);
    call(
        "BatchCommit",
        map(
            "eventId",
            event,
            "batchId",
            batch.get("batchId"),
            "previewVersion",
            batch.get("previewVersion"),
            "selectedRowIds",
            rowIds(batch.get("batchId"))));
    byte[] old =
        (byte[])
            s.row("select fields_cipher from ug_party where event_id=? limit 1", uuid(event))
                .get("fields_cipher");
    s.execute(
        "update ug_cleanup set due_at=now()-interval '1 second' where event_id=?", uuid(event));
    tx.executeWithoutResult(
        status ->
            integration.runCleanup(
                s.row("select * from ug_cleanup where event_id=?", uuid(event))));
    assertEquals(
        0,
        s.db.queryForObject(
            "select count(*) from ug_party where event_id=?", Integer.class, uuid(event)));
    assertNotNull(
        s.row("select anonymous_totals from ug_event where id=?", uuid(event))
            .get("anonymous_totals"));
    assertEquals(
        "ARCHIVED", s.row("select status from ug_event where id=?", uuid(event)).get("status"));
    assertEquals(
        "FAILED",
        s.row("select status from ug_cleanup where event_id=?", uuid(event)).get("status"));
    assertThrows(ApiException.class, () -> s.unseal(old));
  }

  @Test
  void challengeFailuresSurviveOuterRollbackAndLockAfterFive() {
    String event = str(event(10).get("eventId"));
    String challenge = s.id(), context = s.id();
    Context anon = new Context(s);
    anon.session = map("id", uuid(context));
    s.execute(
        "insert into"
            + " ug_challenge(id,kind,event_id,actor_scope,target_cipher,code_digest,context_cipher,expires_at)"
            + " values(?,'SMS',?,'test',?,?,?,?)",
        uuid(challenge),
        uuid(event),
        s.seal(map("contact", "13800000001")),
        s.crypto.hash(challenge + ":123456"),
        s.seal(map("contextId", context)),
        s.now().plusMinutes(5));
    for (int i = 0; i < 5; i++)
      assertThrows(
          ApiException.class,
          () ->
              tx.execute(
                  status ->
                      notifications.handle(
                          "SmsVerify",
                          map("eventId", event, "challengeId", challenge, "code", "000000"),
                          anon)));
    assertEquals(
        5,
        num(
            s.row("select attempts from ug_challenge where id=?", uuid(challenge))
                .get("attempts")));
    assertThrows(
        ApiException.class,
        () ->
            tx.execute(
                status ->
                    notifications.handle(
                        "SmsVerify",
                        map("eventId", event, "challengeId", challenge, "code", "123456"),
                        anon)));
  }

  @Test
  void recoveryRotatesOnlyPrivateAccessAndConsumesChallenge() {
    String event = str(event(10).get("eventId"));
    var party =
        tx.execute(
            status ->
                parties.create(
                    c,
                    s.event(event),
                    map("name", "Recovery Parent", "email", "parent@example.invalid"),
                    "BUILTIN",
                    true,
                    "fixture",
                    null));
    String partyId = str(party.get("partyId")), challenge = s.id(), context = s.id();
    Context anon = new Context(s);
    anon.session = map("id", uuid(context));
    s.execute(
        "insert into"
            + " ug_challenge(id,kind,event_id,party_id,actor_scope,target_cipher,code_digest,context_cipher,expires_at)"
            + " values(?,'RECOVERY',?,?,'test',?,?,?,?)",
        uuid(challenge),
        uuid(event),
        uuid(partyId),
        s.seal(map("contact", "parent@example.invalid")),
        s.crypto.hash(challenge + ":123456"),
        s.seal(map("contextId", context)),
        s.now().plusMinutes(5));
    var old =
        s
            .rows(
                "select id from ug_credential where party_id=? and kind in ('QUERY','CHANGE') and"
                    + " status='ACTIVE'",
                uuid(partyId))
            .stream()
            .map(x -> x.get("id"))
            .toList();
    var access =
        (Map<?, ?>)
            tx.execute(
                status ->
                    notifications.handle(
                        "RecoveryVerify", map("challengeId", challenge, "code", "123456"), anon));
    assertNotNull(access.get("queryToken"));
    assertNotNull(access.get("changeToken"));
    for (Object id : old)
      assertEquals(
          "REVOKED", s.row("select status from ug_credential where id=?", id).get("status"));
    assertEquals(
        1,
        s.db.queryForObject(
            "select count(*) from ug_credential where party_id=? and kind='TICKET' and"
                + " status='ACTIVE'",
            Integer.class,
            uuid(partyId)));
    assertEquals(
        1,
        num(
            s.row("select allocated_count from ug_event where id=?", uuid(event))
                .get("allocated_count")));
    assertThrows(
        ApiException.class,
        () ->
            tx.execute(
                status ->
                    notifications.handle(
                        "RecoveryVerify", map("challengeId", challenge, "code", "123456"), anon)));
  }

  @Test
  void notificationFailureUsesFiveAttemptsAndExponentialBackoff() {
    String event = str(event(10).get("eventId"));
    tx.execute(
        status ->
            parties.create(
                c,
                s.event(event),
                map("email", "parent@example.invalid"),
                "BUILTIN",
                true,
                "fixture",
                null));
    Object id =
        s.row("select id from ug_job where event_id=? and kind='EMAIL'", uuid(event)).get("id");
    s.execute(
        "update ug_job set status='EXPIRED' where kind='EMAIL' and id<>? and status in"
            + " ('PENDING','RUNNING')",
        id);
    for (int attempt = 1; attempt <= 5; attempt++) {
      notifications.deliverNext();
      var job = s.row("select * from ug_job where id=?", id);
      assertEquals(attempt, num(job.get("attempts")));
      assertEquals(attempt < 5 ? "PENDING" : "FAILED", job.get("status"));
      assertNull(job.get("lease_until"));
      if (attempt < 5) {
        long seconds = Duration.between(s.now(), time(job.get("next_run_at"))).toSeconds();
        assertTrue(seconds >= (1L << (attempt - 1)) * 60 - 5);
        s.execute("update ug_job set next_run_at=now() where id=?", id);
      }
    }
    notifications.deliverNext();
    assertEquals(5, num(s.row("select attempts from ug_job where id=?", id).get("attempts")));
  }

  @Test
  void sourcePullIsQueuedAndProviderIoRunsOutsideTheDatabaseTransaction() {
    Object event = event(10).get("eventId");
    var connection =
        call(
            "ConnectionSave",
            map(
                "provider",
                "FEISHU",
                "accountKind",
                "PERSONAL",
                "displayName",
                "Test source",
                "allowedEventIds",
                List.of(event)));
    var source =
        call(
            "SourceSave",
            map(
                "eventId",
                event,
                "connectionId",
                connection.get("connectionId"),
                "resourceId",
                "app/table",
                "mapping",
                map("nameColumn", "name", "countMode", "COMPANIONS"),
                "eligibility",
                "ELIGIBLE"));
    // Updating metadata with an unchanged assignment must retain the composite FK used by the
    // source.
    call(
        "ConnectionSave",
        map(
            "connectionId",
            connection.get("connectionId"),
            "version",
            connection.get("version"),
            "provider",
            "FEISHU",
            "accountKind",
            "PERSONAL",
            "displayName",
            "Renamed source",
            "allowedEventIds",
            List.of(event)));
    var job =
        call(
            "SourcePull",
            map(
                "eventId",
                event,
                "sourceId",
                source.get("sourceId"),
                "version",
                source.get("version")));
    assertEquals("PENDING", ((Map<?, ?>) job.get("status")).get("code"));
    assertEquals(
        0,
        s.db.queryForObject(
            "select count(*) from ug_import_batch where event_id=?", Integer.class, uuid(event)));
    var provider = mock(ProviderAdapter.class);
    when(provider.pull(anyMap(), eq("app/table")))
        .thenAnswer(
            invocation -> {
              assertFalse(
                  org.springframework.transaction.support.TransactionSynchronizationManager
                      .isActualTransactionActive());
              return List.of(
                  new ProviderAdapter.RemoteRow("stable-answer", map("name", "Remote Parent")));
            });
    s.execute(
        "update ug_job set status='EXPIRED' where kind='PULL' and id<>? and status in"
            + " ('PENDING','RUNNING')",
        uuid(job.get("jobId")));
    new SourcePullWorker(s, provider, manager).pullNext();
    verify(provider).pull(anyMap(), eq("app/table"));
    var completed = s.row("select * from ug_job where id=?", uuid(job.get("jobId")));
    assertEquals("SUCCEEDED", completed.get("status"));
    assertNotNull(completed.get("result_ref"));
    assertEquals(
        "stable-answer",
        s.row(
                "select source_record_id from ug_import_row where batch_id=?",
                completed.get("result_ref"))
            .get("source_record_id"));
    assertEquals(
        0,
        num(
            s.row("select allocated_count from ug_event where id=?", uuid(event))
                .get("allocated_count")));
  }

  @Test
  void formulaRowsAreExcludedButOtherRowsRemainCommittable() throws Exception {
    Object event = event(10).get("eventId");
    byte[] bytes;
    try (var book = new org.apache.poi.xssf.usermodel.XSSFWorkbook();
        var out = new java.io.ByteArrayOutputStream()) {
      var sheet = book.createSheet("名单");
      sheet.createRow(0).createCell(0).setCellValue("name");
      sheet.createRow(1).createCell(0).setCellValue("Parent");
      sheet.createRow(2).createCell(0).setCellFormula("1+1");
      book.write(out);
      bytes = out.toByteArray();
    }
    var uploaded =
        call(
            "BatchUpload",
            map(
                "eventId",
                event,
                "sourceLabel",
                "Excel fixture",
                "file",
                new MockMultipartFile(
                    "file",
                    "formula.xlsx",
                    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
                    bytes)));
    var preview =
        call(
            "BatchPreview",
            map(
                "eventId",
                event,
                "batchId",
                uploaded.get("batchId"),
                "version",
                uploaded.get("version"),
                "mapping",
                map("nameColumn", "name", "countMode", "COMPANIONS"),
                "eligibility",
                "ELIGIBLE"));
    assertEquals(1, num(preview.get("validRows")));
    assertEquals(1, num(preview.get("errorRows")));
    var rows =
        s.rows(
            "select id,status from ug_import_row where batch_id=? order by line_number",
            uuid(uploaded.get("batchId")));
    assertEquals("ERROR", rows.get(1).get("status"));
    call(
        "BatchCommit",
        map(
            "eventId",
            event,
            "batchId",
            uploaded.get("batchId"),
            "previewVersion",
            preview.get("previewVersion"),
            "selectedRowIds",
            List.of(str(rows.getFirst().get("id")))));
    assertEquals(
        1,
        s.db.queryForObject(
            "select count(*) from ug_party where event_id=?", Integer.class, uuid(event)));
  }

  @Test
  void smtpAcceptanceScrubsQueueAndDoesNotHoldEventTransaction() throws Exception {
    String event = str(event(10).get("eventId"));
    var party =
        tx.execute(
            status ->
                parties.create(
                    c,
                    s.event(event),
                    map("name", "SMTP Parent", "email", "Parent@example.invalid"),
                    "BUILTIN",
                    true,
                    "fixture",
                    null));
    Object job =
        s.row("select id from ug_job where event_id=? and kind='EMAIL'", uuid(event)).get("id");
    s.execute(
        "update ug_job set status='EXPIRED' where kind='EMAIL' and id<>? and status in"
            + " ('PENDING','RUNNING')",
        job);
    var unlocked = new java.util.concurrent.atomic.AtomicBoolean();
    try (var smtp =
        new SmtpCapture(
            () -> {
              s.db.queryForObject(
                  "select id from ug_event where id=? for update nowait", UUID.class, uuid(event));
              unlocked.set(true);
            })) {
      var deliveryService = smtpService(smtp.port());
      var api =
          new IntegrationService(
              s,
              parties,
              mock(ProviderAdapter.class),
              deliveryService,
              manager,
              contract,
              "./var/unigate-test");
      var resend =
          integrationCall(
              api,
              "EmailResend",
              map(
                  "eventId",
                  event,
                  "partyId",
                  party.get("partyId"),
                  "reason",
                  "Participant requested another email"),
              c);
      job = uuid(resend.get("jobId"));
      s.execute(
          "update ug_job set status='EXPIRED' where kind='EMAIL' and id<>? and status in"
              + " ('PENDING','RUNNING')",
          job);
      deliveryService.deliverNext();
      var message =
          new jakarta.mail.internet.MimeMessage(
              jakarta.mail.Session.getInstance(new java.util.Properties()),
              new java.io.ByteArrayInputStream(smtp.message().getBytes(StandardCharsets.UTF_8)));
      assertEquals("UniGate 活动报名凭证", message.getSubject());
      assertTrue(
          message.getContent().toString().contains("https://unigate.example/me/party#queryToken="));
      assertEquals(
          "Parent@example.invalid",
          ((jakarta.mail.internet.InternetAddress) message.getAllRecipients()[0]).getAddress());
      assertTrue(unlocked.get());
    }
    var delivered = s.row("select * from ug_job where id=?", job);
    assertEquals("SUCCEEDED", delivered.get("status"));
    assertNull(delivered.get("payload_cipher"));
    assertNull(delivered.get("lease_until"));
    assertEquals(1, num(delivered.get("attempts")));
    assertEquals(
        1,
        num(
            s.row("select allocated_count from ug_event where id=?", uuid(event))
                .get("allocated_count")));
  }

  @Test
  void recoveryStartDoesNotRevealMatchAndChallengeIsActuallyEmailed() throws Exception {
    String event = str(event(10).get("eventId"));
    var party =
        tx.execute(
            status ->
                parties.create(
                    c,
                    s.event(event),
                    map("email", "Parent@example.invalid"),
                    "BUILTIN",
                    true,
                    "fixture",
                    null));
    Context guest = anonymous();
    try (var smtp = new SmtpCapture(() -> {})) {
      var service = smtpService(smtp.port());
      var missing =
          notifyCall(
              service,
              "RecoveryStart",
              map(
                  "eventId",
                  event,
                  "partyReference",
                  "UG-DOES-NOT-EXIST",
                  "channel",
                  "EMAIL",
                  "contact",
                  "Absent@example.invalid"),
              guest);
      var matched =
          notifyCall(
              service,
              "RecoveryStart",
              map(
                  "eventId",
                  event,
                  "partyReference",
                  party.get("partyReference"),
                  "channel",
                  "EMAIL",
                  "contact",
                  "Parent@EXAMPLE.INVALID"),
              guest);
      assertEquals(Set.of("challengeId", "expiresAt", "retryAfterSeconds"), missing.keySet());
      assertEquals(missing.keySet(), matched.keySet());
      assertEquals(60, num(missing.get("retryAfterSeconds")));
      assertEquals(
          0,
          s.db.queryForObject(
              "select count(*) from ug_job where business_key=?",
              Integer.class,
              "challenge:" + missing.get("challengeId")));
      var job =
          s.row(
              "select * from ug_job where business_key=?",
              "challenge:" + matched.get("challengeId"));
      var payload = s.unseal(job.get("payload_cipher"));
      assertNotNull(payload.get("code"));
      assertFalse(
          new String((byte[]) job.get("payload_cipher"), StandardCharsets.UTF_8)
              .contains(str(payload.get("code"))));
      s.execute(
          "update ug_job set status='EXPIRED' where kind='EMAIL' and id<>? and status in"
              + " ('PENDING','RUNNING')",
          job.get("id"));
      service.deliverNext();
      var message =
          new jakarta.mail.internet.MimeMessage(
              jakarta.mail.Session.getInstance(new java.util.Properties()),
              new java.io.ByteArrayInputStream(smtp.message().getBytes(StandardCharsets.UTF_8)));
      assertTrue(message.getContent().toString().contains(str(payload.get("code"))));
      assertEquals("UniGate 验证码", message.getSubject());
    }
  }

  @Test
  void smsChallengeProofBindsPhoneEventContextExpiryAndSingleUse() {
    String event = str(event(10).get("eventId"));
    s.execute(
        "update ug_event set policy=jsonb_set(policy,'{phoneVerificationEnabled}','true') where"
            + " id=?",
        uuid(event));
    var env =
        new org.springframework.mock.env.MockEnvironment()
            .withProperty("unigate.notifications.sms-enabled", "true")
            .withProperty("unigate.notifications.sms-endpoint", "https://sms.example.invalid/send");
    var service = new NotificationService(s, parties, env, manager);
    Context guest = anonymous();
    String phone = "+861380000" + Math.abs(s.id().hashCode() % 100000);
    var challenge = notifyCall(service, "SmsStart", map("eventId", event, "phone", phone), guest);
    var job =
        s.row(
            "select * from ug_job where business_key=?",
            "challenge:" + challenge.get("challengeId"));
    String code = str(s.unseal(job.get("payload_cipher")).get("code"));
    assertThrows(
        ApiException.class,
        () -> notifyCall(service, "SmsStart", map("eventId", event, "phone", phone), guest));
    assertThrows(
        ApiException.class,
        () ->
            notifyCall(
                service,
                "SmsVerify",
                map("eventId", event, "challengeId", challenge.get("challengeId"), "code", code),
                anonymous()));
    assertEquals(
        0,
        num(
            s.row(
                    "select attempts from ug_challenge where id=?",
                    uuid(challenge.get("challengeId")))
                .get("attempts")));
    var verified =
        notifyCall(
            service,
            "SmsVerify",
            map("eventId", event, "challengeId", challenge.get("challengeId"), "code", code),
            guest);
    var proof = s.crypto.verify(str(verified.get("verificationProof")), s.json);
    assertEquals(phone, proof.get("phone"));
    assertEquals(event, proof.get("eventId"));
    assertEquals(str(guest.session.get("id")), proof.get("contextId"));
    String token = str(verified.get("verificationProof"));
    assertThrows(
        ApiException.class,
        () ->
            tx.executeWithoutResult(
                status ->
                    parties.verifyPhone(
                        map("fields", map("phone", phone), "phoneVerificationProof", token),
                        anonymous(),
                        s.event(event))));
    var expiredProof = new LinkedHashMap<String, Object>(proof);
    expiredProof.put("expiresAt", s.now().minusSeconds(1).toString());
    String expiredToken = s.crypto.signed(expiredProof, s.json);
    assertThrows(
        ApiException.class,
        () ->
            tx.executeWithoutResult(
                status ->
                    parties.verifyPhone(
                        map("fields", map("phone", phone), "phoneVerificationProof", expiredToken),
                        guest,
                        s.event(event))));
    assertThrows(
        ApiException.class,
        () ->
            tx.executeWithoutResult(
                status ->
                    parties.verifyPhone(
                        map(
                            "fields",
                            map("phone", "+8613900000000"),
                            "phoneVerificationProof",
                            token),
                        guest,
                        s.event(event))));
    tx.executeWithoutResult(
        status ->
            parties.verifyPhone(
                map("fields", map("phone", phone), "phoneVerificationProof", token),
                guest,
                s.event(event)));
    assertThrows(
        ApiException.class,
        () ->
            tx.executeWithoutResult(
                status ->
                    parties.verifyPhone(
                        map("fields", map("phone", phone), "phoneVerificationProof", token),
                        guest,
                        s.event(event))));
    var late = notifyCall(service, "SmsStart", map("eventId", event, "phone", phone + "1"), guest);
    var lateJob =
        s.row("select * from ug_job where business_key=?", "challenge:" + late.get("challengeId"));
    String lateCode = str(s.unseal(lateJob.get("payload_cipher")).get("code"));
    s.execute(
        "update ug_challenge set expires_at=now()-interval '1 second' where id=?",
        uuid(late.get("challengeId")));
    assertThrows(
        ApiException.class,
        () ->
            notifyCall(
                service,
                "SmsVerify",
                map("eventId", event, "challengeId", late.get("challengeId"), "code", lateCode),
                guest));
  }

  @Test
  void duplicateDecisionsPreserveOriginalPartyAndOnlyAllocateSeparateRows() {
    String event = str(event(10).get("eventId"));
    var original =
        tx.execute(
            status ->
                parties.create(
                    c,
                    s.event(event),
                    map("name", "Original Parent", "phone", "13800000001"),
                    "BUILTIN",
                    true,
                    "fixture",
                    null));
    String csv =
        "name,phone,total\nLinked,13800000001,1\nSeparate,13800000001,1\nExcluded,13800000001,1\n";
    var uploaded =
        call(
            "BatchUpload",
            map(
                "eventId",
                event,
                "sourceLabel",
                "Duplicates",
                "file",
                new MockMultipartFile(
                    "file", "dup.csv", "text/csv", csv.getBytes(StandardCharsets.UTF_8))));
    var mapping =
        map(
            "nameColumn",
            "name",
            "phoneColumn",
            "phone",
            "countColumn",
            "total",
            "countMode",
            "TOTAL");
    var preview =
        call(
            "BatchPreview",
            map(
                "eventId",
                event,
                "batchId",
                uploaded.get("batchId"),
                "version",
                uploaded.get("version"),
                "mapping",
                mapping,
                "eligibility",
                "ELIGIBLE"));
    assertEquals(3, num(preview.get("suspectRows")));
    var rows =
        call(
            "BatchRows",
            map(
                "eventId",
                event,
                "batchId",
                uploaded.get("batchId"),
                "status",
                "DUPLICATE_SUSPECT",
                "pageSize",
                100));
    assertEquals(3, num(rows.get("total")));
    var ids = rowIds(uploaded.get("batchId"));
    preview =
        call(
            "DuplicateResolve",
            map(
                "eventId",
                event,
                "batchId",
                uploaded.get("batchId"),
                "rowId",
                ids.get(0),
                "previewVersion",
                preview.get("previewVersion"),
                "resolution",
                "LINK_EXISTING",
                "existingPartyId",
                original.get("partyId"),
                "reason",
                "Same guardian confirmed"));
    preview =
        call(
            "DuplicateResolve",
            map(
                "eventId",
                event,
                "batchId",
                uploaded.get("batchId"),
                "rowId",
                ids.get(1),
                "previewVersion",
                preview.get("previewVersion"),
                "resolution",
                "KEEP_SEPARATE",
                "reason",
                "Different group using a shared phone"));
    preview =
        call(
            "DuplicateResolve",
            map(
                "eventId",
                event,
                "batchId",
                uploaded.get("batchId"),
                "rowId",
                ids.get(2),
                "previewVersion",
                preview.get("previewVersion"),
                "resolution",
                "EXCLUDE",
                "reason",
                "Duplicate submission confirmed"));
    var selected =
        map(
            "eventId",
            event,
            "batchId",
            uploaded.get("batchId"),
            "previewVersion",
            preview.get("previewVersion"),
            "selectedRowIds",
            ids);
    assertThrows(ApiException.class, () -> call("BatchCommit", selected));
    assertEquals(
        1,
        num(
            s.row("select allocated_count from ug_event where id=?", uuid(event))
                .get("allocated_count")));
    selected.put("selectedRowIds", ids.subList(0, 2));
    var committed = call("BatchCommit", selected);
    assertEquals(1, num(committed.get("createdCount")));
    assertEquals(1, num(committed.get("linkedCount")));
    assertEquals(1, num(committed.get("allocatedPeople")));
    assertEquals(
        "Original Parent",
        s.unseal(
                s.row(
                        "select fields_cipher from ug_party where id=?",
                        uuid(original.get("partyId")))
                    .get("fields_cipher"))
            .get("name"));
    var repeated =
        call(
            "BatchUpload",
            map(
                "eventId",
                event,
                "sourceLabel",
                "Same file again",
                "file",
                new MockMultipartFile(
                    "file", "dup.csv", "text/csv", csv.getBytes(StandardCharsets.UTF_8))));
    var repeatedPreview =
        call(
            "BatchPreview",
            map(
                "eventId",
                event,
                "batchId",
                repeated.get("batchId"),
                "version",
                repeated.get("version"),
                "mapping",
                mapping,
                "eligibility",
                "ELIGIBLE"));
    var already =
        call(
            "BatchRows",
            map("eventId", event, "batchId", repeated.get("batchId"), "status", "ALREADY_LINKED"));
    assertEquals(2, num(already.get("total")));
    assertThrows(
        ApiException.class,
        () ->
            call(
                "BatchCommit",
                map(
                    "eventId",
                    event,
                    "batchId",
                    repeated.get("batchId"),
                    "previewVersion",
                    repeatedPreview.get("previewVersion"),
                    "selectedRowIds",
                    rowIds(repeated.get("batchId")))));
    assertEquals(
        2,
        num(
            s.row("select allocated_count from ug_event where id=?", uuid(event))
                .get("allocated_count")));
  }

  @Test
  void connectionSecretsNeverReturnAndResourceAccessStaysWithinAssignedEvent() {
    String event = str(event(10).get("eventId")), other = str(event(10).get("eventId"));
    String secret = "test-client-secret-do-not-return";
    var connection =
        call(
            "ConnectionSave",
            map(
                "provider",
                "FEISHU",
                "accountKind",
                "PERSONAL",
                "displayName",
                "Private application",
                "allowedEventIds",
                List.of(event),
                "clientId",
                "test-app",
                "clientSecret",
                secret));
    String id = str(connection.get("connectionId"));
    assertFalse(connection.toString().contains(secret));
    assertTrue(connection.containsKey("credentialMask"));
    assertEquals(
        secret,
        s.unseal(
                s.row("select credentials_cipher from ug_connection where id=?", uuid(id))
                    .get("credentials_cipher"))
            .get("clientSecret"));
    assertFalse(
        call("ConnectionList", map("provider", "FEISHU", "pageSize", 100))
            .toString()
            .contains(secret));
    var source =
        call(
            "SourceSave",
            map(
                "eventId",
                event,
                "connectionId",
                id,
                "resourceId",
                "app/table",
                "mapping",
                map("countMode", "COMPANIONS"),
                "eligibility",
                "REVIEW_REQUIRED"));
    var listed = call("SourceList", map("eventId", event));
    assertEquals(1, ((List<?>) listed.get("items")).size());
    assertEquals(1, ((List<?>) listed.get("availableConnections")).size());
    assertFalse(listed.toString().contains(secret));
    call(
        "SourceSave",
        map(
            "eventId",
            event,
            "sourceId",
            source.get("sourceId"),
            "version",
            source.get("version"),
            "connectionId",
            id,
            "resourceId",
            "app/newtable",
            "mapping",
            map("nameColumn", "name", "countMode", "COMPANIONS"),
            "eligibility",
            "ELIGIBLE"));
    var adapter = mock(ProviderAdapter.class);
    when(adapter.list(anyMap()))
        .thenReturn(
            List.of(
                map(
                    "resourceId",
                    "app/newtable",
                    "title",
                    "Answers",
                    "kind",
                    "BITABLE",
                    "readable",
                    true)));
    when(adapter.fields(anyMap(), eq("app/newtable")))
        .thenReturn(List.of(map("code", "name", "label", "name")));
    var service =
        new IntegrationService(
            s, parties, adapter, notifications, manager, contract, "./var/unigate-test");
    Context owner = new Context(s);
    owner.account = new LinkedHashMap<>(c.account);
    owner.account.put("roles", List.of("ORGANIZER"));
    assertThrows(
        ApiException.class,
        () ->
            integrationCall(
                service,
                "ResourceFields",
                map("eventId", other, "connectionId", id, "resourceId", "app/newtable"),
                owner));
    verifyNoInteractions(adapter);
    assertEquals(
        1,
        ((List<?>)
                integrationCall(
                        service,
                        "ResourceFields",
                        map("eventId", event, "connectionId", id, "resourceId", "app/newtable"),
                        owner)
                    .get("availableColumns"))
            .size());
    assertEquals(
        1,
        ((List<?>)
                integrationCall(
                        service, "ResourceList", map("eventId", event, "connectionId", id), owner)
                    .get("items"))
            .size());
    var revoked =
        call("ConnectionRevoke", map("connectionId", id, "version", connection.get("version")));
    assertEquals("REVOKED", ((Map<?, ?>) revoked.get("status")).get("code"));
    assertNull(
        s.row("select credentials_cipher from ug_connection where id=?", uuid(id))
            .get("credentials_cipher"));
    assertEquals(1, ((List<?>) call("SourceList", map("eventId", event)).get("items")).size());
  }

  @Test
  void xlsxExportAndDeliveryUseApprovedColumnsAndNeverSendAFile() throws Exception {
    String event = str(event(10).get("eventId"));
    tx.execute(
        status ->
            parties.create(
                c,
                s.event(event),
                map("name", "=SUM(A1:A2)", "email", "parent@example.invalid"),
                "BUILTIN",
                true,
                "fixture",
                null));
    s.execute(
        "update ug_event_grant set allowed_actions=allowed_actions || '[\"EXPORT\"]'::jsonb where"
            + " event_id=? and account_id=?",
        uuid(event),
        c.account.get("id"));
    var exported =
        call(
            "ExportCreate",
            map(
                "eventId",
                event,
                "format",
                "XLSX",
                "columns",
                List.of("name", "approvedSize"),
                "purpose",
                "Approved recipient review",
                "filter",
                map(
                    "status",
                    "APPROVED",
                    "progress",
                    "NONE",
                    "sourceType",
                    "BUILTIN",
                    "createdFrom",
                    s.now().minusMinutes(5).toString(),
                    "createdTo",
                    s.now().plusMinutes(5).toString())));
    assertEquals(1, num(exported.get("rowCount")));
    assertEquals(
        "SUCCEEDED",
        ((Map<?, ?>)
                call("ExportDetail", map("eventId", event, "exportId", exported.get("exportId")))
                    .get("status"))
            .get("code"));
    var file =
        tx.execute(
            status ->
                (IntegrationService.Binary)
                    integration.handle(
                        "ExportDownload",
                        map("eventId", event, "exportId", exported.get("exportId")),
                        c,
                        null));
    try (var book =
        new org.apache.poi.xssf.usermodel.XSSFWorkbook(
            new java.io.ByteArrayInputStream(file.data()))) {
      var cell = book.getSheetAt(0).getRow(1).getCell(0);
      assertEquals(org.apache.poi.ss.usermodel.CellType.STRING, cell.getCellType());
      assertEquals("'=SUM(A1:A2)", cell.getStringCellValue());
    }
    var request =
        map(
            "eventId",
            event,
            "exportId",
            exported.get("exportId"),
            "recipient",
            "Approved school representative",
            "purpose",
            "Review attendance",
            "columns",
            List.of("name"),
            "retentionNote",
            "Delete recipient copy after agreed review");
    var delivery = call("DeliveryRecord", request);
    assertEquals("Approved school representative", delivery.get("recipient"));
    assertEquals(1, num(call("DeliveryList", map("eventId", event)).get("total")));
    assertEquals(
        0,
        s.db.queryForObject(
            "select count(*) from ug_job where event_id=? and kind='EMAIL' and business_key like"
                + " 'delivery:%'",
            Integer.class, uuid(event)));
    request.put("columns", List.of("email"));
    assertThrows(ApiException.class, () -> call("DeliveryRecord", request));
    s.execute(
        "update ug_export set expires_at=now()-interval '1 second' where id=?",
        uuid(exported.get("exportId")));
    assertThrows(
        ApiException.class,
        () ->
            tx.execute(
                status ->
                    integration.handle(
                        "ExportDownload",
                        map("eventId", event, "exportId", exported.get("exportId")),
                        c,
                        null)));
  }

  @Test
  void oauthStateIsBoundToOriginalAdministratorSessionAndConsumedOnce() {
    var connection =
        call(
            "ConnectionSave",
            map(
                "provider",
                "FEISHU",
                "accountKind",
                "PERSONAL",
                "displayName",
                "OAuth fixture",
                "allowedEventIds",
                List.of(),
                "clientId",
                "test-app",
                "clientSecret",
                "test-application-secret-123456"));
    var adapter = mock(ProviderAdapter.class);
    when(adapter.authorizeUrl(anyMap(), anyString()))
        .thenAnswer(
            invocation -> "https://accounts.feishu.cn/test?state=" + invocation.getArgument(1));
    when(adapter.exchange(anyMap(), eq("provider-test-code")))
        .thenReturn(
            map(
                "accessToken",
                "test-access-token-never-return",
                "refreshToken",
                "test-refresh-token",
                "expiresIn",
                3600));
    var service =
        new IntegrationService(
            s, parties, adapter, notifications, manager, contract, "./var/unigate-test");
    c.session = map("id", uuid(s.id()));
    var authorization =
        integrationCall(
            service,
            "ConnectionAuthorize",
            map(
                "connectionId",
                connection.get("connectionId"),
                "version",
                connection.get("version")),
            c);
    String state =
        java.net
            .URI
            .create(str(authorization.get("authorizationUrl")))
            .getQuery()
            .substring("state=".length());
    Context wrong = new Context(s);
    wrong.account = c.account;
    wrong.session = map("id", uuid(s.id()));
    assertThrows(
        ApiException.class,
        () ->
            integrationCall(
                service,
                "ConnectionCallback",
                map("state", state, "code", "provider-test-code"),
                wrong));
    verify(adapter, never()).exchange(anyMap(), anyString());
    var result =
        integrationCall(
            service, "ConnectionCallback", map("state", state, "code", "provider-test-code"), c);
    assertEquals("AUTHORIZED", ((Map<?, ?>) result.get("status")).get("code"));
    assertFalse(result.toString().contains("test-access-token-never-return"));
    assertThrows(
        ApiException.class,
        () ->
            integrationCall(
                service,
                "ConnectionCallback",
                map("state", state, "code", "provider-test-code"),
                c));
    verify(adapter, times(1)).exchange(anyMap(), eq("provider-test-code"));
  }
}
