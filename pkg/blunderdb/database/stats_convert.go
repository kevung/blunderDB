package database

import (
	"encoding/json"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// stats_convert.go bridges the database.* and storage.* stats DTOs so the
// Database stats methods can delegate to the storage.StatsStore (the single
// production implementation, shared with the headless server) while keeping
// their long-standing database.* return types — so the Wails bindings and the
// frontend see no change.
//
// The two DTO families are field-identical and share the same json tags (and,
// where untagged, the same field names) — a property the stats parity tests pin
// byte-for-byte. Converting through a JSON round-trip is therefore exact *by
// construction*: it cannot silently mis-map a field the way hand-written
// assignments could, and a future divergence between the two type sets would
// surface as a parity-test failure. Stats are computed on demand, never on a
// hot path, so the marshal/unmarshal cost is irrelevant.

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
