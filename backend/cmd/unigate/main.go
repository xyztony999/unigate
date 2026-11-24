// Command unigate is the UniGate process entry point. It reproduces
// com.unigate.UniGateApplication (SpringApplication.run + @EnableScheduling)
// together with the IdentityService ApplicationRunners bootstrapAccount and
// resetPasswordFor: load env configuration, connect PostgreSQL, migrate,
// wire the service/API layers, bootstrap the first admin, then run the
// background workers and the HTTP server until SIGINT/SIGTERM.
//
// One-shot reset mode (deploy/reset-admin.sh equivalent):
//
//	UNIGATE_RESET_PASSWORD_FOR=<user> unigate --no-web
//	unigate --reset-password-for=<user>
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"unigate/backend/internal/api"
	"unigate/backend/internal/config"
	"unigate/backend/internal/contract"
	"unigate/backend/internal/service"
	"unigate/backend/internal/store"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))
	if err := run(); err != nil {
		slog.Error("unigate failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	s, err := openStore(ctx, cfg)
	if err != nil {
		return err
	}
	defer s.Pool.Close()

	c, err := contract.Load()
	if err != nil {
		return fmt.Errorf("load contract: %w", err)
	}
	svc := service.New(s, c, cfg)

	// bootstrapAccount ApplicationRunner: create the first admin only when no
	// accounts exist; existing accounts are never overwritten.
	if err := svc.BootstrapAccount(ctx); err != nil {
		return fmt.Errorf("bootstrap account: %w", err)
	}

	// resetPasswordFor ApplicationRunner: one-shot non-web mode only
	// (reset-admin.sh). No HTTP server and no workers are started.
	if cfg.ResetMode {
		return runReset(ctx, svc, cfg)
	}
	return serve(ctx, cfg, svc)
}

// serve reproduces the @EnableScheduling worker orchestration plus the
// embedded web server: main owns every worker goroutine, ctx cancellation
// (SIGINT/SIGTERM) stops them, and shutdown drains HTTP before closing the DB.
func serve(ctx context.Context, cfg *config.Config, svc *service.Service) error {
	handler, err := api.NewRouter(svc, cfg)
	if err != nil {
		return fmt.Errorf("build router: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Background workers (Java @Scheduled beans). svc implements CleanupRunner
	// via Service.RunCleanup (integration cleanup), wired into the lifecycle
	// worker exactly once here.
	workers := []struct {
		name string
		run  func(context.Context)
	}{
		{"source pull", service.NewSourcePullWorker(svc).Run},
		{"lifecycle", service.NewLifecycleWorker(svc, svc).Run},
		{"notification", service.NewNotificationWorker(svc).Run},
	}
	// Bind first so a port conflict fails fast before any worker starts.
	addr := fmt.Sprintf(":%d", cfg.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}

	var wg sync.WaitGroup
	for _, w := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.run(ctx)
		}()
	}

	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()
	slog.InfoContext(ctx, "unigate started", "addr", addr)

	var serveFailure error
	select {
	case err := <-serveErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveFailure = fmt.Errorf("http serve: %w", err)
			slog.ErrorContext(ctx, "http server failed", "error", err)
		}
	case <-ctx.Done():
		slog.InfoContext(ctx, "shutdown signal received")
	}

	// Graceful shutdown: stop workers, drain in-flight requests with a
	// timeout fallback, then return (run closes the pool via defer).
	cancel()
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelShutdown()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.WarnContext(ctx, "graceful shutdown incomplete, forcing close", "error", err)
		if cerr := srv.Close(); cerr != nil {
			slog.WarnContext(ctx, "force close failed", "error", cerr)
		}
	}
	waitWorkers(&wg, 10*time.Second)
	slog.InfoContext(ctx, "unigate stopped")
	return serveFailure
}

// waitWorkers bounds the worker drain. The waiter goroutine always exits:
// ctx is already cancelled, so every worker Run returns eventually.
func waitWorkers(wg *sync.WaitGroup, timeout time.Duration) {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(timeout):
		slog.Warn("workers still draining after shutdown timeout", "timeout", timeout.String())
	}
}

// runReset reproduces IdentityService.resetPasswordFor: issue a random
// temporary password, force a change, bump auth_version, revoke all sessions,
// then print the credentials to stdout exactly like the Java runner and exit.
func runReset(ctx context.Context, svc *service.Service, cfg *config.Config) error {
	temporary, err := svc.ResetToTemporaryPassword(ctx, cfg.ResetPasswordFor)
	if err != nil {
		return fmt.Errorf("reset password for %q: %w", cfg.ResetPasswordFor, err)
	}
	fmt.Printf("\n=== UniGate 密码重置 ===\n用户名: %s\n临时密码: %s\n该账号全部会话已失效, 登录后将强制修改密码; 此密码不会再次显示\n",
		cfg.ResetPasswordFor, temporary)
	slog.InfoContext(ctx, "password reset completed", "username", cfg.ResetPasswordFor)
	return nil
}

// dsnCredentials reports whether the DSN explicitly specifies user/password,
// either as URL userinfo (postgres://user:pass@host/db) or keyword/value pairs
// (host=... user=... password=...). cfg.DBURL is already normalized by config.
func dsnCredentials(dsn string) (hasUser, hasPassword bool) {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		if u, err := url.Parse(dsn); err == nil && u.User != nil {
			_, pass := u.User.Password()
			return u.User.Username() != "", pass
		}
		return false, false
	}
	for _, field := range strings.Fields(dsn) {
		key, _, ok := strings.Cut(field, "=")
		if !ok {
			continue
		}
		switch key {
		case "user":
			hasUser = true
		case "password":
			hasPassword = true
		}
	}
	return hasUser, hasPassword
}

// openStore validates the master key (fail-fast), opens the pgx pool and runs
// the embedded migrations (25 tables, store/migrate). The pool is closed on
// every error path so a failed startup leaks no connections.
func openStore(ctx context.Context, cfg *config.Config) (*store.Store, error) {
	crypto, err := store.NewCrypto(cfg.MasterKey)
	if err != nil {
		return nil, fmt.Errorf("master key: %w", err)
	}
	poolCfg, err := pgxpool.ParseConfig(cfg.DBURL)
	if err != nil {
		return nil, fmt.Errorf("parse db url: %w", err)
	}
	// UNIGATE_DB_USER/PASSWORD apply like Spring datasource username/password
	// whenever the DSN carries no explicit credentials (pgx would otherwise
	// silently fall back to the OS user, unlike Hikari).
	hasUser, hasPassword := dsnCredentials(cfg.DBURL)
	if !hasUser {
		poolCfg.ConnConfig.User = cfg.DBUser
	}
	if !hasPassword {
		poolCfg.ConnConfig.Password = cfg.DBPassword
	}
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create db pool: %w", err)
	}
	pingCtx, cancelPing := context.WithTimeout(ctx, 10*time.Second)
	defer cancelPing()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect database: %w", err)
	}
	keys, err := store.NewKeyVault(crypto, cfg.KeyDirectory)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("key vault: %w", err)
	}
	s := &store.Store{Pool: pool, Crypto: crypto, Keys: keys}
	if err := s.Migrate(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}
