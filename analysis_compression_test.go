package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// TestCompressDecompressRoundTrip verifies that analysis data survives
// compression/decompression unchanged.
func TestCompressDecompressRoundTrip(t *testing.T) {
	analysis := PositionAnalysis{
		PositionID:            42,
		XGID:                  "XGID=test",
		Player1:               "Alice",
		Player2:               "Bob",
		AnalysisType:          "CheckerMove",
		AnalysisEngineVersion: "XG2+",
		CheckerAnalysis: &CheckerAnalysis{
			Moves: []CheckerMove{
				{
					Index: 0, AnalysisDepth: "3-ply", Move: "13/7 8/4",
					Equity: 0.123, PlayerWinChance: 55.12,
					PlayerGammonChance: 12.34, PlayerBackgammonChance: 0.56,
					OpponentWinChance: 44.88, OpponentGammonChance: 11.23,
					OpponentBackgammonChance: 0.34,
				},
				{
					Index: 1, AnalysisDepth: "3-ply", Move: "24/18 13/9",
					Equity: 0.089, PlayerWinChance: 53.01,
					PlayerGammonChance: 10.22, PlayerBackgammonChance: 0.11,
					OpponentWinChance: 46.99, OpponentGammonChance: 12.88,
					OpponentBackgammonChance: 0.45,
				},
			},
		},
		PlayedMoves:  []string{"13/7 8/4"},
		CreationDate: time.Now(),
	}

	// Encode (marshal + compress)
	compressed, err := encodeAnalysisForStorage(&analysis)
	if err != nil {
		t.Fatalf("encodeAnalysisForStorage: %v", err)
	}

	// Must be smaller than raw JSON
	rawJSON, _ := json.Marshal(analysis)
	if len(compressed) >= len(rawJSON) {
		t.Errorf("compressed (%d bytes) not smaller than raw JSON (%d bytes)", len(compressed), len(rawJSON))
	}
	t.Logf("Raw JSON: %d bytes, compressed: %d bytes (%.0f%% reduction)",
		len(rawJSON), len(compressed), (1-float64(len(compressed))/float64(len(rawJSON)))*100)

	// Must NOT start with '{' (compressed, not raw JSON)
	if compressed[0] == '{' {
		t.Error("compressed data starts with '{' — not actually compressed")
	}

	// Decode (decompress + unmarshal)
	decoded, err := decodeAnalysisFromStorage(compressed)
	if err != nil {
		t.Fatalf("decodeAnalysisFromStorage: %v", err)
	}

	// Verify key fields survived
	if decoded.PositionID != analysis.PositionID {
		t.Errorf("PositionID: got %d, want %d", decoded.PositionID, analysis.PositionID)
	}
	if decoded.AnalysisType != analysis.AnalysisType {
		t.Errorf("AnalysisType: got %q, want %q", decoded.AnalysisType, analysis.AnalysisType)
	}
	if decoded.CheckerAnalysis == nil || len(decoded.CheckerAnalysis.Moves) != 2 {
		t.Fatalf("CheckerAnalysis.Moves: got %v, want 2 moves", decoded.CheckerAnalysis)
	}
	if decoded.CheckerAnalysis.Moves[0].Move != "13/7 8/4" {
		t.Errorf("Move[0]: got %q, want %q", decoded.CheckerAnalysis.Moves[0].Move, "13/7 8/4")
	}
	if decoded.CheckerAnalysis.Moves[0].PlayerWinChance != 55.12 {
		t.Errorf("Move[0].PlayerWinChance: got %f, want 55.12", decoded.CheckerAnalysis.Moves[0].PlayerWinChance)
	}
}

// TestDecompressRawJSON verifies backward compat: raw JSON (uncompressed) is
// accepted by decodeAnalysisFromStorage.
func TestDecompressRawJSON(t *testing.T) {
	analysis := PositionAnalysis{
		PositionID:   1,
		AnalysisType: "DoublingCube",
		DoublingCubeAnalysis: &DoublingCubeAnalysis{
			BestCubeAction:   "No Double",
			PlayerWinChances: 60.5,
		},
	}
	rawJSON, _ := json.Marshal(analysis)

	decoded, err := decodeAnalysisFromStorage(rawJSON)
	if err != nil {
		t.Fatalf("decodeAnalysisFromStorage with raw JSON: %v", err)
	}
	if decoded.DoublingCubeAnalysis == nil || decoded.DoublingCubeAnalysis.BestCubeAction != "No Double" {
		t.Errorf("failed to decode raw JSON: %+v", decoded)
	}
}

// TestRecompressAnalysisData verifies the helper that ensures data is compressed.
func TestRecompressAnalysisData(t *testing.T) {
	rawJSON := []byte(`{"positionId":1,"analysisType":"CheckerMove"}`)

	// Raw JSON should be compressed
	recompressed, err := recompressAnalysisData(rawJSON)
	if err != nil {
		t.Fatalf("recompressAnalysisData: %v", err)
	}
	if recompressed[0] == '{' {
		t.Error("recompressed data still starts with '{'")
	}

	// Already compressed data should pass through
	recompressed2, err := recompressAnalysisData(recompressed)
	if err != nil {
		t.Fatalf("recompressAnalysisData on compressed: %v", err)
	}
	if string(recompressed2) != string(recompressed) {
		t.Error("already-compressed data was modified by recompressAnalysisData")
	}

	// Empty data should pass through
	empty, err := recompressAnalysisData([]byte{})
	if err != nil {
		t.Fatalf("recompressAnalysisData on empty: %v", err)
	}
	if len(empty) != 0 {
		t.Error("empty data was not passed through")
	}
}

// TestMigrate_2_2_0_to_2_3_0 verifies that the migration compresses analysis
// data and that the compressed data is readable.
func TestMigrate_2_2_0_to_2_3_0(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "v220.db")

	// Create a v2.2.0 database with raw JSON analysis data
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	// Minimal v2.2.0 schema
	_, err = db.Exec(`
		CREATE TABLE position (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			zobrist_hash INTEGER, decision_type INTEGER, player_on_roll INTEGER,
			dice_1 INTEGER, dice_2 INTEGER, cube_value INTEGER, cube_owner INTEGER,
			score_1 INTEGER, score_2 INTEGER, match_length INTEGER,
			has_jacoby INTEGER, has_beaver INTEGER,
			pip_1 INTEGER, pip_2 INTEGER, pip_diff INTEGER,
			off_1 INTEGER, off_2 INTEGER,
			back_checkers_1 INTEGER, back_checkers_2 INTEGER, no_contact INTEGER,
			occupancy_1 INTEGER, occupancy_2 INTEGER, point_mask_1 INTEGER, point_mask_2 INTEGER,
			state TEXT NOT NULL
		);
		CREATE TABLE analysis (
			id INTEGER PRIMARY KEY,
			position_id INTEGER,
			data JSON,
			best_cube_action TEXT,
			cube_error REAL,
			best_move_equity_error REAL,
			player1_win_rate REAL,
			player1_gammon_rate REAL,
			player1_backgammon_rate REAL,
			player2_win_rate REAL,
			player2_gammon_rate REAL,
			player2_backgammon_rate REAL,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE
		);
		CREATE TABLE comment (id INTEGER PRIMARY KEY, position_id INTEGER, text TEXT,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE);
		CREATE TABLE metadata (key TEXT PRIMARY KEY, value TEXT);
		CREATE TABLE command_history (id INTEGER PRIMARY KEY AUTOINCREMENT, command TEXT, timestamp DATETIME DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE filter_library (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, command TEXT, edit_position TEXT);
		CREATE TABLE search_history (id INTEGER PRIMARY KEY AUTOINCREMENT, command TEXT, position TEXT, timestamp INTEGER);
		CREATE TABLE match (id INTEGER PRIMARY KEY AUTOINCREMENT, player1_name TEXT, player2_name TEXT,
			event TEXT, location TEXT, round TEXT, match_length INTEGER, match_date DATETIME,
			import_date DATETIME DEFAULT CURRENT_TIMESTAMP, file_path TEXT, game_count INTEGER DEFAULT 0,
			match_hash TEXT, canonical_hash TEXT,
			tournament_id INTEGER, last_visited_position INTEGER DEFAULT -1);
		CREATE TABLE game (id INTEGER PRIMARY KEY AUTOINCREMENT, match_id INTEGER, game_number INTEGER,
			initial_score_1 INTEGER, initial_score_2 INTEGER, winner INTEGER, points_won INTEGER,
			move_count INTEGER DEFAULT 0,
			FOREIGN KEY(match_id) REFERENCES match(id) ON DELETE CASCADE);
		CREATE TABLE move (id INTEGER PRIMARY KEY AUTOINCREMENT, game_id INTEGER, move_number INTEGER,
			move_type TEXT, position_id INTEGER, player INTEGER, dice_1 INTEGER, dice_2 INTEGER,
			checker_move TEXT, cube_action TEXT,
			FOREIGN KEY(game_id) REFERENCES game(id) ON DELETE CASCADE,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE SET NULL);
		CREATE TABLE move_analysis (id INTEGER PRIMARY KEY AUTOINCREMENT, move_id INTEGER, analysis_type TEXT,
			depth TEXT, equity INTEGER, equity_error INTEGER, win_rate INTEGER, gammon_rate INTEGER,
			backgammon_rate INTEGER, opponent_win_rate INTEGER, opponent_gammon_rate INTEGER,
			opponent_backgammon_rate INTEGER,
			FOREIGN KEY(move_id) REFERENCES move(id) ON DELETE CASCADE);
		CREATE TABLE collection (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, description TEXT,
			sort_order INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE collection_position (id INTEGER PRIMARY KEY AUTOINCREMENT,
			collection_id INTEGER NOT NULL, position_id INTEGER NOT NULL,
			sort_order INTEGER DEFAULT 0, added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(collection_id) REFERENCES collection(id) ON DELETE CASCADE,
			FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE,
			UNIQUE(collection_id, position_id));
		CREATE TABLE tournament (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, date TEXT,
			location TEXT, sort_order INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_position_zobrist ON position(zobrist_hash);
		CREATE INDEX IF NOT EXISTS idx_analysis_position ON analysis(position_id);
		INSERT INTO metadata (key, value) VALUES ('database_version', '2.2.0');
	`)
	if err != nil {
		t.Fatalf("setup v2.2.0 schema: %v", err)
	}

	// Insert a position with a compact board state
	pos := initialPosition()
	norm := pos.NormalizeForStorage()
	compactState := encodeBoardCompact(norm.Board)
	cols := populatePositionColumns(&pos)
	noContactInt := 0
	if cols.NoContact {
		noContactInt = 1
	}
	result, err := db.Exec(`
		INSERT INTO position (zobrist_hash, decision_type, player_on_roll, dice_1, dice_2,
			cube_value, cube_owner, score_1, score_2, has_jacoby, has_beaver,
			pip_1, pip_2, pip_diff, off_1, off_2,
			back_checkers_1, back_checkers_2, no_contact,
			occupancy_1, occupancy_2, point_mask_1, point_mask_2, state)
		VALUES (?,?,?,?,?, ?,?,?,?, ?,?, ?,?,?,?,?, ?,?,?, ?,?,?,?, ?)`,
		int64(cols.ZobristHash), cols.DecisionType, norm.PlayerOnRoll, cols.Dice1, cols.Dice2,
		cols.CubeValue, cols.CubeOwner, cols.Score1, cols.Score2,
		cols.HasJacoby, cols.HasBeaver,
		cols.Pip1, cols.Pip2, cols.PipDiff, cols.Off1, cols.Off2,
		cols.BackCheckers1, cols.BackCheckers2, noContactInt,
		int64(cols.Occupancy1), int64(cols.Occupancy2), int64(cols.PointMask1), int64(cols.PointMask2),
		compactState)
	if err != nil {
		t.Fatalf("insert position: %v", err)
	}
	posID, _ := result.LastInsertId()

	// Insert analysis as raw JSON (v2.2.0 format)
	analysis := PositionAnalysis{
		PositionID:   int(posID),
		AnalysisType: "CheckerMove",
		CheckerAnalysis: &CheckerAnalysis{
			Moves: []CheckerMove{
				{Index: 0, Move: "13/7 8/4", Equity: 0.123, PlayerWinChance: 55.12,
					PlayerGammonChance: 12.34, OpponentWinChance: 44.88},
				{Index: 1, Move: "24/18", Equity: 0.089, PlayerWinChance: 53.0,
					PlayerGammonChance: 10.0, OpponentWinChance: 47.0},
			},
		},
		PlayedMoves: []string{"13/7 8/4"},
	}
	analysisJSON, _ := json.Marshal(analysis)
	rawJSONSize := len(analysisJSON)

	_, err = db.Exec(`INSERT INTO analysis (position_id, data) VALUES (?, ?)`, posID, string(analysisJSON))
	if err != nil {
		t.Fatalf("insert analysis: %v", err)
	}
	db.Close()

	// Open database — triggers migration to 2.3.0
	d := NewDatabase()
	if err := d.OpenDatabase(dbPath); err != nil {
		t.Fatalf("OpenDatabase: %v", err)
	}

	// Verify version bumped
	ver, _ := d.CheckDatabaseVersion()
	if ver != "2.3.0" {
		t.Fatalf("expected version 2.3.0, got %s", ver)
	}

	// Verify analysis data is now compressed (raw bytes should not start with '{')
	var storedData []byte
	err = d.db.QueryRow(`SELECT data FROM analysis WHERE position_id = ?`, posID).Scan(&storedData)
	if err != nil {
		t.Fatalf("query analysis: %v", err)
	}
	if len(storedData) == 0 {
		t.Fatal("analysis data is empty after migration")
	}
	if storedData[0] == '{' {
		t.Error("analysis data is still raw JSON after migration — compression not applied")
	}
	if len(storedData) >= rawJSONSize {
		t.Errorf("compressed (%d bytes) not smaller than raw JSON (%d bytes)", len(storedData), rawJSONSize)
	}
	t.Logf("Migration: raw JSON %d bytes → compressed %d bytes (%.0f%% reduction)",
		rawJSONSize, len(storedData), (1-float64(len(storedData))/float64(rawJSONSize))*100)

	// Verify analysis is readable through LoadAnalysis
	loaded, err := d.LoadAnalysis(posID)
	if err != nil {
		t.Fatalf("LoadAnalysis: %v", err)
	}
	if loaded.AnalysisType != "CheckerMove" {
		t.Errorf("AnalysisType: got %q, want %q", loaded.AnalysisType, "CheckerMove")
	}
	if loaded.CheckerAnalysis == nil || len(loaded.CheckerAnalysis.Moves) != 2 {
		t.Fatalf("CheckerAnalysis.Moves: got %v, want 2 moves", loaded.CheckerAnalysis)
	}
	if loaded.CheckerAnalysis.Moves[0].Move != "13/7 8/4" {
		t.Errorf("Move[0]: got %q, want %q", loaded.CheckerAnalysis.Moves[0].Move, "13/7 8/4")
	}
}

// TestSaveAndLoadAnalysisCompressed verifies the full save→load round trip
// with compressed storage on a fresh database.
func TestSaveAndLoadAnalysisCompressed(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_compressed.db")

	d := NewDatabase()
	if err := d.SetupDatabase(dbPath); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}

	// Create a position
	pos := initialPosition()
	posID, err := d.SavePosition(&pos)
	if err != nil {
		t.Fatalf("SavePosition: %v", err)
	}

	// Save analysis
	analysis := PositionAnalysis{
		AnalysisType:          "CheckerMove",
		AnalysisEngineVersion: "XG2+",
		CheckerAnalysis: &CheckerAnalysis{
			Moves: []CheckerMove{
				{Index: 0, Move: "13/7 8/4", Equity: 0.123,
					PlayerWinChance: 55.12, PlayerGammonChance: 12.34,
					OpponentWinChance: 44.88, OpponentGammonChance: 11.23},
				{Index: 1, Move: "24/18 13/9", Equity: 0.089,
					PlayerWinChance: 53.0, PlayerGammonChance: 10.0,
					OpponentWinChance: 47.0, OpponentGammonChance: 12.0},
			},
		},
		PlayedMoves: []string{"13/7 8/4"},
	}
	if err := d.SaveAnalysis(posID, analysis); err != nil {
		t.Fatalf("SaveAnalysis: %v", err)
	}

	// Verify raw storage is compressed
	var storedData []byte
	err = d.db.QueryRow(`SELECT data FROM analysis WHERE position_id = ?`, posID).Scan(&storedData)
	if err != nil {
		t.Fatalf("query stored data: %v", err)
	}
	if storedData[0] == '{' {
		t.Error("stored data is not compressed")
	}

	// Load and verify
	loaded, err := d.LoadAnalysis(posID)
	if err != nil {
		t.Fatalf("LoadAnalysis: %v", err)
	}
	if loaded.AnalysisType != "CheckerMove" {
		t.Errorf("AnalysisType: got %q, want %q", loaded.AnalysisType, "CheckerMove")
	}
	if loaded.CheckerAnalysis == nil || len(loaded.CheckerAnalysis.Moves) != 2 {
		t.Fatalf("wrong number of moves: %v", loaded.CheckerAnalysis)
	}
	if loaded.CheckerAnalysis.Moves[0].Move != "13/7 8/4" {
		t.Errorf("Move[0].Move: got %q, want %q", loaded.CheckerAnalysis.Moves[0].Move, "13/7 8/4")
	}

	// Save again (merge path) and verify
	analysis2 := PositionAnalysis{
		AnalysisType: "CheckerMove",
		CheckerAnalysis: &CheckerAnalysis{
			Moves: []CheckerMove{
				{Index: 0, Move: "8/2 6/2", Equity: 0.050,
					PlayerWinChance: 50.0, OpponentWinChance: 50.0},
			},
		},
		PlayedMoves: []string{"8/2 6/2"},
	}
	if err := d.SaveAnalysis(posID, analysis2); err != nil {
		t.Fatalf("SaveAnalysis merge: %v", err)
	}

	loaded2, err := d.LoadAnalysis(posID)
	if err != nil {
		t.Fatalf("LoadAnalysis after merge: %v", err)
	}
	// Should have merged moves
	if loaded2.CheckerAnalysis == nil || len(loaded2.CheckerAnalysis.Moves) < 2 {
		t.Errorf("expected merged moves, got %d", len(loaded2.CheckerAnalysis.Moves))
	}
}

// TestExportWritesUncompressedJSON verifies that ExportDatabase writes
// uncompressed JSON to the export file for backward compatibility.
func TestExportWritesUncompressedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_export_src.db")

	d := NewDatabase()
	if err := d.SetupDatabase(dbPath); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}

	// Create a position with analysis
	pos := initialPosition()
	posID, err := d.SavePosition(&pos)
	if err != nil {
		t.Fatalf("SavePosition: %v", err)
	}

	analysis := PositionAnalysis{
		AnalysisType: "CheckerMove",
		CheckerAnalysis: &CheckerAnalysis{
			Moves: []CheckerMove{
				{Index: 0, Move: "13/7", Equity: 0.1, PlayerWinChance: 55.0, OpponentWinChance: 45.0},
			},
		},
	}
	if err := d.SaveAnalysis(posID, analysis); err != nil {
		t.Fatalf("SaveAnalysis: %v", err)
	}

	// Export
	exportPath := filepath.Join(tmpDir, "exported.db")
	positions, _ := d.LoadAllPositions()
	err = d.ExportDatabase(ExportOptions{
		ExportPath:         exportPath,
		Positions:          positions,
		Metadata:           map[string]string{},
		IncludeAnalysis:    true,
		IncludeComments:    true,
		IncludePlayedMoves: true,
		IncludeMatches:     true,
	})
	if err != nil {
		t.Fatalf("ExportDatabase: %v", err)
	}

	// Open export DB and check that analysis data is raw JSON (starts with '{')
	exportDB, err := sql.Open("sqlite", exportPath)
	if err != nil {
		t.Fatalf("open export DB: %v", err)
	}
	defer exportDB.Close()

	var exportedData string
	err = exportDB.QueryRow(`SELECT data FROM analysis LIMIT 1`).Scan(&exportedData)
	if err != nil {
		t.Fatalf("query export analysis: %v", err)
	}
	if !strings.HasPrefix(exportedData, "{") {
		t.Errorf("exported analysis data is not raw JSON (starts with %q)", string(exportedData[:1]))
	}

	// Verify it's valid JSON
	var exported PositionAnalysis
	if err := json.Unmarshal([]byte(exportedData), &exported); err != nil {
		t.Errorf("exported data is not valid JSON: %v", err)
	}
}

// TestAnalysisCompressionSavings measures the compression ratio on a larger
// analysis dataset to confirm storage savings.
func TestAnalysisCompressionSavings(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_savings.db")

	d := NewDatabase()
	if err := d.SetupDatabase(dbPath); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}

	// Create 50 positions with varied analysis data
	var totalRawJSON int
	var totalCompressed int
	for i := 0; i < 50; i++ {
		pos := initialPosition()
		pos.Dice = [2]int{(i % 6) + 1, (i%5+i%3)%6 + 1}
		pos.Score = [2]int{i % 9, (i + 3) % 9}
		pos.Cube.Value = i + 1 // ensure unique zobrist hash
		posID, err := d.SavePosition(&pos)
		if err != nil {
			t.Fatalf("SavePosition %d: %v", i, err)
		}

		// Build analysis with varying number of moves
		numMoves := 5 + (i % 15) // 5-19 moves
		moves := make([]CheckerMove, numMoves)
		for j := 0; j < numMoves; j++ {
			eq := 0.5 - float64(j)*0.01
			eqErr := float64(j) * 0.01
			moves[j] = CheckerMove{
				Index: j, AnalysisDepth: "3-ply",
				Move:   fmt.Sprintf("%d/%d %d/%d", 24-j, 18-j, 13-j, 7-j),
				Equity: eq, PlayerWinChance: 50 + float64(j),
				PlayerGammonChance:       10 + float64(j%5),
				PlayerBackgammonChance:   float64(j % 3),
				OpponentWinChance:        50 - float64(j),
				OpponentGammonChance:     10 - float64(j%5),
				OpponentBackgammonChance: float64(j % 2),
			}
			if j > 0 {
				moves[j].EquityError = &eqErr
			}
		}

		analysis := PositionAnalysis{
			AnalysisType:          "CheckerMove",
			AnalysisEngineVersion: "XG2+",
			Player1:               "TestPlayer1",
			Player2:               "TestPlayer2",
			CheckerAnalysis:       &CheckerAnalysis{Moves: moves},
			PlayedMoves:           []string{moves[0].Move},
		}

		rawJSON, _ := json.Marshal(analysis)
		totalRawJSON += len(rawJSON)

		if err := d.SaveAnalysis(posID, analysis); err != nil {
			t.Fatalf("SaveAnalysis %d: %v", i, err)
		}

		var storedData []byte
		d.db.QueryRow(`SELECT data FROM analysis WHERE position_id = ?`, posID).Scan(&storedData)
		totalCompressed += len(storedData)
	}

	ratio := float64(totalCompressed) / float64(totalRawJSON) * 100
	t.Logf("50 analyses: raw JSON %d bytes, compressed %d bytes (%.1f%% of original, %.1f%% reduction)",
		totalRawJSON, totalCompressed, ratio, 100-ratio)

	if ratio > 60 {
		t.Errorf("compression ratio %.1f%% is worse than expected (<60%% of original)", ratio)
	}
}

// TestImportV220DatabaseIntoV230 verifies that importing an older v2.2.0 database
// (with uncompressed analysis JSON) into a v2.3.0 database works correctly.
func TestImportV220DatabaseIntoV230(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a v2.3.0 "current" database with one position
	currentPath := filepath.Join(tmpDir, "current.db")
	dCurrent := NewDatabase()
	if err := dCurrent.SetupDatabase(currentPath); err != nil {
		t.Fatalf("SetupDatabase current: %v", err)
	}

	// Create a v2.2.0-style "import" database with uncompressed JSON
	importPath := filepath.Join(tmpDir, "import.db")
	importDB, err := sql.Open("sqlite", importPath)
	if err != nil {
		t.Fatalf("open import: %v", err)
	}

	importDB.Exec(`CREATE TABLE position (id INTEGER PRIMARY KEY AUTOINCREMENT, state TEXT)`)
	importDB.Exec(`CREATE TABLE analysis (id INTEGER PRIMARY KEY, position_id INTEGER, data JSON,
		FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE)`)
	importDB.Exec(`CREATE TABLE comment (id INTEGER PRIMARY KEY, position_id INTEGER, text TEXT,
		FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE)`)
	importDB.Exec(`CREATE TABLE metadata (key TEXT PRIMARY KEY, value TEXT)`)
	importDB.Exec(`INSERT INTO metadata (key, value) VALUES ('database_version', '2.2.0')`)

	// Insert a position + raw JSON analysis
	pos := initialPosition()
	norm := pos.NormalizeForStorage()
	posJSON, _ := json.Marshal(norm)
	importDB.Exec(`INSERT INTO position (state) VALUES (?)`, string(posJSON))

	analysis := PositionAnalysis{
		PositionID:   1,
		AnalysisType: "CheckerMove",
		CheckerAnalysis: &CheckerAnalysis{
			Moves: []CheckerMove{
				{Index: 0, Move: "13/7 8/4", Equity: 0.123, PlayerWinChance: 55.0, OpponentWinChance: 45.0},
			},
		},
	}
	analysisJSON, _ := json.Marshal(analysis)
	importDB.Exec(`INSERT INTO analysis (position_id, data) VALUES (1, ?)`, string(analysisJSON))
	importDB.Close()

	// Import into current DB
	_, err = dCurrent.CommitImportDatabase(importPath)
	if err != nil {
		t.Fatalf("CommitImportDatabase: %v", err)
	}

	// Verify the imported analysis is readable
	positions, err := dCurrent.LoadAllPositions()
	if err != nil {
		t.Fatalf("LoadAllPositions: %v", err)
	}
	if len(positions) == 0 {
		t.Fatal("no positions after import")
	}

	for _, p := range positions {
		loaded, err := dCurrent.LoadAnalysis(p.ID)
		if err == nil && loaded != nil && loaded.AnalysisType == "CheckerMove" {
			if loaded.CheckerAnalysis == nil || len(loaded.CheckerAnalysis.Moves) == 0 {
				t.Error("imported analysis has no moves")
			}
			return // success
		}
	}
	t.Error("could not find imported analysis with CheckerMove type")
}
