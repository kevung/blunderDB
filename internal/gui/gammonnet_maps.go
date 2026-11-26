package gui

import (
	"context"
	"errors"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
)

// The roll and score maps' desktop half. Each call returns at once and
// reports through events carrying the caller's tag: the whole map at 0-ply,
// then one event per refined row or cell, then done. They take the at-rest
// search's slot (beginInteractiveSearch): the batch yields to them as it does
// to the panel, and a newer interactive search supersedes them. Nothing is
// written to the library; the phases already computed are served from
// gammonnet.Maps(), the cache the CLI and the daemon go through as well.

// RollMapRowEvent is the payload of "gammonnet-rolls:row".
type RollMapRowEvent struct {
	Tag   int                  `json:"tag"`
	Index int                  `json:"index"`
	Row   gammonnet.RollMapRow `json:"row"`
}

// RollMapDoneEvent is the payload of "gammonnet-rolls:done".
type RollMapDoneEvent struct {
	Tag int               `json:"tag"`
	Map gammonnet.RollMap `json:"map"`
}

// StartRollMap computes the 21 rolls of pos, whose player on roll has not
// rolled, at 0-ply then at ply. Emits "gammonnet-rolls:row"
// (RollMapRowEvent) per row and depth, then "gammonnet-rolls:done"
// (RollMapDoneEvent), or "gammonnet-rolls:cancelled" {tag} /
// "gammonnet-rolls:error" {tag, message}.
func (a *App) StartRollMap(pos domain.Position, ply, pruneK, tag int) {
	a.startRollMap(pos, ply, pruneK, tag)
}

func (a *App) startRollMap(pos domain.Position, ply, pruneK, tag int) {
	if err := gammonnet.ValidateMapPly(ply); err != nil {
		a.emitBatch("gammonnet-rolls:error", map[string]any{"tag": tag, "message": err.Error()})
		return
	}
	ctx, end := a.beginInteractiveSearch(&a.gnRollMapSeq)
	go func() {
		defer recoverBackground(a.ctx, "gammonNet roll map")
		defer end()
		met, err := a.currentMET()
		var m gammonnet.RollMap
		if err == nil {
			m, err = gammonnet.Maps().RollMap(ctx, "", pos, met, ply, pruneK, gammonnet.LocalExec(0, ply, pruneK),
				func(i int, row gammonnet.RollMapRow) {
					a.emitBatch("gammonnet-rolls:row", RollMapRowEvent{Tag: tag, Index: i, Row: row})
				})
		}
		a.finishMap("gammonnet-rolls", tag, end, err, func() any { return RollMapDoneEvent{Tag: tag, Map: m} })
	}()
}

// CancelRollMap stops the roll map in flight, if it is still the interactive
// search running; the panel's own search is left alone.
func (a *App) CancelRollMap() {
	a.cancelInteractiveSearch(&a.gnRollMapSeq)
}

// RollMapPrefetchEvent is the payload of "gammonnet-rolls:prefetched": the
// child of row Index is held up to Ply, so descending it answers at once.
type RollMapPrefetchEvent struct {
	Tag   int `json:"tag"`
	Index int `json:"index"`
	Ply   int `json:"ply"`
}

// PrefetchRollMaps computes, behind the roll map on screen (pos and its
// rows, at ply), the maps one roll below it: every child at 0 then 1 ply,
// the child of row focus (-1: none) at ply. It runs in the background — the
// computation on screen keeps its cores (ADR-0120) — into gammonnet.Maps(),
// where the descent finds it. A different request cancels the one running;
// the same request again is the run in flight. Emits
// "gammonnet-rolls:prefetched" (RollMapPrefetchEvent) per child and depth.
func (a *App) PrefetchRollMaps(pos domain.Position, rows []gammonnet.RollMapRow, ply, pruneK, focus, tag int) {
	if gammonnet.ValidateMapPly(ply) != nil {
		return
	}
	met, err := a.currentMET()
	if err != nil {
		return
	}
	req := gammonnet.PrefetchRequest{Pos: pos, Rows: rows, MET: met, Ply: ply, PruneK: pruneK, Focus: focus}
	a.rollMapPrefetcher().Start(a.batchCtx(), req, func(i, depth int) {
		a.emitBatch("gammonnet-rolls:prefetched", RollMapPrefetchEvent{Tag: tag, Index: i, Ply: depth})
	})
}

// CancelRollMapPrefetch stops the prefetch running, if any: the branch it
// was for is no longer the one on screen.
func (a *App) CancelRollMapPrefetch() {
	a.rollMapPrefetcher().Stop()
}

func (a *App) rollMapPrefetcher() *gammonnet.Prefetcher {
	a.gnPrefetchOnce.Do(func() {
		a.gnPrefetch = gammonnet.NewPrefetcher(gammonnet.Maps(), "", gammonnet.BackgroundExec(gammonnet.ProcessCores(), 0),
			func(r any) { reportPanic(a.ctx, "gammonNet roll map prefetch", r) })
	})
	return a.gnPrefetch
}

// currentMET is the library's table (ADR-0068), nil for the built-in one.
func (a *App) currentMET() (*engine.MET, error) {
	if a.db == nil {
		return nil, nil
	}
	return database.CurrentMET(a.db)
}

// finishMap emits a map's last event: cancelled when superseded or stopped,
// error, or done with the payload done builds.
func (a *App) finishMap(prefix string, tag int, end func() bool, err error, done func() any) {
	if end() || errors.Is(err, context.Canceled) {
		a.emitBatch(prefix+":cancelled", map[string]int{"tag": tag})
		return
	}
	if err != nil {
		a.emitBatch(prefix+":error", map[string]any{"tag": tag, "message": err.Error()})
		return
	}
	a.emitBatch(prefix+":done", done())
}

// ScoreMapCellEvent is the payload of "gammonnet-scoremap:cell".
type ScoreMapCellEvent struct {
	Tag   int                    `json:"tag"`
	Index int                    `json:"index"`
	Cell  gammonnet.ScoreMapCell `json:"cell"`
}

// ScoreMapDoneEvent is the payload of "gammonnet-scoremap:done".
type ScoreMapDoneEvent struct {
	Tag int                `json:"tag"`
	Map gammonnet.ScoreMap `json:"map"`
}

// StartScoreMap computes pos at every away × away score of a
// matchLength-point match — the best play with dice, the cube verdict
// without — at 0-ply, then at ply from the position's score outwards. Emits
// "gammonnet-scoremap:cell" (ScoreMapCellEvent) per cell and depth, then
// "gammonnet-scoremap:done" (ScoreMapDoneEvent), or
// "gammonnet-scoremap:cancelled" {tag} / "gammonnet-scoremap:error"
// {tag, message}.
func (a *App) StartScoreMap(pos domain.Position, matchLength, ply, pruneK, tag int) {
	if err := validateScoreMap(matchLength, ply); err != nil {
		a.emitBatch("gammonnet-scoremap:error", map[string]any{"tag": tag, "message": err.Error()})
		return
	}
	ctx, end := a.beginInteractiveSearch(&a.gnScoreMapSeq)
	go func() {
		defer recoverBackground(a.ctx, "gammonNet score map")
		defer end()
		met, err := a.currentMET()
		var m gammonnet.ScoreMap
		if err == nil {
			m, err = gammonnet.Maps().ScoreMap(ctx, "", pos, met, matchLength, ply, pruneK, gammonnet.LocalExec(0, ply, pruneK),
				func(i int, cell gammonnet.ScoreMapCell) {
					a.emitBatch("gammonnet-scoremap:cell", ScoreMapCellEvent{Tag: tag, Index: i, Cell: cell})
				})
		}
		a.finishMap("gammonnet-scoremap", tag, end, err, func() any { return ScoreMapDoneEvent{Tag: tag, Map: m} })
	}()
}

// CancelScoreMap stops the score map in flight, if it is still the
// interactive search running.
func (a *App) CancelScoreMap() {
	a.cancelInteractiveSearch(&a.gnScoreMapSeq)
}

// validateScoreMap holds the GUI to the bounds the CLI and the daemon check.
func validateScoreMap(matchLength, ply int) error {
	if err := gammonnet.ValidateMapMatchLength(matchLength); err != nil {
		return err
	}
	return gammonnet.ValidateMapPly(ply)
}
