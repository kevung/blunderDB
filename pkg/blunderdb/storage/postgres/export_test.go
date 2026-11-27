//go:build postgres

package postgres

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// SharedExecer exposes a Storage's execer to the external tests that wrap it.
func SharedExecer(s *Storage) sqlshared.Execer { return s.shared() }

// BackfillAnalysisBestMove runs the best-move pass of runGoBackfills.
func BackfillAnalysisBestMove(ctx context.Context, s *Storage) error {
	_, err := backfillAnalysisBestMove(ctx, s.pool)
	return err
}
