package gui

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/rollout"
	"github.com/kevung/blunderdb/pkg/blunderdb/rollouts"
)

// The rollout of one position (ADR-0060): the engine and the write live on
// *database.Database, shared with the CLI and the daemon; this file is the
// GUI shell and its events, every payload naming the position it is about:
//
//	rollout:progress  RolloutProgressEvent, after every batch of games
//	rollout:done      RolloutDoneEvent
//	rollout:cancelled RolloutDoneEvent (Result: the games finished so far, or nil; never stored)
//	rollout:error     {positionId, message}
//
// Reading what is stored: Database.LoadRollouts, or the Rollouts field of
// the analysis the panel already loads.

// RolloutRequest is one rollout asked by the GUI. PositionID names a position
// of the open database; Position carries a board that is not saved (then
// Store is refused). Settings come from RolloutPresets, edited for a custom
// rollout.
type RolloutRequest struct {
	PositionID int64            `json:"positionId"`
	Position   *domain.Position `json:"position,omitempty"`
	Settings   rollout.Settings `json:"settings"`
	Moves      []string         `json:"moves,omitempty"`
	Store      bool             `json:"store"`
}

// RolloutProgressEvent is rollout:progress: the candidates as they stand.
type RolloutProgressEvent struct {
	Job        int64               `json:"job"`
	PositionID int64               `json:"positionId"`
	Games      int                 `json:"games"`
	MaxGames   int                 `json:"maxGames"`
	Candidates []rollout.Candidate `json:"candidates"`
}

// RolloutDoneEvent is rollout:done and rollout:cancelled.
type RolloutDoneEvent struct {
	Job        int64           `json:"job"`
	PositionID int64           `json:"positionId"`
	Result     *rollout.Result `json:"result"`
	// Record is Result as the panel reads a stored rollout, so a rollout that
	// is not stored (an unsaved board) is shown by the same code.
	Record *domain.RolloutAnalysis `json:"record,omitempty"`
	Stored bool                    `json:"stored"`
}

// RolloutPreset is one of the GUI's starting points.
type RolloutPreset struct {
	Name     string           `json:"name"`
	Settings rollout.Settings `json:"settings"`
}

// RolloutPresets returns the fast and the standard presets; a custom rollout
// is either, edited.
func (a *App) RolloutPresets() []RolloutPreset {
	return []RolloutPreset{{Name: "fast", Settings: rollout.Fast()}, {Name: "standard", Settings: rollout.Standard()}}
}

// StartRollout validates req and rolls it out in the background, cancelling
// any rollout in flight; the outcome arrives as events. An invalid request is
// refused here, before anything starts.
func (a *App) StartRollout(req RolloutRequest) (int64, error) {
	if err := req.Settings.Validate(); err != nil {
		return 0, err
	}
	switch {
	case req.PositionID == 0 && req.Position == nil:
		return 0, errors.New("rollout: no position")
	case req.Store && req.PositionID == 0:
		return 0, errors.New("rollout: only a saved position can store its rollout")
	case req.PositionID != 0 && a.db == nil:
		return 0, errors.New("rollout: no database is open")
	case req.Store:
		if err := a.db.RefuseReadOnly(); err != nil {
			return 0, err
		}
	}

	ctx, stopped, job := a.beginRollout(RolloutStatus{Running: true, Kind: "position", PositionID: req.PositionID, MaxGames: req.Settings.MaxGames})

	go func() {
		defer close(stopped)
		defer recoverBackground(a.ctx, "rollout")
		progress := func(p rollout.Progress) {
			a.noteRollout(stopped, func(st *RolloutStatus) { st.Games, st.MaxGames = p.Games, p.MaxGames })
			a.emitBatch("rollout:progress", RolloutProgressEvent{Job: job, PositionID: req.PositionID, Games: p.Games, MaxGames: p.MaxGames, Candidates: p.Candidates})
		}
		var (
			res *rollout.Result
			err error
		)
		if req.PositionID != 0 {
			res, err = a.db.RolloutPosition(ctx, req.PositionID, req.Settings, req.Moves, req.Store, progress)
		} else {
			res, err = rollout.Run(ctx, *req.Position, req.Settings, rollout.Options{Moves: req.Moves, Progress: progress})
		}

		a.endRollout(stopped)

		switch {
		case err == nil:
			done := RolloutDoneEvent{Job: job, PositionID: req.PositionID, Result: res, Stored: req.Store}
			if res != nil {
				rec := res.Record(time.Now())
				done.Record = &rec
			}
			a.emitBatch("rollout:done", done)
		case ctx.Err() != nil:
			a.emitBatch("rollout:cancelled", RolloutDoneEvent{Job: job, PositionID: req.PositionID, Result: res})
		default:
			a.emitBatch("rollout:error", map[string]any{"job": job, "positionId": req.PositionID, "message": fmt.Sprint(err)})
		}
	}()
	return job, nil
}

// StartRolloutFiltered rolls out, one after the other, every position of the
// open database that query selects and that carries no rollout with
// settings' Signature yet, cancelling any rollout in flight. Each is written
// as it finishes, so cancelling keeps what was done and a rerun resumes.
// It is refused while a gammonNet batch runs: the two batches each take
// every core. An invalid request is refused here; the outcome arrives as
// events:
//
//	rollout-batch:started   {total}, once the positions are gathered
//	rollout-batch:progress  RolloutBatchProgressEvent, after every batch of games
//	rollout-batch:done      RolloutBatchDoneEvent
//	rollout-batch:cancelled RolloutBatchDoneEvent (what was done before)
//	rollout-batch:error     {message}
func (a *App) StartRolloutFiltered(query string, settings rollout.Settings) (int64, error) {
	if err := settings.Validate(); err != nil {
		return 0, err
	}
	if a.db == nil {
		return 0, errors.New("rollout: no database is open")
	}
	if err := a.db.RefuseReadOnly(); err != nil {
		return 0, err
	}
	filters, err := rollouts.ParseQuery(query)
	if err != nil {
		return 0, err
	}
	return a.startRolloutBatch(settings, func(ctx context.Context) (*database.RolloutPlan, error) {
		return a.db.PlanRollout(ctx, filters, settings)
	})
}

// StartRolloutIDs is StartRolloutFiltered for the list the GUI shows: the
// positions ids names, in that order, that carry no rollout of settings'
// Signature yet. The events are those of StartRolloutFiltered.
func (a *App) StartRolloutIDs(ids []int64, settings rollout.Settings) (int64, error) {
	if err := settings.Validate(); err != nil {
		return 0, err
	}
	if a.db == nil {
		return 0, errors.New("rollout: no database is open")
	}
	if err := a.db.RefuseReadOnly(); err != nil {
		return 0, err
	}
	return a.startRolloutBatch(settings, func(ctx context.Context) (*database.RolloutPlan, error) {
		return a.db.PlanRolloutIDs(ctx, ids, settings)
	})
}

// startRolloutBatch runs the batch gather selects, refusing while a gammonNet
// batch runs.
func (a *App) startRolloutBatch(settings rollout.Settings, gather func(context.Context) (*database.RolloutPlan, error)) (int64, error) {
	a.roMu.Lock()
	a.gnBatchMu.Lock()
	gammonNetRunning := a.gnBatchDone != nil
	a.gnBatchMu.Unlock()
	a.roMu.Unlock()
	if gammonNetRunning {
		return 0, errors.New("rollout: a gammonNet batch analysis is already running")
	}

	ctx, stopped, job := a.beginRollout(RolloutStatus{Running: true, Kind: "batch", MaxGames: settings.MaxGames})
	go func() {
		defer close(stopped)
		defer recoverBackground(a.ctx, "rollout batch")
		var sum rollouts.Summary
		plan, err := gather(ctx)
		if err == nil {
			a.noteRollout(stopped, func(st *RolloutStatus) { st.Total = len(plan.Positions) })
			a.emitBatch("rollout-batch:started", map[string]int64{"job": job, "total": int64(len(plan.Positions))})
			sum, err = a.db.RunRolloutPlan(ctx, plan, settings, func(p rollouts.Progress) {
				a.noteRollout(stopped, func(st *RolloutStatus) {
					st.Done, st.Total, st.PositionID, st.Games, st.MaxGames = p.Done, p.Total, p.PositionID, p.Games, p.MaxGames
				})
				a.emitBatch("rollout-batch:progress", RolloutBatchProgressEvent{Job: job, Done: p.Done, Total: p.Total,
					PositionID: p.PositionID, Games: p.Games, MaxGames: p.MaxGames})
			})
		}
		a.endRollout(stopped)

		done := RolloutBatchDoneEvent{Job: job, Total: sum.Total, RolledOut: sum.RolledOut, Refused: sum.Refused, Failed: sum.Failed, Signature: settings.Signature()}
		switch {
		case err != nil && ctx.Err() == nil:
			a.emitBatch("rollout-batch:error", map[string]any{"job": job, "message": fmt.Sprint(err)})
		case sum.Cancelled || ctx.Err() != nil:
			a.emitBatch("rollout-batch:cancelled", done)
		default:
			a.emitBatch("rollout-batch:done", done)
		}
	}()
	return job, nil
}

// CountRolloutIDs says how many of the positions ids names would be rolled
// out by StartRolloutIDs (those without a rollout of settings' Signature), so
// the GUI can ask before a long job starts.
func (a *App) CountRolloutIDs(ids []int64, settings rollout.Settings) (int, error) {
	if err := settings.Validate(); err != nil {
		return 0, err
	}
	if a.db == nil {
		return 0, errors.New("rollout: no database is open")
	}
	positions, err := a.db.PositionsToRolloutIDs(a.batchCtx(), ids, settings)
	return len(positions), err
}

// RolloutBatchProgressEvent is rollout-batch:progress: Done positions are
// finished, the one named is at Games of MaxGames.
type RolloutBatchProgressEvent struct {
	Job        int64 `json:"job"`
	Done       int   `json:"done"`
	Total      int   `json:"total"`
	PositionID int64 `json:"positionId"`
	Games      int   `json:"games"`
	MaxGames   int   `json:"maxGames"`
}

// RolloutBatchDoneEvent is rollout-batch:done and rollout-batch:cancelled.
type RolloutBatchDoneEvent struct {
	Job       int64  `json:"job"`
	Total     int    `json:"total"`
	RolledOut int    `json:"rolledOut"`
	Refused   int    `json:"refused"`
	Failed    int    `json:"failed"`
	Signature string `json:"signature"`
}

// RolloutStatus is what RolloutStatus reports: whether a rollout runs, of one
// position ("position") or of a query ("batch"), and how far it is. For a
// batch, Done of Total positions are behind and PositionID is the one in
// hand, at Games of MaxGames.
type RolloutStatus struct {
	Job        int64  `json:"job"`
	Running    bool   `json:"running"`
	Kind       string `json:"kind,omitempty"`
	PositionID int64  `json:"positionId,omitempty"`
	Done       int    `json:"done"`
	Total      int    `json:"total"`
	Games      int    `json:"games"`
	MaxGames   int    `json:"maxGames"`
}

// RolloutStatus reports the rollout in flight, so a view opened mid-way
// shows it without waiting for the next event.
func (a *App) RolloutStatus() RolloutStatus {
	a.roMu.Lock()
	defer a.roMu.Unlock()
	if a.roDone == nil {
		return RolloutStatus{}
	}
	return a.roStatus
}

// beginRollout cancels the rollout in flight and registers the next one: its
// context, the channel its goroutine closes when it has stopped, and its job
// number.
func (a *App) beginRollout(status RolloutStatus) (context.Context, chan struct{}, int64) {
	a.roMu.Lock()
	defer a.roMu.Unlock()
	if a.roCancel != nil {
		a.roCancel()
	}
	ctx, cancel := context.WithCancel(a.batchCtx())
	stopped := make(chan struct{})
	a.roSeq++
	status.Job = a.roSeq
	a.roCancel, a.roDone, a.roStatus = cancel, stopped, status
	return ctx, stopped, status.Job
}

// noteRollout updates the status of the rollout stopped names, unless a newer
// one has replaced it.
func (a *App) noteRollout(stopped chan struct{}, change func(*RolloutStatus)) {
	a.roMu.Lock()
	defer a.roMu.Unlock()
	if a.roDone == stopped {
		change(&a.roStatus)
	}
}

// endRollout forgets stopped unless a newer rollout has replaced it.
func (a *App) endRollout(stopped chan struct{}) {
	a.roMu.Lock()
	defer a.roMu.Unlock()
	if a.roDone == stopped {
		a.roCancel, a.roDone, a.roStatus = nil, nil, RolloutStatus{}
	}
}

// rolloutBatchRunning reports a rollout of a query in flight; the caller
// holds roMu.
func (a *App) rolloutBatchRunning() bool {
	return a.roDone != nil && a.roStatus.Kind == "batch"
}

// CancelRollout stops the rollout, or the batch, in flight and returns at once; it is not
// stored.
func (a *App) CancelRollout() {
	a.cancelRollout()
}

// cancelRollout also returns a channel closed once the rollout has stopped,
// nil when none was in flight.
func (a *App) cancelRollout() <-chan struct{} {
	a.roMu.Lock()
	defer a.roMu.Unlock()
	if a.roCancel != nil {
		a.roCancel()
		a.roCancel = nil
	}
	return a.roDone
}
