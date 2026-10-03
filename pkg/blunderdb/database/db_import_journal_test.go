package database

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func journalByPath(t *testing.T, db *Database, batchID int64) map[string]domain.ImportFileEntry {
	t.Helper()
	entries, err := db.ImportJournal(batchID)
	if err != nil {
		t.Fatal(err)
	}
	m := map[string]domain.ImportFileEntry{}
	for _, e := range entries {
		m[e.Path] = e
	}
	return m
}

// Every file of a batch is journaled with what it gave: a new match, the match
// that covers a duplicate, an enrichment, or the error.
func TestImportFiles_JournalsEachFile(t *testing.T) {
	files := pipelineFixture(t)
	db := newTestDB(t)
	id, err := db.BeginImportBatch("fixture", "mixed")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ImportFiles(files, ImportFilesOptions{}); err != nil {
		t.Fatal(err)
	}
	j := journalByPath(t, db, id)
	if len(j) != len(files) {
		t.Fatalf("journal has %d files, want %d", len(j), len(files))
	}
	first, copyOf, broken := j[files[0]], j[files[2]], j[files[3]]
	if first.Outcome != domain.JournalNew || first.MatchID == 0 || first.SHA256 == "" || first.MTime == "" || first.Size == 0 {
		t.Errorf("new file journaled as %+v", first)
	}
	if copyOf.Outcome != domain.JournalDuplicate || copyOf.MatchID != first.MatchID {
		t.Errorf("copy journaled as %+v, want a duplicate of match %d", copyOf, first.MatchID)
	}
	if broken.Outcome != domain.JournalError || broken.Error == "" || broken.MatchID != 0 {
		t.Errorf("broken file journaled as %+v", broken)
	}
	enriched := 0
	for _, e := range j {
		if e.Outcome == domain.JournalEnriched {
			enriched++
		}
	}
	if enriched != 2 {
		t.Errorf("%d enriched files journaled, want 2", enriched)
	}
}

// A resumed batch leaves out the files already decided (same path, size,
// mtime), retries the ones that failed, and recognises a known file moved
// elsewhere by its digest without parsing it.
func TestResumeImportBatch(t *testing.T) {
	files := pipelineFixture(t)
	db := newTestDB(t)
	id, err := db.BeginImportBatch("fixture", "mixed")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ImportFiles(files, ImportFilesOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := db.FinishImportBatch(id, domain.ImportReport{}); err != nil {
		t.Fatal(err)
	}
	var before int
	if err := db.db.QueryRow("SELECT COUNT(*) FROM match").Scan(&before); err != nil {
		t.Fatal(err)
	}

	moved := filepath.Join(tempDir(t), "renamed.xg")
	data, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(moved, data, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := db.ResumeImportBatch(id + 99); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("resuming an unknown batch: got %v, want ErrNotFound", err)
	}
	if err := db.ResumeImportBatch(id); err != nil {
		t.Fatal(err)
	}
	all := append(append([]string{}, files...), moved)
	todo := db.PendingImportFiles(all)
	if skipped := len(all) - len(todo); skipped != len(files)-1 || len(todo) != 2 || todo[0] != files[3] || todo[1] != moved {
		t.Fatalf("pending = %v, want the broken file and the moved one", todo)
	}
	out, err := db.ImportFiles(todo, ImportFilesOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if out[0].Status != "failed" {
		t.Errorf("the broken file is retried and fails again, got %q", out[0].Status)
	}
	orig := journalByPath(t, db, id)[files[0]]
	if out[1].Status != "duplicate" || out[1].MatchID != orig.MatchID {
		t.Errorf("the moved file = %+v, want a duplicate of match %d", out[1], orig.MatchID)
	}
	var after int
	if err := db.db.QueryRow("SELECT COUNT(*) FROM match").Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Errorf("resuming wrote matches: %d, was %d", after, before)
	}
	if err := db.FinishImportBatch(id, domain.ImportReport{}); err != nil {
		t.Fatal(err)
	}
	b, err := db.ImportReport(id)
	if err != nil {
		t.Fatal(err)
	}
	if b.Report.MatchesImported != 3 {
		t.Errorf("the resumed batch lost its earlier counts: %+v", b.Report)
	}
}

// In bulk mode a session holds its own connection while the journal is written
// through the pool: on a file database both must coexist, and the journal must
// be complete once the session closes.
func TestImportFiles_BulkModeJournalsEachFile(t *testing.T) {
	old := bulkImportMinFiles
	bulkImportMinFiles = 1
	t.Cleanup(func() { bulkImportMinFiles = old })

	files := pipelineFixture(t)
	db := newTestDB(t)
	id, err := db.BeginImportBatch("fixture", "mixed")
	if err != nil {
		t.Fatal(err)
	}
	var bulk bool
	if _, err := db.ImportFiles(files, ImportFilesOptions{OnBulk: func(bool) { bulk = true }}); err != nil {
		t.Fatal(err)
	}
	if !bulk {
		t.Fatal("the import did not run in bulk mode")
	}
	if j := journalByPath(t, db, id); len(j) != len(files) {
		t.Fatalf("journal has %d files after a bulk import, want %d", len(j), len(files))
	}
}

// A match the import does not write again — skipped under --skip-duplicates,
// or a copy whose deeper analyses replaced the stored ones — is journaled as a
// duplicate of the match that covers it, and a resumed batch leaves it out.
func TestImportFiles_SkippedDuplicateIsJournaledAndResumeSkipsIt(t *testing.T) {
	db := newTestDB(t)
	src := filepath.Join("testdata", "test.mat")
	if _, err := db.BeginImportBatch("first", "batch"); err != nil {
		t.Fatal(err)
	}
	out, err := db.ImportFiles([]string{src}, ImportFilesOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if out[0].Status != "imported" || out[0].MatchID == 0 {
		t.Fatalf("first import = %+v", out[0])
	}
	stored := out[0].MatchID

	// Other bytes, same match: the digest does not recognise it, the match does.
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	again := filepath.Join(tempDir(t), "again.mat")
	if err := os.WriteFile(again, append(data, "\n\n"...), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, skip := range []bool{true, false} {
		db.SetSkipDuplicates(skip)
		id, err := db.BeginImportBatch("again", "batch")
		if err != nil {
			t.Fatal(err)
		}
		out, err := db.ImportFiles([]string{again}, ImportFilesOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if out[0].Status != "duplicate" || out[0].MatchID != stored {
			t.Fatalf("skip=%v: outcome %+v, want a duplicate of match %d", skip, out[0], stored)
		}
		e := journalByPath(t, db, id)[again]
		if e.Outcome != domain.JournalDuplicate || e.MatchID != stored || e.SHA256 == "" {
			t.Errorf("skip=%v: journaled as %+v, want a duplicate of match %d", skip, e, stored)
		}
		if err := db.FinishImportBatch(id, domain.ImportReport{}); err != nil {
			t.Fatal(err)
		}
		if err := db.ResumeImportBatch(id); err != nil {
			t.Fatal(err)
		}
		if todo := db.PendingImportFiles([]string{again}); len(todo) != 0 {
			t.Errorf("skip=%v: a resumed batch would read %v again", skip, todo)
		}
		if err := db.FinishImportBatch(id, domain.ImportReport{}); err != nil {
			t.Fatal(err)
		}
	}
	db.SetSkipDuplicates(false)
}
