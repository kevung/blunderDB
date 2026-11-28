//go:build postgres

package postgres

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// SharedExecer exposes a Storage's execer to the external tests that wrap it.
func SharedExecer(s *Storage) sqlshared.Execer { return s.shared() }

// BackfillAnalysisBestMove runs at most maxBatches batches (0: all) of the
// best-move pass of BackfillDeferred, batch rows each, and reports whether it
// has ended.
func BackfillAnalysisBestMove(ctx context.Context, s *Storage, batch, maxBatches int) (bool, error) {
	was := bestMoveBatch
	bestMoveBatch = batch
	defer func() { bestMoveBatch = was }()
	return backfillAnalysisBestMove(ctx, s.pool, maxBatches)
}
