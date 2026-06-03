package database

import (
	"context"
	"errors"
	"sync"
	"testing"
)

// TestCancelImportWiring verifies the import-cancellation plumbing that
// replaced the legacy importCancelled atomic flag (headless task P8): a
// registered import context is cancelled by CancelImport, the done func
// clears the registration, and CancelImport is a safe no-op when idle.
func TestCancelImportWiring(t *testing.T) {
	d := NewDatabase()

	// Idle: CancelImport must not panic and must do nothing.
	d.CancelImport()

	ctx, done := d.beginCancellableImport()
	if ctx.Err() != nil {
		t.Fatalf("freshly registered import context should be live, got %v", ctx.Err())
	}

	// CancelImport from another goroutine (as the Wails frontend would,
	// while the import holds d.mu) must cancel the registered context.
	d.CancelImport()
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatalf("after CancelImport, ctx.Err() = %v, want context.Canceled", ctx.Err())
	}

	// done() clears the registration; a subsequent CancelImport is a no-op.
	done()
	d.cancelMu.Lock()
	cleared := d.importCancel == nil
	d.cancelMu.Unlock()
	if !cleared {
		t.Fatal("done() should clear the registered cancel func")
	}
	d.CancelImport() // must not panic now that there is no in-flight import
}

// TestCancelImportConcurrent exercises the cancel path under -race: many
// goroutines call CancelImport while imports are repeatedly begun and
// finished, mirroring a frontend hammering the cancel button.
func TestCancelImportConcurrent(t *testing.T) {
	d := NewDatabase()
	var wg sync.WaitGroup

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				d.CancelImport()
			}
		}()
	}
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				_, done := d.beginCancellableImport()
				done()
			}
		}()
	}
	wg.Wait()
}
