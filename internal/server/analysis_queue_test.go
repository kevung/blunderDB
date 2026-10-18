package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/internal/server/middleware"
)

func queueStatus(t *testing.T, srv *Server) analysisQueueResp {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/analyses.queueStatus", strings.NewReader("{}"))
	req.Header.Set(middleware.TenantHeader, "1")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("queueStatus: %d (%s)", rec.Code, rec.Body)
	}
	var out analysisQueueResp
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// The gauges count what the library lacks and report the last sweep's
// counters once it ended.
func TestAnalysisQueueStatusFollowsASweep(t *testing.T) {
	srv, s := newGammonNetTestServer(t)
	pos := bearoffRacePosition()
	if _, err := s.Positions().Save(context.Background(), "t", &pos); err != nil {
		t.Fatal(err)
	}

	before := queueStatus(t, srv)
	if before.Positions != 1 || before.Pending != 1 || before.Analysed != 0 || before.Running || before.Sweep != nil {
		t.Fatalf("before the sweep: %+v", before)
	}

	postNDJSON(t, srv, "/v1/gammonnet.analyzeMissing", "")

	after := queueStatus(t, srv)
	if after.Pending != 0 || after.Analysed != 1 || after.Running {
		t.Fatalf("after the sweep: %+v", after)
	}
	if after.Sweep == nil || after.Sweep.State != "done" || after.Sweep.Done != 1 || after.Sweep.Total != 1 || after.Sweep.Failed != 0 {
		t.Fatalf("sweep = %+v, want done 1/1 without failure", after.Sweep)
	}
}
