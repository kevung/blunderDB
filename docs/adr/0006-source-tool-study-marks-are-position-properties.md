# Source-tool study marks are a sticky position property, not an auto-filled collection

## Status

accepted

## Context

eXtreme Gammon lets a player flag a decision while reviewing a match — "come back to this
one". The mark is stored per move in the `.xg` file (`MoveEntry.Flagged`,
`CubeEntry.FlaggedDouble`). Until now blunderDB dropped it on import, so a user who had
carefully marked forty positions across a season found no trace of them after importing.

Neither GnuBG (SGF) nor BGBlitz records an equivalent, so today only XG produces the mark.

## Decision

`position` carries a boolean `flagged`, written by the importer and by nothing else. It is
ORed into the stored value like `individually_imported` (ADR-0001), never part of the
Zobrist hash, and no gesture inside blunderDB sets or clears it. The search layer exposes it
as one binary filter (`fl`), which combines freely with every other filter.

A flagged cube decision marks **both** positions blunderDB derives from it — the double and
the take/pass. The source file records one decision where blunderDB stores two sides of it,
and the user marked the moment, not one side of it.

Because the mark lives only in the source file and the match hash is computed from the play
rather than from the file, flagging a position after importing its match does not make the
match look new. `WriteMatch` therefore applies flags **even to an exact duplicate it
otherwise skips entirely** — the one thing a skipped import still writes.

## Considered options

- **An auto-filled "flagged positions" Collection**, as the originating issue proposed.
  Rejected on two grounds. It contradicts the glossary, where a Collection is a set the user
  assembles *by hand* — and that is not pedantry: Collection membership is one of the
  reasons a position survives the orphan purge, so an importer writing into a Collection
  would silently change which positions survive deleting a match. It also has no answer to
  "the user removed a member, should the next import put it back?", a question a derived
  property does not raise. A Collection remains available as a user gesture, built from a
  search, which is what a Collection is for.
- **Storing the mark on `move` instead of `position`**, which is where XG puts it. Rejected:
  it only differs when the same position is flagged in one match and present unflagged in
  another — marginal — while costing an `EXISTS` subquery on every search and making a
  standalone position (no `move` row) impossible to mark.
- **Recomputing the mark on re-import** so unflagging in XG propagates. Rejected: under
  deduplication a position is shared between matches, so "recompute" means letting one
  file clear a mark that came from another — the exact trap ADR-0001 documents.
- **Including the mark in the match hash** so a newly flagged file counts as new. Rejected:
  it would re-import the match as a duplicate.
- **A settings toggle for the whole behaviour**, as the issue asked. Rejected once the
  auto-created Collection was dropped: what remains is reading a field the file already
  carries, exactly as comments, analyses and player names are read without asking. A toggle
  would only create a state where the filter silently returns nothing.

## Consequences

- The mark reads as "this position was, at some point, flagged in the source tool" —
  order-independent, and true.
- **Existing databases are not backfilled.** Unlike `individually_imported`, which could be
  reconstructed from the move graph, nothing inside an already-imported database records a
  source-file mark. Existing positions start unflagged and gain the mark when their match is
  imported again — which is precisely why a skipped duplicate still applies flags.
- A flagged position survives the orphan purge on match deletion, joining Collection
  membership, Anki-card membership and `individually_imported`. Without this, deleting a
  match would delete the very positions the filter exists to surface. The retention
  predicate is stated in three places — both Storage backends and the `Database` wrapper the
  GUI and CLI actually run — and they must not drift.
- The mark cannot be removed from within blunderDB. This is deliberate for a durable "worth
  studying" mark, but it means a flag set by mistake is permanent. A transient "come back to
  this" list is a Collection, not a flag.
- `ingest` re-reads the raw XG segments to recover the marks, because the lightweight
  `xgparser.Match` drops them. The keys mirror `ParseXG`'s own record→Move numbering; any
  divergence would attach a mark to the wrong decision, which is why a test pins the mapping
  against a fixture with known flags.
