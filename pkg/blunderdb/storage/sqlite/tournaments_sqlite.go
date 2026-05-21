package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"iter"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type tournamentStore struct{ db execer }

var _ storage.TournamentStore = (*tournamentStore)(nil)

// tournamentSelectCols reads a domain.Tournament; the correlated subquery
// supplies the match count.
const tournamentSelectCols = `t.id, t.name, COALESCE(t.date,''), COALESCE(t.location,''),
	COALESCE(t.sort_order,0), COALESCE(t.created_at,''), COALESCE(t.updated_at,''),
	(SELECT COUNT(*) FROM match m WHERE m.tournament_id = t.id),
	COALESCE(t.comment,'')`

func scanTournament(sc interface{ Scan(...any) error }) (domain.Tournament, error) {
	var t domain.Tournament
	if err := sc.Scan(&t.ID, &t.Name, &t.Date, &t.Location,
		&t.SortOrder, &t.CreatedAt, &t.UpdatedAt, &t.MatchCount, &t.Comment); err != nil {
		return domain.Tournament{}, err
	}
	return t, nil
}

// Create stores a new tournament at the end of the sort order and returns its
// id.
func (s *tournamentStore) Create(ctx context.Context, scope string, name, date, location string) (int64, error) {
	var maxOrder int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(sort_order), -1) FROM tournament`).Scan(&maxOrder); err != nil {
		maxOrder = -1
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO tournament (name, date, location, sort_order) VALUES (?,?,?,?)`,
		name, date, location, maxOrder+1)
	if err != nil {
		return 0, fmt.Errorf("sqlite: create tournament: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("sqlite: create tournament id: %w", err)
	}
	return id, nil
}

// Get returns the tournament with the given id, or ErrNotFound.
func (s *tournamentStore) Get(ctx context.Context, scope string, id int64) (*domain.Tournament, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+tournamentSelectCols+` FROM tournament t WHERE t.id = ?`, id)
	t, err := scanTournament(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("sqlite: get tournament %d: %w", id, storage.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("sqlite: get tournament %d: %w", id, err)
	}
	return &t, nil
}

// List streams every tournament, most recent first.
func (s *tournamentStore) List(ctx context.Context, scope string) iter.Seq2[*domain.Tournament, error] {
	return func(yield func(*domain.Tournament, error) bool) {
		rows, err := s.db.QueryContext(ctx,
			`SELECT `+tournamentSelectCols+` FROM tournament t
			 ORDER BY t.date DESC, t.created_at DESC`)
		if err != nil {
			yield(nil, fmt.Errorf("sqlite: list tournaments: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			t, err := scanTournament(rows)
			if err != nil {
				yield(nil, fmt.Errorf("sqlite: list tournaments: %w", err))
				return
			}
			if !yield(&t, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("sqlite: list tournaments: %w", err))
		}
	}
}

// Update changes a tournament's editable header fields.
func (s *tournamentStore) Update(ctx context.Context, scope string, id int64, name, date, location string) error {
	if _, err := s.db.ExecContext(ctx,
		`UPDATE tournament SET name = ?, date = ?, location = ?, updated_at = datetime('now')
		 WHERE id = ?`, name, date, location, id); err != nil {
		return fmt.Errorf("sqlite: update tournament %d: %w", id, err)
	}
	return nil
}

// UpdateComment sets the free-text comment on a tournament.
func (s *tournamentStore) UpdateComment(ctx context.Context, scope string, id int64, comment string) error {
	if _, err := s.db.ExecContext(ctx,
		`UPDATE tournament SET comment = ?, updated_at = datetime('now') WHERE id = ?`,
		comment, id); err != nil {
		return fmt.Errorf("sqlite: update tournament %d comment: %w", id, err)
	}
	return nil
}

// Delete removes a tournament; its matches are unlinked, not deleted.
func (s *tournamentStore) Delete(ctx context.Context, scope string, id int64) error {
	err := withTx(ctx, s.db, func(tx execer) error {
		if _, err := tx.ExecContext(ctx,
			`UPDATE match SET tournament_id = NULL WHERE tournament_id = ?`, id); err != nil {
			return fmt.Errorf("unlink matches: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM tournament WHERE id = ?`, id); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("sqlite: delete tournament %d: %w", id, err)
	}
	return nil
}

// AddMatch appends a match to a tournament at the end of its match order.
func (s *tournamentStore) AddMatch(ctx context.Context, scope string, tournamentID, matchID int64) error {
	err := withTx(ctx, s.db, func(tx execer) error {
		var maxOrder int
		if err := tx.QueryRowContext(ctx,
			`SELECT COALESCE(MAX(tournament_sort_order), -1) FROM match WHERE tournament_id = ?`,
			tournamentID).Scan(&maxOrder); err != nil {
			maxOrder = -1
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE match SET tournament_id = ?, tournament_sort_order = ? WHERE id = ?`,
			tournamentID, maxOrder+1, matchID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE tournament SET updated_at = datetime('now') WHERE id = ?`, tournamentID); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("sqlite: add match %d to tournament %d: %w", matchID, tournamentID, err)
	}
	return nil
}

// RemoveMatch detaches a match from whatever tournament it belongs to.
func (s *tournamentStore) RemoveMatch(ctx context.Context, scope string, matchID int64) error {
	if _, err := s.db.ExecContext(ctx,
		`UPDATE match SET tournament_id = NULL, tournament_sort_order = 0 WHERE id = ?`,
		matchID); err != nil {
		return fmt.Errorf("sqlite: remove match %d from tournament: %w", matchID, err)
	}
	return nil
}

// SetMatchByName links a match to the tournament with the given name, creating
// it when absent. An empty name detaches the match from any tournament.
func (s *tournamentStore) SetMatchByName(ctx context.Context, scope string, matchID int64, tournamentName string) error {
	name := strings.TrimSpace(tournamentName)
	if name == "" {
		if _, err := s.db.ExecContext(ctx,
			`UPDATE match SET tournament_id = NULL WHERE id = ?`, matchID); err != nil {
			return fmt.Errorf("sqlite: detach match %d from tournament: %w", matchID, err)
		}
		return nil
	}
	err := withTx(ctx, s.db, func(tx execer) error {
		var tournamentID int64
		err := tx.QueryRowContext(ctx,
			`SELECT id FROM tournament WHERE name = ?`, name).Scan(&tournamentID)
		if errors.Is(err, sql.ErrNoRows) {
			res, err := tx.ExecContext(ctx,
				`INSERT INTO tournament (name, date, location) VALUES (?, '', '')`, name)
			if err != nil {
				return fmt.Errorf("create tournament: %w", err)
			}
			if tournamentID, err = res.LastInsertId(); err != nil {
				return fmt.Errorf("create tournament id: %w", err)
			}
		} else if err != nil {
			return fmt.Errorf("lookup tournament: %w", err)
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE match SET tournament_id = ? WHERE id = ?`, tournamentID, matchID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE tournament SET updated_at = datetime('now') WHERE id = ?`, tournamentID); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("sqlite: set tournament %q for match %d: %w", name, matchID, err)
	}
	return nil
}

// ReorderMatches assigns tournament_sort_order to the tournament's matches in
// the order matchIDs lists them.
func (s *tournamentStore) ReorderMatches(ctx context.Context, scope string, tournamentID int64, matchIDs []int64) error {
	err := withTx(ctx, s.db, func(tx execer) error {
		for i, matchID := range matchIDs {
			if _, err := tx.ExecContext(ctx,
				`UPDATE match SET tournament_sort_order = ? WHERE id = ? AND tournament_id = ?`,
				i, matchID, tournamentID); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE tournament SET updated_at = datetime('now') WHERE id = ?`, tournamentID); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("sqlite: reorder matches of tournament %d: %w", tournamentID, err)
	}
	return nil
}

// Matches streams the matches of a tournament in their tournament order.
func (s *tournamentStore) Matches(ctx context.Context, scope string, tournamentID int64) iter.Seq2[*domain.Match, error] {
	return func(yield func(*domain.Match, error) bool) {
		rows, err := s.db.QueryContext(ctx,
			`SELECT `+matchSelectCols+` FROM match m
			 LEFT JOIN tournament t ON m.tournament_id = t.id
			 WHERE m.tournament_id = ?
			 ORDER BY m.tournament_sort_order ASC, m.match_date DESC`, tournamentID)
		if err != nil {
			yield(nil, fmt.Errorf("sqlite: list tournament matches: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			m, err := scanMatch(rows)
			if err != nil {
				yield(nil, fmt.Errorf("sqlite: list tournament matches: %w", err))
				return
			}
			if !yield(&m, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("sqlite: list tournament matches: %w", err))
		}
	}
}

// TournamentOf returns the tournament a match belongs to, or ErrNotFound when
// the match is unknown or not linked to any tournament.
func (s *tournamentStore) TournamentOf(ctx context.Context, scope string, matchID int64) (*domain.Tournament, error) {
	var tournamentID sql.NullInt64
	err := s.db.QueryRowContext(ctx,
		`SELECT tournament_id FROM match WHERE id = ?`, matchID).Scan(&tournamentID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("sqlite: tournament of match %d: %w", matchID, storage.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("sqlite: tournament of match %d: %w", matchID, err)
	}
	if !tournamentID.Valid {
		return nil, fmt.Errorf("sqlite: tournament of match %d: %w", matchID, storage.ErrNotFound)
	}
	return s.Get(ctx, scope, tournamentID.Int64)
}
