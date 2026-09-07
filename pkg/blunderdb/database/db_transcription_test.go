package database

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// The engine's own rules are tested in pkg/blunderdb/transcript. What is
// tested here is the plumbing this file is: that a draft survives the round
// trip through the row, that the Action being typed survives between two
// gestures (the reason a session is held at all), and that closing a draft
// deletes it.

func TestCreateTranscription_ListsAndDefaults(t *testing.T) {
	db := newTestDB(t)

	state, err := db.CreateTranscription(transcript.Header{
		MatchLength: transcript.DefaultMatchLength,
		Player1:     "Kévin",
		Player2:     "Alice",
	})
	if err != nil {
		t.Fatalf("CreateTranscription: %v", err)
	}
	if state.ID == 0 {
		t.Fatal("CreateTranscription returned no id")
	}
	if got := state.Annotated.Document.Header.MatchLength; got != 7 {
		t.Fatalf("match length = %d, want 7", got)
	}
	if state.Annotated.Next.Expects != transcript.KindOpening {
		t.Fatalf("a fresh draft expects %q, want an opening", state.Annotated.Next.Expects)
	}

	list, err := db.ListTranscriptions()
	if err != nil {
		t.Fatalf("ListTranscriptions: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("got %d drafts, want 1", len(list))
	}
	if list[0].Label != "Kévin vs Alice" {
		t.Fatalf("label = %q", list[0].Label)
	}
	if list[0].MatchLength != 7 || list[0].ActionCount != 0 || list[0].MatchID != 0 {
		t.Fatalf("summary = %+v", list[0])
	}
}

// A money draft is Jacoby by default; an all-zero header says "unstated", not
// "Jacoby off".
func TestCreateTranscription_MoneyIsJacobyByDefault(t *testing.T) {
	db := newTestDB(t)

	state, err := db.CreateTranscription(transcript.Header{MatchLength: 0})
	if err != nil {
		t.Fatalf("CreateTranscription: %v", err)
	}
	if !state.Annotated.Document.Header.Jacoby {
		t.Fatal("a money draft came back without Jacoby")
	}
}

// The dice are entered one at a time, each through its own gesture and its own
// Wails round trip, and transcript.Document does not serialise the Entry they
// fill. A draft that reloaded from the row between the two would lose the
// first die — which is why an open draft keeps a live editor.
func TestApplyTranscriptionGesture_EntrySurvivesBetweenCalls(t *testing.T) {
	db := newTestDB(t)

	state, err := db.CreateTranscription(transcript.Header{MatchLength: 7})
	if err != nil {
		t.Fatalf("CreateTranscription: %v", err)
	}

	for _, die := range []int{3, 1} {
		state, err = db.ApplyTranscriptionGesture(state.ID, transcript.Gesture{
			Kind: transcript.GestureEnterDie,
			Die:  die,
		})
		if err != nil {
			t.Fatalf("enter die %d: %v", die, err)
		}
	}

	entry := state.Annotated.Document.Entry
	if entry == nil {
		t.Fatal("no entry is being typed after two dice")
	}
	if entry.Dice != [2]int{3, 1} {
		t.Fatalf("entry dice = %v, want [3 1]", entry.Dice)
	}
}

// A gesture that changes an Action is written to the row at once (ADR-0045
// rule 1): reopening the draft finds it, undo stack and Entry aside.
func TestApplyTranscriptionGesture_PersistsTheDocument(t *testing.T) {
	db := newTestDB(t)

	state, err := db.CreateTranscription(transcript.Header{MatchLength: 7})
	if err != nil {
		t.Fatalf("CreateTranscription: %v", err)
	}
	state, err = db.ApplyTranscriptionGesture(state.ID, transcript.Gesture{
		Kind:        transcript.GestureSetLength,
		MatchLength: 5,
		HasLength:   true,
	})
	if err != nil {
		t.Fatalf("set length: %v", err)
	}

	// Drop the session so the reopen really goes through the stored row.
	db.forgetTranscriptSessions()

	reopened, err := db.OpenTranscription(state.ID)
	if err != nil {
		t.Fatalf("OpenTranscription: %v", err)
	}
	if got := reopened.Annotated.Document.Header.MatchLength; got != 5 {
		t.Fatalf("reopened match length = %d, want 5", got)
	}
}

func TestCloseTranscription_DeletesTheDraft(t *testing.T) {
	db := newTestDB(t)

	state, err := db.CreateTranscription(transcript.Header{MatchLength: 7})
	if err != nil {
		t.Fatalf("CreateTranscription: %v", err)
	}
	if err := db.CloseTranscription(state.ID); err != nil {
		t.Fatalf("CloseTranscription: %v", err)
	}

	list, err := db.ListTranscriptions()
	if err != nil {
		t.Fatalf("ListTranscriptions: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("got %d drafts after closing, want 0", len(list))
	}
	if _, err := db.OpenTranscription(state.ID); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("reopening a closed draft: %v, want ErrNotFound", err)
	}
}

// ── durability (T1.8, fonctionnel.md §3, ADR-0045 rule 1) ────────────
//
// What these hold is the promise the transcription table exists for: a match
// typed in for an hour survives the application being killed, at the cost of
// exactly the two things a crash is allowed to take — the dice half typed and
// the undo stack.

// applyGesture applies one gesture to a draft and fails the test if it is
// refused: a scenario that carried on after a refused gesture would check a
// document that never existed.
func applyGesture(t *testing.T, db *Database, id int64, g transcript.Gesture) *TranscriptionState {
	t.Helper()
	state, err := db.ApplyTranscriptionGesture(id, g)
	if err != nil {
		t.Fatalf("gesture %s: %v", g.Kind, err)
	}
	return state
}

// typeOpening records the opening roll: 6 for player 1, 3 for player 2, so
// player 1 is on roll with that same 6-3.
func typeOpening(t *testing.T, db *Database, id int64) {
	t.Helper()
	applyGesture(t, db, id, transcript.Gesture{Kind: transcript.GestureEnterDie, Die: 6})
	applyGesture(t, db, id, transcript.Gesture{Kind: transcript.GestureEnterDie, Die: 3})
	applyGesture(t, db, id, transcript.Gesture{Kind: transcript.GestureValidate})
}

// typeChecker records one checker play: the roll, the first candidate the
// engine lists for it, and the validation that turns it into an Action. Dice
// of 0 keep the roll already proposed — which is what the play right after an
// opening does, since the winner plays the dice that were just rolled.
func typeChecker(t *testing.T, db *Database, id int64, dice ...int) {
	t.Helper()
	for _, die := range dice {
		applyGesture(t, db, id, transcript.Gesture{Kind: transcript.GestureEnterDie, Die: die})
	}
	applyGesture(t, db, id, transcript.Gesture{Kind: transcript.GestureSelectCandidate, Candidate: 0})
	applyGesture(t, db, id, transcript.Gesture{Kind: transcript.GestureValidate})
}

// readTranscriptionRow reads a draft's row as it stands on disk, document
// column included — what the panel never sees and what durability is about.
func readTranscriptionRow(t *testing.T, db *Database, id int64) *storage.Transcription {
	t.Helper()
	row, err := db.store.Transcriptions().Get(context.Background(), "", id)
	if err != nil {
		t.Fatalf("reading transcription %d: %v", id, err)
	}
	return row
}

// A draft is on disk after every Action, so an application killed mid-typing
// loses the dice being entered and nothing else (flux 21).
//
// The crash is simulated the way a power cut leaves a library: the files are
// copied as they stand — the database and its write-ahead log, never the
// volatile -shm, which SQLite rebuilds — while the original handle is still
// open and has never been closed, checkpointed or optimised. Whatever the
// copy answers, no clean shutdown put it there.
func TestTranscription_SurvivesAnAbruptStop(t *testing.T) {
	dir := tempDir(t)
	live := NewDatabase()
	livePath := filepath.Join(dir, "live.db")
	if err := live.SetupDatabase(livePath); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	closeOnCleanup(t, live)

	state, err := live.CreateTranscription(transcript.Header{
		MatchLength: 7,
		Player1:     "Kévin",
		Player2:     "Alice",
	})
	if err != nil {
		t.Fatalf("CreateTranscription: %v", err)
	}
	id := state.ID

	typeOpening(t, live, id)
	typeChecker(t, live, id)       // player 1 plays the opening's 6-3
	typeChecker(t, live, id, 5, 4) // player 2
	typeChecker(t, live, id, 3, 2) // player 1
	// The crash catches the user in the middle of a correction: the Cursor
	// walked back over the last play and a first die of the new roll is in.
	// Neither is durable, and neither may take an Action down with it.
	applyGesture(t, live, id, transcript.Gesture{Kind: transcript.GestureCursorBack})
	last := applyGesture(t, live, id, transcript.Gesture{Kind: transcript.GestureEnterDie, Die: 6})
	want := last.Annotated.Document.Actions
	if len(want) != 4 {
		t.Fatalf("the scenario recorded %d actions, want 4", len(want))
	}

	crashedPath := filepath.Join(dir, "crashed.db")
	copyCrashedLibrary(t, livePath, crashedPath)

	crashed := NewDatabase()
	if err := crashed.OpenDatabase(crashedPath); err != nil {
		t.Fatalf("reopening after the crash: %v", err)
	}
	closeOnCleanup(t, crashed)

	list, err := crashed.ListTranscriptions()
	if err != nil {
		t.Fatalf("ListTranscriptions: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("got %d drafts after the crash, want 1", len(list))
	}
	if list[0].ID != id || list[0].Label != "Kévin vs Alice" || list[0].ActionCount != len(want) {
		t.Fatalf("summary after the crash = %+v", list[0])
	}

	reopened, err := crashed.OpenTranscription(id)
	if err != nil {
		t.Fatalf("OpenTranscription: %v", err)
	}
	doc := reopened.Annotated.Document
	if !reflect.DeepEqual(doc.Actions, want) {
		t.Fatalf("actions after the crash =\n%+v\nwant\n%+v", doc.Actions, want)
	}
	if doc.Header.MatchLength != 7 || doc.Header.Player1 != "Kévin" || doc.Header.Player2 != "Alice" {
		t.Fatalf("header after the crash = %+v", doc.Header)
	}
	// The two things a crash is allowed to take: the roll being typed and the
	// undo stack. The Cursor lands at the end of the document (§3).
	if doc.Entry != nil {
		t.Fatalf("the dice being typed survived the crash: %+v", doc.Entry)
	}
	if doc.Cursor != len(doc.Actions) {
		t.Fatalf("cursor after the crash = %d, want %d (the end of the document)", doc.Cursor, len(doc.Actions))
	}
	if len(reopened.Annotated.Actions) != len(want) {
		t.Fatalf("the reopened draft replayed %d actions, want %d", len(reopened.Annotated.Actions), len(want))
	}
}

// copyCrashedLibrary copies a live SQLite library — the database file and its
// write-ahead log — to dst, leaving the source open. The -shm is deliberately
// left behind: it is shared memory, rebuilt from the log by whoever opens the
// copy, which is exactly what recovery after a crash does.
func copyCrashedLibrary(t *testing.T, src, dst string) {
	t.Helper()
	for _, suffix := range []string{"", "-wal"} {
		data, err := os.ReadFile(src + suffix)
		if errors.Is(err, os.ErrNotExist) && suffix != "" {
			continue
		}
		if err != nil {
			t.Fatalf("copying %s: %v", src+suffix, err)
		}
		if err := os.WriteFile(dst+suffix, data, 0o600); err != nil {
			t.Fatalf("writing %s: %v", dst+suffix, err)
		}
	}
}

// A gesture that changes an Action writes the row; a gesture that only moves
// the Cursor, fills a die or picks a candidate writes nothing (§3).
//
// The proof is a sentinel written into the document column behind the
// engine's back: a gesture that does not write leaves it there, and a gesture
// that writes replaces it with the document it made.
func TestApplyTranscriptionGesture_OnlyAnActionCostsAWrite(t *testing.T) {
	db := newTestDB(t)

	state, err := db.CreateTranscription(transcript.Header{MatchLength: 7})
	if err != nil {
		t.Fatalf("CreateTranscription: %v", err)
	}
	id := state.ID
	typeOpening(t, db, id)
	typeChecker(t, db, id) // two Actions on disk

	const sentinel = "this row was not rewritten"
	row := readTranscriptionRow(t, db, id)
	row.Document = sentinel
	if _, err := db.store.Transcriptions().Save(context.Background(), "", row); err != nil {
		t.Fatalf("planting the sentinel: %v", err)
	}

	quiet := []transcript.Gesture{
		{Kind: transcript.GestureCursorBack},
		{Kind: transcript.GestureCursorForward},
		{Kind: transcript.GestureEnterDie, Die: 5},
		{Kind: transcript.GestureEnterDie, Die: 4},
		{Kind: transcript.GestureSelectCandidate, Candidate: 0},
	}
	for _, g := range quiet {
		applyGesture(t, db, id, g)
		if got := readTranscriptionRow(t, db, id).Document; got != sentinel {
			t.Fatalf("gesture %s rewrote the row; it changes no Action", g.Kind)
		}
	}

	after := applyGesture(t, db, id, transcript.Gesture{Kind: transcript.GestureValidate})
	if n := len(after.Annotated.Document.Actions); n != 3 {
		t.Fatalf("the validation recorded %d actions, want 3", n)
	}
	stored := readTranscriptionRow(t, db, id)
	if stored.Document == sentinel {
		t.Fatal("the validated Action was never written to the row")
	}
	doc, err := decodeTranscription(stored)
	if err != nil {
		t.Fatalf("decoding the rewritten row: %v", err)
	}
	if !reflect.DeepEqual(doc.Actions, after.Annotated.Document.Actions) {
		t.Fatalf("the row holds\n%+v\nbut the draft is\n%+v", doc.Actions, after.Annotated.Document.Actions)
	}
	// The row states the Cursor a resumption will use, not the one the last
	// gesture happened to leave.
	if doc.Cursor != len(doc.Actions) {
		t.Fatalf("stored cursor = %d, want %d", doc.Cursor, len(doc.Actions))
	}
}
