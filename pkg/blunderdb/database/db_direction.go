package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
)

// The desktop wrapper's implementation of direction.Store (ADR-0047).
//
// Thin: the package holds the rules, this file the statements. The one rule enforced HERE is
// append-only: the (tournament_id, seq) primary key makes a second write at the same seq collide
// instead of replacing a decision.

// DirectionStore returns the Store the direction package runs on.
func (d *Database) DirectionStore() direction.Store { return directionStore{d: d} }

// directionStore runs on the database handle, or inside tx when a gesture must write several
// Directions at once. Inside tx the CALLER holds d.mu for the whole transaction, so the store
// takes no lock of its own.
type directionStore struct {
	d  *Database
	tx *sql.Tx
}

type directionQuerier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func (s directionStore) q() directionQuerier {
	if s.tx != nil {
		return s.tx
	}
	return s.d.db
}

func (s directionStore) rlock() func() {
	if s.tx != nil {
		return func() {}
	}
	s.d.mu.RLock()
	return s.d.mu.RUnlock
}

func (s directionStore) lock() func() {
	if s.tx != nil {
		return func() {}
	}
	s.d.mu.Lock()
	return s.d.mu.Unlock
}

func (s directionStore) GetDirection(ctx context.Context, tournamentID int64) (direction.Record, error) {
	defer s.rlock()()
	return s.get(ctx, tournamentID)
}

func (s directionStore) get(ctx context.Context, tournamentID int64) (direction.Record, error) {
	var (
		rec              direction.Record
		state            string
		created, updated sql.NullString
	)
	err := s.q().QueryRowContext(ctx, `
		SELECT tournament_id, format_version, engine_version, state, config, output_dir, created_at, updated_at
		  FROM direction WHERE tournament_id = ?`, tournamentID).
		Scan(&rec.TournamentID, &rec.FormatVersion, &rec.EngineVersion, &state, &rec.Config, &rec.OutputDir, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return direction.Record{}, direction.ErrNoDirection
	}
	if err != nil {
		return direction.Record{}, fmt.Errorf("reading direction %d: %w", tournamentID, err)
	}
	rec.State = direction.State(state)
	rec.CreatedAt = parseSQLTime(created)
	rec.UpdatedAt = parseSQLTime(updated)
	return rec, nil
}

func (s directionStore) ListDirections(ctx context.Context) ([]direction.Record, error) {
	defer s.rlock()()
	rows, err := s.q().QueryContext(ctx, `
		SELECT tournament_id, format_version, engine_version, state, config, output_dir, created_at, updated_at
		  FROM direction ORDER BY tournament_id`)
	if err != nil {
		return nil, fmt.Errorf("listing directions: %w", err)
	}
	defer rows.Close()
	var out []direction.Record
	for rows.Next() {
		var (
			rec              direction.Record
			state            string
			created, updated sql.NullString
		)
		if err := rows.Scan(&rec.TournamentID, &rec.FormatVersion, &rec.EngineVersion, &state,
			&rec.Config, &rec.OutputDir, &created, &updated); err != nil {
			return nil, err
		}
		rec.State = direction.State(state)
		rec.CreatedAt = parseSQLTime(created)
		rec.UpdatedAt = parseSQLTime(updated)
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (s directionStore) CreateDirection(ctx context.Context, rec direction.Record) error {
	defer s.lock()()
	_, err := s.q().ExecContext(ctx, `
		INSERT INTO direction (tournament_id, format_version, engine_version, state, config, output_dir)
		VALUES (?, ?, ?, ?, ?, ?)`,
		rec.TournamentID, rec.FormatVersion, rec.EngineVersion, string(rec.State), rec.Config, rec.OutputDir)
	if err != nil {
		return fmt.Errorf("creating direction %d: %w", rec.TournamentID, err)
	}
	return nil
}

func (s directionStore) UpdateDirection(ctx context.Context, rec direction.Record) error {
	defer s.lock()()
	res, err := s.q().ExecContext(ctx, `
		UPDATE direction SET format_version = ?, engine_version = ?, state = ?, config = ?,
		       output_dir = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE tournament_id = ?`,
		rec.FormatVersion, rec.EngineVersion, string(rec.State), rec.Config, rec.OutputDir, rec.TournamentID)
	if err != nil {
		return fmt.Errorf("updating direction %d: %w", rec.TournamentID, err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return direction.ErrNoDirection
	}
	return nil
}

func (s directionStore) DeleteDirection(ctx context.Context, tournamentID int64) error {
	defer s.lock()()
	tx, err := s.d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	// The events go with the Direction. The Matches keep their Tournament and lose only the
	// Slot they filled — deleting a Direction never deletes a Match (ADR-0047).
	if _, err := tx.ExecContext(ctx, `DELETE FROM direction_event WHERE tournament_id = ?`, tournamentID); err != nil {
		return fmt.Errorf("deleting direction events: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM direction WHERE tournament_id = ?`, tournamentID); err != nil {
		return fmt.Errorf("deleting direction: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE match SET direction_match_id = '' WHERE tournament_id = ?`, tournamentID); err != nil {
		return fmt.Errorf("clearing slots: %w", err)
	}
	return tx.Commit()
}

func (s directionStore) AppendEvent(ctx context.Context, tournamentID int64, ev direction.StoredEvent) error {
	defer s.lock()()
	// No UPSERT and no OR REPLACE: a second write at this sequence number must fail. The log is
	// append-only, and silently overwriting would lose a decision.
	_, err := s.q().ExecContext(ctx, `
		INSERT INTO direction_event (tournament_id, seq, kind, time, payload)
		VALUES (?, ?, ?, ?, ?)`,
		tournamentID, ev.Seq, ev.Kind, ev.Time.UTC().Format(time.RFC3339Nano), string(ev.Payload))
	if err != nil {
		return fmt.Errorf("appending event %d to direction %d: %w", ev.Seq, tournamentID, err)
	}
	if _, err := s.q().ExecContext(ctx,
		`UPDATE direction SET updated_at = CURRENT_TIMESTAMP WHERE tournament_id = ?`, tournamentID); err != nil {
		return fmt.Errorf("touching direction %d: %w", tournamentID, err)
	}
	return nil
}

func (s directionStore) LoadEvents(ctx context.Context, tournamentID int64) ([]direction.StoredEvent, error) {
	defer s.rlock()()
	rows, err := s.q().QueryContext(ctx, `
		SELECT seq, kind, time, payload FROM direction_event
		 WHERE tournament_id = ? ORDER BY seq`, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("loading direction %d: %w", tournamentID, err)
	}
	defer rows.Close()
	var out []direction.StoredEvent
	for rows.Next() {
		var (
			ev      direction.StoredEvent
			ts      sql.NullString
			payload string
		)
		if err := rows.Scan(&ev.Seq, &ev.Kind, &ts, &payload); err != nil {
			return nil, err
		}
		ev.Time = parseSQLTime(ts)
		ev.Payload = []byte(payload)
		out = append(out, ev)
	}
	return out, rows.Err()
}

// parseSQLTime reads the several shapes SQLite hands back for a DATETIME column. An unreadable
// value yields the zero time rather than an error: a timestamp is information here, never a key.
func parseSQLTime(v sql.NullString) time.Time {
	if !v.Valid || v.String == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05.999999999-07:00", "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, v.String); err == nil {
			return t
		}
	}
	return time.Time{}
}
