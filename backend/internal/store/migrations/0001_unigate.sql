CREATE TABLE ug_account (
id uuid PRIMARY KEY, username varchar(80) NOT NULL UNIQUE, display_name varchar(100) NOT NULL,
password_hash text NOT NULL, roles jsonb NOT NULL, enabled boolean NOT NULL DEFAULT true,
must_change_password boolean NOT NULL DEFAULT true, auth_version bigint NOT NULL DEFAULT 0,
version bigint NOT NULL DEFAULT 0, created_at timestamptz NOT NULL DEFAULT now()
);


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


CREATE TABLE ug_event_grant (
id uuid PRIMARY KEY, event_id uuid NOT NULL REFERENCES ug_event(id), account_id uuid NOT NULL REFERENCES ug_account(id),
role varchar(20) NOT NULL CHECK(role IN ('ORGANIZER','CHECKER','READER')), allowed_fields jsonb NOT NULL,
allowed_actions jsonb NOT NULL, disclosure_cipher bytea, expires_at timestamptz NOT NULL, revoked_at timestamptz,
version bigint NOT NULL DEFAULT 0, UNIQUE(event_id,account_id,role)
);


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


CREATE TABLE ug_party_review (
id uuid PRIMARY KEY, event_id uuid NOT NULL, party_id uuid NOT NULL,
kind varchar(20) NOT NULL CHECK(kind IN ('INITIAL','INCREASE')), requested_size integer NOT NULL CHECK(requested_size>=1),
requested_increase integer NOT NULL CHECK(requested_increase>=0), result varchar(20) NOT NULL,
reason_cipher bytea, actor_id uuid REFERENCES ug_account(id), version bigint NOT NULL DEFAULT 0,
created_at timestamptz NOT NULL DEFAULT now(), decided_at timestamptz,
FOREIGN KEY(event_id,party_id) REFERENCES ug_party(event_id,id)
);


CREATE TABLE ug_credential (
id uuid PRIMARY KEY, event_id uuid NOT NULL, party_id uuid NOT NULL,
kind varchar(16) NOT NULL CHECK(kind IN ('TICKET','QUERY','CHANGE')), token_digest bytea NOT NULL UNIQUE,
token_cipher bytea, credential_version bigint NOT NULL, status varchar(16) NOT NULL,
expires_at timestamptz NOT NULL, created_at timestamptz NOT NULL DEFAULT now(),
FOREIGN KEY(event_id,party_id) REFERENCES ug_party(event_id,id)
);


CREATE TABLE ug_device (
id uuid PRIMARY KEY, event_id uuid NOT NULL REFERENCES ug_event(id), name varchar(100) NOT NULL,
mode varchar(16) NOT NULL CHECK(mode IN ('STAFF','KIOSK')), status varchar(16) NOT NULL,
session_digest bytea UNIQUE, pairing_digest bytea UNIQUE, pairing_expires_at timestamptz,
pairing_failures integer NOT NULL DEFAULT 0, expires_at timestamptz NOT NULL,
last_seen_at timestamptz, version bigint NOT NULL DEFAULT 0, created_by uuid NOT NULL REFERENCES ug_account(id)
);


CREATE TABLE ug_session (
id uuid PRIMARY KEY, token_digest bytea NOT NULL UNIQUE,
kind varchar(20) NOT NULL CHECK(kind IN ('STAFF','ANONYMOUS','PARTY_QUERY','PARTY_CHANGE')),
account_id uuid REFERENCES ug_account(id), event_id uuid REFERENCES ug_event(id), party_id uuid,
capability_version bigint, auth_version bigint, csrf_digest bytea NOT NULL,
created_at timestamptz NOT NULL, last_seen_at timestamptz NOT NULL, expires_at timestamptz NOT NULL, revoked_at timestamptz,
FOREIGN KEY(event_id,party_id) REFERENCES ug_party(event_id,id),
CHECK((kind='STAFF' AND account_id IS NOT NULL) OR (kind='ANONYMOUS' AND account_id IS NULL AND party_id IS NULL) OR (kind IN ('PARTY_QUERY','PARTY_CHANGE') AND event_id IS NOT NULL AND party_id IS NOT NULL))
);


CREATE TABLE ug_connection (
id uuid PRIMARY KEY, provider varchar(20) NOT NULL, account_kind varchar(20) NOT NULL,
display_name varchar(100) NOT NULL, status varchar(20) NOT NULL, client_id varchar(200),
credentials_cipher bytea, key_ref varchar(200), expires_at timestamptz,
version bigint NOT NULL DEFAULT 0, created_by uuid NOT NULL REFERENCES ug_account(id)
);


CREATE TABLE ug_connection_event (
connection_id uuid NOT NULL REFERENCES ug_connection(id), event_id uuid NOT NULL REFERENCES ug_event(id),
PRIMARY KEY(connection_id,event_id)
);


CREATE TABLE ug_challenge (
id uuid PRIMARY KEY, kind varchar(20) NOT NULL, event_id uuid REFERENCES ug_event(id), party_id uuid,
connection_id uuid REFERENCES ug_connection(id), actor_scope varchar(200) NOT NULL,
target_cipher bytea, code_digest bytea NOT NULL, context_cipher bytea,
attempts integer NOT NULL DEFAULT 0 CHECK(attempts BETWEEN 0 AND 5), expires_at timestamptz NOT NULL, consumed_at timestamptz,
FOREIGN KEY(event_id,party_id) REFERENCES ug_party(event_id,id)
);


CREATE TABLE ug_operation (
id uuid PRIMARY KEY, actor_scope varchar(200) NOT NULL, event_scope varchar(50) NOT NULL,
event_id uuid REFERENCES ug_event(id), action varchar(100) NOT NULL, operation_id uuid NOT NULL,
payload_digest bytea NOT NULL, status varchar(20) NOT NULL, result_code varchar(80), resource_id uuid,
result_cipher bytea, private_result_expires_at timestamptz, created_at timestamptz NOT NULL, committed_at timestamptz,
UNIQUE(actor_scope,event_scope,action,operation_id), UNIQUE(id,event_id)
);


CREATE TABLE ug_admission (
id uuid PRIMARY KEY, event_id uuid NOT NULL, party_id uuid NOT NULL,
operation_entry_id uuid NOT NULL UNIQUE, device_id uuid NOT NULL REFERENCES ug_device(id),
actor_id uuid REFERENCES ug_account(id), admitted_count integer NOT NULL CHECK(admitted_count>0),
corrected_count integer NOT NULL DEFAULT 0 CHECK(corrected_count BETWEEN 0 AND admitted_count),
committed_at timestamptz NOT NULL, FOREIGN KEY(event_id,party_id) REFERENCES ug_party(event_id,id),
FOREIGN KEY(operation_entry_id,event_id) REFERENCES ug_operation(id,event_id), UNIQUE(event_id,party_id,id)
);


CREATE TABLE ug_admission_correction (
id uuid PRIMARY KEY, event_id uuid NOT NULL, party_id uuid NOT NULL, original_record_id uuid NOT NULL,
corrected_count integer NOT NULL CHECK(corrected_count>0), reason_cipher bytea NOT NULL,
actor_id uuid NOT NULL REFERENCES ug_account(id), operation_entry_id uuid NOT NULL UNIQUE REFERENCES ug_operation(id),
created_at timestamptz NOT NULL, FOREIGN KEY(event_id,party_id,original_record_id) REFERENCES ug_admission(event_id,party_id,id)
);


CREATE TABLE ug_source (
id uuid PRIMARY KEY, event_id uuid NOT NULL REFERENCES ug_event(id), connection_id uuid NOT NULL REFERENCES ug_connection(id),
resource_id varchar(300) NOT NULL, mapping jsonb NOT NULL, eligibility varchar(30) NOT NULL,
version bigint NOT NULL DEFAULT 0, UNIQUE(event_id,id), FOREIGN KEY(connection_id,event_id) REFERENCES ug_connection_event(connection_id,event_id)
);


CREATE TABLE ug_import_batch (
id uuid PRIMARY KEY, event_id uuid NOT NULL REFERENCES ug_event(id), source_id uuid,
file_digest varchar(64), sheet_ref varchar(200), source_label varchar(100), file_key varchar(300),
mapping jsonb, eligibility varchar(30), status varchar(20) NOT NULL, preview_version bigint NOT NULL DEFAULT 0,
policy_version bigint, source_version bigint, preview_digest bytea, version bigint NOT NULL DEFAULT 0,
expires_at timestamptz NOT NULL, committed_operation_id uuid REFERENCES ug_operation(id),
created_at timestamptz NOT NULL DEFAULT now(), UNIQUE(event_id,id), FOREIGN KEY(event_id,source_id) REFERENCES ug_source(event_id,id)
);


CREATE TABLE ug_import_row (
id uuid PRIMARY KEY, event_id uuid NOT NULL, batch_id uuid NOT NULL, source_record_id varchar(300) NOT NULL,
line_number integer NOT NULL CHECK(line_number>0), row_cipher bytea NOT NULL, content_digest bytea NOT NULL,
status varchar(30) NOT NULL, errors jsonb NOT NULL DEFAULT '[]', requested_size integer,
linked_party_id uuid, resolution varchar(30), reason_cipher bytea,
FOREIGN KEY(event_id,batch_id) REFERENCES ug_import_batch(event_id,id),
FOREIGN KEY(event_id,linked_party_id) REFERENCES ug_party(event_id,id), UNIQUE(batch_id,line_number), UNIQUE(batch_id,source_record_id)
);


CREATE TABLE ug_source_link (
id uuid PRIMARY KEY, event_id uuid NOT NULL, party_id uuid NOT NULL,
source_scope varchar(200) NOT NULL, record_key varchar(300) NOT NULL, source_content_digest bytea NOT NULL,
connection_id uuid REFERENCES ug_connection(id), resource_id varchar(300), source_record_id varchar(300),
batch_id uuid, linked_at timestamptz NOT NULL, FOREIGN KEY(event_id,party_id) REFERENCES ug_party(event_id,id),
UNIQUE(source_scope,record_key)
);


CREATE TABLE ug_job (
id uuid PRIMARY KEY, event_id uuid REFERENCES ug_event(id), kind varchar(20) NOT NULL,
business_key varchar(250) NOT NULL, status varchar(20) NOT NULL, payload_cipher bytea,
attempts integer NOT NULL DEFAULT 0 CHECK(attempts>=0), lease_until timestamptz, next_run_at timestamptz NOT NULL,
result_ref uuid, error_code varchar(80), version bigint NOT NULL DEFAULT 0,
created_at timestamptz NOT NULL, completed_at timestamptz, expires_at timestamptz, UNIQUE(kind,business_key)
);


CREATE TABLE ug_outbox (
id uuid PRIMARY KEY, event_id uuid REFERENCES ug_event(id), event_type varchar(80) NOT NULL,
resource_id uuid, operation_id uuid, schema_version integer NOT NULL DEFAULT 1, payload jsonb NOT NULL,
created_at timestamptz NOT NULL, published_at timestamptz
);


CREATE TABLE ug_export (
id uuid PRIMARY KEY, event_id uuid NOT NULL REFERENCES ug_event(id), actor_id uuid NOT NULL REFERENCES ug_account(id),
job_id uuid NOT NULL UNIQUE REFERENCES ug_job(id), format varchar(10) NOT NULL CHECK(format IN ('CSV','XLSX')),
columns_json jsonb NOT NULL, filter_json jsonb NOT NULL, purpose_cipher bytea NOT NULL,
artifact_key varchar(300), artifact_digest bytea, row_count integer, status varchar(20) NOT NULL,
expires_at timestamptz NOT NULL, version bigint NOT NULL DEFAULT 0, created_at timestamptz NOT NULL
);


CREATE TABLE ug_delivery (
id uuid PRIMARY KEY, event_id uuid NOT NULL REFERENCES ug_event(id), export_id uuid NOT NULL REFERENCES ug_export(id),
recipient_cipher bytea NOT NULL, purpose_cipher bytea NOT NULL, columns_json jsonb NOT NULL,
retention_note_cipher bytea NOT NULL, actor_id uuid NOT NULL REFERENCES ug_account(id), version bigint NOT NULL DEFAULT 0,
created_at timestamptz NOT NULL
);


CREATE TABLE ug_audit (
id uuid PRIMARY KEY, event_id uuid REFERENCES ug_event(id), action varchar(100) NOT NULL,
actor_id uuid REFERENCES ug_account(id), device_id uuid, resource_id uuid, operation_id uuid,
reason_cipher bytea, occurred_at timestamptz NOT NULL
);


CREATE TABLE ug_cleanup (
id uuid PRIMARY KEY, event_id uuid NOT NULL UNIQUE REFERENCES ug_event(id), due_at timestamptz NOT NULL,
status varchar(20) NOT NULL, scope_results jsonb NOT NULL, processed_count integer NOT NULL DEFAULT 0,
failed_count integer NOT NULL DEFAULT 0, completed_at timestamptz, version bigint NOT NULL DEFAULT 0
);


CREATE TABLE ug_deletion_ledger (
id uuid PRIMARY KEY, event_id uuid NOT NULL UNIQUE, due_at timestamptz NOT NULL,
key_ref varchar(200) NOT NULL, key_revoked_at timestamptz, live_cleanup_at timestamptz,
last_covered_backup_expires_at timestamptz, status varchar(30) NOT NULL, updated_at timestamptz NOT NULL
);


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
