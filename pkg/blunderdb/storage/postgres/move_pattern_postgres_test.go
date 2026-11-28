//go:build postgres

package postgres_test

import (
	"context"
	"testing"

	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/storagetest"
)

func TestMovePatternColumn_Postgres(t *testing.T) {
	s, _ := openMatchStore(t)
	exec := pg.SharedExecer(s)
	storagetest.RunMovePatternColumn(t, s, exec, func(ctx context.Context) error {
		// The tenants 077 lists for the pass, the library being unforced here.
		if _, err := exec.Exec(ctx, `INSERT INTO metadata (key, value)
			SELECT 'best_move_backfill_pending', '0 ' || string_agg(DISTINCT tenant_id::text, ' ') FROM analysis
			ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`); err != nil {
			return err
		}
		_, err := pg.BackfillAnalysisBestMove(ctx, s, 2, 0)
		return err
	})
}
