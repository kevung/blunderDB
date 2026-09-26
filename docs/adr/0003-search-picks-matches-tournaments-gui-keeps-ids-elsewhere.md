# GUI search picks matches/tournaments from a list; CLI and command line keep raw IDs

Status: accepted.

## Context

Filtering positions by match or tournament through free-text ID fields asks the user to know
numeric IDs that the search panel never shows. A tournament filter is not a separate query:
the tournament expands to its member match IDs, unioned into the match-ID list
(`getMatchIDsForTournament`).

## Decision

1. The GUI search panel has one filter row, "Matches & Tournaments" (one `filterEnabled`
   flag, standing for both `ma`/`tn` tokens), opening one modal picker with two sections,
   Matches and Tournaments, each text-filterable, with the columns and default sort of
   `MatchPanel`/`TournamentPanel`.
2. `All`/`None` act on the currently filtered subset, not the whole database.
3. Checking a tournament checks its member matches, shown checked-and-disabled; excluding
   one means unchecking the tournament and picking matches individually.
4. The picker always emits the semicolon list on the wire (`ma2;5;9`).
5. `parseFilterIDList` accepts a two-item comma range (`2,7`), a 3+-item comma list and a
   semicolon list; hints and CLI help describe exactly that.
6. Restoring a saved search silently drops IDs that no longer exist and keeps the rest.
7. The picker list is a plain unvirtualized loop, like the panels it mirrors.
8. The command bar (`s ma<ids> tn<ids>`) and the CLI (`--match-ids`, `--tournament-ids`)
   keep raw IDs; `blunderdb list --type tournaments` gives the CLI the same discoverability.

## Consequences

- A new modal (derived from `ExportDatabaseModal`) with sync logic between its sections;
  the search-filter restore path works on ID sets and must resolve tournament membership.
- `parseFilterIDList` only became more permissive: no saved search changes meaning.
- Rejected: fixing only the parser — IDs remain invisible in the panel.
- Rejected: an inline checkbox list — at thousands of matches it fights the panel's height.
- Rejected: two modals — hides the tournament→matches expansion.
- Rejected: per-match exceptions under a checked tournament — hidden override state.
- Rejected: a global `All` — silently selects rows the user never saw.
- Rejected: virtualization now — nothing in the frontend needs it at real sizes.
- Rejected: two independent toggles — one could be checked without its toggle, silently
  dropping half the selection.

## Guard

`pkg/blunderdb/storage/searchfilter/searchfilter_test.go`,
`pkg/blunderdb/database/search_rewrite_test.go`.
