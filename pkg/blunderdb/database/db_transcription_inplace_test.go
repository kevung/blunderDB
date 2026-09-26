package database

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// TestTranscriptionCorrectAPassIntoATake walks the scenario the in-place
// correction exists for, through the BINDING the panel calls — the Editor, the
// session, the Replay that moves the Cursor onto the first Inconsistency — and
// not only through the pure gestures of pkg/blunderdb/transcript.
//
// A pass is typed for a take and the next game started; walking back onto the
// pass and pressing `t` must REPLACE it (not insert before it), the game goes
// on, and the following game's typing stays where it is.
func TestTranscriptionCorrectAPassIntoATake(t *testing.T) {
	db := newTestDB(t)
	state, err := db.CreateTranscription(transcript.Header{MatchLength: 7, Player1: "Kévin", Player2: "Alice"})
	if err != nil {
		t.Fatalf("CreateTranscription: %v", err)
	}
	id := state.ID

	apply := func(g transcript.Gesture) *TranscriptionState {
		t.Helper()
		out, err := db.ApplyTranscriptionGesture(id, g)
		if err != nil {
			t.Fatalf("%s: %v", g.Kind, err)
		}
		return out
	}
	die := func(n int) transcript.Gesture { return transcript.Gesture{Kind: transcript.GestureEnterDie, Die: n} }
	play := func(a, b int) {
		t.Helper()
		apply(die(a))
		apply(die(b))
		apply(transcript.Gesture{Kind: transcript.GestureSelectCandidate, Candidate: 0})
		apply(transcript.Gesture{Kind: transcript.GestureValidate})
	}

	// Game 1: the opening, one play each, a double and a pass.
	apply(die(6))
	apply(die(3))
	apply(transcript.Gesture{Kind: transcript.GestureValidate})
	play(6, 3)
	play(5, 2)
	apply(transcript.Gesture{Kind: transcript.GestureDouble})
	state = apply(transcript.Gesture{Kind: transcript.GesturePass})

	passAt := len(state.Annotated.Document.Actions) - 1
	if state.Annotated.Document.Actions[passAt].Kind != transcript.KindPass {
		t.Fatalf("action %d = %+v, want the pass", passAt, state.Annotated.Document.Actions[passAt])
	}

	// Game 2 is started before the mistake is seen.
	apply(die(4))
	apply(die(1))
	apply(transcript.Gesture{Kind: transcript.GestureValidate})
	state = apply(transcript.Gesture{Kind: transcript.GestureSelectCandidate, Candidate: 0})
	state = apply(transcript.Gesture{Kind: transcript.GestureValidate})

	count := len(state.Annotated.Document.Actions)
	tail := append([]transcript.Action(nil), state.Annotated.Document.Actions[passAt+1:]...)
	if len(tail) < 2 {
		t.Fatalf("the second game holds %d actions; the case is not built", len(tail))
	}

	// Back onto the pass, one step at a time — what `h` does, and what a click
	// on the cell does.
	for i := len(state.Annotated.Document.Actions); i > passAt; i-- {
		state = apply(transcript.Gesture{Kind: transcript.GestureCursorBack})
	}
	if state.Annotated.Entry == nil || state.Annotated.Entry.At != passAt || !state.Annotated.Entry.Replacing {
		t.Fatalf("entry = %+v, want the pass held for replacement at %d", state.Annotated.Entry, passAt)
	}

	state = apply(transcript.Gesture{Kind: transcript.GestureTake})

	actions := state.Annotated.Document.Actions
	if len(actions) != count {
		t.Fatalf("actions = %d, want %d — a correction replaces, it does not add", len(actions), count)
	}
	if actions[passAt].Kind != transcript.KindTake {
		t.Fatalf("action %d = %+v, want a take", passAt, actions[passAt])
	}
	for i, want := range tail {
		if got := actions[passAt+1+i]; got.Kind != want.Kind || got.Dice != want.Dice {
			t.Fatalf("action %d = %+v, want the one that was typed of the next game: %+v", passAt+1+i, got, want)
		}
	}
	// The game the pass had WON is nobody's again, which is the whole point: it
	// is left open by the opening that follows it — the one the rest of the
	// game is now typed in front of — and it is worth no points to anybody.
	if g := state.Annotated.Games[0]; g.Winner != -1 || g.PointsWon != 0 {
		t.Errorf("game 1 = %+v, want it won by nobody — the take was not replayed", g)
	}
	if state.Annotated.Score != [2]int{0, 0} {
		t.Errorf("score = %v, want 0–0: the pass no longer gives a point away", state.Annotated.Score)
	}
}

// TestTranscriptionOpeningRetypedDecidesWhoStarts holds the other half of the
// same session: the opening of a game is where "the big die first is player 1"
// is said, and saying it again must decide again — through the binding, where
// the entry is loaded by the Cursor and the validation replaces in place.
func TestTranscriptionOpeningRetypedDecidesWhoStarts(t *testing.T) {
	db := newTestDB(t)
	state, err := db.CreateTranscription(transcript.Header{MatchLength: 7})
	if err != nil {
		t.Fatalf("CreateTranscription: %v", err)
	}
	id := state.ID

	apply := func(g transcript.Gesture) *TranscriptionState {
		t.Helper()
		out, err := db.ApplyTranscriptionGesture(id, g)
		if err != nil {
			t.Fatalf("%s: %v", g.Kind, err)
		}
		return out
	}
	die := func(n int) transcript.Gesture { return transcript.Gesture{Kind: transcript.GestureEnterDie, Die: n} }

	// 6 then 3: player 1 rolled the higher die and starts.
	apply(die(6))
	apply(die(3))
	state = apply(transcript.Gesture{Kind: transcript.GestureValidate})
	if got := state.Annotated.Document.Actions[0].Side; got != 0 {
		t.Fatalf("side = %d, want player 1", got)
	}

	// One play, then back onto the opening.
	apply(die(6))
	apply(die(3))
	apply(transcript.Gesture{Kind: transcript.GestureSelectCandidate, Candidate: 0})
	state = apply(transcript.Gesture{Kind: transcript.GestureValidate})
	for i := len(state.Annotated.Document.Actions); i > 0; i-- {
		state = apply(transcript.Gesture{Kind: transcript.GestureCursorBack})
	}
	if e := state.Annotated.Entry; e == nil || e.Kind != transcript.KindOpening {
		t.Fatalf("entry = %+v, want the opening slot named as one so the panel types an opening", e)
	}

	// 3 then 6: the higher die is player 2's now, and player 2 starts.
	apply(die(3))
	apply(die(6))
	state = apply(transcript.Gesture{Kind: transcript.GestureValidate})

	opening := state.Annotated.Document.Actions[0]
	if opening.Kind != transcript.KindOpening || opening.Dice != [2]int{3, 6} {
		t.Fatalf("opening = %+v, want the roll retyped", opening)
	}
	if opening.Side != 1 {
		t.Errorf("side = %d, want player 2 — the small die first gives the turn to the top", opening.Side)
	}
	// And the play made with that roll follows it: its camp was proposed by the
	// opening, never chosen by the user (fonctionnel.md §1.2).
	if got := state.Annotated.Document.Actions[1].Side; got != 1 {
		t.Errorf("the first play stayed with player %d; it must follow the opening", got+1)
	}
	if len(state.Annotated.Document.Actions) != 2 {
		t.Errorf("actions = %d, want 2 — the opening was replaced, not inserted", len(state.Annotated.Document.Actions))
	}
}
