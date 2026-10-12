package gui

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/rollout"
)

// queueBehindRollout opens a file database holding one unanalysed position,
// starts a stored rollout that holds the library until the returned release
// is called, and queues the automatic gammonNet batch behind it.
func queueBehindRollout(t *testing.T) (a *App, d *database.Database, release func(), stopped <-chan struct{}) {
	t.Helper()
	dir := t.TempDir()
	d = database.NewDatabase()
	if err := d.SetupDatabase(filepath.Join(dir, "a.db")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	a = NewApp(d)

	p := domain.InitializePosition()
	id, err := d.SavePosition(&p)
	if err != nil {
		t.Fatal(err)
	}
	s := rollout.Fast()
	s.MaxGames, s.MinGames, s.Truncation, s.Candidates = 36, 36, 2, 2
	holding, unblock := make(chan struct{}), make(chan struct{})
	var once sync.Once
	rolloutDone := make(chan struct{})
	go func() {
		defer close(rolloutDone)
		_, _ = d.RolloutPosition(context.Background(), id, s, nil, true, func(rollout.Progress) {
			once.Do(func() { close(holding); <-unblock })
		})
	}()
	<-holding
	var unblockOnce sync.Once
	release = func() { unblockOnce.Do(func() { close(unblock) }); <-rolloutDone }
	t.Cleanup(release)

	a.QueueGammonNetBatch(0, 0, 0)
	a.gnBatchMu.Lock()
	stopped = a.gnBatchDone
	a.gnBatchMu.Unlock()
	if stopped == nil {
		t.Fatal("the queued batch was not registered")
	}
	select {
	case <-stopped:
		t.Fatal("the queued batch ended while the rollout held the library")
	case <-time.After(50 * time.Millisecond):
	}
	return a, d, release, stopped
}

func waitStopped(t *testing.T, stopped <-chan struct{}) {
	t.Helper()
	select {
	case <-stopped:
	case <-time.After(30 * time.Second):
		t.Fatal("the queued batch never ended")
	}
}

// An import's batch started during a stored rollout runs once it ends.
func TestQueuedGammonNetBatch_RunsAfterTheRollout(t *testing.T) {
	_, d, release, stopped := queueBehindRollout(t)
	if n, _ := d.CountPositionsWithoutAnalysis(); n != 1 {
		t.Fatalf("%d positions without analysis during the wait, want 1", n)
	}
	release()
	waitStopped(t, stopped)
	if n, err := d.CountPositionsWithoutAnalysis(); err != nil || n != 0 {
		t.Errorf("%d positions without analysis after the batch (%v), want 0", n, err)
	}
}

// Cancelling a waiting batch ends it at once, the rollout still running.
func TestQueuedGammonNetBatch_CancelledWhileWaiting(t *testing.T) {
	a, d, _, stopped := queueBehindRollout(t)
	a.CancelGammonNetBatch()
	waitStopped(t, stopped)
	if n, _ := d.CountPositionsWithoutAnalysis(); n != 1 {
		t.Errorf("the cancelled batch analysed %d positions", 1-n)
	}
}

// Opening another file stops a waiting batch before the switch.
func TestQueuedGammonNetBatch_StoppedByADatabaseSwitch(t *testing.T) {
	_, d, _, stopped := queueBehindRollout(t)
	if err := d.SetupDatabase(filepath.Join(t.TempDir(), "b.db")); err != nil {
		t.Fatal(err)
	}
	waitStopped(t, stopped)
}
