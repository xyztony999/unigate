package store

import (
	"context"
	"time"
)

// AccountCount returns the number of accounts (bootstrap guard).
func (s *Store) AccountCount(ctx context.Context, sc *Scope) (int64, error) {
	return s.QueryInt(ctx, sc, "select count(*) from ug_account")
}

// InsertAccount creates an account with jsonb roles.
func (s *Store) InsertAccount(ctx context.Context, sc *Scope, id, username, displayName, passwordHash, rolesJSON string) error {
	_, err := s.Exec(ctx, sc,
		"insert into ug_account(id,username,display_name,password_hash,roles) values($1,$2,$3,$4,$5::jsonb)",
		id, username, displayName, passwordHash, rolesJSON)
	return err
}

// AccountByUsernameEnabled lists enabled accounts for a username.
func (s *Store) AccountByUsernameEnabled(ctx context.Context, sc *Scope, username string) ([]map[string]any, error) {
	return s.Rows(ctx, sc, "select * from ug_account where username=$1 and enabled=true", username)
}

// AccountByIDEnabledAuthVersion verifies the session auth version.
func (s *Store) AccountByIDEnabledAuthVersion(ctx context.Context, sc *Scope, id string, authVersion any) ([]map[string]any, error) {
	return s.Rows(ctx, sc, "select * from ug_account where id=$1 and enabled=true and auth_version=$2", id, authVersion)
}

// AccountByID loads one account (404 when absent).
func (s *Store) AccountByID(ctx context.Context, sc *Scope, id string) (map[string]any, error) {
	return s.Row(ctx, sc, "select * from ug_account where id=$1", id)
}

// AccountByIDForUpdate locks an account row.
func (s *Store) AccountByIDForUpdate(ctx context.Context, sc *Scope, id string) (map[string]any, error) {
	return s.Row(ctx, sc, "select * from ug_account where id=$1 for update", id)
}

// AccountVersion reads only the version column.
func (s *Store) AccountVersion(ctx context.Context, sc *Scope, id string) (map[string]any, error) {
	return s.Row(ctx, sc, "select version from ug_account where id=$1", id)
}

// AccountAdvisoryLock serializes account updates (pg_advisory_xact_lock).
func (s *Store) AccountAdvisoryLock(ctx context.Context, sc *Scope) error {
	_, err := s.Exec(ctx, sc, "select pg_advisory_xact_lock(14312111)")
	return err
}

// EnabledAdminCount counts enabled accounts holding ADMIN.
func (s *Store) EnabledAdminCount(ctx context.Context, sc *Scope) (int64, error) {
	return s.QueryInt(ctx, sc, "select count(*) from ug_account where enabled=true and roles @> '[\"ADMIN\"]'::jsonb")
}

// UpdateAccountPassword rotates the password and bumps auth/version.
func (s *Store) UpdateAccountPassword(ctx context.Context, sc *Scope, hash, id string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_account set password_hash=$1,must_change_password=false,auth_version=auth_version+1,version=version+1 where id=$2",
		hash, id)
	return err
}

// UpdateAccountReset sets a temporary password, forces change and bumps versions.
func (s *Store) UpdateAccountReset(ctx context.Context, sc *Scope, hash, id string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_account set password_hash=$1,must_change_password=true,auth_version=auth_version+1,version=version+1 where id=$2",
		hash, id)
	return err
}

// UpdateAccountMeta updates display name/roles/enabled and bumps versions.
func (s *Store) UpdateAccountMeta(ctx context.Context, sc *Scope, displayName, rolesJSON string, enabled bool, id string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_account set display_name=$1,roles=$2::jsonb,enabled=$3,auth_version=auth_version+1,version=version+1 where id=$4",
		displayName, rolesJSON, enabled, id)
	return err
}

// ListAccounts orders accounts by username,id.
func (s *Store) ListAccounts(ctx context.Context, sc *Scope) ([]map[string]any, error) {
	return s.Rows(ctx, sc, "select * from ug_account order by username,id")
}

// SessionByTokenDigest finds a live session by token digest.
func (s *Store) SessionByTokenDigest(ctx context.Context, sc *Scope, digest []byte) ([]map[string]any, error) {
	return s.Rows(ctx, sc, "select * from ug_session where token_digest=$1 and revoked_at is null and expires_at>now()", digest)
}

// SessionByID loads a session row.
func (s *Store) SessionByID(ctx context.Context, sc *Scope, id string) (map[string]any, error) {
	return s.Row(ctx, sc, "select * from ug_session where id=$1", id)
}

// InsertSession creates a session row.
func (s *Store) InsertSession(ctx context.Context, sc *Scope, id string, tokenDigest []byte, kind string, accountID, eventID, partyID any, authVersion any, csrfDigest []byte, created, lastSeen, expires time.Time) error {
	_, err := s.Exec(ctx, sc,
		"insert into ug_session(id,token_digest,kind,account_id,event_id,party_id,auth_version,csrf_digest,created_at,last_seen_at,expires_at) values($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)",
		id, tokenDigest, kind, nullableUUID(accountID), nullableUUID(eventID), nullableUUID(partyID), authVersion, csrfDigest, created, lastSeen, expires)
	return err
}

// TouchStaffSession refreshes last_seen_at for a live staff session.
func (s *Store) TouchStaffSession(ctx context.Context, sc *Scope, digest []byte) error {
	_, err := s.Exec(ctx, sc,
		"update ug_session set last_seen_at=now() where token_digest=$1 and kind='STAFF' and revoked_at is null and expires_at>now() and last_seen_at>now()-interval '30 minutes'",
		digest)
	return err
}

// RevokeSession revokes one session.
func (s *Store) RevokeSession(ctx context.Context, sc *Scope, id string) error {
	_, err := s.Exec(ctx, sc, "update ug_session set revoked_at=now() where id=$1", id)
	return err
}

// RevokeAccountSessions revokes every session of an account.
func (s *Store) RevokeAccountSessions(ctx context.Context, sc *Scope, accountID string) error {
	_, err := s.Exec(ctx, sc, "update ug_session set revoked_at=now() where account_id=$1", accountID)
	return err
}

// RevokePartySessions revokes every session of a party.
func (s *Store) RevokePartySessions(ctx context.Context, sc *Scope, partyID string) error {
	_, err := s.Exec(ctx, sc, "update ug_session set revoked_at=now() where party_id=$1", partyID)
	return err
}

// UpdateSessionCsrf replaces the csrf digest of a session.
func (s *Store) UpdateSessionCsrf(ctx context.Context, sc *Scope, digest []byte, id string) error {
	_, err := s.Exec(ctx, sc, "update ug_session set csrf_digest=$1 where id=$2", digest, id)
	return err
}

// EventGrantsForAccount lists live grants ordered by role priority.
func (s *Store) EventGrantsForAccount(ctx context.Context, sc *Scope, eventID, accountID string) ([]map[string]any, error) {
	return s.Rows(ctx, sc,
		"select * from ug_event_grant where event_id=$1 and account_id=$2 and revoked_at is null and expires_at>now() order by case role when 'ORGANIZER' then 0 when 'CHECKER' then 1 else 2 end",
		eventID, accountID)
}

// GrantByIDForUpdate locks a grant within an event.
func (s *Store) GrantByIDForUpdate(ctx context.Context, sc *Scope, id, eventID string) (map[string]any, error) {
	return s.Row(ctx, sc, "select * from ug_event_grant where id=$1 and event_id=$2 for update", id, eventID)
}

// GrantExists checks for an existing event/account/role grant.
func (s *Store) GrantExists(ctx context.Context, sc *Scope, eventID, accountID, role string) ([]map[string]any, error) {
	return s.Rows(ctx, sc, "select id from ug_event_grant where event_id=$1 and account_id=$2 and role=$3", eventID, accountID, role)
}

// UpsertGrant inserts or updates a grant (unique event/account/role).
func (s *Store) UpsertGrant(ctx context.Context, sc *Scope, id, eventID, accountID, role, fieldsJSON, actionsJSON string, disclosureCipher []byte, expires time.Time) error {
	_, err := s.Exec(ctx, sc,
		"insert into ug_event_grant(id,event_id,account_id,role,allowed_fields,allowed_actions,disclosure_cipher,expires_at) values($1,$2,$3,$4,$5::jsonb,$6::jsonb,$7,$8) on conflict(event_id,account_id,role) do update set allowed_fields=excluded.allowed_fields,allowed_actions=excluded.allowed_actions,disclosure_cipher=excluded.disclosure_cipher,expires_at=excluded.expires_at,revoked_at=null,version=ug_event_grant.version+1",
		id, eventID, accountID, role, fieldsJSON, actionsJSON, disclosureCipher, expires)
	return err
}

// GrantViewRow loads a grant joined with account display data.
func (s *Store) GrantViewRow(ctx context.Context, sc *Scope, eventID, accountID, role string) (map[string]any, error) {
	return s.Row(ctx, sc,
		"select g.*,a.display_name,a.username from ug_event_grant g join ug_account a on a.id=g.account_id where event_id=$1 and account_id=$2 and role=$3",
		eventID, accountID, role)
}

// ListGrantsJoined lists all live grants joined with accounts.
func (s *Store) ListGrantsJoined(ctx context.Context, sc *Scope) ([]map[string]any, error) {
	return s.Rows(ctx, sc,
		"select g.*,a.display_name,a.username from ug_event_grant g join ug_account a on a.id=g.account_id where g.revoked_at is null and g.expires_at>now() order by g.event_id,g.account_id")
}

// RevokeGrant marks a grant revoked and bumps version.
func (s *Store) RevokeGrant(ctx context.Context, sc *Scope, id string) error {
	_, err := s.Exec(ctx, sc, "update ug_event_grant set revoked_at=now(),version=version+1 where id=$1", id)
	return err
}

// ActiveOrganizerCount counts live organizer grants for an event.
func (s *Store) ActiveOrganizerCount(ctx context.Context, sc *Scope, eventID string) (int64, error) {
	return s.QueryInt(ctx, sc,
		"select count(*) from ug_event_grant where event_id=$1 and role='ORGANIZER' and revoked_at is null and expires_at>now()",
		eventID)
}

// OrganizerGrantForAccount finds the caller's organizer grant for export access.
func (s *Store) OrganizerGrantForAccount(ctx context.Context, sc *Scope, eventID, accountID string) (map[string]any, error) {
	return s.Row(ctx, sc,
		"select * from ug_event_grant where event_id=$1 and account_id=$2 and role='ORGANIZER' and revoked_at is null and expires_at>now()",
		eventID, accountID)
}

// ActiveOrganizerGrant checks a still-valid organizer grant for pull workers.
func (s *Store) ActiveOrganizerGrant(ctx context.Context, sc *Scope, eventID, accountID string) ([]map[string]any, error) {
	return s.Rows(ctx, sc,
		"select g.id from ug_event_grant g join ug_account a on g.account_id=a.id where g.event_id=$1 and g.account_id=$2 and g.role='ORGANIZER' and g.revoked_at is null and g.expires_at>now() and a.enabled=true and a.roles @> '[\"ORGANIZER\"]'::jsonb",
		eventID, accountID)
}

// GrantEventOptions lists non-expired events for the grant selector.
func (s *Store) GrantEventOptions(ctx context.Context, sc *Scope) ([]map[string]any, error) {
	return s.Rows(ctx, sc, "select id,title,starts_at from ug_event where personal_due_at>now() order by starts_at desc,id")
}

// LoginChallengeCount counts recent login-limit challenges for a scope.
func (s *Store) LoginChallengeCount(ctx context.Context, sc *Scope, scope string) (int64, error) {
	return s.QueryInt(ctx, sc, "select count(*) from ug_challenge where kind='LOGIN_LIMIT' and actor_scope=$1 and expires_at>now()", scope)
}

// InsertLoginChallenge records a failed login attempt (15 minute window).
func (s *Store) InsertLoginChallenge(ctx context.Context, sc *Scope, id, scope string, codeDigest []byte) error {
	_, err := s.Exec(ctx, sc,
		"insert into ug_challenge(id,kind,actor_scope,code_digest,expires_at) values($1,'LOGIN_LIMIT',$2,$3,now()+interval '15 minutes')",
		id, scope, codeDigest)
	return err
}

// IncrementPairingFailures bumps a device's pairing failure counter (cap 5).
func (s *Store) IncrementPairingFailures(ctx context.Context, sc *Scope, id string) error {
	_, err := s.Exec(ctx, sc, "update ug_device set pairing_failures=least(pairing_failures+1,5) where id=$1", id)
	return err
}

// DeviceBySessionDigest finds an active device by session digest.
func (s *Store) DeviceBySessionDigest(ctx context.Context, sc *Scope, digest []byte) ([]map[string]any, error) {
	return s.Rows(ctx, sc, "select * from ug_device where session_digest=$1 and status='ACTIVE' and expires_at>now()", digest)
}

// DeviceByID loads a device row.
func (s *Store) DeviceByID(ctx context.Context, sc *Scope, id string) (map[string]any, error) {
	return s.Row(ctx, sc, "select * from ug_device where id=$1", id)
}

// DeviceByIDForUpdate locks a device row.
func (s *Store) DeviceByIDForUpdate(ctx context.Context, sc *Scope, id string) (map[string]any, error) {
	return s.Row(ctx, sc, "select * from ug_device where id=$1 for update", id)
}

// DeviceActiveForUpdate locks an active, unexpired device row.
func (s *Store) DeviceActiveForUpdate(ctx context.Context, sc *Scope, id string) (map[string]any, error) {
	return s.Row(ctx, sc, "select * from ug_device where id=$1 and status='ACTIVE' and expires_at>now() for update", id)
}

// DeviceByIDEventForUpdate locks a device within an event.
func (s *Store) DeviceByIDEventForUpdate(ctx context.Context, sc *Scope, id, eventID string) (map[string]any, error) {
	return s.Row(ctx, sc, "select * from ug_device where id=$1 and event_id=$2 for update", id, eventID)
}

// InsertDevice creates an unclaimed device.
func (s *Store) InsertDevice(ctx context.Context, sc *Scope, id, eventID, name, mode string, pairingDigest []byte, pairingExpires, expires time.Time, createdBy string) error {
	_, err := s.Exec(ctx, sc,
		"insert into ug_device(id,event_id,name,mode,status,pairing_digest,pairing_expires_at,expires_at,created_by) values($1,$2,$3,$4,'UNCLAIMED',$5,$6,$7,$8)",
		id, eventID, name, mode, pairingDigest, pairingExpires, expires, createdBy)
	return err
}

// ClaimDevice activates a device and clears the pairing digest.
func (s *Store) ClaimDevice(ctx context.Context, sc *Scope, sessionDigest []byte, id string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_device set status='ACTIVE',session_digest=$1,pairing_digest=null,version=version+1,last_seen_at=now() where id=$2",
		sessionDigest, id)
	return err
}

// RevokeDevice revokes a device and clears its digests.
func (s *Store) RevokeDevice(ctx context.Context, sc *Scope, id string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_device set status='REVOKED',session_digest=null,pairing_digest=null,version=version+1 where id=$1", id)
	return err
}

// ListDevices lists devices of an event ordered by name,id.
func (s *Store) ListDevices(ctx context.Context, sc *Scope, eventID string) ([]map[string]any, error) {
	return s.Rows(ctx, sc, "select * from ug_device where event_id=$1 order by name,id", eventID)
}