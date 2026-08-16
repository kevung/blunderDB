package storage

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// AnalysisStore persists the engine analysis attached to a position. The
// backend transparently compresses/decompresses the analysis payload; callers
// always see a decoded *domain.PositionAnalysis.
type AnalysisStore interface {
	// Save stores (or replaces) the analysis for positionID.
	Save(ctx context.Context, scope string, positionID int64, a *domain.PositionAnalysis) error

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
}
