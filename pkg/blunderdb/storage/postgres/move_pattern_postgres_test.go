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
	storagetest.RunMovePatternColumn(t, s, pg.SharedExecer(s), func(ctx context.Context) error {
		return pg.BackfillAnalysisBestMove(ctx, s)
	})
}
