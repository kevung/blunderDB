package gui

import (
	"context"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
)

// recordingBind stands for the bound *database.Database, recording when it
// was closed.
type recordingBind struct {
	closed  chan struct{}
	closeAt func()
}

func (b *recordingBind) Close() error {
	if b.closeAt != nil {
		b.closeAt()
	}
	close(b.closed)
	return nil
}

// TestShutdownCancelsJobsBeforeClosingTheBinds: jobs stop before the
// database closes (ADR-0045 §8).
func TestShutdownCancelsJobsBeforeClosingTheBinds(t *testing.T) {
	a := NewApp(nil)
	a.ctx = context.Background()

	cancelled := make(chan struct{})
	stopped := make(chan struct{})
	var cancelledBeforeClose bool

	// A simulated batch in flight, closing its stopped channel after cancel.
	a.gnBatchMu.Lock()
	a.gnBatchCancel = func() {
		close(cancelled)
		go func() {
			time.Sleep(20 * time.Millisecond) // the position still mid-search
			close(stopped)
		}()
	}
	a.gnBatchDone = stopped
	a.gnBatchMu.Unlock()

	bind := &recordingBind{closed: make(chan struct{}), closeAt: func() {
		select {
		case <-cancelled:
			cancelledBeforeClose = true
		default:
		}
	}}

	shutdown(a, []interface{}{bind})(context.Background())

	select {
	case <-bind.closed:
	default:
		t.Fatal("shutdown returned without closing the bind")
	}
	if !cancelledBeforeClose {
		t.Error("the bind was closed before the batch was cancelled: shutdown still closes the database under a running job")
	}
	select {
	case <-stopped:
	default:
		t.Error("shutdown closed the bind while the batch had not actually stopped: cancelling is not waiting")
	}
}

// TestShutdownWithoutJobsClosesImmediately: the ordinary quit — nothing
// running — must not pay the grace period, nor block on a nil channel.
func TestShutdownWithoutJobsClosesImmediately(t *testing.T) {
	a := NewApp(nil)
	a.ctx = context.Background()

	bind := &recordingBind{closed: make(chan struct{})}

	done := make(chan struct{})
	go func() {
		shutdown(a, []interface{}{bind})(context.Background())
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown blocked although no job was running")
	}
	select {
	case <-bind.closed:
	default:
		t.Fatal("shutdown returned without closing the bind")
	}
}

// TestStopBackgroundJobsGivesUpAfterTheGrace: a job that will not stop must
// not hold the window open forever — the close goes ahead, with a warning.
func TestStopBackgroundJobsGivesUpAfterTheGrace(t *testing.T) {
	a := NewApp(nil)
	a.ctx = context.Background()

	a.gnBatchMu.Lock()
	a.gnBatchCancel = func() {}
	a.gnBatchDone = make(chan struct{}) // never closed: the job never stops
	a.gnBatchMu.Unlock()

	start := time.Now()
	a.stopBackgroundJobs(50 * time.Millisecond)
	if elapsed := time.Since(start); elapsed < 50*time.Millisecond {
		t.Errorf("stopBackgroundJobs returned after %s, before its own grace period", elapsed)
	} else if elapsed > 2*time.Second {
		t.Errorf("stopBackgroundJobs waited %s for a job that never stops", elapsed)
	}
}

// TestRunGammonNetBatchSignalsItsEnd: the shell publishes the channel
// stopBackgroundJobs waits on, closed when the batch is done.
func TestRunGammonNetBatchSignalsItsEnd(t *testing.T) {
	// The shell refuses to start without a Database; run never touches it.
	d := database.NewDatabase()
	if err := d.SetupDatabase(":memory:"); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	// No a.ctx: the shell must run, and emit nothing, without a window.
	a := NewApp(d)

	release := make(chan struct{})
	a.runGammonNetBatch(func(ctx context.Context, _ func(done, total int)) (database.GammonNetBatchSummary, error) {
		<-release
		return database.GammonNetBatchSummary{}, ctx.Err()
	})

	// Give the shell time to register itself, then check it is in flight.
	deadline := time.Now().Add(2 * time.Second)
	var stopped <-chan struct{}
	for time.Now().Before(deadline) {
		a.gnBatchMu.Lock()
		stopped = a.gnBatchDone
		a.gnBatchMu.Unlock()
		if stopped != nil {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if stopped == nil {
		t.Fatal("the batch shell never published its stopped channel")
	}
	select {
	case <-stopped:
		t.Fatal("the stopped channel was closed while the batch was still running")
	default:
	}

	close(release)
	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("the stopped channel was never closed after the batch finished")
	}

	a.gnBatchMu.Lock()
	defer a.gnBatchMu.Unlock()
	if a.gnBatchCancel != nil || a.gnBatchDone != nil {
		t.Error("the finished batch left itself registered as in flight")
	}
}
