package server

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/mets"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// metRoutes expose the tenant's match equity tables (ADR-0068) over package
// mets, which GUI and CLI share.
func (s *Server) metRoutes() []route {
	type importReq struct {
		// Source is the gnubg .xml, as text. A file that is not UTF-8 travels
		// as SourceBase64, its bytes untouched.
		Source       string `json:"source"`
		SourceBase64 string `json:"source_base64"`
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
			if req.SourceBase64 != "" {
				raw, err := base64.StdEncoding.DecodeString(req.SourceBase64)
				if err != nil {
					return nil, fmt.Errorf("%w: source_base64: %w", storage.ErrInvalid, err)
				}
				return mets.Import(ctx, s.opts.Storage, scope, raw, req.Name)
			}
			return mets.ImportText(ctx, s.opts.Storage, scope, req.Source, req.Name)
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
