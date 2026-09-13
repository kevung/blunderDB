package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// analyses.loadByIds is the batch face of analyses.load, on the contract of
// positions.loadByIds: the caller's order, ids that name nothing (an unknown
// position, or a position without an analysis) skipped, a repeated id
// answered each time it is asked, and an empty list for no id.

// saveAnalysedPosition stores a position distinct by its score and, when
// analysed, an analysis whose XGID names it. It returns the position id.
func saveAnalysedPosition(t *testing.T, ts *httptest.Server, tenant string, n int, analysed bool) int64 {
	t.Helper()
	p := domain.InitializePosition()
	p.DecisionType = domain.CheckerAction
	p.Score = [2]int{n, 0}
	resp := postAsTenant(t, ts, tenant, "/v1/positions.save", positionReq{Position: &p})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("positions.save status = %d, want 200", resp.StatusCode)
	}
	var saved idResp
	if err := json.NewDecoder(resp.Body).Decode(&saved); err != nil {
		t.Fatal(err)
	}
	if !analysed {
		return saved.ID
	}
	a := &domain.PositionAnalysis{XGID: analysisXGID(saved.ID), AnalysisType: "XG Roller++"}
	resp2 := postAsTenant(t, ts, tenant, "/v1/analyses.save", analysisSaveReq{PositionID: saved.ID, Analysis: a})
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("analyses.save status = %d, want 200", resp2.StatusCode)
	}
	return saved.ID
}

func analysisXGID(id int64) string { return "xgid-" + itoa64(id) }

func itoa64(n int64) string { return strconv.FormatInt(n, 10) }

// postAsTenant is post (handlers_domain_test.go) under the tenant given. The
// postgres-tagged postTenant is not reachable from this untagged file.
func postAsTenant(t *testing.T, ts *httptest.Server, tenant, path string, body any) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(http.MethodPost, ts.URL+path, &buf)
	req.Header.Set(middleware.TenantHeader, tenant)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

// loadAnalysesByIDs posts analyses.loadByIds and decodes the answer.
func loadAnalysesByIDs(t *testing.T, ts *httptest.Server, tenant string, ids []int64) []domain.PositionAnalysis {
	t.Helper()
	resp := postAsTenant(t, ts, tenant, "/v1/analyses.loadByIds", idsReq{IDs: ids})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("analyses.loadByIds status = %d, want 200", resp.StatusCode)
	}
	var raw json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		t.Fatal(err)
	}
	if string(raw) == "null" {
		t.Fatalf("analyses.loadByIds answered null, want a JSON array")
	}
	var got []domain.PositionAnalysis
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("decoding %s: %v", raw, err)
	}
	return got
}

func TestAnalysesLoadByIDs(t *testing.T) {
	ts := newTestServer(t)
	p1 := saveAnalysedPosition(t, ts, testTenant, 1, true)
	p2 := saveAnalysedPosition(t, ts, testTenant, 2, true)
	bare := saveAnalysedPosition(t, ts, testTenant, 3, false)

	got := loadAnalysesByIDs(t, ts, testTenant, []int64{bare, p2, 987654321, p1, p2})
	want := []int64{p2, p1, p2}
	if len(got) != len(want) {
		t.Fatalf("analyses.loadByIds: %d analyses %+v, want positions %v", len(got), got, want)
	}
	for i, a := range got {
		if int64(a.PositionID) != want[i] || a.XGID != analysisXGID(want[i]) {
			t.Errorf("analyses.loadByIds[%d] = position %d xgid %q, want position %d xgid %q",
				i, a.PositionID, a.XGID, want[i], analysisXGID(want[i]))
		}
	}

	if empty := loadAnalysesByIDs(t, ts, testTenant, nil); len(empty) != 0 {
		t.Errorf("analyses.loadByIds(no id) = %+v, want []", empty)
	}
}
