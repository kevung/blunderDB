# Transcribing a match is not playing one

Status: accepted.
See also: ADR-0037 (blunderDB does not play), ADR-0041, ADR-0045, ADR-0028.

## Context

A game played over a real board — filmed or on a score sheet — had no way into blunderDB
except through another program's export. Recording it means writing, move by move, dice,
checker plays, cube actions, resignations, the score and Crawford, and correcting them. Six
of the eight items by which ADR-0037 defines the play mode it refuses are exactly this; the
line between the two must be written down.

## Decision

blunderDB records matches played elsewhere; it still plays none. The line is three
properties a transcription has and a play mode cannot:

1. **Nobody decides.** Both sides are facts the user reports. The engine proposes candidates
   so the played move is picked faster than typed; it never chooses, rolls or answers a
   double.
2. **The rules check, they never enforce.** An illegal move that stood at the table is
   recorded as played, marked, and the game goes on from the board it left. `LegalMoves` is
   used to rank, recognise and flag — never to forbid.
3. **The output is a Match**, exactly what an import produces (Games, Moves, Positions),
   analysed by gammonNet, counted in the Performance Rating, exported to `.mat`. Nothing about
   it is special once saved.

Shared with the product, by name: the board play gesture of the Decision exercise
(`quizPlay.js`, same `LegalMoves`), the Eval panel's 0-ply candidates, the `.mat` renderer,
`ingest.WriteMatch`.

Out of scope for good: automatic doubles, and beaver/raccoon as recorded actions (no reader
for them in `.mat` or `Decide`). Jacoby and beaver stay session rules (ADR-0028).

## Consequences

- `LegalMoves` has a consumer that keeps boards it cannot reach and marks them; no code path
  may turn that mark into a refusal.
- gammonGo remains where a playing engine belongs.
- Rejected: refusing transcription under ADR-0037 (it feeds the analysis loop; the detour
  through gnubg or XG needs a program the user may not own); reopening a play mode (the
  shared surface is the cheap half; the three properties stay false for it); typed notation
  without engine candidates (two keystrokes a turn instead of ten, and ranking is not
  deciding).
