package ingest

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// diceGraph is a two-game match between p1 and p2 with the given dice, one
// checker move per roll, hashed as an importer would (the names are in both
// hashes).
func diceGraph(p1, p2 string, games ...[][2]int32) *MatchGraph {
	g := &MatchGraph{Match: domain.Match{Player1Name: p1, Player2Name: p2, MatchLength: 5,
		MatchHash: "h-" + p1 + p2, CanonicalHash: "c-" + p1 + p2}}
	for gi, dice := range games {
		gg := GameGraph{Game: domain.Game{GameNumber: int32(gi + 1)}}
		for mi, d := range dice {
			gg.Moves = append(gg.Moves, MoveGraph{Move: domain.Move{MoveNumber: int32(mi + 1), MoveType: "checker", Player: 1, Dice: d}})
		}
		g.Match.MatchHash += string(rune('a' + len(dice)))
		g.Match.CanonicalHash += string(rune('a' + len(dice)))
		g.Games = append(g.Games, gg)
	}
	return g
}

func TestRenamedMatchIsSignalledNeverMerged(t *testing.T) {
	ctx := context.Background()
	s, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	g1 := [][2]int32{{3, 1}, {6, 4}, {2, 2}, {5, 1}}
	g2 := [][2]int32{{4, 2}, {6, 6}, {1, 3}}
	first := writeGraph(t, s, diceGraph("Alice Martin", "Bob Durand", g1, g2))
	if first.ProbableDuplicate != nil {
		t.Fatalf("first import signalled %+v", first.ProbableDuplicate)
	}

	// The same dice under other spellings, the seats swapped and the order
	// of each roll's dice reversed.
	rev := func(d [][2]int32) [][2]int32 {
		out := make([][2]int32, len(d))
		for i, x := range d {
			out[i] = [2]int32{x[1], x[0]}
		}
		return out
	}
	renamed := writeGraph(t, s, diceGraph("Durand B.", "Martin A.", rev(g1), rev(g2)))
	if renamed.Skipped || renamed.Enriched {
		t.Fatalf("renamed match must be stored as a match of its own: %+v", renamed)
	}
	p := renamed.ProbableDuplicate
	if p == nil || p.OtherID != first.MatchID || p.MatchID != renamed.MatchID || p.Kind != domain.DuplicateSameDice {
		t.Fatalf("ProbableDuplicate = %+v, want same_dice of #%d", p, first.MatchID)
	}

	// The match truncated in its second game, under the original names.
	truncated := writeGraph(t, s, diceGraph("Alice Martin", "Bob Durand", g1, g2[:1]))
	if truncated.ProbableDuplicate != nil || truncated.Skipped {
		t.Fatalf("a truncated match is a second match, not a same-dice suspect: %+v", truncated)
	}

	suspects, backfilled, err := FindDuplicateSuspects(ctx, s.Matches(), "")
	if err != nil {
		t.Fatal(err)
	}
	if backfilled != 0 {
		t.Errorf("backfilled %d dice hashes on matches imported with one", backfilled)
	}
	want := map[[3]int64]bool{
		{0, renamed.MatchID, first.MatchID}:     true,
		{1, first.MatchID, truncated.MatchID}:   true,
		{1, renamed.MatchID, truncated.MatchID}: true,
	}
	if len(suspects) != len(want) {
		t.Fatalf("suspects = %+v, want %d", suspects, len(want))
	}
	for _, s := range suspects {
		k := int64(0)
		if s.Kind == domain.DuplicateLonger {
			k = 1
		}
		if !want[[3]int64{k, s.MatchID, s.OtherID}] {
			t.Errorf("unexpected suspect %+v", s)
		}
	}
}

func TestFindDuplicateSuspectsBackfillsTheDiceHash(t *testing.T) {
	ctx := context.Background()
	s, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	res := writeGraph(t, s, diceGraph("Alice", "Bob", [][2]int32{{3, 1}, {6, 4}}))
	m, err := s.Matches().Get(ctx, "", res.MatchID)
	if err != nil {
		t.Fatal(err)
	}
	want := m.DiceHash
	if want == "" {
		t.Fatal("an imported match must carry its dice hash")
	}
	if err := s.Matches().SetDiceHash(ctx, "", res.MatchID, ""); err != nil {
		t.Fatal(err)
	}
	if _, n, err := FindDuplicateSuspects(ctx, s.Matches(), ""); err != nil || n != 1 {
		t.Fatalf("backfilled = %d, %v; want 1", n, err)
	}
	if m, _ := s.Matches().Get(ctx, "", res.MatchID); m.DiceHash != want {
		t.Errorf("dice hash recomputed from the stored moves = %q, want the import's %q", m.DiceHash, want)
	}
}
