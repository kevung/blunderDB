package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/storagetest"
)

func TestChangeSet_SQLite(t *testing.T) {
	open := func() *sqlite.Storage {
		s, err := sqlite.Open(context.Background(), filepath.Join(t.TempDir(), "lib.db"), nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = s.Close() })
		return s
	}
	ctx := context.Background()
	storagetest.CheckChangeSet(t, open(), ctx, "", open(), ctx, "")
}
