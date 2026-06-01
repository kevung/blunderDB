package middleware

import (
	"context"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// TenantHeader is the request header carrying the opaque tenant identifier.
//
// Authentication is delegated to an upstream reverse-proxy: the daemon trusts
// this header and must never be exposed directly to the public internet. See
// tasks/headless/06-serve-http.md.
const TenantHeader = "X-Tenant-ID"

type tenantKey struct{}

// publicPaths are reachable without a tenant header (ops endpoints).
var publicPaths = map[string]bool{
	"/healthz": true,
	"/readyz":  true,
	"/metrics": true,
}

// Tenant extracts the X-Tenant-ID header and stores it in the request context.
// Requests to non-public paths without a tenant are rejected; the rejection
// itself is delegated to errFn so the server controls the error envelope.
func Tenant(errFn func(http.ResponseWriter, *http.Request, string)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if publicPaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}
			tenant := r.Header.Get(TenantHeader)
			if tenant == "" {
				errFn(w, r, "missing or empty "+TenantHeader+" header")
				return
			}
			ctx := context.WithValue(r.Context(), tenantKey{}, tenant)
			// Also carry the numeric tenant so the PostgreSQL backend can set the
			// app.tenant_id GUC when RLS is enabled (no-op otherwise).
			ctx = storage.WithTenant(ctx, storage.ParseTenant(tenant))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// TenantFromContext returns the tenant scope stored by the Tenant middleware.
// The boolean is false when no tenant is present (e.g. public endpoints).
func TenantFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(tenantKey{}).(string)
	return v, ok
}
