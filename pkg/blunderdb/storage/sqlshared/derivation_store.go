package sqlshared

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// DerivationStore implements storage.DerivationStore over position_derivation.
type DerivationStore struct{ DB Execer }

var _ storage.DerivationStore = (*DerivationStore)(nil)

func (s *DerivationStore) DerivedFrom(ctx context.Context, scope string, positionID int64) (*domain.Derivation, error) {
	if err := requirePositionIn(ctx, s.DB, scope, positionID, "derivation"); err != nil {
		return nil, err
	}
	m, err := s.Derivations(ctx, scope, []int64{positionID})
	if d, ok := m[positionID]; ok {
		return &d, err
	}
	return nil, err
}

func (s *DerivationStore) SetDerivedFrom(ctx context.Context, scope string, positionID int64, d *domain.Derivation) error {
	if d != nil {
		if err := d.Validate(positionID); err != nil {
			return fmt.Errorf("%w: %w", storage.ErrInvalid, err)
		}
	}
	if err := requirePositionIn(ctx, s.DB, scope, positionID, "derivation"); err != nil {
		return err
	}
	return s.DB.Transact(ctx, func(tx Execer) error {
		// The links of a tenant are written one transaction at a time: two
		// crossing writes, each fine against the other's past, would close a
		// loop. The guard (PostgreSQL) is taken before the check; on SQLite
		// the delete below is the first write and takes the write lock.
		if q, gargs := tx.Guard(storage.DerivationGuardKey(scope)); q != "" {
			if _, err := tx.Exec(ctx, q, gargs...); err != nil {
				return errf(tx, "guard derivation", err)
			}
		}
		tenant, targs := tx.TenantFilter("", scope)
		if _, err := tx.Exec(ctx, `DELETE FROM position_derivation WHERE position_id = ? AND `+tenant,
			append([]any{positionID}, targs...)...); err != nil {
			return errf(tx, "clear derivation", err)
		}
		if d == nil {
			return nil
		}
		// An origin or a loop that is refused rolls the delete back with the
		// rest of the transaction: the previous link stays.
		if err := requirePositionIn(ctx, tx, scope, d.FromID, "derivation"); err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return fmt.Errorf("%w: derivation origin %d is not a position of the scope", storage.ErrInvalid, d.FromID)
			}
			return err
		}
		if err := (&DerivationStore{DB: tx}).refuseLoop(ctx, scope, positionID, d.FromID); err != nil {
			return err
		}
		cols, args := tx.TenantColumns(scope)
		cols = append(cols, "position_id", "from_position_id", "move")
		args = append(args, positionID, d.FromID, d.Move)
		if _, err := tx.Exec(ctx, `INSERT INTO position_derivation (`+strings.Join(cols, ", ")+`) VALUES (`+
			Placeholders(len(cols))+`)`, args...); err != nil {
			return errf(tx, "derive position", err)
		}
		return nil
	})
}

// refuseLoop is ErrInvalid when walking up from the origin reaches the
// position itself: it would then derive from its own variant.
func (s *DerivationStore) refuseLoop(ctx context.Context, scope string, positionID, originID int64) error {
	seen := map[int64]bool{}
	for cur := originID; ; {
		if cur == positionID || seen[cur] {
			return fmt.Errorf("%w: position %d cannot derive from %d, which derives from it", storage.ErrInvalid, positionID, originID)
		}
		seen[cur] = true
		m, err := s.Derivations(ctx, scope, []int64{cur})
		if err != nil {
			return err
		}
		next, ok := m[cur]
		if !ok {
			return nil
		}
		cur = next.FromID
	}
}

func (s *DerivationStore) Derivations(ctx context.Context, scope string, positionIDs []int64) (map[int64]domain.Derivation, error) {
	out := make(map[int64]domain.Derivation)
	tenant, targs := s.DB.TenantFilter("", scope)
	for start := 0; start < len(positionIDs); start += byPositionsChunk {
		batch := positionIDs[start:min(start+byPositionsChunk, len(positionIDs))]
		args := make([]any, 0, len(batch)+len(targs))
		for _, id := range batch {
			args = append(args, id)
		}
		args = append(args, targs...)
		rows, err := s.DB.Query(ctx, `SELECT position_id, from_position_id, move FROM position_derivation
			WHERE position_id IN (`+Placeholders(len(batch))+`) AND `+tenant, args...)
		if err != nil {
			return nil, errf(s.DB, "derivations", err)
		}
		for rows.Next() {
			var id int64
			var d domain.Derivation
			if err := rows.Scan(&id, &d.FromID, &d.Move); err != nil {
				rows.Close()
				return nil, errf(s.DB, "derivations", err)
			}
			out[id] = d
		}
		rerr := rows.Err()
		rows.Close()
		if rerr != nil {
			return nil, errf(s.DB, "derivations", rerr)
		}
	}
	return out, nil
}

func (s *DerivationStore) Variants(ctx context.Context, scope string, originID int64) ([]int64, error) {
	if err := requirePositionIn(ctx, s.DB, scope, originID, "derivation"); err != nil {
		return nil, err
	}
	tenant, targs := s.DB.TenantFilter("", scope)
	rows, err := s.DB.Query(ctx, `SELECT position_id FROM position_derivation WHERE from_position_id = ? AND `+tenant+
		` ORDER BY position_id`, append([]any{originID}, targs...)...)
	if err != nil {
		return nil, errf(s.DB, "variants", err)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, errf(s.DB, "variants", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, errf(s.DB, "variants", err)
	}
	return ids, nil
}

// requirePositionIn is ErrNotFound unless the position exists in the scope;
// a foreign key alone would say "constraint failed" on SQLite. what names the
// caller in the error.
func requirePositionIn(ctx context.Context, db Execer, scope string, id int64, what string) error {
	tenant, targs := db.TenantFilter("", scope)
	var one int
	err := db.QueryRow(ctx, `SELECT 1 FROM position WHERE `+tenant+` AND id = ?`, append(targs, id)...).Scan(&one)
	if errors.Is(err, ErrNoRows) {
		return errf(db, what, storage.ErrNotFound)
	}
	if err != nil {
		return errf(db, what, err)
	}
	return nil
}
