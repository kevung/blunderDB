# blunderDB does not play backgammon

## Status

accepted — 2026-09-06. Closes issue #300 (fiche J.10), which had been marked
"écarté" in the plan since it was written and never had the one line that keeps
the question from being reopened.

## Context

blunderDB embeds a full evaluator. gammonNet plays a 2-ply search, judges the
cube through a Janowski model, honours the match score, and answers in a few
tens of milliseconds. Everything a program needs in order to *play* a game of
backgammon is already in the binary.

The question therefore comes back on its own, roughly once per person who
notices: why can I not press a button and play against it?

It is a reasonable question and it deserves a written answer rather than a
shrug, because the honest reason is not technical difficulty.

## Decision

**blunderDB analyses positions. It does not play games.**

A play mode is not the evaluator plus a button. It is the rules of backgammon
end to end — legal-move enforcement at the interface, dice the user believes
are fair, cube offers and answers, a match score that advances, a game history
that can be reviewed and taken back, resignations, the Crawford rule, an
undo — and every one of those is a surface that has to be right, documented in
nine languages, and maintained. The evaluator is the part that already exists;
it is also the small part.

What makes that surface *not worth* building here is what blunderDB is for.
Its user brings games they have already played, and asks what they got wrong.
Nothing in that loop needs a game to be played inside the tool: the games come
from XG, from GNU Backgammon, from an online platform, from a board. A play
mode would be a second product sharing a binary with the first, competing for
the same maintenance, and answering a need every backgammon program on the
market already answers.

**What the demand behind the question actually wants, blunderDB does build**:

- *playing the move yourself instead of reading it* — that is the quiz mode
  (J.4, issue #294), where a position is drawn from a filter, the move is
  played on the board, and the error is measured against the stored analysis.
  It is the input surface of a play mode without the rest of the game.
- *asking the engine what it would do here* — that is the Eval panel, on the
  position already in front of the user.

**If the demand ever exceeds that**, the answer is a different program.
gammonGo exists, embeds `pkg/blunderdb/server.Bootstrap`, and is where a
playing engine belongs.

## Consequences

The question is closed, and this record is the answer to give when it comes
back — not "no", but "here is what that would cost and here is where the useful
half of it already lives".

Nothing changes in the code. The evaluator stays exactly as capable as it was;
what is refused is the surface around it.

Reopening this would mean overturning the sentence at the top, not adding a
feature — and would want a measurement of demand, not an argument that it is
technically easy. It always was.
