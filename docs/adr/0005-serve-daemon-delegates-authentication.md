# The serve daemon performs no authentication and trusts X-Tenant-ID

Status: accepted.

## Context
`blunderdb serve` exposes the storage contract over HTTP; every data method takes a Tenant,
which arrives as the `X-Tenant-ID` header (`internal/server/middleware/tenant.go`). Its host,
gammonGo, already authenticates users, terminates TLS, and fronts the engine either in-process
(`pkg/blunderdb/server.Bootstrap`) or as a reverse proxy in front of `Dockerfile.serve`.

## Decision
1. The daemon performs **no authentication of its own**. It trusts `X-Tenant-ID` as sent and
   rejects only its absence or an invalid form (rule 3).
2. Deployments MUST place it behind an authenticating reverse proxy (or embed it in an
   authenticating parent) that strips any client-supplied `X-Tenant-ID` and injects the
   authenticated tenant. It is never exposed directly to the public internet. PostgreSQL
   Row-Level Security (`serve --rls`) is opt-in defence-in-depth inside that boundary — it
   pins each connection to `app.tenant_id` against a handler bug — not a substitute for the
   proxy.
3. **A tenant is a positive decimal integer**, as `storage.ParseTenant` defines it: canonical
   spelling, no sign, no leading zero, at most int64. The empty scope is the desktop's implicit
   tenant and is never sent over HTTP. Anything else is rejected: 400 `code=invalid` by the
   tenant middleware, an error from `storage.ParseTenant`, `migrate --tenant-id` and
   `call --scope`, `ErrInvalidTenant` from `PurgeTenant`. Mapping an account name to its
   integer is the proxy's job. The SQLite backend keeps the scope as an opaque string; the
   format is enforced at the entry points, not in its tables.

## Consequences
- The trust boundary is the proxy: anyone reaching the port can read and write any tenant by
  naming it. The warning stands in `cmd/serve/main.go`, `serve --help`, `Dockerfile.serve` and
  the headless chapter of the manual; it is never weakened.
- The engine holds no credentials or user management; tenancy is pure data partitioning.
  `tenant.purge` and other administrative methods carry no privilege check of their own.
- Desktop and CLI never start the daemon and pass the empty Tenant directly.
- Rejected: built-in auth (tokens, sessions, mTLS) — a second credential system to keep in sync
  with the host's, plus secret storage, rotation and revocation.
- Rejected: a shared-secret header between proxy and daemon — guards only against network
  exposure, which the private-network rule already covers.
- Deferred: refusing to start without an "I am behind a proxy" flag — friction for every
  legitimate deployment; revisit if a bare deployment happens.
- Rejected: a `tenant(scope TEXT UNIQUE, id BIGSERIAL)` table minting integers from names — it
  moves identity management into the engine; reconsider only for a host without numeric ids.

## Guard
`internal/server/middleware/tenant_test.go`, `pkg/blunderdb/storage/tenant_context_test.go`.
