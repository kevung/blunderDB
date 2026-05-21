package sqlite

import (
	"context"
	"fmt"
	"iter"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type commentStore struct{ db execer }

var _ storage.CommentStore = (*commentStore)(nil)

// commentSelectCols reads a domain.CommentEntry.
const commentSelectCols = `id, position_id, COALESCE(text,''),
	COALESCE(created_at,''), COALESCE(modified_at,'')`

func scanCommentEntry(sc interface{ Scan(...any) error }) (domain.CommentEntry, error) {
	var e domain.CommentEntry
	if err := sc.Scan(&e.ID, &e.PositionID, &e.Text, &e.CreatedAt, &e.ModifiedAt); err != nil {
		return domain.CommentEntry{}, err
	}
	return e, nil
}

// Add appends a new comment entry to a position and returns its id.
func (s *commentStore) Add(ctx context.Context, scope string, positionID int64, text string) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO comment (position_id, text) VALUES (?,?)`, positionID, text)
	if err != nil {
		return 0, fmt.Errorf("sqlite: add comment: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("sqlite: add comment id: %w", err)
	}
	return id, nil
}

// Update changes the text of the comment entry with the given id.
func (s *commentStore) Update(ctx context.Context, scope string, commentID int64, text string) error {
	if _, err := s.db.ExecContext(ctx,
		`UPDATE comment SET text = ?, modified_at = CURRENT_TIMESTAMP WHERE id = ?`,
		text, commentID); err != nil {
		return fmt.Errorf("sqlite: update comment %d: %w", commentID, err)
	}
	return nil
}

// Delete removes a single comment entry by its id.
func (s *commentStore) Delete(ctx context.Context, scope string, commentID int64) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM comment WHERE id = ?`, commentID); err != nil {
		return fmt.Errorf("sqlite: delete comment %d: %w", commentID, err)
	}
	return nil
}

// DeleteForPosition removes every comment entry of a position.
func (s *commentStore) DeleteForPosition(ctx context.Context, scope string, positionID int64) error {
	if _, err := s.db.ExecContext(ctx,
		`DELETE FROM comment WHERE position_id = ?`, positionID); err != nil {
		return fmt.Errorf("sqlite: delete comments of position %d: %w", positionID, err)
	}
	return nil
}

// Text returns the non-empty comment entries of a position joined with blank
// lines, or "" when the position has no comment.
func (s *commentStore) Text(ctx context.Context, scope string, positionID int64) (string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT text FROM comment WHERE position_id = ? AND text != '' ORDER BY id ASC`,
		positionID)
	if err != nil {
		return "", fmt.Errorf("sqlite: comment text of position %d: %w", positionID, err)
	}
	defer rows.Close()
	var parts []string
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err != nil {
			return "", fmt.Errorf("sqlite: comment text of position %d: %w", positionID, err)
		}
		parts = append(parts, text)
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("sqlite: comment text of position %d: %w", positionID, err)
	}
	return strings.Join(parts, "\n\n"), nil
}

// commentSeq streams the comment entries returned by query.
func (s *commentStore) commentSeq(ctx context.Context, what, query string, args ...any) iter.Seq2[*domain.CommentEntry, error] {
	return func(yield func(*domain.CommentEntry, error) bool) {
		rows, err := s.db.QueryContext(ctx, query, args...)
		if err != nil {
			yield(nil, fmt.Errorf("sqlite: %s: %w", what, err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			e, err := scanCommentEntry(rows)
			if err != nil {
				yield(nil, fmt.Errorf("sqlite: %s: %w", what, err))
				return
			}
			if !yield(&e, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("sqlite: %s: %w", what, err))
		}
	}
}

// ByPosition streams the non-empty comment entries of a position, most recent
// first.
func (s *commentStore) ByPosition(ctx context.Context, scope string, positionID int64) iter.Seq2[*domain.CommentEntry, error] {
	return s.commentSeq(ctx, "comments by position",
		`SELECT `+commentSelectCols+` FROM comment
		 WHERE position_id = ? AND text != '' ORDER BY id DESC`, positionID)
}

// ListAll streams every non-empty comment entry, most recent first.
func (s *commentStore) ListAll(ctx context.Context, scope string) iter.Seq2[*domain.CommentEntry, error] {
	return s.commentSeq(ctx, "list comments",
		`SELECT `+commentSelectCols+` FROM comment WHERE text != '' ORDER BY id DESC`)
}

// Search streams non-empty comment entries whose text contains query
// (case-insensitive), most recent first.
func (s *commentStore) Search(ctx context.Context, scope string, query string) iter.Seq2[*domain.CommentEntry, error] {
	return s.commentSeq(ctx, "search comments",
		`SELECT `+commentSelectCols+` FROM comment
		 WHERE text != '' AND text LIKE '%' || ? || '%' ORDER BY id DESC`, query)
}
