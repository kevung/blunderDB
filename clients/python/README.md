# blunderdb — a minimal Python client

A thin client for the `blunderdb serve` engine API. No dependency beyond the
standard library: the daemon speaks POST + JSON (NDJSON for a streaming list),
which `urllib` and `json` cover entirely.

```python
from blunderdb import Client

api = Client("http://127.0.0.1:8080", tenant=1)
print(api.metadata_counts())

for position in api.positions_list({"limit": 10}):
    print(position["id"])
```

## Two halves, on purpose

- `_generated.py` — **one method per route**, generated from the daemon's own
  route table by `go run ./cmd/openapi-gen`. A hand-written surface would drift
  the day a route is added and nobody would notice until a user did; this one
  cannot, because `openapigen_test.go` fails the build while it is stale.
- `client.py` — the **transport**, written by hand: the session, the tenant
  header, the error envelope, the NDJSON decode. What changes with the API is
  generated; what changes with judgement is not.

Method names are `family_operation` in snake_case: `/v1/positions.saveIndividual`
is `positions_save_individual()`. The family is kept because several families
share an operation name (`list`, `delete`), and a bare `list()` would collide.

## Security

The daemon performs **no authentication of its own**. It trusts `X-Tenant-ID`
verbatim and must run behind an authenticating reverse proxy — see
[ADR-0005](../../docs/adr/0005-blunderdb-serve-trusts-its-caller-authentication-belongs-to-the-proxy-in-front.md)
and the *headless* chapter of the documentation. The tenant this client sends
is not a credential, and treating it as one is exactly the mistake that ADR
exists to prevent.

## Errors

A failure raises `APIError`, carrying the daemon's own envelope: `code` (what a
program branches on — `not_found`, `conflict`, `invalid`, `rate_limited`),
`message` (what a person reads), `status` and `details`.

## Idempotency

Most methods need nothing: reads have no effect, and `positions.save` — like the
rest of `positions.*` — is naturally idempotent through the position's Zobrist
content hash. The three that are not (`collections.create`,
`tournaments.create`, `anki.reviewCard`) take an `idempotency_key=` argument: a
retried call carrying the same key replays the first attempt's result instead of
repeating its effect.

## Versioning

See `doc/source/mode_headless.rst` — the `/v1` policy is written there, and it
is the contract this client is generated against.
