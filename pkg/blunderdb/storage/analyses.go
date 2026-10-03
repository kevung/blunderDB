package storage

import (
	"context"
	"iter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// PlayedActions is the checker move and cube action played at a decision, as
// the caller knows them. An empty field is a known "nothing played".
type PlayedActions struct {
	CheckerMove string
	CubeAction  string
}

// AnalysisStore persists the engine analysis attached to a position. The
// backend transparently compresses/decompresses the analysis payload; callers
// always see a decoded *domain.PositionAnalysis.
type AnalysisStore interface {
	// Save stores (or replaces) the analysis for positionID.
	Save(ctx context.Context, scope string, positionID int64, a *domain.PositionAnalysis) error

	// Merge reads the stored analysis of positionID once, hands it to merge
	// (nil when there is none) and saves what merge returns, rounded and
	// encoded as Save would. When the result equals the stored analysis apart
	// from LastModifiedDate, and the played-action columns derived from it are
	// unchanged, nothing is written: re-importing what is already there costs
	// a read, not a rewrite. merge may mutate its argument; returning nil
	// writes nothing. Reports whether a row was written.
	//
	// played is what the caller already knows was played at the position —
	// the decision of the match being imported. The columns take it where the
	// analysis names no played action, without reading the move table; nil
	// (a position imported on its own) reads the match's record as Save does.
	Merge(ctx context.Context, scope string, positionID int64, played *PlayedActions, merge func(existing *domain.PositionAnalysis) *domain.PositionAnalysis) (bool, error)

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
