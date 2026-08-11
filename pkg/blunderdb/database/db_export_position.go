package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/issuance"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// This file is shared by every export path (ExportDatabase, ExportCollections,
// ExportTournaments): the schema an export file gets, and how a position is
// written into it. Before fiche-04 each of the three hand-rolled its own
// two-column "id, state" position table and its own copy of every other
// table's DDL — a database whose zobrist_hash and every scalar column
// (pip_diff, dice_1, score_1, …) stayed NULL forever, because nothing ever
// wrote them. See tasks/plan-amelioration-2026-08/fiche-04-export-db.md.

// newExportDB creates the SQLite file at exportPath from scratch, using the
// same schema a live database gets (storage/sqlite.Bootstrap) instead of a
// hand-written subset — so a reopened export supports the same zobrist dedup,
// indexes and SQL filters as any other database.
//
// The returned *sql.DB is pinned to a single connection and configured with
// durability-relaxing PRAGMAs: the export target is a throwaway file, rebuilt
// from scratch on any error, so mid-build durability buys nothing and an
// fsync per row is what made large exports appear to hang (see the comment
// this replaces in ExportDatabase's history).
func newExportDB(exportPath string) (*sql.DB, error) {
	if _, err := os.Stat(exportPath); err == nil {
		if err := os.Remove(exportPath); err != nil {
			return nil, fmt.Errorf("cannot remove existing export file: %v", err)
		}
	}
	exportDB, err := sql.Open("sqlite", exportPath)
	if err != nil {
		return nil, err
	}
	exportDB.SetMaxOpenConns(1)
	for _, pragma := range []string{
		"PRAGMA journal_mode=OFF",
		"PRAGMA synchronous=OFF",
		"PRAGMA temp_store=MEMORY",
	} {
		if _, err := exportDB.Exec(pragma); err != nil {
			exportDB.Close()
			return nil, fmt.Errorf("cannot configure export database: %v", err)
		}
	}
	if err := sqlite.Bootstrap(context.Background(), exportDB); err != nil {
		exportDB.Close()
		return nil, fmt.Errorf("cannot create export schema: %w", err)
	}
	return exportDB, nil
}

// exportPositionInsertSQL mirrors storage/sqlite's own position insert (see
// positionInsertSQL in positions_sqlite.go): the same zobrist_hash and every
// scalar column a live SavePosition would fill in, deduplicated by hash via
// ON CONFLICT exactly like a live database. Keep the column list identical to
// that one; the two are independent copies (package sqlite does not export
// its version) but must never drift.
const exportPositionInsertSQL = `INSERT INTO position (
	zobrist_hash, decision_type, player_on_roll, dice_1, dice_2,
	cube_value, cube_owner, score_1, score_2,
	has_jacoby, has_beaver,
	pip_1, pip_2, pip_diff, off_1, off_2,
	back_checkers_1, back_checkers_2, no_contact,
	occupancy_1, occupancy_2, point_mask_1, point_mask_2,
	state, individually_imported, flagged
) VALUES (?,?,?,?,?, ?,?,?,?, ?,?, ?,?,?,?,?, ?,?,?, ?,?,?,?, ?,?,?)
ON CONFLICT(zobrist_hash) DO NOTHING`

// exportPositionLookupSQL resolves the existing row id when
// exportPositionInsertSQL's ON CONFLICT fired.
const exportPositionLookupSQL = `SELECT id FROM position WHERE zobrist_hash = ?`

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// insertExportPosition writes p into an export database through ins (prepared
// from exportPositionInsertSQL) and returns its new row id. When p's Zobrist
// hash already exists in the export — an earlier position in the same batch
// normalizing to the same board, or a source database that itself carries a
// pre-fix duplicate — it returns the existing row's id via lookup (prepared
// from exportPositionLookupSQL) instead of erroring: dedup-on-conflict must
// behave the same way on export as storage/sqlite's positionStore.Save does
// on a live database (D1).
func insertExportPosition(ins, lookup *sql.Stmt, p Position) (int64, error) {
	norm := p.NormalizeForStorage()
	cols := populatePositionColumns(&p)
	result, err := ins.Exec(
		int64(cols.ZobristHash), cols.DecisionType, norm.PlayerOnRoll, cols.Dice1, cols.Dice2,
		cols.CubeValue, cols.CubeOwner, cols.Score1, cols.Score2,
		cols.HasJacoby, cols.HasBeaver,
		cols.Pip1, cols.Pip2, cols.PipDiff, cols.Off1, cols.Off2,
		cols.BackCheckers1, cols.BackCheckers2, boolToInt(cols.NoContact),
		int64(cols.Occupancy1), int64(cols.Occupancy2), int64(cols.PointMask1), int64(cols.PointMask2),
		encodeBoardCompact(norm.Board), boolToInt(norm.IndividuallyImported), boolToInt(norm.Flagged),
	)
	if err != nil {
		return 0, err
	}
	if affected, _ := result.RowsAffected(); affected > 0 {
		return result.LastInsertId()
	}
	var id int64
	if err := lookup.QueryRow(int64(cols.ZobristHash)).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

// writeExportMetadata copies metadata into the export database by
// ALLOW-LIST (issuance.Carried), seals and writes a Watermark when origin is
// non-empty, and fills in a default dateOfCreation when the caller did not
// supply one — the same three steps ExportDatabase performs inline. Used by
// ExportCollections and ExportTournaments, which carried metadata by
// inclusion and never wrote a watermark at all before fiche-04.
//
// database_version is not written here: newExportDB's call to
// sqlite.Bootstrap already stamped it.
func writeExportMetadata(exportDB *sql.DB, metadata map[string]string, watermarkOrigin, watermarkNote string) error {
	watermarkDocument, err := sealWatermark(watermarkOrigin, watermarkNote)
	if err != nil {
		return fmt.Errorf("cannot sign the watermark: %w", err)
	}

	for key, value := range issuance.Carried(metadata) {
		if _, err := exportDB.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES (?, ?)`, key, value); err != nil {
			return fmt.Errorf("cannot write metadata %q: %w", key, err)
		}
	}

	if watermarkDocument != "" {
		if _, err := exportDB.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES (?, ?)`,
			issuance.KeyWatermark, watermarkDocument); err != nil {
			return fmt.Errorf("cannot write the watermark into the exported file: %w", err)
		}
	}

	if metadata["dateOfCreation"] == "" {
		currentDate := time.Now().Format("2006-01-02")
		if _, err := exportDB.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES ('dateOfCreation', ?)`, currentDate); err != nil {
			return fmt.Errorf("cannot write default creation date: %w", err)
		}
	}
	return nil
}
