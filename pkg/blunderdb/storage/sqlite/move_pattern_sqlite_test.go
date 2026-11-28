package sqlite_test

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/storagetest"
)

func TestMovePatternColumn_SQLite(t *testing.T) {
	s := openTempDB(t)
	storagetest.RunMovePatternColumn(t, s, sqlite.SharedExecer(s), func(ctx context.Context) error {
		_, err := s.Analyses().(interface {
			BackfillBestMove(context.Context) (int, error)
		}).BackfillBestMove(ctx)
		return err
	})
}
