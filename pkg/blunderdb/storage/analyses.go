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
	// The columns are a projection of `data`, which stays intact, so a bug in
	// the projection is repairable without re-importing anything.
	//
	// An explicit operation, NOT a schema migration: opening a database must
	// not rewrite its analysis columns behind the user's back.
	RepairDenormalisedColumns(ctx context.Context, scope string) (int, error)

	// WithoutAnalysis streams the positions in scope that carry no analysis at
	// all, by ascending id, bounded by opts.
	//
	// One join instead of a Load per position (the daemon's catch-up sweep).
	//
	// A stream, not a snapshot. The caller must NOT write analyses while
	// reading it, or the result would depend on the backend's isolation: drain
	// it first. A fresh call finds whatever is still missing (ADR-0013 resume).
	WithoutAnalysis(ctx context.Context, scope string, opts ListOpts) iter.Seq2[*domain.Position, error]
}
