package store

import (
	"context"
	"time"
)

// PartyRaw locks a party row within an event (PartyService.raw).
func (s *Store) PartyRaw(ctx context.Context, sc *Scope, eventID, partyID string) (map[string]any, error) {
	return s.Row(ctx, sc, "select * from ug_party where event_id=$1 and id=$2 for update", eventID, partyID)
}

// InsertParty creates a party with sealed fields and search HMACs.
func (s *Store) InsertParty(ctx context.Context, sc *Scope, id, eventID, reference string, fieldsCipher []byte, dataKeyRef string, nameHMAC, phoneHMAC, emailHMAC []byte, requestedSize, approvedSize int, status, sourceType, noticeVersion string) error {
	_, err := s.Exec(ctx, sc,
		"insert into ug_party(id,event_id,party_reference,fields_cipher,data_key_ref,name_search_hmac,phone_search_hmac,email_search_hmac,requested_size,approved_size,status,source_type,accepted_notice_version) values($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)",
		id, eventID, reference, fieldsCipher, dataKeyRef, nameHMAC, phoneHMAC, emailHMAC, requestedSize, approvedSize, status, sourceType, noticeVersion)
	return err
}

// ListParties lists parties newest first.
func (s *Store) ListParties(ctx context.Context, sc *Scope, eventID string) ([]map[string]any, error) {
	return s.Rows(ctx, sc, "select * from ug_party where event_id=$1 order by created_at desc,id desc", eventID)
}

// ListPartiesAsc lists parties oldest first (export ordering).
func (s *Store) ListPartiesAsc(ctx context.Context, sc *Scope, eventID string) ([]map[string]any, error) {
	return s.Rows(ctx, sc, "select * from ug_party where event_id=$1 order by created_at,id", eventID)
}

// PartiesByReference finds parties by event and reference.
func (s *Store) PartiesByReference(ctx context.Context, sc *Scope, eventID, reference string) ([]map[string]any, error) {
	return s.Rows(ctx, sc, "select * from ug_party where event_id=$1 and party_reference=$2", eventID, reference)
}

// ListApprovedParties lists approved parties newest first (gate lookup).
func (s *Store) ListApprovedParties(ctx context.Context, sc *Scope, eventID string) ([]map[string]any, error) {
	return s.Rows(ctx, sc, "select * from ug_party where event_id=$1 and status='APPROVED' order by created_at desc,id desc", eventID)
}

// CancelParty cancels a party, zeroing approved/pending sizes.
func (s *Store) CancelParty(ctx context.Context, sc *Scope, id string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_party set status='CANCELLED',approved_size=0,pending_increase=0,version=version+1,updated_at=now() where id=$1", id)
	return err
}

// ChangeParty updates sealed fields and sizes.
func (s *Store) ChangeParty(ctx context.Context, sc *Scope, fieldsCipher []byte, requested, approved, pending int, id string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_party set fields_cipher=$1,requested_size=$2,approved_size=$3,pending_increase=$4,version=version+1,updated_at=now() where id=$5",
		fieldsCipher, requested, approved, pending, id)
	return err
}

// AdmitParty increments net_admitted and marks ever_admitted.
func (s *Store) AdmitParty(ctx context.Context, sc *Scope, count int, id string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_party set net_admitted=net_admitted+$1,ever_admitted=true,version=version+1,updated_at=now() where id=$2",
		count, id)
	return err
}

// CorrectParty decrements net_admitted after a correction.
func (s *Store) CorrectParty(ctx context.Context, sc *Scope, count int, id string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_party set net_admitted=net_admitted-$1,version=version+1,updated_at=now() where id=$2",
		count, id)
	return err
}

// ApproveParty adds approved people and clears pending increase.
func (s *Store) ApproveParty(ctx context.Context, sc *Scope, delta int, id string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_party set status='APPROVED',approved_size=approved_size+$1,pending_increase=0,version=version+1,updated_at=now() where id=$2",
		delta, id)
	return err
}

// RejectParty rejects an initial review or keeps approved size for increases.
func (s *Store) RejectParty(ctx context.Context, sc *Scope, initial bool, id string) error {
	status := "APPROVED"
	if initial {
		status = "REJECTED"
	}
	_, err := s.Exec(ctx, sc,
		"update ug_party set status=$1,pending_increase=0,requested_size=case when $2 then requested_size else approved_size end,version=version+1,updated_at=now() where id=$3",
		status, initial, id)
	return err
}

// BumpPartyVersion touches version/updated_at (credential revoke/reissue).
func (s *Store) BumpPartyVersion(ctx context.Context, sc *Scope, id string) error {
	_, err := s.Exec(ctx, sc, "update ug_party set version=version+1,updated_at=now() where id=$1", id)
	return err
}

// WalkInParty approves a pending party with new fields and size.
func (s *Store) WalkInParty(ctx context.Context, sc *Scope, fieldsCipher []byte, size int, id string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_party set fields_cipher=$1,requested_size=$2,approved_size=$3,status='APPROVED',version=version+1,updated_at=now() where id=$4",
		fieldsCipher, size, size, id)
	return err
}

// IncreaseParty adds onsite people to approved/requested sizes.
func (s *Store) IncreaseParty(ctx context.Context, sc *Scope, count int, id string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_party set approved_size=approved_size+$1,requested_size=requested_size+$2,version=version+1,updated_at=now() where id=$3",
		count, count, id)
	return err
}

// InsertReview creates a PENDING review.
func (s *Store) InsertReview(ctx context.Context, sc *Scope, id, eventID, partyID, kind string, requested, increase int) error {
	_, err := s.Exec(ctx, sc,
		"insert into ug_party_review(id,event_id,party_id,kind,requested_size,requested_increase,result,created_at) values($1,$2,$3,$4,$5,$6,'PENDING',now())",
		id, eventID, partyID, kind, requested, increase)
	return err
}

// ReviewByIDForUpdate locks a review within an event.
func (s *Store) ReviewByIDForUpdate(ctx context.Context, sc *Scope, id, eventID string) (map[string]any, error) {
	return s.Row(ctx, sc, "select * from ug_party_review where id=$1 and event_id=$2 for update", id, eventID)
}

// ListReviews lists reviews joined with party approved_size.
func (s *Store) ListReviews(ctx context.Context, sc *Scope, eventID string) ([]map[string]any, error) {
	return s.Rows(ctx, sc,
		"select r.*,p.approved_size from ug_party_review r join ug_party p on p.id=r.party_id where r.event_id=$1 order by r.created_at desc,r.id desc",
		eventID)
}

// ReviewViewRow loads one review joined with party approved_size.
func (s *Store) ReviewViewRow(ctx context.Context, sc *Scope, id string) (map[string]any, error) {
	return s.Row(ctx, sc,
		"select r.*,p.approved_size from ug_party_review r join ug_party p on p.id=r.party_id where r.id=$1", id)
}

// DecideReview records the decision and encrypted reason.
func (s *Store) DecideReview(ctx context.Context, sc *Scope, result, actorID string, reasonCipher []byte, id string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_party_review set result=$1,actor_id=$2,reason_cipher=$3,decided_at=now(),version=version+1 where id=$4",
		result, actorID, reasonCipher, id)
	return err
}

// RejectPendingReviews rejects all pending reviews of a party.
func (s *Store) RejectPendingReviews(ctx context.Context, sc *Scope, partyID string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_party_review set result='REJECTED',decided_at=now(),version=version+1 where party_id=$1 and result='PENDING'",
		partyID)
	return err
}

// ApprovePendingReviews approves pending reviews during walk-in conversion.
func (s *Store) ApprovePendingReviews(ctx context.Context, sc *Scope, actorID string, reasonCipher []byte, partyID string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_party_review set result='APPROVED',actor_id=$1,reason_cipher=$2,decided_at=now(),version=version+1 where party_id=$3 and result='PENDING'",
		actorID, reasonCipher, partyID)
	return err
}

// InsertCredential issues a capability credential.
func (s *Store) InsertCredential(ctx context.Context, sc *Scope, id, eventID, partyID, kind string, tokenDigest []byte, tokenCipher []byte, credentialVersion any, expires time.Time) error {
	_, err := s.Exec(ctx, sc,
		"insert into ug_credential(id,event_id,party_id,kind,token_digest,token_cipher,credential_version,status,expires_at) values($1,$2,$3,$4,$5,$6,$7,'ACTIVE',$8)",
		id, eventID, partyID, kind, tokenDigest, tokenCipher, credentialVersion, expires)
	return err
}

// CredentialByToken finds an active credential by token digest and kind.
func (s *Store) CredentialByToken(ctx context.Context, sc *Scope, digest []byte, kind string) (map[string]any, error) {
	return s.Row(ctx, sc,
		"select * from ug_credential where token_digest=$1 and kind=$2 and status='ACTIVE' and expires_at>now()",
		digest, kind)
}

// CredentialByTokenEvent finds an active ticket credential within an event.
func (s *Store) CredentialByTokenEvent(ctx context.Context, sc *Scope, digest []byte, eventID string) (map[string]any, error) {
	return s.Row(ctx, sc,
		"select * from ug_credential where token_digest=$1 and event_id=$2 and kind='TICKET' and status='ACTIVE' and expires_at>now()",
		digest, eventID)
}

// ChangeCredentialByToken finds an active change credential for a party.
func (s *Store) ChangeCredentialByToken(ctx context.Context, sc *Scope, digest []byte, partyID string) (map[string]any, error) {
	return s.Row(ctx, sc,
		"select * from ug_credential where token_digest=$1 and kind='CHANGE' and party_id=$2 and status='ACTIVE' and expires_at>now()",
		digest, partyID)
}

// TicketCredentialForCommit re-validates the exact ticket version at commit.
func (s *Store) TicketCredentialForCommit(ctx context.Context, sc *Scope, id, eventID, partyID string, credentialVersion any) (map[string]any, error) {
	return s.Row(ctx, sc,
		"select * from ug_credential where id=$1 and event_id=$2 and party_id=$3 and kind='TICKET' and status='ACTIVE' and credential_version=$4 and expires_at>now()",
		id, eventID, partyID, credentialVersion)
}

// ActiveAccessCredentials lists active QUERY/CHANGE credentials of a party.
func (s *Store) ActiveAccessCredentials(ctx context.Context, sc *Scope, partyID string) ([]map[string]any, error) {
	return s.Rows(ctx, sc,
		"select * from ug_credential where party_id=$1 and kind in ('QUERY','CHANGE') and status='ACTIVE' and expires_at>now()",
		partyID)
}

// TicketCredentials lists ticket credentials newest first.
func (s *Store) TicketCredentials(ctx context.Context, sc *Scope, partyID string) ([]map[string]any, error) {
	return s.Rows(ctx, sc,
		"select * from ug_credential where party_id=$1 and kind='TICKET' order by created_at desc,id desc",
		partyID)
}

// RevokeTicketCredentials revokes active tickets of a party.
func (s *Store) RevokeTicketCredentials(ctx context.Context, sc *Scope, partyID string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_credential set status='REVOKED' where party_id=$1 and kind='TICKET' and status='ACTIVE'", partyID)
	return err
}

// RevokeTicketChangeCredentials revokes active tickets and change credentials.
func (s *Store) RevokeTicketChangeCredentials(ctx context.Context, sc *Scope, partyID string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_credential set status='REVOKED' where party_id=$1 and kind in ('TICKET','CHANGE') and status='ACTIVE'", partyID)
	return err
}

// RevokeAccessCredentials revokes active QUERY/CHANGE credentials.
func (s *Store) RevokeAccessCredentials(ctx context.Context, sc *Scope, partyID string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_credential set status='REVOKED' where party_id=$1 and kind in ('QUERY','CHANGE') and status='ACTIVE'", partyID)
	return err
}

// InsertAdmission records an admission fact.
func (s *Store) InsertAdmission(ctx context.Context, sc *Scope, id, eventID, partyID, operationEntryID, deviceID string, actorID any, count int, committedAt time.Time) error {
	_, err := s.Exec(ctx, sc,
		"insert into ug_admission(id,event_id,party_id,operation_entry_id,device_id,actor_id,admitted_count,committed_at) values($1,$2,$3,$4,$5,$6,$7,$8)",
		id, eventID, partyID, operationEntryID, deviceID, nullableUUID(actorID), count, committedAt)
	return err
}

// AdmissionByIDForUpdate locks an admission within an event.
func (s *Store) AdmissionByIDForUpdate(ctx context.Context, sc *Scope, id, eventID string) (map[string]any, error) {
	return s.Row(ctx, sc, "select * from ug_admission where id=$1 and event_id=$2 for update", id, eventID)
}

// AdmissionByOperationEntry finds the admission for an operation entry.
func (s *Store) AdmissionByOperationEntry(ctx context.Context, sc *Scope, entryID string) (map[string]any, error) {
	return s.Row(ctx, sc, "select * from ug_admission where operation_entry_id=$1", entryID)
}

// ListAdmissions lists admissions joined with operation ids.
func (s *Store) ListAdmissions(ctx context.Context, sc *Scope, eventID string) ([]map[string]any, error) {
	return s.Rows(ctx, sc,
		"select a.*,o.operation_id from ug_admission a join ug_operation o on o.id=a.operation_entry_id where a.event_id=$1 order by a.committed_at desc,a.id desc",
		eventID)
}

// AddCorrectedCount accumulates corrections on an admission.
func (s *Store) AddCorrectedCount(ctx context.Context, sc *Scope, count int, id string) error {
	_, err := s.Exec(ctx, sc, "update ug_admission set corrected_count=corrected_count+$1 where id=$2", count, id)
	return err
}

// InsertCorrection records a correction with encrypted reason.
func (s *Store) InsertCorrection(ctx context.Context, sc *Scope, id, eventID, partyID, originalRecordID string, count int, reasonCipher []byte, actorID, operationEntryID string) error {
	_, err := s.Exec(ctx, sc,
		"insert into ug_admission_correction(id,event_id,party_id,original_record_id,corrected_count,reason_cipher,actor_id,operation_entry_id,created_at) values($1,$2,$3,$4,$5,$6,$7,$8,now())",
		id, eventID, partyID, originalRecordID, count, reasonCipher, actorID, operationEntryID)
	return err
}