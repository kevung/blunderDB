package database

import (
	"testing"
)

// Deleting an analysis unscores the moves played from its position, as the
// contract's Delete does for the serve daemon: a move scored by an analysis
// that no longer exists would keep counting as an error in the search's E
// filter and in the match statistics.
func TestDeleteAnalysisUnscoresPlayedMoves(t *testing.T) {
	db := newTestDBWithXG(t)

	var positionID int64
	if err := db.db.QueryRow(`SELECT mv.position_id FROM move mv
		JOIN analysis a ON a.position_id = mv.position_id
		WHERE mv.decision_error_mp IS NOT NULL ORDER BY mv.id LIMIT 1`).Scan(&positionID); err != nil {
		t.Fatalf("no scored move in the fixture: %v", err)
	}

	if err := db.DeleteAnalysis(positionID); err != nil {
		t.Fatalf("DeleteAnalysis: %v", err)
	}

	var scored int
	if err := db.db.QueryRow(`SELECT COUNT(*) FROM move WHERE position_id = ? AND decision_error_mp IS NOT NULL`, positionID).Scan(&scored); err != nil {
		t.Fatal(err)
	}
	if scored != 0 {
		t.Errorf("%d move(s) of position %d still scored after its analysis was deleted", scored, positionID)
	}
	if _, err := db.LoadAnalysis(positionID); err == nil {
		t.Errorf("analysis of position %d still loads after DeleteAnalysis", positionID)
	}
}
