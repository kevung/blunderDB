# Phase D (optional) — Tenant purge (delete all data for a tenant)

**Goal.** Give an operator running the PostgreSQL `serve` backend a way to
permanently delete every row belonging to one tenant scope — positions,
matches, collections, everything — when that tenant is decommissioned.

**Estimate.** 2-3 days. **Risk.** Medium (destructive operation). **PRs.** 1.

**Prerequisites.** [P3](03-postgres-backend.md) (tenant model, `rlsTables`),
[P6](06-serve-http.md) (HTTP+JSON daemon, frozen error envelope).

## Why

The multi-tenant PostgreSQL backend (P3) never deletes a tenant's data — a
tenant is provisioned implicitly on first write and its rows simply
accumulate. An operator decommissioning a tenant (the consuming service
stopped using that scope, a test tenant needs cleanup, a retention policy
requires it) currently has no way to remove that data short of hand-written
SQL against internal tables.

This is **optional** (own phase, not folded into P3) because it is a
destructive, ops-facing capability rather than something every consumer
needs — SQLite (single-user desktop) has no real tenant concept and must not
grow a no-op or misleading version of this.

## Design

**D1 — Not part of the shared `Storage` interface.** `PurgeTenant` is
PostgreSQL-only, exactly like `ApplyRLS`/`DropRLS` (`rls_postgres.go`): a
method on the concrete `*postgres.Storage` type, invoked elsewhere via an
anonymous-interface type assertion. Adding it to `Stores`/`Storage` would
force a meaningless implementation (or a "not supported" error) onto the
SQLite backend, which has no multi-tenant model to purge.

`pkg/blunderdb/storage/postgres/purge_postgres.go`:

```go
package postgres

import (
	"context"
	"fmt"
)

// PurgeTenant permanently deletes every row belonging to tenantID across all
// tenant-scoped tables (rlsTables). It is idempotent — purging a tenant with
// no data, or purging twice, succeeds with zero rows affected. Runs in a
// single transaction: either every table is purged or none is.
//
// PostgreSQL-only, like ApplyRLS/DropRLS — there is no SQLite equivalent
// (single-user desktop databases have no tenant to purge).
func (s *Storage) PurgeTenant(ctx context.Context, tenantID int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres: purge tenant %d: begin: %w", tenantID, err)
	}
	defer tx.Rollback(ctx) // no-op after a successful Commit

	// Children before parents, so no ON DELETE CASCADE/SET NULL action needs
	// to fire (every referenced row for this tenant is already gone by the
	// time its parent's row is deleted) — explicit rather than relying on
	// cascade ordering.
	order := []string{
		"move_analysis", "anki_review_log", "collection_position",
		"comment", "analysis", "move", "anki_card", "game",
		"collection", "anki_deck", "match", "tournament", "position",
		"filter_library", "command_history", "search_history",
	}
	for _, t := range order {
		if _, err := tx.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE tenant_id = $1`, t), tenantID); err != nil {
			return fmt.Errorf("postgres: purge tenant %d: %s: %w", tenantID, t, err)
		}
	}
	return tx.Commit(ctx)
}
```

`order` must stay a permutation of `rlsTables` (`rls_postgres.go`) — cover
this with a test (see Tests) so the two lists can't silently drift apart.

**D2 — Exposed as a new HTTP endpoint**, `POST /v1/tenant.purge`, not a
CLI-only tool: the whole point is a running `serve` daemon deleting a
tenant's data on request (an operator or an upstream admin panel calling in),
mirroring every other domain operation's `/v1/<family>.<method>` shape rather
than requiring daemon restart/CLI access. `internal/server/handlers_tenant.go`:

```go
package server

import "net/http"

// tenantPurger is satisfied only by the PostgreSQL backend (see
// postgres.Storage.PurgeTenant) — duck-typed the same way serve.go checks for
// ApplyRLS, so the SQLite backend needs no stub method.
type tenantPurger interface {
	PurgeTenant(ctx context.Context, tenantID int64) error
}

func (s *Server) tenantRoutes() []route {
	return []route{
		{http.MethodPost, "/v1/tenant.purge", func(w http.ResponseWriter, r *http.Request) {
			purger, ok := s.opts.Storage.(tenantPurger)
			if !ok {
				writeErrorCode(w, CodeInvalid, "tenant purge not supported on this backend (postgres only)")
				return
			}
			scope := scopeOf(r)
			if err := purger.PurgeTenant(r.Context(), storage.ParseTenant(scope)); err != nil {
				writeErrorCode(w, CodeInternal, "purge failed: "+err.Error())
				return
			}
			writeJSON(w, okResp{OK: true})
		}},
	}
}
```

Add `rs = append(rs, s.tenantRoutes()...)` to `domainRoutes()`
(`internal/server/routes.go`). No request body — the tenant is the caller's
own scope (`X-Tenant-ID`), exactly like every other scoped operation; there is
no "purge someone else's tenant" parameter to avoid a confused-deputy risk in
callers.

No new error code needed: the "not supported on this backend" case reuses
`CodeInvalid`, matching the existing precedent in `handlers_imports.go`
("import format not supported on this server").

## Configuration

None — no new flag. The endpoint exists whenever `serve --backend postgres`
runs; it 400s with `CodeInvalid` on `--backend sqlite`.

## Steps

- [ ] `purge_postgres.go`: `PurgeTenant` as designed above.
- [ ] Guard: a table-driven check (or a `reflect`/set comparison in a test)
      that `order` is exactly `rlsTables` reordered, so a future table added
      to one list is caught if missing from the other.
- [ ] `handlers_tenant.go` + wire into `domainRoutes()`.
- [ ] Update `doc/source/mode_headless.rst`, `.. _headless_postgres:` section:
      one paragraph documenting `POST /v1/tenant.purge` right after the
      existing RLS paragraph. Note in the task sheet (not the prose doc)
      that the `.po` translation files under
      `doc/source/locale/*/LC_MESSAGES/mode_headless.po` are now stale for
      this file — translation is a separate follow-up, not blocking.

## Tests

- [ ] `purge_postgres_test.go` (postgres-tagged, like the RLS/contract
      postgres tests): create two tenants, write at least one row per family
      touched by `order` (position + analysis + comment + match + game +
      move + move_analysis + tournament + collection + collection_position +
      anki_deck + anki_card + anki_review_log + filter_library +
      command_history + search_history) for tenant A, purge tenant A, assert
      every one of those rows is gone for tenant A (`Get`/`Counts` per
      family) **and** tenant B's rows of the same families are untouched.
  - [ ] Idempotency: purge tenant A a second time — no error, zero rows
        affected.
  - [ ] Purging a tenant with no data at all — no error.
  - [ ] `order`/`rlsTables` parity test described above.
- [ ] `handlers_tenant_test.go`: `POST /v1/tenant.purge` against a Postgres
      server returns `{"ok":true}`; against a SQLite server returns
      `CodeInvalid` (400), and the SQLite data is untouched.

## Verification

- [ ] `go test ./...` green on both backends (SQLite: purge endpoint
      guard-tested; Postgres: full purge + isolation + idempotency).
- [ ] Manual: `serve --backend postgres`, seed a tenant via `call`, then
      `blunderdb call tenant.purge --scope <id>`; re-run any `<family>.list`
      for that scope and confirm empty results.

## Gotchas

1. **Not RLS-dependent.** `PurgeTenant` filters by `tenant_id = $1`
   explicitly on every table — it does not rely on `Options.EnableRLS`/the
   `app.tenant_id` GUC being set. RLS being off (the default) must not make
   purge scoped to the wrong tenant or to everything.
2. **Cascades are a bonus, not the mechanism.** `order` deletes children
   before parents so no `ON DELETE CASCADE`/`SET NULL` needs to fire during
   the purge itself — don't rely on a parent-table delete to clean up its
   children implicitly; delete every table in `rlsTables` explicitly.
3. **No "purge someone else's tenant" API.** The endpoint always purges the
   caller's own `X-Tenant-ID` scope. If a future need arises for an operator
   to purge an arbitrary tenant by ID, that is a distinct, higher-privilege
   capability (a separate admin-only endpoint/flag) — do not widen this one.
4. **This phase does not touch SQLite at all.** No new SQLite method, no new
   CLI subcommand under `internal/cli/` (that historical CLI has no tenant
   concept — see `tasks/headless/glossary.md`/P4). The only SQLite-visible
   change is the guarded 400 on the HTTP endpoint.

## PR layout

Single PR: `feat(server): tenant purge endpoint (postgres only)`.

## Done — implementation notes

_(filled in after implementation)_
