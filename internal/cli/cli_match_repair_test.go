package cli

import (
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

func TestCLI_MatchRepair(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	if err := cli.Run([]string{"import", "--db", dbPath, "--type", "match", "--file", testdataPath("test.xg")}); err != nil {
		t.Fatalf("import: %v", err)
	}
	for _, args := range [][]string{
		{"match", "repair", "--db", dbPath, "--id", "1", "--dry-run"},
		{"match", "repair", "--db", dbPath, "--list", "--format", "json"},
		{"match", "repair", "--db", dbPath, "--id", "1"},
	} {
		if err := cli.Run(args); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	rep, err := cli.db.RepairMatch(1, true)
	if err != nil || rep.Differs() {
		t.Errorf("repaired match: %+v, %v", rep, err)
	}

	bare := createMatch(t, cli.db, "A", "B", "2025-01-01", 5, 0)
	if _, err := cli.db.RepairMatch(bare, true); !errors.Is(err, ingest.ErrRepairNoSource) {
		t.Errorf("match without a source: err = %v, want ErrRepairNoSource", err)
	}
	for _, args := range [][]string{
		{"match", "repair", "--db", dbPath},
		{"match", "repair", "--db", dbPath, "--id", "1", "--list"},
	} {
		if err := cli.Run(args); err == nil {
			t.Errorf("%v: want an error", args)
		}
	}
}
