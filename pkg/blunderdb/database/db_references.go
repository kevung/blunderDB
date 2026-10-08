package database

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// SuggestReferencePositions proposes the filter's reference positions,
// narrowed to matchIDs when given, at most size of them (0 = the default):
// the positions whose lesson covers the most recoverable MWC of their
// neighbouring errors (storage.StatsStore.SuggestReferences, ADR-0079).
// Nothing is written: the caller makes a collection, a deck or a quiz of what
// the user keeps.
func (d *Database) SuggestReferencePositions(filter StatsFilter, matchIDs []int64, size int) (*storage.ReferenceSuggestions, error) {
	return d.SuggestReferencePositionsCtx(context.Background(), filter, matchIDs, size)
}

// SuggestReferencePositionsCtx is SuggestReferencePositions with a
// caller-supplied context.
func (d *Database) SuggestReferencePositionsCtx(ctx context.Context, filter StatsFilter, matchIDs []int64, size int) (*storage.ReferenceSuggestions, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.store.Stats().SuggestReferences(ctx, "", storage.ReferenceRequest{
		Filter: toStorageStatsFilter(filter), MatchIDs: matchIDs, Size: size})
}
