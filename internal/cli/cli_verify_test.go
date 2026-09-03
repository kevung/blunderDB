package cli

import (
	"context"
	"strings"
	"testing"
)

// TestCLI_Verify_ReportsOrphans: verify counts and names the child rows whose
// parent is gone (issue #157). The orphans are planted through a dedicated
// connection with foreign keys switched off — the only way to create them
// now that every pooled connection enforces them.
func TestCLI_Verify_ReportsOrphans(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)

	clean := captureStdout(t, func() {
		if err := cli.Run([]string{"verify", "--db", dbPath}); err != nil {
			t.Fatalf("verify (clean): %v", err)
		}
	})
	if !strings.Contains(clean, "Orphaned rows: none") {
		t.Errorf("clean database should report no orphans:\n%s", clean)
	}

	ctx := context.Background()
	conn, err := cli.db.Conn().Conn(ctx)
	if err != nil {
		t.Fatalf("dedicated connection: %v", err)
	}
	for _, s := range []string{
		`PRAGMA foreign_keys = OFF`,
		`INSERT INTO game (id, match_id, game_number) VALUES (9001, 424242, 1)`,
		`INSERT INTO move (id, game_id, move_number, move_type) VALUES (9002, 9001, 1, 'checker')`,
		`INSERT INTO move (id, game_id, move_number, move_type) VALUES (9003, 424242, 2, 'checker')`,
		`INSERT INTO move_analysis (id, move_id, analysis_type) VALUES (9004, 424242, 'checker')`,
		`INSERT INTO move_analysis (id, move_id, analysis_type) VALUES (9005, 424243, 'checker')`,
		`INSERT INTO analysis (position_id, data) VALUES (424242, X'00')`,
		`PRAGMA foreign_keys = ON`,
	} {
		if _, err := conn.ExecContext(ctx, s); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}
	conn.Close()

	out := captureStdout(t, func() {
		if err := cli.Run([]string{"verify", "--db", dbPath}); err != nil {
			t.Fatalf("verify (orphans): %v", err)
		}
	})
	for _, want := range []string{
		"Games without match: 1",
		"Moves without game: 1",
		"Move analyses without move: 2",
		"Analyses without position: 1",
		"WARNING: 5 orphaned row(s) found",
		"Verification complete!",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("verify output lacks %q:\n%s", want, out)
		}
	}
}

// TestCLI_Verify_ReportsSchemaDrift (issue #177): what the open could not add
// against the reference DDL is printed, not only logged. A UNIQUE index that
// duplicate rows keep EnsureSchema from rebuilding is the reproducible case.
func TestCLI_Verify_ReportsSchemaDrift(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)

	clean := captureStdout(t, func() {
		if err := cli.Run([]string{"verify", "--db", dbPath}); err != nil {
			t.Fatalf("verify (clean): %v", err)
		}
	})
	if !strings.Contains(clean, "Schema: matches the reference DDL") {
		t.Errorf("clean database should report no schema drift:\n%s", clean)
	}

	for _, s := range []string{
		`DROP INDEX idx_match_canonical`,
		`INSERT INTO match (player1_name, player2_name, canonical_hash) VALUES ('a', 'b', 'same')`,
		`INSERT INTO match (player1_name, player2_name, canonical_hash) VALUES ('c', 'd', 'same')`,
	} {
		if _, err := cli.db.Conn().Exec(s); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}

	out := captureStdout(t, func() {
		if err := cli.Run([]string{"verify", "--db", dbPath}); err != nil {
			t.Fatalf("verify (drift): %v", err)
		}
	})
	for _, want := range []string{
		"Missing indexes: idx_match_canonical",
		"WARNING: 1 schema element(s) missing",
		"Verification complete!",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("verify output lacks %q:\n%s", want, out)
		}
	}
}

// TestCLI_Verify_ReportsCounterDrift (issue #185): match.game_count and
// game.move_count are written once, at import, from what the source file held
// and are what the match list displays. verify recomputes them from the rows
// and says how far apart they are — without rewriting either.
func TestCLI_Verify_ReportsCounterDrift(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)

	clean := captureStdout(t, func() {
		if err := cli.Run([]string{"verify", "--db", dbPath}); err != nil {
			t.Fatalf("verify (clean): %v", err)
		}
	})
	for _, want := range []string{
		"Counters: game_count and move_count agree with the rows",
		"Constraints: every row satisfies the current DDL",
	} {
		if !strings.Contains(clean, want) {
			t.Errorf("clean database should report %q:\n%s", want, clean)
		}
	}

	// A match that claims four games and holds one, whose single game claims
	// no move and holds one.
	for _, s := range []string{
		`INSERT INTO match (id, player1_name, player2_name, game_count) VALUES (7001, 'a', 'b', 4)`,
		`INSERT INTO game (id, match_id, game_number, move_count) VALUES (7002, 7001, 1, 0)`,
		`INSERT INTO move (id, game_id, move_number, move_type) VALUES (7003, 7002, 1, 'checker')`,
	} {
		if _, err := cli.db.Conn().Exec(s); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}

	out := captureStdout(t, func() {
		if err := cli.Run([]string{"verify", "--db", dbPath}); err != nil {
			t.Fatalf("verify (drift): %v", err)
		}
	})
	for _, want := range []string{
		"Matches whose game_count is wrong: 1 (worst gap: 3 game(s))",
		"Games whose move_count is wrong: 1 (worst gap: 1 move(s))",
		"Nothing is rewritten here.",
		"Verification complete!",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("verify output lacks %q:\n%s", want, out)
		}
	}

	// And nothing was rewritten: the counter still says what the source said.
	var gameCount int
	if err := cli.db.Conn().QueryRow(`SELECT game_count FROM match WHERE id = 7001`).Scan(&gameCount); err != nil {
		t.Fatal(err)
	}
	if gameCount != 4 {
		t.Errorf("verify rewrote game_count to %d; it must only report", gameCount)
	}
}
