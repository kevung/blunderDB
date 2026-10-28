package database

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
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
