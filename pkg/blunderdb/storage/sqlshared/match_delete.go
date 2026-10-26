package sqlshared

import (
	"context"
	"fmt"
)

// RefreshPositionDates recomputes position.match_date for ids, inside tx, from
// every match that still reaches them. The two backends write it differently
// (batched IN lists on SQLite, one ANY array on PostgreSQL), so a backend
// passes its own.
type RefreshPositionDates func(ctx context.Context, tx Execer, scope string, ids []int64) error

// positionIsHeldSQL reports whether anything still holds a position once the
// match that referenced it is gone: deleting a match must not destroy the
// user's work on a position that merely occurred in it. It is written once
// here for both backends; only TenantFilter and Bool differ.
//
// A position is held by: another match's move; a collection membership; an
// Anki card; a Lesson step (ADR-0066); a comment with origin = 'user'; the
// individually_imported mark (ADR-0001); or the source tool's study mark
// (ADR-0006).
//
// Deliberately NOT held by an analysis (every match position has one, so
// nothing would ever be purged), nor by an imported or 'unknown'-origin
// comment (the source file's per-move notes, see ingest/xg.go).
//
// The result is a WHERE fragment correlated against the outer `position` row,
// embedded in PurgeOrphanPositions' set-based DELETE, with the arguments its
// placeholders take, in order.
func positionIsHeldSQL(d Dialect, scope string) (string, []any) {
	var args []any
	holder := func(table, extra string) string {
		tf, targs := d.TenantFilter("", scope)
		args = append(args, targs...)
		return `EXISTS (SELECT 1 FROM ` + table + ` WHERE position_id = position.id AND ` + tf + extra + `)`
	}
	pred := holder("move", "") +
		` OR ` + holder("collection_position", "") +
		` OR ` + holder("anki_card", "") +
		` OR ` + holder("lesson_step", "") +
		` OR ` + holder("comment", ` AND origin = 'user'`) +
		` OR ` + d.Bool("position.individually_imported", true) +
		` OR ` + d.Bool("position.flagged", true)
	return pred, args
}

// PositionIsHeld exposes positionIsHeldSQL so a test of another package can
// assert on the one real definition instead of a copy.
func PositionIsHeld(d Dialect, scope string) (string, []any) { return positionIsHeldSQL(d, scope) }

// PurgeOrphanPositions removes every position in ids (within the scope's
// tenant) that positionIsHeldSQL says nothing holds any more, as a single
// set-based DELETE rather than one EXISTS round-trip per id.
func PurgeOrphanPositions(ctx context.Context, db Execer, scope string, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	tf, args := db.TenantFilter("", scope)
	held, heldArgs := positionIsHeldSQL(db, scope)
	args = append(append(args, int64Args(ids)...), heldArgs...)
	query := `DELETE FROM position WHERE ` + tf + ` AND id IN (` + Placeholders(len(ids)) + `) AND NOT (` + held + `)`
	if _, err := db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("delete orphaned positions: %w", err)
	}
	return nil
}

// matchPositionIDs lists the positions the moves of matchID reference.
func matchPositionIDs(ctx context.Context, db Execer, scope string, matchID int64) ([]int64, error) {
	tf, args := db.TenantFilter("mv", scope)
	rows, err := db.Query(ctx,
		`SELECT DISTINCT mv.position_id
		 FROM move mv INNER JOIN game g ON mv.game_id = g.id
		 WHERE `+tf+` AND g.match_id = ? AND mv.position_id IS NOT NULL`,
		append(args, matchID)...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// invalidateMatchStats drops the cached stats of matchID: the edit about to
// run changes what they were computed from.
func invalidateMatchStats(ctx context.Context, db Execer, scope string, matchID int64) error {
	tf, args := db.TenantFilter("", scope)
	if _, err := db.Exec(ctx, `DELETE FROM match_stats WHERE `+tf+` AND match_id = ?`, append(args, matchID)...); err != nil {
		return fmt.Errorf("invalidate match stats: %w", err)
	}
	return nil
}

// DeleteMatchCascade removes a match and all of its games, moves and move
// analyses (via ON DELETE CASCADE), then deletes any position the match
// referenced that nothing else holds (see positionIsHeldSQL). The cascade and
// the orphan cleanup run in one transaction (the caller's, when db is bound to
// one).
func DeleteMatchCascade(ctx context.Context, db Execer, scope string, id int64, refresh RefreshPositionDates) error {
	err := db.Transact(ctx, func(tx Execer) error {
		// Collected before the cascade removes the moves that name them.
		positionIDs, err := matchPositionIDs(ctx, tx, scope, id)
		if err != nil {
			return fmt.Errorf("collect positions: %w", err)
		}
		if err := invalidateMatchStats(ctx, tx, scope, id); err != nil {
			return err
		}
		tf, args := tx.TenantFilter("", scope)
		// game/move/move_analysis cascade off the match delete.
		if _, err := tx.Exec(ctx, `DELETE FROM match WHERE `+tf+` AND id = ?`, append(args, id)...); err != nil {
			return err
		}
		if err := PurgeOrphanPositions(ctx, tx, scope, positionIDs); err != nil {
			return err
		}
		return refresh(ctx, tx, scope, positionIDs)
	})
	if err != nil {
		return errf(db, fmt.Sprintf("delete match %d", id), err)
	}
	return nil
}

// SwapMatchPlayers swaps player 1 and player 2 for the match — see
// storage.MatchStore. Positions are stored from the side on roll, so no
// position, analysis or played-decision score changes: only what names a seat.
// A match with a draft open on it, or played here, is refused.
func SwapMatchPlayers(ctx context.Context, db Execer, scope string, id int64) error {
	err := db.Transact(ctx, func(tx Execer) error {
		if err := SwapRefusal(ctx, tx, scope, id); err != nil {
			return err
		}
		// The seats trade places: each row now describes the other player.
		if err := invalidateMatchStats(ctx, tx, scope, id); err != nil {
			return err
		}
		tf, targs := tx.TenantFilter("", scope)
		withID := func() []any { return append(append([]any{}, targs...), id) }
		if _, err := tx.Exec(ctx,
			`UPDATE match SET player1_name = player2_name, player2_name = player1_name,
			                  player1_elo = player2_elo, player2_elo = player1_elo,
			                  player1_experience = player2_experience,
			                  player2_experience = player1_experience
			 WHERE `+tf+` AND id = ?`, withID()...); err != nil {
			return fmt.Errorf("swap names: %w", err)
		}
		if _, err := tx.Exec(ctx,
			`UPDATE game SET initial_score_1 = initial_score_2,
			                 initial_score_2 = initial_score_1,
			                 winner = -winner
			 WHERE `+tf+` AND match_id = ?`, withID()...); err != nil {
			return fmt.Errorf("swap game scores: %w", err)
		}
		if _, err := tx.Exec(ctx,
			`UPDATE move SET player = -player
			 WHERE `+tf+` AND game_id IN (SELECT id FROM game WHERE match_id = ?)`, withID()...); err != nil {
			return fmt.Errorf("swap move players: %w", err)
		}
		return nil
	})
	if err != nil {
		return errf(db, fmt.Sprintf("swap players for match %d", id), err)
	}
	return nil
}
