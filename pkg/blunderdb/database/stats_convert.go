package database

import (
	"encoding/json"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Bridges the database.* and storage.* stats DTOs so the Wails bindings keep
// their types. The two families are field-identical with the same json tags
// (pinned by the parity tests), so a JSON round-trip is exact by construction;
// stats are not a hot path.

func jsonConvert(src, dst any) {
	b, _ := json.Marshal(src)
	_ = json.Unmarshal(b, dst)
}

func toStorageStatsFilter(f StatsFilter) storage.StatsFilter {
	var out storage.StatsFilter
	jsonConvert(f, &out)
	return out
}

func toStorageSelectionSpec(s SelectionSpec) storage.SelectionSpec {
	var out storage.SelectionSpec
	jsonConvert(s, &out)
	return out
}

func fromStorageStatsResult(r *storage.StatsResult) *StatsResult {
	if r == nil {
		return nil
	}
	var out StatsResult
	jsonConvert(r, &out)
	return &out
}

func fromStorageDateRange(r storage.StatsDateRange) StatsDateRange {
	var out StatsDateRange
	jsonConvert(r, &out)
	return out
}

func fromStorageMatchDetail(m *storage.MatchDetailStats) *MatchDetailStats {
	if m == nil {
		return nil
	}
	var out MatchDetailStats
	jsonConvert(m, &out)
	return &out
}

func fromStoragePlayerFreq(p []storage.PlayerFrequency) []PlayerFrequency {
	if p == nil {
		return nil
	}
	out := make([]PlayerFrequency, 0, len(p))
	for _, x := range p {
		var y PlayerFrequency
		jsonConvert(x, &y)
		out = append(out, y)
	}
	return out
}
