package database

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// pipelineFixture is a list mixing new matches, a byte-identical copy, a
// cross-format duplicate, an unreadable file and a single position.
func pipelineFixture(t *testing.T) []string {
	t.Helper()
	dir := tempDir(t)
	copyOf := filepath.Join(dir, "copy.xg")
	data, err := os.ReadFile(filepath.Join("testdata", "test.xg"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(copyOf, data, 0o644); err != nil {
		t.Fatal(err)
	}
	broken := filepath.Join(dir, "broken.xg")
	if err := os.WriteFile(broken, []byte("not a match"), 0o644); err != nil {
		t.Fatal(err)
	}
	return []string{
		filepath.Join("testdata", "test.xg"),
		filepath.Join("testdata", "charlot1-charlot2_7p_2025-11-08-2305.xg"),
		copyOf,
		broken,
		filepath.Join("testdata", "charlot1-charlot2_7p_2025-11-08-2305.mat"),
		filepath.Join("testdata", "xgp", "Position 10.xgp"),
		filepath.Join("testdata", "test.mat"),
		filepath.Join("testdata", "match_with_comment.xg"),
	}
}

// The writer decides in file order: the outcome of a list does not depend on
// how many readers raced over it nor on how files are grouped.
func TestImportFiles_DeterministicAcrossWorkers(t *testing.T) {
	files := pipelineFixture(t)
	type shape struct {
		status        string
		match, posID  int64
		positions, of int
	}
	run := func(workers, perTx int) ([]shape, int) {
		db := newTestDB(t)
		out, err := db.ImportFiles(files, ImportFilesOptions{Workers: workers, FilesPerTx: perTx})
		if err != nil {
			t.Fatalf("ImportFiles(%d,%d): %v", workers, perTx, err)
		}
		if len(out) != len(files) {
			t.Fatalf("got %d outcomes for %d files", len(out), len(files))
		}
		s := make([]shape, len(out))
		for i, o := range out {
			if o.Index != i {
				t.Fatalf("outcome %d has index %d", i, o.Index)
			}
			s[i] = shape{o.Status, o.MatchID, o.PositionID, o.Positions, o.DuplicateOf}
		}
		return s, countRows(t, db.db, "position")
	}
	ref, refPositions := run(1, 1)
	for _, c := range [][2]int{{8, 3}} {
		got, positions := run(c[0], c[1])
		for i := range ref {
			if got[i] != ref[i] {
				t.Errorf("workers=%d perTx=%d file %d: %+v, want %+v", c[0], c[1], i, got[i], ref[i])
			}
		}
		if positions != refPositions {
			t.Errorf("workers=%d: %d positions, want %d", c[0], positions, refPositions)
		}
	}

	want := []string{ingest.FileImported, ingest.FileImported, ingest.FileDuplicate, ingest.FileFailed,
		ingest.FileEnriched, ingest.FilePosition}
	for i, w := range want {
		if ref[i].status != w {
			t.Errorf("file %d: status %s, want %s", i, ref[i].status, w)
		}
	}
	if ref[2].of != 0 || ref[2].match != ref[0].match {
		t.Errorf("byte copy: duplicate of %d (match %d), want file 0 (match %d)", ref[2].of, ref[2].match, ref[0].match)
	}
	if ref[0].match >= ref[1].match {
		t.Errorf("match ids not in file order: %d then %d", ref[0].match, ref[1].match)
	}
}

// The pipeline writes what the one-file importers write.
func TestImportFiles_SameDatabaseAsSequential(t *testing.T) {
	files := pipelineFixture(t)
	seq := newTestDB(t)
	for _, f := range files {
		switch filepath.Ext(f) {
		case ".xg":
			_, _ = seq.ImportXGMatch(f)
		case ".mat":
			_, _ = seq.ImportGnuBGMatch(f)
		case ".xgp":
			_, _ = seq.ImportXGPPosition(f)
		}
	}
	par := newTestDB(t)
	if _, err := par.ImportFiles(files, ImportFilesOptions{Workers: 4, FilesPerTx: 3}); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"match", "game", "move", "position", "analysis"} {
		if a, b := countRows(t, seq.db, table), countRows(t, par.db, table); a != b {
			t.Errorf("%s: sequential %d rows, pipeline %d", table, a, b)
		}
	}
}

// A cancellation rolls back the group being written; committed groups stay,
// and every reported file is in the database.
func TestImportFiles_CancelKeepsCommittedGroups(t *testing.T) {
	files := pipelineFixture(t)
	db := newTestDB(t)
	seen := 0
	out, err := db.ImportFiles(files, ImportFilesOptions{Workers: 2, FilesPerTx: 2, OnFile: func(ingest.FileOutcome) {
		seen++
		if seen == 2 {
			db.CancelImport()
		}
	}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if len(out) != 2 {
		t.Fatalf("%d outcomes after cancelling at the first commit, want 2", len(out))
	}
	if n := countRows(t, db.db, "match"); n != 2 {
		t.Errorf("%d matches stored, want the 2 committed", n)
	}
}

// The open batch counts what the pipeline committed.
func TestImportFiles_CountsIntoBatch(t *testing.T) {
	files := pipelineFixture(t)
	db := newTestDB(t)
	id, err := db.BeginImportBatch("fixture", "mixed")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ImportFiles(files, ImportFilesOptions{}); err != nil {
		t.Fatal(err)
	}
	c := db.importBatchCounts
	if c.MatchesImported != 3 || c.MatchesEnriched != 2 || c.MatchesSkipped != 1 {
		t.Errorf("counts = %+v, want 3 imported, 2 enriched, 1 skipped", c)
	}
	var stamped int
	if err := db.db.QueryRow("SELECT COUNT(*) FROM match WHERE import_batch_id = ?", id).Scan(&stamped); err != nil {
		t.Fatal(err)
	}
	if stamped != 3 {
		t.Errorf("%d matches stamped with the batch, want 3", stamped)
	}
}
