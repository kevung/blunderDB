package storage

import (
	"context"
	"strconv"
)

// Tenant propagation for PostgreSQL Row-Level Security (optional, off by
// default). When RLS is enabled, the PostgreSQL backend reads the tenant from
// the operation's context in pgxpool's BeforeAcquire and sets the
// `app.tenant_id` GUC on the connection so the RLS policies filter rows. The
// helper lives here (not in the postgres backend) so the server's tenant
// middleware can populate the context without importing the backend.

type tenantCtxKey struct{}

// ParseTenant converts a scope string to the numeric tenant_id used on the
// PostgreSQL domain tables. An empty or non-numeric scope maps to tenant 0,
// matching the backend's own scope→tenant_id conversion.
func ParseTenant(scope string) int64 {
	if scope == "" {
		return 0
	}
	n, _ := strconv.ParseInt(scope, 10, 64)
	return n
}

// WithTenant returns a context carrying the numeric tenant id. The PostgreSQL
// backend reads it to set the RLS GUC; it is ignored by the SQLite backend and
// when RLS is disabled.
func WithTenant(ctx context.Context, tenant int64) context.Context {
	return context.WithValue(ctx, tenantCtxKey{}, tenant)
}

// TenantFromContext returns the numeric tenant id set by WithTenant.
func TenantFromContext(ctx context.Context) (int64, bool) {
	v, ok := ctx.Value(tenantCtxKey{}).(int64)
	return v, ok
}
