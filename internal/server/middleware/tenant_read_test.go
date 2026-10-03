package middleware

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// probeReadTenants sends one request as tenant 1 carrying lines as its
// X-Read-Tenants header lines, and returns the read set the handler saw (nil
// when the middleware refused), the refusal and the status.
func probeReadTenants(t *testing.T, lines []string, singleTenant, trusted bool) (storage.ReadTenants, string, int) {
	t.Helper()
	var set storage.ReadTenants
	var reject string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		set = ReadTenantsFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})
	mw := Tenant(nil, singleTenant, trusted, func(w http.ResponseWriter, _ *http.Request, msg string) {
		reject = msg
		w.WriteHeader(http.StatusBadRequest)
	})(inner)
	req := httptest.NewRequest(http.MethodPost, "/v1/across.matchesList", nil)
	req.Header.Set(TenantHeader, "1")
	for _, l := range lines {
		req.Header.Add(ReadTenantsHeader, l)
	}
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	return set, reject, rec.Code
}

func TestReadTenants_Accepted(t *testing.T) {
	for _, tc := range []struct {
		lines []string
		want  []string
	}{
		{nil, []string{"1"}},
		{[]string{""}, []string{"1"}},
		{[]string{"   "}, []string{"1"}},
		{[]string{"2"}, []string{"1", "2"}},
		{[]string{" 3 , 2,3, 1 "}, []string{"1", "3", "2"}},
		{[]string{"9223372036854775807"}, []string{"1", "9223372036854775807"}},
		// The bound counts distinct tenants: duplicates past it are no excess.
		{[]string{strings.Repeat("2,", storage.MaxReadTenants) + "3"}, []string{"1", "2", "3"}},
	} {
		set, reject, code := probeReadTenants(t, tc.lines, false, true)
		if code != http.StatusNoContent || !slices.Equal([]string(set), tc.want) {
			t.Errorf("%q: status %d (%s), set %v; want %v", tc.lines, code, reject, set, tc.want)
		}
	}
}

func TestReadTenants_Refused(t *testing.T) {
	for _, tc := range []struct {
		name  string
		lines []string
	}{
		{"name", []string{"alice"}},
		{"negative", []string{"-2"}},
		{"plus sign", []string{"+2"}},
		{"int64 overflow", []string{"9223372036854775808"}},
		{"zero", []string{"0"}},
		{"leading zero", []string{"02"}},
		{"trailing comma", []string{"2,"}},
		{"empty item", []string{"2,,3"}},
		{"blank items", []string{" , "}},
		{"two lines", []string{"2", "3"}},
		{"two lines, one blank", []string{"2", ""}},
		{"too many", []string{distinctTenants(2, storage.MaxReadTenants)}},
	} {
		set, reject, code := probeReadTenants(t, tc.lines, false, true)
		if code != http.StatusBadRequest || set != nil {
			t.Errorf("%s %q: status %d, set %v; want 400", tc.name, tc.lines, code, set)
		}
		if !strings.Contains(reject, ReadTenantsHeader) {
			t.Errorf("%s: refusal does not name the header: %q", tc.name, reject)
		}
	}
}

// TestReadTenants_UntrustedRefusesNotIgnores: a daemon not started with
// --read-tenants refuses a non-blank header rather than drop it, so a proxy
// that forwards a client's value is seen at once.
func TestReadTenants_UntrustedRefusesNotIgnores(t *testing.T) {
	for _, lines := range [][]string{{"2"}, {"1"}, {"alice"}, {"2", "3"}} {
		set, reject, code := probeReadTenants(t, lines, false, false)
		if code != http.StatusBadRequest || set != nil || !strings.Contains(reject, "--read-tenants") {
			t.Errorf("untrusted %q: status %d, set %v, refusal %q; want 400 naming --read-tenants", lines, code, set, reject)
		}
	}
	if set, _, code := probeReadTenants(t, nil, false, false); code != http.StatusNoContent || !slices.Equal([]string(set), []string{"1"}) {
		t.Errorf("untrusted, no header: status %d, set %v; want the writer alone", code, set)
	}
}

func TestReadTenants_SingleTenant(t *testing.T) {
	if set, _, code := probeReadTenants(t, []string{"1"}, true, true); code != http.StatusNoContent || !slices.Equal([]string(set), []string{"1"}) {
		t.Errorf("SQLite listing 1: status %d, set %v", code, set)
	}
	if _, _, code := probeReadTenants(t, []string{"2"}, true, true); code != http.StatusBadRequest {
		t.Errorf("SQLite listing 2: status %d, want 400", code)
	}
}

// distinctTenants lists n distinct tenants from first on, comma-separated.
func distinctTenants(first, n int) string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = strconv.Itoa(first + i)
	}
	return strings.Join(parts, ",")
}

// TestReadTenants_BoundIncludesTheWriter: the 64 count X-Tenant-ID, so 63
// others pass and 64 others do not; listing the writer again costs nothing.
func TestReadTenants_BoundIncludesTheWriter(t *testing.T) {
	if _, reject, code := probeReadTenants(t, []string{distinctTenants(2, storage.MaxReadTenants-1)}, false, true); code != http.StatusNoContent {
		t.Errorf("%d others: status %d (%s), want accepted", storage.MaxReadTenants-1, code, reject)
	}
	if _, _, code := probeReadTenants(t, []string{"1," + distinctTenants(2, storage.MaxReadTenants-1)}, false, true); code != http.StatusNoContent {
		t.Errorf("%d others plus the writer: status %d, want accepted", storage.MaxReadTenants-1, code)
	}
	if _, _, code := probeReadTenants(t, []string{distinctTenants(2, storage.MaxReadTenants)}, false, true); code != http.StatusBadRequest {
		t.Errorf("%d others: status %d, want 400", storage.MaxReadTenants, code)
	}
}
