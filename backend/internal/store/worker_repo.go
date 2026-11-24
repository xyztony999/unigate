package store

import (
	"context"
	"fmt"
	"time"
)

func errcodeNewInvalidTable(table string) error {
	return fmt.Errorf("cleanup table not whitelisted: %s", table)
}

// ExpireJobs marks overdue PENDING/RUNNING jobs of a kind EXPIRED.
func (s *Store) ExpireJobs(ctx context.Context, sc *Scope, kind string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_job set status='EXPIRED',payload_cipher=null,lease_until=null,completed_at=now(),version=version+1 where kind=$1 and status in ('PENDING','RUNNING') and expires_at<=now() and (lease_until is null or lease_until<=now())",
		kind)
	return err
}

// FailExhaustedJobs marks abandoned RUNNING jobs with >=5 attempts FAILED.
func (s *Store) FailExhaustedJobs(ctx context.Context, sc *Scope, kind string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_job set status='FAILED',lease_until=null,error_code='PROVIDER_UNAVAILABLE',completed_at=now(),version=version+1 where kind=$1 and status='RUNNING' and lease_until<=now() and attempts>=5",
		kind)
	return err
}

// ClaimPullJob selects the next ready PULL job (event row SKIP LOCKED).
func (s *Store) ClaimPullJob(ctx context.Context, sc *Scope) ([]map[string]any, error) {
	return s.Rows(ctx, sc,
		"select j.* from ug_job j join ug_event e on j.event_id=e.id where j.kind='PULL' and ((j.status='PENDING' and j.next_run_at<=now()) or (j.status='RUNNING' and j.lease_until<=now())) and j.attempts<5 and j.expires_at>now() and e.personal_due_at>now() order by j.next_run_at,j.id for update of e skip locked limit 1")
}

// ClaimEmailJob selects the next ready EMAIL job (event row SKIP LOCKED).
func (s *Store) ClaimEmailJob(ctx context.Context, sc *Scope) ([]map[string]any, error) {
	return s.Rows(ctx, sc,
		"select j.* from ug_job j join ug_event e on j.event_id=e.id where j.kind='EMAIL' and ((j.status='PENDING' and j.next_run_at<=now()) or (j.status='RUNNING' and j.lease_until<=now())) and j.attempts<5 and j.expires_at>now() and e.personal_due_at>now() and e.status<>'ARCHIVED' order by j.next_run_at,j.id for update of e skip locked limit 1")
}

// LeaseJob marks a job RUNNING with a lease and bumps attempts.
func (s *Store) LeaseJob(ctx context.Context, sc *Scope, leaseUntil time.Time, id string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_job set status='RUNNING',attempts=attempts+1,lease_until=$1,version=version+1 where id=$2",
		leaseUntil, id)
	return err
}

// EventStatusDue reads status/personal_due_at under lock (notification finish).
func (s *Store) EventStatusDue(ctx context.Context, sc *Scope, id string) ([]map[string]any, error) {
	return s.Rows(ctx, sc, "select status,personal_due_at from ug_event where id=$1 for update", id)
}

// FinishJob applies the worker result with optimistic version guard.
func (s *Store) FinishJob(ctx context.Context, sc *Scope, result string, errorCode any, nextRun time.Time, completedAt any, clearPayload bool, id string, version any) error {
	_, err := s.Exec(ctx, sc,
		"update ug_job set status=$1,error_code=$2,lease_until=null,next_run_at=$3,completed_at=$4,payload_cipher=case when $5 then null else payload_cipher end,version=version+1 where id=$6 and version=$7",
		result, errorCode, nextRun, completedAt, clearPayload, id, version)
	return err
}

// FailOrRetryPullJob records a pull failure with backoff.
func (s *Store) FailOrRetryPullJob(ctx context.Context, sc *Scope, status, failure string, nextRun time.Time, completedAt any, id string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_job set status=$1,lease_until=null,error_code=$2,next_run_at=$3,completed_at=$4,version=version+1 where id=$5",
		status, failure, nextRun, completedAt, id)
	return err
}

// ExpireClaimedJob marks a claimed job EXPIRED (deadline passed).
func (s *Store) ExpireClaimedJob(ctx context.Context, sc *Scope, at time.Time, id string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_job set status='EXPIRED',payload_cipher=null,lease_until=null,completed_at=$1,version=version+1 where id=$2",
		at, id)
	return err
}

// SucceedPullJob stores the batch result and clears the payload.
func (s *Store) SucceedPullJob(ctx context.Context, sc *Scope, resultRef string, at time.Time, id string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_job set status='SUCCEEDED',error_code=null,result_ref=$1,lease_until=null,payload_cipher=null,completed_at=$2,version=version+1 where id=$3",
		resultRef, at, id)
	return err
}

// ExpireEventJobs scrubs payloads of an event's jobs (cleanup).
func (s *Store) ExpireEventJobs(ctx context.Context, sc *Scope, eventID string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_job set payload_cipher=null,status='EXPIRED',lease_until=null where event_id=$1", eventID)
	return err
}

// DeleteEventRows removes rows of a whitelisted table for an event (cleanup).
// The table name comes from a fixed internal whitelist, never user input.
func (s *Store) DeleteEventRows(ctx context.Context, sc *Scope, table, eventID string) (int64, error) {
	allowed := map[string]string{
		"ug_delivery":             "delete from ug_delivery where event_id=$1",
		"ug_credential":           "delete from ug_credential where event_id=$1",
		"ug_session":              "delete from ug_session where event_id=$1",
		"ug_challenge":            "delete from ug_challenge where event_id=$1",
		"ug_source_link":          "delete from ug_source_link where event_id=$1",
		"ug_import_row":           "delete from ug_import_row where event_id=$1",
		"ug_import_batch":         "delete from ug_import_batch where event_id=$1",
		"ug_admission_correction": "delete from ug_admission_correction where event_id=$1",
		"ug_admission":            "delete from ug_admission where event_id=$1",
		"ug_party_review":         "delete from ug_party_review where event_id=$1",
		"ug_party":                "delete from ug_party where event_id=$1",
		"ug_audit":                "delete from ug_audit where event_id=$1",
		"ug_outbox":               "delete from ug_outbox where event_id=$1",
		"ug_device":               "delete from ug_device where event_id=$1",
		"ug_source":               "delete from ug_source where event_id=$1",
		"ug_export":               "delete from ug_export where event_id=$1",
		"ug_job":                  "delete from ug_job where event_id=$1",
	}
	sql, ok := allowed[table]
	if !ok {
		return 0, errcodeNewInvalidTable(table)
	}
	return s.Exec(ctx, sc, sql, eventID)
}

// ClearGrantDisclosure nulls disclosure ciphertexts of an event (cleanup).
func (s *Store) ClearGrantDisclosure(ctx context.Context, sc *Scope, eventID string) error {
	_, err := s.Exec(ctx, sc, "update ug_event_grant set disclosure_cipher=null where event_id=$1", eventID)
	return err
}

// DeleteNonRetryOperations removes an event's operations except CleanupRetry.
func (s *Store) DeleteNonRetryOperations(ctx context.Context, sc *Scope, eventID string) error {
	_, err := s.Exec(ctx, sc, "delete from ug_operation where event_id=$1 and action<>'CleanupRetry'", eventID)
	return err
}

// ScrubOperationResults clears stored results of an event's operations.
func (s *Store) ScrubOperationResults(ctx context.Context, sc *Scope, eventID string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_operation set result_cipher=null,private_result_expires_at=null,resource_id=null where event_id=$1", eventID)
	return err
}

// ArchiveEventWithTotals stores anonymous aggregates and archives the event.
func (s *Store) ArchiveEventWithTotals(ctx context.Context, sc *Scope, totalsJSON, eventID string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_event set anonymous_totals=$1::jsonb,status='ARCHIVED',version=version+1 where id=$2",
		totalsJSON, eventID)
	return err
}

// ArchiveEvent forces ARCHIVED status.
func (s *Store) ArchiveEvent(ctx context.Context, sc *Scope, eventID string) error {
	_, err := s.Exec(ctx, sc, "update ug_event set status='ARCHIVED' where id=$1", eventID)
	return err
}

// CleanupTotals aggregates party counters before deletion.
func (s *Store) CleanupTotals(ctx context.Context, sc *Scope, eventID string) (map[string]any, error) {
	return s.Row(ctx, sc,
		"select count(*) application_groups,count(*) filter(where status='PENDING') pending_groups,count(*) filter(where status='APPROVED') approved_groups,coalesce(sum(net_admitted),0) net_admissions,count(*) filter(where source_type='WALK_IN') walk_in_groups from ug_party where event_id=$1",
		eventID)
}

// CleanupAdmissionTotals aggregates admission counters before deletion.
func (s *Store) CleanupAdmissionTotals(ctx context.Context, sc *Scope, eventID string) (map[string]any, error) {
	return s.Row(ctx, sc,
		"select coalesce(sum(admitted_count),0) gross_admissions,coalesce(sum(corrected_count),0) corrected_people from ug_admission where event_id=$1",
		eventID)
}