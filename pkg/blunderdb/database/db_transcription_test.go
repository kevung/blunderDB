package database

import (
	"errors"
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
