package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// exports.plan announces, for a body, the counts that exports.sqlite then
// reports in its header for the same body.
func TestExportPlanAgreesWithExportReport(t *testing.T) {
	ts, srv := newTestServerAndHandler(t)
	ctx := context.Background()
	st := srv.opts.Storage

	fixture, err := os.ReadFile(filepath.Join("..", "..", "testdata", "test.sgf"))
	if err != nil {
		t.Fatal(err)
	}
	uploadImportNamed(t, ts, "/v1/imports.gnubg", "test.sgf", fixture)
	var matchID int64
	for m, err := range st.Matches().List(ctx, testTenant, storage.MatchListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		matchID = m.ID
	}
	if _, err := st.Transcriptions().Save(ctx, testTenant, &storage.Transcription{MatchID: matchID, Label: "brouillon", Document: `{"header":{}}`}); err != nil {
		t.Fatal(err)
	}
	p := domain.InitializePosition()
	posID, err := st.Positions().Save(ctx, testTenant, &p)
	if err != nil {
		t.Fatal(err)
	}
	deck, err := st.Anki().CreateDeck(ctx, testTenant, "Ouvertures", "", domain.AnkiSourceSearch, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Anki().SyncWithPositions(ctx, testTenant, deck, []int64{posID}); err != nil {
		t.Fatal(err)
	}

	f := false
	for _, tc := range []struct {
		name string
		req  any
	}{
		{"whole tenant", nil},
		{"decks named", exportSQLiteReq{DeckIDs: []int64{deck}}},
		{"drafts off", exportSQLiteReq{Transcriptions: &f}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var plan ingest.ExportPlan
			resp := post(t, ts, "/v1/exports.plan", tc.req)
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("exports.plan status = %d", resp.StatusCode)
			}
			if err := json.NewDecoder(resp.Body).Decode(&plan); err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()

			resp = post(t, ts, "/v1/exports.sqlite", tc.req)
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("exports.sqlite status = %d", resp.StatusCode)
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			var report ingest.ExportReport
			if err := json.Unmarshal([]byte(resp.Header.Get(ExportReportHeader)), &report); err != nil {
				t.Fatalf("report header %q: %v", resp.Header.Get(ExportReportHeader), err)
			}
			if report.Positions == 0 {
				t.Fatalf("report counts no position: %+v", report)
			}
			if m := plan.Mismatch(report); m != "" {
				t.Errorf("plan and report disagree on %s", m)
			}

			var page ingest.PlannedIDs
			resp = post(t, ts, "/v1/exports.planIds", exportPlanIDsReq{Family: ingest.PlanPositions, Limit: 1})
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("exports.planIds status = %d", resp.StatusCode)
			}
			if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			if page.Total == 0 || len(page.IDs) != 1 {
				t.Errorf("positions page %+v, want one id", page)
			}
		})
	}
}
