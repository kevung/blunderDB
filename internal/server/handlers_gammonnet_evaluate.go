package server

import (
	"fmt"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
	"github.com/kevung/blunderdb/pkg/blunderdb/mets"
)

// gammonnetEvaluateReq is one bare position to evaluate, given whole or as an
// XGID, like cubeMatrix. Nothing is read from or written to the tenant: the
// answer depends on the position alone.
type gammonnetEvaluateReq struct {
	Position *domain.Position `json:"position,omitempty"`
	XGID     string           `json:"xgid,omitempty"`
	// Ply is 0 to 2 (default 2, the canonical depth); deeper searches belong
	// to analyzeMissing, which runs one at a time.
	Ply *int `json:"ply,omitempty"`
	// Candidates bounds the checker plays returned, 1 to 20 (default 5).
	Candidates int `json:"candidates"`
}

type gammonnetEvaluateResp struct {
	Decision string                       `json:"decision"`
	Depth    string                       `json:"depth"`
	Moves    []domain.CheckerMove         `json:"moves,omitempty"`
	Cube     *domain.DoublingCubeAnalysis `json:"cube,omitempty"`
}

func (s *Server) handleGammonNetEvaluate(w http.ResponseWriter, r *http.Request) {
	var req gammonnetEvaluateReq
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
	if ply < 0 || ply > 2 {
		writeErrorCode(w, CodeInvalid, "ply must be between 0 and 2")
		return
	}
	if req.Candidates == 0 {
		req.Candidates = 5
	}
	if req.Candidates < 1 || req.Candidates > 20 {
		writeErrorCode(w, CodeInvalid, "candidates must be between 1 and 20")
		return
	}
	scope := scopeOf(r)
	if s.refuseAnalysis(w, scope) {
		return
	}
	// The tenant's table values the verdict, as it values its analyses (ADR-0068).
	_, met, err := mets.Current(r.Context(), s.opts.Storage, scope)
	if err != nil {
		writeStorageError(w, err)
		return
	}
	res, err := metered(s, scope, 1, func() (gammonnet.EvalResult, error) {
		return gammonnet.EvaluatePositionWithMET(nil, pos, met, ply, 0, req.Candidates)
	})
	if err != nil {
		writeErrorCode(w, CodeInvalid, fmt.Sprintf("not evaluable: %v", err))
		return
	}
	out := gammonnetEvaluateResp{Decision: "checker", Moves: res.Moves, Cube: res.Cube}
	if res.Cube != nil {
		out.Decision = "cube"
		out.Depth = res.Cube.AnalysisDepth
	} else if len(res.Moves) > 0 {
		out.Depth = res.Moves[0].AnalysisDepth
	}
	writeJSONResp(w, out)
}
