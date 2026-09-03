# The evaluator has one arithmetic contract: no GPU, no WebAssembly kernel

## Status

accepted — decided 2026-09-03. Closes the documentation half of #199 (plan 2026-09b, fiche
C.12). Completes *"The evaluator batches positions, one per lane, and keeps the scalar sum
order"* (0024) by writing down the two backends that record rules out **as a consequence**
without naming them, so the question stops being re-opened once a year by someone reading
"×1.5–2 is deliberately left on the table".

**Changes nothing.** No code, no schema, no `EngineVersion`. It is a line in the record so that
a proposal arrives already answered.

## Context

ADR-0024 made bit-identity the contract of the evaluator, for a reason that is a product
requirement rather than an aesthetic one: a gammonNet analysis stored in a `.db` must be
reproducible on another machine, **otherwise "stale" has no definition** and
`AnalyzeStaleGammonNet` cannot tell a analysis that needs re-running from one that merely ran
elsewhere. The kernel therefore accumulates in float32, in ascending `j`, with multiply and add
as two operations, and `kernel_identity_test.go` compares every path to the pure-Go fallback
with `==` on every bit.

Two acceleration routes come up whenever a neural evaluator is profiled, and neither has been
written down as considered-and-rejected:

**A GPU backend.** The workload is a 196→512→512→256→128→5 MLP, about 600 000 multiply-adds per
position. `EvalBatchWidth` is **8**; a node's fill pass sends its ≤ 12 surviving candidates, and
even the whole root fan-out of a 2-ply search is 63 positions. After the AVX2 kernel one forward
pass costs **17 µs** among siblings (ADR-0024's own table), and `EvaluateBatch` is 52.6 % of a
2-ply decision at a score (profile of 2026-09-03). So the unit of work offered to a device is a
few dozen positions at a time, arriving in a serial dependency chain — one batch's result
decides which positions the next batch contains.

**A WebAssembly kernel.** Go compiles to `js/wasm` and `wasip1/wasm`, and WASM has a 128-bit
SIMD extension, so a browser-side or sandboxed evaluator looks superficially free. The research
notes went and looked: `docs/recherche/P3-parallelisme-expectiminimax.md` and
`P4-gains-algorithmiques-amont.md` both establish that the fast WASM path — **relaxed-SIMD** —
leaves `fma` and `min`/`max` *implementation-defined* ("that lane's result is implementation
defined"), which is precisely the property the bit-identity test exists to forbid. Plain SIMD128
is deterministic; relaxed-SIMD is where the speed is.

## Decision

1. **The evaluator has one arithmetic contract, and a backend that cannot honour it is not a
   backend.** The contract is ADR-0024's: float32 accumulation, ascending `j`, multiply and add
   never fused, `==` against the pure-Go fallback on `kernel_identity_test.go`. This is the
   single criterion; speed is not weighed against it.

2. **No GPU backend.** It fails the contract and it fails the shape of the work:

   - *Exactness.* Vendor BLAS and every tensor-core path reassociate reductions and fuse
     multiply-add by design — that is where their throughput comes from. A GPU kernel written to
     forbid both would be a GPU kernel with its advantage removed, and it would still have to be
     proved bit-identical on hardware this project cannot enumerate, let alone put in CI.
     Reproducibility across *machines* is the requirement, and "the answer depends on which
     driver ran it" is the failure mode, not a tolerance.
   - *Shape.* Batches are 8 lanes wide, at most 63 positions at a root, produced by a serial
     dependency chain. A kernel launch costs microseconds; a forward pass now costs 17. There is
     no arrangement of this search in which the device is fed.
   - *Distribution.* CUDA/ROCm mean cgo, and `cmd/serve` is `CGO_ENABLED=0` on purpose
     (ADR-0011 chose the Go port to avoid cgo); a GPU path would fork the binary into an
     accelerated build and a real build, each answering differently.

   The route is not "hard", it is **incompatible**. Reopening it means reopening ADR-0024's
   decision about stored analyses, and that is the discussion to have — not a benchmark.

3. **No WebAssembly kernel, and the distinction matters.** *Relaxed-SIMD is banned outright*: it
   makes `fma` and `min`/`max` implementation-defined, so a database analysed in a browser and
   one analysed natively would disagree while both claimed the same `EngineVersion`. Plain
   SIMD128 is deterministic and could in principle be made to pass the identity test — but Go's
   own `simd`/`archsimd` support for WASM lands in 1.27 (this project is on 1.25, per P1), and
   the pure-Go fallback already runs correctly under `js/wasm` today. So the position is: **a
   WASM build runs the fallback, which is correct and simply not accelerated** — exactly the
   arm64 situation ADR-0024 already accepted — and a hand-written WASM kernel is not written
   until someone can execute `kernel_identity_test.go` on that target, which is the same bar
   NEON is held to.

4. **This rules out a backend, not a target.** Compiling *other* parts of blunderDB to WASM is
   untouched — `docs/recherche/P12-diagrammes-svg.md` proposes exactly that for the SVG diagram
   generator, and nothing there computes an equity. The rule is about the evaluator's
   arithmetic, and only that.

5. **Where the speed actually is, for the record.** The measured ordering after C.8/C.9/C.10:
   `EvaluateBatch` 52.6 % of a 2-ply decision at a score, `levelSolve` 35.4 %. The cube half has
   a decided answer that is upstream's to write (the closed-form inversion record). The network
   half's remaining ×1.5–2 is the one ADR-0024 deliberately declined, and a distilled 60–100k-MAC
   network — P4's priority-1 upstream item, ×5–9 — is a **new Configuration** decided in
   gammonNet with its strength gauge, not a backend swap. Both are more valuable than either
   route rejected here, and neither costs the contract.

## Considered options

**Ship a GPU path behind a build tag, unused by default.** Rejected: a second arithmetic with no
owner. Either its results are the stored ones — and then the contract is broken for everyone who
builds with the tag — or they are not, and it is a benchmark that lies about the product.

**Relax the contract to a tolerance (1e-6, the gold suites' own) and let both routes in.**
Rejected, and it is the same rejection ADR-0024 already made: the gold tolerance exists to
absorb the *C-to-Go port's* residual, not to license a second result. Under a tolerance,
`AnalyzeStaleGammonNet` cannot answer its own question, and two machines analysing the same
database produce rows that differ in the third decimal with no way to tell which is current.

**Wait for Go 1.27's `archsimd` WASM back end and revisit.** Deferred rather than rejected, and
that is what decision 3 says: the bar is `kernel_identity_test.go` passing on the target, not the
availability of an intrinsic. Nothing about this record has to change for that day to come; a
kernel simply gets added to the seam the way AVX2 did.

**Say nothing and let the profile speak.** Rejected — that is the status quo, and it is why the
question keeps coming back. ADR-0024's "a ×1.5–2 on the kernel is deliberately left on the
table" reads, to someone who has not read the whole record, as an invitation.

## Consequences

- A GPU proposal is answered by this record, and answering it again means arguing with
  ADR-0024's decision about stored analyses rather than with a benchmark.
- A WASM build of blunderDB remains buildable and correct — on the pure-Go fallback — and is
  never faster by being less exact.
- The kernel seam (`kernel.go`, `BLUNDERDB_GAMMONNET_KERNEL=go|avx2|neon`) stays the only place
  an arithmetic path is added, and every entry through it passes the same identity test.
- If bit-identity is ever traded away deliberately — for a quantised network, say — this record
  falls with ADR-0024 and not before.
