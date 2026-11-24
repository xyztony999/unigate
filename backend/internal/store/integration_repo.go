package store

import (
	"context"
	"time"
)

// ListConnections lists connections ordered by id.
func (s *Store) ListConnections(ctx context.Context, sc *Scope) ([]map[string]any, error) {
	return s.Rows(ctx, sc, "select * from ug_connection order by id")
}

// ConnectionByID loads a connection.
func (s *Store) ConnectionByID(ctx context.Context, sc *Scope, id string) (map[string]any, error) {
	return s.Row(ctx, sc, "select * from ug_connection where id=$1", id)
}

// ConnectionByIDForUpdate locks a connection.
func (s *Store) ConnectionByIDForUpdate(ctx context.Context, sc *Scope, id string) (map[string]any, error) {
	return s.Row(ctx, sc, "select * from ug_connection where id=$1 for update", id)
}

// ConnectionAllowed verifies the connection is assigned to the event.
func (s *Store) ConnectionAllowed(ctx context.Context, sc *Scope, id, eventID string) (map[string]any, error) {
	return s.Row(ctx, sc,
		"select c.* from ug_connection c join ug_connection_event ce on c.id=ce.connection_id where c.id=$1 and ce.event_id=$2",
		id, eventID)
}

// InsertConnection creates an UNAUTHORIZED connection.
func (s *Store) InsertConnection(ctx context.Context, sc *Scope, id, provider, accountKind, displayName string, clientID any, credentialsCipher []byte, createdBy string) error {
	_, err := s.Exec(ctx, sc,
		"insert into ug_connection(id,provider,account_kind,display_name,status,client_id,credentials_cipher,created_by) values($1,$2,$3,$4,'UNAUTHORIZED',$5,$6,$7)",
		id, provider, accountKind, displayName, clientID, credentialsCipher, createdBy)
	return err
}

// UpdateConnection applies edits and resets authorization.
func (s *Store) UpdateConnection(ctx context.Context, sc *Scope, accountKind, displayName string, clientID any, credentialsCipher []byte, id string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_connection set account_kind=$1,display_name=$2,client_id=coalesce($3,client_id),credentials_cipher=coalesce($4,credentials_cipher),status='UNAUTHORIZED',version=version+1 where id=$5",
		accountKind, displayName, clientID, credentialsCipher, id)
	return err
}

// RevokeConnection clears managed credentials.
func (s *Store) RevokeConnection(ctx context.Context, sc *Scope, id string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_connection set status='REVOKED',credentials_cipher=null,key_ref=null,expires_at=null,version=version+1 where id=$1", id)
	return err
}

// AuthorizeConnection stores exchanged credentials.
func (s *Store) AuthorizeConnection(ctx context.Context, sc *Scope, credentialsCipher []byte, expires time.Time, id string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_connection set credentials_cipher=$1,status='AUTHORIZED',expires_at=$2,version=version+1 where id=$3",
		credentialsCipher, expires, id)
	return err
}

// ConnectionEvents lists assigned event ids.
func (s *Store) ConnectionEvents(ctx context.Context, sc *Scope, connectionID string) ([]map[string]any, error) {
	return s.Rows(ctx, sc, "select event_id from ug_connection_event where connection_id=$1 order by event_id", connectionID)
}

// DeleteConnectionEvent removes an assignment.
func (s *Store) DeleteConnectionEvent(ctx context.Context, sc *Scope, connectionID, eventID string) error {
	_, err := s.Exec(ctx, sc, "delete from ug_connection_event where connection_id=$1 and event_id=$2", connectionID, eventID)
	return err
}

// InsertConnectionEvent assigns an event (idempotent).
func (s *Store) InsertConnectionEvent(ctx context.Context, sc *Scope, connectionID, eventID string) error {
	_, err := s.Exec(ctx, sc, "insert into ug_connection_event(connection_id,event_id) values($1,$2) on conflict do nothing", connectionID, eventID)
	return err
}

// EventExists checks an event id exists.
func (s *Store) EventExists(ctx context.Context, sc *Scope, id string) (map[string]any, error) {
	return s.Row(ctx, sc, "select id from ug_event where id=$1", id)
}

// SourcesByConnectionEvent checks source references before unassigning.
func (s *Store) SourcesByConnectionEvent(ctx context.Context, sc *Scope, connectionID, eventID string) ([]map[string]any, error) {
	return s.Rows(ctx, sc, "select id from ug_source where connection_id=$1 and event_id=$2", connectionID, eventID)
}

// SourceByIDForUpdate locks a source joined with provider.
func (s *Store) SourceByIDForUpdate(ctx context.Context, sc *Scope, id, eventID string) (map[string]any, error) {
	return s.Row(ctx, sc,
		"select a.*,c.provider from ug_source a join ug_connection c on a.connection_id=c.id where a.id=$1 and a.event_id=$2 for update of a",
		id, eventID)
}

// SourceByID loads a source (no lock).
func (s *Store) SourceByID(ctx context.Context, sc *Scope, id string) (map[string]any, error) {
	return s.Row(ctx, sc, "select * from ug_source where id=$1", id)
}

// SourceLockByID locks a source by id only (BatchCommit source version guard).
func (s *Store) SourceLockByID(ctx context.Context, sc *Scope, id string) (map[string]any, error) {
	return s.Row(ctx, sc, "select * from ug_source where id=$1 for update", id)
}

// OrganizerGrantForExport loads the caller's live ORGANIZER grant row with its
// allowed actions (IntegrationService.exportAccess).
func (s *Store) OrganizerGrantForExport(ctx context.Context, sc *Scope, eventID, accountID string) (map[string]any, error) {
	return s.Row(ctx, sc,
		"select * from ug_event_grant where event_id=$1 and account_id=$2 and role='ORGANIZER' and revoked_at is null and expires_at>now()",
		eventID, accountID)
}

// SourceByIDEvent loads a source within an event.
func (s *Store) SourceByIDEvent(ctx context.Context, sc *Scope, id, eventID string) (map[string]any, error) {
	return s.Row(ctx, sc, "select * from ug_source where id=$1 and event_id=$2", id, eventID)
}

// SourceByIDEventForUpdate locks a source within an event.
func (s *Store) SourceByIDEventForUpdate(ctx context.Context, sc *Scope, id, eventID string) (map[string]any, error) {
	return s.Row(ctx, sc, "select * from ug_source where id=$1 and event_id=$2 for update", id, eventID)
}

// SourceConnectionRef reads connection_id/resource_id of a source.
func (s *Store) SourceConnectionRef(ctx context.Context, sc *Scope, id string) (map[string]any, error) {
	return s.Row(ctx, sc, "select connection_id,resource_id from ug_source where id=$1", id)
}

// SourceProvider reads the provider of a source's connection.
func (s *Store) SourceProvider(ctx context.Context, sc *Scope, id string) (map[string]any, error) {
	return s.Row(ctx, sc,
		"select c.provider from ug_source a join ug_connection c on a.connection_id=c.id where a.id=$1", id)
}

// InsertSource creates a source.
func (s *Store) InsertSource(ctx context.Context, sc *Scope, id, eventID, connectionID, resourceID, mappingJSON, eligibility string) error {
	_, err := s.Exec(ctx, sc,
		"insert into ug_source(id,event_id,connection_id,resource_id,mapping,eligibility) values($1,$2,$3,$4,$5::jsonb,$6)",
		id, eventID, connectionID, resourceID, mappingJSON, eligibility)
	return err
}

// UpdateSource applies source edits.
func (s *Store) UpdateSource(ctx context.Context, sc *Scope, connectionID, resourceID, mappingJSON, eligibility, id string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_source set connection_id=$1,resource_id=$2,mapping=$3::jsonb,eligibility=$4,version=version+1 where id=$5",
		connectionID, resourceID, mappingJSON, eligibility, id)
	return err
}

// ListSources lists sources of an event joined with provider.
func (s *Store) ListSources(ctx context.Context, sc *Scope, eventID string) ([]map[string]any, error) {
	return s.Rows(ctx, sc,
		"select a.*,c.provider from ug_source a join ug_connection c on a.connection_id=c.id where a.event_id=$1 order by a.id",
		eventID)
}

// AvailableConnections lists FEISHU/TENCENT connections assigned to an event.
func (s *Store) AvailableConnections(ctx context.Context, sc *Scope, eventID string) ([]map[string]any, error) {
	return s.Rows(ctx, sc,
		"select c.* from ug_connection c join ug_connection_event ce on c.id=ce.connection_id where ce.event_id=$1 and c.provider in ('FEISHU','TENCENT') order by c.id",
		eventID)
}

// InsertBatch creates an UPLOADED batch.
func (s *Store) InsertBatch(ctx context.Context, sc *Scope, id, eventID, fileDigest, sheetRef string, sourceLabel any, expires time.Time) error {
	_, err := s.Exec(ctx, sc,
		"insert into ug_import_batch(id,event_id,file_digest,sheet_ref,source_label,status,expires_at) values($1,$2,$3,$4,$5,'UPLOADED',$6)",
		id, eventID, fileDigest, sheetRef, sourceLabel, expires)
	return err
}

// InsertPullBatch creates a batch from a source pull.
func (s *Store) InsertPullBatch(ctx context.Context, sc *Scope, id, eventID, sourceID, mappingJSON, eligibility string, sourceVersion any, expires time.Time) error {
	_, err := s.Exec(ctx, sc,
		"insert into ug_import_batch(id,event_id,source_id,mapping,eligibility,status,source_version,expires_at) values($1,$2,$3,$4::jsonb,$5,'UPLOADED',$6,$7)",
		id, eventID, sourceID, mappingJSON, eligibility, sourceVersion, expires)
	return err
}

// BatchByIDForUpdate locks a batch and enforces expiry.
func (s *Store) BatchByIDForUpdate(ctx context.Context, sc *Scope, id, eventID string) (map[string]any, error) {
	return s.Row(ctx, sc, "select * from ug_import_batch where id=$1 and event_id=$2 for update", id, eventID)
}

// UpdateBatchPreview stores mapping/eligibility and bumps preview version.
func (s *Store) UpdateBatchPreview(ctx context.Context, sc *Scope, mappingJSON, eligibility string, policyVersion any, previewDigest []byte, id string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_import_batch set mapping=$1::jsonb,eligibility=$2,status='PREVIEWED',policy_version=$3,preview_version=preview_version+1,preview_digest=$4,version=version+1 where id=$5",
		mappingJSON, eligibility, policyVersion, previewDigest, id)
	return err
}

// BumpBatchPreview bumps preview/version after a duplicate resolution.
func (s *Store) BumpBatchPreview(ctx context.Context, sc *Scope, id string) error {
	_, err := s.Exec(ctx, sc, "update ug_import_batch set preview_version=preview_version+1,version=version+1 where id=$1", id)
	return err
}

// CommitBatch marks a batch committed.
func (s *Store) CommitBatch(ctx context.Context, sc *Scope, operationID, id string) error {
	_, err := s.Exec(ctx, sc, "update ug_import_batch set status='COMMITTED',committed_operation_id=$1,version=version+1 where id=$2", operationID, id)
	return err
}

// InsertImportRow creates an ERROR-status import row.
func (s *Store) InsertImportRow(ctx context.Context, sc *Scope, id, eventID, batchID, sourceRecordID string, lineNumber int, rowCipher, contentDigest []byte) error {
	_, err := s.Exec(ctx, sc,
		"insert into ug_import_row(id,event_id,batch_id,source_record_id,line_number,row_cipher,content_digest,status) values($1,$2,$3,$4,$5,$6,$7,'ERROR')",
		id, eventID, batchID, sourceRecordID, lineNumber, rowCipher, contentDigest)
	return err
}

// BatchRows lists rows of a batch ordered by line,id.
func (s *Store) BatchRows(ctx context.Context, sc *Scope, batchID string) ([]map[string]any, error) {
	return s.Rows(ctx, sc, "select * from ug_import_row where batch_id=$1 order by line_number,id", batchID)
}

// UpdateImportRowPreview rewrites a row after preview normalization.
func (s *Store) UpdateImportRowPreview(ctx context.Context, sc *Scope, rowCipher []byte, status, errorsJSON string, requestedSize int, id string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_import_row set row_cipher=$1,status=$2,errors=$3::jsonb,requested_size=$4,resolution=null,linked_party_id=null,reason_cipher=null where id=$5",
		rowCipher, status, errorsJSON, requestedSize, id)
	return err
}

// ImportRowByIDForUpdate locks a row within a batch.
func (s *Store) ImportRowByIDForUpdate(ctx context.Context, sc *Scope, id, batchID string) (map[string]any, error) {
	return s.Row(ctx, sc, "select * from ug_import_row where id=$1 and batch_id=$2 for update", id, batchID)
}

// ResolveImportRow records a duplicate resolution.
func (s *Store) ResolveImportRow(ctx context.Context, sc *Scope, resolution string, linkedPartyID any, reasonCipher []byte, status, id string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_import_row set resolution=$1,linked_party_id=$2,reason_cipher=$3,status=$4 where id=$5",
		resolution, nullableUUID(linkedPartyID), reasonCipher, status, id)
	return err
}

// CommitImportRow marks a row committed and linked.
func (s *Store) CommitImportRow(ctx context.Context, sc *Scope, partyID, id string) error {
	_, err := s.Exec(ctx, sc, "update ug_import_row set status='COMMITTED',linked_party_id=$1 where id=$2", partyID, id)
	return err
}

// SourceLinks finds links for a scope/record key.
func (s *Store) SourceLinks(ctx context.Context, sc *Scope, scope, key string) ([]map[string]any, error) {
	return s.Rows(ctx, sc, "select party_id,source_content_digest from ug_source_link where source_scope=$1 and record_key=$2", scope, key)
}

// SourceLinkIDs finds link ids for a scope/record key.
func (s *Store) SourceLinkIDs(ctx context.Context, sc *Scope, scope, key string) ([]map[string]any, error) {
	return s.Rows(ctx, sc, "select id from ug_source_link where source_scope=$1 and record_key=$2", scope, key)
}

// InsertSourceLink records a source-to-party link.
func (s *Store) InsertSourceLink(ctx context.Context, sc *Scope, id, eventID, partyID, scope, key string, contentDigest []byte, batchID string, linkedAt time.Time) error {
	_, err := s.Exec(ctx, sc,
		"insert into ug_source_link(id,event_id,party_id,source_scope,record_key,source_content_digest,batch_id,linked_at) values($1,$2,$3,$4,$5,$6,$7,$8)",
		id, eventID, partyID, scope, key, contentDigest, batchID, linkedAt)
	return err
}

// InsertJob creates a job row.
func (s *Store) InsertJob(ctx context.Context, sc *Scope, id string, eventID any, kind, businessKey, status string, payloadCipher []byte, attempts int, nextRun time.Time, resultRef any, errorCode any, created time.Time, completedAt any, expires any) error {
	_, err := s.Exec(ctx, sc,
		"insert into ug_job(id,event_id,kind,business_key,status,payload_cipher,attempts,next_run_at,result_ref,error_code,created_at,completed_at,expires_at) values($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)",
		id, nullableUUID(eventID), kind, businessKey, status, payloadCipher, attempts, nextRun, nullableUUID(resultRef), errorCode, created, completedAt, expires)
	return err
}

// JobByID loads a job.
func (s *Store) JobByID(ctx context.Context, sc *Scope, id string) (map[string]any, error) {
	return s.Row(ctx, sc, "select * from ug_job where id=$1", id)
}

// JobByIDForUpdate locks a job.
func (s *Store) JobByIDForUpdate(ctx context.Context, sc *Scope, id string) (map[string]any, error) {
	return s.Row(ctx, sc, "select * from ug_job where id=$1 for update", id)
}

// JobByIDForUpdateRows locks a job, returning an empty slice when the row is
// gone (worker finish tolerates concurrently deleted jobs, Store.rows style).
func (s *Store) JobByIDForUpdateRows(ctx context.Context, sc *Scope, id string) ([]map[string]any, error) {
	return s.Rows(ctx, sc, "select * from ug_job where id=$1 for update", id)
}

// JobsByEvent lists jobs of an event newest first.
func (s *Store) JobsByEvent(ctx context.Context, sc *Scope, eventID string) ([]map[string]any, error) {
	return s.Rows(ctx, sc, "select * from ug_job where event_id=$1 order by created_at desc,id desc", eventID)
}

// ListJobs lists all jobs newest first.
func (s *Store) ListJobs(ctx context.Context, sc *Scope) ([]map[string]any, error) {
	return s.Rows(ctx, sc, "select * from ug_job order by created_at desc,id desc")
}

// ResetJob requeues a failed job.
func (s *Store) ResetJob(ctx context.Context, sc *Scope, nextRun time.Time, id string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_job set status='PENDING',attempts=0,error_code=null,lease_until=null,next_run_at=$1,completed_at=null,version=version+1 where id=$2",
		nextRun, id)
	return err
}

// CompleteJob marks a job succeeded with a result ref.
func (s *Store) CompleteJob(ctx context.Context, sc *Scope, resultRef string, completedAt time.Time, id string) error {
	_, err := s.Exec(ctx, sc, "update ug_job set status='SUCCEEDED',result_ref=$1,completed_at=$2 where id=$3", resultRef, completedAt, id)
	return err
}

// InsertChallenge creates a challenge (OAuth/SMS/recovery).
func (s *Store) InsertChallenge(ctx context.Context, sc *Scope, id, kind string, eventID, partyID, connectionID any, actorScope string, targetCipher, codeDigest, contextCipher []byte, expires time.Time) error {
	_, err := s.Exec(ctx, sc,
		"insert into ug_challenge(id,kind,event_id,party_id,connection_id,actor_scope,target_cipher,code_digest,context_cipher,expires_at) values($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)",
		id, kind, nullableUUID(eventID), nullableUUID(partyID), nullableUUID(connectionID), actorScope, targetCipher, codeDigest, contextCipher, expires)
	return err
}

// InsertOAuthChallenge creates an OAUTH state challenge.
func (s *Store) InsertOAuthChallenge(ctx context.Context, sc *Scope, id, connectionID, actorScope string, codeDigest, contextCipher []byte, expires time.Time) error {
	_, err := s.Exec(ctx, sc,
		"insert into ug_challenge(id,kind,connection_id,actor_scope,code_digest,context_cipher,expires_at) values($1,'OAUTH',$2,$3,$4,$5,$6)",
		id, connectionID, actorScope, codeDigest, contextCipher, expires)
	return err
}

// OAuthChallengeByDigest locks an OAUTH challenge by state digest.
func (s *Store) OAuthChallengeByDigest(ctx context.Context, sc *Scope, digest []byte) (map[string]any, error) {
	return s.Row(ctx, sc, "select * from ug_challenge where kind='OAUTH' and code_digest=$1 for update", digest)
}

// ConsumeChallenge marks a challenge consumed.
func (s *Store) ConsumeChallenge(ctx context.Context, sc *Scope, at time.Time, id string) error {
	_, err := s.Exec(ctx, sc, "update ug_challenge set consumed_at=$1 where id=$2", at, id)
	return err
}

// ChallengeByIDForUpdate locks a challenge.
func (s *Store) ChallengeByIDForUpdate(ctx context.Context, sc *Scope, id string) ([]map[string]any, error) {
	return s.Rows(ctx, sc, "select * from ug_challenge where id=$1 for update", id)
}

// ChallengeByID loads a challenge.
func (s *Store) ChallengeByID(ctx context.Context, sc *Scope, id string) (map[string]any, error) {
	return s.Row(ctx, sc, "select * from ug_challenge where id=$1", id)
}

// ChallengeByIDEventForUpdate locks a challenge within an event.
func (s *Store) ChallengeByIDEventForUpdate(ctx context.Context, sc *Scope, id, eventID string) (map[string]any, error) {
	return s.Row(ctx, sc, "select * from ug_challenge where id=$1 and event_id=$2 for update", id, eventID)
}

// IncrementChallengeAttempts bumps the brute-force counter.
func (s *Store) IncrementChallengeAttempts(ctx context.Context, sc *Scope, id string) error {
	_, err := s.Exec(ctx, sc, "update ug_challenge set attempts=attempts+1 where id=$1", id)
	return err
}

// UpdateChallengeContext replaces the sealed context.
func (s *Store) UpdateChallengeContext(ctx context.Context, sc *Scope, contextCipher []byte, id string) error {
	_, err := s.Exec(ctx, sc, "update ug_challenge set context_cipher=$1 where id=$2", contextCipher, id)
	return err
}

// ConsumeChallengeWithContext marks consumed and replaces context.
func (s *Store) ConsumeChallengeWithContext(ctx context.Context, sc *Scope, at time.Time, contextCipher []byte, id string) error {
	_, err := s.Exec(ctx, sc, "update ug_challenge set consumed_at=$1,context_cipher=$2 where id=$3", at, contextCipher, id)
	return err
}

// RecentChallenges lists SMS/recovery challenges from the last 10 minutes.
func (s *Store) RecentChallenges(ctx context.Context, sc *Scope) ([]map[string]any, error) {
	return s.Rows(ctx, sc, "select * from ug_challenge where expires_at>now()-interval '10 minutes' and kind in ('SMS','RECOVERY')")
}

// ActiveScopeChallenges checks the 60-second resend window.
func (s *Store) ActiveScopeChallenges(ctx context.Context, sc *Scope, scope string) ([]map[string]any, error) {
	return s.Rows(ctx, sc, "select id from ug_challenge where actor_scope=$1 and expires_at>now()+interval '4 minutes'", scope)
}

// InsertExport records an export artifact.
func (s *Store) InsertExport(ctx context.Context, sc *Scope, id, eventID, actorID, jobID, format, columnsJSON, filterJSON string, purposeCipher []byte, artifactKey string, artifactDigest []byte, rowCount int, status string, expires, created time.Time) error {
	_, err := s.Exec(ctx, sc,
		"insert into ug_export(id,event_id,actor_id,job_id,format,columns_json,filter_json,purpose_cipher,artifact_key,artifact_digest,row_count,status,expires_at,created_at) values($1,$2,$3,$4,$5,$6::jsonb,$7::jsonb,$8,$9,$10,$11,$12,$13,$14)",
		id, eventID, actorID, jobID, format, columnsJSON, filterJSON, purposeCipher, artifactKey, artifactDigest, rowCount, status, expires, created)
	return err
}

// ExportByIDEvent loads an export within an event.
func (s *Store) ExportByIDEvent(ctx context.Context, sc *Scope, id, eventID string) (map[string]any, error) {
	return s.Row(ctx, sc, "select * from ug_export where id=$1 and event_id=$2", id, eventID)
}

// ExportByID loads an export.
func (s *Store) ExportByID(ctx context.Context, sc *Scope, id string) (map[string]any, error) {
	return s.Row(ctx, sc, "select * from ug_export where id=$1", id)
}

// ListExports lists export id/artifact_key of an event.
func (s *Store) ListExports(ctx context.Context, sc *Scope, eventID string) ([]map[string]any, error) {
	return s.Rows(ctx, sc, "select id,artifact_key from ug_export where event_id=$1", eventID)
}

// ClearExportArtifact removes artifact references.
func (s *Store) ClearExportArtifact(ctx context.Context, sc *Scope, id string) error {
	_, err := s.Exec(ctx, sc, "update ug_export set artifact_key=null,artifact_digest=null where id=$1", id)
	return err
}

// ExpireExportArtifacts scrubs purpose and marks expired.
func (s *Store) ExpireExportArtifacts(ctx context.Context, sc *Scope, purposeCipher []byte, eventID string) error {
	_, err := s.Exec(ctx, sc, "update ug_export set purpose_cipher=$1,status='EXPIRED' where event_id=$2", purposeCipher, eventID)
	return err
}

// InsertDelivery records a delivery disclosure.
func (s *Store) InsertDelivery(ctx context.Context, sc *Scope, id, eventID, exportID string, recipientCipher, purposeCipher []byte, columnsJSON string, retentionCipher []byte, actorID string, created time.Time) error {
	_, err := s.Exec(ctx, sc,
		"insert into ug_delivery(id,event_id,export_id,recipient_cipher,purpose_cipher,columns_json,retention_note_cipher,actor_id,created_at) values($1,$2,$3,$4,$5,$6::jsonb,$7,$8,$9)",
		id, eventID, exportID, recipientCipher, purposeCipher, columnsJSON, retentionCipher, actorID, created)
	return err
}

// DeliveryByID loads a delivery.
func (s *Store) DeliveryByID(ctx context.Context, sc *Scope, id string) (map[string]any, error) {
	return s.Row(ctx, sc, "select * from ug_delivery where id=$1", id)
}

// ListDeliveries lists deliveries of an event newest first.
func (s *Store) ListDeliveries(ctx context.Context, sc *Scope, eventID string) ([]map[string]any, error) {
	return s.Rows(ctx, sc, "select * from ug_delivery where event_id=$1 order by created_at desc,id desc", eventID)
}

// CleanupsByEvent lists cleanup records of an event.
func (s *Store) CleanupsByEvent(ctx context.Context, sc *Scope, eventID string) ([]map[string]any, error) {
	return s.Rows(ctx, sc, "select * from ug_cleanup where event_id=$1", eventID)
}

// CleanupByID loads a cleanup record.
func (s *Store) CleanupByID(ctx context.Context, sc *Scope, id string) (map[string]any, error) {
	return s.Row(ctx, sc, "select * from ug_cleanup where id=$1", id)
}

// CleanupByIDForUpdate locks a cleanup within an event.
func (s *Store) CleanupByIDForUpdate(ctx context.Context, sc *Scope, id, eventID string) (map[string]any, error) {
	return s.Row(ctx, sc, "select * from ug_cleanup where id=$1 and event_id=$2 for update", id, eventID)
}

// UpdateCleanupFailed records cleanup results.
func (s *Store) UpdateCleanupFailed(ctx context.Context, sc *Scope, scopeResultsJSON string, processed, failed int, id string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_cleanup set status='FAILED',scope_results=$1::jsonb,processed_count=$2,failed_count=$3,version=version+1 where id=$4",
		scopeResultsJSON, processed, failed, id)
	return err
}

// UpsertDeletionLedger records the independent deletion ledger.
func (s *Store) UpsertDeletionLedger(ctx context.Context, sc *Scope, id, eventID string, dueAt time.Time, keyRef string, keyRevokedAt, liveCleanupAt, updatedAt time.Time) error {
	_, err := s.Exec(ctx, sc,
		"insert into ug_deletion_ledger(id,event_id,due_at,key_ref,key_revoked_at,live_cleanup_at,status,updated_at) values($1,$2,$3,$4,$5,$6,'BACKUP_PENDING',$7) on conflict(event_id) do update set key_revoked_at=excluded.key_revoked_at,live_cleanup_at=excluded.live_cleanup_at,status='BACKUP_PENDING',updated_at=excluded.updated_at",
		id, eventID, dueAt, keyRef, keyRevokedAt, liveCleanupAt, updatedAt)
	return err
}

// DueCleanup claims the next due PENDING cleanup (SKIP LOCKED).
func (s *Store) DueCleanup(ctx context.Context, sc *Scope) ([]map[string]any, error) {
	return s.Rows(ctx, sc,
		"select c.* from ug_cleanup c join ug_event e on c.event_id=e.id where c.due_at<=now() and c.status='PENDING' order by c.due_at,c.id for update of e skip locked limit 1")
}