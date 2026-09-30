package server

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// fuzzExcluded names the route families the fuzzer leaves alone: imports and
// exports stream files and run long, the gammonnet family runs the engine
// (seconds per call, which would starve the search), and /ops/ is outside the
// tenant surface. Everything else — the JSON decoding of every rpc handler
// and the tenant gate in front of them — is fuzzed.
var fuzzExcluded = []string{"/v1/imports.", "/v1/exports.", "/v1/gammonnet.", "/ops/"}

var (
	fuzzSrvOnce  sync.Once
	fuzzSrv      http.Handler
	fuzzSrvPaths []string
	fuzzSrvErr   error
)

// fuzzServer is built once per fuzz worker: a fresh store per input would
// cost more than the request. State written by one input is seen by the
// next, as it would be on a real daemon.
func fuzzServer(t *testing.T) (http.Handler, []string) {
	t.Helper()
	fuzzSrvOnce.Do(func() {
		st, err := sqlite.Open(context.Background(), ":memory:", nil)
		if err != nil {
			fuzzSrvErr = err
			return
		}
		srv, err := New(Options{Storage: st, Logger: slog.New(slog.DiscardHandler)})
		if err != nil {
			fuzzSrvErr = err
			return
		}
		fuzzSrv = srv.Handler()
	outer:
		for _, p := range srv.Paths() {
			for _, ex := range fuzzExcluded {
				if strings.HasPrefix(p, ex) {
					continue outer
				}
			}
			fuzzSrvPaths = append(fuzzSrvPaths, p)
		}
	})
	if fuzzSrvErr != nil {
		t.Fatal(fuzzSrvErr)
	}
	return fuzzSrv, fuzzSrvPaths
}

// FuzzRPC sends an arbitrary body under an arbitrary X-Tenant-ID to a route
// of the tenant surface. The daemon trusts the header but must still read it
// strictly (ADR-0005), and every handler decodes a body a client wrote. The
// contract: a malformed tenant is a 400; a well-formed one never produces a
// 5xx, a panic or a hang; every error is the JSON envelope whose code matches
// the status.
func FuzzRPC(f *testing.F) {
	f.Add(uint16(0), "1", []byte("{}"))
	f.Add(uint16(1), "", []byte("{}"))
	f.Add(uint16(2), "01", []byte(`{"id":1}`))
	f.Add(uint16(3), "1", []byte(`{"limit":-1,"offset":-5}`))
	f.Add(uint16(4), "9223372036854775807", []byte(`{"ids":[1,2,3]}`))
	f.Add(uint16(5), "1", []byte(`{"xgid":"XGID=-b----E-C---eE---c-e----B-:0:0:1:51:0:0:0:7:0"}`))
	f.Add(uint16(6), "1", []byte(`{"query":"s cube p>30 #prime"}`))
	f.Add(uint16(7), "1", []byte(`{"id":1,"text":"x","tags":["#a"]}`))
	f.Add(uint16(8), "1", []byte(`[`))
	f.Add(uint16(9), "-1", []byte(`null`))

	f.Fuzz(func(t *testing.T, route uint16, tenant string, body []byte) {
		h, paths := fuzzServer(t)
		path := paths[int(route)%len(paths)]

		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if tenant != "" {
			req.Header.Set(middleware.TenantHeader, tenant)
		}
		rec := httptest.NewRecorder()
		done := make(chan struct{})
		go func() {
			defer close(done)
			h.ServeHTTP(rec, req)
		}()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Fatalf("%s did not answer within 10s (tenant %q, body %q)", path, tenant, body)
		}

		n, err := storage.ParseTenant(tenant)
		validTenant := err == nil && n > 0
		if !validTenant && rec.Code != http.StatusBadRequest {
			t.Fatalf("%s with tenant %q: status %d, want 400", path, tenant, rec.Code)
		}
		if rec.Code >= 500 {
			t.Fatalf("%s: status %d on tenant %q, body %q\n%s", path, rec.Code, tenant, body, rec.Body.Bytes())
		}
		if rec.Code >= 400 {
			var env errorEnvelope
			if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil || env.Error.Code == "" {
				t.Fatalf("%s: status %d without an error envelope: %q", path, rec.Code, rec.Body.Bytes())
			}
			if got := statusForCode(env.Error.Code); got != rec.Code {
				t.Fatalf("%s: status %d does not match code %q (expects %d)", path, rec.Code, env.Error.Code, got)
			}
		}
	})
}
