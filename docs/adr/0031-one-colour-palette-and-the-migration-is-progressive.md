# One colour palette, and the migration is progressive

## Status

accepted — tokens declared and enforced by a ceiling, migration in progress. Dark mode
(a second set of these same tokens, gated on `prefers-color-scheme` plus a Configuration
setting) is deliberately **not** part of this decision; it depends on the tokens existing
first and is tracked separately (fiche D.9's second half, #209).

## Context

`frontend/src/style.css` declared a type scale (ADR-0008) and nothing else: no colour, no
spacing, no radius. Every component reached for a literal hex value instead, and nothing
stopped two components from reaching for two different values meaning the same thing.
Measured when this was written:

- **108 distinct hex colours**, 803 occurrences across `.svelte`/`.js` files (635 of them
  inside a component's own `<style>` block, the rest in JS chart palettes and generated
  markup); the ten most common values alone accounted for 342 occurrences.
- **Three competing primary blues**: `#1976d2` (24×), `#1a56c4` (10×), `#1a73e8` (8×) — the
  same accent, drifted into three values by three different authors reaching for "a blue"
  instead of a name.
- **Two WCAG AA failures on the smallest text**: `color: #888` (3.54:1) and `color: #999`
  (2.85:1) used as ordinary secondary-text colour, both below the 4.5:1 floor the type
  scale's own smallest size (`--font-size-small`, 11px) requires.
- **Three components silently dropped Nunito**: `StatusBar`, `MatchInfoBar`, `EPCPanel` each
  declared `font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Noto Sans
  JP', sans-serif` — a partial copy of `body`'s stack in `style.css`, missing `'Nunito'`
  itself and several of its fallbacks. Six declarations used a monospace stack, spread across
  three variants (a bare `monospace`, and `'Consolas', 'Monaco', 'Courier New', monospace`
  repeated four times in `StatusBar` alone).
- **34 duplicated CSS rule bodies**: the clearest case was a "titled list of checkboxes with
  All/None" pattern, written three times inside `ExportDatabaseModal.svelte`'s own local
  Svelte 5 snippet, and independently a fourth time (slightly differently) in
  `MatchTournamentPickerModal.svelte`.
- **Zero spacing or radius tokens** — every `padding`, `gap` and `border-radius` was its own
  literal.

## Decision

**One palette, declared once, in `frontend/src/style.css`, alongside two font-family tokens
and a spacing/radius scale:**

```css
:root {
    --color-text: #333333; /* body text, headings — 12.6:1 on white */
    --color-text-muted: #666666; /* secondary text, hints, counts — 5.74:1 on white (WCAG AA);
                                     replaces #888 (3.54:1) and #999 (2.85:1) */
    --color-border: #cccccc;
    --color-surface: #ffffff;
    --color-surface-alt: #f5f5f5;
    --color-primary: #1976d2; /* replaces #1976d2, #1a56c4, #1a73e8 */
    --color-danger: #b3261e;

    --font-family-ui: 'Nunito', -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto',
        'Oxygen', 'Ubuntu', 'Cantarell', 'Fira Sans', 'Droid Sans', 'Helvetica Neue',
        'Noto Sans JP', sans-serif;
    --font-family-mono: 'Consolas', 'Monaco', 'Courier New', monospace;

    --space-1: 4px;
    --space-2: 8px;
    --space-3: 12px;
    --space-4: 16px;
    --radius: 4px;
}
```

Values were taken from what the codebase already does most, the same method ADR-0008 used
for the type scale: `--color-text` is `#333` (already the single most common text colour,
50+ occurrences); `--color-text-muted` is `#666` (already 5.74:1-compliant and already the
third most common secondary-text grey — folding `#888`/`#999` into it fixes the two WCAG
failures as a side effect of consolidating, not as a separate pass); `--color-primary` is
`#1976d2`, the most-used of the three competing blues.

**These are design tokens for the interface's own chrome. They are not the board's
colours.** The board's palette (felt, points, checkers, dice, cube) is a user preference,
persisted through `boardColorsStore.js` / `GetBoardColors`/`SaveBoardColors` and edited in
Configuration's "Couleurs" tab — a deliberate, per-installation choice, unrelated to what
this ADR is about. Nothing here touches that store, its defaults, or the tab; the guard
test below only looks inside `<style>` blocks and cannot see it either way. The board's
Two.js scene reading its colours from these *design* tokens is a different, later change
(I.30), not this one.

**Migration is progressive, not a single pass** — 635 hex literals in 38 components is not
a one-sitting rewrite, and ADR-0008 already established the pattern for why: a transversal,
visual change with no test covering it is done in bounded batches, not committed all at
once. What landed with this decision:

- The three named bugs above, fixed directly: the three competing blues consolidated to
  `var(--color-primary)` (41 occurrences); `#888`/`#999`/`#666` text colours consolidated to
  `var(--color-text-muted)` (103 occurrences, and the two WCAG failures with them); `#333`
  consolidated to `var(--color-text)` (52 occurrences); the three off-Nunito components and
  every monospace stack moved onto `var(--font-family-ui)`/`var(--font-family-mono)`.
- `PickList.svelte`: the three-times-repeated snippet in `ExportDatabaseModal` promoted to a
  shared component, and `MatchTournamentPickerModal`'s independent fourth copy (which had
  grown an inline filter box and a disabled-checkbox state the original snippet didn't)
  rewritten onto it instead — both call sites, one shared implementation, one place the
  duplicated CSS rule bodies now live.
- The search panel's "save this search as a filter" dialog, previously a hand-rolled
  `role="dialog"` + `use:trapFocus` overlay, now built on `Modal.svelte` like every other
  dialog in the application, instead of alongside it.

**A ceiling, not a target of zero, enforces it** — `colorTokens.sync.test.js` counts hex
colour literals inside every component's `<style>` block (style.css itself is exempt: it is
where the tokens live) and fails if the count exceeds `.color-token-budget`. This is
deliberately the same shape as `.svelte-warnings-budget` / `check-svelte-warnings.mjs`
(#205), not the "everything must be a token, no exceptions" shape ADR-0008's
`fontScale.sync.test.js` eventually reached: that test could demand zero because the
migration it guards was already complete when it landed. This one is not. The ceiling
recorded a real count on the day it was written (406, down from 635) and only ever moves
down — a future fix that replaces literals with tokens lowers the count and should lower the
file in the same commit; a new component may still reach for a hex value today, but the
total may not grow past the recorded ceiling.

## Considered options

- **A hard zero from day one**, ADR-0008's eventual shape. Rejected for now: ADR-0008 itself
  took two visual passes and a committed baseline before the count reached zero and the
  guard could demand it; asserting zero against 635 unmigrated literals would either fail
  the suite immediately or force a same-day rewrite of 38 components with no visual check on
  any of them.
- **No guard at all, migrate opportunistically.** Rejected on the same evidence ADR-0008
  cites for the type scale: an unenforced rule is a rule reviewers eventually stop checking,
  and a fix's own gain (406 today) needs to be locked in or the count silently creeps back up
  the way `font-size` did before its own ceiling existed.
- **Scan JS for hex too** (chart palettes, `boardColorsStore.js` defaults). Rejected: a chart
  palette (`components/stats/charts/palette.js`) is a legitimate, colour-scale-specific
  concern that does not belong on the interface's chrome tokens, and the board defaults are
  explicitly a user preference this ADR does not touch. Scoping the guard to `<style>` blocks
  keeps both out without a manual exception list.

## Consequences

- **Migrating a component off its own hex values is now measurable against a shrinking
  number**, the same discipline ADR-0008 established for font sizes: `.color-token-budget`
  went from (implicitly) unbounded to 406 in this change, and the next fiche that touches a
  component's `<style>` block should lower it further rather than leaving new literals beside
  the ones already fixed.
- **Dark mode is not this decision.** It needs every remaining literal to be a token first —
  a hex value can't respond to `prefers-color-scheme`, only a custom property redefined under
  it can — so the ceiling above is the blocker, not a detour. Tracked as fiche D.9's second
  half.
- `PickList.svelte` is not the whole answer to "34 duplicated CSS rule bodies" — it accounts
  for one recurring pattern (a titled checkbox list with All/None), not the other 8
  components named as > 700 lines in fiche D.10's "modules-dieux" audit. Those are a
  different fiche's work.
