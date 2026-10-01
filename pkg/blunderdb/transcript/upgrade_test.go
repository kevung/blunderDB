package transcript

import (
	"encoding/json"
	"reflect"
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
