# engine/gammonnet — invariants

A Go port of gammonNet: its arithmetic is a contract. Read `doc.go` and the header of
`cube.go` first. Each invariant below is a bug if violated, even with every test green.

- **The port follows upstream.** A divergence from the C is a bug; a change to the cube model
  lands in gammonNet's `gn_cube.c` and its spec first, then here, then in
  `testdata/cube_gold.bin`. `EngineVersion` names a real gammonNet tag (ADR-0011).
- **Scales stay inside.** `Decide` returns MWC, `Value` returns 2×MWC−1; `EquityScale`
  (`referential.go`) converts once at the domain edge. Never print or store an internal scale
  (ADR-0019).
- **The live cube curve has no plateau.** `janowskiEquity`/`levelLive` run from `(0, −L)` to
  `(1, +W)`; clamping a tail to the cash equivalent makes `TooGood` unreachable. Jacoby lives in
  `Decide`'s no-double payoffs, not in the curve (ADR-0022).
- **Cube efficiency is per cube state, read at the root.** `DefaultEfficiency` returns one value
  per cube state; `SearchConfig.CubeX` is fixed at the root and `Decide` prices `eDT` at the
  current owner's coefficient — both match upstream line for line; "fixing" either here breaks
  the cube gold (ADR-0029).
- **A conceptual optimisation is decided upstream**: one whose gain survives a change of
  language is written in gammonNet first, with its measurement. Implementation ones (the AVX2
  kernel) stay here (gammonNet ADR-0003).
- **The kernel never fuses and never reassociates**: one position per SIMD lane, ascending-`j`
  float32 sums, multiply and add as two operations (`float32(a*b)` is the fusion barrier; it
  matters on arm64). A new arithmetic path passes `kernel_identity_test.go` bit for bit; a
  requested but unavailable kernel is an error at load, never a fallback (ADR-0024).
- **Parallelism never changes a result.** The batch runs positions over `NumCPU` goroutines,
  each with one serial `Searcher`; the live panel runs one search `WithWorkers`; the two never
  stack. The 21-roll weighted sum is always taken serially in ascending roll order (ADR-0024).
