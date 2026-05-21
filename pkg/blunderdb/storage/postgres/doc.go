// Package postgres is the PostgreSQL-backed implementation of storage.Storage.
//
// It is the backend for blunderDB's server mode (`blunderdb serve`), where a
// single shared schema holds many tenants. SQLite stays the default backend
// for the desktop GUI and CLI; PostgreSQL is selected explicitly via the
// `postgres://` DSN scheme.
//
// Driver. github.com/jackc/pgx/v5 with pgxpool — pure Go (no CGO, like the
// modernc.org/sqlite driver), a built-in bounded connection pool, and
// first-class BYTEA support for the compressed analysis payloads.
//
// Multi-tenancy. Every domain table carries a tenant_id BIGINT NOT NULL
// column. The scope argument threaded through every Storage method carries
// the tenant identifier; application code filters by tenant_id on every
// query. Per-tenant Zobrist dedup is enforced by the composite unique index
// (tenant_id, zobrist_hash). See migrations/README.md and RLS.md.
//
// Schema. PostgreSQL starts fresh at the terminal SQLite schema (v2.7.0); the
// historical SQLite migration chain is not ported. The schema DDL lives in
// migrations/001_initial_v2_7_0.sql, embedded into the binary.
//
// Transaction isolation. Transactions use PostgreSQL's default READ COMMITTED
// level. The aggregate stats queries tolerate this; nothing in the backend
// relies on a stricter level.
//
// PR status. This package currently provides the P3 PR1 skeleton: Open,
// Close, the pool, schema bootstrap and transactions work; the 14 storage
// families are stubbed and return storage.ErrInternal wrapped with a
// "not implemented" message. Each family is implemented in a later P3 PR.
package postgres
