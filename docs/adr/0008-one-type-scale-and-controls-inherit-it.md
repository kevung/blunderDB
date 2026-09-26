# One type scale for the interface, and form controls inherit it

Status: accepted.

## Context

Components each declared their own absolute font sizes (285 declarations, 20 distinct values),
`body` set no size, and form controls kept the browser's control font — a field showed its label
in Nunito and its value in the platform font. Fixed screen by screen, the defect reappeared on
the next screen. The scale values are the ones the densest panels already used (11 px and 12 px
dominated), not a matter of taste.

## Decision

One scale, declared once in `frontend/src/style.css`:

```css
:root {
    --font-size-base: 12px;   /* body text, labels, values, controls, buttons */
    --font-size-small: 11px;  /* dense lists, secondary notes and hints */
    --font-size-title: 15px;  /* panel and dialog titles */
}
input, select, textarea, button { font: inherit; }
```

1. **A component does not declare an absolute font size.** It uses the tokens, or nothing
   (or `font-size: inherit`) and inherits.
2. **Hierarchy is carried by weight and colour, not by size.** A label differs from a value by
   being uppercase, semibold and grey — not smaller. Nothing goes below `--font-size-small`.
3. **Form controls inherit** (`font: inherit`, declared globally). A component that genuinely
   wants the platform control font overrides it explicitly.
4. **Monospace next to proportional text takes `--font-size-small`**, since a monospaced face
   reads a size larger at equal nominal size.

Named exceptions, each behind its own token:

- `--font-size-dialog-title` (20 px) — modal titles. `ConfigModal` and `ProtectedCopyModal`
  deliberately keep `--font-size-title`: compact utility dialogs where a panel-sized title reads
  right.
- `--font-size-dialog-close` (24 px) — dialog close crosses (and `App.svelte`'s drop overlay).
- `--font-size-stat-figure` (28 px) — figures read at a glance: the statistics tabs' large
  numbers and the counters of `FileImportProgressModal` / `ImportProgressModal`.

## Consequences

- Adding an exception means adding it here and to the guard's list, in that order.
- 9 px and 10 px badges became 11 px and rely on colour — "shrink to demote" is rejected.
- Rejected: per-component scales (produced 20 sizes and two conflicting bases). Rejected: a
  rem-based scale (the interface already has its own CSS `zoom`; a second scaling knob would
  make "small" mean two things). Rejected: utility classes (the codebase uses scoped CSS, no
  utility layer).

## Guard

`frontend/src/__tests__/fontScale.sync.test.js`: every `font-size` in a `.svelte` file is a token
or `inherit`, every `font` shorthand is `inherit`, and the three chrome tokens are accepted only
where this ADR places them.
