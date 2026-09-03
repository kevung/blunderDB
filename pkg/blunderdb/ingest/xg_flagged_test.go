package ingest

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// xgFixture is the shared XG match; three of its decisions are flagged in
// eXtreme Gammon — one checker move in game 1, one in game 2, and one cube
// decision in game 2.
func xgFixture() string {
	return filepath.Join("..", "..", "..", "testdata", "test.xg")
}

// TestMapXGCarriesFlags checks that the study marks the user set in XG survive
// the mapping. xgparser reads them into MoveEntry.Flagged / CubeEntry.
// FlaggedDouble but its lightweight Match drops them, so ingest re-reads the raw
// segments — this test is what pins that re-read to the right decisions.
func TestMapXGCarriesFlags(t *testing.T) {
	g, err := MapXG(xgFixture())
	if err != nil {
		t.Fatalf("MapXG: %v", err)
	}

	var checker, cube int
	for gi := range g.Games {
		for mi := range g.Games[gi].Moves {
			p := g.Games[gi].Moves[mi].Position
			if p == nil || !p.Flagged {
				continue
			}
			if p.DecisionType == domain.CubeAction {
				cube++
			} else {
				checker++
			}
		}
	}

	// Two flagged checker moves give one position each. The single flagged cube
	// decision is a double/take, which blunderDB stores as two positions — the
	// double and the take/pass — and both inherit the mark (docs/adr/0006).
	if checker != 2 {
		t.Errorf("flagged checker positions: got %d, want 2", checker)
	}
	if cube != 2 {
		t.Errorf("flagged cube positions: got %d, want 2 (both sides of one flagged cube decision)", cube)
	}
}

// TestFlagsReachStorageAndSurviveReimport is the end-to-end guarantee: the marks
// land in the database, the filter finds exactly them, and re-importing the same
// file — an exact duplicate that writes nothing else — still delivers them.
// That last part is what makes the feature usable on an existing database:
// flagging a position in XG does not change the match hash, so without it the
// mark could only arrive by deleting the match and importing it again.
func TestFlagsReachStorageAndSurviveReimport(t *testing.T) {
	ctx := context.Background()
	s, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}
	defer s.Close()

	write := func() (int, bool) {
		t.Helper()
		g, err := MapXG(xgFixture())
		if err != nil {
			t.Fatalf("MapXG: %v", err)
		}
		tx, err := s.BeginTx(ctx)
		if err != nil {
			t.Fatalf("begin: %v", err)
		}
		res, err := WriteMatch(ctx, tx, "", g, nil)
		if err != nil {
			t.Fatalf("WriteMatch: %v", err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatalf("commit: %v", err)
		}
		return res.FlagsApplied, res.Skipped
	}

	countFlagged := func() int {
		t.Helper()
		n := 0
		for pos, err := range s.Search().Find(ctx, "", domain.SearchFilters{FlaggedFilter: true}, storage.ListOpts{}) {
			if err != nil {
				t.Fatalf("Find: %v", err)
			}
			if !pos.Flagged {
				t.Errorf("position %d came back from the flagged filter with Flagged=false", pos.ID)
			}
			n++
		}
		return n
	}

	if _, skipped := write(); skipped {
		t.Fatal("first import should not be skipped")
	}
	first := countFlagged()
	if first != 4 {
		t.Fatalf("flagged positions after import: got %d, want 4", first)
	}

	applied, skipped := write()
	if !skipped {
		t.Error("re-importing the same file should be detected as an exact duplicate")
	}
	if applied != 4 {
		t.Errorf("a skipped duplicate should still deliver its %d marks, applied %d", 4, applied)
	}
	if got := countFlagged(); got != first {
		t.Errorf("re-import changed the flagged count: got %d, want %d", got, first)
	}
}
