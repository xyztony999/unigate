package store

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

// TxOptions mirrors the Spring TransactionTemplate settings used by the Java
// controller: READ COMMITTED with a 5s timeout for business operations, and a
// REPEATABLE READ read-only snapshot for Statistics.
type TxOptions struct {
	RepeatableRead bool
	ReadOnly       bool
	Timeout        time.Duration
}

// Begin starts a transaction with the given options and returns a Scope bound
// to it. The caller must Commit or Rollback.
func (s *Store) Begin(ctx context.Context, opts TxOptions) (*Scope, error) {
	txOpts := pgx.TxOptions{}
	if opts.RepeatableRead {
		txOpts.IsoLevel = pgx.RepeatableRead
	}
	if opts.ReadOnly {
		txOpts.AccessMode = pgx.ReadOnly
	}
	tx, err := s.Pool.BeginTx(ctx, txOpts)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	sc := &Scope{Tx: tx, Snapshot: opts.ReadOnly && opts.RepeatableRead}
	if opts.Timeout > 0 {
		ms := opts.Timeout.Milliseconds()
		// SET does not accept bind parameters; set_config(..., true) is the
		// parameterized, transaction-local equivalent of SET LOCAL.
		if _, err := tx.Exec(ctx, "select set_config('statement_timeout', $1, true)", strconv.FormatInt(ms, 10)); err != nil {
			_ = tx.Rollback(context.WithoutCancel(ctx))
			return nil, fmt.Errorf("set statement timeout: %w", err)
		}
	}
	return sc, nil
}

// Commit commits the scope's transaction.
func (s *Store) Commit(ctx context.Context, sc *Scope) error {
	if sc.Tx == nil {
		return nil
	}
	if err := sc.Tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	sc.Tx = nil
	return nil
}

// Rollback aborts the scope's transaction (safe after commit).
func (s *Store) Rollback(ctx context.Context, sc *Scope) {
	if sc.Tx != nil {
		_ = sc.Tx.Rollback(context.WithoutCancel(ctx))
		sc.Tx = nil
	}
}