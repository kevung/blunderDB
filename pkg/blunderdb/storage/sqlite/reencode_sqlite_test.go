package sqlite_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/storagetest"
)

// TestReencodeAnalysesUpgradesLegacy_SQLite runs the shared legacy check,
// planting blobs through a second connection to the same file.
func TestReencodeAnalysesUpgradesLegacy_SQLite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reencode.db")
	s, err := sqlite.Open(context.Background(), path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	storagetest.CheckReencodeUpgradesLegacy(t, s, func(id int64, blob []byte) {
		if _, err := raw.Exec(`UPDATE analysis SET data = ? WHERE position_id = ?`, blob, id); err != nil {
			t.Fatal(err)
		}
	})
}
