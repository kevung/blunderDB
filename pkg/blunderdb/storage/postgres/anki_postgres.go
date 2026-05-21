package postgres

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	fsrs "github.com/open-spaced-repetition/go-fsrs/v3"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type ankiStore struct{ db execer }

var _ storage.AnkiStore = (*ankiStore)(nil)

// ankiDeckSelectExpr reads a domain.AnkiDeck; the three correlated subqueries
// supply the card counters. The due-count cutoff is now() rather than a bound
// parameter — PostgreSQL evaluates it server-side.
const ankiDeckSelectExpr = `ad.id, ad.name, COALESCE(ad.description,''),
	ad.source_type, ad.source_id, COALESCE(ad.source_command,''),
	ad.request_retention, ad.maximum_interval, ad.enable_fuzz,
	ad.created_at, ad.updated_at,
	(SELECT COUNT(*) FROM anki_card ac WHERE ac.deck_id = ad.id),
	(SELECT COUNT(*) FROM anki_card ac WHERE ac.deck_id = ad.id AND ac.due <= now()),
	(SELECT COUNT(*) FROM anki_card ac WHERE ac.deck_id = ad.id AND ac.state = 0)`

func scanAnkiDeck(sc scanner) (domain.AnkiDeck, error) {
	var d domain.AnkiDeck
	var createdAt, updatedAt time.Time
	if err := sc.Scan(&d.ID, &d.Name, &d.Description,
		&d.SourceType, &d.SourceID, &d.SourceCommand,
		&d.RequestRetention, &d.MaximumInterval, &d.EnableFuzz,
		&createdAt, &updatedAt,
		&d.CardCount, &d.DueCount, &d.NewCount); err != nil {
		return domain.AnkiDeck{}, err
	}
	d.CreatedAt = tsTime(createdAt)
	d.UpdatedAt = tsTime(updatedAt)
	return d, nil
}

// CreateDeck stores a new spaced-repetition deck and returns its id. The FSRS
// scheduling parameters fall back to the column defaults.
func (s *ankiStore) CreateDeck(ctx context.Context, scope string, name, description, sourceType string, sourceID int64, sourceCommand string) (int64, error) {
	var id int64
	if err := s.db.QueryRow(ctx,
		`INSERT INTO anki_deck (tenant_id, name, description, source_type, source_id, source_command)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		tenantID(scope), name, description, sourceType, sourceID, sourceCommand).Scan(&id); err != nil {
		return 0, fmt.Errorf("postgres: create anki deck: %w", err)
	}
	return id, nil
}

// ListDecks streams every deck with its card counters, oldest first.
func (s *ankiStore) ListDecks(ctx context.Context, scope string) iter.Seq2[*domain.AnkiDeck, error] {
	return func(yield func(*domain.AnkiDeck, error) bool) {
		rows, err := s.db.Query(ctx,
			`SELECT `+ankiDeckSelectExpr+` FROM anki_deck ad
			 WHERE ad.tenant_id = $1 ORDER BY ad.id ASC`, tenantID(scope))
		if err != nil {
			yield(nil, fmt.Errorf("postgres: list anki decks: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			d, err := scanAnkiDeck(rows)
			if err != nil {
				yield(nil, fmt.Errorf("postgres: list anki decks: %w", err))
				return
			}
			if !yield(&d, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("postgres: list anki decks: %w", err))
		}
	}
}

// UpdateDeck changes a deck's name and description.
func (s *ankiStore) UpdateDeck(ctx context.Context, scope string, id int64, name, description string) error {
	if _, err := s.db.Exec(ctx,
		`UPDATE anki_deck SET name = $1, description = $2, updated_at = now()
		 WHERE id = $3 AND tenant_id = $4`, name, description, id, tenantID(scope)); err != nil {
		return fmt.Errorf("postgres: update anki deck %d: %w", id, err)
	}
	return nil
}

// UpdateDeckParams changes a deck's FSRS scheduling parameters.
func (s *ankiStore) UpdateDeckParams(ctx context.Context, scope string, id int64, requestRetention, maximumInterval float64, enableFuzz bool) error {
	if _, err := s.db.Exec(ctx,
		`UPDATE anki_deck SET request_retention = $1, maximum_interval = $2, enable_fuzz = $3,
		 updated_at = now() WHERE id = $4 AND tenant_id = $5`,
		requestRetention, maximumInterval, enableFuzz, id, tenantID(scope)); err != nil {
		return fmt.Errorf("postgres: update anki deck %d params: %w", id, err)
	}
	return nil
}

// DeleteDeck removes a deck; its cards cascade off the anki_card foreign key.
func (s *ankiStore) DeleteDeck(ctx context.Context, scope string, id int64) error {
	if _, err := s.db.Exec(ctx,
		`DELETE FROM anki_deck WHERE id = $1 AND tenant_id = $2`,
		id, tenantID(scope)); err != nil {
		return fmt.Errorf("postgres: delete anki deck %d: %w", id, err)
	}
	return nil
}

// ResetDeck clears the FSRS state of every card in a deck back to new.
func (s *ankiStore) ResetDeck(ctx context.Context, scope string, deckID int64) error {
	if _, err := s.db.Exec(ctx,
		`UPDATE anki_card SET due = now(), stability = 0, difficulty = 0,
		 elapsed_days = 0, scheduled_days = 0, reps = 0, lapses = 0, state = 0, last_review = NULL
		 WHERE deck_id = $1 AND tenant_id = $2`, deckID, tenantID(scope)); err != nil {
		return fmt.Errorf("postgres: reset anki deck %d: %w", deckID, err)
	}
	return nil
}

// Sync reconciles a deck's cards with its source: the positions of a
// collection, or the position ids listed in a search deck's source command.
func (s *ankiStore) Sync(ctx context.Context, scope string, deckID int64) error {
	tenant := tenantID(scope)
	var sourceType, sourceCommand string
	var sourceID int64
	err := s.db.QueryRow(ctx,
		`SELECT source_type, source_id, COALESCE(source_command,'') FROM anki_deck
		 WHERE id = $1 AND tenant_id = $2`, deckID, tenant).
		Scan(&sourceType, &sourceID, &sourceCommand)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("postgres: sync anki deck %d: %w", deckID, storage.ErrNotFound)
	}
	if err != nil {
		return fmt.Errorf("postgres: sync anki deck %d: %w", deckID, err)
	}

	var positionIDs []int64
	switch sourceType {
	case domain.AnkiSourceCollection:
		rows, err := s.db.Query(ctx,
			`SELECT position_id FROM collection_position
			 WHERE collection_id = $1 AND tenant_id = $2 ORDER BY sort_order ASC`,
			sourceID, tenant)
		if err != nil {
			return fmt.Errorf("postgres: sync anki deck %d: %w", deckID, err)
		}
		for rows.Next() {
			var pid int64
			if err := rows.Scan(&pid); err != nil {
				rows.Close()
				return fmt.Errorf("postgres: sync anki deck %d: %w", deckID, err)
			}
			positionIDs = append(positionIDs, pid)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return fmt.Errorf("postgres: sync anki deck %d: %w", deckID, err)
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
	tenant := tenantID(scope)
	err := withTx(ctx, s.db, func(tx execer) error {
		for _, pid := range positionIDs {
			if _, err := tx.Exec(ctx,
				`INSERT INTO anki_card (tenant_id, deck_id, position_id, due, state)
				 VALUES ($1,$2,$3,now(),0) ON CONFLICT (deck_id, position_id) DO NOTHING`,
				tenant, deckID, pid); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(ctx,
			`UPDATE anki_deck SET updated_at = now() WHERE id = $1 AND tenant_id = $2`,
			deckID, tenant); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("postgres: sync anki deck %d: %w", deckID, err)
	}
	return nil
}

// DeckPositions streams the positions linked to a deck's cards, ordered by id.
func (s *ankiStore) DeckPositions(ctx context.Context, scope string, deckID int64) iter.Seq2[*domain.Position, error] {
	return func(yield func(*domain.Position, error) bool) {
		rows, err := s.db.Query(ctx,
			`SELECT `+collectionPositionCols+` FROM anki_card ac
			 INNER JOIN position p ON p.id = ac.position_id
			 WHERE ac.deck_id = $1 AND ac.tenant_id = $2 ORDER BY p.id ASC`,
			deckID, tenantID(scope))
		if err != nil {
			yield(nil, fmt.Errorf("postgres: anki deck positions: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			p, err := scanPosition(rows)
			if err != nil {
				yield(nil, fmt.Errorf("postgres: anki deck positions: %w", err))
				return
			}
			if !yield(&p, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("postgres: anki deck positions: %w", err))
		}
	}
}

// DeckStats returns the review counters for a deck.
func (s *ankiStore) DeckStats(ctx context.Context, scope string, deckID int64) (*domain.AnkiDeckStats, error) {
	var st domain.AnkiDeckStats
	err := s.db.QueryRow(ctx,
		`SELECT
			COALESCE(SUM(CASE WHEN state = 0 THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN state = 1 OR state = 3 THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN state = 2 AND due <= now() THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN due <= now() THEN 1 ELSE 0 END), 0),
			COUNT(*)
		 FROM anki_card WHERE deck_id = $1 AND tenant_id = $2`,
		deckID, tenantID(scope)).
		Scan(&st.NewCount, &st.LearningCount, &st.ReviewCount, &st.DueCount, &st.TotalCount)
	if err != nil {
		return nil, fmt.Errorf("postgres: anki deck %d stats: %w", deckID, err)
	}
	return &st, nil
}

// ankiCardCols reads a domain.AnkiCard; scanAnkiCard formats the two timestamp
// columns into the struct's string fields.
const ankiCardCols = `id, deck_id, position_id, due, stability, difficulty,
	elapsed_days, scheduled_days, reps, lapses, state, last_review`

func scanAnkiCard(sc scanner) (domain.AnkiCard, error) {
	var c domain.AnkiCard
	var due time.Time
	var lastReview *time.Time
	if err := sc.Scan(&c.ID, &c.DeckID, &c.PositionID,
		&due, &c.Stability, &c.Difficulty,
		&c.ElapsedDays, &c.ScheduledDays, &c.Reps, &c.Lapses, &c.State,
		&lastReview); err != nil {
		return domain.AnkiCard{}, err
	}
	c.Due = tsTime(due)
	if lastReview != nil {
		c.LastReview = tsTime(*lastReview)
	}
	return c, nil
}

// nextDueCardSQL orders due cards so learning/relearning come first, then
// review, then new; ties break on the due date.
const nextDueCardSQL = `SELECT ` + ankiCardCols + ` FROM anki_card
	WHERE deck_id = $1 AND tenant_id = $2 AND due <= now()
	ORDER BY
		CASE WHEN state = 1 OR state = 3 THEN 0
		     WHEN state = 2 THEN 1
		     ELSE 2 END,
		due ASC
	LIMIT 1`

// nextDueCard returns the highest-priority card due in a deck, or ErrNotFound.
func (s *ankiStore) nextDueCard(ctx context.Context, tenant, deckID int64) (domain.AnkiCard, error) {
	row := s.db.QueryRow(ctx, nextDueCardSQL, deckID, tenant)
	c, err := scanAnkiCard(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.AnkiCard{}, storage.ErrNotFound
	}
	if err != nil {
		return domain.AnkiCard{}, err
	}
	return c, nil
}

// loadPosition reads the position with the given id.
func (s *ankiStore) loadPosition(ctx context.Context, tenant, id int64) (domain.Position, error) {
	row := s.db.QueryRow(ctx,
		`SELECT `+positionSelectCols+` FROM position WHERE id = $1 AND tenant_id = $2`, id, tenant)
	return scanPosition(row)
}

// NextCard returns the next card due for review in a deck, or ErrNotFound.
func (s *ankiStore) NextCard(ctx context.Context, scope string, deckID int64) (*domain.AnkiReviewCard, error) {
	tenant := tenantID(scope)
	card, err := s.nextDueCard(ctx, tenant, deckID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("postgres: next anki card of deck %d: %w", deckID, err)
	}
	pos, err := s.loadPosition(ctx, tenant, card.PositionID)
	if err != nil {
		return nil, fmt.Errorf("postgres: next anki card of deck %d: %w", deckID, err)
	}
	return &domain.AnkiReviewCard{Card: card, Position: pos}, nil
}

// ReviewCard records a review rating against a card, advances its FSRS
// scheduling state, and returns the next card still due in the same deck (nil
// when none remain).
func (s *ankiStore) ReviewCard(ctx context.Context, scope string, cardID int64, rating int) (*domain.AnkiReviewCard, error) {
	tenant := tenantID(scope)

	var (
		card       domain.AnkiCard
		due        time.Time
		lastReview *time.Time
	)
	err := s.db.QueryRow(ctx,
		`SELECT id, deck_id, position_id, due, stability, difficulty,
		 elapsed_days, scheduled_days, reps, lapses, state, last_review
		 FROM anki_card WHERE id = $1 AND tenant_id = $2`, cardID, tenant).
		Scan(&card.ID, &card.DeckID, &card.PositionID, &due, &card.Stability, &card.Difficulty,
			&card.ElapsedDays, &card.ScheduledDays, &card.Reps, &card.Lapses, &card.State, &lastReview)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("postgres: review anki card %d: %w", cardID, storage.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: review anki card %d: %w", cardID, err)
	}

	var (
		requestRetention float64
		maximumInterval  float64
		enableFuzz       bool
	)
	err = s.db.QueryRow(ctx,
		`SELECT request_retention, maximum_interval, enable_fuzz FROM anki_deck
		 WHERE id = $1 AND tenant_id = $2`, card.DeckID, tenant).
		Scan(&requestRetention, &maximumInterval, &enableFuzz)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("postgres: review anki card %d: deck: %w", cardID, storage.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: review anki card %d: %w", cardID, err)
	}

	now := time.Now().UTC()
	fsrsCard := fsrs.Card{
		Due:           due,
		Stability:     card.Stability,
		Difficulty:    card.Difficulty,
		ElapsedDays:   uint64(card.ElapsedDays),
		ScheduledDays: uint64(card.ScheduledDays),
		Reps:          uint64(card.Reps),
		Lapses:        uint64(card.Lapses),
		State:         fsrs.State(card.State),
	}
	if lastReview != nil {
		fsrsCard.LastReview = *lastReview
	}

	params := fsrs.DefaultParam()
	params.RequestRetention = requestRetention
	params.MaximumInterval = maximumInterval
	params.EnableFuzz = enableFuzz
	next := fsrs.NewFSRS(params).Next(fsrsCard, now, fsrs.Rating(rating)).Card

	if _, err := s.db.Exec(ctx,
		`UPDATE anki_card SET due = $1, stability = $2, difficulty = $3,
		 elapsed_days = $4, scheduled_days = $5, reps = $6, lapses = $7, state = $8, last_review = $9
		 WHERE id = $10 AND tenant_id = $11`,
		next.Due.UTC(), next.Stability, next.Difficulty,
		int64(next.ElapsedDays), int64(next.ScheduledDays), int64(next.Reps), int64(next.Lapses),
		int64(next.State), now, cardID, tenant); err != nil {
		return nil, fmt.Errorf("postgres: review anki card %d: %w", cardID, err)
	}

	nextCard, err := s.nextDueCard(ctx, tenant, card.DeckID)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: review anki card %d: %w", cardID, err)
	}
	pos, err := s.loadPosition(ctx, tenant, nextCard.PositionID)
	if err != nil {
		return nil, fmt.Errorf("postgres: review anki card %d: %w", cardID, err)
	}
	return &domain.AnkiReviewCard{Card: nextCard, Position: pos}, nil
}
