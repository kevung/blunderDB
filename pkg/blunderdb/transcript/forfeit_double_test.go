package transcript

import (
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// pendingDoubleDoc is a 5-point match where White has doubled after Black's first
// play and nobody has answered.
func pendingDoubleDoc(t *testing.T) Document {
	t.Helper()
	doc := docOf(5)
	doc.Actions = append(doc.Actions, firstCandidate(t, doc, domain.Black, 6, 3))
	doc.Actions = append(doc.Actions, Action{Side: domain.White, Kind: KindDouble})
	doc.Cursor = len(doc.Actions)
	return doc
}

// TestForfeitAnswersAPendingDouble: with no side given, the forfeit is the
// answerer's. The doubler wins what brings them to the match length, and nothing
// follows.
func TestForfeitAnswersAPendingDouble(t *testing.T) {
	doc := pendingDoubleDoc(t)
	out, err := Apply(doc, Gesture{Kind: GestureForfeit})
	if err != nil {
		t.Fatal(err)
	}
	last := out.Actions[len(out.Actions)-1]
	if last.Kind != KindForfeit || last.Side != domain.Black {
		t.Fatalf("last Action = %s by %d, want a forfeit by Black", last.Kind, last.Side)
	}
	ann := Replay(out, 0)
	if !ann.Finished || ann.Winner != domain.White || ann.Score != [2]int{0, 5} {
		t.Fatalf("finished %v, winner %d, score %v; want the match to White 0-5", ann.Finished, ann.Winner, ann.Score)
	}
	if _, err := Apply(out, Gesture{Kind: GestureTake}); err == nil {
		t.Error("an answer after the forfeit was accepted")
	}
}

// TestResignAnswersAPendingDoubleAtTheCubeBeforeTheOffer: a resignation is worth
// the cube as it stood before the offer, as a pass is.
func TestResignAnswersAPendingDoubleAtTheCubeBeforeTheOffer(t *testing.T) {
	for level := 1; level <= 3; level++ {
		out, err := Apply(pendingDoubleDoc(t), Gesture{Kind: GestureResign, Level: level})
		if err != nil {
			t.Fatal(err)
		}
		if last := out.Actions[len(out.Actions)-1]; last.Side != domain.Black {
			t.Fatalf("level %d: resigned by %d, want Black", level, last.Side)
		}
		ann := Replay(out, 0)
		if ann.Score != [2]int{0, level} {
			t.Errorf("level %d: score %v; want White +%d", level, ann.Score, level)
		}
	}
}

// TestMATOfAForfeitFacingADouble: the game ends on the doubler's offer, then their
// win, with no Takes or Drops.
func TestMATOfAForfeitFacingADouble(t *testing.T) {
	out, err := Apply(pendingDoubleDoc(t), Gesture{Kind: GestureForfeit})
	if err != nil {
		t.Fatal(err)
	}
	m, games, moves := MatchParts(out)
	mat := ingest.RenderMAT(m, games, moves)
	if strings.Contains(mat, "Takes") || strings.Contains(mat, "Drops") {
		t.Errorf("an answer was written:\n%s", mat)
	}
	lines := strings.Split(strings.TrimRight(mat, "\n"), "\n")
	n := len(lines)
	if n < 2 || !strings.Contains(lines[n-2], "Doubles => 2") || !strings.Contains(lines[n-1], "Wins 5 points and the match") {
		t.Fatalf("want a double then the win and the match:\n%s", mat)
	}
	// Both cells sit in the doubler's column (White, the right one).
	if i := strings.Index(lines[n-2], "Doubles"); i < 20 {
		t.Errorf("double in the left column:\n%s", mat)
	}
	if i := strings.Index(lines[n-1], "Wins"); i < 20 {
		t.Errorf("win in the left column:\n%s", mat)
	}
}
