package cli

import (
	"encoding/json"
	"path/filepath"
	"testing"

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
