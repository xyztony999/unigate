package com.unigate;

import static com.unigate.Store.*;

import java.io.*;
import java.nio.charset.StandardCharsets;
import java.nio.file.*;
import java.security.MessageDigest;
import java.time.*;
import java.util.*;
import java.util.zip.*;
import org.apache.commons.csv.*;
import org.apache.poi.ss.usermodel.*;
import org.apache.poi.xssf.usermodel.XSSFWorkbook;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
import org.springframework.transaction.PlatformTransactionManager;
import org.springframework.transaction.TransactionDefinition;
import org.springframework.transaction.support.TransactionTemplate;
import org.springframework.web.multipart.MultipartFile;

/**
 * Integration boundaries preserve the event transaction and never grant eligibility from remote
 * fields.
 */
@Service
public class IntegrationService {
  private final Store s;
  private final PartyService parties;
  private final ProviderAdapter providers;
  private final NotificationService notifications;
  private final Contract contract;
  private final TransactionTemplate stateTx;
  private final Path storage;

  public IntegrationService(
      Store s,
      PartyService parties,
      ProviderAdapter providers,
      NotificationService notifications,
      PlatformTransactionManager tx,
      Contract contract,
      @Value("${unigate.storage-directory:./var/unigate}") String directory) {
    this.s = s;
    this.parties = parties;
    this.providers = providers;
    this.notifications = notifications;
    this.contract = contract;
    this.stateTx = new TransactionTemplate(tx);
    stateTx.setPropagationBehavior(TransactionDefinition.PROPAGATION_REQUIRES_NEW);
    this.storage = Path.of(directory).toAbsolutePath().normalize();
  }

  public record Binary(byte[] data, String contentType, String filename) {}

  public Object handle(String op, Map<String, Object> r, Context c, UUID entry) {
    return switch (op) {
      case "ConnectionList" -> connections(r, c);
      case "ConnectionSave" -> saveConnection(r, c);
      case "ConnectionRevoke" -> revokeConnection(r, c);
      case "ConnectionAuthorize" -> authorize(r, c);
      case "ConnectionCallback" -> callback(r, c);
      case "ResourceList", "ResourceFields" -> {
        connectionAccess(r, c);
        var connection =
            s.row("select * from ug_connection where id=?", uuid(r.get("connectionId")));
        yield op.equals("ResourceList")
            ? map("items", providers.list(connection))
            : map(
                "resourceId",
                r.get("resourceId"),
                "availableColumns",
                providers.fields(connection, str(r.get("resourceId"))));
      }
      case "SourceSave" -> saveSource(r, c);
      case "SourceList" -> sources(r, c);
      case "SourcePull" -> pull(r, c);
      case "SmsStart", "SmsVerify", "RecoveryStart", "RecoveryVerify" ->
          notifications.handle(op, r, c);
      case "BatchUpload" -> upload(r, c);
      case "BatchDetail" -> {
        event(r, c);
        yield batchView(batch(r));
      }
      case "BatchPreview" -> preview(r, c);
      case "BatchRows" -> rows(r, c);
      case "DuplicateResolve" -> resolve(r, c);
      case "BatchCommit" -> commit(r, c, entry);
      case "JobList" -> jobs(r, c);
      case "JobDetail" -> {
        var j = job(r, c);
        yield jobView(j);
      }
      case "JobRetry" -> retry(r, c);
      case "EmailResend" -> email(r, c);
      case "ExportCreate" -> export(r, c);
      case "ExportDetail" -> {
        event(r, c);
        var x = exportRow(r, c);
        yield exportView(x);
      }
      case "ExportDownload" -> download(r, c);
      case "DeliveryList" -> deliveries(r, c);
      case "DeliveryRecord" -> delivery(r, c);
      case "CleanupDetail" -> cleanupDetail(r, c);
      case "CleanupRetry" -> cleanupRetry(r, c);
      default -> throw new ApiException(404, "RESOURCE_NOT_FOUND", "接口不存在");
    };
  }

  private void admin(Context c) {
    ApiException.require(c.admin(), 403, "FORBIDDEN", "需要管理员权限");
  }

  private Map<String, Object> event(Map<String, Object> r, Context c) {
    String id = str(r.get("eventId"));
    c.organizer(id);
    return s.event(id);
  }

  private void intakeOpen(Map<String, Object> e) {
    ApiException.require(
        Set.of("DRAFT", "READY").contains(str(e.get("status"))),
        409,
        "EVENT_STATE_CONFLICT",
        "检票开始后名单导入已冻结");
  }

  private void connectionAccess(Map<String, Object> r, Context c) {
    if (c.admin()) s.row("select id from ug_connection where id=?", uuid(r.get("connectionId")));
    else {
      c.organizer(str(r.get("eventId")));
      connectionAllowed(str(r.get("connectionId")), str(r.get("eventId")));
    }
  }

  private Map<String, Object> connectionAllowed(String id, String event) {
    return s.row(
        "select c.* from ug_connection c join ug_connection_event ce on c.id=ce.connection_id where"
            + " c.id=? and ce.event_id=?",
        uuid(id),
        uuid(event));
  }

  private Object connections(Map<String, Object> r, Context c) {
    admin(c);
    List<Map<String, Object>> all = s.rows("select * from ug_connection order by id");
    return s.page(
        all.stream()
            .filter(x -> !r.containsKey("provider") || r.get("provider").equals(x.get("provider")))
            .map(this::connectionView)
            .toList(),
        r);
  }

  private Map<String, Object> connectionView(Map<String, Object> x) {
    var out =
        map(
            "connectionId",
            x.get("id"),
            "provider",
            s.value("Provider", x.get("provider")),
            "accountKind",
            s.value("AccountKind", x.get("account_kind")),
            "displayName",
            x.get("display_name"),
            "status",
            s.value("ConnectionStatus", x.get("status")),
            "version",
            x.get("version"),
            "expiresAt",
            x.get("expires_at"),
            "allowedEventIds",
            s
                .rows(
                    "select event_id from ug_connection_event where connection_id=? order by"
                        + " event_id",
                    x.get("id"))
                .stream()
                .map(y -> y.get("event_id"))
                .toList());
    if (x.get("credentials_cipher") != null) out.put("credentialMask", "••••••••");
    return out;
  }

  private Object saveConnection(Map<String, Object> r, Context c) {
    admin(c);
    String id = str(r.get("connectionId"));
    var secret = map("clientSecret", r.get("clientSecret"));
    if (id == null) {
      id = s.id();
      s.execute(
          "insert into"
              + " ug_connection(id,provider,account_kind,display_name,status,client_id,credentials_cipher,created_by)"
              + " values(?,?,?,?,?,?,?,?)",
          uuid(id),
          r.get("provider"),
          r.get("accountKind"),
          r.get("displayName"),
          "UNAUTHORIZED",
          r.get("clientId"),
          secret.isEmpty() ? null : s.seal(secret),
          uuid(c.actorId()));
    } else {
      var old = s.row("select * from ug_connection where id=? for update", uuid(id));
      s.version(old, r.get("version"));
      ApiException.require(
          r.get("provider").equals(old.get("provider")), 409, "PROVIDER_CONFLICT", "既有连接不能更换提供方");
      s.execute(
          "update ug_connection set"
              + " account_kind=?,display_name=?,client_id=coalesce(?,client_id),credentials_cipher=coalesce(?,credentials_cipher),status='UNAUTHORIZED',version=version+1"
              + " where id=?",
          r.get("accountKind"),
          r.get("displayName"),
          r.get("clientId"),
          secret.isEmpty() ? null : s.seal(secret),
          uuid(id));
    }
    Set<String> requested = new HashSet<>();
    for (Object e : (List<?>) r.get("allowedEventIds")) requested.add(str(e));
    for (var existing :
        s.rows("select event_id from ug_connection_event where connection_id=?", uuid(id))) {
      if (!requested.contains(str(existing.get("event_id")))) {
        ApiException.require(
            s.rows(
                    "select id from ug_source where connection_id=? and event_id=?",
                    uuid(id),
                    existing.get("event_id"))
                .isEmpty(),
            409,
            "SOURCE_IN_USE",
            "请先解除活动来源对该连接的引用");
        s.execute(
            "delete from ug_connection_event where connection_id=? and event_id=?",
            uuid(id),
            existing.get("event_id"));
      }
    }
    for (String e : requested) {
      s.row("select id from ug_event where id=?", uuid(e));
      s.execute(
          "insert into ug_connection_event(connection_id,event_id) values(?,?) on conflict do"
              + " nothing",
          uuid(id),
          uuid(e));
    }
    return connectionView(s.row("select * from ug_connection where id=?", uuid(id)));
  }

  private Object revokeConnection(Map<String, Object> r, Context c) {
    admin(c);
    var x = s.row("select * from ug_connection where id=? for update", uuid(r.get("connectionId")));
    s.version(x, r.get("version"));
    s.execute(
        "update ug_connection set"
            + " status='REVOKED',credentials_cipher=null,key_ref=null,expires_at=null,version=version+1"
            + " where id=?",
        x.get("id"));
    return connectionView(s.row("select * from ug_connection where id=?", x.get("id")));
  }

  private Object authorize(Map<String, Object> r, Context c) {
    admin(c);
    var connection =
        s.row("select * from ug_connection where id=? for update", uuid(r.get("connectionId")));
    s.version(connection, r.get("version"));
    String state = s.crypto.token(), id = s.id();
    String url = providers.authorizeUrl(connection, state);
    var expires = s.now().plusMinutes(5);
    s.execute(
        "insert into"
            + " ug_challenge(id,kind,connection_id,actor_scope,code_digest,context_cipher,expires_at)"
            + " values(?,'OAUTH',?,?,?,?,?)",
        uuid(id),
        connection.get("id"),
        c.actorId(),
        s.crypto.hash(state),
        s.seal(
            map("sessionId", c.session.get("id"), "connectionVersion", connection.get("version"))),
        expires);
    return map("authorizationUrl", url, "expiresAt", expires);
  }

  private Object callback(Map<String, Object> r, Context c) {
    admin(c);
    var challenge =
        stateTx.execute(
            status -> {
              var found =
                  s.row(
                      "select * from ug_challenge where kind='OAUTH' and code_digest=? for update",
                      s.crypto.hash(str(r.get("state"))));
              var context = s.unseal(found.get("context_cipher"));
              ApiException.require(
                  found.get("consumed_at") == null
                      && time(found.get("expires_at")).isAfter(s.now())
                      && c.actorId().equals(str(found.get("actor_scope")))
                      && c.session != null
                      && str(c.session.get("id")).equals(str(context.get("sessionId"))),
                  403,
                  "OAUTH_STATE_INVALID",
                  "授权状态无效，请重新发起授权");
              s.execute(
                  "update ug_challenge set consumed_at=? where id=?", s.now(), found.get("id"));
              return found;
            });
    var connection =
        s.row("select * from ug_connection where id=? for update", challenge.get("connection_id"));
    s.version(connection, s.unseal(challenge.get("context_cipher")).get("connectionVersion"));
    var exchanged = providers.exchange(connection, str(r.get("code")));
    var credentials = s.unseal(connection.get("credentials_cipher"));
    credentials.putAll(exchanged);
    Object expiry = s.now().plusSeconds(num(exchanged.get("expiresIn")));
    credentials.remove("expiresIn");
    s.execute(
        "update ug_connection set"
            + " credentials_cipher=?,status='AUTHORIZED',expires_at=?,version=version+1 where id=?",
        s.seal(credentials),
        expiry,
        connection.get("id"));
    return connectionView(s.row("select * from ug_connection where id=?", connection.get("id")));
  }

  private Map<String, Object> source(Map<String, Object> r) {
    return s.row(
        "select a.*,c.provider from ug_source a join ug_connection c on a.connection_id=c.id where"
            + " a.id=? and a.event_id=? for update of a",
        uuid(r.get("sourceId")),
        uuid(r.get("eventId")));
  }

  private Object saveSource(Map<String, Object> r, Context c) {
    intakeOpen(event(r, c));
    var con = connectionAllowed(str(r.get("connectionId")), str(r.get("eventId")));
    ApiException.require(
        Set.of("FEISHU", "TENCENT").contains(str(con.get("provider"))),
        422,
        "INVALID_PROVIDER",
        "来源仅支持飞书或腾讯文档");
    String id = str(r.get("sourceId"));
    if (id == null) {
      id = s.id();
      s.execute(
          "insert into ug_source(id,event_id,connection_id,resource_id,mapping,eligibility)"
              + " values(?,?,?,?,?::jsonb,?)",
          uuid(id),
          uuid(r.get("eventId")),
          uuid(r.get("connectionId")),
          r.get("resourceId"),
          s.stringify(r.get("mapping")),
          r.get("eligibility"));
    } else {
      s.version(source(r), r.get("version"));
      s.execute(
          "update ug_source set"
              + " connection_id=?,resource_id=?,mapping=?::jsonb,eligibility=?,version=version+1"
              + " where id=?",
          uuid(r.get("connectionId")),
          r.get("resourceId"),
          s.stringify(r.get("mapping")),
          r.get("eligibility"),
          uuid(id));
    }
    var q = new LinkedHashMap<>(r);
    q.put("sourceId", id);
    return sourceView(source(q));
  }

  private Map<String, Object> mappingView(Object raw) {
    var m = new LinkedHashMap<>(s.parse(raw));
    m.put("countMode", s.value("CountMode", m.get("countMode")));
    return m;
  }

  private Map<String, Object> sourceView(Map<String, Object> x) {
    return map(
        "sourceId",
        x.get("id"),
        "eventId",
        x.get("event_id"),
        "connectionId",
        x.get("connection_id"),
        "resourceId",
        x.get("resource_id"),
        "provider",
        s.value("Provider", x.get("provider")),
        "mapping",
        mappingView(x.get("mapping")),
        "eligibility",
        s.value("SourceEligibility", x.get("eligibility")),
        "version",
        x.get("version"));
  }

  private Object sources(Map<String, Object> r, Context c) {
    event(r, c);
    return map(
        "items",
        s
            .rows(
                "select a.*,c.provider from ug_source a join ug_connection c on"
                    + " a.connection_id=c.id where a.event_id=? order by a.id",
                uuid(r.get("eventId")))
            .stream()
            .map(this::sourceView)
            .toList(),
        "availableConnections",
        s
            .rows(
                "select c.* from ug_connection c join ug_connection_event ce on"
                    + " c.id=ce.connection_id where ce.event_id=? and c.provider in"
                    + " ('FEISHU','TENCENT') order by c.id",
                uuid(r.get("eventId")))
            .stream()
            .map(
                x ->
                    map(
                        "connectionId",
                        x.get("id"),
                        "displayName",
                        x.get("display_name"),
                        "provider",
                        s.value("Provider", x.get("provider")),
                        "accountKind",
                        s.value("AccountKind", x.get("account_kind")),
                        "status",
                        s.value("ConnectionStatus", x.get("status"))))
            .toList());
  }

  private Object upload(Map<String, Object> r, Context c) {
    var e = event(r, c);
    intakeOpen(e);
    ApiException.require(
        r.get("file") instanceof MultipartFile, 422, "INVALID_FILE", "请选择CSV或XLSX文件");
    var f = (MultipartFile) r.get("file");
    try {
      byte[] bytes = f.getBytes();
      ApiException.require(
          bytes.length > 0 && bytes.length <= 10 * 1024 * 1024,
          422,
          "INVALID_FILE",
          "文件大小须在10MiB内");
      String name =
          Optional.ofNullable(f.getOriginalFilename()).orElse("").toLowerCase(Locale.ROOT);
      List<Map<String, Object>> records = parseFile(bytes, name);
      String digest = HexFormat.of().formatHex(MessageDigest.getInstance("SHA-256").digest(bytes));
      String id = s.id();
      var expiry = min(s.now().plusHours(24), time(e.get("personal_due_at")));
      s.execute(
          "insert into"
              + " ug_import_batch(id,event_id,file_digest,sheet_ref,source_label,status,expires_at)"
              + " values(?,?,?,?,?,'UPLOADED',?)",
          uuid(id),
          e.get("id"),
          digest,
          records.getFirst() instanceof ImportRecord record ? record.sheetRef : "csv",
          r.get("sourceLabel"),
          expiry);
      int line = 0;
      for (var fields : records) {
        line++;
        s.execute(
            "insert into"
                + " ug_import_row(id,event_id,batch_id,source_record_id,line_number,row_cipher,content_digest,status)"
                + " values(?,?,?,?,?,?,?,'ERROR')",
            uuid(s.id()),
            e.get("id"),
            uuid(id),
            String.valueOf(line),
            line,
            s.seal(
                map(
                    "raw",
                    fields,
                    "fileErrors",
                    fields instanceof ImportRecord record && record.formula
                        ? List.of(
                            map(
                                "field",
                                "fields",
                                "code",
                                "FORMULA_NOT_ALLOWED",
                                "message",
                                "此行含公式，请先转换为纯值"))
                        : List.of())),
            s.crypto.hash(s.stringify(fields)));
      }
      var q = map("eventId", e.get("id"), "batchId", id);
      return batchView(batch(q));
    } catch (ApiException ex) {
      throw ex;
    } catch (Exception ex) {
      throw new ApiException(422, "INVALID_FILE", "无法读取文件，请检查编码和文件格式");
    }
  }

  static class ImportRecord extends LinkedHashMap<String, Object> {
    boolean formula;
    String sheetRef;
  }

  static List<Map<String, Object>> parseFile(byte[] bytes, String name) throws Exception {
    List<Map<String, Object>> rows = new ArrayList<>();
    if (name.endsWith(".csv")) {
      String text = new String(bytes, StandardCharsets.UTF_8);
      ApiException.require(!text.contains("\uFFFD"), 422, "INVALID_ENCODING", "CSV须使用UTF-8编码");
      if (text.startsWith("\uFEFF")) text = text.substring(1);
      try (var parser =
          CSVFormat.DEFAULT
              .builder()
              .setHeader()
              .setSkipHeaderRecord(true)
              .setAllowDuplicateHeaderNames(false)
              .get()
              .parse(new StringReader(text))) {
        validateHeaders(parser.getHeaderNames());
        for (var rec : parser) {
          ApiException.require(rec.isConsistent(), 422, "INVALID_FILE", "CSV列数不一致");
          rows.add(new LinkedHashMap<>(rec.toMap()));
          ApiException.require(rows.size() <= 10000, 422, "INVALID_FILE", "最多10000行");
        }
      }
    } else if (name.endsWith(".xlsx")) {
      long size = 0;
      try (var zip = new ZipInputStream(new ByteArrayInputStream(bytes))) {
        byte[] buffer = new byte[8192];
        ZipEntry ent;
        while ((ent = zip.getNextEntry()) != null) {
          ApiException.require(
              !ent.getName().toLowerCase(Locale.ROOT).contains("vbaproject"),
              422,
              "INVALID_FILE",
              "不接受宏文件");
          int n;
          while ((n = zip.read(buffer)) != -1) {
            size += n;
            ApiException.require(size <= 50L * 1024 * 1024, 422, "INVALID_FILE", "解压文件过大");
          }
        }
      }
      try (var book = new XSSFWorkbook(new ByteArrayInputStream(bytes))) {
        Sheet sheet = null;
        int sheetIndex = 0, headerIndex = 0;
        for (int index = 0; index < book.getNumberOfSheets() && sheet == null; index++) {
          var candidate = book.getSheetAt(index);
          for (var row : candidate) {
            boolean nonempty = false;
            for (var cell : row)
              if (cell.getCellType() != CellType.BLANK && !cell.toString().isBlank()) {
                nonempty = true;
                break;
              }
            if (nonempty) {
              sheet = candidate;
              sheetIndex = index;
              headerIndex = row.getRowNum();
              break;
            }
          }
        }
        ApiException.require(sheet != null, 422, "INVALID_FILE", "缺少非空工作表");
        ApiException.require(
            sheet.getLastRowNum() - headerIndex <= 10000, 422, "INVALID_FILE", "最多10000行");
        var first = sheet.getRow(headerIndex);
        ApiException.require(first != null, 422, "INVALID_FILE", "缺少表头");
        List<String> headers = new ArrayList<>();
        DataFormatter fmt = new DataFormatter(Locale.ROOT);
        for (int i = 0; i < first.getLastCellNum(); i++) {
          var cell = first.getCell(i);
          ApiException.require(
              cell == null || cell.getCellType() != CellType.FORMULA, 422, "INVALID_FILE", "不接受公式");
          headers.add(cell == null ? "" : fmt.formatCellValue(cell));
        }
        validateHeaders(headers);
        for (int i = headerIndex + 1; i <= sheet.getLastRowNum(); i++) {
          var row = sheet.getRow(i);
          if (row == null) continue;
          ImportRecord record = new ImportRecord();
          record.sheetRef = "sheet:" + sheetIndex + ":" + sheet.getSheetName();
          for (int j = 0; j < headers.size(); j++) {
            var cell = row.getCell(j);
            if (cell != null && cell.getCellType() == CellType.FORMULA) {
              record.formula = true;
              record.put(headers.get(j), "");
            } else record.put(headers.get(j), cell == null ? "" : fmt.formatCellValue(cell));
          }
          rows.add(record);
        }
      }
    } else throw new ApiException(422, "INVALID_FILE", "仅接受CSV或XLSX文件");
    ApiException.require(!rows.isEmpty(), 422, "INVALID_FILE", "名单不能为空");
    return rows;
  }

  private static void validateHeaders(List<String> headers) {
    ApiException.require(
        !headers.isEmpty()
            && headers.size() <= 200
            && new HashSet<>(headers).size() == headers.size()
            && headers.stream().allMatch(x -> x != null && !x.isBlank() && x.length() <= 200),
        422,
        "INVALID_FILE",
        "表头须唯一非空，最多200列");
  }

  private Map<String, Object> batch(Map<String, Object> r) {
    var b =
        s.row(
            "select * from ug_import_batch where id=? and event_id=? for update",
            uuid(r.get("batchId")),
            uuid(r.get("eventId")));
    ApiException.require(
        time(b.get("expires_at")).isAfter(s.now()), 410, "BATCH_EXPIRED", "暂存名单已过期");
    return b;
  }

  private List<Map<String, Object>> batchRows(Map<String, Object> b) {
    return s.rows(
        "select * from ug_import_row where batch_id=? order by line_number,id", b.get("id"));
  }

  private Map<String, Object> batchView(Map<String, Object> b) {
    var rows = batchRows(b);
    var first =
        rows.isEmpty() ? map() : s.parse(s.unseal(rows.getFirst().get("row_cipher")).get("raw"));
    var counts =
        rows.stream()
            .collect(
                java.util.stream.Collectors.groupingBy(
                    x -> str(x.get("status")), java.util.stream.Collectors.counting()));
    var out =
        map(
            "batchId",
            b.get("id"),
            "eventId",
            b.get("event_id"),
            "sourceId",
            b.get("source_id"),
            "fileDigest",
            b.get("file_digest"),
            "status",
            s.value("BatchStatus", b.get("status")),
            "previewVersion",
            b.get("preview_version"),
            "validRows",
            counts.getOrDefault("VALID", 0L),
            "errorRows",
            counts.getOrDefault("ERROR", 0L),
            "suspectRows",
            counts.getOrDefault("DUPLICATE_SUSPECT", 0L),
            "expiresAt",
            b.get("expires_at"),
            "version",
            b.get("version"),
            "policyVersion",
            b.get("policy_version"),
            "availableColumns",
            first.keySet().stream().map(x -> map("code", x, "label", x)).toList());
    if (b.get("mapping") != null) out.put("mapping", mappingView(b.get("mapping")));
    if (b.get("eligibility") != null)
      out.put("eligibility", s.value("SourceEligibility", b.get("eligibility")));
    return out;
  }

  static Map<String, Object> normalize(Map<String, Object> raw, Map<String, Object> mapping) {
    var fields = map("companionCount", 0);
    for (String f : List.of("name", "phone", "email")) {
      Object col = mapping.get(f + "Column");
      if (col != null) {
        String val = String.valueOf(raw.getOrDefault(col.toString(), "")).strip();
        if (!val.isEmpty()) {
          if (f.equals("phone") || f.equals("email"))
            val = NotificationService.normalizeContact(val, f.equals("email") ? "EMAIL" : "SMS");
          fields.put(f, val);
        }
      }
    }
    if (mapping.get("countColumn") != null) {
      String val = String.valueOf(raw.getOrDefault(mapping.get("countColumn"), "")).strip();
      int n = val.isBlank() ? 0 : Integer.parseInt(val);
      if ("TOTAL".equals(mapping.get("countMode"))) {
        ApiException.require(n >= 1, 422, "INVALID_COUNT", "总人数至少为1");
        n--;
      }
      ApiException.require(n >= 0 && n < 1000000, 422, "INVALID_COUNT", "人数超出范围");
      fields.put("companionCount", n);
    }
    return fields;
  }

  private String sourceScope(Map<String, Object> b) {
    String identity = str(b.get("file_digest")) + ":" + str(b.get("sheet_ref"));
    if (b.get("source_id") != null) {
      var source =
          s.row("select connection_id,resource_id from ug_source where id=?", b.get("source_id"));
      identity = source.get("connection_id") + ":" + source.get("resource_id");
    }
    return HexFormat.of().formatHex(s.crypto.hash(str(b.get("event_id")) + ":" + identity));
  }

  private Object pull(Map<String, Object> r, Context c) {
    var e = event(r, c);
    intakeOpen(e);
    var source = source(r);
    s.version(source, r.get("version"));
    connectionAllowed(str(source.get("connection_id")), str(e.get("id")));
    var expiry = min(s.now().plusHours(24), time(e.get("personal_due_at")));
    return jobView(
        newJob(
            str(e.get("id")),
            "PULL",
            "pull:" + r.get("operationId"),
            "PENDING",
            map(
                "sourceId",
                source.get("id"),
                "sourceVersion",
                source.get("version"),
                "actorId",
                c.actorId()),
            null,
            null,
            expiry));
  }

  private Object preview(Map<String, Object> r, Context c) {
    var e = event(r, c);
    intakeOpen(e);
    var b = batch(r);
    s.version(b, r.get("version"));
    ApiException.require(
        !"COMMITTED".equals(b.get("status")), 409, "BATCH_STATE_CONFLICT", "批次已提交");
    var mapping = s.parse(r.get("mapping"));
    var columns = batchView(b).get("availableColumns");
    Set<String> known = new HashSet<>();
    for (Object x : (List<?>) columns) known.add(str(((Map<?, ?>) x).get("code")));
    for (String k : List.of("nameColumn", "phoneColumn", "emailColumn", "countColumn"))
      if (mapping.containsKey(k))
        ApiException.require(known.contains(str(mapping.get(k))), 422, "MAPPING_INVALID", "映射列不存在");
    Set<String> observed = new HashSet<>();
    var candidates = s.rows("select * from ug_party where event_id=?", e.get("id"));
    for (var row : batchRows(b)) {
      var stored = s.unseal(row.get("row_cipher"));
      var raw = s.parse(stored.get("raw"));
      Map<String, Object> fields = map("companionCount", 0);
      List<Map<String, Object>> errors = new ArrayList<>();
      String status = "VALID";
      try {
        ApiException.require(
            s.list(stored.get("fileErrors")).isEmpty(), 422, "FORMULA_NOT_ALLOWED", "此行含公式");
        fields = normalize(raw, mapping);
        var rules = s.parse(s.parse(e.get("policy")).get("fields"));
        if (!Boolean.TRUE.equals(s.parse(rules.get("companionCount")).get("enabled"))
            && !mapping.containsKey("countColumn")) fields.remove("companionCount");
        contract.validate("PartyInput", fields);
        parties.validateFields(e, fields);
      } catch (RuntimeException ex) {
        status = "ERROR";
        errors.add(map("field", "fields", "code", "INVALID_FIELD", "message", "请检查映射、必填字段和人数"));
      }
      String key = HexFormat.of().formatHex(s.crypto.hash(str(row.get("source_record_id"))));
      var links =
          s.rows(
              "select party_id,source_content_digest from ug_source_link where source_scope=? and"
                  + " record_key=?",
              sourceScope(b),
              key);
      if (!links.isEmpty()) {
        if (MessageDigest.isEqual(
            (byte[]) links.getFirst().get("source_content_digest"),
            (byte[]) row.get("content_digest"))) status = "ALREADY_LINKED";
        else {
          status = "ERROR";
          errors.add(
              map(
                  "field",
                  "sourceRecordId",
                  "code",
                  "SOURCE_CHANGED",
                  "message",
                  "同源记录内容已变化，请负责人核对既有组"));
        }
      }
      if (status.equals("VALID")) {
        for (String f : List.of("phone", "email")) {
          String v = str(fields.get(f));
          if (v != null) {
            String duplicateKey = f + ":" + v;
            if (!observed.add(duplicateKey)) status = "DUPLICATE_SUSPECT";
            for (var p : candidates) {
              var other = s.unseal(p.get("fields_cipher"));
              if (v.equals(other.get(f))) status = "DUPLICATE_SUSPECT";
            }
          }
        }
      }
      s.execute(
          "update ug_import_row set"
              + " row_cipher=?,status=?,errors=?::jsonb,requested_size=?,resolution=null,linked_party_id=null,reason_cipher=null"
              + " where id=?",
          s.seal(
              map(
                  "raw",
                  raw,
                  "fields",
                  fields,
                  "fileErrors",
                  stored.getOrDefault("fileErrors", List.of()))),
          status,
          s.stringify(errors),
          1 + num(fields.get("companionCount")),
          row.get("id"));
    }
    s.execute(
        "update ug_import_batch set"
            + " mapping=?::jsonb,eligibility=?,status='PREVIEWED',policy_version=?,preview_version=preview_version+1,preview_digest=?,version=version+1"
            + " where id=?",
        s.stringify(mapping),
        r.get("eligibility"),
        e.get("policy_version"),
        s.crypto.hash(s.stringify(mapping) + ":" + e.get("policy_version")),
        b.get("id"));
    return batchView(batch(r));
  }

  private Object rows(Map<String, Object> r, Context c) {
    event(r, c);
    var b = batch(r);
    return s.page(
        batchRows(b).stream()
            .filter(x -> !r.containsKey("status") || r.get("status").equals(x.get("status")))
            .map(
                x ->
                    map(
                        "rowId",
                        x.get("id"),
                        "sourceRecordId",
                        x.get("source_record_id"),
                        "lineNumber",
                        x.get("line_number"),
                        "fields",
                        s.unseal(x.get("row_cipher"))
                            .getOrDefault("fields", map("companionCount", 0)),
                        "requestedSize",
                        Math.max(1, num(x.get("requested_size"))),
                        "status",
                        s.value("RowStatus", x.get("status")),
                        "errors",
                        s.list(x.get("errors")),
                        "linkedPartyId",
                        x.get("linked_party_id")))
            .toList(),
        r);
  }

  private Object resolve(Map<String, Object> r, Context c) {
    var e = event(r, c);
    intakeOpen(e);
    var b = batch(r);
    previewVersion(b, r);
    var row =
        s.row(
            "select * from ug_import_row where id=? and batch_id=? for update",
            uuid(r.get("rowId")),
            b.get("id"));
    ApiException.require(
        "DUPLICATE_SUSPECT".equals(row.get("status")), 409, "ROW_STATE_CONFLICT", "此行不需要重复核对");
    String resolution = str(r.get("resolution"));
    Object linked = null;
    if ("LINK_EXISTING".equals(resolution)) {
      var p = parties.raw(str(e.get("id")), str(r.get("existingPartyId")));
      linked = p.get("id");
    }
    s.execute(
        "update ug_import_row set resolution=?,linked_party_id=?,reason_cipher=?,status=? where"
            + " id=?",
        resolution,
        linked,
        s.seal(map("reason", r.get("reason"))),
        "EXCLUDE".equals(resolution) ? "EXCLUDED" : "VALID",
        row.get("id"));
    s.execute(
        "update ug_import_batch set preview_version=preview_version+1,version=version+1 where id=?",
        b.get("id"));
    return batchView(batch(r));
  }

  private void previewVersion(Map<String, Object> b, Map<String, Object> r) {
    ApiException.require(
        "PREVIEWED".equals(b.get("status"))
            && num(b.get("preview_version")) == num(r.get("previewVersion")),
        409,
        "PREVIEW_CONFLICT",
        "请重新预览并核对名单");
  }

  private Object commit(Map<String, Object> r, Context c, UUID operation) {
    var e = event(r, c);
    ApiException.require(
        "READY".equals(e.get("status")), 409, "EVENT_STATE_CONFLICT", "仅已发布准备中的活动可提交名单");
    var b = batch(r);
    previewVersion(b, r);
    ApiException.require(
        num(e.get("policy_version")) == num(b.get("policy_version")),
        409,
        "PREVIEW_CONFLICT",
        "活动字段规则已变化");
    if (b.get("source_id") != null) {
      var source = s.row("select * from ug_source where id=? for update", b.get("source_id"));
      ApiException.require(
          num(source.get("version")) == num(b.get("source_version")),
          409,
          "PREVIEW_CONFLICT",
          "来源配置已变化");
      connectionAllowed(str(source.get("connection_id")), str(e.get("id")));
    }
    Set<String> selected = new HashSet<>();
    for (Object id : (List<?>) r.get("selectedRowIds")) selected.add(str(id));
    var chosen = batchRows(b).stream().filter(x -> selected.contains(str(x.get("id")))).toList();
    ApiException.require(chosen.size() == selected.size(), 409, "PREVIEW_CONFLICT", "存在未知预览行");
    int required = 0;
    for (var row : chosen) {
      ApiException.require(
          "VALID".equals(row.get("status")), 409, "ROW_STATE_CONFLICT", "选中行包含错误或未经核对的重复");
      parties.validateFields(e, s.parse(s.unseal(row.get("row_cipher")).get("fields")));
      if (row.get("linked_party_id") == null && "ELIGIBLE".equals(b.get("eligibility")))
        required += num(row.get("requested_size"));
    }
    ApiException.require(
        num(e.get("allocated_count")) + required <= num(e.get("capacity")),
        409,
        "CAPACITY_EXCEEDED",
        "剩余名额不足，整批未提交");
    int created = 0, linked = 0;
    List<String> lockIds =
        chosen.stream()
            .filter(x -> x.get("linked_party_id") != null)
            .map(x -> str(x.get("linked_party_id")))
            .distinct()
            .sorted()
            .toList();
    for (String id : lockIds) parties.raw(str(e.get("id")), id);
    for (var row : chosen) {
      String partyId = str(row.get("linked_party_id"));
      String key = HexFormat.of().formatHex(s.crypto.hash(str(row.get("source_record_id"))));
      ApiException.require(
          s.rows(
                  "select id from ug_source_link where source_scope=? and record_key=?",
                  sourceScope(b),
                  key)
              .isEmpty(),
          409,
          "SOURCE_CONFLICT",
          "同源记录已关联，整批未提交");
      if (partyId == null) {
        var fields = s.parse(s.unseal(row.get("row_cipher")).get("fields"));
        var fresh = s.row("select * from ug_event where id=? for update", e.get("id"));
        String sourceType =
            b.get("source_id") == null
                ? "FILE"
                : str(
                    s.row(
                            "select c.provider from ug_source a join ug_connection c on"
                                + " a.connection_id=c.id where a.id=?",
                            b.get("source_id"))
                        .get("provider"));
        var p =
            parties.create(
                c,
                fresh,
                fields,
                sourceType,
                "ELIGIBLE".equals(b.get("eligibility")),
                "import-" + b.get("policy_version"),
                operation);
        partyId = str(p.get("partyId"));
        created++;
      } else linked++;
      s.execute(
          "insert into"
              + " ug_source_link(id,event_id,party_id,source_scope,record_key,source_content_digest,batch_id,linked_at)"
              + " values(?,?,?,?,?,?,?,?)",
          uuid(s.id()),
          e.get("id"),
          uuid(partyId),
          sourceScope(b),
          key,
          row.get("content_digest"),
          b.get("id"),
          s.now());
      s.execute(
          "update ug_import_row set status='COMMITTED',linked_party_id=? where id=?",
          uuid(partyId),
          row.get("id"));
    }
    s.execute(
        "update ug_import_batch set status='COMMITTED',committed_operation_id=?,version=version+1"
            + " where id=?",
        operation,
        b.get("id"));
    return map(
        "batchId",
        b.get("id"),
        "operationId",
        r.get("operationId"),
        "createdCount",
        created,
        "linkedCount",
        linked,
        "allocatedPeople",
        required,
        "replayed",
        false);
  }

  private Map<String, Object> jobView(Map<String, Object> j) {
    return map(
        "jobId",
        j.get("id"),
        "eventId",
        j.get("event_id"),
        "kind",
        s.value("JobKind", j.get("kind")),
        "status",
        s.value("JobStatus", j.get("status")),
        "attempts",
        j.get("attempts"),
        "createdAt",
        j.get("created_at"),
        "completedAt",
        j.get("completed_at"),
        "resultRef",
        j.get("result_ref"),
        "errorCode",
        j.get("error_code"),
        "expiresAt",
        j.get("expires_at"),
        "version",
        j.get("version"));
  }

  private Map<String, Object> job(Map<String, Object> r, Context c) {
    var j = s.row("select * from ug_job where id=? for update", uuid(r.get("jobId")));
    if (!c.admin()) {
      c.organizer(str(r.get("eventId")));
      ApiException.require(
          Objects.equals(str(j.get("event_id")), str(r.get("eventId"))),
          404,
          "RESOURCE_NOT_FOUND",
          "未找到任务");
    }
    return j;
  }

  private Object jobs(Map<String, Object> r, Context c) {
    List<Map<String, Object>> all;
    if (c.admin() && !r.containsKey("eventId"))
      all = s.rows("select * from ug_job order by created_at desc,id desc");
    else {
      if (!c.admin()) c.organizer(str(r.get("eventId")));
      all =
          s.rows(
              "select * from ug_job where event_id=? order by created_at desc,id desc",
              uuid(r.get("eventId")));
    }
    return s.page(
        all.stream()
            .filter(x -> !r.containsKey("kind") || r.get("kind").equals(x.get("kind")))
            .map(this::jobView)
            .toList(),
        r);
  }

  private Object retry(Map<String, Object> r, Context c) {
    var j = job(r, c);
    s.version(j, r.get("version"));
    ApiException.require("FAILED".equals(j.get("status")), 409, "JOB_STATE_CONFLICT", "只能重试失败任务");
    ApiException.require(
        j.get("expires_at") == null || time(j.get("expires_at")).isAfter(s.now()),
        410,
        "JOB_EXPIRED",
        "任务已过期");
    if ("PULL".equals(j.get("kind"))) {
      var e = s.event(str(j.get("event_id")));
      intakeOpen(e);
      var payload = s.unseal(j.get("payload_cipher"));
      var source =
          s.row(
              "select * from ug_source where id=? and event_id=? for update",
              uuid(payload.get("sourceId")),
              j.get("event_id"));
      ApiException.require(
          num(source.get("version")) == num(payload.get("sourceVersion")),
          409,
          "PREVIEW_CONFLICT",
          "来源配置已变化，请重新发起拉取");
      s.execute(
          "update ug_job set"
              + " status='PENDING',attempts=0,error_code=null,lease_until=null,next_run_at=?,completed_at=null,version=version+1"
              + " where id=?",
          s.now(),
          j.get("id"));
      return jobView(s.row("select * from ug_job where id=?", j.get("id")));
    }
    if ("CLEANUP".equals(j.get("kind")))
      throw new ApiException(409, "USE_CLEANUP_RETRY", "请从清理记录重试");
    if ("EXPORT".equals(j.get("kind"))) {
      c.organizer(str(j.get("event_id")));
      throw new ApiException(409, "NEW_EXPORT_REQUIRED", "请重新生成并确认导出范围");
    }
    notifications.requireEmail();
    s.execute(
        "update ug_job set"
            + " status='PENDING',attempts=0,error_code=null,lease_until=null,version=version+1,completed_at=null,next_run_at=?"
            + " where id=?",
        s.now(),
        j.get("id"));
    return jobView(s.row("select * from ug_job where id=?", j.get("id")));
  }

  private Map<String, Object> newJob(
      String event,
      String kind,
      String key,
      String status,
      Map<String, Object> payload,
      String error,
      Object ref,
      OffsetDateTime expires) {
    String id = s.id();
    s.execute(
        "insert into"
            + " ug_job(id,event_id,kind,business_key,status,payload_cipher,attempts,next_run_at,result_ref,error_code,created_at,completed_at,expires_at)"
            + " values(?,?,?,?,?,?,?,?,?,?,?,?,?)",
        uuid(id),
        uuid(event),
        kind,
        key,
        status,
        payload.isEmpty() ? null : s.seal(payload),
        "PENDING".equals(status) ? 0 : 1,
        s.now(),
        ref,
        error,
        s.now(),
        Set.of("FAILED", "SUCCEEDED").contains(status) ? s.now() : null,
        expires);
    return s.row("select * from ug_job where id=?", uuid(id));
  }

  private Object email(Map<String, Object> r, Context c) {
    var e = event(r, c);
    var p = parties.raw(str(e.get("id")), str(r.get("partyId")));
    ApiException.require("APPROVED".equals(p.get("status")), 409, "PARTY_NOT_APPROVED", "此组当前不可领票");
    var fields = s.unseal(p.get("fields_cipher"));
    ApiException.require(fields.get("email") != null, 422, "EMAIL_REQUIRED", "此组未提供邮箱");
    notifications.requireEmail();
    var j =
        newJob(
            str(e.get("id")),
            "EMAIL",
            "resend:" + r.get("operationId"),
            "PENDING",
            map("partyId", p.get("id")),
            null,
            p.get("id"),
            time(e.get("personal_due_at")));
    return jobView(j);
  }

  private static final Set<String> EXPORT_COLUMNS =
      Set.of(
          "partyReference",
          "name",
          "phone",
          "email",
          "companionCount",
          "requestedSize",
          "approvedSize",
          "netAdmitted",
          "remainingCount",
          "status",
          "sourceType",
          "createdAt");

  private void exportAccess(Context c, String event, List<?> columns) {
    c.organizer(event);
    var grant =
        s.row(
            "select * from ug_event_grant where event_id=? and account_id=? and role='ORGANIZER'"
                + " and revoked_at is null and expires_at>now()",
            uuid(event),
            uuid(c.actorId()));
    ApiException.require(
        s.list(grant.get("allowed_actions")).contains("EXPORT"),
        403,
        "EXPORT_FORBIDDEN",
        "未获得导出权限");
    Set<String> fields = c.fields(event);
    for (Object col : columns) {
      String x = str(col);
      ApiException.require(EXPORT_COLUMNS.contains(x), 422, "INVALID_EXPORT_COLUMN", "导出列不在允许范围");
      if (Set.of("name", "phone", "email", "companionCount").contains(x))
        ApiException.require(fields.contains(x), 403, "FIELD_FORBIDDEN", "当前授权不包含所选字段");
    }
  }

  private Object export(Map<String, Object> r, Context c) {
    var e = event(r, c);
    List<?> columns = (List<?>) r.get("columns");
    exportAccess(c, str(e.get("id")), columns);
    String id = s.id();
    var expires = min(s.now().plusHours(24), time(e.get("personal_due_at")));
    var j =
        newJob(str(e.get("id")), "EXPORT", "export:" + id, "RUNNING", map(), null, null, expires);
    String key = id + ".artifact";
    try {
      List<Map<String, Object>> data = new ArrayList<>();
      var filter = s.parse(r.get("filter"));
      for (var p :
          s.rows("select * from ug_party where event_id=? order by created_at,id", e.get("id"))) {
        if (!matches(p, filter)) continue;
        var f = s.unseal(p.get("fields_cipher"));
        data.add(
            map(
                "partyReference",
                p.get("party_reference"),
                "name",
                f.get("name"),
                "phone",
                f.get("phone"),
                "email",
                f.get("email"),
                "companionCount",
                f.get("companionCount"),
                "requestedSize",
                p.get("requested_size"),
                "approvedSize",
                p.get("approved_size"),
                "netAdmitted",
                p.get("net_admitted"),
                "remainingCount",
                num(p.get("approved_size")) - num(p.get("net_admitted")),
                "status",
                p.get("status"),
                "sourceType",
                p.get("source_type"),
                "createdAt",
                p.get("created_at")));
      }
      byte[] bytes = renderExport(str(r.get("format")), columns, data);
      Files.createDirectories(storage);
      Files.write(
          storage.resolve(key),
          s.seal(map("base64", Base64.getEncoder().encodeToString(bytes))),
          StandardOpenOption.CREATE_NEW);
      s.execute(
          "insert into"
              + " ug_export(id,event_id,actor_id,job_id,format,columns_json,filter_json,purpose_cipher,artifact_key,artifact_digest,row_count,status,expires_at,created_at)"
              + " values(?,?,?,?,?,?::jsonb,?::jsonb,?,?,?,?,?,?,?)",
          uuid(id),
          e.get("id"),
          uuid(c.actorId()),
          j.get("id"),
          r.get("format"),
          s.stringify(columns),
          s.stringify(filter),
          s.seal(map("purpose", r.get("purpose"))),
          key,
          s.crypto.hash(Base64.getEncoder().encodeToString(bytes)),
          data.size(),
          "SUCCEEDED",
          expires,
          s.now());
      s.execute(
          "update ug_job set status='SUCCEEDED',result_ref=?,completed_at=? where id=?",
          uuid(id),
          s.now(),
          j.get("id"));
      return exportView(s.row("select * from ug_export where id=?", uuid(id)));
    } catch (ApiException ex) {
      throw ex;
    } catch (Exception ex) {
      throw new ApiException(503, "EXPORT_FAILED", "导出文件生成失败");
    }
  }

  private boolean matches(Map<String, Object> p, Map<String, Object> f) {
    for (String k : List.of("status", "sourceType"))
      if (f.containsKey(k) && !f.get(k).equals(p.get(k.equals("sourceType") ? "source_type" : k)))
        return false;
    if (f.containsKey("progress")) {
      int n = num(p.get("net_admitted")), a = num(p.get("approved_size"));
      String progress = n == 0 ? "NONE" : n < a ? "PARTIAL" : "COMPLETE";
      if (!progress.equals(f.get("progress"))) return false;
    }
    var at = time(p.get("created_at"));
    return (!f.containsKey("createdFrom") || !at.isBefore(time(f.get("createdFrom"))))
        && (!f.containsKey("createdTo") || !at.isAfter(time(f.get("createdTo"))));
  }

  static String safeCell(Object value) {
    String x = value == null ? "" : value.toString();
    return x.matches("(?s)^[\\s]*[=+\\-@].*") ? "'" + x : x;
  }

  private static byte[] renderExport(String format, List<?> columns, List<Map<String, Object>> rows)
      throws Exception {
    if ("CSV".equals(format)) {
      StringWriter writer = new StringWriter();
      writer.write('\uFEFF');
      try (var printer = new CSVPrinter(writer, CSVFormat.DEFAULT)) {
        printer.printRecord(columns);
        for (var r : rows)
          printer.printRecord(columns.stream().map(k -> safeCell(r.get(str(k)))).toList());
      }
      return writer.toString().getBytes(StandardCharsets.UTF_8);
    }
    try (var book = new XSSFWorkbook();
        var out = new ByteArrayOutputStream()) {
      var sheet = book.createSheet("UniGate");
      var header = sheet.createRow(0);
      for (int i = 0; i < columns.size(); i++)
        header.createCell(i, CellType.STRING).setCellValue(str(columns.get(i)));
      int n = 1;
      for (var r : rows) {
        var row = sheet.createRow(n++);
        for (int i = 0; i < columns.size(); i++)
          row.createCell(i, CellType.STRING).setCellValue(safeCell(r.get(str(columns.get(i)))));
      }
      book.write(out);
      return out.toByteArray();
    }
  }

  private Map<String, Object> exportRow(Map<String, Object> r, Context c) {
    var x =
        s.row(
            "select * from ug_export where id=? and event_id=?",
            uuid(r.get("exportId")),
            uuid(r.get("eventId")));
    ApiException.require(
        c.actorId().equals(str(x.get("actor_id"))), 403, "EXPORT_FORBIDDEN", "只能访问本人生成的导出");
    exportAccess(c, str(r.get("eventId")), s.list(x.get("columns_json")));
    return x;
  }

  private Map<String, Object> exportView(Map<String, Object> x) {
    return map(
        "exportId",
        x.get("id"),
        "eventId",
        x.get("event_id"),
        "status",
        s.value(
            "JobStatus", time(x.get("expires_at")).isAfter(s.now()) ? x.get("status") : "EXPIRED"),
        "format",
        s.value("ExportFormat", x.get("format")),
        "columns",
        s.list(x.get("columns_json")),
        "rowCount",
        x.get("row_count"),
        "expiresAt",
        x.get("expires_at"),
        "version",
        x.get("version"));
  }

  private Object download(Map<String, Object> r, Context c) {
    event(r, c);
    var x = exportRow(r, c);
    ApiException.require(
        time(x.get("expires_at")).isAfter(s.now()), 410, "EXPORT_EXPIRED", "导出文件已过期");
    ApiException.require("SUCCEEDED".equals(x.get("status")), 409, "EXPORT_NOT_READY", "导出文件尚未生成");
    try {
      Path path = storage.resolve(str(x.get("artifact_key"))).normalize();
      ApiException.require(path.startsWith(storage), 403, "FORBIDDEN", "无权访问文件");
      var payload = s.unseal(Files.readAllBytes(path));
      byte[] bytes = Base64.getDecoder().decode(str(payload.get("base64")));
      ApiException.require(
          MessageDigest.isEqual(
              (byte[]) x.get("artifact_digest"),
              s.crypto.hash(Base64.getEncoder().encodeToString(bytes))),
          410,
          "EXPORT_EXPIRED",
          "导出文件校验失败");
      s.audit(c, str(r.get("eventId")), "ExportDownload", x.get("id"), null, null);
      boolean csv = "CSV".equals(x.get("format"));
      return new Binary(
          bytes,
          csv
              ? "text/csv;charset=UTF-8"
              : "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
          "unigate." + (csv ? "csv" : "xlsx"));
    } catch (ApiException ex) {
      throw ex;
    } catch (Exception ex) {
      throw new ApiException(410, "EXPORT_EXPIRED", "导出文件不可用，请重新生成");
    }
  }

  private Object deliveries(Map<String, Object> r, Context c) {
    event(r, c);
    return s.page(
        s
            .rows(
                "select * from ug_delivery where event_id=? order by created_at desc,id desc",
                uuid(r.get("eventId")))
            .stream()
            .map(this::deliveryView)
            .toList(),
        r);
  }

  private Object delivery(Map<String, Object> r, Context c) {
    event(r, c);
    var x = exportRow(r, c);
    List<?> cols = (List<?>) r.get("columns");
    ApiException.require(
        !cols.isEmpty() && s.list(x.get("columns_json")).containsAll(cols),
        422,
        "DELIVERY_SCOPE_INVALID",
        "交付范围必须属于原导出列");
    ApiException.require(
        "SUCCEEDED".equals(x.get("status")) && time(x.get("expires_at")).isAfter(s.now()),
        410,
        "EXPORT_EXPIRED",
        "原导出不可用");
    String id = s.id();
    s.execute(
        "insert into"
            + " ug_delivery(id,event_id,export_id,recipient_cipher,purpose_cipher,columns_json,retention_note_cipher,actor_id,created_at)"
            + " values(?,?,?,?,?,?::jsonb,?,?,?)",
        uuid(id),
        uuid(r.get("eventId")),
        x.get("id"),
        s.seal(map("value", r.get("recipient"))),
        s.seal(map("value", r.get("purpose"))),
        s.stringify(cols),
        s.seal(map("value", r.get("retentionNote"))),
        uuid(c.actorId()),
        s.now());
    return deliveryView(s.row("select * from ug_delivery where id=?", uuid(id)));
  }

  private Map<String, Object> deliveryView(Map<String, Object> x) {
    return map(
        "deliveryId",
        x.get("id"),
        "eventId",
        x.get("event_id"),
        "exportId",
        x.get("export_id"),
        "recipient",
        s.unseal(x.get("recipient_cipher")).get("value"),
        "purpose",
        s.unseal(x.get("purpose_cipher")).get("value"),
        "retentionNote",
        s.unseal(x.get("retention_note_cipher")).get("value"),
        "columns",
        s.list(x.get("columns_json")),
        "createdAt",
        x.get("created_at"),
        "version",
        x.get("version"));
  }

  private Object cleanupDetail(Map<String, Object> r, Context c) {
    c.organizer(str(r.get("eventId")));
    var e = s.row("select * from ug_event where id=?", uuid(r.get("eventId")));
    var list = s.rows("select * from ug_cleanup where event_id=?", e.get("id"));
    Map<String, Object> x;
    if (list.isEmpty()) {
      String id = s.id();
      s.execute(
          "insert into ug_cleanup(id,event_id,due_at,status,scope_results)"
              + " values(?,?,?,'PENDING',?::jsonb)",
          uuid(id),
          e.get("id"),
          e.get("personal_due_at"),
          s.stringify(scopeResults("PENDING", null)));
      x = s.row("select * from ug_cleanup where id=?", uuid(id));
    } else x = list.getFirst();
    return cleanupView(x);
  }

  private List<Map<String, Object>> scopeResults(String status, String error) {
    return List.of(
            "PRIVATE_ACCESS", "LIVE_ROWS", "TEMP_FILES", "EXPORT_FILES", "TASK_PAYLOADS", "BACKUPS")
        .stream()
        .map(
            k ->
                map(
                    "scope",
                    s.value("CleanupScope", k),
                    "status",
                    s.value("JobStatus", status),
                    "processedCount",
                    0,
                    "failedCount",
                    "FAILED".equals(status) ? 1 : 0,
                    "errorCode",
                    error))
        .toList();
  }

  private Map<String, Object> cleanupView(Map<String, Object> x) {
    var scopes = s.list(x.get("scope_results"));
    return map(
        "cleanupId",
        x.get("id"),
        "eventId",
        x.get("event_id"),
        "dueAt",
        x.get("due_at"),
        "status",
        s.value("JobStatus", x.get("status")),
        "processedCount",
        x.get("processed_count"),
        "failedCount",
        x.get("failed_count"),
        "completedAt",
        x.get("completed_at"),
        "version",
        x.get("version"),
        "scopes",
        scopes.size() == 6 ? scopes : scopeResults("PENDING", null));
  }

  private Object cleanupRetry(Map<String, Object> r, Context c) {
    c.organizer(str(r.get("eventId")));
    s.row("select id from ug_event where id=? for update", uuid(r.get("eventId")));
    var x =
        s.row(
            "select * from ug_cleanup where id=? and event_id=? for update",
            uuid(r.get("cleanupId")),
            uuid(r.get("eventId")));
    s.version(x, r.get("version"));
    ApiException.require(
        "FAILED".equals(x.get("status")) && !time(x.get("due_at")).isAfter(s.now()),
        409,
        "CLEANUP_NOT_RETRYABLE",
        "仅能重试已到期的失败清理");
    return runCleanup(x);
  }

  /** Called by a deployment scheduler in its transaction. Backup completion is never inferred. */
  public Map<String, Object> runCleanup(Map<String, Object> x) {
    Object event = x.get("event_id");
    ApiException.require(
        !time(x.get("due_at")).isAfter(s.now()), 409, "CLEANUP_NOT_DUE", "尚未到清理时间");
    var e = s.row("select * from ug_event where id=? for update", event);
    List<Map<String, Object>> scopes = new ArrayList<>(scopeResults("SUCCEEDED", null));
    // Save anonymous aggregates before deleting linkable facts. Repeated cleanup preserves the
    // original snapshot.
    if (e.get("anonymous_totals") == null) {
      var totals =
          s.row(
              "select count(*) application_groups,count(*) filter(where status='PENDING')"
                  + " pending_groups,count(*) filter(where status='APPROVED')"
                  + " approved_groups,coalesce(sum(net_admitted),0) net_admissions,count(*)"
                  + " filter(where source_type='WALK_IN') walk_in_groups from ug_party where"
                  + " event_id=?",
              event);
      var admission =
          s.row(
              "select coalesce(sum(admitted_count),0)"
                  + " gross_admissions,coalesce(sum(corrected_count),0) corrected_people from"
                  + " ug_admission where event_id=?",
              event);
      totals.putAll(admission);
      totals.put("allocated_people", e.get("allocated_count"));
      s.execute(
          "update ug_event set anonymous_totals=?::jsonb,status='ARCHIVED',version=version+1 where"
              + " id=?",
          s.stringify(totals),
          event);
    }
    s.execute("update ug_event set status='ARCHIVED' where id=?", event);
    boolean filesDeleted = true;
    for (var export : s.rows("select id,artifact_key from ug_export where event_id=?", event)) {
      if (export.get("artifact_key") == null) continue;
      try {
        Path path = storage.resolve(str(export.get("artifact_key"))).normalize();
        if (!path.startsWith(storage)) throw new IOException("invalid managed artifact");
        Files.deleteIfExists(path);
        s.execute(
            "update ug_export set artifact_key=null,artifact_digest=null where id=?",
            export.get("id"));
      } catch (IOException ex) {
        filesDeleted = false;
        scopes.set(
            3,
            map(
                "scope",
                s.value("CleanupScope", "EXPORT_FILES"),
                "status",
                s.value("JobStatus", "FAILED"),
                "processedCount",
                0,
                "failedCount",
                1,
                "errorCode",
                "FILE_DELETE_FAILED"));
      }
    }
    int processed = 0;
    // Reverse foreign-key order. Keep only event aggregates, cleanup metadata and the independent
    // deletion ledger.
    for (String table :
        List.of(
            "ug_delivery",
            "ug_credential",
            "ug_session",
            "ug_challenge",
            "ug_source_link",
            "ug_import_row",
            "ug_import_batch",
            "ug_admission_correction",
            "ug_admission",
            "ug_party_review",
            "ug_party",
            "ug_audit",
            "ug_outbox",
            "ug_device",
            "ug_source"))
      processed += s.execute("delete from " + table + " where event_id=?", event);
    s.execute("update ug_event_grant set disclosure_cipher=null where event_id=?", event);
    if (filesDeleted) {
      s.execute("delete from ug_export where event_id=?", event);
      s.execute("delete from ug_job where event_id=?", event);
    } else {
      s.execute(
          "update ug_export set purpose_cipher=?,status='EXPIRED' where event_id=?",
          s.crypto.seal("{}"),
          event);
      s.execute(
          "update ug_job set payload_cipher=null,status='EXPIRED',lease_until=null where"
              + " event_id=?",
          event);
    }
    // The currently executing CleanupRetry operation is retained until the dispatcher stores its
    // result.
    s.execute("delete from ug_operation where event_id=? and action<>'CleanupRetry'", event);
    s.execute(
        "update ug_operation set result_cipher=null,private_result_expires_at=null,resource_id=null"
            + " where event_id=?",
        event);
    s.revokeEventKey(str(event));
    scopes.set(
        5,
        map(
            "scope",
            s.value("CleanupScope", "BACKUPS"),
            "status",
            s.value("JobStatus", "FAILED"),
            "processedCount",
            0,
            "failedCount",
            1,
            "errorCode",
            "BACKUP_COVERAGE_UNVERIFIED"));
    int failed =
        (int)
            scopes.stream()
                .filter(y -> "FAILED".equals(((Map<?, ?>) y.get("status")).get("code")))
                .count();
    s.execute(
        "update ug_cleanup set"
            + " status='FAILED',scope_results=?::jsonb,processed_count=?,failed_count=?,version=version+1"
            + " where id=?",
        s.stringify(scopes),
        processed,
        failed,
        x.get("id"));
    s.execute(
        "insert into"
            + " ug_deletion_ledger(id,event_id,due_at,key_ref,key_revoked_at,live_cleanup_at,status,updated_at)"
            + " values(?,?,?,?,?,?,'BACKUP_PENDING',?) on conflict(event_id) do update set"
            + " key_revoked_at=excluded.key_revoked_at,live_cleanup_at=excluded.live_cleanup_at,status='BACKUP_PENDING',updated_at=excluded.updated_at",
        uuid(s.id()),
        event,
        x.get("due_at"),
        "event:" + event,
        s.now(),
        s.now(),
        s.now());
    return jobView(
        newJob(
            str(event),
            "CLEANUP",
            "cleanup:" + x.get("id") + ":" + x.get("version"),
            "FAILED",
            map(),
            "BACKUP_COVERAGE_UNVERIFIED",
            x.get("id"),
            null));
  }

  private static OffsetDateTime min(OffsetDateTime a, OffsetDateTime b) {
    return a.isBefore(b) ? a : b;
  }
}
