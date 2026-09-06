# Transcribing a match is not playing one

## Status

accepted — decided 2026-09-07, after a design interview on 2026-09-06/07. Amends 0037:
that record refuses a play mode and lists the surface a play mode needs; this one says
why a *transcription* of a match played elsewhere lands on most of that surface and is
accepted anyway, and where the line between the two runs. Leans on 0041, which drew the
first refinement of 0037 (the engine playing a few plies out of sight to *make* a
position). Detailed by 0044 (the draft, its match, its inconsistencies).

## Context

blunderDB's user brings games already played and asks what went wrong in them. Until now
every game came in through a file — XG, GNU Backgammon, BGBlitz, a `.mat`. A game played
at a club, over a real board, filmed or written on a score sheet, had no door: the
user had to type it into another program first and import the export.

Adding that door means writing, into blunderDB, a move-by-move record of a match — the
dice, the checker plays, the cube actions, the resignations, the score advancing, the
Crawford game — and correcting it, since a score sheet is misread and a video rewound.

ADR-0037 refused a play mode with this sentence: *"a play mode is the rules of
backgammon end to end — legal-move enforcement at the interface, dice the user believes
are fair, cube offers and answers, a match score that advances, a game history that can
be reviewed and taken back, resignations, the Crawford rule, an undo."* Six of those
eight items are exactly what a transcription needs. The question therefore has to be
answered in writing, or the refusal of 0037 will be read as covering this too — or,
worse, the acceptance of this will be read as reopening 0037.

## Decision

**blunderDB records matches played elsewhere. It still plays none.** The line between
the two is not the size of the surface; it is three properties a transcription has and a
play mode cannot have.

1. **Nobody decides.** Both sides of a transcribed match are facts the user reports:
   the dice were rolled at a table, the moves were made by two people, the cube was
   turned by one of them. The engine proposes candidates so the user can pick the one
   that was played faster than by typing it; it never chooses, never rolls, never
   answers a double. A play mode is defined by the program deciding at least one side.

2. **The rules check, they never enforce.** In a play mode an illegal move is refused,
   because the game would otherwise be wrong. In a transcription an illegal move that
   stood at the table is a fact of the match: it is recorded as played, marked as
   illegal, and the game goes on from the board it left. The legal-move generator is
   used to *rank and recognise*, and to *flag* — never to *forbid*.

3. **The output is a Match, the input of the analysis loop.** What a transcription
   produces is exactly what an import produces: a Match with its Games, Moves and
   Positions, analysed by gammonNet like any other, counted in the same Performance
   Rating, exported to the same `.mat`. Nothing about a transcribed match is special
   once it is saved. A play mode would produce a game the program took part in, which is
   a different object with a different question attached ("did I beat it?").

What a transcription **shares** with the rest of the product is named rather than
hidden: the gesture of playing a checker move on the board is the one the Decision
exercise of Training uses (#294, `quizPlay.js`), driven by the same `LegalMoves`; the
candidate list is the Eval panel's 0-ply Evaluation; the `.mat` renderer is the export's;
the writer of the saved Match is the importers' `ingest.WriteMatch`.

What is **out of scope, definitively**: automatic doubles (the author does not want the
rule) and the beaver/raccoon as recorded actions (no reader exists for them: neither the
`.mat` format as blunderDB reads and writes it, nor `Decide`). Jacoby and beaver stay
what 0028 made them — rules of the session, flags on the positions — and gammonNet keeps
applying Jacoby and ignoring beaver, as it does for imported positions.

## Considered options

- **Keep 0037 as is and refuse transcription too.** Rejected: 0037's argument was that a
  play mode answers a need every other program answers and that blunderDB's loop does
  not need it. Transcription is the opposite case — it feeds the loop, and the
  alternative (type the match into gnubg or XG, export, import) is a detour through a
  program the user may not own.
- **Reopen 0037 and accept a play mode alongside**, since so much surface is shared.
  Rejected: the shared surface is the cheap half. What 0037 refused — dice the user
  trusts, an opponent, a game the program is a party to — is precisely what a
  transcription does not need, and the three properties above stay false for a play mode
  however much code it would reuse.
- **Transcription without the engine's candidates**, typed notation only, to keep the
  evaluator out of the picture. Rejected: the candidate list is what makes a turn cost
  two keystrokes instead of ten, and the engine ranking candidates the user then picks
  from is not the engine deciding.

## Consequences

- 0037 gains one line in its *Status*: amended by this record. Its refusal stands for
  what it refused; a reader who meets "legal-move enforcement" or "a score that
  advances" in `pkg/blunderdb/transcript/` is meant to find this record next to it.
- The legal-move generator acquires a second consumer that must tolerate what it does
  not produce: a transcription keeps a board `LegalMoves` cannot reach and marks it. No
  code path may turn that mark into a refusal.
- gammonGo remains where a playing engine belongs; nothing here moves that line.
- The next record, 0044, says what a transcription *is* in the database and how its
  saved Match behaves.
