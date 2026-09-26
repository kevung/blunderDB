# A named theme carries the board palette, and the user still has the last word

Status: accepted.
See also: ADR-0031

## Context

A dark interface around a light board is half a dark theme, and a printable interface around an
ink-heavy board is pointless: the board is most of the window. Yet the board's colours are the
user's preference, not the product's chrome (ADR-0031 rule 2), and that remains true.

## Decision

1. **A named theme carries a board palette as well as the interface tokens, and applying a theme
   applies both.** The themes (`light`, `dark`, `contrast`, `print`) each define the ADR-0031
   colour tokens (listed in `UI_COLOR_TOKENS`), the nine board colours and a `color-scheme`, as
   data in `frontend/src/utils/themes.js` — not as one stylesheet per theme, where a token added
   later would silently be missing from the others.
2. **Choosing a theme writes its board palette through the existing
   `boardColorsStore`/`SaveBoardColors` path.** Nothing new persists or reads it.
3. **The user has the last word, by mechanism.** The *Couleurs* tab still edits the board palette
   directly, and a colour set after choosing a theme stays. Start-up applies the theme's
   interface tokens only, never its board palette — rewriting it at every launch would erase the
   user's colours.
4. **`system` is the default.** It follows `prefers-color-scheme`, including changes
   mid-session.

## Consequences

- The board's colours are still not design tokens and not read by components; a theme only
  proposes values for them. The board renderer is unchanged — a theme is another writer of
  `boardColorsStore`.
- The colour token names are written twice: in `style.css` (light defaults, so the page renders
  before any script) and in `themes.js`. A token missing from a theme would keep the previous
  theme's value and produce unreadable text.

## Guard

`frontend/src/__tests__/themes.test.js`.
