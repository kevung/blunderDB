package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// decisionCountInOutput returns true when any "Decisions:" line in out ends
// with the expected number as its last whitespace-separated field.
func decisionCountInOutput(out, want string) bool {
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "Decisions:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 && fields[len(fields)-1] == want {
				return true
			}
		}
	}
	return false
}

// setupCLIStats returns a CLI with an in-memory DB pre-populated with two
// checker decisions (errMP 200 and 50) and one cube decision (errMP 300)
// belonging to a single match.
func setupCLIStats(t *testing.T) *CLI {
	t.Helper()
	db := NewDatabase()
	if err := db.SetupDatabase(":memory:"); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	t.Cleanup(func() {
		if db.db != nil {
			db.db.Close()
		}
	})

	matchID := createMatch(t, db, "Alice", "Bob", "2025-01-01", 5, 0)
	gameID := createGame(t, db, matchID)
	insertStatsFixtureRow(t, db, matchID, gameID, 200, 0, 0, 1) // checker, player Alice
	insertStatsFixtureRow(t, db, matchID, gameID, 50, 0, 0, 2)  // checker, player Alice
	insertStatsFixtureRow(t, db, matchID, gameID, 300, 1, 0, 3) // cube, player Alice

	return &CLI{db: db, cfg: NewConfig()}
}

// ── TestCLIStats_TextFormat ──────────────────────────────────────────────────

func TestCLIStats_TextFormat(t *testing.T) {
	cli := setupCLIStats(t)

	out := captureStdout(t, func() {
		if err := cli.showStats(StatsFilter{DecisionType: -1}, "pr", "text", 10); err != nil {
			t.Fatalf("showStats: %v", err)
		}
	})

	sections := []string{
		"=== blunderDB Statistics ===",
		"── Totals ──",
		"── PR ──",
		"── Rolling PR ──",
		"── Top",
		"Blunders",
	}
	for _, s := range sections {
		if !strings.Contains(out, s) {
			t.Errorf("expected section %q not found in output:\n%s", s, out)
		}
	}

	// Decisions count: 3 rows inserted
	if !strings.Contains(out, "3") {
		t.Errorf("expected 3 decisions in output:\n%s", out)
	}
}

// ── TestCLIStats_JSONFormat ──────────────────────────────────────────────────

func TestCLIStats_JSONFormat(t *testing.T) {
	cli := setupCLIStats(t)

	out := captureStdout(t, func() {
		if err := cli.showStats(StatsFilter{DecisionType: -1}, "pr", "json", 10); err != nil {
			t.Fatalf("showStats json: %v", err)
		}
	})

	var result StatsResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("JSON unmarshal failed: %v\noutput:\n%s", err, out)
	}

	// Sanity: we have 3 decisions
	if result.Totals.NumDecisions != 3 {
		t.Errorf("want 3 decisions, got %d", result.Totals.NumDecisions)
	}
	if result.PRGlobal <= 0 {
		t.Errorf("want PRGlobal > 0, got %f", result.PRGlobal)
	}
	// JSON must carry key fields
	raw := string([]byte(out))
	for _, key := range []string{`"totals"`, `"pr_global"`, `"pr_rolling"`, `"top_blunders"`} {
		if !strings.Contains(raw, key) {
			t.Errorf("JSON missing key %q:\n%s", key, raw)
		}
	}
}

// ── TestCLIStats_MWCMetric ───────────────────────────────────────────────────

func TestCLIStats_MWCMetric(t *testing.T) {
	cli := setupCLIStats(t)

	out := captureStdout(t, func() {
		if err := cli.showStats(StatsFilter{DecisionType: -1}, "mwc", "text", 10); err != nil {
			t.Fatalf("showStats mwc: %v", err)
		}
	})

	if !strings.Contains(out, "MWC") {
		t.Errorf("expected MWC label in output:\n%s", out)
	}
	if strings.Contains(out, "── PR ──") {
		t.Errorf("unexpected PR section in MWC output:\n%s", out)
	}
}

// ── TestCLIStats_PlayerFilter ────────────────────────────────────────────────

func TestCLIStats_PlayerFilter(t *testing.T) {
	cli := setupCLIStats(t)

	// Filter for a player who does not exist
	out := captureStdout(t, func() {
		filter := StatsFilter{PlayerName: "Nobody", DecisionType: -1}
		if err := cli.showStats(filter, "pr", "text", 10); err != nil {
			t.Fatalf("showStats with player filter: %v", err)
		}
	})
	// Result should show 0 decisions (tabwriter may use varying whitespace)
	if !decisionCountInOutput(out, "0") {
		t.Errorf("expected 0 decisions for unknown player:\n%s", out)
	}
}

// ── TestCLIStats_DecisionTypeFilter ─────────────────────────────────────────

func TestCLIStats_DecisionTypeFilter(t *testing.T) {
	cli := setupCLIStats(t)

	out := captureStdout(t, func() {
		// cube-only: only 1 cube decision inserted
		filter := StatsFilter{DecisionType: 1}
		if err := cli.showStats(filter, "pr", "text", 10); err != nil {
			t.Fatalf("showStats cube filter: %v", err)
		}
	})

	// Only 1 cube decision → Decisions: 1
	if !decisionCountInOutput(out, "1") {
		t.Errorf("expected 1 cube decision:\n%s", out)
	}
	if !strings.Contains(out, "cube only") {
		t.Errorf("expected 'cube only' label in output:\n%s", out)
	}
	// Cube action breakdown section should be present
	if !strings.Contains(out, "Cube Action Breakdown") {
		t.Errorf("expected Cube Action Breakdown section:\n%s", out)
	}
}

// ── TestCLIStats_BackwardCompat ──────────────────────────────────────────────

// TestCLIStats_BackwardCompat verifies that running via cli.Run with the
// legacy invocation (no stats flags) still produces a useful output.
func TestCLIStats_BackwardCompat(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	if _, err := cli.db.ImportXGMatch(testdataPath("test.xg")); err != nil {
		t.Fatalf("ImportXGMatch: %v", err)
	}
	// Close the DB so cli.Run can reopen it from the file
	cli.db.db.Close()
	cli.db.db = nil

	out := captureStdout(t, func() {
		err := cli.Run([]string{"list", "--db", dbPath, "--type", "stats"})
		if err != nil {
			t.Fatalf("cli.Run list --type stats: %v", err)
		}
	})

	if !bytes.Contains([]byte(out), []byte("=== blunderDB Statistics ===")) {
		t.Errorf("missing stats header in backward-compat output:\n%s", out)
	}
}

// ── TestCLIStats_FormatJSONViaCLIRun ────────────────────────────────────────

// TestCLIStats_FormatJSONViaCLIRun exercises the --format json flag end-to-end
// via cli.Run.
func TestCLIStats_FormatJSONViaCLIRun(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	if _, err := cli.db.ImportXGMatch(testdataPath("test.xg")); err != nil {
		t.Fatalf("ImportXGMatch: %v", err)
	}
	cli.db.db.Close()
	cli.db.db = nil

	out := captureStdout(t, func() {
		err := cli.Run([]string{"list", "--db", dbPath, "--type", "stats", "--format", "json"})
		if err != nil {
			t.Fatalf("cli.Run list --type stats --format json: %v", err)
		}
	})

	// cli.Run may print a "Connected to database:" prefix; extract JSON from the
	// first '{' onwards.
	idx := strings.Index(out, "{")
	if idx < 0 {
		t.Fatalf("no JSON object found in output:\n%s", out)
	}
	jsonPart := out[idx:]

	var result StatsResult
	if err := json.Unmarshal([]byte(jsonPart), &result); err != nil {
		t.Fatalf("JSON unmarshal failed: %v\noutput:\n%s", err, jsonPart)
	}
}
