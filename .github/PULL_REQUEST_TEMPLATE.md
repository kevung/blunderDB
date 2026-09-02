## What and why

<!-- One paragraph: the change, and the reason it is needed. -->

## Checklist

- [ ] `make check` passes (vet, tests, golangci-lint, govulncheck, frontend lint/format/tests).
- [ ] A user-visible change (command, shortcut, panel, filter) ships with its French
      documentation in this branch (`doc/source/raccourcis.rst`, `manuel.rst`,
      `cmd_mode.rst`).
- [ ] A modified `.rst` ships with its eight `.po` in the same commit (en, de, el, es,
      fi, it, ja, ru): `scripts/doc-i18n-check.sh` reports `0 untranslated + 0 fuzzy`.
      The online docs deploy from `main` and fall back to French for every gap.
- [ ] A schema change bumps `DatabaseVersion` in `pkg/blunderdb/domain/` and comes
      with a migration in desktop `database`, `storage/sqlite` and
      `storage/postgres/migrations/`, covered by `migration_test.go`.
- [ ] If the retention predicate (`positionIsHeldSQL`) moved, all three copies moved
      the same way (`database/db_match.go`, `storage/sqlite/matches_sqlite.go`,
      `storage/postgres/matches_postgres.go`).
- [ ] Database logic lives on `Database` or the Storage contract and reaches the GUI,
      the CLI and the server alike — no mode-specific fork.
