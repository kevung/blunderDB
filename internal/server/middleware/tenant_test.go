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
	return probeTenantMode(public, path, header, false)
}

// probeTenantMode drives the middleware with the single-tenant rule on or off
// (#240: the SQLite backend has no tenant column and must refuse the tenants it
// cannot actually separate).
func probeTenantMode(public map[string]bool, path, header string, singleTenant bool) (tenantProbe, int) {
	var p tenantProbe
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p.ran = true
		p.scope, p.scopeOK = TenantFromContext(r.Context())
		p.numeric, _ = storage.TenantFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})
	mw := Tenant(public, singleTenant, func(w http.ResponseWriter, _ *http.Request, msg string) {
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
// plausibly send: whitespace is trimmed, blank means missing, a positive
// decimal integer is the tenant, and anything else is rejected as invalid —
// the daemon never shapes a value into a tenant. Before ADR-0005's 2026-09-03
// amendment "tenant-a", "a b" and every other name passed through and landed
// on tenant 0, so all named tenants shared one set of rows.
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
		{"numeric", "7", true, "7", 7},
		{"max int64", "9223372036854775807", true, "9223372036854775807", 1<<63 - 1},
		{"padded name", " tenant-a ", false, "", 0},
		{"alice", "alice", false, "", 0},
		{"default", "default", false, "", 0},
		{"inner spaces", "a b", false, "", 0},
		{"very long", long, false, "", 0},
		{"non-ASCII", "société-é-日本-🎲", false, "", 0},
		{"zero", "0", false, "", 0},
		{"negative", "-1", false, "", 0},
		{"signed", "+1", false, "", 0},
		{"leading zero", "007", false, "", 0},
		{"decimal point", "1.0", false, "", 0},
		{"overflow", "9223372036854775808", false, "", 0},
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
				if !strings.Contains(p.reject, TenantHeader) {
					t.Errorf("rejection %q does not name the header", p.reject)
				}
				if strings.TrimSpace(tc.header) != "" && !strings.Contains(p.reject, storage.TenantFormat) {
					t.Errorf("rejection %q does not name the expected format", p.reject)
				}
				if len(p.reject) > 512 {
					t.Errorf("rejection is %d bytes long: the header value is reflected wholesale", len(p.reject))
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

// A single-tenant backend answers for tenant 1 and refuses the rest, rather
// than serving every caller the same rows behind a header that says otherwise.
func TestTenant_SingleTenantBackendRefusesTheOthers(t *testing.T) {
	public := map[string]bool{"/healthz": true}

	if p, code := probeTenantMode(public, "/v1/positions.list", SingleTenantID, true); !p.ran || code != http.StatusNoContent {
		t.Errorf("tenant %s must pass on a single-tenant backend: ran=%v code=%d", SingleTenantID, p.ran, code)
	}

	p, code := probeTenantMode(public, "/v1/positions.list", "2", true)
	if p.ran || code != http.StatusBadRequest {
		t.Errorf("tenant 2 must be refused on a single-tenant backend: ran=%v code=%d", p.ran, code)
	}
	if !strings.Contains(p.reject, "single-tenant") || !strings.Contains(p.reject, "PostgreSQL") {
		t.Errorf("the refusal must say why and what to do instead, got %q", p.reject)
	}

	// The same request is fine on a multi-tenant backend.
	if p, code := probeTenantMode(public, "/v1/positions.list", "2", false); !p.ran || code != http.StatusNoContent {
		t.Errorf("tenant 2 must pass on a multi-tenant backend: ran=%v code=%d", p.ran, code)
	}

	// A probe stays public either way.
	if p, code := probeTenantMode(public, "/healthz", "", true); !p.ran || code != http.StatusNoContent {
		t.Errorf("/healthz must stay public: ran=%v code=%d", p.ran, code)
	}
}
