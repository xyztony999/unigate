package store

import (
	"context"
	"time"
)

// InsertEvent creates a DRAFT event.
func (s *Store) InsertEvent(ctx context.Context, sc *Scope, id, title string, venue any, startsAt, endsAt time.Time, timeZone string, regStarts, regEnds time.Time, capacity any, policyJSON string, personalDue time.Time, createdBy string) error {
	_, err := s.Exec(ctx, sc,
		"insert into ug_event(id,title,venue,starts_at,ends_at,time_zone,registration_starts_at,registration_ends_at,capacity,status,policy,personal_due_at,created_by) values($1,$2,$3,$4,$5,$6,$7,$8,$9,'DRAFT',$10::jsonb,$11,$12)",
		id, title, venue, startsAt, endsAt, timeZone, regStarts, regEnds, capacity, policyJSON, personalDue, createdBy)
	return err
}

// InsertOrganizerGrant seeds the creator's organizer grant.
func (s *Store) InsertOrganizerGrant(ctx context.Context, sc *Scope, id, eventID, accountID string, expires time.Time) error {
	_, err := s.Exec(ctx, sc,
		"insert into ug_event_grant(id,event_id,account_id,role,allowed_fields,allowed_actions,expires_at) values($1,$2,$3,'ORGANIZER',$4::jsonb,'[\"VIEW_DETAIL\",\"VIEW_STATISTICS\",\"CHECKIN\"]'::jsonb,$5)",
		id, eventID, accountID, "[\"name\",\"phone\",\"email\",\"companionCount\"]", expires)
	return err
}

// InsertCleanup seeds the cleanup record with scope results.
func (s *Store) InsertCleanup(ctx context.Context, sc *Scope, id, eventID string, dueAt time.Time, scopeResultsJSON string) error {
	_, err := s.Exec(ctx, sc,
		"insert into ug_cleanup(id,event_id,due_at,status,scope_results) values($1,$2,$3,'PENDING',$4::jsonb)",
		id, eventID, dueAt, scopeResultsJSON)
	return err
}

// EventByID loads an event without locking.
func (s *Store) EventByID(ctx context.Context, sc *Scope, id string) (map[string]any, error) {
	return s.Row(ctx, sc, "select * from ug_event where id=$1", id)
}

// EventByIDForUpdate locks an event row.
func (s *Store) EventByIDForUpdate(ctx context.Context, sc *Scope, id string) (map[string]any, error) {
	return s.Row(ctx, sc, "select * from ug_event where id=$1 for update", id)
}

// EventDue reads only personal_due_at.
func (s *Store) EventDue(ctx context.Context, sc *Scope, id string) (map[string]any, error) {
	return s.Row(ctx, sc, "select personal_due_at from ug_event where id=$1", id)
}

// ListEventsForAccount lists distinct events the account holds a live grant on.
func (s *Store) ListEventsForAccount(ctx context.Context, sc *Scope, accountID, rolesJSON string) ([]map[string]any, error) {
	return s.Rows(ctx, sc,
		"select distinct e.* from ug_event e join ug_event_grant g on g.event_id=e.id where g.account_id=$1 and g.revoked_at is null and g.expires_at>now() and e.personal_due_at>now() and $2::jsonb @> jsonb_build_array(g.role) order by e.starts_at desc,e.id",
		accountID, rolesJSON)
}

// UpdateEvent applies metadata edits and bumps version.
func (s *Store) UpdateEvent(ctx context.Context, sc *Scope, title string, venue any, startsAt, endsAt time.Time, timeZone string, regStarts, regEnds time.Time, capacity any, personalDue time.Time, id string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_event set title=$1,venue=$2,starts_at=$3,ends_at=$4,time_zone=$5,registration_starts_at=$6,registration_ends_at=$7,capacity=$8,personal_due_at=$9,version=version+1 where id=$10",
		title, venue, startsAt, endsAt, timeZone, regStarts, regEnds, capacity, personalDue, id)
	return err
}

// UpdateEventPolicy replaces the policy and bumps versions.
func (s *Store) UpdateEventPolicy(ctx context.Context, sc *Scope, policyJSON string, retentionDays any, personalDue time.Time, id string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_event set policy=$1::jsonb,policy_version=policy_version+1,retention_days=$2,personal_due_at=$3,version=version+1 where id=$4",
		policyJSON, retentionDays, personalDue, id)
	return err
}

// UpdateEventStatus transitions the event status.
func (s *Store) UpdateEventStatus(ctx context.Context, sc *Scope, status, id string) error {
	_, err := s.Exec(ctx, sc, "update ug_event set status=$1,version=version+1 where id=$2", status, id)
	return err
}

// AllocateEvent adjusts allocated_count within [0,capacity]; returns rows changed.
func (s *Store) AllocateEvent(ctx context.Context, sc *Scope, delta int, id string) (int64, error) {
	return s.Exec(ctx, sc,
		"update ug_event set allocated_count=allocated_count+$1,version=version+1 where id=$2 and allocated_count+$3 between 0 and capacity",
		delta, id, delta)
}

// PartyCount counts parties of an event (policy-in-use guard).
func (s *Store) PartyCount(ctx context.Context, sc *Scope, eventID string) (int64, error) {
	return s.QueryInt(ctx, sc, "select count(*) from ug_party where event_id=$1", eventID)
}

// PartyStats aggregates party counters for Statistics.
func (s *Store) PartyStats(ctx context.Context, sc *Scope, eventID string) (map[string]any, error) {
	return s.Row(ctx, sc,
		"select count(*) groups,count(*) filter(where status='PENDING') pending,count(*) filter(where status='APPROVED') approved,coalesce(sum(net_admitted),0) net,count(*) filter(where source_type='WALK_IN') walkins from ug_party where event_id=$1",
		eventID)
}

// AdmissionGross sums admitted_count for an event.
func (s *Store) AdmissionGross(ctx context.Context, sc *Scope, eventID string) (int64, error) {
	return s.QueryInt(ctx, sc, "select coalesce(sum(admitted_count),0) from ug_admission where event_id=$1", eventID)
}

// AdmissionCorrected sums corrected_count for an event.
func (s *Store) AdmissionCorrected(ctx context.Context, sc *Scope, eventID string) (int64, error) {
	return s.QueryInt(ctx, sc, "select coalesce(sum(corrected_count),0) from ug_admission where event_id=$1", eventID)
}

// OnsiteAdded sums OnsiteIncrease outbox counts for an event.
func (s *Store) OnsiteAdded(ctx context.Context, sc *Scope, eventID string) (int64, error) {
	return s.QueryInt(ctx, sc,
		"select coalesce(sum((payload->>'count')::integer),0) from ug_outbox where event_id=$1 and event_type='OnsiteIncrease'",
		eventID)
}

// InsertOutbox records an outbox event with jsonb payload.
func (s *Store) InsertOutbox(ctx context.Context, sc *Scope, id, eventID, eventType, resourceID, payloadJSON string) error {
	_, err := s.Exec(ctx, sc,
		"insert into ug_outbox(id,event_id,event_type,resource_id,payload,created_at) values($1,$2,$3,$4,$5::jsonb,now())",
		id, eventID, eventType, resourceID, payloadJSON)
	return err
}

// AuditList lists audit rows of an event newest first.
func (s *Store) AuditList(ctx context.Context, sc *Scope, eventID string) ([]map[string]any, error) {
	return s.Rows(ctx, sc, "select * from ug_audit where event_id=$1 order by occurred_at desc,id desc", eventID)
}