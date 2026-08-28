# `blunderdb serve` operates on a library; it does not expose an evaluator

## Status

accepted — decided 2026-08-28

## Context

Once gammonNet is ported into the Go core (ADR-0011), every mode can reach it. The parity
invariant says database logic lives on the shared core and is exposed to the frontend, the
CLI and the server handlers, never forked into a mode-specific helper — and the port honours
that. The remaining question is what shape `serve` exposes.

`serve` is not only blunderDB's headless mode; it is the engine behind gammonGo, whose ADR
0097 makes gammonNet the platform's single evaluator through a dedicated `gammonnet serve`
container, and states the boundary plainly: *blunderDB stores analyses, it does not evaluate*.
A stateless `eval(xgid) → probabilities` endpoint on `blunderdb serve` would give the platform
two entry points to one engine, and two paths to one truth is the failure mode the whole
frontier discipline exists to prevent.

## Decision

**`serve` exposes operations on a library, never a bare evaluator.**

It gains the batch operation — *analyse the positions of this tenant that have no analysis* —
which is the same operation the CLI and the GUI expose, because it acts on stored positions
and writes stored analyses. It gains no stateless evaluation endpoint.

The boundary states in one sentence, which is the best evidence it is drawn in the right
place:

> **`blunderdb serve` operates on a library. `gammonnet serve` evaluates a position.**

ADR 0097 therefore needs no amendment: of the API gammonGo consumes, it remains literally
true that blunderDB does not evaluate. It produces analyses for its own stock. There is no
prohibition to write down and none to forget.

## Considered options

- **Full parity, including a stateless evaluation endpoint.** The invariant honoured to the
  letter, and useful to a third party running blunderDB headless. Rejected: two entry points
  to one engine, and a sibling ADR to amend or a prohibition to document — which is a
  prohibition to eventually forget.
- **Nothing in `serve` at all**, GUI and CLI only. The frontier untouched, no new HTTP
  surface. Rejected: batch analysis of a library is a library operation by definition, and
  denying it headless breaks parity where parity actually means something.

## Consequences

- A third party wanting only to evaluate a position must go through a library, or run
  `gammonnet serve`. A real detour, for a need nobody has expressed.
- What differs between modes is the *form* exposed, because the modes have different roles —
  not because a mode was left half-built. That distinction belongs in `CLI_USAGE.md` and in
  the server-mode documentation.
