package database

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// A second instance opens the library read-only; moving the Cursor of a draft
// writes nothing, so it works there, and leaves the revision alone.
func TestTranscription_ReadOnlyInstanceMovesTheCursor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shared.db")
	writer := NewDatabase()
	if err := writer.SetupDatabase(path); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	defer writer.Close()
	st, err := writer.CreateTranscription(transcript.Header{MatchLength: 7})
	if err != nil {
		t.Fatalf("CreateTranscription: %v", err)
	}
	typeChecker(t, writer, st.ID, 6, 3)
	before := readTranscriptionRow(t, writer, st.ID).Revision

	reader := NewDatabase()
	if err := reader.OpenDatabase(path); err != nil {
		t.Fatalf("OpenDatabase: %v", err)
	}
	defer reader.Close()
	if !reader.IsReadOnly() {
		t.Fatal("the second instance must be read-only")
	}
	if _, err := reader.OpenTranscription(st.ID); err != nil {
		t.Fatalf("OpenTranscription read-only: %v", err)
	}
	moved, err := reader.ApplyTranscriptionGesture(st.ID, transcript.Gesture{Kind: transcript.GestureCursorBack})
	if err != nil {
		t.Fatalf("moving the Cursor read-only: %v", err)
	}
	if moved.Annotated.Document.Cursor != 0 {
		t.Fatalf("Cursor at %d after cursor_back, want 0", moved.Annotated.Document.Cursor)
	}
	if after := readTranscriptionRow(t, writer, st.ID).Revision; after != before {
		t.Fatalf("moving the Cursor moved the revision from %d to %d", before, after)
	}
}

// Another writer moved the row under the desktop's session: the gesture that
// would write is not recorded, and the desktop gets the draft as the row now
// holds it, flagged as a conflict, its undo stack reset.
func TestTranscription_ConflictHandsBackTheFreshDraft(t *testing.T) {
	path := filepath.Join(t.TempDir(), "conflict.db")
	db := NewDatabase()
	if err := db.SetupDatabase(path); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	defer db.Close()
	st, err := db.CreateTranscription(transcript.Header{MatchLength: 7})
	if err != nil {
		t.Fatalf("CreateTranscription: %v", err)
	}
	typeChecker(t, db, st.ID, 6, 3)

	other, err := sqlite.Open(context.Background(), path, nil)
	if err != nil {
		t.Fatalf("second handle: %v", err)
	}
	defer other.Close()
	row := readTranscriptionRow(t, db, st.ID)
	if _, err := other.Transcriptions().Touch(context.Background(), "", st.ID, row.Revision); err != nil {
		t.Fatalf("the other writer: %v", err)
	}

	for _, g := range []transcript.Gesture{{Kind: transcript.GestureEnterDie, Die: 5}, {Kind: transcript.GestureEnterDie, Die: 2},
		{Kind: transcript.GestureSelectCandidate, Candidate: 0}} {
		applyGesture(t, db, st.ID, g)
	}
	got, err := db.ApplyTranscriptionGesture(st.ID, transcript.Gesture{Kind: transcript.GestureValidate})
	if err != nil {
		t.Fatalf("a conflict is handed back as a state, not an error: %v", err)
	}
	if !got.Conflict {
		t.Fatal("the state must be flagged as a conflict")
	}
	if n := len(got.Annotated.Document.Actions); n != 1 || got.CanUndo {
		t.Fatalf("fresh draft: %d actions, canUndo %v; want 1, false", n, got.CanUndo)
	}
	next := applyGesture(t, db, st.ID, transcript.Gesture{Kind: transcript.GestureEnterDie, Die: 5})
	if next.Conflict {
		t.Fatal("the gesture after the conflict is typed on the fresh draft")
	}
}

// Finishing a draft another writer moved writes no Match: the desktop gets the
// draft as it now stands, flagged as a conflict, and can finish again.
func TestTranscription_FinishConflictHandsBackTheFreshDraft(t *testing.T) {
	path := filepath.Join(t.TempDir(), "finish-conflict.db")
	db := NewDatabase()
	if err := db.SetupDatabase(path); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	defer db.Close()
	st, err := db.CreateTranscription(transcript.Header{MatchLength: 7})
	if err != nil {
		t.Fatalf("CreateTranscription: %v", err)
	}
	typeChecker(t, db, st.ID, 6, 3)

	other, err := sqlite.Open(context.Background(), path, nil)
	if err != nil {
		t.Fatalf("second handle: %v", err)
	}
	defer other.Close()
	row := readTranscriptionRow(t, db, st.ID)
	if _, err := other.Transcriptions().Touch(context.Background(), "", st.ID, row.Revision); err != nil {
		t.Fatalf("the other writer: %v", err)
	}

	res, err := db.FinishTranscription(st.ID)
	if err != nil {
		t.Fatalf("a conflict is handed back as a result, not an error: %v", err)
	}
	if !res.Conflict || res.State == nil || !res.State.Conflict || res.MatchID != 0 {
		t.Fatalf("finish behind another writer: %+v; want a conflict with the fresh draft and no Match", res)
	}
	if n := len(res.State.Annotated.Document.Actions); n != 1 || res.State.CanUndo {
		t.Fatalf("fresh draft: %d actions, canUndo %v; want 1, false", n, res.State.CanUndo)
	}
	again, err := db.FinishTranscription(st.ID)
	if err != nil || again.Conflict || again.MatchID == 0 {
		t.Fatalf("finishing again: %+v, %v; want a Match", again, err)
	}
}
