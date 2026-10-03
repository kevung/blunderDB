package database

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// ComputeTrainingStats folds the Decision quiz, the real matches and the Anki
// reviews by calendar window (storage.ComputeTrainingStats).
func (d *Database) ComputeTrainingStats(filter StatsFilter, window string) (*storage.TrainingStats, error) {
	return d.ComputeTrainingStatsCtx(context.Background(), filter, window)
}

// ComputeTrainingStatsCtx is ComputeTrainingStats with a caller-supplied
// context that aborts the queries.
func (d *Database) ComputeTrainingStatsCtx(ctx context.Context, filter StatsFilter, window string) (*storage.TrainingStats, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return storage.ComputeTrainingStats(ctx, d.store, "", toStorageStatsFilter(filter), window)
}
