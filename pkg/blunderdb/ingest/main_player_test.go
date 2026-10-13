package ingest

import (
	"context"
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

func openMemory(t *testing.T) storage.Storage {
	t.Helper()
	s, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func ptrF(v float64) *float64 { return &v }
func ptrI(v int) *int         { return &v }

// A file imported again after its match was swapped to seat the Main player
// first is still the same match, and the per-seat figures it brings land on
// the seat of the player they describe, not on the seat number the file
// gave them.
func TestReimportAfterMainPlayerSwapKeepsFiguresOnTheirPlayer(t *testing.T) {
	ctx := context.Background()
	s := openMemory(t)
	dice := [][2]int32{{3, 1}, {6, 4}, {2, 2}}

	first := writeGraph(t, s, diceGraph("Alice", "Kevin", dice))
	if _, err := storage.DesignateMainPlayer(ctx, s, "", "Kevin"); err != nil {
		t.Fatalf("designate: %v", err)
	}

	again := diceGraph("Alice", "Kevin", dice)
	again.Match.Player1Elo, again.Match.Player2Elo = ptrF(1500), ptrF(1800)
	again.Match.Player1Experience, again.Match.Player2Experience = ptrI(100), ptrI(900)
	res := writeGraph(t, s, again)
	if !res.Skipped || res.MatchID != first.MatchID {
		t.Fatalf("re-import = %+v, want the duplicate of #%d", res, first.MatchID)
	}
	m, err := s.Matches().Get(ctx, "", first.MatchID)
	if err != nil {
		t.Fatal(err)
	}
	if m.Player1Name != "Kevin" || m.Player2Name != "Alice" {
		t.Fatalf("seats %q / %q, want Kevin / Alice", m.Player1Name, m.Player2Name)
	}
	if m.Player1Elo == nil || *m.Player1Elo != 1800 || m.Player2Elo == nil || *m.Player2Elo != 1500 {
		t.Errorf("Elo %v / %v, want Kevin 1800, Alice 1500", m.Player1Elo, m.Player2Elo)
	}
	if m.Player1Experience == nil || *m.Player1Experience != 900 || m.Player2Experience == nil || *m.Player2Experience != 100 {
		t.Errorf("experience %v / %v, want Kevin 900, Alice 100", m.Player1Experience, m.Player2Experience)
	}
}

// seatsReversed reads both matches' names through the aliases, and a match
// between two spellings of one player is never reversed.
func TestSeatsReversedReadsAliases(t *testing.T) {
	ctx := context.Background()
	s := openMemory(t)
	if err := s.Aliases().Set(ctx, "", storage.AliasPlayer, "K. Unger", "Kevin"); err != nil {
		t.Fatal(err)
	}
	tx, err := s.BeginTx(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, c := range []struct {
		s1, s2, f1, f2 string
		want           bool
	}{
		{"Kevin", "Alice", "Alice", "Kevin", true},
		{"Kevin", "Alice", "Alice", "K. Unger", true},
		{"Kevin", "Alice", "K. Unger", "Alice", false},
		{"Kevin", "K. Unger", "K. Unger", "Kevin", false},
	} {
		got, err := seatsReversed(ctx, tx, "",
			&domain.Match{Player1Name: c.s1, Player2Name: c.s2},
			&domain.Match{Player1Name: c.f1, Player2Name: c.f2})
		if err != nil {
			t.Fatal(err)
		}
		if got != c.want {
			t.Errorf("stored %s–%s, file %s–%s: reversed %v, want %v", c.s1, c.s2, c.f1, c.f2, got, c.want)
		}
	}
}

// A same-dice suspect names the new match as it is stored, the Main player
// at seat 1, not in the file's order.
func TestProbableDuplicateNamesStoredSeats(t *testing.T) {
	ctx := context.Background()
	s := openMemory(t)
	if err := s.LibrarySettings().SetMainPlayer(ctx, "", "Kevin"); err != nil {
		t.Fatal(err)
	}
	dice := [][2]int32{{3, 1}, {6, 4}, {2, 2}, {5, 1}}
	first := writeGraph(t, s, diceGraph("Kevin", "Alice", dice))
	renamed := writeGraph(t, s, diceGraph("A. Martin", "Kevin", dice))
	if !renamed.SeatsSwapped {
		t.Fatalf("the Main player at seat 2 was not seated first: %+v", renamed)
	}
	p := renamed.ProbableDuplicate
	if p == nil || p.OtherID != first.MatchID {
		t.Fatalf("ProbableDuplicate = %+v, want same_dice of #%d", p, first.MatchID)
	}
	if p.Players != "Kevin – A. Martin" || p.OtherPlayers != "Kevin – Alice" {
		t.Errorf("suspect %q vs %q, want the stored seats", p.Players, p.OtherPlayers)
	}
	// Kevin sits first in both: the only reading left is seat for seat.
	want := []domain.AliasPairing{{Aliases: []domain.PlayerAlias{{Alias: "A. Martin", Canonical: "Alice"}}}}
	if !reflect.DeepEqual(p.Pairings, want) {
		t.Errorf("pairings %+v, want %+v", p.Pairings, want)
	}
}
