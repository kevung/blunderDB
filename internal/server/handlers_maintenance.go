package server

import (
	"context"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// vacuumer is satisfied only by the SQLite backend (sqlite.Storage.Vacuum):
// PostgreSQL has no file to compact, and autovacuum is the operator's
// business. Duck-typed the same way tenantPurger is, so the PostgreSQL
// backend needs no stub method.
type vacuumer interface {
	Vacuum(ctx context.Context) (storage.VacuumResult, error)
}

// maintenanceRoutes returns the maintenance route family: currently just
// maintenance.vacuum, the daemon's side of the GUI button and the CLI's
// `vacuum`. The three run one implementation on the backend.
func (s *Server) maintenanceRoutes() []route {
	return []route{
		{http.MethodPost, "/v1/maintenance.vacuum", func(w http.ResponseWriter, r *http.Request) {
			v, ok := s.opts.Storage.(vacuumer)
			if !ok {
				writeErrorCode(w, CodeInvalid, "vacuum not supported on this backend (sqlite only)")
				return
			}
			res, err := v.Vacuum(r.Context())
			if err != nil {
				writeErrorCode(w, CodeInternal, "vacuum failed: "+err.Error())
				return
			}
			writeJSONResp(w, res)
		}},
	}
}
