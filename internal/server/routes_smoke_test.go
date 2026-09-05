package server

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// This file is a smoke test driven by the routing table itself, so that a
// route added to domainRoutes is covered the day it lands, without anyone
// writing a per-route test. It checks the wiring, not the semantics:
//
//   - no /v1 route is reachable without X-Tenant-ID (400 + error envelope);
//   - every /v1 route answers a tenant-scoped POST with body {} without
//     falling through to the catch-all (404 "unknown route"), crashing (500,
//     or a recovered panic — which also surfaces as a 500) or answering in
//     an undeclared shape;
//   - the Content-Type matches the handler's shape: NDJSON for rpcStream,
//     JSON for rpc/rpcVoid, and the handful of hand-written handlers that
//     return something else are named explicitly.
//
// A validation answer (400 invalid) or a "not found" for id 0 (404 not_found
// from the storage layer) are both fine: they mean the handler ran and
// rejected an empty request on purpose. The catch-all 404 is recognised by
// its message and is a failure.

// handlerKind/kindOf live in routes.go (production code): server.go's
// streamingPaths reuses the exact same classification to give a streaming
// route a longer read/write deadline (#234), so this test and that behaviour
// can never quietly drift apart.

// customContentTypes names the hand-written handlers and the Content-Type
// they are allowed to answer with on success. Keep it exhaustive: a custom
// handler missing here fails the test, which is the point — its answer shape
// has to be declared somewhere.
var customContentTypes = map[string][]string{
	"/v1/imports.json":                    {ndjsonContentType},
	"/v1/imports.xg":                      {ndjsonContentType},
	"/v1/imports.gnubg":                   {ndjsonContentType},
	"/v1/imports.bgf":                     {ndjsonContentType},
	"/v1/imports.db":                      {ndjsonContentType},
	"/v1/imports.position":                {ndjsonContentType},
	"/v1/imports.cancel":                  {"application/json"},
	"/v1/exports.json":                    {ndjsonContentType},
	"/v1/exports.sqlite":                  {"application/octet-stream"},
	"/v1/matches.exportMat":               {"text/plain"},
	"/ops/tenant.purge":                   {"application/json"},
	"/ops/maintenance.vacuum":             {"application/json"},
	"/v1/gammonnet.analyzeMissing":        {ndjsonContentType},
	"/v1/gammonnet.analyzeMissing.cancel": {"application/json"},
	"/v1/gammonnet.sweepStale":            {ndjsonContentType},
	// search.query streams positions like search.find; it is hand-written only
	// so an unreadable query is refused before the 200 is committed (B.18).
	"/v1/search.query": {ndjsonContentType},

	// These three are ordinary rpc-shaped JSON handlers, but wrapped with
	// withIdempotency (#236) — the closure kindOf sees at runtime is
	// withIdempotency's, not rpc's, so they classify as kindCustom despite
	// answering exactly like any other rpc route.
	"/v1/collections.create": {"application/json"},
	"/v1/tournaments.create": {"application/json"},
	"/v1/anki.reviewCard":    {"application/json"},
}

// smokeServer builds the Server (not just the httptest wrapper) so the test
// can walk the same route table the mux was built from.
func smokeServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	st, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	srv, err := New(Options{
		Storage: st,
		// The probe deliberately trips validation on every route; the
		// request log would drown the failures that matter.
		Logger: slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

func doPost(t *testing.T, url string, tenant string, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if tenant != "" {
		req.Header.Set(middleware.TenantHeader, tenant)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func decodeEnvelope(t *testing.T, body []byte) errorEnvelope {
	t.Helper()
	var env errorEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("body is not an error envelope: %v\n%s", err, body)
	}
	if env.Error.Code == "" {
		t.Fatalf("error envelope without code:\n%s", body)
	}
	return env
}

func mediaType(ct string) string {
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = ct[:i]
	}
	return strings.TrimSpace(ct)
}

// TestRoutesSmoke_TableIsComplete pins the two things the smoke test relies
// on: Paths() enumerates every callable route in the table (the /v1 tenant
// surface plus the /ops/ family, and nothing hiding behind a third prefix),
// and every hand-written handler declares its answer shape.
func TestRoutesSmoke_TableIsComplete(t *testing.T) {
	srv, _ := smokeServer(t)
	byPath := map[string]route{}
	callable := append(append([]route{}, srv.domainRoutes()...), srv.opsRoutes()...)
	for _, rt := range callable {
		if !strings.HasPrefix(rt.pattern, "/v1/") && !strings.HasPrefix(rt.pattern, "/ops/") {
			t.Errorf("callable route %q is under neither /v1/ nor /ops/ — Paths() would not list it", rt.pattern)
		}
		if rt.method != http.MethodPost {
			t.Errorf("domain route %q uses %s; the /v1 surface is POST-only", rt.pattern, rt.method)
		}
		if _, dup := byPath[rt.pattern]; dup {
			t.Errorf("route %q registered twice", rt.pattern)
		}
		byPath[rt.pattern] = rt
	}
	if len(srv.Paths()) != len(byPath) {
		t.Errorf("Paths() lists %d routes, table has %d", len(srv.Paths()), len(byPath))
	}
	for path, rt := range byPath {
		_, declared := customContentTypes[path]
		if kindOf(rt.handler) == kindCustom && !declared {
			t.Errorf("%s is a hand-written handler with no entry in customContentTypes", path)
		}
		if kindOf(rt.handler) != kindCustom && declared {
			t.Errorf("%s is an rpc handler but is listed in customContentTypes", path)
		}
	}
	for path := range customContentTypes {
		if _, ok := byPath[path]; !ok {
			t.Errorf("customContentTypes names %s, which is not a route", path)
		}
	}
}

// TestRoutesSmoke_TenantRequired: every /v1 route is behind the tenant gate.
func TestRoutesSmoke_TenantRequired(t *testing.T) {
	srv, ts := smokeServer(t)
	for _, path := range srv.Paths() {
		t.Run(strings.TrimPrefix(path, "/v1/"), func(t *testing.T) {
			resp := doPost(t, ts.URL+path, "", "{}")
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400\n%s", resp.StatusCode, body)
			}
			if ct := mediaType(resp.Header.Get("Content-Type")); ct != "application/json" {
				t.Errorf("content-type = %q, want application/json", ct)
			}
			env := decodeEnvelope(t, body)
			if env.Error.Code != CodeInvalid {
				t.Errorf("code = %q, want %q", env.Error.Code, CodeInvalid)
			}
			if !strings.Contains(env.Error.Message, middleware.TenantHeader) {
				t.Errorf("message %q does not name the missing header", env.Error.Message)
			}
		})
	}
}

// TestRoutesSmoke_EmptyBody: with a tenant and body {}, every route answers
// something the handler chose — never the catch-all, never a 500.
func TestRoutesSmoke_EmptyBody(t *testing.T) {
	srv, ts := smokeServer(t)
	kinds := map[string]handlerKind{}
	for _, rt := range srv.domainRoutes() {
		kinds[rt.pattern] = kindOf(rt.handler)
	}
	for _, path := range srv.Paths() {
		t.Run(strings.TrimPrefix(path, "/v1/"), func(t *testing.T) {
			resp := doPost(t, ts.URL+path, testTenant, "{}")
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			ct := mediaType(resp.Header.Get("Content-Type"))

			if resp.StatusCode >= 500 {
				t.Fatalf("status = %d: handler crashed or mis-wired on {}\n%s", resp.StatusCode, body)
			}
			if resp.StatusCode >= 400 {
				// Every error, whatever the handler kind, is the JSON envelope.
				if ct != "application/json" {
					t.Fatalf("error content-type = %q, want application/json\n%s", ct, body)
				}
				env := decodeEnvelope(t, body)
				if resp.StatusCode == http.StatusNotFound && strings.HasPrefix(env.Error.Message, "unknown route") {
					t.Fatalf("fell through to the catch-all: %s", env.Error.Message)
				}
				if got := statusForCode(env.Error.Code); got != resp.StatusCode {
					t.Errorf("status %d does not match code %q (expects %d)", resp.StatusCode, env.Error.Code, got)
				}
				return
			}

			// Success: the shape must match the handler's kind.
			switch kinds[path] {
			case kindStream:
				if ct != ndjsonContentType {
					t.Fatalf("rpcStream content-type = %q, want %q", ct, ndjsonContentType)
				}
				for _, line := range strings.Split(strings.TrimSpace(string(body)), "\n") {
					if line == "" {
						continue
					}
					if !json.Valid([]byte(line)) {
						t.Fatalf("ndjson line is not JSON: %q", line)
					}
				}
			case kindJSON:
				if ct != "application/json" {
					t.Fatalf("rpc content-type = %q, want application/json", ct)
				}
				if !json.Valid(body) {
					t.Fatalf("rpc body is not JSON:\n%s", body)
				}
			case kindCustom:
				allowed := customContentTypes[path]
				ok := false
				for _, a := range allowed {
					ok = ok || ct == a
				}
				if !ok {
					t.Fatalf("custom handler content-type = %q, want one of %v", ct, allowed)
				}
			}
		})
	}
}
