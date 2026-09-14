package training

import (
	"math/rand/v2"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
)

// side is one player's chequers in their OWN point numbering: side[p] is the
// number of chequers on their p-point, 1..24, index 0 unused. It is the frame
// a player reads their own position in, so a shape reads the same for either
// colour.
type side [25]int

func (s side) checkers() int {
	total := 0
	for _, n := range s {
		total += n
	}
	return total
}

// highest is the rearmost occupied point, 0 when the side is empty.
func (s side) highest() int {
	for p := 24; p >= 1; p-- {
		if s[p] > 0 {
			return p
		}
	}
	return 0
}

// pool holds the Evaluation exercise's canonical shapes: « contact just
// broken » (ADR-0041 rule 2), in the two ways contact breaks in a real game.
//
//   - A RUNNING break: both sides still have fifteen chequers, and the last two
//     have just passed each other somewhere between the 10- and the 14-point.
//   - A HOLDING break: one side kept an anchor deep in the opponent's board
//     while the other was already bearing off, and has just left it — so one
//     side has chequers off and the other a straggler on its 17- or 18-point.
//     Leaving these out made every generated side a fifteen-chequer one, where
//     a third of the sides of the ten real matches' just-broken races stand
//     under thirty pips (evaluation_histogram_test.go).
//
// The plies do the rest. They are DATA OF THE EXERCISE, not a setting: adding
// one is a code change with its histogram check.
var pool = [...]side{
	// Running breaks.
	{6: 4, 5: 2, 4: 2, 8: 2, 10: 1, 11: 2, 13: 2},     // 116 pips, a midpoint still to clear
	{6: 3, 5: 3, 4: 2, 3: 2, 7: 2, 9: 1, 12: 2},       // 94
	{6: 5, 8: 3, 10: 2, 11: 1, 13: 4},                 // 137, the opening's structure, late
	{6: 4, 5: 3, 4: 3, 3: 2, 2: 1, 9: 1, 11: 1},       // 79, nearly home
	{6: 3, 5: 2, 4: 2, 3: 2, 2: 2, 1: 1, 8: 2, 10: 1}, // 73, deep and gapless
	{6: 5, 5: 2, 7: 2, 8: 2, 9: 1, 13: 3},             // 118
	{6: 3, 5: 3, 4: 2, 8: 3, 11: 2, 12: 2},            // 111
	{6: 4, 4: 3, 3: 1, 7: 1, 9: 2, 10: 2, 14: 2},      // 112, a gap on the five point
	// Holding breaks: the side that was bearing off…
	{1: 2, 2: 2, 3: 2, 4: 1, 5: 1, 6: 1}, // 27 pips, six off
	{2: 3, 3: 2, 4: 2, 6: 1},             // 26, seven off
	{1: 3, 2: 2, 3: 3, 4: 2, 5: 1, 6: 1}, // 34, three off, the ace point crowded
	// …and the side that has just left its anchor.
	{18: 1, 16: 1, 6: 3, 5: 3, 4: 3, 3: 2, 2: 2}, // 89, one runner still out
	{17: 2, 6: 4, 5: 3, 4: 2, 3: 2, 2: 2},        // 91, the anchor leaving together
}

// poolPairs are the (Black, White) shape pairs that make a position with no
// contact: Black's rearmost chequer stands on domain point hb, White's on
// 25-hw, and they have passed each other iff hb < 25-hw.
var poolPairs = func() [][2]int {
	var pairs [][2]int
	for b := range pool {
		for w := range pool {
			if pool[b].highest()+pool[w].highest() <= 24 {
				pairs = append(pairs, [2]int{b, w})
			}
		}
	}
	return pairs
}()

// poolSeed draws a pair of shapes — each side its own, so the two are
// asymmetric the way a real race is — and a roller.
func poolSeed(rng *rand.Rand) gammonnet.Position {
	pair := poolPairs[rng.IntN(len(poolPairs))]
	return layOut(pool[pair[0]], pool[pair[1]], rng.IntN(2))
}

// layOut places two own-frame sides on gammonNet's board. gammonNet index i
// is White's (i+1)-point and Black's (24-i)-point (gammonnet.Position).
func layOut(black, white side, onRoll int) gammonnet.Position {
	var gp gammonnet.Position
	for p := 1; p <= 24; p++ {
		if n := white[p]; n > 0 {
			gp.Points[p-1] = int8(n)
		}
		if n := black[p]; n > 0 {
			gp.Points[24-p] = int8(-n)
		}
	}
	gp.Off[gammonnet.White] = uint8(gammonnet.NumCheckers - white.checkers())
	gp.Off[gammonnet.Black] = uint8(gammonnet.NumCheckers - black.checkers())
	gp.Turn = gammonnet.Black
	if onRoll == domain.White {
		gp.Turn = gammonnet.White
	}
	return gp
}

// walk is one question's making: its dice, the searcher that chooses each
// play, and the clock its deadline is read on.
type walk struct {
	walker   *gammonnet.Searcher
	rng      *rand.Rand
	now      func() time.Time
	deadline time.Time
}

// playOut rolls `plies` times from the seed and lays the snapshot out as a
// question, without its truth yet. It stops early rather than finish the game:
// a position where a side has borne everything off asks nothing. Plies reports
// what was really played.
//
// Past the deadline it drops the walk for a pool seed at zero plies (rule 5).
func (w walk) playOut(seed gammonnet.Position, plies int, source, refusal string) EvaluationQuestion {
	pos := seed
	played := 0
	for ; played < plies; played++ {
		if w.now().After(w.deadline) {
			return framed(poolSeed(w.rng), 0, SourcePool, refusal)
		}
		next, ok := w.onePly(pos)
		if !ok {
			break
		}
		pos = next
	}
	return framed(pos, played, source, refusal)
}

// onePly rolls once and plays gammonNet's best play for the side on roll. A
// dance passes the turn. It reports false — and changes nothing — when the
// play would end the game.
func (w walk) onePly(pos gammonnet.Position) (gammonnet.Position, bool) {
	d1, d2 := w.rng.IntN(6)+1, w.rng.IntN(6)+1
	best, ok, err := w.walker.BestPlay(&pos, d1, d2)
	if err != nil {
		return pos, false
	}
	if !ok {
		pos.Turn = 1 - pos.Turn
		return pos, true
	}
	next := best.Play.Result
	if next.Off[gammonnet.White] >= gammonnet.NumCheckers || next.Off[gammonnet.Black] >= gammonnet.NumCheckers {
		return pos, false
	}
	return next, true
}

// framed is a walked position in the exercise's own frame: money play, cube
// centred, no dice, pre-roll (#322: « argent seulement » — the source makes no
// other kind of position).
func framed(gp gammonnet.Position, played int, source, refusal string) EvaluationQuestion {
	return EvaluationQuestion{
		Generated: true,
		Refusal:   refusal,
		Source:    source,
		Plies:     played,
		Position: domain.Position{
			Board:        toDomain(&gp),
			Cube:         domain.Cube{Owner: domain.None, Value: 0},
			Score:        [2]int{domain.Unlimited, domain.Unlimited},
			PlayerOnRoll: playerOf(gp.Turn),
			DecisionType: domain.CubeAction,
		},
	}
}

func playerOf(turn uint8) int {
	if turn == gammonnet.White {
		return domain.White
	}
	return domain.Black
}

// toDomain is gammonnet.FromDomain's inverse — the board only. Nothing in
// gammonnet needs to go back to a domain position, which is why the inverse is
// here, next to the one caller that does, and held to FromDomain by a
// round-trip test.
func toDomain(gp *gammonnet.Position) domain.Board {
	var b domain.Board
	for i := range b.Points {
		b.Points[i] = domain.Point{Color: domain.None}
	}
	for i := 0; i < gammonnet.NumPoints; i++ {
		n := int(gp.Points[i])
		point := domain.NumPoints - i // gammonNet index i ↔ domain point 24-i
		switch {
		case n > 0:
			b.Points[point] = domain.Point{Checkers: n, Color: domain.White}
		case n < 0:
			b.Points[point] = domain.Point{Checkers: -n, Color: domain.Black}
		}
	}
	if n := int(gp.Bar[gammonnet.White]); n > 0 {
		b.Points[domain.WhiteBar] = domain.Point{Checkers: n, Color: domain.White}
	}
	if n := int(gp.Bar[gammonnet.Black]); n > 0 {
		b.Points[domain.BlackBar] = domain.Point{Checkers: n, Color: domain.Black}
	}
	b.Bearoff[domain.White] = int(gp.Off[gammonnet.White])
	b.Bearoff[domain.Black] = int(gp.Off[gammonnet.Black])
	return b
}
