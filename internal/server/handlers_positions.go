package server

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/parser"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type positionReq struct {
	Position *domain.Position `json:"position"`
}

type existsReq struct {
	Zobrist uint64 `json:"zobrist"`
}

type xgidReq struct {
	XGID string `json:"xgid"`
}

// parseTextReq carries pasted clipboard / file text to parse into a position.
type parseTextReq struct {
	Text string `json:"text"`
}

// xgpReq carries a base64-encoded .xgp single-position file.
type xgpReq struct {
	Data string `json:"data"`
}

// legalMovesReq carries a position (board + dice + player on roll) to enumerate.
type legalMovesReq struct {
	Position *domain.Position `json:"position"`
}

// xgpResp is the parsed position plus its analysis (if the file carried one).
type xgpResp struct {
	Position *domain.Position         `json:"position"`
	Analysis *domain.PositionAnalysis `json:"analysis,omitempty"`
}

type existsResp struct {
	ID    int64 `json:"id"`
	Found bool  `json:"found"`
}

type listReq struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// epcSide holds the Effective Pip Count data for one player. EPC is only
// defined once every checker is in the player's home board (bearing-off zone);
// otherwise EPC is nil but the pip count is still meaningful.
type epcSide struct {
	AllInHome    bool              `json:"all_in_home"`
	CheckerCount int               `json:"checker_count"`
	EPC          *engine.EPCResult `json:"epc,omitempty"`
}

type epcResp struct {
	// Bottom = Black (player index 0), Top = White (player index 1).
	Bottom epcSide `json:"bottom"`
	Top    epcSide `json:"top"`
}

// computeEPCSide extracts a player's home-board checkers and computes EPC when
// the whole army is home. board indices: WhiteBar=0, points 1..24, BlackBar=25.
func computeEPCSide(b *domain.Board, color int) epcSide {
	var home [6]int // home[i] = checkers on this player's (i+1)-point
	total, allHome := 0, true
	for i := 1; i <= 24; i++ {
		pt := b.Points[i]
		if pt.Color != color || pt.Checkers <= 0 {
			continue
		}
		total += pt.Checkers
		// Black bears off toward point 1 → home points are 1..6.
		// White bears off toward point 24 → home points are 24..19.
		var homeIdx int
		if color == domain.Black {
			homeIdx = i - 1 // point 1 → index 0 … point 6 → index 5
		} else {
			homeIdx = 24 - i // point 24 → index 0 … point 19 → index 5
		}
		if homeIdx < 0 || homeIdx > 5 {
			allHome = false
			continue
		}
		home[homeIdx] += pt.Checkers
	}
	bar := domain.WhiteBar
	if color == domain.Black {
		bar = domain.BlackBar
	}
	if b.Points[bar].Color == color && b.Points[bar].Checkers > 0 {
		allHome = false
		total += b.Points[bar].Checkers
	}
	side := epcSide{AllInHome: allHome, CheckerCount: total}
	if allHome && total > 0 {
		if r, err := engine.ComputeEPC(home); err == nil {
			side.EPC = r
		}
	}
	return side
}

func (s *Server) positionRoutes() []route {
	ps := func() storage.PositionStore { return s.opts.Storage.Positions() }
	return []route{
		{http.MethodPost, "/v1/positions.save", rpc(func(ctx context.Context, scope string, req positionReq) (idResp, error) {
			id, err := ps().Save(ctx, scope, req.Position)
			return idResp{ID: id}, err
		})},
		{http.MethodPost, "/v1/positions.update", rpcVoid(func(ctx context.Context, scope string, req positionReq) error {
			return ps().Update(ctx, scope, req.Position)
		})},
		{http.MethodPost, "/v1/positions.load", rpc(func(ctx context.Context, scope string, req idReq) (*domain.Position, error) {
			return ps().Load(ctx, scope, req.ID)
		})},
		{http.MethodPost, "/v1/positions.exists", rpc(func(ctx context.Context, scope string, req existsReq) (existsResp, error) {
			id, found, err := ps().Exists(ctx, scope, req.Zobrist)
			return existsResp{ID: id, Found: found}, err
		})},
		// Decode an XGID string into a Position (pure; no storage). Generic and
		// useful to the Desktop too (paste an XGID). Invalid input → 4xx.
		{http.MethodPost, "/v1/positions.fromXGID", rpc(func(ctx context.Context, scope string, req xgidReq) (*domain.Position, error) {
			pos, err := domain.DecodeXGID(req.XGID)
			if err != nil {
				return nil, fmt.Errorf("%w: %v", storage.ErrInvalid, err)
			}
			return &pos, nil
		})},
		// Parse a single-position XG file (.xgp) into a Position + optional analysis
		// (pure; no storage). The file bytes arrive base64-encoded. Whole matches
		// (.xg) are a separate import, not this single-position path.
		{http.MethodPost, "/v1/positions.fromXGP", rpc(func(ctx context.Context, scope string, req xgpReq) (xgpResp, error) {
			raw, err := base64.StdEncoding.DecodeString(req.Data)
			if err != nil {
				return xgpResp{}, fmt.Errorf("%w: bad base64 data", storage.ErrInvalid)
			}
			f, err := os.CreateTemp("", "blunderdb-*.xgp")
			if err != nil {
				return xgpResp{}, err
			}
			defer os.Remove(f.Name())
			if _, err := f.Write(raw); err != nil {
				f.Close()
				return xgpResp{}, err
			}
			f.Close()

			graphs, err := ingest.MapXGPPosition(f.Name())
			if err != nil {
				return xgpResp{}, fmt.Errorf("%w: %v", storage.ErrInvalid, err)
			}
			if len(graphs) == 0 || graphs[0].Position == nil {
				return xgpResp{}, fmt.Errorf("%w: no position in file", storage.ErrInvalid)
			}
			resp := xgpResp{Position: graphs[0].Position}
			if len(graphs[0].Analyses) > 0 {
				resp.Analysis = graphs[0].Analyses[0]
			}
			return resp, nil
		})},
		// Parse pasted clipboard / file text (bare XGID, XG human-readable export,
		// or blunderDB internal export) into a Position + optional analysis +
		// comment (pure; no storage). Same backend parser the Desktop GUI calls,
		// so the two share one implementation. Invalid input → 4xx.
		{http.MethodPost, "/v1/positions.parseText", rpc(func(ctx context.Context, scope string, req parseTextReq) (parser.Result, error) {
			res, err := parser.ParsePosition(req.Text)
			if err != nil {
				return parser.Result{}, fmt.Errorf("%w: %v", storage.ErrInvalid, err)
			}
			return res, nil
		})},
		// Enumerate every legal complete play for a position's dice (pure; no
		// storage). Powers interactive "play your move" UIs (énigme answer mode,
		// and the Desktop board). Empty slice = no legal move (a dance).
		{http.MethodPost, "/v1/positions.legalMoves", rpc(func(ctx context.Context, scope string, req legalMovesReq) ([]domain.LegalPlay, error) {
			if req.Position == nil {
				return nil, fmt.Errorf("%w: missing position", storage.ErrInvalid)
			}
			plays := domain.LegalMoves(req.Position)
			if plays == nil {
				plays = []domain.LegalPlay{}
			}
			return plays, nil
		})},
		// Effective Pip Count for both players (pure; no storage). Generic race
		// metric, useful to the Desktop too. EPC is nil until a side is all home.
		{http.MethodPost, "/v1/positions.epc", rpc(func(ctx context.Context, scope string, req positionReq) (epcResp, error) {
			if req.Position == nil {
				return epcResp{}, fmt.Errorf("%w: missing position", storage.ErrInvalid)
			}
			return epcResp{
				Bottom: computeEPCSide(&req.Position.Board, domain.Black),
				Top:    computeEPCSide(&req.Position.Board, domain.White),
			}, nil
		})},
		{http.MethodPost, "/v1/positions.delete", rpcVoid(func(ctx context.Context, scope string, req idReq) error {
			return ps().Delete(ctx, scope, req.ID)
		})},
		{http.MethodPost, "/v1/positions.list", rpcStream(func(ctx context.Context, scope string, req listReq) iterPositions {
			return ps().List(ctx, scope, storage.ListOpts{Limit: req.Limit, Offset: req.Offset})
		})},
	}
}
