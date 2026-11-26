package server

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
)

// TestAnalysisPoolServesAnotherTenantDuringAMapsDeepPhase: a map's deep
// phase reaches the pool one cell per unit, so a tenant asking for one
// evaluation while the map refines waits for one cell, not for the rest of
// the map.
func TestAnalysisPoolServesAnotherTenantDuringAMapsDeepPhase(t *testing.T) {
	p := newAnalysisPool(1, nil)
	t.Cleanup(p.close)
	ctx := context.Background()
	const ply = 2
	exec := func(n int, cell func(int, *gammonnet.Searcher)) error {
		return p.each(ctx, "club", n, func(time.Duration) bool { return true }, func(i int, get searcherFor) {
			cell(i, get.searcher(ply, 0))
		})
	}
	// A cube decision at 3 points: nine cells, each a short 2-ply search.
	pos, err := domain.DecodeXGID("XGID=---BBaB-BbA-bC-b--BdAca---:0:0:1:00:0:1:0:3:10")
	if err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var events []string
	var other sync.WaitGroup
	asked := false
	_, err = gammonnet.ComputeScoreMap(ctx, pos, nil, 3, ply, 0, exec, func(_ int, c gammonnet.ScoreMapCell) {
		mu.Lock()
		defer mu.Unlock()
		if c.Ply != ply {
			return
		}
		events = append(events, "cell")
		if !asked {
			asked = true
			other.Add(1)
			go func() {
				defer other.Done()
				p.run(ctx, "alice", func(searcherFor) {
					mu.Lock()
					events = append(events, "alice")
					mu.Unlock()
				})
			}()
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	other.Wait()
	if len(events) == 0 || events[len(events)-1] == "alice" {
		t.Fatalf("events = %v: the other tenant waited for the whole deep phase", events)
	}
}
