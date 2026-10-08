package sqlite

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// ReanchorAnsweredDoubles — see storage.MatchStore.
func (s *matchStore) ReanchorAnsweredDoubles(ctx context.Context, scope string) (int, error) {
	moved := 0
	err := withTx(ctx, s.db, func(tx execer) error {
		byPosition, order, err := answeredOwnedCubeMoves(ctx, tx)
		if err != nil || len(order) == 0 {
			return err
		}
		ps := &positionStore{db: tx}
		loaded, err := ps.LoadByIDs(ctx, scope, order)
		if err != nil {
			return fmt.Errorf("load answered positions: %w", err)
		}
		byID := make(map[int64]*domain.Position, len(loaded))
		for i := range loaded {
			byID[loaded[i].ID] = &loaded[i]
		}
		// Left rows are orphan candidates, checked in one DELETE after the
		// loop: a mid-loop check could not see later repoints.
		left := make([]int64, 0, len(order))
		for _, pid := range order {
			pos, ok := byID[pid]
			if !ok {
				return fmt.Errorf("load answered position %d: %w", pid, storage.ErrNotFound)
			}
			pos.Cube.Owner = domain.None
			newID, err := ps.Save(ctx, scope, pos)
			if err != nil {
				return fmt.Errorf("save answered position: %w", err)
			}
			for _, mv := range byPosition[pid] {
				if _, err := tx.ExecContext(ctx, `UPDATE move SET position_id = ? WHERE id = ?`, newID, mv.move); err != nil {
					return fmt.Errorf("repoint answered move: %w", err)
				}
				if _, err := tx.ExecContext(ctx, positionMatchDateOnMoveSQL, mv.game, newID); err != nil {
					return fmt.Errorf("date answered position: %w", err)
				}
				moved++
			}
			if _, err := tx.ExecContext(ctx, invalidateMatchStatsOfPositionSQL, newID); err != nil {
				return fmt.Errorf("invalidate match stats: %w", err)
			}
			left = append(left, pid)
		}
		if err := RefreshPositionMatchDates(ctx, tx, left); err != nil {
			return err
		}
		return deleteOrphanedPositions(ctx, tx, left)
	})
	if err != nil {
		return 0, fmt.Errorf("sqlite: reanchor answered doubles: %w", err)
	}
	return moved, nil
}

// answeredMove is one row of sqlshared.AnsweredOwnedCubeMovesSQL.
type answeredMove struct{ move, game int64 }

// answeredOwnedCubeMoves groups the rows of sqlshared.AnsweredOwnedCubeMovesSQL
// by position, positions in id order.
func answeredOwnedCubeMoves(ctx context.Context, tx execer) (map[int64][]answeredMove, []int64, error) {
	rows, err := tx.QueryContext(ctx, sqlshared.AnsweredOwnedCubeMovesSQL+` ORDER BY mv.position_id, mv.id`)
	if err != nil {
		return nil, nil, fmt.Errorf("list answered moves: %w", err)
	}
	defer rows.Close()
	byPosition := map[int64][]answeredMove{}
	var order []int64
	for rows.Next() {
		var mv answeredMove
		var pid int64
		if err := rows.Scan(&mv.move, &mv.game, &pid); err != nil {
			return nil, nil, fmt.Errorf("scan answered move: %w", err)
		}
		if _, seen := byPosition[pid]; !seen {
			order = append(order, pid)
		}
		byPosition[pid] = append(byPosition[pid], mv)
	}
	return byPosition, order, rows.Err()
}
