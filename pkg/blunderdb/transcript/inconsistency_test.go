package transcript

import (
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// docOf builds a document straight from its Actions — a Transcription is data, and the
// Inconsistencies are exactly the documents no sequence of gestures would produce on
// purpose.
func docOf(length int, actions ...Action) Document {
	doc := New(length)
	doc.Actions = actions
	doc.Cursor = len(actions)
	return doc
}

func opening(side, d1, d2 int) Action {
	return Action{Side: side, Kind: KindOpening, Dice: [2]int{d1, d2}}
}

// firstCandidate records the play a user would pick for that roll, from the position
// the document has reached — the ordinary, legal Action.
func firstCandidate(t *testing.T, doc Document, side, d1, d2 int) Action {
	t.Helper()
	pos := Replay(doc, 0).Next.Position
	pos.Dice = [2]int{d1, d2}
	pos.PlayerOnRoll = side
	plays := domain.LegalMoves(&pos)
	if len(plays) == 0 {
		t.Fatalf("no legal play for %d%d", d1, d2)
	}
	return Action{Side: side, Kind: KindChecker, Dice: [2]int{d1, d2}, Steps: plays[0].Steps}
}

// TestInconsistenciesAreDetectedAndKept walks the five rows of fonctionnel.md §1.4.
// Each is found by the Replay, and each survives it: an Inconsistency is shown, never
// refused, never deleted, and never written into the Match (ADR-0044, ADR-0045 rule 5).
func TestInconsistenciesAreDetectedAndKept(t *testing.T) {
	t.Run("illegal move", func(t *testing.T) {
		// A play that uses one die when both are playable is no legal play, and the
		// board it left is what the Action says it is.
		doc := docOf(7, opening(domain.Black, 6, 3))
		half := Action{Side: domain.Black, Kind: KindChecker, Dice: [2]int{6, 3},
			Steps: []domain.CheckerStep{{From: 24, To: 18}}}
		board := InitialBoard()
		board.Points[24] = domain.Point{Checkers: 1, Color: domain.Black}
		board.Points[18] = domain.Point{Checkers: 1, Color: domain.Black}
		half.BoardAfter = &board
		doc.Actions = append(doc.Actions, half)

		info := Replay(doc, 0).Actions[1]
		if !hasInconsistency(info, IllegalMove) {
			t.Fatalf("not marked illegal: %+v", info.Inconsistencies)
		}
		if info.After != board {
			t.Error("the recorded board is not the one the game goes on from")
		}
		if len(Replay(doc, 0).Document.Actions) != 2 {
			t.Error("the Replay removed an Action")
		}
	})

	t.Run("double turn", func(t *testing.T) {
		doc := docOf(7, opening(domain.Black, 6, 3))
		first := firstCandidate(t, doc, domain.Black, 6, 3)
		doc.Actions = append(doc.Actions, first)
		again := firstCandidate(t, doc, domain.Black, 5, 2)
		doc.Actions = append(doc.Actions, again)

		if !hasInconsistency(Replay(doc, 0).Actions[2], DoubleTurn) {
			t.Error("two plays in a row by one side are not marked")
		}
		// The opening does not count, and neither does the play that follows it.
		if hasInconsistency(Replay(doc, 0).Actions[1], DoubleTurn) {
			t.Error("the play after an opening was counted as a double turn")
		}
	})

	t.Run("impossible cube: the doubler does not hold it", func(t *testing.T) {
		doc := docOf(7, opening(domain.Black, 6, 3))
		doc.Actions = append(doc.Actions,
			firstCandidate(t, doc, domain.Black, 6, 3),
			Action{Side: domain.White, Kind: KindDouble},
			Action{Side: domain.Black, Kind: KindTake},
			// Player 1 owns the cube; player 2 doubling again cannot be.
			Action{Side: domain.White, Kind: KindDouble},
		)
		ann := Replay(doc, 0)
		if hasInconsistency(ann.Actions[2], ImpossibleCube) {
			t.Error("the first double, from a centred cube, is perfectly possible")
		}
		if !hasInconsistency(ann.Actions[4], ImpossibleCube) {
			t.Error("a double by the side that does not hold the cube is not marked")
		}
	})

	t.Run("impossible cube: an answer with no offer", func(t *testing.T) {
		doc := docOf(7, opening(domain.Black, 6, 3))
		doc.Actions = append(doc.Actions,
			firstCandidate(t, doc, domain.Black, 6, 3),
			Action{Side: domain.White, Kind: KindTake},
		)
		if !hasInconsistency(Replay(doc, 0).Actions[2], ImpossibleCube) {
			t.Error("a take with nothing to take is not marked")
		}
	})

	t.Run("impossible cube: the Crawford game", func(t *testing.T) {
		// Two-point match: player 1 wins the first game, so the second is Crawford.
		doc := docOf(2,
			opening(domain.Black, 6, 3),
			Action{Side: domain.White, Kind: KindResign, Level: 1},
			opening(domain.Black, 5, 2),
			Action{Side: domain.Black, Kind: KindDouble},
		)
		ann := Replay(doc, 0)
		if !ann.Games[1].Crawford {
			t.Fatalf("game 2 is not the Crawford game: %+v", ann.Games)
		}
		if !hasInconsistency(ann.Actions[3], ImpossibleCube) {
			t.Error("a double in the Crawford game is not marked")
		}
		// The away score says which game it is, as the glossary's sentinel.
		if got := ann.Actions[2].Before.Score; got != [2]int{domain.Crawford, 2} {
			t.Errorf("Crawford away score = %v, want [1 2]", got)
		}
	})

	t.Run("impossible cube: above the ceiling", func(t *testing.T) {
		doc := docOf(7, opening(domain.Black, 6, 3))
		doc.Header.MaxCube = 1 // the cube may not pass 2
		doc.Actions = append(doc.Actions,
			firstCandidate(t, doc, domain.Black, 6, 3),
			Action{Side: domain.White, Kind: KindDouble},
			Action{Side: domain.Black, Kind: KindTake},
			Action{Side: domain.Black, Kind: KindDouble},
		)
		if !hasInconsistency(Replay(doc, 0).Actions[4], ImpossibleCube) {
			t.Error("a double above the session's ceiling is not marked")
		}
	})

	t.Run("past the end of the match", func(t *testing.T) {
		// One-point match: the first game ends it, and everything after is past it.
		doc := docOf(1,
			opening(domain.Black, 6, 3),
			Action{Side: domain.White, Kind: KindResign, Level: 1},
			opening(domain.Black, 5, 2),
		)
		ann := Replay(doc, 0)
		if !ann.Finished || ann.Winner != domain.Black {
			t.Fatalf("match = finished %v winner %d", ann.Finished, ann.Winner)
		}
		if !hasInconsistency(ann.Actions[2], PastEnd) {
			t.Error("an Action past the end of the match is not marked")
		}
		if len(ann.Document.Actions) != 3 {
			t.Error("an Action past the end was removed")
		}
	})

	t.Run("inconsistent dice requalify the play as illegal", func(t *testing.T) {
		doc := docOf(7, opening(domain.Black, 6, 3))
		played := firstCandidate(t, doc, domain.Black, 6, 3)
		// The roll is corrected afterwards and the play kept: it no longer uses it.
		played.Dice = [2]int{2, 1}
		doc.Actions = append(doc.Actions, played)

		info := Replay(doc, 0).Actions[1]
		if !hasInconsistency(info, InconsistentDice) {
			t.Fatalf("the mismatch with the roll is not marked: %+v", info.Inconsistencies)
		}
		if !hasInconsistency(info, IllegalMove) {
			t.Error("§1.4 requalifies such a play as illegal; it was not")
		}
	})

	t.Run("nothing is refused and nothing is removed", func(t *testing.T) {
		doc := docOf(1,
			opening(domain.Black, 6, 3),
			Action{Side: domain.White, Kind: KindResign, Level: 1},
			opening(domain.Black, 5, 2),
			Action{Side: domain.Black, Kind: KindDouble},
			Action{Side: domain.Black, Kind: KindTake},
		)
		before := append([]Action(nil), doc.Actions...)
		ann := Replay(doc, 0)
		if !ann.Inconsistent() {
			t.Fatal("this document is inconsistent from end to end")
		}
		if !reflect.DeepEqual(doc.Actions, before) {
			t.Error("the Replay changed the document it was given")
		}
		// And it still saves: the Match is written, warnings and all.
		m, games, moves := MatchParts(doc)
		if m == nil || len(games) == 0 || len(moves) == 0 {
			t.Error("an inconsistent document produced no match")
		}
	})
}
