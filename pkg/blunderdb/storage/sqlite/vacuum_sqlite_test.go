package sqlite

import (
	"bytes"
	"compress/zlib"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// TestVacuum_ReclaimsSpaceAfterDeletes: a file inflated by deletions shrinks
// back down after Vacuum, the reported sizes agree with the file on disk,
// and the rows that were not deleted survive.
func TestVacuum_ReclaimsSpaceAfterDeletes(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "vacuum.db")
	st, err := Open(ctx, dbPath, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer st.Close()

	res, err := st.sqlDB.Exec(`INSERT INTO position (state) VALUES (?)`, "kept-position")
	if err != nil {
		t.Fatalf("insert survivor: %v", err)
	}
	survivorID, _ := res.LastInsertId()

	blob := strings.Repeat("x", 2000)
	tx, err := st.sqlDB.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	for i := 0; i < 3000; i++ {
		if _, err := tx.Exec(`INSERT INTO position (state) VALUES (?)`, blob); err != nil {
			t.Fatalf("insert padding %d: %v", i, err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit padding: %v", err)
	}
	if _, err := st.sqlDB.Exec(`DELETE FROM position WHERE id != ?`, survivorID); err != nil {
		t.Fatalf("delete padding: %v", err)
	}

	result, err := st.Vacuum(ctx)
	if err != nil {
		t.Fatalf("Vacuum: %v", err)
	}
	if result.SizeBefore == 0 {
		t.Fatal("SizeBefore = 0 for a file-backed database")
	}
	if result.SizeAfter >= result.SizeBefore {
		t.Fatalf("file did not shrink: before=%d after=%d", result.SizeBefore, result.SizeAfter)
	}
	info, err := os.Stat(dbPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Size() != result.SizeAfter {
		t.Errorf("SizeAfter=%d, file is %d bytes", result.SizeAfter, info.Size())
	}

	var gotState string
	if err := st.sqlDB.QueryRow(`SELECT state FROM position WHERE id = ?`, survivorID).Scan(&gotState); err != nil {
		t.Fatalf("select survivor: %v", err)
	}
	if gotState != "kept-position" {
		t.Errorf("survivor state = %q", gotState)
	}
}

// TestVacuum_InMemoryDatabase: no file to size, but VACUUM/ANALYZE still run.
func TestVacuum_InMemoryDatabase(t *testing.T) {
	ctx := context.Background()
	st, err := Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer st.Close()

	result, err := st.Vacuum(ctx)
	if err != nil {
		t.Fatalf("Vacuum: %v", err)
	}
	if result.SizeBefore != 0 || result.SizeAfter != 0 {
		t.Errorf("Vacuum on :memory: reported %+v, want zeros", result)
	}
}

// TestVacuum_RecompressesLegacyAnalysisBlobs pins #180's migration path:
// analysis rows written by every release before this one (raw JSON, or zlib
// — never zstd) are upgraded the first time Vacuum runs over them, read back
// unchanged, and left alone (no rewrite, no wasted work) on a second Vacuum.
func TestVacuum_RecompressesLegacyAnalysisBlobs(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "vacuum_recompress.db")
	st, err := Open(ctx, dbPath, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer st.Close()

	pos1, err := st.sqlDB.Exec(`INSERT INTO position (state) VALUES (?)`, "some-position")
	if err != nil {
		t.Fatalf("insert position: %v", err)
	}
	pos1ID, _ := pos1.LastInsertId()
	pos2, err := st.sqlDB.Exec(`INSERT INTO position (state) VALUES (?)`, "another-position")
	if err != nil {
		t.Fatalf("insert second position: %v", err)
	}
	pos2ID, _ := pos2.LastInsertId()

	rawJSON := []byte(`{"xgid":"raw-json-legacy"}`)
	var zlibBuf bytes.Buffer
	zw := zlib.NewWriter(&zlibBuf)
	if _, err := zw.Write([]byte(`{"xgid":"zlib-legacy"}`)); err != nil {
		t.Fatalf("zlib write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zlib close: %v", err)
	}

	rawID := insertLegacyAnalysis(t, st, pos1ID, rawJSON)
	zlibID := insertLegacyAnalysis(t, st, pos2ID, zlibBuf.Bytes())

	if _, err := st.Vacuum(ctx); err != nil {
		t.Fatalf("Vacuum: %v", err)
	}

	for _, tc := range []struct {
		name string
		id   int64
		want string
	}{
		{"raw JSON row", rawID, "raw-json-legacy"},
		{"zlib row", zlibID, "zlib-legacy"},
	} {
		var data []byte
		if err := st.sqlDB.QueryRow(`SELECT data FROM analysis WHERE id = ?`, tc.id).Scan(&data); err != nil {
			t.Fatalf("%s: select: %v", tc.name, err)
		}
		if engine.NeedsRecompression(data) {
			t.Errorf("%s: still not zstd after Vacuum: %q", tc.name, data[:min(4, len(data))])
		}
		a, err := engine.DecodeAnalysisFromStorage(data)
		if err != nil {
			t.Fatalf("%s: decode after vacuum: %v", tc.name, err)
		}
		if a.XGID != tc.want {
			t.Errorf("%s: XGID = %q, want %q", tc.name, a.XGID, tc.want)
		}
	}

	// A second Vacuum must not touch already-current rows (nothing to
	// recompress: exercises the fast "scan, skip everything" path).
	var beforeSecond []byte
	if err := st.sqlDB.QueryRow(`SELECT data FROM analysis WHERE id = ?`, rawID).Scan(&beforeSecond); err != nil {
		t.Fatalf("re-select before second vacuum: %v", err)
	}
	if _, err := st.Vacuum(ctx); err != nil {
		t.Fatalf("second Vacuum: %v", err)
	}
	var afterSecond []byte
	if err := st.sqlDB.QueryRow(`SELECT data FROM analysis WHERE id = ?`, rawID).Scan(&afterSecond); err != nil {
		t.Fatalf("re-select after second vacuum: %v", err)
	}
	if !bytes.Equal(beforeSecond, afterSecond) {
		t.Errorf("second Vacuum rewrote an already-current row")
	}
}

// insertLegacyAnalysis inserts a bare analysis row (bypassing AnalysisStore.Save,
// which would compress through the current codec) so the stored bytes are
// exactly what the test passed in.
func insertLegacyAnalysis(t *testing.T, st *Storage, positionID int64, data []byte) int64 {
	t.Helper()
	res, err := st.sqlDB.Exec(`INSERT INTO analysis (position_id, data) VALUES (?, ?)`, positionID, data)
	if err != nil {
		t.Fatalf("insert legacy analysis: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

// TestFreeSpaceBytes is a light sanity check on the platform-specific helper:
// the working directory's filesystem must report a nonzero amount of free
// space (Vacuum would read zero as "no room").
func TestFreeSpaceBytes(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	free, err := freeSpaceBytes(wd)
	if err != nil {
		t.Fatalf("freeSpaceBytes(%q): %v", wd, err)
	}
	if free == 0 {
		t.Errorf("freeSpaceBytes(%q) = 0, want > 0", wd)
	}
}
