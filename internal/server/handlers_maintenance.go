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

// maintenanceRoutes returns the operator's maintenance routes: currently just
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

// reencodeRoutes returns maintenance.reencode, the daemon's side of the CLI's
// `reencode` (ADR-0070). The pass is expensive but stops at the caller's
// tenant, so it stays outside /ops/ like gammonnet.sweepStale.
func (s *Server) reencodeRoutes() []route {
	return []route{
		{http.MethodPost, "/v1/maintenance.reencode", func(w http.ResponseWriter, r *http.Request) {
			n, err := storage.ReencodeAllAnalyses(r.Context(), s.opts.Storage.Analyses(), scopeOf(r), nil)
			if err != nil {
				writeStorageError(w, fmt.Errorf("reencode: %w", err))
				return
			}
			writeJSONResp(w, map[string]int{"rewritten": n})
		}},
	}
}
