package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"iter"
	"strings"

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

// RepairCrawfordSentinel rewrites the away score of the positions a
// post-Crawford game was imported with the Crawford sentinel on, rehashing
// them. See sqlshared.RepairCrawfordSentinel — the store goes in because the
// repair rehashes through the same Load/Exists/Update this backend serves.
func (s *positionStore) RepairCrawfordSentinel(ctx context.Context, scope string) (int, error) {
	return sqlshared.RepairCrawfordSentinel(ctx, s.shared, scope, s)
}

var _ storage.PositionStore = (*positionStore)(nil)

// positionCols is the column list read back into a Position; the first twelve
// match engine.ReconstructPosition's parameters, and individually_imported,
// flagged and max_cube are applied on top (provenance and session rules, not
// identity — see ADR-0001 and ADR-0028).
const positionCols = `id, state, decision_type, player_on_roll, dice_1, dice_2, ` +
	`cube_value, cube_owner, score_1, score_2, has_jacoby, has_beaver, individually_imported, flagged, max_cube`

// scanPosition reconstructs a Position from a row selected with positionCols.
func scanPosition(sc interface{ Scan(...any) error }) (domain.Position, error) {
	var id int64
	var state string
	var dt, por, d1, d2, cv, co, s1, s2, hj, hb, mc sql.NullInt64
	var individual, flagged sql.NullBool
	if err := sc.Scan(&id, &state, &dt, &por, &d1, &d2, &cv, &co, &s1, &s2, &hj, &hb, &individual, &flagged, &mc); err != nil {
		return domain.Position{}, err
	}
	p := engine.ReconstructPosition(id, state,
		int(dt.Int64), int(por.Int64), int(d1.Int64), int(d2.Int64),
		int(cv.Int64), int(co.Int64), int(s1.Int64), int(s2.Int64),
		int(hj.Int64), int(hb.Int64))
	p.IndividuallyImported = individual.Bool
	p.Flagged = flagged.Bool
	p.MaxCube = int(mc.Int64)
	return p, nil
}

const positionInsertSQL = `INSERT INTO position (
	zobrist_hash, decision_type, player_on_roll, dice_1, dice_2,
	cube_value, cube_owner, score_1, score_2,
	has_jacoby, has_beaver,
	pip_1, pip_2, pip_diff, off_1, off_2,
	back_checkers_1, back_checkers_2, no_contact, game_phase, game_type,
	occupancy_1, occupancy_2, point_mask_1, point_mask_2,
	state, individually_imported, flagged, max_cube
) VALUES (?,?,?,?,?, ?,?,?,?, ?,?, ?,?,?,?,?, ?,?,?,?,?, ?,?,?,?, ?,?,?,?)
ON CONFLICT(zobrist_hash) DO NOTHING`

// positionIDByHashSQL finds the row a deduplicated Save landed on.
const positionIDByHashSQL = `SELECT id FROM position WHERE zobrist_hash = ?`

// markIndividualSQL raises the provenance flag on an already-stored position.
// It only ever sets, never clears — that is what makes the flag sticky.
const markIndividualSQL = `UPDATE position SET individually_imported = 1
	WHERE zobrist_hash = ? AND individually_imported = 0`

// markFlaggedSQL raises the source-tool study mark on an already-stored
// position. Like markIndividualSQL it only ever sets, never clears — that is
// what makes the mark sticky across re-imports and across matches sharing the
// position (docs/adr/0006).
const markFlaggedSQL = `UPDATE position SET flagged = 1
	WHERE zobrist_hash = ? AND flagged = 0`

// Save stores p, deduplicated by Zobrist hash: a position whose hash is already
// present is not re-inserted and Save returns the existing id. p is updated in
// place with the storage-normalised board and the resulting id.
//
// p.IndividuallyImported and p.Flagged are ORed into the stored value rather
// than assigned (ADR-0001), so they do not depend on import order.
//
// A writer can still lose the write lock once busy_timeout is exhausted
// (more often on Windows), so Save retries on SQLITE_BUSY via retryOnBusy.
// That is safe: every statement saveOnce runs is idempotent, and *p is only
// mutated once saveOnce has fully succeeded.
func (s *positionStore) Save(ctx context.Context, scope string, p *domain.Position) (int64, error) {
	id, _, err := s.SaveCreated(ctx, scope, p)
	return id, err
}

// SaveCreated is Save, reporting whether the INSERT added the row. A retry
// after SQLITE_BUSY cannot misreport: the INSERT is a single autocommit
// statement, so an attempt that inserted has nothing left that can be busy.
func (s *positionStore) SaveCreated(ctx context.Context, scope string, p *domain.Position) (int64, bool, error) {
	var (
		id      int64
		created bool
	)
	err := retryOnBusy(func() error {
		var innerErr error
		id, created, innerErr = s.saveOnce(ctx, scope, p)
		return innerErr
	})
	return id, created, err
}

// saveOnce is SaveCreated's single attempt, with no retry of its own.
func (s *positionStore) saveOnce(ctx context.Context, scope string, p *domain.Position) (int64, bool, error) {
	norm := p.NormalizeForStorage()
	cols := engine.PopulatePositionColumns(p)
	res, err := s.db.ExecContext(ctx, positionInsertSQL,
		int64(cols.ZobristHash), cols.DecisionType, norm.PlayerOnRoll, cols.Dice1, cols.Dice2,
		cols.CubeValue, cols.CubeOwner, cols.Score1, cols.Score2,
		cols.HasJacoby, cols.HasBeaver,
		cols.Pip1, cols.Pip2, cols.PipDiff, cols.Off1, cols.Off2,
		cols.BackCheckers1, cols.BackCheckers2, boolToInt(cols.NoContact), int(cols.GamePhase), int(cols.GameType),
		int64(cols.Occupancy1), int64(cols.Occupancy2), int64(cols.PointMask1), int64(cols.PointMask2),
		engine.EncodeBoardState(norm.Board), boolToInt(norm.IndividuallyImported), boolToInt(norm.Flagged),
		cols.MaxCube)
	if err != nil {
		return 0, false, fmt.Errorf("sqlite: save position: %w", err)
	}
	var id int64
	affected, _ := res.RowsAffected()
	created := affected > 0
	if created {
		if id, err = res.LastInsertId(); err != nil {
			return 0, false, fmt.Errorf("sqlite: save position id: %w", err)
		}
	} else {
		// Hash already present: keep the existing row, but let an individual
		// import raise the flag on it. Skipped entirely for match imports, so
		// they stay a pure no-op on a duplicate position.
		if norm.IndividuallyImported {
			if _, err := s.db.ExecContext(ctx, markIndividualSQL, int64(cols.ZobristHash)); err != nil {
				return 0, false, fmt.Errorf("sqlite: mark position individually imported: %w", err)
			}
		}
		// Same for the source-tool study mark: a match import that carries a
		// flag must raise it on the existing row. This is what lets a re-import
		// of an already-known match deliver newly added marks.
		if norm.Flagged {
			if _, err := s.db.ExecContext(ctx, markFlaggedSQL, int64(cols.ZobristHash)); err != nil {
				return 0, false, fmt.Errorf("sqlite: mark position flagged: %w", err)
			}
		}
		if err = s.db.QueryRowContext(ctx, positionIDByHashSQL,
			int64(cols.ZobristHash)).Scan(&id); err != nil {
			return 0, false, fmt.Errorf("sqlite: save position dedup lookup: %w", err)
		}
	}
	norm.ID = id
	*p = norm
	return id, created, nil
}

// RaiseFlag — see storage.PositionStore.
func (s *positionStore) RaiseFlag(ctx context.Context, scope string, p *domain.Position) (bool, error) {
	res, err := s.db.ExecContext(ctx, markFlaggedSQL, int64(engine.PopulatePositionColumns(p).ZobristHash))
	if err != nil {
		return false, fmt.Errorf("sqlite: raise position flag: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("sqlite: raise position flag: %w", err)
	}
	return n > 0, nil
}

const positionUpdateSQL = `UPDATE position SET state = ?,
	zobrist_hash=?, decision_type=?, player_on_roll=?, dice_1=?, dice_2=?,
	cube_value=?, cube_owner=?, score_1=?, score_2=?,
	has_jacoby=?, has_beaver=?, max_cube=?,
	pip_1=?, pip_2=?, pip_diff=?, off_1=?, off_2=?,
	back_checkers_1=?, back_checkers_2=?, no_contact=?, game_phase=?, game_type=?,
	occupancy_1=?, occupancy_2=?, point_mask_1=?, point_mask_2=?
	WHERE id = ?`

// Update overwrites the stored position with the same id as p.
func (s *positionStore) Update(ctx context.Context, scope string, p *domain.Position) error {
	cols := engine.PopulatePositionColumns(p)
	_, err := s.db.ExecContext(ctx, positionUpdateSQL,
		engine.EncodeBoardState(p.Board),
		int64(cols.ZobristHash), cols.DecisionType, p.PlayerOnRoll, cols.Dice1, cols.Dice2,
		cols.CubeValue, cols.CubeOwner, cols.Score1, cols.Score2,
		cols.HasJacoby, cols.HasBeaver, cols.MaxCube,
		cols.Pip1, cols.Pip2, cols.PipDiff, cols.Off1, cols.Off2,
		cols.BackCheckers1, cols.BackCheckers2, boolToInt(cols.NoContact), int(cols.GamePhase), int(cols.GameType),
		int64(cols.Occupancy1), int64(cols.Occupancy2), int64(cols.PointMask1), int64(cols.PointMask2),
		p.ID)
	if isUniqueViolation(err) {
		// The edit turned this position into one that is already stored:
		// the UNIQUE index on zobrist_hash refused it. Say which one.
		if id, found, lookupErr := s.Exists(ctx, scope, cols.ZobristHash); lookupErr == nil && found && id != p.ID {
			return fmt.Errorf("sqlite: update position: %w", &storage.DuplicatePositionError{ExistingID: id})
		}
		return fmt.Errorf("sqlite: update position: %w: %w", storage.ErrConflict, err)
	}
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

// existsBatch keeps each IN list under SQLite's historical 999-parameter
// ceiling.
const existsBatch = 900

// ExistsMany reports the ids of the stored positions among zobrists.
func (s *positionStore) ExistsMany(ctx context.Context, scope string, zobrists []uint64) (map[uint64]int64, error) {
	out := make(map[uint64]int64, len(zobrists))
	for start := 0; start < len(zobrists); start += existsBatch {
		if err := s.existsChunk(ctx, zobrists[start:min(start+existsBatch, len(zobrists))], out); err != nil {
			return nil, fmt.Errorf("sqlite: positions exist: %w", err)
		}
	}
	return out, nil
}

func (s *positionStore) existsChunk(ctx context.Context, chunk []uint64, out map[uint64]int64) error {
	args := make([]any, len(chunk))
	for i, z := range chunk {
		args[i] = int64(z)
	}
	q := `SELECT zobrist_hash, id FROM position WHERE zobrist_hash IN (?` + strings.Repeat(",?", len(chunk)-1) + `)`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var z, id int64
		if err := rows.Scan(&z, &id); err != nil {
			return err
		}
		out[uint64(z)] = id
	}
	return rows.Err()
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

// ListIDs returns the stored position ids ordered by id.
func (s *positionStore) ListIDs(ctx context.Context, scope string, opts storage.ListOpts) ([]int64, error) {
	if opts.Limit > 0 && opts.Offset > 0 {
		if ids, ok, err := s.listIDsFromEnd(ctx, opts); ok || err != nil {
			return ids, err
		}
	}
	query := `SELECT id FROM position ORDER BY id`
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
		return nil, fmt.Errorf("sqlite: list position ids: %w", err)
	}
	defer rows.Close()
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("sqlite: list position ids: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: list position ids: %w", err)
	}
	return ids, nil
}

// listIDsFromEnd answers a window in the second half of the list by walking
// from the last id: OFFSET skips rows one by one, and the GUI opens a library
// on its last page, where an OFFSET from the start walks the whole table. ok
// is false when the window lies in the first half, which the plain query
// reaches sooner. The window is one statement, so it sees one snapshot even
// if a write lands after the count that chose the path.
func (s *positionStore) listIDsFromEnd(ctx context.Context, opts storage.ListOpts) ([]int64, bool, error) {
	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM position`).Scan(&total); err != nil {
		return nil, false, fmt.Errorf("sqlite: list position ids: %w", err)
	}
	if opts.Offset <= total/2 {
		return nil, false, nil
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM (
		SELECT id FROM position ORDER BY id DESC
		LIMIT max(0, min(?1, (SELECT COUNT(*) FROM position) - ?2))
		OFFSET max(0, (SELECT COUNT(*) FROM position) - ?2 - ?1)
	) ORDER BY id`, opts.Limit, opts.Offset)
	if err != nil {
		return nil, false, fmt.Errorf("sqlite: list position ids: %w", err)
	}
	defer rows.Close()
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, false, fmt.Errorf("sqlite: list position ids: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("sqlite: list position ids: %w", err)
	}
	return ids, true, nil
}

// Count returns the number of stored positions.
func (s *positionStore) Count(ctx context.Context, scope string) (int, error) {
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM position`).Scan(&n); err != nil {
		return 0, fmt.Errorf("sqlite: count positions: %w", err)
	}
	return n, nil
}

// IndexOf returns the rank of id in ListIDs's order. The order is by id, so
// the rank is the number of smaller ids: a primary-key range count, not a
// walk of the list.
func (s *positionStore) IndexOf(ctx context.Context, scope string, id int64) (int, bool, error) {
	var exists, rank int
	err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM position WHERE id = ?), (SELECT COUNT(*) FROM position WHERE id < ?)`,
		id, id).Scan(&exists, &rank)
	if err != nil {
		return 0, false, fmt.Errorf("sqlite: index of position: %w", err)
	}
	if exists == 0 {
		return 0, false, nil
	}
	return rank, true, nil
}

// LoadByIDs returns the listed positions in the caller's order, skipping
// unknown ids — see storage.PositionStore. The lookup runs in chunks via
// forEachIn, staying under SQLite's bound-variable limit.
func (s *positionStore) LoadByIDs(ctx context.Context, scope string, ids []int64) ([]domain.Position, error) {
	byID := make(map[int64]domain.Position, len(ids))
	err := forEachIn(ctx, s.db, ids, `SELECT `+positionCols+` FROM position WHERE id IN `, ``, func(rows *sql.Rows) error {
		p, err := scanPosition(rows)
		if err != nil {
			return err
		}
		byID[p.ID] = p
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("sqlite: load positions by ids: %w", err)
	}
	out := make([]domain.Position, 0, len(byID))
	for _, id := range ids {
		if p, ok := byID[id]; ok {
			out = append(out, p)
		}
	}
	return out, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
