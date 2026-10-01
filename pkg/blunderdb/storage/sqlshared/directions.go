package sqlshared

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// DirectionStore implements storage.DirectionStore over the direction and
// direction_event tables. Every statement is confined to the scope's tenant.
//
// Append-only is enforced by the (tournament_id, seq) primary key: the insert
// says ON CONFLICT DO NOTHING and a row count of zero becomes ErrConflict, so
// two directors racing for the same sequence number get one success and one
// typed refusal, never an overwrite.
type DirectionStore struct{ DB Execer }

var _ storage.DirectionStore = (*DirectionStore)(nil)

const directionCols = `tournament_id, format_version, engine_version, state, config, COALESCE(output_dir,''), created_at, updated_at`

func scanDirection(sc interface{ Scan(...any) error }) (direction.Record, error) {
	var (
		rec              direction.Record
		state            string
		created, updated sqlTime
	)
	if err := sc.Scan(&rec.TournamentID, &rec.FormatVersion, &rec.EngineVersion, &state,
		&rec.Config, &rec.OutputDir, &created, &updated); err != nil {
		return direction.Record{}, err
	}
	rec.State = direction.State(state)
	rec.CreatedAt, rec.UpdatedAt = created.t, updated.t
	return rec, nil
}

func (s *DirectionStore) noDirection(tournamentID int64) error {
	return fmt.Errorf("%s: direction %d: %w: %w", s.DB.Name(), tournamentID, direction.ErrNoDirection, storage.ErrNotFound)
}

// Get returns the record of a directed Tournament.
func (s *DirectionStore) Get(ctx context.Context, scope string, tournamentID int64) (direction.Record, error) {
	tenant, targs := s.DB.TenantFilter("", scope)
	rec, err := scanDirection(s.DB.QueryRow(ctx,
		`SELECT `+directionCols+` FROM direction WHERE tournament_id = ? AND `+tenant,
		append([]any{tournamentID}, targs...)...))
	if errors.Is(err, ErrNoRows) {
		return direction.Record{}, s.noDirection(tournamentID)
	}
	if err != nil {
		return direction.Record{}, errf(s.DB, "get direction", err)
	}
	return rec, nil
}

// List returns every record of the scope by Tournament id.
func (s *DirectionStore) List(ctx context.Context, scope string) ([]direction.Record, error) {
	tenant, targs := s.DB.TenantFilter("", scope)
	rows, err := s.DB.Query(ctx,
		`SELECT `+directionCols+` FROM direction WHERE `+tenant+` ORDER BY tournament_id`, targs...)
	if err != nil {
		return nil, errf(s.DB, "list directions", err)
	}
	defer rows.Close()
	var out []direction.Record
	for rows.Next() {
		rec, err := scanDirection(rows)
		if err != nil {
			return nil, errf(s.DB, "list directions", err)
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, errf(s.DB, "list directions", err)
	}
	return out, nil
}

// Create stores the record of a Tournament of the scope.
func (s *DirectionStore) Create(ctx context.Context, scope string, rec direction.Record) error {
	return s.DB.Transact(ctx, func(tx Execer) error {
		tenant, targs := tx.TenantFilter("", scope)
		var one int
		err := tx.QueryRow(ctx, `SELECT 1 FROM tournament WHERE id = ? AND `+tenant,
			append([]any{rec.TournamentID}, targs...)...).Scan(&one)
		if errors.Is(err, ErrNoRows) {
			return fmt.Errorf("%s: create direction: tournament %d: %w", tx.Name(), rec.TournamentID, storage.ErrNotFound)
		}
		if err != nil {
			return errf(tx, "create direction", err)
		}
		cols, args := tx.TenantColumns(scope)
		cols = append(cols, "tournament_id", "format_version", "engine_version", "state", "config", "output_dir")
		args = append(args, rec.TournamentID, rec.FormatVersion, rec.EngineVersion, string(rec.State), rec.Config, rec.OutputDir)
		n, err := tx.Exec(ctx, `INSERT INTO direction (`+strings.Join(cols, ", ")+`) VALUES (`+Placeholders(len(cols))+`)
			ON CONFLICT (tournament_id) DO NOTHING`, args...)
		if err != nil {
			return errf(tx, "create direction", err)
		}
		if n == 0 {
			return fmt.Errorf("%s: create direction %d: %w", tx.Name(), rec.TournamentID, storage.ErrConflict)
		}
		return nil
	})
}

// Update rewrites the record's own facts.
func (s *DirectionStore) Update(ctx context.Context, scope string, rec direction.Record) error {
	tenant, targs := s.DB.TenantFilter("", scope)
	n, err := s.DB.Exec(ctx, `UPDATE direction SET format_version = ?, engine_version = ?, state = ?, config = ?,
		output_dir = ?, updated_at = CURRENT_TIMESTAMP WHERE tournament_id = ? AND `+tenant,
		append([]any{rec.FormatVersion, rec.EngineVersion, string(rec.State), rec.Config, rec.OutputDir, rec.TournamentID}, targs...)...)
	if err != nil {
		return errf(s.DB, "update direction", err)
	}
	if n == 0 {
		return s.noDirection(rec.TournamentID)
	}
	return nil
}

// Delete removes the record, its log and its pairs. The Matches keep their Tournament
// and lose only the Slot they filled.
func (s *DirectionStore) Delete(ctx context.Context, scope string, tournamentID int64) error {
	return s.DB.Transact(ctx, func(tx Execer) error {
		tenant, targs := tx.TenantFilter("", scope)
		args := append([]any{tournamentID}, targs...)
		for _, st := range []struct{ what, sql string }{
			{"delete direction events", `DELETE FROM direction_event WHERE tournament_id = ? AND ` + tenant},
			{"delete direction", `DELETE FROM direction WHERE tournament_id = ? AND ` + tenant},
			{"clear direction slots", `UPDATE match SET direction_match_id = '' WHERE tournament_id = ? AND ` + tenant},
			{"delete direction pairs", `DELETE FROM direction_pair_member WHERE tournament_id = ? AND ` + tenant},
		} {
			if _, err := tx.Exec(ctx, st.sql, args...); err != nil {
				return errf(tx, st.what, err)
			}
		}
		return nil
	})
}

// AppendEvent adds one event to the log, refusing a sequence number already
// written.
func (s *DirectionStore) AppendEvent(ctx context.Context, scope string, tournamentID int64, ev direction.StoredEvent) error {
	return s.DB.Transact(ctx, func(tx Execer) error {
		tenant, targs := tx.TenantFilter("", scope)
		var one int
		err := tx.QueryRow(ctx, `SELECT 1 FROM direction WHERE tournament_id = ? AND `+tenant,
			append([]any{tournamentID}, targs...)...).Scan(&one)
		if errors.Is(err, ErrNoRows) {
			return s.noDirection(tournamentID)
		}
		if err != nil {
			return errf(tx, "append direction event", err)
		}
		cols, args := tx.TenantColumns(scope)
		marks := strings.TrimSuffix(strings.Repeat("?, ", len(cols)), ", ")
		if marks != "" {
			marks += ", "
		}
		cols = append(cols, "tournament_id", "seq", "kind", "time", "payload")
		args = append(args, tournamentID, ev.Seq, ev.Kind, ev.Time.UTC().Format(time.RFC3339Nano), string(ev.Payload))
		n, err := tx.Exec(ctx, `INSERT INTO direction_event (`+strings.Join(cols, ", ")+`)
			VALUES (`+marks+`?, ?, ?, `+tx.TimestampArg()+`, ?)
			ON CONFLICT (tournament_id, seq) DO NOTHING`, args...)
		if err != nil {
			return errf(tx, "append direction event", err)
		}
		if n == 0 {
			return fmt.Errorf("%s: append event %d to direction %d: %w", tx.Name(), ev.Seq, tournamentID, storage.ErrConflict)
		}
		if _, err := tx.Exec(ctx, `UPDATE direction SET updated_at = CURRENT_TIMESTAMP WHERE tournament_id = ? AND `+tenant,
			append([]any{tournamentID}, targs...)...); err != nil {
			return errf(tx, "touch direction", err)
		}
		return nil
	})
}

// LoadEvents returns the log in sequence order.
func (s *DirectionStore) LoadEvents(ctx context.Context, scope string, tournamentID int64) ([]direction.StoredEvent, error) {
	tenant, targs := s.DB.TenantFilter("", scope)
	rows, err := s.DB.Query(ctx, `SELECT seq, kind, time, payload FROM direction_event
		WHERE tournament_id = ? AND `+tenant+` ORDER BY seq`, append([]any{tournamentID}, targs...)...)
	if err != nil {
		return nil, errf(s.DB, "load direction events", err)
	}
	defer rows.Close()
	var out []direction.StoredEvent
	for rows.Next() {
		var (
			ev      direction.StoredEvent
			ts      sqlTime
			payload string
		)
		if err := rows.Scan(&ev.Seq, &ev.Kind, &ts, &payload); err != nil {
			return nil, errf(s.DB, "load direction events", err)
		}
		ev.Time = ts.t
		ev.Payload = []byte(payload)
		out = append(out, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, errf(s.DB, "load direction events", err)
	}
	return out, nil
}

// EventsHead counts the log and reads its last sequence number.
func (s *DirectionStore) EventsHead(ctx context.Context, scope string, tournamentID int64) (count, lastSeq int, err error) {
	tenant, targs := s.DB.TenantFilter("", scope)
	err = s.DB.QueryRow(ctx, `SELECT COUNT(*), COALESCE(MAX(seq), -1) FROM direction_event
		WHERE tournament_id = ? AND `+tenant, append([]any{tournamentID}, targs...)...).Scan(&count, &lastSeq)
	if err != nil {
		return 0, 0, errf(s.DB, "direction events head", err)
	}
	return count, lastSeq, nil
}

// sqlTime scans a timestamp column whatever shape the driver hands back: a
// time.Time from PostgreSQL's TIMESTAMPTZ, text or a time.Time from SQLite's
// DATETIME. Reading it raw rather than through Dialect.TimestampText keeps
// the sub-second part an event was written with. An unreadable value is the
// zero time: a timestamp is information here, never a key.
type sqlTime struct{ t time.Time }

func (st *sqlTime) Scan(src any) error {
	switch v := src.(type) {
	case time.Time:
		st.t = v
	case string:
		st.t = parseTimeText(v)
	case []byte:
		st.t = parseTimeText(string(v))
	default:
		st.t = time.Time{}
	}
	return nil
}

func parseTimeText(s string) time.Time {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05.999999999-07:00", "2006-01-02 15:04:05.999999999"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
