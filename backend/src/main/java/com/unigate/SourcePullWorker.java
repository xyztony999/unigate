package com.unigate;

import static com.unigate.Store.*;

import java.time.OffsetDateTime;
import java.util.*;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;
import org.springframework.transaction.PlatformTransactionManager;
import org.springframework.transaction.TransactionDefinition;
import org.springframework.transaction.support.TransactionTemplate;

/** The provider network request never runs inside a database transaction or event lock. */
@Component
public class SourcePullWorker {
  private final Store s;
  private final ProviderAdapter provider;
  private final TransactionTemplate tx;

  public SourcePullWorker(Store s, ProviderAdapter provider, PlatformTransactionManager manager) {
    this.s = s;
    this.provider = provider;
    tx = new TransactionTemplate(manager);
    tx.setPropagationBehavior(TransactionDefinition.PROPAGATION_REQUIRES_NEW);
  }

  private record Claim(
      Map<String, Object> job,
      Map<String, Object> source,
      Map<String, Object> connection,
      String error) {}

  @Scheduled(fixedDelayString = "${unigate.sources.poll-ms:3000}")
  public void pullNext() {
    try {
      tx.executeWithoutResult(
          status -> {
            s.execute(
                "update ug_job set"
                    + " status='EXPIRED',payload_cipher=null,lease_until=null,completed_at=now(),version=version+1"
                    + " where kind='PULL' and status in ('PENDING','RUNNING') and expires_at<=now()"
                    + " and (lease_until is null or lease_until<=now())");
            s.execute(
                "update ug_job set"
                    + " status='FAILED',lease_until=null,error_code='PROVIDER_UNAVAILABLE',completed_at=now(),version=version+1"
                    + " where kind='PULL' and status='RUNNING' and lease_until<=now() and"
                    + " attempts>=5");
          });
      Claim claim = tx.execute(status -> claim());
      if (claim == null) return;
      List<ProviderAdapter.RemoteRow> records = List.of();
      String error = claim.error();
      if (error == null)
        try {
          records = provider.pull(claim.connection(), str(claim.source().get("resource_id")));
          ApiException.require(!records.isEmpty(), 422, "SOURCE_EMPTY", "来源没有可预览记录");
        } catch (RuntimeException ex) {
          error = ex instanceof ApiException a ? a.code : "PROVIDER_UNAVAILABLE";
        }
      final var result = records;
      final String failure = error;
      tx.executeWithoutResult(status -> finish(claim, result, failure));
    } finally {
      s.eventScope.remove();
    }
  }

  private Claim claim() {
    var ready =
        s.rows(
            "select j.* from ug_job j join ug_event e on j.event_id=e.id where j.kind='PULL' and"
                + " ((j.status='PENDING' and j.next_run_at<=now()) or (j.status='RUNNING' and"
                + " j.lease_until<=now())) and j.attempts<5 and j.expires_at>now() and"
                + " e.personal_due_at>now() order by j.next_run_at,j.id for update of e skip locked"
                + " limit 1");
    if (ready.isEmpty()) return null;
    var previous = s.row("select * from ug_job where id=? for update", ready.getFirst().get("id"));
    boolean pending =
        "PENDING".equals(previous.get("status"))
            && !time(previous.get("next_run_at")).isAfter(s.now());
    boolean abandoned =
        "RUNNING".equals(previous.get("status"))
            && previous.get("lease_until") != null
            && !time(previous.get("lease_until")).isAfter(s.now());
    if (num(previous.get("attempts")) >= 5
        || (!pending && !abandoned)
        || !time(previous.get("expires_at")).isAfter(s.now())) return null;
    s.execute(
        "update ug_job set status='RUNNING',attempts=attempts+1,lease_until=?,version=version+1"
            + " where id=?",
        s.now().plusMinutes(10),
        previous.get("id"));
    var job = s.row("select * from ug_job where id=?", previous.get("id"));
    try {
      var payload = s.unseal(job.get("payload_cipher"));
      var event = s.event(str(job.get("event_id")));
      checkActor(payload, event);
      var source =
          s.row(
              "select * from ug_source where id=? and event_id=?",
              uuid(payload.get("sourceId")),
              event.get("id"));
      ApiException.require(
          num(source.get("version")) == num(payload.get("sourceVersion")),
          409,
          "PREVIEW_CONFLICT",
          "来源已变化");
      var connection = connection(source, event);
      return new Claim(job, source, connection, null);
    } catch (RuntimeException ex) {
      return new Claim(
          job, null, null, ex instanceof ApiException a ? a.code : "PROVIDER_UNAVAILABLE");
    }
  }

  private void checkActor(Map<String, Object> payload, Map<String, Object> event) {
    ApiException.require(
        Set.of("DRAFT", "READY").contains(str(event.get("status"))),
        409,
        "EVENT_STATE_CONFLICT",
        "检票开始后来源拉取已冻结");
    ApiException.require(
        !s.rows(
                "select g.id from ug_event_grant g join ug_account a on g.account_id=a.id where"
                    + " g.event_id=? and g.account_id=? and g.role='ORGANIZER' and g.revoked_at is"
                    + " null and g.expires_at>now() and a.enabled=true and a.roles @>"
                    + " '[\"ORGANIZER\"]'::jsonb",
                event.get("id"),
                uuid(payload.get("actorId")))
            .isEmpty(),
        403,
        "FORBIDDEN",
        "原负责人授权已失效");
  }

  private Map<String, Object> connection(Map<String, Object> source, Map<String, Object> event) {
    var connection =
        s.row(
            "select c.* from ug_connection c join ug_connection_event ce on c.id=ce.connection_id"
                + " where c.id=? and ce.event_id=?",
            source.get("connection_id"),
            event.get("id"));
    ApiException.require(
        !Set.of("REVOKED", "EXPIRED").contains(str(connection.get("status"))),
        503,
        "PROVIDER_UNAVAILABLE",
        "平台连接已失效");
    return connection;
  }

  private void finish(Claim claim, List<ProviderAdapter.RemoteRow> records, String error) {
    var event = s.row("select * from ug_event where id=? for update", claim.job().get("event_id"));
    var jobs = s.rows("select * from ug_job where id=? for update", claim.job().get("id"));
    if (jobs.isEmpty()) return;
    var job = jobs.getFirst();
    if (!"RUNNING".equals(job.get("status"))
        || num(job.get("version")) != num(claim.job().get("version"))) return;
    if (!time(job.get("expires_at")).isAfter(s.now())
        || !time(event.get("personal_due_at")).isAfter(s.now())
        || "ARCHIVED".equals(event.get("status"))) {
      s.execute(
          "update ug_job set"
              + " status='EXPIRED',payload_cipher=null,lease_until=null,completed_at=?,version=version+1"
              + " where id=?",
          s.now(),
          job.get("id"));
      return;
    }
    s.eventScope.set(str(event.get("id")));
    String failure = error;
    Map<String, Object> source = null;
    if (failure == null)
      try {
        var payload = s.unseal(job.get("payload_cipher"));
        checkActor(payload, event);
        source =
            s.row(
                "select * from ug_source where id=? and event_id=? for update",
                claim.source().get("id"),
                event.get("id"));
        ApiException.require(
            num(source.get("version")) == num(claim.source().get("version")),
            409,
            "PREVIEW_CONFLICT",
            "来源已变化");
        var current = connection(source, event);
        ApiException.require(
            num(current.get("version")) == num(claim.connection().get("version")),
            409,
            "PREVIEW_CONFLICT",
            "平台连接已变化");
      } catch (RuntimeException ex) {
        failure = ex instanceof ApiException a ? a.code : "PROVIDER_UNAVAILABLE";
      }
    if (failure != null) {
      boolean retry = "PROVIDER_UNAVAILABLE".equals(failure) && num(job.get("attempts")) < 5;
      long delay = 1L << Math.min(3, Math.max(0, num(job.get("attempts")) - 1));
      s.execute(
          "update ug_job set"
              + " status=?,lease_until=null,error_code=?,next_run_at=?,completed_at=?,version=version+1"
              + " where id=?",
          retry ? "PENDING" : "FAILED",
          failure,
          s.now().plusMinutes(delay),
          retry ? null : s.now(),
          job.get("id"));
      return;
    }
    String id = s.id();
    OffsetDateTime expiry = time(job.get("expires_at"));
    s.execute(
        "insert into"
            + " ug_import_batch(id,event_id,source_id,mapping,eligibility,status,source_version,expires_at)"
            + " values(?,?,?,?::jsonb,?,'UPLOADED',?,?)",
        uuid(id),
        event.get("id"),
        source.get("id"),
        source.get("mapping").toString(),
        source.get("eligibility"),
        source.get("version"),
        expiry);
    int line = 0;
    for (var record : records)
      s.execute(
          "insert into"
              + " ug_import_row(id,event_id,batch_id,source_record_id,line_number,row_cipher,content_digest,status)"
              + " values(?,?,?,?,?,?,?,'ERROR')",
          uuid(s.id()),
          event.get("id"),
          uuid(id),
          record.id(),
          ++line,
          s.seal(map("raw", record.fields())),
          s.crypto.hash(s.stringify(record.fields())));
    s.execute(
        "update ug_job set"
            + " status='SUCCEEDED',error_code=null,result_ref=?,lease_until=null,payload_cipher=null,completed_at=?,version=version+1"
            + " where id=?",
        uuid(id),
        s.now(),
        job.get("id"));
  }
}
