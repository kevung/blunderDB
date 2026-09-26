package database

import (
	"sync"
	"testing"
)

// TestInMemoryConcurrentReadsShareSchema: every pooled connection to
// ":memory:" is a SEPARATE empty database, so concurrent readers would hit
// "no such table"; ConfigurePool pins ":memory:" to one connection.
func TestInMemoryConcurrentReadsShareSchema(t *testing.T) {
	t.Parallel()
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
