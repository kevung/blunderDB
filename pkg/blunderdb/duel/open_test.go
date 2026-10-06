package duel

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// TestTwoOpenDuelsKeepTheirClocks: two Duels of one scope, each with a
// Cadence, played in turn: neither clock stops because the other is played.
func TestTwoOpenDuelsKeepTheirClocks(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	c := newClock()
	svc := clockedService(t, st, c)
	cad := NamedCadences()[0]
	set := func(a, b string) Settings {
		cc := cad
		return Settings{MatchLength: 5, Cadence: &cc, Sides: [2]SideSpec{external(a), external(b)}}
	}
	a, err := svc.Create(ctx, "", set("A", "B"))
	if err != nil {
		t.Fatalf("Create a: %v", err)
	}
	b, err := svc.Create(ctx, "", set("C", "D"))
	if err != nil {
		t.Fatalf("Create b: %v", err)
	}
	a = playAfter(t, svc, c, a, 3, answer(*a.Awaiting))
	b = playAfter(t, svc, c, b, 2, answer(*b.Awaiting))
	if got := a.Actions[0].DecisionMS; !eqMS(got, ms(3000)) {
		t.Errorf("a's first decision took %v, want 3000", show(got))
	}
	if got := b.Actions[0].DecisionMS; !eqMS(got, ms(5000)) {
		t.Errorf("b's first decision took %v, want 5000: its clock ran while a was played", show(got))
	}
	list, err := svc.List(ctx, "")
	if err != nil || len(list) != 2 || !list[0].Open || !list[1].Open {
		t.Errorf("List = %+v, %v; both open", list, err)
	}
}

// blockingSide plays the first legal play, but once armed, its next decision
// waits for release: a Bot that computes for as long as the test wants.
type blockingSide struct {
	mu      sync.Mutex
	armed   bool
	entered chan struct{}
	release chan struct{}
}

func (b *blockingSide) Decide(ctx context.Context, d Decision) (Play, bool, error) {
	b.mu.Lock()
	armed := b.armed
	b.armed = false
	b.mu.Unlock()
	if armed {
		close(b.entered)
		select {
		case <-b.release:
		case <-ctx.Done():
			return Play{}, false, ctx.Err()
		}
	}
	return answer(d), true, nil
}

// TestSlowBotHoldsOnlyItsDuel: while a delegated Side of one Duel computes, a
// Play on another Duel is played at once.
func TestSlowBotHoldsOnlyItsDuel(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	const sideBlock SideKind = "test-block"
	bot := &blockingSide{entered: make(chan struct{}), release: make(chan struct{})}
	svc := New(st, Options{Rand: fixedRand(4), Sides: func(spec SideSpec) (Side, error) {
		if spec.Kind == sideBlock {
			return bot, nil
		}
		return testSides(spec)
	}})
	x, err := svc.Create(ctx, "", Settings{MatchLength: 5, Sides: [2]SideSpec{external("A"), {Kind: sideBlock, Name: "Bot"}}})
	if err != nil {
		t.Fatalf("Create x: %v", err)
	}
	y, err := svc.Create(ctx, "", Settings{MatchLength: 5, Sides: [2]SideSpec{external("C"), external("D")}})
	if err != nil {
		t.Fatalf("Create y: %v", err)
	}

	bot.mu.Lock()
	bot.armed = true
	bot.mu.Unlock()
	// A plays on until the Bot is asked, inside the Play that hands it over.
	done := make(chan error, 1)
	go func() {
		st := x
		for {
			next, err := svc.Play(ctx, "", st.ID, st.Revision, answer(*st.Awaiting))
			if err != nil {
				done <- err
				return
			}
			select {
			case <-bot.entered:
				done <- nil
				return
			default:
			}
			st = next
		}
	}()
	select {
	case <-bot.entered:
	case <-time.After(10 * time.Second):
		t.Fatal("the Bot of x was never asked")
	}

	played := make(chan error, 1)
	go func() {
		_, err := svc.Play(ctx, "", y.ID, y.Revision, answer(*y.Awaiting))
		played <- err
	}()
	select {
	case err := <-played:
		if err != nil {
			t.Errorf("Play on y while x's Bot computes: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Error("a Play on y waited for x's Bot")
	}
	close(bot.release)
	if err := <-done; err != nil {
		t.Errorf("Play on x: %v", err)
	}
}

// TestTwoServicesShareOpenDuels: two Services on one library stand for two
// processes. A Duel opened by one is played by the other; gestures racing on
// one revision leave one played and the others refused as conflicts, never a
// draft written twice; the Duel stays open across a restart, read from its row.
func TestTwoServicesShareOpenDuels(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	one, two := newService(t, st, 6), newService(t, st, 0)
	s, err := one.Create(ctx, "", Settings{MatchLength: 5, Sides: [2]SideSpec{external("A"), external("B")}})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	s, err = two.Play(ctx, "", s.ID, s.Revision, answer(*s.Awaiting))
	if err != nil {
		t.Fatalf("Play from the other Service: %v", err)
	}

	const racers = 8
	errs := make(chan error, racers)
	var wg sync.WaitGroup
	for i := range racers {
		svc := one
		if i%2 == 1 {
			svc = two
		}
		wg.Go(func() {
			_, err := svc.Play(ctx, "", s.ID, s.Revision, answer(*s.Awaiting))
			errs <- err
		})
	}
	wg.Wait()
	close(errs)
	won := 0
	for err := range errs {
		switch {
		case err == nil:
			won++
		case !errors.Is(err, storage.ErrConflict):
			t.Errorf("a racing Play: %v, want ErrConflict", err)
		}
	}
	if won != 1 {
		t.Errorf("%d racing Plays on one revision were played, want 1", won)
	}
	row, err := st.Duels().Get(ctx, "", s.ID)
	if err != nil || row.Revision != s.Revision+1 {
		t.Errorf("draft at revision %v (%v), want %d: written once", row, err, s.Revision+1)
	}

	restarted := newService(t, st, 0)
	list, err := restarted.List(ctx, "")
	if err != nil || len(list) != 1 || !list[0].Open {
		t.Errorf("List after a restart = %+v, %v; the Duel still open", list, err)
	}
	if err := restarted.Suspend(ctx, "", s.ID, 0); err != nil {
		t.Fatalf("Suspend: %v", err)
	}
	if list, err := one.List(ctx, ""); err != nil || len(list) != 1 || list[0].Open {
		t.Errorf("List from the first Service = %+v, %v; suspended by the other", list, err)
	}
}
