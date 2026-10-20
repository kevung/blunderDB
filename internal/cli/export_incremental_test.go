package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
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

// An incremental export never replaces a file: the one there holds a delta the
// cursor already counts as sent, which the other machine may not have merged.
func TestExportIncrementalRefusesAnExistingFile(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	seedWholeLibrary(t, cli)
	file := filepath.Join(t.TempDir(), "inc.db")
	export := func() error {
		var err error
		captureStdout(t, func() {
			err = cli.Run([]string{"export", "--db", dbPath, "--type", "database", "--incremental", "--file", file})
		})
		return err
	}
	if err := export(); err != nil {
		t.Fatalf("first export to a new file: %v", err)
	}
	before, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if err := export(); err == nil || !strings.Contains(err.Error(), "never overwrites") {
		t.Fatalf("second export to the same file: err = %v, want a refusal", err)
	}
	after, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Error("the refused export changed the previous delta")
	}
}

// A comment written on a position an earlier incremental export already sent
// leaves with the next one; the export after that carries it no more.
func TestExportIncrementalCarriesANewCommentOnASentPosition(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	s := seedWholeLibrary(t, cli)
	folder := t.TempDir()
	export := func() ingest.ExportReport {
		var out struct {
			Report ingest.ExportReport `json:"report"`
		}
		raw := captureStdout(t, func() {
			if err := cli.Run([]string{"export", "--db", dbPath, "--type", "database", "--incremental", "--file", folder, "--format", "json"}); err != nil {
				t.Fatalf("export: %v", err)
			}
		})
		if err := json.Unmarshal([]byte(raw), &out); err != nil {
			t.Fatal(err)
		}
		return out.Report
	}
	export()
	pos := s.positions[0]
	captureStdout(t, func() {
		if err := cli.Run([]string{"comment", "add", "--db", dbPath, "--position", strconv.FormatInt(pos, 10), "--text", "premier commentaire", "--author", "K"}); err != nil {
			t.Fatalf("comment add: %v", err)
		}
	})
	if r := export(); r.Comments == 0 || r.Positions == 0 {
		t.Fatalf("export after a new comment = %+v, want the comment and its position", r)
	}
	if r := export(); r.Comments != 0 || r.Positions != 0 {
		t.Fatalf("export after that = %+v, want empty", r)
	}
}
