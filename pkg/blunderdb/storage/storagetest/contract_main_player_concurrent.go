package storagetest

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// mainPlayerRacers is how many gestures run at once, mainPlayerRounds how many
// times the race is replayed: one round seldom shows a lost interleaving.
const (
	mainPlayerRacers = 8
	mainPlayerRounds = 6
)

// race runs fn racers times at once, released together, and returns the
// first error any of them met.
func race(racers int, fn func() error) error {
	var wg sync.WaitGroup
	start := make(chan struct{})
	errs := make(chan error, racers)
	for range racers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errs <- fn()
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

// mainAtSeat1 reports the first match whose seat 1 is not want.
func mainAtSeat1(ctx context.Context, s storage.Storage, ids []int64, want string) error {
	for _, id := range ids {
		m, err := s.Matches().Get(ctx, "", id)
		if err != nil {
			return fmt.Errorf("get match %d: %w", id, err)
		}
		if m.Player1Name != want {
			return fmt.Errorf("match %d seats %q first, want %q", id, m.Player1Name, want)
		}
	}
	return nil
}

// testMainPlayerConcurrentDesignations: designations of one name run at once
// swap each match exactly once — the plans applied add up to the matches
// that had the Main player second — and leave the Main player at seat 1 of
// every match; so do concurrent placements of the Main player on one match.
func testMainPlayerConcurrentDesignations(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	var ids []int64
	for i := range 6 {
		id, _ := importNamed(t, s, fmt.Sprintf("Rival %d", i), "Kevin", false)
		ids = append(ids, id)
	}
	// Puts Kevin back at seat 2 of every match and forgets him, for the next round.
	reset := func() {
		t.Helper()
		if err := storage.ClearMainPlayer(ctx, s, ""); err != nil {
			t.Fatalf("ClearMainPlayer: %v", err)
		}
		for _, id := range ids {
			m, err := s.Matches().Get(ctx, "", id)
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			if m.Player1Name == "Kevin" {
				if err := s.Matches().SwapPlayers(ctx, "", id); err != nil {
					t.Fatalf("SwapPlayers: %v", err)
				}
			}
		}
	}

	for round := range mainPlayerRounds {
		var mu sync.Mutex
		swapped := 0
		err := race(mainPlayerRacers, func() error {
			plan, err := storage.DesignateMainPlayer(ctx, s, "", "Kevin")
			mu.Lock()
			swapped += len(plan.Swap)
			mu.Unlock()
			return err
		})
		if err != nil {
			t.Fatalf("round %d: DesignateMainPlayer: %v", round, err)
		}
		if swapped != len(ids) {
			t.Errorf("round %d: the designations swapped %d matches in all, want %d", round, swapped, len(ids))
		}
		if err := mainAtSeat1(ctx, s, ids, "Kevin"); err != nil {
			t.Fatalf("round %d after designations: %v", round, err)
		}
		reset()
	}

	if err := s.LibrarySettings().SetMainPlayer(ctx, "", "Kevin"); err != nil {
		t.Fatalf("SetMainPlayer: %v", err)
	}
	for round := range mainPlayerRounds {
		var mu sync.Mutex
		placed := 0
		err := race(mainPlayerRacers, func() error {
			tx, err := s.BeginTx(ctx)
			if err != nil {
				return err
			}
			defer func() { _ = tx.Rollback() }()
			ok, err := storage.PlaceMainPlayerFirst(ctx, tx, "", ids[0])
			if err != nil {
				return err
			}
			if err := tx.Commit(); err != nil {
				return err
			}
			if ok {
				mu.Lock()
				placed++
				mu.Unlock()
			}
			return nil
		})
		if err != nil {
			t.Fatalf("round %d: PlaceMainPlayerFirst: %v", round, err)
		}
		if placed != 1 {
			t.Errorf("round %d: %d placements swapped the match, want 1", round, placed)
		}
		if err := mainAtSeat1(ctx, s, ids[:1], "Kevin"); err != nil {
			t.Fatalf("round %d after placements: %v", round, err)
		}
		if err := s.Matches().SwapPlayers(ctx, "", ids[0]); err != nil {
			t.Fatalf("SwapPlayers: %v", err)
		}
	}
}
