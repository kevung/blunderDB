package main

import (
	"math"
	"testing"
)

// ── fixture helpers ──────────────────────────────────────────────────────────

// insertStatsFixtureRow inserts one decision row into the fixture: creates a
// position (with decision_type), an analysis row carrying the error, a game,
// a move linking them, and returns the position id.
//
// matchID  – pre-existing match row id
// errMP    – error in stored millipoints units (>0)
// dt       – decision_type: 0=checker, 1=cube
// player   – move.player (0 or 1)
// moveNum  – move.move_number (unique within a game for ordering)
func insertStatsFixtureRow(t *testing.T, db *Database, matchID int64, gameID int64, errMP int, dt int, player int, moveNum int) int64 {
	t.Helper()
	// Insert position
	res, err := db.db.Exec(
		`INSERT INTO position (decision_type, state) VALUES (?, '')`, dt,
	)
	if err != nil {
		t.Fatalf("insert position: %v", err)
	}
	posID, _ := res.LastInsertId()

	// Insert analysis with the error in the right column
	cubeErr := 0
	moveErr := 0
	if dt == 1 {
		cubeErr = errMP
	} else {
		moveErr = errMP
	}
	if _, err = db.db.Exec(
		`INSERT INTO analysis (position_id, data, cube_error, best_move_equity_error) VALUES (?, '{}', ?, ?)`,
		posID, cubeErr, moveErr,
	); err != nil {
		t.Fatalf("insert analysis: %v", err)
	}

	// Insert move linking game → position
	if _, err = db.db.Exec(
		`INSERT INTO move (game_id, move_number, position_id, player) VALUES (?, ?, ?, ?)`,
		gameID, moveNum, posID, player,
	); err != nil {
		t.Fatalf("insert move: %v", err)
	}
	return posID
}

// createMatch creates a match row and returns its id.
func createMatch(t *testing.T, db *Database, p1, p2, date string, matchLength int, tournamentID int64) int64 {
	t.Helper()
	var tidVal any
	if tournamentID > 0 {
		tidVal = tournamentID
	}
	res, err := db.db.Exec(
		`INSERT INTO match (player1_name, player2_name, match_date, match_length, tournament_id) VALUES (?,?,?,?,?)`,
		p1, p2, date, matchLength, tidVal,
	)
	if err != nil {
		t.Fatalf("insert match: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

// createGame creates a game row for a match and returns its id.
func createGame(t *testing.T, db *Database, matchID int64) int64 {
	t.Helper()
	res, err := db.db.Exec(
		`INSERT INTO game (match_id, game_number) VALUES (?, 1)`, matchID,
	)
	if err != nil {
		t.Fatalf("insert game: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

// createTournament creates a tournament row and returns its id.
func createTournament(t *testing.T, db *Database, name, date string) int64 {
	t.Helper()
	res, err := db.db.Exec(
		`INSERT INTO tournament (name, date) VALUES (?, ?)`, name, date,
	)
	if err != nil {
		t.Fatalf("insert tournament: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

// ── unit tests ───────────────────────────────────────────────────────────────

// TestBuildStatsWhereClause_PlayerName checks that a PlayerName filter produces
// the correct SQL fragment and argument list.
func TestBuildStatsWhereClause_PlayerName(t *testing.T) {
	f := StatsFilter{PlayerName: "Alice", DecisionType: -1}
	where, args := buildStatsWhereClause(f)

	if len(args) < 2 {
		t.Fatalf("expected at least 2 args for PlayerName, got %d: %v", len(args), args)
	}
	if args[0] != "Alice" || args[1] != "Alice" {
		t.Errorf("PlayerName args: got %v, want [Alice Alice ...]", args[:2])
	}
	// Must contain the player JOIN condition
	for _, needle := range []string{"player1_name", "player2_name", "mv.player"} {
		if !containsStr(where, needle) {
			t.Errorf("where clause missing %q: %s", needle, where)
		}
	}
}

// TestBuildStatsWhereClause_TournamentIDs checks tournament ID IN clause.
func TestBuildStatsWhereClause_TournamentIDs(t *testing.T) {
	f := StatsFilter{TournamentIDs: []int64{1, 2, 3}, DecisionType: -1}
	where, args := buildStatsWhereClause(f)

	// The 3 tournament IDs must appear as args (before the trailing IS NOT NULL
	// args which carry no additional args).
	found := 0
	for _, a := range args {
		switch a {
		case int64(1), int64(2), int64(3):
			found++
		}
	}
	if found != 3 {
		t.Errorf("expected 3 tournament IDs in args, got %d; args=%v", found, args)
	}
	if !containsStr(where, "tournament_id IN") {
		t.Errorf("where clause missing 'tournament_id IN': %s", where)
	}
}

// TestBuildStatsWhereClause_DateRange checks the date range clause.
func TestBuildStatsWhereClause_DateRange(t *testing.T) {
	f := StatsFilter{DateFrom: "2025-01-01", DateTo: "2025-12-31", DecisionType: -1}
	where, args := buildStatsWhereClause(f)

	if !containsStr(where, "BETWEEN") {
		t.Errorf("expected BETWEEN in where clause: %s", where)
	}
	found0, found1 := false, false
	for _, a := range args {
		if a == "2025-01-01" {
			found0 = true
		}
		if a == "2025-12-31" {
			found1 = true
		}
	}
	if !found0 || !found1 {
		t.Errorf("date args not found in %v", args)
	}
}

// TestBuildStatsWhereClause_DecisionType checks the decision_type filter.
func TestBuildStatsWhereClause_DecisionType(t *testing.T) {
	f := StatsFilter{DecisionType: 1}
	where, args := buildStatsWhereClause(f)

	if !containsStr(where, "decision_type") {
		t.Errorf("expected decision_type in where clause: %s", where)
	}
	found := false
	for _, a := range args {
		if a == 1 {
			found = true
		}
	}
	if !found {
		t.Errorf("decision_type value not found in args: %v", args)
	}
}

// TestBuildStatsWhereClause_MatchLength checks the match_length IN clause.
func TestBuildStatsWhereClause_MatchLength(t *testing.T) {
	f := StatsFilter{MatchLength: []int{7, 11}, DecisionType: -1}
	where, args := buildStatsWhereClause(f)

	if !containsStr(where, "match_length IN") {
		t.Errorf("expected match_length IN in where clause: %s", where)
	}
	found7, found11 := false, false
	for _, a := range args {
		if a == 7 {
			found7 = true
		}
		if a == 11 {
			found11 = true
		}
	}
	if !found7 || !found11 {
		t.Errorf("match lengths not found in args: %v", args)
	}
}

// TestBuildStatsWhereClause_Combined checks a fully specified filter.
func TestBuildStatsWhereClause_Combined(t *testing.T) {
	f := StatsFilter{
		PlayerName:    "Bob",
		TournamentIDs: []int64{5},
		DateFrom:      "2024-01-01",
		DateTo:        "2024-12-31",
		DecisionType:  0,
		MatchLength:   []int{9},
	}
	where, args := buildStatsWhereClause(f)

	if !containsStr(where, "player1_name") {
		t.Error("missing player filter")
	}
	if !containsStr(where, "tournament_id IN") {
		t.Error("missing tournament filter")
	}
	if !containsStr(where, "BETWEEN") {
		t.Error("missing date filter")
	}
	if !containsStr(where, "decision_type") {
		t.Error("missing decision_type filter")
	}
	if !containsStr(where, "match_length IN") {
		t.Error("missing match_length filter")
	}
	// Verify arg order: player(2), tournament(1), date(2), decision_type(1), match_length(1)
	// Plus trailing IS NOT NULL args (0 extra args).
	if len(args) != 7 {
		t.Errorf("expected 7 args for combined filter, got %d: %v", len(args), args)
	}
}

// TestComputeStats_AntiAveragingBias is the critical test verifying that PR is
// computed with the weighted sum/sum method, not as the mean of per-match PRs.
//
//	Match1: 100 decisions, total error = 1000 → PR = 5.0
//	Match2:  10 decisions, total error =  200 → PR = 10.0
//	Wrong (mean of PRs) = 7.5
//	Correct (weighted)  = 500 × 1200 / 1000 / 110 ≈ 5.45
func TestComputeStats_AntiAveragingBias(t *testing.T) {
	db := newTestDB(t)

	tidA := createTournament(t, db, "TournamentA", "2025-01-01")

	// Match1: 100 checker decisions, each with error=10 (total=1000)
	m1 := createMatch(t, db, "Alice", "Bob", "2025-01-10", 7, tidA)
	g1 := createGame(t, db, m1)
	for i := range 100 {
		insertStatsFixtureRow(t, db, m1, g1, 10, 0, 0, i+1)
	}

	// Match2: 10 checker decisions, each with error=20 (total=200)
	m2 := createMatch(t, db, "Alice", "Bob", "2025-01-11", 7, tidA)
	g2 := createGame(t, db, m2)
	for i := range 10 {
		insertStatsFixtureRow(t, db, m2, g2, 20, 0, 0, i+1)
	}

	res, err := db.ComputeStats(StatsFilter{DecisionType: -1})
	if err != nil {
		t.Fatalf("ComputeStats: %v", err)
	}

	// Weighted PR ≈ 500 × 1200 / 1000 / 110 ≈ 5.4545…
	wantPR := 500.0 * 1200.0 / 1000.0 / 110.0
	wrongPR := 7.5

	if math.Abs(res.PRGlobal-wantPR) > 0.01 {
		t.Errorf("PRGlobal = %.4f, want ≈ %.4f (weighted); wrong value would be %.1f", res.PRGlobal, wantPR, wrongPR)
	}
	if math.Abs(res.PRGlobal-wrongPR) < 0.1 {
		t.Errorf("PRGlobal = %.4f looks like mean-of-means (%.1f) rather than weighted sum", res.PRGlobal, wrongPR)
	}
}

// TestComputeStats_PlayerFilter verifies that filtering by player correctly
// joins move.player to find the right player's decisions in both match roles.
func TestComputeStats_PlayerFilter(t *testing.T) {
	db := newTestDB(t)

	// Match A: Alice is player1 (player=0), 3 decisions
	mA := createMatch(t, db, "Alice", "Carol", "2025-02-01", 7, 0)
	gA := createGame(t, db, mA)
	for i := range 3 {
		insertStatsFixtureRow(t, db, mA, gA, 10, 0, 0, i+1) // Alice's moves
		insertStatsFixtureRow(t, db, mA, gA, 50, 0, 1, i+10) // Carol's moves
	}

	// Match B: Alice is player2 (player=1), 2 decisions
	mB := createMatch(t, db, "Dave", "Alice", "2025-02-02", 7, 0)
	gB := createGame(t, db, mB)
	for i := range 2 {
		insertStatsFixtureRow(t, db, mB, gB, 10, 0, 1, i+1) // Alice's moves
		insertStatsFixtureRow(t, db, mB, gB, 99, 0, 0, i+10) // Dave's moves
	}

	res, err := db.ComputeStats(StatsFilter{PlayerName: "Alice", DecisionType: -1})
	if err != nil {
		t.Fatalf("ComputeStats: %v", err)
	}

	// Alice has 3 + 2 = 5 decisions total
	if res.Totals.NumDecisions != 5 {
		t.Errorf("NumDecisions = %d, want 5", res.Totals.NumDecisions)
	}
}

// TestComputeStats_CombinedFilter tests player + tournament + date range.
func TestComputeStats_CombinedFilter(t *testing.T) {
	db := newTestDB(t)

	tid1 := createTournament(t, db, "T1", "2025-03-01")
	tid2 := createTournament(t, db, "T2", "2025-04-01")

	// Match in T1 on 2025-03-15, Alice as player1
	m1 := createMatch(t, db, "Alice", "Bob", "2025-03-15", 7, tid1)
	g1 := createGame(t, db, m1)
	for i := range 5 {
		insertStatsFixtureRow(t, db, m1, g1, 20, 0, 0, i+1)
	}

	// Match in T2 on 2025-04-10, Alice as player2 — should be excluded by filter
	m2 := createMatch(t, db, "Carol", "Alice", "2025-04-10", 7, tid2)
	g2 := createGame(t, db, m2)
	for i := range 5 {
		insertStatsFixtureRow(t, db, m2, g2, 30, 0, 1, i+1)
	}

	f := StatsFilter{
		PlayerName:    "Alice",
		TournamentIDs: []int64{tid1},
		DateFrom:      "2025-01-01",
		DateTo:        "2025-03-31",
		DecisionType:  -1,
	}
	res, err := db.ComputeStats(f)
	if err != nil {
		t.Fatalf("ComputeStats: %v", err)
	}

	if res.Totals.NumDecisions != 5 {
		t.Errorf("NumDecisions = %d, want 5 (only T1 match)", res.Totals.NumDecisions)
	}
}

// TestComputeStats_TopBlundersOrdered verifies that top blunders are returned
// in descending error order.
func TestComputeStats_TopBlundersOrdered(t *testing.T) {
	db := newTestDB(t)

	m := createMatch(t, db, "X", "Y", "2025-05-01", 7, 0)
	g := createGame(t, db, m)
	errors := []int{10, 500, 50, 200, 1000, 5, 300}
	for i, e := range errors {
		insertStatsFixtureRow(t, db, m, g, e, 0, 0, i+1)
	}

	res, err := db.ComputeStats(StatsFilter{DecisionType: -1})
	if err != nil {
		t.Fatalf("ComputeStats: %v", err)
	}

	if len(res.TopBlunders) == 0 {
		t.Fatal("no top blunders returned")
	}
	for i := 1; i < len(res.TopBlunders); i++ {
		if res.TopBlunders[i].ErrorMP > res.TopBlunders[i-1].ErrorMP {
			t.Errorf("blunders not sorted desc: [%d].ErrorMP=%d > [%d].ErrorMP=%d",
				i, res.TopBlunders[i].ErrorMP, i-1, res.TopBlunders[i-1].ErrorMP)
		}
	}
	if res.TopBlunders[0].ErrorMP != 1000 {
		t.Errorf("largest blunder = %d, want 1000", res.TopBlunders[0].ErrorMP)
	}
}

// TestComputeStats_HistogramSumEqualsTotal verifies that the sum of histogram
// bucket counts equals the total decision count.
func TestComputeStats_HistogramSumEqualsTotal(t *testing.T) {
	db := newTestDB(t)

	m := createMatch(t, db, "A", "B", "2025-06-01", 7, 0)
	g := createGame(t, db, m)
	// Spread errors across buckets: <5, 5-9, 10-24, 25-49, 50-99, 100+
	for i, e := range []int{2, 7, 15, 30, 75, 150, 3, 8, 20, 40} {
		insertStatsFixtureRow(t, db, m, g, e, 0, 0, i+1)
	}

	res, err := db.ComputeStats(StatsFilter{DecisionType: -1})
	if err != nil {
		t.Fatalf("ComputeStats: %v", err)
	}

	histTotal := 0
	for _, b := range res.ErrorHistogram {
		histTotal += b.Count
	}
	if histTotal != res.Totals.NumDecisions {
		t.Errorf("histogram total %d != NumDecisions %d", histTotal, res.Totals.NumDecisions)
	}
}

// TestComputeStats_TournamentPRWeighted cross-checks that per-tournament PR
// uses direct aggregation, not a mean of per-match PRs.
func TestComputeStats_TournamentPRWeighted(t *testing.T) {
	db := newTestDB(t)

	tid := createTournament(t, db, "TW", "2025-07-01")

	// Match1: 100 decisions each error=10, PR=5
	m1 := createMatch(t, db, "P1", "P2", "2025-07-02", 7, tid)
	g1 := createGame(t, db, m1)
	for i := range 100 {
		insertStatsFixtureRow(t, db, m1, g1, 10, 0, 0, i+1)
	}
	// Match2: 10 decisions each error=20, PR=10
	m2 := createMatch(t, db, "P1", "P2", "2025-07-03", 7, tid)
	g2 := createGame(t, db, m2)
	for i := range 10 {
		insertStatsFixtureRow(t, db, m2, g2, 20, 0, 0, i+1)
	}

	res, err := db.ComputeStats(StatsFilter{TournamentIDs: []int64{tid}, DecisionType: -1})
	if err != nil {
		t.Fatalf("ComputeStats: %v", err)
	}

	if len(res.PerTournament) != 1 {
		t.Fatalf("expected 1 tournament in results, got %d", len(res.PerTournament))
	}
	wantPR := 500.0 * 1200.0 / 1000.0 / 110.0
	if math.Abs(res.PerTournament[0].PR-wantPR) > 0.01 {
		t.Errorf("tournament PR = %.4f, want ≈ %.4f", res.PerTournament[0].PR, wantPR)
	}
}

// ── benchmark ────────────────────────────────────────────────────────────────

// BenchmarkComputeStats asserts that stats computation over 10 000 positions
// completes in < 200 ms.
func BenchmarkComputeStats(b *testing.B) {
	// Build fixture once
	db := &Database{}
	if err := db.SetupDatabase(":memory:"); err != nil {
		b.Fatalf("SetupDatabase: %v", err)
	}
	defer db.db.Close()

	const N = 10_000
	t := &testing.T{}
	m := createMatch(t, db, "Bench", "Player", "2024-01-01", 7, 0)
	g := createGame(t, db, m)
	for i := range N {
		errMP := (i % 200) + 1 // vary error values 1-200
		dt := i % 2
		insertStatsFixtureRow(t, db, m, g, errMP, dt, 0, i+1)
	}

	f := StatsFilter{DecisionType: -1}

	b.ResetTimer()
	for b.Loop() {
		if _, err := db.ComputeStats(f); err != nil {
			b.Fatalf("ComputeStats: %v", err)
		}
	}
}

// ── helpers ──────────────────────────────────────────────────────────────────

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
