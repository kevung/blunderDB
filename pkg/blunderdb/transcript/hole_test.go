package transcript

import "testing"

// session applies a gesture the way the database session does
// (database/db_transcription.go's stateOf): the Replay after it starts at
// [Editor.From], and the Cursor is moved onto whatever Inconsistency it lands on.
// A test that skipped this step would never see a Cursor pulled away.
func session(t *testing.T, e *Editor, g Gesture) Annotated {
	t.Helper()
	if err := e.Apply(g); err != nil {
		t.Fatalf("%s: %v", g.Kind, err)
	}
	ann := e.Replay(e.From())
	if ann.Cursor != e.Doc.Cursor {
		e.SeekCursor(ann.Cursor)
		ann = e.Replay(e.Doc.Cursor)
	}
	return ann
}

// holed is typedMatch with its fourth Action deleted: player 1 plays at 2 and
// again at 3, and the turn player 2 had between them is a hole in front of 3.
// The Cursor is where the deletion left it, on 2.
func holed(t *testing.T) (*Editor, Action) {
	t.Helper()
	e := NewEditor(seek(t, typedMatch(t, 7), 3))
	gone := e.Doc.Actions[3]
	ann := session(t, e, Gesture{Kind: GestureDelete})
	if !hasInconsistency(ann.Actions[3], DoubleTurn) || e.Doc.Cursor != 2 {
		t.Fatalf("the fixture's deletion left cursor %d and no double turn at 3; the test proves nothing", e.Doc.Cursor)
	}
	return e, gone
}

// TestAWriteLessGestureKeepsTheCursor holds ADR-0054's first decision: the jump
// to the first Inconsistency answers a Replay, and a gesture that wrote nothing
// replayed nothing.
func TestAWriteLessGestureKeepsTheCursor(t *testing.T) {
	t.Run("a die typed on the decision before a double turn stays there", func(t *testing.T) {
		e, _ := holed(t)
		session(t, e, die(3))
		if e.Doc.Cursor != 2 || e.Doc.Entry == nil || e.Doc.Entry.At != 2 || e.Doc.Entry.Dice[0] != 3 {
			t.Errorf("cursor %d, entry %+v; want the die on the decision at 2", e.Doc.Cursor, e.Doc.Entry)
		}
	})

	t.Run("stepping back past a double turn is not pulled onto it", func(t *testing.T) {
		e, _ := holed(t)
		session(t, e, Gesture{Kind: GestureCursorForward}) // the hole
		session(t, e, Gesture{Kind: GestureCursorForward}) // the double turn
		for _, want := range []int{3, 2, 1} {
			session(t, e, Gesture{Kind: GestureCursorBack})
			if e.Doc.Cursor != want {
				t.Fatalf("cursor = %d, want %d — the double turn is a wall", e.Doc.Cursor, want)
			}
		}
	})

	t.Run("a write still jumps to the first inconsistency", func(t *testing.T) {
		doc := seek(t, typedMatch(t, 7), 2)
		e := NewEditor(doc)
		session(t, e, Gesture{Kind: GestureFlipSide})
		if !e.Doc.HasTouched {
			t.Fatal("flipping a side wrote nothing")
		}
		ann := e.Replay(e.From())
		if ann.Cursor != 2 {
			t.Errorf("cursor = %d, want 2 — the flipped action is the first double turn", ann.Cursor)
		}
	})
}

// TestTheHoleOfADoubleTurnIsAStop holds ADR-0054's second decision: the turn a
// double turn is missing is a cell the Cursor stops on, open for insertion on the
// side whose turn it was — which is how a deleted decision is typed again.
func TestTheHoleOfADoubleTurnIsAStop(t *testing.T) {
	t.Run("forward stops on the hole, then on the action after it", func(t *testing.T) {
		e, _ := holed(t)
		session(t, e, Gesture{Kind: GestureCursorForward})
		en := e.Doc.Entry
		if e.Doc.Cursor != 3 || en == nil || en.Mode != EntryNew || en.At != 3 || en.Side != opponent(e.Doc.Actions[3].Side) {
			t.Fatalf("cursor %d, entry %+v; want an insertion at 3 for the other side", e.Doc.Cursor, en)
		}
		session(t, e, Gesture{Kind: GestureCursorForward})
		if en := e.Doc.Entry; e.Doc.Cursor != 3 || en == nil || en.Mode != EntryReplace || en.At != 3 {
			t.Errorf("cursor %d, entry %+v; want the action at 3 loaded", e.Doc.Cursor, en)
		}
	})

	t.Run("back stops on the hole, then on the action before it", func(t *testing.T) {
		e, _ := holed(t)
		session(t, e, Gesture{Kind: GestureCursorForward})
		session(t, e, Gesture{Kind: GestureCursorForward})
		session(t, e, Gesture{Kind: GestureCursorBack})
		if en := e.Doc.Entry; e.Doc.Cursor != 3 || en == nil || en.Mode != EntryNew {
			t.Fatalf("cursor %d, entry %+v; want the hole", e.Doc.Cursor, en)
		}
		session(t, e, Gesture{Kind: GestureCursorBack})
		if en := e.Doc.Entry; e.Doc.Cursor != 2 || en == nil || en.Mode != EntryReplace || en.At != 2 {
			t.Errorf("cursor %d, entry %+v; want the action at 2 loaded", e.Doc.Cursor, en)
		}
	})

	t.Run("the deleted decision is typed back into the hole", func(t *testing.T) {
		e, gone := holed(t)
		n := len(e.Doc.Actions)
		session(t, e, Gesture{Kind: GestureCursorForward})
		session(t, e, die(gone.Dice[0]))
		session(t, e, die(gone.Dice[1]))
		session(t, e, candidate(0))
		ann := session(t, e, confirm())
		if len(e.Doc.Actions) != n+1 || e.Doc.Actions[3].Side != gone.Side || !sameSteps(e.Doc.Actions[3].Steps, gone.Steps) {
			t.Fatalf("actions %d, action 3 = %+v; want %+v back in its place", len(e.Doc.Actions), e.Doc.Actions[3], gone)
		}
		for i, a := range ann.Actions {
			if len(a.Inconsistencies) > 0 {
				t.Errorf("action %d still flagged: %v", i, a.Inconsistencies)
			}
		}
	})

	t.Run("a coherent sequence has no hole to stop on", func(t *testing.T) {
		doc := seek(t, typedMatch(t, 7), 1)
		for at := 2; at < len(doc.Actions); at++ {
			next, err := Apply(doc, Gesture{Kind: GestureCursorForward})
			if err != nil {
				t.Fatal(err)
			}
			doc = next
			if doc.Cursor != at || doc.Entry == nil || doc.Entry.Mode != EntryReplace {
				t.Fatalf("cursor %d, entry %+v; want the action at %d", doc.Cursor, doc.Entry, at)
			}
		}
	})
}
