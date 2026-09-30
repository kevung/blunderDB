package database

import (
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// The Match a draft has produced is a fact about the library, not a gesture:
// undo and redo walk the draft's own history and must not forget it, or the
// next save would create a second Match.
func TestFinishTranscription_MatchIDSurvivesUndoAndRedo(t *testing.T) {
	db := newTestDB(t)
	id := matDraft(t, db, filepath.Join("testdata", "test.mat"))

	first, err := db.FinishTranscription(id)
	if err != nil {
		t.Fatalf("FinishTranscription: %v", err)
	}
	id = editDraft(t, db, first.MatchID)
	applyGesture(t, db, id, transcript.Gesture{
		Kind:   transcript.GestureSetHeader,
		Header: transcript.Header{Player1: "Alice", Player2: "Bob"},
	})

	undone := applyGesture(t, db, id, transcript.Gesture{Kind: transcript.GestureUndo})
	if got := undone.Annotated.Document.Header.MatchID; got == nil || *got != first.MatchID {
		t.Fatalf("after undo the draft's match id is %v, want %d", got, first.MatchID)
	}
	redone := applyGesture(t, db, id, transcript.Gesture{Kind: transcript.GestureRedo})
	if got := redone.Annotated.Document.Header.MatchID; got == nil || *got != first.MatchID {
		t.Fatalf("after redo the draft's match id is %v, want %d", got, first.MatchID)
	}
	again, err := db.FinishTranscription(id)
	if err != nil {
		t.Fatalf("second FinishTranscription: %v", err)
	}
	if again.MatchID != first.MatchID || !again.Replaced {
		t.Fatalf("finish after undo and redo: match %d replaced=%v, want match %d replaced", again.MatchID, again.Replaced, first.MatchID)
	}
	if n := countTranscriptRows(t, db, `SELECT COUNT(*) FROM match`); n != 1 {
		t.Fatalf("%d matches in the library, want 1", n)
	}
}

// A Match deleted from the library takes the draft back to "never saved": its
// gestures still write, and the next save creates a Match anew.
func TestFinishTranscription_DeletedMatchMakesTheDraftUnsaved(t *testing.T) {
	db := newTestDB(t)
	id := matDraft(t, db, filepath.Join("testdata", "test.mat"))

	first, err := db.FinishTranscription(id)
	if err != nil {
		t.Fatalf("FinishTranscription: %v", err)
	}
	id = editDraft(t, db, first.MatchID)
	if err := db.DeleteMatch(first.MatchID); err != nil {
		t.Fatalf("DeleteMatch: %v", err)
	}

	state, err := db.ApplyTranscriptionGesture(id, transcript.Gesture{
		Kind:   transcript.GestureSetHeader,
		Header: transcript.Header{Player1: "Alice", Player2: "Bob"},
	})
	if err != nil {
		t.Fatalf("a gesture on a draft whose match was deleted: %v", err)
	}
	if got := state.Annotated.Document.Header.MatchID; got != nil {
		t.Fatalf("the draft still names match %d after its deletion", *got)
	}

	again, err := db.FinishTranscription(id)
	if err != nil {
		t.Fatalf("finish after the match was deleted: %v", err)
	}
	if again.Replaced || again.MatchID == 0 || again.MatchID == first.MatchID {
		t.Fatalf("save after deletion: match %d replaced=%v, want a new match", again.MatchID, again.Replaced)
	}
	if n := countTranscriptRows(t, db, `SELECT COUNT(*) FROM match`); n != 1 {
		t.Fatalf("%d matches in the library, want 1", n)
	}
	if n := countTranscriptRows(t, db, `SELECT COUNT(*) FROM transcription`); n != 0 {
		t.Fatalf("%d draft rows after finishing, want 0", n)
	}
}

// A draft read back from disk after its Match was deleted (another run, the
// server) is "never saved" as well.
func TestOpenTranscription_DeletedMatchAcrossSessions(t *testing.T) {
	db := newTestDB(t)
	id := matDraft(t, db, filepath.Join("testdata", "test.mat"))

	first, err := db.FinishTranscription(id)
	if err != nil {
		t.Fatalf("FinishTranscription: %v", err)
	}
	id = editDraft(t, db, first.MatchID)
	if err := db.DeleteMatch(first.MatchID); err != nil {
		t.Fatalf("DeleteMatch: %v", err)
	}
	db.forgetTranscriptSessions()

	state, err := db.OpenTranscription(id)
	if err != nil {
		t.Fatalf("OpenTranscription: %v", err)
	}
	if got := state.Annotated.Document.Header.MatchID; got != nil {
		t.Fatalf("the reopened draft names deleted match %d", *got)
	}
}

// editDraft opens a draft on matchID and returns its id.
func editDraft(t *testing.T, db *Database, matchID int64) int64 {
	t.Helper()
	state, err := db.EditMatchTranscription(matchID)
	if err != nil {
		t.Fatalf("EditMatchTranscription(%d): %v", matchID, err)
	}
	return state.ID
}
