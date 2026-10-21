package database

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

var fillFixtures = []string{
	"testdata/charlot1-charlot2_7p_2025-11-08-2305.xg",
	"testdata/Kev_GammonNetNormal_2026-10-06_7p.xg",
	"testdata/test.xg",
}

// libraryWithoutMatchStats is a library of three matches in one tournament
// whose match_stats table was wiped, as a migration leaves it.
func libraryWithoutMatchStats(t *testing.T, name string) (*Database, string) {
	t.Helper()
	path := filepath.Join(tempDir(t), name)
	d := NewDatabase()
	if err := d.SetupDatabase(path); err != nil {
		t.Fatal(err)
	}
	closeOnCleanup(t, d)
	tid, err := d.CreateTournament("T", "2026-01-01", "X")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range fillFixtures {
		id, err := d.ImportXGMatch(p)
		if err != nil {
			t.Fatal(err)
		}
		if err := d.AddMatchToTournament(tid, id); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := d.db.Exec(`DELETE FROM match_stats`); err != nil {
		t.Fatal(err)
	}
	return d, path
}

// tournamentT is the fixture's own tournament; the imports add one of theirs.
func tournamentT(t *testing.T, ts []Tournament) Tournament {
	t.Helper()
	for _, x := range ts {
		if x.Name == "T" {
			return x
		}
	}
	t.Fatalf("no tournament T in %d", len(ts))
	return Tournament{}
}

func missingStats(t *testing.T, d *Database) int {
	t.Helper()
	ids, err := d.store.Stats().MissingMatchStats(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	return len(ids)
}

type fillRecorder struct {
	mu       sync.Mutex
	progress [][2]int
	ended    chan error
	filled   int
}

func recordFill(d *Database) *fillRecorder {
	r := &fillRecorder{ended: make(chan error, 1)}
	SetMatchStatsFillReport(d, &MatchStatsFillReport{
		Progress: func(done, total int) {
			r.mu.Lock()
			r.progress = append(r.progress, [2]int{done, total})
			r.mu.Unlock()
		},
		End: func(filled int, err error) {
			r.filled = filled
			r.ended <- err
		},
	})
	return r
}

// pauseFill makes the fill one match per lot and stops it after the first
// lot until the returned resume is called.
func pauseFill(t *testing.T) (paused <-chan struct{}, resume func()) {
	t.Helper()
	oldLot, oldHook := matchStatsFillLot, betweenFillLots
	matchStatsFillLot = 1
	p, r := make(chan struct{}), make(chan struct{})
	var once, release sync.Once
	betweenFillLots = func() {
		once.Do(func() { close(p) })
		<-r
	}
	resume = func() { release.Do(func() { close(r) }) }
	t.Cleanup(resume)
	t.Cleanup(func() { matchStatsFillLot, betweenFillLots = oldLot, oldHook })
	return p, resume
}

func waitEnd(t *testing.T, r *fillRecorder) error {
	t.Helper()
	select {
	case err := <-r.ended:
		return err
	case <-time.After(10 * time.Second):
		t.Fatal("the background fill never ended")
		return nil
	}
}

func TestFillMatchStatsInBackground_FillsAndReports(t *testing.T) {
	d, _ := libraryWithoutMatchStats(t, "fill.db")
	r := recordFill(d)
	oldLot := matchStatsFillLot
	matchStatsFillLot = 1
	t.Cleanup(func() { matchStatsFillLot = oldLot })

	if !d.FillMatchStatsInBackground() {
		t.Fatal("no fill started on a library missing its statistics")
	}
	if err := waitEnd(t, r); err != nil {
		t.Fatalf("fill ended with %v", err)
	}
	if n := missingStats(t, d); n != 0 {
		t.Fatalf("%d match(es) still without statistics", n)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.progress) == 0 {
		t.Fatal("no progress reported")
	}
	last := 0
	for _, p := range r.progress {
		if p[1] != len(fillFixtures) || p[0] < last {
			t.Fatalf("progress %v: want a count rising to %d", r.progress, len(fillFixtures))
		}
		last = p[0]
	}
	if last != len(fillFixtures) || r.filled != len(fillFixtures) {
		t.Fatalf("progress ends at %d, filled %d; want %d", last, r.filled, len(fillFixtures))
	}
	if d.MatchStatsFilling() || d.FillMatchStatsInBackground() {
		t.Fatal("a complete library still fills")
	}
}

// While the fill runs, the tournaments come back at once, without the badges
// that would fill the rest of the library in one read hold.
func TestGetAllTournaments_DuringBackgroundFillAnswersAtOnce(t *testing.T) {
	d, _ := libraryWithoutMatchStats(t, "busy.db")
	r := recordFill(d)
	paused, resume := pauseFill(t)
	if !d.FillMatchStatsInBackground() {
		t.Fatal("no fill started")
	}
	<-paused
	got := make(chan []Tournament, 1)
	go func() {
		ts, err := d.GetAllTournaments()
		if err != nil {
			t.Error(err)
		}
		got <- ts
	}()
	select {
	case ts := <-got:
		if x := tournamentT(t, ts); x.PR != 0 {
			t.Fatalf("tournament %+v: want it without its badge", x)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the tournaments wait for the fill")
	}
	if _, err := d.ListMatches(storage.MatchListOpts{Limit: 1}); err != nil {
		t.Fatal(err)
	}
	resume()
	if err := waitEnd(t, r); err != nil {
		t.Fatal(err)
	}
	ts, err := d.GetAllTournaments()
	if err != nil || tournamentT(t, ts).PR == 0 {
		t.Fatalf("after the fill: %+v, %v; want the badge", ts, err)
	}
}

func TestFillMatchStatsInBackground_CancelledByUser(t *testing.T) {
	d, _ := libraryWithoutMatchStats(t, "cancel.db")
	r := recordFill(d)
	paused, resume := pauseFill(t)
	if !d.FillMatchStatsInBackground() {
		t.Fatal("no fill started")
	}
	<-paused
	d.CancelMaintenance()
	resume()
	if err := waitEnd(t, r); !errors.Is(err, context.Canceled) {
		t.Fatalf("end %v; want context.Canceled", err)
	}
	if n := missingStats(t, d); n == 0 {
		t.Fatal("a cancelled fill went on to the end")
	}
}

// Opening another file stops the fill between two lots: it never writes into
// the next library, and the screen, already on that one, is told nothing.
func TestFillMatchStatsInBackground_StopsWhenTheFileChanges(t *testing.T) {
	d, _ := libraryWithoutMatchStats(t, "first.db")
	other, otherPath := libraryWithoutMatchStats(t, "second.db")
	if err := other.Close(); err != nil {
		t.Fatal(err)
	}
	r := recordFill(d)
	paused, resume := pauseFill(t)
	if !d.FillMatchStatsInBackground() {
		t.Fatal("no fill started")
	}
	<-paused
	if err := d.OpenDatabase(otherPath); err != nil {
		t.Fatal(err)
	}
	// Before the second file's temporary directory is checked for leaks.
	closeOnCleanup(t, d)
	if d.MatchStatsFilling() {
		t.Fatal("the fill of the previous file still counts as running")
	}
	resume()
	select {
	case err := <-r.ended:
		t.Fatalf("a fill stopped by the switch reported its end (%v)", err)
	case <-time.After(200 * time.Millisecond):
	}
	if n := missingStats(t, d); n != len(fillFixtures) {
		t.Fatalf("%d match(es) of the next library lack statistics; want all %d", n, len(fillFixtures))
	}
}

func TestFillMatchStatsInBackground_WritesNothingReadOnly(t *testing.T) {
	d, _ := libraryWithoutMatchStats(t, "ro.db")
	recordFill(d)
	d.mu.Lock()
	d.readOnly = true
	d.mu.Unlock()
	t.Cleanup(func() {
		d.mu.Lock()
		d.readOnly = false
		d.mu.Unlock()
	})
	if d.FillMatchStatsInBackground() {
		t.Fatal("a fill started on a read-only library")
	}
}

// Without a report, as in the CLI, nothing fills in the background: the
// reads fill before they answer.
func TestFillMatchStatsInBackground_OffWithoutReport(t *testing.T) {
	d, _ := libraryWithoutMatchStats(t, "cli.db")
	if d.FillMatchStatsInBackground() {
		t.Fatal("a fill started without a report")
	}
	ts, err := d.GetAllTournaments()
	if err != nil || tournamentT(t, ts).PR == 0 {
		t.Fatalf("synchronous path: %+v, %v; want the badge", ts, err)
	}
	if n := missingStats(t, d); n != 0 {
		t.Fatalf("%d match(es) left unfilled by the synchronous path", n)
	}
}
