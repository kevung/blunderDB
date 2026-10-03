package gui

import (
	"context"
	"errors"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/rollout"
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

	a.roMu.Lock()
	if a.roCancel != nil {
		a.roCancel()
	}
	ctx, cancel := context.WithCancel(a.batchCtx())
	stopped := make(chan struct{})
	a.roCancel, a.roDone = cancel, stopped
	a.roMu.Unlock()

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
			res, err = a.db.RolloutPosition(ctx, req.PositionID, req.Settings, req.Moves, req.Store, progress)
		} else {
			res, err = rollout.Run(ctx, *req.Position, req.Settings, rollout.Options{Moves: req.Moves, Progress: progress})
		}

		a.roMu.Lock()
		if a.roDone == stopped {
			a.roCancel, a.roDone = nil, nil
		}
		a.roMu.Unlock()

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

// CancelRollout stops the rollout in flight and returns at once; it is not
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
