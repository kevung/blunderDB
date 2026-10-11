package database

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
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
		// Closed first: once release runs, the maintenance may return before
		// this goroutine is scheduled again.
		close(finished)
		release()
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

// A refusal names the job holding the library, and is a storage conflict.
func TestAcquireJob_RefusalNamesTheHolder(t *testing.T) {
	t.Parallel()
	d := NewDatabase()
	release, err := d.acquireJob(context.Background(), JobRollout, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	_, err = d.acquireJob(context.Background(), JobGammonNet, nil)
	var busy *BusyError
	if !errors.As(err, &busy) || busy.Job != JobRollout {
		t.Fatalf("refusal %v, want a BusyError naming the rollout", err)
	}
	if !errors.Is(err, ErrDatabaseBusy) || !errors.Is(err, storage.ErrConflict) {
		t.Errorf("%v is not ErrDatabaseBusy and ErrConflict", err)
	}
	if err.Error() != "database is busy: rollout" {
		t.Errorf("message %q", err.Error())
	}
}

// startQueuedBatch runs AnalyzeMissingWithGammonNet under a queued ctx while
// a stored rollout holds the library, and returns once the batch waits.
func startQueuedBatch(t *testing.T, d *Database, ctx context.Context) (releaseRollout func(), result <-chan error, summary *GammonNetBatchSummary) {
	t.Helper()
	p := racePosition(8, 17, domain.White)
	if _, err := d.SavePosition(&p); err != nil {
		t.Fatal(err)
	}
	releaseRollout, err := d.acquireJob(context.Background(), JobRollout, nil)
	if err != nil {
		t.Fatal(err)
	}
	waiting := make(chan string, 1)
	done := make(chan error, 1)
	summary = new(GammonNetBatchSummary)
	go func() {
		s, err := d.AnalyzeMissingWithGammonNet(WaitForDatabase(ctx, func(job string) { waiting <- job }), 0, 0, 0, 1, nil, nil)
		*summary = s
		done <- err
	}()
	select {
	case job := <-waiting:
		if job != JobRollout {
			t.Errorf("waits behind %q, want %q", job, JobRollout)
		}
	case err := <-done:
		t.Fatalf("the queued batch did not wait: %v", err)
	case <-time.After(10 * time.Second):
		t.Fatal("the queued batch neither waited nor ran")
	}
	return releaseRollout, done, summary
}

// An automatic batch started during a stored rollout runs once it ends.
func TestQueuedBatch_RunsAfterTheRollout(t *testing.T) {
	t.Parallel()
	d := newBatchTestDB(t)
	releaseRollout, done, summary := startQueuedBatch(t, d, context.Background())
	select {
	case err := <-done:
		t.Fatalf("the batch ran during the rollout: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	releaseRollout()
	if err := <-done; err != nil {
		t.Fatalf("queued batch: %v", err)
	}
	if summary.Evaluated != 1 {
		t.Errorf("summary %+v, want one position evaluated", *summary)
	}
}

// Cancelling a waiting batch ends it without touching the job ahead.
func TestQueuedBatch_CancelledWhileWaiting(t *testing.T) {
	t.Parallel()
	d := newBatchTestDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	releaseRollout, done, _ := startQueuedBatch(t, d, ctx)
	defer releaseRollout()
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled wait: %v, want context.Canceled", err)
	}
	d.cancelMu.Lock()
	held := d.job
	d.cancelMu.Unlock()
	if held == nil || held.name != JobRollout {
		t.Error("cancelling the waiting batch disturbed the rollout")
	}
}
