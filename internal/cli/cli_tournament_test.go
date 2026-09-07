package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The `tournament` sub-command (issue #395).
//
// What the software can do, it must be able to do without a graphical interface. What is held
// here is that each sub-command works on a plain database file — no Wails, no interface — and
// that `verify` EXITS IN ERROR when a warning remains, which is the whole point of an
// after-the-fact check.

// directedTournamentCLI creates a directed tournament in the CLI's database and returns its id.
func directedTournamentCLI(t *testing.T, cli *CLI, entrants int) int64 {
	t.Helper()
	tID, err := cli.db.CreateTournament("Open de Lyon", "2026-09-12", "Lyon")
	if err != nil {
		t.Fatalf("CreateTournament: %v", err)
	}
	cfg := `{"name":"Open de Lyon","tables":{"count":8},
		"phases":[{"kind":"bracket","length":5}]}`
	if err := cli.db.CreateDirection(tID, cfg, 7); err != nil {
		t.Fatalf("CreateDirection: %v", err)
	}
	var players []string
	for i := 0; i < entrants; i++ {
		id := string(rune('a' + i))
		players = append(players, `{"id":"`+id+`","name":"Joueur `+id+`"}`)
	}
	if err := cli.db.EnterParticipants(tID, "["+strings.Join(players, ",")+"]"); err != nil {
		t.Fatalf("EnterParticipants: %v", err)
	}
	return tID
}

func TestCLI_TournamentList(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)

	out := captureStdout(t, func() {
		if err := cli.Run([]string{"tournament", "list", "--db", dbPath}); err != nil {
			t.Fatalf("tournament list: %v", err)
		}
	})
	if !strings.Contains(out, "ID") {
		t.Errorf("expected a header on an empty database, got:\n%s", out)
	}

	tID := directedTournamentCLI(t, cli, 8)
	out = captureStdout(t, func() {
		if err := cli.Run([]string{"tournament", "list", "--db", dbPath, "--format", "json"}); err != nil {
			t.Fatalf("tournament list json: %v", err)
		}
	})
	var rows []struct {
		TournamentID int64  `json:"tournamentId"`
		State        string `json:"state"`
	}
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("json: %v\n%s", err, out)
	}
	if len(rows) != 1 || rows[0].TournamentID != tID {
		t.Fatalf("the directed tournament is not listed: %+v", rows)
	}
}

// verify says nothing and exits zero on a sound direction.
func TestCLI_TournamentVerifyIsQuietWhenSound(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	tID := directedTournamentCLI(t, cli, 8)

	out := captureStdout(t, func() {
		if err := cli.Run([]string{"tournament", "verify", "--db", dbPath, "--id", itoa64(tID)}); err != nil {
			t.Fatalf("verify on a sound direction: %v", err)
		}
	})
	if !strings.Contains(out, "no warning") {
		t.Errorf("expected a clean report, got:\n%s", out)
	}
}

// And it EXITS IN ERROR when a warning remains, printing it: a script that runs this over a
// season's databases wants a status, not a line to grep.
func TestCLI_TournamentVerifyFailsOnAWarning(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	tID := directedTournamentCLI(t, cli, 8)

	// Play the quarter-finals and the semi-finals, then correct a quarter: the bracket is now
	// out of tune, and the engine says so.
	playRoundsCLI(t, cli, tID, 2)
	finished, err := cli.db.FinishedMatches(tID, 40)
	if err != nil {
		t.Fatal(err)
	}
	m := finished[len(finished)-1]
	entries, err := cli.db.History(tID, "", m.MatchID)
	if err != nil {
		t.Fatal(err)
	}
	won := ""
	for _, e := range entries {
		if e.Winner != "" {
			won = e.Winner
		}
	}
	other := m.A
	if won == m.A {
		other = m.B
	}
	if _, err := cli.db.CorrectResult(tID, m.MatchID, other, 0, 0, ""); err != nil {
		t.Fatal(err)
	}

	var out string
	var runErr error
	out = captureStdout(t, func() {
		runErr = cli.Run([]string{"tournament", "verify", "--db", dbPath, "--id", itoa64(tID)})
	})
	if runErr == nil {
		t.Fatal("verify returned no error while a warning remains")
	}
	if !strings.Contains(out, "bracket_wrong_players") {
		t.Errorf("the warning was not printed:\n%s", out)
	}
}

func TestCLI_TournamentStandingsAndPageAndExport(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	tID := directedTournamentCLI(t, cli, 8)
	playRoundsCLI(t, cli, tID, 3)

	csv := captureStdout(t, func() {
		if err := cli.Run([]string{"tournament", "standings", "--db", dbPath, "--id", itoa64(tID)}); err != nil {
			t.Fatalf("standings: %v", err)
		}
	})
	if len(strings.Split(strings.TrimSpace(csv), "\n")) < 9 {
		t.Errorf("expected a header and eight players:\n%s", csv)
	}

	// The page on standard output, so it can be piped.
	page := captureStdout(t, func() {
		if err := cli.Run([]string{"tournament", "page", "--db", dbPath, "--id", itoa64(tID)}); err != nil {
			t.Fatalf("page: %v", err)
		}
	})
	if !strings.Contains(page, "<html") || !strings.Contains(page, "Nicolas Harmand") {
		t.Errorf("the page is not the display page:\n%s", page[:min(300, len(page))])
	}

	// And into a folder, which becomes the direction's.
	dir := t.TempDir()
	out := captureStdout(t, func() {
		if err := cli.Run([]string{"tournament", "page", "--db", dbPath, "--id", itoa64(tID), "--out", dir}); err != nil {
			t.Fatalf("page --out: %v", err)
		}
	})
	written := strings.TrimSpace(out)
	if filepath.Dir(written) != dir {
		t.Errorf("the page went to %q instead of %q", written, dir)
	}
	if _, err := os.Stat(written); err != nil {
		t.Errorf("the page was not written: %v", err)
	}

	// The raw journal: replayable by the engine's own tools, so it must be the events
	// themselves and not a view of them.
	journal := captureStdout(t, func() {
		if err := cli.Run([]string{"tournament", "export", "--db", dbPath, "--id", itoa64(tID)}); err != nil {
			t.Fatalf("export: %v", err)
		}
	})
	var events []map[string]any
	if err := json.Unmarshal([]byte(journal), &events); err != nil {
		t.Fatalf("the journal is not JSON: %v\n%s", err, journal[:min(300, len(journal))])
	}
	if len(events) == 0 || events[0]["kind"] != "created" {
		t.Errorf("the journal does not start at the creation: %+v", events[0])
	}
}

// Every sub-command refuses to guess: no --db, no --id, no work.
func TestCLI_TournamentAsksForItsArguments(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	for _, args := range [][]string{
		{"tournament"},
		{"tournament", "nowhere"},
		{"tournament", "verify", "--db", dbPath},
		{"tournament", "standings", "--db", dbPath},
		{"tournament", "page", "--db", dbPath},
		{"tournament", "export", "--db", dbPath},
	} {
		captureStdout(t, func() {
			if err := cli.Run(args); err == nil {
				t.Errorf("%v returned no error", args)
			}
		})
	}
}

// playRoundsCLI launches everything launchable and enters a result for every running match,
// n times. The lower identifier wins, so the run is deterministic.
func playRoundsCLI(t *testing.T, cli *CLI, tID int64, rounds int) {
	t.Helper()
	for r := 0; r < rounds; r++ {
		for i := 0; i < 4; i++ {
			v, err := cli.db.ConfirmAllProposals(tID)
			if err != nil {
				t.Fatal(err)
			}
			if len(v.Running) > 0 {
				break
			}
		}
		v, err := cli.db.GetDirection(tID)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range v.Running {
			w := m.A
			if string(m.B) < string(m.A) {
				w = m.B
			}
			if _, err := cli.db.EnterResult(tID, string(m.ID), string(w), m.Length, 0, ""); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func itoa64(v int64) string { return strconv.FormatInt(v, 10) }
