# The evaluator has one arithmetic contract: no GPU, no WebAssembly kernel

Status: accepted.
See also: ADR-0024

## Context

ADR-0024 makes bit-identity the evaluator's contract, because a stored analysis must be
reproducible on another machine or "stale" has no definition. Two acceleration routes recur
whenever the evaluator is profiled. A GPU: the work is a small MLP (~600 000 multiply-adds per
position) fed in batches of `EvalBatchWidth` = 8, at most 63 positions at a root, in a serial
dependency chain, at 17 µs per forward pass. A WebAssembly kernel: WASM's fast path,
relaxed-SIMD, leaves `fma` and `min`/`max` implementation-defined.

## Decision

1. **A backend that cannot honour ADR-0024 rule 1 is not a backend.** The criterion is `==`
   against the pure-Go fallback in `kernel_identity_test.go`; speed is not weighed against it.
2. **No GPU backend.** Vendor BLAS and tensor cores reassociate and fuse by design; a GPU kernel
   forbidding both loses its advantage and still cannot be proved on hardware this project
   cannot enumerate. Batches of 8 in a serial chain cannot feed a device whose launch costs
   microseconds. CUDA/ROCm mean cgo, and `cmd/serve` is `CGO_ENABLED=0`. Reopening this means
   reopening ADR-0024, not running a benchmark.
3. **No WebAssembly kernel.** Relaxed-SIMD is banned outright: a browser-analysed and a
   native-analysed database would disagree under the same `EngineVersion`. A WASM build runs
   the pure-Go fallback — correct, not accelerated — and a plain-SIMD128 kernel is added only
   once `kernel_identity_test.go` can run on that target (ADR-0024 rule 3's bar for NEON).
4. **This rules out a backend, not a target.** Compiling other parts of blunderDB to WASM is
   untouched; the rule is about the evaluator's arithmetic only.
5. **Where the speed is**: the cube half is ADR-0032's; the network half's remaining gain is a
   distilled network, a new Configuration decided in gammonNet (ADR-0011 rule 7), not a backend
   swap.

## Consequences

- A GPU or WASM-kernel proposal arrives already answered; answering it again means arguing with
  ADR-0024's decision on stored analyses.
- The kernel seam (`kernel.go`, `BLUNDERDB_GAMMONNET_KERNEL`) stays the only place an arithmetic
  path is added.
- If bit-identity is ever traded away deliberately (a quantised network), this record falls with
  ADR-0024 and not before.
- Rejected: **a GPU path behind a build tag** (a second arithmetic with no owner); **relaxing the
  contract to the gold's 1e-6** (that tolerance absorbs the C-to-Go residual, not a second
  result; `AnalyzeStaleGammonNet` could no longer answer its question); **waiting for a WASM
  SIMD intrinsic** (the bar is the identity test passing, not an intrinsic existing).

## Guard

`pkg/blunderdb/engine/gammonnet/kernel_identity_test.go`.
