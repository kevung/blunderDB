# blunderDB does not play backgammon

Status: accepted.
See also: ADR-0041 (engine plies behind a training position), ADR-0044 (transcription),
ADR-0047 (tournament direction).

## Context
blunderDB embeds a full evaluator (gammonNet: 2-ply search, Janowski cube, match score, tens of
milliseconds). Everything a program needs to *play* is in the binary, so the question "why can I
not play against it?" returns on its own. The honest answer is not technical difficulty.

## Decision
**blunderDB analyses positions. It does not play games.**

1. A play mode is not the evaluator plus a button: it is the rules end to end — legal-move
   enforcement, dice the user trusts, cube offers and answers, an advancing score, reviewable
   and undoable history, resignations, Crawford — each a surface to get right, document in nine
   languages and maintain. The evaluator is the small part.
2. blunderDB's user brings games already played elsewhere (XG, GNU Backgammon, an online
   platform, a board) and asks what they got wrong. Nothing in that loop needs a game played
   inside the tool.
3. What the demand actually wants is built: *playing the move yourself* is the quiz mode
   (position drawn from a filter, move played on the board, error measured against the stored
   analysis); *asking the engine what it would do* is the Eval panel.
4. Features that feed that loop without playing a game inside the tool are compatible with this
   refusal: transcribing a match played elsewhere (ADR-0044) and directing a tournament whose
   Matches are then transcribed or imported (ADR-0047).
5. A playing engine, if ever wanted, belongs in a different program: gammonGo, which embeds
   `pkg/blunderdb/server.Bootstrap`.

## Consequences
- Nothing is refused in the evaluator; what is refused is the surface around it.
- Reopening this means overturning the sentence above, and wants a measurement of demand, not an
  argument that it is technically easy.
