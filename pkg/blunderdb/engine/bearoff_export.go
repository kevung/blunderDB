package engine

import "fmt"

// BearoffIndex converts a one-sided checker arrangement to the combinatorial
// index used by every gnubg bearoff database (one- and two-sided).
// anBoard[i] = checkers on point i+1. Exported for engine/race, which
// addresses two-sided databases with the same indexing.
func BearoffIndex(anBoard [6]int, nPoints, nCheckers int) int {
	return positionBearoff(anBoard[:], nPoints, nCheckers)
}

// RollDistribution returns the probability distribution of the number of
// rolls needed to bear off all checkers of a one-sided position, read from
// the loaded one-sided database: probs[n] = P(bear off in exactly n rolls),
// n = 0..31, under gnubg's one-sided optimal play.
func RollDistribution(anBoard [6]int) ([]float64, error) {
	return RollDistributionPoints(anBoard[:])
}

// RollDistributionPoints is RollDistribution for a position of any width the
// loaded table covers — the seven-, eight- or ten-point tables that let the
// EPC and the convolution answer for a side whose farthest chequer is outside
// the home board (ADR-0027 §9).
func RollDistributionPoints(anBoard []int) ([]float64, error) {
	bearoffMu.RLock()
	db := globalBearoffDB
	bearoffMu.RUnlock()
	if db == nil {
		return nil, fmt.Errorf("bearoff database not loaded")
	}
	total := 0
	for _, c := range anBoard {
		if c < 0 {
			return nil, fmt.Errorf("invalid negative checker count")
		}
		total += c
	}
	if total > db.nCheckers {
		return nil, fmt.Errorf("too many checkers: %d (max %d)", total, db.nCheckers)
	}
	for i := db.nPoints; i < len(anBoard); i++ {
		if anBoard[i] > 0 {
			return nil, fmt.Errorf("a chequer stands on point %d, outside the %d-point table", i+1, db.nPoints)
		}
	}
	board := make([]int, db.nPoints)
	copy(board, anBoard)
	posID := positionBearoff(board, db.nPoints, db.nCheckers)
	return db.getDistribution(posID)
}
