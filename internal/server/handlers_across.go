package server

import (
	"context"
	"fmt"
	"iter"
	"net/http"

	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The across.* family reads several tenants in one request (ADR-0063): the
// writing tenant (X-Tenant-ID) and every tenant X-Read-Tenants lists, in that
// order, each answer tagged with the tenant it came from. The family only
// reads: no route here takes a write, and no other route looks at
// X-Read-Tenants, so a write stays in X-Tenant-ID by construction. Who may
// read whom is the proxy's decision, made before the request arrives; the
// daemon authorises nothing (ADR-0005).
//
// A row id is unique within its tenant only, so a row read across tenants is
// named by (tenant, id): across.matchesGet, across.matchMovePositions and
// across.analysesLoadByIds take both. Bounds (limit, offset) apply to each
// tenant's stream, and the answer is the tenants' pages end to end. A stream
// has no unbounded form here: a limit of 0 means maxPageSize, since one
// request multiplies its stream by up to storage.MaxReadTenants.
//
// A position read across tenants carries its Zobrist hash: a row id names a
// position in its own tenant only, the hash names the same board in every
// tenant, so a caller joins what it writes in its own tenant (a coach's
// comment) to a position read in another by the hash.

// acrossPosition is one position of across.searchFind, with its tenant and
// its Zobrist hash.
type acrossPosition struct {
	Tenant   string           `json:"tenant"`
	Zobrist  uint64           `json:"zobrist"`
	Position *domain.Position `json:"position"`
}

// acrossSearchReq is search.find's request, bounded: Limit 0 means
// maxPageSize per tenant, and more is refused.
type acrossSearchReq struct {
	Filters domain.SearchFilters `json:"filters"`
	Limit   int                  `json:"limit"`
	Offset  int                  `json:"offset"`
}

// pageLimit implements pagedReq (handlers_rpc.go).
func (r acrossSearchReq) pageLimit() int { return r.Limit }

// acrossMatchIDReq names one match among the read tenants by its tenant and
// its id there.
type acrossMatchIDReq struct {
	Tenant  string `json:"tenant"`
	MatchID int64  `json:"matchId"`
}

// acrossMovePosition is one position of a match read in another tenant: its
// tenant, its Zobrist hash and the move played there.
type acrossMovePosition struct {
	Tenant       string                    `json:"tenant"`
	Zobrist      uint64                    `json:"zobrist"`
	MovePosition *domain.MatchMovePosition `json:"movePosition"`
}

// acrossIDsReq names positions of one tenant among the read tenants.
type acrossIDsReq struct {
	Tenant string  `json:"tenant"`
	IDs    []int64 `json:"ids"`
}

// acrossAnalyses holds the analyses of the requested positions of one tenant,
// in id order; a position without one is skipped.
type acrossAnalyses struct {
	Tenant   string                    `json:"tenant"`
	Analyses []domain.PositionAnalysis `json:"analyses"`
}

// acrossMatch is one match of across.matchesList or across.matchesGet, with
// its tenant.
type acrossMatch struct {
	Tenant string        `json:"tenant"`
	Match  *domain.Match `json:"match"`
}

// acrossMatchReq names one match among the read tenants: its tenant, which
// must be one of them, and its id within that tenant.
type acrossMatchReq struct {
	Tenant string `json:"tenant"`
	ID     int64  `json:"id"`
}

// acrossStats is one tenant's stats.compute answer.
type acrossStats struct {
	Tenant string               `json:"tenant"`
	Stats  *storage.StatsResult `json:"stats"`
}

// acrossStatsResp holds one stats.compute answer per read tenant, in the read
// set's order.
type acrossStatsResp struct {
	Results []acrossStats `json:"results"`
}

// acrossPlayers is one tenant's stats.playerTable answer.
type acrossPlayers struct {
	Tenant  string              `json:"tenant"`
	Players []storage.PlayerRow `json:"players"`
}

// acrossPlayersResp holds one stats.playerTable answer per read tenant, in the
// read set's order.
type acrossPlayersResp struct {
	Results []acrossPlayers `json:"results"`
}

type (
	iterAcrossPositions     = iter.Seq2[acrossPosition, error]
	iterAcrossMatches       = iter.Seq2[acrossMatch, error]
	iterAcrossMovePositions = iter.Seq2[acrossMovePosition, error]
)

// boundedLimit is a stream's per-tenant limit: the client's, or maxPageSize
// when it asked for none (rpcStream already refused more than maxPageSize).
func boundedLimit(limit int) int {
	if limit <= 0 {
		return maxPageSize
	}
	return limit
}

// zobristOf is the hash a position is stored under (SavePosition), which
// names the same board in every tenant.
func zobristOf(p *domain.Position) uint64 {
	if p == nil {
		return 0
	}
	norm := p.NormalizeForStorage()
	return engine.ZobristHash(&norm)
}

// readSetOf returns the request's read set, set by the Tenant middleware. Its
// fallback is the scope alone, so a handler reached without the middleware
// (an in-process test) still reads nothing beyond its own tenant.
func readSetOf(ctx context.Context, scope string) storage.ReadTenants {
	if set := middleware.ReadTenantsFromContext(ctx); len(set) > 0 {
		return set
	}
	return storage.ReadTenants{scope}
}

// mapTagged turns a tagged stream into its wire items.
func mapTagged[T, W any](seq iter.Seq2[storage.Tagged[T], error], wire func(storage.Tagged[T]) W) iter.Seq2[W, error] {
	return func(yield func(W, error) bool) {
		for tg, err := range seq {
			var w W
			if err == nil {
				w = wire(tg)
			}
			if !yield(w, err) {
				return
			}
		}
	}
}

func (s *Server) acrossRoutes() []route {
	st := func() storage.Storage { return s.opts.Storage }
	return []route{
		{http.MethodPost, "/v1/across.searchFind", rpcStream(func(ctx context.Context, scope string, req acrossSearchReq) iterAcrossPositions {
			seq := storage.StreamAcross(ctx, readSetOf(ctx, scope), func(ctx context.Context, scope string) iter.Seq2[*domain.Position, error] {
				return st().Search().Find(ctx, scope, req.Filters, storage.ListOpts{Limit: boundedLimit(req.Limit), Offset: req.Offset})
			})
			return mapTagged(seq, func(tg storage.Tagged[*domain.Position]) acrossPosition {
				return acrossPosition{Tenant: tg.Tenant, Zobrist: zobristOf(tg.Item), Position: tg.Item}
			})
		})},
		{http.MethodPost, "/v1/across.matchesList", rpcStream(func(ctx context.Context, scope string, req matchListReq) iterAcrossMatches {
			seq := storage.StreamAcross(ctx, readSetOf(ctx, scope), func(ctx context.Context, scope string) iter.Seq2[*domain.Match, error] {
				return st().Matches().List(ctx, scope, storage.MatchListOpts{
					PlayerName:         req.PlayerName,
					PlayerNameContains: req.PlayerNameContains,
					TournamentIDs:      req.TournamentIDs,
					DateFrom:           req.DateFrom,
					DateTo:             req.DateTo,
					MatchLength:        req.MatchLength,
					Sort:               req.Sort,
					Limit:              boundedLimit(req.Limit),
					Offset:             req.Offset,
				})
			})
			return mapTagged(seq, func(tg storage.Tagged[*domain.Match]) acrossMatch {
				return acrossMatch{Tenant: tg.Tenant, Match: tg.Item}
			})
		})},
		{http.MethodPost, "/v1/across.matchesGet", rpc(func(ctx context.Context, scope string, req acrossMatchReq) (acrossMatch, error) {
			tg, err := storage.ReadOne(ctx, readSetOf(ctx, scope), req.Tenant, func(ctx context.Context, scope string) (*domain.Match, error) {
				return st().Matches().Get(ctx, scope, req.ID)
			})
			return acrossMatch{Tenant: tg.Tenant, Match: tg.Item}, err
		})},
		// The positions of one match of a read tenant, move by move, each with
		// its hash: what a coach reviews a student's match from.
		{http.MethodPost, "/v1/across.matchMovePositions", rpcStream(func(ctx context.Context, scope string, req acrossMatchIDReq) iterAcrossMovePositions {
			seq := storage.StreamOne(ctx, readSetOf(ctx, scope), req.Tenant, func(ctx context.Context, scope string) iter.Seq2[*domain.MatchMovePosition, error] {
				return st().Matches().MovePositions(ctx, scope, req.MatchID)
			})
			return mapTagged(seq, func(tg storage.Tagged[*domain.MatchMovePosition]) acrossMovePosition {
				var z uint64
				if tg.Item != nil {
					z = zobristOf(&tg.Item.Position)
				}
				return acrossMovePosition{Tenant: tg.Tenant, Zobrist: z, MovePosition: tg.Item}
			})
		})},
		{http.MethodPost, "/v1/across.analysesLoadByIds", rpc(func(ctx context.Context, scope string, req acrossIDsReq) (acrossAnalyses, error) {
			if len(req.IDs) > maxPageSize {
				return acrossAnalyses{}, fmt.Errorf("%w: %d ids, at most %d", storage.ErrInvalid, len(req.IDs), maxPageSize)
			}
			tg, err := storage.ReadOne(ctx, readSetOf(ctx, scope), req.Tenant, func(ctx context.Context, scope string) ([]domain.PositionAnalysis, error) {
				byID, err := st().Analyses().LoadMany(ctx, scope, req.IDs)
				if err != nil {
					return nil, err
				}
				out := make([]domain.PositionAnalysis, 0, len(req.IDs))
				for _, id := range req.IDs {
					if a := byID[id]; a != nil {
						cp := *a
						cp.PositionID = int(id)
						out = append(out, cp)
					}
				}
				return out, nil
			})
			return acrossAnalyses{Tenant: tg.Tenant, Analyses: tg.Item}, err
		})},
		{http.MethodPost, "/v1/across.statsCompute", rpc(func(ctx context.Context, scope string, req statsComputeReq) (acrossStatsResp, error) {
			got, err := storage.ReadAcross(ctx, readSetOf(ctx, scope), func(ctx context.Context, scope string) (*storage.StatsResult, error) {
				return st().Stats().Compute(ctx, scope, req.Filter)
			})
			resp := acrossStatsResp{Results: make([]acrossStats, 0, len(got))}
			for _, tg := range got {
				resp.Results = append(resp.Results, acrossStats{Tenant: tg.Tenant, Stats: tg.Item})
			}
			return resp, err
		})},
		{http.MethodPost, "/v1/across.playerTable", rpc(func(ctx context.Context, scope string, req statsComputeReq) (acrossPlayersResp, error) {
			got, err := storage.ReadAcross(ctx, readSetOf(ctx, scope), func(ctx context.Context, scope string) ([]storage.PlayerRow, error) {
				return st().Stats().PlayerTable(ctx, scope, req.Filter)
			})
			resp := acrossPlayersResp{Results: make([]acrossPlayers, 0, len(got))}
			for _, tg := range got {
				resp.Results = append(resp.Results, acrossPlayers{Tenant: tg.Tenant, Players: tg.Item})
			}
			return resp, err
		})},
	}
}
