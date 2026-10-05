package duel

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// fakeClock is the Arbiter's now, moved by hand.
type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time       { return c.t }
func (c *fakeClock) wait(d time.Duration) { c.t = c.t.Add(d) }
func (c *fakeClock) waitS(seconds int)    { c.wait(time.Duration(seconds) * time.Second) }
func newClock() *fakeClock                { return &fakeClock{t: time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)} }
func ms(v int64) *int64                   { return &v }
func clockedService(t *testing.T, st storage.Storage, c *fakeClock) *Service {
	t.Helper()
	return New(st, Options{Sides: testSides, Rand: fixedRand(3), Now: c.now})
}

// playAfter waits, then plays the Play the test Side would.
func playAfter(t *testing.T, svc *Service, c *fakeClock, s *State, seconds int, p Play) *State {
	t.Helper()
	c.waitS(seconds)
	next, err := svc.Play(context.Background(), "", s.ID, s.Revision, p)
	if err != nil {
		t.Fatalf("Play %+v: %v", p, err)
	}
	return next
}

func eqMS(a, b *int64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func show(v *int64) any {
	if v == nil {
		return "unknown"
	}
	return *v
}

// TestDecisionDurations: each decision is timed from the trait, the cube
// decision apart from the play after its roll, a double and its answer on
// their own; the Arbiter's own plays carry none; the Match's Moves carry the
// durations the draft's Actions hold.
func TestDecisionDurations(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	c := newClock()
	svc := clockedService(t, st, c)
	s, err := svc.Create(ctx, "", Settings{MatchLength: 5, Sides: [2]SideSpec{external("A"), external("B")}})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if s.Awaiting == nil || s.Awaiting.Kind != DecideMove || !s.Awaiting.Since.Equal(c.now()) {
		t.Fatalf("the opening move should be awaited since now: %+v", s.Awaiting)
	}
	opener := s.Awaiting.Side
	s = playAfter(t, svc, c, s, 4, answer(*s.Awaiting))
	if s.Awaiting.Kind != DecideCube || s.Awaiting.Side == opener {
		t.Fatalf("the reply's cube decision should be awaited: %+v", s.Awaiting)
	}
	s = playAfter(t, svc, c, s, 2, Play{Side: s.Awaiting.Side, Kind: PlayRoll})
	if s.Awaiting.Kind != DecideMove {
		t.Fatalf("seed 3 should leave the reply a choice: %+v", s.Awaiting)
	}
	s = playAfter(t, svc, c, s, 5, answer(*s.Awaiting))
	s = playAfter(t, svc, c, s, 1, Play{Side: s.Awaiting.Side, Kind: PlayDouble})
	s = playAfter(t, svc, c, s, 3, Play{Side: s.Awaiting.Side, Kind: PlayTake})

	want := []struct {
		kind           transcript.Kind
		decision, cube *int64
	}{
		{transcript.KindChecker, ms(4000), nil},
		{transcript.KindChecker, ms(5000), ms(2000)},
		{transcript.KindDouble, ms(1000), nil},
		{transcript.KindTake, ms(3000), nil},
	}
	if len(s.Actions) < len(want) {
		t.Fatalf("%d actions, want at least %d", len(s.Actions), len(want))
	}
	for i, w := range want {
		a := s.Actions[i]
		if a.Kind != w.kind || !eqMS(a.DecisionMS, w.decision) || !eqMS(a.CubeDecisionMS, w.cube) {
			t.Errorf("action %d: %s %v/%v, want %s %v/%v", i, a.Kind, show(a.DecisionMS), show(a.CubeDecisionMS),
				w.kind, show(w.decision), show(w.cube))
		}
	}
	end, err := svc.Stop(ctx, "", s.ID, s.Revision, true)
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}
	var moves []*domain.Move
	for mv, err := range st.Matches().MovesByMatch(ctx, "", end.Ended.MatchID) {
		if err != nil {
			t.Fatal(err)
		}
		moves = append(moves, mv)
	}
	if len(moves) < len(want) {
		t.Fatalf("%d moves written, want at least %d", len(moves), len(want))
	}
	for i, w := range want {
		mv := moves[i]
		if !eqMS(mv.DecisionMS, w.decision) || !eqMS(mv.CubeDecisionMS, w.cube) {
			t.Errorf("move %d: %v/%v, want %v/%v", i, show(mv.DecisionMS), show(mv.CubeDecisionMS), show(w.decision), show(w.cube))
		}
	}
}

// TestSuspendedDuelStopsTheClock: the time a Duel spends in suspense is not
// counted; a Duel left running when its process stopped has the decision's
// duration unknown, not short.
func TestSuspendedDuelStopsTheClock(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	c := newClock()
	svc := clockedService(t, st, c)
	s, err := svc.Create(ctx, "", Settings{MatchLength: 5, Sides: [2]SideSpec{external("A"), external("B")}})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	c.waitS(3)
	if err := svc.Suspend(ctx, "", s.ID); err != nil {
		t.Fatalf("Suspend: %v", err)
	}
	c.wait(time.Hour)
	s, err = svc.Open(ctx, "", s.ID)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	s = playAfter(t, svc, c, s, 2, answer(*s.Awaiting))
	if got := s.Actions[0].DecisionMS; !eqMS(got, ms(5000)) {
		t.Errorf("a decision across a suspension took %v, want 5000", show(got))
	}

	// A second process opens the Duel this one left running.
	c.waitS(7)
	again := clockedService(t, st, c)
	r, err := again.Open(ctx, "", s.ID)
	if err != nil {
		t.Fatalf("Open from another process: %v", err)
	}
	if r.Awaiting.Kind != DecideCube {
		t.Fatalf("a cube decision should be awaited: %+v", r.Awaiting)
	}
	n := len(r.Actions)
	r = playAfter(t, again, c, r, 1, answer(*r.Awaiting))
	r = playAfter(t, again, c, r, 4, answer(*r.Awaiting))
	if got := r.Actions[n]; got.CubeDecisionMS != nil || !eqMS(got.DecisionMS, ms(4000)) {
		t.Errorf("after a lost time: cube %v (want unknown), play %v (want 4000)", show(got.CubeDecisionMS), show(got.DecisionMS))
	}
}

// TestCadenceCharge: the delay is free, per turn of the clock, never carried
// over; a cube decision and the play after its roll share one delay.
func TestCadenceCharge(t *testing.T) {
	g := &game{doc: document{Cadence: &Cadence{Reserve: 60, Delay: 12}}}
	g.doc.Clock.Reserve = [2]int64{60000, 60000}
	g.account(0, 5000, PlayRoll)
	if g.doc.Clock.Reserve[0] != 60000 || g.doc.Clock.Turn != 5000 {
		t.Fatalf("a cube decision inside the delay: reserve %d, turn %d", g.doc.Clock.Reserve[0], g.doc.Clock.Turn)
	}
	g.account(0, 10000, PlayMove)
	if g.doc.Clock.Reserve[0] != 57000 || g.doc.Clock.Turn != 0 {
		t.Fatalf("the play after the roll: reserve %d (want 57000), turn %d", g.doc.Clock.Reserve[0], g.doc.Clock.Turn)
	}
	g.account(0, 11000, PlayMove)
	if g.doc.Clock.Reserve[0] != 57000 {
		t.Errorf("an unused delay is carried over: reserve %d", g.doc.Clock.Reserve[0])
	}
	g.account(1, 80000, PlayMove)
	if g.doc.Clock.Reserve[1] != 0 || g.doc.Clock.OverTime != 2 {
		t.Errorf("player 2 over the reserve: reserve %d, over time %d", g.doc.Clock.Reserve[1], g.doc.Clock.OverTime)
	}
}

// TestNamedCadences: the tournament reserve is 2 min per point of the
// average length left at the score the Duel starts from.
func TestNamedCadences(t *testing.T) {
	tour, ok := NamedCadence("tournament")
	if !ok || tour.Delay != 12 || tour.TimeOut != "" {
		t.Fatalf("tournament = %+v, %v", tour, ok)
	}
	if got := tour.reserveMS(7, [2]int{0, 0}); got != 14*60000 {
		t.Errorf("7 points at 0-0: %d ms, want 14 min", got)
	}
	if got := tour.reserveMS(7, [2]int{4, 2}); got != 8*60000 {
		t.Errorf("7 points from 4-2: %d ms, want 8 min", got)
	}
	for _, name := range []string{"rapid-3+12", "rapid-2+12", "rapid-3+15"} {
		if c, ok := NamedCadence(name); !ok || c.check(0) != nil {
			t.Errorf("%s = %+v, %v", name, c, ok)
		}
	}
	for _, bad := range []struct {
		c      Cadence
		length int
	}{
		{Cadence{Delay: 12}, 5},
		{Cadence{Reserve: 60, ReservePerPoint: 60}, 5},
		{tour, 0},
		{Cadence{Reserve: 60, TimeOut: "forfeit"}, 5},
	} {
		if err := bad.c.check(bad.length); !errors.Is(err, ErrInvalidCadence) {
			t.Errorf("%+v at %d points: %v, want ErrInvalidCadence", bad.c, bad.length, err)
		}
	}

	ctx := context.Background()
	st := newStore(t)
	c := newClock()
	svc := clockedService(t, st, c)
	if _, err := svc.Create(ctx, "", Settings{Cadence: &tour, Sides: [2]SideSpec{external("A"), external("B")}}); !errors.Is(err, ErrInvalidCadence) {
		t.Errorf("a reserve per point for a money session: %v", err)
	}
	s, err := svc.Create(ctx, "", Settings{MatchLength: 7, Cadence: &tour, Sides: [2]SideSpec{external("A"), external("B")}})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if s.Clock == nil || s.Clock.Reserve != [2]int64{14 * 60000, 14 * 60000} || s.Clock.Cadence.TimeOut != TimeContinue {
		t.Errorf("clock at creation: %+v", s.Clock)
	}
}

// TestLoseOnTime: under TimeLoseMatch, a Play received after the reserve ran
// out does not count; the Match stops there, carries who ran out and the
// Cadence, and no point is invented.
func TestLoseOnTime(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	c := newClock()
	svc := clockedService(t, st, c)
	cad := Cadence{Reserve: 10, Delay: 2, TimeOut: TimeLoseMatch}
	s, err := svc.Create(ctx, "", Settings{MatchLength: 5, Cadence: &cad, Sides: [2]SideSpec{external("A"), external("B")}})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	s = playAfter(t, svc, c, s, 3, answer(*s.Awaiting))
	if s.Clock.Reserve[1-s.Awaiting.Side] != 9000 {
		t.Errorf("3 s on a 2 s delay: reserve %d, want 9000", s.Clock.Reserve[1-s.Awaiting.Side])
	}
	late := s.Awaiting.Side
	played := len(s.Actions)
	end := playAfter(t, svc, c, s, 13, answer(*s.Awaiting))
	if end.Ended == nil || end.Ended.OverTime != late+1 || !end.Ended.StoppedEarly || len(end.Actions) != played {
		t.Fatalf("ended %+v with %d actions; want player %d over time, stopped early, %d actions",
			end.Ended, len(end.Actions), late+1, played)
	}
	o, err := st.Duels().Origin(ctx, "", end.Ended.MatchID)
	if err != nil || o.OverTime != late+1 || !o.StoppedEarly {
		t.Fatalf("origin %+v, %v", o, err)
	}
	var back Cadence
	if err := json.Unmarshal([]byte(o.Cadence), &back); err != nil || back != cad {
		t.Errorf("origin cadence %q, want %+v", o.Cadence, cad)
	}
	games, err := gamesOf(st, end.Ended.MatchID)
	if err != nil || len(games) != 1 || games[0].Winner != domain.WinnerUnfinished || games[0].PointsWon != 0 {
		t.Errorf("games %+v, %v: one, unfinished, no points", games, err)
	}
}

// TestPlayOnAfterTime: under TimeContinue the overrun is noted and the match
// goes on; Flag notes it before the Play, and ends a Duel set to lose.
func TestPlayOnAfterTime(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	c := newClock()
	svc := clockedService(t, st, c)
	cad := Cadence{Reserve: 5, Delay: 0}
	s, err := svc.Create(ctx, "", Settings{MatchLength: 5, Cadence: &cad, Sides: [2]SideSpec{external("A"), external("B")}})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	first := s.Awaiting.Side
	c.waitS(6)
	if s, err = svc.Flag(ctx, "", s.ID); err != nil || s.Clock.OverTime != first+1 || s.Ended != nil {
		t.Fatalf("Flag: %+v, %v", s.Clock, err)
	}
	s = playAfter(t, svc, c, s, 0, answer(*s.Awaiting))
	if len(s.Actions) == 0 || !eqMS(s.Actions[0].DecisionMS, ms(6000)) || s.Clock.Reserve[first] != 0 {
		t.Errorf("played on: %d actions, reserve %d", len(s.Actions), s.Clock.Reserve[first])
	}
	end, err := svc.Stop(ctx, "", s.ID, s.Revision, true)
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if o, err := st.Duels().Origin(ctx, "", end.Ended.MatchID); err != nil || o.OverTime != first+1 {
		t.Errorf("origin %+v, %v: the overrun is a fact of the Match", o, err)
	}

	lose := Cadence{Reserve: 5, TimeOut: TimeLoseMatch}
	s, err = svc.Create(ctx, "", Settings{MatchLength: 5, Cadence: &lose, Sides: [2]SideSpec{external("A"), external("B")}})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	c.waitS(6)
	// Nothing played yet: there is no Match to keep.
	if s, err = svc.Flag(ctx, "", s.ID); err != nil || s.Ended == nil || !s.Ended.Discarded || s.Ended.OverTime == 0 {
		t.Fatalf("Flag before any play: %+v, %v", s, err)
	}
}
