package transcript

import (
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// TestCorrectionInPlace holds the half of fonctionnel.md §2 the Transcript exists for:
// walking back to an Action and typing again replaces it, without adding anything and
// without moving the ones around it.
func TestCorrectionInPlace(t *testing.T) {
	base := func(t *testing.T) Document {
		t.Helper()
		return typedMatch(t, 4)
	}

	t.Run("a roll retyped keeps a play it still allows", func(t *testing.T) {
		doc := base(t)
		at := 1 // player 1's first checker play, made with 6-3
		doc = seek(t, doc, at)
		played := doc.Actions[at].Steps
		// The dice are retyped the other way round — the commonest reading
		// mistake there is, and the one where "si les steps restent un coup légal
		// du nouveau jet, ils sont gardés" (fonctionnel.md §2) has to hold: the
		// user corrected the DICE, and must not be made to pick the play again.
		doc = runSteps(t, doc, []step{
			{"first die", die(3), nil},
			{"second die", die(6), nil},
		})
		if doc.Entry == nil || !doc.Entry.Selected {
			t.Fatalf("entry = %+v, want the recorded play kept and selected", doc.Entry)
		}
		if doc.Entry.Review {
			t.Error("a play the new roll allows must not be marked for review")
		}
		if !sameSteps(doc.Entry.Steps, played) {
			t.Errorf("steps = %v, want the recorded %v", doc.Entry.Steps, played)
		}
	})

	t.Run("a roll the play does not belong to preselects a candidate, marked for review", func(t *testing.T) {
		doc := base(t)
		at := 1
		doc = seek(t, doc, at)
		doc = runSteps(t, doc, []step{
			{"first die", die(2), nil},
			{"second die", die(1), nil},
		})
		if doc.Entry == nil || !doc.Entry.Selected {
			t.Fatalf("entry = %+v, want a candidate of the new roll", doc.Entry)
		}
		if !doc.Entry.Review {
			t.Error("a play the new roll cannot make must be marked for review")
		}
		if !diceCoherent(doc.Entry.Steps, [2]int{2, 1}, doc.Entry.Side) {
			t.Errorf("the preselected play %v is not a play of 21", doc.Entry.Steps)
		}
		// Nothing is written until validation: the Action still says what it said.
		if doc.Actions[at].Dice != [2]int{6, 3} {
			t.Error("a correction wrote before it was validated")
		}
		// And picking through the list does not clear the mark — validating does.
		picked := runSteps(t, doc, []step{{"pick", candidate(0), nil}})
		if !picked.Entry.Review {
			t.Error("the review mark must survive a candidate pick (fonctionnel.md §2)")
		}
		done := runSteps(t, picked, []step{{"validate", confirm(), nil}})
		if done.Entry != nil {
			t.Error("validation left an entry behind")
		}
	})

	t.Run("the cursor comes back where it was, and the document keeps its length", func(t *testing.T) {
		doc := base(t)
		end := len(doc.Actions)
		corrected := runSteps(t, doc, []step{
			{"back", Gesture{Kind: GestureCursorBack}, nil},
			{"back again", Gesture{Kind: GestureCursorBack}, nil},
			{"a new roll", die(5), nil},
			{"its second die", die(2), nil},
			{"a play for it", candidate(0), nil},
			{"validate", confirm(), nil},
		})
		if len(corrected.Actions) != end {
			t.Fatalf("actions = %d, want %d — a correction is not an insertion", len(corrected.Actions), end)
		}
		if corrected.Cursor != end {
			t.Errorf("cursor = %d, want %d (fonctionnel.md §2: it comes back)", corrected.Cursor, end)
		}
		if got := corrected.Actions[end-2].Dice; got != [2]int{5, 2} {
			t.Errorf("the corrected action carries %v", got)
		}
	})

	t.Run("walking away from a change records it, walking to read does not", func(t *testing.T) {
		doc := base(t)
		at := len(doc.Actions) - 2
		doc = seek(t, doc, at)

		// Reading: the Cursor crosses Actions and writes nothing.
		read := runSteps(t, doc, []step{
			{"forward", Gesture{Kind: GestureCursorForward}, nil},
			{"back", Gesture{Kind: GestureCursorBack}, nil},
		})
		for i := range read.Actions {
			if !sameAction(read.Actions[i], doc.Actions[i]) {
				t.Fatalf("walking the transcript rewrote action %d", i)
			}
		}
		if read.HasTouched {
			t.Error("a walk reported an action touched, so every cell would be rewritten on disk")
		}

		// ux.md §4.3's "erreur vue k tours plus tard" spends its last k
		// keystrokes walking forward: the pick has to survive them.
		picked := runSteps(t, doc, []step{{"another play", candidate(1), nil}})
		want := append([]domain.CheckerStep(nil), picked.Entry.Steps...)
		on := runSteps(t, picked, []step{{"forward", Gesture{Kind: GestureCursorForward}, nil}})
		if !sameSteps(on.Actions[at].Steps, want) {
			t.Errorf("the correction was lost by walking on: %v, want %v", on.Actions[at].Steps, want)
		}
		if on.Cursor != at+1 {
			t.Errorf("cursor = %d, want %d — walking on is a step, not a validation's return", on.Cursor, at+1)
		}
	})
}

// TestCorrectionCursorLandsOnTheFirstInconsistency holds the last sentence of
// fonctionnel.md §1.4, through the Editor the session runs.
func TestCorrectionCursorLandsOnTheFirstInconsistency(t *testing.T) {
	doc := typedMatch(t, 4)
	e := NewEditor(doc)
	if e.Replay(e.From()).Inconsistent() {
		t.Fatal("the fixture is already inconsistent")
	}

	// A camp given to the other side, two Actions before the end: the double
	// turn it makes is BEHIND the Cursor, which sits at the end of the document.
	at := len(doc.Actions) - 2
	e.SeekCursor(at)
	if err := e.Apply(Gesture{Kind: GestureFlipSide}); err != nil {
		t.Fatal(err)
	}
	ann := e.Replay(e.From())
	if !ann.Inconsistent() {
		t.Fatal("changing a camp made no inconsistency at all")
	}
	first := -1
	for i := range ann.Actions {
		if len(ann.Actions[i].Inconsistencies) > 0 {
			first = i
			break
		}
	}
	if ann.Cursor != first {
		t.Errorf("cursor = %d, want the first inconsistency at %d", ann.Cursor, first)
	}
	// And nothing was deleted or repaired on the way (ADR-0044).
	if len(ann.Document.Actions) != len(doc.Actions) {
		t.Errorf("actions = %d, want %d", len(ann.Document.Actions), len(doc.Actions))
	}
}

// TestAppendingDoesNotPullTheCursorBack holds the other half of the Cursor jump:
// a plain append names no touched Action, so an Action the rules mark on the way
// in — a play past the end of a won match, flux 11 — does NOT drag the Cursor
// back onto itself and make the next roll correct it.
func TestAppendingDoesNotPullTheCursorBack(t *testing.T) {
	// The play appended is given to the camp that just played, which makes a
	// double turn ON THE APPENDED ACTION — the shape that would pull the Cursor
	// back if an append named itself touched.
	e := NewEditor(typedMatch(t, 7))
	before := len(e.Doc.Actions)
	last := e.Doc.Actions[before-1].Side
	steps := []Gesture{
		{Kind: GestureInsertAfter, HasSide: true, Side: last},
		die(3), die(1), candidate(0), confirm(),
	}
	for _, g := range steps {
		if err := e.Apply(g); err != nil {
			t.Fatalf("%s: %v", g.Kind, err)
		}
	}
	if !hasInconsistency(e.Replay(0).Actions[before], DoubleTurn) {
		t.Fatal("the appended action carries no double turn; the case is not built")
	}
	if e.Doc.HasTouched {
		t.Error("an append reported an action touched; the Cursor would be pulled onto it")
	}
	ann := e.Replay(e.From())
	if ann.Cursor != len(ann.Document.Actions) {
		t.Errorf("cursor = %d, want %d — an append leaves it after the document", ann.Cursor, len(ann.Document.Actions))
	}
	if len(ann.Document.Actions) != before+1 {
		t.Errorf("actions = %d, want %d", len(ann.Document.Actions), before+1)
	}
}

// TestInsertionAndDeletionAreMarkedNotRefused walks flux 13, 14 and 15 of
// fonctionnel.md §6.
func TestInsertionAndDeletionAreMarkedNotRefused(t *testing.T) {
	t.Run("an insertion of the neighbour's own camp makes a double turn", func(t *testing.T) {
		doc := typedMatch(t, 4)
		at := 2
		doc = seek(t, doc, at)
		ins, err := Apply(doc, Gesture{Kind: GestureInsertBefore, HasSide: true, Side: doc.Actions[at-1].Side})
		if err != nil {
			t.Fatal(err)
		}
		if ins.Entry == nil || ins.Entry.At != at || ins.Entry.Mode != EntryNew {
			t.Fatalf("entry = %+v", ins.Entry)
		}
		done := runSteps(t, ins, []step{
			{"a roll", die(3), nil},
			{"its second die", die(1), nil},
			{"a play", candidate(0), nil},
			{"validate", confirm(), nil},
		})
		if len(done.Actions) != len(doc.Actions)+1 {
			t.Fatalf("actions = %d, want one more", len(done.Actions))
		}
		if !hasInconsistency(Replay(done, 0).Actions[at], DoubleTurn) {
			t.Error("an action inserted beside its own camp must be marked as a double turn")
		}
	})

	t.Run("insert after works at the end of the document", func(t *testing.T) {
		doc := typedMatch(t, 4)
		// The Cursor sits past the last Action, where the user spends most of
		// their time: `a` must be an insertion there, not an error.
		ins, err := Apply(doc, Gesture{Kind: GestureInsertAfter})
		if err != nil {
			t.Fatalf("insert after at the end: %v", err)
		}
		if ins.Entry == nil || ins.Entry.At != len(doc.Actions) {
			t.Fatalf("entry = %+v", ins.Entry)
		}
	})

	t.Run("a deletion leaves the cursor on the action that follows", func(t *testing.T) {
		doc := typedMatch(t, 4)
		at := 2
		doc = seek(t, doc, at)
		after, err := Apply(doc, Gesture{Kind: GestureDelete})
		if err != nil {
			t.Fatal(err)
		}
		if after.Cursor != at {
			t.Errorf("cursor = %d, want %d — the action that follows (fonctionnel.md §2)", after.Cursor, at)
		}
		if !after.HasTouched || after.Touched != at {
			t.Errorf("touched = %d/%v, want %d", after.Touched, after.HasTouched, at)
		}
	})
}

// TestUndoRedoWalksTheSessionStack holds the "annuler / rétablir" row of
// fonctionnel.md §2 and the purity of Apply beside it.
func TestUndoRedoWalksTheSessionStack(t *testing.T) {
	e := NewEditor(typedMatch(t, 4))
	if e.CanUndo() {
		t.Error("a fresh session has nothing to undo")
	}
	before := len(e.Doc.Actions)

	e.SeekCursor(1)
	if err := e.Apply(Gesture{Kind: GestureDelete}); err != nil {
		t.Fatal(err)
	}
	if len(e.Doc.Actions) != before-1 || !e.CanUndo() || e.CanRedo() {
		t.Fatalf("after a deletion: %d actions, undo=%v redo=%v", len(e.Doc.Actions), e.CanUndo(), e.CanRedo())
	}
	if !e.Undo() || len(e.Doc.Actions) != before {
		t.Fatalf("undo left %d actions", len(e.Doc.Actions))
	}
	if !e.CanRedo() {
		t.Error("an undone gesture must be redoable")
	}
	if !e.Redo() || len(e.Doc.Actions) != before-1 {
		t.Fatalf("redo left %d actions", len(e.Doc.Actions))
	}

	// Undo and redo are NOT gestures of the document: Apply refuses them rather
	// than pretending to have a stack it cannot have.
	for _, k := range []GestureKind{GestureUndo, GestureRedo} {
		if _, err := Apply(e.Doc, Gesture{Kind: k}); !errors.Is(err, ErrNotPure) {
			t.Errorf("Apply(%s) = %v, want ErrNotPure", k, err)
		}
	}
}

// ── helpers ──────────────────────────────────────────────────────────────

// typedMatch is a draft with an opening and four checker plays typed into it, which is
// the shortest document a correction can be walked back into.
func typedMatch(t *testing.T, length int) Document {
	t.Helper()
	doc := openedMatch(t, length)
	rolls := [][2]int{{0, 0}, {5, 4}, {4, 2}, {6, 5}}
	steps := []step{}
	for i, r := range rolls {
		if i > 0 {
			steps = append(steps,
				step{"a roll", die(r[0]), nil},
				step{"its second die", die(r[1]), nil})
		}
		steps = append(steps,
			step{"a play", candidate(0), nil},
			step{"validate", confirm(), nil})
	}
	return runSteps(t, doc, steps)
}

// seek walks the Cursor back to at, one gesture at a time, exactly as `h` does.
func seek(t *testing.T, doc Document, at int) Document {
	t.Helper()
	for doc.Cursor > at {
		next, err := Apply(doc, Gesture{Kind: GestureCursorBack})
		if err != nil {
			t.Fatal(err)
		}
		doc = next
	}
	return doc
}

func sameSteps(a, b []domain.CheckerStep) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
