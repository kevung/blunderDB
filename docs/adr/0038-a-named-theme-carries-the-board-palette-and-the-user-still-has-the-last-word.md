# A named theme carries the board palette, and the user still has the last word

## Status

accepted — decided 2026-09-06

## Context

ADR-0031 established one colour palette for the interface: seven design tokens in
`style.css`, used by components instead of literal hex values. It also drew a boundary, in
writing: *"These are DESIGN tokens (the UI's own chrome): they are not the board's colours,
which stay a user preference in `boardColorsStore.js` / the Configuration 'Couleurs' tab and
are deliberately left out of this system."*

Named themes (#286, fiche I.30) ask the boundary to be revisited. A dark interface around a
light board is not a dark theme; it is half of one. The board occupies most of the window,
and it is the half the eye rests on. The same is true of the printable theme: an interface
that saves ink around a board that does not is pointless.

The reason ADR-0031 gave for the exclusion — the board's colours are the user's preference,
not the product's chrome — remains true and is not in dispute.

## Decision

**A named theme carries a board palette as well as the interface tokens, and applying a
theme applies both.**

- The four themes (`light`, `dark`, `contrast`, `print`) each define the seven ADR-0031
  tokens AND the nine board colours. They live together in `frontend/src/utils/themes.js`,
  as data rather than as CSS: writing each theme as a stylesheet would duplicate the token
  list once per theme, and a token added later would silently be missing from three of them.
- Choosing a theme writes its board palette through the existing
  `boardColorsStore`/`SaveBoardColors` path. Nothing new persists it; nothing special reads
  it back.

**The user still has the last word, and the mechanism guarantees it rather than promising
it.**

- The Configuration's *Couleurs* tab is unchanged and still edits the board palette
  directly. A colour the user sets after choosing a theme is theirs, and stays.
- Start-up applies the theme's INTERFACE tokens only, never its board palette. The palette
  loaded from the configuration is the user's, and rewriting it at every launch would erase
  their work one session at a time — which is the failure mode this rule exists to prevent.

**`system` is the default.** It follows `prefers-color-scheme` and reacts to it changing
mid-session. A tool does not impose its light or its dark on a desktop that has already
decided.

## Consequences

- ADR-0031's boundary is narrowed, not abandoned: the board's colours are still not design
  tokens, still not read by components, and still edited in their own tab. What changes is
  that a theme may propose a value for them, exactly as it proposes one for the chrome.
- The seven token names are now written twice — in `style.css` (the light defaults, so a
  page renders before any script runs) and in `themes.js`. `UI_COLOR_TOKENS` names them so a
  theme cannot silently omit one; a token added to `style.css` and forgotten in `themes.js`
  keeps the previous theme's value, which is the surest way to produce unreadable text.
- The board renderer needs no change: it already reads `boardColorsStore`, and a theme is
  simply another writer of it.
