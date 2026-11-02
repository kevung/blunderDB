package server

import (
	"context"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/changeset"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The change journal over HTTP: thin calls into package changeset, shared with
// the desktop; the daemon adds only the tenant scope.

func (s *Server) changesetRoutes() []route {
	st := func() storage.Storage { return s.opts.Storage }
	return []route{
		{http.MethodPost, "/v1/changeset.export", rpc(func(ctx context.Context, scope string, req changesetExportReq) (changeset.Set, error) {
			return changeset.Export(ctx, st(), scope, req.Cursor, req.Limit)
		})},
		{http.MethodPost, "/v1/changeset.import", rpc(func(ctx context.Context, scope string, req changesetImportReq) (changeset.Report, error) {
			return changeset.Import(ctx, st(), scope, changeset.Set{Records: req.Records})
		})},
	}
}

// changesetExportReq asks for the page after a cursor: the zero cursor starts
// over, a cursor the last answer gave goes on, and 410 says this journal cannot
// go on from it (start over; an import absorbs the repetition).
type changesetExportReq struct {
	Cursor storage.Cursor `json:"cursor"`
	Limit  int            `json:"limit"`
}

func (r changesetExportReq) pageLimit() int { return r.Limit }

type changesetImportReq struct {
	Records []changeset.Record `json:"records"`
}
