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
// Dialect.TenantFilter / TenantColumns, exactly like CommentStore.
//
// The SQL is the same on both backends: the document is one TEXT column and
// the store never looks inside it, so there is nothing here for a dialect to
// disagree about beyond the tenant and the timestamp rendering.
type TranscriptionStore struct{ DB Execer }

var _ storage.TranscriptionStore = (*TranscriptionStore)(nil)

// selectCols reads a storage.Transcription. match_id is NULL while the draft
// has produced no match; it reads back as 0.
func (s *TranscriptionStore) selectCols() string {
	return `id, ` + s.DB.TimestampText("created_at") + `, ` + s.DB.TimestampText("updated_at") +
		`, COALESCE(format_version,''), COALESCE(match_id, 0), COALESCE(label,''), COALESCE(document,'')`
}

func scanTranscription(sc interface{ Scan(...any) error }) (*storage.Transcription, error) {
	var t storage.Transcription
	if err := sc.Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt, &t.FormatVersion, &t.MatchID, &t.Label, &t.Document); err != nil {
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
	cols = append(cols, "format_version", "match_id", "label", "document")
	args = append(args, t.FormatVersion, nullableID(t.MatchID), t.Label, t.Document)
	id, err := s.DB.Insert(ctx,
		`INSERT INTO transcription (`+strings.Join(cols, ", ")+`) VALUES (`+Placeholders(len(cols))+`)`, args...)
	if err != nil {
		return 0, errf(s.DB, "save transcription", s.DB.Referenced(err))
	}
	return id, nil
}

// update rewrites an existing draft, or reports ErrNotFound.
func (s *TranscriptionStore) update(ctx context.Context, scope string, t *storage.Transcription) error {
	what := fmt.Sprintf("save transcription %d", t.ID)
	tenant, targs := s.DB.TenantFilter("", scope)
	args := append([]any{t.FormatVersion, nullableID(t.MatchID), t.Label, t.Document, t.ID}, targs...)
	n, err := s.DB.Exec(ctx,
		`UPDATE transcription
		 SET format_version = ?, match_id = ?, label = ?, document = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND `+tenant, args...)
	if err != nil {
		return errf(s.DB, what, s.DB.Referenced(err))
	}
	if n == 0 {
		return errf(s.DB, what, storage.ErrNotFound)
	}
	return nil
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
