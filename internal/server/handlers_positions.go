package server

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/race"
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

// pageLimit implements pagedReq (handlers_rpc.go): rpc/rpcStream enforce
// maxPageSize on it before positions.list/listIds ever run.
func (r listReq) pageLimit() int { return r.Limit }

// idsReq carries the ids of the positions to load, in the order wanted back.
type idsReq struct {
	IDs []int64 `json:"ids"`
}

func (s *Server) positionRoutes() []route {
	ps := func() storage.PositionStore { return s.opts.Storage.Positions() }
	return []route{
		{http.MethodPost, "/v1/positions.save", rpc(func(ctx context.Context, scope string, req positionReq) (idResp, error) {
			if req.Position == nil {
				return idResp{}, errMissing("position")
			}
			id, err := ps().Save(ctx, scope, req.Position)
			return idResp{ID: id}, err
		})},
		{http.MethodPost, "/v1/positions.update", rpcVoid(func(ctx context.Context, scope string, req positionReq) error {
			if req.Position == nil {
				return errMissing("position")
			}
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
				return nil, fmt.Errorf("%w: %w", storage.ErrInvalid, err)
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
				return xgpResp{}, fmt.Errorf("%w: %w", storage.ErrInvalid, err)
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
				return parser.Result{}, fmt.Errorf("%w: %w", storage.ErrInvalid, err)
			}
			return res, nil
		})},
		// Enumerate every legal complete play for a position's dice (pure; no
		// storage). Powers interactive "play your move" UIs (énigme answer mode,
		// and the Desktop board). Empty slice = no legal move (a dance).
		{http.MethodPost, "/v1/positions.legalMoves", rpc(func(ctx context.Context, scope string, req legalMovesReq) ([]domain.LegalPlay, error) {
			if req.Position == nil {
				return nil, errMissing("position")
			}
			plays := domain.LegalMoves(req.Position)
			if plays == nil {
				plays = []domain.LegalPlay{}
			}
			return plays, nil
		})},
		// EPC + race zone (pure; no storage). Single implementation shared
		// with the GUI and the CLI (engine/race). Response { bottom, top,
		// race? }: race carries the on-roll win probability (exact, or
		// estimated with its error bounds) and — exact regime only — the
		// money cube verdict; verdicts are never estimated (ADR-0009).
		// Sources: embedded TS-06-06 plus whatever .bd path the operator
		// configures; the daemon never downloads.
		{http.MethodPost, "/v1/positions.epc", rpc(func(ctx context.Context, scope string, req positionReq) (race.Result, error) {
			if req.Position == nil {
				return race.Result{}, errMissing("position")
			}
			return race.Evaluate(req.Position), nil
		})},
		{http.MethodPost, "/v1/positions.delete", rpcVoid(func(ctx context.Context, scope string, req idReq) error {
			return ps().Delete(ctx, scope, req.ID)
		})},
		{http.MethodPost, "/v1/positions.list", rpcStream(func(ctx context.Context, scope string, req listReq) iterPositions {
			return ps().List(ctx, scope, storage.ListOpts{Limit: req.Limit, Offset: req.Offset})
		})},
		// The id list of positions.list, without the positions: a client that
		// browses a library keeps it and fetches windows with positions.loadByIds.
		{http.MethodPost, "/v1/positions.listIds", rpc(func(ctx context.Context, scope string, req listReq) ([]int64, error) {
			return ps().ListIDs(ctx, scope, storage.ListOpts{Limit: req.Limit, Offset: req.Offset})
		})},
		// Positions in the order the ids were given; unknown ids are skipped.
		{http.MethodPost, "/v1/positions.loadByIds", rpc(func(ctx context.Context, scope string, req idsReq) ([]domain.Position, error) {
			return ps().LoadByIDs(ctx, scope, req.IDs)
		})},
	}
}
