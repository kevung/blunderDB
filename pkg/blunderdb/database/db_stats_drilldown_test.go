package database

import (
	"testing"
)

// ── cube fixture helper ───────────────────────────────────────────────────────

// insertCubeFixtureRow inserts one cube-decision row with a specific
// best_cube_action label and cube_error. Returns the position ID.
func insertCubeFixtureRow(t *testing.T, db *Database, matchID, gameID int64,
	cubeAction string, cubeErrMP, moveNum int) int64 {
	t.Helper()
	res, err := db.db.Exec(
		`INSERT INTO position (decision_type, state) VALUES (1, '')`,
	)
	if err != nil {
		t.Fatalf("insert cube position: %v", err)
	}
	posID, _ := res.LastInsertId()

	if _, err = db.db.Exec(
		`INSERT INTO analysis (position_id, data, best_cube_action, cube_error, best_move_equity_error, is_close_cube) VALUES (?, '{}', ?, ?, 0, 1)`,
		posID, cubeAction, cubeErrMP,
	); err != nil {
		t.Fatalf("insert cube analysis: %v", err)
	}
	if _, err = db.db.Exec(
		`INSERT INTO move (game_id, move_number, position_id, player) VALUES (?, ?, ?, 0)`,
		gameID, moveNum, posID,
	); err != nil {
		t.Fatalf("insert cube move: %v", err)
	}
	return posID
}

// ── buildSelectionWhereClause unit tests ─────────────────────────────────────

func TestBuildSelectionWhereClause_All(t *testing.T) {
	whereAdd, orderLimit, args := buildSelectionWhereClause(SelectionSpec{Kind: "all"})
	if whereAdd != "" {
		t.Errorf("expected empty whereAdd for 'all', got %q", whereAdd)
	}
	if orderLimit != "" {
		t.Errorf("expected empty orderLimit for 'all', got %q", orderLimit)
	}
	if len(args) != 0 {
		t.Errorf("expected no args for 'all', got %v", args)
	}
}

func TestBuildSelectionWhereClause_Checker(t *testing.T) {
	whereAdd, _, args := buildSelectionWhereClause(SelectionSpec{Kind: "checker"})
	if !containsStr(whereAdd, "decision_type = 0") {
		t.Errorf("missing decision_type=0 in whereAdd: %q", whereAdd)
	}
	if len(args) != 0 {
		t.Errorf("expected no args for checker without OnlyWithError, got %v", args)
	}
}

func TestBuildSelectionWhereClause_CheckerOnlyWithError(t *testing.T) {
	whereAdd, _, _ := buildSelectionWhereClause(SelectionSpec{Kind: "checker", OnlyWithError: true})
	if !containsStr(whereAdd, "decision_type = 0") {
		t.Errorf("missing decision_type=0: %q", whereAdd)
	}
	if !containsStr(whereAdd, "> 0") {
		t.Errorf("missing > 0 clause: %q", whereAdd)
	}
}

func TestBuildSelectionWhereClause_CubeAction(t *testing.T) {
	whereAdd, _, args := buildSelectionWhereClause(SelectionSpec{Kind: "cube_action", CubeAction: "DoubleTake"})
	if !containsStr(whereAdd, "best_cube_action = ?") {
		t.Errorf("missing best_cube_action=? in whereAdd: %q", whereAdd)
	}
	if len(args) != 1 || args[0] != "DoubleTake" {
		t.Errorf("expected args=[DoubleTake], got %v", args)
	}
}

func TestBuildSelectionWhereClause_ErrorBucket(t *testing.T) {
	whereAdd, _, args := buildSelectionWhereClause(SelectionSpec{Kind: "error_bucket", BucketMinMP: 50, BucketMaxMP: 100})
	if !containsStr(whereAdd, ">= ?") {
		t.Errorf("missing >= ? in whereAdd: %q", whereAdd)
	}
	if !containsStr(whereAdd, "< ?") {
		t.Errorf("missing < ? in whereAdd: %q", whereAdd)
	}
	if len(args) != 2 || args[0] != 50 || args[1] != 100 {
		t.Errorf("expected args=[50, 100], got %v", args)
	}
}

func TestBuildSelectionWhereClause_ErrorBucketUnbounded(t *testing.T) {
	whereAdd, _, args := buildSelectionWhereClause(SelectionSpec{Kind: "error_bucket", BucketMinMP: 100, BucketMaxMP: -1})
	if !containsStr(whereAdd, ">= ?") {
		t.Errorf("missing >= ? in whereAdd: %q", whereAdd)
	}
	if containsStr(whereAdd, "< ?") {
		t.Errorf("unexpected < ? for unbounded bucket: %q", whereAdd)
	}
	if len(args) != 1 || args[0] != 100 {
		t.Errorf("expected args=[100], got %v", args)
	}
}

func TestBuildSelectionWhereClause_LastN(t *testing.T) {
	whereAdd, orderLimit, args := buildSelectionWhereClause(SelectionSpec{Kind: "last_n", LastN: 50})
	if whereAdd != "" {
		t.Errorf("expected empty whereAdd for last_n, got %q", whereAdd)
	}
	if !containsStr(orderLimit, "LIMIT ?") {
		t.Errorf("missing LIMIT ? in orderLimit: %q", orderLimit)
	}
	if len(args) != 1 || args[0] != 50 {
		t.Errorf("expected args=[50], got %v", args)
	}
}

func TestBuildSelectionWhereClause_TopBlunders(t *testing.T) {
	// Default (LastN unset) keeps the historical limit of 10.
	whereAdd, orderLimit, args := buildSelectionWhereClause(SelectionSpec{Kind: "top_blunders"})
	if whereAdd != "" {
		t.Errorf("expected empty whereAdd for top_blunders, got %q", whereAdd)
	}
	if !containsStr(orderLimit, "LIMIT ?") || !containsStr(orderLimit, "DESC") {
		t.Errorf("expected DESC ... LIMIT ? in orderLimit: %q", orderLimit)
	}
	if len(args) != 1 || args[0] != 10 {
		t.Errorf("expected args=[10] (default limit), got %v", args)
	}
}

func TestBuildSelectionWhereClause_TopBlundersLastN(t *testing.T) {
	// LastN overrides the limit so `bl 50` can review the 50 worst decisions.
	_, orderLimit, args := buildSelectionWhereClause(SelectionSpec{Kind: "top_blunders", LastN: 50})
	if !containsStr(orderLimit, "LIMIT ?") {
		t.Errorf("missing LIMIT ? in orderLimit: %q", orderLimit)
	}
	if len(args) != 1 || args[0] != 50 {
		t.Errorf("expected args=[50], got %v", args)
	}
}

// ── integration tests ─────────────────────────────────────────────────────────

// TestDrilldown_InvariantCubeAction verifies that the count returned by
// GetPositionIDsByStatsSelection with Kind="cube_action" matches the
// NumDecisions in the CubeActionBreakdown from ComputeStats.
func TestDrilldown_InvariantCubeAction(t *testing.T) {
	db := newTestDB(t)

	m := createMatch(t, db, "A", "B", "2025-01-01", 7, 0)
	g := createGame(t, db, m)

	// Insert 3 NoDouble and 2 DoubleTake cube decisions
	for i := range 3 {
		insertCubeFixtureRow(t, db, m, g, "NoDouble", 50, i+1)
	}
	for i := range 2 {
		insertCubeFixtureRow(t, db, m, g, "DoubleTake", 80, i+10)
	}

	filter := StatsFilter{DecisionType: -1}
	res, err := db.ComputeStats(filter)
	if err != nil {
		t.Fatalf("ComputeStats: %v", err)
	}

	// Find the NoDouble entry in the breakdown
	var ndDecisions int
	for _, cs := range res.CubeActionBreakdown {
		if cs.Action == "NoDouble" {
			ndDecisions = cs.NumDecisions
			break
		}
	}
	if ndDecisions != 3 {
		t.Fatalf("expected 3 NoDouble in breakdown, got %d", ndDecisions)
	}

	ids, err := db.GetPositionIDsByStatsSelection(filter, SelectionSpec{Kind: "cube_action", CubeAction: "NoDouble"})
	if err != nil {
		t.Fatalf("GetPositionIDsByStatsSelection: %v", err)
	}

	if len(ids) != ndDecisions {
		t.Errorf("drilldown count %d != CubeActionBreakdown.NumDecisions %d for NoDouble", len(ids), ndDecisions)
	}
}

// TestDrilldown_InvariantErrorBucket verifies that the count from drilldown
// matches ErrorBucket.Count for each bucket.
func TestDrilldown_InvariantErrorBucket(t *testing.T) {
	db := newTestDB(t)

	m := createMatch(t, db, "A", "B", "2025-02-01", 7, 0)
	g := createGame(t, db, m)

	// Errors spanning different buckets:
	// <5: 2 rows (errors 2, 3)
	// 5-9: 1 row (error 7)
	// 10-24: 2 rows (errors 12, 20)
	// 50-99: 1 row (error 75)
	// 100+: 1 row (error 150)
	for i, e := range []int{2, 3, 7, 12, 20, 75, 150} {
		insertStatsFixtureRow(t, db, m, g, e, 0, 0, i+1)
	}

	filter := StatsFilter{DecisionType: -1}
	res, err := db.ComputeStats(filter)
	if err != nil {
		t.Fatalf("ComputeStats: %v", err)
	}

	// For each bucket, verify drilldown count matches histogram count
	for _, bucket := range res.ErrorHistogram {
		ids, err := db.GetPositionIDsByStatsSelection(filter, SelectionSpec{
			Kind:        "error_bucket",
			BucketMinMP: bucket.MinMP,
			BucketMaxMP: bucket.MaxMP,
		})
		if err != nil {
			t.Fatalf("GetPositionIDsByStatsSelection bucket [%d,%d): %v", bucket.MinMP, bucket.MaxMP, err)
		}
		if len(ids) != bucket.Count {
			t.Errorf("bucket [%d,%d): drilldown=%d histogram=%d", bucket.MinMP, bucket.MaxMP, len(ids), bucket.Count)
		}
	}
}

// TestDrilldown_InvariantChecker verifies that Kind="checker" returns exactly
// the checker-play positions in the fixture.
func TestDrilldown_InvariantChecker(t *testing.T) {
	db := newTestDB(t)

	m := createMatch(t, db, "A", "B", "2025-03-01", 7, 0)
	g := createGame(t, db, m)

	// 4 checker decisions (dt=0) + 2 cube decisions (dt=1)
	for i := range 4 {
		insertStatsFixtureRow(t, db, m, g, 30, 0, 0, i+1)
	}
	for i := range 2 {
		insertCubeFixtureRow(t, db, m, g, "NoDouble", 30, i+10)
	}

	filter := StatsFilter{DecisionType: -1}
	ids, err := db.GetPositionIDsByStatsSelection(filter, SelectionSpec{Kind: "checker"})
	if err != nil {
		t.Fatalf("GetPositionIDsByStatsSelection: %v", err)
	}
	if len(ids) != 4 {
		t.Errorf("checker drilldown returned %d IDs, want 4", len(ids))
	}
}

// TestDrilldown_OnlyWithError verifies that OnlyWithError=true filters out
// positions with zero error, returning only genuine blunders.
func TestDrilldown_OnlyWithError(t *testing.T) {
	db := newTestDB(t)

	m := createMatch(t, db, "A", "B", "2025-04-01", 7, 0)
	g := createGame(t, db, m)

	// 3 DoubleTake with error 0 (correct plays) + 2 DoubleTake with error > 0
	for i := range 3 {
		insertCubeFixtureRow(t, db, m, g, "DoubleTake", 0, i+1)
	}
	for i := range 2 {
		insertCubeFixtureRow(t, db, m, g, "DoubleTake", 120, i+10)
	}

	filter := StatsFilter{DecisionType: -1}

	// Without OnlyWithError: all 5 DoubleTake positions
	all, err := db.GetPositionIDsByStatsSelection(filter, SelectionSpec{Kind: "cube_action", CubeAction: "DoubleTake"})
	if err != nil {
		t.Fatalf("GetPositionIDsByStatsSelection (all): %v", err)
	}
	if len(all) != 5 {
		t.Errorf("expected 5 DoubleTake positions, got %d", len(all))
	}

	// With OnlyWithError: only the 2 positions with error > 0
	errOnly, err := db.GetPositionIDsByStatsSelection(filter, SelectionSpec{Kind: "cube_action", CubeAction: "DoubleTake", OnlyWithError: true})
	if err != nil {
		t.Fatalf("GetPositionIDsByStatsSelection (OnlyWithError): %v", err)
	}
	if len(errOnly) != 2 {
		t.Errorf("expected 2 blunder DoubleTake positions, got %d", len(errOnly))
	}
}

// TestDrilldown_TournamentShortcut verifies that GetPositionIDsByTournament
// ignores any player/date filter that would otherwise exclude all positions.
func TestDrilldown_TournamentShortcut(t *testing.T) {
	db := newTestDB(t)

	tid := createTournament(t, db, "Open2025", "2025-05-01")
	m := createMatch(t, db, "Alice", "Bob", "2025-05-10", 7, tid)
	g := createGame(t, db, m)
	for i := range 4 {
		insertStatsFixtureRow(t, db, m, g, 50, 0, 0, i+1)
	}

	// A filter that would exclude everything if applied (wrong player name).
	biasFilter := StatsFilter{PlayerName: "Nobody", DecisionType: -1}

	// GetPositionIDsByStatsSelection WITH this filter → 0 results
	filtered, err := db.GetPositionIDsByStatsSelection(biasFilter, SelectionSpec{Kind: "tournament", TournamentID: tid})
	if err != nil {
		t.Fatalf("GetPositionIDsByStatsSelection: %v", err)
	}
	if len(filtered) != 0 {
		t.Errorf("filtered selection with wrong player should return 0 IDs, got %d", len(filtered))
	}

	// GetPositionIDsByTournament IGNORES the filter → returns all 4 positions
	all, err := db.GetPositionIDsByTournament(tid)
	if err != nil {
		t.Fatalf("GetPositionIDsByTournament: %v", err)
	}
	if len(all) != 4 {
		t.Errorf("tournament shortcut should return 4 positions regardless of filter, got %d", len(all))
	}
}

// TestDrilldown_MatchShortcut verifies that GetPositionIDsByMatch ignores any
// filter that would otherwise exclude all positions.
func TestDrilldown_MatchShortcut(t *testing.T) {
	db := newTestDB(t)

	m := createMatch(t, db, "Alice", "Bob", "2025-06-01", 7, 0)
	g := createGame(t, db, m)
	for i := range 3 {
		insertStatsFixtureRow(t, db, m, g, 40, 0, 0, i+1)
	}

	// A filter that would exclude everything if applied.
	biasFilter := StatsFilter{PlayerName: "Nobody", DecisionType: -1}

	filtered, err := db.GetPositionIDsByStatsSelection(biasFilter, SelectionSpec{Kind: "match", MatchID: m})
	if err != nil {
		t.Fatalf("GetPositionIDsByStatsSelection: %v", err)
	}
	if len(filtered) != 0 {
		t.Errorf("filtered selection with wrong player should return 0 IDs, got %d", len(filtered))
	}

	all, err := db.GetPositionIDsByMatch(m)
	if err != nil {
		t.Fatalf("GetPositionIDsByMatch: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("match shortcut should return 3 positions, got %d", len(all))
	}
}

// TestDrilldown_TopBlunders verifies that Kind="top_blunders" returns at most
// 10 IDs and that their order is consistent with result.TopBlunders from
// ComputeStats.
func TestDrilldown_TopBlunders(t *testing.T) {
	db := newTestDB(t)

	m := createMatch(t, db, "A", "B", "2025-07-01", 7, 0)
	g := createGame(t, db, m)

	// 12 positions with distinct errors; top 10 should be the 10 largest.
	errors := []int{5, 800, 30, 600, 10, 1000, 50, 700, 20, 900, 15, 400}
	for i, e := range errors {
		insertStatsFixtureRow(t, db, m, g, e, 0, 0, i+1)
	}

	filter := StatsFilter{DecisionType: -1}
	res, err := db.ComputeStats(filter)
	if err != nil {
		t.Fatalf("ComputeStats: %v", err)
	}
	if len(res.TopBlunders) != 10 {
		t.Fatalf("ComputeStats TopBlunders length = %d, want 10", len(res.TopBlunders))
	}

	ids, err := db.GetPositionIDsByStatsSelection(filter, SelectionSpec{Kind: "top_blunders"})
	if err != nil {
		t.Fatalf("GetPositionIDsByStatsSelection top_blunders: %v", err)
	}
	if len(ids) > 10 {
		t.Errorf("top_blunders returned %d IDs, want ≤ 10", len(ids))
	}
	if len(ids) != len(res.TopBlunders) {
		t.Errorf("top_blunders IDs count %d != ComputeStats TopBlunders count %d", len(ids), len(res.TopBlunders))
	}

	// IDs from drilldown must match IDs from ComputeStats TopBlunders (same order).
	statsIDs := make(map[int64]bool)
	for _, be := range res.TopBlunders {
		statsIDs[be.PositionID] = true
	}
	for _, id := range ids {
		if !statsIDs[id] {
			t.Errorf("drilldown returned position ID %d not in ComputeStats TopBlunders", id)
		}
	}
}

// TestDrilldown_LastN verifies that Kind="last_n" returns at most N IDs.
func TestDrilldown_LastN(t *testing.T) {
	db := newTestDB(t)

	m := createMatch(t, db, "A", "B", "2025-08-01", 7, 0)
	g := createGame(t, db, m)
	for i := range 8 {
		insertStatsFixtureRow(t, db, m, g, 20, 0, 0, i+1)
	}

	filter := StatsFilter{DecisionType: -1}
	ids, err := db.GetPositionIDsByStatsSelection(filter, SelectionSpec{Kind: "last_n", LastN: 5})
	if err != nil {
		t.Fatalf("GetPositionIDsByStatsSelection last_n: %v", err)
	}
	if len(ids) > 5 {
		t.Errorf("last_n=5 returned %d IDs, want ≤ 5", len(ids))
	}
	if len(ids) != 5 {
		t.Errorf("last_n=5 with 8 available should return 5, got %d", len(ids))
	}
}

// TestDrilldown_Position verifies that Kind="position" returns exactly the
// targeted position ID.
func TestDrilldown_Position(t *testing.T) {
	db := newTestDB(t)

	m := createMatch(t, db, "A", "B", "2025-09-01", 7, 0)
	g := createGame(t, db, m)
	target := insertStatsFixtureRow(t, db, m, g, 100, 0, 0, 1)
	insertStatsFixtureRow(t, db, m, g, 200, 0, 0, 2)

	filter := StatsFilter{DecisionType: -1}
	ids, err := db.GetPositionIDsByStatsSelection(filter, SelectionSpec{Kind: "position", PositionID: target})
	if err != nil {
		t.Fatalf("GetPositionIDsByStatsSelection position: %v", err)
	}
	if len(ids) != 1 || ids[0] != target {
		t.Errorf("position drilldown: got %v, want [%d]", ids, target)
	}
}
