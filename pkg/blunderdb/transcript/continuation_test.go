package transcript

import "testing"

// TestDeleteRemovesTheDecisionBeingEdited holds ADR-0050's first decision: Del
// removes the decision the user is editing — written or not — and puts the
// Cursor, and the editing with it, on the one before.
func TestDeleteRemovesTheDecisionBeingEdited(t *testing.T) {
	t.Run("a written action goes, the previous one is loaded", func(t *testing.T) {
		e := NewEditor(seek(t, typedMatch(t, 7), 3))
		before := len(e.Doc.Actions)
		if err := e.Apply(Gesture{Kind: GestureDelete}); err != nil {
			t.Fatal(err)
		}
		if len(e.Doc.Actions) != before-1 {
			t.Fatalf("actions = %d, want %d", len(e.Doc.Actions), before-1)
		}
		// The deletion leaves a double turn at the slot it emptied; the Cursor
		// must NOT be pulled onto it — the user asked to step back.
		ann := e.Replay(e.From())
		if !hasInconsistency(ann.Actions[3], DoubleTurn) {
			t.Fatal("the fixture's deletion made no double turn; the test proves nothing")
		}
		if ann.Cursor != 2 {
			t.Errorf("cursor = %d, want 2 — the Replay pulled it off the previous decision", ann.Cursor)
		}
	})

	t.Run("the first action leaves the cursor on what follows", func(t *testing.T) {
		doc := seek(t, typedMatch(t, 7), 0)
		after, err := Apply(doc, Gesture{Kind: GestureDelete})
		if err != nil {
			t.Fatal(err)
		}
		if after.Cursor != 0 || after.Entry == nil || after.Entry.At != 0 {
			t.Errorf("cursor = %d, entry = %+v; want the new first action", after.Cursor, after.Entry)
		}
	})

	t.Run("at the end, the empty slot is abandoned and the last action loaded", func(t *testing.T) {
		doc := typedMatch(t, 7)
		n := len(doc.Actions)
		after, err := Apply(doc, Gesture{Kind: GestureDelete})
		if err != nil {
			t.Fatal(err)
		}
		if len(after.Actions) != n {
			t.Fatalf("actions = %d, want %d — nothing written is deleted from the empty slot", len(after.Actions), n)
		}
		if after.Cursor != n-1 || after.Entry == nil || after.Entry.Mode != EntryReplace {
			t.Fatalf("cursor = %d, entry = %+v; want the last action loaded", after.Cursor, after.Entry)
		}
		// Pressed again, Del deletes that one.
		again, err := Apply(after, Gesture{Kind: GestureDelete})
		if err != nil {
			t.Fatal(err)
		}
		if len(again.Actions) != n-1 || again.Cursor != n-2 {
			t.Errorf("second Del: %d actions, cursor %d; want %d, %d", len(again.Actions), again.Cursor, n-1, n-2)
		}
	})

	t.Run("a roll half typed at the end is abandoned", func(t *testing.T) {
		doc := runSteps(t, typedMatch(t, 7), []step{{"a die", die(3), nil}})
		n := len(doc.Actions)
		after, err := Apply(doc, Gesture{Kind: GestureDelete})
		if err != nil {
			t.Fatal(err)
		}
		if len(after.Actions) != n || after.Cursor != n-1 {
			t.Errorf("%d actions, cursor %d; want %d, %d", len(after.Actions), after.Cursor, n, n-1)
		}
	})

	t.Run("an insertion slot is abandoned, the action after it is kept", func(t *testing.T) {
		doc := seek(t, typedMatch(t, 7), 3)
		ins, err := Apply(doc, Gesture{Kind: GestureInsertBefore})
		if err != nil {
			t.Fatal(err)
		}
		after, err := Apply(ins, Gesture{Kind: GestureDelete})
		if err != nil {
			t.Fatal(err)
		}
		if len(after.Actions) != len(doc.Actions) {
			t.Fatalf("actions = %d, want %d", len(after.Actions), len(doc.Actions))
		}
		if after.Cursor != 2 {
			t.Errorf("cursor = %d, want 2", after.Cursor)
		}
	})

	t.Run("an empty document has nothing to delete", func(t *testing.T) {
		if _, err := Apply(New(7), Gesture{Kind: GestureDelete}); err == nil {
			t.Error("Del on an empty draft succeeded")
		}
	})
}

// TestTakeCorrectedContinuesTheGame holds ADR-0050's second decision: a pass
// corrected into a take reopens its game, and the user types the rest of it
// right there, until it ends — not over the next game's opening, not at the
// place the correction was started from.
func TestTakeCorrectedContinuesTheGame(t *testing.T) {
	doc := doubledMatch(t)
	pass := len(doc.Actions) - 1
	doubler := doc.Actions[pass-1].Side
	// A second game follows in the record: its opening and a first play.
	doc = runSteps(t, doc, []step{
		{"opening die 1", die(5), nil},
		{"opening die 2", die(2), nil},
		{"validate the opening", confirm(), nil},
		{"a play", candidate(0), nil},
		{"validate", confirm(), nil},
	})
	nextOpening := pass + 1
	if doc.Actions[nextOpening].Kind != KindOpening {
		t.Fatalf("fixture: action %d is %v, want the second game's opening", nextOpening, doc.Actions[nextOpening].Kind)
	}
	total := len(doc.Actions)

	e := NewEditor(seek(t, doc, pass))
	if err := e.Apply(Gesture{Kind: GestureTake}); err != nil {
		t.Fatal(err)
	}
	if e.Doc.Actions[pass].Kind != KindTake {
		t.Fatalf("action %d = %v, want the take", pass, e.Doc.Actions[pass].Kind)
	}
	en := e.Doc.Entry
	if e.Doc.Cursor != pass+1 || en == nil || en.Mode != EntryNew || en.At != pass+1 {
		t.Fatalf("cursor = %d, entry = %+v; want an insertion right after the take", e.Doc.Cursor, en)
	}
	if en.Side != doubler {
		t.Errorf("slot side = %d, want the doubler %d, who rolls after a take", en.Side, doubler)
	}
	if ann := e.Replay(e.From()); ann.Cursor != pass+1 {
		t.Errorf("the Replay moved the Cursor to %d", ann.Cursor)
	}

	// The rest of the game is typed as usual: a roll, and it is INSERTED.
	for _, g := range []Gesture{die(3), die(1), candidate(0), confirm()} {
		if err := e.Apply(g); err != nil {
			t.Fatalf("%s: %v", g.Kind, err)
		}
	}
	if len(e.Doc.Actions) != total+1 || e.Doc.Actions[pass+1].Kind != KindChecker {
		t.Fatalf("the roll was not inserted after the take: %d actions", len(e.Doc.Actions))
	}
	if e.Doc.Actions[pass+2].Kind != KindOpening {
		t.Fatal("the next game's opening was overwritten")
	}
	if en := e.Doc.Entry; en == nil || en.Mode != EntryNew || en.At != pass+2 {
		t.Fatalf("entry = %+v; the game is still running, the next slot must be open", en)
	}

	// The game ends on a double passed: the slot closes, and the Cursor rests on
	// the next game's opening.
	for _, g := range []Gesture{{Kind: GestureDouble}, {Kind: GesturePass}} {
		if err := e.Apply(g); err != nil {
			t.Fatalf("%s: %v", g.Kind, err)
		}
	}
	end := pass + 3
	if e.Doc.Actions[end].Kind != KindPass {
		t.Fatalf("action %d = %v, want the pass", end, e.Doc.Actions[end].Kind)
	}
	if e.Doc.Cursor != end+1 || e.Doc.Actions[e.Doc.Cursor].Kind != KindOpening {
		t.Errorf("cursor = %d, want %d — the next game's opening", e.Doc.Cursor, end+1)
	}
	if en := e.Doc.Entry; en != nil && en.Mode == EntryNew {
		t.Errorf("an insertion slot is still open past the end of the game: %+v", en)
	}
	if ann := Replay(e.Doc, 0); ann.Games[len(ann.Games)-2].Winner < 0 {
		t.Error("the continued game did not end")
	}
}

// TestCorrectionThatStillEndsTheGameReturns holds the other side of the rule: a
// correction that leaves its game closed goes back where the user came from, as
// before (ADR-0049).
func TestCorrectionThatStillEndsTheGameReturns(t *testing.T) {
	doc := doubledMatch(t)
	pass := len(doc.Actions) - 1
	doc = runSteps(t, doc, []step{
		{"opening die 1", die(5), nil},
		{"opening die 2", die(2), nil},
		{"validate the opening", confirm(), nil},
	})
	after, err := Apply(seek(t, doc, pass), Gesture{Kind: GesturePass})
	if err != nil {
		t.Fatal(err)
	}
	if after.Cursor != len(after.Actions) || after.HoldCursor {
		t.Errorf("cursor = %d (hold %v), want the end of the document", after.Cursor, after.HoldCursor)
	}
}

// TestLastActionValidatedAppendsNext holds ADR-0051: the LAST Action, walked back
// to and validated again — re-edited or not — sends the Cursor to the end of the
// document, whatever Return says, and holds it there, so the next roll is appended
// as it was the first time.
func TestLastActionValidatedAppendsNext(t *testing.T) {
	t.Run("walked back and forward again, Enter goes to the end", func(t *testing.T) {
		doc := typedMatch(t, 7)
		n := len(doc.Actions)
		doc = seek(t, doc, 1)
		for doc.Cursor < n-1 {
			next, err := Apply(doc, Gesture{Kind: GestureCursorForward})
			if err != nil {
				t.Fatal(err)
			}
			doc = next
		}
		// A stale place to come back to, behind the last Action — the jump
		// to an Inconsistency leaves one.
		doc.Return, doc.HasReturn = 1, true
		after, err := Apply(doc, confirm())
		if err != nil {
			t.Fatal(err)
		}
		if after.Cursor != n || after.Entry != nil {
			t.Fatalf("cursor = %d, entry = %+v; want the end of the document, nothing typed", after.Cursor, after.Entry)
		}
		if !after.HoldCursor {
			t.Error("the end is not held: a mark on the last Action would pull the Cursor back")
		}
		// And the next roll is a NEW Action.
		next := runSteps(t, after, []step{{"a roll", die(3), nil}, {"its second die", die(1), nil}, {"a play", candidate(0), nil}, {"validate", confirm(), nil}})
		if len(next.Actions) != n+1 {
			t.Errorf("actions = %d, want %d — the roll was not appended", len(next.Actions), n+1)
		}
	})

	t.Run("a mark on the last Action does not pull the Cursor back", func(t *testing.T) {
		e := NewEditor(typedMatch(t, 7))
		n := len(e.Doc.Actions)
		e.SeekCursor(n - 1)
		// Given to the wrong camp, the last play makes a double turn.
		if err := e.Apply(Gesture{Kind: GestureFlipSide}); err != nil {
			t.Fatal(err)
		}
		e.SeekCursor(n - 1)
		for _, g := range []Gesture{die(3), die(1), candidate(0), confirm()} {
			if err := e.Apply(g); err != nil {
				t.Fatalf("%s: %v", g.Kind, err)
			}
		}
		ann := e.Replay(e.From())
		if !ann.Inconsistent() {
			t.Fatal("the fixture lost its double turn; the test proves nothing")
		}
		if ann.Cursor != n {
			t.Errorf("cursor = %d, want %d — the end of the document", ann.Cursor, n)
		}
	})
}
