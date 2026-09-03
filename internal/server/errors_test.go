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

// TestWriteStorageError_SurfacesInServerLog is the end-to-end counterpart of
// middleware.TestLogging_SurfacesStashedErrorAtErrorLevel: a real handler
// (exports.sqlite, chosen because an already-cancelled request context makes
// ingest.SQLiteExporter.Export fail with a plain, non-sentinel error — see
// TestExportSQLiteErrorEnvelope in handlers_imports_test.go) hits
// writeStorageError's CodeInternal path, and the resulting server-side log
// line must carry the real error even though the client only ever sees the
// generic "internal error" body.
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
