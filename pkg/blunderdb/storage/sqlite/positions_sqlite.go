package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"iter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type positionStore struct{ db execer }

var _ storage.PositionStore = (*positionStore)(nil)

// positionCols is the column list read back into a Position; the first twelve
// match engine.ReconstructPosition's parameters, and individually_imported is
// applied on top (it is provenance, not identity — see ADR-0001).
const positionCols = `id, state, decision_type, player_on_roll, dice_1, dice_2, ` +
	`cube_value, cube_owner, score_1, score_2, has_jacoby, has_beaver, individually_imported`

// scanPosition reconstructs a Position from a row selected with positionCols.
func scanPosition(sc interface{ Scan(...any) error }) (domain.Position, error) {
	var id int64
	var state string
	var dt, por, d1, d2, cv, co, s1, s2, hj, hb sql.NullInt64
	var individual sql.NullBool
	if err := sc.Scan(&id, &state, &dt, &por, &d1, &d2, &cv, &co, &s1, &s2, &hj, &hb, &individual); err != nil {
		return domain.Position{}, err
	}
	p := engine.ReconstructPosition(id, state,
		int(dt.Int64), int(por.Int64), int(d1.Int64), int(d2.Int64),
		int(cv.Int64), int(co.Int64), int(s1.Int64), int(s2.Int64),
		int(hj.Int64), int(hb.Int64))
	p.IndividuallyImported = individual.Bool
	return p, nil
}

const positionInsertSQL = `INSERT INTO position (
	zobrist_hash, decision_type, player_on_roll, dice_1, dice_2,
	cube_value, cube_owner, score_1, score_2,
	has_jacoby, has_beaver,
	pip_1, pip_2, pip_diff, off_1, off_2,
	back_checkers_1, back_checkers_2, no_contact,
	occupancy_1, occupancy_2, point_mask_1, point_mask_2,
	state, individually_imported
) VALUES (?,?,?,?,?, ?,?,?,?, ?,?, ?,?,?,?,?, ?,?,?, ?,?,?,?, ?,?)
ON CONFLICT(zobrist_hash) DO NOTHING`

// markIndividualSQL raises the provenance flag on an already-stored position.
// It only ever sets, never clears — that is what makes the flag sticky.
const markIndividualSQL = `UPDATE position SET individually_imported = 1
	WHERE zobrist_hash = ? AND individually_imported = 0`

// Save stores p, deduplicated by Zobrist hash: a position whose hash is already
// present is not re-inserted and Save returns the existing id (D1). p is
// updated in place with the storage-normalised board and the resulting id.
//
// p.IndividuallyImported is ORed into the stored value rather than assigned
// (ADR-0001): a match import (which never sets it) cannot clear the flag on a
// position the user had already imported on its own, and an individual import
// of a position a match had already brought in still marks it. The flag is
// therefore independent of the order the user imports their files in.
func (s *positionStore) Save(ctx context.Context, scope string, p *domain.Position) (int64, error) {
	norm := p.NormalizeForStorage()
	cols := engine.PopulatePositionColumns(p)
	res, err := s.db.ExecContext(ctx, positionInsertSQL,
		int64(cols.ZobristHash), cols.DecisionType, norm.PlayerOnRoll, cols.Dice1, cols.Dice2,
		cols.CubeValue, cols.CubeOwner, cols.Score1, cols.Score2,
		cols.HasJacoby, cols.HasBeaver,
		cols.Pip1, cols.Pip2, cols.PipDiff, cols.Off1, cols.Off2,
		cols.BackCheckers1, cols.BackCheckers2, boolToInt(cols.NoContact),
		int64(cols.Occupancy1), int64(cols.Occupancy2), int64(cols.PointMask1), int64(cols.PointMask2),
		engine.EncodeBoardCompact(norm.Board), boolToInt(norm.IndividuallyImported))
	if err != nil {
		return 0, fmt.Errorf("sqlite: save position: %w", err)
	}
	var id int64
	if affected, _ := res.RowsAffected(); affected > 0 {
		if id, err = res.LastInsertId(); err != nil {
			return 0, fmt.Errorf("sqlite: save position id: %w", err)
		}
	} else {
		// Hash already present: keep the existing row, but let an individual
		// import raise the flag on it. Skipped entirely for match imports, so
		// they stay a pure no-op on a duplicate position.
		if norm.IndividuallyImported {
			if _, err := s.db.ExecContext(ctx, markIndividualSQL, int64(cols.ZobristHash)); err != nil {
				return 0, fmt.Errorf("sqlite: mark position individually imported: %w", err)
			}
		}
		if err = s.db.QueryRowContext(ctx,
			`SELECT id FROM position WHERE zobrist_hash = ?`,
			int64(cols.ZobristHash)).Scan(&id); err != nil {
			return 0, fmt.Errorf("sqlite: save position dedup lookup: %w", err)
		}
	}
	norm.ID = id
	*p = norm
	return id, nil
}

const positionUpdateSQL = `UPDATE position SET state = ?,
	zobrist_hash=?, decision_type=?, player_on_roll=?, dice_1=?, dice_2=?,
	cube_value=?, cube_owner=?, score_1=?, score_2=?,
	has_jacoby=?, has_beaver=?,
	pip_1=?, pip_2=?, pip_diff=?, off_1=?, off_2=?,
	back_checkers_1=?, back_checkers_2=?, no_contact=?,
	occupancy_1=?, occupancy_2=?, point_mask_1=?, point_mask_2=?
	WHERE id = ?`

// Update overwrites the stored position with the same id as p.
func (s *positionStore) Update(ctx context.Context, scope string, p *domain.Position) error {
	cols := engine.PopulatePositionColumns(p)
	_, err := s.db.ExecContext(ctx, positionUpdateSQL,
		engine.EncodeBoardCompact(p.Board),
		int64(cols.ZobristHash), cols.DecisionType, p.PlayerOnRoll, cols.Dice1, cols.Dice2,
		cols.CubeValue, cols.CubeOwner, cols.Score1, cols.Score2,
		cols.HasJacoby, cols.HasBeaver,
		cols.Pip1, cols.Pip2, cols.PipDiff, cols.Off1, cols.Off2,
		cols.BackCheckers1, cols.BackCheckers2, boolToInt(cols.NoContact),
		int64(cols.Occupancy1), int64(cols.Occupancy2), int64(cols.PointMask1), int64(cols.PointMask2),
		p.ID)
	if err != nil {
		return fmt.Errorf("sqlite: update position: %w", err)
	}
	return nil
}

// Load returns the position with the given id, or ErrNotFound.
func (s *positionStore) Load(ctx context.Context, scope string, id int64) (*domain.Position, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+positionCols+` FROM position WHERE id = ?`, id)
	p, err := scanPosition(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("sqlite: load position %d: %w", id, storage.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("sqlite: load position %d: %w", id, err)
	}
	return &p, nil
}

// Exists reports whether a position with the given Zobrist hash is stored.
func (s *positionStore) Exists(ctx context.Context, scope string, zobrist uint64) (int64, bool, error) {
	var id int64
	err := s.db.QueryRowContext(ctx,
		`SELECT id FROM position WHERE zobrist_hash = ?`, int64(zobrist)).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("sqlite: position exists: %w", err)
	}
	return id, true, nil
}

// Delete removes the position with the given id; analysis, comments and
// collection links cascade via foreign keys.
func (s *positionStore) Delete(ctx context.Context, scope string, id int64) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM position WHERE id = ?`, id); err != nil {
		return fmt.Errorf("sqlite: delete position %d: %w", id, err)
	}
	return nil
}

// List streams stored positions ordered by id.
func (s *positionStore) List(ctx context.Context, scope string, opts storage.ListOpts) iter.Seq2[*domain.Position, error] {
	return func(yield func(*domain.Position, error) bool) {
		query := `SELECT ` + positionCols + ` FROM position ORDER BY id`
		var args []any
		switch {
		case opts.Limit > 0:
			query += ` LIMIT ?`
			args = append(args, opts.Limit)
			if opts.Offset > 0 {
				query += ` OFFSET ?`
				args = append(args, opts.Offset)
			}
		case opts.Offset > 0:
			query += ` LIMIT -1 OFFSET ?`
			args = append(args, opts.Offset)
		}
		rows, err := s.db.QueryContext(ctx, query, args...)
		if err != nil {
			yield(nil, fmt.Errorf("sqlite: list positions: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			p, err := scanPosition(rows)
			if err != nil {
				yield(nil, fmt.Errorf("sqlite: list positions: %w", err))
				return
			}
			if !yield(&p, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("sqlite: list positions: %w", err))
		}
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
