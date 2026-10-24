package sqlshared

import "context"

// RescorePositionMoves implements storage.MatchStore.RescorePositionMoves for
// both backends. The per-move columns are scored by the position's analysis
// alone, so the scope adds nothing to the position id.
func RescorePositionMoves(ctx context.Context, db Execer, _ string, positionID int64) error {
	_, err := RescorePlayedDecisionsOf(ctx, db, []int64{positionID})
	if err != nil {
		return errf(db, "rescore position moves", err)
	}
	return nil
}
