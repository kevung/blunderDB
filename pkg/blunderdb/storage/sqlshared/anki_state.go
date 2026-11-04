package sqlshared

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// AnkiStateStore implements storage.AnkiStateStore over anki_deck and
// anki_card.
type AnkiStateStore struct{ DB Execer }

var _ storage.AnkiStateStore = (*AnkiStateStore)(nil)

func (s *AnkiStateStore) Deck(ctx context.Context, scope string, id int64) (*domain.AnkiDeck, error) {
	tenant, targs := s.DB.TenantFilter("", scope)
	var d domain.AnkiDeck
	var fuzz int
	var limit *int64
	err := s.DB.QueryRow(ctx, `SELECT id, name, COALESCE(description,''), source_type, source_id, COALESCE(source_command,''),
		request_retention, maximum_interval, `+s.DB.BoolAsInt("enable_fuzz")+`, session_limit,
		`+s.DB.TimestampText("created_at")+`, `+s.DB.TimestampText("updated_at")+`
		FROM anki_deck WHERE `+tenant+` AND id = ?`, append(targs, id)...).
		Scan(&d.ID, &d.Name, &d.Description, &d.SourceType, &d.SourceID, &d.SourceCommand,
			&d.RequestRetention, &d.MaximumInterval, &fuzz, &limit, &d.CreatedAt, &d.UpdatedAt)
	if errors.Is(err, ErrNoRows) {
		return nil, errf(s.DB, "anki deck", storage.ErrNotFound)
	}
	if err != nil {
		return nil, errf(s.DB, "anki deck", err)
	}
	d.EnableFuzz = fuzz != 0
	if limit != nil {
		v := int(*limit)
		d.SessionLimit = &v
	}
	return &d, nil
}

func (s *AnkiStateStore) PutDeck(ctx context.Context, scope string, d domain.AnkiDeck) (int64, error) {
	var limit any
	if d.SessionLimit != nil {
		limit = *d.SessionLimit
	}
	if d.ID == 0 {
		cols, args := s.DB.TenantColumns(scope)
		cols = append(cols, "name", "description", "source_type", "source_id", "source_command",
			"request_retention", "maximum_interval", "enable_fuzz", "session_limit")
		args = append(args, d.Name, d.Description, d.SourceType, d.SourceID, d.SourceCommand,
			d.RequestRetention, d.MaximumInterval, s.DB.BoolArg(d.EnableFuzz), limit)
		id, err := s.DB.Insert(ctx,
			`INSERT INTO anki_deck (`+strings.Join(cols, ", ")+`) VALUES (`+Placeholders(len(cols))+`)`, args...)
		if err != nil {
			return 0, errf(s.DB, "put anki deck", err)
		}
		return id, nil
	}
	tenant, targs := s.DB.TenantFilter("", scope)
	n, err := s.DB.Exec(ctx, `UPDATE anki_deck SET name = ?, description = ?, source_type = ?, source_id = ?, source_command = ?,
		request_retention = ?, maximum_interval = ?, enable_fuzz = ?, session_limit = ?, updated_at = `+s.DB.TimestampArg()+`
		WHERE id = ? AND `+tenant,
		append([]any{d.Name, d.Description, d.SourceType, d.SourceID, d.SourceCommand, d.RequestRetention,
			d.MaximumInterval, s.DB.BoolArg(d.EnableFuzz), limit, ankiNow(), d.ID}, targs...)...)
	if err != nil {
		return 0, errf(s.DB, "put anki deck", err)
	}
	if n == 0 {
		return 0, errf(s.DB, "put anki deck", storage.ErrNotFound)
	}
	return d.ID, nil
}

func (s *AnkiStateStore) cardCols() string {
	return `id, deck_id, kind, key, position_id, ` + s.DB.TimestampText("due") + `, stability, difficulty,
		elapsed_days, scheduled_days, reps, lapses, state, ` + s.DB.TimestampText("last_review") + `,
		` + s.DB.BoolAsInt("suspended") + `, ` + s.DB.TimestampText("buried_until")
}

func (s *AnkiStateStore) Card(ctx context.Context, scope string, id int64) (*domain.AnkiCard, error) {
	tenant, targs := s.DB.TenantFilter("", scope)
	var c domain.AnkiCard
	var positionID *int64
	var suspended int
	err := s.DB.QueryRow(ctx, `SELECT `+s.cardCols()+` FROM anki_card WHERE `+tenant+` AND id = ?`, append(targs, id)...).
		Scan(&c.ID, &c.DeckID, &c.Kind, &c.Key, &positionID, &c.Due, &c.Stability, &c.Difficulty,
			&c.ElapsedDays, &c.ScheduledDays, &c.Reps, &c.Lapses, &c.State, &c.LastReview, &suspended, &c.BuriedUntil)
	if errors.Is(err, ErrNoRows) {
		return nil, errf(s.DB, "anki card", storage.ErrNotFound)
	}
	if err != nil {
		return nil, errf(s.DB, "anki card", err)
	}
	if positionID != nil {
		c.PositionID = *positionID
	}
	c.Suspended = suspended != 0
	return &c, nil
}

func (s *AnkiStateStore) FindCard(ctx context.Context, scope string, deckID int64, kind, key string) (int64, bool, error) {
	tenant, targs := s.DB.TenantFilter("", scope)
	var id int64
	err := s.DB.QueryRow(ctx, `SELECT id FROM anki_card WHERE `+tenant+` AND deck_id = ? AND kind = ? AND key = ?`,
		append(targs, deckID, kind, key)...).Scan(&id)
	if errors.Is(err, ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, errf(s.DB, "find anki card", err)
	}
	return id, true, nil
}

// optionalTime binds an empty timestamp as NULL: a card never reviewed has no
// last review, and PostgreSQL has no empty timestamp.
func optionalTime(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func (s *AnkiStateStore) PutCard(ctx context.Context, scope string, c domain.AnkiCard) (int64, error) {
	var id int64
	err := s.DB.Transact(ctx, func(tx Execer) error {
		if c.ID != 0 {
			tenant, targs := tx.TenantFilter("", scope)
			n, err := tx.Exec(ctx, `UPDATE anki_card SET due = `+tx.TimestampArg()+`, stability = ?, difficulty = ?, elapsed_days = ?,
				scheduled_days = ?, reps = ?, lapses = ?, state = ?, last_review = `+tx.TimestampArg()+`, suspended = ?,
				buried_until = `+tx.TimestampArg()+` WHERE id = ? AND `+tenant,
				append([]any{optionalTime(c.Due), c.Stability, c.Difficulty, c.ElapsedDays, c.ScheduledDays, c.Reps, c.Lapses,
					c.State, optionalTime(c.LastReview), tx.BoolArg(c.Suspended), optionalTime(c.BuriedUntil), c.ID}, targs...)...)
			if err != nil {
				return err
			}
			if n == 0 {
				return storage.ErrNotFound
			}
			id = c.ID
			return nil
		}
		if err := RequireOwned(ctx, tx, scope, "anki_deck", c.DeckID); err != nil {
			return err
		}
		tenant, targs := tx.TenantFilter("", scope)
		var one int
		switch err := tx.QueryRow(ctx, `SELECT 1 FROM anki_card WHERE `+tenant+` AND deck_id = ? AND kind = ? AND key = ?`,
			append(targs, c.DeckID, c.Kind, c.Key)...).Scan(&one); {
		case err == nil:
			return storage.ErrConflict
		case !errors.Is(err, ErrNoRows):
			return err
		}
		var positionID any
		if c.Kind == domain.AnkiKindPosition {
			pid, err := strconv.ParseInt(c.Key, 10, 64)
			if err != nil {
				return fmt.Errorf("position card key %q: %w", c.Key, storage.ErrInvalid)
			}
			positionID = pid
		}
		cols, args := tx.TenantColumns(scope)
		cols = append(cols, "deck_id", "kind", "key", "position_id", "due", "stability", "difficulty", "elapsed_days",
			"scheduled_days", "reps", "lapses", "state", "last_review", "suspended", "buried_until")
		args = append(args, c.DeckID, c.Kind, c.Key, positionID, optionalTime(c.Due), c.Stability, c.Difficulty, c.ElapsedDays,
			c.ScheduledDays, c.Reps, c.Lapses, c.State, optionalTime(c.LastReview), tx.BoolArg(c.Suspended), optionalTime(c.BuriedUntil))
		ph := Placeholders(len(cols))
		// The timestamp columns bind through the backend's own cast.
		parts := strings.Split(ph, ",")
		for i, col := range cols {
			if col == "due" || col == "last_review" || col == "buried_until" {
				parts[i] = tx.TimestampArg()
			}
		}
		newID, err := tx.Insert(ctx, `INSERT INTO anki_card (`+strings.Join(cols, ", ")+`) VALUES (`+strings.Join(parts, ", ")+`)`, args...)
		id = newID
		return err
	})
	if err != nil {
		return 0, errf(s.DB, "put anki card", s.DB.Referenced(err))
	}
	return id, nil
}
