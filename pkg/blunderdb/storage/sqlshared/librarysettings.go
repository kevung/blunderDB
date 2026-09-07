package sqlshared

import (
	"context"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// LibrarySettingsStore implements storage.LibrarySettingsStore over the two
// key/value rows of ADR-0046. The SQL is one statement per operation on both
// backends; only the table and whether it is scoped differ, and the Dialect
// answers that (LibrarySettingsTable):
//
//   - SQLite writes the rows to the metadata table, which the file already
//     carries — no schema change, and the settings sit next to the Performance
//     Rating objective, where `blunderdb info` and `edit` find them.
//   - PostgreSQL writes them to library_settings, its own tenant-scoped table:
//     metadata there is database infrastructure, global to every tenant and
//     outside Row-Level Security, and has been read-only since #156.
type LibrarySettingsStore struct{ DB Execer }

var _ storage.LibrarySettingsStore = (*LibrarySettingsStore)(nil)

// Load returns the scope's settings. A scope that never stored any reads as
// the defaults, and so does a half-written or hand-edited pair — see
// storage.ParseLibrarySettings.
func (s *LibrarySettingsStore) Load(ctx context.Context, scope string) (storage.LibrarySettings, error) {
	table, scoped := s.DB.LibrarySettingsTable()
	query := `SELECT key, COALESCE(value,'') FROM ` + table + ` WHERE `
	args := []any{}
	if scoped {
		query += s.DB.ScopeColumn() + ` = ? AND `
		args = append(args, s.DB.ScopeArg(scope))
	}
	query += `key IN (?,?)`
	args = append(args, storage.LibrarySettingErrorKey, storage.LibrarySettingBlunderKey)

	rows, err := s.DB.Query(ctx, query, args...)
	if err != nil {
		return storage.LibrarySettings{}, errf(s.DB, "load library settings", err)
	}
	defer rows.Close()
	kv := make(map[string]string, 2)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return storage.LibrarySettings{}, errf(s.DB, "load library settings", err)
		}
		kv[key] = value
	}
	if err := rows.Err(); err != nil {
		return storage.LibrarySettings{}, errf(s.DB, "load library settings", err)
	}
	return storage.ParseLibrarySettings(kv), nil
}

// Save records the pair, rejecting one Validate refuses. The two rows are
// written in one transaction: a library is never left with an error threshold
// above its blunder threshold because a write stopped halfway.
func (s *LibrarySettingsStore) Save(ctx context.Context, scope string, settings storage.LibrarySettings) error {
	if err := settings.Validate(); err != nil {
		return errf(s.DB, "save library settings", err)
	}
	table, scoped := s.DB.LibrarySettingsTable()

	cols, conflict := []string{"key", "value"}, []string{"key"}
	prefix := []any{}
	if scoped {
		cols = append([]string{s.DB.ScopeColumn()}, cols...)
		conflict = append([]string{s.DB.ScopeColumn()}, conflict...)
		prefix = append(prefix, s.DB.ScopeArg(scope))
	}
	// The ON CONFLICT form is common to PostgreSQL and SQLite (>= 3.24).
	upsert := `INSERT INTO ` + table + ` (` + strings.Join(cols, ", ") + `) VALUES (` +
		strings.TrimSuffix(strings.Repeat("?,", len(cols)), ",") + `)
		ON CONFLICT (` + strings.Join(conflict, ", ") + `) DO UPDATE SET value = EXCLUDED.value`

	rows := settings.Rows()
	err := s.DB.Transact(ctx, func(tx Execer) error {
		// Ordered, so a diff of two libraries' write logs lines up.
		for _, key := range []string{storage.LibrarySettingErrorKey, storage.LibrarySettingBlunderKey} {
			args := append(append([]any{}, prefix...), key, rows[key])
			if _, err := tx.Exec(ctx, upsert, args...); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return errf(s.DB, "save library settings", err)
	}
	return nil
}

// librarySettings is the shortcut every shared store that has to draw the line
// uses: the statistics, the players' table and the study queue all read the
// same two numbers through the same Execer they run their own SQL on, for the
// same scope they are answering about.
func librarySettings(ctx context.Context, db Execer, scope string) (storage.LibrarySettings, error) {
	store := &LibrarySettingsStore{DB: db}
	return store.Load(ctx, scope)
}
