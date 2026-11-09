package sqlshared

import "context"

// RescorePositionMoves implements storage.MatchStore.RescorePositionMoves for
// both backends. The per-move columns are scored by the position's analysis
// alone; the scope names the tenant the keys lead with.
func RescorePositionMoves(ctx context.Context, db Execer, scope string, positionID int64) error {
	_, err := RescorePlayedDecisionsOf(ctx, db, scope, []int64{positionID})
	if err != nil {
		return errf(db, "rescore position moves", err)
	}
	return nil
}
