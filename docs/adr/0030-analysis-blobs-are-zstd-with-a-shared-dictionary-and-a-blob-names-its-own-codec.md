# Analysis blobs are zstd with a shared dictionary, and a blob names its own codec

Status: accepted.
See also: ADR-0070 (the binary payload every write produces, and its dictionary)

## Context
`analysis.data` holds the compressed `PositionAnalysis` of one position — repetitive content:
the same field names, move notations and provenance strings on every row. The redundancy is
across rows, which a per-blob compressor never sees. On this repository's own
held-out blobs, zlib-9 reaches 4.37× and zstd-19 with a trained dictionary 7.98× — 45 % smaller,
confirmed on a whole vacuumed database (−46.8 %); decompression is faster than zlib. The write
path is not write-once: a position is re-encoded every time an import merges into it, and on
real JSON blobs (mean 1 993 B) level 19 costs 5.97 ms per blob against 0.31 ms at level 7, for
277 B against 304 B (+9.7 %); on the binary payload of ADR-0070 the gap is under 2 % of a blob.
Files travel between machines and releases, so a blob cannot assume which binary wrote it.

## Decision
1. **zstd with dictionaries trained offline and embedded**
   (`pkg/blunderdb/engine/analysis_dict.bin` for the JSON formats,
   `pkg/blunderdb/engine/analysis_bin_dict.bin` for ADR-0070's payload, via
   `github.com/klauspost/compress/zstd`, pure Go), **level 7 on every write.**
   `EncodeAnalysisForStorage` (`pkg/blunderdb/engine/analysiscodec.go`, reached by
   `CompressAnalysisData`) writes every blob at level 7 (`SpeedBetterCompression`) without a
   content checksum. A level-19 frame (with the checksum flag) is still read.
2. **A blob names its own codec; nothing assumes.** `DecodeAnalysisFromStorage` /
   `DecompressAnalysisData` tell raw JSON (`{`), zlib (header parse of `0x78 …`), JSON in zstd
   (magic `0x28 0xB5 0x2F 0xFD`) and the binary blob (`0xBA`, ADR-0070 rule 1) apart by content,
   never by schema version, side column or release. The signatures cannot collide. The column's
   type and meaning are unchanged, so no `DatabaseVersion` bump.
3. **Embedded dictionaries, no dictionary table.** A zstd frame carries its own
   `Dictionary_ID`; every dictionary is registered in `zstdDecoder`'s set, so old rows keep
   decoding. No database row changes.
4. **Legacy rows are read forever and upgraded opportunistically, never by a schema step.**
   `RecompressAnalysisData` upgrades a legacy row to the binary format when it is touched
   anyway, on the native-`.db` import merge (`NeedsRecompression` is its allocation-free prefix
   check). `compactAnalyses`, run by `sqlite.Storage.Vacuum` before compaction in batches of
   2 000 rows, rewrites every row still in a legacy format, across all cores
   (`RecompressAnalysesConcurrently`); a binary row is left alone. A row the pass cannot read
   is logged and skipped, not fatal.
5. **The decompression-bomb bound covers every codec.** `MaxAnalysisBytes` (16 MiB) bounds zstd
   via `WithDecoderMaxMemory` as it bounds zlib via `io.LimitReader`.

## Consequences
- `cmd/train-analysis-dict` regenerates the dictionary (80/20 split by content hash, ratio
  measured on the held-out part); it needs `zstd --train` and is never a runtime dependency.
- No change to either backend's schema; both call the shared codec.
- Only the SQLite `Vacuum` upgrades legacy rows in bulk. On PostgreSQL the server's compaction
  is the database's own TOAST storage, not a pass this code runs; a file migrated to SQLite is
  upgraded there.
- Blobs carry no content checksum: a corrupted one fails payload decoding instead of the frame
  check.
- Rejected: level 19 on every write, or on compaction — twenty times the CPU of the import path
  for under 2 % of a binary blob's bytes, hours of CPU on a large library.
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
