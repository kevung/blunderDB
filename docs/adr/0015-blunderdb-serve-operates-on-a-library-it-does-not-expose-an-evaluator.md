# `blunderdb serve` operates on a library; it does not expose an evaluator

Status: accepted.
See also: ADR-0013.

## Context
With gammonNet in the Go core (ADR-0011), every mode can reach it. `serve` is the engine behind
gammonGo, whose ADR 0097 makes `gammonnet serve` the platform's single evaluator and states
that blunderDB stores analyses and does not evaluate. A stateless evaluation endpoint here would
give the platform two entry points to one engine.

## Decision
**`serve` exposes operations on a library, never a bare evaluator.** It exposes the batch
operation — analyse this tenant's positions that have no analysis (ADR-0013) — the same one the
CLI and GUI expose. It exposes no stateless `eval(position) → probabilities` endpoint.

> `blunderdb serve` operates on a library. `gammonnet serve` evaluates a position.

## Consequences
- A third party wanting only an evaluation goes through a library or runs `gammonnet serve`.
- Modes differ in the *form* exposed because their roles differ; `CLI_USAGE.md` and the
  server-mode documentation say so.
- Rejected: a stateless evaluation endpoint for full parity — two paths to one truth, and a
  prohibition in gammonGo's ADR to maintain.
- Rejected: nothing in `serve` — batch analysis is a library operation, and denying it headless
  breaks parity where it means something.

## Guard
`internal/server/handlers_gammonnet_test.go`.
