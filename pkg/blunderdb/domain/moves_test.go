package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"testing"
)

// mkPos builds a position from a signed checker map (positive = Black, negative =
// White; index 0 = WhiteBar, 25 = BlackBar, 1..24 = points). Bearoff is derived
// so each side totals 15.
func mkPos(mover, d1, d2 int, pts map[int]int) Position {
	var p Position
	for i := range p.Board.Points {
		p.Board.Points[i] = Point{Checkers: 0, Color: None}
	}
	var bOn, wOn int
	for idx, v := range pts {
		switch {
		case v > 0:
			p.Board.Points[idx] = Point{Checkers: v, Color: Black}
			bOn += v
		case v < 0:
			p.Board.Points[idx] = Point{Checkers: -v, Color: White}
			wOn += -v
		}
	}
	p.Board.Bearoff = [2]int{15 - bOn, 15 - wOn}
	p.PlayerOnRoll = mover
	p.Dice = [2]int{d1, d2}
	p.DecisionType = CheckerAction
	return p
}

func totals(p *Position) (black, white int) {
	for i := 0; i <= 25; i++ {
		switch p.Board.Points[i].Color {
		case Black:
			black += p.Board.Points[i].Checkers
		case White:
			white += p.Board.Points[i].Checkers
		}
	}
	return black + p.Board.Bearoff[Black], white + p.Board.Bearoff[White]
}

// Bear off from a single point with overage: 15 Black on point 3, roll 6-5.
func TestLegalMovesBearOffOverage(t *testing.T) {
	p := mkPos(Black, 6, 5, map[int]int{3: 15})
	plays := LegalMoves(&p)
	if len(plays) != 1 {
		t.Fatalf("expected 1 play, got %d", len(plays))
	}
	r := plays[0].Result
	if r.Board.Points[3].Checkers != 13 || r.Board.Bearoff[Black] != 2 {
		t.Fatalf("bad bear-off result: pt3=%d off=%d", r.Board.Points[3].Checkers, r.Board.Bearoff[Black])
	}
}

// No legal move (dance): Black on the bar, both entry points blocked.
func TestLegalMovesDance(t *testing.T) {
	p := mkPos(Black, 1, 2, map[int]int{25: 1, 6: 14, 24: -2, 23: -2, 1: -11})
	plays := LegalMoves(&p)
	if plays == nil || len(plays) != 0 {
		t.Fatalf("expected a dance (empty non-nil), got %v", plays)
	}
}

// A one-checker race: both orders reach the same square → a single play.
func TestLegalMovesOneCheckerRace(t *testing.T) {
	p := mkPos(Black, 2, 1, map[int]int{5: 1}) // 14 Black already off
	plays := LegalMoves(&p)
	if len(plays) != 1 {
		t.Fatalf("expected 1 play, got %d", len(plays))
	}
	if plays[0].Result.Board.Points[2].Checkers != 1 || plays[0].Result.Board.Points[2].Color != Black {
		t.Fatalf("checker should end on point 2: %+v", plays[0].Result.Board.Points[2])
	}
}

// Hitting a blot sends it to the opponent bar.
func TestLegalMovesHit(t *testing.T) {
	p := mkPos(Black, 2, 2, map[int]int{3: 1, 1: -1}) // Black blot on 3, White blot on 1
	plays := LegalMoves(&p)
	var hit *LegalPlay
	for i := range plays {
		for _, s := range plays[i].Steps {
			if s.Hit {
				hit = &plays[i]
			}
		}
	}
	if hit == nil {
		t.Fatalf("expected a play that hits; got %d plays", len(plays))
	}
	if hit.Result.Board.Points[WhiteBar].Checkers != 1 || hit.Result.Board.Points[WhiteBar].Color != White {
		t.Fatalf("hit White checker should be on the bar: %+v", hit.Result.Board.Points[WhiteBar])
	}
}

// Larger-die rule: when only one die is playable, plays using only the smaller
// die are illegal.
func TestLegalMovesLargerDie(t *testing.T) {
	// Lone Black checker on 13: 13/7 (die 6) or 13/10 (die 3). The 14 checkers on
	// point 7 are blocked (White owns 4 and 1), nothing is home (no bear-off), and
	// after either move the other die is blocked — so only one die plays and it
	// must be the larger (6): the only legal play is 13/7.
	p := mkPos(Black, 6, 3, map[int]int{13: 1, 7: 14, 4: -2, 1: -2, 24: -11})
	plays := LegalMoves(&p)
	if len(plays) != 1 {
		t.Fatalf("larger-die: expected 1 play, got %d: %+v", len(plays), plays)
	}
	s := plays[0].Steps[0]
	if s.From != 13 || s.To != 7 {
		t.Fatalf("larger-die: expected 13/7 (die 6), got %d/%d", s.From, s.To)
	}
}

// Doubles give four moves of the same value.
func TestLegalMovesDoublesUseFour(t *testing.T) {
	// 4 Black checkers on 24 with 6-6-6-6; each goes 24/18 (open). All four play.
	p := mkPos(Black, 6, 6, map[int]int{24: 4, 6: 11, 18: 0})
	plays := LegalMoves(&p)
	if len(plays) == 0 {
		t.Fatal("expected at least one play")
	}
	for _, pl := range plays {
		if len(pl.Steps) != 4 {
			t.Fatalf("doubles must use 4 dice, got %d (%s)", len(pl.Steps), pl.Notation)
		}
	}
}

// notation is MOVER-RELATIVE ("24" always names the mover's own back
// checkers), the convention NormalizeMove-based matching against gnubg/XG
// candidate lists depends on. The opening is symmetric, so White and Black
// rolling the same dice must render the identical SET of notations; only
// White can catch an absolute-index rendering, since for Black the two
// numberings coincide.
// Notation is mover-relative: a play and its mirror image for the other colour
// read the same, and the two colours reach mirrored boards. Among plays that
// reach the same board the one kept is the first generated, and generation
// walks the board in absolute order, so Black and White need not keep the
// same representative: that choice is fixed by TestLegalMovesOutputIsStable,
// not compared here.
func TestNotationIsMoverRelative(t *testing.T) {
	mirrorStep := func(s CheckerStep) CheckerStep {
		m := s
		if s.From == BlackBar {
			m.From = WhiteBar
		} else {
			m.From = 25 - s.From
		}
		if s.To != Off {
			m.To = 25 - s.To
		}
		return m
	}
	mirrorBoard := func(b [28]int) [28]int {
		var m [28]int
		for i := 0; i <= 25; i++ {
			m[25-i] = -b[i]
		}
		m[26], m[27] = b[27], b[26]
		return m
	}
	for _, d := range [][2]int{{3, 1}, {6, 5}, {5, 1}, {6, 1}, {4, 4}} {
		black := InitializePosition()
		black.PlayerOnRoll = Black
		black.Dice = d
		white := InitializePosition()
		white.PlayerOnRoll = White
		white.Dice = d

		blackPlays, whitePlays := LegalMoves(&black), LegalMoves(&white)
		if len(blackPlays) == 0 || len(blackPlays) != len(whitePlays) {
			t.Fatalf("dice %v: %d black plays, %d white plays", d, len(blackPlays), len(whitePlays))
		}
		whiteBoards := map[[28]int]bool{}
		for _, pl := range whitePlays {
			whiteBoards[signedBoard(&pl.Result)] = true
		}
		for _, pl := range blackPlays {
			mirrored := make([]CheckerStep, len(pl.Steps))
			for i, s := range pl.Steps {
				mirrored[i] = mirrorStep(s)
			}
			if got := notation(mirrored, White); got != pl.Notation {
				t.Errorf("dice %v: black %q mirrors to white %q", d, pl.Notation, got)
			}
			if !whiteBoards[mirrorBoard(signedBoard(&pl.Result))] {
				t.Errorf("dice %v: no white play mirrors black %q", d, pl.Notation)
			}
		}
	}
}

// gnubg and XG both write bar entries as lowercase "bar/24", and NormalizeMove
// does not case-fold, so a capitalised "Bar" would fail every comparison.
func TestNotationBarIsLowercase(t *testing.T) {
	p := mkPos(Black, 6, 1, map[int]int{BlackBar: 1, 13: 4, 8: 3, 6: 5, 24: 2})
	plays := LegalMoves(&p)
	if len(plays) == 0 {
		t.Fatal("expected at least one legal play entering from the bar")
	}
	for _, pl := range plays {
		if strings.Contains(pl.Notation, "Bar") {
			t.Errorf("notation %q uses capitalised Bar, want lowercase bar", pl.Notation)
		}
	}
}

// Every generated play conserves 15 checkers per side and uses the right number
// of dice; results are distinct.
func TestLegalMovesInvariants(t *testing.T) {
	cases := [][2]int{{3, 1}, {6, 5}, {2, 2}, {6, 4}, {5, 5}}
	for _, d := range cases {
		p := InitializePosition()
		p.Dice = d
		plays := LegalMoves(&p)
		if len(plays) == 0 {
			t.Fatalf("opening %v should have legal plays", d)
		}
		seen := map[string]bool{}
		for _, pl := range plays {
			b, w := totals(&pl.Result)
			if b != 15 || w != 15 {
				t.Fatalf("%v: checker leak black=%d white=%d (%s)", d, b, w, pl.Notation)
			}
			k := boardKey(&pl.Result)
			if seen[k] {
				t.Fatalf("%v: duplicate result %s", d, pl.Notation)
			}
			seen[k] = true
		}
	}
}

// signedBoard is the test's own oracle for "same resulting board": checker
// counts signed by owner, so the colour an empty point happens to carry cannot
// tell two boards apart.
func signedBoard(p *Position) [28]int {
	var s [28]int
	for i := 0; i <= 25; i++ {
		pt := p.Board.Points[i]
		switch {
		case pt.Checkers == 0:
		case pt.Color == Black:
			s[i] = pt.Checkers
		case pt.Color == White:
			s[i] = -pt.Checkers
		}
	}
	s[26], s[27] = p.Board.Bearoff[Black], p.Board.Bearoff[White]
	return s
}

// A position read back from storage may carry a colour on its empty points. A
// point emptied while moving takes the mover's colour, an untouched one keeps
// its own: two plays reaching the same board must still count once.
func TestLegalMovesDedupIgnoresEmptyPointColour(t *testing.T) {
	cases := []struct {
		name string
		xgid string
		want int
	}{
		{"5-2", "XGID=-b----E-C---eE---c-e----B-:0:0:1:52:0:0:0:0:10", 8},
		{"6-3", "XGID=-a-B--E-B-a-dDB--b-bcb----:0:0:1:63:0:0:0:0:10", 14},
	}
	for _, c := range cases {
		base, err := DecodeXGID(c.xgid)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		starts := []Position{base}
		if c.name == "5-2" {
			opening := InitializePosition()
			opening.Dice = [2]int{5, 2}
			starts = append(starts, opening)
		}
		for _, p0 := range starts {
			for _, paint := range []int{None, Black, White} {
				p := p0
				for i := 0; i <= 25; i++ {
					if p.Board.Points[i].Checkers == 0 {
						p.Board.Points[i].Color = paint
					}
				}
				for _, mover := range []int{p0.PlayerOnRoll, 1 - p0.PlayerOnRoll} {
					p.PlayerOnRoll = mover
					plays := LegalMoves(&p)
					seen := map[[28]int]string{}
					for _, pl := range plays {
						k := signedBoard(&pl.Result)
						if prev, dup := seen[k]; dup {
							t.Errorf("%s paint=%d mover=%d: %q and %q reach the same board", c.name, paint, mover, prev, pl.Notation)
						}
						seen[k] = pl.Notation
					}
					if mover == p0.PlayerOnRoll && len(plays) != c.want {
						t.Errorf("%s paint=%d: %d plays, want %d", c.name, paint, len(plays), c.want)
					}
				}
			}
		}
	}
}

// legalMovesGolden is the digest of LegalMoves over the corpus below, both
// colours on roll and every roll of the dice. The plays kept, their order and
// their notation are stored and matched by string (analysis merge, rollouts),
// so on a position whose empty points carry no colour this output must not
// move by a byte. Update the digest only with a change that is meant to rename
// or reorder stored plays.
const legalMovesGolden = "56ea0622ee5a4515f45704d5f6c16231e6ee46a4269a72158d1b1752d61a6cb6"

func TestLegalMovesOutputIsStable(t *testing.T) {
	var starts []Position
	for _, pair := range ogidXGIDPairs(loadOGIDCorpus(t)) {
		p, err := DecodeXGID(pair[1])
		if err != nil {
			t.Fatalf("%s: %v", pair[1], err)
		}
		starts = append(starts, p)
	}
	opening := InitializePosition()
	for i := 0; i <= 25; i++ {
		if opening.Board.Points[i].Checkers == 0 {
			opening.Board.Points[i].Color = None
		}
	}
	starts = append(starts, opening)

	h := sha256.New()
	for n, p0 := range starts {
		for i := 0; i <= 25; i++ {
			if p0.Board.Points[i].Checkers == 0 && p0.Board.Points[i].Color != None {
				t.Fatalf("start %d has a coloured empty point %d", n, i)
			}
		}
		for _, mover := range []int{Black, White} {
			for d1 := 1; d1 <= 6; d1++ {
				for d2 := 1; d2 <= d1; d2++ {
					p := p0
					p.PlayerOnRoll = mover
					p.Dice = [2]int{d1, d2}
					fmt.Fprintf(h, "%d|%d|%d%d|", n, mover, d1, d2)
					for _, pl := range LegalMoves(&p) {
						fmt.Fprintf(h, "%s;", pl.Notation)
					}
					fmt.Fprintln(h)
				}
			}
		}
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != legalMovesGolden {
		t.Fatalf("LegalMoves output changed: digest %s, want %s", got, legalMovesGolden)
	}
}

// Only one die plays, and the only larger-die candidate is a bear-off that the
// smaller die makes exactly. Black (on roll) holds 2(3) 3(3) 4(4) 6(3) with 2
// off; White holds Black's 1-point. Roll 5-2: the 5 has no move (6/1 is
// blocked, and no overage bear-off while a checker sits on 6), so the 2 is
// played alone, anywhere: 6/4, 4/2, 2/off (3/1 is blocked). 2/off covers five
// pips' worth of distance, but it was played with the 2, so it does not make
// the larger die playable.
func TestLegalMovesSmallerDieExactBearOffIsNotTheLargerDie(t *testing.T) {
	pts := map[int]int{2: 3, 3: 3, 4: 4, 6: 3, 1: -2,
		11: -1, 12: -1, 17: -2, 18: -1, 19: -2, 20: -2, 22: -3, 23: -1}
	want := []string{"2/off", "4/2", "6/4"}

	p := mkPos(Black, 5, 2, pts)
	if b, w := totals(&p); b != 15 || w != 15 {
		t.Fatalf("bad fixture: black=%d white=%d", b, w)
	}
	assertNotations(t, LegalMoves(&p), want)

	q := mkPos(Black, 2, 5, pts) // dice order is irrelevant
	assertNotations(t, LegalMoves(&q), want)

	// The same position from White's side (mirrored board): same rule.
	mirrored := map[int]int{}
	for idx, v := range pts {
		mirrored[25-idx] = -v
	}
	r := mkPos(White, 5, 2, mirrored)
	assertNotations(t, LegalMoves(&r), want)
}

// A lone checker on the 2-point with 5-2: the 5 bears it off by overage (no
// checker higher), so the larger die is playable and the play survives the
// larger-die rule.
func TestLegalMovesLargerDieOverageBearOffStillLegal(t *testing.T) {
	p := mkPos(Black, 5, 2, map[int]int{2: 1, 20: -15})
	assertNotations(t, LegalMoves(&p), []string{"2/off"})
}

// Doubles are untouched by the larger-die rule: 2-2 in the same position plays
// four twos.
func TestLegalMovesDoublesUnaffectedByLargerDieRule(t *testing.T) {
	p := mkPos(Black, 2, 2, map[int]int{2: 3, 3: 3, 4: 4, 6: 3, 1: -2,
		11: -1, 12: -1, 17: -2, 18: -1, 19: -2, 20: -2, 22: -3, 23: -1})
	plays := LegalMoves(&p)
	if len(plays) == 0 {
		t.Fatal("2-2: expected legal plays")
	}
	for _, pl := range plays {
		if len(pl.Steps) != 4 {
			t.Fatalf("2-2: play %q uses %d dice, want 4", pl.Notation, len(pl.Steps))
		}
	}
}

func assertNotations(t *testing.T, plays []LegalPlay, want []string) {
	t.Helper()
	got := make([]string, 0, len(plays))
	for _, pl := range plays {
		got = append(got, pl.Notation)
	}
	sort.Strings(got)
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("plays = %v, want %v", got, want)
	}
}
