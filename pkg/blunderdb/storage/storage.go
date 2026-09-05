// Package storage defines the persistence contract for the blunderDB engine.
//
// The Storage interface is composed of per-family sub-interfaces. Two
// concrete backends implement it under sibling sub-packages: storage/sqlite
// (the desktop app and single-user CLI) and storage/postgres (the
// multi-tenant `serve` daemon, with optional row-level security). Both are
// held to the same shared contract suite in storage/storagetest. The
// Database wrapper kept for the Wails GUI delegates to a Storage value.
//
// Design notes (see tasks/headless/02-storage-interface.md):
//   - Every method takes a context.Context for cancellation/deadlines.
//   - Every data method takes a scope string: the tenant identifier
//     (domain term: Tenant, see CONTEXT.md). The desktop app and CLI pass
//     "" — the single implicit tenant of a private database; the serve
//     daemon passes the caller's tenant so rows never cross tenants.
//   - List-style methods return a range-friendly iter.Seq2 so large result
//     sets can be streamed instead of fully materialised.
//   - Save/Update/Delete return (int64, error) or error. SQLite uses
//     LastInsertId internally; PostgreSQL uses RETURNING id — the
//     interface hides the difference.
//
// This file declares the contract only — no backend implementation lives
// here. The DTO structs declared alongside the sub-interfaces are the
// storage-layer vocabulary shared by both backends and the server handlers.
//
// # Concurrency and isolation
//
// A Storage value is safe for concurrent use by multiple goroutines: the
// backends rely on the connection pool (SQLite: *sql.DB with busy_timeout and
// per-DSN PRAGMAs; PostgreSQL: pgxpool), not a process-wide lock (P5). There
// is no global serialization — only the per-operation atomicity each backend's
// statements/transactions provide.
//
// Reads observe committed data with READ COMMITTED semantics: a long-running
// scan (e.g. stats or a full search) no longer blocks writers and may not see
// writes committed after it began. Operations that must be atomic across
// several statements run inside BeginTx. SQLite remains a single writer at a
// time; concurrent writers wait up to busy_timeout for the write lock rather
// than failing with SQLITE_BUSY.
package storage

import (
	"context"
	"fmt"
)

// Stores groups the per-family accessors shared by Storage and Tx.
type Stores interface {
	Positions() PositionStore
	Analyses() AnalysisStore
	Matches() MatchStore
	Comments() CommentStore
	Collections() CollectionStore
	Tournaments() TournamentStore
	Anki() AnkiStore
	Filters() FilterStore
	Session() SessionStore
	Search() SearchStore
	SearchHistory() SearchHistoryStore
	Stats() StatsStore
	History() CommandHistoryStore
	Metadata() MetadataStore
}

// Storage is the root persistence interface implemented by every backend.
type Storage interface {
	Stores

	// BeginTx starts a transaction. The returned Tx exposes the same family
	// accessors; work is visible to the rest of the process only after Commit.
	BeginTx(ctx context.Context) (Tx, error)

	// Close releases the backend's resources.
	Close() error

	// Version reports the schema version recorded in the database.
	Version(ctx context.Context) (string, error)

	// Migrate brings the database up to the current schema version.
	Migrate(ctx context.Context) error
}

// VacuumResult reports the file-size effect of a backend's Vacuum, in bytes.
// Vacuum is a backend capability rather than a Storage method: only the
// SQLite backend has a file to compact (sqlite.Storage.Vacuum), and the
// daemon duck-types it the way it does PurgeTenant, so PostgreSQL needs no
// stub. Both sizes are 0 for an in-memory database.
type VacuumResult struct {
	SizeBefore int64 `json:"sizeBefore"`
	SizeAfter  int64 `json:"sizeAfter"`
}

// Options configures a backend at open time.
type Options struct {
	// MigrationProgress, if set, is invoked during Migrate to report progress.
	MigrationProgress func(phase string, done, total int)

	// EnableRLS turns on PostgreSQL Row-Level Security enforcement: the backend
	// sets the `app.tenant_id` GUC per connection (from WithTenant in the
	// operation context) so the RLS policies filter rows as defence-in-depth.
	// Off by default; ignored by the SQLite backend. The policies themselves are
	// installed by Storage.ApplyRLS (opt-in).
	EnableRLS bool
}

// ListOpts bounds and orders a List query. Zero values mean "no limit" /
// "from the start" / "natural order".
type ListOpts struct {
	Limit  int
	Offset int
}

// SQL renders a bound as a query suffix, with the values INLINED rather than
// bound as parameters.
//
// They are Go ints, so there is nothing to inject, and inlining keeps the
// clause independent of each backend's placeholder numbering — the stores that
// write their own SQL number theirs `$1`, `$2` while the shared ones use `?`,
// and a bound that has to be appended to both cannot use either.
//
// `unboundedLimit` is what the dialect writes for "no limit, but an offset":
// SQLite's grammar requires a LIMIT before OFFSET and documents -1 as
// unbounded; PostgreSQL takes an OFFSET alone.
func (o ListOpts) SQL(unboundedLimit string) string {
	switch {
	case o.Limit <= 0 && o.Offset <= 0:
		return ""
	case o.Limit <= 0:
		return fmt.Sprintf(" %s OFFSET %d", unboundedLimit, o.Offset)
	case o.Offset <= 0:
		return fmt.Sprintf(" LIMIT %d", o.Limit)
	default:
		return fmt.Sprintf(" LIMIT %d OFFSET %d", o.Limit, o.Offset)
	}
}
