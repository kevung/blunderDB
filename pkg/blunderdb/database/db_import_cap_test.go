package database

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// TestCommitImportDatabaseCapped: the desktop import follows the daemon's cap
// (ingest.PlanCap) — the first positions in source id order that something
// holds once imported, a match reaching a left-out position left out whole,
// and an uncapped import of the same file afterwards bringing the rest.
func TestCommitImportDatabaseCapped(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := t.TempDir()

	srcPath := filepath.Join(dir, "capped.db")
	src, err := sqlite.Open(ctx, srcPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	var ids []int64
	for n := 1; n <= 4; n++ {
		p := domain.InitializePosition()
		p.Score = [2]int{n, 0}
		p.IndividuallyImported = n == 2
		id, err := src.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	for hash, pid := range map[string]int64{"first": ids[0], "last": ids[3]} {
		mid, err := src.Matches().Save(ctx, "", &domain.Match{Player1Name: "Bob", Player2Name: "Zoé", MatchLength: 5, MatchHash: hash})
		if err != nil {
			t.Fatal(err)
		}
		gid, _ := src.Matches().CreateGame(ctx, "", &domain.Game{MatchID: mid, GameNumber: 1})
		if _, err := src.Matches().CreateMove(ctx, "", &domain.Move{GameID: gid, MoveNumber: 1, PositionID: pid,
			MoveType: "checker", Player: 1, Dice: [2]int32{3, 1}, CheckerMove: "8/5 6/5"}); err != nil {
			t.Fatal(err)
		}
	}
	src.Close()

	d := NewDatabase()
	if err := d.SetupDatabase(filepath.Join(dir, "mine.db")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })

	counts := func() (positions, matches int) {
		t.Helper()
		positions, err := d.store.Positions().Count(ctx, "")
		if err != nil {
			t.Fatal(err)
		}
		if matches, err = d.store.Matches().Count(ctx, "", storage.MatchListOpts{}); err != nil {
			t.Fatal(err)
		}
		return positions, matches
	}

	res, err := d.CommitImportDatabaseCapped(srcPath, 3)
	if err != nil {
		t.Fatalf("capped import: %v", err)
	}
	// p1 is held by match "first", p2 is individually imported; p3 would be
	// held by nothing once imported, and p4 is past the cap.
	if res["added"] != 2 || res["leftOut"] != 2 || res["matches"] != 1 || res["matchesLeftOut"] != 1 {
		t.Fatalf("capped result = %v; want 2 added, 2 left out, 1 match, 1 match left out", res)
	}
	if p, m := counts(); p != 2 || m != 1 {
		t.Fatalf("after capped import: %d positions, %d matches; want 2, 1", p, m)
	}

	res, err = d.CommitImportDatabase(srcPath)
	if err != nil {
		t.Fatalf("uncapped import: %v", err)
	}
	if res["added"] != 2 || res["leftOut"] != 0 || res["matches"] != 1 || res["matchesLeftOut"] != 0 {
		t.Fatalf("uncapped result = %v; want the remaining position and match", res)
	}
	if p, m := counts(); p != 4 || m != 2 {
		t.Fatalf("after uncapped import: %d positions, %d matches; want 4, 2", p, m)
	}

	if _, err := d.CommitImportDatabaseCapped(srcPath, -1); err == nil {
		t.Fatal("negative cap: no error")
	}
}

// TestCommitImportDatabaseCarriesFlag: the source tool's study mark travels
// with the desktop import and, as in the daemon's, holds its position under
// the cap; on a position the target already has, it is raised.
func TestCommitImportDatabaseCarriesFlag(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := t.TempDir()

	srcPath := filepath.Join(dir, "flagged.db")
	src, err := sqlite.Open(ctx, srcPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	positions := make([]domain.Position, 3)
	for n := range positions {
		positions[n] = domain.InitializePosition()
		positions[n].Score = [2]int{n + 1, 0}
		positions[n].Flagged = n != 0
		if _, err := src.Positions().Save(ctx, "", &positions[n]); err != nil {
			t.Fatal(err)
		}
	}
	src.Close()

	d := NewDatabase()
	if err := d.SetupDatabase(filepath.Join(dir, "mine.db")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	held := positions[2]
	held.Flagged = false
	if _, err := d.store.Positions().Save(ctx, "", &held); err != nil {
		t.Fatal(err)
	}

	// p1 is held by nothing, p2 by its mark; p3 is past the cap.
	res, err := d.CommitImportDatabaseCapped(srcPath, 2)
	if err != nil {
		t.Fatalf("capped import: %v", err)
	}
	if res["added"] != 1 || res["leftOut"] != 2 {
		t.Fatalf("capped result = %v; want 1 added, 2 left out", res)
	}
	if _, err := d.CommitImportDatabase(srcPath); err != nil {
		t.Fatalf("uncapped import: %v", err)
	}
	for n, p := range positions {
		id, ok, err := d.store.Positions().Exists(ctx, "", engine.ZobristHash(&p))
		if err != nil || !ok {
			t.Fatalf("p%d not held after the uncapped import: %v", n+1, err)
		}
		got, err := d.store.Positions().Load(ctx, "", id)
		if err != nil {
			t.Fatal(err)
		}
		if got.Flagged != p.Flagged {
			t.Fatalf("p%d flagged = %v, want %v", n+1, got.Flagged, p.Flagged)
		}
	}
}
