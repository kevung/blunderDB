package postgres

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type statsStore struct{ db execer }

var _ storage.StatsStore = (*statsStore)(nil)

func (*statsStore) DateRange(context.Context, string) (storage.StatsDateRange, error) {
	return storage.StatsDateRange{}, notImpl("Stats", "DateRange")
}
func (*statsStore) Compute(context.Context, string, storage.StatsFilter) (*storage.StatsResult, error) {
	return nil, notImpl("Stats", "Compute")
}
func (*statsStore) PositionIDsBySelection(context.Context, string, storage.StatsFilter, storage.SelectionSpec) ([]int64, error) {
	return nil, notImpl("Stats", "PositionIDsBySelection")
}
func (*statsStore) PositionIDsByTournament(context.Context, string, int64) ([]int64, error) {
	return nil, notImpl("Stats", "PositionIDsByTournament")
}
func (*statsStore) PositionIDsByMatch(context.Context, string, int64) ([]int64, error) {
	return nil, notImpl("Stats", "PositionIDsByMatch")
}
func (*statsStore) PlayerNames(context.Context, string) ([]storage.PlayerFrequency, error) {
	return nil, notImpl("Stats", "PlayerNames")
}
func (*statsStore) MatchDetail(context.Context, string, int64) (*storage.MatchDetailStats, error) {
	return nil, notImpl("Stats", "MatchDetail")
}
