package sqlshared

import (
	"context"
	"fmt"
	"iter"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// CommentStore implements storage.CommentStore. comment is a domain table:
// every statement is confined to the scope's tenant through
// Dialect.TenantFilter / TenantColumns.
type CommentStore struct{ DB Execer }

var _ storage.CommentStore = (*CommentStore)(nil)

// selectCols is the column list that reads a domain.CommentEntry. created_at
// is set by the schema default; modified_at stays NULL until the first Update
// and reads back as "".
func (s *CommentStore) selectCols() string {
	return `id, position_id, COALESCE(text,''), ` +
		s.DB.TimestampText("created_at") + `, ` + s.DB.TimestampText("modified_at")
}

func scanCommentEntry(sc interface{ Scan(...any) error }) (domain.CommentEntry, error) {
	var e domain.CommentEntry
	if err := sc.Scan(&e.ID, &e.PositionID, &e.Text, &e.CreatedAt, &e.ModifiedAt); err != nil {
		return domain.CommentEntry{}, err
	}
	return e, nil
}

// Add appends a new comment entry to a position and returns its id.
func (s *CommentStore) Add(ctx context.Context, scope string, positionID int64, text string) (int64, error) {
	cols, args := s.DB.TenantColumns(scope)
	cols = append(cols, "position_id", "text")
	args = append(args, positionID, text)
	id, err := s.DB.Insert(ctx,
		`INSERT INTO comment (`+strings.Join(cols, ", ")+`) VALUES (`+Placeholders(len(cols))+`)`, args...)
	if err != nil {
		return 0, errf(s.DB, "add comment", s.DB.Referenced(err))
	}
	return id, nil
}

// Update changes the text of the comment entry with the given id.
func (s *CommentStore) Update(ctx context.Context, scope string, commentID int64, text string) error {
	tenant, targs := s.DB.TenantFilter("", scope)
	if _, err := s.DB.Exec(ctx,
		`UPDATE comment SET text = ?, modified_at = CURRENT_TIMESTAMP WHERE id = ? AND `+tenant,
		append([]any{text, commentID}, targs...)...); err != nil {
		return errf(s.DB, fmt.Sprintf("update comment %d", commentID), err)
	}
	return nil
}

// Delete removes a single comment entry by its id.
func (s *CommentStore) Delete(ctx context.Context, scope string, commentID int64) error {
	tenant, targs := s.DB.TenantFilter("", scope)
	if _, err := s.DB.Exec(ctx,
		`DELETE FROM comment WHERE id = ? AND `+tenant,
		append([]any{commentID}, targs...)...); err != nil {
		return errf(s.DB, fmt.Sprintf("delete comment %d", commentID), err)
	}
	return nil
}

// DeleteForPosition removes every comment entry of a position.
func (s *CommentStore) DeleteForPosition(ctx context.Context, scope string, positionID int64) error {
	tenant, targs := s.DB.TenantFilter("", scope)
	if _, err := s.DB.Exec(ctx,
		`DELETE FROM comment WHERE position_id = ? AND `+tenant,
		append([]any{positionID}, targs...)...); err != nil {
		return errf(s.DB, fmt.Sprintf("delete comments of position %d", positionID), err)
	}
	return nil
}

// Text returns the non-empty comment entries of a position joined with blank
// lines, or "" when the position has no comment.
func (s *CommentStore) Text(ctx context.Context, scope string, positionID int64) (string, error) {
	tenant, targs := s.DB.TenantFilter("", scope)
	rows, err := s.DB.Query(ctx,
		`SELECT text FROM comment WHERE position_id = ? AND `+tenant+` AND text != '' ORDER BY id ASC`,
		append([]any{positionID}, targs...)...)
	if err != nil {
		return "", errf(s.DB, fmt.Sprintf("comment text of position %d", positionID), err)
	}
	defer rows.Close()
	var parts []string
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err != nil {
			return "", errf(s.DB, fmt.Sprintf("comment text of position %d", positionID), err)
		}
		parts = append(parts, text)
	}
	if err := rows.Err(); err != nil {
		return "", errf(s.DB, fmt.Sprintf("comment text of position %d", positionID), err)
	}
	return strings.Join(parts, "\n\n"), nil
}

// commentSeq streams the comment entries returned by query.
func (s *CommentStore) commentSeq(ctx context.Context, what, query string, args ...any) iter.Seq2[*domain.CommentEntry, error] {
	return func(yield func(*domain.CommentEntry, error) bool) {
		rows, err := s.DB.Query(ctx, query, args...)
		if err != nil {
			yield(nil, errf(s.DB, what, err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			e, err := scanCommentEntry(rows)
			if err != nil {
				yield(nil, errf(s.DB, what, err))
				return
			}
			if !yield(&e, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, errf(s.DB, what, err))
		}
	}
}

// ByPosition streams the non-empty comment entries of a position, most recent
// first.
func (s *CommentStore) ByPosition(ctx context.Context, scope string, positionID int64) iter.Seq2[*domain.CommentEntry, error] {
	tenant, targs := s.DB.TenantFilter("", scope)
	return s.commentSeq(ctx, "comments by position",
		`SELECT `+s.selectCols()+` FROM comment
		 WHERE position_id = ? AND `+tenant+` AND text != '' ORDER BY id DESC`,
		append([]any{positionID}, targs...)...)
}

// ListAll streams every non-empty comment entry, most recent first.
func (s *CommentStore) ListAll(ctx context.Context, scope string) iter.Seq2[*domain.CommentEntry, error] {
	tenant, targs := s.DB.TenantFilter("", scope)
	return s.commentSeq(ctx, "list comments",
		`SELECT `+s.selectCols()+` FROM comment WHERE `+tenant+` AND text != '' ORDER BY id DESC`,
		targs...)
}

// Search streams non-empty comment entries whose text contains query
// (case-insensitive), most recent first.
func (s *CommentStore) Search(ctx context.Context, scope string, query string) iter.Seq2[*domain.CommentEntry, error] {
	tenant, targs := s.DB.TenantFilter("", scope)
	return s.commentSeq(ctx, "search comments",
		`SELECT `+s.selectCols()+` FROM comment
		 WHERE `+tenant+` AND text != '' AND text `+s.DB.ILike()+` '%' || ? || '%' ORDER BY id DESC`,
		append(targs, query)...)
}
