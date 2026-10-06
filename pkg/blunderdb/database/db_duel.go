package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/duel"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// Duels (ADR-0072, ADR-0073): a façade over duel.Service, the Arbiter the CLI
// and the daemon drive too. The desktop is one scope, "", and plays one Duel
// at a time; the Service remembers which one is open for as long as this
// library stays open.

// DuelState is what every Duel binding hands back: the Duel as the Arbiter
// sees it, and its match sheet for the Transcription's view.
type DuelState struct {
	State *duel.State          `json:"state"`
	Sheet transcript.Annotated `json:"sheet"`
	// Conflict: the draft moved under the gesture, which was NOT played; State
	// is the Duel as it now stands.
	Conflict bool `json:"conflict"`
}

// DuelOffer is what the creation form proposes.
type DuelOffer struct {
	Cadences  []duel.Cadence `json:"cadences"`
	BotLevels []string       `json:"botLevels"`
	// Levels: the same levels with their search depth, for the form's labels.
	Levels []duel.LevelInfo `json:"levels"`
}

// duelService returns the Arbiter over the open library, made on first use.
// Caller holds d.mu; lock ORDER mu -> duelMu, and forgetDuels takes duelMu
// alone before mu.
//
// A Service made between forgetDuels and the replacement of the library is
// on the replaced store: it is recognised by its store and dropped, its open
// Duel naming a row of that library.
func (d *Database) duelService() *duel.Service {
	d.duelMu.Lock()
	defer d.duelMu.Unlock()
	if d.duelSvc != nil && d.duelOn != d.store {
		d.duelSvc = nil
	}
	if d.duelSvc == nil {
		d.duelSvc = duel.New(d.store, duel.Options{})
		d.duelOn = d.store
	}
	return d.duelSvc
}

// withDuels runs fn under the READ lock: the Service has its own lock and the
// store its own transactions, and a Play waits for the Bot's move — the
// library must stay readable meanwhile. The read lock still keeps the library
// from being closed or replaced, and excludes the legacy wrapper's writers.
func (d *Database) withDuels(fn func(*duel.Service) error) error {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil || d.store == nil {
		return fmt.Errorf("no database is currently open")
	}
	return fn(d.duelService())
}

// duelState runs a gesture that hands back a Duel. A gesture refused because
// the draft moved is answered with the Duel as it stands, flagged.
func (d *Database) duelState(id int64, fn func(context.Context, *duel.Service) (*duel.State, error)) (*DuelState, error) {
	var st *duel.State
	ctx := context.Background()
	err := d.withDuels(func(svc *duel.Service) (err error) {
		st, err = fn(ctx, svc)
		if errors.Is(err, storage.ErrConflict) {
			if fresh, ferr := svc.Get(ctx, "", id); ferr == nil {
				st = fresh
				return errConflictAnswered
			}
		}
		return err
	})
	if errors.Is(err, errConflictAnswered) {
		return &DuelState{State: st, Sheet: st.Sheet, Conflict: true}, nil
	}
	if err != nil {
		return nil, err
	}
	return &DuelState{State: st, Sheet: st.Sheet}, nil
}

var errConflictAnswered = errors.New("duel moved: answered with its fresh state")

// DuelOffer returns the named Cadences and the Bot's levels.
func (d *Database) DuelOffer() DuelOffer {
	return DuelOffer{Cadences: duel.NamedCadences(), BotLevels: append([]string(nil), duel.BotLevels...), Levels: duel.LevelInfos()}
}

// ListDuels returns the Duels in suspense, most recently played first.
func (d *Database) ListDuels() ([]duel.Summary, error) {
	var out []duel.Summary
	err := d.withDuels(func(svc *duel.Service) (err error) {
		out, err = svc.List(context.Background(), "")
		return err
	})
	return out, err
}

// CreateDuel creates a Duel and opens it, the one at the board before going
// into suspense. A Start or a Cadence the rules refuse is an error naming why.
func (d *Database) CreateDuel(set duel.Settings) (*DuelState, error) {
	return d.duelState(0, func(ctx context.Context, svc *duel.Service) (*duel.State, error) {
		st, err := svc.Create(ctx, "", set)
		if err == nil {
			err = d.takeBoard(ctx, svc, st)
		}
		return st, err
	})
}

// OpenDuel resumes a Duel in suspense where it stopped, its clocks running
// again, and puts the one at the board in suspense.
func (d *Database) OpenDuel(id int64) (*DuelState, error) {
	return d.duelState(id, func(ctx context.Context, svc *duel.Service) (*duel.State, error) {
		st, err := svc.Open(ctx, "", id)
		if err == nil {
			err = d.takeBoard(ctx, svc, st)
		}
		return st, err
	})
}

// takeBoard puts st at the board and the Duel it replaces in suspense. The
// Arbiter lets several Duels be open at once; the desktop shows one, and the
// clocks of the one it left stand still. A Duel ended meanwhile has nothing
// to suspend, and an ended st leaves the board empty.
func (d *Database) takeBoard(ctx context.Context, svc *duel.Service, st *duel.State) error {
	next := st.ID
	if st.Ended != nil {
		next = 0
	}
	d.duelMu.Lock()
	prev := d.duelBoard
	d.duelBoard = next
	d.duelMu.Unlock()
	if prev == 0 || prev == st.ID {
		return nil
	}
	if err := svc.Suspend(ctx, "", prev, 0); err != nil && !errors.Is(err, storage.ErrNotFound) {
		return err
	}
	return nil
}

// SuspendDuel puts the Duel in suspense, its clocks stopped.
func (d *Database) SuspendDuel(id int64) error {
	return d.withDuels(func(svc *duel.Service) error {
		d.duelMu.Lock()
		if d.duelBoard == id {
			d.duelBoard = 0
		}
		d.duelMu.Unlock()
		return svc.Suspend(context.Background(), "", id, 0)
	})
}

// PlayDuel plays the player's decision on the open Duel, at the revision the
// panel last drew; the Bot answers in the same call.
func (d *Database) PlayDuel(id, revision int64, p duel.Play) (*DuelState, error) {
	return d.duelState(id, func(ctx context.Context, svc *duel.Service) (*duel.State, error) {
		return svc.Play(ctx, "", id, revision, p)
	})
}

// FlagDuel has the Arbiter look at the awaited Side's clock.
func (d *Database) FlagDuel(id int64) (*DuelState, error) {
	return d.duelState(id, func(ctx context.Context, svc *duel.Service) (*duel.State, error) {
		return svc.Flag(ctx, "", id)
	})
}

// StopDuel ends the Duel before its end: thrown away, nothing of it is
// written; kept, a money session's Match is written as it stands, and a match
// in points is refused (duel.Service.Stop).
func (d *Database) StopDuel(id, revision int64, keep bool) (*DuelState, error) {
	return d.duelState(id, func(ctx context.Context, svc *duel.Service) (*duel.State, error) {
		return svc.Stop(ctx, "", id, revision, keep)
	})
}

// ForfeitDuel has side (0 player 1, 1 player 2) give the match up: the Duel
// ends, its Match written won by the other side.
func (d *Database) ForfeitDuel(id, revision int64, side int) (*DuelState, error) {
	return d.duelState(id, func(ctx context.Context, svc *duel.Service) (*duel.State, error) {
		return svc.Forfeit(ctx, "", id, revision, side)
	})
}

// ContributeDuel records side's (0 player 1, 1 player 2) contribution to a
// Duel created with a combined seed; the last one in starts the dice.
func (d *Database) ContributeDuel(id, revision int64, side int, contribution string) (*DuelState, error) {
	return d.duelState(id, func(ctx context.Context, svc *duel.Service) (*duel.State, error) {
		return svc.Contribute(ctx, "", id, revision, side, contribution)
	})
}

// GetMatchOrigin returns the origin of a Match played here — Start, revealed
// seed and its fingerprint, stop before the end, Cadence and overrun, Bot —
// nil when the Match was not played here, ErrNotFound when there is none.
func (d *Database) GetMatchOrigin(matchID int64) (*duel.Origin, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil || d.store == nil {
		return nil, fmt.Errorf("no database is currently open")
	}
	return duel.ReadOrigin(context.Background(), d.store, "", matchID)
}

// forgetDuels puts the Duel at the board in suspense and drops the Service
// when the handle is replaced or closed, BEFORE d.mu is taken for writing: its
// clocks stop with the library, and a Duel id of the previous library must
// not answer for the next one. Other open Duels of the library belong to
// whoever opened them and keep running.
func (d *Database) forgetDuels() {
	d.duelMu.Lock()
	svc := d.duelSvc
	board := d.duelBoard
	d.duelSvc = nil
	d.duelOn = nil
	d.duelBoard = 0
	d.duelMu.Unlock()
	if svc == nil || board == 0 {
		return
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil || d.store == nil {
		return
	}
	_ = svc.Suspend(context.Background(), "", board, 0)
}
