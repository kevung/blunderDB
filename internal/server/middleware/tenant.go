package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// TenantHeader is the request header carrying the tenant identifier.
//
// Authentication is delegated to an upstream reverse-proxy: the daemon trusts
// this header and must never be exposed directly to the public internet. See
// docs/adr/0005 (and its 2026-09-03 amendment for the header's format).
const TenantHeader = "X-Tenant-ID"

type tenantKey struct{}

// Tenant extracts the X-Tenant-ID header and stores it in the request context.
// Requests to paths outside public without a tenant are rejected; the
// rejection itself is delegated to errFn so the server controls the error
// envelope. public is the set of paths reachable without a tenant (the ops
// endpoints) — the server derives it from its routing table so the two can
// never drift apart.
//
// The header value is a tenant identifier as storage.ParseTenant defines it:
// a positive decimal integer. Surrounding whitespace is trimmed (a proxy that
// pads the value must not create a distinct tenant), a value that is blank
// once trimmed counts as missing, and anything that is not a positive decimal
// integer is rejected — "alice", "default", "0" and "1.0" alike. The daemon
// used to pass such values through verbatim, and every backend then mapped
// them to tenant 0, so all named tenants shared one set of rows. Mapping a
// name to its integer is the proxy's job (ADR-0005).
func Tenant(public map[string]bool, errFn func(http.ResponseWriter, *http.Request, string)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if public[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}
			tenant := strings.TrimSpace(r.Header.Get(TenantHeader))
			if tenant == "" {
				errFn(w, r, "missing or empty "+TenantHeader+" header")
				return
			}
			numeric, err := storage.ParseTenant(tenant)
			if err != nil {
				errFn(w, r, TenantHeader+" header must be "+storage.TenantFormat+", got "+quoteHeader(tenant))
				return
			}
			ctx := context.WithValue(r.Context(), tenantKey{}, tenant)
			// Also carry the numeric tenant so the PostgreSQL backend can set the
			// app.tenant_id GUC when RLS is enabled (no-op otherwise).
			ctx = storage.WithTenant(ctx, numeric)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// quoteHeader quotes a header value for an error message, truncating it so a
// multi-kilobyte header cannot be reflected wholesale into the response.
func quoteHeader(v string) string {
	const maxRunes = 64
	if r := []rune(v); len(r) > maxRunes {
		v = string(r[:maxRunes]) + "…"
	}
	return strconv.Quote(v)
}

// TenantFromContext returns the tenant scope stored by the Tenant middleware.
// The boolean is false when no tenant is present (e.g. public endpoints).
func TenantFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(tenantKey{}).(string)
	return v, ok
}
