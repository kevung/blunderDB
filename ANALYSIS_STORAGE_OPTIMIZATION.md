# Analysis Storage Optimization Plan

## Problem

The `analysis.data` column stores the full `PositionAnalysis` struct as verbose
JSON. This is the largest consumer of database storage:

- Each `CheckerMove` repeats ~155 characters of field names (`playerWinChance`,
  `opponentBackgammonChance`, …). With 10-20 candidate moves per position,
  field names alone consume 1500-3100 bytes per analysis row.
- `DoublingCubeAnalysis` adds ~250 bytes of field names.
- Typical analysis JSON: 2500-4000 bytes per row.

Meanwhile the scalar columns (`cube_error`, `player1_win_rate`, etc.) already
store the fields needed for filtering/searching. The JSON blob is only fully
parsed when displaying a single position's analysis detail.

## Approach: zlib compression of analysis.data

Compress the JSON blob with zlib before writing to the `data` column.
Auto-detect compressed vs raw JSON on read for full backward compatibility.

**Why zlib rather than short JSON keys or binary encoding:**

| Criterion | zlib | Short keys | Binary (msgpack) |
|---|---|---|---|
| Storage savings | 60-80% | 35-50% | 50-65% |
| Code complexity | Low (wrap marshal) | Medium (custom types) | Medium (new dependency) |
| Backward compat | Auto-detect magic byte | Must support both key sets | Must support both formats |
| Debug-ability | `zlib -d` on blob | Readable but cryptic | Not human-readable |

## Implementation

### 1. Helper functions (db.go)

```go
func compressAnalysisData(jsonData []byte) ([]byte, error)      // zlib best-compression
func decompressAnalysisData(data []byte) ([]byte, error)         // auto-detect: '{' = raw JSON
func encodeAnalysisForStorage(a *PositionAnalysis) ([]byte, error) // marshal + compress
func decodeAnalysisFromStorage(data []byte) (*PositionAnalysis, error) // decompress + unmarshal
```

### 2. All read paths

Change `var analysisJSON string` → `var analysisData []byte` and replace
`json.Unmarshal([]byte(analysisJSON), &a)` with `decodeAnalysisFromStorage(analysisData)`.

Affected functions:
- `SaveAnalysis` (read existing for merge)
- `LoadAnalysis`
- `saveAnalysisInTx` (read existing for merge)
- `saveCubeAnalysisForCheckerPositionInTx`
- `saveBGFCubeAnalysisForCheckerPositionInTx`
- `CommitImportDatabase` (read import DB + current DB)
- `AnalyzeImportDatabase` (read both DBs for comparison)
- `ExportDatabase` (read current DB)

### 3. All write paths

Replace `json.Marshal(analysis)` → `encodeAnalysisForStorage(&analysis)` and
pass `[]byte` directly to `sql.Exec` (SQLite stores as BLOB).

Export writes **uncompressed** JSON for backward compatibility with older
blunderDB versions.

### 4. Migration (2.2.0 → 2.3.0)

- Read all analysis rows in batches
- For each row: if not already compressed, compress and update
- Bump `database_version` to `2.3.0`
- Run `ANALYZE`

### 5. Version bump

`DatabaseVersion = "2.3.0"` in `model.go`

## What does NOT change

- Struct types (`PositionAnalysis`, `CheckerMove`, etc.) — unchanged
- Frontend API — Wails serializes Go structs, unaffected by storage format
- Scalar filter columns — unchanged, search performance preserved
- Import speed — zlib compression of ~3KB blobs is <10µs/op
- Search performance — scalar columns drive SQL filtering; JSON blob only
  parsed for display

## Expected savings

- Analysis data column: 60-80% smaller
- Overall DB size: 30-50% smaller (analysis is the dominant table)
- Import: negligible overhead (~10µs per position for compression)
- Load single analysis: negligible overhead (~5µs for decompression)
