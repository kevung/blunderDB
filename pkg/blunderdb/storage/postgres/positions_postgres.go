package postgres

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// positionStore also holds the dialect-neutral view of the same connection
// (shared), for the handful of operations written once in sqlshared.
type positionStore struct {
	db     execer
	shared sqlshared.Execer
}

// ReclassifyDerived recomputes the derived phase of every position whose stored
// value disagrees with the classifier (ADR-0035). See sqlshared.ReclassifyDerived.
func (s *positionStore) ReclassifyDerived(ctx context.Context, scope string) (int, error) {
	return sqlshared.ReclassifyDerived(ctx, s.shared, scope)
}

var _ storage.PositionStore = (*positionStore)(nil)

// scanner is satisfied by both pgx.Row and pgx.Rows.
type scanner interface{ Scan(dest ...any) error }

// tenantID maps a scope string to the tenant_id column value through
// storage.ParseTenant: the empty scope is tenant 0 (the desktop's implicit
// tenant), any other scope must be a positive decimal integer — the numeric
// identifier the proxy in front of the daemon puts in X-Tenant-ID.
//
// A scope ParseTenant rejects is a programming error, not a data condition:
// every entry point (the HTTP tenant middleware, `migrate --tenant-id`,
// `call --scope`, PurgeTenant) validates the scope before it reaches a Store
// method, so this function panics rather than threading an error through the
// hundred call sites below. It used to map "alice" to 0 silently, which made
// every named tenant share one set of rows (ADR-0005, amendment 2026-09-03).
func tenantID(scope string) int64 {
	n, err := storage.ParseTenant(scope)
	if err != nil {
		panic("postgres: " + err.Error() + " (the caller must validate the scope with storage.ParseTenant)")
	}
	return n
}

// positionSelectCols is the column list read back into a Position; the first
// twelve match engine.ReconstructPosition's parameters, and
// individually_imported, flagged and max_cube are applied on top (provenance
// and session rules, not identity — see docs/adr/0001 and ADR-0028).
const positionSelectCols = `id, state, decision_type, player_on_roll, dice_1, dice_2, ` +
	`cube_value, cube_owner, score_1, score_2, has_jacoby, has_beaver, individually_imported, flagged, max_cube`

// qualifiedPositionCols is positionSelectCols with every column prefixed by a
// table alias, so a query that joins selects the same list rather than a second
// copy of it that would drift from the first.
func qualifiedPositionCols(alias string) string {
	return alias + "." + strings.ReplaceAll(positionSelectCols, ", ", ", "+alias+".")
}

// scanPosition reconstructs a Position from a row selected with
// positionSelectCols. The denormalised integer columns are nullable, so they
// scan into pointers; has_jacoby/has_beaver are BOOLEAN.
func scanPosition(sc scanner) (domain.Position, error) {
	var id int64
	var state string
	var dt, por, d1, d2, cv, co, s1, s2 *int64
	var hj, hb *bool
	var individual, flagged *bool
	var mc *int64
	if err := sc.Scan(&id, &state, &dt, &por, &d1, &d2, &cv, &co, &s1, &s2, &hj, &hb, &individual, &flagged, &mc); err != nil {
		return domain.Position{}, err
	}
	p := engine.ReconstructPosition(id, state,
		derefInt(dt), derefInt(por), derefInt(d1), derefInt(d2),
		derefInt(cv), derefInt(co), derefInt(s1), derefInt(s2),
		boolToIntPtr(hj), boolToIntPtr(hb))
	p.IndividuallyImported = individual != nil && *individual
	p.Flagged = flagged != nil && *flagged
	p.MaxCube = derefInt(mc)
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
	back_checkers_1, back_checkers_2, no_contact, game_phase, game_type,
	occupancy_1, occupancy_2, point_mask_1, point_mask_2,
	state, individually_imported, flagged, max_cube
) VALUES ($1,$2,$3,$4,$5,$6, $7,$8,$9,$10, $11,$12, $13,$14,$15,$16,$17, $18,$19,$20,$21,$22, $23,$24,$25,$26, $27,$28,$29,$30)
ON CONFLICT (tenant_id, zobrist_hash) DO NOTHING
RETURNING id`

// markIndividualSQL raises the provenance flag on an already-stored position.
// It only ever sets, never clears — that is what makes the flag sticky.
const markIndividualSQL = `UPDATE position SET individually_imported = TRUE
	WHERE tenant_id = $1 AND zobrist_hash = $2 AND NOT individually_imported`

// markFlaggedSQL raises the source-tool study mark on an already-stored
// position. Like markIndividualSQL it only ever sets, never clears — that is
// what makes the mark sticky across re-imports and across matches sharing the
// position (docs/adr/0006).
const markFlaggedSQL = `UPDATE position SET flagged = TRUE
	WHERE tenant_id = $1 AND zobrist_hash = $2 AND NOT flagged`

// Save stores p, deduplicated per tenant by Zobrist hash: a position whose
// (tenant_id, zobrist_hash) is already present is not re-inserted and Save
// returns the existing id. p is updated in place with the storage-normalised
// board and the resulting id.
//
// p.IndividuallyImported and p.Flagged are ORed into the stored value rather
// than assigned
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
		cols.HasJacoby != 0, cols.HasBeaver != 0, cols.MaxCube,
		cols.Pip1, cols.Pip2, cols.PipDiff, cols.Off1, cols.Off2,
		cols.BackCheckers1, cols.BackCheckers2, cols.NoContact, int(cols.GamePhase), int(cols.GameType),
		int64(cols.Occupancy1), int64(cols.Occupancy2), int64(cols.PointMask1), int64(cols.PointMask2),
		engine.EncodeBoardCompact(norm.Board), norm.IndividuallyImported, norm.Flagged,
		cols.MaxCube).Scan(&id)
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
		// Same for the source-tool mark: a match import that carries a flag
		// must raise it on the existing row. This is what lets a re-import of an
		// already-known match deliver newly added flags.
		if norm.Flagged {
			if _, err := s.db.Exec(ctx, markFlaggedSQL, tenant, int64(cols.ZobristHash)); err != nil {
				return 0, fmt.Errorf("postgres: mark position flagged: %w", err)
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
	has_jacoby=$11, has_beaver=$12, max_cube=$13,
	pip_1=$14, pip_2=$15, pip_diff=$16, off_1=$17, off_2=$18,
	back_checkers_1=$19, back_checkers_2=$20, no_contact=$21, game_phase=$22, game_type=$23,
	occupancy_1=$24, occupancy_2=$25, point_mask_1=$26, point_mask_2=$27
	WHERE id = $28 AND tenant_id = $29`

// Update overwrites the stored position with the same id as p.
func (s *positionStore) Update(ctx context.Context, scope string, p *domain.Position) error {
	cols := engine.PopulatePositionColumns(p)
	_, err := s.db.Exec(ctx, positionUpdateSQL,
		engine.EncodeBoardCompact(p.Board),
		int64(cols.ZobristHash), cols.DecisionType, p.PlayerOnRoll, cols.Dice1, cols.Dice2,
		cols.CubeValue, cols.CubeOwner, cols.Score1, cols.Score2,
		cols.HasJacoby != 0, cols.HasBeaver != 0, cols.MaxCube,
		cols.Pip1, cols.Pip2, cols.PipDiff, cols.Off1, cols.Off2,
		cols.BackCheckers1, cols.BackCheckers2, cols.NoContact, int(cols.GamePhase), int(cols.GameType),
		int64(cols.Occupancy1), int64(cols.Occupancy2), int64(cols.PointMask1), int64(cols.PointMask2),
		p.ID, tenantID(scope))
	if isUniqueViolation(err) {
		// The edit turned this position into one that is already stored:
		// the UNIQUE index on (tenant_id, zobrist_hash) refused it. Say which one.
		if id, found, lookupErr := s.Exists(ctx, scope, cols.ZobristHash); lookupErr == nil && found && id != p.ID {
			return fmt.Errorf("postgres: update position: %w", &storage.DuplicatePositionError{ExistingID: id})
		}
		return fmt.Errorf("postgres: update position: %w: %w", storage.ErrConflict, err)
	}
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

// ListIDs returns the tenant's stored position ids ordered by id.
func (s *positionStore) ListIDs(ctx context.Context, scope string, opts storage.ListOpts) ([]int64, error) {
	query := `SELECT id FROM position WHERE tenant_id = $1 ORDER BY id`
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
		return nil, fmt.Errorf("postgres: list position ids: %w", err)
	}
	defer rows.Close()
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("postgres: list position ids: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: list position ids: %w", err)
	}
	return ids, nil
}

// loadByIDsChunk bounds how many ids one `= ANY($2)` query carries.
// PostgreSQL has no bound-parameter limit forcing this the way SQLite's
// forEachIn chunking does, but an unbounded id list still lets one query
// hold a multi-million-element array (the server's own request-body cap
// admits roughly that many int64 ids) and scan proportionally, with no
// chance to notice a caller's cancelled context between rounds (#232).
const loadByIDsChunk = 1000

// LoadByIDs returns the listed positions in the caller's order, skipping
// unknown ids. ids is walked in loadByIDsChunk-sized rounds, each its own
// `= ANY($2)` query.
func (s *positionStore) LoadByIDs(ctx context.Context, scope string, ids []int64) ([]domain.Position, error) {
	if len(ids) == 0 {
		return []domain.Position{}, nil
	}
	byID := make(map[int64]domain.Position, len(ids))
	for start := 0; start < len(ids); start += loadByIDsChunk {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		batch := ids[start:min(start+loadByIDsChunk, len(ids))]
		if err := func() error {
			rows, err := s.db.Query(ctx,
				`SELECT `+positionSelectCols+` FROM position WHERE tenant_id = $1 AND id = ANY($2)`,
				tenantID(scope), batch)
			if err != nil {
				return err
			}
			defer rows.Close()
			for rows.Next() {
				p, err := scanPosition(rows)
				if err != nil {
					return err
				}
				byID[p.ID] = p
			}
			return rows.Err()
		}(); err != nil {
			return nil, fmt.Errorf("postgres: load positions by ids: %w", err)
		}
	}
	out := make([]domain.Position, 0, len(ids))
	for _, id := range ids {
		if p, ok := byID[id]; ok {
			out = append(out, p)
		}
	}
	return out, nil
}
