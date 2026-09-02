package database

import (
	"context"
	"fmt"
	"iter"
	"log/slog"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The collection family is an adapter over storage.CollectionStore: the SQL
// lives once, in storage/sqlite, under the contract suite both backends pass
// (storagetest, Collection/*). Every method takes d.mu the way it always did
// and passes the desktop's implicit tenant ("") as scope.

// Collection represents a collection of positions. It mirrors
// storage.Collection field for field: Wails binds it under the database
// namespace, so it keeps its own name here and is converted at the edge.
type Collection struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	SortOrder     int    `json:"sortOrder"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
	PositionCount int    `json:"positionCount"`
}

// CollectionPosition represents a position in a collection with its order
type CollectionPosition struct {
	ID           int64    `json:"id"`
	CollectionID int64    `json:"collectionId"`
	PositionID   int64    `json:"positionId"`
	SortOrder    int      `json:"sortOrder"`
	AddedAt      string   `json:"addedAt"`
	Position     Position `json:"position"`
}

// collectionStore returns the collection family of the open database, or the
// error every wrapper method reports when no database is open. The caller
// holds d.mu.
func (d *Database) collectionStore() (storage.CollectionStore, error) {
	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}
	return d.store.Collections(), nil
}

// collectCollections drains a collection stream into the slice the callers
// expect (nil when the stream is empty, as the row scan it replaces produced).
func collectCollections(seq iter.Seq2[*storage.Collection, error]) ([]Collection, error) {
	var collections []Collection
	for c, err := range seq {
		if err != nil {
			return nil, err
		}
		collections = append(collections, Collection(*c))
	}
	return collections, nil
}

// CreateCollection creates a new collection
func (d *Database) CreateCollection(name string, description string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	cs, err := d.collectionStore()
	if err != nil {
		return 0, err
	}
	return cs.Create(context.Background(), "", name, description)
}

// GetAllCollections returns all collections with their position counts
func (d *Database) GetAllCollections() ([]Collection, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	cs, err := d.collectionStore()
	if err != nil {
		return nil, err
	}
	return collectCollections(cs.List(context.Background(), ""))
}

// UpdateCollection updates a collection's name and description
func (d *Database) UpdateCollection(id int64, name string, description string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	cs, err := d.collectionStore()
	if err != nil {
		return err
	}
	return cs.Update(context.Background(), "", id, name, description)
}

// DeleteCollection deletes a collection and all its position associations
func (d *Database) DeleteCollection(id int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	cs, err := d.collectionStore()
	if err != nil {
		return err
	}
	return cs.Delete(context.Background(), "", id)
}

// ReorderCollections updates the sort order of all collections
func (d *Database) ReorderCollections(collectionIDs []int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	cs, err := d.collectionStore()
	if err != nil {
		return err
	}
	return cs.Reorder(context.Background(), "", collectionIDs)
}

// AddPositionToCollection adds a position to a collection
func (d *Database) AddPositionToCollection(collectionID int64, positionID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	cs, err := d.collectionStore()
	if err != nil {
		return err
	}
	return cs.AddPosition(context.Background(), "", collectionID, positionID)
}

// AddPositionsToCollection adds multiple positions to a collection
func (d *Database) AddPositionsToCollection(collectionID int64, positionIDs []int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	cs, err := d.collectionStore()
	if err != nil {
		return err
	}
	return cs.AddPositions(context.Background(), "", collectionID, positionIDs)
}

// RemovePositionFromCollection removes a position from a collection
func (d *Database) RemovePositionFromCollection(collectionID int64, positionID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	cs, err := d.collectionStore()
	if err != nil {
		return err
	}
	return cs.RemovePosition(context.Background(), "", collectionID, positionID)
}

// RemovePositionsFromCollection removes multiple positions from a collection
func (d *Database) RemovePositionsFromCollection(collectionID int64, positionIDs []int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	cs, err := d.collectionStore()
	if err != nil {
		return err
	}
	return cs.RemovePositions(context.Background(), "", collectionID, positionIDs)
}

// GetPositionIndexMap returns a map of position ID to its 1-based index in the database
func (d *Database) GetPositionIndexMap() (map[int64]int, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	cs, err := d.collectionStore()
	if err != nil {
		return nil, err
	}
	return cs.PositionIndexMap(context.Background(), "")
}

// GetCollectionPositions returns all positions in a collection
func (d *Database) GetCollectionPositions(collectionID int64) ([]Position, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	cs, err := d.collectionStore()
	if err != nil {
		return nil, err
	}
	var positions []Position
	for p, err := range cs.Positions(context.Background(), "", collectionID) {
		if err != nil {
			return nil, err
		}
		positions = append(positions, *p)
	}
	return positions, nil
}

// ReorderCollectionPositions updates the sort order of positions within a collection
func (d *Database) ReorderCollectionPositions(collectionID int64, positionIDs []int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	cs, err := d.collectionStore()
	if err != nil {
		return err
	}
	return cs.ReorderPositions(context.Background(), "", collectionID, positionIDs)
}

// MovePositionBetweenCollections moves a position from one collection to another
func (d *Database) MovePositionBetweenCollections(fromCollectionID int64, toCollectionID int64, positionID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	cs, err := d.collectionStore()
	if err != nil {
		return err
	}
	return cs.MovePosition(context.Background(), "", fromCollectionID, toCollectionID, positionID)
}

// CopyPositionToCollection copies a position to a collection (position can be in multiple collections)
func (d *Database) CopyPositionToCollection(toCollectionID int64, positionID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	cs, err := d.collectionStore()
	if err != nil {
		return err
	}
	return cs.CopyPosition(context.Background(), "", toCollectionID, positionID)
}

// GetCollectionByID returns a collection by its ID
func (d *Database) GetCollectionByID(id int64) (*Collection, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	cs, err := d.collectionStore()
	if err != nil {
		return nil, err
	}
	c, err := cs.Get(context.Background(), "", id)
	if err != nil {
		return nil, err
	}
	collection := Collection(*c)
	return &collection, nil
}

// GetPositionCollections returns all collections that contain a specific position
func (d *Database) GetPositionCollections(positionID int64) ([]Collection, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	cs, err := d.collectionStore()
	if err != nil {
		return nil, err
	}
	return collectCollections(cs.CollectionsOf(context.Background(), "", positionID))
}

// ExportCollections exports specific collections to a database file. watermark
// and watermarkNote mirror ExportDatabase's Watermark/WatermarkNote: an empty
// watermark means the export carries none. See db_export_position.go for the
// schema and position-writing code shared with the other two export paths.
func (d *Database) ExportCollections(exportPath string, collectionIDs []int64, metadata map[string]string, includeAnalysis bool, includeComments bool, watermark string, watermarkNote string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	cs, err := d.collectionStore()
	if err != nil {
		return err
	}
	ctx := context.Background()

	// Read each collection and its membership up front — the rows as the
	// export must reproduce them (rank, added_at) and the positions they link,
	// deduplicated across collections in the order the selection lists them.
	// Each stream is drained before the next query runs: the store may sit on
	// a single pooled connection.
	type exportedCollection struct {
		row     *storage.Collection
		members []storage.CollectionPosition
	}
	exported := make([]exportedCollection, 0, len(collectionIDs))
	var positions []Position
	var positionIDs []int64
	seen := make(map[int64]bool)
	for _, collectionID := range collectionIDs {
		row, err := cs.Get(ctx, "", collectionID)
		if err != nil {
			slog.Warn("reading collection for export", "collectionID", collectionID, "err", err)
			continue
		}
		var members []storage.CollectionPosition
		for m, err := range cs.Members(ctx, "", collectionID) {
			if err != nil {
				return fmt.Errorf("cannot read the positions of collection %d to export: %w", collectionID, err)
			}
			members = append(members, *m)
			if !seen[m.PositionID] {
				seen[m.PositionID] = true
				positions = append(positions, m.Position)
				positionIDs = append(positionIDs, m.PositionID)
			}
		}
		exported = append(exported, exportedCollection{row: row, members: members})
	}

	exportDB, err := newExportDB(exportPath)
	if err != nil {
		return err
	}
	defer exportDB.Close()

	if err := writeExportMetadata(exportDB, metadata, watermark, watermarkNote); err != nil {
		return err
	}

	// Read (if requested) every position's analysis/comment in one batched
	// statement each, instead of the N+1 per-position SELECTs this used to run —
	// the same helpers ExportDatabase uses (db_export.go).
	var analysisByPosition map[int64][]byte
	if includeAnalysis {
		if analysisByPosition, err = d.analysisForPositions(positionIDs); err != nil {
			return fmt.Errorf("cannot read analyses to export: %w", err)
		}
	}
	var commentByPosition map[int64]string
	if includeComments {
		if commentByPosition, err = d.commentsForPositions(positionIDs); err != nil {
			return fmt.Errorf("cannot read comments to export: %w", err)
		}
	}

	tx, err := exportDB.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	insertPosition, err := tx.Prepare(exportPositionInsertSQL)
	if err != nil {
		return fmt.Errorf("cannot prepare the position insert: %w", err)
	}
	defer insertPosition.Close()
	lookupPosition, err := tx.Prepare(exportPositionLookupSQL)
	if err != nil {
		return fmt.Errorf("cannot prepare the position lookup: %w", err)
	}
	defer lookupPosition.Close()

	// Export positions and create ID mapping
	oldToNewID := make(map[int64]int64, len(positions))
	for _, pos := range positions {
		posID := pos.ID
		newID, insErr := insertExportPosition(insertPosition, lookupPosition, pos)
		if insErr != nil {
			slog.Warn("inserting position into collection export database", "positionID", posID, "err", insErr)
			continue
		}
		oldToNewID[posID] = newID

		if includeAnalysis {
			if data, ok := analysisByPosition[posID]; ok {
				// Decompress for export compatibility (the export's analysis.data holds
				// plain JSON, like ExportDatabase's).
				jsonData, decErr := decompressAnalysisData(data)
				if decErr != nil {
					slog.Warn("decompressing analysis for collection export", "positionID", posID, "err", decErr)
				} else if _, insErr := tx.Exec(`INSERT INTO analysis (position_id, data) VALUES (?, ?)`, newID, string(jsonData)); insErr != nil {
					slog.Warn("inserting analysis into collection export database", "positionID", posID, "err", insErr)
				}
			}
		}

		if includeComments {
			if text := commentByPosition[posID]; text != "" {
				if _, insErr := tx.Exec(`INSERT INTO comment (position_id, text) VALUES (?, ?)`, newID, text); insErr != nil {
					slog.Warn("inserting comment into collection export database", "positionID", posID, "err", insErr)
				}
			}
		}
	}

	// Export collections and their position mappings
	for _, c := range exported {
		result, insErr := tx.Exec(`INSERT INTO collection (name, description, sort_order, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
			c.row.Name, c.row.Description, c.row.SortOrder, c.row.CreatedAt, c.row.UpdatedAt)
		if insErr != nil {
			slog.Warn("inserting collection into export database", "collectionID", c.row.ID, "err", insErr)
			continue
		}
		newCollectionID, idErr := result.LastInsertId()
		if idErr != nil {
			err = fmt.Errorf("failed to get last insert ID: %w", idErr)
			return err
		}

		for _, m := range c.members {
			newPosID, ok := oldToNewID[m.PositionID]
			if !ok {
				continue
			}
			if _, insErr := tx.Exec(`INSERT INTO collection_position (collection_id, position_id, sort_order, added_at) VALUES (?, ?, ?, ?)`,
				newCollectionID, newPosID, m.SortOrder, m.AddedAt); insErr != nil {
				slog.Warn("inserting collection_position into export database", "collectionID", c.row.ID, "err", insErr)
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

// CollectionCoverage reports, for every collection, how many of its positions are part of
// the given selection.
//
// The export writes a collection's membership only for positions it is actually exporting,
// so a collection whose positions are not all in the selection arrives truncated. That used
// to happen in silence: the recipient believed they had the collection and had a fragment of
// it. This lets the export screen say so before anything is written.
func (d *Database) CollectionCoverage(positionIDs []int64) (map[int64]int, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	cs, err := d.collectionStore()
	if err != nil {
		return nil, err
	}
	return cs.Coverage(context.Background(), "", positionIDs)
}
