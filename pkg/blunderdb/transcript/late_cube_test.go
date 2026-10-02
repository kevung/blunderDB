package transcript

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// passedGame is a game that ends on a pass: two plays, a double, a pass.
func passedGame(t *testing.T) Document {
	t.Helper()
	doc := docOf(5)
	doc.Actions = append(doc.Actions, firstCandidate(t, doc, domain.Black, 3, 1))
	doc.Actions = append(doc.Actions, firstCandidate(t, doc, domain.White, 5, 2))
	doc.Actions = append(doc.Actions,
		Action{Side: domain.Black, Kind: KindDouble},
		Action{Side: domain.White, Kind: KindPass})
	doc.Cursor = len(doc.Actions)
	return doc
}

// assertLateCube checks that the Actions from `from` on are cube actions kept in
// the finished first game, each marked, and that no game was opened for them.
func assertLateCube(t *testing.T, ann Annotated, from int) {
	t.Helper()
	if len(ann.Games) != 1 {
		t.Fatalf("a cube action after the end of a game opened a game: %d games", len(ann.Games))
	}
	g := ann.Games[0]
	if !g.Finished || g.Winner != domain.Black || g.PointsWon != 1 {
		t.Errorf("the finished game changed: %+v", g)
	}
	if g.Last != len(ann.Actions)-1 {
		t.Errorf("the late cube actions are not in the finished game: last %d", g.Last)
	}
	for i := from; i < len(ann.Actions); i++ {
		info := ann.Actions[i]
		if info.GameIndex != 0 || info.OpensGame {
			t.Errorf("action %d: game %d, opens %v", i, info.GameIndex, info.OpensGame)
		}
		if !hasInconsistency(info, ImpossibleCube) {
			t.Errorf("action %d is not marked: %+v", i, info.Inconsistencies)
		}
	}
	if !ann.Next.GameStart {
		t.Error("the next Action no longer opens a game")
	}
	if ann.Score != [2]int{1, 0} {
		t.Errorf("score %v, want 1-0", ann.Score)
	}
}

func TestCubeActionAfterTheEndOfAGameStaysInIt(t *testing.T) {
	doc := passedGame(t)
	n := len(doc.Actions)
	doc.Actions = append(doc.Actions,
		Action{Side: domain.Black, Kind: KindDouble},
		Action{Side: domain.White, Kind: KindTake})
	doc.Cursor = len(doc.Actions)

	ann := Replay(doc, 0)
	assertLateCube(t, ann, n)

	// The next game opens on the next play, at the score the first one left.
	doc.Actions = append(doc.Actions, firstCandidate(t, doc, domain.White, 4, 2))
	ann = Replay(doc, 0)
	if len(ann.Games) != 2 || !ann.Actions[len(ann.Actions)-1].OpensGame {
		t.Fatalf("the next play does not open game 2: %d games", len(ann.Games))
	}
	if ann.Games[1].InitialScore != [2]int{1, 0} {
		t.Errorf("game 2 starts at %v", ann.Games[1].InitialScore)
	}

	// Every late cube action is still a Move of the finished game.
	p := Build(doc)
	if got := len(p.Moves[p.Games[0].ID]); got != n+2 {
		t.Errorf("game 1 has %d moves, want %d", got, n+2)
	}
}

func TestDoubleTypedAfterTheEndOfAGameStaysInIt(t *testing.T) {
	doc := passedGame(t)
	n := len(doc.Actions)
	doc, err := Apply(doc, Gesture{Kind: GestureDouble})
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Actions) != n+1 || doc.Actions[n].Kind != KindDouble {
		t.Fatalf("the double was not recorded: %+v", doc.Actions)
	}
	assertLateCube(t, Replay(doc, 0), n)
}

// TestFromMATCubeActionAfterTheFinalBearOff reads a match whose file records a
// double and a take after the last checker was borne off: they stay in the one
// game, marked, and open none.
func TestFromMATCubeActionAfterTheFinalBearOff(t *testing.T) {
	text, err := os.ReadFile(filepath.Join("testdata", "late_cube.mat"))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := FromMAT(string(text))
	if err != nil {
		t.Fatal(err)
	}
	ann := Replay(doc, 0)
	if len(ann.Games) != 1 {
		t.Fatalf("%d games, want 1", len(ann.Games))
	}
	n := len(ann.Actions)
	if n < 2 || ann.Actions[n-2].Kind != KindDouble || ann.Actions[n-1].Kind != KindTake {
		t.Fatalf("the file's last two actions are not a double and a take")
	}
	for _, info := range ann.Actions[n-2:] {
		if info.GameIndex != 0 || !hasInconsistency(info, ImpossibleCube) {
			t.Errorf("action %d: game %d, %+v", info.Index, info.GameIndex, info.Inconsistencies)
		}
	}
	if g := ann.Games[0]; !g.Finished || g.Winner < 0 || !ann.Finished {
		t.Errorf("game 1: %+v", g)
	}
}
