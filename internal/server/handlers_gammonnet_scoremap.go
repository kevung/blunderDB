package server

import (
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
	"github.com/kevung/blunderdb/pkg/blunderdb/mets"
)

// /v1/gammonnet.scoreMap — the best decision of one position at every score
// of a match: the best play with dice, the cube verdict without. Like
// cubeMatrix, a property of the position: only the tenant's match equity
// table is read, nothing is written, each cell is a unit of the shared
// analysis workers.

type scoreMapReq struct {
	Position *domain.Position `json:"position,omitempty"`
	XGID     string           `json:"xgid,omitempty"`
	// MatchLength is odd, 3 to 15 (default 9).
	MatchLength int `json:"matchLength"`
	// Ply is 0 to 2 (default 2).
	Ply    *int `json:"ply,omitempty"`
	PruneK int  `json:"pruneK"`
}

func (s *Server) handleGammonNetScoreMap(w http.ResponseWriter, r *http.Request) {
	var req scoreMapReq
	if err := decodeJSON(r, &req); err != nil {
		writeDecodeError(w, "invalid JSON body", err)
		return
	}
	pos, err := cubeMatrixPositionOf(cubeMatrixReq{Position: req.Position, XGID: req.XGID})
	if err != nil {
		writeErrorCode(w, CodeInvalid, err.Error())
		return
	}
	if req.MatchLength == 0 {
		req.MatchLength = gammonnet.DefaultMapMatchLength
	}
	if err := gammonnet.ValidateMapMatchLength(req.MatchLength); err != nil {
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

	scope := scopeOf(r)
	if s.refuseAnalysis(w, scope) {
		return
	}
	_, met, err := mets.Current(r.Context(), s.opts.Storage, scope)
	if err != nil {
		writeStorageError(w, err)
		return
	}
	m, err := gammonnet.Maps().ScoreMap(r.Context(), scope, pos, met, req.MatchLength, ply, req.PruneK, s.analysisExec(r, scope, ply, req.PruneK), nil)
	writeMapResult(w, m, err)
}
