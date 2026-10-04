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
	barBoard := func(int) domain.Board { return domain.Board{} }
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
		{"bar/15 on 6-4", barBoard(domain.Black), step(domain.BlackBar, 15), [2]int{6, 4}, true},
		{"bar/15 on 6-5", barBoard(domain.Black), step(domain.BlackBar, 15), [2]int{6, 5}, false},
		{"bar/16 on 3-3", barBoard(domain.Black), step(domain.BlackBar, 16), [2]int{3, 3}, true},
		{"bar/15 on 6-4, entry point blocked, other entry open", blocked(19), step(domain.BlackBar, 15), [2]int{6, 4}, true},
		{"6/off on 4-2", domain.Board{}, step(6, domain.Off), [2]int{4, 2}, true},
		{"5/off on 3-2", domain.Board{}, step(5, domain.Off), [2]int{3, 2}, true},
		{"6/off on 3-2", domain.Board{}, step(6, domain.Off), [2]int{3, 2}, false},
		{"6/off on 4-3 over-covers on the last die", domain.Board{}, step(6, domain.Off), [2]int{4, 3}, true},
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

func TestDiceCoherentWhiteCompound(t *testing.T) {
	step := []domain.CheckerStep{{From: 1, To: 11}}
	if !diceCoherent(domain.Board{}, step, [2]int{6, 4}, domain.White) {
		t.Error("1/11 on 6-4 is coherent for White")
	}
	if diceCoherent(domain.Board{}, step, [2]int{6, 5}, domain.White) {
		t.Error("1/11 on 6-5 is not coherent")
	}
	if !diceCoherent(domain.Board{}, []domain.CheckerStep{{From: domain.WhiteBar, To: 10}}, [2]int{6, 4}, domain.White) {
		t.Error("bar/10 on 6-4 is coherent for White")
	}
	if !diceCoherent(domain.Board{}, []domain.CheckerStep{{From: 20, To: domain.Off}}, [2]int{4, 1}, domain.White) {
		t.Error("20/off on 4-1 is coherent for White")
	}
}

func TestReplayCompoundStepIsNotInconsistentDice(t *testing.T) {
	doc := docOf(7)
	played := firstCandidate(t, doc, domain.Black, 6, 4)
	played.Steps = []domain.CheckerStep{{From: 24, To: 14}}
	doc.Actions = append(doc.Actions, played)
	if info := Replay(doc, 0).Actions[0]; hasInconsistency(info, InconsistentDice) {
		t.Errorf("24/14 on 6-4 marked: %+v", info.Inconsistencies)
	}
}
