# One type scale for the interface, and form controls inherit it

## Status

accepted — the rule applies to new and modified code from now on; the migration of existing
components is deliberately gradual (see *Consequences*).

## Context

Users kept reporting the same thing about different screens: the panel "isn't pretty", the
sizes "don't go together", the dialog is "biscornu". Each report was answered locally, and
the defect reappeared on the next screen — including in the two components fixed most
recently, which ended up with **two different base sizes** (12 px in the metadata panel,
13 px in the settings dialog). Fixing it screen by screen was reproducing it.

Measured across `frontend/src` when this was written:

- **285 `font-size` declarations** in components, using **20 distinct values** — 9, 10, 11,
  12, 13, 14, 15, 16, 18, 20, 28 px, `1.5rem`… About five per component.
- `style.css` sets a font *family* on `body` and **no font size at all**. There is therefore
  no reference: the root size is the browser's, and every component starts from scratch in
  absolute pixels.
- **Twelve components contain form controls whose font family is still the browser's.** A
  control inherits neither size nor family. Several of them do set a *size* on their inputs
  through class selectors, which hides the problem just enough that nobody notices the
  family is wrong.

The last point is the one that is felt rather than seen: a field shows its label in Nunito
and its value in the platform's control font. Nothing looks broken; the screen just looks
untidy.

## Decision

**One scale, declared once, in `frontend/src/style.css`:**

```css
:root {
    --font-size-base: 12px;   /* body text, labels, values, controls, buttons */
    --font-size-small: 11px;  /* dense lists, secondary notes and hints */
    --font-size-title: 15px;  /* panel and dialog titles */
}
```

The values are taken from what the codebase already does, not from taste. Counting the
panels that carry the most type — matches, Anki, search, tournaments — gives 55 declarations
at 11 px and 28 at 12 px against a handful at every other value. Choosing anything else
would have meant a visible change on every dense list for no reason. (An earlier draft of
this decision proposed 13 px, inferred from a single settings dialog; the repository-wide
count corrected it.)

The tokens exist as of this decision. The companion rule —

```css
input, select, textarea, button { font: inherit; }
```

— is deliberately **not** applied globally yet: it would change the appearance of every field
in the application in one commit, and nothing tests that. Each component declares it for
itself as it is migrated; the global line is the last step, once no component depends on the
browser's control font any more.

Four rules follow from it, and they are what a reviewer should check:

1. **A component does not declare an absolute font size.** It uses the tokens, or nothing at
   all and inherits.
2. **Hierarchy is carried by weight and colour, not by size.** A label differs from a value
   by being uppercase, semibold and grey — no smaller.
3. **Form controls inherit** (`font: inherit`). This single line is what fixes the twelve
   components at once; setting only a size leaves the family wrong.
4. **Monospace is nudged to `0.92em`** where it sits next to proportional text, because a
   monospaced face reads a size larger at equal nominal size.

Named exceptions, because a rule with no exceptions gets broken silently instead of
argued: **chrome** (a dialog's title, its close cross), and **figures meant to be read at a
glance** — the large numbers in the statistics tabs, at 20 and 28 px, are the point of those
tabs and are not "text".

## Considered options

- **Leave each component to its own scale**, the status quo. Rejected on evidence: it is what
  produced 20 sizes and two conflicting bases within a fortnight, and every fix was a
  reaction to a user noticing, never to a test.
- **A rem-based scale** (`0.8rem`, `0.9rem`…). Rejected: the interface already has its own
  zoom, applied as a CSS `zoom` on the root and persisted as a user setting. Adding a second,
  independent scaling mechanism would make "small text" mean two different things depending
  on which knob moved.
- **Utility classes** (`.text-sm`, `.text-base`). Rejected: this codebase styles components
  with scoped CSS and no utility layer; introducing one for typography alone would leave two
  idioms side by side.
- **Migrating every component at once.** Rejected as a *decision*, not as a goal — see below.

## Consequences

- **The migration is transversal and visual, and no test covers it.** Adding
  `font: inherit` for controls changes the appearance of every field in the application at
  once. It is therefore done screen by screen, with the change confirmed on screen before
  moving on. The rule binds new and modified code immediately; existing code converges.
- The four heaviest components — `MatchPanel` (33 declarations), `AnkiPanel` (29, seven
  distinct sizes), `SearchPanel` (26), `TournamentPanel` (17) — hold 37% of all declarations
  and are where the migration pays best.
- **Progress is measurable**, which is the point of writing the baseline down. Counting
  declarations and distinct values says whether the codebase is converging:

  ```bash
  cd frontend/src && grep -ro 'font-size:[^;]*;' components/ *.svelte | wc -l
  ```

- Sizes below `--font-size-small` disappear: 9 px and 10 px were used to de-emphasise
  counters and badges, which is exactly the "shrink to demote" habit rule 2 rejects. Those
  become 11 px and lean on colour instead — a deliberate, visible change on a few badges.
- A reviewer now has something to point at. "Why is this 11 px?" has an answer other than
  taste.
