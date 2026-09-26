package cli

import (
	"os"
	"strings"
	"testing"
)

// TestCLI_Vacuum_ReclaimsSpace: create, fill, delete, vacuum, file shrinks.
// Padding rows go straight through SQL so the size delta is large and
// deterministic.
func TestCLI_Vacuum_ReclaimsSpace(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)

	blob := strings.Repeat("x", 2000)
	tx, err := RawConn(cli.db).Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	stmt, err := tx.Prepare(`INSERT INTO position (state) VALUES (?)`)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	for i := 0; i < 2000; i++ {
		if _, err := stmt.Exec(blob); err != nil {
			stmt.Close()
			t.Fatalf("insert padding %d: %v", i, err)
		}
	}
	stmt.Close()
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit padding: %v", err)
	}
	if _, err := RawConn(cli.db).Exec(`DELETE FROM position`); err != nil {
		t.Fatalf("delete padding: %v", err)
	}

	sizeBeforeCLI, err := os.Stat(dbPath)
	if err != nil {
		t.Fatalf("stat before: %v", err)
	}

	out := captureStdout(t, func() {
		if err := cli.Run([]string{"vacuum", "--db", dbPath}); err != nil {
			t.Fatalf("vacuum: %v", err)
		}
	})

	if !strings.Contains(out, "Reclaimed:") {
		t.Errorf("vacuum output missing a reclaimed-space report:\n%s", out)
	}

	sizeAfterCLI, err := os.Stat(dbPath)
	if err != nil {
		t.Fatalf("stat after: %v", err)
	}
	if sizeAfterCLI.Size() >= sizeBeforeCLI.Size() {
		t.Errorf("file did not shrink: before=%d after=%d", sizeBeforeCLI.Size(), sizeAfterCLI.Size())
	}
}

// TestCLI_Vacuum_MissingDBFlag guards the required-flag validation shared
// with the other commands.
func TestCLI_Vacuum_MissingDBFlag(t *testing.T) {
	t.Parallel()
	cli := setupCLI(t)
	err := cli.Run([]string{"vacuum"})
	if err == nil {
		t.Fatal("expected error when --db is missing")
	}
}
