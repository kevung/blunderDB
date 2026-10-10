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

// StatsMatchIDsCtx lists the matches a corpus figure counts under filter
// (storage.StatsStore.MatchIDs): what a ranking row, a window or a player
// opens in the match list.
func (d *Database) StatsMatchIDsCtx(ctx context.Context, filter StatsFilter) ([]int64, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.store.Stats().MatchIDs(ctx, "", toStorageStatsFilter(filter))
}

// StatsMatchIDs is StatsMatchIDsCtx for the GUI binding, which passes no context.
func (d *Database) StatsMatchIDs(filter StatsFilter) ([]int64, error) {
	return d.StatsMatchIDsCtx(context.Background(), filter)
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

// HeadToHead is HeadToHeadCtx for the GUI binding, which passes no context.
func (d *Database) HeadToHead(playerA, playerB string, filter StatsFilter) (*storage.HeadToHead, error) {
	return d.HeadToHeadCtx(context.Background(), playerA, playerB, filter)
}

// PRByWindow is PRByWindowCtx for the GUI binding.
func (d *Database) PRByWindow(filter StatsFilter, months int) ([]storage.WindowStats, error) {
	return d.PRByWindowCtx(context.Background(), filter, months)
}

// PlayerRanking is PlayerRankingCtx for the GUI binding.
func (d *Database) PlayerRanking(filter StatsFilter, minDecisions int) ([]storage.RankedPlayer, error) {
	return d.PlayerRankingCtx(context.Background(), filter, minDecisions)
}

// PlayerContrastCtx lists the positions two players both decided and answered
// differently (storage.StatsStore.PlayerContrast).
func (d *Database) PlayerContrastCtx(ctx context.Context, playerA, playerB string, filter StatsFilter) (*storage.PlayerContrast, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.store.Stats().PlayerContrast(ctx, "", playerA, playerB, toStorageStatsFilter(filter))
}

// PlayerContrast is PlayerContrastCtx for the GUI binding.
func (d *Database) PlayerContrast(playerA, playerB string, filter StatsFilter) (*storage.PlayerContrast, error) {
	return d.PlayerContrastCtx(context.Background(), playerA, playerB, filter)
}
