package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kevung/blunderdb/internal/server/metrics"
	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

func newQuotaTestServer(t *testing.T, q TenantQuotas) (*httptest.Server, *Server) {
	t.Helper()
	st, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	srv, err := New(Options{Storage: st, Metrics: metrics.New(), Quotas: q})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts, srv
}

// errorOf reads an error envelope and closes the body.
func errorOf(t *testing.T, resp *http.Response) (int, errorBody) {
	t.Helper()
	defer resp.Body.Close()
	var env errorEnvelope
	b, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(b, &env)
	return resp.StatusCode, env.Error
}

func TestQuotaLedger_DayAndImports(t *testing.T) {
	clock := time.Date(2026, 10, 3, 23, 0, 0, 0, time.UTC)
	q := newQuotaLedger(TenantQuotas{AnalysisSecondsPerDay: 10, MaxConcurrentImports: 1}, func() time.Time { return clock })

	spend := q.spender("a")
	if !spend(9 * time.Second) {
		t.Fatal("9 s of 10: some is left")
	}
	if spend(time.Second) || !q.analysisExhausted("a") {
		t.Fatal("10 s of 10: spent")
	}
	if q.analysisExhausted("b") {
		t.Fatal("another tenant's allowance is its own")
	}
	clock = clock.Add(2 * time.Hour)
	if q.analysisExhausted("a") {
		t.Fatal("a new UTC day starts a new allowance")
	}

	if !q.beginImport("a") || q.beginImport("a") {
		t.Fatal("one import at once: the second is refused")
	}
	if !q.beginImport("b") {
		t.Fatal("another tenant's slot is its own")
	}
	q.endImport("a")
	if !q.beginImport("a") {
		t.Fatal("a slot given back is claimable")
	}
}

func TestQuotaUnlimitedByDefault(t *testing.T) {
	q := newQuotaLedger(TenantQuotas{}, nil)
	q.charge("a", 24*time.Hour)
	for range 5 {
		if !q.beginImport("a") {
			t.Fatal("no bound: every import is accepted")
		}
	}
	if q.analysisExhausted("a") {
		t.Fatal("no bound: engine time is never spent")
	}
}

func TestQuotaRefusesImportPastMaxPositions(t *testing.T) {
	ts, srv := newQuotaTestServer(t, TenantQuotas{MaxPositions: 1})
	p := domain.InitializePosition()
	if _, err := srv.opts.Storage.Positions().Save(context.Background(), testTenant, &p); err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", "data.ndjson")
	_, _ = fw.Write([]byte("{}\n"))
	mw.Close()
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/imports.json", &body)
	req.Header.Set(middleware.TenantHeader, testTenant)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	status, e := errorOf(t, resp)
	if status != http.StatusRequestEntityTooLarge || e.Code != CodeStorageQuotaExceeded || e.Details["quota"] != "maxPositions" {
		t.Fatalf("import past the bound: %d %+v; want 413 storage_quota_exceeded", status, e)
	}
	if _, n := srv.quota.usage(testTenant); n != 0 {
		t.Errorf("a refused import holds no slot: %d", n)
	}
}

// TestQuotaRefusesImportPastMaxStoredBytes: a tenant that holds its disk
// space is refused a new import, and tenants.quota reports what it holds.
func TestQuotaRefusesImportPastMaxStoredBytes(t *testing.T) {
	ts, srv := newQuotaTestServer(t, TenantQuotas{MaxStoredBytes: 1})
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", "data.ndjson")
	_, _ = fw.Write([]byte("{}\n"))
	mw.Close()
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/imports.json", &body)
	req.Header.Set(middleware.TenantHeader, testTenant)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	status, e := errorOf(t, resp)
	if status != http.StatusRequestEntityTooLarge || e.Code != CodeStorageQuotaExceeded || e.Details["quota"] != "maxStoredBytes" {
		t.Fatalf("import past the bound: %d %+v; want 413 storage_quota_exceeded on maxStoredBytes", status, e)
	}
	if _, n := srv.quota.usage(testTenant); n != 0 {
		t.Errorf("a refused import holds no slot: %d", n)
	}
	var got tenantQuotaResp
	postDecode(t, ts, "/v1/tenants.quota", nil, &got)
	if got.Limits.MaxStoredBytes != 1 || got.Usage.StoredBytes <= 0 {
		t.Fatalf("tenants.quota = %+v; want the bound and the space held", got)
	}
}

// TestQuotaPositionsHelpSaysWhatIsChecked: --quota-positions promises what
// the daemon holds — an import refused at its start — and no bound on the
// unit writes, which the daemon does not check.
func TestQuotaPositionsHelpSaysWhatIsChecked(t *testing.T) {
	ts, srv := newQuotaTestServer(t, TenantQuotas{MaxPositions: 1})
	p := domain.InitializePosition()
	if _, err := srv.opts.Storage.Positions().Save(context.Background(), testTenant, &p); err != nil {
		t.Fatal(err)
	}
	q := domain.InitializePosition()
	q.Dice = [2]int{3, 1}
	resp := post(t, ts, "/v1/positions.save", positionReq{Position: &q})
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("positions.save at the bound: %d; the help says unit writes are not bounded", resp.StatusCode)
	}
	for _, claim := range []string{"when an import starts", "positions.save", "not bounded"} {
		if !strings.Contains(quotaPositionsHelp, claim) {
			t.Errorf("--quota-positions help %q does not say %q", quotaPositionsHelp, claim)
		}
	}
}

func TestQuotaRefusesAnalysisOnceSpent(t *testing.T) {
	ts, srv := newQuotaTestServer(t, TenantQuotas{AnalysisSecondsPerDay: 1})
	srv.quota.charge(testTenant, 2*time.Second)
	xgid := "XGID=-b----E-C---eE---c-e----B-:0:0:1:52:0:0:0:0:10"
	for _, c := range []struct {
		path string
		body any
	}{
		{"/v1/gammonnet.evaluate", gammonnetEvaluateReq{XGID: xgid}},
		{"/v1/gammonnet.cubeMatrix", cubeMatrixReq{XGID: xgid}},
		{"/v1/gammonnet.compare", gammonnetCompareReq{}},
		{"/v1/gammonnet.analyzeMissing", nil},
	} {
		resp := post(t, ts, c.path, c.body)
		status, e := errorOf(t, resp)
		resp.Body.Close()
		if status != http.StatusTooManyRequests || e.Code != CodeQuotaExceeded || e.Details["quota"] != "analysisSecondsPerDay" {
			t.Errorf("%s once spent: %d %+v; want 429 quota_exceeded", c.path, status, e)
		}
	}
}

func TestTenantsQuotaReportsLimitsAndUse(t *testing.T) {
	ts, srv := newQuotaTestServer(t, TenantQuotas{MaxPositions: 100, AnalysisSecondsPerDay: 60})
	ply := 0
	resp := post(t, ts, "/v1/gammonnet.evaluate", gammonnetEvaluateReq{XGID: "XGID=-b----E-C---eE---c-e----B-:0:0:1:52:0:0:0:0:10", Ply: &ply})
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("evaluate under the bound: %d", resp.StatusCode)
	}
	if spent, _ := srv.quota.usage(testTenant); spent <= 0 {
		t.Fatal("an evaluation is charged to its tenant")
	}

	var got tenantQuotaResp
	postDecode(t, ts, "/v1/tenants.quota", nil, &got)
	if got.Limits.MaxPositions != 100 || got.Limits.AnalysisSecondsPerDay != 60 || got.Usage.AnalysisSecondsToday <= 0 {
		t.Fatalf("tenants.quota = %+v", got)
	}
}

// TestQuotaStopsSweepMidway: a sweep that runs the tenant's allowance out keeps
// what it computed and ends on quota_exceeded, well before its total.
func TestQuotaStopsSweepMidway(t *testing.T) {
	ts, srv := newQuotaTestServer(t, TenantQuotas{AnalysisSecondsPerDay: 1})
	ctx := context.Background()
	saved := 0
	for d1 := 1; d1 <= 6; d1++ {
		for d2 := 1; d2 <= 6; d2++ {
			for roll := 0; roll <= 1; roll++ {
				p := domain.InitializePosition()
				p.Dice, p.PlayerOnRoll = [2]int{d1, d2}, roll
				if _, err := srv.opts.Storage.Positions().Save(ctx, testTenant, &p); err != nil {
					t.Fatal(err)
				}
				saved++
			}
		}
	}
	srv.quota.charge(testTenant, time.Second-time.Nanosecond)

	resp := post(t, ts, "/v1/gammonnet.analyzeMissing", gammonnetAnalyzeReq{Ply: 0})
	defer resp.Body.Close()
	var last map[string]any
	dec := json.NewDecoder(resp.Body)
	for dec.More() {
		last = nil
		if err := dec.Decode(&last); err != nil {
			t.Fatal(err)
		}
	}
	if last["event"] != "quota_exceeded" {
		t.Fatalf("last event = %v; want quota_exceeded", last)
	}
	done, total := last["done"].(float64), last["total"].(float64)
	if done < 1 || done >= total {
		t.Errorf("sweep stopped at %v of %v (saved %d); want partway", done, total, saved)
	}
}

// saveOpeningRolls stores n distinct opening positions, each with a roll and
// an analysis that is not gammonNet's, and returns their ids.
func saveOpeningRolls(t *testing.T, srv *Server, n int) []int64 {
	t.Helper()
	ctx := context.Background()
	var ids []int64
	for d1 := 1; d1 <= 6 && len(ids) < n; d1++ {
		for d2 := 1; d2 <= 6 && len(ids) < n; d2++ {
			p := domain.InitializePosition()
			p.Dice = [2]int{d1, d2}
			id, err := srv.opts.Storage.Positions().Save(ctx, testTenant, &p)
			if err != nil {
				t.Fatal(err)
			}
			a := &domain.PositionAnalysis{CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
				{Move: "13/7", Equity: 0.10, AnalysisEngine: "XG"},
			}}}
			if err := srv.opts.Storage.Analyses().Save(ctx, testTenant, id, a); err != nil {
				t.Fatal(err)
			}
			ids = append(ids, id)
		}
	}
	return ids
}

// TestQuotaCutComparisonSaysSo: a comparison the tenant's time runs out on
// answers what it folded, flagged, never as the whole library.
func TestQuotaCutComparisonSaysSo(t *testing.T) {
	ts, srv := newQuotaTestServer(t, TenantQuotas{AnalysisSecondsPerDay: 1})
	srv.engineWorkers = 1
	saveOpeningRolls(t, srv, 6)
	srv.quota.charge(testTenant, time.Second-time.Nanosecond)

	var got gammonnetCompareResp
	postDecode(t, ts, "/v1/gammonnet.compare", gammonnetCompareReq{Ply: 0}, &got)
	if !got.QuotaExceeded || got.Gathered != 6 || got.Compared+got.Refused+got.Failed >= 6 {
		t.Fatalf("comparison cut by the quota = %+v; want quotaExceeded over 6 gathered", got)
	}

	ts2, srv2 := newQuotaTestServer(t, TenantQuotas{})
	saveOpeningRolls(t, srv2, 2)
	var whole gammonnetCompareResp
	postDecode(t, ts2, "/v1/gammonnet.compare", gammonnetCompareReq{Ply: 0}, &whole)
	if whole.QuotaExceeded || whole.Gathered != 2 {
		t.Fatalf("comparison under no quota = %+v", whole)
	}
}

// TestQuotaChargesCubeMatrixInCPUSeconds: a cube matrix spread over several
// workers costs its wall time once per worker, the unit a sweep's workers
// charge in, so it costs more than the request's own wall time. The grid is
// large enough for the search to dwarf the HTTP round trip the test's clock
// also counts.
func TestQuotaChargesCubeMatrixInCPUSeconds(t *testing.T) {
	ts, srv := newQuotaTestServer(t, TenantQuotas{AnalysisSecondsPerDay: 3600})
	srv.engineWorkers = 4
	start := time.Now()
	resp := post(t, ts, "/v1/gammonnet.cubeMatrix", cubeMatrixReq{XGID: "XGID=-b----E-C---eE---c-e----B-:0:0:1:00:0:0:0:0:10", MatchLength: 9, Ply: 0})
	resp.Body.Close()
	wall := time.Since(start)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("cubeMatrix: %d", resp.StatusCode)
	}
	if spent, _ := srv.quota.usage(testTenant); spent <= wall {
		t.Fatalf("cube matrix on 4 workers charged %v for %v of wall time; want CPU time, above it", spent, wall)
	}
}

// TestQuotaBoundsRollouts: a rollout is engine time like any other: charged
// when it runs, a filter rollout the time runs out on stops on
// quota_exceeded, and both are refused once the day's time is spent.
func TestQuotaBoundsRollouts(t *testing.T) {
	ts, srv := newQuotaTestServer(t, TenantQuotas{AnalysisSecondsPerDay: 3600})
	ids := saveOpeningRolls(t, srv, 4)

	resp := post(t, ts, "/v1/rollout.position", rolloutPositionReq{PositionID: ids[0], Rollout: tinyRollout})
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("rollout.position under the bound: %d", resp.StatusCode)
	}
	spent, _ := srv.quota.usage(testTenant)
	if spent <= 0 || spent >= time.Hour {
		t.Fatalf("a rollout is charged to its tenant: %v", spent)
	}

	srv.quota.charge(testTenant, time.Hour-spent-time.Nanosecond)
	resp = post(t, ts, "/v1/rollout.filter", rolloutFilterReq{Rollout: tinyRollout})
	var last map[string]any
	dec := json.NewDecoder(resp.Body)
	for dec.More() {
		last = nil
		if err := dec.Decode(&last); err != nil {
			t.Fatal(err)
		}
	}
	resp.Body.Close()
	if last["event"] != "quota_exceeded" {
		t.Fatalf("rollout.filter running the time out ended on %v; want quota_exceeded", last)
	}

	for _, c := range []struct {
		path string
		body any
	}{
		{"/v1/rollout.position", rolloutPositionReq{PositionID: ids[0], Rollout: tinyRollout}},
		{"/v1/rollout.filter", rolloutFilterReq{Rollout: tinyRollout}},
	} {
		resp := post(t, ts, c.path, c.body)
		status, e := errorOf(t, resp)
		resp.Body.Close()
		if status != http.StatusTooManyRequests || e.Code != CodeQuotaExceeded {
			t.Errorf("%s once spent: %d %+v; want 429 quota_exceeded", c.path, status, e)
		}
	}
}
