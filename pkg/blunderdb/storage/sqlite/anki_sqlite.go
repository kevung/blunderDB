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
		 elapsed_days = 0, scheduled_days = 0, reps = 0, lapses = 0, state = 0, last_review = '',
		 suspended = 0, buried_until = NULL
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
	// Queue counters exclude suspended/buried cards; TotalCount counts the
	// whole deck regardless of availability. `avail` is the availability
	// predicate inlined per counter (each binds its own current-time param).
	const avail = `suspended = 0 AND (buried_until IS NULL OR buried_until <= ?)`
	var st domain.AnkiDeckStats
	err := s.db.QueryRowContext(ctx,
		`SELECT
			COALESCE(SUM(CASE WHEN state = 0 AND `+avail+` THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN (state = 1 OR state = 3) AND `+avail+` THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN state = 2 AND due <= ? AND `+avail+` THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN due <= ? AND `+avail+` THEN 1 ELSE 0 END), 0),
			COUNT(*)
		 FROM anki_card WHERE deck_id = ?`,
		now, now, now, now, now, now, deckID).Scan(&st.NewCount, &st.LearningCount, &st.ReviewCount, &st.DueCount, &st.TotalCount)
	if err != nil {
		return nil, fmt.Errorf("sqlite: anki deck %d stats: %w", deckID, err)
	}
	return &st, nil
}

// Forecast projects how many cards come due over the next `days` calendar days,
// offset 0 absorbing every overdue card. (The sqlite backend is the
// single-tenant Desktop store, so scope is unused.)
func (s *ankiStore) Forecast(ctx context.Context, scope string, deckID int64, days int) ([]domain.AnkiForecastDay, error) {
	switch {
	case days <= 0:
		days = 30
	case days > 365:
		days = 365
	}

	query := `SELECT MAX(0, CAST(julianday(date(due)) - julianday(date('now')) AS INTEGER)) AS day_offset, COUNT(*)
		FROM anki_card
		WHERE suspended = 0 AND date(due) < date('now', '+' || ? || ' days')`
	args := []any{days}
	if deckID != 0 {
		query += ` AND deck_id = ?`
		args = append(args, deckID)
	}
	query += ` GROUP BY day_offset`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("sqlite: anki forecast: %w", err)
	}
	defer rows.Close()
	counts := make(map[int]int, days)
	for rows.Next() {
		var off, n int
		if err := rows.Scan(&off, &n); err != nil {
			return nil, fmt.Errorf("sqlite: anki forecast: %w", err)
		}
		counts[off] = n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: anki forecast: %w", err)
	}
	return storage.BuildForecast(time.Now().UTC(), days, counts), nil
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

// ankiAvailableSQL is the predicate that excludes suspended and still-buried
// cards from the review queue. It binds the current time as its single
// parameter (the buried_until comparison).
const ankiAvailableSQL = `suspended = 0 AND (buried_until IS NULL OR buried_until <= ?)`

// nextDueCardSQL orders due cards so learning/relearning come first, then
// review, then new; ties break on the due date.
const nextDueCardSQL = `SELECT ` + ankiCardCols + ` FROM anki_card
	WHERE deck_id = ? AND due <= ? AND ` + ankiAvailableSQL + `
	ORDER BY
		CASE WHEN state = 1 OR state = 3 THEN 0
		     WHEN state = 2 THEN 1
		     ELSE 2 END,
		due ASC
	LIMIT 1`

// nextDueCard returns the highest-priority card due in a deck, or ErrNotFound.
func (s *ankiStore) nextDueCard(ctx context.Context, deckID int64) (domain.AnkiCard, error) {
	now := ankiNow()
	row := s.db.QueryRowContext(ctx, nextDueCardSQL, deckID, now, now)
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
	info := fsrs.NewFSRS(params).Next(fsrsCard, now, fsrs.Rating(rating))
	next := info.Card

	if _, err := s.db.ExecContext(ctx,
		`UPDATE anki_card SET due = ?, stability = ?, difficulty = ?,
		 elapsed_days = ?, scheduled_days = ?, reps = ?, lapses = ?, state = ?, last_review = ?
		 WHERE id = ?`,
		next.Due.UTC().Format(ankiTimeLayout), next.Stability, next.Difficulty,
		next.ElapsedDays, next.ScheduledDays, next.Reps, next.Lapses, int(next.State),
		now.Format(ankiTimeLayout), cardID); err != nil {
		return nil, fmt.Errorf("sqlite: review anki card %d: %w", cardID, err)
	}

	// Append the review to the immutable log. The recorded state is the one the
	// card was in *before* this review (info.ReviewLog.State); stability and
	// difficulty are the post-review values.
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO anki_review_log
		 (card_id, deck_id, position_id, rating, state,
		  stability, difficulty, elapsed_days, scheduled_days, reviewed_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		cardID, card.DeckID, card.PositionID, rating, int(info.ReviewLog.State),
		next.Stability, next.Difficulty,
		int(info.ReviewLog.ElapsedDays), int(next.ScheduledDays),
		now.Format(ankiTimeLayout)); err != nil {
		return nil, fmt.Errorf("sqlite: review anki card %d: log: %w", cardID, err)
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

// SetCardSuspended suspends or unsuspends a card.
func (s *ankiStore) SetCardSuspended(ctx context.Context, scope string, cardID int64, suspended bool) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE anki_card SET suspended = ? WHERE id = ?`, boolToInt(suspended), cardID)
	if err != nil {
		return fmt.Errorf("sqlite: suspend anki card %d: %w", cardID, err)
	}
	return checkAnkiRowAffected(res, cardID, "suspend")
}

// BuryCard hides a card until the start of the next day (UTC).
func (s *ankiStore) BuryCard(ctx context.Context, scope string, cardID int64) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE anki_card SET buried_until = datetime(date('now', '+1 day')) WHERE id = ?`, cardID)
	if err != nil {
		return fmt.Errorf("sqlite: bury anki card %d: %w", cardID, err)
	}
	return checkAnkiRowAffected(res, cardID, "bury")
}

// RemoveCard deletes a single card from its deck.
func (s *ankiStore) RemoveCard(ctx context.Context, scope string, cardID int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM anki_card WHERE id = ?`, cardID)
	if err != nil {
		return fmt.Errorf("sqlite: remove anki card %d: %w", cardID, err)
	}
	return checkAnkiRowAffected(res, cardID, "remove")
}

// checkAnkiRowAffected maps a no-op update/delete to ErrNotFound.
func checkAnkiRowAffected(res sql.Result, cardID int64, op string) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("sqlite: %s anki card %d: %w", op, cardID, err)
	}
	if n == 0 {
		return fmt.Errorf("sqlite: %s anki card %d: %w", op, cardID, storage.ErrNotFound)
	}
	return nil
}

// reviewLogCols reads a domain.AnkiReviewLog.
const reviewLogCols = `id, card_id, deck_id, position_id, rating, state,
	stability, difficulty, elapsed_days, scheduled_days, COALESCE(reviewed_at,'')`

func scanReviewLog(sc interface{ Scan(...any) error }) (domain.AnkiReviewLog, error) {
	var l domain.AnkiReviewLog
	if err := sc.Scan(&l.ID, &l.CardID, &l.DeckID, &l.PositionID, &l.Rating, &l.State,
		&l.Stability, &l.Difficulty, &l.ElapsedDays, &l.ScheduledDays, &l.ReviewedAt); err != nil {
		return domain.AnkiReviewLog{}, err
	}
	return l, nil
}

// ReviewLog streams the recorded review events, most recent first. A deckID of
// 0 spans every deck; limit <= 0 means no limit. (The sqlite backend is the
// single-tenant Desktop store, so scope is unused.)
func (s *ankiStore) ReviewLog(ctx context.Context, scope string, deckID int64, limit int) iter.Seq2[*domain.AnkiReviewLog, error] {
	return func(yield func(*domain.AnkiReviewLog, error) bool) {
		query := `SELECT ` + reviewLogCols + ` FROM anki_review_log`
		args := []any{}
		if deckID != 0 {
			query += ` WHERE deck_id = ?`
			args = append(args, deckID)
		}
		query += ` ORDER BY reviewed_at DESC, id DESC`
		if limit > 0 {
			query += fmt.Sprintf(` LIMIT %d`, limit)
		}
		rows, err := s.db.QueryContext(ctx, query, args...)
		if err != nil {
			yield(nil, fmt.Errorf("sqlite: anki review log: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			l, err := scanReviewLog(rows)
			if err != nil {
				yield(nil, fmt.Errorf("sqlite: anki review log: %w", err))
				return
			}
			if !yield(&l, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("sqlite: anki review log: %w", err))
		}
	}
}

// OptimizeParams suggests (and optionally applies) a tuned request_retention for
// a deck, derived from the pass rate on its review-state reviews (ANK-E2/B10).
// The Go FSRS port has no weight trainer, so this is a request-retention nudge,
// not a full weight re-fit. (scope is unused: single-tenant Desktop store.)
func (s *ankiStore) OptimizeParams(ctx context.Context, scope string, deckID int64, apply bool) (*domain.AnkiOptimizeResult, error) {
	var current float64
	err := s.db.QueryRowContext(ctx,
		`SELECT request_retention FROM anki_deck WHERE id = ?`, deckID).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("sqlite: optimize anki deck %d: %w", deckID, storage.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("sqlite: optimize anki deck %d: %w", deckID, err)
	}

	var total, passed int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(SUM(CASE WHEN rating >= 2 THEN 1 ELSE 0 END), 0)
		 FROM anki_review_log WHERE deck_id = ? AND state = 2`, deckID).Scan(&total, &passed); err != nil {
		return nil, fmt.Errorf("sqlite: optimize anki deck %d: %w", deckID, err)
	}

	res := &domain.AnkiOptimizeResult{SampleSize: total, CurrentRetention: current}
	if total > 0 {
		res.ObservedRetention = float64(passed) / float64(total)
	}
	res.SuggestedRetention = domain.SuggestRetention(current, res.ObservedRetention, total)

	if apply && res.SuggestedRetention != current {
		if _, err := s.db.ExecContext(ctx,
			`UPDATE anki_deck SET request_retention = ?, updated_at = datetime('now') WHERE id = ?`,
			res.SuggestedRetention, deckID); err != nil {
			return nil, fmt.Errorf("sqlite: optimize anki deck %d: %w", deckID, err)
		}
		res.Applied = true
	}
	return res, nil
}
