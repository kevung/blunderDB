package ingest

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// replacePos returns a distinct checker position per n, so each move of the
// graphs below hashes onto its own row.
func replacePos(n int) *domain.Position {
	p := domain.InitializePosition()
	p.DecisionType = domain.CheckerAction
	p.Score = [2]int{n, 0}
	return &p
}

// replaceGraph builds a one-game graph whose moves visit the given positions,
// in order.
func replaceGraph(m domain.Match, positions ...*domain.Position) *MatchGraph {
	g := &MatchGraph{
		Match: m,
		Games: []GameGraph{{Game: domain.Game{GameNumber: 1, Winner: 0, PointsWon: 1}}},
	}
	for i, p := range positions {
		g.Games[0].Moves = append(g.Games[0].Moves, MoveGraph{
			Move: domain.Move{
				MoveNumber: int32(i + 1), MoveType: "checker", Player: 1,
				Dice: [2]int32{3, 1}, CheckerMove: "8/5 6/5",
			},
			Position: p,
		})
	}
	return g
}

// TestWriteMatchReplace is the replacement mode of ADR-0045 §2-3: saving a
// corrected transcription rewrites the match it already produced instead of
// making a second one. The match keeps its id — a tournament, a collection and
// the last-visited position all point at it — its games and moves are rewritten
// from scratch, the positions of unchanged Actions land back on their existing
// rows by deduplication, and the ones only the corrected Action reached are
// purged by the ordinary retention rule. Nothing goes to the trash: replacing a
// match twenty times during a review is not twenty deletions.
func TestWriteMatchReplace(t *testing.T) {
	ctx := context.Background()
	s, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	// The first save. Every position below occurs in the match; they differ
	// only in what else will hold them when the correction drops them.
	kept := replacePos(1)      // still played in the corrected version
	plain := replacePos(8)     // idem, and held by nothing else at all
	corrected := replacePos(2) // the misread die: held by nothing
	inCollection := replacePos(3)
	inDeck := replacePos(4)
	userCommented := replacePos(5)
	individual := replacePos(6)
	individual.IndividuallyImported = true
	flagged := replacePos(7)
	flagged.Flagged = true

	v1 := replaceGraph(domain.Match{
		Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7,
		Event: "Open de Paris", GameCount: 1,
		MatchHash: "draft-1", CanonicalHash: "draft-c1",
	}, kept, plain, corrected, inCollection, inDeck, userCommented, individual, flagged)

	res1 := writeGraph(t, s, v1)
	if res1.MatchID == 0 || res1.Skipped || res1.Replaced {
		t.Fatalf("first write: res = %+v", res1)
	}
	matchID := res1.MatchID

	// The ids WriteMatch settled on, read back off the graph it filled in.
	id := func(i int) int64 { return v1.Games[0].Moves[i].Move.PositionID }
	keptID, plainID, correctedID := id(0), id(1), id(2)
	collectionID, deckPosID, commentedID := id(3), id(4), id(5)
	individualID, flaggedID := id(6), id(7)

	// An analysis on a position both versions play. It must still be there
	// afterwards: purging before the corrected rows are written would delete
	// this position (nothing holds it at that instant), cascade its analysis
	// away and put the position back under a new id — which is precisely the
	// re-analysis ADR-0045 §8 says a re-save must not trigger.
	if err := s.Analyses().Save(ctx, "", plainID, &domain.PositionAnalysis{}); err != nil {
		t.Fatalf("Save analysis: %v", err)
	}

	// What the user does with the match during the review.
	coll, err := s.Collections().Create(ctx, "", "keep", "")
	if err != nil {
		t.Fatalf("Create collection: %v", err)
	}
	if err := s.Collections().AddPosition(ctx, "", coll, collectionID); err != nil {
		t.Fatalf("AddPosition: %v", err)
	}
	deck, err := s.Anki().CreateDeck(ctx, "", "deck", "", domain.AnkiSourceSearch, 0, "")
	if err != nil {
		t.Fatalf("CreateDeck: %v", err)
	}
	if err := s.Anki().SyncWithPositions(ctx, "", deck, []int64{deckPosID}); err != nil {
		t.Fatalf("SyncWithPositions: %v", err)
	}
	if _, err := s.Comments().Add(ctx, "", commentedID, "I keep missing this one"); err != nil {
		t.Fatalf("Add comment: %v", err)
	}
	// A note on a position the correction does NOT touch: it must come back
	// with its position, which is the whole point of deduplicating.
	if _, err := s.Comments().Add(ctx, "", keptID, "watch the 5 point"); err != nil {
		t.Fatalf("Add comment on kept position: %v", err)
	}

	// The corrected transcription: one position gone, one new, the rest of the
	// header restated.
	added := replacePos(9)
	v2 := replaceGraph(domain.Match{
		Player1Name: "Alice", Player2Name: "Robert", MatchLength: 7,
		Event: "Open de Paris", GameCount: 1,
		MatchHash: "draft-2", CanonicalHash: "draft-c2",
	}, kept, plain, added)
	v2.ReplaceMatchID = matchID

	res2 := writeGraph(t, s, v2)
	if !res2.Replaced {
		t.Errorf("res.Replaced = false, want true (%+v)", res2)
	}
	if res2.Skipped || res2.Enriched {
		t.Errorf("a replacement is neither a skip nor an enrich: %+v", res2)
	}
	if res2.MatchID != matchID {
		t.Fatalf("replacement changed the match id: %d, want %d", res2.MatchID, matchID)
	}

	// One match, one game, two moves: the old rows are gone, not doubled.
	counts, err := s.Metadata().Counts(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if counts.Matches != 1 || counts.Games != 1 || counts.Moves != 3 {
		t.Errorf("counts = matches %d games %d moves %d, want 1/1/3",
			counts.Matches, counts.Games, counts.Moves)
	}

	m, err := s.Matches().Get(ctx, "", matchID)
	if err != nil {
		t.Fatalf("Get match: %v", err)
	}
	if m.Player2Name != "Robert" {
		t.Errorf("header not restated: player2 = %q, want %q", m.Player2Name, "Robert")
	}

	// The unchanged position kept its row — same id, so the comment written on
	// it during the review is still on it.
	if got := v2.Games[0].Moves[0].Move.PositionID; got != keptID {
		t.Errorf("unchanged position moved rows: id %d, want %d (dedup by Zobrist)", got, keptID)
	}
	if got := v2.Games[0].Moves[1].Move.PositionID; got != plainID {
		t.Errorf("unchanged position held by nothing moved rows: id %d, want %d", got, plainID)
	}
	if _, err := s.Analyses().Load(ctx, "", plainID); err != nil {
		t.Errorf("analysis of an unchanged position lost by the replacement: %v", err)
	}
	var texts []string
	for c, err := range s.Comments().ByPosition(ctx, "", keptID) {
		if err != nil {
			t.Fatalf("ByPosition: %v", err)
		}
		texts = append(texts, c.Text)
	}
	if len(texts) != 1 || texts[0] != "watch the 5 point" {
		t.Errorf("comment on the unchanged position lost: %q", texts)
	}

	// The new position exists; the corrected one, held by nothing, is gone;
	// everything the user did something with survives.
	addedID := v2.Games[0].Moves[2].Move.PositionID
	for _, tc := range []struct {
		name string
		id   int64
		kept bool
	}{
		{"still played after the correction", keptID, true},
		{"still played, and held by nothing else", plainID, true},
		{"written by the correction", addedID, true},
		{"dropped by the correction, held by nothing", correctedID, false},
		{"dropped, but in a collection", collectionID, true},
		{"dropped, but in an Anki deck", deckPosID, true},
		{"dropped, but commented by the user", commentedID, true},
		{"dropped, but individually imported", individualID, true},
		{"dropped, but flagged", flaggedID, true},
	} {
		_, err := s.Positions().Load(ctx, "", tc.id)
		switch {
		case tc.kept && err != nil:
			t.Errorf("position %s was purged: %v", tc.name, err)
		case !tc.kept && !errors.Is(err, storage.ErrNotFound):
			t.Errorf("position %s survived: got %v, want ErrNotFound", tc.name, err)
		}
	}

	// A replacement is housekeeping, not a gesture: nothing enters the trash.
	if n, err := s.Trash().Count(ctx, ""); err != nil {
		t.Fatalf("Trash Count: %v", err)
	} else if n != 0 {
		t.Errorf("trash holds %d entries, want 0 — a replacement never snapshots", n)
	}
}
