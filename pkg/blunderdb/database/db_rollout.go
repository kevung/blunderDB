package database

import (
	"context"
	"errors"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/rollout"
	"github.com/kevung/blunderdb/pkg/blunderdb/rollouts"
)

// Rollouts on the local library: the same gather, loop and write as the
// serve daemon (pkg/blunderdb/rollouts), under this wrapper's lock. The lock
// is taken around each read and each write, never around the games: a
// rollout runs for seconds to minutes and the library stays usable meanwhile.
//
// The methods taking a context or a callback are bound to Wails like every
// method of *Database but cannot be called from the webview, which supplies
// neither; the GUI goes through App.StartRollout and App.StartRolloutFiltered.
// Writing an arbitrary result stays unexported: storeRollout trusts res.

// RolloutPosition rolls positionID out with s — its plays when it has dice,
// its cube decision otherwise, moves naming the plays when set — and, with
// store, writes the finished rollout beside its analysis. A cancelled
// rollout returns the games finished so far with ctx's error, unstored. A
// position that cannot be read fails with rollouts.ErrLoad.
func (d *Database) RolloutPosition(ctx context.Context, positionID int64, s rollout.Settings, moves []string, store bool, progress func(rollout.Progress)) (*rollout.Result, error) {
	d.mu.RLock()
	gen := d.generation
	pos, err := rollouts.Load(ctx, d.store, "", positionID)
	d.mu.RUnlock()
	if err != nil {
		return nil, err
	}
	res, err := rollouts.Run(ctx, pos, s, moves, progress)
	if err != nil {
		return res, err
	}
	if store {
		if err := d.storeRollout(gen, positionID, res); err != nil {
			return res, err
		}
	}
	return res, nil
}

// storeRollout writes a finished rollout on positionID (ADR-0060 §8): a
// second Analysis, beside the imported or evaluated one, replacing nothing.
// It refuses once the open file is no longer the one of generation gen.
func (d *Database) storeRollout(gen uint64, positionID int64, res *rollout.Result) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.generation != gen {
		return ErrDatabaseChanged
	}
	return rollouts.Store(context.Background(), d.store, "", positionID, res)
}

// LoadRollouts returns the rollouts stored on positionID, newest first.
func (d *Database) LoadRollouts(positionID int64) ([]domain.RolloutAnalysis, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return rollouts.List(context.Background(), d.store, "", positionID)
}

// PositionsToRollout snapshots the positions f selects that carry no rollout
// of s's Signature yet: what RolloutFiltered would roll out.
func (d *Database) PositionsToRollout(ctx context.Context, f SearchFilters, s rollout.Settings) ([]Position, error) {
	positions, _, err := d.positionsToRollout(ctx, f, s)
	return positions, err
}

// positionsToRollout is PositionsToRollout with the generation it read.
func (d *Database) positionsToRollout(ctx context.Context, f SearchFilters, s rollout.Settings) ([]Position, uint64, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	positions, err := rollouts.Gather(ctx, d.store, "", f, s)
	return positions, d.generation, err
}

// RolloutFiltered rolls out, one after the other, every position f selects
// that carries no rollout of s's Signature yet, writing each as it finishes.
// Cancelling ctx keeps what was written; running again resumes.
func (d *Database) RolloutFiltered(ctx context.Context, f SearchFilters, s rollout.Settings, progress func(rollouts.Progress)) (rollouts.Summary, error) {
	if err := s.Validate(); err != nil {
		return rollouts.Summary{}, err
	}
	positions, gen, err := d.positionsToRollout(ctx, f, s)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return rollouts.Summary{Cancelled: true}, nil
		}
		return rollouts.Summary{}, err
	}
	return d.rolloutPositionsAt(ctx, gen, positions, s, progress)
}

// RolloutPositions rolls out positions — a PositionsToRollout snapshot — one
// after the other, writing each as it finishes.
func (d *Database) RolloutPositions(ctx context.Context, positions []Position, s rollout.Settings, progress func(rollouts.Progress)) (rollouts.Summary, error) {
	return d.rolloutPositionsAt(ctx, d.currentGeneration(), positions, s, progress)
}

func (d *Database) rolloutPositionsAt(ctx context.Context, gen uint64, positions []Position, s rollout.Settings, progress func(rollouts.Progress)) (rollouts.Summary, error) {
	return rollouts.Batch(ctx, positions, s, progress, func(positionID int64, res *rollout.Result) error {
		return d.storeRollout(gen, positionID, res)
	})
}
