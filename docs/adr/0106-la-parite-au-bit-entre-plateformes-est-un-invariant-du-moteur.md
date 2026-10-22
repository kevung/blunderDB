# La parité au bit entre plateformes est un invariant du moteur

Status: accepted.
See also: ADR-0024 (the kernel never fuses), ADR-0093 (parallelism never changes a result)

## Context

The Go compiler fuses `a*b + c` into one fused multiply-add on arm64 always, and on amd64 from
`GOAMD64=v3`: one rounding where the default amd64 build takes two. `math.Exp` is fused too,
in assembly: on arm64, and on amd64 at run time whenever the CPU has FMA. The network kernel
already barred fusion (ADR-0024), but the cube model, the MET, the race convolution, EPC, the
averages over the rolls in the search and the rollout accumulators did not. Measured on the
gold corpora, an engine built at `GOAMD64=v3` answered differently in the last bits from the
default amd64 build, and the same drift had turned a Crawford cube test red on macOS arm64.

## Decision

1. **Every platform reproduces amd64 at the default `GOAMD64` level, bit for bit**, for every
   package under `pkg/blunderdb/engine/`. An analysis compared across machines, a gold, a
   fingerprint is the same number everywhere.
2. **A product that is summed is rounded on its own**: `float64(a*b)` (`float32(a*b)` in
   float32), a conversion the compiler must round and so cannot fuse. It is a no-op on default
   amd64, so no value there moves.
3. **The sigmoid uses `portableExp`** (`exp.go`), Go's pure algorithm with the same barriers,
   instead of `math.Exp`. That the sigmoid's float32 result is unchanged was checked once,
   exhaustively, over every float32 input on an FMA-capable amd64; only the corpus digest
   (decision 4) holds it continuously.
4. **Held by three tests.** `TestEngineEmitsNoFusedMultiplyAdd` reads the compiler's `-S`
   listing of `engine/...` for arm64 and amd64-v3 and fails on any fused instruction (a canary
   package proves the detector). `TestEvaluationBitsMatchThePinnedDigest` pins the SHA-256 of
   the engine's answers on the gold corpora; it runs on macOS arm64 at every push (`test-os`).
   The `parity-amd64-v3` job of `build.yml` runs both, and the golds, at `GOAMD64=v3`.

## Consequences

- A new sum of products in the engine either carries its barrier or turns the assembler test
  red; the digest catches a library call that is not portable.
- An intentional change of the engine's arithmetic re-pins the digest, from default amd64.
- `pkg/blunderdb/domain` (statistics such as MWC7 intervals) is outside this invariant.
