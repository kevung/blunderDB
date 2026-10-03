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
// docs/adr/0005 (and its amendment for the header's format).
const TenantHeader = "X-Tenant-ID"

// ReadTenantsHeader is the request header listing the tenants a read spans
// besides X-Tenant-ID (ADR-0061): comma-separated tenants, in the order the
// answers come back. Like X-Tenant-ID, the daemon trusts it: the proxy that
// authenticates the caller decides whom it may read and writes this header;
// the daemon authorises nothing (ADR-0005). The daemon honours it only when
// started with --read-tenants; otherwise a non-blank value is refused. Only the
// /v1/across.* reads look at it — every other route, and every write, stays on
// X-Tenant-ID alone.
const ReadTenantsHeader = "X-Read-Tenants"

type tenantKey struct{}

type readTenantsKey struct{}

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
// integer is rejected — "alice", "default", "0" and "1.0" alike: mapping a
// name to its integer is the proxy's job, never the daemon's (ADR-0005).
// SingleTenantID is the only tenant a single-tenant backend answers for. It is
// "1" rather than "0" because a tenant is a POSITIVE integer (ADR-0005's
// amendment).
const SingleTenantID = "1"

// Tenant enforces the X-Tenant-ID header. singleTenant, when true, additionally
// refuses any value but SingleTenantID — see Options.SingleTenant for why a
// backend without a tenant column must say so rather than quietly serve
// everyone the same rows.
//
// trustReadTenants, when false (the default), refuses any request carrying a
// non-blank X-Read-Tenants: a proxy written before the header existed strips
// X-Tenant-ID but lets this one through, and honouring it would let any client
// behind such a proxy read other tenants. See Options.TrustReadTenants.
func Tenant(public map[string]bool, singleTenant, trustReadTenants bool, errFn func(http.ResponseWriter, *http.Request, string)) func(http.Handler) http.Handler {
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
			if singleTenant && tenant != SingleTenantID {
				errFn(w, r, "this daemon runs on a single-tenant backend (SQLite) and answers only for "+
					TenantHeader+" "+SingleTenantID+", got "+quoteHeader(tenant)+
					"; a deployment with real tenants needs the PostgreSQL backend")
				return
			}
			readSet, msg := readTenants(r, tenant, singleTenant, trustReadTenants)
			if msg != "" {
				errFn(w, r, msg)
				return
			}
			ctx := context.WithValue(r.Context(), tenantKey{}, tenant)
			ctx = context.WithValue(ctx, readTenantsKey{}, readSet)
			// Also carry the numeric tenant so the PostgreSQL backend can set the
			// app.tenant_id GUC when RLS is enabled (no-op otherwise).
			ctx = storage.WithTenant(ctx, numeric)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// readTenants builds the request's read set from X-Read-Tenants: the writing
// tenant first, then each listed one. An absent or blank header is the writing
// tenant alone — the behaviour before the header existed. Anything else is
// refused for the whole request, whatever its route, rather than read as a
// narrower set than meant:
//   - a header the daemon was not told to trust (trusted false): ignoring it
//     would hide a proxy that forwards it from the client;
//   - more than one header line: a proxy that appends instead of replacing
//     would otherwise merge the client's list with its own;
//   - a malformed list, or more than storage.MaxReadTenants distinct tenants;
//   - on a single-tenant backend, any tenant but SingleTenantID: SQLite has no
//     other tenant, and the header widens nothing there.
func readTenants(r *http.Request, writer string, singleTenant, trusted bool) (storage.ReadTenants, string) {
	lines := r.Header.Values(ReadTenantsHeader)
	raw := strings.TrimSpace(strings.Join(lines, ","))
	if raw == "" {
		return storage.ReadTenants{writer}, ""
	}
	if !trusted {
		return nil, ReadTenantsHeader + " is not accepted: this daemon was started without --read-tenants " +
			"(BLUNDERDB_READ_TENANTS), so its proxy is not known to set the header; strip it at the proxy, " +
			"or enable it once the proxy removes any value a client sends"
	}
	if len(lines) > 1 {
		return nil, ReadTenantsHeader + " must be sent once, got " + strconv.Itoa(len(lines)) +
			" header lines; the proxy must replace the client's value, not append to it"
	}
	// The bound of storage.MaxReadTenants counts distinct tenants, so it is
	// NewReadTenants's to apply, after duplicates are dropped; the header's
	// own size is already capped by the HTTP server.
	parts := strings.Split(raw, ",")
	listed := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if _, err := storage.ParseTenant(p); err != nil || p == "" {
			return nil, ReadTenantsHeader + " must be a comma-separated list of " + storage.TenantFormat +
				", got " + quoteHeader(raw)
		}
		if singleTenant && p != SingleTenantID {
			return nil, "this daemon runs on a single-tenant backend (SQLite): " + ReadTenantsHeader +
				" may list only " + SingleTenantID + ", got " + quoteHeader(raw)
		}
		listed = append(listed, p)
	}
	set, err := storage.NewReadTenants(writer, listed)
	if err != nil {
		return nil, ReadTenantsHeader + ": " + err.Error()
	}
	return set, ""
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

// ReadTenantsFromContext returns the read set stored by the Tenant middleware:
// X-Tenant-ID first, then the tenants X-Read-Tenants lists. It is nil when no
// tenant is present (public endpoints).
func ReadTenantsFromContext(ctx context.Context) storage.ReadTenants {
	v, _ := ctx.Value(readTenantsKey{}).(storage.ReadTenants)
	return v
}
