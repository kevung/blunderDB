package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"iter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type collectionStore struct{ db execer }

var _ storage.CollectionStore = (*collectionStore)(nil)

// collectionSelectCols reads a storage.Collection; the correlated subquery
// supplies the position count.
const collectionSelectCols = `c.id, c.name, COALESCE(c.description,''), COALESCE(c.sort_order,0),
	COALESCE(c.created_at,''), COALESCE(c.updated_at,''),
	(SELECT COUNT(*) FROM collection_position cp WHERE cp.collection_id = c.id)`

func scanCollection(sc interface{ Scan(...any) error }) (storage.Collection, error) {
	var c storage.Collection
	if err := sc.Scan(&c.ID, &c.Name, &c.Description, &c.SortOrder,
		&c.CreatedAt, &c.UpdatedAt, &c.PositionCount); err != nil {
		return storage.Collection{}, err
	}
	return c, nil
}

// Create stores a new collection at the end of the sort order and returns its
// id.
func (s *collectionStore) Create(ctx context.Context, scope string, name, description string) (int64, error) {
	var maxOrder int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(sort_order), -1) FROM collection`).Scan(&maxOrder); err != nil {
		maxOrder = -1
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO collection (name, description, sort_order) VALUES (?,?,?)`,
		name, description, maxOrder+1)
	if err != nil {
		return 0, fmt.Errorf("sqlite: create collection: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("sqlite: create collection id: %w", err)
	}
	return id, nil
}

// Get returns the collection with the given id, or ErrNotFound.
func (s *collectionStore) Get(ctx context.Context, scope string, id int64) (*storage.Collection, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+collectionSelectCols+` FROM collection c WHERE c.id = ?`, id)
	c, err := scanCollection(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("sqlite: get collection %d: %w", id, storage.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("sqlite: get collection %d: %w", id, err)
	}
	return &c, nil
}

// List streams every collection in sort order.
func (s *collectionStore) List(ctx context.Context, scope string) iter.Seq2[*storage.Collection, error] {
	return func(yield func(*storage.Collection, error) bool) {
		rows, err := s.db.QueryContext(ctx,
			`SELECT `+collectionSelectCols+` FROM collection c ORDER BY c.sort_order ASC`)
		if err != nil {
			yield(nil, fmt.Errorf("sqlite: list collections: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			c, err := scanCollection(rows)
			if err != nil {
				yield(nil, fmt.Errorf("sqlite: list collections: %w", err))
				return
			}
			if !yield(&c, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("sqlite: list collections: %w", err))
		}
	}
}

// Update changes a collection's name and description.
func (s *collectionStore) Update(ctx context.Context, scope string, id int64, name, description string) error {
	if _, err := s.db.ExecContext(ctx,
		`UPDATE collection SET name = ?, description = ?, updated_at = datetime('now')
		 WHERE id = ?`, name, description, id); err != nil {
		return fmt.Errorf("sqlite: update collection %d: %w", id, err)
	}
	return nil
}

// Delete removes a collection; its position memberships cascade off the
// collection_position foreign key.
func (s *collectionStore) Delete(ctx context.Context, scope string, id int64) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM collection WHERE id = ?`, id); err != nil {
		return fmt.Errorf("sqlite: delete collection %d: %w", id, err)
	}
	return nil
}

// Reorder assigns sort_order to collections in the order collectionIDs lists.
func (s *collectionStore) Reorder(ctx context.Context, scope string, collectionIDs []int64) error {
	err := withTx(ctx, s.db, func(tx execer) error {
		for i, id := range collectionIDs {
			if _, err := tx.ExecContext(ctx,
				`UPDATE collection SET sort_order = ? WHERE id = ?`, i, id); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("sqlite: reorder collections: %w", err)
	}
	return nil
}

// addPositionTx inserts one membership row at the end of the collection's
// order (a no-op when the position is already a member) and bumps the
// collection's updated_at. It runs on the caller-provided execer.
func addPositionTx(ctx context.Context, tx execer, collectionID, positionID int64) error {
	var maxOrder int
	if err := tx.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(sort_order), -1) FROM collection_position WHERE collection_id = ?`,
		collectionID).Scan(&maxOrder); err != nil {
		maxOrder = -1
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT OR IGNORE INTO collection_position (collection_id, position_id, sort_order, added_at)
		 VALUES (?,?,?,datetime('now'))`,
		collectionID, positionID, maxOrder+1); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE collection SET updated_at = datetime('now') WHERE id = ?`, collectionID); err != nil {
		return err
	}
	return nil
}

// AddPosition adds a position to a collection at the end of its order.
func (s *collectionStore) AddPosition(ctx context.Context, scope string, collectionID, positionID int64) error {
	err := withTx(ctx, s.db, func(tx execer) error {
		return addPositionTx(ctx, tx, collectionID, positionID)
	})
	if err != nil {
		return fmt.Errorf("sqlite: add position %d to collection %d: %w", positionID, collectionID, err)
	}
	return nil
}

// AddPositions adds several positions to a collection, appending them in the
// given order.
func (s *collectionStore) AddPositions(ctx context.Context, scope string, collectionID int64, positionIDs []int64) error {
	err := withTx(ctx, s.db, func(tx execer) error {
		var maxOrder int
		if err := tx.QueryRowContext(ctx,
			`SELECT COALESCE(MAX(sort_order), -1) FROM collection_position WHERE collection_id = ?`,
			collectionID).Scan(&maxOrder); err != nil {
			maxOrder = -1
		}
		for i, positionID := range positionIDs {
			if _, err := tx.ExecContext(ctx,
				`INSERT OR IGNORE INTO collection_position (collection_id, position_id, sort_order, added_at)
				 VALUES (?,?,?,datetime('now'))`,
				collectionID, positionID, maxOrder+1+i); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE collection SET updated_at = datetime('now') WHERE id = ?`, collectionID); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("sqlite: add positions to collection %d: %w", collectionID, err)
	}
	return nil
}

// RemovePosition removes a position from a collection.
func (s *collectionStore) RemovePosition(ctx context.Context, scope string, collectionID, positionID int64) error {
	err := withTx(ctx, s.db, func(tx execer) error {
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM collection_position WHERE collection_id = ? AND position_id = ?`,
			collectionID, positionID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE collection SET updated_at = datetime('now') WHERE id = ?`, collectionID); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("sqlite: remove position %d from collection %d: %w", positionID, collectionID, err)
	}
	return nil
}

// RemovePositions removes several positions from a collection.
func (s *collectionStore) RemovePositions(ctx context.Context, scope string, collectionID int64, positionIDs []int64) error {
	err := withTx(ctx, s.db, func(tx execer) error {
		for _, positionID := range positionIDs {
			if _, err := tx.ExecContext(ctx,
				`DELETE FROM collection_position WHERE collection_id = ? AND position_id = ?`,
				collectionID, positionID); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE collection SET updated_at = datetime('now') WHERE id = ?`, collectionID); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("sqlite: remove positions from collection %d: %w", collectionID, err)
	}
	return nil
}

// ReorderPositions assigns sort_order to a collection's positions in the order
// positionIDs lists them.
func (s *collectionStore) ReorderPositions(ctx context.Context, scope string, collectionID int64, positionIDs []int64) error {
	err := withTx(ctx, s.db, func(tx execer) error {
		for i, positionID := range positionIDs {
			if _, err := tx.ExecContext(ctx,
				`UPDATE collection_position SET sort_order = ?
				 WHERE collection_id = ? AND position_id = ?`,
				i, collectionID, positionID); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE collection SET updated_at = datetime('now') WHERE id = ?`, collectionID); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("sqlite: reorder positions of collection %d: %w", collectionID, err)
	}
	return nil
}

// MovePosition removes a position from one collection and appends it to
// another.
func (s *collectionStore) MovePosition(ctx context.Context, scope string, fromCollectionID, toCollectionID, positionID int64) error {
	err := withTx(ctx, s.db, func(tx execer) error {
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM collection_position WHERE collection_id = ? AND position_id = ?`,
			fromCollectionID, positionID); err != nil {
			return err
		}
		if err := addPositionTx(ctx, tx, toCollectionID, positionID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE collection SET updated_at = datetime('now') WHERE id = ?`, fromCollectionID); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("sqlite: move position %d to collection %d: %w", positionID, toCollectionID, err)
	}
	return nil
}

// CopyPosition adds a position to a collection without removing it elsewhere.
func (s *collectionStore) CopyPosition(ctx context.Context, scope string, toCollectionID, positionID int64) error {
	return s.AddPosition(ctx, scope, toCollectionID, positionID)
}

// collectionPositionCols is the position column list, prefixed for the join
// with collection_position; the order matches scanPosition.
const collectionPositionCols = `p.id, p.state, p.decision_type, p.player_on_roll, p.dice_1, p.dice_2, ` +
	`p.cube_value, p.cube_owner, p.score_1, p.score_2, p.has_jacoby, p.has_beaver, p.individually_imported`

// Positions streams the positions of a collection in their collection order.
func (s *collectionStore) Positions(ctx context.Context, scope string, collectionID int64) iter.Seq2[*domain.Position, error] {
	return func(yield func(*domain.Position, error) bool) {
		rows, err := s.db.QueryContext(ctx,
			`SELECT `+collectionPositionCols+` FROM position p
			 INNER JOIN collection_position cp ON p.id = cp.position_id
			 WHERE cp.collection_id = ?
			 ORDER BY cp.sort_order ASC`, collectionID)
		if err != nil {
			yield(nil, fmt.Errorf("sqlite: list collection positions: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			p, err := scanPosition(rows)
			if err != nil {
				yield(nil, fmt.Errorf("sqlite: list collection positions: %w", err))
				return
			}
			if !yield(&p, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("sqlite: list collection positions: %w", err))
		}
	}
}

// CollectionsOf streams the collections a position belongs to, in sort order.
func (s *collectionStore) CollectionsOf(ctx context.Context, scope string, positionID int64) iter.Seq2[*storage.Collection, error] {
	return func(yield func(*storage.Collection, error) bool) {
		rows, err := s.db.QueryContext(ctx,
			`SELECT `+collectionSelectCols+` FROM collection c
			 INNER JOIN collection_position cp ON c.id = cp.collection_id
			 WHERE cp.position_id = ?
			 ORDER BY c.sort_order ASC`, positionID)
		if err != nil {
			yield(nil, fmt.Errorf("sqlite: list collections of position: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			c, err := scanCollection(rows)
			if err != nil {
				yield(nil, fmt.Errorf("sqlite: list collections of position: %w", err))
				return
			}
			if !yield(&c, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("sqlite: list collections of position: %w", err))
		}
	}
}

// PositionIndexMap returns, for every stored position id, its 1-based display
// index (positions ordered by id).
func (s *collectionStore) PositionIndexMap(ctx context.Context, scope string) (map[int64]int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM position ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("sqlite: position index map: %w", err)
	}
	defer rows.Close()
	out := make(map[int64]int)
	index := 1
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("sqlite: position index map: %w", err)
		}
		out[id] = index
		index++
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: position index map: %w", err)
	}
	return out, nil
}
