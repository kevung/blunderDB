package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// transferPullPath is the daemon's transfer (ADR-0102): a pull from a tenant
// of X-Read-Tenants into X-Tenant-ID. It is the one route that reads the read
// set and writes, and it writes only in the writer's tenant: ADR-0065 lets a
// tenant read others and write its own, so a transfer is always pulled by the
// tenant it lands in, never pushed into another.
const transferPullPath = "/v1/across.transferPull"

// transferPullReq names the tenant to pull from and what to take of it: the
// export options of exports.sqlite, read the same way (a field left out keeps
// the whole-library preset, an unknown key is refused). The watermark,
// password and incremental options are refused: a transfer writes nothing
// that says where the copy came from, and nothing in its source.
type transferPullReq struct {
	Tenant string          `json:"tenant"`
	Export exportSQLiteReq `json:"export"`
}

// handleTransferPull copies the selection of req.Tenant into the caller's
// tenant and answers the ingest.TransferReport. It holds the writer's import
// slot (the quota of imports.*), and Server.Run cancels it on shutdown like an
// import.
func (s *Server) handleTransferPull(w http.ResponseWriter, r *http.Request) {
	var req transferPullReq
	if err := decodeJSON(r, &req); err != nil {
		writeDecodeError(w, "invalid JSON body", err)
		return
	}
	scope := scopeOf(r)
	if req.Tenant == "" {
		writeStorageError(w, fmt.Errorf("%w: tenant is required: the tenant of %s to pull from", storage.ErrInvalid, middleware.ReadTenantsHeader))
		return
	}
	if !readSetOf(r.Context(), scope).Contains(req.Tenant) {
		writeStorageError(w, fmt.Errorf("%w: tenant %q is not one of this request's read tenants", storage.ErrInvalid, req.Tenant))
		return
	}
	if s.refuseImport(r.Context(), w, scope) {
		return
	}
	defer s.quota.endImport(scope)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	id := s.imports.start(scope, cancel)
	defer s.imports.finish(id)

	st := s.opts.Storage
	rep, err := ingest.Transfer(ctx, ingest.TransferEnd{S: st, Scope: req.Tenant}, ingest.TransferEnd{S: st, Scope: scope}, req.Export.options(), nil)
	if err != nil {
		writeStorageError(w, err)
		return
	}
	writeJSONResp(w, rep)
}
