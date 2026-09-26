# gammonNet is ported to Go, and the representation boundary sits at the evaluator's edge

Status: accepted.

## Context

blunderDB embeds [gammonNet](https://github.com/kevung/gammonNet) (MIT) so a position can be
evaluated with no XG, no gnubg and no network, in every mode. Upstream publishes only
WebAssembly, which would exist in the GUI alone and break CLI/GUI/server parity. Four of its
C modules are missing here (encoding, forward pass, search, cube); the others have blunderDB
counterparts. Reusing those inside the search loop is a ×100 trap: `domain.LegalMoves`
allocates two strings per candidate, ~5 000 candidates per 2-ply decision.

## Decision

1. **gammonNet's network, encoding, search and cube model are ported to Go**, targeting the
   Configuration upstream publishes, unchanged. Race leaves fall back to the network, as
   upstream's artifact does; blunderDB's two-sided table serves the panel, never the search's
   leaves.
2. **The representation boundary sits at the evaluator's edge.** A `domain.Position` is
   converted once, on entry; inside, everything stays in the engine's representation,
   allocation-free. blunderDB's routines serve the edge (notation, panel) and cold paths.
3. **Two move generators, one truth.** `domain.LegalMoves` stays canonical outside the search;
   the ported generator serves the search; over a corpus × all 21 rolls both produce the same
   set of resulting positions.
4. **Where blunderDB's counterpart is better and cold, it wins**: the MET (`engine/met.go`,
   the same Kazaross-XG2 table plus a Zadeh fallback beyond 25-away), precomputed per search
   into six outcome values; the Zobrist codec. The eval cache is ported, per search.
5. **The weights are embedded in float32** (`go:embed`), the reference artifact.
6. **The port is proven at two levels.** Network parity on upstream's `verify/reference.bin`
   at upstream's criterion, 1e-6 (measured 5.960e-08). Search parity on versioned gold files:
   the same chosen move at each ply, equities to 1e-6.
7. **The port follows the C.** A discrepancy from gammonNet is a bug, never an improvisation.
   A conceptual change — one whose gain survives a change of language — is written in
   gammonNet first, with its measurement, and the port follows; an implementation-only change
   (e.g. the AVX2 kernel) stays here. The gold files are regenerated only from a fixed C
   reference, never to accommodate a local change.
8. **`EngineVersion` names a real upstream tag** (`gammonNet vX.Y.Z`). Any change to the
   Configuration or to valuation semantics bumps it, and every stored analysis under an older
   label is stale as a whole (`AnalyzeStaleGammonNet`). Depth lives in `AnalysisDepth`, never
   in the label; other engines keep product names, since their version does not change a
   stored analysis.

## Consequences

- The evaluator is available to GUI, CLI and `serve`; no cgo, `cmd/serve` stays
  `CGO_ENABLED=0`.
- The release ships gammonNet's `LICENSE`, `NOTICE` and attribution; the panel carries a
  discreet link.
- Network parity covers the forward pass only; the encoding is proved separately (the opening
  position encodes identically from both sides; geometry pinned against `domain.LegalMoves`).
- Performance of the Go kernel is ADR-0024's subject.
- Rejected: **WASM in the webview** (one mode only); **cgo** (four-target C builds, broken
  `CGO_ENABLED=0`, C faults in-process); **a subprocess** (does not exist upstream, second
  binary to package); **a faithful replica** porting rules/MET/bearoff too (two generators
  and two METs in one binary, every upstream bump a re-port).

## Guard

`pkg/blunderdb/engine/gammonnet/`: `parity_test.go`, `gold_test.go`, `moves_diff_test.go`,
`encoding_test.go`, `staleness_test.go`.
