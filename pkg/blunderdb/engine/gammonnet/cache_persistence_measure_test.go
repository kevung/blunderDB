package gammonnet

import (
	"fmt"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// Mesure du cache persistant, et la raison pour laquelle il n'existe pas.
//
// Sur seize décisions consécutives d'une partie à 2-ply, porter le cache
// d'une décision à la suivante économise 2,8 % des évaluations (2^16
// entrées), 3,8 % sans aucune éviction (2^22) — le plafond absolu, avant
// entrées-sorties et contrôle de version. Et la réanalyse d'une base, son
// bénéficiaire naturel, est justement le cas où le réseau a changé et le
// fichier serait à jeter. Ces mesures restent pour rendre le refus révisable.
//
// TestMeasureCacheAcrossDecisions walks one game at 2-ply twice: one Searcher
// whose cache carries across decisions, then a fresh Searcher per decision
// (today's behaviour). The gap between the eval counts is the whole prize.
func TestMeasureCacheAcrossDecisions(t *testing.T) {
	if testing.Short() {
		t.Skip("measurement, not a gate")
	}
	positions := walkOneGame(t, 16)
	if len(positions) < 4 {
		t.Skip("could not walk a game")
	}

	shared, err := NewSearcher(DefaultConfig(2))
	if err != nil {
		t.Fatalf("NewSearcher: %v", err)
	}
	for i := range positions {
		if _, _, err := shared.BestPlay(&positions[i], 3, 1); err != nil {
			t.Fatalf("BestPlay: %v", err)
		}
	}
	carried, _, carriedHits := shared.Counters()

	var fresh uint64
	for i := range positions {
		s, err := NewSearcher(DefaultConfig(2))
		if err != nil {
			t.Fatalf("NewSearcher: %v", err)
		}
		if _, _, err := s.BestPlay(&positions[i], 3, 1); err != nil {
			t.Fatalf("BestPlay: %v", err)
		}
		e, _, _ := s.Counters()
		fresh += e
	}

	saved := 0.0
	if fresh > 0 {
		saved = 100 * float64(fresh-carried) / float64(fresh)
	}
	fmt.Printf("decisions=%d  fresh-cache evals=%d  carried-cache evals=%d  carried hits=%d  saved=%.1f%%\n",
		len(positions), fresh, carried, carriedHits, saved)
}

// walkOneGame plays the engine's own best move from the opening for n plies
// and returns the positions it passed through — consecutive positions of one
// game, which is the shape a batch analysis actually sees.
func walkOneGame(t *testing.T, n int) []Position {
	t.Helper()
	dp, err := domain.DecodeXGID(openingXGID)
	if err != nil {
		t.Fatalf("DecodeXGID: %v", err)
	}
	dp.PlayerOnRoll = domain.White
	p, err := FromDomain(&dp)
	if err != nil {
		t.Fatalf("FromDomain: %v", err)
	}
	s, err := NewSearcher(DefaultConfig(2))
	if err != nil {
		t.Fatalf("NewSearcher: %v", err)
	}
	dice := [][2]int{{3, 1}, {6, 5}, {5, 2}, {4, 3}, {6, 2}, {5, 4}, {6, 3}, {2, 1},
		{6, 4}, {5, 3}, {4, 2}, {3, 2}, {6, 1}, {5, 1}, {4, 1}, {3, 1}}
	out := make([]Position, 0, n)
	cur := p
	for i := 0; i < n && i < len(dice); i++ {
		out = append(out, cur)
		best, ok, err := s.BestPlay(&cur, dice[i][0], dice[i][1])
		if err != nil {
			t.Fatalf("BestPlay at ply %d: %v", i, err)
		}
		if !ok {
			t.Logf("no legal play at ply %d, stopping the walk", i)
			break
		}
		cur = best.Play.Result
		// The generator returns the resulting position; the side to move
		// alternates, and Turn is an index (White=0, Black=1), not a sign.
		if cur.Turn == White {
			cur.Turn = Black
		} else {
			cur.Turn = White
		}
	}
	return out
}

// TestMeasureCacheAcrossDecisionsUnbounded is the same measurement with a
// cache that never evicts (2^22 entries, ~240 MB): the upper bound of any
// persistent cache.
func TestMeasureCacheAcrossDecisionsUnbounded(t *testing.T) {
	if testing.Short() {
		t.Skip("measurement, not a gate")
	}
	positions := walkOneGame(t, 16)
	if len(positions) < 4 {
		t.Skip("could not walk a game")
	}
	cfg := DefaultConfig(2)
	net, err := embeddedNetwork()
	if err != nil {
		t.Fatalf("network: %v", err)
	}
	prune, err := embeddedPruneNetwork()
	if err != nil {
		t.Fatalf("prune network: %v", err)
	}
	build := func() *Searcher { return newSearcherWithCache(cfg, net, prune, 22) }

	shared := build()
	for i := range positions {
		if _, _, err := shared.BestPlay(&positions[i], 3, 1); err != nil {
			t.Fatalf("BestPlay: %v", err)
		}
	}
	carried, _, _ := shared.Counters()

	var fresh uint64
	for i := range positions {
		s := build()
		if _, _, err := s.BestPlay(&positions[i], 3, 1); err != nil {
			t.Fatalf("BestPlay: %v", err)
		}
		e, _, _ := s.Counters()
		fresh += e
	}
	saved := 0.0
	if fresh > 0 {
		saved = 100 * float64(fresh-carried) / float64(fresh)
	}
	fmt.Printf("UNBOUNDED cache: decisions=%d  fresh=%d  carried=%d  saved=%.1f%%\n",
		len(positions), fresh, carried, saved)
}
