package transcript

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// checkerCount is what a side holds on a board: its points, its bar and what it
// has borne off. Every board a Replay hands out holds exactly 15 of each.
func checkerCount(b domain.Board, side int) int {
	n := b.Bearoff[side]
	for _, p := range b.Points {
		if p.Color == side {
			n += p.Checkers
		}
	}
	return n
}

func assertFifteenEach(t *testing.T, label string, ann Annotated) {
	t.Helper()
	for i, info := range ann.Actions {
		for side := 0; side < 2; side++ {
			if info.HasPosition {
				if n := checkerCount(info.Before.Board, side); n != domain.CheckersPerPlayer {
					t.Errorf("%s: before Action %d, side %d holds %d checkers", label, i, side, n)
				}
			}
			if n := checkerCount(info.After, side); n != domain.CheckersPerPlayer {
				t.Errorf("%s: after Action %d, side %d holds %d checkers", label, i, side, n)
			}
		}
	}
	for side := 0; side < 2; side++ {
		if n := checkerCount(ann.Next.Position.Board, side); n != domain.CheckersPerPlayer {
			t.Errorf("%s: next position, side %d holds %d checkers", label, side, n)
		}
	}
}

func hits(p domain.LegalPlay) bool {
	for _, s := range p.Steps {
		if s.Hit {
			return true
		}
	}
	return false
}

func legalPlays(t *testing.T, pos domain.Position, side int, dice [2]int) []domain.LegalPlay {
	t.Helper()
	pos.Dice, pos.PlayerOnRoll, pos.DecisionType = dice, side, domain.CheckerAction
	return domain.LegalMoves(&pos)
}

// hitThenEnter finds three plays: player 1 leaves a blot, player 2 hits it (and
// could have played a quiet move instead), player 1 enters from the bar. It
// returns the document and the quiet alternative to the hit.
func hitThenEnter(t *testing.T) (Document, []domain.CheckerStep) {
	t.Helper()
	rolls := [][2]int{}
	for a := 1; a <= 6; a++ {
		for b := 1; b <= 6; b++ {
			rolls = append(rolls, [2]int{a, b})
		}
	}
	h := Header{MatchLength: 5}
	for _, d0 := range rolls {
		if d0[0] <= d0[1] {
			continue
		}
		start := Replay(Document{Header: h}, 0).Next.Position
		for _, p0 := range legalPlays(t, start, domain.Black, d0) {
			a0 := Action{Side: domain.Black, Kind: KindChecker, Dice: d0, Steps: p0.Steps}
			pos1 := Replay(Document{Header: h, Actions: []Action{a0}}, 0).Next.Position
			for _, d1 := range rolls {
				var hit, quiet *domain.LegalPlay
				plays := legalPlays(t, pos1, domain.White, d1)
				for i := range plays {
					if hits(plays[i]) && hit == nil {
						hit = &plays[i]
					} else if !hits(plays[i]) && quiet == nil {
						quiet = &plays[i]
					}
				}
				if hit == nil || quiet == nil {
					continue
				}
				a1 := Action{Side: domain.White, Kind: KindChecker, Dice: d1, Steps: hit.Steps}
				pos2 := Replay(Document{Header: h, Actions: []Action{a0, a1}}, 0).Next.Position
				for _, d2 := range rolls {
					for _, p2 := range legalPlays(t, pos2, domain.Black, d2) {
						if p2.Steps[0].From != barOf(domain.Black) && p2.Steps[len(p2.Steps)-1].From != barOf(domain.Black) {
							continue
						}
						a2 := Action{Side: domain.Black, Kind: KindChecker, Dice: d2, Steps: p2.Steps}
						doc := Document{Header: h, Actions: []Action{a0, a1, a2}}
						return doc, quiet.Steps
					}
				}
			}
		}
	}
	t.Fatal("no hit-then-enter sequence found")
	return Document{}, nil
}

// TestCorrectionReplaysWhatFollows holds the replay after a correction in the middle
// of a draft: the Actions after the corrected one are replayed from the new board,
// and a play that no longer fits it never conjures or loses a checker.
func TestCorrectionReplaysWhatFollows(t *testing.T) {
	doc, quiet := hitThenEnter(t)

	t.Run("a hit corrected into a quiet play", func(t *testing.T) {
		ed := NewEditor(doc)
		before := ed.Replay(0)
		assertFifteenEach(t, "as typed", before)
		if before.Inconsistent() {
			t.Fatalf("the typed sequence is already inconsistent: %+v", before.Actions)
		}

		ed.Doc.Actions = append([]Action(nil), ed.Doc.Actions...)
		ed.Doc.Actions[1].Steps = quiet
		after := ed.Replay(1)
		assertFifteenEach(t, "corrected", after)
		if full := Replay(ed.Doc, 1); full.Actions[2].After != after.Actions[2].After {
			t.Errorf("incremental replay differs from a full one after Action 2")
		}
		// The entry from the bar now has no checker to move: it is marked.
		if len(after.Actions[2].Inconsistencies) == 0 {
			t.Error("an entry from an empty bar must be marked inconsistent")
		}
		if after.Actions[2].Before.Board != after.Actions[1].After {
			t.Error("Action 2 must be replayed from the board Action 1 now leaves")
		}
	})

	t.Run("a quiet play corrected into a hit", func(t *testing.T) {
		quietDoc := doc
		quietDoc.Actions = append([]Action(nil), doc.Actions...)
		hit := quietDoc.Actions[1].Steps
		quietDoc.Actions[1].Steps = quiet
		ed := NewEditor(quietDoc)
		assertFifteenEach(t, "as typed", ed.Replay(0))

		ed.Doc.Actions = append([]Action(nil), ed.Doc.Actions...)
		ed.Doc.Actions[1].Steps = hit
		assertFifteenEach(t, "corrected", ed.Replay(1))
	})

	// The correction empties the point a later play of the same side starts from:
	// that play is replayed as written, from nothing.
	t.Run("a correction empties the source of a later play", func(t *testing.T) {
		st := func(from, to int) domain.CheckerStep { return domain.CheckerStep{From: from, To: to} }
		doc := Document{Header: Header{MatchLength: 5}, Actions: []Action{
			{Side: domain.Black, Kind: KindChecker, Dice: [2]int{4, 3}, Steps: []domain.CheckerStep{st(13, 9), st(13, 10)}},
			{Side: domain.White, Kind: KindChecker, Dice: [2]int{6, 1}, Steps: []domain.CheckerStep{st(12, 18), st(17, 18)}},
			{Side: domain.Black, Kind: KindChecker, Dice: [2]int{2, 1}, Steps: []domain.CheckerStep{st(24, 22), st(24, 23)}},
		}}
		ed := NewEditor(doc)
		typed := ed.Replay(0)
		assertFifteenEach(t, "as typed", typed)
		if typed.Inconsistent() {
			t.Fatal("the typed sequence is already inconsistent")
		}

		ed.Doc.Actions = append([]Action(nil), ed.Doc.Actions...)
		ed.Doc.Actions[0].Steps = []domain.CheckerStep{st(24, 20), st(24, 21)}
		after := ed.Replay(0)
		assertFifteenEach(t, "corrected", after)
		if len(after.Actions[2].Inconsistencies) == 0 {
			t.Error("a play from an emptied point must be marked inconsistent")
		}
		if after.Actions[2].After != after.Actions[2].Before.Board {
			t.Error("a play whose checkers are gone must leave the board as it found it")
		}
	})
}

// TestReplayConservesCheckersOnAnyStep holds the same count for a play written
// against the wrong board: from an empty point, from the opponent's point, onto a
// point the opponent holds.
func TestReplayConservesCheckersOnAnyStep(t *testing.T) {
	start := InitialBoard()
	for from := 0; from <= 25; from++ {
		for to := -1; to <= 25; to++ {
			for side := 0; side < 2; side++ {
				_, b := resolveSteps(start, side, []domain.CheckerStep{{From: from, To: to}})
				for s := 0; s < 2; s++ {
					if n := checkerCount(b, s); n != domain.CheckersPerPlayer {
						t.Fatalf("step %d/%d by side %d leaves side %d with %d checkers", from, to, side, s, n)
					}
				}
			}
		}
	}
}

// TestReplayIgnoresAMiscountedBoardAfter holds that a board entered by hand with a
// checker too many is not handed on: the play's steps are replayed instead.
func TestReplayIgnoresAMiscountedBoardAfter(t *testing.T) {
	bad := InitialBoard()
	bad.Points[20] = domain.Point{Checkers: 1, Color: domain.Black}
	doc := Document{Header: Header{MatchLength: 5}, Actions: []Action{
		{Side: domain.Black, Kind: KindChecker, Dice: [2]int{4, 3},
			Steps: []domain.CheckerStep{{From: 7, To: 3}}, BoardAfter: &bad},
	}}
	assertFifteenEach(t, "miscounted", Replay(doc, 0))
}
