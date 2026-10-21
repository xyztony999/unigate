# UniGate 数据与事务设计

版本v0.3；2026-10-21；ADR-042采用API合同，ADR-043采用项目检查，D3按项目依据完成；不声明原版检查全通过。以下为未执行的设计DDL/SQL，不是数据库迁移或业务实现；PostgreSQL18为技术候选，无数据库验证结果。

## 1. 聚合与持久映射

Event→ug_event/policy；RegistrationParty→ug_party及review/credential/admission/correction/source_link；ImportBatch→batch/row；SourceConnection→connection/connection_event/source；DeviceSession→device，身份账号/授权/session/challenge；ExportJob→job/export/delivery；CleanupRun→cleanup/deletion_ledger；operation/audit/outbox为跨过程基础记录。统计只读事实，不独立写人员余额。

policy JSON严格映射EventPolicyInput（version独立列）；告知文本按收集字段、用途、主办方配置和留存模板生成，noticeVersion为内容+policyVersion摘要。不得给客户端任意JSON透传。源映射/原始行/任务载荷JSON也按API DTO白名单验证。API导出的code+label仅展示，DB保存code。派生progress由netAdmitted与approvedSize算，remainingCount由差值算；Review.approvedSize由对应组读取，Batch行数按行状态统计，不重复存不一致计数。

匿名连续性标记clientContextId映射ug_session.id，与HttpOnly cookie随机token及其摘要分开，公开ID本身不能认证。有效bootstrap复用该行，新身份建新行；ANON写及RegistrationResult请求头标记须与当前cookie对应行一致。核销在STAFF/kiosk统一actor_scope=GATE，活动+GateCommit+operationId唯一；稳定deviceId/partyId/原版本/人数进入payload_digest，避免换组/换端落到新作用域再扣数。实际账号另记审计/授权，换账号不重写原事实。MyGrantList只取当前account_id授权，GrantEventOptions仅活动标识/标题/时间用于管理选择，不泄漏名单或到场数。

PartyView.hasAdmissionHistory取ug_party.ever_admitted，DeviceSessionView.eventStatus关联ug_event.status，PartyPage.asOf取本次一致读快照时间，GrantView工作人员名称从ug_account获权投影；均无新增持久化冗余字段。BatchView.availableColumns取受管暂存头/来源字段元数据，ResourceFields只读获权资源列，不保存答卷样本；最多200列。CleanupView.scopes映射ug_cleanup.scope_results的六个受管范围，RegistrationResult只读原ug_operation.result_cipher及私密响应有效期，不重新占额/发通知。

SourceListView.availableConnections只由当前活动ug_connection_event关联ug_connection读取connectionId/displayName/provider/accountKind/status，过滤名单支持的FEISHU/TENCENT，不返回credentials_cipher或其他活动关联。首次配置来源也能选连接，无需开放管理员全局连接表或手输密钥。

fields_cipher为活动数据密钥封装加密后的PartyInput；手机号/邮箱规范化HMAC为非唯一精确查找键，姓名亦按规范化完整值HMAC。页面模糊搜索仅对单活动受权候选集解密过滤，500人目标足够；不建立明文PII全文索引。工作人员电话尾号搜索同样活动内内存过滤最多10个脱敏候选，不把完整字段回传。query日志禁敏感词。每活动密钥独立，主密钥独立于数据库/备份存放，候选外部密钥服务或部署秘密挂载，版本化轮换；备份不含可独立恢复的所有解密材料。

## 2. 数据字典与设计DDL

所有id由应用生成UUID，时间timestamptz，计数int，版本bigint（API限制非负int32时达到上限前停止写并升级合同，不截断）；初始版本0。未列NOT NULL的列允许SQL NULL，对应API通常省略。秘密/理由/收件人加密字段采用受管密文包，包括nonce/algorithm/keyVersion，禁止靠列名cipher却明文保存。所有状态code必须应用按枚举池校验；DDL中已列CHECK为核心不变量，剩余枚举由实现迁移补齐而非自由文本业务值。

该DDL为字段/约束完整草案，未通过SQL引擎执行。引用关系默认RESTRICT，由清理服务按依赖逆序删除。聚合内复合FK防跨活动关联；部分设备/操作跨活动一致性还需在核心事务守卫，不声称外键可覆盖全部业务不变量。

### 2.1 ug_account

工作人员身份；慢哈希口令，不包含参与者账户。

```sql
CREATE TABLE ug_account (
id uuid PRIMARY KEY, username varchar(80) NOT NULL UNIQUE, display_name varchar(100) NOT NULL,
password_hash text NOT NULL, roles jsonb NOT NULL, enabled boolean NOT NULL DEFAULT true,
must_change_password boolean NOT NULL DEFAULT true, auth_version bigint NOT NULL DEFAULT 0,
version bigint NOT NULL DEFAULT 0, created_at timestamptz NOT NULL DEFAULT now()
);
```

### 2.2 ug_event

活动聚合与容量唯一计数；policy含字段/告知/版本快照。

```sql
CREATE TABLE ug_event (
id uuid PRIMARY KEY, title varchar(200) NOT NULL, venue varchar(300),
starts_at timestamptz NOT NULL, ends_at timestamptz NOT NULL, time_zone varchar(80) NOT NULL,
registration_starts_at timestamptz NOT NULL, registration_ends_at timestamptz NOT NULL,
capacity integer NOT NULL CHECK(capacity BETWEEN 1 AND 1000000),
allocated_count integer NOT NULL DEFAULT 0 CHECK(allocated_count >= 0 AND allocated_count <= capacity),
status varchar(30) NOT NULL CHECK(status IN ('DRAFT','READY','CHECKIN_OPEN','CHECKIN_PAUSED','ENDED','ARCHIVED')),
policy jsonb NOT NULL, policy_version bigint NOT NULL DEFAULT 0,
retention_days integer NOT NULL DEFAULT 30 CHECK(retention_days BETWEEN 1 AND 3650),
personal_due_at timestamptz NOT NULL, version bigint NOT NULL DEFAULT 0,
anonymous_totals jsonb, created_by uuid NOT NULL REFERENCES ug_account(id), created_at timestamptz NOT NULL DEFAULT now(),
CHECK(registration_starts_at <= registration_ends_at AND registration_ends_at <= starts_at AND starts_at <= ends_at)
);
```

### 2.3 ug_event_grant

每活动角色/字段/动作授权，撤销留版本。

```sql
CREATE TABLE ug_event_grant (
id uuid PRIMARY KEY, event_id uuid NOT NULL REFERENCES ug_event(id), account_id uuid NOT NULL REFERENCES ug_account(id),
role varchar(20) NOT NULL CHECK(role IN ('ORGANIZER','CHECKER','READER')), allowed_fields jsonb NOT NULL,
allowed_actions jsonb NOT NULL, disclosure_cipher bytea, expires_at timestamptz NOT NULL, revoked_at timestamptz,
version bigint NOT NULL DEFAULT 0, UNIQUE(event_id,account_id,role)
);
```

### 2.4 ug_party

报名组；fields_cipher存四基础字段，HMAC检索非唯一。

```sql
CREATE TABLE ug_party (
id uuid PRIMARY KEY, event_id uuid NOT NULL REFERENCES ug_event(id), party_reference varchar(50) NOT NULL,
fields_cipher bytea NOT NULL, data_key_ref varchar(200) NOT NULL, name_search_hmac bytea, phone_search_hmac bytea, email_search_hmac bytea,
requested_size integer NOT NULL CHECK(requested_size >= 1), approved_size integer NOT NULL DEFAULT 0 CHECK(approved_size >= 0),
pending_increase integer NOT NULL DEFAULT 0 CHECK(pending_increase >= 0), net_admitted integer NOT NULL DEFAULT 0,
ever_admitted boolean NOT NULL DEFAULT false, status varchar(20) NOT NULL CHECK(status IN ('PENDING','APPROVED','REJECTED','CANCELLED')),
source_type varchar(20) NOT NULL, accepted_notice_version varchar(80) NOT NULL,
version bigint NOT NULL DEFAULT 0, created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
UNIQUE(event_id,id), UNIQUE(event_id,party_reference), CHECK(net_admitted BETWEEN 0 AND approved_size),
CHECK(status='APPROVED' OR (approved_size=0 AND net_admitted=0))
);
```

### 2.5 ug_party_review

初次和差额审核，不取代原批准数。

```sql
CREATE TABLE ug_party_review (
id uuid PRIMARY KEY, event_id uuid NOT NULL, party_id uuid NOT NULL,
kind varchar(20) NOT NULL CHECK(kind IN ('INITIAL','INCREASE')), requested_size integer NOT NULL CHECK(requested_size>=1),
requested_increase integer NOT NULL CHECK(requested_increase>=0), result varchar(20) NOT NULL,
reason_cipher bytea, actor_id uuid REFERENCES ug_account(id), version bigint NOT NULL DEFAULT 0,
created_at timestamptz NOT NULL DEFAULT now(), decided_at timestamptz,
FOREIGN KEY(event_id,party_id) REFERENCES ug_party(event_id,id)
);
```

### 2.6 ug_credential

票/查询/变更能力，类型隔离；可再展示的值加密。

```sql
CREATE TABLE ug_credential (
id uuid PRIMARY KEY, event_id uuid NOT NULL, party_id uuid NOT NULL,
kind varchar(16) NOT NULL CHECK(kind IN ('TICKET','QUERY','CHANGE')), token_digest bytea NOT NULL UNIQUE,
token_cipher bytea, credential_version bigint NOT NULL, status varchar(16) NOT NULL,
expires_at timestamptz NOT NULL, created_at timestamptz NOT NULL DEFAULT now(),
FOREIGN KEY(event_id,party_id) REFERENCES ug_party(event_id,id)
);
```

### 2.7 ug_device

授权浏览器端；领取码与设备会话均摘要。

```sql
CREATE TABLE ug_device (
id uuid PRIMARY KEY, event_id uuid NOT NULL REFERENCES ug_event(id), name varchar(100) NOT NULL,
mode varchar(16) NOT NULL CHECK(mode IN ('STAFF','KIOSK')), status varchar(16) NOT NULL,
session_digest bytea UNIQUE, pairing_digest bytea UNIQUE, pairing_expires_at timestamptz,
pairing_failures integer NOT NULL DEFAULT 0, expires_at timestamptz NOT NULL,
last_seen_at timestamptz, version bigint NOT NULL DEFAULT 0, created_by uuid NOT NULL REFERENCES ug_account(id)
);
```

### 2.8 ug_session

身份/匿名/本组受限会话；浏览器只有随机cookie。

```sql
CREATE TABLE ug_session (
id uuid PRIMARY KEY, token_digest bytea NOT NULL UNIQUE,
kind varchar(20) NOT NULL CHECK(kind IN ('STAFF','ANONYMOUS','PARTY_QUERY','PARTY_CHANGE')),
account_id uuid REFERENCES ug_account(id), event_id uuid REFERENCES ug_event(id), party_id uuid,
capability_version bigint, auth_version bigint, csrf_digest bytea NOT NULL,
created_at timestamptz NOT NULL, last_seen_at timestamptz NOT NULL, expires_at timestamptz NOT NULL, revoked_at timestamptz,
FOREIGN KEY(event_id,party_id) REFERENCES ug_party(event_id,id),
CHECK((kind='STAFF' AND account_id IS NOT NULL) OR (kind='ANONYMOUS' AND account_id IS NULL AND party_id IS NULL) OR (kind IN ('PARTY_QUERY','PARTY_CHANGE') AND event_id IS NOT NULL AND party_id IS NOT NULL))
);
```

### 2.9 ug_connection

平台/通知服务连接，所有凭据为加密包。

```sql
CREATE TABLE ug_connection (
id uuid PRIMARY KEY, provider varchar(20) NOT NULL, account_kind varchar(20) NOT NULL,
display_name varchar(100) NOT NULL, status varchar(20) NOT NULL, client_id varchar(200),
credentials_cipher bytea, key_ref varchar(200), expires_at timestamptz,
version bigint NOT NULL DEFAULT 0, created_by uuid NOT NULL REFERENCES ug_account(id)
);
```

### 2.10 ug_connection_event

管理员显式授予活动可用连接，不扩展文档本身提供方权限。

```sql
CREATE TABLE ug_connection_event (
connection_id uuid NOT NULL REFERENCES ug_connection(id), event_id uuid NOT NULL REFERENCES ug_event(id),
PRIMARY KEY(connection_id,event_id)
);
```

### 2.11 ug_challenge

短信/恢复/OAuth短期挑战；不存明文验证码。

```sql
CREATE TABLE ug_challenge (
id uuid PRIMARY KEY, kind varchar(20) NOT NULL, event_id uuid REFERENCES ug_event(id), party_id uuid,
connection_id uuid REFERENCES ug_connection(id), actor_scope varchar(200) NOT NULL,
target_cipher bytea, code_digest bytea NOT NULL, context_cipher bytea,
attempts integer NOT NULL DEFAULT 0 CHECK(attempts BETWEEN 0 AND 5), expires_at timestamptz NOT NULL, consumed_at timestamptz,
FOREIGN KEY(event_id,party_id) REFERENCES ug_party(event_id,id)
);
```

### 2.12 ug_operation

业务幂等键/响应；scope由服务端算，GLOBAL不为空。

```sql
CREATE TABLE ug_operation (
id uuid PRIMARY KEY, actor_scope varchar(200) NOT NULL, event_scope varchar(50) NOT NULL,
event_id uuid REFERENCES ug_event(id), action varchar(100) NOT NULL, operation_id uuid NOT NULL,
payload_digest bytea NOT NULL, status varchar(20) NOT NULL, result_code varchar(80), resource_id uuid,
result_cipher bytea, private_result_expires_at timestamptz, created_at timestamptz NOT NULL, committed_at timestamptz,
UNIQUE(actor_scope,event_scope,action,operation_id), UNIQUE(id,event_id)
);
```

### 2.13 ug_admission

成功核销追加事实，不原地删除/改人数。

```sql
CREATE TABLE ug_admission (
id uuid PRIMARY KEY, event_id uuid NOT NULL, party_id uuid NOT NULL,
operation_entry_id uuid NOT NULL UNIQUE, device_id uuid NOT NULL REFERENCES ug_device(id),
actor_id uuid REFERENCES ug_account(id), admitted_count integer NOT NULL CHECK(admitted_count>0),
corrected_count integer NOT NULL DEFAULT 0 CHECK(corrected_count BETWEEN 0 AND admitted_count),
committed_at timestamptz NOT NULL, FOREIGN KEY(event_id,party_id) REFERENCES ug_party(event_id,id),
FOREIGN KEY(operation_entry_id,event_id) REFERENCES ug_operation(id,event_id), UNIQUE(event_id,party_id,id)
);
```

### 2.14 ug_admission_correction

关联原记录的追加更正。

```sql
CREATE TABLE ug_admission_correction (
id uuid PRIMARY KEY, event_id uuid NOT NULL, party_id uuid NOT NULL, original_record_id uuid NOT NULL,
corrected_count integer NOT NULL CHECK(corrected_count>0), reason_cipher bytea NOT NULL,
actor_id uuid NOT NULL REFERENCES ug_account(id), operation_entry_id uuid NOT NULL UNIQUE REFERENCES ug_operation(id),
created_at timestamptz NOT NULL, FOREIGN KEY(event_id,party_id,original_record_id) REFERENCES ug_admission(event_id,party_id,id)
);
```

### 2.15 ug_source

活动来源资源、字段映射和资格解释。

```sql
CREATE TABLE ug_source (
id uuid PRIMARY KEY, event_id uuid NOT NULL REFERENCES ug_event(id), connection_id uuid NOT NULL REFERENCES ug_connection(id),
resource_id varchar(300) NOT NULL, mapping jsonb NOT NULL, eligibility varchar(30) NOT NULL,
version bigint NOT NULL DEFAULT 0, UNIQUE(event_id,id), FOREIGN KEY(connection_id,event_id) REFERENCES ug_connection_event(connection_id,event_id)
);
```

### 2.16 ug_import_batch

文件/API预览批次，持久化规则/来源指纹。

```sql
CREATE TABLE ug_import_batch (
id uuid PRIMARY KEY, event_id uuid NOT NULL REFERENCES ug_event(id), source_id uuid,
file_digest varchar(64), sheet_ref varchar(200), source_label varchar(100), file_key varchar(300),
mapping jsonb, eligibility varchar(30), status varchar(20) NOT NULL, preview_version bigint NOT NULL DEFAULT 0,
policy_version bigint, source_version bigint, preview_digest bytea, version bigint NOT NULL DEFAULT 0,
expires_at timestamptz NOT NULL, committed_operation_id uuid REFERENCES ug_operation(id),
created_at timestamptz NOT NULL DEFAULT now(), UNIQUE(event_id,id), FOREIGN KEY(event_id,source_id) REFERENCES ug_source(event_id,id)
);
```

### 2.17 ug_import_row

暂存原始/规范化行和错误，敏感值密文。

```sql
CREATE TABLE ug_import_row (
id uuid PRIMARY KEY, event_id uuid NOT NULL, batch_id uuid NOT NULL, source_record_id varchar(300) NOT NULL,
line_number integer NOT NULL CHECK(line_number>0), row_cipher bytea NOT NULL, content_digest bytea NOT NULL,
status varchar(30) NOT NULL, errors jsonb NOT NULL DEFAULT '[]', requested_size integer,
linked_party_id uuid, resolution varchar(30), reason_cipher bytea,
FOREIGN KEY(event_id,batch_id) REFERENCES ug_import_batch(event_id,id),
FOREIGN KEY(event_id,linked_party_id) REFERENCES ug_party(event_id,id), UNIQUE(batch_id,line_number), UNIQUE(batch_id,source_record_id)
);
```

### 2.18 ug_source_link

同源唯一关联；source_scope和record_key为规范化摘要。

```sql
CREATE TABLE ug_source_link (
id uuid PRIMARY KEY, event_id uuid NOT NULL, party_id uuid NOT NULL,
source_scope varchar(200) NOT NULL, record_key varchar(300) NOT NULL, source_content_digest bytea NOT NULL,
connection_id uuid REFERENCES ug_connection(id), resource_id varchar(300), source_record_id varchar(300),
batch_id uuid, linked_at timestamptz NOT NULL, FOREIGN KEY(event_id,party_id) REFERENCES ug_party(event_id,id),
UNIQUE(source_scope,record_key)
);
```

### 2.19 ug_job

持久任务/租约/副作用业务键，输入秘密密文引用。

```sql
CREATE TABLE ug_job (
id uuid PRIMARY KEY, event_id uuid REFERENCES ug_event(id), kind varchar(20) NOT NULL,
business_key varchar(250) NOT NULL, status varchar(20) NOT NULL, payload_cipher bytea,
attempts integer NOT NULL DEFAULT 0 CHECK(attempts>=0), lease_until timestamptz, next_run_at timestamptz NOT NULL,
result_ref uuid, error_code varchar(80), version bigint NOT NULL DEFAULT 0,
created_at timestamptz NOT NULL, completed_at timestamptz, expires_at timestamptz, UNIQUE(kind,business_key)
);
```

### 2.20 ug_outbox

事务后最小事件，不含票码/完整联系方式。

```sql
CREATE TABLE ug_outbox (
id uuid PRIMARY KEY, event_id uuid REFERENCES ug_event(id), event_type varchar(80) NOT NULL,
resource_id uuid, operation_id uuid, schema_version integer NOT NULL DEFAULT 1, payload jsonb NOT NULL,
created_at timestamptz NOT NULL, published_at timestamptz
);
```

### 2.21 ug_export

受控文件与原创建人/列范围，下载再次鉴权。

```sql
CREATE TABLE ug_export (
id uuid PRIMARY KEY, event_id uuid NOT NULL REFERENCES ug_event(id), actor_id uuid NOT NULL REFERENCES ug_account(id),
job_id uuid NOT NULL UNIQUE REFERENCES ug_job(id), format varchar(10) NOT NULL CHECK(format IN ('CSV','XLSX')),
columns_json jsonb NOT NULL, filter_json jsonb NOT NULL, purpose_cipher bytea NOT NULL,
artifact_key varchar(300), artifact_digest bytea, row_count integer, status varchar(20) NOT NULL,
expires_at timestamptz NOT NULL, version bigint NOT NULL DEFAULT 0, created_at timestamptz NOT NULL
);
```

### 2.22 ug_delivery

对外副本登记；不自动调用发送服务。

```sql
CREATE TABLE ug_delivery (
id uuid PRIMARY KEY, event_id uuid NOT NULL REFERENCES ug_event(id), export_id uuid NOT NULL REFERENCES ug_export(id),
recipient_cipher bytea NOT NULL, purpose_cipher bytea NOT NULL, columns_json jsonb NOT NULL,
retention_note_cipher bytea NOT NULL, actor_id uuid NOT NULL REFERENCES ug_account(id), version bigint NOT NULL DEFAULT 0,
created_at timestamptz NOT NULL
);
```

### 2.23 ug_audit

最小审计；理由可含PII，密文按事件到期处理。

```sql
CREATE TABLE ug_audit (
id uuid PRIMARY KEY, event_id uuid REFERENCES ug_event(id), action varchar(100) NOT NULL,
actor_id uuid REFERENCES ug_account(id), device_id uuid, resource_id uuid, operation_id uuid,
reason_cipher bytea, occurred_at timestamptz NOT NULL
);
```

### 2.24 ug_cleanup

分范围清理状态，不能只看逻辑删除。

```sql
CREATE TABLE ug_cleanup (
id uuid PRIMARY KEY, event_id uuid NOT NULL UNIQUE REFERENCES ug_event(id), due_at timestamptz NOT NULL,
status varchar(20) NOT NULL, scope_results jsonb NOT NULL, processed_count integer NOT NULL DEFAULT 0,
failed_count integer NOT NULL DEFAULT 0, completed_at timestamptz, version bigint NOT NULL DEFAULT 0
);
```

### 2.25 ug_deletion_ledger

独立可恢复删除依据；备份内旧副本不能覆盖新账本。

```sql
CREATE TABLE ug_deletion_ledger (
id uuid PRIMARY KEY, event_id uuid NOT NULL UNIQUE, due_at timestamptz NOT NULL,
key_ref varchar(200) NOT NULL, key_revoked_at timestamptz, live_cleanup_at timestamptz,
last_covered_backup_expires_at timestamptz, status varchar(30) NOT NULL, updated_at timestamptz NOT NULL
);
```

## 3. 索引、约束与仓储

```sql
CREATE INDEX ix_event_scope_sort ON ug_event(starts_at DESC,id DESC);
CREATE INDEX ix_grant_actor_event ON ug_event_grant(account_id,event_id,expires_at);
CREATE INDEX ix_party_list ON ug_party(event_id,created_at DESC,id DESC);
CREATE INDEX ix_party_status ON ug_party(event_id,status,created_at DESC,id DESC);
CREATE INDEX ix_party_phone ON ug_party(event_id,phone_search_hmac);
CREATE INDEX ix_party_email ON ug_party(event_id,email_search_hmac);
CREATE INDEX ix_party_name ON ug_party(event_id,name_search_hmac);
CREATE UNIQUE INDEX ux_pending_review ON ug_party_review(party_id,kind) WHERE result='PENDING';
CREATE UNIQUE INDEX ux_active_capability ON ug_credential(party_id,kind) WHERE status='ACTIVE';
CREATE INDEX ix_review_list ON ug_party_review(event_id,created_at DESC,id DESC);
CREATE INDEX ix_admission_list ON ug_admission(event_id,committed_at DESC,id DESC);
CREATE INDEX ix_correction_original ON ug_admission_correction(original_record_id);
CREATE INDEX ix_batch_rows ON ug_import_row(batch_id,line_number,id);
CREATE INDEX ix_job_claim ON ug_job(next_run_at,id) WHERE status IN ('PENDING','RUNNING');
CREATE INDEX ix_outbox_claim ON ug_outbox(created_at,id) WHERE published_at IS NULL;
CREATE INDEX ix_audit_list ON ug_audit(event_id,occurred_at DESC,id DESC);
CREATE INDEX ix_delivery_list ON ug_delivery(event_id,created_at DESC,id DESC);
CREATE INDEX ix_cleanup_due ON ug_cleanup(due_at,id) WHERE status <> 'SUCCEEDED';
```

同一活动内同源记录只关联一次，不能悄悄挪组；来源scope固定含eventId，具体字符串为HMAC(eventId+connectionId+resourceId)或HMAC(eventId+fileDigest+sheetRef)，record_key取源recordId/行号。唯一键因此只防同活动同源重复，不阻止合法跨活动报名。摘要碰撞/来源版本变化视冲突人工处理，不以手机号唯一约束替代。

Repository：Event/Party/Decision/Credential/Admission/Import/Device/Identity/Connection/Job/Export/Cleanup/Audit各管对应表。跨聚合事务由应用服务统一，同连接不允许repository内部REQUIRES_NEW抢先提交。数据库应用角色不得任意执行DDL；读者、kiosk只访问HTTP权限服务，无数据库账户。

## 4. 写事务SQL清单与I/O

### 4.1 核销T-GATE

参数:eventId、稳定partyId（由有效scanContext解析）、operationId、canonicalDigest、expectedPartyVersion、count、actor/device。先校验类型/当前会话，认领operation唯一键（冲突重验范围后走原结果）。以下为同一事务内的关键SQL模板，不省略应用资格/票/设备校验：

```sql
BEGIN;
INSERT INTO ug_operation(id,actor_scope,event_scope,event_id,action,operation_id,payload_digest,status,created_at)
VALUES (:entryId,:actorScope,:eventScope,:eventId,'GateCommit',:operationId,:digest,'IN_PROGRESS',now())
ON CONFLICT(actor_scope,event_scope,action,operation_id) DO NOTHING;
SELECT * FROM ug_event WHERE id=:eventId FOR UPDATE;
SELECT * FROM ug_party WHERE event_id=:eventId AND id=:partyId FOR UPDATE;
-- 按统一顺序锁所需grant/device/credential，重新验证OPEN/资格/有效票及授权。
UPDATE ug_party SET net_admitted=net_admitted+:count,ever_admitted=true,version=version+1,updated_at=now()
WHERE id=:partyId AND event_id=:eventId AND version=:expectedVersion
  AND status='APPROVED' AND :count>0 AND net_admitted+:count<=approved_size;
-- 行数必须为1；否则整事务回滚，不能接着INSERT成功事实。
INSERT INTO ug_admission(id,event_id,party_id,operation_entry_id,device_id,actor_id,admitted_count,committed_at)
VALUES (:recordId,:eventId,:partyId,:entryId,:deviceId,:actorId,:count,now());
-- 同事务追加audit/outbox，并把ug_operation改SUCCEEDED和密文GateResult。
COMMIT;
```

ON CONFLICT为0行时必须分支读取/等待原条目，不能继续当新动作；已知失败可另短事务保存REJECTED但不得覆盖成功。数据库回滚/死锁后查原操作再同ID重试。事件/组/权限行全部重新验证，不能仅靠上述UPDATE判断全部权限已通过。

### 4.2 容量T-ALLOCATE/T-REVIEW/T-INCREASE/T-CANCEL

Event FOR UPDATE→排序Party FOR UPDATE→Decision/credentials/grant。批准/增额在同事务执行 `UPDATE ug_event SET allocated_count=allocated_count+:delta,version=version+1 WHERE id=:eventId AND allocated_count+:delta BETWEEN 0 AND capacity`，必须1行；更新对应组approved_size、决定、票及operation，再提交。初次整组delta=requestedSize；审核增员delta=requestedIncrease；减少/取消delta负值但需无历史核销和允许窗口，取消撤票并保留决定。核销/纠错不调用容量释放。聚合总计约束用此唯一服务维护，SQL CHECK不能自行验证跨表SUM；周期只读对账发现差异报警/停相关写，禁止后台自动覆盖事实计数。

### 4.3 导入T-IMPORT

解析已在事务外。认领operation→Event锁→batch与source策略版本校验→按partyId锁所有关联组→逐行核对预览版本和source_link唯一键→一次算总delta检查容量→插入/关联全部选中行→更新batch COMMITTED及计数/operation/audit/outbox。任一行错误/来源键竞争/超时即全回滚；预览时排除错误，提交不跳过失败行。分块insert可以，分块commit禁止。成功后同批再改选中集合须400，不生成新有效票。10000行是输入保护上限，不是保证5秒可完成，超时拒绝后可由负责人分成可审阅的小批，仍每批原子。

### 4.4 纠错T-CORRECT与凭证T-REISSUE

纠错Event→Party→原admission锁，校验累计更正<=原记录人数、net-count>=0，更新admission.corrected_count和party.net_admitted/version，追加correction+audit+operation。ever_admitted仍true，不能纠成0后自助退报名。补发Event→Party→有效credential锁，先旧票REVOKED再插新票（受唯一部分索引约束）；保留批准额/净核销。私密能力仅显式rotate时一并失效并新建。

### 4.5 阶段、授权与任务

T-PHASE锁Event后更新stage/version，与所有名额/核销排序。T-GRANT锁Event后grant/受权账号（多账号按ID排序），再防最后负责人/管理员越权。全局最后管理员变更需专用全局事务advisory-lock固定键，不与活动锁反向混用。T-JOB以SELECT FOR UPDATE SKIP LOCKED认领next_run_at到期任务、更新租约并短提交，再外部I/O；结果事务按version/lease确认，过期worker不能覆盖新attempt。任务business_key和外部可用幂等键双层去重，SMTP未知不能保证邮件只收到一封。

## 5. 读模型、权限与清理SQL清单

Q-PARTY按event/授权字段/筛选+created_at,id降序分页，count与列表同事务读快照；敏感字段解密仅获权范围。Q-GATE最多10条脱敏、STAFF模式；不返回ticket/private token。Q-STATS在REPEATABLE READ只读事务同快照计算当前批准额、原核销、更正及净数，返回asOf=事务快照时间；大导出也同快照而非拼接不同时间分页。Q-OP按服务端actor/event/action和operationId读取，不用operationId全库探测。

T-CLEANUP先锁Event并ARCHIVED/停写/失效会话与能力，记录独立deletion_ledger，随后分范围实际处理：导出/暂存外部文件，delivery/export、rows/links/batches、challenges/sessions、corrections/admissions/reviews/credentials/party、operation私密响应/可关联audit/outbox/job。删除顺序满足FK，清理前只保活动总计。connection_event/source在清理相关活动前解除，长期平台连接不随一个活动消失。设备会话和活动grant到期撤销，残留身份元数据按其自身管理期限处理。任何保留的event/设备/授权元数据不得仍能回查已删参与者。外部存储确认后才标对应scope成功；失败保存无PII错误码并可重试。备份范围保持PENDING直到密钥失效/对应备份确已轮转验证；API可以展示部分scope失败或等待，不将live库成功等同全副本清完。

恢复流程在隔离环境应用最新独立删除账本，撤销已到期活动数据密钥、删除旧任务/文件/会话、重算无身份总计，确认后再开放网络。账本保存至少覆盖最长在存备份及恢复验证窗口，不含组/姓名/联系方式；它本身不得从更旧备份覆盖。未做恢复演练前此处是设计验收要求，不是已达成保证。

## 6. 更新记录

| 版本 | 日期 | 类型 | 说明 |
|---|---|---|---|
| v0.1 | 2026-10-21 | 初始数据设计 | 聚合映射、25表设计DDL、索引及事务/清理SQL清单；未执行迁移或验证并发。 |
| v0.2 | 2026-10-21 | 契约一致性 | 明确匿名公开上下文标记、稳定核销scope和本人授权读取；复用已有session/grant表，无新增表或已执行迁移。 |
| v0.3 | 2026-10-21 | 项目依据归档 | 固定GATE作用域及稳定目标摘要；明确字段元数据、授权目标选择、原报名响应和清理范围映射，D3按ADR-043完成；未执行SQL。 |
