package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// captureSlog redirects the package-level slog default to a buffer for the
// duration of the test (restored via t.Cleanup), so a test can assert on
// what a function logged via the bare slog.Warn/Info/... package funcs
// (as opposed to an injected *slog.Logger).
func captureSlog(t *testing.T) *strings.Builder {
	t.Helper()
	var buf strings.Builder
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(old) })
	return &buf
}

// setupExportTestDB creates a source database populated with test data:
// - 2 positions with analysis (including played moves), comments
// - 2 filter_library entries
// - 1 match → 1 game → 2 moves (with move_analysis)
// - 1 collection containing both positions
// - 1 tournament linked to the match
// It returns the Database handle and a cleanup function.
func setupExportTestDB(t *testing.T) (*Database, string, func()) {
	t.Helper()

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "source.db")

	db := NewDatabase()
	if err := db.SetupDatabase(srcPath); err != nil {
		t.Fatalf("SetupDatabase failed: %v", err)
	}

	// --- Positions ---
	pos1 := InitializePosition()
	id1, err := db.SavePosition(&pos1)
	if err != nil {
		t.Fatalf("SavePosition 1 failed: %v", err)
	}

	pos2 := InitializePosition()
	pos2.Dice = [2]int{6, 5}
	id2, err := db.SavePosition(&pos2)
	if err != nil {
		t.Fatalf("SavePosition 2 failed: %v", err)
	}

	// --- Analysis with played moves ---
	analysis1 := PositionAnalysis{
		PositionID:        int(id1),
		XGID:              "test-xgid-1",
		AnalysisType:      "XG Roller++",
		PlayedMove:        "8/5 6/5",
		PlayedMoves:       []string{"8/5 6/5"},
		PlayedCubeAction:  "No double",
		PlayedCubeActions: []string{"No double"},
		CheckerAnalysis: &CheckerAnalysis{
			Moves: []CheckerMove{
				{Index: 1, Move: "8/5 6/5", Equity: 0.123, PlayerWinChance: 55.0},
				{Index: 2, Move: "13/10 6/5", Equity: 0.100, PlayerWinChance: 54.0},
			},
		},
		DoublingCubeAnalysis: &DoublingCubeAnalysis{
			BestCubeAction:   "No double",
			PlayerWinChances: 55.0,
		},
		CreationDate:     time.Now(),
		LastModifiedDate: time.Now(),
	}
	if err := db.SaveAnalysis(id1, analysis1); err != nil {
		t.Fatalf("SaveAnalysis 1 failed: %v", err)
	}

	analysis2 := PositionAnalysis{
		PositionID:   int(id2),
		XGID:         "test-xgid-2",
		AnalysisType: "XG Roller++",
		PlayedMove:   "24/13",
		PlayedMoves:  []string{"24/13"},
		CheckerAnalysis: &CheckerAnalysis{
			Moves: []CheckerMove{
				{Index: 1, Move: "24/13", Equity: 0.05, PlayerWinChance: 52.0},
			},
		},
		CreationDate:     time.Now(),
		LastModifiedDate: time.Now(),
	}
	if err := db.SaveAnalysis(id2, analysis2); err != nil {
		t.Fatalf("SaveAnalysis 2 failed: %v", err)
	}

	// --- Comments ---
	if err := db.SaveComment(id1, "This is comment 1"); err != nil {
		t.Fatalf("SaveComment 1 failed: %v", err)
	}
	if err := db.SaveComment(id2, "This is comment 2"); err != nil {
		t.Fatalf("SaveComment 2 failed: %v", err)
	}

	// --- Filter library (direct SQL, SaveFilter checks version) ---
	rawDB := db.db
	_, err = rawDB.Exec(`INSERT INTO filter_library (name, command) VALUES (?, ?)`, "blunders", "eq > 0.1")
	if err != nil {
		t.Fatalf("Insert filter 1 failed: %v", err)
	}
	_, err = rawDB.Exec(`INSERT INTO filter_library (name, command) VALUES (?, ?)`, "doubles", "cube")
	if err != nil {
		t.Fatalf("Insert filter 2 failed: %v", err)
	}

	// --- Match, game, moves, move_analysis ---
	matchRes, err := rawDB.Exec(`INSERT INTO match (player1_name, player2_name, event, location, round, match_length, match_date, import_date, file_path, game_count, match_hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"Alice", "Bob", "Test Event", "Paris", "1", 7,
		time.Now(), time.Now(), "/test/match.xg", 1, "hash123")
	if err != nil {
		t.Fatalf("Insert match failed: %v", err)
	}
	matchID, _ := matchRes.LastInsertId()

	gameRes, err := rawDB.Exec(`INSERT INTO game (match_id, game_number, initial_score_1, initial_score_2, winner, points_won, move_count)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, matchID, 1, 0, 0, 1, 1, 2)
	if err != nil {
		t.Fatalf("Insert game failed: %v", err)
	}
	gameID, _ := gameRes.LastInsertId()

	move1Res, err := rawDB.Exec(`INSERT INTO move (game_id, move_number, move_type, position_id, player, dice_1, dice_2, checker_move, cube_action)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, gameID, 1, "checker", id1, 0, 3, 1, "8/5 6/5", "")
	if err != nil {
		t.Fatalf("Insert move 1 failed: %v", err)
	}
	moveID1, _ := move1Res.LastInsertId()

	move2Res, err := rawDB.Exec(`INSERT INTO move (game_id, move_number, move_type, position_id, player, dice_1, dice_2, checker_move, cube_action)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, gameID, 2, "checker", id2, 1, 6, 5, "24/13", "")
	if err != nil {
		t.Fatalf("Insert move 2 failed: %v", err)
	}
	moveID2, _ := move2Res.LastInsertId()

	// Move analysis for each move
	_, err = rawDB.Exec(`INSERT INTO move_analysis (move_id, analysis_type, depth, equity, equity_error, win_rate, gammon_rate, backgammon_rate, opponent_win_rate, opponent_gammon_rate, opponent_backgammon_rate)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, moveID1, "checker", "3-ply", 0.123, 0.0, 55.0, 10.0, 1.0, 44.0, 8.0, 0.5)
	if err != nil {
		t.Fatalf("Insert move_analysis 1 failed: %v", err)
	}
	_, err = rawDB.Exec(`INSERT INTO move_analysis (move_id, analysis_type, depth, equity, equity_error, win_rate, gammon_rate, backgammon_rate, opponent_win_rate, opponent_gammon_rate, opponent_backgammon_rate)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, moveID2, "checker", "3-ply", 0.05, 0.0, 52.0, 9.0, 0.5, 47.0, 7.0, 0.3)
	if err != nil {
		t.Fatalf("Insert move_analysis 2 failed: %v", err)
	}

	// --- Collection ---
	collRes, err := rawDB.Exec(`INSERT INTO collection (name, description, sort_order) VALUES (?, ?, ?)`,
		"Test Collection", "A test collection", 0)
	if err != nil {
		t.Fatalf("Insert collection failed: %v", err)
	}
	collID, _ := collRes.LastInsertId()

	_, err = rawDB.Exec(`INSERT INTO collection_position (collection_id, position_id, sort_order) VALUES (?, ?, ?)`, collID, id1, 0)
	if err != nil {
		t.Fatalf("Insert collection_position 1 failed: %v", err)
	}
	_, err = rawDB.Exec(`INSERT INTO collection_position (collection_id, position_id, sort_order) VALUES (?, ?, ?)`, collID, id2, 1)
	if err != nil {
		t.Fatalf("Insert collection_position 2 failed: %v", err)
	}

	// --- Tournament ---
	tournRes, err := rawDB.Exec(`INSERT INTO tournament (name, date, location, sort_order) VALUES (?, ?, ?, ?)`,
		"Test Tournament", "2025-01-01", "Paris", 0)
	if err != nil {
		t.Fatalf("Insert tournament failed: %v", err)
	}
	tournID, _ := tournRes.LastInsertId()

	// Link match to tournament
	_, err = rawDB.Exec(`UPDATE match SET tournament_id = ? WHERE id = ?`, tournID, matchID)
	if err != nil {
		t.Fatalf("Link match to tournament failed: %v", err)
	}

	cleanup := func() {
		db.db.Close()
	}

	return db, dir, cleanup
}

// openExportDB opens the export database for inspection.
func openExportDB(t *testing.T, path string) *sql.DB {
	t.Helper()
	edb, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("failed to open export db: %v", err)
	}
	return edb
}

// countRows returns the number of rows in a table.
func countRows(t *testing.T, db *sql.DB, table string) int {
	t.Helper()
	var count int
	err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
	if err != nil {
		t.Fatalf("countRows(%s) failed: %v", table, err)
	}
	return count
}

// loadExportAnalysis returns all PositionAnalysis from the export db.
func loadExportAnalysis(t *testing.T, db *sql.DB) []PositionAnalysis {
	t.Helper()
	rows, err := db.Query("SELECT data FROM analysis")
	if err != nil {
		t.Fatalf("query analysis: %v", err)
	}
	defer rows.Close()
	var analyses []PositionAnalysis
	for rows.Next() {
		var dataJSON string
		if err := rows.Scan(&dataJSON); err != nil {
			t.Fatalf("scan analysis: %v", err)
		}
		var a PositionAnalysis
		if err := json.Unmarshal([]byte(dataJSON), &a); err != nil {
			t.Fatalf("unmarshal analysis: %v", err)
		}
		analyses = append(analyses, a)
	}
	return analyses
}

// allPositions loads all positions from source db for export.
func allPositions(t *testing.T, db *Database) []Position {
	t.Helper()
	positions, err := db.LoadAllPositions()
	if err != nil {
		t.Fatalf("LoadAllPositions: %v", err)
	}
	return positions
}

// exportParams holds all export options in a readable struct for test clarity.
type exportParams struct {
	includeAnalysis      bool
	includeComments      bool
	includeFilterLibrary bool
	includePlayedMoves   bool
	includeMatches       bool
	includeCollections   bool
	collectionIDs        []int64
	matchIDs             []int64
	tournamentIDs        []int64
}

// defaultExportParams returns all-true export params (export everything).
func defaultExportParams() exportParams {
	return exportParams{
		includeAnalysis:      true,
		includeComments:      true,
		includeFilterLibrary: true,
		includePlayedMoves:   true,
		includeMatches:       true,
		includeCollections:   false,
		collectionIDs:        nil,
		matchIDs:             nil,
		tournamentIDs:        nil,
	}
}

func doExport(t *testing.T, db *Database, exportPath string, p exportParams) {
	t.Helper()
	positions := allPositions(t, db)
	metadata := map[string]string{
		"user":        "testuser",
		"description": "test export",
	}
	err := db.ExportDatabase(ExportOptions{
		ExportPath:           exportPath,
		Positions:            positions,
		Metadata:             metadata,
		IncludeAnalysis:      p.includeAnalysis,
		IncludeComments:      p.includeComments,
		IncludeFilterLibrary: p.includeFilterLibrary,
		IncludePlayedMoves:   p.includePlayedMoves,
		IncludeMatches:       p.includeMatches,
		IncludeCollections:   p.includeCollections,
		CollectionIDs:        p.collectionIDs,
		MatchIDs:             p.matchIDs,
		TournamentIDs:        p.tournamentIDs,
	})
	if err != nil {
		t.Fatalf("ExportDatabase failed: %v", err)
	}
}

// --- Test: Export everything (baseline) ---

func TestExport_AllOptions_Baseline(t *testing.T) {
	db, dir, cleanup := setupExportTestDB(t)
	defer cleanup()

	exportPath := filepath.Join(dir, "export.db")
	p := defaultExportParams()
	doExport(t, db, exportPath, p)

	edb := openExportDB(t, exportPath)
	defer edb.Close()

	// Positions
	if n := countRows(t, edb, "position"); n != 2 {
		t.Errorf("expected 2 positions, got %d", n)
	}

	// Analysis
	if n := countRows(t, edb, "analysis"); n != 2 {
		t.Errorf("expected 2 analysis rows, got %d", n)
	}

	// Verify played moves are present in analysis
	analyses := loadExportAnalysis(t, edb)
	playedMovesFound := false
	for _, a := range analyses {
		if len(a.PlayedMoves) > 0 || a.PlayedMove != "" {
			playedMovesFound = true
		}
	}
	if !playedMovesFound {
		t.Error("expected played moves in analysis but found none")
	}

	// Comments
	if n := countRows(t, edb, "comment"); n != 2 {
		t.Errorf("expected 2 comments, got %d", n)
	}

	// Filter library
	if n := countRows(t, edb, "filter_library"); n != 2 {
		t.Errorf("expected 2 filter_library entries, got %d", n)
	}

	// Matches
	if n := countRows(t, edb, "match"); n != 1 {
		t.Errorf("expected 1 match, got %d", n)
	}
	if n := countRows(t, edb, "game"); n != 1 {
		t.Errorf("expected 1 game, got %d", n)
	}
	if n := countRows(t, edb, "move"); n != 2 {
		t.Errorf("expected 2 moves, got %d", n)
	}
	if n := countRows(t, edb, "move_analysis"); n != 2 {
		t.Errorf("expected 2 move_analysis rows, got %d", n)
	}

	// Metadata
	var user string
	err := edb.QueryRow(`SELECT value FROM metadata WHERE key = 'user'`).Scan(&user)
	if err != nil || user != "testuser" {
		t.Errorf("expected metadata user='testuser', got '%s' (err: %v)", user, err)
	}
}

// --- Test: includeAnalysis=false ---

func TestExport_NoAnalysis(t *testing.T) {
	db, dir, cleanup := setupExportTestDB(t)
	defer cleanup()

	exportPath := filepath.Join(dir, "export.db")
	p := defaultExportParams()
	p.includeAnalysis = false
	doExport(t, db, exportPath, p)

	edb := openExportDB(t, exportPath)
	defer edb.Close()

	// Should have positions but NO analysis
	if n := countRows(t, edb, "position"); n != 2 {
		t.Errorf("expected 2 positions, got %d", n)
	}
	if n := countRows(t, edb, "analysis"); n != 0 {
		t.Errorf("expected 0 analysis rows when includeAnalysis=false, got %d", n)
	}

	// Comments should still be present
	if n := countRows(t, edb, "comment"); n != 2 {
		t.Errorf("expected 2 comments, got %d", n)
	}
}

// --- Test: includeComments=false ---

func TestExport_NoComments(t *testing.T) {
	db, dir, cleanup := setupExportTestDB(t)
	defer cleanup()

	exportPath := filepath.Join(dir, "export.db")
	p := defaultExportParams()
	p.includeComments = false
	doExport(t, db, exportPath, p)

	edb := openExportDB(t, exportPath)
	defer edb.Close()

	if n := countRows(t, edb, "comment"); n != 0 {
		t.Errorf("expected 0 comments when includeComments=false, got %d", n)
	}

	// Analysis should still be present
	if n := countRows(t, edb, "analysis"); n != 2 {
		t.Errorf("expected 2 analysis rows, got %d", n)
	}
}

// --- Test: includeFilterLibrary=false ---

func TestExport_NoFilterLibrary(t *testing.T) {
	db, dir, cleanup := setupExportTestDB(t)
	defer cleanup()

	exportPath := filepath.Join(dir, "export.db")
	p := defaultExportParams()
	p.includeFilterLibrary = false
	doExport(t, db, exportPath, p)

	edb := openExportDB(t, exportPath)
	defer edb.Close()

	if n := countRows(t, edb, "filter_library"); n != 0 {
		t.Errorf("expected 0 filter_library entries when includeFilterLibrary=false, got %d", n)
	}
}

// --- Test: includePlayedMoves=false (CRITICAL: reported bug) ---

func TestExport_NoPlayedMoves(t *testing.T) {
	db, dir, cleanup := setupExportTestDB(t)
	defer cleanup()

	exportPath := filepath.Join(dir, "export.db")
	p := defaultExportParams()
	p.includePlayedMoves = false
	doExport(t, db, exportPath, p)

	edb := openExportDB(t, exportPath)
	defer edb.Close()

	// Analysis should still be present
	if n := countRows(t, edb, "analysis"); n != 2 {
		t.Errorf("expected 2 analysis rows, got %d", n)
	}

	// But NO played move data in any analysis
	analyses := loadExportAnalysis(t, edb)
	for i, a := range analyses {
		if a.PlayedMove != "" {
			t.Errorf("analysis[%d]: expected empty PlayedMove, got %q", i, a.PlayedMove)
		}
		if len(a.PlayedMoves) > 0 {
			t.Errorf("analysis[%d]: expected empty PlayedMoves, got %v", i, a.PlayedMoves)
		}
		if a.PlayedCubeAction != "" {
			t.Errorf("analysis[%d]: expected empty PlayedCubeAction, got %q", i, a.PlayedCubeAction)
		}
		if len(a.PlayedCubeActions) > 0 {
			t.Errorf("analysis[%d]: expected empty PlayedCubeActions, got %v", i, a.PlayedCubeActions)
		}
	}

	// Checker analysis data (non-played-move) should still be there
	for i, a := range analyses {
		if a.CheckerAnalysis == nil {
			t.Errorf("analysis[%d]: checker analysis should still be present", i)
		}
	}
}

// --- Test: includeMatches=false ---

func TestExport_NoMatches(t *testing.T) {
	db, dir, cleanup := setupExportTestDB(t)
	defer cleanup()

	exportPath := filepath.Join(dir, "export.db")
	p := defaultExportParams()
	p.includeMatches = false
	doExport(t, db, exportPath, p)

	edb := openExportDB(t, exportPath)
	defer edb.Close()

	if n := countRows(t, edb, "match"); n != 0 {
		t.Errorf("expected 0 matches when includeMatches=false, got %d", n)
	}
	if n := countRows(t, edb, "game"); n != 0 {
		t.Errorf("expected 0 games when includeMatches=false, got %d", n)
	}
	if n := countRows(t, edb, "move"); n != 0 {
		t.Errorf("expected 0 moves when includeMatches=false, got %d", n)
	}
	if n := countRows(t, edb, "move_analysis"); n != 0 {
		t.Errorf("expected 0 move_analysis when includeMatches=false, got %d", n)
	}

	// Positions and analysis should still be present
	if n := countRows(t, edb, "position"); n != 2 {
		t.Errorf("expected 2 positions, got %d", n)
	}
	if n := countRows(t, edb, "analysis"); n != 2 {
		t.Errorf("expected 2 analysis rows, got %d", n)
	}
}

// --- Test: specific match IDs ---

func TestExport_SpecificMatchIDs(t *testing.T) {
	db, dir, cleanup := setupExportTestDB(t)
	defer cleanup()

	// Get match IDs from source
	var matchID int64
	err := db.db.QueryRow("SELECT id FROM match LIMIT 1").Scan(&matchID)
	if err != nil {
		t.Fatalf("failed to get match ID: %v", err)
	}

	exportPath := filepath.Join(dir, "export.db")
	p := defaultExportParams()
	p.matchIDs = []int64{matchID}
	doExport(t, db, exportPath, p)

	edb := openExportDB(t, exportPath)
	defer edb.Close()

	if n := countRows(t, edb, "match"); n != 1 {
		t.Errorf("expected 1 match, got %d", n)
	}

	// Use a non-existent match ID
	exportPath2 := filepath.Join(dir, "export2.db")
	p2 := defaultExportParams()
	p2.matchIDs = []int64{99999}
	doExport(t, db, exportPath2, p2)

	edb2 := openExportDB(t, exportPath2)
	defer edb2.Close()

	if n := countRows(t, edb2, "match"); n != 0 {
		t.Errorf("expected 0 matches for non-existent ID, got %d", n)
	}
}

// --- Test: includeCollections=true with collection IDs ---

func TestExport_WithCollections(t *testing.T) {
	db, dir, cleanup := setupExportTestDB(t)
	defer cleanup()

	// Get collection ID
	var collID int64
	err := db.db.QueryRow("SELECT id FROM collection LIMIT 1").Scan(&collID)
	if err != nil {
		t.Fatalf("failed to get collection ID: %v", err)
	}

	exportPath := filepath.Join(dir, "export.db")
	p := defaultExportParams()
	p.includeCollections = true
	p.collectionIDs = []int64{collID}
	doExport(t, db, exportPath, p)

	edb := openExportDB(t, exportPath)
	defer edb.Close()

	if n := countRows(t, edb, "collection"); n != 1 {
		t.Errorf("expected 1 collection, got %d", n)
	}
	if n := countRows(t, edb, "collection_position"); n != 2 {
		t.Errorf("expected 2 collection_position entries, got %d", n)
	}
}

// --- Test: includeCollections=false ---

func TestExport_NoCollections(t *testing.T) {
	db, dir, cleanup := setupExportTestDB(t)
	defer cleanup()

	exportPath := filepath.Join(dir, "export.db")
	p := defaultExportParams()
	p.includeCollections = false
	p.collectionIDs = nil
	doExport(t, db, exportPath, p)

	edb := openExportDB(t, exportPath)
	defer edb.Close()

	if n := countRows(t, edb, "collection"); n != 0 {
		t.Errorf("expected 0 collections when includeCollections=false, got %d", n)
	}
	if n := countRows(t, edb, "collection_position"); n != 0 {
		t.Errorf("expected 0 collection_position entries, got %d", n)
	}
}

// --- Test: tournaments ---

func TestExport_WithTournaments(t *testing.T) {
	db, dir, cleanup := setupExportTestDB(t)
	defer cleanup()

	// Get tournament and match IDs
	var tournID int64
	err := db.db.QueryRow("SELECT id FROM tournament LIMIT 1").Scan(&tournID)
	if err != nil {
		t.Fatalf("failed to get tournament ID: %v", err)
	}

	exportPath := filepath.Join(dir, "export.db")
	p := defaultExportParams()
	p.tournamentIDs = []int64{tournID}
	doExport(t, db, exportPath, p)

	edb := openExportDB(t, exportPath)
	defer edb.Close()

	if n := countRows(t, edb, "tournament"); n != 1 {
		t.Errorf("expected 1 tournament, got %d", n)
	}

	// The match should have tournament_id set
	var exportedTournID sql.NullInt64
	err = edb.QueryRow("SELECT tournament_id FROM match LIMIT 1").Scan(&exportedTournID)
	if err != nil {
		t.Fatalf("failed to query match tournament_id: %v", err)
	}
	if !exportedTournID.Valid || exportedTournID.Int64 == 0 {
		t.Error("expected match to have a tournament_id in export")
	}
}

func TestExport_NoTournaments(t *testing.T) {
	db, dir, cleanup := setupExportTestDB(t)
	defer cleanup()

	exportPath := filepath.Join(dir, "export.db")
	p := defaultExportParams()
	p.tournamentIDs = nil
	doExport(t, db, exportPath, p)

	edb := openExportDB(t, exportPath)
	defer edb.Close()

	if n := countRows(t, edb, "tournament"); n != 0 {
		t.Errorf("expected 0 tournaments when no IDs provided, got %d", n)
	}
}

// TestExport_UnknownCollectionAndTournamentAreLoggedAndCounted covers the
// "silently ignored scan" fix: a stale collection or tournament id (deleted
// between the caller listing it and the export running, say) used to just
// vanish from the export with no trace anywhere. ExportDatabase now logs a
// slog.Warn for each skipped row and a final aggregate "skipped" count — not
// threaded through the return value (this function already has 40+
// early-return error paths and three external CLI callers; see the comment
// on the `skipped` local in db_export.go), but a real, discoverable signal
// instead of nothing. The export itself must still succeed: an unknown id is
// exactly the kind of single-row problem that must not fail the whole run.
func TestExport_UnknownCollectionAndTournamentAreLoggedAndCounted(t *testing.T) {
	db, dir, cleanup := setupExportTestDB(t)
	defer cleanup()

	logs := captureSlog(t)

	exportPath := filepath.Join(dir, "export.db")
	positions := allPositions(t, db)
	err := db.ExportDatabase(ExportOptions{
		ExportPath:         exportPath,
		Positions:          positions,
		IncludeCollections: true,
		CollectionIDs:      []int64{999999},
		TournamentIDs:      []int64{999999},
	})
	if err != nil {
		t.Fatalf("ExportDatabase with an unknown collection/tournament id should still succeed: %v", err)
	}

	out := logs.String()
	if !strings.Contains(out, "reading collection") {
		t.Errorf("log missing the skipped-collection warning:\n%s", out)
	}
	if !strings.Contains(out, "reading tournament") {
		t.Errorf("log missing the skipped-tournament warning:\n%s", out)
	}
	if !strings.Contains(out, "export completed with some rows skipped") {
		t.Errorf("log missing the aggregate skipped-count summary:\n%s", out)
	}
	if !strings.Contains(out, "skipped=2") {
		t.Errorf("aggregate summary does not report skipped=2:\n%s", out)
	}
}

// --- Test: everything disabled except positions ---

func TestExport_PositionsOnly(t *testing.T) {
	db, dir, cleanup := setupExportTestDB(t)
	defer cleanup()

	exportPath := filepath.Join(dir, "export.db")
	p := exportParams{
		includeAnalysis:      false,
		includeComments:      false,
		includeFilterLibrary: false,
		includePlayedMoves:   false,
		includeMatches:       false,
		includeCollections:   false,
		collectionIDs:        nil,
		matchIDs:             nil,
		tournamentIDs:        nil,
	}
	doExport(t, db, exportPath, p)

	edb := openExportDB(t, exportPath)
	defer edb.Close()

	// Only positions
	if n := countRows(t, edb, "position"); n != 2 {
		t.Errorf("expected 2 positions, got %d", n)
	}
	if n := countRows(t, edb, "analysis"); n != 0 {
		t.Errorf("expected 0 analysis, got %d", n)
	}
	if n := countRows(t, edb, "comment"); n != 0 {
		t.Errorf("expected 0 comments, got %d", n)
	}
	if n := countRows(t, edb, "filter_library"); n != 0 {
		t.Errorf("expected 0 filters, got %d", n)
	}
	if n := countRows(t, edb, "match"); n != 0 {
		t.Errorf("expected 0 matches, got %d", n)
	}
	if n := countRows(t, edb, "collection"); n != 0 {
		t.Errorf("expected 0 collections, got %d", n)
	}
	if n := countRows(t, edb, "tournament"); n != 0 {
		t.Errorf("expected 0 tournaments, got %d", n)
	}
}

// --- Test: PlayedMoves=true properly merges from move table ---

func TestExport_PlayedMovesMergeFromMoveTable(t *testing.T) {
	db, dir, cleanup := setupExportTestDB(t)
	defer cleanup()

	exportPath := filepath.Join(dir, "export.db")
	p := defaultExportParams()
	p.includePlayedMoves = true
	doExport(t, db, exportPath, p)

	edb := openExportDB(t, exportPath)
	defer edb.Close()

	analyses := loadExportAnalysis(t, edb)
	if len(analyses) != 2 {
		t.Fatalf("expected 2 analyses, got %d", len(analyses))
	}

	// Each analysis should have played move data
	for i, a := range analyses {
		if len(a.PlayedMoves) == 0 {
			t.Errorf("analysis[%d]: expected non-empty PlayedMoves when includePlayedMoves=true", i)
		}
	}
}

// --- Test: analysis without analysis means no played moves either ---

func TestExport_NoAnalysis_ImpliesNoPlayedMoves(t *testing.T) {
	db, dir, cleanup := setupExportTestDB(t)
	defer cleanup()

	exportPath := filepath.Join(dir, "export.db")
	p := defaultExportParams()
	p.includeAnalysis = false
	p.includePlayedMoves = true // should be irrelevant when analysis is off
	doExport(t, db, exportPath, p)

	edb := openExportDB(t, exportPath)
	defer edb.Close()

	if n := countRows(t, edb, "analysis"); n != 0 {
		t.Errorf("expected 0 analysis when includeAnalysis=false (even with includePlayedMoves=true), got %d", n)
	}
}

// --- Test: position ID remapping ---

func TestExport_PositionIDRemapping(t *testing.T) {
	db, dir, cleanup := setupExportTestDB(t)
	defer cleanup()

	exportPath := filepath.Join(dir, "export.db")
	p := defaultExportParams()
	doExport(t, db, exportPath, p)

	edb := openExportDB(t, exportPath)
	defer edb.Close()

	// Verify that position IDs in the export are sequential starting from 1
	rows, err := edb.Query("SELECT id FROM position ORDER BY id")
	if err != nil {
		t.Fatalf("query position IDs: %v", err)
	}
	defer rows.Close()

	expected := int64(1)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan ID: %v", err)
		}
		if id != expected {
			t.Errorf("expected position ID %d, got %d", expected, id)
		}
		expected++
	}

	// Verify analysis position_id matches
	analysisRows, err := edb.Query("SELECT position_id FROM analysis ORDER BY position_id")
	if err != nil {
		t.Fatalf("query analysis position_ids: %v", err)
	}
	defer analysisRows.Close()

	for analysisRows.Next() {
		var posID int64
		if err := analysisRows.Scan(&posID); err != nil {
			t.Fatalf("scan analysis posID: %v", err)
		}
		// Verify it references a valid position
		var exists int
		err := edb.QueryRow("SELECT COUNT(*) FROM position WHERE id = ?", posID).Scan(&exists)
		if err != nil || exists != 1 {
			t.Errorf("analysis references non-existent position_id %d", posID)
		}
	}

	// Verify analysis JSON positionId matches the SQL position_id
	analyses := loadExportAnalysis(t, edb)
	aPosIDRows, _ := edb.Query("SELECT position_id FROM analysis ORDER BY position_id")
	defer aPosIDRows.Close()
	for i := 0; aPosIDRows.Next(); i++ {
		var sqlPosID int64
		aPosIDRows.Scan(&sqlPosID)
		if int64(analyses[i].PositionID) != sqlPosID {
			t.Errorf("analysis[%d]: JSON positionId=%d != SQL position_id=%d",
				i, analyses[i].PositionID, sqlPosID)
		}
	}
}

// --- Test: move position_id remapping in match export ---

func TestExport_MovePositionIDRemapping(t *testing.T) {
	db, dir, cleanup := setupExportTestDB(t)
	defer cleanup()

	exportPath := filepath.Join(dir, "export.db")
	p := defaultExportParams()
	doExport(t, db, exportPath, p)

	edb := openExportDB(t, exportPath)
	defer edb.Close()

	// All moves should reference valid position IDs in the export
	moveRows, err := edb.Query("SELECT position_id FROM move WHERE position_id IS NOT NULL")
	if err != nil {
		t.Fatalf("query move position_ids: %v", err)
	}
	defer moveRows.Close()

	count := 0
	for moveRows.Next() {
		var posID int64
		if err := moveRows.Scan(&posID); err != nil {
			t.Fatalf("scan move posID: %v", err)
		}
		var exists int
		err := edb.QueryRow("SELECT COUNT(*) FROM position WHERE id = ?", posID).Scan(&exists)
		if err != nil || exists != 1 {
			t.Errorf("move references non-existent position_id %d in export", posID)
		}
		count++
	}
	if count == 0 {
		t.Error("expected at least one move with position_id, got none")
	}
}

// --- Test: metadata ---

func TestExport_Metadata(t *testing.T) {
	db, dir, cleanup := setupExportTestDB(t)
	defer cleanup()

	exportPath := filepath.Join(dir, "export.db")
	positions := allPositions(t, db)
	metadata := map[string]string{
		"user":           "Jean",
		"description":    "My blunder collection",
		"dateOfCreation": "2025-06-15",
	}
	err := db.ExportDatabase(ExportOptions{
		ExportPath:           exportPath,
		Positions:            positions,
		Metadata:             metadata,
		IncludeAnalysis:      true,
		IncludeComments:      true,
		IncludeFilterLibrary: true,
		IncludePlayedMoves:   true,
		IncludeMatches:       true,
	})
	if err != nil {
		t.Fatalf("ExportDatabase failed: %v", err)
	}

	edb := openExportDB(t, exportPath)
	defer edb.Close()

	// Check each metadata value
	for key, expectedValue := range metadata {
		var value string
		err := edb.QueryRow("SELECT value FROM metadata WHERE key = ?", key).Scan(&value)
		if err != nil {
			t.Errorf("metadata key %q not found: %v", key, err)
		} else if value != expectedValue {
			t.Errorf("metadata %q: expected %q, got %q", key, expectedValue, value)
		}
	}

	// Check database_version is set
	var version string
	err = edb.QueryRow("SELECT value FROM metadata WHERE key = 'database_version'").Scan(&version)
	if err != nil {
		t.Errorf("database_version not set: %v", err)
	}
	if version != DatabaseVersion {
		t.Errorf("expected database_version=%s, got %s", DatabaseVersion, version)
	}
}

// --- Test: default dateOfCreation ---

func TestExport_DefaultDateOfCreation(t *testing.T) {
	db, dir, cleanup := setupExportTestDB(t)
	defer cleanup()

	exportPath := filepath.Join(dir, "export.db")
	positions := allPositions(t, db)
	metadata := map[string]string{} // empty - no dateOfCreation
	err := db.ExportDatabase(ExportOptions{
		ExportPath:           exportPath,
		Positions:            positions,
		Metadata:             metadata,
		IncludeAnalysis:      true,
		IncludeComments:      true,
		IncludeFilterLibrary: true,
		IncludePlayedMoves:   true,
		IncludeMatches:       true,
	})
	if err != nil {
		t.Fatalf("ExportDatabase failed: %v", err)
	}

	edb := openExportDB(t, exportPath)
	defer edb.Close()

	var dateOfCreation string
	err = edb.QueryRow("SELECT value FROM metadata WHERE key = 'dateOfCreation'").Scan(&dateOfCreation)
	if err != nil {
		t.Errorf("dateOfCreation should be auto-set: %v", err)
	}
	if dateOfCreation == "" {
		t.Error("dateOfCreation should not be empty when not provided")
	}
}

// --- Test: export overwrites existing file ---

func TestExport_OverwritesExistingFile(t *testing.T) {
	db, dir, cleanup := setupExportTestDB(t)
	defer cleanup()

	exportPath := filepath.Join(dir, "export.db")

	// First export
	p := defaultExportParams()
	doExport(t, db, exportPath, p)

	// Second export (should overwrite)
	doExport(t, db, exportPath, p)

	edb := openExportDB(t, exportPath)
	defer edb.Close()

	// Should have exactly 2 positions (not 4)
	if n := countRows(t, edb, "position"); n != 2 {
		t.Errorf("expected 2 positions after overwrite, got %d", n)
	}
}

// --- Test: export with no database open ---

func TestExport_NoDatabaseOpen(t *testing.T) {
	db := NewDatabase()
	dir := t.TempDir()
	exportPath := filepath.Join(dir, "export.db")

	err := db.ExportDatabase(ExportOptions{
		ExportPath:           exportPath,
		IncludeAnalysis:      true,
		IncludeComments:      true,
		IncludeFilterLibrary: true,
		IncludePlayedMoves:   true,
		IncludeMatches:       true,
	})
	if err == nil {
		t.Error("expected error when no database is open")
	}
}

// --- Test: checker_move in move table when includePlayedMoves=false ---
// The move table (game record) should still contain checker_move because it's
// part of the game transcription, not the "played move" annotation on analysis.

func TestExport_MoveTableCheckerMovePresent_EvenWithoutPlayedMoves(t *testing.T) {
	db, dir, cleanup := setupExportTestDB(t)
	defer cleanup()

	exportPath := filepath.Join(dir, "export.db")
	p := defaultExportParams()
	p.includePlayedMoves = false
	doExport(t, db, exportPath, p)

	edb := openExportDB(t, exportPath)
	defer edb.Close()

	// move rows should still have checker_move since they're part of game record
	var checkerMove string
	err := edb.QueryRow("SELECT checker_move FROM move LIMIT 1").Scan(&checkerMove)
	if err != nil {
		t.Fatalf("query checker_move: %v", err)
	}
	if checkerMove == "" {
		t.Error("move table should still have checker_move (game record), even with includePlayedMoves=false")
	}
}

// --- Test: DoublingCubeAnalysis preserved when includePlayedMoves=false ---

func TestExport_CubeAnalysisPreserved_NoPlayedMoves(t *testing.T) {
	db, dir, cleanup := setupExportTestDB(t)
	defer cleanup()

	exportPath := filepath.Join(dir, "export.db")
	p := defaultExportParams()
	p.includePlayedMoves = false
	doExport(t, db, exportPath, p)

	edb := openExportDB(t, exportPath)
	defer edb.Close()

	analyses := loadExportAnalysis(t, edb)
	cubeAnalysisFound := false
	for _, a := range analyses {
		if a.DoublingCubeAnalysis != nil {
			cubeAnalysisFound = true
			if a.DoublingCubeAnalysis.BestCubeAction == "" {
				t.Error("DoublingCubeAnalysis.BestCubeAction should be preserved")
			}
		}
	}
	if !cubeAnalysisFound {
		t.Error("expected at least one analysis with DoublingCubeAnalysis")
	}
}

// --- fiche-04: the exported file must be a database on the CURRENT schema ---
//
// Before the fix, ExportDatabase hand-rolled a two-column "position(id, state,
// individually_imported)" table and stamped database_version to the CURRENT
// version. Reopening that file worked (ensureAllTablesExist backfills the
// missing columns on open) but left zobrist_hash and every scalar column
// (dice_1, score_1, pip_diff, …) NULL forever, because nothing had ever
// written them: the migration chain never runs its ALTER TABLE steps for a
// database that already claims to be current. Dedup and every SQL-column
// search were silently dead on a reopened export.

func TestExportDatabase_RoundTrip_ScalarColumnsAndDedup(t *testing.T) {
	db, dir, cleanup := setupExportTestDB(t)
	defer cleanup()

	exportPath := filepath.Join(dir, "roundtrip.db")
	p := defaultExportParams()
	doExport(t, db, exportPath, p)

	reopened := NewDatabase()
	if err := reopened.OpenDatabase(exportPath); err != nil {
		t.Fatalf("OpenDatabase(export): %v", err)
	}
	defer reopened.Close()

	positions, err := reopened.LoadAllPositions()
	if err != nil {
		t.Fatalf("LoadAllPositions: %v", err)
	}
	if len(positions) != 2 {
		t.Fatalf("expected 2 positions, got %d", len(positions))
	}

	var nullHashes int
	if err := reopened.db.QueryRow(`SELECT COUNT(*) FROM position WHERE zobrist_hash IS NULL`).Scan(&nullHashes); err != nil {
		t.Fatalf("query zobrist_hash: %v", err)
	}
	if nullHashes != 0 {
		t.Errorf("expected 0 NULL zobrist_hash after export+reopen, got %d", nullHashes)
	}

	var nullScalars int
	if err := reopened.db.QueryRow(`SELECT COUNT(*) FROM position WHERE dice_1 IS NULL OR score_1 IS NULL OR score_2 IS NULL OR pip_diff IS NULL`).Scan(&nullScalars); err != nil {
		t.Fatalf("query scalar columns: %v", err)
	}
	if nullScalars != 0 {
		t.Errorf("expected 0 rows with a NULL scalar column, got %d", nullScalars)
	}

	// A pure-SQL dice search (what search_sqlite.go's DiceRollFilter runs) must find
	// the position saved with Dice=[6,5] in setupExportTestDB.
	var found int
	if err := reopened.db.QueryRow(`SELECT COUNT(*) FROM position WHERE dice_1 = 6 AND dice_2 = 5`).Scan(&found); err != nil {
		t.Fatalf("query dice search: %v", err)
	}
	if found != 1 {
		t.Errorf("expected 1 position with dice 6-5 findable by SQL, got %d", found)
	}

	// Search index presence: the unique zobrist index is what makes dedup possible
	// at all (fiche-04's own criterion — "index créés").
	var indexCount int
	if err := reopened.db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_position_zobrist'`).Scan(&indexCount); err != nil {
		t.Fatalf("query sqlite_master: %v", err)
	}
	if indexCount != 1 {
		t.Error("expected the exported file to carry idx_position_zobrist")
	}

	// Re-importing the export into a database that already holds these positions
	// must create no duplicates — the actual scenario a recipient hits when they
	// already have some of a teacher's course and receive an updated export.
	before, err := db.LoadAllPositions()
	if err != nil {
		t.Fatalf("LoadAllPositions on the source db: %v", err)
	}
	beforeCount := len(before)
	result, err := db.CommitImportDatabase(exportPath)
	if err != nil {
		t.Fatalf("CommitImportDatabase(export) into the source db: %v", err)
	}
	if added, _ := result["added"].(int); added != 0 {
		t.Errorf("re-importing the export into its own source added %d new positions, want 0", added)
	}
	after, err := db.LoadAllPositions()
	if err != nil {
		t.Fatalf("LoadAllPositions after re-import: %v", err)
	}
	if len(after) != beforeCount {
		t.Errorf("re-importing the export duplicated positions: before=%d after=%d", beforeCount, len(after))
	}
}
