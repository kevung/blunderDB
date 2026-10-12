package cli

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// `export --plan` prints the counts that the same export, run for real,
// reports in its JSON — and writes no file.
func TestExportPlanAgreesWithExportReport(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	seedWholeLibrary(t, cli)
	out := filepath.Join(t.TempDir(), "cli.db")

	var plan ingest.ExportPlan
	planJSON := captureStdout(t, func() {
		if err := cli.Run([]string{"export", "--db", dbPath, "--type", "database", "--plan", "--format", "json"}); err != nil {
			t.Fatalf("export --plan: %v", err)
		}
	})
	if err := json.Unmarshal([]byte(planJSON), &plan); err != nil {
		t.Fatalf("plan JSON %q: %v", planJSON, err)
	}
	if fileExists(out) {
		t.Fatal("--plan wrote a file")
	}

	var done struct {
		Report ingest.ExportReport `json:"report"`
	}
	doneJSON := captureStdout(t, func() {
		if err := cli.Run([]string{"export", "--db", dbPath, "--type", "database", "--file", out, "--format", "json"}); err != nil {
			t.Fatalf("export: %v", err)
		}
	})
	if err := json.Unmarshal([]byte(doneJSON), &done); err != nil {
		t.Fatalf("export JSON %q: %v", doneJSON, err)
	}
	if done.Report.Positions == 0 || done.Report.Matches == 0 {
		t.Fatalf("the report counts nothing: %+v", done.Report)
	}
	if m := plan.Mismatch(done.Report); m != "" {
		t.Errorf("plan and report disagree on %s", m)
	}

	var page ingest.PlannedIDs
	pageJSON := captureStdout(t, func() {
		if err := cli.Run([]string{"export", "--db", dbPath, "--type", "database", "--plan-ids", "matches", "--limit", "1", "--format", "json"}); err != nil {
			t.Fatalf("export --plan-ids: %v", err)
		}
	})
	if err := json.Unmarshal([]byte(pageJSON), &page); err != nil {
		t.Fatalf("plan-ids JSON %q: %v", pageJSON, err)
	}
	if page.Total != plan.Matches || len(page.IDs) != 1 {
		t.Errorf("matches page %+v, want 1 id of %d", page, plan.Matches)
	}
}

// --plan counts a database export; with a type that writes no database it is
// refused, never taken for an export.
func TestExportPlanRefusesOtherTypes(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	out := filepath.Join(t.TempDir(), "positions.txt")
	captureStdout(t, func() {
		if err := cli.Run([]string{"export", "--db", dbPath, "--type", "positions", "--file", out, "--plan"}); err == nil {
			t.Error("--plan with --type positions was accepted")
		}
	})
	if fileExists(out) {
		t.Error("--plan with --type positions wrote a file")
	}
}
