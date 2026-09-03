package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// B.8 (#176): total-failure errors, --fail-on-error, --format json
// ---------------------------------------------------------------------------

// writePositionFile writes one JSON-serialized Position per non-empty line,
// mirroring the `import --type position` file format.
func writePositionFile(t *testing.T, lines ...string) string {
	t.Helper()
	path := filepath.Join(tempDir(t), "positions.txt")
	content := ""
	for _, l := range lines {
		content += l + "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func validPositionLine(t *testing.T) string {
	t.Helper()
	pos := InitializePosition()
	data, err := json.Marshal(pos)
	if err != nil {
		t.Fatalf("marshal position: %v", err)
	}
	return string(data)
}

func TestCLI_ImportPosition_NoneImportedIsAnError(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	path := writePositionFile(t, "not json", "also not json")

	err := cli.Run([]string{"import", "--db", dbPath, "--type", "position", "--file", path})
	if err == nil {
		t.Fatal("expected an error when zero positions were imported (#176)")
	}
}

func TestCLI_ImportPosition_EmptyFileIsAnError(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	path := writePositionFile(t)

	err := cli.Run([]string{"import", "--db", dbPath, "--type", "position", "--file", path})
	if err == nil {
		t.Fatal("expected an error when a position file imports nothing")
	}
}

func TestCLI_ImportPosition_PartialFailureDefaultsToSuccess(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	path := writePositionFile(t, validPositionLine(t), "garbage line")

	err := cli.Run([]string{"import", "--db", dbPath, "--type", "position", "--file", path})
	if err != nil {
		t.Fatalf("a partial failure without --fail-on-error must still succeed: %v", err)
	}
}

func TestCLI_ImportPosition_FailOnErrorFailsPartialImport(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	path := writePositionFile(t, validPositionLine(t), "garbage line")

	err := cli.Run([]string{"import", "--db", dbPath, "--type", "position", "--file", path, "--fail-on-error"})
	if err == nil {
		t.Fatal("expected --fail-on-error to fail a partially-failed import")
	}
}

func TestCLI_ImportPosition_JSON(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	path := writePositionFile(t, validPositionLine(t), "garbage line")

	out := captureStdout(t, func() {
		err := cli.Run([]string{"import", "--db", dbPath, "--type", "position", "--file", path, "--format", "json"})
		if err != nil {
			t.Fatalf("position import json: %v", err)
		}
	})

	trimmed := bytes.TrimSpace([]byte(out))
	if !json.Valid(trimmed) {
		t.Fatalf("stdout is not a single valid JSON document:\n%s", out)
	}
	var result importPositionResult
	if err := json.Unmarshal(trimmed, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Imported != 1 || result.Failed != 1 {
		t.Errorf("expected imported=1 failed=1, got %+v", result)
	}
}

func TestCLI_ImportBatch_NoFilesIsAnError(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	emptyDir := tempDir(t)

	err := cli.Run([]string{"import", "--db", dbPath, "--type", "batch", "--dir", emptyDir})
	if err == nil {
		t.Fatal("expected an error when a batch import directory has no supported files")
	}
}

func TestCLI_ImportBatch_AllFailedIsAnError(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	dir := tempDir(t)
	if err := os.WriteFile(filepath.Join(dir, "bad.xg"), []byte("not a real match file"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	err := cli.Run([]string{"import", "--db", dbPath, "--type", "batch", "--dir", dir})
	if err == nil {
		t.Fatal("expected an error when every file in a batch import failed")
	}
}

func TestCLI_ImportBatch_PartialFailureDefaultsToSuccess(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	dir := tempDir(t)
	if err := os.WriteFile(filepath.Join(dir, "bad.xg"), []byte("not a real match file"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	good, err := os.ReadFile(testdataPath("test.xg"))
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "good.xg"), good, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	err = cli.Run([]string{"import", "--db", dbPath, "--type", "batch", "--dir", dir})
	if err != nil {
		t.Fatalf("a batch with at least one success must not fail by default: %v", err)
	}
}

func TestCLI_ImportBatch_FailOnErrorFailsPartialImport(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	dir := tempDir(t)
	if err := os.WriteFile(filepath.Join(dir, "bad.xg"), []byte("not a real match file"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	good, err := os.ReadFile(testdataPath("test.xg"))
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "good.xg"), good, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	err = cli.Run([]string{"import", "--db", dbPath, "--type", "batch", "--dir", dir, "--fail-on-error"})
	if err == nil {
		t.Fatal("expected --fail-on-error to fail a batch with at least one failure")
	}
}

func TestCLI_ImportBatch_JSON(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	dir := tempDir(t)
	good, err := os.ReadFile(testdataPath("test.xg"))
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "good.xg"), good, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	out := captureStdout(t, func() {
		err := cli.Run([]string{"import", "--db", dbPath, "--type", "batch", "--dir", dir, "--format", "json"})
		if err != nil {
			t.Fatalf("batch import json: %v", err)
		}
	})

	trimmed := bytes.TrimSpace([]byte(out))
	if !json.Valid(trimmed) {
		t.Fatalf("stdout is not a single valid JSON document:\n%s", out)
	}
	var result importBatchResult
	if err := json.Unmarshal(trimmed, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Success != 1 || result.Total != 1 || len(result.Files) != 1 {
		t.Errorf("unexpected batch summary: %+v", result)
	}
}

func TestCLI_ImportMatch_JSON(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)

	out := captureStdout(t, func() {
		err := cli.Run([]string{"import", "--db", dbPath, "--type", "match", "--file", testdataPath("test.xg"), "--format", "json"})
		if err != nil {
			t.Fatalf("match import json: %v", err)
		}
	})

	trimmed := bytes.TrimSpace([]byte(out))
	if !json.Valid(trimmed) {
		t.Fatalf("stdout is not a single valid JSON document:\n%s", out)
	}
	var result importMatchResult
	if err := json.Unmarshal(trimmed, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Type != "match" || result.MatchID == 0 {
		t.Errorf("unexpected match import result: %+v", result)
	}
}

func TestCLI_Import_UnknownFormat(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	err := cli.Run([]string{"import", "--db", dbPath, "--type", "match", "--file", testdataPath("test.xg"), "--format", "yaml"})
	if err == nil {
		t.Fatal("expected an error for an unknown --format value")
	}
}
