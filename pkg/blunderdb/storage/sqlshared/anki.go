package sqlshared

import (
	"errors"
	"fmt"
	"iter"
	"strconv"
	"strings"
	"time"

	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/anki"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// AnkiStore implements every storage.AnkiStore method except Forecast
// (B.14, #182): its day-offset bucketing is genuine date-arithmetic
// divergence — SQLite has no DATE type and computes it through julianday(),
// PostgreSQL natively — so each backend still writes that one query itself,
// embedding AnkiStore and shadowing the method (the stats_sqlite.go/
// stats_postgres.go precedent for StatsStore.DateRange).
//
// Every "now" this store writes or compares against is computed once in Go
// (anki.TimeLayout, UTC) and bound through Execer.TimestampArg — the same
// trick TimestampText already uses for reads — so due/last_review/
// created_at/updated_at/reviewed_at need no per-backend SQL at all: SQLite's
// columns are already TEXT in that layout, and PostgreSQL casts the bound
// string with "::timestamptz". This is also why BuryCard's "start of
// tomorrow, UTC" is computed in Go (time.Truncate(24h) is UTC-safe: no DST)
// rather than through either backend's date arithmetic.
//
// Positions is the position loader this store borrows rather than owns:
// position scanning stays backend-specific (position, analyses, matches —
// see the package doc), so the two places anki needs one (the next/random
// card served, and DeckPositions) go through storage.PositionStore's
// Load/LoadByIDs instead of a JOIN this package would have to write per
// dialect.
type AnkiStore struct {
	DB        Execer
	Positions storage.PositionStore
}

// ankiNow is "now", formatted the way every anki_* timestamp column is bound
// and compared: anki.TimeLayout, UTC, second precision.
func ankiNow() string { return time.Now().UTC().Format(anki.TimeLayout) }

// ankiDeckSelectCols reads a domain.AnkiDeck; the three correlated subqueries
// supply the card counters. now is the due-count cutoff.
func (s *AnkiStore) ankiDeckSelectCols() string {
	return `ad.id, ad.name, COALESCE(ad.description,''),
		ad.source_type, ad.source_id, COALESCE(ad.source_command,''),
		ad.request_retention, ad.maximum_interval, ` + s.DB.BoolAsInt("ad.enable_fuzz") + `, ad.session_limit,
		` + s.DB.TimestampText("ad.created_at") + `, ` + s.DB.TimestampText("ad.updated_at") + `,
		(SELECT COUNT(*) FROM anki_card ac WHERE ac.deck_id = ad.id),
		(SELECT COUNT(*) FROM anki_card ac WHERE ac.deck_id = ad.id AND ac.due <= ` + s.DB.TimestampArg() + `),
		(SELECT COUNT(*) FROM anki_card ac WHERE ac.deck_id = ad.id AND ac.state = 0)`
}

func scanAnkiDeck(sc interface{ Scan(...any) error }) (domain.AnkiDeck, error) {
	var d domain.AnkiDeck
	var enableFuzz int
	// NULL is "no limit" and stays nil in the domain; 0 is a limit that
	// serves nothing. *int64 is what keeps the two apart across the
	// boundary, on both backends (database/sql and pgx both support scanning
	// a nullable column into a pointer-to-pointer destination).
	var sessionLimit *int64
	if err := sc.Scan(&d.ID, &d.Name, &d.Description,
		&d.SourceType, &d.SourceID, &d.SourceCommand,
		&d.RequestRetention, &d.MaximumInterval, &enableFuzz, &sessionLimit,
		&d.CreatedAt, &d.UpdatedAt,
		&d.CardCount, &d.DueCount, &d.NewCount); err != nil {
		return domain.AnkiDeck{}, err
	}
	d.EnableFuzz = enableFuzz != 0
	if sessionLimit != nil {
		v := int(*sessionLimit)
		d.SessionLimit = &v
	}
	return d, nil
}

// CreateDeck stores a new spaced-repetition deck and returns its id.
func (s *AnkiStore) CreateDeck(ctx context.Context, scope string, name, description, sourceType string, sourceID int64, sourceCommand string) (int64, error) {
	cols, args := s.DB.TenantColumns(scope)
	cols = append(cols, "name", "description", "source_type", "source_id", "source_command", "maximum_interval")
	// maximum_interval is written explicitly rather than left to the column
	// default: the default a NEW deck gets is a product decision (ADR-0026
	// rule 7), and the DDL's 36500 stays what it is so existing decks keep
	// theirs.
	args = append(args, name, description, sourceType, sourceID, sourceCommand, float64(domain.AnkiDefaultMaximumInterval))
	id, err := s.DB.Insert(ctx,
		`INSERT INTO anki_deck (`+strings.Join(cols, ", ")+`) VALUES (`+Placeholders(len(cols))+`)`, args...)
	if err != nil {
		return 0, errf(s.DB, "create anki deck", err)
	}
	return id, nil
}

// ListDecks streams every deck with its card counters, oldest first.
func (s *AnkiStore) ListDecks(ctx context.Context, scope string) iter.Seq2[*domain.AnkiDeck, error] {
	return func(yield func(*domain.AnkiDeck, error) bool) {
		tenant, targs := s.DB.TenantFilter("ad", scope)
		rows, err := s.DB.Query(ctx,
			`SELECT `+s.ankiDeckSelectCols()+` FROM anki_deck ad WHERE `+tenant+` ORDER BY ad.id ASC`,
			append([]any{ankiNow()}, targs...)...)
		if err != nil {
			yield(nil, errf(s.DB, "list anki decks", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			d, err := scanAnkiDeck(rows)
			if err != nil {
				yield(nil, errf(s.DB, "list anki decks", err))
				return
			}
			if !yield(&d, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, errf(s.DB, "list anki decks", err))
		}
	}
}

// UpdateDeck changes a deck's name and description.
func (s *AnkiStore) UpdateDeck(ctx context.Context, scope string, id int64, name, description string) error {
	tenant, targs := s.DB.TenantFilter("", scope)
	if _, err := s.DB.Exec(ctx,
		`UPDATE anki_deck SET name = ?, description = ?, updated_at = `+s.DB.TimestampArg()+`
		 WHERE id = ? AND `+tenant,
		append([]any{name, description, ankiNow(), id}, targs...)...); err != nil {
		return errf(s.DB, fmt.Sprintf("update anki deck %d", id), err)
	}
	return nil
}

// UpdateDeckParams changes a deck's FSRS scheduling parameters.
func (s *AnkiStore) UpdateDeckParams(ctx context.Context, scope string, id int64, requestRetention, maximumInterval float64, enableFuzz bool, sessionLimit *int) error {
	tenant, targs := s.DB.TenantFilter("", scope)
	var limit any
	if sessionLimit != nil {
		limit = *sessionLimit
	}
	if _, err := s.DB.Exec(ctx,
		`UPDATE anki_deck SET request_retention = ?, maximum_interval = ?, enable_fuzz = ?,
		 session_limit = ?, updated_at = `+s.DB.TimestampArg()+` WHERE id = ? AND `+tenant,
		append([]any{requestRetention, maximumInterval, s.DB.BoolArg(enableFuzz), limit, ankiNow(), id}, targs...)...); err != nil {
		return errf(s.DB, fmt.Sprintf("update anki deck %d params", id), err)
	}
	return nil
}

// DeleteDeck removes a deck; its cards cascade off the anki_card foreign key.
func (s *AnkiStore) DeleteDeck(ctx context.Context, scope string, id int64) error {
	tenant, targs := s.DB.TenantFilter("", scope)
	if _, err := s.DB.Exec(ctx,
		`DELETE FROM anki_deck WHERE id = ? AND `+tenant,
		append([]any{id}, targs...)...); err != nil {
		return errf(s.DB, fmt.Sprintf("delete anki deck %d", id), err)
	}
	return nil
}

// ResetDeck clears the FSRS state of every card in a deck back to new.
func (s *AnkiStore) ResetDeck(ctx context.Context, scope string, deckID int64) error {
	tenant, targs := s.DB.TenantFilter("", scope)
	if _, err := s.DB.Exec(ctx,
		`UPDATE anki_card SET due = `+s.DB.TimestampArg()+`, stability = 0, difficulty = 0,
		 elapsed_days = 0, scheduled_days = 0, reps = 0, lapses = 0, state = 0, last_review = NULL,
		 `+s.DB.Bool("suspended", false)+`, buried_until = NULL
		 WHERE deck_id = ? AND `+tenant,
		append([]any{ankiNow(), deckID}, targs...)...); err != nil {
		return errf(s.DB, fmt.Sprintf("reset anki deck %d", deckID), err)
	}
	return nil
}

// Sync reconciles a deck's cards with its source: the positions of a
// collection, or the position ids listed in a search deck's source command.
func (s *AnkiStore) Sync(ctx context.Context, scope string, deckID int64) error {
	tenant, targs := s.DB.TenantFilter("", scope)
	var sourceType, sourceCommand string
	var sourceID int64
	err := s.DB.QueryRow(ctx,
		`SELECT source_type, source_id, COALESCE(source_command,'') FROM anki_deck WHERE id = ? AND `+tenant,
		append([]any{deckID}, targs...)...).Scan(&sourceType, &sourceID, &sourceCommand)
	if errors.Is(err, ErrNoRows) {
		return errf(s.DB, fmt.Sprintf("sync anki deck %d", deckID), storage.ErrNotFound)
	}
	if err != nil {
		return errf(s.DB, fmt.Sprintf("sync anki deck %d", deckID), err)
	}

	var positionIDs []int64
	switch sourceType {
	case domain.AnkiSourceCollection:
		cptenant, cpargs := s.DB.TenantFilter("", scope)
		rows, err := s.DB.Query(ctx,
			`SELECT position_id FROM collection_position WHERE collection_id = ? AND `+cptenant+` ORDER BY sort_order ASC`,
			append([]any{sourceID}, cpargs...)...)
		if err != nil {
			return errf(s.DB, fmt.Sprintf("sync anki deck %d", deckID), err)
		}
		for rows.Next() {
			var pid int64
			if err := rows.Scan(&pid); err != nil {
				rows.Close()
				return errf(s.DB, fmt.Sprintf("sync anki deck %d", deckID), err)
			}
			positionIDs = append(positionIDs, pid)
		}
		rerr := rows.Err()
		rows.Close()
		if rerr != nil {
			return errf(s.DB, fmt.Sprintf("sync anki deck %d", deckID), rerr)
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
// The insert's own conflict-avoidance is the plain SQL-standard
// "ON CONFLICT ... DO NOTHING", which SQLite (>= 3.24, same as the metadata/
// session upserts elsewhere in this package) and PostgreSQL both execute
// identically — no INSERT OR IGNORE/dialect split needed here.
func (s *AnkiStore) SyncWithPositions(ctx context.Context, scope string, deckID int64, positionIDs []int64) error {
	// Une décision de videau est deux questions (#276) : si la source en
	// sélectionne une moitié, l'autre complète la décision plutôt que
	// d'ajouter autre chose. Voir anki_cube_pairs.go.
	positionIDs = completeCubePairs(ctx, s.DB, scope, positionIDs)
	err := s.DB.Transact(ctx, func(tx Execer) error {
		now := ankiNow()
		for _, pid := range positionIDs {
			cols, args := tx.TenantColumns(scope)
			cols = append(cols, "deck_id", "position_id", "due", "state")
			args = append(args, deckID, pid, now, 0)
			if _, err := tx.Exec(ctx,
				`INSERT INTO anki_card (`+strings.Join(cols, ", ")+`) VALUES (`+Placeholders(len(cols))+`)
				 ON CONFLICT (deck_id, position_id) DO NOTHING`, args...); err != nil {
				return err
			}
		}
		tenant, targs := tx.TenantFilter("", scope)
		if _, err := tx.Exec(ctx,
			`UPDATE anki_deck SET updated_at = `+tx.TimestampArg()+` WHERE id = ? AND `+tenant,
			append([]any{now, deckID}, targs...)...); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return errf(s.DB, fmt.Sprintf("sync anki deck %d", deckID), err)
	}
	return nil
}

// DeckPositions streams the positions linked to a deck's cards, ordered by
// position id — via storage.PositionStore.LoadByIDs (one round trip) rather
// than a JOIN, so this store never has to know how a backend selects a
// position (see the type doc).
func (s *AnkiStore) DeckPositions(ctx context.Context, scope string, deckID int64) iter.Seq2[*domain.Position, error] {
	return func(yield func(*domain.Position, error) bool) {
		tenant, targs := s.DB.TenantFilter("", scope)
		rows, err := s.DB.Query(ctx,
			`SELECT position_id FROM anki_card WHERE deck_id = ? AND `+tenant+` ORDER BY position_id ASC`,
			append([]any{deckID}, targs...)...)
		if err != nil {
			yield(nil, errf(s.DB, "anki deck positions", err))
			return
		}
		var ids []int64
		for rows.Next() {
			var pid int64
			if err := rows.Scan(&pid); err != nil {
				rows.Close()
				yield(nil, errf(s.DB, "anki deck positions", err))
				return
			}
			ids = append(ids, pid)
		}
		rerr := rows.Err()
		rows.Close()
		if rerr != nil {
			yield(nil, errf(s.DB, "anki deck positions", rerr))
			return
		}
		positions, err := s.Positions.LoadByIDs(ctx, scope, ids)
		if err != nil {
			yield(nil, errf(s.DB, "anki deck positions", err))
			return
		}
		for i := range positions {
			if !yield(&positions[i], nil) {
				return
			}
		}
	}
}

// DeckStats returns the review counters for a deck.
func (s *AnkiStore) DeckStats(ctx context.Context, scope string, deckID int64) (*domain.AnkiDeckStats, error) {
	now := ankiNow()
	tenant, targs := s.DB.TenantFilter("", scope)
	// Queue counters exclude suspended/buried cards; TotalCount counts the
	// whole deck regardless of availability. avail is the availability
	// predicate inlined per counter, each occurrence binding its own
	// current-time argument for the buried_until comparison.
	avail := s.DB.Bool("suspended", false) + ` AND (buried_until IS NULL OR buried_until <= ` + s.DB.TimestampArg() + `)`
	query := `SELECT
			COALESCE(SUM(CASE WHEN state = 0 AND ` + avail + ` THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN (state = 1 OR state = 3) AND ` + avail + ` THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN state = 2 AND due <= ` + s.DB.TimestampArg() + ` AND ` + avail + ` THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN due <= ` + s.DB.TimestampArg() + ` AND ` + avail + ` THEN 1 ELSE 0 END), 0),
			COUNT(*)
		 FROM anki_card WHERE deck_id = ? AND ` + tenant
	// Six "now" placeholders precede deckID's, in this order: avail (new),
	// avail (learning), due<= then avail (review), due<= then avail (due).
	// All six bind the identical value, so their relative order does not
	// actually matter — only the count does.
	args := []any{now, now, now, now, now, now, deckID}
	args = append(args, targs...)
	var st domain.AnkiDeckStats
	err := s.DB.QueryRow(ctx, query, args...).
		Scan(&st.NewCount, &st.LearningCount, &st.ReviewCount, &st.DueCount, &st.TotalCount)
	if err != nil {
		return nil, errf(s.DB, fmt.Sprintf("anki deck %d stats", deckID), err)
	}
	return &st, nil
}

// ankiCardCols reads a domain.AnkiCard.
func (s *AnkiStore) ankiCardCols() string {
	return `id, deck_id, position_id, ` + s.DB.TimestampText("due") + `, stability, difficulty,
		elapsed_days, scheduled_days, reps, lapses, state, ` + s.DB.TimestampText("last_review")
}

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

// ankiAvailable is the predicate that excludes suspended and still-buried
// cards from the review queue. It binds the current time as its single
// parameter (the buried_until comparison).
func (s *AnkiStore) ankiAvailable() string {
	return s.DB.Bool("suspended", false) + ` AND (buried_until IS NULL OR buried_until <= ` + s.DB.TimestampArg() + `)`
}

// nextDueCard returns the highest-priority card due in a deck, or
// storage.ErrNotFound.
//
// The last ORDER BY term is RANDOM(), and it is deliberate — not a tie-break
// left unwritten (ADR-0026 rule 9). Every new card of a freshly synced deck
// carries the SAME due timestamp, so `due ASC` separates none of them and the
// engine falls back on insertion order: the order of the match the positions
// came from. A session then served consecutive moves of one game, which are
// correlated, in the sequence they were played — blocking, where the learning
// literature wants interleaving. Randomising the ties is the fix, and it is
// the behaviour, not an option: no display-order setting is exposed anywhere.
// RANDOM() is spelled uppercase but is the same function as PostgreSQL's
// random(): both fold unquoted identifiers case-insensitively.
func (s *AnkiStore) nextDueCard(ctx context.Context, scope string, deckID int64) (domain.AnkiCard, error) {
	tenant, targs := s.DB.TenantFilter("", scope)
	now := ankiNow()
	query := `SELECT ` + s.ankiCardCols() + ` FROM anki_card
		WHERE deck_id = ? AND due <= ` + s.DB.TimestampArg() + ` AND ` + s.ankiAvailable() + ` AND ` + tenant + `
		ORDER BY
			CASE WHEN state = 1 OR state = 3 THEN 0
			     WHEN state = 2 THEN 1
			     ELSE 2 END,
			due ASC,
			RANDOM()
		LIMIT 1`
	args := append([]any{deckID, now, now}, targs...)
	c, err := scanAnkiCard(s.DB.QueryRow(ctx, query, args...))
	if errors.Is(err, ErrNoRows) {
		return domain.AnkiCard{}, storage.ErrNotFound
	}
	if err != nil {
		return domain.AnkiCard{}, err
	}
	return c, nil
}

// NextCard returns the next card due for review in a deck, or ErrNotFound.
func (s *AnkiStore) NextCard(ctx context.Context, scope string, deckID int64) (*domain.AnkiReviewCard, error) {
	card, err := s.nextDueCard(ctx, scope, deckID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, err
		}
		return nil, errf(s.DB, fmt.Sprintf("next anki card of deck %d", deckID), err)
	}
	pos, err := s.Positions.Load(ctx, scope, card.PositionID)
	if err != nil {
		return nil, errf(s.DB, fmt.Sprintf("next anki card of deck %d", deckID), err)
	}
	return &domain.AnkiReviewCard{Card: card, Position: *pos}, nil
}

// randomCard draws one card of the deck at random, optionally skipping
// excludePositionID (0 = no exclusion), or storage.ErrNotFound when none
// qualifies.
func (s *AnkiStore) randomCard(ctx context.Context, scope string, deckID, excludePositionID int64) (domain.AnkiCard, error) {
	tenant, targs := s.DB.TenantFilter("", scope)
	query := `SELECT ` + s.ankiCardCols() + ` FROM anki_card WHERE deck_id = ? AND ` + tenant
	args := append([]any{deckID}, targs...)
	if excludePositionID != 0 {
		query += ` AND position_id != ?`
		args = append(args, excludePositionID)
	}
	query += ` ORDER BY RANDOM() LIMIT 1`
	c, err := scanAnkiCard(s.DB.QueryRow(ctx, query, args...))
	if errors.Is(err, ErrNoRows) {
		return domain.AnkiCard{}, storage.ErrNotFound
	}
	if err != nil {
		return domain.AnkiCard{}, err
	}
	return c, nil
}

// RandomCard draws one card of a deck at random for a cram session, ignoring
// the FSRS schedule and the card's availability (see storage.AnkiStore).
func (s *AnkiStore) RandomCard(ctx context.Context, scope string, deckID, excludePositionID int64) (*domain.AnkiReviewCard, error) {
	card, err := s.randomCard(ctx, scope, deckID, excludePositionID)
	if errors.Is(err, storage.ErrNotFound) && excludePositionID != 0 {
		// Only the excluded card remains (single-card deck): draw again
		// without the exclusion so cram still serves it.
		card, err = s.randomCard(ctx, scope, deckID, 0)
	}
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, err
		}
		return nil, errf(s.DB, fmt.Sprintf("random anki card of deck %d", deckID), err)
	}
	pos, err := s.Positions.Load(ctx, scope, card.PositionID)
	if err != nil {
		return nil, errf(s.DB, fmt.Sprintf("random anki card of deck %d", deckID), err)
	}
	return &domain.AnkiReviewCard{Card: card, Position: *pos}, nil
}

// ReviewCard records a review rating against a card, advances its FSRS
// scheduling state, and returns the next card still due in the same deck (nil
// when none remain).
//
// The card update and the review-log append are one transaction: a log entry
// without its card advance (or the reverse) would make the log lie about the
// schedule it is supposed to explain. Looking up the next card happens after
// the commit — failing to load it must not undo a grade that was given.
func (s *AnkiStore) ReviewCard(ctx context.Context, scope string, cardID int64, rating int) (*domain.AnkiReviewCard, error) {
	var deckID int64
	err := s.DB.Transact(ctx, func(tx Execer) error {
		tenant, targs := tx.TenantFilter("", scope)
		card, err := scanAnkiCard(tx.QueryRow(ctx,
			`SELECT `+s.ankiCardCols()+` FROM anki_card WHERE id = ? AND `+tenant,
			append([]any{cardID}, targs...)...))
		if errors.Is(err, ErrNoRows) {
			return storage.ErrNotFound
		}
		if err != nil {
			return err
		}
		deckID = card.DeckID

		var (
			params     anki.Params
			enableFuzz int
		)
		dtenant, dtargs := tx.TenantFilter("", scope)
		err = tx.QueryRow(ctx,
			`SELECT request_retention, maximum_interval, `+tx.BoolAsInt("enable_fuzz")+` FROM anki_deck WHERE id = ? AND `+dtenant,
			append([]any{card.DeckID}, dtargs...)...).Scan(&params.RequestRetention, &params.MaximumInterval, &enableFuzz)
		if errors.Is(err, ErrNoRows) {
			return fmt.Errorf("deck: %w", storage.ErrNotFound)
		}
		if err != nil {
			return err
		}
		params.EnableFuzz = enableFuzz != 0

		next, log, err := anki.ScheduleNext(card, params, rating, time.Now())
		if errors.Is(err, anki.ErrInvalidRating) {
			return fmt.Errorf("%w: %w", storage.ErrInvalid, err)
		}
		if err != nil {
			return err
		}

		utenant, utargs := tx.TenantFilter("", scope)
		if _, err := tx.Exec(ctx,
			`UPDATE anki_card SET due = `+tx.TimestampArg()+`, stability = ?, difficulty = ?,
			 elapsed_days = ?, scheduled_days = ?, reps = ?, lapses = ?, state = ?, last_review = `+tx.TimestampArg()+`
			 WHERE id = ? AND `+utenant,
			append([]any{next.Due, next.Stability, next.Difficulty,
				next.ElapsedDays, next.ScheduledDays, next.Reps, next.Lapses, next.State,
				next.LastReview, cardID}, utargs...)...); err != nil {
			return err
		}
		cols, cargs := tx.TenantColumns(scope)
		cols = append(cols, "card_id", "deck_id", "position_id", "rating", "state",
			"stability", "difficulty", "elapsed_days", "scheduled_days", "reviewed_at")
		cargs = append(cargs, log.CardID, log.DeckID, log.PositionID, log.Rating, log.State,
			log.Stability, log.Difficulty, log.ElapsedDays, log.ScheduledDays, log.ReviewedAt)
		if _, err := tx.Exec(ctx,
			`INSERT INTO anki_review_log (`+strings.Join(cols, ", ")+`) VALUES (`+Placeholders(len(cols))+`)`,
			cargs...); err != nil {
			return fmt.Errorf("log: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, errf(s.DB, fmt.Sprintf("review anki card %d", cardID), err)
	}

	nextCard, err := s.nextDueCard(ctx, scope, deckID)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, errf(s.DB, fmt.Sprintf("review anki card %d", cardID), err)
	}
	pos, err := s.Positions.Load(ctx, scope, nextCard.PositionID)
	if err != nil {
		return nil, errf(s.DB, fmt.Sprintf("review anki card %d", cardID), err)
	}
	return &domain.AnkiReviewCard{Card: nextCard, Position: *pos}, nil
}

// checkAnkiRowAffected maps a no-op update/delete to storage.ErrNotFound.
func checkAnkiRowAffected(n int64, cardID int64, op string) error {
	if n == 0 {
		return fmt.Errorf("%s anki card %d: %w", op, cardID, storage.ErrNotFound)
	}
	return nil
}

// SetCardSuspended suspends or unsuspends a card.
func (s *AnkiStore) SetCardSuspended(ctx context.Context, scope string, cardID int64, suspended bool) error {
	tenant, targs := s.DB.TenantFilter("", scope)
	n, err := s.DB.Exec(ctx,
		`UPDATE anki_card SET suspended = ? WHERE id = ? AND `+tenant,
		append([]any{s.DB.BoolArg(suspended), cardID}, targs...)...)
	if err != nil {
		return errf(s.DB, fmt.Sprintf("suspend anki card %d", cardID), err)
	}
	if err := checkAnkiRowAffected(n, cardID, "suspend"); err != nil {
		return errf(s.DB, "", err)
	}
	return nil
}

// BuryCard hides a card until the start of the next day (UTC). Computed in Go
// (see the type doc) rather than through either backend's date arithmetic.
func (s *AnkiStore) BuryCard(ctx context.Context, scope string, cardID int64) error {
	tenant, targs := s.DB.TenantFilter("", scope)
	tomorrow := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, 1).Format(anki.TimeLayout)
	n, err := s.DB.Exec(ctx,
		`UPDATE anki_card SET buried_until = `+s.DB.TimestampArg()+` WHERE id = ? AND `+tenant,
		append([]any{tomorrow, cardID}, targs...)...)
	if err != nil {
		return errf(s.DB, fmt.Sprintf("bury anki card %d", cardID), err)
	}
	if err := checkAnkiRowAffected(n, cardID, "bury"); err != nil {
		return errf(s.DB, "", err)
	}
	return nil
}

// RemoveCard deletes a single card from its deck.
func (s *AnkiStore) RemoveCard(ctx context.Context, scope string, cardID int64) error {
	tenant, targs := s.DB.TenantFilter("", scope)
	n, err := s.DB.Exec(ctx,
		`DELETE FROM anki_card WHERE id = ? AND `+tenant, append([]any{cardID}, targs...)...)
	if err != nil {
		return errf(s.DB, fmt.Sprintf("remove anki card %d", cardID), err)
	}
	if err := checkAnkiRowAffected(n, cardID, "remove"); err != nil {
		return errf(s.DB, "", err)
	}
	return nil
}

// reviewLogCols reads a domain.AnkiReviewLog.
func (s *AnkiStore) reviewLogCols() string {
	return `id, card_id, deck_id, position_id, rating, state,
		stability, difficulty, elapsed_days, scheduled_days, ` + s.DB.TimestampText("reviewed_at")
}

func scanReviewLog(sc interface{ Scan(...any) error }) (domain.AnkiReviewLog, error) {
	var l domain.AnkiReviewLog
	if err := sc.Scan(&l.ID, &l.CardID, &l.DeckID, &l.PositionID, &l.Rating, &l.State,
		&l.Stability, &l.Difficulty, &l.ElapsedDays, &l.ScheduledDays, &l.ReviewedAt); err != nil {
		return domain.AnkiReviewLog{}, err
	}
	return l, nil
}

// ReviewLog streams the recorded review events, most recent first. A deckID
// of 0 spans every deck in the tenant; limit <= 0 means no limit.
func (s *AnkiStore) ReviewLog(ctx context.Context, scope string, deckID int64, limit int) iter.Seq2[*domain.AnkiReviewLog, error] {
	return func(yield func(*domain.AnkiReviewLog, error) bool) {
		tenant, targs := s.DB.TenantFilter("", scope)
		query := `SELECT ` + s.reviewLogCols() + ` FROM anki_review_log WHERE ` + tenant
		args := append([]any{}, targs...)
		if deckID != 0 {
			query += ` AND deck_id = ?`
			args = append(args, deckID)
		}
		query += ` ORDER BY reviewed_at DESC, id DESC`
		if limit > 0 {
			query += fmt.Sprintf(` LIMIT %d`, limit)
		}
		rows, err := s.DB.Query(ctx, query, args...)
		if err != nil {
			yield(nil, errf(s.DB, "anki review log", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			l, err := scanReviewLog(rows)
			if err != nil {
				yield(nil, errf(s.DB, "anki review log", err))
				return
			}
			if !yield(&l, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, errf(s.DB, "anki review log", err))
		}
	}
}

// Retention measures a deck's pass rate on review-state cards against its
// target retention. Read-only by contract (ADR-0026 rule 5): it used to also
// suggest a new target and write it back, which is the feedback loop FSRS's
// authors reject.
func (s *AnkiStore) Retention(ctx context.Context, scope string, deckID int64) (*domain.AnkiRetention, error) {
	dtenant, dtargs := s.DB.TenantFilter("", scope)
	var target float64
	err := s.DB.QueryRow(ctx,
		`SELECT request_retention FROM anki_deck WHERE id = ? AND `+dtenant,
		append([]any{deckID}, dtargs...)...).Scan(&target)
	if errors.Is(err, ErrNoRows) {
		return nil, errf(s.DB, fmt.Sprintf("anki deck %d retention", deckID), storage.ErrNotFound)
	}
	if err != nil {
		return nil, errf(s.DB, fmt.Sprintf("anki deck %d retention", deckID), err)
	}

	ltenant, ltargs := s.DB.TenantFilter("", scope)
	var total, passed int
	if err := s.DB.QueryRow(ctx,
		`SELECT COUNT(*), COALESCE(SUM(CASE WHEN rating >= 2 THEN 1 ELSE 0 END), 0)
		 FROM anki_review_log WHERE deck_id = ? AND state = 2 AND `+ltenant,
		append([]any{deckID}, ltargs...)...).Scan(&total, &passed); err != nil {
		return nil, errf(s.DB, fmt.Sprintf("anki deck %d retention", deckID), err)
	}

	res := &domain.AnkiRetention{SampleSize: total, TargetRetention: target}
	if total > 0 {
		res.ObservedRetention = float64(passed) / float64(total)
	}
	return res, nil
}

// ReviewsByGameType counts the POSITIONS reviewed since `since`, grouped by
// the position's derived plan of play (#275).
//
// Positions and not reviews: a card revised four times in a month is one
// position studied, and counting the repetitions would make a month of
// cramming look like a month of coverage. That distinction is the whole
// reason this is a query rather than a sum of the review log.
func (s *AnkiStore) ReviewsByGameType(ctx context.Context, scope string, since string) (map[string]int, error) {
	tenant, targs := s.DB.TenantFilter("rl", scope)
	rows, err := s.DB.Query(ctx,
		`SELECT COALESCE(p.game_type, 0), COUNT(DISTINCT rl.position_id)
		 FROM anki_review_log rl
		 INNER JOIN position p ON p.id = rl.position_id
		 WHERE `+tenant+` AND rl.reviewed_at >= ?
		 GROUP BY p.game_type`,
		append(targs, since)...)
	if err != nil {
		return nil, errf(s.DB, "count reviews by game type", err)
	}
	defer rows.Close()
	out := make(map[string]int)
	for rows.Next() {
		var gameType, n int
		if err := rows.Scan(&gameType, &n); err != nil {
			return nil, errf(s.DB, "count reviews by game type", err)
		}
		out[domain.GameType(gameType).String()] = n
	}
	return out, rows.Err()
}
