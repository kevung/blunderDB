# The trash is a snapshot table, not a deleted_at column

## Status

accepted — implemented in 2.19.0 (`trash` table), issue #285 (fiche I.29).

## Context

blunderDB destroys things on a confirmation dialog and nothing else: twelve
`confirmAction` call sites in the frontend, each irreversible the instant it is
accepted. Deleting a collection, an Anki deck's card, a comment, a position —
all of them are final, and the only recovery is a backup the user may not have.

The reflex implementation of undo is a `deleted_at` column on each table, with
every read filtered to `deleted_at IS NULL`. It is the wrong shape here, for a
reason specific to this code base: **the reads are not in one place**. There are
around fifty search filters composing SQL against `position`; two statistics
back ends (`storage/sqlite/stats_sqlite.go` and its Postgres twin) that
aggregate over it; the retention predicate `positionIsHeldSQL`, written three
times by invariant; the Zobrist uniqueness index that dedup depends on; the
Anki scheduler; the export path. A `deleted_at` column obliges every one of them
to learn about it, and a single one that forgets does not fail loudly — it
silently counts deleted positions in a PR, or hands a deleted position to a
review session.

The Zobrist index makes it worse. `idx_position_zobrist` is UNIQUE and is the
mechanism by which imports deduplicate. A soft-deleted row keeps its hash, so
re-importing the match that position came from would collide with a row the
user believes gone.

## Decision

**Deleting stays a real DELETE, and what was deleted is snapshotted into a
`trash` table first.** One row per deleted thing: `kind` (position, collection,
comment, anki_card), `label` (what the trash list shows), `payload` (everything
needed to put it back, as JSON) and `deleted_at`.

**No live query changes.** Not one of the fifty filters, not the statistics,
not the retention predicate, not the uniqueness index. A trash row is dead
weight that nothing but the trash panel and the purge ever reads. That is the
entire point of the choice.

**Restoring a position re-saves it** through `SavePosition`, which means it
comes back on the same row as before when nothing has taken its Zobrist hash in
the meantime, and merges into the existing row when something has. The
deduplication invariant is not weakened; it is the mechanism by which restore
works at all.

**The purge is `vacuum`'s, and the retention is 30 days.** `blunderdb vacuum`
drops trash rows older than that, as it already drops what else has become
dead. Nothing purges on open.

## Consequences

Undo is cheap to add to a new gesture: snapshot, delete, done. It is also
cheap to *not* add — a gesture that does not snapshot behaves exactly as it did
before, with no half-state.

A restored position does not bring back what cascaded off it. Deleting a
position deletes its analyses and comments by foreign key; the snapshot carries
what the payload holds, and what the payload holds is stated per kind rather
than promised in general. Restoring is "put this back", not "undo the
transaction".

The trash is per database, travels with the file, and is **not** carried by an
export: `issuance.Carried` is an allow-list and `trash` is not on it. Somebody
else's machine has no business receiving what this user deleted.

The table grows with deletions and shrinks on vacuum. A user who deletes a
thousand positions and never vacuums carries a thousand JSON snapshots; that is
disk, not correctness, and `blunderdb verify` reports the count.
