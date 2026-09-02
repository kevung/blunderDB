package storage

import (
	"context"
	"errors"
	"fmt"
	"strconv"
)

// Tenant propagation for PostgreSQL Row-Level Security (optional, off by
// default). When RLS is enabled, the PostgreSQL backend reads the tenant from
// the operation's context in pgxpool's BeforeAcquire and sets the
// `app.tenant_id` GUC on the connection so the RLS policies filter rows. The
// helper lives here (not in the postgres backend) so the server's tenant
// middleware can populate the context without importing the backend.

type tenantCtxKey struct{}

// ErrInvalidTenant is returned (wrapped) by ParseTenant when a scope is not a
// positive decimal integer. Callers that turn it into a user-facing rejection
// (the HTTP tenant middleware, `migrate --tenant-id`, `call --scope`) test for
// it with errors.Is.
var ErrInvalidTenant = errors.New("invalid tenant")

// TenantFormat describes the only accepted spelling of a tenant, for error
// messages: the digits of a positive integer, no sign, no leading zero, no
// surrounding whitespace, at most int64.
const TenantFormat = "a positive decimal integer (1, 2, 42, …)"

// ParseTenant converts a scope string to the numeric tenant_id used on the
// PostgreSQL domain tables.
//
// The empty scope is the desktop's single implicit tenant and maps to 0. Any
// other scope must be the canonical decimal spelling of a positive integer —
// `strconv.FormatInt(n, 10)` for some n ≥ 1. Everything else ("alice",
// "default", "0", "-1", "007", "1.0", " 7") is an error wrapping
// ErrInvalidTenant. It used to be silently mapped to tenant 0, so every
// named tenant shared one set of rows — see ADR-0005, amendment 2026-09-03.
// A tenant is an integer; mapping a name to that integer is the proxy's job.
func ParseTenant(scope string) (int64, error) {
	if scope == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(scope, 10, 64)
	if err != nil || n < 1 || strconv.FormatInt(n, 10) != scope {
		return 0, fmt.Errorf("%w: scope %q is not %s", ErrInvalidTenant, scope, TenantFormat)
	}
	return n, nil
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
