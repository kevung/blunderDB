package database

import (
	"math"
	"testing"
)

// TestMatchBadgesEqualDetail locks the contract that a match's list-row badge
// (filled by applyMatchBadges via the storage backend) reports the same PR as
// the per-player MatchDetail view. Both are derived from the same counted
// decisions, so they must agree exactly; this guards the badge port against
// drifting from the detail computation.
func TestMatchBadgesEqualDetail(t *testing.T) {
	db := newTestDB(t)

	tid := createTournament(t, db, "T", "2025-07-01")
	m := createMatch(t, db, "P1", "P2", "2025-07-02", 7, tid)
	g := createGame(t, db, m)
	// Player 1 (encoding 0): 40 checker decisions, error 10mp each.
	for i := range 40 {
		insertStatsFixtureRow(t, db, m, g, 10, 0, 0, i+1)
	}
	// Player 2 (encoding 1): 20 checker decisions, error 25mp each.
	for i := range 20 {
		insertStatsFixtureRow(t, db, m, g, 25, 0, 1, 100+i+1)
	}

	detail, err := db.GetMatchDetailStats(m)
	if err != nil {
		t.Fatalf("GetMatchDetailStats: %v", err)
	}

	// Exercise the badge-application path directly (the GUI/CLI list methods
	// route through this same helper).
	matches := []Match{{ID: m}}
	if err := db.applyMatchBadges(matches); err != nil {
		t.Fatalf("applyMatchBadges: %v", err)
	}
	badge := &matches[0]

	const eps = 1e-9
	if math.Abs(badge.PR-detail.Player1.PR) > eps {
		t.Errorf("player1: badge PR %.6f != detail PR %.6f", badge.PR, detail.Player1.PR)
	}
	if math.Abs(badge.PR2-detail.Player2.PR) > eps {
		t.Errorf("player2: badge PR %.6f != detail PR %.6f", badge.PR2, detail.Player2.PR)
	}
	if math.Abs(badge.MWCLoss-detail.Player1.MWCLoss) > eps {
		t.Errorf("player1: badge MWCLoss %.6f != detail MWCLoss %.6f", badge.MWCLoss, detail.Player1.MWCLoss)
	}
	if math.Abs(badge.MWCLoss2-detail.Player2.MWCLoss) > eps {
		t.Errorf("player2: badge MWCLoss %.6f != detail MWCLoss %.6f", badge.MWCLoss2, detail.Player2.MWCLoss)
	}
	// Sanity: player1 PR (error 10) must be lower than player2 PR (error 25).
	if !(badge.PR > 0 && badge.PR2 > badge.PR) {
		t.Errorf("unexpected PR ordering: p1=%.3f p2=%.3f", badge.PR, badge.PR2)
	}
}

// TestTournamentBadgesEqualComputeStats checks the tournament list-row badge PR
// matches the per-tournament PR from ComputeStats (the panel view).
func TestTournamentBadgesEqualComputeStats(t *testing.T) {
	db := newTestDB(t)

	tid := createTournament(t, db, "TB", "2025-07-01")
	m1 := createMatch(t, db, "A", "B", "2025-07-02", 7, tid)
	g1 := createGame(t, db, m1)
	for i := range 50 {
		insertStatsFixtureRow(t, db, m1, g1, 12, 0, 0, i+1)
	}
	m2 := createMatch(t, db, "A", "B", "2025-07-03", 7, tid)
	g2 := createGame(t, db, m2)
	for i := range 30 {
		insertStatsFixtureRow(t, db, m2, g2, 8, 0, 1, i+1)
	}

	res, err := db.ComputeStats(StatsFilter{TournamentIDs: []int64{tid}, DecisionType: -1})
	if err != nil {
		t.Fatalf("ComputeStats: %v", err)
	}
	if len(res.PerTournament) != 1 {
		t.Fatalf("expected 1 tournament, got %d", len(res.PerTournament))
	}
	wantPR := res.PerTournament[0].PR

	tournaments, err := db.GetAllTournaments()
	if err != nil {
		t.Fatalf("GetTournaments: %v", err)
	}
	var got *Tournament
	for i := range tournaments {
		if tournaments[i].ID == tid {
			got = &tournaments[i]
		}
	}
	if got == nil {
		t.Fatalf("tournament %d not returned by GetTournaments", tid)
	}
	if math.Abs(got.PR-wantPR) > 1e-9 {
		t.Errorf("tournament badge PR %.6f != ComputeStats PR %.6f", got.PR, wantPR)
	}
}
