// SPDX-License-Identifier: MIT

package gammonnet

import (
	"math/rand"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// A targeted re-run and the full search agree on the play the full search
// deepens first: the panel replaces a row with it, and the row must not move
// for having been asked alone.
func TestEvaluatePlaysAgreesWithTheFullSearch(t *testing.T) {
	plies := []int{0, 1}
	if !testing.Short() {
		plies = append(plies, 2)
	}
	rng := rand.New(rand.NewSource(20261128))
	scores := [][2]int{{-1, -1}, {3, 5}, {2, 2}}
	for _, ply := range plies {
		checked := 0
		for attempt := 0; checked < 3 && attempt < 200; attempt++ {
			pos := randomBoard(rng, domain.White+attempt%2)
			pos.Score = scores[attempt%len(scores)]
			pos.Dice = [2]int{1 + rng.Intn(6), 1 + rng.Intn(6)}
			full, err := EvaluatePositionWithMET(nil, pos, nil, ply, 0, 0)
			if err != nil || len(full.Moves) < 3 {
				continue
			}
			checked++
			last := full.Moves[len(full.Moves)-1].Move
			got, err := EvaluatePlaysWithMET(nil, pos, nil, ply, 0, []string{last, full.Moves[0].Move})
			if err != nil {
				t.Fatalf("ply %d: %v", ply, err)
			}
			if len(got) != 2 {
				t.Fatalf("ply %d: %d rows for two plays", ply, len(got))
			}
			if got[0].Move != full.Moves[0].Move || got[0].Equity != full.Moves[0].Equity {
				t.Fatalf("ply %d: best %s %.6f alone, %s %.6f in the full search", ply, got[0].Move, got[0].Equity, full.Moves[0].Move, full.Moves[0].Equity)
			}
			if got[0].EquityError != nil || got[1].EquityError == nil || *got[1].EquityError != got[0].Equity-got[1].Equity {
				t.Fatalf("ply %d: errors not measured against the best asked play", ply)
			}
			if got[1].AnalysisDepth != DepthLabel(ply) {
				t.Fatalf("ply %d: depth %q", ply, got[1].AnalysisDepth)
			}
		}
		if checked == 0 {
			t.Fatalf("ply %d: no position checked", ply)
		}
	}
}

func TestEvaluatePlaysRefusesAPlayThatIsNotLegal(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	pos := randomBoard(rng, domain.White)
	pos.Score = [2]int{-1, -1}
	pos.Dice = [2]int{3, 1}
	if _, err := EvaluatePlaysWithMET(nil, pos, nil, 0, 0, []string{"25/1(15)"}); err == nil {
		t.Fatal("an illegal play was evaluated")
	}
	pos.Dice = [2]int{0, 0}
	if _, err := EvaluatePlaysWithMET(nil, pos, nil, 0, 0, []string{"8/5 6/5"}); err == nil {
		t.Fatal("plays were evaluated without dice")
	}
}
