package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/internal/server/metrics"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// The page's files load without a tenant (a browser reaching the daemon without a proxy must
// not get a blank page), while a data call still demands one.
func TestWebFilesNeedNoTenant(t *testing.T) {
	st, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	srv, err := New(Options{Storage: st, Metrics: metrics.New(), EnableWebUI: true})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	page, err := http.Get(ts.URL + "/app/")
	if err != nil {
		t.Fatal(err)
	}
	defer page.Body.Close()
	if page.StatusCode != http.StatusOK {
		t.Fatalf("GET /app/: %d, want 200", page.StatusCode)
	}
	body := new(strings.Builder)
	buf := make([]byte, 4096)
	for n, _ := page.Body.Read(buf); n > 0; n, _ = page.Body.Read(buf) {
		body.Write(buf[:n])
	}
	i := strings.Index(body.String(), "/assets/")
	if i < 0 {
		t.Skip("built page names no asset")
	}
	rest := body.String()[i:]
	asset := rest[:strings.IndexAny(rest, `"'`)]
	res, err := http.Get(ts.URL + "/app" + asset)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Errorf("asset %s without tenant: %d, want 200", asset, res.StatusCode)
	}

	call, err := http.Post(ts.URL+"/v1/metadata.counts", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	call.Body.Close()
	if call.StatusCode != http.StatusBadRequest {
		t.Errorf("/v1 without tenant: %d, want 400", call.StatusCode)
	}
}
