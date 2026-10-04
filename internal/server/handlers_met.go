package server

import (
	"context"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/mets"
)

// metRoutes expose the tenant's match equity tables (ADR-0068) over package
// mets, which GUI and CLI share.
func (s *Server) metRoutes() []route {
	type importReq struct {
		// Source is the gnubg .xml, as text.
		Source string `json:"source"`
		// Name is used when the file names no table.
		Name string `json:"name"`
	}
	type positionReq struct {
		PositionID int64 `json:"position_id"`
	}
	return []route{
		{http.MethodPost, "/v1/met.list", rpc(func(ctx context.Context, scope string, _ struct{}) ([]*domain.MatchEquityTable, error) {
			return mets.Overview(ctx, s.opts.Storage, scope)
		})},
		{http.MethodPost, "/v1/met.import", rpc(func(ctx context.Context, scope string, req importReq) (*domain.MatchEquityTable, error) {
			return mets.Import(ctx, s.opts.Storage, scope, []byte(req.Source), req.Name)
		})},
		{http.MethodPost, "/v1/met.setCurrent", rpc(func(ctx context.Context, scope string, req idReq) (struct{}, error) {
			return struct{}{}, s.opts.Storage.MatchEquityTables().SetCurrent(ctx, scope, req.ID)
		})},
		{http.MethodPost, "/v1/met.ofAnalysis", rpc(func(ctx context.Context, scope string, req positionReq) (mets.Status, error) {
			pos, err := s.opts.Storage.Positions().Load(ctx, scope, req.PositionID)
			if err != nil {
				return mets.Status{}, err
			}
			return mets.AnalysisStatus(ctx, s.opts.Storage, scope, req.PositionID, pos.IsMoney())
		})},
	}
}
