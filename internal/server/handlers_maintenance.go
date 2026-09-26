package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// vacuumer is satisfied only by SQLite (PostgreSQL has no file to compact);
// duck-typed like tenantPurger so PostgreSQL needs no stub.
type vacuumer interface {
	Vacuum(ctx context.Context) (storage.VacuumResult, error)
}

// maintenanceRoutes returns the maintenance route family: currently just
// maintenance.vacuum, the daemon's side of the GUI button and the CLI's
// `vacuum`. The three run one implementation on the backend.
func (s *Server) maintenanceRoutes() []route {
	return []route{
		{http.MethodPost, "/ops/maintenance.vacuum", func(w http.ResponseWriter, r *http.Request) {
			v, ok := s.opts.Storage.(vacuumer)
			if !ok {
				writeErrorCode(w, CodeInvalid, "vacuum not supported on this backend (sqlite only)")
				return
			}
			res, err := v.Vacuum(r.Context())
			if err != nil {
				// Never err.Error() to the client: it names the database file.
				// writeStorageError masks and logs it.
				writeStorageError(w, fmt.Errorf("vacuum: %w", err))
				return
			}
			writeJSONResp(w, res)
		}},
	}
}
