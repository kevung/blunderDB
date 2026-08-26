package ingest

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// xgLuckFixture is an analysed XG match whose rolls carry luck. The same match
// exists as a gnuBG .sgf next to it, which is how the sign convention and the
// unit were checked in the first place: LU[-0.00537] against ErrLuck -0.00305,
// roll for roll, 189 rolls on both sides.
func xgLuckFixture() string {
	return filepath.Join("..", "..", "..", "testdata", "charlot1-charlot2_7p_2025-11-08-2305.xg")
}

// TestMapXGCarriesLuck pins the re-read of MoveEntry.ErrLuck: like the study
// flags, luck never reaches xgparser.Match, so ingest recovers it from the raw
// segments and it can just as easily land on the wrong decision.
func TestMapXGCarriesLuck(t *testing.T) {
	g, err := MapXG(xgLuckFixture())
	if err != nil {
		t.Fatalf("MapXG: %v", err)
	}

	var withLuck, cubeWithLuck, checkerTotal int
	var first *int32
	for gi := range g.Games {
		for mi := range g.Games[gi].Moves {
			mv := g.Games[gi].Moves[mi].Move
			if mv.MoveType == "checker" {
				checkerTotal++
			}
			if mv.LuckMP == nil {
				continue
			}
			withLuck++
			if mv.MoveType == "cube" {
				cubeWithLuck++
			}
			if first == nil {
				v := *mv.LuckMP
				first = &v
			}
		}
	}

	// Every checker decision of this match was analysed, so every one of them
	// carries luck — including the five rolls whose luck is exactly zero, which
	// are neutral rolls and not missing data.
	if withLuck != checkerTotal {
		t.Errorf("rolls carrying luck: got %d, want %d (one per checker decision)", withLuck, checkerTotal)
	}
	if checkerTotal != 189 {
		t.Errorf("checker decisions in the fixture: got %d, want 189", checkerTotal)
	}
	// A cube decision has no dice of its own, so it must never carry luck.
	if cubeWithLuck != 0 {
		t.Errorf("cube decisions carrying luck: got %d, want 0", cubeWithLuck)
	}
	// The first roll: ErrLuck -0.00305 in the file, rounded to millipoints and
	// keeping its sign. A conversion that dropped the sign or the scale would
	// still look plausible in aggregate, so pin one value exactly.
	if first == nil {
		t.Fatal("no roll carried luck at all")
	}
	if *first != -3 {
		t.Errorf("first roll's luck: got %d, want -3 millipoints", *first)
	}
}

// TestLuckOrNothing covers the other half of the rule: telling a neutral roll
// from a match whose luck was never computed, which XG writes identically as 0.
// A single zero says nothing; a match of nothing but zeroes says the analysis
// never ran, because dice are not neutral 189 times in a row.
func TestLuckOrNothing(t *testing.T) {
	cases := []struct {
		name string
		luck map[flagKey]int32
		want bool // true when the data should be kept
	}{
		{"all zero is no data at all", map[flagKey]int32{{0, 0}: 0, {0, 1}: 0, {0, 2}: 0}, false},
		{"empty is no data", map[flagKey]int32{}, false},
		{"one real value keeps the zeroes around it", map[flagKey]int32{{0, 0}: 0, {0, 1}: -3, {0, 2}: 0}, true},
		{"a single unlucky roll is data", map[flagKey]int32{{0, 0}: -329}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := luckOrNothing(c.luck)
			if c.want && got == nil {
				t.Errorf("luck was dropped, want it kept")
			}
			if !c.want && got != nil {
				t.Errorf("luck was kept (%v), want it dropped as unknown", got)
			}
		})
	}
}

// TestMapXGLuckSurvivesToStorage is the end-to-end half: the value read from
// the segments must reach the database and come back with its sign, rather than
// being flattened to zero on the way through.
func TestMapXGLuckSurvivesToStorage(t *testing.T) {
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
	res, err := WriteMatch(ctx, tx, "", g, nil)
	if err != nil {
		t.Fatalf("WriteMatch: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	var known, positive, negative int
	for mv, err := range s.Matches().MovesByMatch(ctx, "", res.MatchID) {
		if err != nil {
			t.Fatalf("MovesByMatch: %v", err)
		}
		if mv.LuckMP == nil {
			continue
		}
		known++
		switch {
		case *mv.LuckMP > 0:
			positive++
		case *mv.LuckMP < 0:
			negative++
		}
	}
	if known != 189 {
		t.Errorf("rolls with a known luck in storage: got %d, want 189", known)
	}
	// Both signs must survive: a column or scan that lost the sign would leave
	// a match where nobody was ever unlucky.
	if positive == 0 || negative == 0 {
		t.Errorf("signed luck lost on the way to storage: %d lucky, %d unlucky rolls", positive, negative)
	}
}
