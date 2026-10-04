package transcript

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// TestFirstPlayRetypedChangesCamp: the dice of a game's first play are the opening
// roll, so typing ANOTHER roll there decides once more who starts; the same two dice,
// in either order, change nothing. Nothing but the first play moves.
func TestFirstPlayRetypedChangesCamp(t *testing.T) {
	retype := func(t *testing.T, doc Document, d1, d2 int) Document {
		t.Helper()
		return runSteps(t, seek(t, doc, 0), []step{
			{"first die", die(d1), nil},
			{"second die", die(d2), nil},
			{"validate", confirm(), nil},
		})
	}

	t.Run("another roll changes camp with the winner", func(t *testing.T) {
		doc := runSteps(t, openedMatch(t, 7), []step{
			{"a play", candidate(0), nil},
			{"validate", confirm(), nil},
		})
		played := doc.Actions[0]
		if same := retype(t, doc, 3, 6).Actions[0]; same.Side != domain.Black || same.Dice != [2]int{6, 3} ||
			!sameSteps(same.Steps, played.Steps) {
			t.Errorf("the same roll retyped = %+v, want the play untouched", same)
		}
		if a := retype(t, doc, 2, 5).Actions[0]; a.Side != domain.White || a.Dice != [2]int{2, 5} ||
			!diceCoherent(domain.Board{}, a.Steps, a.Dice, a.Side) {
			t.Errorf("first play = %+v, want a play of player 2's 52", a)
		}
	})

	t.Run("the rest of the game keeps its sides", func(t *testing.T) {
		doc := typedMatch(t, 7)
		sides := make([]int, len(doc.Actions))
		for i, a := range doc.Actions {
			sides[i] = a.Side
		}

		doc = retype(t, doc, 2, 5)

		for i := 1; i < len(doc.Actions); i++ {
			if doc.Actions[i].Side != sides[i] {
				t.Fatalf("action %d changed camp; only the first play follows the opening roll", i)
			}
		}
	})
}

// TestCubeGestureWritesAtTheCursorsSlot: the cube gestures act WHERE THE CURSOR IS
// (fonctionnel.md §2) — `t` on a walked-back pass replaces it rather than inserting
// a take in front of it.
func TestCubeGestureWritesAtTheCursorsSlot(t *testing.T) {
	t.Run("a take replaces the pass under the cursor", func(t *testing.T) {
		doc := doubledMatch(t)
		at := len(doc.Actions) - 1 // the pass
		if doc.Actions[at].Kind != KindPass {
			t.Fatalf("the fixture ends on %s, not a pass", doc.Actions[at].Kind)
		}
		passer := doc.Actions[at].Side
		count := len(doc.Actions)

		doc = seek(t, doc, at)
		out, err := Apply(doc, Gesture{Kind: GestureTake})
		if err != nil {
			t.Fatal(err)
		}

		if len(out.Actions) != count {
			t.Fatalf("actions = %d, want %d — a correction replaces, it does not add", len(out.Actions), count)
		}
		if got := out.Actions[at]; got.Kind != KindTake || got.Side != passer {
			t.Fatalf("action %d = %+v, want a take by the camp that had answered", at, got)
		}
		// And the game goes on: the pass ended it, the take does not.
		ann := Replay(out, 0)
		if ann.Games[len(ann.Games)-1].Finished {
			t.Error("the game is still recorded as finished, so the take was not replayed")
		}
	})

	t.Run("a double fills the slot an insertion opened", func(t *testing.T) {
		doc := typedMatch(t, 7)
		at := 2
		doc = seek(t, doc, at)
		count := len(doc.Actions)

		ins, err := Apply(doc, Gesture{Kind: GestureInsertBefore})
		if err != nil {
			t.Fatal(err)
		}
		out, err := Apply(ins, Gesture{Kind: GestureDouble})
		if err != nil {
			t.Fatal(err)
		}

		if len(out.Actions) != count+1 {
			t.Fatalf("actions = %d, want one more", len(out.Actions))
		}
		if got := out.Actions[at]; got.Kind != KindDouble {
			t.Fatalf("action %d = %+v, want the double in the slot the insertion opened", at, got)
		}
	})

	t.Run("at the end of the document it still appends", func(t *testing.T) {
		doc := typedMatch(t, 7)
		count := len(doc.Actions)
		out, err := Apply(doc, Gesture{Kind: GestureDouble})
		if err != nil {
			t.Fatal(err)
		}
		if len(out.Actions) != count+1 || out.Actions[count].Kind != KindDouble {
			t.Fatalf("actions = %d, last = %+v", len(out.Actions), lastAction(t, out))
		}
	})

	t.Run("a play left selected is validated first, and the cube action follows it", func(t *testing.T) {
		// The budget of ux.md §4.2: `d` alone doubles after a play has been
		// picked. The play must be written where it was being typed, and the
		// double right after it — not at the other end of the document.
		doc := typedMatch(t, 7)
		at := 1
		doc = seek(t, doc, at)
		count := len(doc.Actions)
		doc = runSteps(t, doc, []step{
			{"a roll", die(4), nil},
			{"its second die", die(2), nil},
			{"a play", candidate(0), nil},
		})
		out, err := Apply(doc, Gesture{Kind: GestureDouble})
		if err != nil {
			t.Fatal(err)
		}
		if len(out.Actions) != count+1 {
			t.Fatalf("actions = %d, want one more (the play replaced, the double inserted)", len(out.Actions))
		}
		if out.Actions[at].Dice != [2]int{4, 2} {
			t.Errorf("action %d = %+v, want the roll that was being typed", at, out.Actions[at])
		}
		if out.Actions[at+1].Kind != KindDouble {
			t.Errorf("action %d = %+v, want the double just after the play", at+1, out.Actions[at+1])
		}
	})
}

// TestInsertionGoesOnInserting holds the flow the user is in when a pass turns
// out to have been a take: the rest of that game has to be typed IN, ahead of the
// game that was started after it, and each Action of it must not overwrite the
// opening that follows.
func TestInsertionGoesOnInserting(t *testing.T) {
	doc := typedMatch(t, 7)
	at := 2
	doc = seek(t, doc, at)
	count := len(doc.Actions)

	doc, err := Apply(doc, Gesture{Kind: GestureInsertBefore})
	if err != nil {
		t.Fatal(err)
	}
	doc = runSteps(t, doc, []step{
		{"a roll", die(4), nil},
		{"its second die", die(2), nil},
		{"a play", candidate(0), nil},
		{"validate", confirm(), nil},
	})

	if len(doc.Actions) != count+1 {
		t.Fatalf("actions = %d, want one more", len(doc.Actions))
	}
	if doc.Entry == nil || doc.Entry.Mode != EntryNew || doc.Entry.At != at+1 {
		t.Fatalf("entry = %+v, want a new insertion waiting at %d", doc.Entry, at+1)
	}
	// The second Action of the passage is inserted too, and the Action that was
	// there — the one the user is typing IN FRONT OF — is still there.
	after := doc.Actions[at+1]
	doc = runSteps(t, doc, []step{
		{"a roll", die(5), nil},
		{"its second die", die(1), nil},
		{"a play", candidate(0), nil},
		{"validate", confirm(), nil},
	})
	if len(doc.Actions) != count+2 {
		t.Fatalf("actions = %d, want two more — the second roll overwrote instead of inserting", len(doc.Actions))
	}
	if !sameAction(doc.Actions[at+2], after) {
		t.Errorf("action %d = %+v, want the one that was there: %+v", at+2, doc.Actions[at+2], after)
	}
}

// TestEntryInfoDrawsTheActionBeingTyped: the Transcript shows a correction AS IT IS
// TYPED, not the recorded Action until validation (ADR-0048).
func TestEntryInfoDrawsTheActionBeingTyped(t *testing.T) {
	t.Run("a roll retyped in place is reported with its play", func(t *testing.T) {
		doc := typedMatch(t, 7)
		at := 1
		doc = seek(t, doc, at)
		doc = runSteps(t, doc, []step{
			{"a roll", die(4), nil},
			{"its second die", die(2), nil},
			{"a play", candidate(0), nil},
		})

		e := Replay(doc, 0).Entry
		if e == nil {
			t.Fatal("no entry was reported while one is being typed")
		}
		if e.At != at || !e.Replacing {
			t.Fatalf("entry = %+v, want a replacement at %d", e, at)
		}
		if e.Kind != KindChecker {
			t.Errorf("kind = %q, want a checker play", e.Kind)
		}
		if e.Dice != [2]int{4, 2} {
			t.Errorf("dice = %v, want the roll being typed", e.Dice)
		}
		if e.Notation == "" {
			t.Error("the play picked has no notation, so the cell cannot be drawn")
		}
		// Nothing is written until validation: the Action still says what it said.
		if doc.Actions[at].Dice == [2]int{4, 2} {
			t.Error("the action was rewritten before the validation")
		}
	})

	t.Run("a game's first play says so, whatever the end of the document expects", func(t *testing.T) {
		doc := typedMatch(t, 7)
		doc = seek(t, doc, 0)
		e := Replay(doc, 0).Entry
		if e == nil || !e.GameStart || e.Kind != KindChecker {
			t.Fatalf("entry = %+v, want the game's first play named as one", e)
		}
		if ann := Replay(doc, 0); ann.Next.GameStart {
			t.Fatal("the fixture's end expects a new game too; the case is not built")
		}
	})

	t.Run("an insertion says which slot it will fill", func(t *testing.T) {
		doc := typedMatch(t, 7)
		doc = seek(t, doc, 2)
		doc, err := Apply(doc, Gesture{Kind: GestureInsertBefore})
		if err != nil {
			t.Fatal(err)
		}
		e := Replay(doc, 0).Entry
		if e == nil || e.Replacing || e.At != 2 {
			t.Fatalf("entry = %+v, want an insertion at 2", e)
		}
		if e.Kind != KindChecker {
			t.Errorf("kind = %q, want a checker play", e.Kind)
		}
	})
}

// doubledMatch is a draft whose last game has been doubled and passed: the shape a
// transcriber corrects when the answer was the other one.
func doubledMatch(t *testing.T) Document {
	t.Helper()
	doc := typedMatch(t, 7)
	for _, g := range []Gesture{{Kind: GestureDouble}, {Kind: GesturePass}} {
		next, err := Apply(doc, g)
		if err != nil {
			t.Fatalf("%s: %v", g.Kind, err)
		}
		doc = next
	}
	if lastAction(t, doc).Kind != KindPass {
		t.Fatalf("the fixture does not end on a pass: %+v", lastAction(t, doc))
	}
	if doc.Actions[len(doc.Actions)-2].Side == doc.Actions[len(doc.Actions)-1].Side {
		t.Fatalf("the answer is on the doubler's side: %+v", doc.Actions)
	}
	return doc
}
