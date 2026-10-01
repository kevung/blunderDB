package transcript

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// TestUpgradeDropsTheOpenings reads a version-2 draft — openings, a tie re-rolled with
// the game's declared score on the tie, and an opening that cut a game short — and
// checks the version-3 document it becomes.
func TestUpgradeDropsTheOpenings(t *testing.T) {
	const old = `{"format_version":2,"header":{"match_length":5,"jacoby":false,"beaver":false,"date":"0001-01-01T00:00:00Z"},"actions":[
		{"side":0,"kind":"opening","dice":[3,1]},
		{"side":0,"kind":"checker","dice":[3,1],"steps":[{"from":8,"to":5},{"from":6,"to":5}]},
		{"side":1,"kind":"checker","dice":[3,1],"steps":[{"from":17,"to":20},{"from":19,"to":20}]},
		{"side":0,"kind":"double"},
		{"side":1,"kind":"pass"},
		{"side":0,"kind":"opening","dice":[4,4],"score":[3,0]},
		{"side":1,"kind":"opening","dice":[2,5]},
		{"side":1,"kind":"checker","dice":[5,2],"steps":[{"from":12,"to":17},{"from":12,"to":14}]},
		{"side":0,"kind":"checker","dice":[3,1],"steps":[{"from":8,"to":5},{"from":6,"to":5}]},
		{"side":0,"kind":"opening","dice":[6,1]},
		{"side":0,"kind":"checker","dice":[6,1],"steps":[{"from":13,"to":7},{"from":8,"to":7}]}
	],"cursor":9}`
	var v2 Document
	if err := json.Unmarshal([]byte(old), &v2); err != nil {
		t.Fatal(err)
	}
	doc := Upgrade(v2)

	if doc.FormatVersion != FormatVersion {
		t.Errorf("version = %d", doc.FormatVersion)
	}
	kinds := make([]Kind, len(doc.Actions))
	for i, a := range doc.Actions {
		kinds[i] = a.Kind
	}
	want := []Kind{KindChecker, KindChecker, KindDouble, KindPass, KindChecker, KindChecker, KindChecker}
	if !reflect.DeepEqual(kinds, want) {
		t.Fatalf("kinds = %v, want %v", kinds, want)
	}
	if doc.Actions[0].Score != nil {
		t.Errorf("the first game declared nothing, got %v", *doc.Actions[0].Score)
	}
	// The tie's score passes to the game's first play, made by the re-roll's winner.
	if a := doc.Actions[4]; a.Score == nil || *a.Score != [2]int{3, 0} || a.Side != domain.White {
		t.Errorf("game 2's first play = %+v", a)
	}
	// The opening that cut game 2 short is now a declared score — the one game 2
	// was being played at — on game 3's first play.
	if a := doc.Actions[6]; a.Score == nil || *a.Score != [2]int{3, 0} {
		t.Errorf("game 3's first play = %+v", a)
	}
	if doc.Cursor != 6 {
		t.Errorf("cursor = %d, want 6 — the Action after the opening it stood on", doc.Cursor)
	}

	ann := Replay(doc, 0)
	if len(ann.Games) != 3 || ann.Games[1].Winner != -1 || ann.Games[0].Winner != domain.Black {
		t.Fatalf("games = %+v", ann.Games)
	}
	if g := ann.Games[2]; g.InitialScore != [2]int{3, 0} || g.Declared && g.DerivedScore != [2]int{3, 0} {
		t.Errorf("game 3 = %+v", g)
	}
	for i, info := range ann.Actions {
		for _, inc := range info.Inconsistencies {
			if i != 4 || inc.Kind != ScoreMismatch {
				t.Errorf("action %d: %+v", i, inc)
			}
		}
	}

	if again := Upgrade(doc); !reflect.DeepEqual(again, doc) {
		t.Error("a current document is not returned unchanged")
	}
}

// v2 builds a version-2 draft of a 5-point match from its Actions' JSON.
func v2(t *testing.T, actions string) Document {
	t.Helper()
	var doc Document
	blob := `{"format_version":2,"header":{"match_length":5},"actions":[` + actions + `],"cursor":99}`
	if err := json.Unmarshal([]byte(blob), &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

const wonGame = `{"side":0,"kind":"opening","dice":[3,1]},
	{"side":0,"kind":"checker","dice":[3,1],"steps":[{"from":8,"to":5},{"from":6,"to":5}]},
	{"side":1,"kind":"checker","dice":[3,1],"steps":[{"from":17,"to":20},{"from":19,"to":20}]},
	{"side":0,"kind":"double"},{"side":1,"kind":"pass"}`

// TestUpgradeKeepsATrailingOpening: a draft closed right after an opening keeps what
// that opening said — the score declared on it, or the end of the game it cut short —
// for the next Action written at the end.
func TestUpgradeKeepsATrailingOpening(t *testing.T) {
	declared := Upgrade(v2(t, wonGame+`,{"side":0,"kind":"opening","dice":[4,2],"score":[3,0]}`))
	if declared.NextScore == nil || *declared.NextScore != [2]int{3, 0} {
		t.Fatalf("the declared score was lost: %v", declared.NextScore)
	}
	if n := Replay(declared, 0).Next; !n.GameStart || n.Position.Score != [2]int{2, 5} {
		t.Errorf("next = %+v, want game 2 at the declared 3-0", n)
	}
	doc := runSteps(t, declared, []step{
		{"die 4", die(4), nil}, {"die 2", die(2), nil}, {"a play", candidate(0), nil}, {"validate", confirm(), nil},
	})
	if a := lastAction(t, doc); a.Score == nil || *a.Score != [2]int{3, 0} || doc.NextScore != nil {
		t.Errorf("the first play written = %+v, next score %v", a, doc.NextScore)
	}

	cut := Upgrade(v2(t, `{"side":0,"kind":"opening","dice":[3,1]},
		{"side":0,"kind":"checker","dice":[3,1],"steps":[{"from":8,"to":5},{"from":6,"to":5}]},
		{"side":1,"kind":"opening","dice":[1,6]}`))
	if cut.NextScore == nil || *cut.NextScore != [2]int{} {
		t.Fatalf("the cut is lost: %v", cut.NextScore)
	}
	if n := Replay(cut, 0).Next; !n.GameStart || n.GameNumber != 2 {
		t.Errorf("next = %+v, want game 2", n)
	}
}

// TestUpgradeFirstPlayDice: the first play takes the opening roll in its order —
// player 1's die, player 2's — and one the opening does not give is still marked.
func TestUpgradeFirstPlayDice(t *testing.T) {
	won := Upgrade(v2(t, `{"side":1,"kind":"opening","dice":[2,5]},
		{"side":1,"kind":"checker","dice":[5,2],"steps":[{"from":12,"to":17},{"from":12,"to":14}]}`))
	if a := won.Actions[0]; a.Dice != [2]int{2, 5} || a.Side != domain.White {
		t.Errorf("player 2's opening play = %+v, want dice [2 5]", a)
	}
	if Replay(won, 0).Inconsistent() {
		t.Errorf("a regular opening is marked: %+v", Replay(won, 0).Actions[0].Inconsistencies)
	}

	for name, actions := range map[string]string{
		"played by the loser": `{"side":0,"kind":"opening","dice":[6,3]},
			{"side":1,"kind":"checker","dice":[6,3],"steps":[{"from":1,"to":7},{"from":12,"to":15}]}`,
		"another roll": `{"side":0,"kind":"opening","dice":[6,3]},
			{"side":0,"kind":"checker","dice":[5,2],"steps":[{"from":13,"to":8},{"from":13,"to":11}]}`,
	} {
		if info := Replay(Upgrade(v2(t, actions)), 0).Actions[0]; !hasInconsistency(info, InconsistentDice) {
			t.Errorf("%s: not marked: %+v", name, info.Inconsistencies)
		}
	}
}

// TestUpgradedLoserPlayIsRepairable: a first play the opening's loser made reads back
// marked, and either gesture that says who really started repairs it — the roll typed
// again in player order, or `s`.
func TestUpgradedLoserPlayIsRepairable(t *testing.T) {
	loser := Upgrade(v2(t, `{"side":0,"kind":"opening","dice":[6,3]},
		{"side":1,"kind":"checker","dice":[6,3],"steps":[{"from":1,"to":7},{"from":12,"to":15}]}`))
	if !hasInconsistency(Replay(loser, 0).Actions[0], InconsistentDice) {
		t.Fatal("fixture: the loser's play is not marked")
	}

	retyped := runSteps(t, loser, []step{
		{"back", Gesture{Kind: GestureCursorBack}, nil},
		{"player 1's die", die(3), nil}, {"player 2's die", die(6), nil},
		{"validate", confirm(), nil},
	})
	if a := retyped.Actions[0]; a.Side != domain.White || a.Dice != [2]int{3, 6} {
		t.Errorf("retyped = %+v, want player 2's 36", a)
	}
	if Replay(retyped, 0).Inconsistent() {
		t.Errorf("retyping the roll did not repair it: %+v", Replay(retyped, 0).Actions[0].Inconsistencies)
	}

	flipped := runSteps(t, loser, []step{
		{"back", Gesture{Kind: GestureCursorBack}, nil},
		{"flip", Gesture{Kind: GestureFlipSide}, nil},
	})
	if a := flipped.Actions[0]; a.Side != domain.Black || a.Dice != [2]int{6, 3} {
		t.Errorf("flipped = %+v, want player 1 with the roll as it stands", a)
	}
	// The play is still player 2's steps — illegal for player 1 — but the opening
	// itself no longer contradicts the camp.
	for _, inc := range Replay(flipped, 0).Actions[0].Inconsistencies {
		if strings.Contains(inc.Detail, "won the opening roll") {
			t.Errorf("`s` did not repair the opening: %+v", inc)
		}
	}
}

// TestUpgradeOtherRollKeepsItsDice: a first play made with a roll the opening did not
// give keeps its own dice, and is marked.
func TestUpgradeOtherRollKeepsItsDice(t *testing.T) {
	doc := Upgrade(v2(t, `{"side":0,"kind":"opening","dice":[6,3]},
		{"side":0,"kind":"checker","dice":[5,2],"steps":[{"from":13,"to":8},{"from":13,"to":11}]}`))
	if d := doc.Actions[0].Dice; !sameRoll(d, [2]int{5, 2}) {
		t.Errorf("dice = %v, want the play's 52", d)
	}
	if !hasInconsistency(Replay(doc, 0).Actions[0], InconsistentDice) {
		t.Error("not marked")
	}
}

// TestNextScore holds the boundary waiting at the end: it is corrected or cleared
// by GestureSetScore at the end slot, survives JSON, an insertion in the middle and
// the deletion of the last Action, and is used only by an Action appended at the end.
func TestNextScore(t *testing.T) {
	doc := Upgrade(v2(t, wonGame+`,{"side":0,"kind":"opening","dice":[4,2],"score":[3,0]}`))
	end := len(doc.Actions)

	set, err := Apply(doc, Gesture{Kind: GestureSetScore, At: end, Score: scoreOf(2, 0)})
	if err != nil || set.NextScore == nil || *set.NextScore != [2]int{2, 0} {
		t.Fatalf("set at the end: %v %v", set.NextScore, err)
	}
	cleared, err := Apply(doc, Gesture{Kind: GestureSetScore, At: end})
	if err != nil || cleared.NextScore != nil {
		t.Fatalf("cleared at the end: %v %v", cleared.NextScore, err)
	}

	blob, _ := json.Marshal(doc)
	var back Document
	if err := json.Unmarshal(blob, &back); err != nil || back.NextScore == nil || *back.NextScore != [2]int{3, 0} {
		t.Fatalf("JSON round trip: %v %v", back.NextScore, err)
	}

	mid := doc.clone()
	mid.Cursor = 1
	ins, err := Apply(mid, Gesture{Kind: GestureInsertBefore})
	if err != nil {
		t.Fatal(err)
	}
	ins, err = Apply(ins, Gesture{Kind: GestureResign, Level: 1})
	if err != nil {
		t.Fatal(err)
	}
	if ins.NextScore == nil || len(ins.Actions) != end+1 {
		t.Errorf("an insertion in the middle used the boundary: %v", ins.NextScore)
	}

	del := doc.clone()
	del.Cursor = end - 1
	del, err = Apply(del, Gesture{Kind: GestureDelete})
	if err != nil || del.NextScore == nil || *del.NextScore != [2]int{3, 0} {
		t.Errorf("deleting the last Action: %v %v", del.NextScore, err)
	}
}
