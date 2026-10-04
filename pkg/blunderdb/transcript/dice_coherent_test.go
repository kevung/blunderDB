package transcript

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// TestDiceCoherentCompoundSteps: a step played with several dice is coherent when the
// intermediate points are ones the mover may land on.
func TestDiceCoherentCompoundSteps(t *testing.T) {
	blocked := func(pt int) domain.Board {
		var b domain.Board
		b.Points[pt] = domain.Point{Color: domain.White, Checkers: 2}
		return b
	}
	blot := func(pt int) domain.Board {
		var b domain.Board
		b.Points[pt] = domain.Point{Color: domain.White, Checkers: 1}
		return b
	}
	step := func(from, to int) []domain.CheckerStep { return []domain.CheckerStep{{From: from, To: to}} }
	cases := []struct {
		name  string
		board domain.Board
		steps []domain.CheckerStep
		dice  [2]int
		want  bool
	}{
		{"24/14 on 6-4", domain.Board{}, step(24, 14), [2]int{6, 4}, true},
		{"24/14 on 4-6", domain.Board{}, step(24, 14), [2]int{4, 6}, true},
		{"24/14 on 6-5", domain.Board{}, step(24, 14), [2]int{6, 5}, false},
		{"24/14 on 6-4, 18 blocked and 20 open", blocked(18), step(24, 14), [2]int{6, 4}, true},
		{"24/14 on 6-4, both intermediates blocked", func() domain.Board {
			b := blocked(18)
			b.Points[20] = domain.Point{Color: domain.White, Checkers: 3}
			return b
		}(), step(24, 14), [2]int{6, 4}, false},
		{"24/14 on 6-4, hit on the intermediate", blot(18), step(24, 14), [2]int{6, 4}, true},
		{"24/16 on 4-4", domain.Board{}, step(24, 16), [2]int{4, 4}, true},
		{"24/12 on 4-4", domain.Board{}, step(24, 12), [2]int{4, 4}, true},
		{"24/8 on 4-4", domain.Board{}, step(24, 8), [2]int{4, 4}, true},
		{"24/4 on 4-4", domain.Board{}, step(24, 4), [2]int{4, 4}, false},
		{"24/16 on 4-4 with 20 blocked", blocked(20), step(24, 16), [2]int{4, 4}, false},
		{"compound step leaves no die for a second one", domain.Board{},
			[]domain.CheckerStep{{From: 24, To: 14}, {From: 8, To: 4}}, [2]int{6, 4}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := diceCoherent(c.board, c.steps, c.dice, domain.Black); got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}
