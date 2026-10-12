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
	"github.com/kevung/blunderdb/pkg/blunderdb/capture"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// The daemon reads a capture with the package the GUI and the CLI use, and
// refuses an image without a board as a client error.
func TestPositionsFromImage(t *testing.T) {
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
	post := func(data []byte) *httptest.ResponseRecorder {
		body := `{"data":"` + base64.StdEncoding.EncodeToString(data) + `"}`
		req := httptest.NewRequest(http.MethodPost, "/v1/positions.fromImage", strings.NewReader(body))
		req.Header.Set(middleware.TenantHeader, "1")
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		return rec
	}

	img, err := os.ReadFile("../../pkg/blunderdb/capture/testdata/xg-dice62.jpg")
	if err != nil {
		t.Fatal(err)
	}
	rec := post(img)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var rd capture.Reading
	if err := json.Unmarshal(rec.Body.Bytes(), &rd); err != nil {
		t.Fatal(err)
	}
	if got := domain.EncodeXGIDBoard(&rd.Position); got != "a--abBC-C--AbDa-aaAd-b--A-" {
		t.Errorf("board %s", got)
	}
	if rec := post([]byte("not an image")); rec.Code != http.StatusBadRequest {
		t.Errorf("garbage: status %d, want 400", rec.Code)
	}
}
