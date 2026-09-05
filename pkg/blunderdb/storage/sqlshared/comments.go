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

// Upsert rewrites the oldest non-empty comment entry of a position, or adds
// one when the position has none, and returns the entry's id. It is the
// single-comment view the desktop wrapper's SaveComment/LoadComment pair
// exposes: a load-edit-save cycle must land on the same row, never append.
func (s *CommentStore) Upsert(ctx context.Context, scope string, positionID int64, text string) (int64, error) {
	what := fmt.Sprintf("upsert comment of position %d", positionID)
	var id int64
	err := s.DB.Transact(ctx, func(tx Execer) error {
		tenant, targs := tx.TenantFilter("", scope)
		var oldest *int64
		if err := tx.QueryRow(ctx,
			`SELECT MIN(id) FROM comment WHERE position_id = ? AND text != '' AND `+tenant,
			append([]any{positionID}, targs...)...).Scan(&oldest); err != nil {
			return err
		}
		if oldest != nil {
			id = *oldest
			_, err := tx.Exec(ctx,
				`UPDATE comment SET text = ?, modified_at = CURRENT_TIMESTAMP WHERE id = ? AND `+tenant,
				append([]any{text, id}, targs...)...)
			return err
		}
		cols, args := tx.TenantColumns(scope)
		cols = append(cols, "position_id", "text")
		args = append(args, positionID, text)
		newID, err := tx.Insert(ctx,
			`INSERT INTO comment (`+strings.Join(cols, ", ")+`) VALUES (`+Placeholders(len(cols))+`)`, args...)
		if err != nil {
			return err
		}
		id = newID
		return nil
	})
	if err != nil {
		return 0, errf(s.DB, what, s.DB.Referenced(err))
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

// byPositionsChunk bounds one IN (...) list, leaving headroom under
// SQLite's bound-variable limit (999 on older builds) for the query's tenant
// placeholder.
const byPositionsChunk = 900

// ByPositions returns the non-empty comments of the given positions, keyed by
// position id and oldest first within a position — see storage.CommentStore.
func (s *CommentStore) ByPositions(ctx context.Context, scope string, positionIDs []int64) (map[int64][]*domain.CommentEntry, error) {
	out := make(map[int64][]*domain.CommentEntry)
	if len(positionIDs) == 0 {
		return out, nil
	}
	tenant, targs := s.DB.TenantFilter("", scope)
	for start := 0; start < len(positionIDs); start += byPositionsChunk {
		batch := positionIDs[start:min(start+byPositionsChunk, len(positionIDs))]
		args := make([]any, 0, len(batch)+len(targs))
		for _, id := range batch {
			args = append(args, id)
		}
		args = append(args, targs...)
		rows, err := s.DB.Query(ctx,
			`SELECT `+s.selectCols()+` FROM comment
			 WHERE position_id IN (`+Placeholders(len(batch))+`) AND `+tenant+` AND text != '' ORDER BY id ASC`,
			args...)
		if err != nil {
			return nil, errf(s.DB, "comments by positions", err)
		}
		for rows.Next() {
			e, err := scanCommentEntry(rows)
			if err != nil {
				rows.Close()
				return nil, errf(s.DB, "comments by positions", err)
			}
			out[e.PositionID] = append(out[e.PositionID], &e)
		}
		rerr := rows.Err()
		rows.Close()
		if rerr != nil {
			return nil, errf(s.DB, "comments by positions", rerr)
		}
	}
	return out, nil
}

// ListAll streams every non-empty comment entry, most recent first, bounded by
// opts.
//
// The bound is the caller's, and there is no default: a stream is not held in
// memory, so an unbounded one costs time and bandwidth but never the server's
// footing. A silent default limit would be worse than a slow answer — the
// client would read a truncated list believing it complete. What opts buys is
// the ability to PAGE, for a client that wants to (issue #237).
func (s *CommentStore) ListAll(ctx context.Context, scope string, opts storage.ListOpts) iter.Seq2[*domain.CommentEntry, error] {
	tenant, targs := s.DB.TenantFilter("", scope)
	limit, largs := s.DB.LimitOffset(opts.Limit, opts.Offset)
	return s.commentSeq(ctx, "list comments",
		`SELECT `+s.selectCols()+` FROM comment WHERE `+tenant+` AND text != '' ORDER BY id DESC`+limit,
		append(targs, largs...)...)
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
