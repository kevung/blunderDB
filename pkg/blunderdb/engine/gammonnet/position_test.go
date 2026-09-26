// SPDX-License-Identifier: MIT

package gammonnet

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// bearoffBoard puts every checker of each colour on one point.
func bearoffBoard(whitePoint, blackPoint int) domain.Position {
	var p domain.Position
	p.Board.Points[whitePoint] = domain.Point{Checkers: 15, Color: domain.White}
	p.Board.Points[blackPoint] = domain.Point{Checkers: 15, Color: domain.Black}
	p.PlayerOnRoll = domain.White
	p.Dice = [2]int{1, 1}
	p.Score = [2]int{-1, -1}
	return p
}

// FromDomain's geometry fails silently, so it is pinned against an
// independent oracle, domain.LegalMoves: a White checker on domain point 24
// bears off on a 1 (asserted from domain alone), and FromDomain must place
// that point at gammonNet index 0, White's ace point.
func TestDomainPoint24IsWhitesAcePointAndMapsToIndexZero(t *testing.T) {
	p := bearoffBoard(24, 1)

	plays := domain.LegalMoves(&p)
	if len(plays) == 0 {
		t.Fatal("no legal play from a board of fifteen White checkers on point 24 with 1-1")
	}
	var bearsOff bool
	for _, play := range plays {
		for _, step := range play.Steps {
			if step.From == 24 && step.To == domain.Off {
				bearsOff = true
			}
		}
	}
	if !bearsOff {
		t.Fatal("domain point 24 is not White's ace point: a checker there does not bear off on a 1.\n" +
			"the mapping below is written against that premise and must be revisited")
	}

	gn, err := FromDomain(&p)
	if err != nil {
		t.Fatalf("FromDomain: %v", err)
	}
	if gn.Points[0] != 15 {
		t.Errorf("gammonNet index 0 holds %d, want 15 White checkers "+
			"(index 0 is White's ace point by gammonNet's convention)", gn.Points[0])
	}
	// domain point 1 is Black's ace point, which gammonNet indexes 23.
	if gn.Points[23] != -15 {
		t.Errorf("gammonNet index 23 holds %d, want -15 Black checkers", gn.Points[23])
	}
	if gn.Turn != White {
		t.Errorf("turn = %d, want White (%d) — the identifiers are inverted between the two conventions", gn.Turn, White)
	}
}

func TestFromDomainCarriesBarAndBearoff(t *testing.T) {
	p := bearoffBoard(24, 1)
	p.Board.Points[24] = domain.Point{Checkers: 12, Color: domain.White}
	p.Board.Points[domain.WhiteBar] = domain.Point{Checkers: 1, Color: domain.White}
	p.Board.Bearoff[domain.White] = 2
	p.Board.Points[1] = domain.Point{Checkers: 13, Color: domain.Black}
	p.Board.Points[domain.BlackBar] = domain.Point{Checkers: 2, Color: domain.Black}
	p.Board.Bearoff[domain.Black] = 0

	gn, err := FromDomain(&p)
	if err != nil {
		t.Fatalf("FromDomain: %v", err)
	}
	if gn.Bar[White] != 1 || gn.Bar[Black] != 2 {
		t.Errorf("bar = %v, want [1 2] (White, Black)", gn.Bar)
	}
	if gn.Off[White] != 2 || gn.Off[Black] != 0 {
		t.Errorf("off = %v, want [2 0] (White, Black)", gn.Off)
	}
}

func TestFromDomainRefusesIncoherentBoards(t *testing.T) {
	for _, tc := range []struct {
		name string
		mut  func(*domain.Position)
	}{
		{"not fifteen checkers", func(p *domain.Position) {
			p.Board.Points[24] = domain.Point{Checkers: 14, Color: domain.White}
		}},
		{"no player on roll", func(p *domain.Position) { p.PlayerOnRoll = -1 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := bearoffBoard(24, 1)
			tc.mut(&p)
			if _, err := FromDomain(&p); err == nil {
				t.Fatal("converted a board it should have refused")
			}
		})
	}
}

// A raw evaluation's cost is the first number the search tranche needs. Run it
// on an idle machine: under load the figure measures the machine, not the code.
func BenchmarkEvaluate(b *testing.B) {
	net, err := embeddedNetwork()
	if err != nil {
		b.Fatal(err)
	}
	ev := NewEvaluator(net)
	p := bearoffBoard(24, 1)
	gn, err := FromDomain(&p)
	if err != nil {
		b.Fatal(err)
	}
	var features [NumFeatures]float32
	if !Encode(&gn, &features) {
		b.Fatal("encoding refused")
	}
	var probs [NumOutputs]float32

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := ev.Evaluate(features[:], &probs); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEvaluateBatch is the per-position cost on DISTINCT positions, as
// the search meets them; BenchmarkEvaluate's single position stays in L1 and
// flatters any kernel. Reported per position, so scalar and batched runs
// divide.
func BenchmarkEvaluateBatch(b *testing.B) {
	net, err := embeddedNetwork()
	if err != nil {
		b.Fatal(err)
	}
	ev := NewEvaluator(net)

	// EvalBatchWidth positions that differ, so no two share an encoding.
	batch := make([][NumFeatures]float32, EvalBatchWidth)
	for i := range batch {
		p := bearoffBoard(24-i, 1+i)
		gn, err := FromDomain(&p)
		if err != nil {
			b.Fatal(err)
		}
		if !Encode(&gn, &batch[i]) {
			b.Fatal("encoding refused")
		}
	}
	var probs [NumOutputs]float32

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := range batch {
			if err := ev.Evaluate(batch[j][:], &probs); err != nil {
				b.Fatal(err)
			}
		}
	}
	b.StopTimer()
	// b.N iterations of EvalBatchWidth positions: report the unit the kernel
	// is specified in.
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*EvalBatchWidth), "ns/position")
}

// BenchmarkDecision2Ply is one canonical 2-ply k=12 decision, serially, on
// one core.
func BenchmarkDecision2Ply(b *testing.B) {
	if testing.Short() {
		b.Skip("a 2-ply decision costs seconds")
	}
	dp, err := domain.DecodeXGID(openingXGID)
	if err != nil {
		b.Fatal(err)
	}
	dp.PlayerOnRoll = domain.White
	p, err := FromDomain(&dp)
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
		if _, ok, err := s.BestPlay(&pos, 3, 1); err != nil || !ok {
			b.Fatalf("search refused: %v", err)
		}
	}
	b.StopTimer()
	filled, slotted := s.BatchFill()
	if slotted > 0 {
		b.ReportMetric(100*float64(filled)/float64(slotted), "%fill")
	}
}

// BenchmarkDecision2PlyMatch is the same decision as the application runs it
// (ConfigForPosition, match and cube-valued leaves): only at a score does the
// cube model build a stake chain.
func BenchmarkDecision2PlyMatch(b *testing.B) {
	if testing.Short() {
		b.Skip("a 2-ply decision costs seconds")
	}
	dp, err := domain.DecodeXGID(openingXGID)
	if err != nil {
		b.Fatal(err)
	}
	dp.PlayerOnRoll = domain.White
	p, err := FromDomain(&dp)
	if err != nil {
		b.Fatal(err)
	}
	cfg := DefaultConfig(2)
	cfg.UseMatch = true
	cfg.Match = MatchState{AwayOnRoll: 5, AwayOpponent: 5, Cube: 1}
	cfg.UseCube = true
	cfg.CubeOwner = CubeCentred
	cfg.CubeX = DefaultEfficiency(CubeCentred)
	s, err := NewSearcher(cfg)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pos := p
		if _, ok, err := s.BestPlay(&pos, 3, 1); err != nil || !ok {
			b.Fatalf("search refused: %v", err)
		}
	}
}
