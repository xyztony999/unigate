package service

import (
	"context"
	"fmt"

	"unigate/backend/internal/errcode"
	"unigate/backend/internal/store"
)

// checkLogin enforces the 5-attempt login window. The counter reads run in the
// caller scope; the writes commit independently (REQUIRES_NEW) so a rejected
// business transaction never rolls back the brute-force counter.
func (svc *Service) checkLogin(ctx context.Context, sc *store.Scope, username, ip string) error {
	scope := svc.loginScope(username, ip)
	count, err := svc.S.LoginChallengeCount(ctx, sc, scope)
	if err != nil {
		return err
	}
	if count >= 5 {
		return errcode.New(429, "RATE_LIMITED", "登录尝试过多，请稍后再试")
	}
	return nil
}

// failedLogin records a failed attempt in its own committed transaction.
func (svc *Service) failedLogin(ctx context.Context, username, ip string) error {
	scope := svc.loginScope(username, ip)
	return svc.independentTx(ctx, func(sc *store.Scope) error {
		return svc.S.InsertLoginChallenge(ctx, sc, svc.S.ID(), scope, svc.S.Crypto.Hash(svc.S.Crypto.Token()))
	})
}

// failedPairing bumps a device pairing failure counter independently.
func (svc *Service) failedPairing(ctx context.Context, id string) error {
	return svc.independentTx(ctx, func(sc *store.Scope) error {
		return svc.S.IncrementPairingFailures(ctx, sc, id)
	})
}

// independentTx runs fn in a fresh READ COMMITTED transaction that commits even
// when the surrounding request transaction later rolls back (REQUIRES_NEW).
func (svc *Service) independentTx(ctx context.Context, fn func(sc *store.Scope) error) error {
	sc, err := svc.S.Begin(ctx, store.TxOptions{})
	if err != nil {
		return err
	}
	defer svc.S.Rollback(ctx, sc)
	if err := fn(sc); err != nil {
		return err
	}
	if err := svc.S.Commit(ctx, sc); err != nil {
		return fmt.Errorf("commit independent tx: %w", err)
	}
	return nil
}