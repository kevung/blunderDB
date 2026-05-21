package postgres

import (
	"context"
	"fmt"
	"iter"
	"strings"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type commentStore struct{ db execer }

var _ storage.CommentStore = (*commentStore)(nil)

// commentSelectExpr reads a domain.CommentEntry. created_at is NOT NULL
// (DEFAULT now()); modified_at stays NULL until the first Update.
const commentSelectExpr = `id, position_id, COALESCE(text,''), created_at, modified_at`

func scanCommentEntry(sc scanner) (domain.CommentEntry, error) {
	var e domain.CommentEntry
	var createdAt time.Time
	var modifiedAt *time.Time
	if err := sc.Scan(&e.ID, &e.PositionID, &e.Text, &createdAt, &modifiedAt); err != nil {
		return domain.CommentEntry{}, err
	}
	e.CreatedAt = tsTime(createdAt)
	if modifiedAt != nil {
		e.ModifiedAt = tsTime(*modifiedAt)
	}
	return e, nil
}

// Add appends a new comment entry to a position and returns its id.
func (s *commentStore) Add(ctx context.Context, scope string, positionID int64, text string) (int64, error) {
	var id int64
	if err := s.db.QueryRow(ctx,
		`INSERT INTO comment (tenant_id, position_id, text) VALUES ($1,$2,$3) RETURNING id`,
		tenantID(scope), positionID, text).Scan(&id); err != nil {
		return 0, fmt.Errorf("postgres: add comment: %w", err)
	}
	return id, nil
}

// Update changes the text of the comment entry with the given id.
func (s *commentStore) Update(ctx context.Context, scope string, commentID int64, text string) error {
	if _, err := s.db.Exec(ctx,
		`UPDATE comment SET text = $1, modified_at = now() WHERE id = $2 AND tenant_id = $3`,
		text, commentID, tenantID(scope)); err != nil {
		return fmt.Errorf("postgres: update comment %d: %w", commentID, err)
	}
	return nil
}

// Delete removes a single comment entry by its id.
func (s *commentStore) Delete(ctx context.Context, scope string, commentID int64) error {
	if _, err := s.db.Exec(ctx,
		`DELETE FROM comment WHERE id = $1 AND tenant_id = $2`,
		commentID, tenantID(scope)); err != nil {
		return fmt.Errorf("postgres: delete comment %d: %w", commentID, err)
	}
	return nil
}

// DeleteForPosition removes every comment entry of a position.
func (s *commentStore) DeleteForPosition(ctx context.Context, scope string, positionID int64) error {
	if _, err := s.db.Exec(ctx,
		`DELETE FROM comment WHERE position_id = $1 AND tenant_id = $2`,
		positionID, tenantID(scope)); err != nil {
		return fmt.Errorf("postgres: delete comments of position %d: %w", positionID, err)
	}
	return nil
}

// Text returns the non-empty comment entries of a position joined with blank
// lines, or "" when the position has no comment.
func (s *commentStore) Text(ctx context.Context, scope string, positionID int64) (string, error) {
	rows, err := s.db.Query(ctx,
		`SELECT text FROM comment
		 WHERE position_id = $1 AND tenant_id = $2 AND text != '' ORDER BY id ASC`,
		positionID, tenantID(scope))
	if err != nil {
		return "", fmt.Errorf("postgres: comment text of position %d: %w", positionID, err)
	}
	defer rows.Close()
	var parts []string
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err != nil {
			return "", fmt.Errorf("postgres: comment text of position %d: %w", positionID, err)
		}
		parts = append(parts, text)
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("postgres: comment text of position %d: %w", positionID, err)
	}
	return strings.Join(parts, "\n\n"), nil
}

// commentSeq streams the comment entries returned by query.
func (s *commentStore) commentSeq(ctx context.Context, what, query string, args ...any) iter.Seq2[*domain.CommentEntry, error] {
	return func(yield func(*domain.CommentEntry, error) bool) {
		rows, err := s.db.Query(ctx, query, args...)
		if err != nil {
			yield(nil, fmt.Errorf("postgres: %s: %w", what, err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			e, err := scanCommentEntry(rows)
			if err != nil {
				yield(nil, fmt.Errorf("postgres: %s: %w", what, err))
				return
			}
			if !yield(&e, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("postgres: %s: %w", what, err))
		}
	}
}

// ByPosition streams the non-empty comment entries of a position, most recent
// first.
func (s *commentStore) ByPosition(ctx context.Context, scope string, positionID int64) iter.Seq2[*domain.CommentEntry, error] {
	return s.commentSeq(ctx, "comments by position",
		`SELECT `+commentSelectExpr+` FROM comment
		 WHERE position_id = $1 AND tenant_id = $2 AND text != '' ORDER BY id DESC`,
		positionID, tenantID(scope))
}

// ListAll streams every non-empty comment entry, most recent first.
func (s *commentStore) ListAll(ctx context.Context, scope string) iter.Seq2[*domain.CommentEntry, error] {
	return s.commentSeq(ctx, "list comments",
		`SELECT `+commentSelectExpr+` FROM comment
		 WHERE tenant_id = $1 AND text != '' ORDER BY id DESC`, tenantID(scope))
}

// Search streams non-empty comment entries whose text contains query
// (case-insensitive), most recent first.
func (s *commentStore) Search(ctx context.Context, scope string, query string) iter.Seq2[*domain.CommentEntry, error] {
	return s.commentSeq(ctx, "search comments",
		`SELECT `+commentSelectExpr+` FROM comment
		 WHERE tenant_id = $1 AND text != '' AND text ILIKE '%' || $2 || '%'
		 ORDER BY id DESC`, tenantID(scope), query)
}
