package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
)

// B.8 (#176): --format json on the nine commands that lacked it. Each test
// only checks that stdout is a single valid JSON document carrying the
// expected shape — the human-readable text path already has its own
// coverage elsewhere.

func TestCLI_VacuumJSON(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)

	out := captureStdout(t, func() {
		if err := cli.Run([]string{"vacuum", "--db", dbPath, "--format", "json"}); err != nil {
			t.Fatalf("vacuum json: %v", err)
		}
	})

	trimmed := bytes.TrimSpace([]byte(out))
	if !json.Valid(trimmed) {
		t.Fatalf("stdout is not a single valid JSON document:\n%s", out)
	}
	var result struct {
		SizeBefore int64 `json:"size_before"`
		SizeAfter  int64 `json:"size_after"`
	}
	if err := json.Unmarshal(trimmed, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.SizeBefore == 0 || result.SizeAfter == 0 {
		t.Errorf("unexpected vacuum json: %+v", result)
	}
}

func TestCLI_VerifyJSON(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)

	out := captureStdout(t, func() {
		if err := cli.Run([]string{"verify", "--db", dbPath, "--format", "json"}); err != nil {
			t.Fatalf("verify json: %v", err)
		}
	})

	trimmed := bytes.TrimSpace([]byte(out))
	if !json.Valid(trimmed) {
		t.Fatalf("stdout is not a single valid JSON document:\n%s", out)
	}
	var result verifyResult
	if err := json.Unmarshal(trimmed, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Stats == nil {
		t.Errorf("unexpected verify json: %+v", result)
	}
}

func TestCLI_CreateJSON(t *testing.T) {
	dbPath := tempDir(t) + "/created.db"
	cli := &CLI{db: NewDatabase()}
	closeOnCleanup(t, cli.db)

	out := captureStdout(t, func() {
		if err := cli.Run([]string{"create", "--db", dbPath, "--user", "Alice", "--format", "json"}); err != nil {
			t.Fatalf("create json: %v", err)
		}
	})

	trimmed := bytes.TrimSpace([]byte(out))
	if !json.Valid(trimmed) {
		t.Fatalf("stdout is not a single valid JSON document:\n%s", out)
	}
	var result struct {
		Path string `json:"path"`
		User string `json:"user"`
	}
	if err := json.Unmarshal(trimmed, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.User != "Alice" {
		t.Errorf("unexpected create json: %+v", result)
	}
}

func TestCLI_EditJSON(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)

	out := captureStdout(t, func() {
		if err := cli.Run([]string{"edit", "--db", dbPath, "--user", "Bob", "--format", "json"}); err != nil {
			t.Fatalf("edit json: %v", err)
		}
	})

	trimmed := bytes.TrimSpace([]byte(out))
	if !json.Valid(trimmed) {
		t.Fatalf("stdout is not a single valid JSON document:\n%s", out)
	}
	var result struct {
		Changes []string `json:"changes"`
	}
	if err := json.Unmarshal(trimmed, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result.Changes) != 1 {
		t.Errorf("unexpected edit json: %+v", result)
	}
}

func TestCLI_DeleteJSON(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	if err := cli.Run([]string{"import", "--db", dbPath, "--type", "match", "--file", testdataPath("test.xg")}); err != nil {
		t.Fatalf("import: %v", err)
	}
	matches, _ := cli.db.GetAllMatches()
	if len(matches) == 0 {
		t.Fatal("no matches after import")
	}

	out := captureStdout(t, func() {
		if err := cli.Run([]string{"delete", "--db", dbPath, "--type", "match",
			"--id", fmt.Sprintf("%d", matches[0].ID), "--confirm", "--format", "json"}); err != nil {
			t.Fatalf("delete json: %v", err)
		}
	})

	trimmed := bytes.TrimSpace([]byte(out))
	if !json.Valid(trimmed) {
		t.Fatalf("stdout is not a single valid JSON document:\n%s", out)
	}
	var result struct {
		MatchID int64 `json:"match_id"`
		Deleted bool  `json:"deleted"`
	}
	if err := json.Unmarshal(trimmed, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !result.Deleted {
		t.Errorf("unexpected delete json: %+v", result)
	}
}

func TestCLI_ExportPositionsJSON(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	if err := cli.Run([]string{"import", "--db", dbPath, "--type", "match", "--file", testdataPath("test.xg")}); err != nil {
		t.Fatalf("import: %v", err)
	}
	outFile := tempDir(t) + "/positions.txt"

	out := captureStdout(t, func() {
		if err := cli.Run([]string{"export", "--db", dbPath, "--type", "positions", "--file", outFile, "--format", "json"}); err != nil {
			t.Fatalf("export json: %v", err)
		}
	})

	trimmed := bytes.TrimSpace([]byte(out))
	if !json.Valid(trimmed) {
		t.Fatalf("stdout is not a single valid JSON document:\n%s", out)
	}
	var result struct {
		File      string `json:"file"`
		Positions int    `json:"positions"`
	}
	if err := json.Unmarshal(trimmed, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Positions == 0 {
		t.Errorf("unexpected export json: %+v", result)
	}
}

func TestCLI_AnalyzeJSON_NothingToDo(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)

	out := captureStdout(t, func() {
		if err := cli.Run([]string{"analyze", "--db", dbPath, "--format", "json"}); err != nil {
			t.Fatalf("analyze json: %v", err)
		}
	})

	trimmed := bytes.TrimSpace([]byte(out))
	if !json.Valid(trimmed) {
		t.Fatalf("stdout is not a single valid JSON document:\n%s", out)
	}
	var result analyzeResult
	if err := json.Unmarshal(trimmed, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("unexpected analyze json: %+v", result)
	}
}

// B.8 (#176): parseSearchFlags is now unit-testable on its own, without a
// database — it used to be inseparable from runSearch's 400+ lines.

func TestParseSearchFlags_MissingDB(t *testing.T) {
	t.Parallel()
	_, _, err := parseSearchFlags([]string{"--decision", "cube"})
	if err == nil {
		t.Fatal("expected an error for a missing --db")
	}
}

func TestParseSearchFlags_BuildsParams(t *testing.T) {
	t.Parallel()
	params, dbPath, err := parseSearchFlags([]string{
		"--db", "database.db", "--format", "JSON", "--limit", "5",
		"--decision", "cube", "--cube", "2",
	})
	if err != nil {
		t.Fatalf("parseSearchFlags: %v", err)
	}
	if dbPath != "database.db" {
		t.Errorf("expected dbPath=database.db, got %q", dbPath)
	}
	if params.format != "json" {
		t.Errorf("expected format to be lowercased to json, got %q", params.format)
	}
	if params.limit != 5 {
		t.Errorf("expected limit=5, got %d", params.limit)
	}
	if !params.filters.IncludeCube || params.filters.Filter.Cube.Value != 2 {
		t.Errorf("expected cube filter to be set, got %+v", params.filters.Filter.Cube)
	}
	if params.filters.Filter.DecisionType != CubeAction {
		t.Errorf("expected decision type cube, got %v", params.filters.Filter.DecisionType)
	}
}

func TestParseSearchFlags_InvalidDice(t *testing.T) {
	t.Parallel()
	_, _, err := parseSearchFlags([]string{"--db", "database.db", "--dice", "7"})
	if err == nil {
		t.Fatal("expected an error for an out-of-range die")
	}
}

func TestParseSearchFlags_MutuallyExclusiveComments(t *testing.T) {
	t.Parallel()
	_, _, err := parseSearchFlags([]string{"--db", "database.db", "--has-comment", "--no-comment"})
	if err == nil {
		t.Fatal("expected an error for --has-comment and --no-comment together")
	}
}

func TestRenderResults_Formats(t *testing.T) {
	t.Parallel()
	cli, dbPath := setupCLIWithDB(t)
	if err := cli.Run([]string{"import", "--db", dbPath, "--type", "match", "--file", testdataPath("test.xg")}); err != nil {
		t.Fatalf("import: %v", err)
	}
	positions, err := cli.db.LoadAllPositions()
	if err != nil || len(positions) == 0 {
		t.Fatalf("LoadAllPositions: %v (n=%d)", err, len(positions))
	}
	one := positions[:1]

	var tableBuf bytes.Buffer
	if err := cli.renderResults(&tableBuf, one, "table"); err != nil {
		t.Fatalf("renderResults table: %v", err)
	}
	if !bytes.Contains(tableBuf.Bytes(), []byte("ID")) {
		t.Errorf("table output missing header:\n%s", tableBuf.String())
	}

	var jsonBuf bytes.Buffer
	if err := cli.renderResults(&jsonBuf, one, "json"); err != nil {
		t.Fatalf("renderResults json: %v", err)
	}
	if !json.Valid(bytes.TrimSpace(jsonBuf.Bytes())) {
		t.Errorf("json output is not valid JSON:\n%s", jsonBuf.String())
	}

	var xgidBuf bytes.Buffer
	if err := cli.renderResults(&xgidBuf, one, "xgid"); err != nil {
		t.Fatalf("renderResults xgid: %v", err)
	}
}
