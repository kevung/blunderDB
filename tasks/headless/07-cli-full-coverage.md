# P7 — CLI: 100 % coverage of `Storage`

**Goal.** Make every method on `Storage` callable from the CLI via a
generic dispatcher `blunderdb call <family>.<method> --json '{…}'`,
while preserving the historical subcommands (`create`, `import`, `export`,
`list`, `match`, `verify`, `delete`, `info`, `edit`, `search`) for
backward compatibility with existing user scripts.

**Estimate.** 3-4 days. **Risk.** Low. **PRs.** 1.

**Prerequisites.** [P6](06-serve-http.md) — to share the route table.

## Why

- The current CLI exposes ~12-30 % of `Storage`. The gaps (collections,
  anki, filters, comments, session, edit, fine-grained position ops)
  matter for production debugging and integration testing.
- Sharing the route table with the server ensures behavioural parity:
  the CLI and HTTP API run the same handler functions.

## Design

The route table is the source of truth:

```go
// pkg/blunderdb/api/routes.go
type Method struct {
    Family   string                  // "positions"
    Name     string                  // "save"
    Request  reflect.Type            // PositionsSaveRequest
    Response reflect.Type            // PositionsSaveResponse
    Invoke   func(ctx context.Context, scope string, s storage.Storage, req any) (any, error)
}

var Routes = []Method{
    {Family: "positions", Name: "save",   Request: ..., Response: ..., Invoke: positionsSave},
    {Family: "positions", Name: "load",   …},
    …
}
```

The HTTP server (P6) loops over `Routes` to mount handlers.
The CLI dispatcher (this phase) loops over `Routes` to register `call`
subcommands.

## CLI usage

```bash
# Generic dispatcher
blunderdb call positions.save --json '{"position": {...}}'
blunderdb call positions.list --json '{}' --format ndjson
blunderdb call matches.delete --json '{"id": 42}'
blunderdb call session.saveLastVisitedPosition --scope user-1 --json '{...}'

# Discover available methods
blunderdb call --list
blunderdb call positions.save --help    # prints request schema (from reflection)
```

Flags:
```
--db <path>            [default $BLUNDERDB_DSN or sqlite::memory:]
--backend <kind>       [default inferred from --db]
--scope  <string>      [default '']
--json   <string>      [request body, default '{}']
--json-file <path>     [reads request body from file]
--format <json|ndjson> [default json; ndjson for streaming list endpoints]
```

Output: stdout is the JSON (or NDJSON) response body. Non-zero exit code
on error; the error envelope goes to stdout for parseability.

## Backward compatibility

Historical subcommands stay as-is in `internal/cli/cmd_<name>.go`. They
internally invoke the same `Routes` entries:

```go
// internal/cli/cmd_search.go
func (c *CLI) runSearch(args []string) error {
    // parse legacy flags
    req := buildSearchRequest(flags)
    resp, err := api.Routes.Find("search.byFilter").Invoke(ctx, "", c.storage, req)
    return printer.Print(resp, err)
}
```

This avoids forking logic (per CLAUDE.md "parité CLI/GUI").

## Steps

- [ ] Move the dispatching definition of `api.Routes` from
      `internal/server/routes.go` to `pkg/blunderdb/api/routes.go` (or
      formalise it if it already lives there from P6). Make it
      independent of HTTP — `Invoke` takes Go types, not
      `http.ResponseWriter`.
- [ ] Implement `internal/cli/cmd_call.go`:
  - Parses `<family>.<method>`.
  - Looks up the route.
  - JSON-unmarshals the request body into `Routes[i].Request`.
  - Calls `Routes[i].Invoke(ctx, scope, storage, req)`.
  - JSON-encodes the response or pipes the iterator to NDJSON.
- [ ] Implement `blunderdb call --list` (table of available methods).
- [ ] Implement `blunderdb call <family>.<method> --help` (uses Go
      reflection on `Request` type to print field names and types).
- [ ] Update the historical subcommands to dispatch through `Routes`
      where it doesn't change observable behaviour. Where the legacy
      output format differs from JSON (e.g. `list` prints a table), keep
      a thin formatter wrapper.

## Tests

- [ ] Parametrised test: for every entry in `api.Routes`, the CLI
      dispatcher accepts a valid `--json` body and returns either a JSON
      response or an NDJSON stream. Verifies coverage is complete.
- [ ] `cmd_call_test.go`: spot-check three families end-to-end against an
      in-memory `Storage`.
- [ ] Backward-compat tests: existing `cli_test.go` cases continue to pass
      verbatim.

## Documentation

- `CLI_USAGE.md` gains a "Generic call dispatcher" section.
- `blunderdb call --list` output is included in the README as a quick
  reference.
- `examples/cli/*.sh` mirrors `examples/curl/*.sh` from P6.

## Gotchas

1. **Backend selection.** The CLI may target SQLite (file) or Postgres
   (DSN). `--db` accepts both forms; the dispatcher decides which backend
   to open.
2. **Streaming output.** `--format ndjson` must flush per record;
   important when piping into `jq -c`. Use `bufio.Writer` with explicit
   `Flush` per line.
3. **Per-request scope.** Defaults to empty `""`. For methods that
   require a non-empty scope in server mode (none, by design — server uses
   the header), this is a no-op for the CLI.
4. **Error format.** Same envelope as P6 to keep parsing identical:
   `{"error":{"code":"...","message":"..."}}`.

## Verification

- [ ] `blunderdb call --list` shows ≥ 186 methods (one per `Storage`
      method).
- [ ] Random sample of 10 methods invokable end-to-end via `call`.
- [ ] Legacy `./blunderdb import …`, `./blunderdb list --type stats
      --format json`, etc. continue to behave as before
      (byte-equivalent JSON where applicable).
- [ ] `go test ./...` green.

## PR layout

Single PR: `feat(cli): generic call dispatcher matching Storage surface`.
