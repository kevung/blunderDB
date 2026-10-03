package database

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// TestReimportedDuplicateDeliversItsStudyMarks: a mark added in XG after the
// first import does not change the match hash, so the re-import is a
// duplicate — and still lands the mark in the database (ADR-0006). The error
// is ErrDuplicateMatch, typed with how many marks it carried; a re-import
// carrying none reports zero.
func TestReimportedDuplicateDeliversItsStudyMarks(t *testing.T) {
	ctx := context.Background()
	d := NewDatabase()
	if err := d.SetupDatabase(filepath.Join(t.TempDir(), "flags.db")); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	defer d.Close()
	fixture := filepath.Join("testdata", "test.xg")
	if _, err := d.ImportXGMatch(fixture); err != nil {
		t.Fatalf("first import: %v", err)
	}

	countFlagged := func() int {
		var n int
		if err := d.conn().QueryRow(`SELECT count(*) FROM position WHERE flagged = 1`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	before := countFlagged()

	reimport := func(flag bool) error {
		graph, err := ingest.MapXG(fixture)
		if err != nil {
			t.Fatal(err)
		}
		if flag {
			for gi := range graph.Games {
				for mi := range graph.Games[gi].Moves {
					if p := graph.Games[gi].Moves[mi].Position; p != nil && !p.Flagged {
						p.Flagged = true
						goto marked
					}
				}
			}
			t.Fatal("no unflagged position in the fixture")
		}
	marked:
		d.mu.Lock()
		defer d.mu.Unlock()
		_, err = d.writeImportedMatch(ctx, graph)
		return err
	}

	err := reimport(false)
	var dup *DuplicateMatchError
	if !errors.As(err, &dup) || !errors.Is(err, ErrDuplicateMatch) {
		t.Fatalf("plain re-import: err = %v, want a DuplicateMatchError", err)
	}
	if dup.FlagsApplied != 0 {
		t.Errorf("plain re-import: FlagsApplied = %d, want 0", dup.FlagsApplied)
	}

	err = reimport(true)
	if !errors.As(err, &dup) || !errors.Is(err, ErrDuplicateMatch) {
		t.Fatalf("marked re-import: err = %v, want a DuplicateMatchError", err)
	}
	if dup.FlagsApplied != 1 {
		t.Errorf("FlagsApplied = %d, want 1: only the newly raised mark counts", dup.FlagsApplied)
	}
	if got := countFlagged(); got != before+1 {
		t.Errorf("flagged positions = %d after the marked re-import, want %d: the mark was rolled back", got, before+1)
	}
}
