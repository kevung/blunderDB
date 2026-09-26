# Analysis blobs are zstd with a shared dictionary, and a blob names its own codec

Status: accepted.

## Context
`analysis.data` holds the compressed JSON `PositionAnalysis` of one position (2–20 KB of
repetitive JSON: the same field names, move notations and provenance strings on every row). The
redundancy is across rows, which a per-blob compressor never sees. On this repository's own
held-out blobs, zlib-9 reaches 4.37× and zstd-19 with a trained dictionary 7.98× — 45 % smaller,
confirmed on a whole vacuumed database (−46.8 %); decompression is faster than zlib, compression
~11× slower on a write-once path. Files travel between machines and releases, so a blob cannot
assume which binary wrote it.

## Decision
1. **zstd level 19 with one dictionary trained offline and embedded**
   (`pkg/blunderdb/engine/analysis_dict.bin`, via `github.com/klauspost/compress/zstd`, pure Go).
   `CompressAnalysisData` (`pkg/blunderdb/engine/analysiscodec.go`) is the only writer of a new
   blob.
2. **A blob names its own codec; nothing assumes.** `DecompressAnalysisData` tells raw JSON
   (`{`), zlib (header parse of `0x78 …`) and zstd (magic `0x28 0xB5 0x2F 0xFD`) apart by
   content, never by schema version, side column or release. The signatures cannot collide. The
   column's type and meaning are unchanged, so no `DatabaseVersion` bump.
3. **One embedded dictionary, no dictionary table.** A zstd frame carries its own
   `Dictionary_ID`; a second dictionary is added by registering its bytes in `zstdDecoder`'s
   set, and old rows keep decoding. No database row changes.
4. **Legacy rows are read forever and upgraded opportunistically, never by a schema step.**
   `RecompressAnalysisData` upgrades a row when it is touched anyway: on the native-`.db` import
   merge, and in `recompressLegacyAnalyses`, run by `sqlite.Storage.Vacuum` before compaction in
   batches of 2 000 rows per transaction. A row the pass cannot read is logged and skipped, not
   fatal. `NeedsRecompression` is the allocation-free prefix check that skips current rows.
5. **The decompression-bomb bound covers every codec.** `MaxAnalysisBytes` (16 MiB) bounds zstd
   via `WithDecoderMaxMemory` as it bounds zlib via `io.LimitReader`.

## Consequences
- `cmd/train-analysis-dict` regenerates the dictionary (80/20 split by content hash, ratio
  measured on the held-out part); it needs `zstd --train` and is never a runtime dependency.
- No change to either backend's schema; both call the shared codec.
- Rejected: zstd without a dictionary — leaves the cross-row redundancy, the actual lever.
- Rejected: CBOR/MessagePack — the gap to JSON shrinks to ~10–20 % after compression, sometimes
  reverses, for the cost of a second wire format.
- Rejected: migrating every row in a schema step — rewrites the table at open time for a gain
  `Vacuum` already delivers when the user expects to wait.
- Deferred: a `_zstd_dicts` table with a per-row dictionary id — needed only for several
  dictionaries.

## Guard
`pkg/blunderdb/engine/codec_fuzz_test.go` (one seed per format, a bomb per compressed codec),
`analysiscodec_bomb_test.go` (a 1 GiB-claiming frame refused in under a second),
`analysiscodec_golden_test.go`.
