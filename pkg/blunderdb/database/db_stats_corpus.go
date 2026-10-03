package database

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// HeadToHeadCtx is the record of two players against each other
// (storage.StatsStore.HeadToHead).
func (d *Database) HeadToHeadCtx(ctx context.Context, playerA, playerB string, filter StatsFilter) (*storage.HeadToHead, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.store.Stats().HeadToHead(ctx, "", playerA, playerB, toStorageStatsFilter(filter))
}

// PRByWindowCtx is the PR over a sliding window of months
// (storage.StatsStore.PRByWindow).
func (d *Database) PRByWindowCtx(ctx context.Context, filter StatsFilter, months int) ([]storage.WindowStats, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.store.Stats().PRByWindow(ctx, "", toStorageStatsFilter(filter), months)
}

// PlayerRankingCtx ranks the players of the filter by PR above a floor of
// counted decisions (storage.RankPlayers over the players table).
func (d *Database) PlayerRankingCtx(ctx context.Context, filter StatsFilter, minDecisions int) ([]storage.RankedPlayer, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	rows, err := d.store.Stats().PlayerTable(ctx, "", toStorageStatsFilter(filter))
	if err != nil {
		return nil, err
	}
	return storage.RankPlayers(rows, minDecisions), nil
}
