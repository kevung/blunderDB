package database

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Of jobs racing for the library, exactly one gets it; the others are
// refused, not queued (ADR-0088).
func TestAcquireJob_OneWinnerUnderRace(t *testing.T) {
	t.Parallel()
	d := NewDatabase()
	ctx := context.Background()
	var won, refused atomic.Int32
	var wg sync.WaitGroup
	releases := make(chan func(), 16)
	start := make(chan struct{})
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			release, err := d.acquireJob(ctx, "job", nil)
			switch {
			case err == nil:
				won.Add(1)
				releases <- release
			case errors.Is(err, ErrDatabaseBusy):
				refused.Add(1)
			default:
				t.Errorf("acquire: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()
	if won.Load() != 1 || refused.Load() != 15 {
		t.Fatalf("%d won, %d refused; want 1 and 15", won.Load(), refused.Load())
	}
	(<-releases)()
	release, err := d.acquireJob(ctx, "next", nil)
	if err != nil {
		t.Fatalf("after release: %v", err)
	}
	release()
}

// A maintenance run is refused while a gammonNet batch or a rollout holds
// the library, and a job replacing a cancelled one waits for it to finish
// instead of being refused.
func TestAcquireJob_SharedWithMaintenanceAndReplacement(t *testing.T) {
	t.Parallel()
	d := NewDatabase()
	if err := d.SetupDatabase(filepath.Join(tempDir(t), "slot.db")); err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	batchCtx, cancelBatch := context.WithCancel(context.Background())
	release, err := d.acquireJob(batchCtx, "a gammonNet analysis", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.RunMaintenance("stats.matches"); !errors.Is(err, ErrDatabaseBusy) {
		t.Fatalf("maintenance during a batch: %v, want ErrDatabaseBusy", err)
	}

	cancelBatch()
	finished := make(chan struct{})
	go func() {
		time.Sleep(20 * time.Millisecond)
		release()
		close(finished)
	}()
	r, err := d.RunMaintenance("stats.matches")
	if err != nil {
		t.Fatalf("maintenance after the batch was cancelled: %v", err)
	}
	select {
	case <-finished:
	default:
		t.Error("the run did not wait for the cancelled batch to finish")
	}
	if r.Outcome != "done" {
		t.Errorf("run %+v", r)
	}
}

// StopMaintenance ends a run and waits for it; it leaves another job alone.
func TestStopMaintenance_CancelsOnlyMaintenance(t *testing.T) {
	t.Parallel()
	d := NewDatabase()
	other, cancelOther := context.WithCancel(context.Background())
	defer cancelOther()
	release, err := d.acquireJob(other, "a rollout", nil)
	if err != nil {
		t.Fatal(err)
	}
	d.StopMaintenance(context.Background())
	if other.Err() != nil {
		t.Fatal("StopMaintenance cancelled a rollout")
	}
	release()

	runCtx, cancel := context.WithCancel(context.Background())
	release, err = d.acquireJob(runCtx, "a maintenance operation", cancel)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		<-runCtx.Done()
		release()
	}()
	d.StopMaintenance(context.Background())
	d.cancelMu.Lock()
	held := d.job
	d.cancelMu.Unlock()
	if held != nil {
		t.Error("StopMaintenance returned before the run released the library")
	}
}
