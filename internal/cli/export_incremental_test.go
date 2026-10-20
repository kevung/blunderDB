package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// `export --incremental` into a folder writes a dated file there, carries the
// library the first time and nothing the second, and the plan of an
// incremental export agrees with what it then writes.
func TestExportIncrementalIntoAFolder(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	seedWholeLibrary(t, cli)
	folder := t.TempDir()

	run := func(args ...string) string {
		return captureStdout(t, func() {
			base := []string{"export", "--db", dbPath, "--type", "database", "--incremental", "--file", folder, "--format", "json"}
			if err := cli.Run(append(base, args...)); err != nil {
				t.Fatalf("export %v: %v", args, err)
			}
		})
	}

	var plan ingest.ExportPlan
	if err := json.Unmarshal([]byte(run("--plan")), &plan); err != nil {
		t.Fatal(err)
	}
	var first struct {
		File   string              `json:"file"`
		Report ingest.ExportReport `json:"report"`
	}
	if err := json.Unmarshal([]byte(run()), &first); err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(first.File) != folder || !strings.HasPrefix(filepath.Base(first.File), "sync-") {
		t.Fatalf("file %q, want a dated name inside %s", first.File, folder)
	}
	if first.Report.Positions == 0 || first.Report.Matches == 0 {
		t.Fatalf("first incremental export carries nothing: %+v", first.Report)
	}
	if m := plan.Mismatch(first.Report); m != "" {
		t.Errorf("plan and report disagree on %s", m)
	}
	if entries, _ := os.ReadDir(folder); len(entries) != 1 {
		t.Errorf("the folder holds %d entries, want the export alone", len(entries))
	}

	var second struct {
		Report ingest.ExportReport `json:"report"`
	}
	if err := json.Unmarshal([]byte(run()), &second); err != nil {
		t.Fatal(err)
	}
	if r := second.Report; r.Positions != 0 || r.Matches != 0 || r.Collections != 0 || r.Tournaments != 0 {
		t.Errorf("second incremental export = %+v, want empty", r)
	}
}
