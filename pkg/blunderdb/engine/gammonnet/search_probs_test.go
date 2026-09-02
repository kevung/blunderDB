// SPDX-License-Identifier: MIT

package gammonnet

import (
	"math"
	"math/rand"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

func TestInvertProbsIsAnInvolution(t *testing.T) {
	in := [NumOutputs]float32{0.6, 0.2, 0.05, 0.1, 0.02}
	once := invertProbs(&in)
	twice := invertProbs(&once)
	for i := range in {
		if math.Abs(float64(twice[i]-in[i])) > 1e-7 {
			t.Errorf("index %d: invertProbs(invertProbs(p)) = %v, want %v", i, twice[i], in[i])
		}
	}
}

// The linearity identity gn_search_probs is built to satisfy: the money
// equity of the distribution Probs returns must equal what BestPlay's own
// scalar recursion assigns to the same position at the same depth. If the
// two walks ever pick a different play out from under each other, or the
// perspective inversion is wrong, this is where it shows up — cheaply,
// without a C reference.
func TestProbsMoneyEquityMatchesPositionEquity(t *testing.T) {
	rng := rand.New(rand.NewSource(20260829))
	for ply := 0; ply <= 2; ply++ {
		cfg := DefaultConfig(ply)
		s, err := NewSearcher(cfg)
		if err != nil {
			t.Fatal(err)
		}
		trials := 8
		if ply == 2 {
			trials = 3 // 2-ply is expensive; a handful is enough to catch a wrong sign or a wrong depth.
		}
		for i := 0; i < trials; i++ {
			onRoll := domain.White
			if i%2 == 1 {
				onRoll = domain.Black
			}
			b := randomBoard(rng, onRoll)
			pos, err := FromDomain(&b)
			if err != nil || pos.isOver() {
				continue
			}

			wantEquity, ok := s.positionEquity(&pos, cfg.Ply, 0, false, nil, CubeCentred)
			if !ok {
				t.Fatalf("ply=%d: positionEquity refused", ply)
			}

			probs, ok := s.Probs(&pos)
			if !ok {
				t.Fatalf("ply=%d: Probs refused", ply)
			}
			gotEquity := float64(MoneyEquity(&probs))

			if math.Abs(gotEquity-wantEquity) > 1e-4 {
				t.Errorf("ply=%d trial=%d: MoneyEquity(Probs(pos))=%v, positionEquity(pos)=%v (Δ=%v)",
					ply, i, gotEquity, wantEquity, math.Abs(gotEquity-wantEquity))
			}
		}
	}
}

// randomTerminalPosition builds a finished game at a random stake: plain,
// gammon, or backgammon via the bar. Turn names the loser, matching the
// convention terminalEquity and terminalProbs both document.
func randomTerminalPosition(rng *rand.Rand) Position {
	winner := White
	if rng.Intn(2) == 1 {
		winner = Black
	}
	loser := Black
	if winner == Black {
		loser = White
	}

	var p Position
	p.Turn = uint8(loser)
	p.Off[winner] = NumCheckers
	switch rng.Intn(3) {
	case 0:
		p.Off[loser] = 1 // plain: the loser has borne off at least once
	case 1:
		// gammon: loser has borne off nothing, nothing on the bar, and (all
		// Points left zero) nothing sitting in the winner's home board.
	case 2:
		p.Bar[loser] = 1 // backgammon via the bar
	}
	return p
}

// A terminal position's distribution must be a one-hot vector agreeing with
// terminalEquity's own valuation of the same position.
func TestTerminalProbsMatchesTerminalEquity(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 20; i++ {
		p := randomTerminalPosition(rng)
		probs := terminalProbs(&p)
		got := float64(MoneyEquity(&probs))
		want := terminalEquity(&p)
		if math.Abs(got-want) > 1e-6 {
			t.Errorf("trial %d: MoneyEquity(terminalProbs)=%v, terminalEquity=%v", i, got, want)
		}
	}
}
