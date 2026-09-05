package storage

import (
	"context"
	"iter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// AnalysisStore persists the engine analysis attached to a position. The
// backend transparently compresses/decompresses the analysis payload; callers
// always see a decoded *domain.PositionAnalysis.
type AnalysisStore interface {
	// Save stores (or replaces) the analysis for positionID.
	Save(ctx context.Context, scope string, positionID int64, a *domain.PositionAnalysis) error

	// LoadMany decodes the analyses of the given positions, keyed by position
	// id, in one round trip per batch. A position without an analysis has no
	// entry. A stored payload that cannot be decoded has no entry either, and
	// is logged: one corrupt row must not block reading everything else, and
	// the caller of a batch — an export, a listing — has nothing to do about
	// it but leave it out.
	LoadMany(ctx context.Context, scope string, ids []int64) (map[int64]*domain.PositionAnalysis, error)

	// Load returns the analysis for positionID, or ErrNotFound.
	Load(ctx context.Context, scope string, positionID int64) (*domain.PositionAnalysis, error)

	// Delete removes the analysis for positionID.
	Delete(ctx context.Context, scope string, positionID int64) error

	// RepairDenormalisedColumns recomputes the scalar columns of every analysis
	// in scope from its stored JSON, and returns how many rows actually changed.
	//
	// The columns are a projection of `data`, which stays intact — so a bug in
	// the projection is repairable without re-importing anything. It has already
	// been needed once: the XG importer writes a no-double as BOTH "No Double"
	// and "Double No", and the latter used to be read as a DOUBLE, giving the
	// column the error of the double that never happened (kevung/blunderDB#115).
	// Fixing the reader did nothing for rows already written.
	//
	// It is a deliberate, explicit operation and NOT a schema migration: the
	// schema is unchanged, and rewriting every user's analysis columns on the
	// mere act of opening a database is not something a tool should do behind
	// their back.
	RepairDenormalisedColumns(ctx context.Context, scope string) (int, error)

	// WithoutAnalysis streams the positions in scope that carry no analysis at
	// all, by ascending id, bounded by opts.
	//
	// It exists because the alternative is a query per position: the daemon's
	// catch-up sweep used to list every position, then ask Load about each one
	// and keep the ErrNotFound ones (G.11, #239). On a library of any size that
	// is one round trip per row to learn a fact the database can state in one
	// join — and it had to materialise the whole library first, because the
	// SQLite pool is a single connection and a second query cannot run while
	// the first still holds its rows open.
	//
	// A stream, not a snapshot: the caller decides how much of it to hold. What
	// it must NOT do is write analyses while reading it — a position the sweep
	// has just filled would otherwise be a row the cursor has yet to reach, and
	// what the query means would depend on the backend's isolation. The sweep
	// drains it first, deliberately, which is also the resume mechanism
	// ADR-0013 asks for: a fresh call finds whatever is still missing.
	WithoutAnalysis(ctx context.Context, scope string, opts ListOpts) iter.Seq2[*domain.Position, error]
}
