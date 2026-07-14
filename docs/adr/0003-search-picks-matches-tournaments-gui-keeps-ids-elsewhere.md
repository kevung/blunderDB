# GUI search picks matches/tournaments from a list; CLI and command line keep raw IDs

## Status

accepted

## Context

`SearchPanel.svelte` filters positions by match and tournament with two free-text
fields (`matchIDsInput`, `tournamentIDsInput`), each turned into a token (`ma...`,
`tn...`) that the backend parses via `parseFilterIDList`
(`storage/sqlite/search_helpers_sqlite.go`). A user has to already know the numeric
IDs of the matches or tournaments they want — IDs that are not shown anywhere in the
search panel itself, only in the separate `MatchPanel`/`TournamentPanel` browsers.

The parser itself is narrower than its own UI hint claims. The placeholder text
("ID or range (e.g. 1,3,5-9)") and the CLI's `--match-ids`/`--tournament-ids` help
text both advertise a comma-separated list and a dash range, but `parseFilterIDList`
only recognizes two wire formats: exactly two comma-separated numbers as an inclusive
range (`2,7` → 2..7), or a semicolon-joined explicit list (`5;10;15`). A 3+-item comma
list (`1,3,5`, the hint's own example) or a dash range (`5-9`) matches neither branch;
the parse error is swallowed by the caller (`if ids, err := parseFilterIDList(...); err
== nil`), and the search silently degrades to zero results.

Tournament filtering is not a separate query path: a tournament ID is expanded to its
member match IDs and OR'd (unioned) into the same match-ID list used by direct match
selection (`search_sqlite.go`, `getMatchIDsForTournament`).

`ExportDatabaseModal.svelte` already solves "let the user pick which matches/
tournaments to act on" with a checkbox-list-plus-All/None modal, backed by the same
`GetAllMatches()`/`GetAllTournaments()` Wails calls that back `MatchPanel`/
`TournamentPanel`. Neither those calls nor those panels paginate or virtualize —
they load and render every row unconditionally — and that has been adequate at the
user's real database sizes ("thousands or more" matches).

## Decision

- The GUI search panel replaces both free-text ID fields with a single combined
  modal picker (reusing `ExportDatabaseModal`'s checkbox-list-plus-All/None pattern),
  opened from either the Match IDs or Tournament IDs filter toggle. It has two
  sections — Matches and Tournaments — each independently text-filterable (by player
  name / date / event / tournament name for matches; by name / date / location for
  tournaments), with the same columns and default sort as the existing
  `MatchPanel`/`TournamentPanel` browsers.
- `All`/`None` act on the currently filtered/visible subset, not the whole database —
  consistent with treating the text filter as narrowing the working set rather than
  merely the view.
- The union semantics stay: checking a tournament is equivalent to checking all of its
  member matches. The modal makes this visible instead of leaving it an invisible
  implementation detail — checking a tournament shows its member match rows as
  checked-and-disabled in the Matches section, so excluding one of them requires
  unchecking the tournament and picking matches individually.
- The picker always emits the safe semicolon-joined list format on the wire
  (`ma2;5;9`), never the ambiguous comma-range form, sidestepping
  `parseFilterIDList`'s narrow parsing for every GUI-driven search.
- `parseFilterIDList` itself is fixed to accept a real comma-separated list (3+
  items) alongside the existing two-item range and semicolon list, and the
  misleading placeholder/help text (GUI hint, CLI `--match-ids`/`--tournament-ids`
  help) is corrected to describe what's actually accepted.
- Restoring a saved/history search silently drops any match/tournament ID that no
  longer exists and keeps the rest — consistent with how a deleted match's positions
  are already handled elsewhere (no dangling reference is surfaced as an error).
- The picker list is a plain, unvirtualized loop like `MatchPanel`/`TournamentPanel` —
  not a new windowing/virtualization pattern. Those panels already render every row
  at the same real-world scale without a reported problem, and the modal's own text
  filter narrows what's on screen at once.
- The command line (in-app `s ma<ids> tn<ids>` command-bar syntax) and the OS-level
  CLI (`--match-ids`, `--tournament-ids`) are untouched: both are scripting-oriented
  surfaces where a numeric ID is the correct unit, and this decision only replaces
  the *interactive* GUI entry point. `cli list` gains a `--type tournaments` mode
  (mirroring the existing `--type matches`) so CLI users have the same
  ID-discoverability the GUI picker now has, closing a pre-existing parity gap.
- **Amendment (found during implementation):** the search panel's two separate
  "Match IDs"/"Tournament IDs" filter toggles were themselves still ID-centric
  labels for a control that no longer deals in IDs, and having two independent
  toggles open the same shared modal let one get checked (e.g. a tournament
  selected while opened from "Match IDs") without its own toggle turning on —
  silently dropping that half of the selection from the actual search. Both
  toggles are merged into one filter row, "Matches & Tournaments", backed by a
  single `filterEnabled` flag; the search's active-filter computation treats it as
  shorthand for both underlying `Match IDs`/`Tournament IDs` tokens, so enabling it
  once is sufficient regardless of whether the user ends up picking matches,
  tournaments, or both.

## Considered options

- **Keep the free-text field, fix only the hint/parser.** Rejected: it still asks a
  human to know and type numeric IDs, which is the actual complaint driving this
  work — a correct parser doesn't change that the IDs aren't visible anywhere in the
  search panel.
- **Inline expandable checkbox list inside the search panel** instead of a modal.
  Rejected: at "thousands or more" matches, an inline list would either need its own
  scroll container competing with the rest of the panel's height, or a hard cap that
  reintroduces the original discoverability problem. A modal gets a full viewport to
  work with and keeps the collapsed panel compact.
- **Two separate modals** (Matches-only, Tournaments-only). Rejected: since
  tournament selection is really match-ID-list expansion under the hood, keeping
  both sections in one modal lets the visual-sync behavior (tournament checks its
  matches) work naturally; two modals would hide that relationship or require
  duplicating it awkwardly across dialogs.
- **Independently clickable "exception" matches** under a checked tournament (uncheck
  one match while the tournament stays checked, i.e. tournament-minus-one). Rejected:
  adds hidden per-match override state that has to be tracked, explained, and kept in
  sync if the tournament's membership changes later — complexity this feature exists
  to remove, not add.
- **`All` always selects every match regardless of the active filter.** Rejected: with
  a filter narrowing the visible rows, a global `All` reads as `All` correctly at a
  glance but silently selects rows the user never saw, which is more surprising than
  useful.
- **Add virtualization to the picker now.** Rejected: no part of the frontend
  virtualizes lists today, and the panels this picker's data comes from already
  handle the same real-world scale unvirtualized. Building windowing for a problem
  not yet observed is speculative; revisit if the picker specifically turns out to be
  slow.

## Consequences

- The search panel's Match IDs / Tournament IDs filters go from "type numbers you
  have to already know" to "look at a list and check boxes," at the cost of a new
  modal component (largely copied from `ExportDatabaseModal`) and the visual-sync/
  disabled-checkbox logic between its two sections.
- `parseFilterIDList` becomes strictly more permissive (adds real comma lists) with
  no change to its existing range/semicolon behavior, so no existing saved search or
  CLI invocation changes meaning.
- `matchIDsInput`/`tournamentIDsInput` and their raw-text UI disappear from
  `SearchPanel.svelte`; anything that inspected or set them directly (e.g.
  `searchFilterService.js`'s restore path) now targets a selected-ID array/set
  instead of a string, and must resolve tournament membership to render the
  disabled-match sync state.
- CLI gains `list --type tournaments`, a small, independent addition that also
  benefits any CLI-only workflow that had no prior way to enumerate tournament IDs.
- Command-line/CLI users see no behavior change beyond the parser accepting more
  input shapes; the raw-ID surface they rely on for scripting is deliberately left
  alone.
