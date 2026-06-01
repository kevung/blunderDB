package database

import (
	"sync"
	"testing"
)

// TestInMemoryConcurrentReadsShareSchema guards against a regression where the
// in-memory GUI database (SetupDatabase(":memory:")) lost its schema under
// concurrent reads.
//
// Root cause: with database/sql, every pooled connection to ":memory:" is a
// SEPARATE, empty in-memory database. SetupDatabase creates the schema on one
// connection, but Database's RWMutex allows multiple concurrent readers, so the
// pool would open extra connections — each with no tables — yielding
// "SQL logic error: no such table: match/tournament/search_history". This
// surfaced in the GUI before any database file was opened, when switching to
// tabs (matches/tournaments/stats) that fire several Wails queries at once.
//
// Fix: ConfigurePool pins ":memory:" to a single connection (db.go).
func TestInMemoryConcurrentReadsShareSchema(t *testing.T) {
	d := NewDatabase()
	if err := d.SetupDatabase(":memory:"); err != nil {
		t.Fatalf("SetupDatabase(:memory:): %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	const goroutines = 64
	var wg sync.WaitGroup
	errs := make(chan error, goroutines*3)
	for range goroutines {
		wg.Go(func() {
			if _, err := d.GetAllMatches(); err != nil {
				errs <- err
			}
			if _, err := d.GetAllTournaments(); err != nil {
				errs <- err
			}
			if _, err := d.LoadSearchHistory(); err != nil {
				errs <- err
			}
		})
	}
	wg.Wait()
	close(errs)

	failures := 0
	for err := range errs {
		failures++
		if failures <= 3 {
			t.Errorf("concurrent read on in-memory DB failed: %v", err)
		}
	}
	if failures > 0 {
		t.Errorf("%d concurrent in-memory reads failed (expected 0)", failures)
	}
}
