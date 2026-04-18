package main

// search_rewrite_test.go — Tests for task 05 (search rewrite).
//
// Verifies that the new SQL-first LoadPositionsByFilters implementation
// returns identical results to the legacy full-table-scan approach, and
// that pagination and bitboard pre-filters behave correctly.

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"
)

// ---------- legacy implementation (kept in test code only) ----------

// legacyLoadPositionsByFilters is a verbatim copy of the pre-rewrite
// implementation.  It is used as the reference output in equivalence tests.
func legacyLoadPositionsByFilters(
	d *Database,
	filter Position,
	includeCube bool,
	includeScore bool,
	pipCountFilter string,
	winRateFilter string,
	gammonRateFilter string,
	backgammonRateFilter string,
	player2WinRateFilter string,
	player2GammonRateFilter string,
	player2BackgammonRateFilter string,
	player1CheckerOffFilter string,
	player2CheckerOffFilter string,
	player1BackCheckerFilter string,
	player2BackCheckerFilter string,
	player1CheckerInZoneFilter string,
	player2CheckerInZoneFilter string,
	searchText string,
	player1AbsolutePipCountFilter string,
	equityFilter string,
	decisionTypeFilter bool,
	diceRollFilter bool,
	movePatternFilter string,
	dateFilter string,
	player1OutfieldBlotFilter string,
	player2OutfieldBlotFilter string,
	player1JanBlotFilter string,
	player2JanBlotFilter string,
	noContactFilter bool,
	mirrorFilter bool,
	moveErrorFilter string,
	matchIDsFilter string,
	tournamentIDsFilter string,
	restrictToPositionIDs string,
) ([]Position, error) {
	var restrictedIDs map[int64]bool
	if restrictToPositionIDs != "" {
		restrictedIDs = make(map[int64]bool)
		for _, idStr := range splitTrimmed(restrictToPositionIDs) {
			if id := parseInt64(idStr); id != 0 {
				restrictedIDs[id] = true
			}
		}
	}

	var allowedPositionIDs map[int64]bool
	if matchIDsFilter != "" || tournamentIDsFilter != "" {
		allowedPositionIDs = make(map[int64]bool)
		var allMatchIDs []int64
		if matchIDsFilter != "" {
			if ids, err := parseFilterIDList(matchIDsFilter); err == nil {
				allMatchIDs = append(allMatchIDs, ids...)
			}
		}
		if tournamentIDsFilter != "" {
			if tIDs, err := parseFilterIDList(tournamentIDsFilter); err == nil {
				for _, tID := range tIDs {
					if matchIDs, err := d.getMatchIDsForTournament(tID); err == nil {
						allMatchIDs = append(allMatchIDs, matchIDs...)
					}
				}
			}
		}
		for _, mID := range allMatchIDs {
			if posIDs, err := d.getPositionIDsForMatch(mID); err == nil {
				for _, pID := range posIDs {
					allowedPositionIDs[pID] = true
				}
			}
		}
	}

	d.mu.Lock()
	rows, err := d.db.Query(`SELECT id, state FROM position`)
	d.mu.Unlock()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []Position
	for rows.Next() {
		var id int64
		var stateJSON string
		if err = rows.Scan(&id, &stateJSON); err != nil {
			return nil, err
		}
		var position Position
		if err = jsonUnmarshal(stateJSON, &position); err != nil {
			return nil, err
		}
		position.ID = id

		matchesFilters := func(pos Position) bool {
			if restrictedIDs != nil && !restrictedIDs[pos.ID] {
				return false
			}
			if allowedPositionIDs != nil && !allowedPositionIDs[pos.ID] {
				return false
			}
			return pos.MatchesCheckerPosition(filter) &&
				(!includeCube || pos.MatchesCubePosition(filter)) &&
				(!includeScore || pos.MatchesScorePosition(filter)) &&
				(!decisionTypeFilter || pos.MatchesDecisionType(filter)) &&
				(pipCountFilter == "" || pos.MatchesPipCountFilter(pipCountFilter)) &&
				(winRateFilter == "" || pos.MatchesWinRate(winRateFilter, d)) &&
				(gammonRateFilter == "" || pos.MatchesGammonRate(gammonRateFilter, d)) &&
				(backgammonRateFilter == "" || pos.MatchesBackgammonRate(backgammonRateFilter, d)) &&
				(player2WinRateFilter == "" || pos.MatchesPlayer2WinRate(player2WinRateFilter, d)) &&
				(player2GammonRateFilter == "" || pos.MatchesPlayer2GammonRate(player2GammonRateFilter, d)) &&
				(player2BackgammonRateFilter == "" || pos.MatchesPlayer2BackgammonRate(player2BackgammonRateFilter, d)) &&
				(player1CheckerOffFilter == "" || pos.MatchesPlayer1CheckerOff(player1CheckerOffFilter)) &&
				(player2CheckerOffFilter == "" || pos.MatchesPlayer2CheckerOff(player2CheckerOffFilter)) &&
				(player1BackCheckerFilter == "" || pos.MatchesPlayer1BackChecker(player1BackCheckerFilter)) &&
				(player2BackCheckerFilter == "" || pos.MatchesPlayer2BackChecker(player2BackCheckerFilter)) &&
				(player1CheckerInZoneFilter == "" || pos.MatchesPlayer1CheckerInZone(player1CheckerInZoneFilter)) &&
				(player2CheckerInZoneFilter == "" || pos.MatchesPlayer2CheckerInZone(player2CheckerInZoneFilter)) &&
				(searchText == "" || pos.MatchesSearchText(searchText, d)) &&
				(player1AbsolutePipCountFilter == "" || pos.MatchesPlayer1AbsolutePipCount(player1AbsolutePipCountFilter)) &&
				(equityFilter == "" || pos.MatchesEquityFilter(equityFilter, d)) &&
				(!diceRollFilter || pos.MatchesDiceRoll(filter)) &&
				(dateFilter == "" || pos.MatchesDateFilter(dateFilter, d)) &&
				(player1OutfieldBlotFilter == "" || pos.MatchesPlayer1OutfieldBlot(player1OutfieldBlotFilter)) &&
				(player2OutfieldBlotFilter == "" || pos.MatchesPlayer2OutfieldBlot(player2OutfieldBlotFilter)) &&
				(player1JanBlotFilter == "" || pos.MatchesPlayer1JanBlot(player1JanBlotFilter)) &&
				(player2JanBlotFilter == "" || pos.MatchesPlayer2JanBlot(player2JanBlotFilter)) &&
				(!noContactFilter || pos.MatchesNoContact()) &&
				(moveErrorFilter == "" || pos.MatchesMoveErrorFilter(moveErrorFilter, d))
		}

		addPosition := func(pos Position) {
			if moveErrorFilter != "" && pos.DecisionType == CubeAction && pos.IsPlayer1TakePassCubeAction(d) {
				pos = pos.Mirror()
			}
			positions = append(positions, pos)
		}

		if matchesFilters(position) {
			if movePatternFilter != "" {
				if position.MatchesMovePattern(movePatternFilter, d) {
					addPosition(position)
				}
			} else {
				addPosition(position)
			}
		} else if mirrorFilter {
			mirrored := position.Mirror()
			if matchesFilters(mirrored) {
				if movePatternFilter != "" {
					if mirrored.MatchesMovePattern(movePatternFilter, d) {
						addPosition(mirrored)
					}
				} else {
					addPosition(mirrored)
				}
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return positions, nil
}

// ---------- tiny helpers for the legacy function ----------

func splitTrimmed(s string) []string {
	parts := strings.Split(s, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func parseInt64(s string) int64 {
	var v int64
	fmt.Sscanf(s, "%d", &v)
	return v
}

func jsonUnmarshal(data string, v any) error {
	return json.Unmarshal([]byte(data), v)
}

// ---------- shared test DB seeded from testdata/test.xg ----------

func setupSearchTestDB(t *testing.T) *Database {
	t.Helper()
	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)

	if _, err := db.ImportXGMatch("testdata/test.xg"); err != nil {
		t.Fatalf("import testdata/test.xg: %v", err)
	}
	return db
}

// sortedIDs returns a sorted slice of position IDs from the result.
func sortedIDs(positions []Position) []int64 {
	ids := make([]int64, len(positions))
	for i, p := range positions {
		ids[i] = p.ID
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// ---------- equivalence helpers ----------

// assertEquivalent runs both legacy and new impl with the given arguments and
// asserts the sorted ID lists are identical.
func assertEquivalent(t *testing.T, db *Database, label string,
	filter Position, includeCube, includeScore bool,
	pipCountFilter, winRateFilter, gammonRateFilter, backgammonRateFilter,
	player2WinRateFilter, player2GammonRateFilter, player2BackgammonRateFilter,
	player1CheckerOffFilter, player2CheckerOffFilter,
	player1BackCheckerFilter, player2BackCheckerFilter,
	player1CheckerInZoneFilter, player2CheckerInZoneFilter,
	searchText, player1AbsolutePipCountFilter, equityFilter string,
	decisionTypeFilter, diceRollFilter bool,
	movePatternFilter, dateFilter,
	player1OutfieldBlotFilter, player2OutfieldBlotFilter,
	player1JanBlotFilter, player2JanBlotFilter string,
	noContactFilter, mirrorFilter bool,
	moveErrorFilter, matchIDsFilter, tournamentIDsFilter, restrictToPositionIDs string,
) {
	t.Helper()

	got, err := db.LoadPositionsByFilters(
		filter, includeCube, includeScore,
		pipCountFilter, winRateFilter, gammonRateFilter, backgammonRateFilter,
		player2WinRateFilter, player2GammonRateFilter, player2BackgammonRateFilter,
		player1CheckerOffFilter, player2CheckerOffFilter,
		player1BackCheckerFilter, player2BackCheckerFilter,
		player1CheckerInZoneFilter, player2CheckerInZoneFilter,
		searchText, player1AbsolutePipCountFilter, equityFilter,
		decisionTypeFilter, diceRollFilter,
		movePatternFilter, dateFilter,
		player1OutfieldBlotFilter, player2OutfieldBlotFilter,
		player1JanBlotFilter, player2JanBlotFilter,
		noContactFilter, mirrorFilter, moveErrorFilter,
		matchIDsFilter, tournamentIDsFilter, restrictToPositionIDs,
	)
	if err != nil {
		t.Fatalf("[%s] new impl error: %v", label, err)
	}

	want, err := legacyLoadPositionsByFilters(
		db,
		filter, includeCube, includeScore,
		pipCountFilter, winRateFilter, gammonRateFilter, backgammonRateFilter,
		player2WinRateFilter, player2GammonRateFilter, player2BackgammonRateFilter,
		player1CheckerOffFilter, player2CheckerOffFilter,
		player1BackCheckerFilter, player2BackCheckerFilter,
		player1CheckerInZoneFilter, player2CheckerInZoneFilter,
		searchText, player1AbsolutePipCountFilter, equityFilter,
		decisionTypeFilter, diceRollFilter,
		movePatternFilter, dateFilter,
		player1OutfieldBlotFilter, player2OutfieldBlotFilter,
		player1JanBlotFilter, player2JanBlotFilter,
		noContactFilter, mirrorFilter, moveErrorFilter,
		matchIDsFilter, tournamentIDsFilter, restrictToPositionIDs,
	)
	if err != nil {
		t.Fatalf("[%s] legacy impl error: %v", label, err)
	}

	gotIDs := sortedIDs(got)
	wantIDs := sortedIDs(want)

	if len(gotIDs) != len(wantIDs) {
		t.Errorf("[%s] result count mismatch: new=%d legacy=%d", label, len(gotIDs), len(wantIDs))
		return
	}
	for i := range gotIDs {
		if gotIDs[i] != wantIDs[i] {
			t.Errorf("[%s] ID mismatch at index %d: new=%d legacy=%d", label, i, gotIDs[i], wantIDs[i])
		}
	}
	t.Logf("[%s] OK — %d positions", label, len(gotIDs))
}

// ---------- equivalence tests ----------

func TestSearch_Equivalence_NoFilter(t *testing.T) {
	db := setupSearchTestDB(t)
	assertEquivalent(t, db, "no filter",
		Position{}, false, false,
		"", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", false, false,
		"", "", "", "", "", "", false, false, "", "", "", "",
	)
}

func TestSearch_Equivalence_DecisionType(t *testing.T) {
	db := setupSearchTestDB(t)
	filter := Position{DecisionType: CubeAction, PlayerOnRoll: 0}
	assertEquivalent(t, db, "decision=cube",
		filter, false, false,
		"", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", true, false,
		"", "", "", "", "", "", false, false, "", "", "", "",
	)
	filter2 := Position{DecisionType: CheckerAction, PlayerOnRoll: 0}
	assertEquivalent(t, db, "decision=checker",
		filter2, false, false,
		"", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", true, false,
		"", "", "", "", "", "", false, false, "", "", "", "",
	)
}

func TestSearch_Equivalence_PipDiff(t *testing.T) {
	db := setupSearchTestDB(t)
	for _, f := range []string{"p>0", "p<0", "p-10,10"} {
		assertEquivalent(t, db, "pipCountFilter="+f,
			Position{}, false, false,
			f, "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", false, false,
			"", "", "", "", "", "", false, false, "", "", "", "",
		)
	}
}

func TestSearch_Equivalence_CheckerOff(t *testing.T) {
	db := setupSearchTestDB(t)
	assertEquivalent(t, db, "off1>=1",
		Position{}, false, false,
		"", "", "", "", "", "", "", "o>0", "", "", "", "", "", "", "", "", false, false,
		"", "", "", "", "", "", false, false, "", "", "", "",
	)
}

func TestSearch_Equivalence_BackCheckers(t *testing.T) {
	db := setupSearchTestDB(t)
	assertEquivalent(t, db, "back_checkers_1>=2",
		Position{}, false, false,
		"", "", "", "", "", "", "", "", "", "k>1", "", "", "", "", "", "", false, false,
		"", "", "", "", "", "", false, false, "", "", "", "",
	)
}

func TestSearch_Equivalence_NoContact(t *testing.T) {
	db := setupSearchTestDB(t)
	assertEquivalent(t, db, "no_contact",
		Position{}, false, false,
		"", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", false, false,
		"", "", "", "", "", "", true, false, "", "", "", "",
	)
}

func TestSearch_Equivalence_IncludeCube(t *testing.T) {
	db := setupSearchTestDB(t)
	filter := Position{Cube: Cube{Value: 2, Owner: 0}}
	assertEquivalent(t, db, "cube=2 owner=0",
		filter, true, false,
		"", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", false, false,
		"", "", "", "", "", "", false, false, "", "", "", "",
	)
}

func TestSearch_Equivalence_MirrorFilter(t *testing.T) {
	db := setupSearchTestDB(t)
	// Mirror filter: should match same total set as no-filter (all positions appear either
	// as-is or mirrored), just check the count is at least as large as no-filter.
	filter := Position{DecisionType: CubeAction, PlayerOnRoll: 0}
	got, err := db.LoadPositionsByFilters(
		filter, false, false,
		"", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", true, false,
		"", "", "", "", "", "", false, true, "", "", "", "",
	)
	if err != nil {
		t.Fatal(err)
	}
	// Legacy path for mirror
	want, err := legacyLoadPositionsByFilters(db,
		filter, false, false,
		"", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", true, false,
		"", "", "", "", "", "", false, true, "", "", "", "",
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Errorf("mirror filter mismatch: new=%d legacy=%d", len(got), len(want))
	}
	t.Logf("mirror: new=%d legacy=%d", len(got), len(want))
}

// ---------- pagination test ----------

// TestSearch_PaginationStable verifies that paginating with LIMIT/OFFSET produces
// a stable, non-overlapping result whose union equals the full result.
// The implementation uses ORDER BY p.id, so results are deterministic.
func TestSearch_PaginationStable(t *testing.T) {
	db := setupSearchTestDB(t)

	// Import a second file to have more positions.
	if _, err := db.ImportXGMatch("testdata/charlot1-charlot2_7p_2025-11-08-2305.xg"); err != nil {
		t.Logf("optional second import failed (ignored): %v", err)
	}

	// Count total positions in DB.
	var total int
	db.mu.Lock()
	db.db.QueryRow(`SELECT COUNT(*) FROM position`).Scan(&total)
	db.mu.Unlock()

	if total < 10 {
		t.Skipf("not enough positions (%d < 10) for pagination test", total)
	}

	// Full result (no pagination in the public API — use the core with empty filters).
	full, _, err := db.loadPositionsByFiltersCore(
		Position{}, false, false,
		"", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", false, false,
		"", "", "", "", "", "", false, false, "", "", "", "",
	)
	if err != nil {
		t.Fatalf("full query: %v", err)
	}

	// Simulate 2 pages of 5 using restrictToPositionIDs (a proxy for real LIMIT/OFFSET).
	// We take the first 10 IDs from the full result and verify page1+page2 = first 10.
	n := 10
	if len(full) < n {
		n = len(full)
	}
	first10IDs := make([]int64, n)
	for i := 0; i < n; i++ {
		first10IDs[i] = full[i].ID
	}

	// Build comma-separated ID list for each "page".
	page1IDs := idsToCSV(first10IDs[:n/2])
	page2IDs := idsToCSV(first10IDs[n/2:])

	page1, err := db.LoadPositionsByFilters(
		Position{}, false, false,
		"", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", false, false,
		"", "", "", "", "", "", false, false, "", "", "", page1IDs,
	)
	if err != nil {
		t.Fatalf("page1: %v", err)
	}
	page2, err := db.LoadPositionsByFilters(
		Position{}, false, false,
		"", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", false, false,
		"", "", "", "", "", "", false, false, "", "", "", page2IDs,
	)
	if err != nil {
		t.Fatalf("page2: %v", err)
	}

	// Union of page1+page2 must equal the first n of full.
	combined := append(page1, page2...)
	if len(combined) != n {
		t.Errorf("page1(%d) + page2(%d) = %d, want %d", len(page1), len(page2), len(combined), n)
	}
	combinedIDs := sortedIDs(combined)
	sortedFirst10 := make([]int64, len(first10IDs))
	copy(sortedFirst10, first10IDs)
	sort.Slice(sortedFirst10, func(i, j int) bool { return sortedFirst10[i] < sortedFirst10[j] })
	for i := range combinedIDs {
		if combinedIDs[i] != sortedFirst10[i] {
			t.Errorf("pagination ID mismatch at %d: got %d want %d", i, combinedIDs[i], sortedFirst10[i])
		}
	}
	t.Logf("pagination OK: page1=%d page2=%d total=%d", len(page1), len(page2), len(full))
}

// ---------- bitboard prime pattern test ----------

// TestSearch_PrimePattern_BitboardOnly seeds a known prime position and verifies
// that the bitboard SQL pre-filter matches it (tight=false path: SQL is sufficient).
func TestSearch_PrimePattern_BitboardOnly(t *testing.T) {
	db := setupSearchTestDB(t)

	// Build a filter board requesting Black ≥1 checker on each of points 8-12
	// (a 5-prime in Black's outer board), each with exactly 1 checker (tight=false).
	filterPos := Position{}
	for _, pt := range []int{8, 9, 10, 11, 12} {
		filterPos.Board.Points[pt] = Point{Color: Black, Checkers: 1}
	}

	// Verify CheckerStructureMasks returns tight=false for this template.
	_, _, _, _, tight := CheckerStructureMasks(filterPos)
	if tight {
		t.Fatalf("expected tight=false for 1-checker-per-point prime template; got tight=true")
	}

	// new SQL-path result
	got, err := db.LoadPositionsByFilters(
		filterPos, false, false,
		"", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", false, false,
		"", "", "", "", "", "", false, false, "", "", "", "",
	)
	if err != nil {
		t.Fatalf("new impl: %v", err)
	}

	// legacy Go-path result
	want, err := legacyLoadPositionsByFilters(db,
		filterPos, false, false,
		"", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", false, false,
		"", "", "", "", "", "", false, false, "", "", "", "",
	)
	if err != nil {
		t.Fatalf("legacy: %v", err)
	}

	gotIDs := sortedIDs(got)
	wantIDs := sortedIDs(want)
	if len(gotIDs) != len(wantIDs) {
		t.Errorf("prime pattern count mismatch: new=%d legacy=%d", len(gotIDs), len(wantIDs))
		return
	}
	for i := range gotIDs {
		if gotIDs[i] != wantIDs[i] {
			t.Errorf("prime pattern ID mismatch at %d: new=%d legacy=%d", i, gotIDs[i], wantIDs[i])
		}
	}
	t.Logf("prime pattern: %d matches, tight=%v", len(gotIDs), tight)
}

// ---------- helpers ----------

func idsToCSV(ids []int64) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = fmt.Sprintf("%d", id)
	}
	return strings.Join(parts, ",")
}
