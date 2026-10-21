package server

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// An id that names no collection is the client's mistake on every route that
// takes one, never an internal error.
func TestCollectionRoutes_UnknownIDIsNotFound(t *testing.T) {
	st, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	srv, err := New(Options{Storage: st, Logger: slog.New(slog.DiscardHandler)})
	if err != nil {
		t.Fatal(err)
	}
	h := srv.Handler()
	for _, route := range []string{"get", "freeze", "evaluate", "setFilter", "update", "delete"} {
		req := httptest.NewRequest(http.MethodPost, "/v1/collections."+route, bytes.NewReader([]byte(`{"id":1,"name":"x"}`)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-ID", "1")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Errorf("collections.%s: status %d, want 404", route, rec.Code)
		}
	}
}

// Freezing a hand-made collection is refused as a client error, not a 500.
func TestCollectionsFreeze_NotLivingIsClientError(t *testing.T) {
	st, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	srv, err := New(Options{Storage: st, Logger: slog.New(slog.DiscardHandler)})
	if err != nil {
		t.Fatal(err)
	}
	h := srv.Handler()
	post := func(path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-ID", "1")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	if rec := post("/v1/collections.create", `{"name":"plain"}`); rec.Code != http.StatusOK {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	if rec := post("/v1/collections.freeze", `{"id":1}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("freeze: status %d, want 400: %s", rec.Code, rec.Body.String())
	}
}
