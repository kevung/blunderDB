package cli

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// countMatFiles returns the number of *.mat files directly under dir.
func countMatFiles(t *testing.T, dir string) int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir %s: %v", dir, err)
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".mat") {
			n++
		}
	}
	return n
}

// TestCLI_ExportMAT_Batch imports two distinct matches and exports them all as
// .mat files into a directory (no --match-ids = all matches).
func TestCLI_ExportMAT_Batch(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	if err := cli.Run([]string{"import", "--db", dbPath, "--type", "match", "--file", testdataPath("test.mat")}); err != nil {
		t.Fatalf("import mat: %v", err)
	}
	if err := cli.Run([]string{"import", "--db", dbPath, "--type", "match", "--file", testdataPath("TachiAI_V_player_Nov_2__2025__16_55.bgf")}); err != nil {
		t.Fatalf("import bgf: %v", err)
	}

	outDir := filepath.Join(t.TempDir(), "out")
	if err := cli.Run([]string{"export", "--db", dbPath, "--type", "mat", "--dir", outDir}); err != nil {
		t.Fatalf("export mat batch: %v", err)
	}
	if got := countMatFiles(t, outDir); got != 2 {
		t.Errorf("expected 2 .mat files, got %d", got)
	}
}

// TestCLI_ExportMAT_Single exports one match by id to an explicit --file.
func TestCLI_ExportMAT_Single(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	if err := cli.Run([]string{"import", "--db", dbPath, "--type", "match", "--file", testdataPath("test.mat")}); err != nil {
		t.Fatalf("import mat: %v", err)
	}
	matches, err := cli.db.GetAllMatches()
	if err != nil || len(matches) == 0 {
		t.Fatalf("GetAllMatches: %v (n=%d)", err, len(matches))
	}
	id := matches[0].ID

	out := filepath.Join(t.TempDir(), "game.mat")
	if err := cli.Run([]string{"export", "--db", dbPath, "--type", "mat", "--match-ids", strconv.FormatInt(id, 10), "--file", out}); err != nil {
		t.Fatalf("export mat single: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read exported file: %v", err)
	}
	if !strings.Contains(string(data), "point match") {
		t.Errorf("exported .mat missing header:\n%s", data)
	}
}

// TestCLI_ExportMAT_FileWithMultipleErrors: --file with several matches must be
// rejected (a .mat file holds exactly one match).
func TestCLI_ExportMAT_FileWithMultipleErrors(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	if err := cli.Run([]string{"import", "--db", dbPath, "--type", "match", "--file", testdataPath("test.mat")}); err != nil {
		t.Fatalf("import mat: %v", err)
	}
	if err := cli.Run([]string{"import", "--db", dbPath, "--type", "match", "--file", testdataPath("TachiAI_V_player_Nov_2__2025__16_55.bgf")}); err != nil {
		t.Fatalf("import bgf: %v", err)
	}
	out := filepath.Join(t.TempDir(), "game.mat")
	// No --match-ids = all matches (2) → --file must error.
	if err := cli.Run([]string{"export", "--db", dbPath, "--type", "mat", "--file", out}); err == nil {
		t.Fatal("expected error exporting multiple matches to a single --file")
	}
}

// TestCLI_ExportMAT_RequiresFileOrDir: type mat with neither --file nor --dir.
func TestCLI_ExportMAT_RequiresFileOrDir(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	if err := cli.Run([]string{"import", "--db", dbPath, "--type", "match", "--file", testdataPath("test.mat")}); err != nil {
		t.Fatalf("import mat: %v", err)
	}
	if err := cli.Run([]string{"export", "--db", dbPath, "--type", "mat"}); err == nil {
		t.Fatal("expected error when type mat has neither --file nor --dir")
	}
}
