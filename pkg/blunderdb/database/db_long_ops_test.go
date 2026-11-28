package database

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/progress"
)

// longOpsLibrary holds two matches, enough for an operation to report more
// than once.
func longOpsLibrary(t *testing.T) *Database {
	t.Helper()
	db := newTestDB(t)
	for _, f := range []string{"match_with_comment.xg", "charlot1-charlot2_7p_2025-11-08-2305.xg"} {
		if _, err := db.ImportXGMatch(filepath.Join("testdata", f)); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
	}
	return db
}

// cancelOnFirstUnit is a context that reports progress and is cancelled as
// soon as one unit is done.
func cancelOnFirstUnit(t *testing.T) (context.Context, *[][2]int) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	var seen [][2]int
	return progress.With(ctx, func(done, total int) {
		seen = append(seen, [2]int{done, total})
		if done >= 1 {
			cancel()
		}
	}), &seen
}

func TestExportDatabaseCtx_ReportsProgress(t *testing.T) {
	t.Parallel()
	db := longOpsLibrary(t)
	var last [2]int
	ctx := progress.With(context.Background(), func(done, total int) { last = [2]int{done, total} })
	path := filepath.Join(t.TempDir(), "out.db")
	rep, err := db.ExportDatabaseCtx(ctx, ExportOptions{ExportPath: path, AllPositions: true, IncludeMatches: true, Metadata: map[string]string{}})
	if err != nil {
		t.Fatal(err)
	}
	if want := rep.Positions + rep.Matches; last[0] != want || last[1] != want || want == 0 {
		t.Fatalf("last report %v, want %d of %d", last, want, want)
	}
}

func TestExportDatabaseCtx_CancelLeavesNoFile(t *testing.T) {
	t.Parallel()
	db := longOpsLibrary(t)
	ctx, _ := cancelOnFirstUnit(t)
	path := filepath.Join(t.TempDir(), "out.db")
	_, err := db.ExportDatabaseCtx(ctx, ExportOptions{ExportPath: path, AllPositions: true, IncludeMatches: true, Metadata: map[string]string{}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("a cancelled export left %s behind (%v)", path, statErr)
	}
	// The library is free again: a second export runs to its end.
	if _, err := db.ExportDatabase(ExportOptions{ExportPath: path, AllPositions: true, Metadata: map[string]string{}}); err != nil {
		t.Fatal(err)
	}
}

func TestPaperArchivePageCtx_Cancel(t *testing.T) {
	t.Parallel()
	db := longOpsLibrary(t)
	ctx, seen := cancelOnFirstUnit(t)
	_, _, err := PaperArchivePageCtx(ctx, db, nil, "")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if len(*seen) == 0 || (*seen)[0][1] != 2 {
		t.Fatalf("progress %v, want reports out of 2 matches", *seen)
	}
}

func TestImportPaperArchiveCtx_CancelWritesNoMatch(t *testing.T) {
	t.Parallel()
	src := longOpsLibrary(t)
	page := filepath.Join(t.TempDir(), "archive.html")
	if _, err := src.ExportPaperArchive(nil, page, ""); err != nil {
		t.Fatal(err)
	}
	dst := newTestDB(t)
	// One file: its one unit read cancels before any match is written.
	ctx, _ := cancelOnFirstUnit(t)
	res, err := ImportPaperArchiveCtx(ctx, dst, []string{page})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if res != nil && len(res.Imported) != 0 {
		t.Fatalf("imported %v after a cancel", res.Imported)
	}
	var n int
	if err := dst.db.QueryRow(`SELECT COUNT(*) FROM match`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("%d matches written after a cancel (%v)", n, err)
	}
	// Uncancelled, the same file imports both matches, reporting to the end.
	var last [2]int
	ctx = progress.With(context.Background(), func(done, total int) { last = [2]int{done, total} })
	res, err = ImportPaperArchiveCtx(ctx, dst, []string{page})
	if err != nil || len(res.Imported) != 2 || last != [2]int{3, 3} {
		t.Fatalf("res %+v err %v last %v", res, err, last)
	}
}

func TestMatchSheetHTMLCtx_Cancel(t *testing.T) {
	t.Parallel()
	db := longOpsLibrary(t)
	ids := matchIDs(t, db)
	ctx, _ := cancelOnFirstUnit(t)
	// The first report is 0 of 2; the cancel comes with the second, before
	// the second match is read.
	if _, err := MatchSheetHTMLCtx(ctx, db, ids, MatchSheetOptions{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func matchIDs(t *testing.T, db *Database) []int64 {
	t.Helper()
	rows, err := db.db.Query(`SELECT id FROM match ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return ids
}
