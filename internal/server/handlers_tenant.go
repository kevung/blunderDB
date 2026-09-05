package server

import (
	"context"
	"fmt"
	"net/http"
)

// tenantPurger is satisfied only by the PostgreSQL backend (see
// postgres.Storage.PurgeTenant) — duck-typed the same way serve.go checks for
// ApplyRLS, so the SQLite backend needs no stub method.
type tenantPurger interface {
	PurgeTenant(ctx context.Context, scope string) error
}

// tenantRoutes returns the tenant-lifecycle route family: currently just
// tenant.purge, an ops-facing capability for decommissioning a tenant.
func (s *Server) tenantRoutes() []route {
	return []route{
		{http.MethodPost, "/ops/tenant.purge", func(w http.ResponseWriter, r *http.Request) {
			purger, ok := s.opts.Storage.(tenantPurger)
			if !ok {
				writeErrorCode(w, CodeInvalid, "tenant purge not supported on this backend (postgres only)")
				return
			}
			if err := purger.PurgeTenant(r.Context(), scopeOf(r)); err != nil {
				// Never err.Error() to the client: a PostgreSQL error can
				// quote the DSN, a table or a statement (#160).
				// writeStorageError masks it and stashes the cause for the
				// server-side log line.
				writeStorageError(w, fmt.Errorf("purge tenant: %w", err))
				return
			}
			writeJSONResp(w, okResp{OK: true})
		}},
	}
}
