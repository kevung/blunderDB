package database

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// A writer queued on Database.mu while the tournament badges fill the
// library must not keep the match list's page waiting for the whole fill:
// the fill gives the lock back between lots.
func TestGetAllTournaments_FillDoesNotStarvePageBehindQueuedWriter(t *testing.T) {
	d := NewDatabase()
	if err := d.SetupDatabase(filepath.Join(tempDir(t), "prefill.db")); err != nil {
		t.Fatal(err)
	}
	closeOnCleanup(t, d)

	tid, err := d.CreateTournament("T", "2026-01-01", "X")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{
		"testdata/charlot1-charlot2_7p_2025-11-08-2305.xg",
		"testdata/Kev_GammonNetNormal_2026-10-06_7p.xg",
		"testdata/test.xg",
		"testdata/HsbtMarseille_main_ronde4_LamourDeCaslouGildas_UngerKevin_7p.xg",
	} {
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
	oldLot, oldHook := matchStatsFillLot, betweenFillLots
	matchStatsFillLot = 1
	paused, resume := make(chan struct{}), make(chan struct{})
	var once sync.Once
	betweenFillLots = func() {
		once.Do(func() { close(paused) })
		<-resume
	}
	var release sync.Once
	// Registered after closeOnCleanup, so it runs before it: a failing test
	// frees the paused fill instead of leaving the close to wait on its lock.
	t.Cleanup(func() { release.Do(func() { close(resume) }) })
	t.Cleanup(func() { matchStatsFillLot, betweenFillLots = oldLot, oldHook })

	done := make(chan struct{})
	go func() {
		_, _ = d.GetAllTournaments()
		close(done)
	}()
	select {
	case <-paused:
	case <-time.After(5 * time.Second):
		t.Fatal("the fill does not yield the lock between lots")
	}
	// A writer queues while the fill is between two lots; the page must
	// still be served before the fill resumes.
	go func() {
		d.mu.Lock()
		d.mu.Unlock() //nolint:staticcheck // empty critical section: a queued writer is the point
	}()
	time.Sleep(20 * time.Millisecond)
	served := make(chan error, 1)
	go func() {
		_, err := d.ListMatches(storage.MatchListOpts{Limit: 1})
		served <- err
	}()
	select {
	case err := <-served:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the page waits behind the fill")
	}
	release.Do(func() { close(resume) })
	<-done
}
