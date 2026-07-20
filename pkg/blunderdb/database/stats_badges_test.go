package database

import (
	"context"
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

// TestMatchBadgesScopedByIDs pins that passing matchIDs to MatchBadges returns
// exactly the same badge as the full-database scan for the requested match, and
// excludes the others — the property that lets a list page compute PR without
// scanning every decision in the library.
func TestMatchBadgesScopedByIDs(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	ss := db.store.Stats()

	tid := createTournament(t, db, "T", "2025-07-01")
	m1 := createMatch(t, db, "P1", "P2", "2025-07-02", 7, tid)
	g1 := createGame(t, db, m1)
	for i := range 40 {
		insertStatsFixtureRow(t, db, m1, g1, 10, 0, 0, i+1)
	}
	m2 := createMatch(t, db, "Q1", "Q2", "2025-07-03", 7, tid)
	g2 := createGame(t, db, m2)
	for i := range 20 {
		insertStatsFixtureRow(t, db, m2, g2, 25, 0, 0, i+1)
	}

	full, err := ss.MatchBadges(ctx, "", nil)
	if err != nil {
		t.Fatalf("MatchBadges(nil): %v", err)
	}
	if len(full) != 2 {
		t.Fatalf("full scan: got %d matches, want 2", len(full))
	}

	scoped, err := ss.MatchBadges(ctx, "", []int64{m1})
	if err != nil {
		t.Fatalf("MatchBadges([m1]): %v", err)
	}
	if len(scoped) != 1 {
		t.Fatalf("scoped: got %d matches, want 1 (only m1)", len(scoped))
	}
	if _, ok := scoped[m2]; ok {
		t.Errorf("scoped result leaked m2")
	}
	if scoped[m1] != full[m1] {
		t.Errorf("scoped badge for m1 %+v != full-scan badge %+v", scoped[m1], full[m1])
	}
}

// TestTournamentBadgesReferencePlayer checks the tournament list-row badge
// reports the reference player's own PR (the person present in the most of the
// tournament's matches), not a both-players pool. Here player "A" plays in both
// matches (vs B, then vs C), so A is the reference; the badge PR must equal
// ComputeStats filtered on A — never the pooled value that blends in B and C.
func TestTournamentBadgesReferencePlayer(t *testing.T) {
	db := newTestDB(t)

	tid := createTournament(t, db, "TB", "2025-07-01")
	// Match 1: A (player1) plays badly, B (player2) plays a little.
	m1 := createMatch(t, db, "A", "B", "2025-07-02", 7, tid)
	g1 := createGame(t, db, m1)
	for i := range 50 {
		insertStatsFixtureRow(t, db, m1, g1, 12, 0, 0, i+1) // A moves
	}
	for i := range 10 {
		insertStatsFixtureRow(t, db, m1, g1, 4, 0, 1, 100+i) // B moves
	}
	// Match 2: A (player1) again, opponent C (player2).
	m2 := createMatch(t, db, "A", "C", "2025-07-03", 7, tid)
	g2 := createGame(t, db, m2)
	for i := range 30 {
		insertStatsFixtureRow(t, db, m2, g2, 8, 0, 0, i+1) // A moves
	}
	for i := range 40 {
		insertStatsFixtureRow(t, db, m2, g2, 20, 0, 1, 100+i) // C moves (worst)
	}

	// Expected badge = A's own PR across the tournament (A appears in 2 matches;
	// B and C in 1 each), which is exactly ComputeStats filtered on A.
	res, err := db.ComputeStats(StatsFilter{PlayerName: "A", TournamentIDs: []int64{tid}, DecisionType: -1})
	if err != nil {
		t.Fatalf("ComputeStats(A): %v", err)
	}
	if len(res.PerTournament) != 1 {
		t.Fatalf("expected 1 tournament, got %d", len(res.PerTournament))
	}
	wantPR := res.PerTournament[0].PR

	// Sanity: the pooled PR (both players) must differ, else the test proves
	// nothing about the reference-player split.
	pooled, err := db.ComputeStats(StatsFilter{TournamentIDs: []int64{tid}, DecisionType: -1})
	if err != nil {
		t.Fatalf("ComputeStats(pooled): %v", err)
	}
	if math.Abs(pooled.PerTournament[0].PR-wantPR) < 1e-6 {
		t.Fatalf("test setup invalid: pooled PR == A's PR (%.4f)", wantPR)
	}

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
	if got.RefPlayer != "A" {
		t.Errorf("reference player = %q, want %q", got.RefPlayer, "A")
	}
	if math.Abs(got.PR-wantPR) > 1e-9 {
		t.Errorf("tournament badge PR %.6f != A's ComputeStats PR %.6f", got.PR, wantPR)
	}
}
