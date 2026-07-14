package postgres

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type positionStore struct{ db execer }

var _ storage.PositionStore = (*positionStore)(nil)

// scanner is satisfied by both pgx.Row and pgx.Rows.
type scanner interface{ Scan(dest ...any) error }

// tenantID maps a scope string to the tenant_id column value. An empty scope
// maps to tenant 0 (the reserved public tenant). gammonGo sends the numeric
// tenant identifier via the X-Tenant-ID header for every call.
func tenantID(scope string) int64 {
	if scope == "" {
		return 0
	}
	n, _ := strconv.ParseInt(scope, 10, 64)
	return n
}

// positionSelectCols is the column list read back into a Position; the first
// twelve match engine.ReconstructPosition's parameters, and
// individually_imported is applied on top (provenance, not identity — see
// docs/adr/0001).
const positionSelectCols = `id, state, decision_type, player_on_roll, dice_1, dice_2, ` +
	`cube_value, cube_owner, score_1, score_2, has_jacoby, has_beaver, individually_imported`

// scanPosition reconstructs a Position from a row selected with
// positionSelectCols. The denormalised integer columns are nullable, so they
// scan into pointers; has_jacoby/has_beaver are BOOLEAN.
func scanPosition(sc scanner) (domain.Position, error) {
	var id int64
	var state string
	var dt, por, d1, d2, cv, co, s1, s2 *int64
	var hj, hb *bool
	var individual *bool
	if err := sc.Scan(&id, &state, &dt, &por, &d1, &d2, &cv, &co, &s1, &s2, &hj, &hb, &individual); err != nil {
		return domain.Position{}, err
	}
	p := engine.ReconstructPosition(id, state,
		derefInt(dt), derefInt(por), derefInt(d1), derefInt(d2),
		derefInt(cv), derefInt(co), derefInt(s1), derefInt(s2),
		boolToIntPtr(hj), boolToIntPtr(hb))
	p.IndividuallyImported = individual != nil && *individual
	return p, nil
}

func derefInt(p *int64) int {
	if p == nil {
		return 0
	}
	return int(*p)
}

func boolToIntPtr(p *bool) int {
	if p != nil && *p {
		return 1
	}
	return 0
}

const positionInsertSQL = `INSERT INTO position (
	tenant_id, zobrist_hash, decision_type, player_on_roll, dice_1, dice_2,
	cube_value, cube_owner, score_1, score_2,
	has_jacoby, has_beaver,
	pip_1, pip_2, pip_diff, off_1, off_2,
	back_checkers_1, back_checkers_2, no_contact,
	occupancy_1, occupancy_2, point_mask_1, point_mask_2,
	state, individually_imported
) VALUES ($1,$2,$3,$4,$5,$6, $7,$8,$9,$10, $11,$12, $13,$14,$15,$16,$17, $18,$19,$20, $21,$22,$23,$24, $25,$26)
ON CONFLICT (tenant_id, zobrist_hash) DO NOTHING
RETURNING id`

// markIndividualSQL raises the provenance flag on an already-stored position.
// It only ever sets, never clears — that is what makes the flag sticky.
const markIndividualSQL = `UPDATE position SET individually_imported = TRUE
	WHERE tenant_id = $1 AND zobrist_hash = $2 AND NOT individually_imported`

// Save stores p, deduplicated per tenant by Zobrist hash: a position whose
// (tenant_id, zobrist_hash) is already present is not re-inserted and Save
// returns the existing id. p is updated in place with the storage-normalised
// board and the resulting id.
//
// p.IndividuallyImported is ORed into the stored value rather than assigned
// (docs/adr/0001): a match import (which never sets it) cannot clear the flag on
// a position the user had already imported on its own, and an individual import
// of a position a match had already brought in still marks it. The flag is
// therefore independent of the order the user imports their files in.
func (s *positionStore) Save(ctx context.Context, scope string, p *domain.Position) (int64, error) {
	tenant := tenantID(scope)
	norm := p.NormalizeForStorage()
	cols := engine.PopulatePositionColumns(p)

	var id int64
	err := s.db.QueryRow(ctx, positionInsertSQL,
		tenant, int64(cols.ZobristHash), cols.DecisionType, norm.PlayerOnRoll, cols.Dice1, cols.Dice2,
		cols.CubeValue, cols.CubeOwner, cols.Score1, cols.Score2,
		cols.HasJacoby != 0, cols.HasBeaver != 0,
		cols.Pip1, cols.Pip2, cols.PipDiff, cols.Off1, cols.Off2,
		cols.BackCheckers1, cols.BackCheckers2, cols.NoContact,
		int64(cols.Occupancy1), int64(cols.Occupancy2), int64(cols.PointMask1), int64(cols.PointMask2),
		engine.EncodeBoardCompact(norm.Board), norm.IndividuallyImported).Scan(&id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		// Hash already present for this tenant: keep the existing row, but let
		// an individual import raise the flag on it. Skipped entirely for match
		// imports, so they stay a pure no-op on a duplicate position.
		if norm.IndividuallyImported {
			if _, err := s.db.Exec(ctx, markIndividualSQL, tenant, int64(cols.ZobristHash)); err != nil {
				return 0, fmt.Errorf("postgres: mark position individually imported: %w", err)
			}
		}
		if err = s.db.QueryRow(ctx,
			`SELECT id FROM position WHERE tenant_id = $1 AND zobrist_hash = $2`,
			tenant, int64(cols.ZobristHash)).Scan(&id); err != nil {
			return 0, fmt.Errorf("postgres: save position dedup lookup: %w", err)
		}
	case err != nil:
		return 0, fmt.Errorf("postgres: save position: %w", err)
	}
	norm.ID = id
	*p = norm
	return id, nil
}

const positionUpdateSQL = `UPDATE position SET state = $1,
	zobrist_hash=$2, decision_type=$3, player_on_roll=$4, dice_1=$5, dice_2=$6,
	cube_value=$7, cube_owner=$8, score_1=$9, score_2=$10,
	has_jacoby=$11, has_beaver=$12,
	pip_1=$13, pip_2=$14, pip_diff=$15, off_1=$16, off_2=$17,
	back_checkers_1=$18, back_checkers_2=$19, no_contact=$20,
	occupancy_1=$21, occupancy_2=$22, point_mask_1=$23, point_mask_2=$24
	WHERE id = $25 AND tenant_id = $26`

// Update overwrites the stored position with the same id as p.
func (s *positionStore) Update(ctx context.Context, scope string, p *domain.Position) error {
	cols := engine.PopulatePositionColumns(p)
	_, err := s.db.Exec(ctx, positionUpdateSQL,
		engine.EncodeBoardCompact(p.Board),
		int64(cols.ZobristHash), cols.DecisionType, p.PlayerOnRoll, cols.Dice1, cols.Dice2,
		cols.CubeValue, cols.CubeOwner, cols.Score1, cols.Score2,
		cols.HasJacoby != 0, cols.HasBeaver != 0,
		cols.Pip1, cols.Pip2, cols.PipDiff, cols.Off1, cols.Off2,
		cols.BackCheckers1, cols.BackCheckers2, cols.NoContact,
		int64(cols.Occupancy1), int64(cols.Occupancy2), int64(cols.PointMask1), int64(cols.PointMask2),
		p.ID, tenantID(scope))
	if err != nil {
		return fmt.Errorf("postgres: update position: %w", err)
	}
	return nil
}

// Load returns the position with the given id, or ErrNotFound.
func (s *positionStore) Load(ctx context.Context, scope string, id int64) (*domain.Position, error) {
	row := s.db.QueryRow(ctx,
		`SELECT `+positionSelectCols+` FROM position WHERE id = $1 AND tenant_id = $2`,
		id, tenantID(scope))
	p, err := scanPosition(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("postgres: load position %d: %w", id, storage.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: load position %d: %w", id, err)
	}
	return &p, nil
}

// Exists reports whether a position with the given Zobrist hash is stored for
// the scope's tenant, returning its id when found.
func (s *positionStore) Exists(ctx context.Context, scope string, zobrist uint64) (int64, bool, error) {
	var id int64
	err := s.db.QueryRow(ctx,
		`SELECT id FROM position WHERE tenant_id = $1 AND zobrist_hash = $2`,
		tenantID(scope), int64(zobrist)).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("postgres: position exists: %w", err)
	}
	return id, true, nil
}

// Delete removes the position with the given id; analysis, comments and
// collection links cascade via foreign keys.
func (s *positionStore) Delete(ctx context.Context, scope string, id int64) error {
	if _, err := s.db.Exec(ctx,
		`DELETE FROM position WHERE id = $1 AND tenant_id = $2`,
		id, tenantID(scope)); err != nil {
		return fmt.Errorf("postgres: delete position %d: %w", id, err)
	}
	return nil
}

// List streams stored positions ordered by id.
func (s *positionStore) List(ctx context.Context, scope string, opts storage.ListOpts) iter.Seq2[*domain.Position, error] {
	return func(yield func(*domain.Position, error) bool) {
		query := `SELECT ` + positionSelectCols + ` FROM position WHERE tenant_id = $1 ORDER BY id`
		args := []any{tenantID(scope)}
		if opts.Limit > 0 {
			args = append(args, opts.Limit)
			query += fmt.Sprintf(" LIMIT $%d", len(args))
		}
		if opts.Offset > 0 {
			args = append(args, opts.Offset)
			query += fmt.Sprintf(" OFFSET $%d", len(args))
		}
		rows, err := s.db.Query(ctx, query, args...)
		if err != nil {
			yield(nil, fmt.Errorf("postgres: list positions: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			p, err := scanPosition(rows)
			if err != nil {
				yield(nil, fmt.Errorf("postgres: list positions: %w", err))
				return
			}
			if !yield(&p, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("postgres: list positions: %w", err))
		}
	}
}
