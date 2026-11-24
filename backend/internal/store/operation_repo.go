package store

import (
	"context"
)

// InsertOperation attempts the idempotency insert; returns rows inserted (0 or 1).
func (s *Store) InsertOperation(ctx context.Context, sc *Scope, id, actorScope, eventScope string, eventID any, action, operationID string, digest []byte) (int64, error) {
	return s.Exec(ctx, sc,
		"insert into ug_operation(id,actor_scope,event_scope,event_id,action,operation_id,payload_digest,status,created_at) values($1,$2,$3,$4,$5,$6,$7,'IN_PROGRESS',now()) on conflict(actor_scope,event_scope,action,operation_id) do nothing",
		id, actorScope, eventScope, nullableUUID(eventID), action, operationID, digest)
}

// OperationByScope finds the prior operation for an idempotency key.
func (s *Store) OperationByScope(ctx context.Context, sc *Scope, actorScope, eventScope, action, operationID string) (map[string]any, error) {
	return s.Row(ctx, sc,
		"select * from ug_operation where actor_scope=$1 and event_scope=$2 and action=$3 and operation_id=$4",
		actorScope, eventScope, action, operationID)
}

// CompleteOperation stores the sealed result bundle and marks success.
func (s *Store) CompleteOperation(ctx context.Context, sc *Scope, resourceID any, resultCipher []byte, privateExpiry any, id string) error {
	_, err := s.Exec(ctx, sc,
		"update ug_operation set status='SUCCEEDED',result_code='OK',resource_id=$1,result_cipher=$2,private_result_expires_at=$3,committed_at=now() where id=$4",
		nullableUUID(resourceID), resultCipher, privateExpiry, id)
	return err
}

// RegistrationOperation finds the caller's succeeded registration result.
func (s *Store) RegistrationOperation(ctx context.Context, sc *Scope, actorScope, eventScope, operationID string) (map[string]any, error) {
	return s.Row(ctx, sc,
		"select * from ug_operation where actor_scope=$1 and event_scope=$2 and action='SubmitRegistration' and operation_id=$3 and status='SUCCEEDED' and private_result_expires_at>now()",
		actorScope, eventScope, operationID)
}

// OperationsByEventAction lists operations for result lookup.
func (s *Store) OperationsByEventAction(ctx context.Context, sc *Scope, eventID, operationID, action string) ([]map[string]any, error) {
	return s.Rows(ctx, sc,
		"select * from ug_operation where event_id=$1 and operation_id=$2 and action=$3",
		eventID, operationID, action)
}

// GateOperations joins GateCommit operations with admission facts.
func (s *Store) GateOperations(ctx context.Context, sc *Scope, operationID, eventID string) ([]map[string]any, error) {
	return s.Rows(ctx, sc,
		"select o.*,a.device_id,a.party_id,a.actor_id from ug_operation o left join ug_admission a on a.operation_entry_id=o.id where o.operation_id=$1 and o.actor_scope='GATE' and o.event_id=$2 and o.action='GateCommit'",
		operationID, eventID)
}