package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"iter"
	"strconv"
	"strings"
	"time"

	fsrs "github.com/open-spaced-repetition/go-fsrs/v3"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type ankiStore struct{ db execer }

var _ storage.AnkiStore = (*ankiStore)(nil)

// ankiTimeLayout is the textual datetime format used for all anki_card
// timestamp columns.
const ankiTimeLayout = "2006-01-02 15:04:05"

func ankiNow() string { return time.Now().UTC().Format(ankiTimeLayout) }

// ankiDeckSelectCols reads a domain.AnkiDeck; the three correlated subqueries
// supply the card counters. The single placeholder is the "now" cutoff used by
// the due-count subquery.
const ankiDeckSelectCols = `ad.id, ad.name, COALESCE(ad.description,''),
	ad.source_type, ad.source_id, COALESCE(ad.source_command,''),
	ad.request_retention, ad.maximum_interval, ad.enable_fuzz,
	COALESCE(ad.created_at,''), COALESCE(ad.updated_at,''),
	(SELECT COUNT(*) FROM anki_card ac WHERE ac.deck_id = ad.id),
	(SELECT COUNT(*) FROM anki_card ac WHERE ac.deck_id = ad.id AND ac.due <= ?),
	(SELECT COUNT(*) FROM anki_card ac WHERE ac.deck_id = ad.id AND ac.state = 0)`

func scanAnkiDeck(sc interface{ Scan(...any) error }) (domain.AnkiDeck, error) {
	var d domain.AnkiDeck
	var enableFuzz int
	if err := sc.Scan(&d.ID, &d.Name, &d.Description,
		&d.SourceType, &d.SourceID, &d.SourceCommand,
		&d.RequestRetention, &d.MaximumInterval, &enableFuzz,
		&d.CreatedAt, &d.UpdatedAt,
		&d.CardCount, &d.DueCount, &d.NewCount); err != nil {
		return domain.AnkiDeck{}, err
	}
	d.EnableFuzz = enableFuzz != 0
	return d, nil
}

// CreateDeck stores a new spaced-repetition deck and returns its id.
func (s *ankiStore) CreateDeck(ctx context.Context, scope string, name, description, sourceType string, sourceID int64, sourceCommand string) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO anki_deck (name, description, source_type, source_id, source_command)
		 VALUES (?,?,?,?,?)`,
		name, description, sourceType, sourceID, sourceCommand)
	if err != nil {
		return 0, fmt.Errorf("sqlite: create anki deck: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("sqlite: create anki deck id: %w", err)
	}
	return id, nil
}

// ListDecks streams every deck with its card counters, oldest first.
func (s *ankiStore) ListDecks(ctx context.Context, scope string) iter.Seq2[*domain.AnkiDeck, error] {
	return func(yield func(*domain.AnkiDeck, error) bool) {
		rows, err := s.db.QueryContext(ctx,
			`SELECT `+ankiDeckSelectCols+` FROM anki_deck ad ORDER BY ad.id ASC`, ankiNow())
		if err != nil {
			yield(nil, fmt.Errorf("sqlite: list anki decks: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			d, err := scanAnkiDeck(rows)
			if err != nil {
				yield(nil, fmt.Errorf("sqlite: list anki decks: %w", err))
				return
			}
			if !yield(&d, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("sqlite: list anki decks: %w", err))
		}
	}
}

// UpdateDeck changes a deck's name and description.
func (s *ankiStore) UpdateDeck(ctx context.Context, scope string, id int64, name, description string) error {
	if _, err := s.db.ExecContext(ctx,
		`UPDATE anki_deck SET name = ?, description = ?, updated_at = datetime('now')
		 WHERE id = ?`, name, description, id); err != nil {
		return fmt.Errorf("sqlite: update anki deck %d: %w", id, err)
	}
	return nil
}

// UpdateDeckParams changes a deck's FSRS scheduling parameters.
func (s *ankiStore) UpdateDeckParams(ctx context.Context, scope string, id int64, requestRetention, maximumInterval float64, enableFuzz bool) error {
	if _, err := s.db.ExecContext(ctx,
		`UPDATE anki_deck SET request_retention = ?, maximum_interval = ?, enable_fuzz = ?,
		 updated_at = datetime('now') WHERE id = ?`,
		requestRetention, maximumInterval, boolToInt(enableFuzz), id); err != nil {
		return fmt.Errorf("sqlite: update anki deck %d params: %w", id, err)
	}
	return nil
}

// DeleteDeck removes a deck; its cards cascade off the anki_card foreign key.
func (s *ankiStore) DeleteDeck(ctx context.Context, scope string, id int64) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM anki_deck WHERE id = ?`, id); err != nil {
		return fmt.Errorf("sqlite: delete anki deck %d: %w", id, err)
	}
	return nil
}

// ResetDeck clears the FSRS state of every card in a deck back to new.
func (s *ankiStore) ResetDeck(ctx context.Context, scope string, deckID int64) error {
	if _, err := s.db.ExecContext(ctx,
		`UPDATE anki_card SET due = ?, stability = 0, difficulty = 0,
		 elapsed_days = 0, scheduled_days = 0, reps = 0, lapses = 0, state = 0, last_review = ''
		 WHERE deck_id = ?`, ankiNow(), deckID); err != nil {
		return fmt.Errorf("sqlite: reset anki deck %d: %w", deckID, err)
	}
	return nil
}

// Sync reconciles a deck's cards with its source: the positions of a
// collection, or the position ids listed in a search deck's source command.
func (s *ankiStore) Sync(ctx context.Context, scope string, deckID int64) error {
	var sourceType, sourceCommand string
	var sourceID int64
	err := s.db.QueryRowContext(ctx,
		`SELECT source_type, source_id, COALESCE(source_command,'') FROM anki_deck WHERE id = ?`,
		deckID).Scan(&sourceType, &sourceID, &sourceCommand)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("sqlite: sync anki deck %d: %w", deckID, storage.ErrNotFound)
	}
	if err != nil {
		return fmt.Errorf("sqlite: sync anki deck %d: %w", deckID, err)
	}

	var positionIDs []int64
	switch sourceType {
	case domain.AnkiSourceCollection:
		rows, err := s.db.QueryContext(ctx,
			`SELECT position_id FROM collection_position WHERE collection_id = ? ORDER BY sort_order ASC`,
			sourceID)
		if err != nil {
			return fmt.Errorf("sqlite: sync anki deck %d: %w", deckID, err)
		}
		for rows.Next() {
			var pid int64
			if err := rows.Scan(&pid); err != nil {
				rows.Close()
				return fmt.Errorf("sqlite: sync anki deck %d: %w", deckID, err)
			}
			positionIDs = append(positionIDs, pid)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return fmt.Errorf("sqlite: sync anki deck %d: %w", deckID, err)
		}
	case domain.AnkiSourceSearch:
		for tok := range strings.SplitSeq(sourceCommand, ",") {
			if pid, err := strconv.ParseInt(strings.TrimSpace(tok), 10, 64); err == nil {
				positionIDs = append(positionIDs, pid)
			}
		}
	}
	return s.SyncWithPositions(ctx, scope, deckID, positionIDs)
}

// SyncWithPositions adds a card for every position not yet in the deck and
// touches the deck's updated_at. Existing cards keep their scheduling state.
func (s *ankiStore) SyncWithPositions(ctx context.Context, scope string, deckID int64, positionIDs []int64) error {
	err := withTx(ctx, s.db, func(tx execer) error {
		now := ankiNow()
		for _, pid := range positionIDs {
			if _, err := tx.ExecContext(ctx,
				`INSERT OR IGNORE INTO anki_card (deck_id, position_id, due, state)
				 VALUES (?,?,?,0)`, deckID, pid, now); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE anki_deck SET updated_at = datetime('now') WHERE id = ?`, deckID); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("sqlite: sync anki deck %d: %w", deckID, err)
	}
	return nil
}

// DeckPositions streams the positions linked to a deck's cards, ordered by id.
func (s *ankiStore) DeckPositions(ctx context.Context, scope string, deckID int64) iter.Seq2[*domain.Position, error] {
	return func(yield func(*domain.Position, error) bool) {
		rows, err := s.db.QueryContext(ctx,
			`SELECT `+collectionPositionCols+` FROM anki_card ac
			 INNER JOIN position p ON p.id = ac.position_id
			 WHERE ac.deck_id = ? ORDER BY p.id ASC`, deckID)
		if err != nil {
			yield(nil, fmt.Errorf("sqlite: anki deck positions: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			p, err := scanPosition(rows)
			if err != nil {
				yield(nil, fmt.Errorf("sqlite: anki deck positions: %w", err))
				return
			}
			if !yield(&p, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("sqlite: anki deck positions: %w", err))
		}
	}
}

// DeckStats returns the review counters for a deck.
func (s *ankiStore) DeckStats(ctx context.Context, scope string, deckID int64) (*domain.AnkiDeckStats, error) {
	now := ankiNow()
	var st domain.AnkiDeckStats
	err := s.db.QueryRowContext(ctx,
		`SELECT
			COALESCE(SUM(CASE WHEN state = 0 THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN state = 1 OR state = 3 THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN state = 2 AND due <= ? THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN due <= ? THEN 1 ELSE 0 END), 0),
			COUNT(*)
		 FROM anki_card WHERE deck_id = ?`,
		now, now, deckID).Scan(&st.NewCount, &st.LearningCount, &st.ReviewCount, &st.DueCount, &st.TotalCount)
	if err != nil {
		return nil, fmt.Errorf("sqlite: anki deck %d stats: %w", deckID, err)
	}
	return &st, nil
}

// ankiCardCols reads a domain.AnkiCard.
const ankiCardCols = `id, deck_id, position_id, COALESCE(due,''), stability, difficulty,
	elapsed_days, scheduled_days, reps, lapses, state, COALESCE(last_review,'')`

func scanAnkiCard(sc interface{ Scan(...any) error }) (domain.AnkiCard, error) {
	var c domain.AnkiCard
	if err := sc.Scan(&c.ID, &c.DeckID, &c.PositionID,
		&c.Due, &c.Stability, &c.Difficulty,
		&c.ElapsedDays, &c.ScheduledDays, &c.Reps, &c.Lapses, &c.State,
		&c.LastReview); err != nil {
		return domain.AnkiCard{}, err
	}
	return c, nil
}

// nextDueCardSQL orders due cards so learning/relearning come first, then
// review, then new; ties break on the due date.
const nextDueCardSQL = `SELECT ` + ankiCardCols + ` FROM anki_card
	WHERE deck_id = ? AND due <= ?
	ORDER BY
		CASE WHEN state = 1 OR state = 3 THEN 0
		     WHEN state = 2 THEN 1
		     ELSE 2 END,
		due ASC
	LIMIT 1`

// nextDueCard returns the highest-priority card due in a deck, or ErrNotFound.
func (s *ankiStore) nextDueCard(ctx context.Context, deckID int64) (domain.AnkiCard, error) {
	row := s.db.QueryRowContext(ctx, nextDueCardSQL, deckID, ankiNow())
	c, err := scanAnkiCard(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.AnkiCard{}, storage.ErrNotFound
	}
	if err != nil {
		return domain.AnkiCard{}, err
	}
	return c, nil
}

// loadPosition reads the position with the given id.
func (s *ankiStore) loadPosition(ctx context.Context, id int64) (domain.Position, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+positionCols+` FROM position WHERE id = ?`, id)
	return scanPosition(row)
}

// NextCard returns the next card due for review in a deck, or ErrNotFound.
func (s *ankiStore) NextCard(ctx context.Context, scope string, deckID int64) (*domain.AnkiReviewCard, error) {
	card, err := s.nextDueCard(ctx, deckID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("sqlite: next anki card of deck %d: %w", deckID, err)
	}
	pos, err := s.loadPosition(ctx, card.PositionID)
	if err != nil {
		return nil, fmt.Errorf("sqlite: next anki card of deck %d: %w", deckID, err)
	}
	return &domain.AnkiReviewCard{Card: card, Position: pos}, nil
}

// ReviewCard records a review rating against a card, advances its FSRS
// scheduling state, and returns the next card still due in the same deck (nil
// when none remain).
func (s *ankiStore) ReviewCard(ctx context.Context, scope string, cardID int64, rating int) (*domain.AnkiReviewCard, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+ankiCardCols+` FROM anki_card WHERE id = ?`, cardID)
	card, err := scanAnkiCard(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("sqlite: review anki card %d: %w", cardID, storage.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("sqlite: review anki card %d: %w", cardID, err)
	}

	var (
		deckID           int64
		requestRetention float64
		maximumInterval  float64
		enableFuzz       int
	)
	err = s.db.QueryRowContext(ctx,
		`SELECT id, request_retention, maximum_interval, enable_fuzz FROM anki_deck WHERE id = ?`,
		card.DeckID).Scan(&deckID, &requestRetention, &maximumInterval, &enableFuzz)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("sqlite: review anki card %d: deck: %w", cardID, storage.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("sqlite: review anki card %d: %w", cardID, err)
	}

	now := time.Now().UTC()
	fsrsCard := fsrs.Card{
		Stability:     card.Stability,
		Difficulty:    card.Difficulty,
		ElapsedDays:   uint64(card.ElapsedDays),
		ScheduledDays: uint64(card.ScheduledDays),
		Reps:          uint64(card.Reps),
		Lapses:        uint64(card.Lapses),
		State:         fsrs.State(card.State),
	}
	if t, err := time.Parse(ankiTimeLayout, card.Due); err == nil {
		fsrsCard.Due = t
	}
	if t, err := time.Parse(ankiTimeLayout, card.LastReview); err == nil {
		fsrsCard.LastReview = t
	}

	params := fsrs.DefaultParam()
	params.RequestRetention = requestRetention
	params.MaximumInterval = maximumInterval
	params.EnableFuzz = enableFuzz != 0
	next := fsrs.NewFSRS(params).Next(fsrsCard, now, fsrs.Rating(rating)).Card

	if _, err := s.db.ExecContext(ctx,
		`UPDATE anki_card SET due = ?, stability = ?, difficulty = ?,
		 elapsed_days = ?, scheduled_days = ?, reps = ?, lapses = ?, state = ?, last_review = ?
		 WHERE id = ?`,
		next.Due.UTC().Format(ankiTimeLayout), next.Stability, next.Difficulty,
		next.ElapsedDays, next.ScheduledDays, next.Reps, next.Lapses, int(next.State),
		now.Format(ankiTimeLayout), cardID); err != nil {
		return nil, fmt.Errorf("sqlite: review anki card %d: %w", cardID, err)
	}

	nextCard, err := s.nextDueCard(ctx, card.DeckID)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("sqlite: review anki card %d: %w", cardID, err)
	}
	pos, err := s.loadPosition(ctx, nextCard.PositionID)
	if err != nil {
		return nil, fmt.Errorf("sqlite: review anki card %d: %w", cardID, err)
	}
	return &domain.AnkiReviewCard{Card: nextCard, Position: pos}, nil
}
