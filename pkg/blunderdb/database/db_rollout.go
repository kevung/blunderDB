package database

import (
	"context"
	"errors"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/rollout"
	"github.com/kevung/blunderdb/pkg/blunderdb/rollouts"
)

// Rollouts on the local library: the same gather, loop and write the serve
// daemon runs (pkg/blunderdb/rollouts), under this wrapper's lock. The lock is
// taken around each read and each write, never around the games: a rollout
// runs for seconds to minutes and the library stays usable meanwhile.

// RolloutPosition rolls positionID out with s — its plays when it has dice,
// its cube decision otherwise, moves naming the plays when set — and, with
// store, writes the finished rollout beside its analysis. A cancelled
// rollout returns the games finished so far with ctx's error, unstored.
func (d *Database) RolloutPosition(ctx context.Context, positionID int64, s rollout.Settings, moves []string, store bool, progress func(rollout.Progress)) (*rollout.Result, error) {
	d.mu.RLock()
	pos, err := d.store.Positions().Load(ctx, "", positionID)
	d.mu.RUnlock()
	if err != nil {
		return nil, err
	}
	res, err := rollout.Run(ctx, *pos, s, rollout.Options{Moves: moves, Progress: progress})
	if err != nil {
		return res, err
	}
	if store {
		if err := d.StoreRollout(positionID, res); err != nil {
			return res, err
		}
	}
	return res, nil
}

// StoreRollout writes a finished rollout on positionID (ADR-0060 §8): a
// second Analysis, beside the imported or evaluated one, replacing nothing.
func (d *Database) StoreRollout(positionID int64, res *rollout.Result) error {
	d.mu.Lock()
	defer d.mu.Unlock()
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
	d.mu.RLock()
	defer d.mu.RUnlock()
	return rollouts.Gather(ctx, d.store, "", f, s)
}

// RolloutFiltered rolls out, one after the other, every position f selects
// that carries no rollout of s's Signature yet, writing each as it finishes.
// Cancelling ctx keeps what was written; running again resumes.
func (d *Database) RolloutFiltered(ctx context.Context, f SearchFilters, s rollout.Settings, progress func(rollouts.Progress)) (rollouts.Summary, error) {
	if err := s.Validate(); err != nil {
		return rollouts.Summary{}, err
	}
	positions, err := d.PositionsToRollout(ctx, f, s)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return rollouts.Summary{Cancelled: true}, nil
		}
		return rollouts.Summary{}, err
	}
	return d.RolloutPositions(ctx, positions, s, progress)
}

// RolloutPositions rolls out positions — a PositionsToRollout snapshot — one
// after the other, writing each as it finishes.
func (d *Database) RolloutPositions(ctx context.Context, positions []Position, s rollout.Settings, progress func(rollouts.Progress)) (rollouts.Summary, error) {
	return rollouts.Batch(ctx, positions, s, progress, d.StoreRollout)
}
