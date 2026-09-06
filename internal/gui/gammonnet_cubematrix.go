package gui

import (
	"context"
	"fmt"
	"sync"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
)

// The cube matrix's desktop half (issue #267, fiche I.11).
//
// Two tiers, like the Eval panel's own escalation (#125): the grid comes back
// at 0-ply on the gesture — a few milliseconds, so the shape of the answer is
// on screen at once — and the configured display depth follows once the user
// has stopped moving. What differs from the panel's escalation is that a grid
// is not one search but matchLength² of them, so the deep tier is explicitly
// cancellable and a superseded run's answer is dropped rather than shown.
//
// The grid itself is gammonnet.ComputeCubeMatrix, shared with the CLI's
// cubematrix and the daemon's /v1/gammonnet.cubeMatrix: three surfaces, one
// answer.

// cubeMatrixRun is the single in-flight deep sweep. Only one exists: the
// modal shows one position at a time, and a superseded sweep has nothing left
// to say.
type cubeMatrixRun struct {
	mu     sync.Mutex
	cancel context.CancelFunc
	gen    int
}

var gnCubeMatrix cubeMatrixRun

// ComputeCubeMatrix returns the cube verdict at every away × away score of a
// matchLength-point match, for the position given.
//
// ply 0 is the immediate tier and runs synchronously on every core; the
// deeper tier is the same call with the configured display ply, which the
// frontend issues once the position has settled. Either can be superseded:
// CancelCubeMatrix stops whatever is running, and a cancelled sweep reports
// its cancellation rather than half a grid.
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

// CancelCubeMatrix stops the sweep in flight, if any — closing the modal, or
// leaving the position it was computed for. Cancellation is cooperative and
// checked between cells, so a 9-point grid stops within one cell's search.
func (a *App) CancelCubeMatrix() {
	gnCubeMatrix.mu.Lock()
	if gnCubeMatrix.cancel != nil {
		gnCubeMatrix.cancel()
		gnCubeMatrix.cancel = nil
	}
	gnCubeMatrix.mu.Unlock()
}
