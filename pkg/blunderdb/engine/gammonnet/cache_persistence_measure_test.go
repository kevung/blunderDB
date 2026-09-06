package gammonnet

import (
	"fmt"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// Mesure du cache persistant (#297, fiche J.7) — et la raison pour laquelle il
// n'a pas été écrit.
//
// RÉSULTAT, mesuré le 2026-09-06 sur seize décisions consécutives d'une même
// partie, à 2-ply : porter le cache d'une décision à la suivante économise
// 2,8 % des évaluations du réseau avec le cache par défaut (2^16 entrées), et
// 3,8 % avec un cache assez grand pour ne JAMAIS évincer (2^22, ~240 Mo).
//
// 3,8 % est le plafond absolu de ce qu'un cache persistant pourrait rendre :
// pas d'éviction, pas de contrôle de version, pas d'entrées-sorties. Un vrai
// cache sur disque paierait les trois et rendrait moins. Et son bénéficiaire
// naturel — la réanalyse d'une base — est précisément le cas où le fichier
// doit être JETÉ, puisqu'il est indexé par la version du réseau et que
// `analyze --stale` existe pour le cas où ce réseau a changé.
//
// La moitié « cache persistant » de la fiche est donc écartée, avec ce
// fichier pour preuve plutôt que sur une intuition. Ces deux mesures restent
// dans l'arbre : elles se relancent, et c'est ce qui rendra le refus révisable
// si un jour la recherche change de forme.
//
// TestMeasureCacheAcrossDecisions measures what a cache that OUTLIVES one
// decision would actually save.
//
// The question the fiche leaves open is not whether the cache would be exact —
// it is, by construction, since the key is the whole position and the value is
// the network's own output. The question is whether it would save anything.
//
// The measurement walks one game, evaluating each position at 2-ply, twice:
// once with a single Searcher whose cache carries from one decision to the
// next, and once with a fresh Searcher per decision — which is today's
// behaviour for the live panel, and across runs of the batch. The gap between
// the two eval counts is the whole prize.
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
// cache large enough never to evict (2^22 entries, ~240 MB), which is the
// UPPER BOUND of what any persistent cache could ever save: no eviction, no
// version gate, no I/O. A persistent cache cannot beat this number; it can
// only pay more for less.
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
