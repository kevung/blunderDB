package transcript

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// TestOpeningCorrectedCarriesTheFirstPlay holds the one exception to "the side
// belongs to the Action": the play a game starts with follows the opening that
// names its camp, because the user never chose that camp — the panel proposed it
// and carried the roll over (fonctionnel.md §1.2).
//
// Nothing else would say it: measured, the play stays legal for either camp from
// the starting board and an opening bears no turn, so the Replay marks neither an
// illegal move nor a double turn.
func TestOpeningCorrectedCarriesTheFirstPlay(t *testing.T) {
	// The opening is 6-3 for player 1, who then plays it.
	opened := func(t *testing.T) Document {
		t.Helper()
		return runSteps(t, openedMatch(t, 7), []step{
			{"a play", candidate(0), nil},
			{"validate", confirm(), nil},
		})
	}
	// 3 then 6 on the opening cell: the higher die is player 2's now.
	retype := func(t *testing.T, doc Document) Document {
		t.Helper()
		return runSteps(t, seek(t, doc, 0), []step{
			{"first die", die(3), nil},
			{"second die", die(6), nil},
			{"validate", confirm(), nil},
		})
	}

	t.Run("the play changes camp with the winner", func(t *testing.T) {
		doc := opened(t)
		if doc.Actions[0].Side != domain.Black || doc.Actions[1].Side != domain.Black {
			t.Fatalf("the fixture does not start with player 1: %+v", doc.Actions)
		}
		played := append([]domain.CheckerStep(nil), doc.Actions[1].Steps...)

		doc = retype(t, doc)

		if doc.Actions[0].Side != domain.White {
			t.Fatalf("opening = %+v, want player 2 as the winner", doc.Actions[0])
		}
		if doc.Actions[1].Side != domain.White {
			t.Errorf("the first play stayed with player %d; it must follow the opening", doc.Actions[1].Side+1)
		}
		// Only the side moved: the play itself is the one that was recorded.
		if !sameSteps(doc.Actions[1].Steps, played) {
			t.Errorf("steps = %v, want the recorded %v", doc.Actions[1].Steps, played)
		}
		if len(doc.Actions) != 2 {
			t.Errorf("actions = %d, want 2 — nothing was added", len(doc.Actions))
		}
	})

	t.Run("a play already given to the other camp is left alone", func(t *testing.T) {
		doc := seek(t, opened(t), 1)
		flipped, err := Apply(doc, Gesture{Kind: GestureFlipSide})
		if err != nil {
			t.Fatal(err)
		}
		if flipped.Actions[1].Side != domain.White {
			t.Fatalf("the fixture was not flipped: %+v", flipped.Actions[1])
		}

		if out := retype(t, flipped); out.Actions[1].Side != domain.White {
			t.Error("a side the user had chosen was overwritten by the opening")
		}
	})

	t.Run("the rest of the game keeps its sides", func(t *testing.T) {
		doc := typedMatch(t, 7)
		sides := make([]int, len(doc.Actions))
		for i, a := range doc.Actions {
			sides[i] = a.Side
		}

		doc = retype(t, doc)

		for i := 2; i < len(doc.Actions); i++ {
			if doc.Actions[i].Side != sides[i] {
				t.Fatalf("action %d changed camp; only the first play follows the opening", i)
			}
		}
	})
}

// TestCubeGestureWritesAtTheCursorsSlot holds the half of fonctionnel.md §2 that
// the four cube gestures had been left out of: they act WHERE THE CURSOR IS, like
// the digit key and like every other gesture of the panel.
//
// The case it comes from is the one a transcriber makes twice an evening: a pass
// typed for a take. Until this, walking back onto the pass and pressing `t` put a
// take IN FRONT of it and left the pass standing — two Actions to correct, a game
// that still ended, and no way at all to go on typing the rest of that game.
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

// TestEntryInfoDrawsTheActionBeingTyped holds what the Transcript needs to show a
// correction AS IT IS TYPED. Without it the cell went on showing the Action that
// was recorded until the validation, so the user read one thing and the document
// said another — the gap ADR-0048 calls a cognitive one, and the reason a
// correction looked like it had done nothing.
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

	t.Run("an opening slot says so, whatever the end of the document expects", func(t *testing.T) {
		doc := typedMatch(t, 7)
		doc = seek(t, doc, 0)
		e := Replay(doc, 0).Entry
		if e == nil || e.Kind != KindOpening {
			t.Fatalf("entry = %+v, want the opening slot named as one", e)
		}
		if ann := Replay(doc, 0); ann.Next.Expects == KindOpening {
			t.Fatal("the fixture's end expects an opening too; the case is not built")
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
