package ingest

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

func TestDBImportMatches_SQLite(t *testing.T) {
	target, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()
	checkDBImportMatches(t, target, "")
}

// checkDBImportMatches: a native .db import brings its matches with their
// games and moves, its tournaments (merged by name), its drafts and aliases;
// a living collection's query is renumbered onto the target's rows, or frozen
// when it names a row the file does not hold. A second import of the same
// file changes nothing.
func checkDBImportMatches(t *testing.T, target storage.Storage, scope string) {
	t.Helper()
	ctx := context.Background()

	srcPath := filepath.Join(t.TempDir(), "club.db")
	src, err := sqlite.Open(ctx, srcPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	var pos []int64
	for _, dice := range [][2]int{{3, 1}, {6, 5}} {
		p := domain.InitializePosition()
		p.Dice = dice
		id, err := src.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatal(err)
		}
		pos = append(pos, id)
	}
	tour, err := src.Tournaments().Create(ctx, "", "Open de Lyon", "2026-03-01", "Lyon")
	if err != nil {
		t.Fatal(err)
	}
	if err := src.Tournaments().UpdateComment(ctx, "", tour, "finale perdue"); err != nil {
		t.Fatal(err)
	}
	saveMatch := func(hash, p1 string) int64 {
		id, err := src.Matches().Save(ctx, "", &domain.Match{Player1Name: p1, Player2Name: "Zoé",
			MatchLength: 5, MatchHash: hash, Comment: "note de " + p1})
		if err != nil {
			t.Fatal(err)
		}
		g, err := src.Matches().CreateGame(ctx, "", &domain.Game{MatchID: id, GameNumber: 1})
		if err != nil {
			t.Fatal(err)
		}
		for i, p := range pos {
			if _, err := src.Matches().CreateMove(ctx, "", &domain.Move{GameID: g, MoveNumber: int32(i + 1),
				PositionID: p, MoveType: "checker", Player: 1, Dice: [2]int32{3, 1}, CheckerMove: "8/5 6/5"}); err != nil {
				t.Fatal(err)
			}
		}
		return id
	}
	m1 := saveMatch("h-bob", "Bob")
	if err := src.Tournaments().AddMatch(ctx, "", tour, m1); err != nil {
		t.Fatal(err)
	}
	if _, err := src.Transcriptions().Save(ctx, "", &storage.Transcription{FormatVersion: "1",
		MatchID: m1, Label: "Bob – Zoé", Document: `{"header":{"player1":"Bob"}}`}); err != nil {
		t.Fatal(err)
	}
	if err := src.Aliases().Set(ctx, "", storage.AliasPlayer, "Bobby", "Bob"); err != nil {
		t.Fatal(err)
	}
	for name, query := range map[string]string{
		"Du match":   fmt.Sprintf("ma%d", m1),
		"Du tournoi": fmt.Sprintf("tn%d", tour),
		"Fantôme":    "ma999",
	} {
		if _, err := storage.CreateCollection(ctx, src, "", name, "", query); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	src.Close()

	// The target numbers its rows otherwise: a match and a tournament of its
	// own come first, and it already holds the source's tournament by name.
	own := domain.InitializePosition()
	own.Dice = [2]int{4, 2}
	if _, err := target.Positions().Save(ctx, scope, &own); err != nil {
		t.Fatal(err)
	}
	if _, err := target.Matches().Save(ctx, scope, &domain.Match{Player1Name: "X", Player2Name: "Y",
		MatchLength: 3, MatchHash: "h-own"}); err != nil {
		t.Fatal(err)
	}
	if _, err := target.Tournaments().Create(ctx, scope, "Le mien", "", ""); err != nil {
		t.Fatal(err)
	}
	heldTour, err := target.Tournaments().Create(ctx, scope, "Open de Lyon", "", "")
	if err != nil {
		t.Fatal(err)
	}

	sum, err := DBImporter{S: target}.Import(ctx, scope, Source{Path: srcPath}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Matches != 1 || sum.SkippedDuplicates != 0 || sum.Tournaments != 0 ||
		sum.Transcriptions != 1 || sum.Aliases != 1 || len(sum.Unreadable) != 0 {
		t.Fatalf("summary = %+v, want 1 match, 0 tournament created, 1 draft, 1 alias", sum)
	}
	if !slices.Equal(sum.LivingCollectionsFrozen, []string{"Fantôme"}) {
		t.Errorf("frozen = %v, want [Fantôme]", sum.LivingCollectionsFrozen)
	}

	newMatch, found, err := target.Matches().FindByHash(ctx, scope, "h-bob", "")
	if err != nil || !found {
		t.Fatalf("imported match not found by its hash: %v", err)
	}
	m, err := target.Matches().Get(ctx, scope, newMatch)
	if err != nil {
		t.Fatal(err)
	}
	if m.Comment != "note de Bob" || m.TournamentID == nil || *m.TournamentID != heldTour {
		t.Errorf("imported match = comment %q, tournament %v; want its note, filed in the held tournament %d",
			m.Comment, m.TournamentID, heldTour)
	}
	var movePositions []int64
	for mv, err := range target.Matches().MovesByMatch(ctx, scope, newMatch) {
		if err != nil {
			t.Fatal(err)
		}
		movePositions = append(movePositions, mv.PositionID)
	}
	if len(movePositions) != 2 || slices.Contains(movePositions, 0) {
		t.Fatalf("imported moves reach %v, want two target positions", movePositions)
	}

	var drafts []*storage.Transcription
	for d, err := range target.Transcriptions().List(ctx, scope) {
		if err != nil {
			t.Fatal(err)
		}
		drafts = append(drafts, d)
	}
	if len(drafts) != 1 || drafts[0].MatchID != newMatch {
		t.Errorf("drafts = %+v, want one, linked to match %d", drafts, newMatch)
	}

	byName := map[string]storage.Collection{}
	for c, err := range target.Collections().List(ctx, scope) {
		if err != nil {
			t.Fatal(err)
		}
		byName[c.Name] = *c
	}
	if q := byName["Du match"].FilterQuery; q != fmt.Sprintf("s ma%d", newMatch) {
		t.Errorf("living match query = %q, want s ma%d", q, newMatch)
	}
	if q := byName["Du tournoi"].FilterQuery; q != fmt.Sprintf("s tn%d", heldTour) {
		t.Errorf("living tournament query = %q, want s tn%d", q, heldTour)
	}
	ev, err := storage.EvaluateCollection(ctx, target, scope, byName["Du match"].ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	got := slices.Sorted(slices.Values(ev.PositionIDs))
	want := slices.Sorted(slices.Values(movePositions))
	if !slices.Equal(got, want) {
		t.Errorf("renumbered living collection selects %v, want the match's positions %v", got, want)
	}
	if c := byName["Fantôme"]; c.ID == 0 || c.FilterQuery != "" {
		t.Errorf("frozen collection = %+v, want a hand-made list", c)
	}

	again, err := DBImporter{S: target}.Import(ctx, scope, Source{Path: srcPath}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if again.Matches != 0 || again.SkippedDuplicates != 1 || again.Tournaments != 0 ||
		again.Transcriptions != 0 || again.Aliases != 0 || again.Collections != 0 {
		t.Errorf("second import = %+v, want nothing new and the match known", again)
	}
}
