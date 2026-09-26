package server

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/internal/server/metrics"
	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// TestWriteStorageError_SurfacesInServerLog: exports.sqlite on a cancelled
// context fails with a plain error; the client sees "internal error" while
// the server log line carries the real cause.
func TestWriteStorageError_SurfacesInServerLog(t *testing.T) {
	st, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	var buf strings.Builder
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	srv, err := New(Options{Storage: st, Metrics: metrics.New(), Logger: logger})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodPost, "/v1/exports.sqlite", nil).WithContext(ctx)
	req.Header.Set(middleware.TenantHeader, testTenant)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "context canceled") {
		t.Fatalf("the masked client body must not leak the real error: %s", rec.Body.String())
	}

	out := buf.String()
	if !strings.Contains(out, "level=ERROR") {
		t.Errorf("server log not at Error level for a 500:\n%s", out)
	}
	if !strings.Contains(out, "context canceled") {
		t.Errorf("server log missing the real cause (client-masked error must still be logged server-side):\n%s", out)
	}
}
