package transcript

import (
	"reflect"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// step is one gesture of a scenario and what the document must look like once it has
// been applied.
type step struct {
	name  string
	g     Gesture
	check func(t *testing.T, doc Document)
}

// runSteps applies the gestures in order, checking the document after each one. A
// gesture that fails stops the scenario: Apply is pure, so a failure would leave every
// later check reading a document that never existed.
func runSteps(t *testing.T, doc Document, steps []step) Document {
	t.Helper()
	for _, s := range steps {
		next, err := Apply(doc, s.g)
		if err != nil {
			t.Fatalf("%s: %v", s.name, err)
		}
		doc = next
		if s.check != nil {
			s.check(t, doc)
		}
	}
	return doc
}

func die(n int) Gesture { return Gesture{Kind: GestureEnterDie, Die: n} }
func confirm() Gesture  { return Gesture{Kind: GestureValidate} }
func candidate(i int) Gesture {
	return Gesture{Kind: GestureSelectCandidate, Candidate: i}
}

// openedMatch returns a draft whose opening has been recorded: player 1 won it 6-3 and
// is on roll with that same roll.
func openedMatch(t *testing.T, length int) Document {
	t.Helper()
	doc, err := Apply(New(length), Gesture{Kind: GestureEnterDie, Die: 6})
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range []Gesture{die(3), confirm()} {
		if doc, err = Apply(doc, g); err != nil {
			t.Fatal(err)
		}
	}
	return doc
}

func lastAction(t *testing.T, doc Document) Action {
	t.Helper()
	if len(doc.Actions) == 0 {
		t.Fatal("the document has no action")
	}
	return doc.Actions[len(doc.Actions)-1]
}

// TestGesturesByKind walks one synthetic document per kind of Action through the
// gestures that record it, checking the document after every gesture — the table of
// fonctionnel.md §1.2 read through the gestures of §2.
func TestGesturesByKind(t *testing.T) {
	t.Run("opening", func(t *testing.T) {
		runSteps(t, New(7), []step{
			{"first die", die(6), func(t *testing.T, doc Document) {
				if doc.Entry == nil || doc.Entry.Dice != [2]int{6, 0} {
					t.Fatalf("entry dice = %v", doc.Entry)
				}
			}},
			{"second die", die(3), func(t *testing.T, doc Document) {
				if doc.Entry.Dice != [2]int{6, 3} {
					t.Fatalf("entry dice = %v", doc.Entry.Dice)
				}
			}},
			{"a third die starts the roll again", die(5), func(t *testing.T, doc Document) {
				if doc.Entry.Dice != [2]int{5, 0} {
					t.Fatalf("entry dice = %v", doc.Entry.Dice)
				}
			}},
			{"back to 6-3", die(3), nil},
			{"validate", confirm(), func(t *testing.T, doc Document) {
				a := lastAction(t, doc)
				if a.Kind != KindOpening || a.Dice != [2]int{5, 3} {
					t.Fatalf("action = %+v", a)
				}
				// Player 1 rolled the higher die, so player 1 starts.
				if a.Side != domain.Black {
					t.Errorf("side = %d, want player 1", a.Side)
				}
				if doc.Cursor != 1 {
					t.Errorf("cursor = %d, want 1", doc.Cursor)
				}
				ann := Replay(doc, 0)
				if ann.Next.Expects != KindChecker || ann.Next.Side != domain.Black {
					t.Errorf("next = %+v", ann.Next)
				}
				if len(ann.Games) != 1 || ann.Games[0].Number != 1 {
					t.Errorf("games = %+v", ann.Games)
				}
			}},
		})
	})

	t.Run("opening tie is kept and re-rolled", func(t *testing.T) {
		doc := runSteps(t, New(7), []step{
			{"4", die(4), nil}, {"4", die(4), nil},
			{"validate", confirm(), func(t *testing.T, doc Document) {
				ann := Replay(doc, 0)
				if ann.Next.Expects != KindOpening {
					t.Errorf("a tie must be followed by another opening, got %s", ann.Next.Expects)
				}
				if len(ann.Games) != 1 {
					t.Errorf("a tie opens no second game: %d games", len(ann.Games))
				}
			}},
		})
		// The tie stays in the document: nothing removes an Action.
		if len(doc.Actions) != 1 || doc.Actions[0].Dice != [2]int{4, 4} {
			t.Errorf("the tie was not kept: %+v", doc.Actions)
		}
		// And it produces neither Move nor Position.
		_, _, moves := MatchParts(doc)
		for _, mvs := range moves {
			if len(mvs) != 0 {
				t.Errorf("an opening produced %d moves", len(mvs))
			}
		}
	})

	t.Run("checker", func(t *testing.T) {
		doc := openedMatch(t, 7)
		if doc.Entry != nil {
			t.Fatalf("validation left an entry behind: %+v", doc.Entry)
		}
		// The roll of the opening is not typed again.
		cands := Candidates(doc)
		if len(cands) == 0 {
			t.Fatal("no candidate for the opening roll")
		}
		runSteps(t, doc, []step{
			{"select the first candidate", candidate(0), func(t *testing.T, doc Document) {
				if !doc.Entry.Selected || len(doc.Entry.Steps) == 0 {
					t.Fatalf("entry = %+v", doc.Entry)
				}
			}},
			{"validate", confirm(), func(t *testing.T, doc Document) {
				a := lastAction(t, doc)
				if a.Kind != KindChecker || a.Side != domain.Black || a.Dice != [2]int{6, 3} {
					t.Fatalf("action = %+v", a)
				}
				if a.BoardAfter != nil {
					t.Error("a legal play stores no board: the board is derived")
				}
				ann := Replay(doc, 0)
				info := ann.Actions[len(ann.Actions)-1]
				if len(info.Inconsistencies) != 0 {
					t.Errorf("a candidate is legal by construction: %+v", info.Inconsistencies)
				}
				if info.After != cands[0].Result.Board {
					t.Error("the board left is not the candidate's")
				}
				if ann.Next.Side != domain.White {
					t.Errorf("the turn did not pass: next side %d", ann.Next.Side)
				}
			}},
		})
	})

	t.Run("dance", func(t *testing.T) {
		doc := openedMatch(t, 7)
		doc = runSteps(t, doc, []step{
			{"select", candidate(0), nil},
			{"validate", confirm(), nil},
			{"a roll for player 2", die(2), nil},
			{"and its second die", die(1), nil},
			{"dance", Gesture{Kind: GestureDance}, func(t *testing.T, doc Document) {
				a := lastAction(t, doc)
				if a.Kind != KindDance || a.Side != domain.White || a.Dice != [2]int{2, 1} {
					t.Fatalf("action = %+v", a)
				}
			}},
		})
		// A dance is a Move like any checker Move, and reads "Cannot Move".
		_, games, moves := MatchParts(doc)
		last := moves[games[0].ID][len(moves[games[0].ID])-1]
		if last.MoveType != "checker" || last.CheckerMove != "Cannot Move" {
			t.Errorf("dance move = %+v", last)
		}
	})

	t.Run("double and take", func(t *testing.T) {
		runSteps(t, openedMatch(t, 7), []step{
			{"select", candidate(0), nil},
			{"validate", confirm(), nil},
			{"double", Gesture{Kind: GestureDouble}, func(t *testing.T, doc Document) {
				a := lastAction(t, doc)
				if a.Kind != KindDouble || a.Side != domain.White {
					t.Fatalf("action = %+v", a)
				}
				ann := Replay(doc, 0)
				if ann.Next.Expects != KindTake || ann.Next.Side != domain.Black {
					t.Errorf("next = %+v", ann.Next)
				}
			}},
			{"take", Gesture{Kind: GestureTake}, func(t *testing.T, doc Document) {
				a := lastAction(t, doc)
				if a.Kind != KindTake || a.Side != domain.Black {
					t.Fatalf("action = %+v", a)
				}
				ann := Replay(doc, 0)
				// The cube is at 2 and player 1 owns it; the doubler rolls next.
				if got := ann.Next.Position.Cube; got.Value != 1 || got.Owner != domain.Black {
					t.Errorf("cube = %+v", got)
				}
				if ann.Next.Side != domain.White {
					t.Errorf("next side = %d, want the doubler", ann.Next.Side)
				}
				// The taker's own decision was on the cube AT THE OFFERED LEVEL.
				taken := ann.Actions[len(ann.Actions)-1]
				if taken.Before.Cube.Value != 1 || taken.Before.DecisionType != domain.CubeAction {
					t.Errorf("take position = %+v", taken.Before.Cube)
				}
				if len(taken.Inconsistencies) != 0 {
					t.Errorf("an answered double is consistent: %+v", taken.Inconsistencies)
				}
			}},
		})
	})

	t.Run("pass ends the game at the value before the offer", func(t *testing.T) {
		runSteps(t, openedMatch(t, 7), []step{
			{"select", candidate(0), nil},
			{"validate", confirm(), nil},
			{"double", Gesture{Kind: GestureDouble}, nil},
			{"pass", Gesture{Kind: GesturePass}, func(t *testing.T, doc Document) {
				ann := Replay(doc, 0)
				g := ann.Games[0]
				if g.Winner != domain.White || g.PointsWon != 1 {
					t.Errorf("game = %+v, want player 2 winning 1 point", g)
				}
				if ann.Score != [2]int{0, 1} {
					t.Errorf("score = %v", ann.Score)
				}
				if ann.Next.Expects != KindOpening || ann.Next.GameNumber != 2 {
					t.Errorf("next = %+v", ann.Next)
				}
			}},
		})
	})

	t.Run("resign", func(t *testing.T) {
		doc := openedMatch(t, 7)
		doc = runSteps(t, doc, []step{
			{"resign a gammon", Gesture{Kind: GestureResign, HasSide: true, Side: domain.White, Level: 2},
				func(t *testing.T, doc Document) {
					a := lastAction(t, doc)
					if a.Kind != KindResign || a.Side != domain.White || a.Level != 2 {
						t.Fatalf("action = %+v", a)
					}
					ann := Replay(doc, 0)
					g := ann.Games[0]
					if g.Winner != domain.Black || g.PointsWon != 2 {
						t.Errorf("game = %+v, want player 1 winning 2 points", g)
					}
				}},
		})
		// A resignation adds no Move at all — only the winner and the points.
		_, games, moves := MatchParts(doc)
		if n := len(moves[games[0].ID]); n != 0 {
			t.Errorf("a resignation produced %d moves", n)
		}
		if games[0].Winner != 0 || games[0].PointsWon != 2 {
			t.Errorf("game = %+v", games[0])
		}
	})
}

// TestGesturesOnTheDocument covers the gestures that move the Cursor or edit an Action
// already recorded — the second half of fonctionnel.md §2.
func TestGesturesOnTheDocument(t *testing.T) {
	base := func(t *testing.T) Document {
		doc := openedMatch(t, 7)
		return runSteps(t, doc, []step{
			{"select", candidate(0), nil},
			{"validate", confirm(), nil},
			{"roll for player 2", die(5), nil},
			{"and its second die", die(4), nil},
			{"select", candidate(0), nil},
			{"validate", confirm(), nil},
			{"player 1 rolls again", die(4), nil},
			{"and its second die", die(2), nil},
			{"select", candidate(0), nil},
			{"validate", confirm(), nil},
		})
	}

	t.Run("cursor back loads the action it lands on", func(t *testing.T) {
		doc := base(t)
		back, err := Apply(doc, Gesture{Kind: GestureCursorBack})
		if err != nil {
			t.Fatal(err)
		}
		if back.Cursor != len(doc.Actions)-1 {
			t.Fatalf("cursor = %d", back.Cursor)
		}
		if back.Entry == nil || back.Entry.Mode != EntryReplace || back.Entry.Dice != [2]int{4, 2} {
			t.Fatalf("entry = %+v", back.Entry)
		}
	})

	t.Run("a correction replaces in place and gives the cursor back", func(t *testing.T) {
		doc := base(t)
		at := len(doc.Actions) - 1
		corrected := runSteps(t, doc, []step{
			{"walk back", Gesture{Kind: GestureCursorBack}, nil},
			{"correct the roll", die(6), nil},
			{"second die", die(1), nil},
			{"pick a play for it", candidate(0), nil},
			{"validate", confirm(), nil},
		})
		if len(corrected.Actions) != len(doc.Actions) {
			t.Fatalf("a correction added an action: %d vs %d", len(corrected.Actions), len(doc.Actions))
		}
		if got := corrected.Actions[at].Dice; got != [2]int{6, 1} {
			t.Errorf("corrected dice = %v", got)
		}
		if corrected.Cursor != len(corrected.Actions) {
			t.Errorf("the cursor did not come back: %d", corrected.Cursor)
		}
	})

	t.Run("insert proposes the side that keeps the sequence coherent", func(t *testing.T) {
		doc := base(t)
		doc.Cursor = len(doc.Actions) - 1 // on player 1's second play
		ins, err := Apply(doc, Gesture{Kind: GestureInsertAfter})
		if err != nil {
			t.Fatal(err)
		}
		if ins.Entry == nil || ins.Entry.Side != domain.White {
			t.Fatalf("entry = %+v, want player 2 after player 1's play", ins.Entry)
		}
		if ins.Entry.At != len(doc.Actions) {
			t.Errorf("the entry lands at %d, want %d", ins.Entry.At, len(doc.Actions))
		}
	})

	t.Run("delete leaves the sides of the others alone", func(t *testing.T) {
		doc := base(t)
		doc.Cursor = 2 // player 2's play, between two plays of player 1
		after, err := Apply(doc, Gesture{Kind: GestureDelete})
		if err != nil {
			t.Fatal(err)
		}
		if len(after.Actions) != len(doc.Actions)-1 {
			t.Fatalf("actions = %d", len(after.Actions))
		}
		if after.Actions[2].Side != domain.Black {
			t.Error("deleting an Action moved another Action's side")
		}
		// Which is the whole point: the two plays of player 1 now follow each other,
		// and that ONE local double turn is what the user is shown — not a match
		// whose every side was flipped underneath them (ADR-0045 rule 4).
		if !hasInconsistency(Replay(after, 0).Actions[2], DoubleTurn) {
			t.Error("the deletion left no double turn where it must")
		}
	})

	t.Run("change side", func(t *testing.T) {
		doc := base(t)
		doc.Cursor = 1
		after, err := Apply(doc, Gesture{Kind: GestureFlipSide})
		if err != nil {
			t.Fatal(err)
		}
		if after.Actions[1].Side != domain.White {
			t.Fatalf("side = %d", after.Actions[1].Side)
		}
		if !Replay(after, 0).Inconsistent() {
			t.Error("a play given to the other camp must be marked, never corrected")
		}
	})

	t.Run("change the length restates the session's rules", func(t *testing.T) {
		doc := base(t)
		money, err := Apply(doc, Gesture{Kind: GestureSetLength, HasLength: true, MatchLength: 0})
		if err != nil {
			t.Fatal(err)
		}
		if !money.Header.Jacoby {
			t.Error("a money session is Jacoby by default")
		}
		ann := Replay(money, 0)
		if got := ann.Actions[1].Before.Score; got != [2]int{domain.Unlimited, domain.Unlimited} {
			t.Errorf("money score = %v", got)
		}
		if ann.Actions[1].Before.HasJacoby != 1 {
			t.Error("the session's rules are posted on every position")
		}
		back, err := Apply(money, Gesture{Kind: GestureSetLength, HasLength: true, MatchLength: 5})
		if err != nil {
			t.Fatal(err)
		}
		if back.Header.Jacoby || back.Header.Beaver {
			t.Error("a match carries neither Jacoby nor beaver")
		}
		if got := Replay(back, 0).Actions[1].Before.Score; got != [2]int{5, 5} {
			t.Errorf("away score = %v", got)
		}
	})

	t.Run("swap the players turns the board around", func(t *testing.T) {
		doc := base(t)
		doc.Header.Player1, doc.Header.Player2 = "Alice", "Bob"
		before := Replay(doc, 0)
		after, err := Apply(doc, Gesture{Kind: GestureSwapPlayers})
		if err != nil {
			t.Fatal(err)
		}
		if after.Header.Player1 != "Bob" || after.Header.Player2 != "Alice" {
			t.Errorf("names = %q/%q", after.Header.Player1, after.Header.Player2)
		}
		swapped := Replay(after, 0)
		if swapped.Inconsistent() {
			t.Fatal("swapping the players made the match illegal")
		}
		// Every board is the mirror of the one it was.
		for i := range before.Actions {
			if swapped.Actions[i].After != mirrorBoard(before.Actions[i].After) {
				t.Fatalf("action %d: the board was not turned around", i)
			}
			if swapped.Actions[i].Side != opponent(before.Actions[i].Side) {
				t.Fatalf("action %d: the side was not exchanged", i)
			}
		}
	})

	t.Run("undo and redo", func(t *testing.T) {
		e := NewEditor(base(t))
		before := len(e.Doc.Actions)
		if err := e.Apply(Gesture{Kind: GestureResign, HasSide: true, Side: domain.White, Level: 1}); err != nil {
			t.Fatal(err)
		}
		if len(e.Doc.Actions) != before+1 {
			t.Fatalf("actions = %d", len(e.Doc.Actions))
		}
		if !e.Undo() || len(e.Doc.Actions) != before {
			t.Fatalf("undo left %d actions", len(e.Doc.Actions))
		}
		if !e.Redo() || len(e.Doc.Actions) != before+1 {
			t.Fatalf("redo left %d actions", len(e.Doc.Actions))
		}
		for e.Undo() { //nolint:revive // drain the stack
		}
		if e.Undo() {
			t.Error("undo kept going on an empty stack")
		}
	})

	t.Run("a new draft inherits the length of the last one", func(t *testing.T) {
		doc := base(t)
		fresh, err := Apply(doc, Gesture{Kind: GestureCreate})
		if err != nil {
			t.Fatal(err)
		}
		if fresh.Header.MatchLength != 7 || len(fresh.Actions) != 0 {
			t.Errorf("draft = %+v", fresh.Header)
		}
		if Replay(fresh, 0).Next.Expects != KindOpening {
			t.Error("an empty draft expects an opening")
		}
	})
}

// hasInconsistency reports whether the Action carries that kind of Inconsistency.
func hasInconsistency(info ActionInfo, kind InconsistencyKind) bool {
	for _, in := range info.Inconsistencies {
		if in.Kind == kind {
			return true
		}
	}
	return false
}

// TestSwapPlayersIsTheSameMatchFromTheOtherSide is the recipe of T3.1's "inverser
// les joueurs": the gesture exchanges the names, gives every Action to the other
// camp and turns the board around — and the proof is a Replay, not a diff of the
// document. The same match must come back, read from the other side: the same
// games, the same points, won by the other player, with the score and the cube
// exchanged and not one Inconsistency more than before.
//
// The gesture is checked on a document that has actually derived something: a
// double taken (a cube with an owner, which the mirror has to move too) and a game
// finished by a resignation, whose winner is a fact of the Replay and not of any
// Action.
func TestSwapPlayersIsTheSameMatchFromTheOtherSide(t *testing.T) {
	doc := docOf(5, opening(domain.Black, 6, 3))
	doc.Actions = append(doc.Actions, firstCandidate(t, doc, domain.Black, 6, 3))
	doc.Actions = append(doc.Actions,
		Action{Side: domain.White, Kind: KindDouble},
		Action{Side: domain.Black, Kind: KindTake},
		// Player 2 gives the game up at the doubled cube: 2 points to player 1.
		Action{Side: domain.White, Kind: KindResign, Level: 1},
	)
	doc.Actions = append(doc.Actions, opening(domain.White, 2, 5))
	doc.Actions = append(doc.Actions, firstCandidate(t, doc, domain.White, 5, 2))
	doc.Header.Player1, doc.Header.Player2 = "Alice", "Bob"
	doc.Cursor = len(doc.Actions)

	before := Replay(doc, 0)
	if before.Score != [2]int{2, 0} || len(before.Games) != 2 {
		t.Fatalf("the fixture is not the match it means to be: score %v, %d games", before.Score, len(before.Games))
	}

	after, err := Apply(doc, Gesture{Kind: GestureSwapPlayers})
	if err != nil {
		t.Fatal(err)
	}
	if after.Header.Player1 != "Bob" || after.Header.Player2 != "Alice" {
		t.Fatalf("names = %q/%q", after.Header.Player1, after.Header.Player2)
	}
	if len(after.Actions) != len(doc.Actions) {
		t.Fatalf("the gesture changed the document: %d actions for %d", len(after.Actions), len(doc.Actions))
	}

	swapped := Replay(after, 0)
	if swapped.Inconsistent() && !before.Inconsistent() {
		t.Fatal("swapping the players made the match illegal")
	}
	if got, want := swapped.Score, [2]int{before.Score[1], before.Score[0]}; got != want {
		t.Errorf("score = %v, want %v", got, want)
	}
	if len(swapped.Games) != len(before.Games) {
		t.Fatalf("games = %d, want %d", len(swapped.Games), len(before.Games))
	}
	for i, g := range before.Games {
		got := swapped.Games[i]
		if got.PointsWon != g.PointsWon || got.Finished != g.Finished {
			t.Errorf("game %d: %d points (finished %v), want %d (%v)", i+1, got.PointsWon, got.Finished, g.PointsWon, g.Finished)
		}
		if g.Finished && got.Winner != opponent(g.Winner) {
			t.Errorf("game %d: won by player %d, want player %d", i+1, got.Winner+1, opponent(g.Winner)+1)
		}
		if got.InitialScore != [2]int{g.InitialScore[1], g.InitialScore[0]} {
			t.Errorf("game %d: initial score %v, want %v reversed", i+1, got.InitialScore, g.InitialScore)
		}
	}
	// The cube is one of the two players' as well: the taker of the double is now
	// the other camp, and the position the match goes on from says so.
	if bc, sc := before.Next.Position.Cube, swapped.Next.Position.Cube; bc.Value != sc.Value {
		t.Errorf("cube value = %d, want %d", sc.Value, bc.Value)
	}
	// And the side on roll is the other one, since the same player is on roll.
	if got, want := swapped.Next.Side, opponent(before.Next.Side); got != want {
		t.Errorf("side on roll = %d, want %d", got, want)
	}
	// Swapping twice is the identity: the document comes back as it was.
	back, err := Apply(after, Gesture{Kind: GestureSwapPlayers})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(back.Actions, doc.Actions) || back.Header != doc.Header {
		t.Error("swapping the players twice did not give the document back")
	}
}

// TestSetHeaderWritesTheMetadataAndNothingElse holds the contract of T3.1's
// metadata pane: the pane states the descriptive head of the document, and the
// three things it is NOT asked for survive it — the length, the session's rules,
// and the match id a first save posted. A form that sent a zero length would
// otherwise turn a 7-point match into a money session, and a form that dropped
// the match id would file a second Match at the next save.
func TestSetHeaderWritesTheMetadataAndNothingElse(t *testing.T) {
	doc := openedMatch(t, 7)
	matchID := int64(42)
	doc.Header.MatchID = &matchID
	doc.Header.MaxCube = 3
	doc.Header.Player1 = "Alice"

	tournament := int64(7)
	when := time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC)
	after, err := Apply(doc, Gesture{Kind: GestureSetHeader, Header: Header{
		Player1:      "Alice Martin",
		Player2:      "Bob",
		Event:        "Open de Paris",
		Location:     "Paris",
		Round:        "1/4",
		Date:         when,
		Transcriber:  "Kévin",
		TournamentID: &tournament,
	}})
	if err != nil {
		t.Fatal(err)
	}
	h := after.Header
	if h.Player1 != "Alice Martin" || h.Player2 != "Bob" || h.Event != "Open de Paris" ||
		h.Location != "Paris" || h.Round != "1/4" || !h.Date.Equal(when) || h.Transcriber != "Kévin" {
		t.Fatalf("the metadata was not written: %+v", h)
	}
	if h.TournamentID == nil || *h.TournamentID != tournament {
		t.Errorf("tournament = %v, want %d", h.TournamentID, tournament)
	}
	if h.MatchLength != 7 || h.Jacoby || h.Beaver || h.MaxCube != 3 {
		t.Errorf("the session's own fields were overwritten: %+v", h)
	}
	if h.MatchID == nil || *h.MatchID != matchID {
		t.Errorf("match id = %v, want %d: a form never posts one, and never takes one away", h.MatchID, matchID)
	}
	// And nothing of the match itself moved.
	if len(after.Actions) != len(doc.Actions) {
		t.Errorf("actions = %d, want %d", len(after.Actions), len(doc.Actions))
	}
}
