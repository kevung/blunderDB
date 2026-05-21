// Package storage defines the persistence contract for the blunderDB engine.
//
// The Storage interface is composed of per-family sub-interfaces. A concrete
// backend (currently SQLite, PostgreSQL later) implements it under a sibling
// sub-package. The Database wrapper kept for the Wails GUI delegates to a
// Storage value.
//
// Design notes (see tasks/headless/02-storage-interface.md):
//   - Every method takes a context.Context for cancellation/deadlines.
//   - Every data method takes a scope string. It is empty ("") today; it
//     becomes the per-tenant identifier once PostgreSQL multi-tenancy lands.
//   - List-style methods return a range-friendly iter.Seq2 so large result
//     sets can be streamed instead of fully materialised.
//   - Save/Update/Delete return (int64, error) or error. SQLite uses
//     LastInsertId internally; PostgreSQL will use RETURNING id — the
//     interface hides the difference.
//
// This file (and the rest of package storage) is the interface scaffolding
// only — no backend implementation lives here. The DTO structs declared
// alongside the sub-interfaces are the storage-layer vocabulary; reconciling
// them with the equivalents currently in package database happens in the
// implementation PRs.
package storage

import "context"

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

// Options configures a backend at open time.
type Options struct {
	// MigrationProgress, if set, is invoked during Migrate to report progress.
	MigrationProgress func(phase string, done, total int)
}

// ListOpts bounds and orders a List query. Zero values mean "no limit" /
// "from the start" / "natural order".
type ListOpts struct {
	Limit  int
	Offset int
}
