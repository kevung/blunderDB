package database

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// TestSchemaV200_PositionColumns verifies that a fresh v2.0.0 database has all
// expected scalar columns on the position table and all v2.0.0 indexes.
func TestSchemaV200_PositionColumns(t *testing.T) {
	d := NewDatabase()
	if err := d.SetupDatabase(":memory:"); err != nil {
		t.Fatalf("SetupDatabase failed: %v", err)
	}
	defer d.db.Close()

	wantPositionCols := []string{
		"id", "zobrist_hash", "decision_type", "player_on_roll",
		"dice_1", "dice_2", "cube_value", "cube_owner",
		"score_1", "score_2", "match_length", "has_jacoby", "has_beaver",
		"pip_1", "pip_2", "pip_diff", "off_1", "off_2",
		"back_checkers_1", "back_checkers_2", "no_contact",
		"occupancy_1", "occupancy_2", "point_mask_1", "point_mask_2",
		"state",
	}

	var sql_ string
	if err := d.db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name='position'`).Scan(&sql_); err != nil {
		t.Fatalf("could not read position schema: %v", err)
	}
	for _, col := range wantPositionCols {
		if !strings.Contains(sql_, col) {
			t.Errorf("position table missing column: %s", col)
		}
	}

	wantAnalysisCols := []string{
		"id", "position_id", "data",
		"best_cube_action", "cube_error", "best_move_equity_error",
		"player1_win_rate", "player1_gammon_rate", "player1_backgammon_rate",
		"player2_win_rate", "player2_gammon_rate", "player2_backgammon_rate",
	}
	if err := d.db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name='analysis'`).Scan(&sql_); err != nil {
		t.Fatalf("could not read analysis schema: %v", err)
	}
	for _, col := range wantAnalysisCols {
		if !strings.Contains(sql_, col) {
			t.Errorf("analysis table missing column: %s", col)
		}
	}
}

// TestSchemaV200_Indexes verifies that all expected v2.0.0-and-later indexes
// are created. idx_analysis_win_gammon_covering superseded the 2-column
// idx_analysis_win_gammon (fiche-05 T3): the old name is no longer created on
// a fresh database, so it is not in wantIndexes below. idx_position_score and
// idx_analysis_win1 (E3, index redundancy pass) are likewise gone — both were
// strict column prefixes of an index still in the list
// (idx_position_score_cube, idx_analysis_win_gammon_covering respectively).
func TestSchemaV200_Indexes(t *testing.T) {
	d := NewDatabase()
	if err := d.SetupDatabase(":memory:"); err != nil {
		t.Fatalf("SetupDatabase failed: %v", err)
	}
	defer d.db.Close()

	wantIndexes := []string{
		"idx_position_zobrist",
		"idx_position_decision_pip",
		"idx_position_decision_dice",
		"idx_position_pip_diff",
		"idx_position_dice",
		"idx_position_off",
		"idx_position_score_cube",
		"idx_analysis_position",
		"idx_analysis_win_gammon_covering",
		"idx_analysis_cube_error",
		"idx_analysis_move_error",
		"idx_move_position",
		"idx_move_game",
		"idx_game_match",
	}
	rows, err := d.db.Query(`SELECT name FROM sqlite_master WHERE type='index'`)
	if err != nil {
		t.Fatalf("could not query indexes: %v", err)
	}
	defer rows.Close()
	gotSet := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		gotSet[name] = true
	}
	for _, idx := range wantIndexes {
		if !gotSet[idx] {
			t.Errorf("index missing: %s", idx)
		}
	}
}

// TestSchemaV200_SavePositionColumns verifies that SavePosition writes scalar columns.
func TestSchemaV200_SavePositionColumns(t *testing.T) {
	d := NewDatabase()
	if err := d.SetupDatabase(":memory:"); err != nil {
		t.Fatalf("SetupDatabase failed: %v", err)
	}
	defer d.db.Close()

	pos := &Position{
		Board:        initialBoard(),
		Cube:         Cube{Owner: -1, Value: 0},
		Dice:         [2]int{3, 1},
		Score:        [2]int{5, 5},
		PlayerOnRoll: 0,
		DecisionType: CheckerAction,
	}
	id, err := d.SavePosition(pos)
	if err != nil {
		t.Fatalf("SavePosition failed: %v", err)
	}

	var zobrist int64
	var pip1, pip2 int
	if err := d.db.QueryRow(`SELECT zobrist_hash, pip_1, pip_2 FROM position WHERE id=?`, id).
		Scan(&zobrist, &pip1, &pip2); err != nil {
		t.Fatalf("could not read back position columns: %v", err)
	}
	if zobrist == 0 {
		t.Error("zobrist_hash should not be 0 for initial position")
	}
	if pip1 != 167 || pip2 != 167 {
		t.Errorf("expected pip1=167 pip2=167 for initial position, got pip1=%d pip2=%d", pip1, pip2)
	}
}

// TestSchemaV200_DatabaseVersion verifies that SetupDatabase writes the current DatabaseVersion.
func TestSchemaV200_DatabaseVersion(t *testing.T) {
	d := NewDatabase()
	if err := d.SetupDatabase(":memory:"); err != nil {
		t.Fatalf("SetupDatabase failed: %v", err)
	}
	defer d.db.Close()

	var version string
	if err := d.db.QueryRow(`SELECT value FROM metadata WHERE key='database_version'`).Scan(&version); err != nil {
		t.Fatalf("could not read version: %v", err)
	}
	if version != DatabaseVersion {
		t.Errorf("expected DatabaseVersion %s, got %s", DatabaseVersion, version)
	}
}

// TestOpen_RepairsPositionsWithoutScalars covers the databases the bug above
// already damaged: rows CommitImportDatabase inserted with their state alone.
// Opening such a database must give those rows their hash and scalar columns
// back (from the full JSON state, the only faithful record) and fold the
// duplicates the missing hash let through onto the row the index holds —
// carrying the analysis, comment and collection membership across — without a
// schema version bump.
func TestOpen_RepairsPositionsWithoutScalars(t *testing.T) {
	path := filepath.Join(t.TempDir(), "damaged.db")
	d := NewDatabase()
	if err := d.SetupDatabase(path); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}

	// A healthy row, saved the ordinary way.
	healthy := InitializePosition()
	healthyID, err := d.SavePosition(&healthy)
	if err != nil {
		t.Fatalf("SavePosition: %v", err)
	}

	// insertDamaged writes a row exactly the way the old importer did.
	insertDamaged := func(pos Position) int64 {
		t.Helper()
		data, err := json.Marshal(pos.NormalizeForStorage())
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		res, err := d.db.Exec(`INSERT INTO position (state, individually_imported) VALUES (?, 1)`, string(data))
		if err != nil {
			t.Fatalf("insert damaged row: %v", err)
		}
		id, _ := res.LastInsertId()
		return id
	}

	// A damaged duplicate of the healthy row, carrying what the healthy one lacks.
	dupID := insertDamaged(healthy)
	if err := d.SaveAnalysis(dupID, PositionAnalysis{XGID: "dup-xgid", AnalysisType: "XG Roller++"}); err != nil {
		t.Fatalf("SaveAnalysis: %v", err)
	}
	if err := d.SaveComment(dupID, "moved along"); err != nil {
		t.Fatalf("SaveComment: %v", err)
	}
	collID, err := d.CreateCollection("course", "")
	if err != nil {
		t.Fatalf("CreateCollection: %v", err)
	}
	if err := d.AddPositionToCollection(collID, dupID); err != nil {
		t.Fatalf("AddPositionToCollection: %v", err)
	}

	// A damaged row that is a position of its own: dice 6-5 live only in its JSON.
	lone := InitializePosition()
	lone.Dice = [2]int{6, 5}
	loneID := insertDamaged(lone)

	// A damaged row nobody can decode must not block the open.
	if _, err := d.db.Exec(`INSERT INTO position (state) VALUES ('{not json')`); err != nil {
		t.Fatalf("insert undecodable row: %v", err)
	}

	if n := countPositionsMissingScalars(t, d); n != 3 {
		t.Fatalf("fixture: %d rows without scalars, want 3", n)
	}
	d.Close()

	if err := d.OpenDatabase(path); err != nil {
		t.Fatalf("OpenDatabase: %v", err)
	}
	defer d.Close()

	// Only the undecodable row is left without scalars.
	if n := countPositionsMissingScalars(t, d); n != 1 {
		t.Errorf("after open: %d rows without scalars, want 1 (the undecodable one)", n)
	}
	var total int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM position`).Scan(&total); err != nil {
		t.Fatal(err)
	}
	if total != 3 {
		t.Errorf("after open: %d positions, want 3 (healthy, lone, undecodable)", total)
	}

	// The duplicate folded onto the healthy row, with its belongings.
	if _, err := d.LoadPosition(int(dupID)); err == nil {
		t.Errorf("duplicate row %d still exists", dupID)
	}
	if a, err := d.LoadAnalysis(healthyID); err != nil || a.XGID != "dup-xgid" {
		t.Errorf("analysis did not follow the duplicate onto %d: %v, %v", healthyID, a, err)
	} else if a.PositionID != int(healthyID) {
		t.Errorf("moved analysis names position %d inside its blob, want %d", a.PositionID, healthyID)
	}
	if c, err := d.LoadComment(healthyID); err != nil || c != "moved along" {
		t.Errorf("comment did not follow the duplicate: %q, %v", c, err)
	}
	members, err := d.GetCollectionPositions(collID)
	if err != nil {
		t.Fatalf("GetCollectionPositions: %v", err)
	}
	if len(members) != 1 || members[0].ID != healthyID {
		t.Errorf("collection membership did not follow the duplicate: %+v", members)
	}
	kept, err := d.LoadPosition(int(healthyID))
	if err != nil {
		t.Fatalf("LoadPosition(healthy): %v", err)
	}
	if !kept.IndividuallyImported {
		t.Errorf("sticky provenance was not raised on the kept row")
	}

	// The lone row got its columns back from its JSON, so the search sees it
	// and the store recognises it.
	repaired, err := d.LoadPosition(int(loneID))
	if err != nil {
		t.Fatalf("LoadPosition(lone): %v", err)
	}
	if repaired.Dice != [2]int{6, 5} || repaired.Score != lone.Score {
		t.Errorf("lone row lost its identity: dice=%v score=%v", repaired.Dice, repaired.Score)
	}
	found, err := d.LoadPositionsByFilters(SearchFilters{
		Filter:         Position{Dice: [2]int{6, 5}, PlayerOnRoll: lone.PlayerOnRoll, DecisionType: lone.DecisionType},
		DiceRollFilter: true,
	})
	if err != nil {
		t.Fatalf("LoadPositionsByFilters: %v", err)
	}
	if len(found) != 1 || found[0].ID != loneID {
		t.Errorf("dice filter found %+v, want the repaired row %d", found, loneID)
	}
	again := lone
	if res, err := d.SaveIndividualPosition(&again); err != nil || !res.Existed || res.ID != loneID {
		t.Errorf("SaveIndividualPosition after repair: %+v, %v (want existed id %d)", res, err, loneID)
	}

	// Idempotent: a second open has nothing to do and changes nothing.
	d.Close()
	if err := d.OpenDatabase(path); err != nil {
		t.Fatalf("OpenDatabase #2: %v", err)
	}
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM position`).Scan(&total); err != nil {
		t.Fatal(err)
	}
	if total != 3 {
		t.Errorf("second open changed the row count: %d", total)
	}
}
