package sqlshared

import (
	"context"
	"strconv"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// LibrarySettingsStore implements storage.LibrarySettingsStore over the two
// key/value rows of ADR-0046. The SQL is one statement per operation on both
// backends; only the table and whether it is scoped differ, and the Dialect
// answers that (LibrarySettingsTable):
//
//   - SQLite writes the rows to the metadata table, next to the Performance
//     Rating objective, where `blunderdb info` and `edit` find them.
//   - PostgreSQL writes them to library_settings, its own tenant-scoped table:
//     metadata there is global to every tenant, outside Row-Level Security,
//     and read-only.
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
	before, err := s.Load(ctx, scope)
	if err != nil {
		return errf(s.DB, "save library settings", err)
	}
	err = s.DB.Transact(ctx, func(tx Execer) error {
		// match_stats counts errors and blunders at these thresholds: moving
		// either makes every stored row stale.
		if before.BlunderThresholdMP != settings.BlunderThresholdMP || before.ErrorThresholdMP != settings.ErrorThresholdMP {
			if err := InvalidateAllMatchStats(ctx, tx, scope); err != nil {
				return err
			}
		}
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

// PileCollection reads the id of the Pile collection; 0 when none was named
// or the row is unreadable (a comfort row, like the thresholds).
func (s *LibrarySettingsStore) PileCollection(ctx context.Context, scope string) (int64, error) {
	table, scoped := s.DB.LibrarySettingsTable()
	query := `SELECT COALESCE(value,'') FROM ` + table + ` WHERE `
	args := []any{}
	if scoped {
		query += s.DB.ScopeColumn() + ` = ? AND `
		args = append(args, s.DB.ScopeArg(scope))
	}
	query += `key = ?`
	args = append(args, storage.LibrarySettingPileKey)
	rows, err := s.DB.Query(ctx, query, args...)
	if err != nil {
		return 0, errf(s.DB, "load pile collection", err)
	}
	defer rows.Close()
	var id int64
	if rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return 0, errf(s.DB, "load pile collection", err)
		}
		if n, perr := strconv.ParseInt(v, 10, 64); perr == nil && n > 0 {
			id = n
		}
	}
	if err := rows.Err(); err != nil {
		return 0, errf(s.DB, "load pile collection", err)
	}
	return id, nil
}

// SetPileCollection records the id of the Pile collection.
func (s *LibrarySettingsStore) SetPileCollection(ctx context.Context, scope string, collectionID int64) error {
	table, scoped := s.DB.LibrarySettingsTable()
	cols, conflict := []string{"key", "value"}, []string{"key"}
	args := []any{}
	if scoped {
		cols = append([]string{s.DB.ScopeColumn()}, cols...)
		conflict = append([]string{s.DB.ScopeColumn()}, conflict...)
		args = append(args, s.DB.ScopeArg(scope))
	}
	args = append(args, storage.LibrarySettingPileKey, strconv.FormatInt(collectionID, 10))
	upsert := `INSERT INTO ` + table + ` (` + strings.Join(cols, ", ") + `) VALUES (` +
		strings.TrimSuffix(strings.Repeat("?,", len(cols)), ",") + `)
		ON CONFLICT (` + strings.Join(conflict, ", ") + `) DO UPDATE SET value = EXCLUDED.value`
	if _, err := s.DB.Exec(ctx, upsert, args...); err != nil {
		return errf(s.DB, "save pile collection", err)
	}
	return nil
}
