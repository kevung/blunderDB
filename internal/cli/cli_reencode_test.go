package cli

import (
	"strings"
	"testing"
)

// TestCLI_Reencode_RewritesLegacyAnalyses: a raw-JSON analysis is rewritten,
// and the report says how many.
func TestCLI_Reencode_RewritesLegacyAnalyses(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	res, err := RawConn(cli.db).Exec(`INSERT INTO position (state) VALUES ('legacy')`)
	if err != nil {
		t.Fatal(err)
	}
	posID, _ := res.LastInsertId()
	if _, err := RawConn(cli.db).Exec(`INSERT INTO analysis (position_id, data) VALUES (?, ?)`,
		posID, []byte(`{"xgid":"XGID=legacy"}`)); err != nil {
		t.Fatal(err)
	}
	out := captureStdout(t, func() {
		if err := cli.Run([]string{"reencode", "--db", dbPath, "--format", "json"}); err != nil {
			t.Fatalf("reencode: %v", err)
		}
	})
	if !strings.Contains(out, `"rewritten": 1`) {
		t.Errorf("reencode report = %q; want one analysis rewritten", out)
	}
}
