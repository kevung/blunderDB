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
		 elapsed_days = 0, scheduled_days = 0, reps = 0, lapses = 0, state = 0, last_review = NULL,
		 suspended = FALSE, buried_until = NULL
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
	// Queue counters exclude suspended/buried cards; TotalCount counts the
	// whole deck regardless of availability.
	err := s.db.QueryRow(ctx,
		`SELECT
			COALESCE(SUM(CASE WHEN state = 0 AND `+ankiAvailableSQL+` THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN (state = 1 OR state = 3) AND `+ankiAvailableSQL+` THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN state = 2 AND due <= now() AND `+ankiAvailableSQL+` THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN due <= now() AND `+ankiAvailableSQL+` THEN 1 ELSE 0 END), 0),
			COUNT(*)
		 FROM anki_card WHERE deck_id = $1 AND tenant_id = $2`,
		deckID, tenantID(scope)).
		Scan(&st.NewCount, &st.LearningCount, &st.ReviewCount, &st.DueCount, &st.TotalCount)
	if err != nil {
		return nil, fmt.Errorf("postgres: anki deck %d stats: %w", deckID, err)
	}
	return &st, nil
}

// ankiForecastDays clamps a requested forecast horizon into a sane range:
// non-positive falls back to 30 days, and the horizon is capped at one year.
func ankiForecastDays(days int) int {
	switch {
	case days <= 0:
		return 30
	case days > 365:
		return 365
	}
	return days
}

// Forecast projects how many cards come due over the next `days` calendar days,
// offset 0 absorbing every overdue card. deckID 0 spans the whole tenant.
func (s *ankiStore) Forecast(ctx context.Context, scope string, deckID int64, days int) ([]domain.AnkiForecastDay, error) {
	tenant := tenantID(scope)
	days = ankiForecastDays(days)

	query := `SELECT GREATEST(0, (CAST(due AS DATE) - CAST(now() AS DATE))) AS day_offset, COUNT(*)
		FROM anki_card
		WHERE tenant_id = $1 AND suspended = FALSE AND CAST(due AS DATE) < CAST(now() AS DATE) + $2::int`
	args := []any{tenant, days}
	if deckID != 0 {
		query += ` AND deck_id = $3`
		args = append(args, deckID)
	}
	query += ` GROUP BY day_offset`

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("postgres: anki forecast: %w", err)
	}
	defer rows.Close()
	counts := make(map[int]int, days)
	for rows.Next() {
		var off, n int
		if err := rows.Scan(&off, &n); err != nil {
			return nil, fmt.Errorf("postgres: anki forecast: %w", err)
		}
		counts[off] = n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: anki forecast: %w", err)
	}
	return storage.BuildForecast(time.Now().UTC(), days, counts), nil
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

// ankiAvailableSQL is the predicate that excludes suspended and still-buried
// cards from the review queue.
const ankiAvailableSQL = `suspended = FALSE AND (buried_until IS NULL OR buried_until <= now())`

// nextDueCardSQL orders due cards so learning/relearning come first, then
// review, then new; ties break on the due date.
const nextDueCardSQL = `SELECT ` + ankiCardCols + ` FROM anki_card
	WHERE deck_id = $1 AND tenant_id = $2 AND due <= now() AND ` + ankiAvailableSQL + `
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
	info := fsrs.NewFSRS(params).Next(fsrsCard, now, fsrs.Rating(rating))
	next := info.Card

	if _, err := s.db.Exec(ctx,
		`UPDATE anki_card SET due = $1, stability = $2, difficulty = $3,
		 elapsed_days = $4, scheduled_days = $5, reps = $6, lapses = $7, state = $8, last_review = $9
		 WHERE id = $10 AND tenant_id = $11`,
		next.Due.UTC(), next.Stability, next.Difficulty,
		int64(next.ElapsedDays), int64(next.ScheduledDays), int64(next.Reps), int64(next.Lapses),
		int64(next.State), now, cardID, tenant); err != nil {
		return nil, fmt.Errorf("postgres: review anki card %d: %w", cardID, err)
	}

	// Append the review to the immutable log. The recorded state is the one the
	// card was in *before* this review (info.ReviewLog.State); stability and
	// difficulty are the post-review values.
	if _, err := s.db.Exec(ctx,
		`INSERT INTO anki_review_log
		 (tenant_id, card_id, deck_id, position_id, rating, state,
		  stability, difficulty, elapsed_days, scheduled_days, reviewed_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		tenant, cardID, card.DeckID, card.PositionID, rating, int64(info.ReviewLog.State),
		next.Stability, next.Difficulty,
		int64(info.ReviewLog.ElapsedDays), int64(next.ScheduledDays), now); err != nil {
		return nil, fmt.Errorf("postgres: review anki card %d: log: %w", cardID, err)
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

// SetCardSuspended suspends or unsuspends a card.
func (s *ankiStore) SetCardSuspended(ctx context.Context, scope string, cardID int64, suspended bool) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE anki_card SET suspended = $1 WHERE id = $2 AND tenant_id = $3`,
		suspended, cardID, tenantID(scope))
	if err != nil {
		return fmt.Errorf("postgres: suspend anki card %d: %w", cardID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("postgres: suspend anki card %d: %w", cardID, storage.ErrNotFound)
	}
	return nil
}

// BuryCard hides a card until the start of the next day (UTC).
func (s *ankiStore) BuryCard(ctx context.Context, scope string, cardID int64) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE anki_card SET buried_until = (CAST(now() AS DATE) + 1)
		 WHERE id = $1 AND tenant_id = $2`, cardID, tenantID(scope))
	if err != nil {
		return fmt.Errorf("postgres: bury anki card %d: %w", cardID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("postgres: bury anki card %d: %w", cardID, storage.ErrNotFound)
	}
	return nil
}

// RemoveCard deletes a single card from its deck.
func (s *ankiStore) RemoveCard(ctx context.Context, scope string, cardID int64) error {
	tag, err := s.db.Exec(ctx,
		`DELETE FROM anki_card WHERE id = $1 AND tenant_id = $2`, cardID, tenantID(scope))
	if err != nil {
		return fmt.Errorf("postgres: remove anki card %d: %w", cardID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("postgres: remove anki card %d: %w", cardID, storage.ErrNotFound)
	}
	return nil
}

// reviewLogCols reads a domain.AnkiReviewLog; scanReviewLog formats the
// timestamp column into the struct's string field.
const reviewLogCols = `id, card_id, deck_id, position_id, rating, state,
	stability, difficulty, elapsed_days, scheduled_days, reviewed_at`

func scanReviewLog(sc scanner) (domain.AnkiReviewLog, error) {
	var l domain.AnkiReviewLog
	var reviewedAt time.Time
	if err := sc.Scan(&l.ID, &l.CardID, &l.DeckID, &l.PositionID, &l.Rating, &l.State,
		&l.Stability, &l.Difficulty, &l.ElapsedDays, &l.ScheduledDays, &reviewedAt); err != nil {
		return domain.AnkiReviewLog{}, err
	}
	l.ReviewedAt = tsTime(reviewedAt)
	return l, nil
}

// ReviewLog streams the recorded review events, most recent first. A deckID of
// 0 spans every deck in the tenant; limit <= 0 means no limit.
func (s *ankiStore) ReviewLog(ctx context.Context, scope string, deckID int64, limit int) iter.Seq2[*domain.AnkiReviewLog, error] {
	tenant := tenantID(scope)
	return func(yield func(*domain.AnkiReviewLog, error) bool) {
		query := `SELECT ` + reviewLogCols + ` FROM anki_review_log
			 WHERE tenant_id = $1`
		args := []any{tenant}
		if deckID != 0 {
			query += ` AND deck_id = $2`
			args = append(args, deckID)
		}
		query += ` ORDER BY reviewed_at DESC, id DESC`
		if limit > 0 {
			query += fmt.Sprintf(` LIMIT %d`, limit)
		}
		rows, err := s.db.Query(ctx, query, args...)
		if err != nil {
			yield(nil, fmt.Errorf("postgres: anki review log: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			l, err := scanReviewLog(rows)
			if err != nil {
				yield(nil, fmt.Errorf("postgres: anki review log: %w", err))
				return
			}
			if !yield(&l, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("postgres: anki review log: %w", err))
		}
	}
}

// OptimizeParams suggests (and optionally applies) a tuned request_retention for
// a deck, derived from the pass rate on its review-state reviews (ANK-E2/B10).
// The Go FSRS port has no weight trainer, so this is a request-retention nudge,
// not a full weight re-fit.
func (s *ankiStore) OptimizeParams(ctx context.Context, scope string, deckID int64, apply bool) (*domain.AnkiOptimizeResult, error) {
	tenant := tenantID(scope)

	var current float64
	err := s.db.QueryRow(ctx,
		`SELECT request_retention FROM anki_deck WHERE id = $1 AND tenant_id = $2`,
		deckID, tenant).Scan(&current)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("postgres: optimize anki deck %d: %w", deckID, storage.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: optimize anki deck %d: %w", deckID, err)
	}

	var total, passed int
	if err := s.db.QueryRow(ctx,
		`SELECT COUNT(*), COALESCE(SUM(CASE WHEN rating >= 2 THEN 1 ELSE 0 END), 0)
		 FROM anki_review_log WHERE deck_id = $1 AND tenant_id = $2 AND state = 2`,
		deckID, tenant).Scan(&total, &passed); err != nil {
		return nil, fmt.Errorf("postgres: optimize anki deck %d: %w", deckID, err)
	}

	res := &domain.AnkiOptimizeResult{SampleSize: total, CurrentRetention: current}
	if total > 0 {
		res.ObservedRetention = float64(passed) / float64(total)
	}
	res.SuggestedRetention = domain.SuggestRetention(current, res.ObservedRetention, total)

	if apply && res.SuggestedRetention != current {
		if _, err := s.db.Exec(ctx,
			`UPDATE anki_deck SET request_retention = $1, updated_at = now()
			 WHERE id = $2 AND tenant_id = $3`,
			res.SuggestedRetention, deckID, tenant); err != nil {
			return nil, fmt.Errorf("postgres: optimize anki deck %d: %w", deckID, err)
		}
		res.Applied = true
	}
	return res, nil
}
