package database

import (
	"context"

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
