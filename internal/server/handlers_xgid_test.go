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

func TestPositionsFromXGID(t *testing.T) {
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
		req := httptest.NewRequest(http.MethodPost, "/v1/positions.fromXGID", strings.NewReader(body))
		req.Header.Set(middleware.TenantHeader, "t")
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		return rec
	}

	// Valid XGID → 200 + decoded position.
	rec := post(`{"xgid":"XGID=a--aB-BBA--acDa-Ab-db---BA:0:0:1:64:2:0:0:13:10"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("valid XGID: got %d (%s)", rec.Code, rec.Body)
	}
	var pos domain.Position
	if err := json.Unmarshal(rec.Body.Bytes(), &pos); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if pos.Dice != [2]int{6, 4} || pos.PlayerOnRoll != domain.Black {
		t.Fatalf("unexpected decode: dice=%v onRoll=%d", pos.Dice, pos.PlayerOnRoll)
	}
	if pos.Board.Points[13].Checkers != 4 || pos.Board.Points[13].Color != domain.Black {
		t.Fatalf("point 13 wrong: %+v", pos.Board.Points[13])
	}

	// Invalid XGID → 400 invalid envelope.
	bad := post(`{"xgid":"not-a-valid-xgid"}`)
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("invalid XGID: got %d (%s), want 400", bad.Code, bad.Body)
	}
}
