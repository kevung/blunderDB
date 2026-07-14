package postgres

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type collectionStore struct{ db execer }

var _ storage.CollectionStore = (*collectionStore)(nil)

// collectionSelectExpr reads a storage.Collection; the correlated subquery
// supplies the position count.
const collectionSelectExpr = `c.id, c.name, COALESCE(c.description,''), COALESCE(c.sort_order,0),
	c.created_at, c.updated_at,
	(SELECT COUNT(*) FROM collection_position cp WHERE cp.collection_id = c.id)`

func scanCollection(sc scanner) (storage.Collection, error) {
	var c storage.Collection
	var createdAt, updatedAt time.Time
	if err := sc.Scan(&c.ID, &c.Name, &c.Description, &c.SortOrder,
		&createdAt, &updatedAt, &c.PositionCount); err != nil {
		return storage.Collection{}, err
	}
	c.CreatedAt = tsTime(createdAt)
	c.UpdatedAt = tsTime(updatedAt)
	return c, nil
}

// Create stores a new collection at the end of the sort order and returns its
// id.
func (s *collectionStore) Create(ctx context.Context, scope string, name, description string) (int64, error) {
	tenant := tenantID(scope)
	var maxOrder int
	if err := s.db.QueryRow(ctx,
		`SELECT COALESCE(MAX(sort_order), -1) FROM collection WHERE tenant_id = $1`,
		tenant).Scan(&maxOrder); err != nil {
		maxOrder = -1
	}
	var id int64
	if err := s.db.QueryRow(ctx,
		`INSERT INTO collection (tenant_id, name, description, sort_order)
		 VALUES ($1,$2,$3,$4) RETURNING id`,
		tenant, name, description, maxOrder+1).Scan(&id); err != nil {
		return 0, fmt.Errorf("postgres: create collection: %w", err)
	}
	return id, nil
}

// Get returns the collection with the given id, or ErrNotFound.
func (s *collectionStore) Get(ctx context.Context, scope string, id int64) (*storage.Collection, error) {
	row := s.db.QueryRow(ctx,
		`SELECT `+collectionSelectExpr+` FROM collection c WHERE c.id = $1 AND c.tenant_id = $2`,
		id, tenantID(scope))
	c, err := scanCollection(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("postgres: get collection %d: %w", id, storage.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: get collection %d: %w", id, err)
	}
	return &c, nil
}

// List streams every collection in sort order.
func (s *collectionStore) List(ctx context.Context, scope string) iter.Seq2[*storage.Collection, error] {
	return func(yield func(*storage.Collection, error) bool) {
		rows, err := s.db.Query(ctx,
			`SELECT `+collectionSelectExpr+` FROM collection c
			 WHERE c.tenant_id = $1 ORDER BY c.sort_order ASC`, tenantID(scope))
		if err != nil {
			yield(nil, fmt.Errorf("postgres: list collections: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			c, err := scanCollection(rows)
			if err != nil {
				yield(nil, fmt.Errorf("postgres: list collections: %w", err))
				return
			}
			if !yield(&c, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("postgres: list collections: %w", err))
		}
	}
}

// Update changes a collection's name and description.
func (s *collectionStore) Update(ctx context.Context, scope string, id int64, name, description string) error {
	if _, err := s.db.Exec(ctx,
		`UPDATE collection SET name = $1, description = $2, updated_at = now()
		 WHERE id = $3 AND tenant_id = $4`,
		name, description, id, tenantID(scope)); err != nil {
		return fmt.Errorf("postgres: update collection %d: %w", id, err)
	}
	return nil
}

// Delete removes a collection; its position memberships cascade off the
// collection_position foreign key.
func (s *collectionStore) Delete(ctx context.Context, scope string, id int64) error {
	if _, err := s.db.Exec(ctx,
		`DELETE FROM collection WHERE id = $1 AND tenant_id = $2`,
		id, tenantID(scope)); err != nil {
		return fmt.Errorf("postgres: delete collection %d: %w", id, err)
	}
	return nil
}

// Reorder assigns sort_order to collections in the order collectionIDs lists.
func (s *collectionStore) Reorder(ctx context.Context, scope string, collectionIDs []int64) error {
	tenant := tenantID(scope)
	err := withTx(ctx, s.db, func(tx execer) error {
		for i, id := range collectionIDs {
			if _, err := tx.Exec(ctx,
				`UPDATE collection SET sort_order = $1 WHERE id = $2 AND tenant_id = $3`,
				i, id, tenant); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("postgres: reorder collections: %w", err)
	}
	return nil
}

// addPositionTx inserts one membership row at the end of the collection's
// order (a no-op when the position is already a member) and bumps the
// collection's updated_at. It runs on the caller-provided execer.
func addPositionTx(ctx context.Context, tx execer, tenant, collectionID, positionID int64) error {
	var maxOrder int
	if err := tx.QueryRow(ctx,
		`SELECT COALESCE(MAX(sort_order), -1) FROM collection_position WHERE collection_id = $1`,
		collectionID).Scan(&maxOrder); err != nil {
		maxOrder = -1
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO collection_position (tenant_id, collection_id, position_id, sort_order)
		 VALUES ($1,$2,$3,$4) ON CONFLICT (collection_id, position_id) DO NOTHING`,
		tenant, collectionID, positionID, maxOrder+1); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE collection SET updated_at = now() WHERE id = $1 AND tenant_id = $2`,
		collectionID, tenant); err != nil {
		return err
	}
	return nil
}

// AddPosition adds a position to a collection at the end of its order.
func (s *collectionStore) AddPosition(ctx context.Context, scope string, collectionID, positionID int64) error {
	tenant := tenantID(scope)
	err := withTx(ctx, s.db, func(tx execer) error {
		return addPositionTx(ctx, tx, tenant, collectionID, positionID)
	})
	if err != nil {
		return fmt.Errorf("postgres: add position %d to collection %d: %w", positionID, collectionID, err)
	}
	return nil
}

// AddPositions adds several positions to a collection, appending them in the
// given order.
func (s *collectionStore) AddPositions(ctx context.Context, scope string, collectionID int64, positionIDs []int64) error {
	tenant := tenantID(scope)
	err := withTx(ctx, s.db, func(tx execer) error {
		var maxOrder int
		if err := tx.QueryRow(ctx,
			`SELECT COALESCE(MAX(sort_order), -1) FROM collection_position WHERE collection_id = $1`,
			collectionID).Scan(&maxOrder); err != nil {
			maxOrder = -1
		}
		for i, positionID := range positionIDs {
			if _, err := tx.Exec(ctx,
				`INSERT INTO collection_position (tenant_id, collection_id, position_id, sort_order)
				 VALUES ($1,$2,$3,$4) ON CONFLICT (collection_id, position_id) DO NOTHING`,
				tenant, collectionID, positionID, maxOrder+1+i); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(ctx,
			`UPDATE collection SET updated_at = now() WHERE id = $1 AND tenant_id = $2`,
			collectionID, tenant); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("postgres: add positions to collection %d: %w", collectionID, err)
	}
	return nil
}

// RemovePosition removes a position from a collection.
func (s *collectionStore) RemovePosition(ctx context.Context, scope string, collectionID, positionID int64) error {
	tenant := tenantID(scope)
	err := withTx(ctx, s.db, func(tx execer) error {
		if _, err := tx.Exec(ctx,
			`DELETE FROM collection_position WHERE collection_id = $1 AND position_id = $2`,
			collectionID, positionID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx,
			`UPDATE collection SET updated_at = now() WHERE id = $1 AND tenant_id = $2`,
			collectionID, tenant); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("postgres: remove position %d from collection %d: %w", positionID, collectionID, err)
	}
	return nil
}

// RemovePositions removes several positions from a collection.
func (s *collectionStore) RemovePositions(ctx context.Context, scope string, collectionID int64, positionIDs []int64) error {
	tenant := tenantID(scope)
	err := withTx(ctx, s.db, func(tx execer) error {
		for _, positionID := range positionIDs {
			if _, err := tx.Exec(ctx,
				`DELETE FROM collection_position WHERE collection_id = $1 AND position_id = $2`,
				collectionID, positionID); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(ctx,
			`UPDATE collection SET updated_at = now() WHERE id = $1 AND tenant_id = $2`,
			collectionID, tenant); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("postgres: remove positions from collection %d: %w", collectionID, err)
	}
	return nil
}

// ReorderPositions assigns sort_order to a collection's positions in the order
// positionIDs lists them.
func (s *collectionStore) ReorderPositions(ctx context.Context, scope string, collectionID int64, positionIDs []int64) error {
	tenant := tenantID(scope)
	err := withTx(ctx, s.db, func(tx execer) error {
		for i, positionID := range positionIDs {
			if _, err := tx.Exec(ctx,
				`UPDATE collection_position SET sort_order = $1
				 WHERE collection_id = $2 AND position_id = $3`,
				i, collectionID, positionID); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(ctx,
			`UPDATE collection SET updated_at = now() WHERE id = $1 AND tenant_id = $2`,
			collectionID, tenant); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("postgres: reorder positions of collection %d: %w", collectionID, err)
	}
	return nil
}

// MovePosition removes a position from one collection and appends it to
// another.
func (s *collectionStore) MovePosition(ctx context.Context, scope string, fromCollectionID, toCollectionID, positionID int64) error {
	tenant := tenantID(scope)
	err := withTx(ctx, s.db, func(tx execer) error {
		if _, err := tx.Exec(ctx,
			`DELETE FROM collection_position WHERE collection_id = $1 AND position_id = $2`,
			fromCollectionID, positionID); err != nil {
			return err
		}
		if err := addPositionTx(ctx, tx, tenant, toCollectionID, positionID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx,
			`UPDATE collection SET updated_at = now() WHERE id = $1 AND tenant_id = $2`,
			fromCollectionID, tenant); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("postgres: move position %d to collection %d: %w", positionID, toCollectionID, err)
	}
	return nil
}

// CopyPosition adds a position to a collection without removing it elsewhere.
func (s *collectionStore) CopyPosition(ctx context.Context, scope string, toCollectionID, positionID int64) error {
	return s.AddPosition(ctx, scope, toCollectionID, positionID)
}

// collectionPositionCols is the position column list, prefixed for the join
// with collection_position; the order matches engine.ReconstructPosition.
const collectionPositionCols = `p.id, p.state, p.decision_type, p.player_on_roll, p.dice_1, p.dice_2, ` +
	`p.cube_value, p.cube_owner, p.score_1, p.score_2, p.has_jacoby, p.has_beaver, p.individually_imported`

// Positions streams the positions of a collection in their collection order.
func (s *collectionStore) Positions(ctx context.Context, scope string, collectionID int64) iter.Seq2[*domain.Position, error] {
	return func(yield func(*domain.Position, error) bool) {
		rows, err := s.db.Query(ctx,
			`SELECT `+collectionPositionCols+` FROM position p
			 INNER JOIN collection_position cp ON p.id = cp.position_id
			 WHERE cp.collection_id = $1 AND cp.tenant_id = $2
			 ORDER BY cp.sort_order ASC`,
			collectionID, tenantID(scope))
		if err != nil {
			yield(nil, fmt.Errorf("postgres: list collection positions: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			p, err := scanPosition(rows)
			if err != nil {
				yield(nil, fmt.Errorf("postgres: list collection positions: %w", err))
				return
			}
			if !yield(&p, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("postgres: list collection positions: %w", err))
		}
	}
}

// CollectionsOf streams the collections a position belongs to, in sort order.
func (s *collectionStore) CollectionsOf(ctx context.Context, scope string, positionID int64) iter.Seq2[*storage.Collection, error] {
	return func(yield func(*storage.Collection, error) bool) {
		rows, err := s.db.Query(ctx,
			`SELECT `+collectionSelectExpr+` FROM collection c
			 INNER JOIN collection_position cp ON c.id = cp.collection_id
			 WHERE cp.position_id = $1 AND c.tenant_id = $2
			 ORDER BY c.sort_order ASC`,
			positionID, tenantID(scope))
		if err != nil {
			yield(nil, fmt.Errorf("postgres: list collections of position: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			c, err := scanCollection(rows)
			if err != nil {
				yield(nil, fmt.Errorf("postgres: list collections of position: %w", err))
				return
			}
			if !yield(&c, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("postgres: list collections of position: %w", err))
		}
	}
}

// PositionIndexMap returns, for every stored position id, its 1-based display
// index (positions ordered by id).
func (s *collectionStore) PositionIndexMap(ctx context.Context, scope string) (map[int64]int, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id FROM position WHERE tenant_id = $1 ORDER BY id ASC`, tenantID(scope))
	if err != nil {
		return nil, fmt.Errorf("postgres: position index map: %w", err)
	}
	defer rows.Close()
	out := make(map[int64]int)
	index := 1
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("postgres: position index map: %w", err)
		}
		out[id] = index
		index++
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: position index map: %w", err)
	}
	return out, nil
}
