# P6 PR3 / P8 — Imports & exports over `Storage`

**Goal.** Expose `imports.*` / `exports.*` on the `blunderdb serve` daemon
in a **backend-agnostic** way: the import/export pipeline writes through the
`storage.Storage` / `storage.Tx` interfaces instead of the SQLite-only
`Database` wrapper, so it works identically on SQLite and PostgreSQL.

This is the merge of P6 PR3 (the HTTP surface) and P8 (streaming imports
with `context`), because the two cannot be cleanly separated: the daemon
needs an importer that is not tied to the global-mutex `Database`.

**Estimate.** 6-9 days. **Risk.** High (re-implements dedup, transactions,
multi-format mapping). **PRs.** 3 (a/b/c).

## Why not the `Database` wrapper

The existing import (~6 900 L across `db_import_xg.go`, `db_import_gnubg.go`,
`db_import_bgf.go`, `db_import_db.go`, `db_import_common.go`) is soldered to
`*Database`: `d.mu.Lock()`, `d.db.Begin()`, raw `tx.Exec` INSERTs, and the
dedup helpers `checkDuplicateMatchLocked` / `savePositionInTx`. It also takes
**file paths**, not readers. Reusing it would (a) only work for SQLite and
(b) drag the global mutex into the server. The arbitrated decision is to
re-target the writes onto the `Storage` interface.

## Architecture

```
parser (xg/gnubg/bgf)  ──►  mapping → domain.Match graph  ──►  im{port,Writer}
                                                                    │
                                                          storage.Tx (Save/Create…)
                                                                    ▼
                                                        SQLite or PostgreSQL
```

Two new interfaces (package `pkg/blunderdb/ingest`, backend-independent):

```go
// Importer turns a parsed source into stored domain objects via Storage.
type Importer interface {
    // Import reads src (already in memory or a temp file), writes through
    // the Storage, emits progress, honours ctx cancellation, and returns a
    // summary. dedup is by match hash + canonical hash + position Zobrist.
    Import(ctx context.Context, scope string, src Source, prog func(Progress)) (Summary, error)
}

// Exporter streams stored data out in a chosen format.
type Exporter interface {
    Export(ctx context.Context, scope string, w io.Writer, opts ExportOptions) error
}
```

`Source` carries the format + a `fs.File`/`io.ReaderAt` (parsers need
seeking, so the daemon spools the multipart upload to a temp file — never the
whole file in RAM). `Progress` is `{Matches, Games, Positions int}`.
`Summary` is `{SavedPositions, SkippedDuplicates, MatchID int64, …}`.

The **format-specific mapping** (parser struct → `domain.Match`/`Game`/`Move`/
`Position`/`PositionAnalysis`) is lifted out of the current `db_import_*.go`
files into pure functions in `ingest/<format>.go` that have **no** `*sql.DB`
dependency. The write step calls the shared `matchWriter` (below).

### `matchWriter` — the shared Storage-based sink

```go
type matchWriter struct{ tx storage.Tx }

func (mw *matchWriter) WriteMatch(ctx, scope, m *domain.Match) (int64, error)
// → tx.Matches().Save, then per game CreateGame, per move CreateMove,
//   per position Positions().Save (Zobrist dedup), Analyses().Save,
//   comments, tournament auto-link.
```

All within a single `storage.Tx` so an import is atomic and cancellable
(rollback on `ctx.Done()`), satisfying P8's cancellation requirement
(`POST /v1/imports.cancel`).

## Storage interface extensions (PR3b)

The dedup the importer needs is **not** on `Storage` today. Add to
`MatchStore` (and implement on both backends + contract tests):

```go
// FindByHash returns the id of a match with the given content hash, or
// (0,false). canonicalHash enables cross-format dedup.
FindByHash(ctx, scope string, hash, canonicalHash string) (id int64, found bool, err error)

// SetHashes records the content + canonical hash of a match (set at import).
SetHashes(ctx, scope string, id int64, hash, canonicalHash string) error
```

The `position` Zobrist dedup already lives in `PositionStore.Save`. The
canonical/match-hash functions (`ComputeMatchHash`,
`ComputeCanonicalMatchHashFrom{XG,GnuBG}`) are already pure — move them to
`ingest/hash.go` and reuse verbatim.

> Schema note: the `match` table already stores hashes for the GUI/CLI path;
> the Storage backends must read/write the same columns so a DB imported via
> the daemon dedups against one imported via the GUI. No `DatabaseVersion`
> bump expected (columns exist); verify in `db_schema.go` /
> `schema_sqlite.go` / Postgres `001_*.sql`.

## HTTP surface (PR3a)

```
POST /v1/imports.xg        multipart: file=…   → NDJSON progress
POST /v1/imports.gnubg     multipart: file=…   → NDJSON progress
POST /v1/imports.bgf       multipart: file=…   → NDJSON progress
POST /v1/imports.db        multipart: file=…   → NDJSON progress (native .db)
POST /v1/imports.position  multipart: file=…   → NDJSON progress
POST /v1/imports.json      multipart: file=…   → NDJSON progress (interchange)
POST /v1/imports.cancel    body {import_id}    → {ok}
POST /v1/exports.json      body {opts}         → application/x-ndjson stream
POST /v1/exports.db        body {opts}         → application/octet-stream
```

Progress stream:
```
{"event":"started","import_id":"…"}
{"event":"progress","matches":1,"games":12,"positions":340}
{"event":"done","saved_positions":340,"skipped_duplicates":3}
```

Server bits:
- multipart parse with a raised `MaxBytesReader` (imports are large); spool to
  a `os.CreateTemp` file, `defer os.Remove`.
- per-tenant `map[string]context.CancelFunc` on the `Server`, guarded by a
  mutex, keyed by a generated `import_id`. `imports.cancel` cancels it.
- progress callback marshals NDJSON events and flushes.

## Decomposition

| PR | Scope | Backend-agnostic? |
|----|-------|-------------------|
| **3a** | `ingest` interfaces; HTTP transport (multipart, NDJSON progress, cancellation map); **JSON interchange** export+import implemented over `Storage` (no parser needed) as the first working, fully backend-agnostic path; routes wired; fake-importer httptest. | yes |
| **3b** | `MatchStore` dedup extensions (both backends + contract); `matchWriter`; **XG** mapping lifted into `ingest/xg.go`; `imports.xg` end-to-end; fixture tests. | yes |
| **3c** | **GnuBG**, **BGF**, native **.db**, Jellyfish **.mat**, position **text/XGP** mappings migrated onto `matchWriter`; remaining routes; per-format fixture tests. | yes |

The GUI/CLI keep using the `Database` wrapper unchanged (its `db_import_*.go`
stay) until a later cleanup migrates them onto `ingest` too — out of scope
here; tracked as a follow-up so there is no behavioural change for the
desktop app in this chantier.

## Tests

- `ingest_json_test.go` — round-trip: seed a Storage, export JSON, import into
  a fresh Storage, assert counts + a spot-checked position match.
- `imports_http_test.go` — multipart upload of a small fixture → progress
  events in order → `done`; `imports.cancel` mid-stream → rollback, row count
  unchanged.
- `dedup_test.go` (contract) — importing the same match twice stores it once;
  cross-format canonical dedup.
- Per-format: reuse the existing `testdata/` fixtures; assert the same row
  counts as the `Database`-wrapper import produces (parity).

## XG mapper extraction — concrete plan (PR3b remaining)

The foundation (dedup `FindByHash`, `WriteMatch`/`MatchGraph`) is in. The
remaining work is the XG parser → `MatchGraph` mapper. Key finding from
`db_import_xg.go`:

- `createPositionFromXG(...) *Position` is already a near-pure mapper — lift
  it into `ingest/xg.go` (drop the `*Database` receiver).
- The analysis builders `saveCheckerAnalysisToPositionInTx` /
  `saveCubeAnalysisToPositionInTx` / `saveCubeAnalysisForCheckerPositionInTx`
  (~700 L) **build *and* write** the `PositionAnalysis` in one pass. Split
  each into a pure `buildCheckerAnalysis(...) *domain.PositionAnalysis` (plus
  the move-string/hit/multiplier helpers `inferMoveMultipliers`,
  `convertXGMoveToStringWithHits`, `mergeSlides*`, `translateAnalysisDepth`,
  which are already pure) and let `WriteMatch` do the persisting.
- `importMoveWithCacheAndRawCube` orchestration becomes "append a `MoveGraph`
  to the current `GameGraph`".
- Reuse `ComputeMatchHash` / `ComputeCanonicalMatchHashFromXG` verbatim for
  `MatchGraph.Match.{MatchHash,CanonicalHash}`.

**Parity gate (do not skip).** A count-only test is insufficient: a wrong
mapper yields right row counts but corrupt equities. Add a parity test that
imports `testdata/*.xg` via the new mapper into a Storage AND via the legacy
`Database.ImportXGMatch`, then diffs positions+analyses field-by-field
(equities, errors, move strings). Only ship when the diff is empty.

The GUI/CLI keep `Database.ImportXGMatch`; once the pure builders exist, a
later cleanup can re-point the legacy writer at them to delete the
duplicated analysis-building code.

## Gotchas

1. **Parsers need a file path / seeking.** Spool multipart to a temp file;
   pass its path/`*os.File`. Do not assume a one-shot `io.Reader`.
2. **Atomicity vs. progress.** Progress events are emitted mid-transaction;
   if the import is cancelled/rolled back, the client already saw "progress".
   Document that `done` is the only authoritative event; absence of `done`
   ⇒ rolled back.
3. **Tx size.** A huge match in one `Tx` may strain memory/locks. Acceptable
   for typical match files; chunked commits are a later optimisation (note in
   `ingest/doc.go`).
4. **Hash parity with the GUI.** The daemon and the GUI must compute identical
   match hashes or cross-path dedup breaks. Reuse the exact existing hash
   functions; cover with a parity test.
5. **Postgres `tenant_id`.** When P4 multi-tenancy lands on Postgres, the
   `matchWriter` already threads `scope`; nothing extra here.

## Verification

- `curl -F file=@game.xg -H 'X-Tenant-ID: u1' :8080/v1/imports.xg` streams
  progress and ends with `done`; a second identical upload reports
  `skipped_duplicates` and the same `match_id`.
- `:8080/v1/exports.json` of a seeded DB re-imports into an empty DB with
  identical `metadata.counts`.
- Same flows succeed against `--backend postgres`.
- `imports.cancel` aborts an in-flight import within ~200 ms; row count
  unchanged.
