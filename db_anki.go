package main

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/open-spaced-repetition/go-fsrs/v3"
)

// =====================================================================
// Anki / FSRS functions
// =====================================================================

func (d *Database) newFSRS(deck AnkiDeck) *fsrs.FSRS {
	p := fsrs.DefaultParam()
	p.RequestRetention = deck.RequestRetention
	p.MaximumInterval = deck.MaximumInterval
	p.EnableFuzz = deck.EnableFuzz
	return fsrs.NewFSRS(p)
}

// CreateAnkiDeck creates a new spaced repetition deck
func (d *Database) CreateAnkiDeck(name, description, sourceType string, sourceID int64, sourceCommand string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return 0, fmt.Errorf("no database is currently open")
	}

	result, err := d.db.Exec(`
		INSERT INTO anki_deck (name, description, source_type, source_id, source_command, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, datetime('now'), datetime('now'))
	`, name, description, sourceType, sourceID, sourceCommand)
	if err != nil {
		return 0, fmt.Errorf("error creating anki deck: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return id, nil
}

// GetAllAnkiDecks returns all Anki decks with card counts
func (d *Database) GetAllAnkiDecks() ([]AnkiDeck, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	now := time.Now().UTC().Format("2006-01-02 15:04:05")

	rows, err := d.db.Query(`
		SELECT
			ad.id, ad.name, COALESCE(ad.description, ''),
			ad.source_type, ad.source_id, COALESCE(ad.source_command, ''),
			ad.request_retention, ad.maximum_interval, ad.enable_fuzz,
			COALESCE(strftime('%Y-%m-%d %H:%M:%S', ad.created_at), ''),
			COALESCE(strftime('%Y-%m-%d %H:%M:%S', ad.updated_at), ''),
			COUNT(ac.id) as card_count,
			COALESCE(SUM(CASE WHEN ac.due <= ? THEN 1 ELSE 0 END), 0) as due_count,
			COALESCE(SUM(CASE WHEN ac.state = 0 THEN 1 ELSE 0 END), 0) as new_count
		FROM anki_deck ad
		LEFT JOIN anki_card ac ON ad.id = ac.deck_id
		GROUP BY ad.id
		ORDER BY ad.id ASC
	`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var decks []AnkiDeck
	for rows.Next() {
		var dk AnkiDeck
		var enableFuzz int
		err := rows.Scan(&dk.ID, &dk.Name, &dk.Description,
			&dk.SourceType, &dk.SourceID, &dk.SourceCommand,
			&dk.RequestRetention, &dk.MaximumInterval, &enableFuzz,
			&dk.CreatedAt, &dk.UpdatedAt,
			&dk.CardCount, &dk.DueCount, &dk.NewCount)
		if err != nil {
			return nil, err
		}
		dk.EnableFuzz = enableFuzz != 0
		decks = append(decks, dk)
	}

	if decks == nil {
		decks = []AnkiDeck{}
	}
	return decks, nil
}

// UpdateAnkiDeck updates name and description of a deck
func (d *Database) UpdateAnkiDeck(id int64, name, description string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	_, err := d.db.Exec(`
		UPDATE anki_deck SET name = ?, description = ?, updated_at = datetime('now')
		WHERE id = ?
	`, name, description, id)
	return err
}

// UpdateAnkiDeckParams updates FSRS parameters of a deck
func (d *Database) UpdateAnkiDeckParams(id int64, requestRetention float64, maximumInterval float64, enableFuzz bool) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	fuzzInt := 0
	if enableFuzz {
		fuzzInt = 1
	}

	_, err := d.db.Exec(`
		UPDATE anki_deck SET request_retention = ?, maximum_interval = ?, enable_fuzz = ?, updated_at = datetime('now')
		WHERE id = ?
	`, requestRetention, maximumInterval, fuzzInt, id)
	return err
}

// DeleteAnkiDeck deletes a deck and all its cards
func (d *Database) DeleteAnkiDeck(id int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	_, err := d.db.Exec(`DELETE FROM anki_deck WHERE id = ?`, id)
	return err
}

// SyncAnkiDeck populates cards from the deck's source (collection or search)
func (d *Database) SyncAnkiDeck(deckID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	var sourceType string
	var sourceID int64
	var sourceCommand string
	err := d.db.QueryRow(`SELECT source_type, source_id, source_command FROM anki_deck WHERE id = ?`, deckID).
		Scan(&sourceType, &sourceID, &sourceCommand)
	if err != nil {
		return fmt.Errorf("deck not found: %w", err)
	}

	var positionIDs []int64

	if sourceType == AnkiSourceCollection {
		rows, err := d.db.Query(`
			SELECT position_id FROM collection_position WHERE collection_id = ? ORDER BY sort_order ASC
		`, sourceID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var pid int64
			if err := rows.Scan(&pid); err != nil {
				return err
			}
			positionIDs = append(positionIDs, pid)
		}
	} else if sourceType == AnkiSourceSearch {
		if sourceCommand != "" {
			for _, s := range strings.Split(sourceCommand, ",") {
				s = strings.TrimSpace(s)
				if pid, err := strconv.ParseInt(s, 10, 64); err == nil {
					positionIDs = append(positionIDs, pid)
				}
			}
		}
	}

	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	for _, pid := range positionIDs {
		d.db.Exec(`
			INSERT OR IGNORE INTO anki_card (deck_id, position_id, due, state)
			VALUES (?, ?, ?, 0)
		`, deckID, pid, now)
	}

	d.db.Exec(`UPDATE anki_deck SET updated_at = datetime('now') WHERE id = ?`, deckID)

	return nil
}

// SyncAnkiDeckWithPositions syncs a deck with explicit position IDs (for search-based decks)
func (d *Database) SyncAnkiDeckWithPositions(deckID int64, positionIDs []int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	for _, pid := range positionIDs {
		d.db.Exec(`
			INSERT OR IGNORE INTO anki_card (deck_id, position_id, due, state)
			VALUES (?, ?, ?, 0)
		`, deckID, pid, now)
	}

	d.db.Exec(`UPDATE anki_deck SET updated_at = datetime('now') WHERE id = ?`, deckID)

	return nil
}

// GetAnkiDeckPositions returns all positions associated with a deck's cards.
func (d *Database) GetAnkiDeckPositions(deckID int64) ([]Position, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	rows, err := d.db.Query(`
		SELECT `+positionSelectColsP+`
		FROM anki_card ac
		JOIN position p ON p.id = ac.position_id
		WHERE ac.deck_id = ?
		ORDER BY p.id
	`, deckID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []Position
	for rows.Next() {
		pos, err := scanPositionRow(rows)
		if err != nil {
			return nil, err
		}
		positions = append(positions, pos)
	}
	return positions, rows.Err()
}

// GetAnkiDeckStats returns review statistics for a deck
func (d *Database) GetAnkiDeckStats(deckID int64) (AnkiDeckStats, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	var stats AnkiDeckStats
	if d.db == nil {
		return stats, fmt.Errorf("no database is currently open")
	}

	now := time.Now().UTC().Format("2006-01-02 15:04:05")

	err := d.db.QueryRow(`
		SELECT
			COALESCE(SUM(CASE WHEN state = 0 THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN state = 1 OR state = 3 THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN state = 2 AND due <= ? THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN due <= ? THEN 1 ELSE 0 END), 0),
			COUNT(*)
		FROM anki_card WHERE deck_id = ?
	`, now, now, deckID).Scan(&stats.NewCount, &stats.LearningCount, &stats.ReviewCount, &stats.DueCount, &stats.TotalCount)

	return stats, err
}

// GetNextAnkiCard returns the next card due for review in a deck
func (d *Database) GetNextAnkiCard(deckID int64) (*AnkiReviewCard, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	now := time.Now().UTC().Format("2006-01-02 15:04:05")

	var card AnkiCard
	err := d.db.QueryRow(`
		SELECT id, deck_id, position_id,
			COALESCE(due, ''), stability, difficulty,
			elapsed_days, scheduled_days, reps, lapses, state,
			COALESCE(last_review, '')
		FROM anki_card
		WHERE deck_id = ? AND due <= ?
		ORDER BY
			CASE WHEN state = 1 OR state = 3 THEN 0
			     WHEN state = 2 THEN 1
			     ELSE 2 END,
			due ASC
		LIMIT 1
	`, deckID, now).Scan(&card.ID, &card.DeckID, &card.PositionID,
		&card.Due, &card.Stability, &card.Difficulty,
		&card.ElapsedDays, &card.ScheduledDays, &card.Reps, &card.Lapses, &card.State,
		&card.LastReview)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	pos, err := d.loadPositionByIDUnlocked(card.PositionID)
	if err != nil {
		return nil, fmt.Errorf("error loading position for card: %w", err)
	}

	return &AnkiReviewCard{
		Card:     card,
		Position: pos,
	}, nil
}

// loadPositionByIDUnlocked loads a position without locking (caller must hold lock)
func (d *Database) loadPositionByIDUnlocked(positionID int64) (Position, error) {
	row := d.db.QueryRow(`SELECT `+positionSelectCols+` FROM position WHERE id = ?`, positionID)
	return scanPositionRow(row)
}

// ReviewAnkiCard submits a review rating for a card and updates its FSRS state
func (d *Database) ReviewAnkiCard(cardID int64, rating int) (*AnkiReviewCard, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	var card AnkiCard
	err := d.db.QueryRow(`
		SELECT id, deck_id, position_id,
			COALESCE(due, ''), stability, difficulty,
			elapsed_days, scheduled_days, reps, lapses, state,
			COALESCE(last_review, '')
		FROM anki_card WHERE id = ?
	`, cardID).Scan(&card.ID, &card.DeckID, &card.PositionID,
		&card.Due, &card.Stability, &card.Difficulty,
		&card.ElapsedDays, &card.ScheduledDays, &card.Reps, &card.Lapses, &card.State,
		&card.LastReview)
	if err != nil {
		return nil, fmt.Errorf("card not found: %w", err)
	}

	var deck AnkiDeck
	var enableFuzz int
	err = d.db.QueryRow(`SELECT id, request_retention, maximum_interval, enable_fuzz FROM anki_deck WHERE id = ?`, card.DeckID).
		Scan(&deck.ID, &deck.RequestRetention, &deck.MaximumInterval, &enableFuzz)
	if err != nil {
		return nil, fmt.Errorf("deck not found: %w", err)
	}
	deck.EnableFuzz = enableFuzz != 0

	fsrsCard := fsrs.Card{
		Stability:     card.Stability,
		Difficulty:    card.Difficulty,
		ElapsedDays:   uint64(card.ElapsedDays),
		ScheduledDays: uint64(card.ScheduledDays),
		Reps:          uint64(card.Reps),
		Lapses:        uint64(card.Lapses),
		State:         fsrs.State(card.State),
	}

	now := time.Now().UTC()
	if card.Due != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", card.Due); err == nil {
			fsrsCard.Due = t
		}
	}
	if card.LastReview != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", card.LastReview); err == nil {
			fsrsCard.LastReview = t
		}
	}

	f := d.newFSRS(deck)
	grade := fsrs.Rating(rating)
	schedulingInfo := f.Next(fsrsCard, now, grade)
	newCard := schedulingInfo.Card

	_, err = d.db.Exec(`
		UPDATE anki_card SET
			due = ?, stability = ?, difficulty = ?,
			elapsed_days = ?, scheduled_days = ?, reps = ?,
			lapses = ?, state = ?, last_review = ?
		WHERE id = ?
	`,
		newCard.Due.UTC().Format("2006-01-02 15:04:05"),
		newCard.Stability, newCard.Difficulty,
		newCard.ElapsedDays, newCard.ScheduledDays, newCard.Reps,
		newCard.Lapses, int(newCard.State),
		now.Format("2006-01-02 15:04:05"),
		cardID)
	if err != nil {
		return nil, fmt.Errorf("error updating card: %w", err)
	}

	var nextCard AnkiCard
	nowStr := now.Format("2006-01-02 15:04:05")
	err = d.db.QueryRow(`
		SELECT id, deck_id, position_id,
			COALESCE(due, ''), stability, difficulty,
			elapsed_days, scheduled_days, reps, lapses, state,
			COALESCE(last_review, '')
		FROM anki_card
		WHERE deck_id = ? AND due <= ?
		ORDER BY
			CASE WHEN state = 1 OR state = 3 THEN 0
			     WHEN state = 2 THEN 1
			     ELSE 2 END,
			due ASC
		LIMIT 1
	`, card.DeckID, nowStr).Scan(&nextCard.ID, &nextCard.DeckID, &nextCard.PositionID,
		&nextCard.Due, &nextCard.Stability, &nextCard.Difficulty,
		&nextCard.ElapsedDays, &nextCard.ScheduledDays, &nextCard.Reps, &nextCard.Lapses, &nextCard.State,
		&nextCard.LastReview)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	pos, err := d.loadPositionByIDUnlocked(nextCard.PositionID)
	if err != nil {
		return nil, err
	}

	return &AnkiReviewCard{
		Card:     nextCard,
		Position: pos,
	}, nil
}

// ResetAnkiDeck resets all cards in a deck to new state
func (d *Database) ResetAnkiDeck(deckID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	_, err := d.db.Exec(`
		UPDATE anki_card SET
			due = ?, stability = 0, difficulty = 0,
			elapsed_days = 0, scheduled_days = 0, reps = 0,
			lapses = 0, state = 0, last_review = ''
		WHERE deck_id = ?
	`, now, deckID)
	return err
}
