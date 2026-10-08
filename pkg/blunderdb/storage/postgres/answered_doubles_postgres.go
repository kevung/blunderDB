package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// ReanchorAnsweredDoubles — see storage.MatchStore; the SQLite twin's steps,
// confined to the tenant.
func (s *matchStore) ReanchorAnsweredDoubles(ctx context.Context, scope string) (int, error) {
	moved := 0
	err := s.inTx(ctx, "reanchor answered doubles", func(tx pgx.Tx) error {
		var err error
		moved, err = reanchorAnsweredDoublesTx(ctx, tx, scope)
		return err
	})
	return moved, err
}

// reanchorAnsweredDoublesTx is ReanchorAnsweredDoubles inside tx, which the
// caller opens: the store's own transaction, or the migration's unforced one.
func reanchorAnsweredDoublesTx(ctx context.Context, tx pgx.Tx, scope string) (int, error) {
	tenant := tenantID(scope)
	byPosition, order, err := answeredOwnedCubeMoves(ctx, tx, tenant)
	if err != nil || len(order) == 0 {
		return 0, err
	}
	ps := &positionStore{db: tx}
	loaded, err := ps.LoadByIDs(ctx, scope, order)
	if err != nil {
		return 0, fmt.Errorf("load answered positions: %w", err)
	}
	byID := make(map[int64]*domain.Position, len(loaded))
	for i := range loaded {
		byID[loaded[i].ID] = &loaded[i]
	}
	moved := 0
	left := make([]int64, 0, len(order))
	landed := make([]int64, 0, len(order))
	for _, pid := range order {
		pos, ok := byID[pid]
		if !ok {
			return 0, fmt.Errorf("load answered position %d: %w", pid, storage.ErrNotFound)
		}
		pos.Cube.Owner = domain.None
		// Provenance stays with the left row: Save ORs it into the landed
		// one, which would then be held and filtered as the user's own.
		pos.IndividuallyImported, pos.Flagged = false, false
		newID, err := ps.Save(ctx, scope, pos)
		if err != nil {
			return 0, fmt.Errorf("save answered position: %w", err)
		}
		for _, mv := range byPosition[pid] {
			if _, err := tx.Exec(ctx, `UPDATE move SET position_id = $1 WHERE id = $2 AND tenant_id = $3`, newID, mv.move, tenant); err != nil {
				return 0, fmt.Errorf("repoint answered move: %w", err)
			}
			if _, err := tx.Exec(ctx, positionMatchDateOnMoveSQL, mv.game, newID, tenant); err != nil {
				return 0, fmt.Errorf("date answered position: %w", err)
			}
			moved++
		}
		if _, err := tx.Exec(ctx, `DELETE FROM match_stats WHERE tenant_id = $1 AND match_id IN
			(SELECT g.match_id FROM move mv JOIN game g ON g.id = mv.game_id AND g.tenant_id = mv.tenant_id
			  WHERE mv.position_id = $2 AND mv.tenant_id = $1)`, tenant, newID); err != nil {
			return 0, fmt.Errorf("invalidate match stats: %w", err)
		}
		left = append(left, pid)
		landed = append(landed, newID)
	}
	// A moved move is scored by the analysis of the row it lands on.
	if _, err := sqlshared.RescorePlayedDecisionsOf(ctx, binder{tx}.shared(), landed); err != nil {
		return 0, fmt.Errorf("rescore answered moves: %w", err)
	}
	if err := refreshPositionMatchDates(ctx, tx, tenant, left); err != nil {
		return 0, err
	}
	if err := deleteOrphanedPositions(ctx, tx, tenant, left); err != nil {
		return 0, err
	}
	return moved, nil
}

// responseAnalysisTables are the tables dropGammonNetResponseAnalyses reads
// or writes, every tenant's rows of which it must see.
var responseAnalysisTables = []string{
	"analysis", "position", "move", "game",
	"match_stats", "match_stats_cell", "match_stats_position",
}

// dropGammonNetResponseAnalyses runs sqlshared.DropGammonNetResponseAnalyses
// over every tenant in one unforced transaction: analyses, moves and match
// statistics are reached by id, which no tenant shares. One of
// runGoBackfills' passes, ahead of reanchorAnsweredDoubles so a moved answer
// is rescored against what its new row keeps.
func dropGammonNetResponseAnalyses(ctx context.Context, conn beginner) (bool, error) {
	n := 0
	complete, err := inUnforcedTx(ctx, conn, responseAnalysisTables, func(tx pgx.Tx) error {
		var err error
		n, err = sqlshared.DropGammonNetResponseAnalyses(ctx, binder{tx}.shared())
		return err
	})
	if err != nil {
		return false, err
	}
	if n > 0 {
		slog.Info("dropped gammonNet's verdicts on take/pass positions for reanalysis", "analyses", n)
	}
	return complete, nil
}

// reanchorProbeTables are the tables the tenant listing of
// reanchorAnsweredDoubles reads, every tenant's rows of which it must see.
var reanchorProbeTables = []string{"move", "game", "match", "position"}

// reanchorAnsweredDoubles runs ReanchorAnsweredDoubles over every tenant
// that holds an answered owned-cube move. Only the listing lifts FORCE; each
// tenant is then moved in its own transaction bound to that tenant, so the
// retention check before the purge sees exactly the rows a request of that
// tenant would — a check run unbound on a FORCEd table would see none and
// purge a held position. A tenant written is a tenant kept: the next start
// lists only those left. One of runGoBackfills' passes.
func reanchorAnsweredDoubles(ctx context.Context, conn beginner) (bool, error) {
	var tenants []int64
	complete, err := inUnforcedTx(ctx, conn, reanchorProbeTables, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT DISTINCT mv.tenant_id FROM (`+sqlshared.AnsweredOwnedCubeMovesSQL+`
			  AND g.tenant_id = mv.tenant_id AND m.tenant_id = mv.tenant_id AND p.tenant_id = mv.tenant_id) a
			JOIN move mv ON mv.id = a.id ORDER BY 1`)
		if err != nil {
			return fmt.Errorf("postgres: list tenants with answered doubles: %w", err)
		}
		tenants, err = pgx.CollectRows(rows, pgx.RowTo[int64])
		return err
	})
	if err != nil {
		return false, err
	}
	total := 0
	for _, tenant := range tenants {
		n, err := reanchorTenantAnsweredDoubles(ctx, conn, tenant)
		if err != nil {
			return false, fmt.Errorf("postgres: reanchor answered doubles of tenant %d: %w", tenant, err)
		}
		total += n
	}
	if total > 0 {
		slog.Info("moved the transcribed answers to the double onto the ownerless cube", "moves", total)
	}
	return complete, nil
}

// reanchorTenantAnsweredDoubles is one tenant's transaction of
// reanchorAnsweredDoubles, its app.tenant_id bound for the transaction only.
func reanchorTenantAnsweredDoubles(ctx context.Context, conn beginner, tenant int64) (int, error) {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if _, err := tx.Exec(ctx, `SELECT set_config('app.tenant_id', $1, true)`, strconv.FormatInt(tenant, 10)); err != nil {
		return 0, fmt.Errorf("bind tenant: %w", err)
	}
	scope := ""
	if tenant != 0 {
		scope = strconv.FormatInt(tenant, 10)
	}
	n, err := reanchorAnsweredDoublesTx(ctx, tx, scope)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return n, nil
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
