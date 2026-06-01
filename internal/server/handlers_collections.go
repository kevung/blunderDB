package server

import (
	"context"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type collectionCreateReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type collectionUpdateReq struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type collectionReorderReq struct {
	CollectionIDs []int64 `json:"collectionIds"`
}

type collPositionReq struct {
	CollectionID int64 `json:"collectionId"`
	PositionID   int64 `json:"positionId"`
}

type collPositionsReq struct {
	CollectionID int64   `json:"collectionId"`
	PositionIDs  []int64 `json:"positionIds"`
}

type movePositionReq struct {
	FromCollectionID int64 `json:"fromCollectionId"`
	ToCollectionID   int64 `json:"toCollectionId"`
	PositionID       int64 `json:"positionId"`
}

type copyPositionReq struct {
	ToCollectionID int64 `json:"toCollectionId"`
	PositionID     int64 `json:"positionId"`
}

type collectionIDReq struct {
	CollectionID int64 `json:"collectionId"`
}

func (s *Server) collectionRoutes() []route {
	cs := func() storage.CollectionStore { return s.opts.Storage.Collections() }
	return []route{
		{http.MethodPost, "/v1/collections.create", rpc(func(ctx context.Context, scope string, req collectionCreateReq) (idResp, error) {
			id, err := cs().Create(ctx, scope, req.Name, req.Description)
			return idResp{ID: id}, err
		})},
		{http.MethodPost, "/v1/collections.get", rpc(func(ctx context.Context, scope string, req idReq) (*storage.Collection, error) {
			return cs().Get(ctx, scope, req.ID)
		})},
		{http.MethodPost, "/v1/collections.list", rpcStream(func(ctx context.Context, scope string, _ struct{}) iterColls {
			return cs().List(ctx, scope)
		})},
		{http.MethodPost, "/v1/collections.update", rpcVoid(func(ctx context.Context, scope string, req collectionUpdateReq) error {
			return cs().Update(ctx, scope, req.ID, req.Name, req.Description)
		})},
		{http.MethodPost, "/v1/collections.delete", rpcVoid(func(ctx context.Context, scope string, req idReq) error {
			return cs().Delete(ctx, scope, req.ID)
		})},
		{http.MethodPost, "/v1/collections.reorder", rpcVoid(func(ctx context.Context, scope string, req collectionReorderReq) error {
			return cs().Reorder(ctx, scope, req.CollectionIDs)
		})},
		{http.MethodPost, "/v1/collections.addPosition", rpcVoid(func(ctx context.Context, scope string, req collPositionReq) error {
			return cs().AddPosition(ctx, scope, req.CollectionID, req.PositionID)
		})},
		{http.MethodPost, "/v1/collections.addPositions", rpcVoid(func(ctx context.Context, scope string, req collPositionsReq) error {
			return cs().AddPositions(ctx, scope, req.CollectionID, req.PositionIDs)
		})},
		{http.MethodPost, "/v1/collections.removePosition", rpcVoid(func(ctx context.Context, scope string, req collPositionReq) error {
			return cs().RemovePosition(ctx, scope, req.CollectionID, req.PositionID)
		})},
		{http.MethodPost, "/v1/collections.removePositions", rpcVoid(func(ctx context.Context, scope string, req collPositionsReq) error {
			return cs().RemovePositions(ctx, scope, req.CollectionID, req.PositionIDs)
		})},
		{http.MethodPost, "/v1/collections.reorderPositions", rpcVoid(func(ctx context.Context, scope string, req collPositionsReq) error {
			return cs().ReorderPositions(ctx, scope, req.CollectionID, req.PositionIDs)
		})},
		{http.MethodPost, "/v1/collections.movePosition", rpcVoid(func(ctx context.Context, scope string, req movePositionReq) error {
			return cs().MovePosition(ctx, scope, req.FromCollectionID, req.ToCollectionID, req.PositionID)
		})},
		{http.MethodPost, "/v1/collections.copyPosition", rpcVoid(func(ctx context.Context, scope string, req copyPositionReq) error {
			return cs().CopyPosition(ctx, scope, req.ToCollectionID, req.PositionID)
		})},
		{http.MethodPost, "/v1/collections.positions", rpcStream(func(ctx context.Context, scope string, req collectionIDReq) iterPositions {
			return cs().Positions(ctx, scope, req.CollectionID)
		})},
		{http.MethodPost, "/v1/collections.collectionsOf", rpcStream(func(ctx context.Context, scope string, req positionIDReq) iterColls {
			return cs().CollectionsOf(ctx, scope, req.PositionID)
		})},
		{http.MethodPost, "/v1/collections.positionIndexMap", rpc(func(ctx context.Context, scope string, _ struct{}) (map[int64]int, error) {
			return cs().PositionIndexMap(ctx, scope)
		})},
	}
}
