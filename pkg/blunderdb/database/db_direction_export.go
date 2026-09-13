package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// A Direction travels with its tournament (issue #396, ADR-0047 §9).
//
// A director sends their database to a colleague, or archives it. What leaves is what the
// PRODUCER decided to put in a file they are making — the export is theirs — and nothing at all
// is written on the recipient's side: opening or importing a database leaves no trace in
// anyone's own (the invariant ADR-0007 states, and which this file does not touch).
//
// The copy lives here and not in `ingest` for a reason of perimeter: the direction tables are
// the desktop wrapper's, the daemon exposes nothing of them, and ADR-0039 closed the web front
// to tournaments. `ingest` is backend-agnostic and serves the daemon; teaching it about
// directions would make a promise the other backend never keeps.
//
// The addition is by ALLOW-LIST, like the metadata: the columns copied are named here, one by
// one. A column added to `direction` next year does not travel to someone else's machine
// because it happened to exist.

// exportedDirectionColumns is what a Direction takes with it. Named explicitly, never `SELECT *`:
// the day a column is added, this list is where someone decides whether it leaves.
var exportedDirectionColumns = []string{
	"tournament_id", "format_version", "engine_version", "state", "config", "created_at", "updated_at",
}

// exportedEventColumns is the journal itself, which is the whole truth of a Direction.
var exportedEventColumns = []string{"tournament_id", "seq", "kind", "time", "payload"}

// exportDirections copies the Directions of the exported tournaments into the file just
// written. `tourMap` maps a source tournament id to the one the new file assigned.
//
// `output_dir` is DELIBERATELY absent from the copy: it is a path on the producer's machine,
// and a folder that does not exist on the recipient's would make their first refresh fail for
// a display they never asked for.
func (d *Database) exportDirections(ctx context.Context, path string, tourMap map[int64]int64) error {
	if len(tourMap) == 0 {
		return nil
	}
	// NO LOCK HERE, and it is not an oversight: this runs from inside ExportDatabaseCtx, which
	// already holds d.mu for writing. Go's RWMutex is not reentrant, so taking it again — even
	// for reading — deadlocks the export outright. Found by the first run of this file's tests,
	// which sat for the full ten minutes of the package timeout.
	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	// The same driver and the same PRAGMAs as every other SQLite handle in blunderDB.
	dst, err := sql.Open("sqlite", sqlite.DSN(path))
	if err != nil {
		return fmt.Errorf("direction export: opening %s: %w", path, err)
	}
	defer dst.Close()

	tx, err := dst.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("direction export: begin: %w", err)
	}
	defer tx.Rollback()

	for src, newID := range tourMap {
		copied, err := d.copyDirection(ctx, tx, src, newID)
		if err != nil {
			return err
		}
		if !copied {
			continue
		}
		if err := d.copyDirectionEvents(ctx, tx, src, newID); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("direction export: commit: %w", err)
	}
	return nil
}

// copyDirection writes one Direction row, and says whether there was one to write.
func (d *Database) copyDirection(ctx context.Context, tx *sql.Tx, src, newID int64) (bool, error) {
	var (
		tournamentID                 int64
		formatVersion                int
		engineVersion, state, config string
		createdAt, updatedAt         sql.NullString
	)
	err := d.db.QueryRowContext(ctx, `
		SELECT tournament_id, format_version, engine_version, state, config, created_at, updated_at
		  FROM direction WHERE tournament_id = ?`, src).
		Scan(&tournamentID, &formatVersion, &engineVersion, &state, &config, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("direction export: reading direction %d: %w", src, err)
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO direction (tournament_id, format_version, engine_version, state, config, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, COALESCE(NULLIF(?, ''), CURRENT_TIMESTAMP), COALESCE(NULLIF(?, ''), CURRENT_TIMESTAMP))`,
		newID, formatVersion, engineVersion, state, config, createdAt.String, updatedAt.String)
	if err != nil {
		return false, fmt.Errorf("direction export: writing direction %d: %w", newID, err)
	}
	return true, nil
}

// copyDirectionEvents copies the journal, in order. Order is not decoration: a journal is
// replayed from its first event, and a hole or a swap would replay to another tournament.
func (d *Database) copyDirectionEvents(ctx context.Context, tx *sql.Tx, src, newID int64) error {
	rows, err := d.db.QueryContext(ctx, `
		SELECT seq, kind, time, payload FROM direction_event
		 WHERE tournament_id = ? ORDER BY seq`, src)
	if err != nil {
		return fmt.Errorf("direction export: reading the journal of %d: %w", src, err)
	}
	defer rows.Close()
	type ev struct {
		seq     int
		kind    string
		at      sql.NullString
		payload []byte
	}
	var events []ev
	for rows.Next() {
		var e ev
		if err := rows.Scan(&e.seq, &e.kind, &e.at, &e.payload); err != nil {
			return fmt.Errorf("direction export: reading an event of %d: %w", src, err)
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("direction export: reading the journal of %d: %w", src, err)
	}
	for _, e := range events {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO direction_event (tournament_id, seq, kind, time, payload) VALUES (?, ?, ?, ?, ?)`,
			newID, e.seq, e.kind, e.at.String, e.payload); err != nil {
			return fmt.Errorf("direction export: writing event %d of %d: %w", e.seq, newID, err)
		}
	}
	return nil
}

// directionAfterWrite is the hook handed to ingest: it runs on the freshly written plain file,
// before a protected export seals it.
func (d *Database) directionAfterWrite() func(context.Context, string, ingest.ExportReport) error {
	return func(ctx context.Context, path string, report ingest.ExportReport) error {
		return d.exportDirections(ctx, path, report.TournamentMap)
	}
}
