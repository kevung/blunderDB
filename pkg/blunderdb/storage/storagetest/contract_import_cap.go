package storagetest

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// cappedSourcePositions is how many positions cappedSource holds.
const cappedSourcePositions = 5

// cappedSource writes a native database of cappedSourcePositions positions in
// ascending id, and two matches: "first" stands on the first position,
// "last" on the last one. It returns its path.
func cappedSource(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "capped.db")
	src, err := sqlite.Open(ctx, path, nil)
	if err != nil {
		t.Fatalf("open source: %v", err)
	}
	defer src.Close()
	var ids []int64
	for n := range cappedSourcePositions {
		p := provenancePos(n + 1)
		id, err := src.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("save source position: %v", err)
		}
		ids = append(ids, id)
	}
	for _, m := range []struct {
		hash string
		pos  int64
	}{{"first", ids[0]}, {"last", ids[len(ids)-1]}} {
		match := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 5, MatchHash: m.hash}
		mid, err := src.Matches().Save(ctx, "", &match)
		if err != nil {
			t.Fatalf("save source match: %v", err)
		}
		gid, err := src.Matches().CreateGame(ctx, "", &domain.Game{MatchID: mid, GameNumber: 1})
		if err != nil {
			t.Fatalf("CreateGame: %v", err)
		}
		mv := domain.Move{GameID: gid, MoveNumber: 1, MoveType: "checker", PositionID: m.pos, Player: 1, Dice: [2]int32{3, 1}, CheckerMove: "8/5 6/5"}
		if _, err := src.Matches().CreateMove(ctx, "", &mv); err != nil {
			t.Fatalf("CreateMove: %v", err)
		}
	}
	return path
}

func countRows(t *testing.T, s storage.Storage) (positions, matches int) {
	t.Helper()
	ctx := context.Background()
	positions, err := s.Positions().Count(ctx, "")
	if err != nil {
		t.Fatalf("count positions: %v", err)
	}
	if matches, err = s.Matches().Count(ctx, "", storage.MatchListOpts{}); err != nil {
		t.Fatalf("count matches: %v", err)
	}
	return positions, matches
}

// testDBImportCapped: a capped native import carries the first positions in
// source id order, announces that total before writing, leaves out whole the
// matches reaching a position it did not carry, and an uncapped import of the
// same file afterwards brings the rest complete.
func testDBImportCapped(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	path := cappedSource(t)
	var first *ingest.Progress
	var totals []int
	sum, err := ingest.DBImporter{S: s}.Import(ctx, "", ingest.Source{Format: ingest.FormatNativeDB, Path: path, MaxPositions: 3}, func(p ingest.Progress) {
		if first == nil {
			first = &p
		}
		totals = append(totals, p.TotalPositions)
	})
	if err != nil {
		t.Fatalf("capped import: %v", err)
	}
	if first == nil || first.Positions != 0 || first.TotalPositions != 3 {
		t.Fatalf("first progress = %+v, want the total 3 before any position", first)
	}
	for _, n := range totals {
		if n != 3 {
			t.Fatalf("progress totals %v, want 3 throughout", totals)
		}
	}
	if sum.SavedPositions != 3 || sum.PositionsLeftOut != 2 || sum.Matches != 1 || sum.MatchesLeftOut != 1 {
		t.Fatalf("capped summary = saved %d, left out %d, matches %d, matches left out %d; want 3, 2, 1, 1",
			sum.SavedPositions, sum.PositionsLeftOut, sum.Matches, sum.MatchesLeftOut)
	}
	if p, m := countRows(t, s); p != 3 || m != 1 {
		t.Fatalf("after capped import: %d positions, %d matches; want 3, 1", p, m)
	}
	scores := map[int]bool{}
	for p, err := range s.Positions().List(ctx, "", storage.ListOpts{}) {
		if err != nil {
			t.Fatalf("list positions: %v", err)
		}
		scores[p.Score[0]] = true
	}
	if !scores[1] || !scores[2] || !scores[3] {
		t.Fatalf("carried positions of scores %v, want 1, 2 and 3: the cap keeps the first source ids", scores)
	}

	sum, err = ingest.DBImporter{S: s}.Import(ctx, "", ingest.Source{Format: ingest.FormatNativeDB, Path: path}, nil)
	if err != nil {
		t.Fatalf("uncapped import: %v", err)
	}
	if sum.PositionsLeftOut != 0 || sum.MatchesLeftOut != 0 || sum.Matches != 1 {
		t.Fatalf("uncapped summary = %+v; want the remaining match, nothing left out", sum)
	}
	if p, m := countRows(t, s); p != cappedSourcePositions || m != 2 {
		t.Fatalf("after uncapped import: %d positions, %d matches; want %d, 2", p, m, cappedSourcePositions)
	}
}

// testDBImportCancelLeavesNothing: an import cancelled midway, or failing,
// writes no row at all.
func testDBImportCancelLeavesNothing(t *testing.T, s storage.Storage) {
	path := cappedSource(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err := ingest.DBImporter{S: s}.Import(ctx, "", ingest.Source{Format: ingest.FormatNativeDB, Path: path}, func(p ingest.Progress) {
		if p.Positions == 2 {
			cancel()
		}
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled import: err = %v, want context.Canceled", err)
	}
	if p, m := countRows(t, s); p != 0 || m != 0 {
		t.Fatalf("after a cancelled import: %d positions, %d matches; want none", p, m)
	}

	failing := ingest.DBImporter{S: s, BeforeCommit: func(context.Context, storage.Tx, ingest.Summary) error {
		return errors.New("refused")
	}}
	if _, err := failing.Import(context.Background(), "", ingest.Source{Format: ingest.FormatNativeDB, Path: path}, nil); err == nil {
		t.Fatal("failing import: no error")
	}
	if p, m := countRows(t, s); p != 0 || m != 0 {
		t.Fatalf("after a failed import: %d positions, %d matches; want none", p, m)
	}
}

// testDBAnalyzeWritesNothing: the analysis counts what the file would bring
// against what the target holds, and writes nothing.
func testDBAnalyzeWritesNothing(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	path := cappedSource(t)
	held := provenancePos(1)
	if _, err := s.Positions().Save(ctx, "", &held); err != nil {
		t.Fatalf("save held position: %v", err)
	}
	a, err := ingest.AnalyzeDB(ctx, s, "", path)
	if err != nil {
		t.Fatalf("AnalyzeDB: %v", err)
	}
	if a.Positions != cappedSourcePositions || a.PositionsHeld != 1 || a.Matches != 2 || a.MatchesKnown != 0 || a.SourceVersion == "" {
		t.Fatalf("analysis = %+v; want %d positions, 1 held, 2 matches, a version", a, cappedSourcePositions)
	}
	if p, m := countRows(t, s); p != 1 || m != 0 {
		t.Fatalf("after analysis: %d positions, %d matches; want the 1 held, no match", p, m)
	}
	if _, err := ingest.AnalyzeDB(ctx, s, "", filepath.Join(t.TempDir(), "missing.db")); err == nil {
		t.Fatal("AnalyzeDB of a missing file: no error")
	}
}
