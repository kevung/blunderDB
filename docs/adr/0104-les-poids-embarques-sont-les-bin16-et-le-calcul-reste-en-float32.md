# Les poids embarqués sont les `.bin16` de gammonNet v1.8.1, le calcul reste en float32

Status: accepted. Replaces ADR-0011 decisions 5 and 6, and ADR-0093 decision 7.
See also: ADR-0011, `tasks/mesure-float16-score-2026-10-22.md`

## Context

gammonNet v1.8.1 publishes its network as float16 transport files (BGN6, `.bin16`) with a
float16 reference vector file (`reference16.bin`). ADR-0011 embedded the float32 weights and
proved parity on `verify/reference.bin`; ADR-0093 kept `EngineVersion` at v1.7.0 because
v1.8.0 changed no value. Rounding the weights to half precision does change the outputs, so
neither paragraph holds any more.

## Decision

1. **The embedded weights are the `.bin16` of gammonNet v1.8.1** (`go:embed`), for the main
   network and the pruning network. Each half is converted exactly to float32 on load.
2. **The computation stays in float32**: the forward pass, the kernels (scalar and AVX2) and
   the search are unchanged. Only the stored weights are rounded.
3. **The port is proven at two levels.** Network parity on upstream's `reference16.bin`, at
   upstream's criterion 1e-6 (measured 5.96e-08, scalar and batched kernels). Search parity on
   the versioned gold files, regenerated from the `.bin16` by the portable C recipe
   (`testdata/gold/README.md`): same chosen move at each ply, equities to 1e-6.
4. **The cost of rounding was measured**, at score and in money
   (`tasks/mesure-float16-score-2026-10-22.md`): outputs move by about 1e-4 (max|Δ| 3.84e-4
   against float32 on the network), no cube decision changes over 11 000 comparisons, and the
   12 checker-play changes out of about 14 900 are ties or cost at most 3.25e-4 in float32.
5. **`EngineVersion` is `gammonNet v1.8.1`.** The values moved, so every analysis stored under
   `gammonNet v1.7.0` is stale as a whole (ADR-0011 decision 8, `AnalyzeStaleGammonNet`).
   The demo database is rebuilt for the same reason.
