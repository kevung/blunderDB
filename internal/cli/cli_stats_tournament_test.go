package cli

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

func TestCLI_StatsTournament(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	res, err := RawConn(cli.db).Exec(`INSERT INTO tournament (name, date) VALUES ('Open', '2025-06-10')`)
	if err != nil {
		t.Fatalf("insert tournament: %v", err)
	}
	tid, _ := res.LastInsertId()
	for i, opp := range []string{"Bob", "Carol"} {
		m := createMatch(t, cli.db, "Alice", opp, "2025-06-10", 7, tid)
		g := createGame(t, cli.db, m)
		insertStatsFixtureRow(t, cli.db, m, g, 20+10*i, 0, 0, 1)
		insertStatsFixtureRow(t, cli.db, m, g, 5, 0, 1, 2)
	}

	out := captureStdout(t, func() {
		if err := cli.Run([]string{"stats", "tournament", "--db", dbPath, "--id", strconv.FormatInt(tid, 10), "--format", "json"}); err != nil {
			t.Fatalf("stats tournament: %v", err)
		}
	})
	var review struct {
		Player  string `json:"player"`
		Matches int    `json:"matches"`
		Rounds  []struct {
			Opponent string `json:"opponent"`
		} `json:"rounds"`
	}
	if err := json.Unmarshal([]byte(out), &review); err != nil {
		t.Fatalf("decode: %v\n%s", err, out)
	}
	if review.Player != "Alice" || review.Matches != 2 || len(review.Rounds) != 2 || review.Rounds[1].Opponent != "Carol" {
		t.Errorf("review: %+v", review)
	}

	text := captureStdout(t, func() {
		if err := cli.Run([]string{"stats", "tournament", "--db", dbPath, "--id", strconv.FormatInt(tid, 10)}); err != nil {
			t.Fatalf("stats tournament: %v", err)
		}
	})
	for _, want := range []string{"Alice, 2 matches", "Usual level unknown", "By round:", "By pace:"} {
		if !strings.Contains(text, want) {
			t.Errorf("text output lacks %q:\n%s", want, text)
		}
	}
	if err := cli.Run([]string{"stats", "tournament", "--db", dbPath}); err == nil {
		t.Errorf("a review without --id is an error")
	}
	if err := cli.Run([]string{"stats", "tournament", "--db", dbPath, "--id", "999"}); err == nil {
		t.Errorf("an unknown tournament is an error")
	}
}
