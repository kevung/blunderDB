package direction

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
)

// clubConfig is the format the study recommends and the one lot 1 must direct end to end:
// two-life Swiss until the sum of lives reaches 16, then a bracket with byes.
func clubConfig() tournoi.Config {
	return tournoi.Config{
		Name:   "Open de Lyon",
		Tables: tournoi.Tables{Count: 8},
		Phases: []tournoi.PhaseConfig{
			{Kind: tournoi.KindSwissLives, Length: 7, Target: 16},
			{Kind: tournoi.KindLivesBracket, Length: 9, FinalLength: 11},
		},
	}
}

func entrants(n int) []tournoi.Player {
	out := make([]tournoi.Player, n)
	for i := range out {
		id := fmt.Sprintf("j%02d", i+1)
		out[i] = tournoi.Player{ID: tournoi.PlayerID(id), Name: "Joueur " + id, Rating: float64(3 + i%12)}
	}
	return out
}

// runToEnd directs a whole tournament: it confirms every proposal and enters a result for every
// match, exactly as the panel will. It returns the number of matches played.
func runToEnd(t *testing.T, d *Direction, now time.Time) int {
	t.Helper()
	ctx := context.Background()
	matches := 0
	for step := 0; step < 4000; step++ {
		acts := d.Propose()
		if len(acts) == 0 {
			break
		}
		progressed := false
		for _, a := range acts {
			switch a.Kind {
			case tournoi.ActWait:
				continue
			case tournoi.ActFinish:
				if err := d.Finish(ctx, now); err != nil {
					t.Fatalf("finish: %v", err)
				}
				return matches
			}
			ev, err := d.EventFor(a, now)
			if err != nil {
				t.Fatalf("event for %s: %v", a.Kind, err)
			}
			if err := d.Apply(ctx, ev); err != nil {
				t.Fatalf("apply %s: %v", a.Kind, err)
			}
			progressed = true
			if a.Kind == tournoi.ActStartMatch {
				matches++
				now = now.Add(time.Minute)
				// The winner is the lower id, so the run is deterministic.
				w := ev.A
				if string(ev.B) < string(ev.A) {
					w = ev.B
				}
				if err := d.Apply(ctx, tournoi.ResultEvent(ev.MatchID, w, a.Length, 2, now)); err != nil {
					t.Fatalf("result: %v", err)
				}
			}
		}
		if !progressed {
			break
		}
	}
	return matches
}

func newDraft(t *testing.T, store Store, cfg tournoi.Config) *Direction {
	t.Helper()
	d, err := Create(context.Background(), store, 1, cfg)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	return d
}

// TestTournoiDeClubDeBoutEnBout: a 24-player club tournament runs to a complete ranking with no
// standing warning. This is the shape lot 1 must direct.
func TestTournoiDeClubDeBoutEnBout(t *testing.T) {
	ctx := context.Background()
	store := newMemStore()
	d := newDraft(t, store, clubConfig())
	now := time.Date(2026, 9, 12, 9, 0, 0, 0, time.UTC)
	if err := d.Start(ctx, 7, now, entrants(24)); err != nil {
		t.Fatalf("start: %v", err)
	}
	matches := runToEnd(t, d, now)
	if matches == 0 {
		t.Fatal("no match was played")
	}
	if !d.State().Finished {
		t.Error("the tournament should be finished")
	}
	if w := d.Warnings(); len(w) != 0 {
		t.Errorf("a clean run should leave no warning, got %v", w)
	}
	r := d.Ranking()
	if len(r) != 24 {
		t.Fatalf("ranking of 24 expected, got %d", len(r))
	}
	if r[0].Rank != 1 || r[0].Note.Kind == tournoi.NoteNone {
		t.Errorf("the winner should be ranked first with a note: %+v", r[0])
	}
	if d.Record().State != StateFinished {
		t.Errorf("record state %q, want %q", d.Record().State, StateFinished)
	}
}

// TestLEtatEstRejoueJamaisStocke: reopening replays the log and lands on exactly the same
// tournament. Nothing derived is persisted, which is what makes a crash free.
func TestLEtatEstRejoueJamaisStocke(t *testing.T) {
	ctx := context.Background()
	store := newMemStore()
	d := newDraft(t, store, clubConfig())
	now := time.Date(2026, 9, 12, 9, 0, 0, 0, time.UTC)
	if err := d.Start(ctx, 3, now, entrants(16)); err != nil {
		t.Fatal(err)
	}
	runToEnd(t, d, now)

	reopened, err := Open(ctx, store, 1)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	before, _ := json.Marshal(d.Ranking())
	after, _ := json.Marshal(reopened.Ranking())
	if string(before) != string(after) {
		t.Errorf("the replayed ranking differs from the live one\nlive:     %s\nreplayed: %s", before, after)
	}
	if len(reopened.Journal()) != len(d.Journal()) {
		t.Errorf("replayed journal has %d events, live has %d", len(reopened.Journal()), len(d.Journal()))
	}
	if len(reopened.Warnings()) != 0 {
		t.Errorf("replay raised warnings: %v", reopened.Warnings())
	}
}

// TestCoupureEntreDeuxDecisions: a crash between two decisions leaves the log at exactly what
// the director last saw — the event is written before it is applied, never after.
func TestCoupureEntreDeuxDecisions(t *testing.T) {
	ctx := context.Background()
	store := newMemStore()
	d := newDraft(t, store, clubConfig())
	now := time.Date(2026, 9, 12, 9, 0, 0, 0, time.UTC)
	// 24 entrants: twice the switch threshold, so the Swiss phase actually pairs. Eight
	// players at two lives already sum to the target of 16 and the phase would be over
	// before it began.
	if err := d.Start(ctx, 5, now, entrants(24)); err != nil {
		t.Fatal(err)
	}
	// Confirm a few proposals, then cut the power.
	acts := d.Propose()
	if len(acts) == 0 {
		t.Fatal("no proposal")
	}
	ev, err := d.EventFor(acts[0], now)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Apply(ctx, ev); err != nil {
		t.Fatal(err)
	}
	written := len(d.Journal())

	store.failAt = written // the next write fails
	next := d.Propose()
	if len(next) > 0 && next[0].Kind == tournoi.ActStartMatch {
		ev2, err := d.EventFor(next[0], now)
		if err != nil {
			t.Fatal(err)
		}
		if err := d.Apply(ctx, ev2); err == nil {
			t.Fatal("the write was supposed to fail")
		}
	}
	store.failAt = -1

	reopened, err := Open(ctx, store, 1)
	if err != nil {
		t.Fatalf("reopen after the cut: %v", err)
	}
	if got := len(reopened.Journal()); got != written {
		t.Errorf("after the cut the log holds %d events, want %d — a decision was lost or half-written", got, written)
	}
	if len(reopened.Warnings()) != 0 {
		t.Errorf("the state after the cut is inconsistent: %v", reopened.Warnings())
	}
	// And the tournament goes on from there.
	if len(reopened.Propose()) == 0 {
		t.Error("the tournament should be resumable after the cut")
	}
}

// TestJournalAppendOnly: nothing rewrites a sequence number. A correction is one more event.
func TestJournalAppendOnly(t *testing.T) {
	ctx := context.Background()
	store := newMemStore()
	d := newDraft(t, store, clubConfig())
	now := time.Date(2026, 9, 12, 9, 0, 0, 0, time.UTC)
	if err := d.Start(ctx, 5, now, entrants(24)); err != nil {
		t.Fatal(err)
	}
	var started tournoi.Event
	for _, a := range d.Propose() {
		if a.Kind != tournoi.ActStartMatch {
			continue
		}
		ev, err := d.EventFor(a, now)
		if err != nil {
			t.Fatal(err)
		}
		if err := d.Apply(ctx, ev); err != nil {
			t.Fatal(err)
		}
		started = ev
		break
	}
	if started.MatchID == "" {
		t.Fatal("no match was started")
	}
	if err := d.Apply(ctx, tournoi.ResultEvent(started.MatchID, started.A, 7, 3, now)); err != nil {
		t.Fatal(err)
	}
	before := len(d.Journal())
	if err := d.Apply(ctx, tournoi.CorrectionEvent(started.MatchID, started.B, 7, 5, now)); err != nil {
		t.Fatalf("correction: %v", err)
	}
	if len(d.Journal()) != before+1 {
		t.Errorf("a correction adds an event, it does not replace one: %d then %d", before, len(d.Journal()))
	}
	if w := d.State().Matches[started.MatchID].Winner; w != started.B {
		t.Errorf("the correction should have taken effect: winner %s", w)
	}
	// The first result is still in the log, in its place: the history is complete.
	kinds := make([]string, 0, len(d.Journal()))
	for _, ev := range d.Journal() {
		kinds = append(kinds, string(ev.Kind))
	}
	if strings.Count(strings.Join(kinds, " "), "result") < 2 {
		t.Errorf("both the result and its correction should be in the log: %v", kinds)
	}
	for i, ev := range d.Journal() {
		if ev.Seq != i {
			t.Errorf("event %d carries sequence %d: the log must be contiguous", i, ev.Seq)
		}
	}
}

// TestAucunTexteTraduitNeSortDuPaquet: labels, notes, warnings and wait reasons are codes.
// blunderDB speaks nine languages and these values live in the database.
func TestAucunTexteTraduitNeSortDuPaquet(t *testing.T) {
	ctx := context.Background()
	store := newMemStore()
	d := newDraft(t, store, clubConfig())
	now := time.Date(2026, 9, 12, 9, 0, 0, 0, time.UTC)
	if err := d.Start(ctx, 11, now, entrants(16)); err != nil {
		t.Fatal(err)
	}
	check := func(v any, what string) {
		t.Helper()
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		var raw any
		if err := json.Unmarshal(b, &raw); err != nil {
			t.Fatal(err)
		}
		var walk func(any, string)
		walk = func(x any, path string) {
			switch v := x.(type) {
			case map[string]any:
				for k, e := range v {
					// Only what the director typed travels as free text.
					if k == "text" || k == "name" || k == "club" {
						continue
					}
					walk(e, path+"."+k)
				}
			case []any:
				for _, e := range v {
					walk(e, path)
				}
			case string:
				for _, r := range v {
					if r > 127 || r == ' ' {
						t.Errorf("%s: %s is %q — a sentence, not a code", what, path, v)
						return
					}
				}
			}
		}
		walk(raw, what)
	}
	for i := 0; i < 30; i++ {
		acts := d.Propose()
		check(acts, "proposals")
		check(d.Ranking(), "ranking")
		check(d.Warnings(), "warnings")
		advanced := false
		for _, a := range acts {
			if a.Kind != tournoi.ActStartMatch && a.Kind != tournoi.ActDraw && a.Kind != tournoi.ActBye {
				continue
			}
			ev, err := d.EventFor(a, now)
			if err != nil {
				t.Fatal(err)
			}
			if err := d.Apply(ctx, ev); err != nil {
				t.Fatal(err)
			}
			advanced = true
			if a.Kind == tournoi.ActStartMatch {
				if err := d.Apply(ctx, tournoi.ResultEvent(ev.MatchID, ev.A, 7, 3, now)); err != nil {
					t.Fatal(err)
				}
			}
		}
		if !advanced {
			break
		}
	}
}

// TestBrouillonPuisDemarrage: while a Direction is a draft the configuration is rewritten in
// place and nothing is logged; the first start freezes it.
func TestBrouillonPuisDemarrage(t *testing.T) {
	ctx := context.Background()
	store := newMemStore()
	d := newDraft(t, store, clubConfig())
	if d.Record().State != StateDraft {
		t.Fatalf("a new Direction is a draft, not %q", d.Record().State)
	}
	if len(d.Journal()) != 0 {
		t.Error("a draft writes nothing to the log")
	}
	// The director changes their mind: three lives instead of two.
	cfg := clubConfig()
	cfg.Phases[0].Lives = 3
	cfg.Phases[0].Target = 0
	if err := d.SetConfig(ctx, cfg); err != nil {
		t.Fatalf("changing a draft configuration: %v", err)
	}
	got, err := d.Config()
	if err != nil {
		t.Fatal(err)
	}
	if got.Phases[0].Lives != 3 {
		t.Errorf("the draft configuration was not kept: %d lives", got.Phases[0].Lives)
	}
	if len(d.Journal()) != 0 {
		t.Error("changing a draft configuration writes nothing to the log")
	}

	now := time.Now()
	if err := d.Start(ctx, 1, now, entrants(8)); err != nil {
		t.Fatal(err)
	}
	if d.Record().State != StateRunning {
		t.Errorf("after starting, state %q", d.Record().State)
	}
	if d.State().Config.Phases[0].Lives != 3 {
		t.Error("the started tournament should run on the edited configuration")
	}
	// And now the configuration is frozen: a change is an event, not a rewrite.
	if err := d.SetConfig(ctx, clubConfig()); err == nil {
		t.Error("rewriting the configuration of a started tournament must be refused")
	}
}

// TestTournoiSansDirection: a Tournament assembled from imported files has no Direction, and
// that is the ordinary case — not an error to hide.
func TestTournoiSansDirection(t *testing.T) {
	_, err := Open(context.Background(), newMemStore(), 42)
	if !errors.Is(err, ErrNoDirection) {
		t.Errorf("opening an undirected tournament: %v, want ErrNoDirection", err)
	}
}

// TestClotureEtLectureSeule: once closed, the Direction refuses further decisions.
func TestClotureEtLectureSeule(t *testing.T) {
	ctx := context.Background()
	store := newMemStore()
	d := newDraft(t, store, clubConfig())
	now := time.Now()
	if err := d.Start(ctx, 2, now, entrants(24)); err != nil {
		t.Fatal(err)
	}
	runToEnd(t, d, now)
	if !d.State().Finished {
		t.Fatal("the tournament should be finished")
	}
	err := d.Apply(ctx, tournoi.PlayerAddedEvent(tournoi.Player{ID: "tardif", Name: "Tardif"}, now))
	if !errors.Is(err, ErrFinished) {
		t.Errorf("a closed tournament refuses a decision: %v", err)
	}
}

// TestSuppressionDeLaDirection: deleting the Direction takes its log with it.
func TestSuppressionDeLaDirection(t *testing.T) {
	ctx := context.Background()
	store := newMemStore()
	d := newDraft(t, store, clubConfig())
	if err := d.Start(ctx, 1, time.Now(), entrants(4)); err != nil {
		t.Fatal(err)
	}
	if err := d.Delete(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(ctx, store, 1); !errors.Is(err, ErrNoDirection) {
		t.Errorf("after deletion: %v, want ErrNoDirection", err)
	}
}

// TestLaVersionDuMoteurEstEnregistree: a replay problem must be traceable to the engine that
// wrote the log.
func TestLaVersionDuMoteurEstEnregistree(t *testing.T) {
	d := newDraft(t, newMemStore(), clubConfig())
	if d.Record().EngineVersion != EngineVersion {
		t.Errorf("engine version %q, want %q", d.Record().EngineVersion, EngineVersion)
	}
	if d.Record().FormatVersion != tournoi.JournalVersion {
		t.Errorf("journal format %d, want %d", d.Record().FormatVersion, tournoi.JournalVersion)
	}
}
