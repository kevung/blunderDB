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

// StudyPositionIDs are the positions of the recurring errors of a filter, by
// group rank (0: the worst groups), drawn down to size when size is above zero
// (storage.StudyIDs).
func (d *Database) StudyPositionIDs(filter StatsFilter, rank, size int) ([]int64, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return storage.StudyIDs(context.Background(), d.store, "", toStorageStatsFilter(filter), rank, size)
}

// CreateStudyDeck makes an Anki deck of exactly these positions
// (storage.CreateStudyDeck).
func (d *Database) CreateStudyDeck(name string, ids []int64) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return storage.CreateStudyDeck(context.Background(), d.store, "", name, ids)
}
