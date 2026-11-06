package transcript

import (
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// TestForfeitGestureGivesTheMatchUp: the gesture records a KindForfeit, not a
// resignation — a resignation of the same game at 5 away would leave the match
// open at 1-0 — and is timed, undone and closes the document like a won match.
func TestForfeitGestureGivesTheMatchUp(t *testing.T) {
	doc := docOf(5)
	doc.Actions = append(doc.Actions, firstCandidate(t, doc, domain.Black, 6, 3))
	doc.Cursor = len(doc.Actions)
	e := NewEditor(doc)
	if err := e.Apply(at(Gesture{Kind: GestureForfeit, HasSide: true, Side: domain.White}, 42000)); err != nil {
		t.Fatal(err)
	}
	last := e.Doc.Actions[len(e.Doc.Actions)-1]
	if last.Kind != KindForfeit || last.Side != domain.White {
		t.Fatalf("last Action = %s by %d, want a forfeit by White", last.Kind, last.Side)
	}
	if !sameMS(last.TickMS, ms(42000)) {
		t.Errorf("forfeit Repère = %s, want 42000", show(last.TickMS))
	}
	ann := Replay(e.Doc, 0)
	if !ann.Finished || ann.Winner != domain.Black || ann.Score != [2]int{5, 0} {
		t.Fatalf("finished %v, winner %d, score %v; want the match to Black 5-0", ann.Finished, ann.Winner, ann.Score)
	}
	for _, g := range []Gesture{{Kind: GestureEnterDie, Die: 6}, {Kind: GestureResign, Level: 1}, {Kind: GestureForfeit}} {
		if _, err := Apply(e.Doc, g); !errors.Is(err, ErrMatchOver) {
			t.Errorf("%s after the forfeit: err = %v, want ErrMatchOver", g.Kind, err)
		}
	}
	if !e.Undo() || len(e.Doc.Actions) != 1 || Replay(e.Doc, 0).Finished {
		t.Fatalf("undo left %d actions, finished %v", len(e.Doc.Actions), Replay(e.Doc, 0).Finished)
	}
}

// TestForfeitGestureDefaultsToTheSideOnRoll: without a side, the forfeit is the
// side to act's, as a resignation is.
func TestForfeitGestureDefaultsToTheSideOnRoll(t *testing.T) {
	doc := docOf(0)
	doc.Actions = append(doc.Actions, firstCandidate(t, doc, domain.Black, 6, 3))
	doc.Cursor = len(doc.Actions)
	out, err := Apply(doc, Gesture{Kind: GestureForfeit})
	if err != nil {
		t.Fatal(err)
	}
	ann := Replay(out, 0)
	if got := out.Actions[len(out.Actions)-1].Side; got != domain.White {
		t.Errorf("forfeit by %d, want White, on roll", got)
	}
	if !ann.Finished || ann.Winner != domain.Black || ann.Score != [2]int{1, 0} {
		t.Errorf("money: finished %v, winner %d, score %v; want a single to Black, over", ann.Finished, ann.Winner, ann.Score)
	}
}
