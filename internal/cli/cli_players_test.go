package cli

import (
	"encoding/csv"
	"strings"
	"testing"
)

// setupCLIPlayers returns a CLI whose database holds one match between Alice
// and Bob, with two counted decisions for Alice and one for Bob.
func setupCLIPlayers(t *testing.T) *CLI {
	t.Helper()
	db := NewDatabase()
	if err := db.SetupDatabase(":memory:"); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	matchID := createMatch(t, db, "Alice", "Bob", "2025-01-01", 5, 0)
	gameID := createGame(t, db, matchID)
	insertStatsFixtureRow(t, db, matchID, gameID, 200, 0, 0, 1) // checker, Alice
	insertStatsFixtureRow(t, db, matchID, gameID, 50, 0, 0, 2)  // checker, Alice
	insertStatsFixtureRow(t, db, matchID, gameID, 100, 0, 1, 3) // checker, Bob

	return &CLI{db: db}
}

func TestCLIPlayers_TextFormat(t *testing.T) {
	cli := setupCLIPlayers(t)

	out := captureStdout(t, func() {
		if err := cli.showPlayerTable(StatsFilter{DecisionType: -1}, "text"); err != nil {
			t.Fatalf("showPlayerTable: %v", err)
		}
	})

	for _, want := range []string{"=== Players ===", "Alice", "Bob", "Snowie", "Luck"} {
		if !strings.Contains(out, want) {
			t.Errorf("output is missing %q:\n%s", want, out)
		}
	}
	// No luck was recorded, and an unmeasured figure must read as unknown
	// rather than as a perfectly average zero.
	if !strings.Contains(out, "—") {
		t.Errorf("an unmeasured luck must be shown as a dash:\n%s", out)
	}
}

func TestCLIPlayers_CSVFormat(t *testing.T) {
	cli := setupCLIPlayers(t)

	out := captureStdout(t, func() {
		if err := cli.showPlayerTable(StatsFilter{DecisionType: -1}, "csv"); err != nil {
			t.Fatalf("showPlayerTable: %v", err)
		}
	})

	records, err := csv.NewReader(strings.NewReader(out)).ReadAll()
	if err != nil {
		t.Fatalf("output is not valid CSV: %v\n%s", err, out)
	}
	if len(records) != 3 {
		t.Fatalf("got %d CSV lines, want a header and two players:\n%s", len(records), out)
	}

	header := records[0]
	if header[0] != "player" {
		t.Errorf("first column: got %q, want \"player\"", header[0])
	}
	luckCol := -1
	for i, h := range header {
		if h == "luck_rate_mp" {
			luckCol = i
		}
	}
	if luckCol == -1 {
		t.Fatalf("no luck_rate_mp column in %v", header)
	}

	// The luck field must be EMPTY, not "0": a spreadsheet averaging this
	// column has to skip the players nobody measured rather than count them as
	// exactly average.
	for _, row := range records[1:] {
		if row[luckCol] != "" {
			t.Errorf("player %q: got luck %q, want an empty field for an unmeasured value",
				row[0], row[luckCol])
		}
	}

	names := []string{records[1][0], records[2][0]}
	if !(names[0] == "Alice" && names[1] == "Bob") && !(names[0] == "Bob" && names[1] == "Alice") {
		t.Errorf("players: got %v, want Alice and Bob", names)
	}
}

func TestCLIPlayers_JSONFormat(t *testing.T) {
	cli := setupCLIPlayers(t)

	out := captureStdout(t, func() {
		if err := cli.showPlayerTable(StatsFilter{DecisionType: -1}, "json"); err != nil {
			t.Fatalf("showPlayerTable: %v", err)
		}
	})
	for _, want := range []string{`"name"`, `"pr"`, `"luck_known"`, "Alice", "Bob"} {
		if !strings.Contains(out, want) {
			t.Errorf("JSON output is missing %q:\n%s", want, out)
		}
	}
}

func TestCLIPlayers_EmptyDatabase(t *testing.T) {
	db := NewDatabase()
	if err := db.SetupDatabase(":memory:"); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	defer db.Close()
	cli := &CLI{db: db}

	out := captureStdout(t, func() {
		if err := cli.showPlayerTable(StatsFilter{DecisionType: -1}, "text"); err != nil {
			t.Fatalf("showPlayerTable on an empty database: %v", err)
		}
	})
	if !strings.Contains(out, "No player") {
		t.Errorf("an empty database should say so plainly, got:\n%s", out)
	}
}
