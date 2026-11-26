package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
	"github.com/kevung/blunderdb/pkg/blunderdb/mets"
)

// /v1/gammonnet.rolls — what each of the 21 rolls does for the player about
// to roll. Like cubeMatrix and evaluate, a property of one position: nothing
// is read from the tenant but its match equity table, nothing is written, and
// the depth stops at 2 ply, the bound evaluate keeps for a request answered
// in line. Each roll is a unit of the shared analysis workers.

type rollsReq struct {
	Position *domain.Position `json:"position,omitempty"`
	XGID     string           `json:"xgid,omitempty"`
	// Ply is 0 to 2 (default 2).
	Ply    *int `json:"ply,omitempty"`
	PruneK int  `json:"pruneK"`
	// Dice and Move, when Dice is set, play the position first: the answer is
	// then the opponent's replies to that play. Move is any legal play's
	// notation, empty for a roll that dances.
	Dice *[2]int `json:"dice,omitempty"`
	Move string  `json:"move,omitempty"`
}

func (s *Server) handleGammonNetRolls(w http.ResponseWriter, r *http.Request) {
	var req rollsReq
	if err := decodeJSON(r, &req); err != nil {
		writeDecodeError(w, "invalid JSON body", err)
		return
	}
	pos, err := cubeMatrixPositionOf(cubeMatrixReq{Position: req.Position, XGID: req.XGID})
	if err != nil {
		writeErrorCode(w, CodeInvalid, err.Error())
		return
	}
	ply := 2
	if req.Ply != nil {
		ply = *req.Ply
	}
	if err := gammonnet.ValidateMapPly(ply); err != nil {
		writeErrorCode(w, CodeInvalid, err.Error())
		return
	}
	if req.PruneK <= 0 {
		req.PruneK = 12
	}
	if req.Dice != nil {
		if pos, err = gammonnet.PositionAfterPlay(pos, req.Dice[0], req.Dice[1], req.Move); err != nil {
			writeErrorCode(w, CodeInvalid, err.Error())
			return
		}
	} else if req.Move != "" {
		writeErrorCode(w, CodeInvalid, "move needs dice")
		return
	}

	scope := scopeOf(r)
	if s.refuseAnalysis(w, scope) {
		return
	}
	_, met, err := mets.Current(r.Context(), s.opts.Storage, scope)
	if err != nil {
		writeStorageError(w, err)
		return
	}
	m, err := s.maps.RollMap(r.Context(), scope, pos, met, ply, req.PruneK, s.analysisExec(r, scope, ply, req.PruneK), nil)
	writeMapResult(w, m, err)
}

// analysisExec runs a map's cells as units of the shared analysis workers,
// charged to the tenant. A map is one answer: it is finished and charged even
// if the tenant's time runs out midway.
func (s *Server) analysisExec(r *http.Request, scope string, ply, pruneK int) gammonnet.CellExec {
	charge := s.quota.spender(scope)
	return func(n int, cell func(int, *gammonnet.Searcher)) error {
		return s.analysis.each(r.Context(), scope, n, func(d time.Duration) bool {
			charge(d)
			return true
		}, func(i int, get searcherFor) {
			cell(i, get.searcher(ply, pruneK))
		})
	}
}

// writeMapResult answers a map request: nothing when the client left, 503
// when the daemon is stopping, the engine's refusal as invalid.
func writeMapResult(w http.ResponseWriter, result any, err error) {
	switch {
	case errors.Is(err, context.Canceled):
		return // nobody is listening
	case errors.Is(err, errPoolClosed):
		writeErrorCode(w, CodeUnavailable, err.Error())
		return
	case err != nil:
		writeErrorCode(w, CodeInvalid, err.Error())
		return
	}
	writeJSONResp(w, result)
}
