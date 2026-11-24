package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"unigate/backend/internal/errcode"
	"unigate/backend/internal/store"
)

// This file reproduces the lifecycle half of IntegrationService: cleanup
// detail/retry and runCleanup. Backup completion is never inferred: the
// BACKUPS scope always reports BACKUP_COVERAGE_UNVERIFIED until an operator
// confirms coverage, so the cleanup record stays FAILED with a deletion
// ledger tombstone even after every live row is gone.

// cleanupScopes is the fixed six-scope retention order.
var cleanupScopes = []string{"PRIVATE_ACCESS", "LIVE_ROWS", "TEMP_FILES", "EXPORT_FILES", "TASK_PAYLOADS", "BACKUPS"}

// cleanupTables is the reverse foreign-key deletion order; only event
// aggregates, cleanup metadata and the independent deletion ledger survive.
var cleanupTables = []string{
	"ug_delivery", "ug_credential", "ug_session", "ug_challenge",
	"ug_source_link", "ug_import_row", "ug_import_batch",
	"ug_admission_correction", "ug_admission", "ug_party_review",
	"ug_party", "ug_audit", "ug_outbox", "ug_device", "ug_source",
}

// RunCleanup implements CleanupRunner for LifecycleWorker: it executes the
// cleanup inside the worker's transaction and discards the job view (the
// scheduler only needs success/failure).
func (svc *Service) RunCleanup(ctx context.Context, sc *store.Scope, cleanup map[string]any) error {
	_, err := svc.runCleanup(ctx, sc, cleanup)
	return err
}

func (svc *Service) cleanupDetail(ctx context.Context, sc *store.Scope, r map[string]any, c *Req) (any, error) {
	ev := store.StrOrEmpty(r["eventId"])
	if err := c.Organizer(ctx, ev); err != nil {
		return nil, err
	}
	// Deliberately no expiry/archive gate: organizers must still see the
	// cleanup state of expired events.
	e, err := svc.S.EventByID(ctx, sc, ev)
	if err != nil {
		return nil, err
	}
	list, err := svc.S.CleanupsByEvent(ctx, sc, store.StrOrEmpty(e["id"]))
	if err != nil {
		return nil, err
	}
	var x map[string]any
	if len(list) == 0 {
		id := svc.S.ID()
		due, err := store.Time(e["personal_due_at"])
		if err != nil {
			return nil, fmt.Errorf("event due time: %w", err)
		}
		scopeJSON, err := store.Stringify(svc.scopeResults("PENDING", nil))
		if err != nil {
			return nil, err
		}
		if err := svc.S.InsertCleanup(ctx, sc, id, store.StrOrEmpty(e["id"]), due, scopeJSON); err != nil {
			return nil, err
		}
		x, err = svc.S.CleanupByID(ctx, sc, id)
		if err != nil {
			return nil, err
		}
	} else {
		x = list[0]
	}
	return svc.cleanupView(x), nil
}

// scopeResults builds the six scope entries with enum-rendered status.
func (svc *Service) scopeResults(status string, errorCode any) []any {
	out := make([]any, 0, len(cleanupScopes))
	failed := 0
	if status == "FAILED" {
		failed = 1
	}
	for _, k := range cleanupScopes {
		out = append(out, store.Map(
			"scope", svc.value("CleanupScope", k),
			"status", svc.value("JobStatus", status),
			"processedCount", 0,
			"failedCount", failed,
			"errorCode", errorCode,
		))
	}
	return out
}

func (svc *Service) cleanupView(x map[string]any) map[string]any {
	scopes := store.List(x["scope_results"])
	if len(scopes) != 6 {
		scopes = svc.scopeResults("PENDING", nil)
	}
	return store.Map(
		"cleanupId", x["id"],
		"eventId", x["event_id"],
		"dueAt", x["due_at"],
		"status", svc.value("JobStatus", x["status"]),
		"processedCount", x["processed_count"],
		"failedCount", x["failed_count"],
		"completedAt", x["completed_at"],
		"version", x["version"],
		"scopes", scopes,
	)
}

func (svc *Service) cleanupRetry(ctx context.Context, sc *store.Scope, r map[string]any, c *Req) (any, error) {
	ev := store.StrOrEmpty(r["eventId"])
	if err := c.Organizer(ctx, ev); err != nil {
		return nil, err
	}
	if _, err := svc.S.EventByIDForUpdate(ctx, sc, ev); err != nil {
		return nil, err
	}
	x, err := svc.S.CleanupByIDForUpdate(ctx, sc, store.StrOrEmpty(r["cleanupId"]), ev)
	if err != nil {
		return nil, err
	}
	if err := store.Version(x, r["version"]); err != nil {
		return nil, err
	}
	due, err := store.Time(x["due_at"])
	if err != nil {
		return nil, fmt.Errorf("cleanup due time: %w", err)
	}
	if err := errcode.Require(store.StrOrEmpty(x["status"]) == "FAILED" && !due.After(svc.S.Now()),
		409, "CLEANUP_NOT_RETRYABLE", "仅能重试已到期的失败清理"); err != nil {
		return nil, err
	}
	return svc.runCleanup(ctx, sc, x)
}

// runCleanup executes one due cleanup in the caller's transaction
// (IntegrationService.runCleanup, @Transactional end to end).
func (svc *Service) runCleanup(ctx context.Context, sc *store.Scope, x map[string]any) (map[string]any, error) {
	event := store.StrOrEmpty(x["event_id"])
	due, err := store.Time(x["due_at"])
	if err != nil {
		return nil, fmt.Errorf("cleanup due time: %w", err)
	}
	if err := errcode.Require(!due.After(svc.S.Now()), 409, "CLEANUP_NOT_DUE", "尚未到清理时间"); err != nil {
		return nil, err
	}
	e, err := svc.S.EventByIDForUpdate(ctx, sc, event)
	if err != nil {
		return nil, err
	}
	scopes := svc.scopeResults("SUCCEEDED", nil)
	// Save anonymous aggregates before deleting linkable facts. Repeated
	// cleanup preserves the original snapshot.
	if e["anonymous_totals"] == nil {
		totals, err := svc.S.CleanupTotals(ctx, sc, event)
		if err != nil {
			return nil, err
		}
		admission, err := svc.S.CleanupAdmissionTotals(ctx, sc, event)
		if err != nil {
			return nil, err
		}
		for k, v := range admission {
			totals[k] = v
		}
		totals["allocated_people"] = e["allocated_count"]
		totalsJSON, err := store.Stringify(totals)
		if err != nil {
			return nil, err
		}
		if err := svc.S.ArchiveEventWithTotals(ctx, sc, totalsJSON, event); err != nil {
			return nil, err
		}
	}
	if err := svc.S.ArchiveEvent(ctx, sc, event); err != nil {
		return nil, err
	}
	storage, err := filepath.Abs(svc.Cfg.StorageDirectory)
	if err != nil {
		return nil, fmt.Errorf("storage directory: %w", err)
	}
	filesDeleted := true
	exports, err := svc.S.ListExports(ctx, sc, event)
	if err != nil {
		return nil, err
	}
	for _, export := range exports {
		if export["artifact_key"] == nil {
			continue
		}
		if err := deleteManagedArtifact(storage, store.StrOrEmpty(export["artifact_key"])); err != nil {
			filesDeleted = false
			scopes[3] = store.Map(
				"scope", svc.value("CleanupScope", "EXPORT_FILES"),
				"status", svc.value("JobStatus", "FAILED"),
				"processedCount", 0,
				"failedCount", 1,
				"errorCode", "FILE_DELETE_FAILED",
			)
			continue
		}
		if err := svc.S.ClearExportArtifact(ctx, sc, store.StrOrEmpty(export["id"])); err != nil {
			return nil, err
		}
	}
	processed := int64(0)
	for _, table := range cleanupTables {
		n, err := svc.S.DeleteEventRows(ctx, sc, table, event)
		if err != nil {
			return nil, err
		}
		processed += n
	}
	if err := svc.S.ClearGrantDisclosure(ctx, sc, event); err != nil {
		return nil, err
	}
	if filesDeleted {
		if _, err := svc.S.DeleteEventRows(ctx, sc, "ug_export", event); err != nil {
			return nil, err
		}
		if _, err := svc.S.DeleteEventRows(ctx, sc, "ug_job", event); err != nil {
			return nil, err
		}
	} else {
		blank, err := svc.S.Crypto.Seal("{}")
		if err != nil {
			return nil, err
		}
		if err := svc.S.ExpireExportArtifacts(ctx, sc, blank, event); err != nil {
			return nil, err
		}
		if err := svc.S.ExpireEventJobs(ctx, sc, event); err != nil {
			return nil, err
		}
	}
	// The currently executing CleanupRetry operation is retained until the
	// dispatcher stores its result.
	if err := svc.S.DeleteNonRetryOperations(ctx, sc, event); err != nil {
		return nil, err
	}
	if err := svc.S.ScrubOperationResults(ctx, sc, event); err != nil {
		return nil, err
	}
	if err := svc.S.RevokeEventKey(sc, event); err != nil {
		return nil, err
	}
	scopes[5] = store.Map(
		"scope", svc.value("CleanupScope", "BACKUPS"),
		"status", svc.value("JobStatus", "FAILED"),
		"processedCount", 0,
		"failedCount", 1,
		"errorCode", "BACKUP_COVERAGE_UNVERIFIED",
	)
	failed := 0
	for _, y := range scopes {
		if store.StrOrEmpty(store.Parse(store.Parse(y)["status"])["code"]) == "FAILED" {
			failed++
		}
	}
	scopeJSON, err := store.Stringify(scopes)
	if err != nil {
		return nil, err
	}
	if err := svc.S.UpdateCleanupFailed(ctx, sc, scopeJSON, int(processed), failed, store.StrOrEmpty(x["id"])); err != nil {
		return nil, err
	}
	now := svc.S.Now()
	if err := svc.S.UpsertDeletionLedger(ctx, sc, svc.S.ID(), event, due, "event:"+event, now, now, now); err != nil {
		return nil, err
	}
	j, err := svc.newJob(ctx, sc, event, "CLEANUP",
		"cleanup:"+store.StrOrEmpty(x["id"])+":"+store.StrOrEmpty(x["version"]),
		"FAILED", store.Map(), "BACKUP_COVERAGE_UNVERIFIED", x["id"], nil)
	if err != nil {
		return nil, err
	}
	return svc.jobView(j), nil
}

// deleteManagedArtifact removes a file only when it stays inside the managed
// storage directory (path traversal guard shared with export download).
func deleteManagedArtifact(storage, key string) error {
	p := filepath.Clean(filepath.Join(storage, key))
	if p != storage && !strings.HasPrefix(p, storage+string(os.PathSeparator)) {
		return fmt.Errorf("invalid managed artifact")
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete artifact: %w", err)
	}
	return nil
}
