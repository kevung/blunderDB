package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/searchquery"
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

// collectionPositionsReq names a collection and, optionally, a page of it;
// zero bounds mean the whole collection.
type collectionPositionsReq struct {
	CollectionID int64 `json:"collectionId"`
	Limit        int   `json:"limit"`
	Offset       int   `json:"offset"`
}

func (r collectionPositionsReq) pageLimit() int { return r.Limit }

// collectionFilterReq makes a collection living, or (with an empty query)
// turns it back into a hand-made list.
type collectionFilterReq struct {
	ID    int64  `json:"id"`
	Query string `json:"query"`
}

// livingFilters is Database.livingFilters for the daemon: a living collection
// is read by its saved query, a hand-made one by its membership rows.
func (s *Server) livingFilters(ctx context.Context, scope string, collectionID int64) (domain.SearchFilters, bool, error) {
	col, err := s.opts.Storage.Collections().Get(ctx, scope, collectionID)
	if err != nil {
		return domain.SearchFilters{}, false, err
	}
	return searchquery.Living(collectionID, strings.TrimSpace(col.FilterQuery))
}

func (s *Server) collectionRoutes() []route {
	cs := func() storage.CollectionStore { return s.opts.Storage.Collections() }
	return []route{
		// Idempotent: Create has no natural dedup key, so a retry would make a
		// second, identically named collection.
		{http.MethodPost, "/v1/collections.create", s.withIdempotency(rpc(func(ctx context.Context, scope string, req collectionCreateReq) (idResp, error) {
			id, err := cs().Create(ctx, scope, req.Name, req.Description)
			return idResp{ID: id}, err
		}))},
		{http.MethodPost, "/v1/collections.get", rpc(func(ctx context.Context, scope string, req idReq) (*storage.Collection, error) {
			return cs().Get(ctx, scope, req.ID)
		})},
		{http.MethodPost, "/v1/collections.list", rpcStream(func(ctx context.Context, scope string, _ struct{}) iterColls {
			return cs().List(ctx, scope)
		})},
		{http.MethodPost, "/v1/collections.update", rpcVoid(func(ctx context.Context, scope string, req collectionUpdateReq) error {
			return cs().Update(ctx, scope, req.ID, req.Name, req.Description)
		})},
		// Collection VIVANTE : la route pose la requête, le client la réévalue
		// à chaque ouverture.
		{http.MethodPost, "/v1/collections.setFilter", rpcVoid(func(ctx context.Context, scope string, req collectionFilterReq) error {
			return cs().SetFilterQuery(ctx, scope, req.ID, req.Query)
		})},
		// Figer : la requête cesse de commander, ses positions d'aujourd'hui
		// deviennent l'appartenance.
		{http.MethodPost, "/v1/collections.freeze", rpc(func(ctx context.Context, scope string, req idReq) (int, error) {
			filters, living, err := s.livingFilters(ctx, scope, req.ID)
			if err != nil {
				return 0, err
			}
			if !living {
				return 0, fmt.Errorf("collection %d is not living: nothing to freeze", req.ID)
			}
			return storage.FreezeCollection(ctx, s.opts.Storage, scope, req.ID, filters)
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
		{http.MethodPost, "/v1/collections.positions", rpcStream(func(ctx context.Context, scope string, req collectionPositionsReq) iterPositions {
			opts := storage.ListOpts{Limit: req.Limit, Offset: req.Offset}
			filters, living, err := s.livingFilters(ctx, scope, req.CollectionID)
			if err != nil {
				return func(yield func(*domain.Position, error) bool) { yield(nil, err) }
			}
			if living {
				return s.opts.Storage.Search().Find(ctx, scope, filters, opts)
			}
			return cs().Positions(ctx, scope, req.CollectionID, opts)
		})},
		{http.MethodPost, "/v1/collections.positionIds", rpc(func(ctx context.Context, scope string, req collectionPositionsReq) ([]int64, error) {
			opts := storage.ListOpts{Limit: req.Limit, Offset: req.Offset}
			filters, living, err := s.livingFilters(ctx, scope, req.CollectionID)
			if err != nil {
				return nil, err
			}
			if living {
				return s.opts.Storage.Search().FindIDs(ctx, scope, filters, opts)
			}
			return cs().PositionIDs(ctx, scope, req.CollectionID, opts)
		})},
		{http.MethodPost, "/v1/collections.countPositions", rpc(func(ctx context.Context, scope string, req collectionPositionsReq) (int, error) {
			filters, living, err := s.livingFilters(ctx, scope, req.CollectionID)
			if err != nil {
				return 0, err
			}
			if living {
				return s.opts.Storage.Search().Count(ctx, scope, filters)
			}
			return cs().CountPositions(ctx, scope, req.CollectionID)
		})},
		{http.MethodPost, "/v1/collections.indexOfPosition", rpc(func(ctx context.Context, scope string, req collPositionReq) (int, error) {
			filters, living, err := s.livingFilters(ctx, scope, req.CollectionID)
			if err != nil {
				return -1, err
			}
			var index int
			var found bool
			if living {
				index, found, err = s.opts.Storage.Search().IndexOf(ctx, scope, filters, req.PositionID)
			} else {
				index, found, err = cs().IndexOfPosition(ctx, scope, req.CollectionID, req.PositionID)
			}
			if err != nil || !found {
				return -1, err
			}
			return index, nil
		})},
		{http.MethodPost, "/v1/collections.collectionsOf", rpcStream(func(ctx context.Context, scope string, req positionIDReq) iterColls {
			return cs().CollectionsOf(ctx, scope, req.PositionID)
		})},
		// La Pile : le geste « à revoir plus tard ». Une position sans ligne
		// stockée (brouillon) est d'abord écrite comme position apportée seule.
		{http.MethodPost, "/v1/pile.toggle", rpc(func(ctx context.Context, scope string, req positionReq) (storage.PileToggle, error) {
			return storage.TogglePile(ctx, s.opts.Storage, scope, req.Position)
		})},
		{http.MethodPost, "/v1/pile.state", rpc(func(ctx context.Context, scope string, req positionIDReq) (bool, error) {
			return storage.PositionOnPile(ctx, s.opts.Storage, scope, req.PositionID)
		})},
		{http.MethodPost, "/v1/collections.positionIndexMap", rpc(func(ctx context.Context, scope string, _ struct{}) (map[int64]int, error) {
			return cs().PositionIndexMap(ctx, scope)
		})},
	}
}
