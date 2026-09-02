# A finite deck is paced by the session, not by the day

## Status

accepted — decided 2026-09-02, the same day as ADR-0025, which it extends: that one settled how a
review card asks its question, this one settles what the user may tune about the rhythm of asking.

Schema-affecting (one new per-deck column), engine-neutral, and it **removes** a shipped server
route rather than adding one.

Its evidence base is `doc/archive/2026-09-anki-pacing-research.md`, an external research report
commissioned for this decision. Read its confidence levels before quoting a figure from it: it
separates what the Anki manual documents from what community folklore asserts.

## Context

### The obvious answer was the wrong one

The Anki-shaped answer to "let me control my review pace" is a daily cap: *new cards/day* (default
20) and *maximum reviews/day* (default 200). It is what everybody has seen, so it is what gets
asked for, and this decision started out heading there.

It is wrong here, and the reason is the shape of the corpus. Anki's daily caps exist for a deck
that **grows without end** — a language learner adds cards forever, so the rate of introduction is
the only thing standing between them and a wall of reviews. A blunderDB deck is built from a
collection or a saved search: 20 to 300 positions, a **stable** corpus. The whole thing is
introduced in a few sessions, after which the regime is reviews only, and their daily volume is
already bounded by the size of the deck and by FSRS itself.

A daily cap on a finite deck therefore lands in one of two states, both bad: high enough never to
bite, and it is a setting that does nothing; low enough to bite, and it manufactures a backlog on a
deck that would have fitted in one sitting. The chess trainers that solved this problem before us —
Chessable, ChessTempo — bound the **session**, not the day.

A cap on *reviews* is worse still and is rejected outright: it hides cards that are already due,
they pile up unseen, and the Anki manual's own remedy for a backlog is to raise that cap to 9999.
A cap that truncates a due queue does not remove work, it postpones it onto tomorrow's.

### The optimiser we already shipped is the one mechanism the authors warn against

`domain.SuggestRetention` measures the observed pass rate and nudges `request_retention` to close
the gap with the deck's target — writing it back when asked. It is tested, and served at
`/v1/anki.optimizeParams`. It reached no GUI, which was already a parity break.

It is also, precisely, the mechanism FSRS's authors reject. Desired retention is a **choice** on
the work/knowledge trade-off; true retention is the outcome. The documented way to close a gap
between them is to re-fit the weights, not to chase the outcome by moving the target. No author
endorses an automatic loop, and `go-fsrs/v3` ships no weight trainer for us to re-fit with anyway.

Measuring is not the problem. Steering is.

### Nobody chose the order cards come out in

Cards are drawn learning/relearning first, then review, then new, ties broken by `due ASC`. But
every new card of a freshly synced deck carries the *same* due timestamp — 1395 of them shared one
second in the validation database — so `due ASC` breaks nothing and SQLite returns insertion order:
**the order of the match**. A session serves position 1, then 2, then 3: consecutive moves of one
game, which are correlated, in the sequence they were played. That is blocking, and the literature
this decision rests on says interleaving beats it for category learning.

## Decision

1. **Pacing is per deck, in the deck's own settings.** Not in the application's configuration
   panel, which holds what is global (interface, colours, bearoff, engine, identity). A deck is
   thematic — the rhythm wanted on "my cube errors at score" is not the one wanted on "opening
   positions" — and the three FSRS settings it already carries live there. No global defaults
   layer, no preset inheritance: that machinery exists in Anki for someone with forty decks.

2. **The limit is a number of cards per session, never a daily cap.** Neither new cards nor
   reviews are capped by day. A session limit needs no notion of "day" — no rollover hour, no
   timezone travelling through the Storage contract, no UTC boundary cutting a European evening in
   half — and it stops between two questions rather than in the middle of one, which a duration
   would not. Card counts stay raw: nothing is ever silently truncated.

3. **Unlimited is the default, and the setting is opt-in.** Existing decks are unaffected — the
   new column is born null. `0` means "no cards this session", not "unlimited": freezing a deck
   while preparing for a tournament is a real use, and conflating it with "no limit" is a trap Anki
   is known for.

4. **Reaching the limit is said out loud.** "Review complete! N cards reviewed" is a lie when a
   limit cut the queue. The end of a capped session names the limit and says what remains. There is
   no "keep going anyway" button: cram mode already serves more positions and, unlike an override,
   it does not schedule anything.

5. **We measure retention, we never steer it.** `SuggestRetention` and the `apply` path go;
   the measurement stays and is renamed for what it does. The deck settings show observed retention
   against the target, over a sample size — information the user acts on, or not. Renaming is part
   of the decision, not cosmetics: left called "optimize", the write-back gets rebuilt by whoever
   trusts the verb.

6. **The FSRS weights are not exposed, and no optimiser is faked.** `go-fsrs/v3` ships the
   scheduler only. The default weights are trained on hundreds of millions of reviews and are
   documented as excellent as-is; the measured superiority of optimised weights over defaults is
   real but modest, and defaults win outright in about one case in six. A button that cannot
   re-fit anything must not be shown.

7. **Maximum interval defaults to one year for new decks.** 36500 days is Anki's, meant for
   vocabulary one never wants to see again. A position FSRS defers by eleven years has not been
   learned, it has left the deck without anyone deciding so — and a player's own game changes
   faster than that. Existing decks keep their stored value: a changed default must never
   reschedule cards that exist.

8. **Changing retention is not retroactive, and the panel says so.** Each card adopts the new
   rhythm at its next review. Rewriting existing due dates from a model would move a card due
   tomorrow to six months out without being asked. The cost is that the change is invisible for
   days, so the setting carries a line saying it applies as reviews happen — otherwise the user
   concludes the setting is broken.

9. **Ties in the draw order are broken at random, and no order setting is exposed.** Random is the
   behaviour, not an option. It must be written deliberately and commented as such: this project
   already has an accidental non-total `ORDER BY` whose results differ between backends, and only
   the comment distinguishes the two situations.

## Considered options

- **New-cards-per-day cap, Anki-style** — rejected under rule 2: the argument for it (introducing
  50 positions today manufactures 50 due dates in three days) is true but secondary on a finite
  corpus, where the peak is bounded by the deck itself.
- **Session limit expressed as a duration** — rejected: a cube decision at score is pondered for
  three minutes and a forced move for three seconds, so a clock cuts at the mercy of content, and
  it cuts *during* a position — exactly when the user has committed their thinking and not yet seen
  the answer.
- **Global defaults in the configuration panel, overridable per deck** — rejected under rule 1.
- **Wiring the existing `OptimizeParams` to the GUI** — this was the plan until the research
  contradicted it; see rule 5.
- **An exposed display-order setting** — rejected under rule 9: cost in cognitive load, no gain,
  and the default it would compete with is the one the literature already favours.

## Consequences

- A new per-deck column means a `DatabaseVersion` bump with its migration in **three** places —
  `database`, `storage/sqlite`, `storage/postgres` — plus the continuous-chain test.
- `/v1/anki.optimizeParams` disappears. The break is deliberate and its only consumer is the `call`
  CLI; no GUI ever reached it.
- Rule 9 changes which card a session serves first. Any test asserting a specific first card must
  become an assertion about the *set* drawn, not its sequence.
