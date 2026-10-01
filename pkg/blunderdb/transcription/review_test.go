package transcription

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// A Finish refused for a stale revision reloads the session from the row, as
// a refused gesture does: finishing again at the revision the conflict named
// goes through instead of failing forever against the old one.
func TestFinishConflictReloadsTheSession(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	a, b := New(store, Options{}), New(store, Options{})

	st, err := a.Create(ctx, "1", transcript.Header{MatchLength: 7})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	st = play(t, a, "1", st, checkerAction(6, 3))
	sb, err := b.Open(ctx, "1", st.ID)
	if err != nil {
		t.Fatalf("Open on b: %v", err)
	}
	play(t, a, "1", st, checkerAction(5, 2))

	_, err = b.Finish(ctx, "1", sb.ID, Expect{Session: sb.SessionID, Revision: sb.Revision})
	var stale *StaleError
	if !errors.As(err, &stale) {
		t.Fatalf("Finish behind a: got %v, want a StaleError", err)
	}
	res, err := b.Finish(ctx, "1", sb.ID, Expect{Session: sb.SessionID, Revision: stale.Revision})
	if err != nil {
		t.Fatalf("Finish at the revision the conflict named: %v", err)
	}
	if res.Moves != 2 {
		t.Fatalf("finished with %d moves, want 2 (a's both)", res.Moves)
	}
}

// Only a change of the durable document advances the revision: a die typed
// in the Entry or a Cursor moved writes nothing, so another tab's moving
// Cursor cannot make this one's next gesture a conflict.
func TestCursorAndEntryDoNotAdvanceTheRevision(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	svc := New(store, Options{})
	st, err := svc.Create(ctx, "1", transcript.Header{MatchLength: 7})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	st = play(t, svc, "1", st, checkerAction(6, 3))
	at := st.Revision

	for _, g := range []transcript.Gesture{
		{Kind: transcript.GestureCursorBack},
		{Kind: transcript.GestureCursorForward},
		{Kind: transcript.GestureEnterDie, Die: 4},
	} {
		next, err := svc.Apply(ctx, "1", st.ID, Expect{Session: st.SessionID, Revision: at}, g)
		if err != nil {
			t.Fatalf("Apply %s: %v", g.Kind, err)
		}
		if next.Revision != at {
			t.Fatalf("Apply %s moved the revision to %d, want %d", g.Kind, next.Revision, at)
		}
		row, err := store.Transcriptions().Get(ctx, "1", st.ID)
		if err != nil || row.Revision != at {
			t.Fatalf("Apply %s wrote the row: %+v, %v", g.Kind, row, err)
		}
	}
}

// The conflict carries the fresh state — the draft, its revision, the
// session and its Cursor — so the client redraws without a second read.
func TestConflictCarriesTheFreshState(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	a, b := New(store, Options{}), New(store, Options{})
	st, err := a.Create(ctx, "1", transcript.Header{MatchLength: 7})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	sb, err := b.Open(ctx, "1", st.ID)
	if err != nil {
		t.Fatalf("Open on b: %v", err)
	}
	play(t, a, "1", st, checkerAction(6, 3))

	sb = play(t, b, "1", sb, checkerAction(4, 1)[:3])
	_, err = b.Apply(ctx, "1", sb.ID, Expect{Session: sb.SessionID, Revision: sb.Revision}, transcript.Gesture{Kind: transcript.GestureValidate})
	var stale *StaleError
	if !errors.As(err, &stale) {
		t.Fatalf("b behind a: got %v, want a StaleError", err)
	}
	fresh := stale.State
	if fresh == nil {
		t.Fatal("the conflict must carry the fresh state")
	}
	if fresh.Revision != stale.Revision || fresh.SessionID != sb.SessionID {
		t.Fatalf("fresh state at revision %d in session %q, want %d in %q", fresh.Revision, fresh.SessionID, stale.Revision, sb.SessionID)
	}
	if n := len(fresh.Annotated.Document.Actions); n != 1 {
		t.Fatalf("fresh state holds %d actions, want 1 (a's)", n)
	}
	if fresh.Annotated.Document.Cursor != 1 || fresh.CanUndo {
		t.Fatalf("fresh state: cursor %d, canUndo %v; want 1, false", fresh.Annotated.Document.Cursor, fresh.CanUndo)
	}
	details := stale.ErrorDetails()
	if details["revision"] != stale.Revision || details["state"] != fresh {
		t.Fatalf("error details %v must name the revision and the state", details)
	}
}
