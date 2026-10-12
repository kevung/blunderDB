package cli

import (
	"strings"
	"testing"
)

// TestCLI_Maintenance_RunThenStatus: a run is reported with its outcome, and
// the status that follows reads it back as the line's last run; the
// operation may come after the flags too.
func TestCLI_Maintenance_RunThenStatus(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	out := captureStdout(t, func() {
		if err := cli.Run([]string{"maintenance", "run", "--db", dbPath, "--format", "json", "analysis.legacy"}); err != nil {
			t.Fatalf("maintenance run: %v", err)
		}
	})
	if !strings.Contains(out, `"outcome": "done"`) {
		t.Errorf("run report = %q; want outcome done", out)
	}
	out = captureStdout(t, func() {
		if err := cli.Run([]string{"maintenance", "status", "--db", dbPath}); err != nil {
			t.Fatalf("maintenance status: %v", err)
		}
	})
	for _, want := range []string{"analysis.legacy", "done, 0 changed", "analysis.missing", "(count on demand)"} {
		if !strings.Contains(out, want) {
			t.Errorf("status = %q; want %q", out, want)
		}
	}
}

func TestCLI_Maintenance_RefusesWithoutOperation(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	if err := cli.Run([]string{"maintenance", "count", "--db", dbPath}); err == nil || !strings.Contains(err.Error(), "missing operation") {
		t.Errorf("count without an operation: %v", err)
	}
}
