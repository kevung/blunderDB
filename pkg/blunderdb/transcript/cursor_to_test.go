package transcript

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// realMatch is a long, real document — a whole tournament match — with one
// checker play removed so that a double turn leaves a hole to aim at.
func realMatch(t *testing.T) (doc Document, hole int) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "test.mat"))
	if err != nil {
		t.Fatal(err)
	}
	doc, err = FromMAT(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Actions) < 200 {
		t.Fatalf("the fixture has %d Actions, too few to measure a distance", len(doc.Actions))
	}
	for i := 40; i < len(doc.Actions)-1; i++ {
		if doc.Actions[i].Kind == KindChecker && doc.Actions[i-1].Kind == KindChecker && doc.Actions[i+1].Kind == KindChecker {
			doc.Actions = append(doc.Actions[:i:i], doc.Actions[i+1:]...)
			if !holeBefore(doc, i) {
				t.Fatalf("removing Action %d left no hole", i)
			}
			return doc, i
		}
	}
	t.Fatal("no checker play to remove")
	return doc, 0
}

// walkTo steps the Cursor, one cursor_back or cursor_forward at a time, until
// it stands where the click aims: the walk the panel used to send.
func walkTo(t *testing.T, doc Document, at int, hole bool) (Document, int) {
	t.Helper()
	arrived := func(d Document) bool { return d.Cursor == at && onHole(d) == (hole && holeBefore(d, at)) }
	steps := 0
	for !arrived(doc) {
		kind := GestureCursorForward
		if at < doc.Cursor || (at == doc.Cursor && hole) {
			kind = GestureCursorBack
		}
		next, err := Apply(doc, Gesture{Kind: kind})
		if err != nil {
			t.Fatal(err)
		}
		doc = next
		if steps++; steps > 2*len(doc.Actions)+2 {
			t.Fatalf("the walk to %d (hole %v) never arrives", at, hole)
		}
	}
	return doc, steps
}

func TestCursorToLandsWhereTheWalkStops(t *testing.T) {
	doc, hole := realMatch(t)
	n := len(doc.Actions)
	starts := []int{0, hole - 1, hole, n / 2, n}
	targets := []struct {
		at   int
		hole bool
	}{{0, false}, {hole - 1, false}, {hole, false}, {hole, true}, {hole + 1, false}, {n - 1, false}, {n, false}}
	for _, from := range starts {
		start, _ := walkTo(t, doc, from, false)
		for _, to := range targets {
			walked, _ := walkTo(t, start, to.at, to.hole)
			jumped, err := Apply(start, Gesture{Kind: GestureCursorTo, At: to.at, Hole: to.hole})
			if err != nil {
				t.Fatal(err)
			}
			if jumped.Cursor != walked.Cursor || !reflect.DeepEqual(jumped.Entry, walked.Entry) {
				t.Errorf("from %d to %d (hole %v): jump gives cursor %d entry %+v, the walk %d entry %+v",
					from, to.at, to.hole, jumped.Cursor, jumped.Entry, walked.Cursor, walked.Entry)
			}
			if !reflect.DeepEqual(jumped.Actions, walked.Actions) {
				t.Errorf("from %d to %d: the jump wrote what the walk did not", from, to.at)
			}
			if Replay(jumped, 0).Cursor != jumped.Cursor && !jumped.HoldCursor {
				t.Errorf("from %d to %d: the Replay would pull the Cursor off the clicked cell", from, to.at)
			}
		}
	}
}

// A correction left behind is committed, as the walk's first step commits it.
func TestCursorToCommitsTheCorrectionItLeaves(t *testing.T) {
	doc := typedMatch(t, 7)
	doc = seek(t, doc, 2)
	edited, err := Apply(doc, Gesture{Kind: GestureEnterDie, Die: 1})
	if err != nil {
		t.Fatal(err)
	}
	if edited, err = Apply(edited, Gesture{Kind: GestureEnterDie, Die: 1}); err != nil {
		t.Fatal(err)
	}
	walked, _ := walkTo(t, edited, len(edited.Actions), false)
	jumped, err := Apply(edited, Gesture{Kind: GestureCursorTo, At: len(edited.Actions)})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(jumped.Actions, walked.Actions) {
		t.Fatalf("the jump and the walk leave different Actions:\n%+v\n%+v", jumped.Actions[2], walked.Actions[2])
	}
	if jumped.Actions[2].Dice == doc.Actions[2].Dice {
		t.Fatal("the correction was not committed")
	}
	if jumped.Cursor != len(jumped.Actions) || !jumped.HoldCursor {
		t.Fatalf("cursor %d held %v, want the end, held", jumped.Cursor, jumped.HoldCursor)
	}
}

// An undo after a jump returns the Cursor where it was in one stroke.
func TestCursorToIsOneStepOfTheStack(t *testing.T) {
	doc, _ := realMatch(t)
	start, _ := walkTo(t, doc, 3, false)
	ed := NewEditor(start)
	if err := ed.Apply(Gesture{Kind: GestureCursorTo, At: len(doc.Actions) - 3}); err != nil {
		t.Fatal(err)
	}
	if !ed.Undo() || ed.Doc.Cursor != 3 {
		t.Fatalf("after one undo the Cursor is at %d, want 3", ed.Doc.Cursor)
	}
}
