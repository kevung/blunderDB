package server

import (
	"errors"
	"net/http"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
)

// /v1/gammonnet.cubeMatrix — the cube verdict at every score of a match
// (issue #267, fiche I.11).
//
// The daemon's half is thin on purpose: the grid is a property of a POSITION,
// not of a library, so there is nothing to gather, nothing to store and no
// tenant to scope. The route exists so a client that has no desktop — the very
// case ADR-0005's deployment describes — can ask the same question the Eval
// panel's tab asks, and get the same numbers, because both call the same
// gammonnet.ComputeCubeMatrix.
//
// It takes a position rather than an id for the same reason: the grid answers
// "at what score would I double this", and "this" is very often a board the
// user is editing and has never saved.

// cubeMatrixReq asks for one grid. Exactly one of Position or XGID is
// expected; XGID is the form a script or a paste has to hand.
type cubeMatrixReq struct {
	Position *domain.Position `json:"position,omitempty"`
	XGID     string           `json:"xgid,omitempty"`
	// MatchLength is the match the grid spans (default 7). Ply defaults to
	// the canonical 2.
	MatchLength int `json:"matchLength"`
	Ply         int `json:"ply"`
	PruneK      int `json:"pruneK"`
}

func (s *Server) handleGammonNetCubeMatrix(w http.ResponseWriter, r *http.Request) {
	var req cubeMatrixReq
	if err := decodeJSON(r, &req); err != nil {
		writeDecodeError(w, "invalid JSON body", err)
		return
	}

	pos, err := cubeMatrixPositionOf(req)
	if err != nil {
		writeErrorCode(w, CodeInvalid, err.Error())
		return
	}
	if req.MatchLength == 0 {
		req.MatchLength = 7
	}
	if req.MatchLength < 1 || req.MatchLength > 25 {
		writeErrorCode(w, CodeInvalid, "matchLength must be between 1 and 25")
		return
	}
	if req.PruneK <= 0 {
		req.PruneK = 12
	}

	matrix, err := gammonnet.ComputeCubeMatrix(r.Context(), pos, req.MatchLength, req.Ply, req.PruneK, 0)
	if err != nil {
		writeErrorCode(w, CodeInvalid, err.Error())
		return
	}
	writeJSONResp(w, matrix)
}

// cubeMatrixPositionOf reads the one position the request carries, refusing
// both the empty request and the ambiguous one — a body giving a position AND
// an XGID has two answers and no way to say which was meant.
func cubeMatrixPositionOf(req cubeMatrixReq) (domain.Position, error) {
	switch {
	case req.Position != nil && req.XGID != "":
		return domain.Position{}, errors.New("give either position or xgid, not both")
	case req.Position != nil:
		return *req.Position, nil
	case req.XGID != "":
		return domain.DecodeXGID(req.XGID)
	default:
		return domain.Position{}, errors.New("a position or an xgid is required")
	}
}
