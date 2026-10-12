package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func seedStatsViews(t *testing.T) (*CLI, string) {
	t.Helper()
	cli, dbPath := setupCLIWithDB(t)
	matchID := createMatch(t, cli.db, "Alice", "Bob", "2025-01-01", 5, 0)
	gameID := createGame(t, cli.db, matchID)
	insertStatsFixtureRow(t, cli.db, matchID, gameID, 200, 0, 0, 1)
	insertStatsFixtureRow(t, cli.db, matchID, gameID, 300, 1, 0, 2)
	return cli, dbPath
}

func TestCLI_StatsProgressionAndBreakdownJSON(t *testing.T) {
	cli, dbPath := seedStatsViews(t)
	for _, tc := range []struct {
		sub  string
		keys []string
	}{
		{"progression", []string{"PerMatch", "PerTournament", "PRRolling"}},
		{"breakdown", []string{"PerPhase", "PerGameType", "PerTag", "PerScore", "CubeActionBreakdown", "CubeDirections"}},
	} {
		out := captureStdout(t, func() {
			if err := cli.Run([]string{"stats", tc.sub, "--db", dbPath, "--format", "json"}); err != nil {
				t.Fatalf("stats %s: %v", tc.sub, err)
			}
		})
		var got map[string]json.RawMessage
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("stats %s: not JSON: %v\n%s", tc.sub, err, out)
		}
		for _, k := range tc.keys {
			if _, ok := got[k]; !ok {
				t.Errorf("stats %s lacks %q", tc.sub, k)
			}
		}
	}
	for _, sub := range []string{"progression", "breakdown"} {
		out := captureStdout(t, func() {
			if err := cli.Run([]string{"stats", sub, "--db", dbPath}); err != nil {
				t.Fatalf("stats %s text: %v", sub, err)
			}
		})
		if strings.TrimSpace(out) == "" {
			t.Errorf("stats %s text printed nothing", sub)
		}
	}
}

func TestCLI_StatsReportWritesASelfContainedFile(t *testing.T) {
	cli, dbPath := seedStatsViews(t)
	out := filepath.Join(tempDir(t), "rapport.html")
	if err := cli.Run([]string{"stats", "report", "--db", dbPath, "--html", "--lang", "fr", "--output", out}); err != nil {
		t.Fatalf("stats report: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	html := string(data)
	for _, want := range []string{"<!doctype html>", `lang="fr"`, "Rapport blunderDB", "Alice"} {
		if !strings.Contains(html, want) {
			t.Errorf("report lacks %q", want)
		}
	}
	if strings.Contains(html, "<script") {
		t.Error("report holds a script")
	}
	if err := cli.Run([]string{"stats", "report", "--db", dbPath}); err == nil {
		t.Error("stats report without --html: want an error")
	}
}

func TestCLI_PlayersMergeAndSwap(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	matchID := createMatch(t, cli.db, "Doe J.", "Bob", "2025-01-01", 5, 0)

	if err := cli.Run([]string{"players", "merge", "--db", dbPath, "--into", "John Doe", "Doe J."}); err != nil {
		t.Fatalf("players merge: %v", err)
	}
	aliases, err := cli.db.ListAliases("player")
	if err != nil {
		t.Fatalf("ListAliases: %v", err)
	}
	found := false
	for _, a := range aliases {
		found = found || (a.Alias == "Doe J." && a.Canonical == "John Doe")
	}
	if !found {
		t.Errorf("merge recorded no alias Doe J. -> John Doe: %+v", aliases)
	}

	before, err := cli.db.GetMatchByID(matchID)
	if err != nil {
		t.Fatalf("LoadMatch: %v", err)
	}
	if err := cli.Run([]string{"players", "swap", "--db", dbPath, "1"}); err != nil {
		t.Fatalf("players swap: %v", err)
	}
	after, err := cli.db.GetMatchByID(matchID)
	if err != nil {
		t.Fatalf("LoadMatch: %v", err)
	}
	if after.Player1Name != before.Player2Name || after.Player2Name != before.Player1Name {
		t.Errorf("swap left %q/%q, was %q/%q", after.Player1Name, after.Player2Name, before.Player1Name, before.Player2Name)
	}
	for _, args := range [][]string{
		{"players", "merge", "--db", dbPath, "A"},
		{"players", "swap", "--db", dbPath, "x"},
	} {
		if err := cli.Run(args); err == nil {
			t.Errorf("%v: want an error", args)
		}
	}
}

func TestCLI_PlayersMain(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	second := createMatch(t, cli.db, "Bob", "Doe J.", "2025-01-01", 5, 0)
	first := createMatch(t, cli.db, "Doe J.", "Carol", "2025-01-02", 5, 0)
	if err := cli.db.SetAlias("player", "Doe J.", "John Doe"); err != nil {
		t.Fatalf("SetAlias: %v", err)
	}

	if err := cli.Run([]string{"players", "main", "--db", dbPath, "--dry-run", "John Doe"}); err != nil {
		t.Fatalf("players main --dry-run: %v", err)
	}
	if m, _ := cli.db.GetMatchByID(second); m.Player1Name != "Bob" {
		t.Errorf("the dry run swapped match %d", second)
	}
	if err := cli.Run([]string{"players", "main", "--db", dbPath, "John Doe"}); err != nil {
		t.Fatalf("players main: %v", err)
	}
	if m, _ := cli.db.GetMatchByID(second); m.Player1Name != "Doe J." || m.Player2Name != "Bob" {
		t.Errorf("match %d: %q / %q, want the Main player first", second, m.Player1Name, m.Player2Name)
	}
	if m, _ := cli.db.GetMatchByID(first); m.Player1Name != "Doe J." {
		t.Errorf("match %d, already the Main player's first, was swapped", first)
	}
	if name, err := cli.db.MainPlayer(); err != nil || name != "John Doe" {
		t.Errorf("MainPlayer = %q, %v", name, err)
	}
	if ids, err := cli.db.PlayerMatchIDs("John Doe"); err != nil || len(ids) != 2 {
		t.Errorf("PlayerMatchIDs = %v, %v; want both matches", ids, err)
	}
	if err := cli.Run([]string{"players", "main", "--db", dbPath, "--clear"}); err != nil {
		t.Fatalf("players main --clear: %v", err)
	}
	if name, _ := cli.db.MainPlayer(); name != "" {
		t.Errorf("MainPlayer after --clear = %q", name)
	}
	if m, _ := cli.db.GetMatchByID(second); m.Player1Name != "Doe J." {
		t.Errorf("--clear swapped match %d back", second)
	}
	if err := cli.Run([]string{"players", "main", "--db", dbPath, "--clear", "X"}); err == nil {
		t.Error("--clear with a name: want an error")
	}
}
