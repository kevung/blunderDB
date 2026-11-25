package gui

import (
	"context"
	"log/slog"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/jobs"
)

// The registry of durable tasks (ADR-0115): every task that outlives a click
// is listed in a.jobs, and the frontend follows it on one stream:
//
//	jobs:changed jobs.Info, when a task is registered or changes
//	jobs:removed {id}, when a task is no longer listed
//
// The events each task sent before the registry (gammonnet-batch:*,
// rollout:*, rollout-batch:*, maintenance:*, the import progress, bearoff:*,
// sync:status) are still sent: the panels that read them stay as they are.

// ListJobs returns every task listed, running, waiting or finished.
func (a *App) ListJobs() []jobs.Info { return a.jobs.List() }

// CancelJob stops a task; it reports its end on jobs:changed.
func (a *App) CancelJob(id int64) error { return a.jobs.Cancel(id) }

// PauseJob stops a task at its next checkpoint, or keeps a waiting one from
// taking its turn.
func (a *App) PauseJob(id int64) error { return a.jobs.Pause(id) }

// ResumeJob lets a paused task go on.
func (a *App) ResumeJob(id int64) error { return a.jobs.Resume(id) }

// MoveJob puts a waiting task at index (0 first) of its library's queue.
func (a *App) MoveJob(id int64, index int) error { return a.jobs.Move(id, index) }

// RemoveJob unlists a finished task, or takes a waiting one off its queue.
func (a *App) RemoveJob(id int64) error { return a.jobs.Remove(id) }

// beginVisibleComputation marks a computation whose result is on screen —
// the evaluation, the maps of rolls and of scores — as running until end is
// called: the durable tasks of lower priority yield to it at their
// checkpoints, without polling.
func (a *App) beginVisibleComputation() (end func()) { return a.jobs.EnterVisible() }

// jobCode is the job code of the database's BusyError for a task's kind: the
// one the frontend already names in gammonnet-batch:waiting.
func jobCode(k jobs.Kind) string {
	switch k {
	case jobs.GammonNetMissing, jobs.GammonNetStale, jobs.GammonNetMatch:
		return database.JobGammonNet
	case jobs.Rollout, jobs.RolloutBatch:
		return database.JobRollout
	case jobs.Maintenance, jobs.MatchStatsFill:
		return database.JobMaintenance
	}
	return string(k)
}

// stopRegisteredJobs cancels the tasks of group (0: every library) and waits
// for them, until deadline.
func (a *App) stopRegisteredJobs(deadline context.Context, group int, grace time.Duration) {
	for _, late := range a.jobs.Stop(deadline, group) {
		slog.Warn("a job writing the database did not stop within the grace period; going ahead",
			"job", string(late.Kind), "id", late.ID, "grace", grace)
	}
}
