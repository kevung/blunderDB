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
}
