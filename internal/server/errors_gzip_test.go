package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/internal/server/middleware"
)

// The cause of an internal error must reach the log line whether or not a
// wrapping writer (compression) sits between the handler and the logger.
func TestInternalErrorCauseIsLoggedWithAndWithoutGzip(t *testing.T) {
	for _, tc := range []struct {
		name, acceptEncoding string
	}{
		{"plain", ""},
		{"gzip", "gzip"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf strings.Builder
			logger := slog.New(slog.NewTextHandler(&buf, nil))
			var h http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				writeStorageError(w, errors.New("database is locked (5)"))
			})
			h = middleware.Compress(h)
			h = middleware.Logging(logger, map[string]bool{}, nil)(h)

			req := httptest.NewRequest(http.MethodGet, "/v1/x", nil)
			if tc.acceptEncoding != "" {
				req.Header.Set("Accept-Encoding", tc.acceptEncoding)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			out := buf.String()
			if !strings.Contains(out, "level=ERROR") || !strings.Contains(out, "database is locked (5)") {
				t.Errorf("cause lost from the log line:\n%s", out)
			}
		})
	}
}

type testKeyedReq struct{}

func (testKeyedReq) readKey() readKey { return readKey{} }

// A conditional read route wraps the writer in etagWriter; the cause of the
// handler's internal error must still reach the log.
func TestConditionalReadInternalErrorCauseIsLogged(t *testing.T) {
	var buf strings.Builder
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	version := func(context.Context, string, readKey) (string, string, error) { return "v1", "", nil }
	var h http.Handler = rpcRead(version, func(context.Context, string, testKeyedReq) (int, error) {
		return 0, errors.New("database is locked (5)")
	})
	h = middleware.Logging(logger, map[string]bool{}, nil)(h)

	req := httptest.NewRequest(http.MethodPost, "/v1/x", strings.NewReader("{}"))
	h.ServeHTTP(httptest.NewRecorder(), req)

	out := buf.String()
	if !strings.Contains(out, "level=ERROR") || !strings.Contains(out, "database is locked (5)") {
		t.Errorf("cause lost from the log line:\n%s", out)
	}
}
