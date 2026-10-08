package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// ReanchorAnsweredDoubles — see storage.MatchStore; the SQLite twin's steps,
// confined to the tenant.
func (s *matchStore) ReanchorAnsweredDoubles(ctx context.Context, scope string) (int, error) {
	tenant := tenantID(scope)
	moved := 0
	err := s.inTx(ctx, "reanchor answered doubles", func(tx pgx.Tx) error {
		byPosition, order, err := answeredOwnedCubeMoves(ctx, tx, tenant)
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
				if _, err := tx.Exec(ctx, `UPDATE move SET position_id = $1 WHERE id = $2 AND tenant_id = $3`, newID, mv.move, tenant); err != nil {
					return fmt.Errorf("repoint answered move: %w", err)
				}
				if _, err := tx.Exec(ctx, positionMatchDateOnMoveSQL, mv.game, newID, tenant); err != nil {
					return fmt.Errorf("date answered position: %w", err)
				}
				moved++
			}
			if _, err := tx.Exec(ctx, `DELETE FROM match_stats WHERE tenant_id = $1 AND match_id IN
				(SELECT g.match_id FROM move mv JOIN game g ON g.id = mv.game_id AND g.tenant_id = mv.tenant_id
				  WHERE mv.position_id = $2 AND mv.tenant_id = $1)`, tenant, newID); err != nil {
				return fmt.Errorf("invalidate match stats: %w", err)
			}
			left = append(left, pid)
		}
		if err := refreshPositionMatchDates(ctx, tx, tenant, left); err != nil {
			return err
		}
		return deleteOrphanedPositions(ctx, tx, tenant, left)
	})
	return moved, err
}

// answeredMove is one row of sqlshared.AnsweredOwnedCubeMovesSQL.
type answeredMove struct{ move, game int64 }

// answeredOwnedCubeMoves groups the tenant's rows of
// sqlshared.AnsweredOwnedCubeMovesSQL by position, positions in id order.
func answeredOwnedCubeMoves(ctx context.Context, tx pgx.Tx, tenant int64) (map[int64][]answeredMove, []int64, error) {
	rows, err := tx.Query(ctx, sqlshared.AnsweredOwnedCubeMovesSQL+`
	  AND mv.tenant_id = $1 AND g.tenant_id = $1 AND m.tenant_id = $1 AND p.tenant_id = $1
	ORDER BY mv.position_id, mv.id`, tenant)
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
