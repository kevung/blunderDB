# gammonNet is ported to Go, and the representation boundary sits at the evaluator's edge

## Status

accepted — decided 2026-08-28, tracked in the gammonNet integration issues

## Context

[gammonNet](https://github.com/kevung/gammonNet) v1.0.1 (MIT) is a measured backgammon
evaluator: a `strehl-prob5-512-512-256-128` network, expectiminimax search to 4 ply with a
pruning network, Kazaross-XG2 match equities, a Janowski cube model. Its published strength
is *equivalent to GNU Backgammon at 2-ply, confirmed* — "superior" is explicitly not
established, and eXtreme Gammon was not measured. Error rate against a gnubg 3-ply arbiter
over 600 contact decisions: PR 1.088 at 0-ply, 0.375 at 2-ply with pruning `k=12` (the
default, ×3.9 cheaper).

blunderDB wants it embedded so a library position can be evaluated with no XG, no gnubg and
no network. Four ways in were available, and the shape of the release decided against three
of them:

- **The published artifacts are WebAssembly only** — `gammonnet.wasm` (85 KB), its `.mjs`
  glue, the `api/` worker and pool modules, and the weights. `libgammonnet.so` exists in the
  source tree but is not a release artifact. Consuming the WASM inside the Wails webview
  would cost nothing to build, but the evaluator would then exist **only in GUI mode** — no
  `blunderdb` subcommand, no batch analysis of a library, nothing headless. That forks a
  capability by mode, which is the one thing the CLI/GUI/server parity invariant forbids.
- **cgo** would give parity, at the price of building a C library for four CI targets
  including a macOS universal fat binary, breaking `CGO_ENABLED=0` for `cmd/serve`, and
  putting a C crash inside the process.
- **A `gammonnet serve` subprocess** does not exist yet upstream, and shipping a second
  binary through AUR, flatpak, `.exe` and `.dmg` is heavy.

What made a Go port small was measuring what blunderDB already has. Of gammonNet's eleven C
modules, four are genuinely missing here: `gn_encoding` (the 196-feature perspective
encoding), `gn_infer` (the MLP forward pass and the `BGNN` weight loader), `gn_search`, and
`gn_cube`. The rest have counterparts: `domain.LegalMoves`, `engine/met.go` (**the same**
Kazaross-XG2 table, 25-away, pre- and post-Crawford, with a Zadeh fallback beyond 25 where
gammonNet refuses the state), `engine/race/` and the Zobrist/bitboard codecs. And
`domain.Position.Score` is already an *away* score — the exact shape of `GnMatchState`.

Reusing those counterparts wholesale, however, would have been a performance disaster in one
place. `domain.LegalMoves` returns `[]LegalPlay`, where every play carries a `Steps` slice, a
full `Result Position` (26 points of `{Checkers, Color}`), **and a `Notation string`** — and
deduplication runs through `boardKey(p *Position) string`, a second string per play. Two
string allocations per candidate, in a 2-ply `k=12` search that generates on the order of
5000 plays per decision, is a factor of a hundred, not a detail. `race.Evaluate` is worse
still as a leaf oracle: it takes a `*domain.Position` and runs EPC, convolution, a calibrated
correction and a two-sided lookup.

## Decision

**gammonNet's network, encoding, search and cube model are ported to Go.** The port targets
the Configuration upstream publishes, unchanged: the same network, the same search, the same
match equities, and the same endgame behaviour — race leaves fall back to the network,
because gammonNet's exact table (1.2 GiB) is not in the artifact either, and its absence
costs a measured 0.00028 equity per bearoff decision. blunderDB's own two-sided table serves
the *panel*, never the search's leaves. The Configuration is therefore upstream's, and the
label `gammonNet v1.0.1` is honest — conditioned on the proof below.

**The representation boundary sits at the evaluator's edge, never inside its loop.** A
`domain.Position` is converted to the engine's representation **once**, on entry. Inside,
everything stays in the engine's representation, allocation-free, using the ported routines.
blunderDB's routines serve at the edge — presentation, notation, the panel — and on cold
paths.

**Two move generators therefore coexist, and a differential test keeps them one truth.**
`domain.LegalMoves` remains the canonical generator for everything outside the search
(it is exposed at `/v1/positions.legalMoves`); the ported generator serves the search. Over a
corpus of positions × all 21 rolls, the two must produce the **same set of resulting
positions**. Having two implementations is acceptable; having two answers is not.

**Where blunderDB's counterpart is better and cold enough, it wins.** The MET stays
blunderDB's — same table, plus a fallback beyond 25-away that gammonNet refuses — but the
distribution→MWC conversion is precomputed once per search into six outcome values, since the
score cannot change during one, leaving the leaf six multiply-adds. The Zobrist codec stays
blunderDB's: it carries the deduplication invariant. The eval cache is ported, and improves
in the crossing: upstream declares itself not thread-safe *on purpose* ("parallelism is by
PROCESS ... never by thread"), a premise a per-search Go cache simply does not have.

**The weights are embedded, in float32.** `go:embed` of the `.bin` artifact (2 113 592 bytes)
next to the 1.4 MB and 6.8 MB bearoff tables already embedded. A desktop application
transports nothing, so float16 — a *transport* format, halving a download for 0.015 % of
decisions moved — answers a constraint that does not exist here. The reference artifact is
what gets written into users' databases.

**The port is proven, not asserted, at two levels.** Network parity: on the published
`verify/reference.bin` — 2000 positions of pre-encoded features with their reference outputs —
the five probabilities must reproduce the C reference to gammonNet's own published criterion,
**1e-6**. (The often-quoted 4.77e-07 is the worst deviation gammonNet *measured* across seven
platforms, not the threshold it set; a measurement used as a threshold fails on a machine that
is merely different rather than wrong.) Measured on this port: **max|Δ| = 5.960e-08** — one ulp
of float32 near 1, which is the signature of the hidden layers being bit-exact and the only
divergence coming from the final sigmoid, where the reference calls `expf` and Go rounds a
float64 `exp`. That exactness is not free: the accumulation must stay float32, ascending from
the bias, and each product carries an explicit `float32(...)` conversion to forbid the compiler
from contracting the multiply-add into an FMA on the architectures where Go fuses. Search parity: on a versioned gold file, the **chosen
move** must match the C reference at each ply, with equities to 1e-6. The gold file is
regenerated deliberately on an upstream bump, never silently, which means the C reference
must be buildable outside CI and that procedure must be written down.

## Considered options

- **WASM in the webview.** Zero build cost, the same artifact as gammonGo, no cgo. Rejected:
  the evaluator would exist in one mode only.
- **cgo against `libgammonnet`.** Full parity. Rejected: four-target C builds including a
  macOS universal fat binary, `CGO_ENABLED=0` broken for `cmd/serve`, and a C fault inside
  the process — the same reasoning that removed cgo from gammonGo's server.
- **A `gammonnet serve` subprocess.** Isolates faults, full parity. Rejected: the mode does
  not exist upstream, and a second binary burdens every packaging target.
- **A faithful replica**, porting `gn_rules`, `gn_met` and `gn_bearoff` too, for end-to-end
  parity with the C reference. Rejected: two move generators *and* two METs in one binary,
  an endgame that regresses against what blunderDB already does better, and every upstream
  bump becoming a re-porting exercise.

## Consequences

- The evaluator is available to the GUI, to the CLI, and to `serve` — in the form each mode
  warrants (see the library-operations boundary in the serve API).
- `AnalysisEngineVersion` gains the value `gammonNet v1.0.1`, the only engine label carrying
  a version. The others (`XG`, `GNU Backgammon`, `BGBlitz`) are product names, because their
  version does not change a stored analysis; a weights bump does. It is the same string
  gammonGo already writes, so one concept keeps one key across both products.
- Depth belongs in `AnalysisDepth`, as it does for every other engine — never in the name.
- The release ships `LICENSE`, `NOTICE` and `THIRD-PARTY.md`; the source tree carries an SPDX
  MIT header per file, and the vendored network keeps Alexander Strehl's paternity. The port
  carries the notice and the attribution alongside the weights, and repeats them in the
  Acknowledgements.
- Network parity covers the forward pass only — `verify/reference.bin` supplies features, not
  positions. The **encoding** and the domain→engine conversion need their own proof, and the
  strongest one available is internal: the opening position is symmetric, so it must encode
  identically from both players' point of view (which catches a reversed mirroring and a
  swapped colour identifier at once), and the geometry is pinned against `domain.LegalMoves`
  — if domain point 24 really is White's ace point, a checker there bears off on a 1.
- A visible, discreet attribution — one word and a link to the repository — accompanies the
  panel.
