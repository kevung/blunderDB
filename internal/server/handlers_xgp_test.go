package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

func TestPositionsFromXGP(t *testing.T) {
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
		req := httptest.NewRequest(http.MethodPost, "/v1/positions.fromXGP", strings.NewReader(body))
		req.Header.Set(middleware.TenantHeader, "t")
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		return rec
	}

	// A real .xgp single-position fixture → 200 + a complete position.
	raw, err := os.ReadFile("../../testdata/xgp/Position 10.xgp")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	b64 := base64.StdEncoding.EncodeToString(raw)
	body, _ := json.Marshal(map[string]string{"data": b64})

	rec := post(string(body))
	if rec.Code != http.StatusOK {
		t.Fatalf("valid xgp: got %d (%s)", rec.Code, rec.Body)
	}
	var out struct {
		Position *domain.Position `json:"position"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Position == nil {
		t.Fatal("nil position")
	}
	var on [2]int
	for _, p := range out.Position.Board.Points {
		if p.Color == domain.Black || p.Color == domain.White {
			on[p.Color] += p.Checkers
		}
	}
	// A complete backgammon position: 15 checkers per side (board + borne off).
	black := on[domain.Black] + out.Position.Board.Bearoff[domain.Black]
	white := on[domain.White] + out.Position.Board.Bearoff[domain.White]
	if black != 15 || white != 15 {
		t.Fatalf("incomplete position: black=%d white=%d (on=%v bearoff=%v)",
			black, white, on, out.Position.Board.Bearoff)
	}

	// Garbage base64 → 400.
	if bad := post(`{"data":"@@@not-base64@@@"}`); bad.Code != http.StatusBadRequest {
		t.Fatalf("bad base64: got %d, want 400", bad.Code)
	}
	// Valid base64 but not an XGP file → 400.
	notxgp := base64.StdEncoding.EncodeToString([]byte("hello world"))
	if bad := post(`{"data":"` + notxgp + `"}`); bad.Code != http.StatusBadRequest {
		t.Fatalf("non-xgp: got %d, want 400", bad.Code)
	}
}
