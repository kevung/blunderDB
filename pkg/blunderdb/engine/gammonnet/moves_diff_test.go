package gammonnet

import (
	"math"
	"math/rand"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// randomBoard builds a structurally valid domain position: fifteen checkers a
// side, no point holding both colours. Many are unreachable by legal play, and
// that is deliberate — the two generators implement the same rules and must
// agree on any valid board, and unreachable boards reach the bar, bear-off and
// dance branches far more often than a corpus of real positions does.
func randomBoard(rng *rand.Rand, onRoll int) domain.Position {
	var p domain.Position
	p.PlayerOnRoll = onRoll
	p.Score = [2]int{-1, -1}

	owner := make([]int, domain.NumPoints+2) // 0 = free
	for _, colour := range []int{domain.White, domain.Black} {
		left := 15
		// Some checkers off, some on the bar, the rest on the board.
		off := rng.Intn(4)
		if rng.Intn(3) == 0 {
			off = rng.Intn(15)
		}
		left -= off
		p.Board.Bearoff[colour] = off

		bar := 0
		if rng.Intn(4) == 0 && left > 0 {
			bar = 1 + rng.Intn(min(3, left))
			left -= bar
		}
		barIdx := domain.WhiteBar
		if colour == domain.Black {
			barIdx = domain.BlackBar
		}
		p.Board.Points[barIdx] = domain.Point{Checkers: bar, Color: colour}

		for left > 0 {
			idx := 1 + rng.Intn(domain.NumPoints)
			if owner[idx] != 0 && owner[idx] != colour+1 {
				continue
			}
			n := 1 + rng.Intn(min(5, left))
			owner[idx] = colour + 1
			p.Board.Points[idx].Color = colour
			p.Board.Points[idx].Checkers += n
			left -= n
		}
	}
	return p
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// boardKey identifies a resulting position by its checkers alone. The turn is
// excluded on purpose: this generator switches it on the result and domain's
// does not, and that difference is a convention, not a disagreement.
type boardKey struct {
	points [NumPoints]int8
	bar    [2]uint8
	off    [2]uint8
}

func keyOf(p *Position) boardKey {
	return boardKey{points: p.Points, bar: p.Bar, off: p.Off}
}

// Two implementations of the rules are acceptable. Two answers are not.
//
// Over a corpus of positions × all 21 distinct rolls, the search's generator and
// domain.LegalMoves must produce exactly the same SET of resulting positions —
// including agreeing on which positions are a dance.
func TestGeneratorsAgreeWithDomain(t *testing.T) {
	rng := rand.New(rand.NewSource(20260828))
	var g Generator
	plays := make([]Play, MaxPlays)

	const boards = 400
	var checked, dances int

	for b := 0; b < boards; b++ {
		onRoll := domain.White
		if b%2 == 1 {
			onRoll = domain.Black
		}
		dp := randomBoard(rng, onRoll)
		gp, err := FromDomain(&dp)
		if err != nil {
			t.Fatalf("board %d: %v", b, err)
		}

		for d1 := 1; d1 <= 6; d1++ {
			for d2 := d1; d2 <= 6; d2++ {
				dp.Dice = [2]int{d1, d2}

				n := g.LegalPlays(&gp, d1, d2, plays)
				if n < 0 {
					t.Fatalf("board %d roll %d-%d: generation refused a valid input", b, d1, d2)
				}
				mine := map[boardKey]bool{}
				for i := 0; i < n; i++ {
					mine[keyOf(&plays[i].Result)] = true
				}
				if len(mine) != n {
					t.Fatalf("board %d roll %d-%d: %d plays but %d distinct results — duplicates returned",
						b, d1, d2, n, len(mine))
				}

				theirs := map[boardKey]bool{}
				for _, lp := range domain.LegalMoves(&dp) {
					res := lp.Result
					res.PlayerOnRoll = dp.PlayerOnRoll
					conv, err := FromDomain(&res)
					if err != nil {
						t.Fatalf("board %d roll %d-%d: converting a domain result: %v", b, d1, d2, err)
					}
					theirs[keyOf(&conv)] = true
				}

				checked++
				if len(mine) == 0 && len(theirs) == 0 {
					dances++
					continue
				}
				if !sameSet(mine, theirs) {
					t.Fatalf("board %d roll %d-%d disagree\n  XGID-ish board: %v\n  search generator: %d results\n  domain.LegalMoves: %d results\n  only in search: %d, only in domain: %d",
						b, d1, d2, dp.Board.Points, len(mine), len(theirs),
						countMissing(mine, theirs), countMissing(theirs, mine))
				}
			}
		}
	}
	t.Logf("%d (board, roll) pairs agreed; %d were a dance on both sides", checked, dances)
	if dances == 0 {
		t.Error("no dance in the whole corpus — the bar branch is not being exercised")
	}
}

func sameSet(a, b map[boardKey]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

func countMissing(a, b map[boardKey]bool) int {
	n := 0
	for k := range a {
		if !b[k] {
			n++
		}
	}
	return n
}

// The 21 distinct rolls and their weights.
//
// The C header claims they "sum to exactly 1 — 6 * (1/36) + 15 * (2/36)", and
// that is true of the GROUPED expression. Summed one roll at a time, in roll
// order, they give 1.0000000000000002. Both are correct; they are different
// computations.
//
// The distinction is not pedantry, it is the requirement: the search
// accumulates `sum += weight * best` over the rolls in ascending index order,
// in float64. Reproducing the total means reproducing that order, so this test
// pins the order and the type, and checks the total to within a few ulps rather
// than asserting an equality that only one of the two groupings satisfies.
func TestRollTableMatchesTheReference(t *testing.T) {
	rolls := buildRolls()
	if len(rolls) != 21 {
		t.Fatalf("%d rolls, want 21", len(rolls))
	}
	// Order: (1,1)(1,2)...(1,6)(2,2)...(2,6)(3,3)...(6,6).
	if rolls[0] != roll(1, 1, 1.0/36.0) || rolls[1] != roll(1, 2, 2.0/36.0) {
		t.Errorf("roll order starts %v %v", rolls[0], rolls[1])
	}
	if rolls[6] != roll(2, 2, 1.0/36.0) || rolls[20] != roll(6, 6, 1.0/36.0) {
		t.Errorf("roll order ends %v ... %v", rolls[6], rolls[20])
	}

	var sum float64
	for _, r := range rolls {
		sum += r.weight
	}
	if d := math.Abs(sum - 1); d > 4*math.Nextafter(1, 2)-4 {
		t.Errorf("weights sum to %v (off by %v)", sum, sum-1)
	}
	var doubles int
	for _, r := range rolls {
		if r.d1 == r.d2 {
			doubles++
		}
	}
	if doubles != 6 {
		t.Errorf("%d doubles, want 6", doubles)
	}
}

func roll(d1, d2 int8, w float64) diceRoll { return diceRoll{d1: d1, d2: d2, weight: w} }
