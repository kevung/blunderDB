package sqlite

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
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

// TestVacuum_RecompressesLegacyAnalysisBlobs: analysis rows in a legacy codec
// (raw JSON or zlib) are upgraded the first time Vacuum runs over them, read back
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
	pos3, err := st.sqlDB.Exec(`INSERT INTO position (state) VALUES (?)`, "third-position")
	if err != nil {
		t.Fatalf("insert third position: %v", err)
	}
	pos3ID, _ := pos3.LastInsertId()

	zero := 0.0
	full := func(xgid string) domain.PositionAnalysis {
		return domain.PositionAnalysis{
			XGID: xgid, Player1: "Alice", Player2: "Bob",
			DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{PlayerWinChances: 51.2, OpponentWinChances: 48.8},
			CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
				{Move: "8/2 6/2", Equity: 0.25}, {Move: "13/7", Equity: 0.1, EquityError: &zero},
			}},
			PlayedMoves: []string{"8/2 6/2"},
		}
	}
	// What each row must decode to before and after Vacuum.
	wantAnalysis := map[string]domain.PositionAnalysis{
		"raw JSON row": full("raw-json-legacy"), "zlib row": full("zlib-legacy"),
		"binary level-7 row": full("zstd-write-path"),
	}
	rawJSON, err := json.Marshal(ptr(wantAnalysis["raw JSON row"]))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	zlibJSON, err := json.Marshal(ptr(wantAnalysis["zlib row"]))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var zlibBuf bytes.Buffer
	zw := zlib.NewWriter(&zlibBuf)
	if _, err := zw.Write(zlibJSON); err != nil {
		t.Fatalf("zlib write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zlib close: %v", err)
	}

	rawID := insertLegacyAnalysis(t, st, pos1ID, rawJSON)
	zlibID := insertLegacyAnalysis(t, st, pos2ID, zlibBuf.Bytes())
	// A write-path blob is already binary: Vacuum must leave it byte for byte.
	fastJSON, err := json.Marshal(ptr(wantAnalysis["binary level-7 row"]))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	fast, err := engine.CompressAnalysisData(fastJSON)
	if err != nil {
		t.Fatalf("compress: %v", err)
	}
	if engine.NeedsRecompression(fast) {
		t.Fatal("a write-path blob counts as legacy")
	}
	fastID := insertLegacyAnalysis(t, st, pos3ID, fast)

	original := map[string][]byte{"raw JSON row": rawJSON, "zlib row": zlibBuf.Bytes(), "binary level-7 row": fast}

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
		{"binary level-7 row", fastID, "zstd-write-path"},
	} {
		var data []byte
		if err := st.sqlDB.QueryRow(`SELECT data FROM analysis WHERE id = ?`, tc.id).Scan(&data); err != nil {
			t.Fatalf("%s: select: %v", tc.name, err)
		}
		if engine.NeedsRecompression(data) {
			t.Errorf("%s: not re-encoded after Vacuum: %q", tc.name, data[:min(5, len(data))])
		}
		a, err := engine.DecodeAnalysisFromStorage(data)
		if err != nil {
			t.Fatalf("%s: decode after vacuum: %v", tc.name, err)
		}
		if a.XGID != tc.want {
			t.Errorf("%s: XGID = %q, want %q", tc.name, a.XGID, tc.want)
		}
		// The whole analysis, not just its key: checker, cube and moves.
		want := wantAnalysis[tc.name]
		before, err := engine.DecodeAnalysisFromStorage(original[tc.name])
		if err != nil {
			t.Fatalf("%s: decode original: %v", tc.name, err)
		}
		if !reflect.DeepEqual(a, before) || a.CheckerAnalysis == nil || len(a.CheckerAnalysis.Moves) != len(want.CheckerAnalysis.Moves) ||
			a.DoublingCubeAnalysis == nil || len(a.PlayedMoves) != 1 {
			t.Errorf("%s: decodes differently after Vacuum\n got %+v\nwant %+v", tc.name, a, before)
		}
	}

	var fastAfter []byte
	if err := st.sqlDB.QueryRow(`SELECT data FROM analysis WHERE id = ?`, fastID).Scan(&fastAfter); err != nil {
		t.Fatalf("select binary row: %v", err)
	}
	if !bytes.Equal(fast, fastAfter) {
		t.Errorf("Vacuum rewrote a binary blob")
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

// TestVacuum_KeepsPoolTempStoreInMemory: the file-backed temp store VACUUM
// needs lives on a dedicated connection only; the pool keeps temp_store=MEMORY
// for everything else.
func TestVacuum_KeepsPoolTempStoreInMemory(t *testing.T) {
	ctx := context.Background()
	st, err := Open(ctx, filepath.Join(t.TempDir(), "v.db"), nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer st.Close()

	if _, err := st.Vacuum(ctx); err != nil {
		t.Fatalf("Vacuum: %v", err)
	}
	var mode int
	if err := st.sqlDB.QueryRow(`PRAGMA temp_store`).Scan(&mode); err != nil {
		t.Fatalf("PRAGMA temp_store: %v", err)
	}
	if mode != 2 { // 2 = MEMORY
		t.Errorf("pool temp_store = %d after Vacuum, want 2 (MEMORY)", mode)
	}
}

func ptr[T any](v T) *T { return &v }

// TestVacuum_ReportsAFileThatKeptItsSize: when the file on disk is still
// larger than its pages once the VACUUM is done — what Windows produces when
// another connection maps the file and SQLite drops the refused truncation —
// Vacuum says so instead of reporting a success. The refusal cannot be
// produced off Windows, so the size the check reads is inflated.
func TestVacuum_ReportsAFileThatKeptItsSize(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "vacuum.db")
	st, err := Open(ctx, dbPath, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer st.Close()
	if _, err := st.sqlDB.Exec(`INSERT INTO position (state) VALUES ('p')`); err != nil {
		t.Fatal(err)
	}
	st.sizeOf = func(path string) (int64, error) {
		n, err := fileSize(path)
		return n + 4096, err
	}

	res, err := st.Vacuum(ctx)
	if !errors.Is(err, storage.ErrVacuumNotShrunk) {
		t.Fatalf("Vacuum: err = %v, want storage.ErrVacuumNotShrunk", err)
	}
	if res.SizeBefore == 0 || res.SizeAfter == 0 {
		t.Fatalf("sizes not reported with the error: %+v", res)
	}
}

// TestVacuum_FileSizeMatchesItsPages: the check above compares the file with
// page_count*page_size; on a platform that truncates, the two agree after a
// vacuum, so the check never fires on a vacuum that did shrink the file.
func TestVacuum_FileSizeMatchesItsPages(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "vacuum.db")
	st, err := Open(ctx, dbPath, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer st.Close()
	if _, err := st.Vacuum(ctx); err != nil {
		t.Fatalf("Vacuum: %v", err)
	}
	want, err := pagesSize(ctx, st.sqlDB)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := fileSize(dbPath); got != want {
		t.Fatalf("file %d bytes, pages %d", got, want)
	}
}

// TestMmapSize: no connection maps the file on Windows, where a mapping held
// by any connection makes the in-place VACUUM unable to shrink the file.
func TestMmapSize(t *testing.T) {
	if got := mmapSizeFor("windows"); got != "0" {
		t.Errorf("mmap_size on windows = %s, want 0", got)
	}
	if got := mmapSizeFor("linux"); got != "268435456" {
		t.Errorf("mmap_size on linux = %s, want 268435456", got)
	}
	if want := "mmap_size(" + mmapSizeFor(runtime.GOOS) + ")"; !strings.Contains(DSN("x.db"), url.QueryEscape(want)) {
		t.Errorf("DSN does not carry %s: %s", want, DSN("x.db"))
	}
}
