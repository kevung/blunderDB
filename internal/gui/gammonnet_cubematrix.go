package gui

import (
	"context"
	"fmt"
	"sync"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
)

// The cube matrix's desktop half: a 0-ply grid at the gesture, then the
// display depth once settled. A grid is matchLength² searches, so the deep
// tier is cancellable. The grid is gammonnet.ComputeCubeMatrix, shared with
// the CLI and the daemon.

// cubeMatrixRun is the single in-flight deep sweep.
type cubeMatrixRun struct {
	mu     sync.Mutex
	cancel context.CancelFunc
	gen    int
}

var gnCubeMatrix cubeMatrixRun

// ComputeCubeMatrix returns the cube verdict at every away × away score of a
// matchLength-point match, for the position given.
//
// ply 0 is the immediate tier; the frontend repeats the call at display ply
// once settled. A cancelled sweep reports cancellation, never half a grid.
func (a *App) ComputeCubeMatrix(pos domain.Position, matchLength, ply, pruneK int) (gammonnet.CubeMatrix, error) {
	if matchLength < 1 || matchLength > 25 {
		return gammonnet.CubeMatrix{}, fmt.Errorf("match length %d is outside 1-25", matchLength)
	}

	gnCubeMatrix.mu.Lock()
	if gnCubeMatrix.cancel != nil {
		gnCubeMatrix.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	gnCubeMatrix.cancel = cancel
	gnCubeMatrix.gen++
	mine := gnCubeMatrix.gen
	gnCubeMatrix.mu.Unlock()

	defer func() {
		gnCubeMatrix.mu.Lock()
		if gnCubeMatrix.gen == mine {
			gnCubeMatrix.cancel = nil
		}
		gnCubeMatrix.mu.Unlock()
		cancel()
	}()

	return gammonnet.ComputeCubeMatrix(ctx, pos, matchLength, ply, pruneK, 0)
}

// CancelCubeMatrix stops the sweep in flight, if any, within one cell's
// search.
func (a *App) CancelCubeMatrix() {
	gnCubeMatrix.mu.Lock()
	if gnCubeMatrix.cancel != nil {
		gnCubeMatrix.cancel()
		gnCubeMatrix.cancel = nil
	}
	gnCubeMatrix.mu.Unlock()
}
