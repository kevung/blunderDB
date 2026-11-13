package reach

import (
	"math/rand"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/reach/reachtest"
)

func opening() domain.Position {
	var p domain.Position
	for i := range p.Board.Points {
		p.Board.Points[i] = domain.Point{Color: domain.None}
	}
	for pt, n := range map[int]int{24: 2, 13: 5, 8: 3, 6: 5} {
		p.Board.Points[pt] = domain.Point{Checkers: n, Color: domain.Black}
	}
	for pt, n := range map[int]int{1: 2, 12: 5, 17: 3, 19: 5} {
		p.Board.Points[pt] = domain.Point{Checkers: n, Color: domain.White}
	}
	return p
}

// randomGame plays one game to its end with random rolls and random legal
// plays, through domain.LegalMoves, and returns every position met, each
// with its side on roll set.
func randomGame(rng *rand.Rand) []domain.Position {
	p := opening()
	p.PlayerOnRoll = rng.Intn(2)
	var out []domain.Position
	for len(out) < 400 {
		out = append(out, p)
		p.Dice = [2]int{rng.Intn(6) + 1, rng.Intn(6) + 1}
		plays := domain.LegalMoves(&p)
		next := p
		if len(plays) > 0 {
			next = plays[rng.Intn(len(plays))].Result
		}
		next.PlayerOnRoll = 1 - p.PlayerOnRoll
		next.Dice = [2]int{}
		p = next
		if offCount(&p, domain.Black) == 15 || offCount(&p, domain.White) == 15 {
			out = append(out, p)
			return out
		}
	}
	return out
}

func offCount(p *domain.Position, color int) int {
	n := 0
	for i := 0; i <= 25; i++ {
		if p.Board.Points[i].Color == color {
			n += p.Board.Points[i].Checkers
		}
	}
	return 15 - n
}

// sidesOf is the (mover, opponent) pair of p as the generator sees it.
func sidesOf(p *domain.Position) state {
	s := Of(p)
	var black, white Side
	// Of loses which side is which; rebuild the pair in order.
	for i := 1; i <= 24; i++ {
		pt := p.Board.Points[i]
		switch {
		case pt.Checkers > 0 && pt.Color == domain.Black:
			black[i] = int8(pt.Checkers)
		case pt.Checkers > 0 && pt.Color == domain.White:
			white[25-i] = int8(pt.Checkers)
		}
	}
	black[25] = int8(p.Board.Points[domain.BlackBar].Checkers)
	white[25] = int8(p.Board.Points[domain.WhiteBar].Checkers)
	fillOff(&black)
	fillOff(&white)
	if NewStructure(black, white) != s {
		panic("sidesOf disagrees with Of")
	}
	if p.PlayerOnRoll == domain.White {
		return state{white, black}
	}
	return state{black, white}
}

// TestPlaysAgreeWithLegalMoves holds the generator to domain.LegalMoves, roll
// by roll, over positions from whole random games: the bar, hits, blocked
// rolls and bearing off all occur.
func TestPlaysAgreeWithLegalMoves(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	checked := 0
	for g := 0; g < 12; g++ {
		for _, p := range randomGame(rng) {
			for d1 := 1; d1 <= 6; d1++ {
				for d2 := d1; d2 <= 6; d2++ {
					q := p
					q.Dice = [2]int{d1, d2}
					want := map[state]bool{}
					legal := domain.LegalMoves(&q)
					if len(legal) == 0 {
						want[sidesOf(&p)] = true
					}
					for _, l := range legal {
						r := l.Result
						r.PlayerOnRoll = p.PlayerOnRoll
						want[sidesOf(&r)] = true
					}
					got := map[state]bool{}
					for _, s := range plays(sidesOf(&p), d1, d2) {
						got[s] = true
					}
					if len(got) != len(want) {
						t.Fatalf("roll %d-%d: %d plays, LegalMoves has %d", d1, d2, len(got), len(want))
					}
					for s := range want {
						if !got[s] {
							t.Fatalf("roll %d-%d: a legal play is missing", d1, d2)
						}
					}
					checked++
				}
			}
		}
	}
	if checked < 5000 {
		t.Fatalf("only %d position-rolls checked", checked)
	}
}

// TestDistanceAgreesWithTheOracle compares the distance with the oracle's on
// pairs taken from random games — the same position, one play apart, two
// plays apart, in both orders — and on pairs from different games, which are
// almost never within reach.
func TestDistanceAgreesWithTheOracle(t *testing.T) {
	if testing.Short() {
		t.Skip("the oracle enumerates every play of every roll")
	}
	rng := rand.New(rand.NewSource(11))
	games := [][]domain.Position{randomGame(rng), randomGame(rng)}
	type pair struct{ a, b domain.Position }
	var pairs []pair
	for _, g := range games {
		for i := 0; i+3 < len(g); i += 13 {
			for gap := 0; gap <= 3; gap++ {
				pairs = append(pairs, pair{g[i], g[i+gap]}, pair{g[i+gap], g[i]})
			}
		}
	}
	for i := 0; i < 8; i++ {
		a := games[0][rng.Intn(len(games[0]))]
		b := games[1][rng.Intn(len(games[1]))]
		pairs = append(pairs, pair{a, b})
	}
	ones, twos := 0, 0
	for _, pr := range pairs {
		want, wok := reachtest.Distance(&pr.a, &pr.b, 2)
		got, gok := NewTarget(Of(&pr.a), 2).Distance(Of(&pr.b))
		back, bok := NewTarget(Of(&pr.b), 2).Distance(Of(&pr.a))
		if wok != gok || (wok && want != got) {
			t.Fatalf("distance %d/%v, oracle %d/%v", got, gok, want, wok)
		}
		if gok != bok || got != back {
			t.Fatalf("not symmetric: %d/%v one way, %d/%v the other", got, gok, back, bok)
		}
		switch {
		case gok && got == 1:
			ones++
		case gok && got == 2:
			twos++
		}
	}
	if ones == 0 || twos == 0 {
		t.Fatalf("the sample never exercised a distance: %d at 1, %d at 2", ones, twos)
	}
}

// TestEitherSideMayPlayFirst pins that a structure one play away is found
// whichever side made that play, and that the side on roll of the stored
// position does not matter.
func TestEitherSideMayPlayFirst(t *testing.T) {
	p := opening()
	for _, mover := range []int{domain.Black, domain.White} {
		q := p
		q.PlayerOnRoll = mover
		q.Dice = [2]int{3, 1}
		after := domain.LegalMoves(&q)[0].Result
		for _, view := range []domain.Position{after, after.Mirror()} {
			if d, ok := NewTarget(Of(&p), 2).Distance(Of(&view)); !ok || d != 1 {
				t.Errorf("mover %d: distance %d/%v, want 1", mover, d, ok)
			}
		}
	}
	if d, ok := NewTarget(Of(&p), 2).Distance(Of(new(domain.Position))); ok {
		t.Errorf("an empty board is not within two plays of the opening: got %d", d)
	}
}

func TestCheckOrder(t *testing.T) {
	if CheckOrder(MaxOrder) != nil || CheckOrder(MaxOrder+1) == nil {
		t.Fatal("CheckOrder must serve up to MaxOrder and refuse beyond")
	}
}
