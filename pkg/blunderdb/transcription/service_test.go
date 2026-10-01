package transcription

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

func newStore(t *testing.T) storage.Storage {
	t.Helper()
	ctx := context.Background()
	st, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

// clock is a settable clock for the TTL.
type clock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *clock) Now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.now }
func (c *clock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// checkerAction is the gestures of one checker play: two dice, the first
// candidate, validate.
func checkerAction(d1, d2 int) []transcript.Gesture {
	return []transcript.Gesture{
		{Kind: transcript.GestureEnterDie, Die: d1},
		{Kind: transcript.GestureEnterDie, Die: d2},
		{Kind: transcript.GestureSelectCandidate, Candidate: 0},
		{Kind: transcript.GestureValidate},
	}
}

// play applies gestures in the given session, each naming the revision the
// previous one returned, and returns the last state.
func play(t *testing.T, svc *Service, scope string, st *State, gs []transcript.Gesture) *State {
	t.Helper()
	for _, g := range gs {
		next, err := svc.Apply(context.Background(), scope, st.ID, Expect{Session: st.SessionID, Revision: st.Revision}, g)
		if err != nil {
			t.Fatalf("Apply %s: %v", g.Kind, err)
		}
		if next.Revision <= st.Revision {
			t.Fatalf("Apply %s: revision %d does not advance past %d", g.Kind, next.Revision, st.Revision)
		}
		st = next
	}
	return st
}

// An expired session answers ErrSessionGone; reopening gives a new session
// on the full document, at the revision the row holds, and typing goes on:
// no gesture is lost, only the undo stack.
func TestSessionExpiresThenReopensWithoutLoss(t *testing.T) {
	ctx := context.Background()
	clk := &clock{now: time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)}
	svc := New(newStore(t), Options{TTL: 30 * time.Minute, Now: clk.Now})

	st, err := svc.Create(ctx, "1", transcript.Header{MatchLength: 7})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	st = play(t, svc, "1", st, checkerAction(6, 3))
	if !st.CanUndo {
		t.Fatal("a live session holds its undo stack")
	}

	clk.advance(31 * time.Minute)
	_, err = svc.Apply(ctx, "1", st.ID, Expect{Session: st.SessionID, Revision: st.Revision}, transcript.Gesture{Kind: transcript.GestureEnterDie, Die: 5})
	if !errors.Is(err, ErrSessionGone) {
		t.Fatalf("Apply after the TTL: got %v, want ErrSessionGone", err)
	}

	re, err := svc.Open(ctx, "1", st.ID)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if re.SessionID == st.SessionID || re.SessionID == "" {
		t.Fatalf("reopening must give a new session, got %q (was %q)", re.SessionID, st.SessionID)
	}
	if re.Revision != st.Revision {
		t.Fatalf("reopened at revision %d, want %d", re.Revision, st.Revision)
	}
	if got, want := len(re.Annotated.Document.Actions), len(st.Annotated.Document.Actions); got != want || got != 1 {
		t.Fatalf("reopened with %d actions, want %d (= 1)", got, want)
	}
	if re.CanUndo {
		t.Error("the undo stack does not survive its session")
	}
	re = play(t, svc, "1", re, checkerAction(5, 2))
	if n := len(re.Annotated.Document.Actions); n != 2 {
		t.Fatalf("typing after the reopen: %d actions, want 2", n)
	}
}

// Two gestures naming the same revision: one is recorded, the other is
// refused with the revision it lost to.
func TestConcurrentGesturesOneConflict(t *testing.T) {
	ctx := context.Background()
	svc := New(newStore(t), Options{})
	st, err := svc.Create(ctx, "1", transcript.Header{MatchLength: 7})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	var wg sync.WaitGroup
	errs := make([]error, 2)
	start := make(chan struct{})
	for i := range errs {
		wg.Go(func() {
			<-start
			_, errs[i] = svc.Apply(ctx, "1", st.ID, Expect{Session: st.SessionID, Revision: st.Revision},
				transcript.Gesture{Kind: transcript.GestureEnterDie, Die: 6})
		})
	}
	close(start)
	wg.Wait()

	var stale *StaleError
	ok, conflicts := 0, 0
	for _, err := range errs {
		switch {
		case err == nil:
			ok++
		case errors.As(err, &stale) && errors.Is(err, storage.ErrConflict):
			conflicts++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if ok != 1 || conflicts != 1 {
		t.Fatalf("got %d recorded, %d conflicts; want 1 and 1", ok, conflicts)
	}
	if stale.Revision != st.Revision+1 {
		t.Errorf("the conflict names revision %d, want %d", stale.Revision, st.Revision+1)
	}
}

// Two services on one store are two daemon instances: the second's session
// is behind once the first has written, its write is refused, its session
// is reloaded from the row, and the gesture replayed at the fresh revision
// lands on top of the first's.
func TestTwoInstancesConflictThenReplay(t *testing.T) {
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

	_, err = b.Apply(ctx, "1", sb.ID, Expect{Session: sb.SessionID}, transcript.Gesture{Kind: transcript.GestureEnterDie, Die: 4})
	var stale *StaleError
	if !errors.As(err, &stale) {
		t.Fatalf("b behind a: got %v, want a StaleError", err)
	}
	fresh, err := b.Get(ctx, "1", sb.ID)
	if err != nil || fresh.Revision != stale.Revision {
		t.Fatalf("Get after the conflict: %+v, %v; want revision %d", fresh, err, stale.Revision)
	}
	sb.Revision = stale.Revision
	sb = play(t, b, "1", sb, checkerAction(4, 1))
	if n := len(sb.Annotated.Document.Actions); n != 2 {
		t.Fatalf("after the replay: %d actions, want 2 (a's and b's)", n)
	}
}

// A session belongs to its tenant: another tenant naming it finds nothing.
func TestSessionsAreIsolatedByTenant(t *testing.T) {
	ctx := context.Background()
	svc := New(newStore(t), Options{})
	st, err := svc.Create(ctx, "1", transcript.Header{MatchLength: 7})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	_, err = svc.Apply(ctx, "2", st.ID, Expect{Session: st.SessionID, Revision: st.Revision},
		transcript.Gesture{Kind: transcript.GestureEnterDie, Die: 6})
	if !errors.Is(err, ErrSessionGone) {
		t.Fatalf("tenant 2 naming tenant 1's session: got %v, want ErrSessionGone", err)
	}
	if !svc.WithEditor("1", st.ID, func(*transcript.Editor) {}) {
		t.Fatal("tenant 1's session must survive")
	}
}

// Finish writes the Match and deletes the draft; a stale revision finishes
// nothing; Abandon under a stale revision deletes nothing.
func TestFinishAndAbandonNameTheirRevision(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	svc := New(store, Options{})

	st, err := svc.Create(ctx, "1", transcript.Header{MatchLength: 7, Player1: "Alice", Player2: "Bob"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	st = play(t, svc, "1", st, checkerAction(6, 3))

	if _, err := svc.Finish(ctx, "1", st.ID, Expect{Session: st.SessionID, Revision: st.Revision - 1}); !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("Finish at a stale revision: got %v, want a conflict", err)
	}
	res, err := svc.Finish(ctx, "1", st.ID, Expect{Session: st.SessionID, Revision: st.Revision})
	if err != nil {
		t.Fatalf("Finish: %v", err)
	}
	if _, err := store.Matches().Get(ctx, "1", res.MatchID); err != nil {
		t.Fatalf("the Match must exist after Finish: %v", err)
	}
	if _, err := store.Transcriptions().Get(ctx, "1", st.ID); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("the draft must be gone after Finish: %v", err)
	}
	if svc.WithEditor("1", st.ID, func(*transcript.Editor) {}) {
		t.Error("Finish must release the session")
	}

	other, err := svc.Create(ctx, "1", transcript.Header{MatchLength: 5})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.Abandon(ctx, "1", other.ID, Expect{Revision: other.Revision + 5}); !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("Abandon at a stale revision: got %v, want a conflict", err)
	}
	if err := svc.Abandon(ctx, "1", other.ID, Expect{Revision: other.Revision}); err != nil {
		t.Fatalf("Abandon: %v", err)
	}
	if _, err := store.Transcriptions().Get(ctx, "1", other.ID); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("the draft must be gone after Abandon: %v", err)
	}
}
