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
	fresh, err := s.isFreshDB(ctx)
	if err != nil {
		pool.Close()
		return nil, err
	}
	if fresh {
		if err := bootstrap(ctx, pool); err != nil {
			pool.Close()
			return nil, err
		}
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

// Migrate brings the database up to the current schema version. A fresh
// database is bootstrapped to the v2.7.0 schema. PostgreSQL tracks its own
// forward migration chain and does not replay the historical SQLite chain
// (see migrations/README.md).
func (s *Storage) Migrate(ctx context.Context) error {
	fresh, err := s.isFreshDB(ctx)
	if err != nil {
		return err
	}
	if fresh {
		return bootstrap(ctx, s.pool)
	}
	return nil
}

// isFreshDB reports whether the database has no schema yet (no metadata
// table). to_regclass returns NULL for a relation that does not exist.
func (s *Storage) isFreshDB(ctx context.Context) (bool, error) {
	var reg *string
	if err := s.pool.QueryRow(ctx,
		`SELECT to_regclass('public.metadata')`).Scan(&reg); err != nil {
		return false, fmt.Errorf("postgres: probe schema: %w", err)
	}
	return reg == nil, nil
}
