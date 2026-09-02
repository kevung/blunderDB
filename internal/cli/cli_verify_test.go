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
