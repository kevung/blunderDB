# One type scale for the interface, and form controls inherit it

## Status

accepted — and applied: every component now uses the tokens. `input, select, textarea,
button { font: inherit }` is declared globally in `style.css` since 2026-08-11 (it was
declared per component until then — see *Consequences*). The absolute sizes that remain are
the named exceptions below, each now behind its own token.

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

— was deliberately **not** applied globally at first: it would have changed the appearance
of every field in the application in one commit, and nothing tested that. Each component
declared it for itself as it was migrated, nineteen times over. Once no component depended
on the browser's control font any more, the per-component copies were removed and the line
was moved into `style.css`, declared once (2026-08-11).

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
argued, each now behind its own token rather than a repeated magic number:

- **Dialog titles** — `--font-size-dialog-title` (20 px). A modal's title is scanned before
  its body is read, so it sits a size above a panel title. Two small utility dialogs,
  `ConfigModal` and `ProtectedCopyModal`, keep `--font-size-title` (15 px) on their title
  instead — a deliberate choice, not an oversight, made when the token was introduced
  (2026-08-11): both are compact, low-ceremony dialogs where a panel-sized title reads right.
- **Dialog close crosses** — `--font-size-dialog-close` (24 px).
- **Figures meant to be read at a glance** — `--font-size-stat-figure` (28 px): the large
  numbers in the statistics tabs, and the running counters in the import-progress dialogs
  (`FileImportProgressModal`, `ImportProgressModal`). Both are numbers the screen exists to
  show, not "text".
- **Monospace nudged to `0.92em`** (rule 4 above) and explicit `font-size: inherit` (which is
  rule 1's "or nothing at all, and inherits" spelled out) are not exceptions to name — they
  already comply with the rule as written.

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

- **The migration was transversal and visual, and no test covered it.** It was therefore
  done in two passes — the four heaviest panels first, checked on screen, then the remaining
  twenty-seven components — rather than in one commit. `font: inherit` was declared by each
  component that had controls (nineteen of them by the end) instead of once globally, so a
  future component that deliberately wanted the platform's control font could still say so.
- **The global line landed on 2026-08-11**, once the per-component count had grown to
  nineteen with no exception among them: every component with form controls wanted the
  inherited font, so the reason to keep the rule local (letting a future component opt out)
  had never been exercised. `input, select, textarea, button { font: inherit; }` now lives
  once in `style.css`; the nineteen local copies (and their repeated comment) are gone. A
  component that genuinely wants the platform's control font can still override with its own
  `font-family`/`font-size` — the global rule does not forbid that, it just stops it being the
  silent default.
- The four heaviest components — `MatchPanel` (33 declarations), `AnkiPanel` (29, seven
  distinct sizes), `SearchPanel` (26), `TournamentPanel` (17) — held 37% of all declarations
  and were migrated first, for that reason.
- **Progress is measurable**, which is the point of writing the baseline down. The migration
  took the codebase from 285 declarations over 20 distinct values to 263 using the tokens and
  20 absolute ones, all of them named exceptions. What should stay near zero is the count of
  *unexplained* absolute values:

  ```bash
  cd frontend/src && grep -rho 'font-size:[^;]*;' components/ *.svelte \
      | grep -v 'var(--font-size' | sort | uniq -c | sort -rn
  ```

- Sizes below `--font-size-small` disappear: 9 px and 10 px were used to de-emphasise
  counters and badges, which is exactly the "shrink to demote" habit rule 2 rejects. Those
  become 11 px and lean on colour instead — a deliberate, visible change on a few badges.
- A reviewer now has something to point at. "Why is this 11 px?" has an answer other than
  taste.
- **The exceptions themselves converged onto tokens on 2026-08-11** (fiche-08). The 20
  absolute values counted above had, by then, fragmented into duplicates: dialog titles alone
  spanned four values (a 15 px token use, 20 px, `1.25rem`, and a 12 px base-token misuse in
  `MergePlayersModal`), close crosses spanned three (24 px, `1.5rem`, 18 px), and three
  absolutes sat outside any named exception (`App.svelte`'s drop overlay at `1.3rem`, and the
  28 px import counters, which had never been folded into the statistics-figure exception).
  Two tokens fixed the first two: `--font-size-dialog-title` (20 px) and
  `--font-size-dialog-close` (24 px). The statistics-figure exception grew a token too,
  `--font-size-stat-figure` (28 px), reused by the import-progress counters since they are the
  same kind of figure. Before:

  ```
        4 font-size: 28px;
        4 font-size: 20px;
        3 font-size: 1.5rem;
        2 font-size: inherit;
        2 font-size: 24px;
        2 font-size: 18px;
        2 font-size: 0.92em;
        1 font-size: 1.3rem;
        1 font-size: 1.25rem;
  ```

  After:

  ```
        2 font-size: inherit;
        2 font-size: 0.92em;
  ```

  Both remaining lines already comply with rule 1 as written (`inherit` *is* "nothing at all,
  and inherits") and rule 4 (the monospace nudge) — zero unexplained absolute values.
  `ConfigModal` and `ProtectedCopyModal` keep `--font-size-title` on their dialog title
  instead of adopting the new `--font-size-dialog-title`: both are compact utility dialogs
  where the smaller, panel-scale title was judged to read right, so they were left alone
  rather than folded into the new token for uniformity's sake.
