package direction

import (
	"context"
	"testing"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
)

// BenchmarkOpen measures what Open costs, since the whole design rests on replaying rather than
// storing: every open of a Direction replays its entire log.
//
// The number that matters is the one for a real tournament. A 64-player two-life event plays
// about 127 matches, and each match costs three events at most (start, result, and the
// occasional correction), plus one entry per player: roughly 450 events. If that ever stopped
// being instant the answer would be an incremental replay or a cached state, and the measure
// would have to be written here — which is why the benchmark exists rather than a bare claim.
func BenchmarkOpen(b *testing.B) {
	ctx := context.Background()
	store := newMemStore()
	cfg := tournoi.Config{Name: "Bench", Tables: tournoi.Tables{Count: 16},
		Phases: []tournoi.PhaseConfig{
			{Kind: tournoi.KindSwissLives, Length: 7, Target: 16},
			{Kind: tournoi.KindLivesBracket, Length: 9},
		}}
	now := time.Date(2026, 9, 12, 9, 0, 0, 0, time.UTC)
	d, err := Create(ctx, store, 1, cfg, 7, now)
	if err != nil {
		b.Fatal(err)
	}
	players := make([]tournoi.Player, 64)
	for i := range players {
		players[i] = tournoi.Player{ID: tournoi.PlayerID(string(rune('a'+i%26)) + string(rune('a'+i/26))), Name: "p"}
	}
	for _, p := range players {
		if err := d.Enter(ctx, p, now); err != nil {
			b.Fatal(err)
		}
	}
	// Run the tournament to the end so the log has its real length.
	for step := 0; step < 4000; step++ {
		acts := d.Propose()
		if len(acts) == 0 {
			break
		}
		moved := false
		for _, a := range acts {
			if a.Kind == tournoi.ActWait {
				continue
			}
			if a.Kind == tournoi.ActFinish {
				_ = d.Finish(ctx, now)
				moved = true
				break
			}
			ev, err := d.EventFor(a, now)
			if err != nil {
				b.Fatal(err)
			}
			if err := d.Apply(ctx, ev); err != nil {
				b.Fatal(err)
			}
			moved = true
			if a.Kind == tournoi.ActStartMatch {
				now = now.Add(time.Minute)
				if err := d.Apply(ctx, tournoi.ResultEvent(ev.MatchID, ev.A, a.Length, 2, now)); err != nil {
					b.Fatal(err)
				}
			}
		}
		if !moved {
			break
		}
	}
	b.ReportMetric(float64(len(d.Journal())), "events")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Open(ctx, store, 1); err != nil {
			b.Fatal(err)
		}
	}
}
