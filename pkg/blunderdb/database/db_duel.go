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
}

// duelService returns the Arbiter over the open library, made on first use.
// Caller holds d.mu; lock ORDER mu -> duelMu, and forgetDuels takes duelMu
// alone before mu.
func (d *Database) duelService() *duel.Service {
	d.duelMu.Lock()
	defer d.duelMu.Unlock()
	if d.duelSvc == nil {
		d.duelSvc = duel.New(d.store, duel.Options{})
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
	return DuelOffer{Cadences: duel.NamedCadences(), BotLevels: append([]string(nil), duel.BotLevels...)}
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

// CreateDuel creates a Duel and opens it, the one open before going into
// suspense. A Start or a Cadence the rules refuse is an error naming why.
func (d *Database) CreateDuel(set duel.Settings) (*DuelState, error) {
	return d.duelState(0, func(ctx context.Context, svc *duel.Service) (*duel.State, error) {
		return svc.Create(ctx, "", set)
	})
}

// OpenDuel resumes a Duel in suspense where it stopped, its clocks running
// again.
func (d *Database) OpenDuel(id int64) (*DuelState, error) {
	return d.duelState(id, func(ctx context.Context, svc *duel.Service) (*duel.State, error) {
		return svc.Open(ctx, "", id)
	})
}

// SuspendDuel puts the open Duel in suspense, its clocks stopped.
func (d *Database) SuspendDuel(id int64) error {
	return d.withDuels(func(svc *duel.Service) error {
		return svc.Suspend(context.Background(), "", id)
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

// StopDuel ends the Duel before its end: kept, the Match is written as it
// stands; thrown away, nothing of it is.
func (d *Database) StopDuel(id, revision int64, keep bool) (*DuelState, error) {
	return d.duelState(id, func(ctx context.Context, svc *duel.Service) (*duel.State, error) {
		return svc.Stop(ctx, "", id, revision, keep)
	})
}

// forgetDuels puts the open Duel in suspense and drops the Service when the
// handle is replaced or closed, BEFORE d.mu is taken for writing: the clocks
// stop with the library, and a Duel id of the previous library must not
// answer for the next one.
func (d *Database) forgetDuels() {
	d.duelMu.Lock()
	svc := d.duelSvc
	d.duelSvc = nil
	d.duelMu.Unlock()
	if svc == nil {
		return
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil || d.store == nil {
		return
	}
	ctx := context.Background()
	rows, err := svc.List(ctx, "")
	if err != nil {
		return
	}
	for _, r := range rows {
		if r.Open {
			_ = svc.Suspend(ctx, "", r.ID)
		}
	}
}
