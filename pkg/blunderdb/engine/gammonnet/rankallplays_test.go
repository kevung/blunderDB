// SPDX-License-Identifier: MIT

package gammonnet

import (
	"fmt"
	"math/rand"
	"reflect"
	"sync"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// rankingPositions is a mixed run of checker questions — money and match
// scores, centred and owned cubes — so a pooled searcher is re-aimed at a
// different referential at nearly every call.
func rankingPositions(t *testing.T) []domain.Position {
	t.Helper()
	rng := rand.New(rand.NewSource(20261022))
	scores := [][2]int{{-1, -1}, {3, 5}, {1, 1}, {7, 2}, {2, 4}}
	cubes := []domain.Cube{
		{Owner: domain.None, Value: 0},
		{Owner: domain.White, Value: 1},
		{Owner: domain.Black, Value: 2},
	}
	var out []domain.Position
	for attempt := 0; len(out) < 12 && attempt < 400; attempt++ {
		onRoll := domain.White
		if attempt%2 == 1 {
			onRoll = domain.Black
		}
		pos := randomBoard(rng, onRoll)
		pos.Score = scores[attempt%len(scores)]
		pos.Cube = cubes[attempt%len(cubes)]
		pos.Dice = [2]int{1 + rng.Intn(6), 1 + rng.Intn(6)}
		if _, err := freshRanking(pos); err != nil {
			continue
		}
		out = append(out, pos)
	}
	if len(out) == 0 {
		t.Fatal("no position could be ranked: the test proves nothing")
	}
	return out
}

// freshRanking is RankAllPlaysWithMET on a searcher built for the call alone.
func freshRanking(pos domain.Position) ([]domain.CheckerMove, error) {
	cfg, _, err := ConfigForPositionMET(&pos, nil, 0, 0)
	if err != nil {
		return nil, err
	}
	cfg.PruneK = 0
	s, err := NewSearcher(cfg)
	if err != nil {
		return nil, err
	}
	return RankAllPlaysWithMET(s, pos, nil)
}

// TestRankAllPlaysConcurrentPooledMatchesFresh: the pooled searcher a nil argument takes
// answers bit for bit what a brand-new one does, from goroutines that share
// the pool at once (the CI race job selects it by name). No configuration or cache entry
// of one caller leaks into another's ranking.
func TestRankAllPlaysConcurrentPooledMatchesFresh(t *testing.T) {
	positions := rankingPositions(t)
	want := make([][]domain.CheckerMove, len(positions))
	for i, pos := range positions {
		moves, err := freshRanking(pos)
		if err != nil {
			t.Fatalf("position %d: fresh searcher: %v", i, err)
		}
		if len(moves) == 0 {
			t.Fatalf("position %d (dice %v): no play ranked", i, pos.Dice)
		}
		want[i] = moves
	}

	const callers, rounds = 8, 3
	var wg sync.WaitGroup
	errs := make(chan string, callers*rounds*len(positions))
	for c := range callers {
		wg.Go(func() {
			for r := range rounds {
				// Each caller walks the run from its own offset, so
				// concurrent calls are aimed at different referentials.
				for k := range positions {
					i := (k + c + r) % len(positions)
					got, err := RankAllPlaysWithMET(nil, positions[i], nil)
					if err != nil {
						errs <- fmt.Sprintf("position %d: pooled searcher: %v", i, err)
						continue
					}
					if !reflect.DeepEqual(got, want[i]) {
						errs <- fmt.Sprintf("position %d (score %v, dice %v): pooled ranking differs from the fresh searcher's", i, positions[i].Score, positions[i].Dice)
					}
				}
			}
		})
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Fatal(e)
	}
}
