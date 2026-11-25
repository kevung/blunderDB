package server

import (
	"net/http"
	"testing"
)

// The matrix keeps its length bounds and default, and takes the depth bound
// of every in-line computation on a bare position (ADR-0116).
func TestGammonNetCubeMatrix_DepthBound(t *testing.T) {
	ts := newTestServer(t)
	r := post(t, ts, "/v1/gammonnet.cubeMatrix", cubeMatrixReq{XGID: rollsXGID, Ply: 3})
	r.Body.Close()
	if r.StatusCode != http.StatusBadRequest {
		t.Fatalf("3-ply matrix: status %d, want 400", r.StatusCode)
	}
	r = post(t, ts, "/v1/gammonnet.cubeMatrix", cubeMatrixReq{XGID: rollsXGID, MatchLength: 25})
	r.Body.Close()
	if r.StatusCode != http.StatusOK {
		t.Fatalf("25-point 0-ply matrix: status %d, want 200", r.StatusCode)
	}
}
