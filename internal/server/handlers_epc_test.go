package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

func TestPositionsEPC(t *testing.T) {
	ctx := context.Background()
	s, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	srv, err := New(Options{Storage: s})
	if err != nil {
		t.Fatal(err)
	}

	post := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/v1/positions.epc", strings.NewReader(body))
		req.Header.Set(middleware.TenantHeader, "t")
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		return rec
	}

	// Build a position: Black all home (15 on points 1-6), White NOT all home
	// (a straggler on point 13). EPC must be present for Black, nil for White.
	var pos domain.Position
	set := func(point, color, n int) {
		pos.Board.Points[point] = domain.Point{Color: color, Checkers: n}
	}
	set(1, domain.Black, 3)
	set(2, domain.Black, 3)
	set(3, domain.Black, 3)
	set(4, domain.Black, 2)
	set(5, domain.Black, 2)
	set(6, domain.Black, 2) // = 15 Black, all home
	set(24, domain.White, 14)
	set(13, domain.White, 1) // straggler → White not all home

	body, _ := json.Marshal(map[string]any{"position": pos})
	rec := post(string(body))
	if rec.Code != http.StatusOK {
		t.Fatalf("epc: got %d (%s)", rec.Code, rec.Body)
	}
	var resp epcResp
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.Bottom.AllInHome || resp.Bottom.CheckerCount != 15 {
		t.Fatalf("bottom: allHome=%v count=%d, want true/15", resp.Bottom.AllInHome, resp.Bottom.CheckerCount)
	}
	if resp.Bottom.EPC == nil {
		t.Fatal("bottom EPC should be computed when all home")
	}
	// Pip count for 3·1+3·2+3·3+2·4+2·5+2·6 = 3+6+9+8+10+12 = 48.
	if resp.Bottom.EPC.PipCount != 48 {
		t.Fatalf("bottom pip count = %d, want 48", resp.Bottom.EPC.PipCount)
	}
	if resp.Bottom.EPC.EPC < float64(resp.Bottom.EPC.PipCount) {
		t.Fatalf("EPC %.2f should be >= pip count %d (wastage >= 0)", resp.Bottom.EPC.EPC, resp.Bottom.EPC.PipCount)
	}
	if resp.Top.AllInHome || resp.Top.EPC != nil {
		t.Fatalf("top: allHome=%v epc=%v, want not-home/nil", resp.Top.AllInHome, resp.Top.EPC)
	}
	if resp.Top.CheckerCount != 15 {
		t.Fatalf("top checker count = %d, want 15", resp.Top.CheckerCount)
	}

	// Missing position → 400.
	if bad := post(`{}`); bad.Code != http.StatusBadRequest {
		t.Fatalf("missing position: got %d, want 400", bad.Code)
	}
}
