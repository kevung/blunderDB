# blunderDB

blunderDB stores backgammon positions and the engine analyses attached to them, so a
player can search their own blunders across the matches and positions they have imported.

## Language

### Positions and their origin

**Position**:
A backgammon decision point: board, cube, dice, score and match flags. Identified by its
Zobrist hash, so the same position imported twice is one row, never two.
_Avoid_: board, node, entry

**Session rules (Jacoby, beaver)**:
Which optional rules the *session* was played under. They are columns of the Position and
are shown on the board, but they are not part of its identity (ADR-0028): only an XGID
ever carries them, so hashing them split one money position into two rows depending on
whether it came in pasted or from a file.
_Avoid_: match flags, position rules

**Deduplication**:
The rule that a Position's identity is its Zobrist hash. Any import that produces an
already-known Position lands on the existing row and enriches it (analyses merged,
comments appended) rather than creating a second one. The row therefore carries every
play ever recorded on it; a filter on the played move's error (`E>x`) scores the Position
by the largest of those errors — "did I ever blunder here?" — never by whichever play
happens to be read first (#167).

**Individually imported Position**:
A Position that entered the database on its own — written from the board, or read from a
position file — as opposed to arriving as part of a Match. Because of Deduplication, this
is a *sticky* property: a Position that was individually imported at least once keeps the
property forever, even if a Match containing it is imported afterwards. It is set by the
import that created or re-touched the row, and is never set or cleared by a user gesture.
_Avoid_: manual position, hand-added position, favourite, marked position

**Scratch board**:
A board the user composes rather than reads from the library — the Eval panel's and the
Search panel's. What stands on it has no identity: no edit made there can rewrite a stored
Position, and nothing known about the position it was copied from travels with it. Saving
a scratch board is individually importing a Position, never updating one; the board stays
a scratch board afterwards.
_Avoid_: edit board, draft position, custom position

**Match-sourced Position**:
A Position reachable from a Match through the `move` → `game` → `match` chain. Not the
complement of "individually imported": a Position can be both.

**Flagged Position**:
A Position the user marked as worth studying *in the tool the match came from* — today
only eXtreme Gammon, which records it per move. Like the individually-imported property it
is sticky, never part of the Zobrist hash, and never set or cleared by a gesture inside
blunderDB: it is a fact of the source file. So a Position flagged in one Match stays flagged
even when a later import brings the same Position in unflagged.
A flagged cube decision marks *both* Positions blunderDB derives from it — the double and
the take/pass — because the source records one decision and blunderDB splits it in two.
_Avoid_: bookmark, starred, favourite — a Flagged Position is durable and read-only; a
transient "come back to this" list is a Collection — the Pile.

**Game phase**:
Which part of the game a Position stands in: *opening*, *middlegame*, *race* or *bearoff*.
A **derived** label — computed from the board alone, stored in an indexed column, never
editable, and recomputed by `blunderdb repair` (ADR-0035). Three of its four boundaries
are gnubg's and are sourced; where the opening stops is a stated convention, not a
standard. It is not a *type of game* (holding, backgame, blitz…): those are a separate,
larger classification that no publication gives thresholds for.
_Avoid_: game type, position class, stage

**Comment origin**:
Who wrote a Comment: the user, or the importer of the file it came in with (`xg`,
`gnubg`, `bgf`), or `unknown` for a Comment written before the column existed. It is a
fact of how the row entered the database — but EDITING a Comment makes it the user's,
because after the edit the sentence is theirs. It is what lets the Orphan purge tell a
note the user typed from a per-move remark lifted out of a file.
_Avoid_: comment author, comment source

**Import batch**:
One import the user launched, with what it read and what it found. Matches point back at
their batch, which is what lets the end-of-import report speak about *this file* rather
than about the database. Deleting a batch never deletes its matches.

**Trash**:
What was deleted, kept for thirty days so it can be put back. A *snapshot*, not a
soft-delete flag: the delete really happens, and a JSON copy of what was deleted is
written first, so no search filter, statistic or retention rule has to know about it
(ADR-0036). Restoring a Position re-Saves it, so Deduplication decides where it
lands — it never creates a duplicate, and it never gives back the old id, because
the original row is gone. `blunderdb vacuum` empties it; an export never carries it,
and the Orphan purge never fills it (that purge is housekeeping, not a gesture).
_Avoid_: recycle bin, archive, soft delete

**Orphan purge**:
The sweep that runs when a Match is deleted: each Position the Match referenced is removed
unless something else still holds it. What "holds" a Position is a deliberate list — another
Match's move, Collection membership, an Anki card, a Comment the *user* wrote, or being
individually imported. An Analysis never holds a Position: every Match position has one, so
counting it would mean never purging anything. Neither does a Comment that is not the
user's — importers attach the source file's per-move notes as Comments, and until the
Comment origin existed no Comment held anything, which is why a note the user had written
was lost with the Match.

### Knowing what a position is worth

**Analysis**:
A *record* of what an engine concluded about a Position, stored against it and read back
later. It carries its provenance (`AnalysisEngine`, e.g. XG, GNUbg, BGBlitz, gammonNet) and
its `AnalysisDepth`. A Position has at most one Analysis row; several engines coexist inside
it, each entry tagged with the engine that produced it. An Analysis is *relit*, never
recomputed: a Position with no identity — one the user has just composed on the board — cannot
have one.
_Avoid_: evaluation, eval, engine output

**Evaluation**:
The *result of a computation* the embedded engine performs on the position currently on the
board, whatever it is, saved or not. It has no identity, is never loaded from the database,
and is valid only for as long as the position does not move. An Evaluation may later be
written down and so *become* an Analysis; until it is, it is not one.
_Avoid_: analysis, live analysis, instant analysis

The distinction is the reason two panels exist: one reads records, the other computes. They
render similar tables and share the components that draw them, but they are not two views of
one thing.

**Played action**:
What somebody actually did in a Position — the checker move, or the cube action. It is a
fact of a *Move*, not of the Position, and it is written at import whether or not any
analysis came with the file. It matters because the error a Performance Rating is a sum of
is the error of the action that was *played*: an Analysis states it when the file it came
from stated it, and when the Analysis is one blunderDB computed itself — a position does not
remember what anybody did with it — the Move table supplies it instead
(`engine.PlayedActionsFor`, #268). A Position met twice therefore carries the gap of its
first recorded occurrence only.
_Avoid_: user move, chosen move (a candidate is not a played action until a Move records it)

**Error**:
A decision whose cost — the gap between the Played action and the best one, in the
Position's Referential — reaches the library's *error threshold*. Every number that
counts "errors" (a match sheet, a player's row, the study queue an import proposes) uses
this one line; a decision below it is imprecise, not wrong.
_Avoid_: mistake, inaccuracy, doubtful (gnubg's word for its own lowest tier)

**Blunder**:
An Error whose cost reaches the library's *blunder threshold*. Every Blunder is an Error;
the statistics, the library counter in the status bar and the search it opens agree on the
set, and a Position played several ways is scored by the largest of its recorded costs
(see Deduplication). The word is never a fixed number: it means whatever this library says.
_Avoid_: bad move, hall of shame, "0.100" (a value, not a term)

**7-point MWC loss** (*Perte MWC (éq. 7 pts)*):
The match winning chances a player gave away over a match, every counted decision together
(checker, cube, take/pass, cubeful), rescaled to a 7-point match: L₇ = L·√(7/N). It counts no
decisions, so it complements the Performance Rating rather than replacing it; over several
matches the losses and the √N are summed before the ratio. It carries a 95 % interval and an
Elo reading against the engine (secondary); a money game has none (ADR-0075).
_Avoid_: M1 (the research note's label), intrinsic Elo (the secondary reading, not the value)

**Library setting**:
A preference that belongs to the library rather than to the machine — the two thresholds
above, the Performance Rating objective — so that the same file counts the same blunders
wherever it is opened, and so a daemon tenant carries its own. It is not carried by an
export: a threshold is the owner's reading habit, not a fact of the positions.
_Avoid_: preference, option, configuration (those follow the machine)

**Tag**:
A `#word` inside a Comment. Nothing declares one, no column holds one, and that is the
point: the vocabulary is the user's own prose. blunderDB *suggests* a list drawn from the
backgammon literature and counts what has actually been used, but a tag absent from the
suggestion is worth exactly as much as one on it. Matching is DELIMITED — the comment's
tags are extracted and compared whole, so `#prime` is not `#priming` — which is why a tag
search is its own filter and not a spelling of the free-text one. Several tags narrow
together (a Position carries many tags, so naming two means "both"), unlike the phase and
provenance filters, where a Position has one value and naming two can only mean "either".
_Avoid_: label, category (both suggest a closed, declared set — a tag is neither)

**Neighbouring Position** (of a target):
A stored Position that poses *the same problem* as the target with a nearby checker
structure — never merely the same drawing. "Nearby" is a transport distance in checker-pips,
seen from the side on roll, and it is measured only inside the target's *equivalence class*:
the same kind of decision (checker or cube); for a cube decision, the same regime (money or
match); and, when the target belongs to a Match, a different Match — the plies before and
after a decision are its closest structures and never its neighbours. Dice, score and cube
value are outside both the distance and the class: the ordinary search filters narrow on them.
The target may be a drawn board as well as a stored Position. Neighbours are *ranked*, not
filtered, and a ranking with nothing under the asked distance is empty, not padded.
_Avoid_: similar position (suggests the drawing alone), duplicate (a duplicate is the same
Position, identified by its hash — distance zero within another Match is a neighbour, not a
duplicate)

**Network**:
The weights, and only the weights — `strehl-prob5-512-512-256-256`. A network changes name
only when its weights change: neither the search wrapped around it, nor a quantisation, nor a
port to another language makes it a new one.

**Configuration**:
A network *plus* the search, the endgame tables and the match-equity table around it — the
whole of what produces a number. `gammonNet 2-ply` names a Configuration, not a Network. Two
Configurations sharing a Network are still two Configurations.

**Canonical parameters**:
The Configuration blunderDB writes down: 2-ply, pruning `k=12`, and the library's Match
equity table (Kazaross-XG2 unless the library chose another). What the user adjusts for
comfort while reading the board is a different setting, and it never reaches the database.

**Match equity table** (MET):
The table of match-winning chances at each score a match-score number is computed with. A
property of the library, not of the application: the built-in Kazaross-XG2 unless the
library imported a gnubg `.xml` and made it current. Every analysis records the table it was
computed with; one computed with another table than the current one is shown as *different
MET* and left out of comparisons. Money numbers never depend on it (ADR-0068).
_Avoid_: MET setting (there is no global switch).

**Referential**:
The scale a number about a Position is expressed in, and which of two questions it answers.
*Money* answers "how many points is this worth", on the points scale, and gammons are worth
exactly two points to anybody. *Match* answers "how much of the match is this worth", on the
normalised scale `2 × MWC − 1`, where a gammon is worth whatever the score makes it worth —
almost everything at 2-away/4-away, nothing at all to a leader who is 1-away.

A Referential is a **property of the Position, never a preference**: the Away score selects it,
and every number blunderDB shows or stores about that Position is in it. The two scales are
not convertible without the distribution that produced them, so a table mixing them is not a
table with two units — it is a table that cannot be read.
_Avoid_: unit, equity mode, money mode

**Away score**:
How many points a player still needs to win the match, from that player's side. blunderDB
carries the Crawford rule *inside* this number rather than beside it, and the two smallest
values are sentinels rather than counts:
- `-1` — money play. There is no match; the Referential is money.
- `0` — the player needs one point, and the Crawford game is behind us.
- `1` — the player needs one point, and this *is* the Crawford game.
- `n ≥ 2` — the player needs n points.

So `0` and `1` describe the same distance to victory and different rules. Any code turning an
Away score into something an engine can use must decode both sentinels; reading `0` as a
distance is reading "has already won".
_Avoid_: score, points away, match score

**Regime**:
Which kind of answer the panel is giving about a race, always stated on screen, never inferred
by the reader:
- *exact* — read from a two-sided bearoff database. A lookup. The answer.
- *evaluated* — computed by the engine, which plays the trajectory out. Not a lookup, not a
  guess either.
- *estimated* — a snapshot summary of a trajectory. Legitimate for a win probability
  (convolution plus a calibrated correction), and **never** used for a cube verdict.

A Regime qualifies a *fact* about a race — a win probability — and, where it is entitled to
one, the *decision* built on it. Only *exact* is barred from answering outside the money
Referential: its equities are money whatever the Away score, so at a match score it keeps the
win probability and yields the verdict to *evaluated*.

**Bearoff database**:
A table of race answers over a *domain* — two-sided (one entry per pair of home boards, the
cube verdict's source) or one-sided (one entry per home board, the EPC's source). Where the
table came from — shipped, downloaded, generated on this machine, or supplied by the user — is
its *origin*, and the origin changes nothing about the Regime: a lookup in a generated table is
still *exact*. What can differ is whether the table is **verified**.
_Avoid_: bearoff file, .bd (the file is one carrier of a database, not the concept)

**Domain** (of a bearoff database):
The largest position a database answers — for a two-sided table, checkers per side within the
home board; for a one-sided table, the farthest point a checker may stand on. A position inside
the domain gets the table's answer; outside, the other Regimes take over. Widening the domain
never changes an answer inside it.

**Verified** (bearoff database):
Identical, byte for byte, to what gnubg's own generator produces for the same domain, attested
by a fingerprint blunderDB knows for that domain. A generated database whose fingerprint is
unknown is *unverified*: its answers are believed exact, and the panel says so. Verification is
a property of the table, never of the answer — *exact* stays *exact* either way.
_Avoid_: validated, certified, authentic

**Position fact**:
A quantity that belongs to the board itself and to no choice a player might make: pip count,
EPC, wastage, mean rolls and their dispersion, the pre-roll probability vector (win, gammon and
backgammon), and the cubeless equity in the position's Referential. Facts do not depend on the
dice; they are true before the roll. They come in two kinds, and the difference is what decides
how they are read (ADR-0018):

- **Per-side facts** — the race block: EPC, pip count, wastage, mean rolls, dispersion. Each
  belongs to one player's home board, so two players' can be set side by side and subtracted.
  They are always read in `bottom` / `top` / `Δ` rows.
- **The pre-roll vector** — win, gammon and backgammon chances and the cubeless equity. The
  engine computes it at the trait (`invertProbs`, `CubelessValue`), and it reads equally well
  per side; it therefore takes the axis of whatever it is compared against. See Baseline.

_Avoid_: stats, position evaluation, summary

**Baseline**:
The pre-roll vector rendered in the axis of the Decision it heads, so each option can be read
against it. With dice on the board the Decision is a list of checker plays at the trait, and the
Baseline is a band in that same list's columns, pinned above the rows. It is never part of the
ranking: the gap between the Baseline and any play holds the value of the roll — the luck of the
position (ADR-0010) — not the merit of the play, which is why it carries no error figure.
_Avoid_: reference row, row zero, first line

**Decision type (pions / videau)**:
Whether a Decision is about checkers or about the cube. The French interface says *pions* and
*videau* wherever it names the type (filters, statistics, training, reports), "PR pions" and
"PR videau" for the matching Performance Rating, and keeps *coup* for one play. An error is
counted in mp (millipoints of equity, mMWC at a match score), and any abbreviation on screen
carries a tooltip that spells it out.

**Decision**:
The answer to the question the board is asking — the ranked checker plays when dice are on the
board, the cube actions and their verdict when there are none. A Decision is read **per
option**, never per player. The board asks exactly one question at a time, so a Position has
exactly one Decision to show; a race with dice on it is asking about checkers, and its pre-roll
cube verdict answers a question nobody asked.

A cube Decision always has the same three named options — no double, double-take, double-pass — in
that order, whatever Regime produced their equities. Their order carries no information, unlike a
ranked play list where the order *is* the answer (ADR-0020).
_Avoid_: analysis, verdict (the verdict is one part of a cube Decision, not the whole)

**Verdict**:
Which of a cube Decision's options is the right one: *no double*, *double-take*, *double-pass* or
*too good*. It is a value, not a sentence — the panel's own evaluation names it as one of those
four, while an imported analysis carries whatever string its engine wrote and is reported as-is.

Two things are named in the Verdict's place rather than left blank, because a blank means "still
computing" and nothing else:
- **no decision** — the Regime is not entitled to one (*estimated* never yields a cube verdict).
- **refused** — the engine declined the position, typically a match score beyond the MET's horizon.

A fifth case is not a Verdict at all: where the cube cannot be turned — the opponent owns it, or
the game is the Crawford game — no option is available, so no option carries an error. The
equities still inform; nothing advises.
_Avoid_: best action, recommendation, conclusion

The pair is the panel's whole layout rule: the per-side facts in a table whose rows are the
players, the one Decision the board is asking for, and — when that Decision is a list at the
trait — the Baseline at its head. See ADR-0017 for the partition, ADR-0018 for the axis of a
fact, ADR-0020 for the shape of a cube Decision.

### Sets of positions the user curates

**Collection**:
A named, ordered set of Positions the user assembles by hand. Membership is a user
gesture, unlike the individually-imported property.

**Pile** (interface: *pile*):
The Collection the quick "come back to this" gesture aims at: one key puts the Position on
the board onto the Pile, the same key takes it off. An ordinary Collection in every other
respect — renamed, reordered, exported, emptied like any other — created on first use and
again if it was deleted. The gesture works wherever a Position is shown, a Duel included:
there the Position is not yet in the library, so the gesture brings it in on its own, at
once, and it stays on the Pile whatever becomes of the Duel.
_Avoid_: bookmark, stack, to-study list, study queue (computed from the error threshold),
flag (a Flagged Position is a fact of the source file)

**Anki deck**:
A set of Positions — or the 36 Score cards — turned into spaced-repetition cards. A deck of
scores is created on request and filled by the application; the user never enters a score.

**Review card**:
One Position, or one Score card, of an Anki deck, presented as a question. For a Position the
question is the board and the Answer its stored Analysis; for a score, the question is the
score and the Answer its Score card. A card asks one question and receives one grade —
there is no notion of a partly answered card.

**Board answer** (of an Anki deck):
An option of the deck: a checker Review card is answered by playing the move on the board; the
quiz judges it and proposes a grade, which the user confirms or changes. Off by default — the
user grades their own recall (ADR-0040). Never offered for cube cards or decks of scores.

**Session limit** (of an Anki deck):
How many Review cards one sitting serves before it ends. A property of the deck, like its
target retention — never a daily quota: a deck is a finite corpus, so the daily volume is
already bounded by its size, and a cap on a day either never bites or manufactures a
backlog. Unlimited by default; zero means "no cards this session", which is not the same
thing as no limit.

**Observed retention** (of an Anki deck):
The pass rate actually measured over a deck's review log, read against its target
retention. An observation, never a control: the target is the user's choice on the
work/knowledge trade-off, and steering it to chase the observation is the one mechanism
FSRS's authors reject.

**Answer** (of a Review card):
The stored Analysis of the card's Position — never a live evaluation. A card whose Position
carries no Analysis has no Answer, which is a state the panel names rather than hides: an
absent answer is not a hidden one.

**Comment**:
Free text attached to a Position. The model allows several per Position (match import adds
one row per note found in the source file), but the GUI treats a Position as having a
single comment: it loads and edits whichever row comes back first. Known debt — a Position
that arrived with two comments shows only one of them, arbitrarily.

A Comment carries no provenance: text the user typed and text an importer lifted from a
source file are indistinguishable. So "the Position is commented" means only *some* non-empty
text is attached — never "the user annotated it". Empty text is not a Comment: a row whose
text is `''` counts as absent everywhere (search, listing, export).

Match and Tournament each carry their own comment field. Those are annotations of the Match
or the Tournament, not of its Positions: a commented Match does not make its 300 Positions
commented, and no Position-level rule in this glossary reads them.

**Lesson**:
An ordered sequence of Steps a coach writes once for a student and hands over in an exported
database. Reading it records nothing (ADR-0007); its only follow-up is the Steps the student
marks done (ADR-0069). Deleting it is final and leaves the Collections and Positions its
Steps show. "Parcours" may name it in the interface; it is not a second object.
_Avoid_: study queue (computed, no text), course, path.

**Step**:
One stop of a Lesson: a title, a text and, optionally, a Collection and a Position. A Step may
be text alone. A Step holds its Position (it is not purged with its match); when the
Collection or Position it shows is deleted, the Step and its text stay.

**Step done**:
The student's explicit gesture on a Step of a Lesson, dated, in the student's own library —
the only thing that writes progress. Opening, reading, importing or exporting a Lesson never
marks a Step, and no export carries the marks (ADR-0069).
_Avoid_: seen, visited, read (none of them is recorded).

### Training

**Training** (interface: *Entraînement*):
The tab where the user drills what is *calculated* under the clock — pip counts, EPCs,
score tables, evaluations, decisions — and reads the Journal of it. Its counterpart is the
Anki deck, which keeps what is *retained* over days; the two never schedule the same thing.
_Avoid_: quiz (the old name of one exercise), drill, practice mode

**Exercise**:
One kind of question Training can ask, declared by four properties: its Seed source, its
Numbers, its surface (the board, or nothing) and its Answer mode. Five exist: Pips, Bearoff,
Scores, Evaluation, Decision. An exercise is named after what one looks at (the bear-off),
not after the number it asks (the EPC).
_Avoid_: drill, module, quiz type

**Question**:
One position or one score put to the user by an Exercise, carrying one or more Numbers and
timed as a whole: one reading of the board, one time.

**Number** (of a Question):
One value the Question asks for — a pip count, an EPC, a table cell, a cube action —
with a truth and a tolerance. Faults are counted per Number, never per Question.
_Avoid_: item (code only), field, answer

**Answer mode** (of an Exercise):
How a Number is answered: *entered* (typed; the application grades within the tolerance
and keeps the signed deviation), *declared* (revealed; right by default, the user ticks the
Numbers they got wrong), or *chosen* (one of a few options, graded exactly). A property of
the Exercise — what is estimated is entered, what is counted or recalled is declared —
never a per-session switch.

**Reveal** (interface: *Révéler*):
The gesture that stops the clock and shows the truth of a declared Question; ticking
faults afterwards is not timed. Unrelated to Défi, which reveals a live Evaluation zone by
zone with no clock and no grade.

**Fault**:
A Number the user declared wrong, or an entered Number outside its tolerance. The fault
rate of an Exercise is faults over Numbers asked.
_Avoid_: error (reserved for the analysed blunder), miss

**Out of time**:
A Question whose per-question limit elapsed before an answer: it reveals itself, every
Number counts as a Fault, and no deviation enters the mean.

**Seed**:
What a Question is played out from — a canonical shape from the Exercise's pool, the board
as it stood at launch, or a Position of the library — before the engine plays a few plies
to reach the Question. An internal term: the interface names only the three sources.
_Avoid_: template, starting position

**Seed source** (interface: *source*):
Where the Seed comes from — *pool* (*vivier*), *board* (*plateau*), or *library* (*base*) —
chosen at launch and remembered. A Seed outside the Exercise's domain is refused by name,
never adapted.

**Training session**:
Everything between « Démarrer » and « Terminer » in Training; « Quitter » discards it. It
has no fixed length and is recorded as one row of the Journal with its Numbers. Not to be
confused with the Session rules (Jacoby, beaver) nor with the Session limit of an Anki deck.
_Avoid_: run, round, set

**Journal**:
The record of Training sessions and their Numbers, kept in the library's own tables so it
travels with the file; read at rest in the Training tab as a per-Exercise summary
(*bilan*) and a per-Number detail. Anki reviews are not in it; Stats › Training reads its Decision sessions and the Anki review
log side by side with the real PR, without copying either.
_Avoid_: history, statistics, log

**Score card** (interface: *fiche de score*):
The take points (cube 2 and 4, long race and last roll) and gammon values (cube 1, 2, 4)
of one unordered match score, both faces side by side, showing only the cells the reference
tables define. One component, two hosts: a declared Question of the Scores Exercise, and a
Review card of a deck of scores. The gammon value at a centred cube is *gv1*.
_Avoid_: take point table (that is the reference modal), score sheet

**Défi** (code: *challenge*):
The Eval panel's own mode that re-masks its zones on every edit and lets the user reveal
them one by one — a convenience for the position in front of them, outside any Exercise,
with no clock and no grade.
_Avoid_: challenge mode (interface), training mode

### Recording a match

**Transcription**:
The move-by-move record of a match played *elsewhere* — at a club, over a real board, from
a video or a score sheet — while the user writes or corrects it. A **draft**, distinct
from the Match it produces: it lives in the library as one document, survives a crash of
the application, and stays the source of truth for as long as it is open — saving it
materialises a Match, saving it again *replaces* that Match on the same id. Until saved,
a Transcription is in no statistic, no search and no export of positions. It is not a
Duel (ADR-0044: nobody decides, the rules check and never enforce), not an
Evaluation (though it shows one at every Action), not a Collection.
_Avoid_: recording, live match, play mode (a Duel), transcript (see below)

**Transcript**:
The *rendering* of a Match or of a Transcription as two columns — player 1 left, player 2
right, one row per turn, the cube action and the end of the game in the column of whoever
acted — which is the layout of the `.mat` file and of every score sheet. The Match panel's
move list and the `.mat` text are two Transcripts of one Match.
_Avoid_: transcription (the draft), move list

**Action** (of a Transcription or a Duel):
One player's act at one moment of the match: a roll and the checker play it was used for
(or the dance it forced, or the mark that says the play was never written down), a double
or redouble, a take, a pass, a resignation. A double and
its answer are two Actions, each with its own Position and Decision. An Action *owns its
side*: proposed by the trait at entry, the side belongs to the Action once recorded, so
inserting or deleting an Action never changes who played the ones after it. When the
Transcription is saved, a checker or cube Action becomes a Move with its Played action and
its Position; a resignation becomes a fact of the Game (winner, points) and no Move.
_Avoid_: move (the stored row), turn (both players' Actions on one Transcript row),
element, entry, ply (a search depth)

**Cursor**:
The Action a Transcription is currently about: the board shows the position it is
played from, the panel lists its candidates, and the next recorded Action goes there — in
place, correcting it, or before or after it on an explicit gesture. The Cursor points at
an Action of the draft, never at a Position of the library.

**Replay**:
Recomputing every derived fact of a Transcription from one Action onwards by playing the
Actions again in order: the board each Action leaves, the score, the Crawford game, where
each game ends and with how many points, and the Inconsistencies. Every correction
triggers one, and nothing else changes a derived fact. What a Replay computes is never
typed and never stored in the saved Match; what it cannot compute — an illegal move's
resulting board — is kept on the Action.
_Avoid_: repropagation, recompute, validation

**Inconsistency** (of an Action):
A derived fact a Replay attaches to an Action, shown to the user and refused by nothing:
an *illegal move* (the board it left is reachable by no legal play from the board before
it), a *double turn* (two consecutive Actions of the same side), an *impossible cube
action* (a double by a player who does not hold the cube, an answer with no offer), an
Action *past the end* of the match after its length was shortened, an *unrecorded move*
(the roll is known and the play is not — gnubg writes "???" in the cell; the players did
nothing wrong, the RECORD is incomplete, and every board after it is unchecked). An
Inconsistency is
kept and marked — never deleted by the software, never written into the saved Match as
data — and the Cursor jumps to the first one after a Replay. An illegal move exported to
a `.mat` is exported as played, with a warning that gnubg and XG will flag it and diverge
from there.
_Avoid_: error (the analysed blunder), fault (Training), invalid move (what gnubg
refuses — a Transcription refuses nothing)

**Timecode** (of an Action; interface: *repère*):
The instant, in the media time of the video attached to a Transcription, at which an Action
happened: the roll (the dice land) and the action (the play is finished, the double offered,
the answer given). Media time, not wall-clock time: pausing, rewinding or speeding the video
up does not move it. Posted on the Action while typing, carried by the saved Move, and the
source of every Decision time a Transcription has — the Replay derives the durations from
consecutive Timecodes, never the other way round (ADR-0082). Unknown, never zero, when the
Transcription has no video or the Action was never stamped.
_Avoid_: timestamp (a wall-clock date), tick, duration (what is derived from it), frame

### Playing a match

**Duel** (interface: *match*, under the gesture *Jouer*):
A match being played *inside* blunderDB, under its arbitration: the dice are rolled here,
the rules are enforced, the score advances, and each of the two Sides decides for itself.
A **draft**, like a Transcription: it survives a crash of the application and is resumed
where it stopped; finishing it materialises an ordinary Match, and nothing about that Match
is special once saved. It is the opposite of a Transcription on two of the three counts of
ADR-0044 — somebody decides, and the rules refuse — and the same on the third. A finished
Duel is never reopened as a Transcription: its Actions were arbitrated, there is nothing
to correct (ADR-0072). A Duel is *open* while it is played, its clocks running, or in suspense; that is in its
row, and several Duels can be open at once — the desktop keeps one at the board and
suspends the one it leaves. A Duel is left three ways: *paused* (in suspense, its clocks stopped,
resumed where it stopped), *cancelled* (nothing of it is written) or *forfeited*. A match
in points is written whole or not at all — the Arbiter refuses to keep one cut short, and
losing on time is a forfeit by the player out of time; only a money session, which has no
end of its own, is kept as it stands when stopped (ADR-0072 rule 10).
_Avoid_: played match (every Match was played), play (a checker play), game, partie (one
Game of a match), live match, play mode, sparring (true of one configuration only)

**Forfeit** (interface: *abandonner le match*):
The Action by which a Side gives the whole match up, at any moment: the game in progress
ends won by the other Side, for the points that bring it to the length — at money play, a
single at the cube's value — and the Duel ends as a match won. Not a resignation, which
gives one game up, and not a stop, which never gives anything (ADR-0074).
_Avoid_: resign the match, concede, stop and keep (only a money session is kept as it stands)

**Side** (of a Duel):
One of the two players of a Duel. A Side is *external* — its decisions arrive through an
interface (the board, the CLI, the API), from a human or from a program blunderDB knows
nothing about — or *delegated* to a Bot. Human against engine is one of each; two external
Sides is two people playing through a client such as gammonGo; two delegated Sides is the
engine playing itself. The Arbiter treats the three alike. An external Side may *declare* the Bot
playing behind it — one a client runs itself: the Match's origin records it as declared,
never attested, apart from the Bots the Arbiter played.
_Avoid_: seat (the per-seat columns of a Match), player (a name in a Match), opponent

**Bot**:
What a Side of a Duel is delegated to: a Configuration of the engine plus a *playing
policy* — everything a player does that an evaluator does not: answering a double,
offering or accepting a resignation, playing weaker than it can, taking its time. A
Bot is *delegated* when the Arbiter plays it, *declared* when an external Side says it
plays behind it; only the first is attested.
_Avoid_: engine (the evaluator), AI, computer, gammonNet (the Network and its search)

**Arbiter**:
blunderDB's role in a Duel: it rolls the dice, enforces the rules, keeps the score and the
clock, and *refuses* an Action the rules do not allow — where a Replay keeps it and marks
it. Who may sit on an external Side, pairing two people, knowing they are present: none of
that is the Arbiter's business; it belongs to the client.
_Avoid_: referee, server, game master

**Dice seed** (of a Duel):
The secret every roll of a Duel is computed from, drawn when the Duel is created. Its
fingerprint is published before the first roll and the seed itself is revealed with the
finished Match, so that anyone can recompute the rolls and see they were never fitted to
the position. Resuming a Duel or taking a move back never changes a roll. With a *combined
seed*, each external Side also contributes a short text before the first roll, and the
rolls come from the seed combined with the contributions: neither the Arbiter, which
sealed first, nor a Side, which contributes blind, chose them. Unrelated to the
Seed of a Training Question, and to a Match's dice fingerprint, which identifies the rolls
of a match to recognise the same match twice.
_Avoid_: seed (Training), dice hash (the Match's fingerprint)

**Start** (of a Duel; interface: *départ*):
Where a Duel begins: a Position — its board, its cube, who has the trait, its score, and
a roll if it carries one. By default the opening position at the start of the match; it
may be the Position on the board, or the opening position at a chosen score. Only the
first game begins at the Start: the games after it begin at the opening position, at the
score reached. A Start the rules do not allow is refused by name, never adjusted. A Match
played from a Start other than the default carries it as part of its origin, and cannot
be written as a `.mat`, which has no way to begin a game elsewhere.
_Avoid_: seed (a Training Question's), setup, starting position

**Cadence** (of a Duel):
The clock a Duel is played under: a *reserve* per Side for the whole match, and a *delay*
at each turn that runs first and costs nothing — the reserve only goes down once the delay
is spent. The clock that runs is that of the Side a decision is awaited from, and it is the
Arbiter that keeps it. It runs while the Duel is open, whichever process holds it and
even if that process stops, until the Duel is suspended. A Duel may have no Cadence. A Side whose reserve is spent is *over
time*: a fact the Arbiter reports, whose consequence is a setting of the Duel — the Duel
goes on and the overrun is noted, or the match is lost.
_Avoid_: time control, out of time (a Training Question), increment (a Cadence has none)

**Decision time** (interface: *durée*):
How long a Side took over one Decision of a Duel, counted by the Arbiter from the moment
the Side has the trait until it acts. A turn holds up to two: from the trait to the roll —
the cube Decision, taken even by rolling — and from the roll to the play. A double, a take,
a pass, a resignation each have their own. It does not depend on the Cadence: the delay is
part of it, and a Duel with no Cadence measures it all the same. It becomes a property of
the play in the saved Match. A Transcription with a video attached derives it from its
Timecodes (ADR-0082); any other Match that was not played here has none — unknown, never
zero.
_Avoid_: thinking time, time per move (a turn is two Decisions), clock time (the Cadence)

### Directing a tournament

**Tournament**:
A named event (name, date, location, comment) that Matches belong to. Until now it was
only ever an *afterthought*: a label put on Matches that came in from files. It can also be
created *before* its Matches exist — see Direction. Deleting a Tournament unlinks its
Matches, never deletes them.
_Avoid_: event (the `.mat` header field), competition, organisation (the club or
federation that runs it)

**Direction** (of a Tournament):
Everything the tournament director decided while running the Tournament: the format, the
entries, every match launched, every result, correction, withdrawal, draw and phase change,
in the order they happened. It is the *source of truth* of a directed Tournament — the
standings, the brackets, the pairings and the next thing to do are all derived from it and
never stored on their own. A Direction is only ever *extended*, never edited: a wrong result
is corrected by a later correction, a match launched by mistake by a later cancellation. A
Tournament made from imported files has no Direction; a Tournament run with blunderDB has
one; a BMAB-style Tournament, run here and whose matches are then transcribed or imported,
has both. The engine that derives state from a Direction is Nicomaque (ADR-0047).
_Avoid_: journal (the Training Journal), log, event log, replay (a Transcription's Replay) —
these are how Nicomaque names its own internals and must not leak into the interface

**Participant** (of a Direction):
An entry in one Tournament's Direction: a name, a club, an entry rating. It belongs to that
Direction alone. It is not a Player and not a person: the only link to a Player is a name
the director chose to spell the same, so that the Matches that fill this Participant's slots
carry that Player's literal name. Choosing an existing Player at entry time fixes the name
and pre-fills the rating from that Player's PR; nothing is inferred afterwards.
In a doubles event a Participant is a pair: two members, each with a name, a club and a
rating; its label "A / B" is derived and its entry rating is the mean of the two. The Matches
of a pair carry that label as the Player on its side (ADR-0056).
_Avoid_: player (the literal name in a Match), entrant, competitor, member

**Slot** (interface: *emplacement*):
One match of a Direction — two Participants, a length, a table, a phase, a result — seen as
the place a Match of the library can fill. A Slot exists as soon as the director launches
the match, and stays empty unless a Transcription is started from it or an existing Match
is attached to it by an explicit gesture. The result of a Slot is what the director said
(pre-filled from the Match when one is attached, never derived from it); a Match whose final
score disagrees with it is shown as a warning, not reconciled. Deleting a Direction empties
every Slot's link; the Matches keep their Tournament.
_Avoid_: fixture, pairing (that is the proposal, before the match is launched), game

**Directory** (interface: *annuaire*):
The Participants of every Direction in the database, seen as one list deduplicated by name,
each with the club and rating of their latest entry. A *view*, never a table: it is
recomputed from the Directions, exported and imported as CSV, and copied from one
Tournament into the entries of the next. It is not an identity: two spellings are two rows.
A doubles pair appears as its two members, never as a row "A / B".
_Avoid_: player list, address book, roster (a roster is one Tournament's Participants)

**Rencontre** (interface: *événement*):
Several directed Tournaments played on the same tables, on the same dates, by the same
director — a weekend festival with its main event, its speed and its doubles. It owns the
tables: their number, their properties (Table, Salle), the output folder and the one wall page
that shows every table whatever the event and whatever its Salle. Each event stays a Tournament with its own Direction, Matches and standings; a hall
gesture (a table out of service, a break) is written into every member Direction, so each
one still replays alone. Two Participants spelled the same in two events of a Rencontre are
one person for availability: the Rencontre knows they cannot sit at two tables. A Tournament
belongs to at most one Rencontre and may be attached or detached at any time (ADR-0056).
The interface says *événement* (en: *Event*): an *événement* gathers *épreuves*, the two words
never stand for one another. The technical name stays `rencontre` in code, schema, CLI and
API, because `event` is already taken there (the SSE stream, a Direction's events).
_Avoid_: festival (an event's name, not an object), meeting, réunion, event group, épreuve
(one Tournament of it)

**Table**:
A numbered place where one match is played. The number is its identity — "table 23" in the
CLI, the API and the players' mouths — unique within a Rencontre (numbering runs on from one
Salle to the next). A table may carry properties: a name shown beside the number ("Stream"),
a Salle, and a reservation — *reserved* (never proposed, the director places a match there by
hand) or *assigned* to persons by name (their matches go there when it is free; otherwise it
behaves as reserved). The properties belong to the Rencontre, or to the directed Tournament
when it plays alone; they change only what is proposed, never what a Direction replays
(ADR-0058).
_Avoid_: board (the backgammon board), seat

**Salle** (interface: *salle*):
A named group of the Tables of a Rencontre — "salle A = tables 1–20". It is a label on each
Table, not an object of its own: a Rencontre whose tables carry no Salle has one implicit Salle
covering them all. An event of the Rencontre may be restricted to some Salles; it is then
proposed, moved and swapped only onto their tables. The grid of every table of the Rencontre,
grouped by Salle, is the view *Toutes les tables* — never "Salle", which names a part of it,
nor "Événement", which names the panel that manages the Rencontre (ADR-0058).
_Avoid_: hall (for the whole Rencontre), room, venue, area

### Players

**Player**:
A name exactly as it appears in an imported Match (`player1_name`/`player2_name`).
blunderDB has no notion of a person behind the name: the same human signing under
two spellings is two Players, and every per-player statistic, list or table is keyed
by this literal name. Merging spellings is a destructive user gesture (merge players),
never an inference. Distinct from Tenant — the Tenant owns the data; a Player merely
appears in it.
_Avoid_: user, account, identity

### Who owns the data

**Tenant**:
The owner of a set of Positions, Matches, Collections and decks. On the desktop
there is exactly one, implicit Tenant: the person whose database file it is. In
server mode each caller is a distinct Tenant, and nothing one Tenant stores is
ever visible to another — except to a read whose Read tenants include it.
Deduplication, the Orphan purge, and every other rule
in this glossary apply *within* one Tenant — the same board position stored by
two Tenants is two rows, not one.
_Avoid_: user, account, customer

**Scope**:
The storage layer's spelling of Tenant: every persistence call carries a scope,
and the empty scope denotes the desktop's single implicit Tenant. In server
mode a scope is the Tenant's positive decimal integer (`1`, `42`), never a
name — the proxy maps names to integers, the daemon refuses anything else
(ADR-0005). "Scope" and "Tenant" name the same concept;
prefer Tenant in prose and design discussion.

**Read tenants**:
The Tenants one server read spans: the writing Tenant (`X-Tenant-ID`) first, then
those the authenticating proxy lists in `X-Read-Tenants`, at most 64 in all. Only
the `across.*` reads use them, every answer names the Tenant it came from, and a
write still lands in the writing Tenant alone. The daemon honours the header
only when started with `--read-tenants`, and refuses it otherwise. The relation behind the list (a
coach and their students, a club) lives with the caller; blunderDB authorises
nothing (ADR-0005, ADR-0063). On SQLite, and on the desktop, the Read tenants are
the single Tenant.
_Avoid_: shared tenant, linked tenants, permissions

**Coach comment**:
A Comment a coach writes on a student's board, stored in the coach's own Tenant on
the coach's copy of that board (same Zobrist hash, the coach's id) and read back
next to the student's Position by the hash, never by the id. Nothing is written in
the student's Tenant. It is an ordinary Comment of origin `user`, not a second kind
(ADR-0065).
_Avoid_: annotation, shared comment

**Shared library**:
The Collections of a Tenant read in place by the Tenants whose Read tenants list
it. Nothing is copied, unlike sharing a Collection by file (export, then import),
which hands the receiver a copy it then owns.
_Avoid_: shared collection, public collection

**Club ranking**:
One ranking over the player tables of the Read tenants, best PR first, each row
naming its Tenant; a player name is never merged across Tenants. It measures play
(PR, errors), unlike the season ranking of directed tournaments, which sums places
(ADR-0062, ADR-0065).
_Avoid_: leaderboard, club Elo

### Handing a database to someone else

**Watermark**:
The producer's signed statement of where a database comes from — what it is, who made it,
and whatever they chose to attach to it (terms of use, a contact address). It is applied at
export and nowhere else, and it is the mirror image of the database's own metadata: `user`
and `description` belong to the Holder and are theirs to edit, a Watermark belongs to the
producer and is read-only everywhere else.
It is **tamper-evident and unforgeable, never unremovable** — a Holder with SQL tools can
delete it, and the design says so out loud rather than pretending otherwise.
_Avoid_: licence, DRM, protection — a Watermark forbids nothing and identifies nobody; it
attributes a file to its source.

**Issuer identity**:
The durable identity a Watermark is signed with. It belongs to a person, not to a database:
every file the same producer marks carries the same public fingerprint, which is what lets a
recipient check that a file really came from them. It comes into existence without being
asked for, on the first watermarked export, and moves from one machine to another as a
single file.

**Protected file**:
An exported database wrapped in an encrypted container, opened once with a password and
thereafter an ordinary database. It protects the file **in transit** — the stray copy, the
attachment forwarded by mistake — not the database, since whoever the password was given to
can open it. Its header is cleartext, so a Watermark stays readable without the password.

**What is deliberately absent**:
Nothing records who holds a database, who opened it, or where its contents came from. The
recipient's side writes nothing at all. This is a decision, not an omission: see ADR-0007.

### Who the interface is for

The three people a panel is designed for (ADR-0085). A panel names the one it serves first;
its header strip puts that person's next gesture at the right end.

**Reviewer**:
The competitor who reviews: imports their tournament matches, looks for their errors,
comments on them and files them. Typical task: "go over Saturday's match, keep the five worst
errors". Matches, Tournaments, Analysis, Comments, Collections, Search, Stats.

**Trainee**:
The player who trains: short sessions, hands on the keyboard, eyes on the board. Typical
task: "today's cards, then a duel against gammonNet". Anki, Training, Duel, Eval.

**Coach**:
The coach or club organiser: transcribes filmed matches, runs tournaments, prepares series of
positions for students. Typical task: "transcribe the final, attach it to the tournament,
export a collection". Transcription, Tournaments, Collections, Metadata, Search.
_Avoid_: user — say which of the three.

## The host environment

**Watched folder**:
A directory blunderDB LOOKS at while it runs, importing each match file that *appears* in
it (#258). Two words carry the whole notion. *Appears*: what the folder already held when
the watch started is recorded as known and never imported — importing the folder as it
stands is a separate, explicit gesture. *Looks*: the folder is polled, not subscribed to,
because the shares and synchronised directories these folders live on report no file-system
events anyway, and a fallback that has to exist regardless is better as the only path than
as the second one. It is a setting the user turns on and points somewhere; blunderDB never
guesses where somebody's matches live.
_Avoid_: auto-import, sync folder (nothing is synchronised, and nothing is written back)

**Assistant**:
The in-app client of blunderDB's own MCP tools (ADR-0064): a sentence goes to a language
model the user chose, the model calls the tools, and a search it builds opens as a new view
named after the sentence. It reads freely; every write it prepares waits for the user's
confirmation. What the tools return is the database's; what the model writes is *free text*,
marked as such and never taken for a measurement. Off by default, and no model ships with
blunderDB.
_Avoid_: AI, chatbot (it answers with the tools or not at all), agent (it acts only through the
tools, and writes only when confirmed)

**Host capability**:
A facility blunderDB consumes from the machine, OS or desktop it runs on but does **not**
own — its presence and its shape are not guaranteed and vary from system to system.
Examples: an image clipboard tool, an installed font, the keyboard layout, the locale, a
writable config/data directory, the WebView renderer. Each host capability has a *state*
(present / absent / degraded) and a *fallback strategy*.
_Avoid_: dependency (reserved for Go/npm packages), platform (too coarse — clipboard tools
and keyboard layouts vary *within* one platform).

**Essential capability**:
A Host capability without which blunderDB cannot fulfil its core mission of storing and
searching positions. There are exactly two: a **writable config/data directory** and the
**WebView renderer**. When one is absent, blunderDB fails loud and early with an actionable
message rather than entering a half-broken state.

**Optional capability**:
A Host capability whose absence must never block the core product — an image clipboard,
a CJK font, single-instance locking, an expected keyboard layout, a specific locale. When
absent or in a different shape, blunderDB degrades gracefully: it detects, falls back on an
embedded or native substitute where possible, and surfaces a non-blocking notice rather than
an interrupting error.

**Fallback strategy**:
The ordered ladder (its rungs) blunderDB walks when an Optional capability is absent or
degraded: prefer a substitute it *ships* (an embedded font) or a *native* mechanism (the
WebView's own clipboard) over requiring the user to install an external tool, and only then —
if every rung fails — show a non-blocking notice explaining what is unavailable and how to
restore it.

**Capability probe**:
The thin piece of code that inspects the host and reports the raw *facts* about one Host
capability's state as plain data — e.g. "xclip present, wl-copy absent, session is Wayland".
A probe does no deciding: it only gathers facts (a `LookPath`, a `Getenv`, a `Stat`) and
carries no fallback logic of its own. Kept deliberately dumb so it needs only a smoke test;
the decision it feeds lives in the Fallback policy.
_Avoid_: detector (too vague — a probe reports, it does not choose).

**Fallback policy**:
The pure function that turns a Capability probe's facts into a chosen rung of the Fallback
strategy — facts in, decision out, no I/O. Because all the *risk* (which rung is right) lives
here and it touches nothing external, it is exhaustively unit-testable with hand-written fact
values, without simulating a whole host. Pairs with a Capability probe per capability
(clipboard, font, locale, paths).
