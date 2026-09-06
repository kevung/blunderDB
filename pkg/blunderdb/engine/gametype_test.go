package engine

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// board builds a position from a compact description: point → (count, colour).
func gtBoard(black, white map[int]int, offBlack, offWhite int) domain.Board {
	var b domain.Board
	for i := range b.Points {
		b.Points[i] = domain.Point{Checkers: 0, Color: domain.None}
	}
	for p, n := range black {
		b.Points[p] = domain.Point{Checkers: n, Color: domain.Black}
	}
	for p, n := range white {
		b.Points[p] = domain.Point{Checkers: n, Color: domain.White}
	}
	b.Bearoff = [2]int{offBlack, offWhite}
	return b
}

func gtPos(b domain.Board, onRoll int) domain.Position {
	return domain.Position{Board: b, PlayerOnRoll: onRoll, Score: [2]int{-1, -1}}
}

// TestClassifyGameType_SourcedBoundaries pins the three rules that are not
// this project's inventions: they are gnubg's, and P5 sources them.
func TestClassifyGameType_SourcedBoundaries(t *testing.T) {
	over := gtPos(gtBoard(map[int]int{}, map[int]int{24: 3}, 15, 12), domain.Black)
	if got := ClassifyGameType(&over); got != domain.TypeOver {
		t.Errorf("over: got %v", got)
	}

	// Black's rearmost (6) is below White's rearmost (10): they have crossed.
	race := gtPos(gtBoard(map[int]int{6: 5, 5: 5, 4: 5}, map[int]int{10: 5, 12: 5, 14: 5}, 0, 0), domain.Black)
	if got := ClassifyGameType(&race); got != domain.TypeRace {
		t.Errorf("race: got %v", got)
	}

	// Crunch: Black has thirteen checkers piled on its points 1 and 2 and two
	// left elsewhere — at most six active, gnubg's threshold.
	crunch := gtPos(gtBoard(map[int]int{1: 7, 2: 6, 20: 2}, map[int]int{19: 5, 18: 5, 8: 5}, 0, 0), domain.Black)
	if got := ClassifyGameType(&crunch); got != domain.TypeCrunch {
		t.Errorf("crunch: got %v", got)
	}
}

// TestClassifyGameType_Plans walks one position per human plan. These are
// CONVENTIONS, and the test's job is to keep them from drifting silently, not
// to claim they are the only reading.
func TestClassifyGameType_Plans(t *testing.T) {
	cases := []struct {
		name  string
		black map[int]int
		white map[int]int
		want  domain.GameType
	}{
		{
			// Two anchors in White's home board (19..24 from Black's side).
			name:  "backgame",
			black: map[int]int{23: 2, 22: 2, 13: 5, 8: 3, 6: 3},
			white: map[int]int{1: 2, 12: 5, 17: 3, 19: 5},
			want:  domain.TypeBackgame,
		},
		{
			// One anchor, on White's ace point, and far behind in the race.
			name:  "acepoint",
			black: map[int]int{24: 2, 13: 5, 8: 4, 6: 4},
			white: map[int]int{19: 5, 20: 4, 21: 4, 12: 2},
			want:  domain.TypeAcePoint,
		},
		{
			// Four home points made and White on the bar.
			name:  "blitz",
			black: map[int]int{1: 2, 2: 2, 3: 2, 4: 2, 13: 5, 8: 2},
			white: map[int]int{domain.WhiteBar: 1, 19: 5, 20: 4, 17: 5},
			want:  domain.TypeBlitz,
		},
		{
			// One high anchor for Black (20), none for White.
			name:  "holding",
			black: map[int]int{20: 2, 13: 5, 8: 4, 6: 4},
			white: map[int]int{19: 4, 21: 4, 12: 4, 17: 3},
			want:  domain.TypeHolding,
		},
		{
			// A high anchor each: Black on 20, White on 5.
			name:  "mutualholding",
			black: map[int]int{20: 2, 13: 5, 8: 4, 6: 4},
			white: map[int]int{5: 2, 19: 5, 12: 5, 17: 3},
			want:  domain.TypeMutualHolding,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := gtPos(gtBoard(tc.black, tc.white, 0, 0), domain.Black)
			if got := ClassifyGameType(&p); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// TestClassifyGameType_IsMirrorSymmetric is the property that matters most:
// the same physical position, seen from either side, must classify by the same
// rules. The label differs (it describes the side on roll), but the rule that
// produced it must be the same one — which is what mirroring guarantees.
func TestClassifyGameType_IsMirrorSymmetric(t *testing.T) {
	b := gtBoard(map[int]int{20: 2, 13: 5, 8: 4, 6: 4}, map[int]int{19: 4, 21: 4, 12: 4, 17: 3}, 0, 0)
	black := gtPos(b, domain.Black)
	white := gtPos(b, domain.White)
	if ClassifyGameType(&black) != domain.TypeHolding {
		t.Fatalf("black on roll: got %v", ClassifyGameType(&black))
	}
	// White holds no high anchor here, so White is not the one holding.
	if got := ClassifyGameType(&white); got == domain.TypeHolding {
		t.Errorf("white on roll must not inherit Black's plan: got %v", got)
	}
}

// TestClassifyGameType_NilAndEmpty keeps the classifier total: it is called on
// every write, and a panic there would refuse an import.
func TestClassifyGameType_NilAndEmpty(t *testing.T) {
	if got := ClassifyGameType(nil); got != domain.TypeUnknown {
		t.Errorf("nil: got %v", got)
	}
	empty := gtPos(gtBoard(nil, nil, 0, 0), domain.Black)
	if got := ClassifyGameType(&empty); got == domain.TypeUnknown {
		t.Errorf("an empty board is still classified, got %v", got)
	}
}
