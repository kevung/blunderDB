# Une écriture par ids tient la bibliothèque où ils ont été lus

Status: accepted.
See also: ADR-0101 (library groups), ADR-0055 (context budget)

## Context

A gesture reads rows of the library on screen (a collection's positions, a match, a lesson, a
Direction), often waits (a confirmation, further reads), then writes naming those rows by id.
With several libraries open (ADR-0101) the bound `*Database` changes library on a switch, an
open or a creation, and an id of library A names another row, or none, in library B. The
rollout already refused ids of another group (`StartRolloutIDs(..., group)`), and
`confirmAction` answers no once another library came on screen; every other write had only the
latter, and nothing held once a write was sent.

Two ways were open: give each write method a token of the library read (dozens of signatures,
and the CLI and the daemon would carry a token they never need), or one guard every write goes
through.

## Decision

1. **One control point on `Database`**: `HoldLibrary(path) (ticket, error)` refuses with
   `ErrLibraryReplaced` unless the handle carries the library at `path`, and holds it there:
   `OpenDatabase`, `SetupDatabase` and `Exchange` refuse with a `*BusyError` (`JobWrite`) until
   `ReleaseLibrary(ticket)`. The check and the hold are one step under `mu` and `cancelMu`, so
   no library comes under the write between them. `Close` is never held off, and a hold lapses
   after `holdLease` (a window reloaded mid-write never blocks the switches).
2. **The path names the library**, not the generation: no file is open in two groups, and the
   frontend knows the path synchronously at the moment it reads the rows, where the generation
   would cost a round trip. A file recreated or reopened under the same path is covered by the
   frontend's own count (`libraryReplaced`, now also called after a creation).
3. **One frontend helper**: `libraryMark()` is taken when the rows are read (before the
   confirmation), and `writingOnLibrary(mark, write)` sends the write only if that library is
   still on screen, under the backend's hold and in `writesInFlight` (a switch waits for it). A
   refusal says so in the status bar (`libraryGroups.writeDropped`); the write never runs.
   The Direction keeps its mark from its opening (`directed`); a merge of a dropped library is
   refused at commit if the library changed since its analysis.
4. **Parity**: the hold lives on `Database`, so the CLI and the daemon have it too; they never
   change library under a request and have no use for it today.

## Consequences

A new write naming rows by id goes through `writingOnLibrary`; nothing else is needed. The
tests have no backend: `test-setup.js` grants the hold (`stores/libraryHold.js`), and a test of
a refusal makes `HoldLibrary` reject.
