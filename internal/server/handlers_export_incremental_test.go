package server

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// exports.sqlite with "incremental" carries what entered the tenant since the
// last incremental export under the same cursor: everything the first time,
// nothing the second, and a cursor of another name starts over.
func TestExportSQLiteIncremental(t *testing.T) {
	ts := newTestServer(t)
	fixture, err := os.ReadFile(filepath.Join("..", "..", "testdata", "test.sgf"))
	if err != nil {
		t.Fatal(err)
	}
	uploadImportNamed(t, ts, "/v1/imports.gnubg", "test.sgf", fixture)

	export := func(cursor string) ingest.ExportReport {
		t.Helper()
		resp := post(t, ts, "/v1/exports.sqlite", map[string]any{"incremental": true, "syncCursor": cursor})
		defer resp.Body.Close()
		_, _ = io.Copy(io.Discard, resp.Body)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status %d", resp.StatusCode)
		}
		var r ingest.ExportReport
		if err := json.Unmarshal([]byte(resp.Header.Get(ExportReportHeader)), &r); err != nil {
			t.Fatal(err)
		}
		return r
	}

	if r := export("laptop"); r.Positions == 0 || r.Matches != 1 {
		t.Fatalf("first incremental export = %+v, want the match and its positions", r)
	}
	if r := export("laptop"); r.Positions != 0 || r.Matches != 0 {
		t.Errorf("second incremental export = %+v, want empty", r)
	}
	if r := export("desktop"); r.Matches != 1 {
		t.Errorf("a new cursor = %+v, want the whole tenant", r)
	}
}
