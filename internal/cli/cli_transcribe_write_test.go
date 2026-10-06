package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// The CLI's writing options go through the panel's Database methods: edit
// opens a draft on a match, finish replaces that match in place, abandon
// drops a draft and leaves the match alone.
func TestTranscribeWrite_EditFinishAbandon(t *testing.T) {
	c, dbPath := setupCLIWithDB(t)
	matchID, err := c.db.ImportGnuBGMatch(filepath.Join("testdata", "charlot1-charlot2_7p_2025-11-08-2305.mat"))
	if err != nil {
		t.Fatalf("ImportGnuBGMatch: %v", err)
	}
	c.db.Close()
	id := itoa64

	var edit transcribeWriteResult
	out, err := runTranscribeOnFile(t, "--db", dbPath, "--match", id(matchID), "--edit", "--format", "json")
	if err != nil {
		t.Fatalf("--edit: %v", err)
	}
	if err := json.Unmarshal([]byte(out), &edit); err != nil || edit.DraftID == 0 {
		t.Fatalf("--edit printed %q (%v)", out, err)
	}

	var finish transcribeWriteResult
	out, err = runTranscribeOnFile(t, "--db", dbPath, "--draft", id(edit.DraftID), "--finish", "--format", "json")
	if err != nil {
		t.Fatalf("--finish: %v", err)
	}
	if err := json.Unmarshal([]byte(out), &finish); err != nil || finish.Finish == nil {
		t.Fatalf("--finish printed %q (%v)", out, err)
	}
	if finish.Finish.MatchID != matchID || !finish.Finish.Replaced {
		t.Fatalf("--finish wrote match %d (replaced %v), want match %d replaced", finish.Finish.MatchID, finish.Finish.Replaced, matchID)
	}

	var again transcribeWriteResult
	out, err = runTranscribeOnFile(t, "--db", dbPath, "--match", id(matchID), "--edit", "--format", "json")
	if err != nil {
		t.Fatalf("second --edit: %v", err)
	}
	if err := json.Unmarshal([]byte(out), &again); err != nil || again.DraftID == 0 {
		t.Fatalf("second --edit printed %q (%v)", out, err)
	}
	if _, err := runTranscribeOnFile(t, "--db", dbPath, "--draft", id(again.DraftID), "--abandon"); err != nil {
		t.Fatalf("--abandon: %v", err)
	}
	if _, err := runTranscribeOnFile(t, "--db", dbPath, "--match", id(matchID), "--check"); err != nil {
		t.Fatalf("the match does not replay after the abandon: %v", err)
	}
}

func TestTranscribeWrite_RefusesAMismatchedSource(t *testing.T) {
	for _, args := range [][]string{
		{"--db", ":memory:", "--match", "1", "--finish"},
		{"--db", ":memory:", "--draft", "1", "--edit"},
		{"--draft", "1", "--abandon"},
		{"--db", ":memory:", "--draft", "1", "--finish", "--abandon"},
	} {
		if _, err := runTranscribeText(t, args...); err == nil {
			t.Errorf("transcribe %v succeeded", args)
		}
	}
}

// runTranscribeOnFile is runTranscribeText closing its database afterwards: a
// handle left open keeps the library's write lock, and the next command
// would open it read-only.
func runTranscribeOnFile(t *testing.T, args ...string) (string, error) {
	t.Helper()
	c := &CLI{db: NewDatabase()}
	defer c.db.Close()
	var err error
	out := captureStdout(t, func() { err = c.runTranscribe(args) })
	return out, err
}

// A draft that never produced a Match loses everything typed in it when
// abandoned: the CLI asks for --yes, as the panel asks for a confirmation.
func TestTranscribeWrite_AbandonOfANeverFinishedDraftNeedsYes(t *testing.T) {
	c, dbPath := setupCLIWithDB(t)
	state, err := c.db.CreateTranscription(transcript.Header{MatchLength: 7})
	if err != nil {
		t.Fatalf("CreateTranscription: %v", err)
	}
	c.db.Close()

	if _, err := runTranscribeOnFile(t, "--db", dbPath, "--draft", itoa64(state.ID), "--abandon"); err == nil {
		t.Fatal("--abandon without --yes deleted a draft that never produced a match")
	}
	if _, err := runTranscribeOnFile(t, "--db", dbPath, "--draft", itoa64(state.ID), "--check"); err != nil {
		t.Fatalf("the refused abandon took the draft anyway: %v", err)
	}
	if _, err := runTranscribeOnFile(t, "--db", dbPath, "--draft", itoa64(state.ID), "--abandon", "--yes"); err != nil {
		t.Fatalf("--abandon --yes: %v", err)
	}
}

// --materialize writes a match from a document of Actions in one step; an
// illegal Action fails the command with its rank and writes nothing.
func TestTranscribeMaterialize(t *testing.T) {
	c, dbPath := setupCLIWithDB(t)
	state, err := c.db.CreateTranscription(transcript.Header{MatchLength: 7, Player1: "A", Player2: "B"})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range [][2]int{{3, 1}, {4, 2}, {5, 3}} {
		for _, g := range []transcript.Gesture{
			{Kind: transcript.GestureEnterDie, Die: d[0]}, {Kind: transcript.GestureEnterDie, Die: d[1]},
			{Kind: transcript.GestureSelectCandidate, Candidate: 0}, {Kind: transcript.GestureValidate},
		} {
			if state, err = c.db.ApplyTranscriptionGesture(state.ID, g); err != nil {
				t.Fatal(err)
			}
		}
	}
	doc := state.Annotated.Document
	if err := c.db.AbandonTranscription(state.ID); err != nil {
		t.Fatal(err)
	}
	c.db.Close()

	write := func(actions []transcript.Action) string {
		raw, _ := json.Marshal(materializeDocument{Header: doc.Header, Actions: actions})
		p := filepath.Join(t.TempDir(), "match.json")
		if err := os.WriteFile(p, raw, 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	bad := append([]transcript.Action(nil), doc.Actions...)
	bad[2] = bad[1]
	if _, err := runTranscribeOnFile(t, "--db", dbPath, "--materialize", write(bad), "--format", "json"); err == nil || !strings.Contains(err.Error(), "action 2 refused") {
		t.Fatalf("an illegal Action must refuse with its rank: %v", err)
	}

	out, err := runTranscribeOnFile(t, "--db", dbPath, "--materialize", write(doc.Actions), "--format", "json")
	if err != nil {
		t.Fatalf("--materialize: %v", err)
	}
	var res database.TranscriptionSaveResult
	if err := json.Unmarshal([]byte(out), &res); err != nil || res.MatchID == 0 || res.Moves != 3 {
		t.Fatalf("--materialize printed %q (%v)", out, err)
	}
	if _, err := runTranscribeOnFile(t, "--db", dbPath, "--match", itoa64(res.MatchID), "--check"); err != nil {
		t.Fatalf("the written match does not replay: %v", err)
	}
}
