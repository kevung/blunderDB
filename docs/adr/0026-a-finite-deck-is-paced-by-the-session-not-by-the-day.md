# A finite deck is paced by the session, not by the day

Status: accepted.
See also: ADR-0025

## Context

Anki's daily caps exist for a deck that grows without end. A blunderDB deck is built from a
collection or a saved search — 20 to 300 positions, a stable corpus — so a daily cap either
never bites or manufactures a backlog on a deck that fits in one sitting; a review cap hides
due cards and postpones them. FSRS's authors treat desired retention as a choice and warn
against steering it automatically; `go-fsrs/v3` ships no weight trainer. And every new card of
a synced deck shares one due timestamp, so `due ASC` serves a match's consecutive moves in
order — blocking, where interleaving is known to help.

## Decision

1. **Pacing is per deck, in the deck's settings**, not in the application configuration. No
   global defaults layer, no preset inheritance.
2. **The limit is a number of cards per session, never a daily cap** — neither new cards nor
   reviews are capped by day. No notion of day, rollover or timezone; the session stops
   between two questions. Card counts stay raw. Cram mode is never bounded by it.
3. **Unlimited is the default.** The `session_limit` column is null unless set; `0` means
   "no cards this session", not "unlimited" — nil and zero are distinct states.
4. **Reaching the limit is said out loud**: the end of a capped session names the limit and
   what remains. No "keep going anyway" button; cram mode serves more without scheduling.
5. **Retention is measured, never steered.** The deck settings show observed retention
   against the target over a sample size; nothing writes the target back. The measurement is
   named for what it does, not "optimize".
6. **The FSRS weights are not exposed and no optimiser is faked**: defaults are documented as
   excellent, and a button that cannot re-fit must not be shown.
7. **Maximum interval defaults to one year for new decks.** Existing decks keep their value:
   a changed default never reschedules existing cards.
8. **Changing retention is not retroactive**, and the setting says it applies as reviews
   happen.
9. **Ties in the draw order are broken at random**, deliberately and commented as such; no
   order setting is exposed.

## Consequences

- `anki_deck.session_limit` is a schema column, in all three schema homes.
- There is no `/v1/anki.optimizeParams` route and no `apply` path.
- A test may assert the set of cards a session draws, never their sequence (rule 9).
- Rejected: new-cards-per-day cap — on a finite corpus the peak is bounded by the deck.
- Rejected: a session duration — cuts at the mercy of content, and mid-position.
- Rejected: global defaults overridable per deck (rule 1).
- Rejected: an exposed display-order setting (rule 9).

## Guard

`frontend/src/__tests__/ankiService.sessionLimit.test.js`,
`frontend/src/__tests__/AnkiPanel.sessionLimit.test.js`,
`pkg/blunderdb/storage/sqlite/anki_sqlite_test.go`,
`pkg/blunderdb/storage/postgres/anki_postgres_test.go`.
