package gui

import (
	"context"
	"errors"
	"sync"

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
// written to the library.

// mapRun remembers which interactive search a map kind started, so its
// Cancel stops that search and never the panel's.
type mapRun struct {
	mu  sync.Mutex
	seq uint64
}

func (r *mapRun) set(seq uint64) {
	r.mu.Lock()
	r.seq = seq
	r.mu.Unlock()
}

func (r *mapRun) get() uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.seq
}

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

// StartRollMapAfterPlay is StartRollMap on the position pos reaches once
// played with die1-die2 and move (any legal play, the best or not), the
// opponent on roll: the replies to that play. move is empty for a dance.
func (a *App) StartRollMapAfterPlay(pos domain.Position, die1, die2 int, move string, ply, pruneK, tag int) {
	after, err := gammonnet.PositionAfterPlay(pos, die1, die2, move)
	if err != nil {
		a.emitBatch("gammonnet-rolls:error", map[string]any{"tag": tag, "message": err.Error()})
		return
	}
	a.startRollMap(after, ply, pruneK, tag)
}

var gnRollMap mapRun

func (a *App) startRollMap(pos domain.Position, ply, pruneK, tag int) {
	ctx, seq, end := a.beginInteractiveSearch()
	gnRollMap.set(seq)
	go func() {
		defer recoverBackground(a.ctx, "gammonNet roll map")
		met, err := a.currentMET()
		var m gammonnet.RollMap
		if err == nil {
			m, err = gammonnet.ComputeRollMap(ctx, pos, met, ply, pruneK, gammonnet.LocalExec(0, ply, pruneK),
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
	a.cancelInteractiveSearch(gnRollMap.get())
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
