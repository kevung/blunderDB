# P4 — Decouple session-like state with `scope`

**Goal.** Add a `scope TEXT NOT NULL DEFAULT ''` column to all "current
session" state tables so multiple users can co-exist in the same database
without colliding on session/filter/history keys.

**Estimate.** 3-4 days. **Risk.** Medium (`metadata` UNIQUE index
manipulation). **PRs.** 1.

**Prerequisites.** [P2](02-storage-interface.md). Recommended to run
**after** [P3](03-postgres-backend.md) so both backends get the migration
in one PR.

## Why

The following state is currently global to the DB file:
- `metadata` rows with keys like `session_*`, `last_visited_match`,
  `last_visited_position`.
- `command_history` — replayable command line history.
- `search_history` — past searches.
- `filter_library` — saved filter expressions (named).

In single-user mode (GUI/CLI), there is only one user, so a global state is
fine — `scope = ""`. In server mode, several tenants share one DB. Without
`scope`, two tenants would overwrite each other's last position, last
search, etc.

## Schema changes

For each table in `(metadata, command_history, search_history, filter_library)`:

```sql
-- SQLite
ALTER TABLE metadata ADD COLUMN scope TEXT NOT NULL DEFAULT '';
DROP INDEX IF EXISTS idx_metadata_key;        -- if any
CREATE UNIQUE INDEX idx_metadata_scope_key ON metadata(scope, key);
```

```sql
-- PostgreSQL (003_session_scope.sql)
ALTER TABLE metadata     ADD COLUMN scope TEXT NOT NULL DEFAULT '';
ALTER TABLE command_history  ADD COLUMN scope TEXT NOT NULL DEFAULT '';
ALTER TABLE search_history   ADD COLUMN scope TEXT NOT NULL DEFAULT '';
ALTER TABLE filter_library   ADD COLUMN scope TEXT NOT NULL DEFAULT '';

ALTER TABLE metadata DROP CONSTRAINT IF EXISTS metadata_pkey;
ALTER TABLE metadata ADD CONSTRAINT metadata_pkey PRIMARY KEY (scope, key);

CREATE INDEX IF NOT EXISTS idx_command_history_scope_created
  ON command_history(scope, created_at);
CREATE INDEX IF NOT EXISTS idx_search_history_scope_created
  ON search_history(scope, created_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_filter_library_scope_name
  ON filter_library(scope, name);
```

## Sentinel scopes

- `""` (empty) — single-user mode (GUI/CLI). Default for everything that
  pre-dates P4.
- `"global"` — reserved for entries that must be DB-wide
  (`database_version`, `met_table_id`, …). The migration **does not** move
  these rows; their `scope` stays `''` because they are the only row of
  that key anyway. The applicative split is enforced by callers (they pass
  `""` explicitly).

(Alternative: split `metadata` into `metadata_global` and `metadata_scoped`
tables. Rejected as over-engineering.)

## Interface changes

`SessionStore`, `FilterStore`, `CommandHistoryStore`, `SearchHistoryStore`
gain a `scope string` first argument (after `ctx`) on every method.

```go
type SessionStore interface {
    Save(ctx context.Context, scope string, s *domain.SessionState) error
    Load(ctx context.Context, scope string) (*domain.SessionState, error)
    SaveLastVisitedPosition(ctx context.Context, scope string, matchID, posIdx int64) error
    LoadLastVisitedPosition(ctx context.Context, scope string) (matchID int64, posIdx int64, err error)
    // …
}
```

In `Database` wrapper (kept for Wails), the public methods do **not**
change signature — they always pass `scope = ""` to the underlying
`Storage`. This preserves the GUI's API.

## Version bump

- `DatabaseVersion` in `pkg/blunderdb/domain/version.go` (or its current
  location post-P1): `"2.7.0"` → `"2.8.0"`.
- `db_migration.go` (or its successor) adds `migrate_2_7_0_to_2_8_0` that
  performs the SQLite `ALTER TABLE` + index recreation.
- `migration_test.go` adds a case: open a v2.7.0 DB fixture
  (`testdata/v2_7_0.db`), run migration, check the new column is present
  and existing rows have `scope = ''`.

## Steps

- [ ] Add `scope` column to all four tables (SQLite + Postgres).
- [ ] Recreate uniqueness constraints to include `scope`.
- [ ] Update `*Store` interface signatures in
      `pkg/blunderdb/storage/session.go`, `filters.go`, `history.go`,
      `search.go`.
- [ ] Update SQLite + Postgres implementations accordingly.
- [ ] In the `Database` wrapper, pass `scope = ""` through to all
      delegated calls. Public method signatures unchanged.
- [ ] Bump `DatabaseVersion` to `"2.8.0"`.
- [ ] Add `migrate_2_7_0_to_2_8_0` SQLite migration; add
      `migrations/003_session_scope.sql` for Postgres.
- [ ] Update `tasks/headless/glossary.md` if needed (`scope` is already
      documented there).

## Gotchas

1. **`metadata` UNIQUE/PRIMARY KEY recreation.** The current schema has a
   `PRIMARY KEY (key)` constraint (verify in `db_schema.go`). Dropping it
   requires `CREATE TABLE metadata_new` + copy + rename in SQLite (no
   `ALTER TABLE DROP CONSTRAINT`). Standard SQLite migration pattern.
   Provide a careful, tested migration. Do not lose data.
2. **`INSERT OR REPLACE INTO metadata` callers** must now pass `scope`.
   Audit every callsite (`grep -rn 'INTO metadata'`).
3. **`filter_library` unique name constraint** changes from `UNIQUE(name)`
   to `UNIQUE(scope, name)`. Callers that assume a name is global must be
   updated.
4. **GUI behaviour preserved.** Because the wrapper passes `scope = ""`,
   the desktop user sees no behavioural difference.
5. **Postgres migration ordering.** Apply this migration only if the
   `001_initial_v2_7_0.sql` has been applied. The Postgres migration
   chain tracks its own version separately from SQLite, but for clarity
   keep the chain aligned: `001` = v2.7.0 schema, `003` = `scope` column
   (skip `002` for parity with planned P5 changes if any, or use `002`
   freely — pick one and document).

## Tests

- [ ] `migration_test.go`: open a v2.7.0 fixture, run migration to v2.8.0,
      verify columns and data integrity.
- [ ] Contract test: `SessionStore.Save(ctx, "scope1", s1)` followed by
      `SessionStore.Save(ctx, "scope2", s2)` → both readable independently.
- [ ] Contract test: `FilterStore.Save(ctx, "scope1", "myfilter", expr)`
      and same name in `"scope2"` co-exist.
- [ ] GUI smoke test: open an existing user DB (v2.7.0), confirm migration
      runs, last-visited-position is preserved.

## Verification

- [ ] All tests green on SQLite and Postgres.
- [ ] `wails build` + open an existing user DB → no regression.
- [ ] Two scopes in the same DB show independent session states (manual
      check via test or `blunderdb call session.save --scope a` /
      `--scope b` once P7 lands).

## PR layout

Single PR: `feat(storage): introduce scope on session-like tables (v2.8.0)`.
