package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// postAs issues a POST with the given tenant header and an optional JSON
// body. It is the untagged twin of postTenant (handlers_tenant_test.go, which
// only builds with -tags postgres), so the SQLite tests here run on the
// default path.
func postAs(t *testing.T, ts *httptest.Server, tenant, path string, body any) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req, _ := http.NewRequest(http.MethodPost, ts.URL+path, &buf)
	req.Header.Set(middleware.TenantHeader, tenant)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

// TestSessionScopedByTenant is the HTTP half of issue #156 on the SQLite
// backend: tenant 1 saves a session, tenant 2 loads an empty one, tenant 1
// reads its own back. Before schema 2.16.0 the session was six rows of the
// global metadata table.
func TestSessionScopedByTenant(t *testing.T) {
	ts := newTestServer(t)
	assertSessionIsolated(t, ts)
}

// assertSessionIsolated runs the session isolation scenario against ts; the
// PostgreSQL test (handlers_tenant_test.go) reuses it.
func assertSessionIsolated(t *testing.T, ts *httptest.Server) {
	t.Helper()
	saved := storage.SessionState{
		LastSearchCommand: "cube", LastSearchPosition: "xgid-one",
		LastPositionIndex: 2, LastPositionIDs: []int64{4, 5}, HasActiveSearch: true,
		ViewsJSON: `{"tabs":["one"]}`,
	}
	resp := postAs(t, ts, "1", "/v1/session.save", sessionSaveReq{State: saved})
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("tenant 1 session.save status = %d, want 200", resp.StatusCode)
	}

	load := func(tenant string) storage.SessionState {
		t.Helper()
		resp := postAs(t, ts, tenant, "/v1/session.load", nil)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("tenant %s session.load status = %d, want 200", tenant, resp.StatusCode)
		}
		var got storage.SessionState
		if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		return got
	}

	if got := load("1"); got.LastSearchCommand != saved.LastSearchCommand || got.ViewsJSON != saved.ViewsJSON ||
		got.LastPositionIndex != saved.LastPositionIndex || len(got.LastPositionIDs) != 2 || !got.HasActiveSearch {
		t.Errorf("tenant 1 reads back %+v, want %+v", got, saved)
	}
	if got := load("2"); got.LastSearchCommand != "" || got.ViewsJSON != "" || got.HasActiveSearch || len(got.LastPositionIDs) != 0 {
		t.Errorf("tenant 2 sees tenant 1's session: %+v", got)
	}

	// Clearing tenant 2 (which has nothing) leaves tenant 1's session alone.
	resp = postAs(t, ts, "2", "/v1/session.clear", nil)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("tenant 2 session.clear status = %d, want 200", resp.StatusCode)
	}
	if got := load("1"); got.LastSearchCommand != saved.LastSearchCommand {
		t.Errorf("tenant 2's clear wiped tenant 1's session: %+v", got)
	}
}

// TestMetadataWriteRoutesGone pins the removal of metadata.load, metadata.save
// and metadata.setVersion from /v1 (#156): each now falls through to the
// catch-all — 404 with the error envelope — while metadata.version and
// metadata.counts still answer.
func TestMetadataWriteRoutesGone(t *testing.T) {
	ts := newTestServer(t)
	for _, path := range []string{"/v1/metadata.load", "/v1/metadata.save", "/v1/metadata.setVersion"} {
		resp := postAs(t, ts, "1", path, map[string]any{"version": "9.9.9", "metadata": map[string]string{"x": "y"}})
		var env errorEnvelope
		err := json.NewDecoder(resp.Body).Decode(&env)
		resp.Body.Close()
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if resp.StatusCode != http.StatusNotFound || env.Error.Code != CodeNotFound {
			t.Errorf("%s: status %d code %q, want 404 %q", path, resp.StatusCode, env.Error.Code, CodeNotFound)
		}
	}
	for _, path := range []string{"/v1/metadata.version", "/v1/metadata.counts"} {
		resp := postAs(t, ts, "1", path, nil)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s: status %d, want 200", path, resp.StatusCode)
		}
	}
}
