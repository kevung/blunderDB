package transcript

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// handPlayed builds a five-Action document whose third Action is a play entered on
// the board, illegal (one die of two), with the board it left and the board it was
// entered on, as validate records them. The first play is opening, the one a
// correction rewrites.
func handPlayed(t *testing.T, opening []domain.CheckerStep) (Document, Action) {
	t.Helper()
	h := Header{MatchLength: 5}
	doc := Document{Header: h}
	next := func(side int, dice [2]int) {
		pos := Replay(doc, 0).Next.Position
		plays := legalPlays(t, pos, side, dice)
		if len(plays) == 0 {
			t.Fatalf("no play for %v", dice)
		}
		doc.Actions = append(doc.Actions, Action{Side: side, Kind: KindChecker, Dice: dice, Steps: plays[0].Steps})
	}
	doc.Actions = append(doc.Actions, Action{Side: domain.Black, Kind: KindChecker, Dice: [2]int{3, 1}, Steps: opening})
	next(domain.White, [2]int{6, 4})
	start := Replay(doc, 0).Next.Position.Board
	steps := []domain.CheckerStep{{From: 13, To: 11}}
	_, after := resolveSteps(start, domain.Black, steps)
	hand := Action{Side: domain.Black, Kind: KindChecker, Dice: [2]int{2, 1}, Steps: steps,
		BoardAfter: &after, BoardBefore: &start}
	doc.Actions = append(doc.Actions, hand)
	next(domain.White, [2]int{5, 2})
	next(domain.Black, [2]int{4, 3})
	return doc, hand
}

func stripHandBoards(doc Document) Document {
	out := doc.clone()
	for i := range out.Actions {
		out.Actions[i].BoardAfter, out.Actions[i].BoardBefore = nil, nil
	}
	return out
}

// assertReplayedChain holds that every board of doc is the one its plays leave when
// they are replayed: no board kept from before a correction stands in the chain.
func assertReplayedChain(t *testing.T, label string, doc Document) {
	t.Helper()
	got, want := Replay(doc, 0), Replay(stripHandBoards(doc), 0)
	for i := range want.Actions {
		if got.Actions[i].Before.Board != want.Actions[i].Before.Board {
			t.Errorf("%s: Action %d starts from a board the plays do not lead to", label, i)
		}
		if got.Actions[i].After != want.Actions[i].After {
			t.Errorf("%s: Action %d leaves a board the plays do not lead to", label, i)
		}
	}
	if got.Next.Position.Board != want.Next.Position.Board {
		t.Errorf("%s: the next position is not the replayed one", label)
	}
	assertFifteenEach(t, label, got)
}

// TestCorrectionUpstreamRunsThroughAHandBoard: correcting a play before a board
// entered by hand changes every board after it, the hand board included.
func TestCorrectionUpstreamRunsThroughAHandBoard(t *testing.T) {
	doc, hand := handPlayed(t, []domain.CheckerStep{{From: 8, To: 5}, {From: 6, To: 5}})
	if got := Replay(doc, 0).Actions[2].After; got != *hand.BoardAfter {
		t.Fatal("uncorrected: the hand board is not the one the game goes on from")
	}

	corrected := doc.clone()
	corrected.Actions[0].Steps = []domain.CheckerStep{{From: 24, To: 21}, {From: 21, To: 20}}
	ann := Replay(corrected, 0)
	if ann.Actions[2].After == *hand.BoardAfter {
		t.Fatal("the hand board masks the correction")
	}
	if !hasInconsistency(ann.Actions[2], IllegalMove) {
		t.Error("the one-die play is no longer marked illegal")
	}
	assertReplayedChain(t, "corrected", corrected)

	// Undoing the correction gives the hand board back: it is a fact of the board
	// it was entered on.
	if got := Replay(stripHandBoards(corrected), 0); got.Actions[2].After == *hand.BoardAfter {
		t.Fatal("test setup: the correction does not reach the hand play")
	}
	if got := Replay(doc, 0).Actions[2].After; got != *hand.BoardAfter {
		t.Error("the hand board is lost once its board is back")
	}
}

// TestHandBoardHoldsOnItsOwnBoard: a board given by hand that the steps do not
// describe stands as long as nothing upstream changes.
func TestHandBoardHoldsOnItsOwnBoard(t *testing.T) {
	doc, hand := handPlayed(t, []domain.CheckerStep{{From: 8, To: 5}, {From: 6, To: 5}})
	moved := *hand.BoardAfter
	moved.Points[11].Checkers--
	if moved.Points[11].Checkers == 0 {
		moved.Points[11].Color = domain.None
	}
	moved.Points[10] = domain.Point{Checkers: moved.Points[10].Checkers + 1, Color: domain.Black}
	doc.Actions[2].BoardAfter = &moved

	ann := Replay(doc, 0)
	if ann.Actions[2].After != moved {
		t.Fatal("a hand board on an unchanged board is not kept")
	}
	if ann.Actions[3].Before.Board != moved {
		t.Error("the next play does not start from the hand board")
	}
	assertFifteenEach(t, "kept", ann)
}

// TestDraftWithoutBoardBefore: a draft saved before BoardBefore existed keeps a hand
// board only where its steps lead, so a correction upstream still runs down the chain.
func TestDraftWithoutBoardBefore(t *testing.T) {
	doc, hand := handPlayed(t, []domain.CheckerStep{{From: 8, To: 5}, {From: 6, To: 5}})
	doc.Actions[2].BoardBefore = nil
	if got := Replay(doc, 0).Actions[2].After; got != *hand.BoardAfter {
		t.Fatal("old draft, uncorrected: the hand board is not the one the game goes on from")
	}
	doc.Actions[0].Steps = []domain.CheckerStep{{From: 24, To: 21}, {From: 21, To: 20}}
	assertReplayedChain(t, "old draft corrected", doc)
}

// TestValidateStampsTheHandBoard: a play entered on the board records the board it
// was entered on next to the board it left.
func TestValidateStampsTheHandBoard(t *testing.T) {
	doc := New(5)
	steps := []domain.CheckerStep{{From: 8, To: 5}}
	_, after := resolveSteps(InitialBoard(), domain.Black, steps)
	var err error
	for _, g := range []Gesture{die(3), die(1), {Kind: GestureEnterPlay, Steps: steps, BoardAfter: &after}, confirm()} {
		if doc, err = Apply(doc, g); err != nil {
			t.Fatalf("%s: %v", g.Kind, err)
		}
	}
	a := doc.Actions[0]
	if a.BoardAfter == nil || a.BoardBefore == nil || *a.BoardBefore != InitialBoard() {
		t.Fatalf("hand play recorded without the board it was entered on: %+v", a)
	}
}
