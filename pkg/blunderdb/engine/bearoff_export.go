package engine

import "fmt"

// BearoffIndex converts a one-sided checker arrangement on 6 points to the
// combinatorial index used by every gnubg bearoff database (one- and
// two-sided). anBoard[0..5] = checkers on points 1-6. Exported for
// engine/race, which addresses two-sided databases with the same indexing.
func BearoffIndex(anBoard [6]int, nPoints, nCheckers int) int {
	return positionBearoff(anBoard, nPoints, nCheckers)
}

// RollDistribution returns the probability distribution of the number of
// rolls needed to bear off all checkers of a one-sided position, read from
// the embedded OS-06-15 database: probs[n] = P(bear off in exactly n rolls),
// n = 0..31, under gnubg's one-sided optimal play.
func RollDistribution(anBoard [6]int) ([]float64, error) {
	db := oneSided()
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
	posID := positionBearoff(anBoard, db.nPoints, db.nCheckers)
	return db.getDistribution(posID)
}
