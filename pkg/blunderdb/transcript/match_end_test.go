package transcript

import (
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// wonAtThree is a 3-point match player 1 wins 3-0 from two points away, by the
// same jump that overshoots the Crawford game as 5-3 + 4 in a 9-point match.
func wonAtThree(t *testing.T) Document {
	t.Helper()
	doc := docOf(3)
	doc.Actions = append(doc.Actions, firstCandidate(t, doc, domain.Black, 6, 3),
		Action{Side: domain.White, Kind: KindResign, Level: 1})
	doc.Actions = append(doc.Actions, firstCandidate(t, doc, domain.Black, 5, 2),
		Action{Side: domain.White, Kind: KindResign, Level: 2})
	doc.Cursor = len(doc.Actions)
	return doc
}

func TestMatchEndOpensNoNextGame(t *testing.T) {
	ann := Replay(wonAtThree(t), 0)
	if !ann.Finished || ann.Winner != domain.Black || ann.Score != [2]int{3, 0} {
		t.Fatalf("finished %v, winner %d, score %v", ann.Finished, ann.Winner, ann.Score)
	}
	n := ann.Next
	if !n.MatchOver || n.Expects != "" || n.GameStart || n.Crawford {
		t.Errorf("next = over %v, expects %q, game start %v, Crawford %v; a won match awaits nothing",
			n.MatchOver, n.Expects, n.GameStart, n.Crawford)
	}
	if n.GameNumber != 2 {
		t.Errorf("game number = %d, want the last one played", n.GameNumber)
	}
	// Away score 0 reads "post-Crawford": the winner keeps the score of the game
	// the match ended on, the only one a Position can state.
	if got := n.Position.Score; got != [2]int{2, 3} {
		t.Errorf("away score = %v, want the last game's [2 3]", got)
	}
	if n.Position.Board == InitialBoard() {
		t.Error("the board is set up for a game that will not be played")
	}
}

func TestMatchEndStopsNewEntries(t *testing.T) {
	doc := wonAtThree(t)
	for _, g := range []Gesture{
		{Kind: GestureEnterDie, Die: 6},
		{Kind: GestureDouble},
		{Kind: GestureTake},
		{Kind: GesturePass},
		{Kind: GestureResign, Level: 1},
		{Kind: GestureDance},
	} {
		got, err := Apply(doc, g)
		if !errors.Is(err, ErrMatchOver) {
			t.Errorf("%s: err = %v, want ErrMatchOver", g.Kind, err)
		}
		if len(got.Actions) != len(doc.Actions) {
			t.Errorf("%s wrote an Action past the end", g.Kind)
		}
	}
}

func TestMatchEndKeepsCorrections(t *testing.T) {
	doc := wonAtThree(t)
	back, err := Apply(doc, Gesture{Kind: GestureCursorBack})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(back, Gesture{Kind: GestureEnterDie, Die: 6}); err != nil {
		t.Errorf("correcting a recorded Action: %v", err)
	}
	// A correction lowering the last game's stake reopens the match.
	shorter, err := Apply(doc, Gesture{Kind: GestureSetLength, HasLength: true, MatchLength: 5})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(shorter, Gesture{Kind: GestureEnterDie, Die: 6}); err != nil {
		t.Errorf("the match is no longer won: %v", err)
	}
}

// A play selected but not validated can be the one that wins the match: the cube key
// records it, and the cube Action it would have led to has no game left to belong to.
func TestMatchEndAfterTheValidationACubeGestureCommits(t *testing.T) {
	var doc Document
	var last Action
	for seed := int64(1); ; seed++ {
		doc = playLegalMatch(t, seed, 1).Doc.clone()
		last = doc.Actions[len(doc.Actions)-1]
		if last.Kind == KindChecker {
			break
		}
	}
	doc.Actions = doc.Actions[:len(doc.Actions)-1]
	doc.Cursor, doc.Entry = len(doc.Actions), nil
	for _, d := range last.Dice {
		var err error
		if doc, err = Apply(doc, die(d)); err != nil {
			t.Fatal(err)
		}
	}
	played := len(doc.Actions)
	won := false
	for i := range Candidates(doc) {
		selected, err := Apply(doc, candidate(i))
		if err != nil {
			t.Fatal(err)
		}
		got, err := Apply(selected, Gesture{Kind: GestureDouble})
		if err != nil {
			t.Fatal(err)
		}
		if !Replay(got, 0).Finished {
			continue
		}
		won = true
		if len(got.Actions) != played+1 {
			t.Errorf("%d Actions, want %d: the winning play and no double", len(got.Actions), played+1)
		}
	}
	if !won {
		t.Fatal("no candidate wins the match")
	}
}
