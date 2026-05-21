package database

import (
	"path/filepath"
	"testing"
)

// newTestDB creates a file-backed database in t.TempDir with the current schema.
// Cleanup is registered automatically via t.Cleanup.
func newTestDB(t *testing.T) *Database {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db := NewDatabase()
	if err := db.SetupDatabase(dbPath); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	t.Cleanup(func() {
		if db.db != nil {
			db.db.Close()
		}
	})
	return db
}

// newTestDBWithXG creates a file-backed database and imports testdata/test.xg.
func newTestDBWithXG(t *testing.T) *Database {
	t.Helper()
	db := newTestDB(t)
	if _, err := db.ImportXGMatch(filepath.Join("testdata", "test.xg")); err != nil {
		t.Fatalf("ImportXGMatch: %v", err)
	}
	return db
}

// importTestMatch imports testdata/test.sgf and returns the match ID.
func importTestMatch(t *testing.T, db *Database) int64 {
	t.Helper()
	matchID, err := db.ImportGnuBGMatch(filepath.Join("testdata", "test.sgf"))
	if err != nil {
		t.Fatalf("ImportGnuBGMatch: %v", err)
	}
	return matchID
}

// getPositionIDs returns position IDs from the database (up to limit).
func getPositionIDs(t *testing.T, db *Database, limit int) []int64 {
	t.Helper()
	rows, err := db.db.Query(`SELECT id FROM position ORDER BY id LIMIT ?`, limit)
	if err != nil {
		t.Fatalf("query positions: %v", err)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan: %v", err)
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		t.Fatal("no positions in database")
	}
	return ids
}

// initialBoard returns the standard starting backgammon position.
// Board indices 0=WhiteBar, 1..24=board points, 25=BlackBar.
func initialBoard() Board {
	var b Board
	// Black checkers
	b.Points[24] = Point{Checkers: 2, Color: Black}
	b.Points[13] = Point{Checkers: 5, Color: Black}
	b.Points[8] = Point{Checkers: 3, Color: Black}
	b.Points[6] = Point{Checkers: 5, Color: Black}
	// White checkers
	b.Points[1] = Point{Checkers: 2, Color: White}
	b.Points[12] = Point{Checkers: 5, Color: White}
	b.Points[17] = Point{Checkers: 3, Color: White}
	b.Points[19] = Point{Checkers: 5, Color: White}
	return b
}

// initialPosition returns the standard starting position (checker decision).
func initialPosition() Position {
	return Position{
		Board:        initialBoard(),
		Cube:         Cube{Owner: None, Value: 0}, // 0 = exponent for cube-at-1
		PlayerOnRoll: 0,
		DecisionType: CheckerAction,
	}
}

// bearoffPosition returns a pure race where both players have only their
// home-board checkers left.
func bearoffPosition() Position {
	var b Board
	// Black racing home (points 1-6)
	b.Points[1] = Point{Checkers: 3, Color: Black}
	b.Points[2] = Point{Checkers: 3, Color: Black}
	b.Points[3] = Point{Checkers: 3, Color: Black}
	b.Points[4] = Point{Checkers: 3, Color: Black}
	b.Points[5] = Point{Checkers: 3, Color: Black}
	// White racing home (points 19-24)
	b.Points[20] = Point{Checkers: 3, Color: White}
	b.Points[21] = Point{Checkers: 3, Color: White}
	b.Points[22] = Point{Checkers: 3, Color: White}
	b.Points[23] = Point{Checkers: 3, Color: White}
	b.Points[24] = Point{Checkers: 3, Color: White}
	return Position{Board: b, Cube: Cube{Owner: None, Value: 0}, PlayerOnRoll: 0, DecisionType: CheckerAction}
}

// cubePosition creates a cube-decision position. cubeExp is the cube exponent:
// 0 = cube at 1 (initial), 1 = cube at 2, 2 = cube at 4, …
func cubePosition(cubeExp int, cubeOwner int) Position {
	return Position{
		Board:        initialBoard(),
		Cube:         Cube{Owner: cubeOwner, Value: cubeExp},
		PlayerOnRoll: 0,
		DecisionType: CubeAction,
	}
}
