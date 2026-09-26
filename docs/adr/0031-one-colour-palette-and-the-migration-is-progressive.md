# One colour palette, and the migration is progressive

Status: accepted.
See also: ADR-0008, ADR-0038

## Context

`style.css` declared a type scale (ADR-0008) and no colour, spacing or radius. Components reached
for literal hex values: 108 distinct colours, three competing primary blues, secondary text in
`#888` (3.54:1) and `#999` (2.85:1) — below the 4.5:1 WCAG AA floor at 11 px — three components
dropping Nunito from a partial copy of the font stack, and no spacing token at all. Hundreds of
literals across dozens of components cannot be migrated in one visually-unchecked pass.

## Decision

1. **One palette, declared once in `frontend/src/style.css`**, with two font-family tokens and a
   spacing/radius scale:

   ```css
   :root {
       --color-text: #333333;        /* 12.6:1 on white */
       --color-text-muted: #666666;  /* secondary text — 5.74:1, WCAG AA */
       --color-border: #cccccc;
       --color-surface: #ffffff;
       --color-surface-alt: #f5f5f5;
       --color-primary: #1976d2;
       --color-danger: #b3261e;
       --font-family-ui: 'Nunito', -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto',
           'Oxygen', 'Ubuntu', 'Cantarell', 'Fira Sans', 'Droid Sans', 'Helvetica Neue',
           'Noto Sans JP', sans-serif;
       --font-family-mono: 'Consolas', 'Monaco', 'Courier New', monospace;
       --space-1: 4px; --space-2: 8px; --space-3: 12px; --space-4: 16px;
       --radius: 4px;
   }
   ```

   Values are the ones the codebase already used most (ADR-0008's method). Secondary text is
   `--color-text-muted`, never a lighter grey.
2. **These are the interface's chrome tokens, not the board's colours.** The board palette is a
   user preference (`boardColorsStore.js`, `GetBoardColors`/`SaveBoardColors`, Configuration's
   *Couleurs* tab); how a theme proposes values for it is ADR-0038.
3. **Components use the tokens and the shared components.** A font stack is
   `var(--font-family-ui)` / `var(--font-family-mono)`, never a local copy. A titled checkbox
   list with All/None is `PickList.svelte`; a dialog is built on `Modal.svelte`.
4. **Migration is progressive, held by a ceiling that only moves down.** The count of hex
   literals inside component `<style>` blocks may not exceed `frontend/.color-token-budget`; a
   change that removes literals lowers the file in the same commit. `style.css` is exempt (the
   tokens live there); JS chart palettes (`components/stats/charts/palette.js`) and board
   defaults are out of scope.

## Consequences

- A new component may still reach for a hex value, but the total cannot grow.
- A hex literal cannot follow a theme — only a custom property redefined per theme can — so every
  remaining literal is a spot a theme (ADR-0038) does not reach.
- Rejected: a hard zero from day one (fails immediately or forces an unchecked mass rewrite).
  Rejected: no guard (an unenforced rule stops being checked and the count creeps back).
  Rejected: scanning JS too (chart palettes and board defaults are not chrome).

## Guard

`frontend/src/__tests__/colorTokens.sync.test.js` against `frontend/.color-token-budget`.
