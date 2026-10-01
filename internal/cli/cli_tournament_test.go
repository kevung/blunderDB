package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The `tournament` sub-command: each sub-command works on a plain database
// file with no GUI, and `verify` exits in error while a warning remains.

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

// `tournament list` names the Rencontre a directed tournament plays in (ADR-0056 §6).
func TestCLI_TournamentListNamesTheRencontre(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	tID := directedTournamentCLI(t, cli, 8)

	r, err := cli.db.CreateRencontre("Festival", "2026-10-03", "2026-10-04", 8)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cli.db.AttachToRencontre(tID, r.ID); err != nil {
		t.Fatal(err)
	}

	text := captureStdout(t, func() {
		if err := cli.Run([]string{"tournament", "list", "--db", dbPath}); err != nil {
			t.Fatalf("tournament list: %v", err)
		}
	})
	if !strings.Contains(text, "Festival") {
		t.Errorf("tournament list does not name the Rencontre:\n%s", text)
	}

	out := captureStdout(t, func() {
		if err := cli.Run([]string{"tournament", "list", "--db", dbPath, "--format", "json"}); err != nil {
			t.Fatalf("tournament list json: %v", err)
		}
	})
	var rows []struct {
		RencontreName string `json:"rencontreName"`
	}
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("json: %v\n%s", err, out)
	}
	if len(rows) != 1 || rows[0].RencontreName != "Festival" {
		t.Fatalf("json rows do not name the Rencontre: %+v", rows)
	}
}

// `tournament page --rencontre` writes the room's wall page, mutually exclusive with --id
// (ADR-0056 §6).
func TestCLI_TournamentPageRencontre(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	a := directedTournamentCLI(t, cli, 8)

	r, err := cli.db.CreateRencontre("Festival", "2026-10-03", "2026-10-04", 8)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cli.db.AttachToRencontre(a, r.ID); err != nil {
		t.Fatal(err)
	}

	if err := cli.Run([]string{"tournament", "page", "--db", dbPath}); err == nil {
		t.Error("neither --id nor --rencontre should be rejected")
	}
	if err := cli.Run([]string{"tournament", "page", "--db", dbPath, "--id", itoa64(a), "--rencontre", itoa64(r.ID)}); err == nil {
		t.Error("both --id and --rencontre should be rejected")
	}

	page := captureStdout(t, func() {
		if err := cli.Run([]string{"tournament", "page", "--db", dbPath, "--rencontre", itoa64(r.ID)}); err != nil {
			t.Fatalf("page --rencontre: %v", err)
		}
	})
	if !strings.Contains(page, "<html") || !strings.Contains(page, "Nicolas Harmand") || !strings.Contains(page, "Open de Lyon") {
		t.Errorf("the wall page is missing its shell or its member event:\n%s", page[:min(300, len(page))])
	}

	dir := t.TempDir()
	out := captureStdout(t, func() {
		if err := cli.Run([]string{"tournament", "page", "--db", dbPath, "--rencontre", itoa64(r.ID), "--out", dir}); err != nil {
			t.Fatalf("page --rencontre --out: %v", err)
		}
	})
	written := strings.TrimSpace(out)
	if filepath.Base(written) != "index.html" || filepath.Dir(written) != dir {
		t.Errorf("the wall page went to %q instead of %s/index.html", written, dir)
	}
	if _, err := os.Stat(written); err != nil {
		t.Errorf("the wall page was not written: %v", err)
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

// `tournament move` is the table grid's drag-and-drop: a free table moves the match, a taken
// one swaps the two, an out-of-service one is refused.
func TestCLI_TournamentMove(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	tID := directedTournamentCLI(t, cli, 6)
	if _, err := cli.db.StartMatchManually(tID, "a", "b", 0, 1); err != nil {
		t.Fatal(err)
	}
	v, err := cli.db.StartMatchManually(tID, "c", "d", 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	tables := map[string]int{}
	for _, m := range v.Running {
		tables[string(m.ID)] = m.Table
	}
	var m1, m2 string
	for id, tb := range tables {
		if tb == 1 {
			m1 = id
		} else {
			m2 = id
		}
	}
	run := func(table string) error {
		return cli.Run([]string{"tournament", "move", "--db", dbPath, "--id", itoa64(tID), "--match", m1, "--table", table})
	}
	if err := run("2"); err != nil {
		t.Fatalf("swap: %v", err)
	}
	got, err := cli.db.GetDirection(tID)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range got.Running {
		want := map[string]int{m1: 2, m2: 1}[string(m.ID)]
		if m.Table != want {
			t.Errorf("match %s on table %d, want %d", m.ID, m.Table, want)
		}
	}
	if err := run("5"); err != nil {
		t.Fatalf("move to a free table: %v", err)
	}
	if err := cli.Run([]string{"tournament", "move", "--db", dbPath, "--id", itoa64(tID), "--match", m1}); err == nil {
		t.Error("move without --table must fail")
	}
	if err := cli.Run([]string{"tournament", "move", "--db", dbPath, "--id", itoa64(tID), "--match", "nope", "--table", "3"}); err == nil {
		t.Error("an unknown match must fail")
	}
}
