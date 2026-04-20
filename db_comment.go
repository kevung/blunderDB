package main

import (
	"database/sql"
)

func (d *Database) DeleteComment(positionID int64) error {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	_, err := d.db.Exec(`DELETE FROM comment WHERE position_id = ?`, positionID)
	if err != nil {
		return err
	}
	return nil
}

// AddComment inserts a new comment entry for a position (allows multiple per position)
func (d *Database) AddComment(positionID int64, text string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec(`INSERT INTO comment (position_id, text) VALUES (?, ?)`, positionID, text)
	if err != nil {
		return err
	}
	return nil
}

// UpdateCommentEntry updates a specific comment by its ID
func (d *Database) UpdateCommentEntry(commentID int64, text string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.commentTableHasTimestamps() {
		_, err := d.db.Exec(`UPDATE comment SET text = ?, modified_at = CURRENT_TIMESTAMP WHERE id = ?`, text, commentID)
		if err != nil {
			return err
		}
	} else {
		_, err := d.db.Exec(`UPDATE comment SET text = ? WHERE id = ?`, text, commentID)
		if err != nil {
			return err
		}
	}
	return nil
}

// DeleteCommentEntry deletes a specific comment by its ID
func (d *Database) DeleteCommentEntry(commentID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec(`DELETE FROM comment WHERE id = ?`, commentID)
	if err != nil {
		return err
	}
	return nil
}

// SaveComment saves a comment for a given position ID
func (d *Database) SaveComment(positionID int64, text string) error {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	// Check if a comment already exists for the given position ID
	var existingID int64
	err := d.db.QueryRow(`SELECT id FROM comment WHERE position_id = ?`, positionID).Scan(&existingID)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if existingID > 0 {
		// Update the existing comment
		_, err = d.db.Exec(`UPDATE comment SET text = ? WHERE id = ?`, text, existingID)
		if err != nil {
			return err
		}
	} else {
		// Insert a new comment
		_, err = d.db.Exec(`INSERT INTO comment (position_id, text) VALUES (?, ?)`, positionID, text)
		if err != nil {
			return err
		}
	}

	return nil
}

// LoadComment loads a comment for a given position ID
func (d *Database) LoadComment(positionID int64) (string, error) {
	d.mu.RLock()         // Lock the mutex
	defer d.mu.RUnlock() // Unlock the mutex when the function returns

	var text string
	err := d.db.QueryRow(`SELECT text FROM comment WHERE position_id = ?`, positionID).Scan(&text)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // No comment found
		}
		return "", err
	}
	return text, nil
}

// commentTableHasTimestamps checks whether the comment table has created_at/modified_at columns.
func (d *Database) commentTableHasTimestamps() bool {
	var dummy string
	err := d.db.QueryRow(`SELECT COALESCE(created_at, '') FROM comment LIMIT 1`).Scan(&dummy)
	// If the column doesn't exist, the error message contains "no such column"
	if err != nil && err != sql.ErrNoRows {
		return false
	}
	return true
}

// GetCommentsByPosition returns all non-empty comments for a given position, ordered by comment ID descending
func (d *Database) GetCommentsByPosition(positionID int64) ([]CommentEntry, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var rows *sql.Rows
	var err error
	hasTS := d.commentTableHasTimestamps()
	if hasTS {
		rows, err = d.db.Query(`SELECT id, position_id, text, COALESCE(created_at, ''), COALESCE(modified_at, '') FROM comment WHERE position_id = ? AND text != '' ORDER BY id DESC`, positionID)
	} else {
		rows, err = d.db.Query(`SELECT id, position_id, text FROM comment WHERE position_id = ? AND text != '' ORDER BY id DESC`, positionID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []CommentEntry
	for rows.Next() {
		var e CommentEntry
		if hasTS {
			if err := rows.Scan(&e.ID, &e.PositionID, &e.Text, &e.CreatedAt, &e.ModifiedAt); err != nil {
				return nil, err
			}
		} else {
			if err := rows.Scan(&e.ID, &e.PositionID, &e.Text); err != nil {
				return nil, err
			}
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// GetAllComments returns all non-empty comments, ordered by comment ID descending (most recent first)
func (d *Database) GetAllComments() ([]CommentEntry, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var rows *sql.Rows
	var err error
	hasTS := d.commentTableHasTimestamps()
	if hasTS {
		rows, err = d.db.Query(`SELECT id, position_id, text, COALESCE(created_at, ''), COALESCE(modified_at, '') FROM comment WHERE text != '' ORDER BY id DESC`)
	} else {
		rows, err = d.db.Query(`SELECT id, position_id, text FROM comment WHERE text != '' ORDER BY id DESC`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []CommentEntry
	for rows.Next() {
		var e CommentEntry
		if hasTS {
			if err := rows.Scan(&e.ID, &e.PositionID, &e.Text, &e.CreatedAt, &e.ModifiedAt); err != nil {
				return nil, err
			}
		} else {
			if err := rows.Scan(&e.ID, &e.PositionID, &e.Text); err != nil {
				return nil, err
			}
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// SearchComments searches for comments containing the given query string (case-insensitive)
func (d *Database) SearchComments(query string) ([]CommentEntry, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var rows *sql.Rows
	var err error
	hasTS := d.commentTableHasTimestamps()
	if hasTS {
		rows, err = d.db.Query(`SELECT id, position_id, text, COALESCE(created_at, ''), COALESCE(modified_at, '') FROM comment WHERE text != '' AND text LIKE '%' || ? || '%' ORDER BY id DESC`, query)
	} else {
		rows, err = d.db.Query(`SELECT id, position_id, text FROM comment WHERE text != '' AND text LIKE '%' || ? || '%' ORDER BY id DESC`, query)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []CommentEntry
	for rows.Next() {
		var e CommentEntry
		if hasTS {
			if err := rows.Scan(&e.ID, &e.PositionID, &e.Text, &e.CreatedAt, &e.ModifiedAt); err != nil {
				return nil, err
			}
		} else {
			if err := rows.Scan(&e.ID, &e.PositionID, &e.Text); err != nil {
				return nil, err
			}
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
