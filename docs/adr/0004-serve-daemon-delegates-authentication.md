# The serve daemon performs no authentication and trusts X-Tenant-ID

## Status

accepted

## Context

`blunderdb serve` exposes the engine as an HTTP + JSON daemon so a shared
database can serve several players. Every data method in the storage contract
takes a Tenant (see `CONTEXT.md`); over HTTP the caller's Tenant arrives as the
`X-Tenant-ID` request header (`internal/server/middleware/tenant.go`). Someone
has to decide whether that header can be believed — that is, someone has to
authenticate the caller and bind them to a tenant.

The daemon exists to be embedded: its intended parent is gammonGo, a host that
already authenticates its users, terminates TLS, and fronts the engine either
in-process (`pkg/blunderdb/server.Bootstrap`) or as a reverse proxy in front of
the container built by `Dockerfile.serve`.

## Decision

The daemon performs **no authentication of its own**. It trusts `X-Tenant-ID`
as sent and rejects only its absence. Deployments MUST place it behind an
authenticating reverse proxy (or embed it in an authenticating parent process)
that strips any client-supplied `X-Tenant-ID` and injects the authenticated
tenant. It must never be exposed directly to the public internet.

PostgreSQL Row-Level Security (`serve --rls`) is available as opt-in
defence-in-depth *inside* that boundary: it pins each connection to
`app.tenant_id` so a handler bug cannot leak rows across tenants. It is not a
substitute for the proxy — RLS still believes whatever tenant the header named.

## Considered options

- **Built-in auth (tokens, sessions, mTLS).** Rejected: the host already has an
  authenticated user model, and a second account/credential system inside the
  engine would have to be kept in sync with it. Every scheme also drags in
  secret storage, rotation and revocation — none of which the engine needs for
  its actual job.
- **Shared-secret header between proxy and daemon.** Rejected as a default: it
  protects only against misconfigured network exposure, which the deployment
  rule (private network / same host) already addresses with less machinery.
- **Refuse to start without an explicit "I am behind a proxy" flag.** Deferred:
  friction for every legitimate deployment to catch a mistake the docs, the
  `--help` text, and the Dockerfile all warn about. Revisit if a bare
  deployment actually happens.

## Consequences

- The trust boundary is the proxy. Anyone who can reach the daemon's port can
  read and write **any** tenant's data by naming it in a header. The warning is
  stated in `cmd/serve/main.go`, the `serve --help` usage text,
  `Dockerfile.serve`, and the user manual's headless chapter.
- The engine stays free of credentials and user management; tenancy remains a
  pure data-partitioning concern (one `scope` argument end to end).
- `tenant.purge` and other administrative methods carry no extra privilege
  check — the proxy decides who may call them, like everything else.
- Local single-user use is unaffected: the desktop app and CLI never start the
  daemon and pass the implicit empty Tenant directly.
