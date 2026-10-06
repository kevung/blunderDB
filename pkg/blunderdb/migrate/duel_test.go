package migrate

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// TestRunCarriesTheDuel: a migration moves the origin of a Match played
// here onto the new Match id, its moves with the durations of their
// decisions, and the Duels in suspense with their seed.
func TestRunCarriesTheDuel(t *testing.T) {
	ctx := context.Background()
	open := func() storage.Storage {
		s, err := sqlite.Open(ctx, ":memory:", nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := s.Migrate(ctx); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { s.Close() })
		return s
	}
	src, dst := open(), open()
	matchID, err := src.Matches().Save(ctx, "", &domain.Match{Player1Name: "A", Player2Name: "B", MatchLength: 3, MatchHash: "h-duel"})
	if err != nil {
		t.Fatal(err)
	}
	if err := src.Duels().SetOrigin(ctx, "", &storage.MatchOrigin{MatchID: matchID, Start: "xgid", DiceSeed: "ab",
		OverTime: 1, Cadence: `{"reservePerPoint":120,"delay":12}`}); err != nil {
		t.Fatal(err)
	}
	gameID, err := src.Matches().CreateGame(ctx, "", &domain.Game{MatchID: matchID, GameNumber: 1})
	if err != nil {
		t.Fatal(err)
	}
	decision := int64(7300)
	for i, mv := range []domain.Move{
		{MoveType: "cube", Player: 1, CubeAction: "Double", DecisionMS: &decision},
		{MoveType: "cube", Player: -1, CubeAction: "Take"},
	} {
		mv.GameID, mv.MoveNumber = gameID, int32(i)
		if _, err := src.Matches().CreateMove(ctx, "", &mv); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := src.Duels().Save(ctx, "", &storage.Duel{FormatVersion: "1", Document: "{}", DiceSeed: "cd", Open: true}); err != nil {
		t.Fatal(err)
	}

	rep, err := Run(ctx, src, dst, "", Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rep.Duels != 1 {
		t.Errorf("report counts %d duels, want 1", rep.Duels)
	}
	var newID int64
	for m, err := range dst.Matches().List(ctx, "", storage.MatchListOpts{}) {
		if err == nil && m.MatchHash == "h-duel" {
			newID = m.ID
		}
	}
	if newID == 0 {
		t.Fatal("the match did not migrate")
	}
	if o, err := dst.Duels().Origin(ctx, "", newID); err != nil || o.Start != "xgid" || o.DiceSeed != "ab" ||
		o.OverTime != 1 || o.Cadence != `{"reservePerPoint":120,"delay":12}` {
		t.Errorf("origin after migration = %+v, %v", o, err)
	}
	// The durations of the decisions travel with the moves; an unknown one
	// stays unknown.
	var moves []*domain.Move
	for mv, err := range dst.Matches().MovesByMatch(ctx, "", newID) {
		if err != nil {
			t.Fatal(err)
		}
		moves = append(moves, mv)
	}
	if len(moves) != 2 || moves[0].DecisionMS == nil || *moves[0].DecisionMS != decision || moves[1].DecisionMS != nil {
		t.Errorf("migrated moves = %+v", moves)
	}
	var seeds []string
	var ids []int64
	for e, err := range dst.Duels().List(ctx, "") {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, e.ID)
	}
	for _, id := range ids {
		d, err := dst.Duels().Get(ctx, "", id)
		if err != nil {
			t.Fatal(err)
		}
		seeds = append(seeds, d.DiceSeed)
		if d.Open {
			t.Errorf("duel %d arrived open; a migrated Duel arrives in suspense", id)
		}
	}
	if len(seeds) != 1 || seeds[0] != "cd" {
		t.Errorf("migrated duels' seeds = %v, want [cd]", seeds)
	}
}
