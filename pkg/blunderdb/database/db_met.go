package database

import (
	"github.com/kevung/gnubgparser"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// The GNUbg Match Equity Table machinery (Kazaross-XG2 + Zadeh fallback) lives
// in package engine (engine/met.go) so both the SQLite Storage backend and this
// wrapper can convert equities to MWC. Only convertGnuBGCubeMWCToEMG stays here:
// it is coupled to the gnubgparser types and used solely by the GnuBG import.

// convertGnuBGCubeMWCToEMG converts GNUbg cubeful equity values from Match Winning
// Chances (MWC, 0.0-1.0 scale) to Equivalent Money Game equity (EMG) for match play.
//
// This is a faithful implementation of GNUbg's mwc2eq() function from eval.c.
// The conversion uses the match equity table to compute MWC reference points:
//
//	rMwcWin  = getME(score0, score1, matchTo, fMove, cube, fMove, ...)    // MWC if "I" win
//	rMwcLose = getME(score0, score1, matchTo, fMove, cube, !fMove, ...)   // MWC if "I" lose
//	EMG = (2*MWC - (rMwcWin + rMwcLose)) / (rMwcWin - rMwcLose)
//
// This linear mapping ensures EMG=+1 when MWC=rMwcWin and EMG=-1 when MWC=rMwcLose.
//
// Parameters:
//   - score0, score1: absolute match scores for player 0 and player 1
//   - fMove: which player (0 or 1) is making the cube decision
//   - cubeValue: current cube value
//   - matchLength: match length (0 for money game)
func convertGnuBGCubeMWCToEMG(analysis *gnubgparser.CubeAnalysis, score0, score1, fMove, cubeValue, matchLength int) {
	if matchLength <= 0 || analysis == nil {
		return // Money game or nil analysis — no conversion needed
	}

	// For cube decisions, the game is never Crawford (cube is dead in Crawford games)
	fCrawford := false

	// MWC reference points: what happens if I win/lose the current game at this cube level
	// Use float32 throughout to match GNUbg's C float arithmetic exactly.
	// GNUbg reads DA values as float (via "(float) g_ascii_strtod"), stores MET as float,
	// and computes mwc2eq entirely in float. Using float64 would introduce subtle differences
	// that can flip BestAction at decision boundaries.
	mwcWin := float32(engine.GnuBGGetME(score0, score1, matchLength, fMove, cubeValue, fMove, fCrawford))
	mwcLose := float32(engine.GnuBGGetME(score0, score1, matchLength, fMove, cubeValue, 1-fMove, fCrawford))

	denom := mwcWin - mwcLose
	if denom < 1e-7 && denom > -1e-7 {
		// Degenerate case (e.g. dead cube) — keep raw values, set DP=1.0
		analysis.CubefulDoublePass = 1.0
		return
	}

	sum := mwcWin + mwcLose

	// Truncate input MWC to float32 to match GNUbg's "(float) g_ascii_strtod" behavior
	ndMwc := float32(analysis.CubefulNoDouble)
	dtMwc := float32(analysis.CubefulDoubleTake)

	// Convert ND and DT from MWC to EMG using GNUbg's mwc2eq formula (in float32)
	analysis.CubefulNoDouble = float64((2.0*ndMwc - sum) / denom)
	analysis.CubefulDoubleTake = float64((2.0*dtMwc - sum) / denom)
	analysis.CubefulDoublePass = 1.0 // DP is always 1.0 in EMG (by definition of the mapping)

	// Recompute BestAction with the converted EMG values
	effectiveDouble := analysis.CubefulDoubleTake
	if analysis.CubefulDoublePass < analysis.CubefulDoubleTake {
		effectiveDouble = analysis.CubefulDoublePass
	}
	if effectiveDouble > analysis.CubefulNoDouble {
		if analysis.CubefulDoubleTake <= analysis.CubefulDoublePass {
			analysis.BestAction = "Double, Take"
		} else {
			analysis.BestAction = "Double, Pass"
		}
	} else {
		analysis.BestAction = "No Double"
	}
}

// ConvertEMGLossToMWCLoss is re-exported from package engine so the database
// package (and its callers, e.g. the CLI) keep referencing the unqualified
// name. New code should call engine.ConvertEMGLossToMWCLoss directly.
var ConvertEMGLossToMWCLoss = engine.ConvertEMGLossToMWCLoss
