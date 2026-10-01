package sqlshared

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// TranscriptionStore implements storage.TranscriptionStore over the
// transcription table. transcription is a domain table — it points at match —
// so every statement is confined to the scope's tenant through
// Dialect.TenantFilter / TenantColumns. The document is one TEXT column the
// store never looks inside.
type TranscriptionStore struct{ DB Execer }

var _ storage.TranscriptionStore = (*TranscriptionStore)(nil)

// selectCols reads a storage.Transcription. match_id is NULL while the draft
// has produced no match; it reads back as 0.
func (s *TranscriptionStore) selectCols() string {
	return `id, ` + s.DB.TimestampText("created_at") + `, ` + s.DB.TimestampText("updated_at") +
		`, COALESCE(format_version,''), COALESCE(match_id, 0), COALESCE(label,''), COALESCE(document,''), revision`
}

func scanTranscription(sc interface{ Scan(...any) error }) (*storage.Transcription, error) {
	var t storage.Transcription
	if err := sc.Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt, &t.FormatVersion, &t.MatchID, &t.Label, &t.Document, &t.Revision); err != nil {
		return nil, err
	}
	return &t, nil
}

// List streams the scope's drafts, most recently updated first. The id breaks
// ties so the order is total: two drafts saved in the same second (the
// resolution SQLite's CURRENT_TIMESTAMP has) would otherwise come back in
// whatever order the backend felt like.
func (s *TranscriptionStore) List(ctx context.Context, scope string) iter.Seq2[*storage.Transcription, error] {
	return func(yield func(*storage.Transcription, error) bool) {
		tenant, targs := s.DB.TenantFilter("", scope)
		rows, err := s.DB.Query(ctx,
			`SELECT `+s.selectCols()+` FROM transcription WHERE `+tenant+
				` ORDER BY updated_at DESC, id DESC`, targs...)
		if err != nil {
			yield(nil, errf(s.DB, "list transcriptions", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			t, err := scanTranscription(rows)
			if err != nil {
				yield(nil, errf(s.DB, "list transcriptions", err))
				return
			}
			if !yield(t, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, errf(s.DB, "list transcriptions", err))
		}
	}
}

// Get returns one draft, or ErrNotFound.
func (s *TranscriptionStore) Get(ctx context.Context, scope string, id int64) (*storage.Transcription, error) {
	what := fmt.Sprintf("get transcription %d", id)
	tenant, targs := s.DB.TenantFilter("", scope)
	t, err := scanTranscription(s.DB.QueryRow(ctx,
		`SELECT `+s.selectCols()+` FROM transcription WHERE id = ? AND `+tenant,
		append([]any{id}, targs...)...))
	if errors.Is(err, ErrNoRows) {
		return nil, errf(s.DB, what, storage.ErrNotFound)
	}
	if err != nil {
		return nil, errf(s.DB, what, err)
	}
	return t, nil
}

// Save inserts t when it carries no id and rewrites the row in place
// otherwise, touching updated_at either way, and returns the row's id.
func (s *TranscriptionStore) Save(ctx context.Context, scope string, t *storage.Transcription) (int64, error) {
	if t == nil {
		return 0, errf(s.DB, "save transcription", storage.ErrInvalid)
	}
	if t.ID != 0 {
		return t.ID, s.update(ctx, scope, t)
	}
	cols, args := s.DB.TenantColumns(scope)
	cols = append(cols, "format_version", "match_id", "label", "document", "revision")
	args = append(args, t.FormatVersion, nullableID(t.MatchID), t.Label, t.Document, 1)
	id, err := s.DB.Insert(ctx,
		`INSERT INTO transcription (`+strings.Join(cols, ", ")+`) VALUES (`+Placeholders(len(cols))+`)`, args...)
	if err != nil {
		return 0, errf(s.DB, "save transcription", s.DB.Referenced(err))
	}
	t.Revision = 1
	return id, nil
}

// update rewrites an existing draft under t.Revision's expectation.
func (s *TranscriptionStore) update(ctx context.Context, scope string, t *storage.Transcription) error {
	what := fmt.Sprintf("save transcription %d", t.ID)
	tenant, targs := s.DB.TenantFilter("", scope)
	args := append([]any{t.FormatVersion, nullableID(t.MatchID), t.Label, t.Document, t.ID, t.Revision, t.Revision}, targs...)
	n, err := s.DB.Exec(ctx,
		`UPDATE transcription
		 SET format_version = ?, match_id = ?, label = ?, document = ?,
		     revision = revision + 1, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND (CAST(? AS BIGINT) = 0 OR revision = ?) AND `+tenant, args...)
	if err != nil {
		return errf(s.DB, what, s.DB.Referenced(err))
	}
	if n == 0 {
		return errf(s.DB, what, s.missOrStale(ctx, scope, t.ID))
	}
	rev, err := s.revision(ctx, scope, t.ID)
	if err != nil {
		return errf(s.DB, what, err)
	}
	t.Revision = rev
	return nil
}

// Touch advances an unchanged draft's revision under the caller's
// expectation (0: none) and returns the new one.
func (s *TranscriptionStore) Touch(ctx context.Context, scope string, id, revision int64) (int64, error) {
	what := fmt.Sprintf("touch transcription %d", id)
	tenant, targs := s.DB.TenantFilter("", scope)
	n, err := s.DB.Exec(ctx,
		`UPDATE transcription SET revision = revision + 1
		 WHERE id = ? AND (CAST(? AS BIGINT) = 0 OR revision = ?) AND `+tenant,
		append([]any{id, revision, revision}, targs...)...)
	if err != nil {
		return 0, errf(s.DB, what, err)
	}
	if n == 0 {
		return 0, errf(s.DB, what, s.missOrStale(ctx, scope, id))
	}
	rev, err := s.revision(ctx, scope, id)
	if err != nil {
		return 0, errf(s.DB, what, err)
	}
	return rev, nil
}

// revision reads a row's revision back after a write. The write and this read
// may straddle another writer's; the value is then that writer's, which the
// caller's next expectation will fail against — a 409 rather than a lost
// write.
func (s *TranscriptionStore) revision(ctx context.Context, scope string, id int64) (int64, error) {
	tenant, targs := s.DB.TenantFilter("", scope)
	var rev int64
	err := s.DB.QueryRow(ctx, `SELECT revision FROM transcription WHERE id = ? AND `+tenant,
		append([]any{id}, targs...)...).Scan(&rev)
	if errors.Is(err, ErrNoRows) {
		return 0, storage.ErrNotFound
	}
	return rev, err
}

// missOrStale tells why a conditional write touched no row: the draft is gone
// (ErrNotFound) or another writer moved it on (ErrConflict).
func (s *TranscriptionStore) missOrStale(ctx context.Context, scope string, id int64) error {
	if _, err := s.revision(ctx, scope, id); err != nil {
		return err
	}
	return storage.ErrConflict
}

// Annotations counts the analyses (position and move) and comments a Match's
// positions and moves hold. Every table is confined to the scope.
func (s *TranscriptionStore) Annotations(ctx context.Context, scope string, matchID int64) (int, int, error) {
	what := fmt.Sprintf("annotations of match %d", matchID)
	tg, ga := s.DB.TenantFilter("g", scope)
	ta, aa := s.DB.TenantFilter("a", scope)
	tma, maa := s.DB.TenantFilter("ma", scope)
	tc, ca := s.DB.TenantFilter("c", scope)
	positions := `SELECT mv.position_id FROM move mv JOIN game g ON g.id = mv.game_id WHERE g.match_id = ? AND ` + tg
	var args []any
	args = append(args, matchID)
	args = append(args, ga...)
	args = append(args, aa...)
	args = append(args, matchID)
	args = append(args, ga...)
	args = append(args, maa...)
	args = append(args, matchID)
	args = append(args, ga...)
	args = append(args, ca...)
	var analyses, comments int
	err := s.DB.QueryRow(ctx, `
		SELECT (SELECT COUNT(*) FROM analysis a WHERE a.position_id IN (`+positions+`) AND `+ta+`)
		     + (SELECT COUNT(*) FROM move_analysis ma JOIN move mv ON mv.id = ma.move_id
		                                              JOIN game g ON g.id = mv.game_id
		         WHERE g.match_id = ? AND `+tg+` AND `+tma+`),
		       (SELECT COUNT(*) FROM comment c WHERE c.position_id IN (`+positions+`) AND `+tc+`)`,
		args...).Scan(&analyses, &comments)
	if err != nil {
		return 0, 0, errf(s.DB, what, err)
	}
	return analyses, comments, nil
}

// Delete removes a draft, or reports ErrNotFound.
func (s *TranscriptionStore) Delete(ctx context.Context, scope string, id int64) error {
	what := fmt.Sprintf("delete transcription %d", id)
	tenant, targs := s.DB.TenantFilter("", scope)
	n, err := s.DB.Exec(ctx,
		`DELETE FROM transcription WHERE id = ? AND `+tenant, append([]any{id}, targs...)...)
	if err != nil {
		return errf(s.DB, what, err)
	}
	if n == 0 {
		return errf(s.DB, what, storage.ErrNotFound)
	}
	return nil
}

// nullableID binds 0 as NULL: "this draft has produced no match" is the
// absence of a reference, not a reference to match 0, and a foreign key would
// reject the latter.
func nullableID(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}
