// SPDX-License-Identifier: MIT

package gammonnet

import (
	"fmt"
	"math"
	"math/rand"
	"runtime"
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

			wantEquity, ok := s.positionEquity(&pos, cfg.Ply, 0, nil, CubeCentred)
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

// TestParallelProbsIsBitIdentical is TestParallelSearchIsBitIdentical
// (search_test.go), extended to Probs (#195/C.8): the panel's cube decision
// calls Probs on a searcher built with WithWorkers, and probsAt's own root
// loop is the one place this file lets a worker count change which core
// computes a term — never the order the twenty-one are summed in. Any
// worker count must return the exact same five floats, to the bit.
func TestParallelProbsIsBitIdentical(t *testing.T) {
	if testing.Short() {
		t.Skip("a 2-ply Probs call costs seconds")
	}
	p := openingPosition(t)

	serial, err := NewSearcher(DefaultConfig(2))
	if err != nil {
		t.Fatal(err)
	}
	want, ok := serial.Probs(&p)
	if !ok {
		t.Fatal("serial Probs refused")
	}

	// Comme TestParallelSearchIsBitIdentical : un, deux, la machine, et un
	// chiffre au-delà de ce que la file peut porter (WithWorkers borne à
	// Filter[depth] × 21).
	for _, nw := range []int{1, 2, runtime.NumCPU(), 64} {
		t.Run(fmt.Sprintf("workers=%d", nw), func(t *testing.T) {
			par, err := NewSearcher(DefaultConfig(2))
			if err != nil {
				t.Fatal(err)
			}
			par = par.WithWorkers(nw)
			pp := p
			got, ok := par.Probs(&pp)
			if !ok {
				t.Fatal("parallel Probs refused")
			}
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("output %d: serial %.17g, parallel(nw=%d) %.17g — not bit-identical",
						i, want[i], nw, got[i])
				}
			}
		})
	}
}

// BenchmarkProbsSerial2Ply and BenchmarkProbsParallel2Ply are the pair C.8
// (#195) is measured against: one canonical 2-ply Probs call — the panel's
// cube decision (gammonnet_eval.go) — serially and with a worker per core.
// Before the fix, probsAt's own twenty-one root rolls each opened and closed
// their own worker barrier (21 separate deepenLevel calls of ~21 tasks each);
// after, they are combined into deepenGroups' single queue, exactly as
// rankPlays' phase three already does for Plays/BestPlay.
func BenchmarkProbsSerial2Ply(b *testing.B) {
	if testing.Short() {
		b.Skip("a 2-ply decision costs seconds")
	}
	p, err := probsBenchPosition()
	if err != nil {
		b.Fatal(err)
	}
	s, err := NewSearcher(DefaultConfig(2))
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pos := p
		if _, ok := s.Probs(&pos); !ok {
			b.Fatal("Probs refused")
		}
	}
}

func BenchmarkProbsParallel2Ply(b *testing.B) {
	if testing.Short() {
		b.Skip("a 2-ply decision costs seconds")
	}
	p, err := probsBenchPosition()
	if err != nil {
		b.Fatal(err)
	}
	s, err := NewSearcher(DefaultConfig(2))
	if err != nil {
		b.Fatal(err)
	}
	s = s.WithWorkers(runtime.NumCPU())

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pos := p
		if _, ok := s.Probs(&pos); !ok {
			b.Fatal("Probs refused")
		}
	}
}

func probsBenchPosition() (Position, error) {
	dp, err := domain.DecodeXGID(openingXGID)
	if err != nil {
		return Position{}, err
	}
	dp.PlayerOnRoll = domain.White
	return FromDomain(&dp)
}
