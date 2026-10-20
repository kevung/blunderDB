package database

import (
	"slices"
	"testing"
	"time"
)

// pushPositionIDsPastEstimate makes the next position id exceed
// EstimateExactBelow, so the estimate stops counting the positions and the
// blunders are known only from their cache.
func pushPositionIDsPastEstimate(t *testing.T, db *Database) {
	t.Helper()
	if _, err := db.db.Exec(`UPDATE sqlite_sequence SET seq = ? WHERE name = 'position'`, EstimateExactBelow+1000); err != nil {
		t.Fatalf("sqlite_sequence: %v", err)
	}
	pos := InitializePosition()
	pos.Dice = [2]int{6, 5}
	pos.Score = [2]int{3, 2}
	if _, err := db.SavePosition(&pos); err != nil {
		t.Fatalf("SavePosition: %v", err)
	}
}

// A write on a large library forgets the cached blunder count; the estimate
// must not leave the desktop counter at "?" for good: it recounts in the
// background and says when the count is back.
func TestEstimateRecountsForgottenBlundersInBackground(t *testing.T) {
	db := newTestDBWithXG(t)
	ready := make(chan struct{}, 1)
	SetBlunderCountReady(db, func() {
		select {
		case ready <- struct{}{}:
		default:
		}
	})

	exact, err := db.GetDatabaseStats()
	if err != nil {
		t.Fatalf("GetDatabaseStats: %v", err)
	}
	pushPositionIDsPastEstimate(t, db) // the write forgets the cache

	first, err := db.GetDatabaseStatsEstimate()
	if err != nil {
		t.Fatalf("GetDatabaseStatsEstimate: %v", err)
	}
	if approx, _ := first["approximate"].([]string); !slices.Contains(approx, "positions") {
		t.Fatalf("approximate = %v, want positions estimated", first["approximate"])
	}
	if _, known := first["blunder_count"]; known {
		t.Fatalf("blunder_count = %v right after a write; the cache should be gone", first["blunder_count"])
	}

	select {
	case <-ready:
	case <-time.After(30 * time.Second):
		t.Fatal("the background recount never reported the count back")
	}
	again, err := db.GetDatabaseStatsEstimate()
	if err != nil {
		t.Fatalf("GetDatabaseStatsEstimate: %v", err)
	}
	if again["blunder_count"] != exact["blunder_count"] {
		t.Fatalf("blunder_count after the recount = %v, want the exact %v", again["blunder_count"], exact["blunder_count"])
	}
}

// Closing the file stops its recount: nothing reports a count of a file that
// is gone.
func TestCloseCancelsBlunderRecount(t *testing.T) {
	db := newTestDBWithXG(t)
	pushPositionIDsPastEstimate(t, db)
	if _, err := db.GetDatabaseStatsEstimate(); err != nil {
		t.Fatalf("GetDatabaseStatsEstimate: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	db.recount.mu.Lock()
	running := db.recount.running
	db.recount.mu.Unlock()
	if running != nil {
		t.Fatal("a recount is still registered after Close")
	}
}
