package duel

import (
	"slices"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func i64p(v int64) *int64 { return &v }

func TestReplayChargesBeyondTheDelayPerTurnAndStopsAtZero(t *testing.T) {
	c := Cadence{Reserve: 60, Delay: 12}
	turns := storage.MatchTurns{Turns: []storage.MatchTurn{
		{Player: 0, CubeMS: i64p(5000), PlayMS: i64p(10000)}, // one turn of 15 s: 3 s past the delay
		{Player: 1}, // unknown durations charge nothing
		{Player: 1, Cube: true, PlayMS: i64p(20000)}, // 8 s past the delay
		{Player: 0, PlayMS: i64p(100000)},            // beyond the reserve
		{Player: 0, PlayMS: i64p(1000)},              // inside the delay, reserve stays at 0
	}}
	got := c.Replay(7, turns)
	if got.Start != [2]int64{60000, 60000} {
		t.Fatalf("start = %v", got.Start)
	}
	if want := []int64{57000, 60000, 52000, 0, 0}; !slices.Equal(got.Remaining, want) {
		t.Fatalf("remaining = %v, want %v", got.Remaining, want)
	}
}

func TestReplayCountsAReserveFromTheStartingScore(t *testing.T) {
	c, _ := NamedCadence("tournament")
	got := c.Replay(7, storage.MatchTurns{Score: [2]int32{1, 2}, Turns: []storage.MatchTurn{{Player: 1, PlayMS: i64p(32000)}}})
	// 120 s × (14 − 3) / 2 = 660 s; 32 s − 12 s delay is charged.
	if got.Start[0] != 660000 || got.Remaining[0] != 640000 {
		t.Fatalf("start %v remaining %v", got.Start, got.Remaining)
	}
}
