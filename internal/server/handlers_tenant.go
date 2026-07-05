package server

import (
	"context"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// tenantPurger is satisfied only by the PostgreSQL backend (see
// postgres.Storage.PurgeTenant) — duck-typed the same way serve.go checks for
// ApplyRLS, so the SQLite backend needs no stub method.
type tenantPurger interface {
	PurgeTenant(ctx context.Context, tenantID int64) error
}

// tenantRoutes returns the tenant-lifecycle route family: currently just
// tenant.purge, an ops-facing capability for decommissioning a tenant.
func (s *Server) tenantRoutes() []route {
	return []route{
		{http.MethodPost, "/v1/tenant.purge", func(w http.ResponseWriter, r *http.Request) {
			purger, ok := s.opts.Storage.(tenantPurger)
			if !ok {
				writeErrorCode(w, CodeInvalid, "tenant purge not supported on this backend (postgres only)")
				return
			}
			scope := scopeOf(r)
			if err := purger.PurgeTenant(r.Context(), storage.ParseTenant(scope)); err != nil {
				writeErrorCode(w, CodeInternal, "purge failed: "+err.Error())
				return
			}
			writeJSONResp(w, okResp{OK: true})
		}},
	}
}
