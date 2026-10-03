package gui

import (
	"context"
	"errors"
	"fmt"

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
// Store is refused). Settings come from RolloutPresets, edited for « Libre ».
type RolloutRequest struct {
	PositionID int64            `json:"positionId"`
	Position   *domain.Position `json:"position,omitempty"`
	Settings   rollout.Settings `json:"settings"`
	Moves      []string         `json:"moves,omitempty"`
	Store      bool             `json:"store"`
}

// RolloutProgressEvent is rollout:progress: the candidates as they stand.
type RolloutProgressEvent struct {
	PositionID int64               `json:"positionId"`
	Games      int                 `json:"games"`
	MaxGames   int                 `json:"maxGames"`
	Candidates []rollout.Candidate `json:"candidates"`
}

// RolloutDoneEvent is rollout:done and rollout:cancelled.
type RolloutDoneEvent struct {
	PositionID int64           `json:"positionId"`
	Result     *rollout.Result `json:"result"`
	Stored     bool            `json:"stored"`
}

// RolloutPreset is one of the GUI's starting points.
type RolloutPreset struct {
	Name     string           `json:"name"`
	Settings rollout.Settings `json:"settings"`
}

// RolloutPresets returns « Rapide » (fast) and « Standard »; « Libre » is
// either, edited.
func (a *App) RolloutPresets() []RolloutPreset {
	return []RolloutPreset{{Name: "fast", Settings: rollout.Fast()}, {Name: "standard", Settings: rollout.Standard()}}
}

// StartRollout validates req and rolls it out in the background, cancelling
// any rollout in flight; the outcome arrives as events. An invalid request is
// refused here, before anything starts.
func (a *App) StartRollout(req RolloutRequest) error {
	if err := req.Settings.Validate(); err != nil {
		return err
	}
	switch {
	case req.PositionID == 0 && req.Position == nil:
		return errors.New("rollout: no position")
	case req.Store && req.PositionID == 0:
		return errors.New("rollout: only a saved position can store its rollout")
	case req.PositionID != 0 && a.db == nil:
		return errors.New("rollout: no database is open")
	}

	ctx, stopped := a.beginRollout()

	go func() {
		defer close(stopped)
		defer recoverBackground(a.ctx, "rollout")
		progress := func(p rollout.Progress) {
			a.emitBatch("rollout:progress", RolloutProgressEvent{PositionID: req.PositionID, Games: p.Games, MaxGames: p.MaxGames, Candidates: p.Candidates})
		}
		var (
			res *rollout.Result
			err error
		)
		if req.PositionID != 0 {
			res, err = database.RolloutPosition(ctx, a.db, req.PositionID, req.Settings, req.Moves, req.Store, progress)
		} else {
			res, err = rollout.Run(ctx, *req.Position, req.Settings, rollout.Options{Moves: req.Moves, Progress: progress})
		}

		a.endRollout(stopped)

		switch {
		case err == nil:
			a.emitBatch("rollout:done", RolloutDoneEvent{PositionID: req.PositionID, Result: res, Stored: req.Store})
		case ctx.Err() != nil:
			a.emitBatch("rollout:cancelled", RolloutDoneEvent{PositionID: req.PositionID, Result: res})
		default:
			a.emitBatch("rollout:error", map[string]any{"positionId": req.PositionID, "message": fmt.Sprint(err)})
		}
	}()
	return nil
}

// StartRolloutFiltered rolls out, one after the other, every position of the
// open database that query selects and that carries no rollout with
// settings' Signature yet, cancelling any rollout in flight. Each is written
// as it finishes, so cancelling keeps what was done and a rerun resumes.
// An invalid request is refused here; the outcome arrives as events:
//
//	rollout-batch:progress  RolloutBatchProgressEvent, after every batch of games
//	rollout-batch:done      RolloutBatchDoneEvent
//	rollout-batch:cancelled RolloutBatchDoneEvent (what was done before)
//	rollout-batch:error     {message}
func (a *App) StartRolloutFiltered(query string, settings rollout.Settings) error {
	if err := settings.Validate(); err != nil {
		return err
	}
	if a.db == nil {
		return errors.New("rollout: no database is open")
	}
	filters, err := rollouts.ParseQuery(query)
	if err != nil {
		return err
	}

	ctx, stopped := a.beginRollout()
	go func() {
		defer close(stopped)
		defer recoverBackground(a.ctx, "rollout batch")
		sum, err := database.RolloutFiltered(ctx, a.db, filters, settings, func(p rollouts.Progress) {
			a.emitBatch("rollout-batch:progress", RolloutBatchProgressEvent{Done: p.Done, Total: p.Total,
				PositionID: p.PositionID, Games: p.Games, MaxGames: p.MaxGames})
		})
		a.endRollout(stopped)

		done := RolloutBatchDoneEvent{Total: sum.Total, RolledOut: sum.RolledOut, Refused: sum.Refused, Failed: sum.Failed, Signature: settings.Signature()}
		switch {
		case err != nil && ctx.Err() == nil:
			a.emitBatch("rollout-batch:error", map[string]any{"message": fmt.Sprint(err)})
		case sum.Cancelled || ctx.Err() != nil:
			a.emitBatch("rollout-batch:cancelled", done)
		default:
			a.emitBatch("rollout-batch:done", done)
		}
	}()
	return nil
}

// RolloutBatchProgressEvent is rollout-batch:progress: Done positions are
// finished, the one named is at Games of MaxGames.
type RolloutBatchProgressEvent struct {
	Done       int   `json:"done"`
	Total      int   `json:"total"`
	PositionID int64 `json:"positionId"`
	Games      int   `json:"games"`
	MaxGames   int   `json:"maxGames"`
}

// RolloutBatchDoneEvent is rollout-batch:done and rollout-batch:cancelled.
type RolloutBatchDoneEvent struct {
	Total     int    `json:"total"`
	RolledOut int    `json:"rolledOut"`
	Refused   int    `json:"refused"`
	Failed    int    `json:"failed"`
	Signature string `json:"signature"`
}

// beginRollout cancels the rollout in flight and registers the next one:
// its context, and the channel its goroutine closes when it has stopped.
func (a *App) beginRollout() (context.Context, chan struct{}) {
	a.roMu.Lock()
	defer a.roMu.Unlock()
	if a.roCancel != nil {
		a.roCancel()
	}
	ctx, cancel := context.WithCancel(a.batchCtx())
	stopped := make(chan struct{})
	a.roCancel, a.roDone = cancel, stopped
	return ctx, stopped
}

// endRollout forgets stopped unless a newer rollout has replaced it.
func (a *App) endRollout(stopped chan struct{}) {
	a.roMu.Lock()
	defer a.roMu.Unlock()
	if a.roDone == stopped {
		a.roCancel, a.roDone = nil, nil
	}
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
