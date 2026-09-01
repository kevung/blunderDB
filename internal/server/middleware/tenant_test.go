package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// tenantProbe runs one request through Tenant and reports what the inner
// handler saw: whether it ran, the scope string, and the numeric tenant.
type tenantProbe struct {
	ran     bool
	scope   string
	scopeOK bool
	numeric int64
	reject  string
}

func probeTenant(public map[string]bool, path, header string) (tenantProbe, int) {
	var p tenantProbe
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p.ran = true
		p.scope, p.scopeOK = TenantFromContext(r.Context())
		p.numeric, _ = storage.TenantFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})
	mw := Tenant(public, func(w http.ResponseWriter, _ *http.Request, msg string) {
		p.reject = msg
		w.WriteHeader(http.StatusBadRequest)
	})(inner)
	req := httptest.NewRequest(http.MethodPost, path, nil)
	if header != "" {
		req.Header.Set(TenantHeader, header)
	}
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	return p, rec.Code
}

func TestTenant_PublicPathsBypassTheGate(t *testing.T) {
	public := map[string]bool{"/healthz": true}
	p, code := probeTenant(public, "/healthz", "")
	if !p.ran || code != http.StatusNoContent {
		t.Fatalf("public path without header: ran=%v code=%d", p.ran, code)
	}
	if p.scopeOK {
		t.Errorf("public path carried a tenant %q into the context", p.scope)
	}
	// A path that is not in the set is gated, even if it looks like ops.
	if p, code := probeTenant(public, "/metrics", ""); p.ran || code != http.StatusBadRequest {
		t.Errorf("/metrics outside the public set: ran=%v code=%d, want rejected", p.ran, code)
	}
}

func TestTenant_MissingHeaderIsRejected(t *testing.T) {
	p, code := probeTenant(nil, "/v1/positions.list", "")
	if p.ran || code != http.StatusBadRequest {
		t.Fatalf("ran=%v code=%d, want rejected 400", p.ran, code)
	}
	if !strings.Contains(p.reject, TenantHeader) {
		t.Errorf("rejection %q does not name the header", p.reject)
	}
}

// TestTenant_MalformedValues pins the contract for values a proxy could
// plausibly send: whitespace is trimmed, blank means missing, and anything
// else is an opaque identifier taken verbatim — the daemon shapes nothing.
func TestTenant_MalformedValues(t *testing.T) {
	long := strings.Repeat("x", 64*1024)
	cases := []struct {
		name    string
		header  string
		ran     bool
		scope   string
		numeric int64
	}{
		{"spaces only", "   ", false, "", 0},
		{"tab only", "\t", false, "", 0},
		{"padded numeric", "  42  ", true, "42", 42},
		{"padded name", " tenant-a ", true, "tenant-a", 0},
		{"inner spaces kept", "a b", true, "a b", 0},
		{"very long", long, true, long, 0},
		{"non-ASCII", "société-é-日本-🎲", true, "société-é-日本-🎲", 0},
		{"numeric", "7", true, "7", 7},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, code := probeTenant(nil, "/v1/positions.list", tc.header)
			if p.ran != tc.ran {
				t.Fatalf("ran = %v, want %v (code %d, reject %q)", p.ran, tc.ran, code, p.reject)
			}
			if !tc.ran {
				if code != http.StatusBadRequest {
					t.Errorf("code = %d, want 400", code)
				}
				return
			}
			if p.scope != tc.scope || !p.scopeOK {
				t.Errorf("scope = %q (ok=%v), want %q", p.scope, p.scopeOK, tc.scope)
			}
			if p.numeric != tc.numeric {
				t.Errorf("numeric tenant = %d, want %d", p.numeric, tc.numeric)
			}
		})
	}
}
