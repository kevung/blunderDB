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
   after `holdLease`, two minutes (a window reloaded mid-write never blocks the switches): a
   write still running after that loses its hold. Nothing follows from it today: the frontend's
   switches wait for `writesInFlight` whatever the backend holds, and the frontend is the only
   one to switch the library of the GUI's handle.
   `RecreateDatabase(path)` (the creation over an existing file) is a Setup at that path through
   the same `beginReplace`: refused while a write holds the library, and the file is erased only
   after the handle let the previous library go and took the file's write lock, so no write
   lands in an erased file and no other instance loses its file. Its `-wal` and `-shm` go with
   it: a journal left beside the new file would be replayed into it.
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
4. **Parity**: the guard has an object only in the GUI, the one mode where the bound library
   changes under a gesture. The method sits on `Database` so that the code is not forked by
   mode; the daemon (`serve`) does not go through `database.Database`, and the CLI never calls
   it.

## Consequences

A new write naming rows by id read before an await or a confirmation goes through
`writingOnLibrary`; nothing else is needed. The sites, exactly:

- `CollectionPanel`: add/remove the board's position, remove the selected or one row, the
  single removal after its confirmation;
- `MatchPanel`: delete a match, repair from the source, swap the players;
- `TournamentPanel`: delete a tournament, swap the players of one of its matches;
- `AnkiPanel`: delete or reset a deck, set a card aside, undo a grade (each grade keeps the mark
  of its library, and the grades of a library left are dropped); `ankiService.addPositionToDeck`;
- every inline edit (`utils/inlineEdit.svelte.js` marks the library when the edit begins and
  hands the mark to `onSave`): a collection's name, a deck's name, a tournament's row and comment,
  a match's row, tournament and comment;
- `LessonEditorModal`: every write of the editor (`after`), the deletions after their confirmation;
- `positionService`: `deletePosition` (marked before its confirmation), `updatePosition`;
- `positionActions`: the board's menu marks once built (studied, Anki deck);
  `sendActions`: add to a collection from the same menu;
- `studyQueueService`: the studied mark, from the queue or the menu;
- `importService.savePositionAndAnalysis`: the analysis and comment of the row just saved;
  `scratchBoard`: the verdict on the saved board; `recurringStudy`: a deck or a collection of
  ids, marked by its caller where the ids were read (`trainingMissed`: before reading the missed
  ones);
- `rolloutService.startRolloutOfSearch`: the ids of the list, after their count and confirmation;
- `transcriptionSave`: finish, abandon, edit a match's transcription; `duelService`: every gesture;
- `directionStore` and `rencontreStore`: every write, through `directed` (marked at the opening);
- `importService`: the merge of a dropped library, refused at commit (`libraryStillOnScreen`).

A write sent at the gesture that reads its ids, with no await before it, goes through
`writingOnScreen`: not sent while a switch is in flight, and a switch begun after it waits for
it in `writesInFlight`. These are the `#` tags saved into the comment (`commandProcessor`), the
comments of `CommentPanel`, a lesson step marked done, a restore or a discard from the trash,
and the rollout of the position on the board. The other writes of the library panels (a
creation, a reorder, a click on a row) send no id read before a wait and carry no guard: a
switch does not wait for them.

A gesture whose write is not sent says nothing more: its store function answers `null`, and the
Direction's `act`, its settings and the creation of a Direction then show no success message
and keep their sheet open, leaving `libraryGroups.writeDropped` in the status bar. An open or a
creation refused before any change (`ErrLibraryAlreadyOpen`, a path refused) leaves the screen
on the library the backend kept (`IsOpenDatabase`), and the screen forgets the library left only
once the backend carries the one coming. The
tests have no backend: `test-setup.js` grants the hold (`stores/libraryHold.js`), and a test of
a refusal makes `HoldLibrary` reject.
