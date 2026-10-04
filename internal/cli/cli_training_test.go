package cli

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// training missed lists the positions answered wrong, each once and the most
// recently missed first, and turns them into a collection and a deck.
func TestCLI_TrainingMissed(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	_, ids := seedCollection(t, cli, "c", 3)
	a, b, c := ids[0], ids[1], ids[2]
	mp := func(v int) *int { return &v }
	item := func(pos int64, wrong bool, cost *int) storage.TrainingItem {
		return storage.TrainingItem{NumberType: "decision.checker", Wrong: wrong, PositionID: &pos, Answer: "13/7", ErrorMp: cost}
	}
	older, err := cli.db.SaveTrainingSession(storage.TrainingSession{Exercise: "decision", SeedSource: "library", NumbersAsked: 2, Faults: 1,
		Items: []storage.TrainingItem{item(a, true, mp(80)), item(b, false, mp(0))}})
	if err != nil {
		t.Fatalf("SaveTrainingSession: %v", err)
	}
	// Out of time: wrong, with no cost — still missed. a, missed again, is
	// listed once.
	if _, err := cli.db.SaveTrainingSession(storage.TrainingSession{Exercise: "decision", SeedSource: "library", NumbersAsked: 2, Faults: 2,
		Items: []storage.TrainingItem{item(c, true, nil), item(a, true, mp(40))}}); err != nil {
		t.Fatalf("SaveTrainingSession: %v", err)
	}

	run := func(args ...string) struct {
		PositionIDs  []int64
		DeckID       int64
		CollectionID int64
	} {
		t.Helper()
		var res struct {
			PositionIDs  []int64
			DeckID       int64
			CollectionID int64
		}
		out := captureStdout(t, func() {
			if err := cli.Run(append([]string{"training", "missed", "--db", dbPath, "--format", "json"}, args...)); err != nil {
				t.Fatalf("training missed %v: %v", args, err)
			}
		})
		if err := json.Unmarshal([]byte(out), &res); err != nil {
			t.Fatalf("json: %v\n%s", err, out)
		}
		return res
	}

	all := run()
	if len(all.PositionIDs) != 2 || !slices.Contains(all.PositionIDs, a) || !slices.Contains(all.PositionIDs, c) {
		t.Errorf("missed = %v, want %d and %d, each once (%d was answered right)", all.PositionIDs, a, c, b)
	}
	if got := run("--session", itoa64(older)).PositionIDs; !slices.Equal(got, []int64{a}) {
		t.Errorf("missed in session %d = %v, want [%d]", older, got, a)
	}
	if got := run("--exercise", "pips").PositionIDs; len(got) != 0 {
		t.Errorf("missed in pips = %v, want none", got)
	}

	made := run("--collection", "Mes ratés", "--deck", "Ratés")
	if made.CollectionID == 0 || made.DeckID == 0 {
		t.Fatalf("collection %d, deck %d: want both created", made.CollectionID, made.DeckID)
	}
	inCollection, err := cli.db.GetCollectionPositions(made.CollectionID)
	if err != nil || len(inCollection) != 2 {
		t.Errorf("collection holds %d positions (err %v), want 2", len(inCollection), err)
	}
	inDeck, err := cli.db.GetAnkiDeckPositions(made.DeckID)
	if err != nil || len(inDeck) != 2 {
		t.Errorf("deck holds %d positions (err %v), want 2", len(inDeck), err)
	}

	if err := cli.Run([]string{"training", "missed", "--db", dbPath, "--exercise", "pips", "--deck", "rien"}); err == nil {
		t.Error("a deck of no missed position must be refused")
	}
}
