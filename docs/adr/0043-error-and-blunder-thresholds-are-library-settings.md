# Error and blunder thresholds are library settings

## Status

accepted — decided 2026-09-07. Extends 0005 (the daemon has no privileged tenant) and
0007 (what an export carries); reads the glossary terms *Error*, *Blunder* and *Library
setting* added with it.

## Context

Three numbers drew the same line in three places, none of them settable. The statistics
called a decision a blunder at 100 millipoints, in a Go constant of `storage/sqlshared`
read eight times; the status bar's library counter repeated the SQL by hand; and the
study queue an import proposes retained decisions from 50, with the comment "half of what
the statistics call a blunder, because the decisions worth revisiting start well below
the ones worth being ashamed of". "Error", meanwhile, meant *any* non-zero cost, so a
three-millipoint miss was one. The user asked for both lines to be theirs, and for a
weaker "error" under "blunder".

Two things made the obvious answer wrong. A per-machine setting (the XDG config file,
where language and colours live) would make one file count different blunders on two
computers, and the CLI, which never reads that file, would contradict the desktop on the
word. And the per-library precedent, the Performance Rating objective in `metadata`,
turned out to be desktop-only: on the daemon `metadata` is one table for every tenant,
outside Row-Level Security, whose load/save routes were removed after #156.

## Decision

1. **Two thresholds, nested.** An Error is a decision whose cost reaches the error
   threshold; a Blunder is an Error whose cost reaches the blunder threshold. Error ≤
   blunder is enforced on write. Defaults 50 and 100: the blunder line does not move, the
   error line takes the value the study queue already drew, so the queue does not change
   and the "Errors" columns stop counting noise — one changelog line says so. XG (20/80)
   and gnubg (40/80) are offered as named presets, never as the truth.

2. **They are Library settings**, behind one accessor of the `storage` contract. SQLite
   keeps them in `metadata`, next to the objective; PostgreSQL gets a tenant-scoped
   key/value table under RLS (schema 2.20.0, PostgreSQL side only — the SQLite DDL does
   not change, and `CheckVersion` compares the major). `migrate` copies them; `blunderdb
   edit` sets them and `info` prints them; the daemon exposes them on a tenant-scoped
   `/v1` route. They are **not** in `issuance.CarriedMetadataKeys`: a threshold is the
   owner's reading habit, not a fact of the positions.

3. **One reader.** Every consumer — the statistics, the players' rows, the status bar
   counter, the study queue — reads the accessor; no constant survives. The counter
   promises the set the search it opens returns, so it scores a Position played several
   ways by its largest cost like `E>x` does (#167), instead of the denormalised first
   play.

4. **Where they are set.** A *Library* tab of the configuration window, greyed without
   an open database, which also takes in Vacuum and Repair — actions on the open library
   that sat among machine settings without saying so. Thresholds are entered in equity
   with three decimals, the unit every table shows, and stored in millipoints, the unit
   the command line speaks; the counter's link pre-fills the current value (`s E>80`),
   so the conversion is visible.

## Considered options

- *Per machine, in the config file.* Rejected: the file would count differently on two
  machines and the CLI could not follow.
- *Defaults on the daemon, no tenant setting.* Rejected: a migrated library would stop
  counting the blunders it counted on the desktop, and the web front would contradict it.
- *A third category next to the existing "Errors".* Rejected: three words on every
  screen for two lines.
- *A search token (`blunder`, `erreur`) resolving to the current threshold.* Deferred: a
  saved `E>80` says exactly what it returns (#203), and the token can be added without
  undoing anything if usage asks for it.
- *Keeping the study queue's own 50.* Rejected: three lines for two words, and the
  setting would not touch the list users see most.

## Consequences

- The JSON reason `blunder` in the import report keeps its string (CLI contract); its
  documentation says "error, in the library's sense".
- The error histogram keeps its fixed magnitudes (…50–100, 100+): it shows a distribution,
  not categories, and is documented as such.
- The manual's status bar section no longer names "one hundred millipoints".
