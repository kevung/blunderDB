package postgres

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type tournamentStore struct{ db execer }

var _ storage.TournamentStore = (*tournamentStore)(nil)

// tsTime formats a TIMESTAMPTZ for the string CreatedAt/UpdatedAt fields,
// matching the SQLite backend's "YYYY-MM-DD HH:MM:SS" rendering.
func tsTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// tournamentSelectExpr reads a domain.Tournament. The correlated subquery
// counts the tournament's matches; both rows carry the same tenant_id.
const tournamentSelectExpr = `t.id, t.name, COALESCE(t.date,''), COALESCE(t.location,''),
	COALESCE(t.sort_order,0), t.created_at, t.updated_at,
	(SELECT COUNT(*) FROM match m WHERE m.tournament_id = t.id AND m.tenant_id = t.tenant_id),
	COALESCE(t.comment,'')`

func scanTournament(sc scanner) (domain.Tournament, error) {
	var t domain.Tournament
	var createdAt, updatedAt time.Time
	if err := sc.Scan(&t.ID, &t.Name, &t.Date, &t.Location,
		&t.SortOrder, &createdAt, &updatedAt, &t.MatchCount, &t.Comment); err != nil {
		return domain.Tournament{}, err
	}
	t.CreatedAt = tsTime(createdAt)
	t.UpdatedAt = tsTime(updatedAt)
	return t, nil
}

// Create stores a new tournament at the end of the sort order and returns its
// id.
func (s *tournamentStore) Create(ctx context.Context, scope string, name, date, location string) (int64, error) {
	tenant := tenantID(scope)
	var maxOrder int
	if err := s.db.QueryRow(ctx,
		`SELECT COALESCE(MAX(sort_order), -1) FROM tournament WHERE tenant_id = $1`,
		tenant).Scan(&maxOrder); err != nil {
		maxOrder = -1
	}
	var id int64
	if err := s.db.QueryRow(ctx,
		`INSERT INTO tournament (tenant_id, name, date, location, sort_order)
		 VALUES ($1,$2,$3,$4,$5) RETURNING id`,
		tenant, name, date, location, maxOrder+1).Scan(&id); err != nil {
		return 0, fmt.Errorf("postgres: create tournament: %w", err)
	}
	return id, nil
}

// Get returns the tournament with the given id, or ErrNotFound.
func (s *tournamentStore) Get(ctx context.Context, scope string, id int64) (*domain.Tournament, error) {
	row := s.db.QueryRow(ctx,
		`SELECT `+tournamentSelectExpr+` FROM tournament t WHERE t.id = $1 AND t.tenant_id = $2`,
		id, tenantID(scope))
	t, err := scanTournament(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("postgres: get tournament %d: %w", id, storage.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: get tournament %d: %w", id, err)
	}
	return &t, nil
}

// List streams every tournament, most recent first.
func (s *tournamentStore) List(ctx context.Context, scope string) iter.Seq2[*domain.Tournament, error] {
	return func(yield func(*domain.Tournament, error) bool) {
		rows, err := s.db.Query(ctx,
			`SELECT `+tournamentSelectExpr+` FROM tournament t
			 WHERE t.tenant_id = $1
			 ORDER BY t.date DESC, t.created_at DESC`, tenantID(scope))
		if err != nil {
			yield(nil, fmt.Errorf("postgres: list tournaments: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			t, err := scanTournament(rows)
			if err != nil {
				yield(nil, fmt.Errorf("postgres: list tournaments: %w", err))
				return
			}
			if !yield(&t, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("postgres: list tournaments: %w", err))
		}
	}
}

// Update changes a tournament's editable header fields.
func (s *tournamentStore) Update(ctx context.Context, scope string, id int64, name, date, location string) error {
	if _, err := s.db.Exec(ctx,
		`UPDATE tournament SET name = $1, date = $2, location = $3, updated_at = now()
		 WHERE id = $4 AND tenant_id = $5`,
		name, date, location, id, tenantID(scope)); err != nil {
		return fmt.Errorf("postgres: update tournament %d: %w", id, err)
	}
	return nil
}

// UpdateComment sets the free-text comment on a tournament.
func (s *tournamentStore) UpdateComment(ctx context.Context, scope string, id int64, comment string) error {
	if _, err := s.db.Exec(ctx,
		`UPDATE tournament SET comment = $1, updated_at = now() WHERE id = $2 AND tenant_id = $3`,
		comment, id, tenantID(scope)); err != nil {
		return fmt.Errorf("postgres: update tournament %d comment: %w", id, err)
	}
	return nil
}

// Delete removes a tournament; its matches are unlinked (ON DELETE SET NULL on
// match.tournament_id), not deleted.
func (s *tournamentStore) Delete(ctx context.Context, scope string, id int64) error {
	if _, err := s.db.Exec(ctx,
		`DELETE FROM tournament WHERE id = $1 AND tenant_id = $2`,
		id, tenantID(scope)); err != nil {
		return fmt.Errorf("postgres: delete tournament %d: %w", id, err)
	}
	return nil
}

// AddMatch appends a match to a tournament at the end of its match order.
func (s *tournamentStore) AddMatch(ctx context.Context, scope string, tournamentID, matchID int64) error {
	tenant := tenantID(scope)
	err := withTx(ctx, s.db, func(tx execer) error {
		var maxOrder int
		if err := tx.QueryRow(ctx,
			`SELECT COALESCE(MAX(tournament_sort_order), -1) FROM match
			 WHERE tournament_id = $1 AND tenant_id = $2`,
			tournamentID, tenant).Scan(&maxOrder); err != nil {
			maxOrder = -1
		}
		if _, err := tx.Exec(ctx,
			`UPDATE match SET tournament_id = $1, tournament_sort_order = $2
			 WHERE id = $3 AND tenant_id = $4`,
			tournamentID, maxOrder+1, matchID, tenant); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx,
			`UPDATE tournament SET updated_at = now() WHERE id = $1 AND tenant_id = $2`,
			tournamentID, tenant); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("postgres: add match %d to tournament %d: %w", matchID, tournamentID, err)
	}
	return nil
}

// RemoveMatch detaches a match from whatever tournament it belongs to.
func (s *tournamentStore) RemoveMatch(ctx context.Context, scope string, matchID int64) error {
	if _, err := s.db.Exec(ctx,
		`UPDATE match SET tournament_id = NULL, tournament_sort_order = 0
		 WHERE id = $1 AND tenant_id = $2`,
		matchID, tenantID(scope)); err != nil {
		return fmt.Errorf("postgres: remove match %d from tournament: %w", matchID, err)
	}
	return nil
}

// SetMatchByName links a match to the tournament with the given name, creating
// it when absent. An empty name detaches the match from any tournament.
func (s *tournamentStore) SetMatchByName(ctx context.Context, scope string, matchID int64, tournamentName string) error {
	tenant := tenantID(scope)
	name := strings.TrimSpace(tournamentName)
	if name == "" {
		if _, err := s.db.Exec(ctx,
			`UPDATE match SET tournament_id = NULL WHERE id = $1 AND tenant_id = $2`,
			matchID, tenant); err != nil {
			return fmt.Errorf("postgres: detach match %d from tournament: %w", matchID, err)
		}
		return nil
	}
	err := withTx(ctx, s.db, func(tx execer) error {
		var tournamentID int64
		err := tx.QueryRow(ctx,
			`SELECT id FROM tournament WHERE name = $1 AND tenant_id = $2`,
			name, tenant).Scan(&tournamentID)
		if errors.Is(err, pgx.ErrNoRows) {
			if err := tx.QueryRow(ctx,
				`INSERT INTO tournament (tenant_id, name, date, location)
				 VALUES ($1, $2, '', '') RETURNING id`,
				tenant, name).Scan(&tournamentID); err != nil {
				return fmt.Errorf("create tournament: %w", err)
			}
		} else if err != nil {
			return fmt.Errorf("lookup tournament: %w", err)
		}
		if _, err := tx.Exec(ctx,
			`UPDATE match SET tournament_id = $1 WHERE id = $2 AND tenant_id = $3`,
			tournamentID, matchID, tenant); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx,
			`UPDATE tournament SET updated_at = now() WHERE id = $1 AND tenant_id = $2`,
			tournamentID, tenant); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("postgres: set tournament %q for match %d: %w", name, matchID, err)
	}
	return nil
}

// ReorderMatches assigns tournament_sort_order to the tournament's matches in
// the order matchIDs lists them.
func (s *tournamentStore) ReorderMatches(ctx context.Context, scope string, tournamentID int64, matchIDs []int64) error {
	tenant := tenantID(scope)
	err := withTx(ctx, s.db, func(tx execer) error {
		for i, matchID := range matchIDs {
			if _, err := tx.Exec(ctx,
				`UPDATE match SET tournament_sort_order = $1
				 WHERE id = $2 AND tournament_id = $3 AND tenant_id = $4`,
				i, matchID, tournamentID, tenant); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(ctx,
			`UPDATE tournament SET updated_at = now() WHERE id = $1 AND tenant_id = $2`,
			tournamentID, tenant); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("postgres: reorder matches of tournament %d: %w", tournamentID, err)
	}
	return nil
}

// Matches streams the matches of a tournament in their tournament order.
func (s *tournamentStore) Matches(ctx context.Context, scope string, tournamentID int64) iter.Seq2[*domain.Match, error] {
	return func(yield func(*domain.Match, error) bool) {
		rows, err := s.db.Query(ctx,
			`SELECT `+matchSelectCols+` FROM match m
			 LEFT JOIN tournament t ON m.tournament_id = t.id
			 WHERE m.tournament_id = $1 AND m.tenant_id = $2
			 ORDER BY m.tournament_sort_order ASC, m.match_date DESC NULLS LAST`,
			tournamentID, tenantID(scope))
		if err != nil {
			yield(nil, fmt.Errorf("postgres: list tournament matches: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			m, err := scanMatch(rows)
			if err != nil {
				yield(nil, fmt.Errorf("postgres: list tournament matches: %w", err))
				return
			}
			if !yield(&m, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("postgres: list tournament matches: %w", err))
		}
	}
}

// TournamentOf returns the tournament a match belongs to, or ErrNotFound when
// the match is unknown or not linked to any tournament.
func (s *tournamentStore) TournamentOf(ctx context.Context, scope string, matchID int64) (*domain.Tournament, error) {
	tenant := tenantID(scope)
	var tournamentID *int64
	err := s.db.QueryRow(ctx,
		`SELECT tournament_id FROM match WHERE id = $1 AND tenant_id = $2`,
		matchID, tenant).Scan(&tournamentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("postgres: tournament of match %d: %w", matchID, storage.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: tournament of match %d: %w", matchID, err)
	}
	if tournamentID == nil {
		return nil, fmt.Errorf("postgres: tournament of match %d: %w", matchID, storage.ErrNotFound)
	}
	return s.Get(ctx, scope, *tournamentID)
}
