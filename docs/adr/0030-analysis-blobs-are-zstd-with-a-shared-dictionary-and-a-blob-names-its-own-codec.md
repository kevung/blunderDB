# Analysis blobs are zstd with a shared dictionary, and a blob names its own codec

## Status

accepted — decided 2026-09-03. Closes #180 (plan 2026-09b, fiche B.12). Evidence in
`docs/recherche/P11-compression-blobs.md`. No `DatabaseVersion` bump: the change is
entirely inside the bytes of an unchanged `analysis.data` BLOB/BYTEA column.

## Context

`analysis.data` holds the JSON-encoded `PositionAnalysis` for one position — checker
move candidates, cube verdict, provenance — compressed before it reaches SQLite or
PostgreSQL. Until this record every release compressed it with zlib level 9
(`compress/zlib`, stdlib), with no dictionary. P11 (external research, see the file
above) measured what the codec was leaving on the table and answered two questions
before any code changed: which compressor, and whether a trained dictionary is worth
the added moving part.

**A dictionary is the lever, not the algorithm.** P11 cites zstd's own man page
(*"Typical gains range from ~10% (at 64KB) to x5 better (at <1KB)"*) and a published
Facebook benchmark on small JSON records (~300 B, ×1.33 → ×5.86–6.83). blunderDB's
blobs (2–20 KB of repetitive JSON — the same field names, the same small set of move
notations, the same handful of provenance strings, on every row) sit exactly in the
regime a shared dictionary is built for: the redundancy is *across* rows, not inside
any one of them, and a per-blob compressor never sees it. A binary serialisation
(CBOR, MessagePack) was considered and rejected on the same report's evidence: the gap
to JSON shrinks to ~10–20 % once compression is applied, sometimes reverses, so
changing the wire format would have bought little for a real migration cost (every
past blob would need a new decoder).

**Measured on this repository's own data** (`cmd/train-analysis-dict`, corpus = every
match fixture under `testdata/` re-imported plus the embedded demo database, split
80/20 by a deterministic hash of each blob's content, dictionary trained on the 80%
split only, ratio measured on the 20% held out — never on data the dictionary
memorised):

| codec | held-out test set (804-896 real blobs across runs) | ratio | reduction vs raw |
|---|---|---|---|
| zlib level 9, no dict (today, every release before this one) | ~2.07 MB → ~473 KB | 4.37x | 77.1 % |
| **zstd level 19 + trained dictionary (this record)** | ~2.07 MB → ~259 KB | **7.98x** | **87.5 %** |

zstd+dict is **45.2 % smaller than zlib-9 on the same blobs**. A whole-database
comparison (3,855 real analysis rows, both built into a fresh SQLite file and
`VACUUM`ed) confirms it is not an artefact of per-blob accounting: the zstd+dict file
is **46.8 % smaller** than the zlib-9 file (1,359,872 B vs 2,555,904 B). Per-blob
latency, same corpus: zlib-9 compress 230 µs / decompress 37 µs; zstd-19+dict compress
2.67 ms / decompress 25 µs. Compression is ~11x slower — expected at level 19, and
irrelevant here because an analysis is written once and read often — while
decompression, the path every search and every panel open pays for, comes out
*faster* than zlib. These are this repository's own numbers, not P11's estimates;
P11 itself flagged its ×1.5–3 estimate as conservative for 2–20 KB blobs, and the
measurement landed above that range.

**A file is exchanged between machines, so a blob cannot assume today's binary wrote
it.** Every 2.x release before this one is zlib; older exports and hand-built test
databases hold raw JSON. A reader that assumed one format would corrupt or refuse a
file that opened fine last week.

## Decision

1. **zstd level 19, with one dictionary trained offline and embedded in the binary,
   via `github.com/klauspost/compress/zstd`** (pure Go, no cgo, the same posture as
   the bearoff `.bd` tables and `gnubg_os6.bd`). `CompressAnalysisData`
   (`pkg/blunderdb/engine/analysiscodec.go`) is the only thing that ever writes a new
   blob from here on.

2. **A blob names its own codec; nothing assumes.** `DecompressAnalysisData` tells
   raw JSON (`{`), zlib (`0x78 0x9C` — actually detected by attempting a zlib header
   parse), and zstd (magic `0x28 0xB5 0x2F 0xFD`, RFC 8878 §3.1.1) apart by content,
   never by a schema version, a side-channel column, or an assumption about which
   release wrote the row. The three signatures cannot collide. This is why the change
   needs no `DatabaseVersion` bump: the column's type and meaning are unchanged, only
   the bytes inside it vary, self-describing, exactly the invariant `CLAUDE.md` states
   for this column.

3. **One embedded dictionary, not a `_zstd_dicts` table.** P11's own reference design
   (`sqlite-zstd`) stores dictionaries in a database table so a second dictionary can
   be introduced without recompressing existing rows, each row remembering which
   `Dictionary_ID` compressed it. blunderDB does the equivalent with **less
   moving state**: the dictionary ships as a build-time asset
   (`pkg/blunderdb/engine/analysis_dict.bin`, trained by `cmd/train-analysis-dict` and
   embedded with `//go:embed`), the same mechanism already used for `gnubg_os6.bd`. A
   zstd frame carries its own `Dictionary_ID`, so `klauspost/compress/zstd`'s decoder
   already picks the right dictionary among however many are registered — nothing
   here forecloses adding a second dictionary later (a bigger corpus, a
   `PositionAnalysis` field change) by registering its bytes alongside this one in
   `zstdDecoder`'s dict set, at which point the row-remembers-its-ID design becomes
   necessary. It is not needed for a single dictionary, and no database row changes if
   it is added.

4. **Existing zlib and raw-JSON rows are read forever, upgraded opportunistically,
   never migrated in place by a schema step.** `RecompressAnalysisData` upgrades a row
   to zstd when it is touched anyway: on the native-`.db` import merge path, and in a
   full-table pass `sqlite.Storage.Vacuum` runs before compacting the file
   (`recompressLegacyAnalyses`, batched at 2,000 rows/transaction — P11's own
   1,000–5,000 recommendation). Vacuum already rewrites the whole file and already
   asks the user to accept an unpredictable-duration operation, which makes it the
   natural, honest trigger for a pass that is otherwise easy to never get around to.
   A row this pass cannot read is skipped and logged, not fatal: refusing the whole
   compaction over one bad row would be worse than leaving a few bytes unreclaimed.

5. **The decompression bomb bound already on this path is kept, and extended to the
   new codec rather than replaced.** `MaxAnalysisBytes` (16 MiB — a real analysis blob
   is a few KB, the largest ever stored is under 1 MB) now bounds zstd via
   `WithDecoderMaxMemory`, exactly as it bounded zlib via the existing `io.LimitReader`
   check. `codec_fuzz_test.go` carries one seed per format (raw JSON, zlib, zstd) plus
   a decompression bomb for each of the two compressed ones;
   `analysiscodec_bomb_test.go` proves a crafted 1 GiB-claiming zstd frame is refused
   in under a second, not after allocating for it — the posture the `.dbx` container
   (fiche A.11) already established for an exchanged file's other compressed
   payload.

## Considered options

**Ship no dictionary, zstd level 19 alone.** Rejected on the same measurement this
record cites: P11's threshold for abandoning the dictionary was "gain measured
< 15 % vs zstd without one" — this repository's blobs are small enough (2–20 KB, JSON,
the same field names and move-notation strings on every row) that the dictionary is
exactly the lever P11's small-object citations describe, and skipping it would have
left the ~45 % gap over zlib-9 on the table for no engineering saved (the codec
already needs the embed machinery for the alternative anyway).

**A binary serialisation (CBOR/MessagePack) instead of, or in addition to, JSON.**
Rejected — P11's finding that the gap collapses to ~10–20 % post-compression, and can
reverse, meant the schema-evolution cost of a second wire format bought little. JSON
stays; the dictionary is the actual lever.

**Migrate every row in a schema step (`DatabaseVersion` bump).** Rejected. The
invariant this repository already states — a schema bump is for DDL, not for bytes
inside an unchanged column — applies directly: nothing about `analysis`'s columns or
types changes, so forcing a migration would rewrite the whole table at open time
(unpredictable duration on a database the user did not ask to wait on) for a benefit
`Vacuum`'s existing opportunistic pass already delivers, at a moment the user already
expects to wait.

**A `_zstd_dicts` table with a per-row `Dictionary_ID`, mirroring `sqlite-zstd`.**
Deferred, not rejected: correct design for *N* dictionaries, unnecessary complexity
for the one this record ships. Revisit when a second dictionary is actually trained.

## Consequences

- New writes are always zstd+dict; reads accept all three formats forever.
  `NeedsRecompression` is the cheap (allocation-free) prefix check both the vacuum
  pass and any future caller use to skip already-current rows.
- `go.mod` gains `github.com/klauspost/compress` (pure Go, no cgo, already fuzz-tested
  upstream via OSS-Fuzz per P11).
- `cmd/train-analysis-dict` is a dev-time asset generator (the same role
  `scripts/build-demo-db.sh` plays for the demo database), not a shipped CLI
  subcommand and not a runtime dependency: `zstd --train` (the reference C CLI) is
  required only to regenerate `analysis_dict.bin`, never to read or write a database.
- No `DatabaseVersion` bump, no new migration step, no change to either storage
  backend's schema. The PostgreSQL and SQLite `Save`/`Load` paths are unchanged except
  for the shared `engine.CompressAnalysisData`/`DecompressAnalysisData` calls they
  already made.
- Regenerating the dictionary (a materially larger or different corpus, a
  `PositionAnalysis` field change) is `go run ./cmd/train-analysis-dict`; existing
  rows compressed under the old dictionary keep decoding because the frame's own
  `Dictionary_ID` selects among every dictionary `zstdDecoder` knows, per point 3
  above.
