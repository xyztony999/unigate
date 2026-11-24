package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"unigate/backend/internal/errcode"
	"unigate/backend/internal/store"
)

// This file reproduces com.unigate.SourcePullWorker and LifecycleWorker.
// Jobs are claimed with SKIP LOCKED on the event row; the provider network
// request never runs inside a database transaction or event lock. Pull
// failures update job state only and never touch business eligibility.

// runPollLoop runs tick on a fixed delay until ctx is cancelled. The first
// tick fires immediately (Spring @Scheduled fixedDelay startup behaviour).
// Workers expose Run(ctx); main owns the goroutine and its cancellation.
func runPollLoop(ctx context.Context, name string, interval time.Duration, tick func(context.Context) error) {
	slog.InfoContext(ctx, name+" started", "pollMs", interval.Milliseconds())
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, name+" stopped")
			return
		case <-timer.C:
		}
		if err := tick(ctx); err != nil {
			slog.ErrorContext(ctx, name+" tick failed", "error", err)
		}
		timer.Reset(interval)
	}
}

// pullClaim is the leased pull unit (SourcePullWorker.Claim).
type pullClaim struct {
	job        map[string]any
	source     map[string]any
	connection map[string]any
	errCode    string
}

// SourcePullWorker drains PULL jobs (Feishu/Tencent source previews).
type SourcePullWorker struct {
	svc *Service
}

// NewSourcePullWorker wires the pull worker.
func NewSourcePullWorker(svc *Service) *SourcePullWorker {
	return &SourcePullWorker{svc: svc}
}

// Run polls until ctx is cancelled.
func (w *SourcePullWorker) Run(ctx context.Context) {
	runPollLoop(ctx, "source pull worker", pollInterval(w.svc.Cfg.SourcesPollMS, 3000), w.pullNext)
}

// PullNext processes a single PULL job tick. Run polls it on a fixed delay;
// integration tests drive it directly like the Java pullNext() call.
func (w *SourcePullWorker) PullNext(ctx context.Context) error {
	return w.pullNext(ctx)
}

// pullNext processes one PULL job tick (SourcePullWorker.pullNext).
func (w *SourcePullWorker) pullNext(ctx context.Context) error {
	svc := w.svc
	if err := svc.independentTx(ctx, func(sc *store.Scope) error {
		if err := svc.S.ExpireJobs(ctx, sc, "PULL"); err != nil {
			return err
		}
		return svc.S.FailExhaustedJobs(ctx, sc, "PULL")
	}); err != nil {
		return err
	}
	var claim *pullClaim
	if err := svc.independentTx(ctx, func(sc *store.Scope) error {
		c, err := svc.claimPullJob(ctx, sc)
		if err != nil {
			return err
		}
		claim = c
		return nil
	}); err != nil {
		return err
	}
	if claim == nil {
		return nil
	}

	// Provider call runs after the claim transaction committed: no database
	// transaction or event lock is held during the network request.
	records := []RemoteRow{}
	failure := claim.errCode
	if failure == "" {
		var err error
		records, err = svc.providerPull(ctx, claim.connection, store.StrOrEmpty(claim.source["resource_id"]))
		if err == nil && len(records) == 0 {
			err = errcode.New(422, "SOURCE_EMPTY", "来源没有可预览记录")
		}
		if err != nil {
			failure = providerErrorCode(err)
		}
	}
	return svc.independentTx(ctx, func(sc *store.Scope) error {
		return svc.finishPullJob(ctx, sc, claim, records, failure)
	})
}

// claimPullJob leases the next ready PULL job and resolves source/connection.
func (svc *Service) claimPullJob(ctx context.Context, sc *store.Scope) (*pullClaim, error) {
	ready, err := svc.S.ClaimPullJob(ctx, sc)
	if err != nil {
		return nil, err
	}
	if len(ready) == 0 {
		return nil, nil
	}
	previous, err := svc.S.JobByIDForUpdate(ctx, sc, store.StrOrEmpty(ready[0]["id"]))
	if err != nil {
		return nil, err
	}
	now := svc.S.Now()
	pending := false
	if store.StrOrEmpty(previous["status"]) == "PENDING" {
		nextRun, err := store.Time(previous["next_run_at"])
		if err != nil {
			return nil, fmt.Errorf("job next run: %w", err)
		}
		pending = !nextRun.After(now)
	}
	abandoned := false
	if store.StrOrEmpty(previous["status"]) == "RUNNING" && previous["lease_until"] != nil {
		lease, err := store.Time(previous["lease_until"])
		if err != nil {
			return nil, fmt.Errorf("job lease: %w", err)
		}
		abandoned = !lease.After(now)
	}
	expires, err := store.Time(previous["expires_at"])
	if err != nil {
		return nil, fmt.Errorf("job expiry: %w", err)
	}
	if store.Num(previous["attempts"]) >= 5 || (!pending && !abandoned) || !expires.After(now) {
		return nil, nil
	}
	if err := svc.S.LeaseJob(ctx, sc, now.Add(10*time.Minute), store.StrOrEmpty(previous["id"])); err != nil {
		return nil, err
	}
	job, err := svc.S.JobByID(ctx, sc, store.StrOrEmpty(previous["id"]))
	if err != nil {
		return nil, err
	}
	claim := &pullClaim{job: job}
	if err := svc.buildPullClaim(ctx, sc, claim); err != nil {
		claim.errCode = providerErrorCode(err)
	}
	return claim, nil
}

// buildPullClaim validates actor/source/connection of a claimed job; failures
// are captured as the claim error code instead of aborting the lease.
func (svc *Service) buildPullClaim(ctx context.Context, sc *store.Scope, claim *pullClaim) error {
	payload, err := svc.S.Unseal(claim.job["payload_cipher"])
	if err != nil {
		return err
	}
	event, err := svc.S.Event(ctx, sc, store.StrOrEmpty(claim.job["event_id"]))
	if err != nil {
		return err
	}
	if err := svc.checkPullActor(ctx, sc, payload, event); err != nil {
		return err
	}
	source, err := svc.S.SourceByIDEvent(ctx, sc, store.StrOrEmpty(payload["sourceId"]), store.StrOrEmpty(event["id"]))
	if err != nil {
		return err
	}
	if err := errcode.Require(store.Num(source["version"]) == store.Num(payload["sourceVersion"]),
		409, "PREVIEW_CONFLICT", "来源已变化"); err != nil {
		return err
	}
	connection, err := svc.pullConnection(ctx, sc, source, event)
	if err != nil {
		return err
	}
	claim.source = source
	claim.connection = connection
	return nil
}

// checkPullActor freezes pulls after check-in starts and re-verifies the
// original organizer's grant (SourcePullWorker.checkActor).
func (svc *Service) checkPullActor(ctx context.Context, sc *store.Scope, payload, event map[string]any) error {
	status := store.StrOrEmpty(event["status"])
	if err := errcode.Require(status == "DRAFT" || status == "READY",
		409, "EVENT_STATE_CONFLICT", "检票开始后来源拉取已冻结"); err != nil {
		return err
	}
	grants, err := svc.S.ActiveOrganizerGrant(ctx, sc, store.StrOrEmpty(event["id"]), store.StrOrEmpty(payload["actorId"]))
	if err != nil {
		return err
	}
	if err := errcode.Require(len(grants) > 0, 403, "FORBIDDEN", "原负责人授权已失效"); err != nil {
		return err
	}
	return nil
}

// pullConnection resolves the event-assigned connection and rejects dead ones.
func (svc *Service) pullConnection(ctx context.Context, sc *store.Scope, source, event map[string]any) (map[string]any, error) {
	connection, err := svc.S.ConnectionAllowed(ctx, sc, store.StrOrEmpty(source["connection_id"]), store.StrOrEmpty(event["id"]))
	if err != nil {
		return nil, err
	}
	status := store.StrOrEmpty(connection["status"])
	if err := errcode.Require(status != "REVOKED" && status != "EXPIRED",
		503, "PROVIDER_UNAVAILABLE", "平台连接已失效"); err != nil {
		return nil, err
	}
	return connection, nil
}

// finishPullJob records the pull outcome under the event lock with the
// optimistic version guard, re-validating actor/source/connection versions.
func (svc *Service) finishPullJob(ctx context.Context, sc *store.Scope, claim *pullClaim, records []RemoteRow, failure string) error {
	event, err := svc.S.EventByIDForUpdate(ctx, sc, store.StrOrEmpty(claim.job["event_id"]))
	if err != nil {
		return err
	}
	jobs, err := svc.S.JobByIDForUpdateRows(ctx, sc, store.StrOrEmpty(claim.job["id"]))
	if err != nil {
		return err
	}
	if len(jobs) == 0 {
		return nil
	}
	job := jobs[0]
	if store.StrOrEmpty(job["status"]) != "RUNNING" || store.Num(job["version"]) != store.Num(claim.job["version"]) {
		return nil
	}
	now := svc.S.Now()
	jobExpires, err := store.Time(job["expires_at"])
	if err != nil {
		return fmt.Errorf("job expiry: %w", err)
	}
	due, err := store.Time(event["personal_due_at"])
	if err != nil {
		return fmt.Errorf("event due time: %w", err)
	}
	if !jobExpires.After(now) || !due.After(now) || store.StrOrEmpty(event["status"]) == "ARCHIVED" {
		return svc.S.ExpireClaimedJob(ctx, sc, now, store.StrOrEmpty(job["id"]))
	}
	sc.EventID = store.StrOrEmpty(event["id"])

	source := claim.source
	if failure == "" {
		source, err = svc.revalidatePull(ctx, sc, claim, job, event)
		if err != nil {
			failure = providerErrorCode(err)
		}
	}
	if failure != "" {
		// Only transient provider failures retry; business conflicts fail the
		// job immediately. Eligibility data is never modified here.
		retry := failure == "PROVIDER_UNAVAILABLE" && store.Num(job["attempts"]) < 5
		delay := time.Duration(1<<min(3, max(0, store.Num(job["attempts"])-1))) * time.Minute
		status := "FAILED"
		var completedAt any = now
		if retry {
			status = "PENDING"
			completedAt = nil
		}
		return svc.S.FailOrRetryPullJob(ctx, sc, status, failure, now.Add(delay), completedAt, store.StrOrEmpty(job["id"]))
	}

	batchID := svc.S.ID()
	mappingJSON, err := store.Stringify(source["mapping"])
	if err != nil {
		return err
	}
	if err := svc.S.InsertPullBatch(ctx, sc, batchID, store.StrOrEmpty(event["id"]), store.StrOrEmpty(source["id"]),
		mappingJSON, store.StrOrEmpty(source["eligibility"]), source["version"], jobExpires); err != nil {
		return err
	}
	line := 0
	for _, record := range records {
		line++
		rowCipher, err := svc.S.Seal(sc, store.Map("raw", record.Fields))
		if err != nil {
			return err
		}
		text, err := store.Stringify(record.Fields)
		if err != nil {
			return err
		}
		if err := svc.S.InsertImportRow(ctx, sc, svc.S.ID(), store.StrOrEmpty(event["id"]), batchID,
			record.ID, line, rowCipher, svc.S.Crypto.Hash(text)); err != nil {
			return err
		}
	}
	return svc.S.SucceedPullJob(ctx, sc, batchID, now, store.StrOrEmpty(job["id"]))
}

// revalidatePull re-checks actor/source/connection under lock before writing
// the batch; it returns the locked source row used for the import snapshot.
func (svc *Service) revalidatePull(ctx context.Context, sc *store.Scope, claim *pullClaim, job, event map[string]any) (map[string]any, error) {
	payload, err := svc.S.Unseal(job["payload_cipher"])
	if err != nil {
		return nil, err
	}
	if err := svc.checkPullActor(ctx, sc, payload, event); err != nil {
		return nil, err
	}
	source, err := svc.S.SourceByIDEventForUpdate(ctx, sc, store.StrOrEmpty(claim.source["id"]), store.StrOrEmpty(event["id"]))
	if err != nil {
		return nil, err
	}
	if err := errcode.Require(store.Num(source["version"]) == store.Num(claim.source["version"]),
		409, "PREVIEW_CONFLICT", "来源已变化"); err != nil {
		return nil, err
	}
	current, err := svc.pullConnection(ctx, sc, source, event)
	if err != nil {
		return nil, err
	}
	if err := errcode.Require(store.Num(current["version"]) == store.Num(claim.connection["version"]),
		409, "PREVIEW_CONFLICT", "平台连接已变化"); err != nil {
		return nil, err
	}
	return source, nil
}

// CleanupRunner is the integration cleanup entry point
// (IntegrationService.runCleanup). Main wires the implementation once the
// integration module is ported; the worker stays decoupled from it.
type CleanupRunner interface {
	RunCleanup(ctx context.Context, sc *store.Scope, cleanup map[string]any) error
}

// LifecycleWorker drains due cleanup records (LifecycleWorker.expireNext).
type LifecycleWorker struct {
	svc     *Service
	cleanup CleanupRunner
}

// NewLifecycleWorker wires the lifecycle worker with its cleanup runner.
func NewLifecycleWorker(svc *Service, cleanup CleanupRunner) *LifecycleWorker {
	return &LifecycleWorker{svc: svc, cleanup: cleanup}
}

// Run polls until ctx is cancelled. Without a wired cleanup runner it exits
// immediately instead of silently skipping due cleanups.
func (w *LifecycleWorker) Run(ctx context.Context) {
	if w.cleanup == nil {
		slog.ErrorContext(ctx, "lifecycle worker not started: cleanup runner not wired")
		return
	}
	runPollLoop(ctx, "lifecycle worker", pollInterval(w.svc.Cfg.LifecyclePollMS, 60000), w.expireNext)
}

// expireNext claims one due cleanup and runs it inside a single transaction
// (the Java method is @Transactional end to end).
func (w *LifecycleWorker) expireNext(ctx context.Context) error {
	sc, err := w.svc.S.Begin(ctx, store.TxOptions{})
	if err != nil {
		return err
	}
	defer w.svc.S.Rollback(ctx, sc)
	due, err := w.svc.S.DueCleanup(ctx, sc)
	if err != nil {
		return err
	}
	if len(due) > 0 {
		row, err := w.svc.S.CleanupByIDForUpdate(ctx, sc, store.StrOrEmpty(due[0]["id"]), store.StrOrEmpty(due[0]["event_id"]))
		if err != nil {
			return err
		}
		if err := w.cleanup.RunCleanup(ctx, sc, row); err != nil {
			return err
		}
	}
	if err := w.svc.S.Commit(ctx, sc); err != nil {
		return fmt.Errorf("commit lifecycle tx: %w", err)
	}
	return nil
}
