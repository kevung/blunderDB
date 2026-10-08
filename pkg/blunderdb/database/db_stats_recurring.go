package database

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// ComputeRecurringErrors groups the filter's errors by plan of play and theme,
// heaviest summed cost first (storage.StatsStore.RecurringErrors).
func (d *Database) ComputeRecurringErrors(filter StatsFilter) (*storage.RecurringErrors, error) {
	return d.ComputeRecurringErrorsCtx(context.Background(), filter)
}

// ComputeRecurringErrorsCtx is ComputeRecurringErrors with a caller-supplied
// context that aborts the classification query.
func (d *Database) ComputeRecurringErrorsCtx(ctx context.Context, filter StatsFilter) (*storage.RecurringErrors, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.store.Stats().RecurringErrors(ctx, "", toStorageStatsFilter(filter))
}

// ComputeStudyPlan ranks the filter's error families by the winning chances
// studying them would recover (storage.StatsStore.StudyPlan, ADR-0077).
func (d *Database) ComputeStudyPlan(filter StatsFilter) (*storage.StudyPlan, error) {
	return d.ComputeStudyPlanCtx(context.Background(), filter)
}

// ComputeStudyPlanCtx is ComputeStudyPlan with a caller-supplied context.
func (d *Database) ComputeStudyPlanCtx(ctx context.Context, filter StatsFilter) (*storage.StudyPlan, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.store.Stats().StudyPlan(ctx, "", toStorageStatsFilter(filter))
}

// StudyPlanPositionIDs are the positions of the plan's families picked by
// rank (0 = the first three), drawn at random when size > 0 — what a quiz or a
// deck "from my study plan" uses.
func (d *Database) StudyPlanPositionIDs(filter StatsFilter, rank, size int) ([]int64, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return storage.StudyPlanIDs(context.Background(), d.store, "", toStorageStatsFilter(filter), rank, size)
}

// StudyPlanQueue is the study queue of the plan's families picked by rank,
// one entry per position, the largest excess first.
func (d *Database) StudyPlanQueue(filter StatsFilter, rank int) ([]domain.StudyQueueEntry, error) {
	plan, err := d.ComputeStudyPlan(filter)
	if err != nil {
		return nil, err
	}
	return plan.QueueEntries(rank), nil
}
