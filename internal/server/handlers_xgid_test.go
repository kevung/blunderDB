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
		req.Header.Set(middleware.TenantHeader, "1")
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

// TestPositionsFromXGIDWritesTheCrawfordSentinel: field 7 is the Crawford
// flag; one point away is away `0` after the Crawford game, `1` in it
// (CONTEXT.md, « Away score »).
func TestPositionsFromXGIDWritesTheCrawfordSentinel(t *testing.T) {
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
	post := func(route string, body any) *httptest.ResponseRecorder {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, route, strings.NewReader(string(raw)))
		req.Header.Set(middleware.TenantHeader, "1")
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		return rec
	}

	const board = "-B-CBBB---a---A---ABcbbbd-"
	for _, c := range []struct {
		name, xgid string
		want       [2]int
	}{
		{"after the Crawford game", "XGID=" + board + ":1:-1:1:21:3:6:0:7:10", [2]int{4, domain.PostCrawford}},
		{"in the Crawford game", "XGID=" + board + ":1:-1:1:21:3:6:1:7:10", [2]int{4, domain.Crawford}},
		// A 1-point match is its own Crawford game, field 7 or not; the DMP
		// after the Crawford game of a longer match is not.
		{"a 1-point match with field 7 = 0", "XGID=" + board + ":0:0:1:21:0:0:0:1:10", [2]int{domain.Crawford, domain.Crawford}},
		{"DMP after the Crawford game of a 7-point match", "XGID=" + board + ":1:-1:1:21:6:6:0:7:10", [2]int{domain.PostCrawford, domain.PostCrawford}},
	} {
		t.Run(c.name+"/fromXGID", func(t *testing.T) {
			rec := post("/v1/positions.fromXGID", map[string]string{"xgid": c.xgid})
			if rec.Code != http.StatusOK {
				t.Fatalf("got %d (%s)", rec.Code, rec.Body)
			}
			var pos domain.Position
			if err := json.Unmarshal(rec.Body.Bytes(), &pos); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if pos.Score != c.want {
				t.Errorf("away score = %v, want %v", pos.Score, c.want)
			}
		})
		t.Run(c.name+"/parseText", func(t *testing.T) {
			rec := post("/v1/positions.parseText", map[string]string{"text": c.xgid})
			if rec.Code != http.StatusOK {
				t.Fatalf("got %d (%s)", rec.Code, rec.Body)
			}
			var res struct {
				Position domain.Position `json:"position"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if res.Position.Score != c.want {
				t.Errorf("away score = %v, want %v", res.Position.Score, c.want)
			}
		})
	}
}

// TestPositionsFromOGID: fromOGID and parseText (reading an OGID like a
// pasted XGID) return what the same position's XGID returns, Crawford "C"
// included.
func TestPositionsFromOGID(t *testing.T) {
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
	post := func(route string, body any) *httptest.ResponseRecorder {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, route, strings.NewReader(string(raw)))
		req.Header.Set(middleware.TenantHeader, "1")
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		return rec
	}

	const start = "11jjjjjhhhccccc:ooddddd88866666:N0N:65:W::6:5:"
	for _, c := range []struct {
		name, ogid string
		want       [2]int
	}{
		{"in the Crawford game", start + "7C:", [2]int{2, domain.Crawford}},
		{"after the Crawford game", start + "7:", [2]int{2, domain.PostCrawford}},
		{"the canonical example", "cccccggggg:ddddiiiiii:N0N:63:W:IW:4:3:7:1:15", [2]int{4, 3}},
	} {
		t.Run(c.name+"/fromOGID", func(t *testing.T) {
			rec := post("/v1/positions.fromOGID", map[string]string{"ogid": c.ogid})
			if rec.Code != http.StatusOK {
				t.Fatalf("got %d (%s)", rec.Code, rec.Body)
			}
			var pos domain.Position
			if err := json.Unmarshal(rec.Body.Bytes(), &pos); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if pos.Score != c.want || pos.PlayerOnRoll != domain.White {
				t.Errorf("away score = %v, on roll %d; want %v, White", pos.Score, pos.PlayerOnRoll, c.want)
			}
		})
		t.Run(c.name+"/parseText", func(t *testing.T) {
			rec := post("/v1/positions.parseText", map[string]string{"text": "OGID=" + c.ogid})
			if rec.Code != http.StatusOK {
				t.Fatalf("got %d (%s)", rec.Code, rec.Body)
			}
			var res struct {
				Position domain.Position `json:"position"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if res.Position.Score != c.want {
				t.Errorf("away score = %v, want %v", res.Position.Score, c.want)
			}
		})
	}

	if bad := post("/v1/positions.fromOGID", map[string]string{"ogid": "11ccc!c:66666:N0N"}); bad.Code != http.StatusBadRequest {
		t.Fatalf("invalid OGID: got %d (%s), want 400", bad.Code, bad.Body)
	}
}
