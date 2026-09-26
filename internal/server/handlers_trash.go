package server

import (
	"context"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/trash"
)

// The trash over HTTP (ADR-0036): thin calls into package trash, shared with
// the desktop; the daemon adds only the tenant scope. Deleting through the
// trash has its own routes so positions.delete does not start leaving rows
// behind for existing clients.

func (s *Server) trashRoutes() []route {
	st := func() storage.Storage { return s.opts.Storage }
	return []route{
		{http.MethodPost, "/v1/trash.list", rpc(func(ctx context.Context, scope string, req trashListReq) ([]*domain.TrashEntry, error) {
			return st().Trash().List(ctx, scope, domain.TrashKind(req.Kind),
				storage.ListOpts{Limit: req.Limit, Offset: req.Offset})
		})},
		{http.MethodPost, "/v1/trash.count", rpc(func(ctx context.Context, scope string, _ struct{}) (countResp, error) {
			n, err := st().Trash().Count(ctx, scope)
			return countResp{Count: n}, err
		})},
		{http.MethodPost, "/v1/trash.restore", rpc(func(ctx context.Context, scope string, req idReq) (idResp, error) {
			id, err := trash.Restore(ctx, st(), scope, req.ID)
			return idResp{ID: id}, err
		})},
		{http.MethodPost, "/v1/trash.discard", rpcVoid(func(ctx context.Context, scope string, req idReq) error {
			return st().Trash().Discard(ctx, scope, req.ID)
		})},
		{http.MethodPost, "/v1/trash.empty", rpc(func(ctx context.Context, scope string, req trashEmptyReq) (purgedResp, error) {
			n, err := st().Trash().Purge(ctx, scope, req.OlderThanDays)
			return purgedResp{Purged: n}, err
		})},
		// Deleting through the trash. Separate routes, not a flag on the
		// existing deletes: a client that has always called positions.delete
		// must keep getting a delete.
		{http.MethodPost, "/v1/trash.deletePosition", rpc(func(ctx context.Context, scope string, req idReq) (idResp, error) {
			id, err := trash.Position(ctx, st(), scope, req.ID)
			return idResp{ID: id}, err
		})},
		{http.MethodPost, "/v1/trash.deleteCollection", rpc(func(ctx context.Context, scope string, req idReq) (idResp, error) {
			id, err := trash.Collection(ctx, st(), scope, req.ID)
			return idResp{ID: id}, err
		})},
		{http.MethodPost, "/v1/trash.deleteComment", rpc(func(ctx context.Context, scope string, req idReq) (idResp, error) {
			id, err := trash.CommentEntry(ctx, st(), scope, req.ID)
			return idResp{ID: id}, err
		})},
	}
}

// trashListReq narrows a listing to one kind and pages it. An unknown kind
// matches nothing rather than erroring.
type trashListReq struct {
	Kind   string `json:"kind,omitempty"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

func (r trashListReq) pageLimit() int { return r.Limit }

// trashEmptyReq says how old an entry has to be to be dropped. 0 empties the
// trash; domain.TrashRetentionDays is what `vacuum` passes.
type trashEmptyReq struct {
	OlderThanDays int `json:"olderThanDays"`
}

type countResp struct {
	Count int `json:"count"`
}

type purgedResp struct {
	Purged int `json:"purged"`
}
