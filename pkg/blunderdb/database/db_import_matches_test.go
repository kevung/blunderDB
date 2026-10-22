package database

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// TestCommitImportDatabase_Matches: the desktop import of a .db follows the
// daemon's rule (ingest.MergeMatches) — the match comes with its moves, the
// tournament, the draft and the alias with it, the living query is renumbered
// onto the target's rows, and a second import of the same file adds nothing.
func TestCommitImportDatabase_Matches(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := t.TempDir()

	srcPath := filepath.Join(dir, "club.db")
	src, err := sqlite.Open(ctx, srcPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	p := domain.InitializePosition()
	p.Dice = [2]int{3, 1}
	pid, _ := src.Positions().Save(ctx, "", &p)
	tour, _ := src.Tournaments().Create(ctx, "", "Open de Lyon", "", "")
	mid, err := src.Matches().Save(ctx, "", &domain.Match{Player1Name: "Bob", Player2Name: "Zoé", MatchLength: 5, MatchHash: "h-bob"})
	if err != nil {
		t.Fatal(err)
	}
	_ = src.Tournaments().AddMatch(ctx, "", tour, mid)
	gid, _ := src.Matches().CreateGame(ctx, "", &domain.Game{MatchID: mid, GameNumber: 1})
	if _, err := src.Matches().CreateMove(ctx, "", &domain.Move{GameID: gid, MoveNumber: 1, PositionID: pid,
		MoveType: "checker", Player: 1, Dice: [2]int32{3, 1}, CheckerMove: "8/5 6/5"}); err != nil {
		t.Fatal(err)
	}
	if _, err := src.Transcriptions().Save(ctx, "", &storage.Transcription{FormatVersion: "1", MatchID: mid,
		Label: "Bob – Zoé", Document: `{"header":{}}`}); err != nil {
		t.Fatal(err)
	}
	_ = src.Aliases().Set(ctx, "", storage.AliasPlayer, "Bobby", "Bob")
	if _, err := storage.CreateCollection(ctx, src, "", "Du match", "", fmt.Sprintf("ma%d", mid)); err != nil {
		t.Fatal(err)
	}
	src.Close()

	d := NewDatabase()
	if err := d.SetupDatabase(filepath.Join(dir, "mine.db")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	// A match of its own first, so the imported one is numbered otherwise.
	own := domain.InitializePosition()
	own.Dice = [2]int{4, 2}
	_, _ = d.SavePosition(&own)
	if _, err := d.store.Matches().Save(ctx, "", &domain.Match{Player1Name: "X", Player2Name: "Y", MatchLength: 3, MatchHash: "h-own"}); err != nil {
		t.Fatal(err)
	}

	for pass, want := range []int{1, 0} {
		res, err := d.CommitImportDatabase(srcPath)
		if err != nil {
			t.Fatalf("pass %d: %v", pass+1, err)
		}
		for _, key := range []string{"matches", "tournaments", "transcriptions", "aliases"} {
			if res[key] != want {
				t.Errorf("pass %d: %s = %v, want %d", pass+1, key, res[key], want)
			}
		}
		if res["matchesKnown"] != 1-want {
			t.Errorf("pass %d: matchesKnown = %v, want %d", pass+1, res["matchesKnown"], 1-want)
		}
	}

	newMatch, found, err := d.store.Matches().FindByHash(ctx, "", "h-bob", "")
	if err != nil || !found {
		t.Fatalf("imported match not found: %v", err)
	}
	cols, err := d.GetAllCollections()
	if err != nil {
		t.Fatal(err)
	}
	query := "(missing)"
	for _, c := range cols {
		if c.Name == "Du match" {
			query = c.FilterQuery
		}
	}
	if query != fmt.Sprintf("s ma%d", newMatch) {
		t.Errorf("living query = %q, want s ma%d", query, newMatch)
	}
}
