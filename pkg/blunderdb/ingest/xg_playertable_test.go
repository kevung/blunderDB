package ingest

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// TestPlayerTableOnRealMatch runs the players table over a real imported match
// rather than a hand-built fixture. Two things can only be checked this way.
//
// The first is the win/loss derivation. No winner is stored on a match, so it
// is recomputed from per-game points and the seat encoding of game.winner — a
// convention that is easy to state backwards, and that a fixture written from
// the same assumption would happily confirm. This match ended 7-x, so exactly
// one of its two players must come out with a win and the other with a loss.
//
// The second is that luck reaches the table: 189 analysed rolls, split between
// the two players.
func TestPlayerTableOnRealMatch(t *testing.T) {
	ctx := context.Background()
	s, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}
	defer s.Close()

	g, err := MapXG(xgLuckFixture())
	if err != nil {
		t.Fatalf("MapXG: %v", err)
	}
	tx, err := s.BeginTx(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if _, err := WriteMatch(ctx, tx, "", g, nil); err != nil {
		t.Fatalf("WriteMatch: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	rows, err := s.Stats().PlayerTable(ctx, "", storage.StatsFilter{DecisionType: -1})
	if err != nil {
		t.Fatalf("PlayerTable: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("players: got %d rows, want 2", len(rows))
	}

	var wins, losses, luckRolls int
	for _, r := range rows {
		if r.Matches != 1 {
			t.Errorf("%s: got %d matches, want 1", r.Name, r.Matches)
		}
		if r.Decisions == 0 {
			t.Errorf("%s: no counted decision in an analysed match", r.Name)
		}
		if r.PR <= 0 {
			t.Errorf("%s: got PR %v, want > 0", r.Name, r.PR)
		}
		if r.SnowieER <= 0 {
			t.Errorf("%s: got Snowie ER %v, want > 0", r.Name, r.SnowieER)
		}
		if _, ok := r.LuckRateMP(); !ok {
			t.Errorf("%s: no luck measured on an analysed match", r.Name)
		}
		wins += r.Wins
		losses += r.Losses
		luckRolls += r.LuckRolls
	}

	// A finished match gives exactly one win and one loss — never two of
	// either, which is what a seat-encoding mistake would produce.
	if wins != 1 || losses != 1 {
		t.Errorf("a finished match: got %d wins and %d losses across both players, want 1 and 1",
			wins, losses)
	}
	if luckRolls != 189 {
		t.Errorf("measured rolls across both players: got %d, want 189", luckRolls)
	}

	// The Snowie rate shares its denominator between the players, so the two
	// rates must add up to the rate of the match as a whole — the same
	// property the global Snowie ER has, now read through the table.
	all, err := s.Stats().Compute(ctx, "", storage.StatsFilter{DecisionType: -1})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	sum := rows[0].SnowieER + rows[1].SnowieER
	if diff := sum - all.SnowieGlobal; diff > 1e-9 || diff < -1e-9 {
		t.Errorf("Snowie ER per player sums to %v, want the unfiltered %v", sum, all.SnowieGlobal)
	}
}
