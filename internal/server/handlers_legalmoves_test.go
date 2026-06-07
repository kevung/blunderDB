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

func TestPositionsLegalMoves(t *testing.T) {
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
		req := httptest.NewRequest(http.MethodPost, "/v1/positions.legalMoves", strings.NewReader(body))
		req.Header.Set(middleware.TenantHeader, "t")
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		return rec
	}

	// Opening position, roll 3-1 → some legal plays, each a complete (full-board) play.
	pos := domain.InitializePosition()
	pos.Dice = [2]int{3, 1}
	body, _ := json.Marshal(map[string]any{"position": pos})
	rec := post(string(body))
	if rec.Code != http.StatusOK {
		t.Fatalf("legalMoves: got %d (%s)", rec.Code, rec.Body)
	}
	var plays []domain.LegalPlay
	if err := json.Unmarshal(rec.Body.Bytes(), &plays); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(plays) == 0 {
		t.Fatal("opening 3-1 should have legal plays")
	}
	for _, p := range plays {
		var b, w int
		for i := 0; i <= 25; i++ {
			switch p.Result.Board.Points[i].Color {
			case domain.Black:
				b += p.Result.Board.Points[i].Checkers
			case domain.White:
				w += p.Result.Board.Points[i].Checkers
			}
		}
		if b+p.Result.Board.Bearoff[domain.Black] != 15 || w+p.Result.Board.Bearoff[domain.White] != 15 {
			t.Fatalf("a play does not conserve checkers: %s", p.Notation)
		}
	}

	// Missing position → 400.
	if bad := post(`{}`); bad.Code != http.StatusBadRequest {
		t.Fatalf("missing position: got %d, want 400", bad.Code)
	}
}
