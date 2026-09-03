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

// maxConnsEnv overrides the connection-pool upper bound. See pool defaults.
const maxConnsEnv = "BLUNDERDB_POSTGRES_MAX_CONNS"

// Connection-pool defaults. MaxConns is overridable via maxConnsEnv.
const (
	defaultMaxConns        = 50
	defaultMinConns        = 5
	defaultMaxConnLifetime = time.Hour
	defaultHealthCheck     = 30 * time.Second
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
	cfg.MaxConns = maxConnsFromEnv()
	cfg.MinConns = defaultMinConns
	cfg.MaxConnLifetime = defaultMaxConnLifetime
	cfg.HealthCheckPeriod = defaultHealthCheck

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

// maxConnsFromEnv reads the pool upper bound from maxConnsEnv, falling back to
// defaultMaxConns when the variable is unset or malformed.
func maxConnsFromEnv() int32 {
	if v := os.Getenv(maxConnsEnv); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return int32(n)
		}
	}
	return defaultMaxConns
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

// migrationLockKey is the fixed pg_advisory_lock key guarding Migrate:
// arbitrary but stable across builds and processes, so two of them (two
// daemon replicas starting together, or the daemon racing a `blunderdb
// migrate` invocation) contending for it serialize rather than run the
// bootstrap/forward-migration sequence concurrently (#231). It is the ASCII
// bytes of "_migrat" (7 bytes, 56 bits) read big-endian as an integer — no
// hashing, so any two builds of this binary agree on it by construction,
// and it is human-verifiable rather than a value someone has to trust.
// Session-scoped (pg_advisory_lock, not the _xact variant): it must survive
// several separate statements run outside one transaction — some migrations
// use DO blocks / multi-statement batches via the simple query protocol,
// which cannot all share a single explicit transaction.
const migrationLockKey int64 = 0x5f6d6967726174 // "_migrat"

// Migrate brings the database up to the current schema version. A fresh
// database is bootstrapped to the v2.7.0 baseline; then any forward
// migrations (002+) not yet recorded in schema_migrations are applied;
// finally database_version is (re)written from domain.DatabaseVersion in one
// place, never by an individual migration file (#231) — a chain interrupted
// partway through used to leave the true (newer) schema declaring an older
// version, which /readyz then reported as a mismatch even though the
// interruption's real effect (a half-applied migration) is exactly what
// /readyz cannot see either way.
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
// every forward migration have both already succeeded (#231).
func setDatabaseVersion(ctx context.Context, db execer) error {
	if _, err := db.Exec(ctx,
		`INSERT INTO metadata (key, value) VALUES ('database_version', $1)
		 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`,
		domain.DatabaseVersion); err != nil {
		return fmt.Errorf("postgres: set database_version: %w", err)
	}
	return nil
}
