package transcript

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// shownCandidates is the list the panel ranks: LegalMoves on the Position with
// the roll high die first, whichever order the dice were typed or written in.
func shownCandidates(pos domain.Position, dice [2]int) []domain.LegalPlay {
	pos.Dice = CandidateDice(dice)
	pos.DecisionType = domain.CheckerAction
	return domain.LegalMoves(&pos)
}

// assertEveryLineWrites picks every line of the list shown for Action `at` and
// checks the decision written is that line.
func assertEveryLineWrites(t *testing.T, doc Document, at int, shown []domain.LegalPlay) {
	t.Helper()
	if len(shown) < 2 {
		t.Fatalf("the roll offers %d plays: the test needs a choice", len(shown))
	}
	for i := range shown {
		next, err := Apply(doc, candidate(i))
		if err != nil {
			t.Fatal(err)
		}
		next, err = Apply(next, confirm())
		if err != nil {
			t.Fatal(err)
		}
		if got := Replay(next, 0).Actions[at].Notation; got != shown[i].Notation {
			t.Errorf("line %d shows %q, the decision written is %q", i, shown[i].Notation, got)
		}
	}
}

// The panel addresses `select_candidate` by an index in its list; the engine
// resolves it in [Candidates]. The opening roll, low die typed first, is the
// one where the Position (high die first) and the document disagree on order.
func TestReselectOpeningPlayAddressesTheListShown(t *testing.T) {
	doc := runSteps(t, New(7), []step{
		{"die 1", die(1), nil}, {"die 3", die(3), nil}, {"pick", candidate(0), nil}, {"ok", confirm(), nil},
		{"d1", die(5), nil}, {"d2", die(2), nil}, {"pick", candidate(0), nil}, {"ok", confirm(), nil},
		{"back", Gesture{Kind: GestureCursorBack}, nil},
		{"back", Gesture{Kind: GestureCursorBack}, nil},
	})
	info := Replay(doc, 0).Actions[0]
	assertEveryLineWrites(t, doc, 0, shownCandidates(info.Before, info.Before.Dice))
}

// An ordinary decision, dice typed in either order: the fresh entry lists the
// typed roll, the reselection after ↑ lists the Action's Position; both must
// address the same plays as the engine.
func TestCandidatesIgnoreTheOrderTheDiceWereTyped(t *testing.T) {
	for _, roll := range [][2]int{{3, 5}, {5, 3}} {
		opening := []step{
			{"d1", die(3), nil}, {"d2", die(1), nil}, {"pick", candidate(0), nil}, {"ok", confirm(), nil},
			{"d1", die(roll[0]), nil}, {"d2", die(roll[1]), nil},
		}
		fresh := runSteps(t, New(7), opening)
		assertEveryLineWrites(t, fresh, 1, shownCandidates(EntryPosition(fresh), roll))

		written := runSteps(t, New(7), append(opening,
			step{"pick", candidate(0), nil}, step{"ok", confirm(), nil},
			step{"back", Gesture{Kind: GestureCursorBack}, nil},
		))
		info := Replay(written, 0).Actions[1]
		if info.Before.Dice != roll {
			t.Fatalf("roll %v: the Action's Position carries %v", roll, info.Before.Dice)
		}
		assertEveryLineWrites(t, written, 1, shownCandidates(info.Before, info.Before.Dice))
	}
}
