package gui

import (
	"context"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/jobs"
)

// A background computation stops at its checkpoint while a visible one runs,
// and goes on as soon as it ends.
func TestVisibleComputation_HoldsTheBackgroundBatch(t *testing.T) {
	a := NewApp(nil)
	task := a.jobs.Start(jobs.Spec{Kind: jobs.GammonNetMissing, Library: 1, Priority: jobs.Background, Compute: true, Pausable: true})
	end := a.beginVisibleComputation()
	passed := make(chan struct{})
	go func() { task.Yield(context.Background()); close(passed) }()
	select {
	case <-passed:
		t.Fatal("the batch went on under a visible computation")
	case <-time.After(30 * time.Millisecond):
	}
	end()
	select {
	case <-passed:
	case <-time.After(5 * time.Second):
		t.Fatal("the batch stayed held once the visible computation ended")
	}
	task.Finish(nil)
	if got := a.ListJobs(); len(got) != 1 || got[0].Priority != "background" || got[0].State != jobs.Done {
		t.Fatalf("tasks %+v", got)
	}
}
