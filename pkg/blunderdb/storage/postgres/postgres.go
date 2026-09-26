package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Environment variables overriding the connection-pool defaults below.
// Each is a Go duration string ("5s", "30m", "1h") except maxConnsEnv and
// minConnsEnv, plain integers; absent or unparsable, a setting keeps its
// default.
const (
	maxConnsEnv          = "BLUNDERDB_POSTGRES_MAX_CONNS"
	minConnsEnv          = "BLUNDERDB_POSTGRES_MIN_CONNS"
	maxConnLifetimeEnv   = "BLUNDERDB_POSTGRES_MAX_CONN_LIFETIME"
	healthCheckPeriodEnv = "BLUNDERDB_POSTGRES_HEALTH_CHECK_PERIOD"
	connectTimeoutEnv    = "BLUNDERDB_POSTGRES_CONNECT_TIMEOUT"
	maxConnIdleTimeEnv   = "BLUNDERDB_POSTGRES_MAX_CONN_IDLE_TIME"
)

// Connection-pool defaults, each overridable via the env vars above.
// connectTimeout makes an unreachable database fail fast rather than wait for
// the OS TCP timeout; maxConnIdleTime shrinks the pool back after a spike.
const (
	defaultMaxConns        = 50
	defaultMinConns        = 5
	defaultMaxConnLifetime = time.Hour
	defaultHealthCheck     = 30 * time.Second
	defaultConnectTimeout  = 5 * time.Second
	defaultMaxConnIdleTime = 30 * time.Minute
)

// Storage is the PostgreSQL implementation of storage.Storage.
type Storage struct {
	binder
	pool *pgxpool.Pool
}

var _ storage.Storage = (*Storage)(nil)

// Open connects to the PostgreSQL database at dsn (a postgres:// URL or
// libpq key/value string), establishes a bounded connection pool, and
// bootstraps the v2.7.0 schema if the database is empty. The returned Storage
// owns the pool: Close shuts it down.
func Open(ctx context.Context, dsn string, opts *storage.Options) (*Storage, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: parse dsn: %w", err)
	}
	cfg.MaxConns = int32(intFromEnv(maxConnsEnv, defaultMaxConns))
	cfg.MinConns = int32(intFromEnv(minConnsEnv, defaultMinConns))
	cfg.MaxConnLifetime = durationFromEnv(maxConnLifetimeEnv, defaultMaxConnLifetime)
	cfg.HealthCheckPeriod = durationFromEnv(healthCheckPeriodEnv, defaultHealthCheck)
	cfg.MaxConnIdleTime = durationFromEnv(maxConnIdleTimeEnv, defaultMaxConnIdleTime)
	cfg.ConnConfig.ConnectTimeout = durationFromEnv(connectTimeoutEnv, defaultConnectTimeout)

	if opts != nil && opts.EnableRLS {
		configureRLSPool(cfg)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("postgres: open pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping: %w", err)
	}

	s := &Storage{binder: binder{db: pool}, pool: pool}
	if err := s.Migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return s, nil
}

// intFromEnv reads a positive integer from the named environment variable,
// falling back to def when it is unset, non-numeric, or not positive.
func intFromEnv(name string, def int) int {
	if v := os.Getenv(name); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

// durationFromEnv reads a Go duration string ("5s", "30m", "1h") from the
// named environment variable, falling back to def when it is unset, does not
// parse, or is not positive.
func durationFromEnv(name string, def time.Duration) time.Duration {
	if v := os.Getenv(name); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return def
}

// PoolStats reports a snapshot of the connection pool: acquired, idle, the
// configured maximum, and pgxpool's cumulative EmptyAcquireCount. It feeds
// blunderdb_pg_pool_* (Server.sweepPoolStats); SQLite has no equivalent.
func (s *Storage) PoolStats() (acquired, idle, max int32, waitCount int64) {
	stat := s.pool.Stat()
	return stat.AcquiredConns(), stat.IdleConns(), stat.MaxConns(), stat.EmptyAcquireCount()
}

// DatabaseSizeBytes reports the current database's on-disk size in bytes via
// pg_database_size(current_database()), for the whole instance (every
// tenant), feeding the daemon's blunderdb_database_size_bytes gauge.
func (s *Storage) DatabaseSizeBytes(ctx context.Context) (int64, error) {
	var n int64
	err := s.pool.QueryRow(ctx, `SELECT pg_database_size(current_database())`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("postgres: database size: %w", err)
	}
	return n, nil
}

// Close shuts the connection pool down.
func (s *Storage) Close() error {
	if s.pool != nil {
		s.pool.Close()
	}
	return nil
}

// BeginTx starts a transaction whose family accessors run inside it. The
// transaction uses PostgreSQL's default READ COMMITTED isolation level.
func (s *Storage) BeginTx(ctx context.Context) (storage.Tx, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: begin tx: %w", err)
	}
	return &txImpl{binder: binder{db: tx}, tx: tx, ctx: ctx}, nil
}

// Version reports the schema version recorded in the metadata table.
func (s *Storage) Version(ctx context.Context) (string, error) {
	var v string
	err := s.pool.QueryRow(ctx,
		`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("postgres: database version: %w", storage.ErrNotFound)
	}
	if err != nil {
		return "", fmt.Errorf("postgres: database version: %w", err)
	}
	return v, nil
}

// migrationLockKey is the fixed pg_advisory_lock key guarding Migrate, stable
// across builds (the ASCII bytes of "_migrat" read big-endian), so concurrent
// migrators serialize. Session-scoped, not _xact: some migrations are
// multi-statement batches that cannot share one explicit transaction.
const migrationLockKey int64 = 0x5f6d6967726174 // "_migrat"

// Migrate brings the database up to the current schema version. A fresh
// database is bootstrapped to the v2.7.0 baseline; then any forward
// migrations (002+) not yet recorded in schema_migrations are applied;
// finally database_version is (re)written from domain.DatabaseVersion in one
// place, never by an individual migration file.
//
// The whole sequence runs on one connection acquired for the duration, held
// under a session-level pg_advisory_lock: two processes calling Migrate at
// once (two daemon replicas starting together, or the daemon racing a
// `blunderdb migrate` invocation) serialize instead of racing the same DDL.
func (s *Storage) Migrate(ctx context.Context) error {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("postgres: acquire migration connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, migrationLockKey); err != nil {
		return fmt.Errorf("postgres: acquire migration lock: %w", err)
	}
	defer func() {
		if _, err := conn.Exec(context.WithoutCancel(ctx), `SELECT pg_advisory_unlock($1)`, migrationLockKey); err != nil {
			// The lock is session-scoped: it is released when the connection
			// closes even if this Exec itself fails or ctx is already done, so
			// this is not a leak — just unable to log a clean release.
			_ = err
		}
	}()

	fresh, err := isFreshDB(ctx, conn)
	if err != nil {
		return err
	}
	if fresh {
		if err := bootstrap(ctx, conn); err != nil {
			return err
		}
	}
	if err := migrateForward(ctx, conn); err != nil {
		return err
	}
	return setDatabaseVersion(ctx, conn)
}

// isFreshDB reports whether the database has no schema yet (no metadata
// table). to_regclass returns NULL for a relation that does not exist.
func isFreshDB(ctx context.Context, db execer) (bool, error) {
	var reg *string
	if err := db.QueryRow(ctx,
		`SELECT to_regclass('public.metadata')`).Scan(&reg); err != nil {
		return false, fmt.Errorf("postgres: probe schema: %w", err)
	}
	return reg == nil, nil
}

// setDatabaseVersion (re)writes the metadata row Version reads, from the
// Go constant — the one place Migrate names a version, after bootstrap and
// every forward migration have both already succeeded.
func setDatabaseVersion(ctx context.Context, db execer) error {
	if _, err := db.Exec(ctx,
		`INSERT INTO metadata (key, value) VALUES ('database_version', $1)
		 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`,
		domain.DatabaseVersion); err != nil {
		return fmt.Errorf("postgres: set database_version: %w", err)
	}
	return nil
}
