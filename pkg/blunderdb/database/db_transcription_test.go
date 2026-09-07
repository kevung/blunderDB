package database

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
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

// The opening, end to end through the plumbing (fonctionnel.md §6 flux 2): two
// dice, the stronger one starts, and the roll it won with is the roll of the
// first checker Action — the user does not type it again.
func TestApplyTranscriptionGesture_Opening(t *testing.T) {
	db := newTestDB(t)

	state, err := db.CreateTranscription(transcript.Header{MatchLength: 7})
	if err != nil {
		t.Fatalf("CreateTranscription: %v", err)
	}

	for _, g := range []transcript.Gesture{
		{Kind: transcript.GestureEnterDie, Die: 2},
		{Kind: transcript.GestureEnterDie, Die: 5},
		{Kind: transcript.GestureValidate},
	} {
		if state, err = db.ApplyTranscriptionGesture(state.ID, g); err != nil {
			t.Fatalf("gesture %s: %v", g.Kind, err)
		}
	}

	actions := state.Annotated.Document.Actions
	if len(actions) != 1 || actions[0].Kind != transcript.KindOpening {
		t.Fatalf("actions = %+v, want one opening", actions)
	}
	if actions[0].Dice != [2]int{2, 5} {
		t.Fatalf("opening dice = %v, want [2 5]", actions[0].Dice)
	}
	// Player 2 rolled the higher die, so player 2 starts and a checker play is
	// what the document expects next.
	if actions[0].Side != 1 {
		t.Fatalf("side = %d, want player 2", actions[0].Side)
	}
	if state.Annotated.Next.Expects != transcript.KindChecker || state.Annotated.Next.Side != 1 {
		t.Fatalf("next = %+v", state.Annotated.Next)
	}
	// And the candidates offered are those of the opening roll, not of a roll
	// the user would have to type again.
	cands := transcript.Candidates(state.Annotated.Document)
	if len(cands) == 0 {
		t.Fatal("no candidate for the opening roll")
	}
}

// A tie is kept, produces neither Move nor Position, and another opening is
// expected — the panel's "relance".
func TestApplyTranscriptionGesture_OpeningTie(t *testing.T) {
	db := newTestDB(t)

	state, err := db.CreateTranscription(transcript.Header{MatchLength: 7})
	if err != nil {
		t.Fatalf("CreateTranscription: %v", err)
	}
	for _, g := range []transcript.Gesture{
		{Kind: transcript.GestureEnterDie, Die: 4},
		{Kind: transcript.GestureEnterDie, Die: 4},
		{Kind: transcript.GestureValidate},
	} {
		if state, err = db.ApplyTranscriptionGesture(state.ID, g); err != nil {
			t.Fatalf("gesture %s: %v", g.Kind, err)
		}
	}

	if len(state.Annotated.Document.Actions) != 1 {
		t.Fatalf("the tie was not kept: %+v", state.Annotated.Document.Actions)
	}
	if state.Annotated.Next.Expects != transcript.KindOpening {
		t.Fatalf("next = %+v, want another opening", state.Annotated.Next)
	}
}

// The money draft the form offers: a length of 0 with the session's rules, which
// are flags of every Position and never Actions (ADR-0028, ADR-0044).
func TestCreateTranscription_MoneyRulesAreKept(t *testing.T) {
	db := newTestDB(t)

	state, err := db.CreateTranscription(transcript.Header{MatchLength: 0, Jacoby: false, Beaver: true})
	if err != nil {
		t.Fatalf("CreateTranscription: %v", err)
	}
	header := state.Annotated.Document.Header
	if header.MatchLength != 0 || header.Jacoby || !header.Beaver {
		t.Fatalf("header = %+v, want a money draft without Jacoby and with the beaver", header)
	}
}

// TestTranscriptionMAT_IsTheExportRenderer: the panel's ".mat text" pane is a
// view OF THE EXPORT, not a second opinion about it. What is checked here is
// therefore the join and nothing else — the open draft goes through
// transcript.MatchParts and ingest.RenderMAT, so the pane shows the two
// columns of a score sheet with the header the file will carry, and no row is
// written on the way.
func TestTranscriptionMAT_IsTheExportRenderer(t *testing.T) {
	db := newTestDB(t)

	state, err := db.CreateTranscription(transcript.Header{
		MatchLength: 7,
		Player1:     "Kévin",
		Player2:     "Alice",
	})
	if err != nil {
		t.Fatalf("CreateTranscription: %v", err)
	}
	typeOpening(t, db, state.ID)
	typeChecker(t, db, state.ID)
	typeChecker(t, db, state.ID, 5, 2)

	text, err := db.TranscriptionMAT(state.ID)
	if err != nil {
		t.Fatalf("TranscriptionMAT: %v", err)
	}

	doc := db.transcriptSessions[state.ID].Doc
	if want := ingest.RenderMAT(transcript.MatchParts(doc)); text != want {
		t.Fatalf("the pane's text is not the exporter's:\n--- got ---\n%s\n--- want ---\n%s", text, want)
	}
	for _, want := range []string{"7 point match", " Game 1", "Kévin : 0", "Alice : 0", "63:", "52:"} {
		if !strings.Contains(text, want) {
			t.Fatalf("the .mat text does not carry %q:\n%s", want, text)
		}
	}
}

// Undo and redo are the two gestures that do not go through transcript.Apply:
// the stack is state and it lives in the session's Editor. What this file owes
// them is the plumbing — that they reach the stack, that they report both of
// its sides, and that taking a gesture back leaves the ROW without it, since a
// crash must not resurrect what the user undid (fonctionnel.md §3).
func TestApplyTranscriptionGesture_UndoAndRedo(t *testing.T) {
	db := newTestDB(t)

	state, err := db.CreateTranscription(transcript.Header{MatchLength: 7})
	if err != nil {
		t.Fatalf("CreateTranscription: %v", err)
	}
	id := state.ID
	if state.CanUndo || state.CanRedo {
		t.Fatalf("a fresh draft reports undo=%v redo=%v", state.CanUndo, state.CanRedo)
	}
	typeOpening(t, db, id)
	typeChecker(t, db, id)
	typeChecker(t, db, id, 5, 4)

	before := applyGesture(t, db, id, transcript.Gesture{Kind: transcript.GestureCursorBack})
	want := len(before.Annotated.Document.Actions)
	if !before.CanUndo {
		t.Fatal("a draft with gestures behind it reports nothing to undo")
	}

	undone := applyGesture(t, db, id, transcript.Gesture{Kind: transcript.GestureUndo})
	if !undone.CanRedo {
		t.Error("an undone gesture is not reported as redoable")
	}
	redone := applyGesture(t, db, id, transcript.Gesture{Kind: transcript.GestureRedo})
	if redone.CanRedo {
		t.Error("a redone gesture is still reported as redoable")
	}

	// Undoing the last validation must leave the row one Action short: the
	// draft on disk is the draft the user has, never one gesture ahead of it.
	for range 3 {
		applyGesture(t, db, id, transcript.Gesture{Kind: transcript.GestureUndo})
	}
	stored, err := decodeTranscription(readTranscriptionRow(t, db, id))
	if err != nil {
		t.Fatalf("decoding the row: %v", err)
	}
	if len(stored.Actions) >= want {
		t.Fatalf("the row still holds %d actions after three undos, want fewer than %d", len(stored.Actions), want)
	}
}

// A stack with nothing on it is a keystroke that does nothing — never an error
// the panel has to show.
func TestApplyTranscriptionGesture_UndoOnAnEmptyStack(t *testing.T) {
	db := newTestDB(t)
	state, err := db.CreateTranscription(transcript.Header{MatchLength: 7})
	if err != nil {
		t.Fatalf("CreateTranscription: %v", err)
	}
	for _, kind := range []transcript.GestureKind{transcript.GestureUndo, transcript.GestureRedo} {
		if _, err := db.ApplyTranscriptionGesture(state.ID, transcript.Gesture{Kind: kind}); err != nil {
			t.Errorf("%s on an empty stack: %v", kind, err)
		}
	}
}

// After a gesture, the Cursor lands on the first Inconsistency the gesture left
// behind — and the SESSION agrees with what was handed back, so the next `h`
// counts from the cell the user is looking at (fonctionnel.md §1.4).
func TestApplyTranscriptionGesture_CursorLandsOnTheInconsistency(t *testing.T) {
	db := newTestDB(t)
	state, err := db.CreateTranscription(transcript.Header{MatchLength: 7})
	if err != nil {
		t.Fatalf("CreateTranscription: %v", err)
	}
	id := state.ID
	typeOpening(t, db, id)
	typeChecker(t, db, id)
	typeChecker(t, db, id, 5, 4)
	typeChecker(t, db, id, 4, 2)

	// Walk back to the middle and give that Action to the other camp: the
	// double turn it makes is behind the Cursor, which the gesture leaves put.
	applyGesture(t, db, id, transcript.Gesture{Kind: transcript.GestureCursorBack})
	applyGesture(t, db, id, transcript.Gesture{Kind: transcript.GestureCursorBack})
	flipped := applyGesture(t, db, id, transcript.Gesture{Kind: transcript.GestureFlipSide})

	if !flipped.Annotated.Inconsistent() {
		t.Fatal("changing a camp made no inconsistency")
	}
	at := flipped.Annotated.Cursor
	if at < 0 || at >= len(flipped.Annotated.Actions) {
		t.Fatalf("cursor = %d, outside the document", at)
	}
	if len(flipped.Annotated.Actions[at].Inconsistencies) == 0 {
		t.Fatalf("the cursor landed on action %d, which carries no inconsistency", at)
	}
	// And the session was moved there too, not only the answer.
	if got := flipped.Annotated.Document.Cursor; got != at {
		t.Fatalf("the document's cursor is %d while the answer says %d", got, at)
	}
	// Nothing was deleted or repaired on the way (ADR-0044).
	if n := len(flipped.Annotated.Document.Actions); n != 4 {
		t.Fatalf("actions = %d, want 4", n)
	}
}

// Reopening a draft puts the Cursor at the end, Inconsistencies or not: a
// resumption continues after the last written Action (fonctionnel.md §3), and
// an Inconsistency the user read and chose to keep must not drag them back to
// it at every open.
func TestOpenTranscription_DoesNotJumpToAnInconsistency(t *testing.T) {
	db := newTestDB(t)
	state, err := db.CreateTranscription(transcript.Header{MatchLength: 7})
	if err != nil {
		t.Fatalf("CreateTranscription: %v", err)
	}
	id := state.ID
	typeOpening(t, db, id)
	typeChecker(t, db, id)
	typeChecker(t, db, id, 5, 4)
	applyGesture(t, db, id, transcript.Gesture{Kind: transcript.GestureCursorBack})
	applyGesture(t, db, id, transcript.Gesture{Kind: transcript.GestureFlipSide})

	db.forgetTranscriptSessions()
	reopened, err := db.OpenTranscription(id)
	if err != nil {
		t.Fatalf("OpenTranscription: %v", err)
	}
	if !reopened.Annotated.Inconsistent() {
		t.Fatal("the fixture lost its inconsistency across the reopen")
	}
	if got, want := reopened.Annotated.Cursor, len(reopened.Annotated.Document.Actions); got != want {
		t.Fatalf("reopened cursor = %d, want %d (the end of the document)", got, want)
	}
	if reopened.CanUndo || reopened.CanRedo {
		t.Error("a reopened draft carries an undo stack; it is in memory and a stop loses it")
	}
}

// TestCreateTranscription_DatesAndCreditsTheDraft holds the two defaults of
// T3.1 that only the library can state: today's date, and the transcriber —
// the library's own `user` metadata, which is who is sitting in front of the
// board. Both are DEFAULTS: the metadata pane overwrites them, and overwriting
// the transcriber must not touch the library's setting.
func TestCreateTranscription_DatesAndCreditsTheDraft(t *testing.T) {
	db := newTestDB(t)
	if err := db.SaveMetadata(map[string]string{"user": "  Kévin Unger  "}); err != nil {
		t.Fatalf("SaveMetadata: %v", err)
	}

	state, err := db.CreateTranscription(transcript.Header{MatchLength: 7})
	if err != nil {
		t.Fatalf("CreateTranscription: %v", err)
	}
	header := state.Annotated.Document.Header
	if header.Transcriber != "Kévin Unger" {
		t.Errorf("transcriber = %q, want the library's user", header.Transcriber)
	}
	if header.Date.IsZero() || time.Since(header.Date) > time.Hour {
		t.Errorf("date = %v, want today", header.Date)
	}

	// A header that states them keeps them: the default fills a silence, it
	// never overwrites an answer.
	when := time.Date(2019, 3, 2, 0, 0, 0, 0, time.UTC)
	stated, err := db.CreateTranscription(transcript.Header{MatchLength: 7, Date: when, Transcriber: "Alice"})
	if err != nil {
		t.Fatalf("CreateTranscription: %v", err)
	}
	if h := stated.Annotated.Document.Header; h.Transcriber != "Alice" || !h.Date.Equal(when) {
		t.Errorf("a stated header was overwritten: %+v", h)
	}

	// And the pane's own gesture leaves the library's user alone.
	if _, err := db.ApplyTranscriptionGesture(state.ID, transcript.Gesture{
		Kind:   transcript.GestureSetHeader,
		Header: transcript.Header{Transcriber: "Bob"},
	}); err != nil {
		t.Fatalf("set_header: %v", err)
	}
	meta, err := db.LoadMetadata()
	if err != nil {
		t.Fatalf("LoadMetadata: %v", err)
	}
	if strings.TrimSpace(meta["user"]) != "Kévin Unger" {
		t.Errorf("the library's user became %q: a draft credits itself, it does not rename the library", meta["user"])
	}
}

// TestSaveTranscription_AttachesTheTournament is T3.1's "rattachement au
// moment de l'enregistrement": the tournament named in the draft's header is
// the tournament the Match belongs to once it is written, and clearing it
// detaches the Match at the next save.
func TestSaveTranscription_AttachesTheTournament(t *testing.T) {
	db := newTestDB(t)
	id := matDraft(t, db, filepath.Join("testdata", "test.mat"))

	tournamentID, err := db.CreateTournament("Open de Paris", "2026-09-07", "Paris")
	if err != nil {
		t.Fatalf("CreateTournament: %v", err)
	}
	if _, err := db.ApplyTranscriptionGesture(id, transcript.Gesture{
		Kind:   transcript.GestureSetHeader,
		Header: transcript.Header{Player1: "Alice", Player2: "Bob", TournamentID: &tournamentID},
	}); err != nil {
		t.Fatalf("set_header: %v", err)
	}

	saved, err := db.SaveTranscriptionAsMatch(id)
	if err != nil {
		t.Fatalf("SaveTranscriptionAsMatch: %v", err)
	}
	tour, err := db.GetMatchTournament(saved.MatchID)
	if err != nil {
		t.Fatalf("GetMatchTournament: %v", err)
	}
	if tour == nil || tour.ID != tournamentID {
		t.Fatalf("match %d belongs to %v, want tournament %d", saved.MatchID, tour, tournamentID)
	}

	// The names travelled with it, and they are the ones the pane wrote.
	matches, err := db.GetTournamentMatches(tournamentID)
	if err != nil {
		t.Fatalf("GetTournamentMatches: %v", err)
	}
	if len(matches) != 1 || matches[0].Player1Name != "Alice" || matches[0].Player2Name != "Bob" {
		t.Fatalf("tournament matches = %+v", matches)
	}

	// Clearing the field detaches at the next save: the attachment is decided
	// by the draft, at every save, and not once and for all.
	if _, err := db.ApplyTranscriptionGesture(id, transcript.Gesture{
		Kind:   transcript.GestureSetHeader,
		Header: transcript.Header{Player1: "Alice", Player2: "Bob"},
	}); err != nil {
		t.Fatalf("set_header: %v", err)
	}
	again, err := db.SaveTranscriptionAsMatch(id)
	if err != nil {
		t.Fatalf("second SaveTranscriptionAsMatch: %v", err)
	}
	if again.MatchID != saved.MatchID {
		t.Fatalf("the second save produced match %d, want %d", again.MatchID, saved.MatchID)
	}
	if tour, err := db.GetMatchTournament(again.MatchID); err != nil {
		t.Fatalf("GetMatchTournament: %v", err)
	} else if tour != nil {
		t.Errorf("the match is still in tournament %d after the draft let it go", tour.ID)
	}
}
