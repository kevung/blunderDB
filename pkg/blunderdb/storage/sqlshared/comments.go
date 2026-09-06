package sqlshared

import (
	"context"
	"fmt"
	"iter"
	"sort"
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
		s.DB.TimestampText("created_at") + `, ` + s.DB.TimestampText("modified_at") +
		`, COALESCE(origin,'unknown')`
}

func scanCommentEntry(sc interface{ Scan(...any) error }) (domain.CommentEntry, error) {
	var e domain.CommentEntry
	var origin string
	if err := sc.Scan(&e.ID, &e.PositionID, &e.Text, &e.CreatedAt, &e.ModifiedAt, &origin); err != nil {
		return domain.CommentEntry{}, err
	}
	e.Origin = domain.ParseCommentOrigin(origin)
	return e, nil
}

// Add appends a comment the user wrote and returns its id.
func (s *CommentStore) Add(ctx context.Context, scope string, positionID int64, text string) (int64, error) {
	return s.AddFrom(ctx, scope, positionID, text, domain.CommentOriginUser)
}

// AddFrom appends a comment entry carrying its provenance and returns its id.
func (s *CommentStore) AddFrom(ctx context.Context, scope string, positionID int64, text string, origin domain.CommentOrigin) (int64, error) {
	cols, args := s.DB.TenantColumns(scope)
	cols = append(cols, "position_id", "text", "origin")
	args = append(args, positionID, text, string(domain.ParseCommentOrigin(string(origin))))
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
			// The text becomes the user's, so the row's provenance does
			// too: an imported note the user rewrites is no longer the
			// importer's sentence, and the purge that spares user comments
			// must spare this one.
			_, err := tx.Exec(ctx,
				`UPDATE comment SET text = ?, origin = ?, modified_at = CURRENT_TIMESTAMP WHERE id = ? AND `+tenant,
				append([]any{text, string(domain.CommentOriginUser), id}, targs...)...)
			return err
		}
		cols, args := tx.TenantColumns(scope)
		cols = append(cols, "position_id", "text", "origin")
		args = append(args, positionID, text, string(domain.CommentOriginUser))
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

// Update changes the text of the comment entry with the given id. The entry
// becomes the user's, whoever wrote it first: they have rewritten it.
func (s *CommentStore) Update(ctx context.Context, scope string, commentID int64, text string) error {
	tenant, targs := s.DB.TenantFilter("", scope)
	if _, err := s.DB.Exec(ctx,
		`UPDATE comment SET text = ?, origin = ?, modified_at = CURRENT_TIMESTAMP WHERE id = ? AND `+tenant,
		append([]any{text, string(domain.CommentOriginUser), commentID}, targs...)...); err != nil {
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

// Tags — see storage.CommentStore.
//
// One query, tallied in Go. A tag cannot be found by a GROUP BY: it is a
// `#word` inside prose, nothing declares it, and no column holds it. So the
// comment text of the tenant is read once and domain.ExtractTags does the
// rest — the same extraction the per-tag statistics use, so the vocabulary
// panel and the statistics can never disagree about what a tag is.
//
// Only rows carrying a '#' at all are read: a library where nobody uses tags
// costs one index-less scan of a column, and one where everybody does reads
// only the comments that can possibly contribute.
func (s *CommentStore) Tags(ctx context.Context, scope string) ([]domain.TagCount, error) {
	tenant, targs := s.DB.TenantFilter("", scope)
	rows, err := s.DB.Query(ctx,
		`SELECT position_id, COALESCE(text,'') FROM comment
		 WHERE `+tenant+` AND text `+s.DB.ILike()+` '%#%'`, targs...)
	if err != nil {
		return nil, errf(s.DB, "read tags", err)
	}
	defer rows.Close()

	// A tag counts a POSITION once, however many of its comments carry it.
	seen := map[string]map[int64]bool{}
	for rows.Next() {
		var positionID int64
		var text string
		if err := rows.Scan(&positionID, &text); err != nil {
			return nil, errf(s.DB, "scan tags", err)
		}
		for _, tag := range domain.ExtractTags(text) {
			if seen[tag] == nil {
				seen[tag] = map[int64]bool{}
			}
			seen[tag][positionID] = true
		}
	}
	if err := rows.Err(); err != nil {
		return nil, errf(s.DB, "read tags", err)
	}

	out := make([]domain.TagCount, 0, len(seen))
	for tag, positions := range seen {
		out = append(out, domain.TagCount{Tag: tag, Count: len(positions)})
	}
	// Most used first, alphabetically within a count: the panel is read to
	// find the tags one actually uses, and a stable order is what lets two
	// runs be compared.
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Tag < out[j].Tag
	})
	return out, nil
}
