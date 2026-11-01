package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/changeset"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// One library's export, posted page by page into another, arrives under its
// uuid, and the report says what became of it.
func TestChangeset_ExportImportOverHTTP(t *testing.T) {
	open := func() *Server {
		st, err := sqlite.Open(context.Background(), ":memory:", nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { st.Close() })
		srv, err := New(Options{Storage: st, Logger: slog.New(slog.DiscardHandler), TrustReadTenants: true})
		if err != nil {
			t.Fatal(err)
		}
		return srv
	}
	a, b := open(), open()
	for _, body := range []string{`{"position":` + initialPositionJSON(t) + `}`, `{"position":` + otherPositionJSON(t) + `}`} {
		if rec := serveAcross(t, a, "1", "", "/v1/positions.save", body); rec.Code != http.StatusOK {
			t.Fatalf("positions.save: %d %s", rec.Code, rec.Body)
		}
	}
	cursor, added := json.RawMessage(`{}`), 0
	for page := 0; ; page++ {
		rec := serveAcross(t, a, "1", "", "/v1/changeset.export", `{"cursor":`+string(cursor)+`,"limit":1}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("export: %d %s", rec.Code, rec.Body)
		}
		var set struct {
			Cursor  json.RawMessage   `json:"cursor"`
			More    bool              `json:"more"`
			Records []json.RawMessage `json:"records"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &set); err != nil {
			t.Fatal(err)
		}
		records, _ := json.Marshal(set.Records)
		imp := serveAcross(t, b, "1", "", "/v1/changeset.import", `{"records":`+string(records)+`}`)
		if imp.Code != http.StatusOK {
			t.Fatalf("import: %d %s", imp.Code, imp.Body)
		}
		var rep changeset.Report
		if err := json.Unmarshal(imp.Body.Bytes(), &rep); err != nil {
			t.Fatal(err)
		}
		added += rep.Added
		cursor = set.Cursor
		if !set.More {
			break
		}
		if page > 10 {
			t.Fatal("the export never ends")
		}
	}
	if added != 2 {
		t.Errorf("added %d, want the 2 positions", added)
	}
}
