# blunderDB CLI Documentation

The blunderDB application supports both GUI and command-line interface (CLI) modes in a **single binary**. The CLI provides powerful tools for batch operations, automation, and scripting.

## Building

Build the blunderDB binary using Wails (the `webkit2_41` tag matches
webkit2gtk-4.1 on Arch and ubuntu-latest; drop it for webkit2gtk-4.0 on
ubuntu-22.04, as CI does):

```bash
wails build -tags webkit2_41
```

The binary will be located at `build/bin/blunderDB`.

## Usage

The same binary works for both GUI and CLI modes:
- **GUI Mode**: Run without arguments: `./blunderDB` 
- **CLI Mode**: Provide CLI commands as arguments: `./blunderDB import --db database.db ...`

When you provide a CLI command as the first argument, it automatically runs in headless CLI mode without displaying the frontend.

### Basic Syntax

```bash
./blunderDB <command> [options]
```

### Available Commands

<!-- BEGIN GENERATED COMMAND LIST (cmd/cli-doc-gen; do not edit by hand, run `go run ./cmd/cli-doc-gen`) -->

- `create` - Create a new database with optional metadata
- `import` - Import data into the database (match, position, batch)
- `export` - Export data from the database
- `identity` - Show or move your issuer identity
- `open` - Open a password-protected copy into an ordinary database
- `list` - List database contents
- `search` - Search positions with filters
- `match` - Display match positions and analysis
- `lesson` - Manage lessons (ordered steps showing collections and positions; export them)
- `study` - The study backlog: your unhandled blunders across every import, and the "studied" mark
- `collection` - Manage collections (list, show, create, rename, delete, export)
- `anki` - Spaced-repetition decks (decks, stats, forecast, sync)
- `stats` - Statistics computed apart from list --type stats (recurring)
- `training` - The Training journal (sessions, missed positions)
- `cubematrix` - Cube verdict at every score of a match, for one position
- `epc` - EPC, win probability and money cube verdict (bearoff)
- `rollout` - Roll out a position's plays or cube decision (gammonNet)
- `bearoff` - Generate, list, verify and delete the bearoff tables
- `analyze` - Write a gammonNet analysis for every position missing one
- `duel` - Play a Duel one Action per call (create, show, move, double, take, pass, stop)
- `transcribe` - Replay a .mat, a match or a draft and report its inconsistencies
- `tournament` - Read a directed tournament (list, verify, standings, page, export)
- `players` - Other spellings of a player (alias add, list, remove, suggest)
- `events` - Other spellings of an event (alias add, list, remove, suggest)
- `info` - Display database metadata
- `edit` - Edit database metadata
- `verify` - Verify database integrity
- `vacuum` - Compact the database file, reclaiming freed space
- `repair` - Recompute the analysis columns from the analyses themselves
- `met` - List, import or choose the database's match equity table
- `reencode` - Rewrite analyses stored by older releases in the compact format
- `delete` - Delete data from the database
- `comment` - Comments on a position, signed by their author (add, list)
- `trash` - What was deleted through the trash, and how to put it back
- `completion` - Print a shell completion script (bash, zsh, fish)
- `help` - Show this help message
- `version` - Show version information
- `healthcheck` - Probe a running daemon's /readyz; exit 0 when it is ready
- `mcp` - Serve the database's tools to an AI assistant (Model Context Protocol, stdio)

<!-- END GENERATED COMMAND LIST -->

Use `blunderDB <command> --help` for more information about a command.

## Create Command

Create a new blunderDB database file with optional metadata.

```bash
./blunderDB create --db <path> [options]
```

**Options:**
- `--db` - Path to the database file to create (required)
- `--user` - Set the database owner name
- `--description` - Set a description for the database
- `--force` - Overwrite if the file already exists
- `--format` - Output format: `text` (default) or `json` (path, version, user, description, created)

The `.db` extension is added automatically if missing. Parent directories are created as needed.

**Examples:**
```bash
# Create a new database
./blunderDB create --db mymatches.db

# Create with metadata
./blunderDB create --db mymatches.db --user "John" --description "2025 tournament matches"

# Overwrite existing database
./blunderDB create --db mymatches.db --force
```

**Example output:**
```
Creating database: mymatches.db
Successfully created database with schema version 2.3.0

Database Information:
  Version: 2.3.0
  User: John
  Description: 2025 tournament matches
  Created: 2025-11-03 14:30:00
```

## Import Command

Import match files (.xg, .sgf, .mat, .txt, .bgf, .ogxm) or XGP position files (.xgp) into a database.

### Import Match

```bash
./blunderDB import --db database.db --type match --file match.xg
```

### Import XGP Position

```bash
./blunderDB import --db database.db --type match --file position.xgp
```

XGP files are single-position files exported from eXtreme Gammon. They contain
the position along with its analysis (checker moves and/or cube decisions).

**Options:**
- `--db` - Path to the database file (required)
- `--type` - Import type: `match` or `position` (required)
- `--file` - Path to the file to import (required)
- `--format` - Output format: `text` (default) or `json` (match/position details as one document)

**Example:**
```bash
# Import an XG match file
./blunderDB import --db mymatches.db --type match --file test.xg

# Output:
# Connected to database: mymatches.db
# Importing match from: test.xg
# Successfully imported match (ID: 1)
#
# Match Details:
#   Players: Player1 vs Player2
#   Event: Tournament Name
#   Match Length: 7
#   Games: 15
```

### Import Positions

Import positions from a text file (JSON format, one position per line):

```bash
./blunderDB import --db database.db --type position --file positions.txt
```

**Position file format:**
Each line should be a JSON-serialized Position object.

**Options:**
- `--format` - Output format: `text` (default) or `json` (an `{"imported": N, "failed": N}` summary)
- `--fail-on-error` - Exit non-zero when any line failed, even if others succeeded

Importing nothing at all — every line failed, or the file had no position
lines — is always an error, whatever `--fail-on-error` says: nothing imported
is never a silent success. A **partial** failure (some lines succeeded, some
did not) only fails the run when `--fail-on-error` is passed.

### Batch Import

Import all match files from a directory at once:

```bash
./blunderDB import --db database.db --type batch --dir ./matches/
```

**Options:**
- `--dir` - Path to the directory to scan (required for batch)
- `--recursive` - Recursively scan subdirectories (default: true)
- `--format` - Output format: `text` (default, the summary table below) or `json`
- `--fail-on-error` - Exit non-zero when any file failed to import, even if others succeeded
- `--skip-duplicates` - Skip a match already in the database outright (study marks aside)

Supported file types: `.xg`, `.xgp`, `.sgf`, `.mat`, `.txt`, `.bgf`, `.ogxm`.

A match already in the database is recognised by its play (players, length,
dice, moves, cube), not by its analysis. Its match, game and move rows are
never rewritten, but by default its analyses still reach the stored
positions, and one strictly deeper than the stored entry replaces it: a
Roller++ version of a match replaces the 3-ply one whatever the import order,
a shallower or equal one changes nothing. The line reads `DUPLICATE (N
analyses deepened)` and the report counts those duplicates apart.
`--skip-duplicates` restores the plain skip. A truncated match later
completed (more games) is a different match and is imported as a second one.

A batch that finds no supported file, or where every file failed or was a
duplicate (nothing at all got imported), is always an error. A duplicate is
not a failure by itself: re-running a batch import over a directory that was
already imported, with no new files added, stays a success.

**Examples:**
```bash
# Batch import all files recursively
./blunderDB import --db database.db --type batch --dir ./matches/

# Batch import (non-recursive)
./blunderDB import --db database.db --type batch --dir ./matches/ --recursive=false

# Machine-readable output, failing the run if any file errored
./blunderDB import --db database.db --type batch --dir ./matches/ --format json --fail-on-error
```

**Example output:**
```
Batch importing from: ./matches/ (recursive)

Status  File                  Match ID  Players              Games  Positions
------  ----                  --------  -------              -----  ---------
✓       tournament/match1.xg  1         Alice vs Bob         12     234
✓       tournament/match2.xg  2         Carol vs Dave        8      156
⊘       tournament/match3.xg  —         —                    —      —          (duplicate)
✗       bad_file.xg           —         —                    —      —          (parse error)

Imported: 2 matches, Skipped: 1 duplicates, Failed: 1 errors
```

## Export Command

Export database contents to files.

### Export Entire Database

```bash
./blunderDB export --db database.db --type database --file export.db
```

This creates a complete copy of the database including all positions, analyses, matches, and metadata.
A Duel in suspense stays behind: its dice seed is every roll to come, and it leaves by no route
before the Duel ends. A finished Duel travels as its Match, seed revealed.

### Export Positions

Export all positions to a JSON text file:

```bash
./blunderDB export --db database.db --type positions --file positions.txt
```

Each position is exported as a JSON object on a separate line.

**Options:**
- `--db` - Path to the source database file (required)
- `--type` - Export type: `database`, `positions`, `matches`, or `mat` (required)
- `--file` - Path to the output file (required for all types except `mat`, where `--file` or `--dir` is required)
- `--dir` - Output directory for `mat` batch export (one auto-named `.mat` per match)
- `--analysis` - Include analysis in database export (default: true)
- `--comments` - Include comments in database export (default: true)
- `--filters` - Include filter library in database export (default: true)
- `--played-moves` - Include played moves in analysis (default: true)
- `--matches` - Include matches in database export (default: true)
- `--collections` - Include collections in database export (default: false)
- `--collection-ids` - Comma-separated collection IDs to export
- `--match-ids` - Comma-separated match IDs to export (empty = all)
- `--tournament-ids` - Comma-separated tournament IDs to export
- `--format` - Output format: `text` (default) or `json` (a summary document — file path, byte count, counts)

### Export Database Without Matches

```bash
./blunderDB export --db database.db --type database --file export.db --matches=false
```

This creates a copy of the database with positions, analyses, and comments, but without match data.

### Export Matches Only

Export only match data (with linked positions) to a new database:

```bash
./blunderDB export --db database.db --type matches --file matches.db
```

This creates a new database containing only the match structure and linked positions.

### Export Matches as .mat Transcripts

Export one or more matches as Jellyfish/gnubg `.mat` text transcripts (the format XG re-imports). A `.mat` file holds exactly one match, so:

- Use `--file` to export a single match (selected with `--match-ids`) to that exact path:

```bash
./blunderDB export --db database.db --type mat --match-ids 5 --file game.mat
```

- Use `--dir` to export several matches (or all matches, when `--match-ids` is omitted) as auto-named files into a directory:

```bash
./blunderDB export --db database.db --type mat --match-ids 5,9,12 --dir out/
./blunderDB export --db database.db --type mat --dir out/
```

Auto-named files follow the scheme `Player1_Player2_YYYY-MM-DD_Np.mat` (money games use `unlimited` instead of `Np`); the match id is appended on a name collision. Passing `--file` with more than one match is an error. Analysis and comments are not part of the `.mat` format (it is a pure move transcript).

## Marking and protecting an export

`export` can do two extra, independent things, both optional and freely combined:

- `--watermark "<origin>"` writes a **signed statement of where the file comes from** into it, with `--watermark-note` for free text (terms of use, a contact address).
- `--password <pw>` wraps the result in an encrypted container (`.dbx`).

```bash
./blunderDB export --db cours.db --type database --file cours-diffusion.dbx \
    --watermark "Cours de Jean Dupont — 12 mars 2026" \
    --watermark-note "Merci de ne pas rediffuser." \
    --password secret
```

**What a watermark is.** It is signed with your issuer identity, so it is **tamper-evident and unforgeable**: nobody can alter it, and nobody can fabricate one in your name. It is **not unremovable** — the file is a plain SQLite database and blunderDB is free software — and it prevents nothing. It says where the file came from.

A protected export is always named `.dbx`: if you pass `--file cours.db --password …`, the file is written as `cours.dbx`. blunderDB recognises a protected file by its contents rather than its name, but a `.db` file holding encrypted bytes misleads every other tool you own.

The container is **AES-256-GCM**, with the key derived from the password by **Argon2id** (64 MiB, 3 passes, 4 lanes) and a salt drawn per file. GCM authenticates the payload, so a wrong password is rejected instead of producing a corrupt database, and it is checked on every open.

**What a password protects.** The file *in transit*: the stray copy in a downloads folder, the attachment forwarded by mistake. Not the database — whoever you gave the password to can open it. The container's header is cleartext, so `blunderdb info` reads the origin without the password.

**What neither does.** Nothing is tracked. blunderDB records nothing on the recipient's side: no registry of who opened a file, no log, no trace carried into a database that imports one. See `docs/adr/0007-watermarks-mark-origin-and-nothing-else.md`.

## Identity Command

Show or move your **issuer identity** — the Ed25519 key every watermark is signed with. It is created by itself the first time you watermark a file; there is nothing to set up. It belongs to a person, not to a database, so everything you mark carries one public fingerprint.

```bash
./blunderDB identity                                        # show name and fingerprint
./blunderDB identity --name "Jean Dupont"                   # change the display name
./blunderDB identity --export jean.bdbid --passphrase pw    # carry it to another machine
./blunderDB identity --import jean.bdbid --passphrase pw
./blunderDB identity --format json                          # name, fingerprint, storage path
```

The exported file lets anyone holding it sign in your name — do not share it. The passphrase is optional and applies only to that transferred file; the local one is deliberately unprotected, so an ordinary user never meets a secret they did not ask for.

Renaming changes only a label: files already marked keep the name they were sealed with, and keep verifying.

## Open Command

Turn a password-protected file (`.dbx`) into an ordinary database. The password is asked for once; from then on it is a normal file.

```bash
./blunderDB open --db cours.dbx --password secret
./blunderDB open --db cours.dbx --password secret --file ./mon-cours.db
```

**What the password protects:** the *transport* of the file — the stray copy in a downloads folder, the attachment forwarded by mistake. Not the database: whoever the password was given to can open it.

The container's header is **cleartext**, so `blunderdb info` reads a protected file's origin without its password.

## Search Command

Search for positions in the database using filters.

```bash
./blunderDB search --db database.db [options]
```

**Options:**
- `--db` - Path to the database file (required)
- `--export` - Export results to a new database file
- `--limit` - Maximum number of results (0 = no limit)
- `--format` - Output format: `table`, `json`, `xgid` (default: table)
- `--decision` - Filter by decision type: `checker`, `cube`
- `--dice` - Filter by dice roll. Use `5,3` to match positions where both dice were rolled (any order); use `5` to match positions where a 5 appeared on either die. Implies `--decision checker` when no decision flag is set.
- `--pip-min` / `--pip-max` - Pip count difference range
- `--winrate-min` / `--winrate-max` - Win rate range (%)
- `--cube` - Filter by cube value
- `--score1` / `--score2` - Filter by player scores
- `--match-length` - Filter by match length
- `--error-min` - Minimum equity error
- `--move-error-min` / `--move-error-max` - Played move error range (millipoints)
- `--has-analysis` - Only positions with analysis
- `--off1-min` / `--off2-min` - Minimum checkers off for player 1/2
- `--individual` - Only positions imported on their own — the ones you added yourself, not the ones a match import brought in
- `--flagged` - Only positions you marked for study in the source tool (eXtreme Gammon flags). Not backfilled: existing matches must be imported again to deliver their marks
- `--has-comment` - Only positions carrying a comment. Origin is not recorded, so a note you typed and one a match import lifted from the source file both count. Match and tournament comments are not consulted
- `--no-comment` - Only positions carrying no comment. Mutually exclusive with `--has-comment`
- `--match-ids` - Filter by match IDs: comma-separated list e.g. `1,3,5`, OR a two-value range e.g. `2,7` (2 through 7), OR a semicolon list e.g. `2;7`
- `--tournament-ids` - Filter by tournament IDs: comma-separated list e.g. `1,3,5`, OR a two-value range e.g. `2,7` (2 through 7), OR a semicolon list e.g. `2;7`
- `--position-ids` - Filter by position IDs: a two-value range e.g. `2,7` (2 through 7), OR an explicit semicolon list e.g. `5;10;15`

### Examples

```bash
# Search cube decisions
./blunderDB search --db database.db --decision cube

# Search positions with errors >= 0.1
./blunderDB search --db database.db --error-min 0.1

# Search in specific matches (2, 5, and 9)
./blunderDB search --db database.db --match-ids 2,5,9

# Search in a tournament
./blunderDB search --db database.db --tournament-ids 1

# Search positions where dice were 6-5 (either order)
./blunderDB search --db database.db --dice 6,5

# Search positions where a 6 was rolled on either die
./blunderDB search --db database.db --dice 6

# Search and export to new database
./blunderDB search --db database.db --decision cube --export cubes.db

# Output as JSON
./blunderDB search --db database.db --format json --limit 10

# Positions flagged for study in XG
./blunderDB search --db database.db --flagged

# Every commented position
./blunderDB search --db database.db --has-comment

# Blunders still waiting to be annotated
./blunderDB search --db database.db --no-comment --error-min 0.1
```

## List Command

Display database contents and statistics.

### List Matches

```bash
./blunderDB list --db database.db --type matches
```

Shows all imported matches with details:
- Match ID
- Player names
- Event information
- Location
- Match length
- Number of games
- Import date
- Source file path

**Example output:**
```
Found 2 match(es):

ID: 1
  Players: Player1 vs Player2
  Event: World Championship
  Location: Monte Carlo
  Match Length: 25
  Games: 48
  Imported: 2025-11-03 14:30:00
  File: /path/to/match1.xg

ID: 2
  Players: Player3 vs Player4
  Match Length: 7
  Games: 12
  Imported: 2025-11-03 15:45:00
  File: /path/to/match2.xg
```

### List Tournaments

```bash
./blunderDB list --db database.db --type tournaments
```

Shows all tournaments with details:
- Tournament ID
- Name
- Date
- Location
- Number of matches

**Example output:**
```
Found 2 tournament(s):

ID: 1
  Name: World Championship
  Date: 2026-01-01
  Location: Monte Carlo
  Matches: 5

ID: 2
  Name: Marseille Open
  Date: 2026-03-15
  Matches: 3
```

### List Positions

```bash
./blunderDB list --db database.db --type positions --limit 20
```

Shows position details:
- Position ID
- Score
- Player on roll
- Decision type (checker play or cube action)

**Options:**
- `--limit` - Maximum number of items to display (default: 10)
- `--offset` - Number of positions to skip before listing; only that window is read

### Show Database Statistics

```bash
./blunderDB list --db database.db --type stats
```

Displays comprehensive performance statistics: PR/MWC metrics, Snowie Error Rate, rolling performance, top blunders, cube-action breakdown, and an error histogram.

**Options (stats-specific):**
- `--metric pr|mwc` — Metric displayed in the text report (default: `pr`). `mwc` shows WC-loss values; money-game positions show `—`.
- `--player <name>` — Restrict to decisions where the named player is on move.
- `--tournament <id[,id,…]>` — Restrict to one or more tournament IDs (comma-separated).
- `--from <YYYY-MM-DD>` — Include only matches on or after this date.
- `--to <YYYY-MM-DD>` — Include only matches on or before this date.
- `--decision-type all|checker|cube` — Restrict to a decision kind (default: `all`).
- `--top-blunders N` — Number of top blunders listed (default: 10).
- `--format text|json` — Output format (default: `text`). `json` marshals the full `StatsResult` struct.

**Examples:**

```bash
# Basic text report
./blunderDB list --db database.db --type stats

# MWC metric with player filter
./blunderDB list --db database.db --type stats --metric mwc --player "Alice"

# Checker-play only, last 6 months
./blunderDB list --db database.db --type stats \
  --decision-type checker --from 2025-01-01

# Machine-readable JSON for scripting
./blunderDB list --db database.db --type stats --format json
```

**Text output sections:**

1. **Header** — DB path, active filters, chosen metric.
2. **Totals** — positions, matches, tournaments, decisions.
3. **PR / Snowie ER / MWC** — global, checker, and cube values. PR counts only unforced checker plays and close cube decisions (seuil 0.16 d'équité), aligned with eXtreme Gammon. Snowie ER uses the same error numerator but divides by the total moves of both players (forced included).
4. **Rolling PR / MWC** — values for N = 5, 10, 50, 100, 250, 500, 1000 most-recent decisions.
5. **Top N Blunders** — position ID, type, error in EMG, MWC loss, date, players.
6. **Cube Action Breakdown** — per action: decisions, blunders, blunder %, PR, MWC.
7. **Error Histogram** — decision counts by error-magnitude bucket (0–0.005 … ≥0.1 EMG).

**JSON output fields** (top-level):

| Field | Type | Description |
|---|---|---|
| `totals` | object | `num_positions`, `num_matches`, `num_tournaments`, `num_decisions` |
| `pr_global` | float | Global PR (unforced checker + close cube decisions) |
| `pr_checker` | float | Checker-play PR (unforced moves only) |
| `pr_cube` | float | Cube-action PR (close cube decisions only) |
| `snowie_global` | float | Snowie Error Rate (same error sum, denominator = total moves of both players) |
| `pr_rolling` | object | Rolling PR keyed by N (5 … 1000) |
| `mwc_global` | float | Total MWC loss (match-play decisions) |
| `mwc_available` | bool | `false` for money-game-only data sets |
| `per_tournament` | array | Per-tournament PR and MWC |
| `per_match` | array | Per-match PR and MWC |
| `cube_action_breakdown` | array | Per cube action stats |
| `error_histogram` | array | Bucket counts |
| `top_blunders` | array | Top blunder entries |

### Show One Row Per Player

```bash
./blunderDB list --db database.db --type players
```

Prints a comparison table with one row per player in the database — matches, wins/losses, counted decisions, global/checker/cube PR, Snowie Error Rate, errors, blunders and luck. This is the command-line half of the Stats panel's Players tab.

A row is keyed by a player **name exactly as it appears in the matches**, so someone who signed under two spellings gets two rows.

**Options (players-specific):**
- `--from <YYYY-MM-DD>` / `--to <YYYY-MM-DD>` — Restrict to matches in a date range, e.g. the days of a tournament.
- `--tournament <id[,id,…]>` — Restrict to one or more tournament IDs.
- `--format text|json|csv` — Output format (default: `text`).

`--player` and `--decision-type` are **not** applied here: the table covers every player, and it already splits checker from cube into separate columns.

**Examples:**

```bash
# Ranking over a competition's dates
./blunderDB list --db database.db --type players --from 2026-03-01 --to 2026-03-08

# CSV, for a spreadsheet or a script
./blunderDB list --db database.db --type players --format csv
```

**Reading the output:** a `—` (an empty field in CSV) marks a figure that was never measured, which is not the same as zero. Luck in particular is only available for matches imported since database schema 2.15.0, and only from formats that carry it (XG, gnuBG — not BGF or Jellyfish `.mat`). Re-importing the source file is not enough — the import recognises a duplicate and applies only the study marks it newly raises (the report says `DUPLICATE (N study marks applied)`): delete the match, then import it again. The `luck_rolls` column says how many rolls the average covers.

**CSV columns:** `player`, `matches`, `wins`, `losses`, `decisions`, `pr`, `pr_checker`, `pr_cube`, `snowie_er`, `errors`, `blunders`, `luck_rate_mp`, `luck_rolls`.


## Delete Command

Remove data from the database.

### Delete Match

```bash
./blunderDB delete --db database.db --type match --id 1 --confirm
```

Deletes a match and all associated data (games, moves, analyses). Without `--confirm`, the command will prompt for confirmation.

**Options:**
- `--db` - Path to the database file (required)
- `--type` - Delete type: `match` (required)
- `--id` - ID of the item to delete (required)
- `--confirm` - Skip confirmation prompt (optional)
- `--format` - Output format: `text` (default) or `json` (`{"match_id": N, "deleted": true}`)

**Example:**
```bash
# Delete with confirmation prompt
./blunderDB delete --db database.db --type match --id 1

# Output:
# Match ID: 1
#   Players: Player1 vs Player2
#   Event: Tournament
#   Games: 15
#
# Are you sure you want to delete this match? (yes/no): yes
# Successfully deleted match ID 1

# Delete without prompt
./blunderDB delete --db database.db --type match --id 1 --confirm
```

## Match Command

Display match positions and analysis data.

```bash
./blunderDB match --db database.db --id <match_id> [options]
```

**Options:**
- `--db` - Path to the database file (required)
- `--id` - Match ID to display (required)
- `--format` - Output format: `json`, `text`, or `summary` (default: json)
- `--output` - Output file path (default: stdout)

**Examples:**
```bash
# Display match as JSON
./blunderDB match --db database.db --id 1

# Display match summary
./blunderDB match --db database.db --id 1 --format summary

# Save match data to a file
./blunderDB match --db database.db --id 1 --format text --output match1.txt
```

**Example output (summary):**
```
Match: Alice vs Bob
  Match Length: 7
  Games: 12
  Total Positions: 234

  Game 1: 18 positions
  Game 2: 22 positions
  ...
```

**Example output (text):**
```
Position 1 [Game 1, Move 1]
  Player on roll: Player 1 (X)
  Score: 0-0
  Cube: 1 (centered)
  Dice: 3-1

Position 2 [Game 1, Move 2]
  Player on roll: Player 2 (O)
  Score: 0-0
  Cube: 1 (centered)
  Dice: 6-4
  ...
```

## Collection Command

Manage collections — the hand-picked sets of positions of the GUI's
Collections panel. Every sub-command takes `--db`; `list` and `show` print
`text` (default), `json` or `csv` like `list`.

```bash
./blunderDB collection <sub-command> [options]
```

**Sub-commands:**
- `list` - List collections: id, name, number of positions, description
- `show --id <id>` - List the positions of one collection: id, 1-based index in
  the database (the number the GUI's status bar shows), score, decision type
  and XGID
- `create --name <name> [--description <text>]` - Create an empty collection
- `rename --id <id> --name <name> [--description <text>]` - Rename a collection
  (the description is kept unless given)
- `delete --id <id> [--confirm]` - Delete a collection; its positions stay in
  the database
- `export --id <id[,id…]> --out <file.db> [--analysis=false] [--comments=false]
  [--watermark <text>] [--watermark-note <text>]` - Export one or more
  collections to a new database file, the same call the GUI's export dialog
  makes (see *Marking and protecting an export*)

The XGID printed by `show` is the one stored with the position's analysis when
there is one (BGF and XGP imports); otherwise it is generated from the board
exactly as the GUI's *Copy position* does — the match length is then the
larger away score, since a stored position does not retain the real one.

**Examples:**
```bash
# All collections
./blunderDB collection list --db database.db

# Positions of collection 3, as CSV for a spreadsheet
./blunderDB collection show --db database.db --id 3 --format csv

# Create, rename, delete
./blunderDB collection create --db database.db --name "Blitz openings"
./blunderDB collection rename --db database.db --id 3 --name "Openings"
./blunderDB collection delete --db database.db --id 3 --confirm

# Export two collections, marked with their origin
./blunderDB collection export --db database.db --id 3,4 --out openings.db \
    --watermark "Cours de Jean Dupont - 12 mars 2026"
```

**Example output (`show`):**
```
Collection 1: Openings
  Positions: 2

ID  Index  Score  Type  XGID
--  -----  -----  ----  ----
12  12     7-7    cube  -a-B-aD-C---cD---cbeB-----:0:0:1:00:0:0:0:7:0
40  40     7-7    cube  --BEBBB----a--b--cbbBbba--:0:0:1:00:0:0:0:7:0
```

## Study Command

The study backlog: the reference player's blunders, across every import, that
nothing has dealt with yet — no comment, no Anki card, in no collection, no
"studied" mark. Every sub-command takes `--db`.

```bash
./blunderDB study <sub-command> [options]
```

**Sub-commands:**
- `queue [--limit <n>] [--format text|json]` - The backlog, costliest first,
  at most 50 positions
- `mark --id <id>` - Mark a position studied: it leaves the backlog. The mark is
  the user's own data and is never exported
- `unmark --id <id>` - Withdraw the mark: the position returns to the backlog

```bash
./blunderDB study queue --db database.db --limit 20
./blunderDB study mark --db database.db --id 1234
```

## Lesson Command

Manage lessons — ordered steps a coach writes once for a student. Each step has
a title, a text and may show a collection, a position, both or neither. A lesson
travels in an exported database (`export`, `.dbx` with `--password`) and is read
in the GUI with the `:le` command. Every sub-command takes `--db`; `list` and
`show` print `text` (default) or `json`.

```bash
./blunderDB lesson <sub-command> [options]
```

**Sub-commands:**
- `list` - List lessons: id, name, number of steps
- `show --id <id>` - The steps of one lesson, in order
- `create --name <name> [--description <text>]` - Create an empty lesson
- `edit --id <id> [--name <name>] [--description <text>]` - Rename a lesson or
  change its description
- `delete --id <id> --confirm` - Delete a lesson and its steps, for good (there
  is no trash for lessons); the collections and positions its steps showed stay
- `add-step --lesson <id> [--title <t>] [--text <t> | --text-file <f>]
  [--collection <id>] [--position <id>]` - Append a step
- `edit-step --lesson <id> --step <id> [...]` - Change a step; only the given
  fields change, `--collection 0` / `--position 0` clear what it showed
- `remove-step --lesson <id> --step <id>` - Remove a step
- `reorder --lesson <id> --steps <id,id,…>` - Set the order of the steps
- `export --id <id[,id…]> --out <file> [--password <pw>] [--watermark <text>]` -
  Export lessons with the collections and positions their steps show

Importing a file that holds a lesson creates it with its steps; a lesson whose
name already exists is left untouched, so importing the same file twice changes
nothing. Reading a lesson records nothing on the reader's side.

**Examples:**
```bash
./blunderDB lesson create --db database.db --name "Playing against a prime"
./blunderDB lesson add-step --db database.db --lesson 1 \
    --title "Timing" --text "Count the pips." --collection 3
./blunderDB lesson export --db database.db --id 1 --out lesson.dbx \
    --password secret --watermark "Course of 12 March"
```

## Stats Command

Group the errors of a filter by plan of play and theme, costliest first — the
*Recurring errors* table of the Stats panel's Errors tab. The global statistics
stay under `list --type stats`.

```bash
./blunderDB stats recurring --db <file> [options]
```

**Options:** `--player`, `--tournament`, `--from`, `--to` and
`--decision-type` filter as `list --type stats` does; `--limit <n>` bounds the
groups shown in text (default 20, `0` for all); `--format json` carries each
group with the full list of its positions.

A checker theme is `gammon`, `blots`, `point` or `passive` (the rules of the
Analysis panel's explanation sentence); a cube theme is the direction of the
cube error: `offer_missed`, `offer_premature`, `answer_wrong_pass`,
`answer_wrong_take`. The errors no rule names stay out of the ranking, listed
apart with one line per plan of play (`Unthemed` in JSON): the explanation
rules only speak from 60 mp, above the Error threshold. An error is a
counted decision costing at least the library's Error threshold; `COST (PR)` is
the share of the filter's PR the group accounts for.

```bash
./blunderDB stats recurring --db database.db --player "Alice"
./blunderDB stats recurring --db database.db --decision-type checker --format json
```

## Training Command

Read back the journal of the Training tab and turn the missed questions into
study material. The journal is written by the GUI, or by a daemon client over
`training.save`; the CLI asks no question itself.

```bash
./blunderDB training sessions --db <file> [--exercise <name>] [--limit <n>] [--format json]
./blunderDB training missed --db <file> [--session <id>] [--deck <name>] [--collection <name>]
```

`training sessions` lists the sessions, most recent first, with the id
`--session` takes; the PR exists for the Decision exercise only.
`training missed` returns the positions answered wrong, each once, the most
recently missed first — a question that ran out of time counts as missed; only
Decision questions keep their position. `--deck` makes an Anki deck of them,
`--collection` a collection.

```bash
./blunderDB training missed --db database.db --session 12 --deck "Monday's misses"
./blunderDB training missed --db database.db --collection "My misses" --format json
```

## Comment Command

Several people can annotate the same database: each comment is signed by whoever
wrote it, so a coach's notes and a student's questions stay apart. `--author`
signs what `add` writes (without it, the comment is unsigned); on `list` it keeps
one author's comments, whole name and any case, as the search's `au"…"` token
does. An import keeps the source's signatures and never signs in the importer's
name.

```bash
./blunderDB comment add --db base.db --position 412 --text "Cube too early" --author Alice
./blunderDB comment list --db base.db --author alice --format json
```

## Anki Command

Inspect and maintain the spaced-repetition (FSRS) decks of the GUI's Anki
panel. Reviewing a card needs the board and stays in the GUI; the CLI lists,
measures and resynchronises.

```bash
./blunderDB anki <sub-command> [options]
```

**Sub-commands:**
- `decks [--format text|json|csv]` - List decks with their source, card count,
  cards due and new cards
- `stats --deck <id> [--format text|json]` - Review statistics of one deck:
  total, new, learning, review due, due now, plus its FSRS parameters
- `forecast [--deck <id>] [--days <n>] [--format text|json|csv]` - Cards coming
  due per calendar day (UTC) over the next `n` days (default 30, max 365); day
  0 holds every overdue card; `--deck 0` (the default) covers every deck
- `sync --deck <id>` - Add a card for every position of the deck's source that
  has none yet; existing cards keep their scheduling state

A deck built from a collection re-reads its collection. A deck built from a
search stores the search as the GUI saved it (command, board and the position
ids found at the time): the search grammar lives in the GUI, so the CLI
resynchronises from the stored ids and says so on stderr — open the deck in
the GUI to re-run the search itself.

**Examples:**
```bash
./blunderDB anki decks --db database.db
./blunderDB anki stats --db database.db --deck 2 --format json
./blunderDB anki forecast --db database.db --deck 2 --days 14
./blunderDB anki sync --db database.db --deck 2
```

**Example output (`forecast`):**
```
Day         Due
---         ---
2026-09-02  12
2026-09-03  4
2026-09-04  0
...

37 card(s) due over 14 day(s)
```

## Cubematrix Command

Print the cube verdict for one position at **every score** of a 5-, 7- or
9-point match, so a decision can be read against the score rather than at money
play alone. The position comes in as an XGID or an OGID; nothing is written to a database,
and the command works without one.

```bash
./blunderDB cubematrix [options] <XGID|OGID>
```

**Options:**
- `--match-length` - Match length: `5` (default), `7` or `9`
- `--format` - Output format: `text` (default) or `json`

Equities are **normalised** (gnubg's `mwc2eq`, ±1 = winning/losing the current
cube), the one scale that leaves the engine at a match score — see ADR-0019.

**Example:**
```bash
./blunderDB cubematrix 'XGID=-b----E-C---eE---c-e----B-:0:0:1:00:0:0:0:7:10'
./blunderDB cubematrix --match-length 5 --format json '<XGID>'
```

## EPC Command

Compute the Effective Pip Count, the win probability and the money cube
verdict for a bearoff position given as an XGID or an OGID. Pure computation: no
database file is involved.

```bash
./blunderDB epc [options] '<XGID|OGID>'
```

**Options:**
- `--format` - Output format: `text` or `json` (default: text)
- `--bearoff-ts` - Optional two-sided bearoff database (`.bd`) widening the
  embedded TS-06-06 (also read from the `BLUNDERDB_TS_PATH` environment
  variable). The widest valid database wins; an invalid file is ignored
  with a warning.

**Regimes.** Inside the two-sided database domain the win probability and
the money cube analysis (cubeless, ND, D/T, D/P, verdict) are **exact**.
Outside it, the win probability is **estimated** (convolution of the
one-sided roll distributions plus a calibrated correction) and printed with
its measured error bound; the cube verdict is deliberately never estimated
(ADR-0009).

**Examples:**
```bash
# Exact regime (both players within 6 checkers)
./blunderDB epc 'XGID=-BBB------------------bbb-:0:0:1:00:0:0:0:0:10'

# With the downloaded TS-06-11 (exact up to 11 checkers per player)
./blunderDB epc --bearoff-ts ~/.local/share/blunderdb/gnubg_ts6x11.bd 'XGID=…'
```

## Rollout Command

Roll a position out with gammonNet, to settle what a search cannot: two plays a
few thousandths apart, or a cube decision the cube model is unsure of. With
dice on the position, its plays are rolled out (the best few at the rollout's
ply, at least 2, or the ones named with `--move`); without dice, its cube decision (No
double and Double/Take; Double/Pass is +1 exactly). Pure computation: nothing
is stored and no database is needed.

```bash
./blunderDB rollout [options] '<XGID|OGID>'
```

**Options:**
- `--preset` - `fast` (default: 216 games, truncated at 7 half-moves, stop at
  JSD 3 after 108) or `standard` (1296 games, truncated at 11, stop at JSD 3
  after 324). Both play at 0 ply — gammonNet's network alone, for plays, cube
  actions and leaves; `--ply 1` or more plays deeper, at several times the
  cost. Every flag below overrides the preset
- `--games`, `--min-games`, `--truncation`, `--jsd`, `--ply`, `--candidates` -
  the rollout's own parameters (`--truncation 0` plays every game to its end,
  `--jsd 0` never stops early)
- `--move` - a play to roll out, in blunderDB notation (repeatable)
- `--seed` - dice seed (fixed by default: the same command prints the same numbers)
- `--jobs` - games played at once (one per core by default); never changes the numbers
- `--format` - `text` (default) or `json`

Every candidate plays the same dice, the luck of each roll is taken out of each
game (variance reduction), the first two rolls are stratified, and a game stops
where the two-sided bearoff table covers it. Each line gives the equity, its
95 % interval, the games played and the JSD — the gap to the best in standard
deviations of the difference. The cube is played inside the games (cubeful), so
the ranking is more reliable than the absolute equity. Equities are money points
per unit of cube, or normalised equity at a match score (ADR-0019). Ctrl-C
prints what the finished games concluded. The choices are ADR-0060's.

**Examples:**
```bash
./blunderDB rollout 'XGID=-b----E-C---eE---c-e----B-:0:0:1:31:0:0:0:0:10'
./blunderDB rollout --move '8/5 6/5' --move '24/23 13/10' '<XGID>'
./blunderDB rollout --preset standard --format json '<XGID>'
```

## Bearoff Command

Generate, list, verify and delete the bearoff databases the EPC engine and the
race evaluator read. The tables are **generated on the machine that needs them**
rather than shipped in the binary or downloaded (ADR-0027); a generated table is
byte-identical to gnubg's `makebearoff` output for the same domain, and that is
what `verify` checks.

```bash
./blunderDB bearoff <subcommand> [options]
```

**Subcommands:**
- `list` - The tables present, with their domain, size and checksum
- `generate` - Build a table; the two-sided sweep is resumable through a `.ckpt`
- `verify` - Check a table against the recorded SHA-256 for its domain
- `delete` - Remove a generated table

Run `./blunderDB bearoff <subcommand> --help` for the flags of each: the
generated flag reference below carries them verbatim.

**Example:**
```bash
./blunderDB bearoff list
./blunderDB bearoff verify
```

## Analyze Command

Write a gammonNet analysis for every position that has none — the catch-up
sweep for a library built before this feature existed (ADR-0013, ADR-0015).
The same operation as the GUI's automatic post-import analysis and its
explicit "analyze now" button, and as `serve`'s `/v1/gammonnet.analyzeMissing`
for a tenant.

With `--stale`, the command instead re-runs gammonNet on every position whose
stored analysis is entirely its own but was written at an older engine
version, or at a different depth than `--ply` now asks for (#191) — the GUI's
"re-analyse stale positions" button and `serve`'s `/v1/gammonnet.sweepStale`.
A position carrying any XG, GNUbg or BGBlitz analysis is never touched by
either sweep (ADR-0013's protection is unconditional).

```bash
./blunderDB analyze --db database.db [options]
```

**Options:**
- `--ply` - Search depth (default: 2, the canonical parameter)
- `--prune-k` - Pruning width (default: 12, the canonical parameter)
- `--candidates` - Candidate moves kept per checker decision (default: 10)
- `--jobs` - Positions analysed in parallel (default: the number of CPUs)
- `--stale` - Re-analyse positions whose gammonNet analysis is outdated,
  instead of filling gaps
- `--match` - Restrict the sweep to the positions of one match (0, the
  default, means the whole library)
- `--format` - Output format: `text` (default, progress lines) or `json` (a single summary document, printed at the end)

**Parallelism (`--jobs`).** The positions of a sweep are independent — no
search informs the next — so they are spread over `--jobs` goroutines, each
holding one reused searcher. The analyses written are identical whatever
`--jobs` says, bit for bit; only the wall-clock time changes. Use `--jobs 1`
to leave the machine free for something else. Cancellation is unaffected:
Ctrl-C stops the run before any further position is started, and everything
already computed is still written.

**The gap rule (ADR-0013).** A position already carrying any analysis — from
XG, GNUbg, BGBlitz, or a prior gammonNet run — is left untouched, regardless
of which engine is missing. Only a position with no analysis at all is
written. This makes the command safe to re-run at any time, and safe to
interrupt: Ctrl-C cancels cleanly, nothing already written is lost, and the
next run picks up exactly where the last one left off — no journal needed,
because "positions with no analysis" is re-derived fresh every time.

**One match only (`--match`).** With the id `list --type matches` prints, the
sweep is restricted to the positions that match walks through — the same gap
rule, the same guarantees, a narrower scope. A match that has just been
imported gets its analyses without the rest of the library being swept, and a
match corrected and analysed a second time costs only the positions the
correction created, since everything else already carries an analysis. It
combines with neither `--stale` nor `--compare`, which both look at positions
that already have an analysis; asking for both is an error rather than a
silently ignored scope.

**Refused, not failed.** A position gammonNet declines to evaluate — a match
score beyond its MET's horizon, a cube decision the model refuses — reports
"refused" in the end-of-run summary, not "failed": it is not retried to no
effect, because the same request would be refused again every time. The
summary always breaks the run down as `evaluated / refused / failed`; only
`failed` positions (a read, evaluation or write error unrelated to
refusal) are retried on the next run.

**Example:**
```bash
./blunderDB analyze --db database.db
# Analyzing 1204 position(s) with gammonNet (2-ply, k=12, 16 job(s))...
#   1/1204 (0%)
#   61/1204 (5%)
#   ...
#   1204/1204 (100%)
# Done.
# evaluated: 1198, refused: 6, failed: 0

# One core only, on a machine that has other work to do
./blunderDB analyze --db database.db --jobs 1

# One match only, the one just imported
./blunderDB analyze --db database.db --match 12

# Re-analyse everything gammonNet wrote at an outdated version or depth,
# at 3-ply
./blunderDB analyze --db database.db --stale --ply 3
```

## Transcribe Command

Replay a transcription and report what the replay finds. The engine is the pure
`transcript` package the transcription panel already runs, so this command adds
no rule of its own: it is the CLI's window onto it.

The source is one of three, and exactly one: a `.mat` file, a match of a
library, or a transcription draft being typed. A match has no document of its
own — a match is what a transcription *produces* — so it is replayed through the
`.mat` it would export, which means what is checked is what an export of it
would contain.

```bash
./blunderDB transcribe --mat <file> [--check] [--render <output>]
./blunderDB transcribe --db <path> --match <id> --check
./blunderDB transcribe --db <path> --draft <id> --check
```

**Options:**

- `--mat` - `.mat` file to replay
- `--db` - Database holding the match or the draft
- `--match` - Match id to replay (requires `--db`)
- `--draft` - Transcription draft id to replay (requires `--db`)
- `--check` - List the inconsistencies found (the default)
- `--render` - Write the transcription back as a `.mat` file to this path
- `--format` - Output format: `text` (default) or `json`

`--check` names each inconsistency with the action number and the game it sits
in: an illegal play, two turns in a row for the same player, an impossible cube
action, an action past the end of the match, a play whose steps do not use its
own dice.

**An inconsistency is reported, never held against the input.** Nothing is
refused for one, and the exit status stays 0 whatever the replay finds — an
illegal play in a `.mat` is what the players did, or what the file's author
typed, not a defect in the file. A non-zero status is kept for a real failure:
an unreadable file, a database that will not open, an output that cannot be
written. A script that wants to act on the findings reads them from
`--format json`, where "this file is broken" and "this game had an illegal
move" do not get confused with one another.

`--render` writes the transcription back out as a `.mat`, which is how the round
trip is checked on a real file outside the test suite.

**Examples:**

```bash
# What does this .mat contain that a replay cannot account for?
./blunderDB transcribe --mat match.mat --check

# match.mat: 7 point match, 4 game(s), 203 action(s)
#   Final score: 9-2
# Inconsistencies: none

# Machine-readable, for a script
./blunderDB transcribe --mat match.mat --check --format json

# Round trip: render it back out and compare
./blunderDB transcribe --mat match.mat --render out.mat
diff match.mat out.mat

# A match of the library, and a draft being typed
./blunderDB transcribe --db database.db --match 5 --check
./blunderDB transcribe --db database.db --draft 3 --check
```

## Tournament Command

Read a directed tournament without a graphical interface. Directing one
interactively is the engine's own console (Nicomaque ships it); these
sub-commands read, none of them waits for input, and `move` is the only one that
writes.

Use it after a tournament rather than during one: `verify` is the check you run
on last season's databases, `standings` is what goes into the accounts, `page`
is what goes on the second screen, and `export` is the way out — the raw journal
is the whole truth of a direction, and a tool that reads it needs no blunderDB.

```bash
./blunderDB tournament list --db database.db
./blunderDB tournament verify --db database.db --id 3
./blunderDB tournament standings --db database.db --id 3 > standings.csv
./blunderDB tournament page --db database.db --id 3 --out /tmp/display
./blunderDB tournament export --db database.db --id 3 > journal.json
./blunderDB tournament move --db database.db --id 3 --match m4 --table 7
```

`move` changes the table of a running match, the gesture of dragging one table
onto another in the grid. When the destination is taken the two matches swap
tables; an out-of-service table is refused.

`verify` **exits in error** when a warning remains after the replay: a script
that runs it over a season's databases wants a status, not a line to grep.

## Players Command

One person often signs several ways across files ("Doe J.", "John Doe"). An
**alias** says a name is another spelling of a canonical name. The matches keep
the names their files wrote; the stats, the players table and the `pl"…"` search
read every spelling as one person, and every later import stores the canonical
name. The match fingerprints (`match_hash`, `canonical_hash`) keep the file's
names, so a file imported before its alias existed still finds itself; a match
whose dice are a stored match's and whose names differ only by known aliases is
recognised as that match and enriches it. The GUI's merge of players records the
same aliases.

```bash
blunderdb players alias add --db base.db "Doe J." "John Doe"
blunderdb players alias list --db base.db --format json
blunderdb players alias remove --db base.db "Doe J."
blunderdb players alias suggest --db base.db   # case, accents, punctuation, word order; nothing recorded
```

The table stays flat: a canonical that is itself an alias is followed to its own
canonical, and the aliases of a name that becomes an alias move with it.

## Events Command

The same four actions for event names (a file's Event field). On import, a match
whose event is an alias is filed under the canonical event's tournament.

```bash
blunderdb events alias add --db base.db "Open 2025" "Autumn Open 2025"
blunderdb events alias list --db base.db
```

## Duel Command

Plays a Duel without an interface, one Action per call. Each call is its own
process: it reloads the Duel from the database, plays one Action and writes it
back. A Duel played this way has no Cadence and no decision time (move times
are unknown, never zero).

```bash
./blunderDB duel create --db database.db (--length <n> | --money [--jacoby]) [--side1 <side>] [--side2 <side>]
./blunderDB duel show --db database.db --id <id>
./blunderDB duel list --db database.db
./blunderDB duel move --db database.db --id <id> --play "24/18 13/11"
./blunderDB duel roll|double|take|pass|resign --db database.db --id <id>
./blunderDB duel contribute --db database.db --id <id> --side 1|2 --value <text>
./blunderDB duel stop|discard --db database.db --id <id>
```

**Subcommands:**
- `create`: starts a Duel and plays up to the first decision of an external
  side. `--length` (1 to 25 points) or `--money`; `--start` (XGID);
  `--name1`, `--name2`; `--side1`, `--side2` (`external`, or `bot:<level>` with
  `instant`, `normal` or `thorough`, or `external:<configuration>@<engine>`, an
  external side declaring the Bot that plays behind it, recorded in the Match's
  origin as declared, not attested); `--discard-at-end` drops the draft
  instead of writing the Match; `--combined-seed` rolls nothing before each
  external side has contributed to the seed. Two bots play the whole match in one call; a
  money session between two bots is refused, since it would never end.
- `show`: score, what the Duel waits for and, for a move, the legal plays. The
  dice seed is never shown before the end.
- `list`: the Duels in progress.
- `roll`, `move`, `double`, `take`, `pass`, `resign`: one Action by the side
  the Duel waits for (`--side 1|2` names it); `--level` (1 to 3) is the value
  of a resignation; `--revision` refuses the Action if the Duel moved.
- `contribute`: an external side's contribution to a combined seed (`--side
  1|2`, `--value`, 1 to 64 bytes), once, before the first roll; the last one
  starts the dice. The rolls then come from HMAC-SHA256 of the sealed seed over
  each side's contribution, length-prefixed, and the Match's origin carries the
  contributions.
- `stop`: stops the Duel and writes the Match as it stands.
- `discard`: drops the Duel; nothing is written.

All take `--db` and `--format` (`text` or `json`).

## Trash Command

What was deleted through the trash, and how to put it back. A delete is still a
delete: a JSON snapshot of what disappears is written first, and nothing else in
the database knows the trash exists — no search filter, no statistic, no
retention rule reads it.

```bash
./blunderDB trash <subcommand> --db database.db [options]
```

**Subcommands:**
- `list` - What is in the trash, most recently deleted first
- `restore --id N` - Put entry N back, and drop it from the trash
- `discard --id N` - Drop entry N now, without restoring it
- `empty [--older-than D]` - Drop everything, or only what is older than D days
- `delete --kind K --id N` - Delete an object **through** the trash, so the
  gesture can be undone; `K` is `position`, `collection` or `comment`

**Options:**
- `--db` - Path to the database file (required)
- `--kind` - `position`, `collection` or `comment` for `delete`; narrows `list`, which also takes `anki_card`
- `--id` - Trash entry (`restore`, `discard`) or object (`delete`) id
- `--limit` - Rows to list (default 50)
- `--older-than` - `empty` only: keep entries newer than this many days
- `--format` - Output format: `text` (default) or `json`

`blunderdb delete` still deletes outright: a script that deletes a position
expects it gone, and leaving a snapshot behind in silence would grow a file
nobody asked to grow. `trash delete` is the one that keeps the undo.

Restoring a position goes back through Zobrist deduplication: it never creates a
duplicate, but it does not return the old id either — the original row is gone.
A restored position is the same position under a new number.

Anything older than thirty days is dropped by `blunderdb vacuum` — never at open.

**Example:**
```bash
# Delete a position, keeping the undo
./blunderDB trash delete --db database.db --kind position --id 412

# Look at the trash, then put an entry back
./blunderDB trash list --db database.db
./blunderDB trash restore --db database.db --id 3

# Keep only what is less than thirty days old
./blunderDB trash empty --db database.db --older-than 30
```

## Repair Command

Recompute what the database derives from what it stores: the scalar columns of
every analysis, from the JSON they are a projection of, each position's
`game_phase` and `game_type`, from its board, and the Crawford sentinel of every
away score, from the match the position came from or the XGID it came in with. Those derived values are what
the search filters, the statistics and the cube verdict read, so this is what to
run after a fix to how an imported analysis is parsed, or after a change to how
a phase or a game type is decided. Nothing runs it automatically.

```bash
./blunderDB repair --db database.db
```

**Options:**
- `--db` - Path to the database file (required)
- `--format` - Output format: `text` (default) or `json`
- `--stats` - Also recompute the per-match statistics from scratch
- `--duplicates` - Only list the suspected duplicate matches (see below)

**Suspected duplicates.** `repair --duplicates` runs none of the passes above:
it lists the pairs of matches whose dice say they are one match, and merges
nothing. Two kinds: the same length, initial score and dice in every game
under other player names (`#12 (…) has the dice of #7 (…)`), and a match whose
dice continue another's, a truncated match later completed (`#31 (…) is a
longer version of #30 (…)`). Under a same-dice pair, the text output gives the
`players alias add` commands that would make the two matches name the same
players: one reading when a name is common to both (the other name can only be
the other player), else two — seat for seat, then crosswise — numbered, of which
only one is to be run. The spelling of the later match is the alias, that of the
earlier one the canonical name. `--format json` returns `{"suspects": [{"kind":
"same_dice"|"longer", "matchId", "otherId", "players", "otherPlayers",
"pairings": [{"aliases": [{"alias", "canonical"}]}]}]}`, `pairings` on a
same-dice pair only. A
match imported before the dice hash existed gets it on the way, which is the
only thing this mode writes. An import also signals a match whose dice are
already stored under other names: `probable duplicate of #N under other
names` under its line, and in the import report.

The analyses are left untouched: this repairs only what was derived from them,
and a position with no analysis keeps its empty columns. Use `analyze` to
compute the analyses that are missing. Running it on a database that is already
up to date rewrites nothing and costs one scan.

The Crawford pass is the one that touches the positions themselves, and it is
worth knowing why. An away score of `1` means "one point to go, and this IS the
Crawford game"; `0` means "one point to go, Crawford behind us". Until the
importers wrote that distinction, every post-Crawford position was stored as a
Crawford one and read with a dead cube — where the trailer in fact doubles at
the first opportunity. Correcting the score changes the position's Zobrist hash,
so such a row is rehashed, and merged with its correctly stored twin when the
database already holds one: the analysis, the comments, the collections, the
Anki cards and their review journal, the match moves and the trash entries that
name it follow the surviving row. A position no match
points at is corrected only on the word of the XGID it brought from another
program (XG, BGBlitz…): when that XGID's Crawford field says the game is not the
Crawford one, and the XGID describes this very position. The other way round, a
position without a match stored at `0` on both sides becomes `1` when the XGID it
brought is a 1-point match that describes it: a 1-point match's only game starts
one point from the goal, so it is the Crawford game, as the importers write it.
The DMP after the Crawford game of a longer match stays at `0` — its XGID gives
that longer match. An XGID blunderDB rewrote itself only echoes the stored score
and proves nothing. Any other position without a match is left alone — nothing
contradicts what its score says.

The JSON report has one counter per pass: `repaired` (analysis columns),
`phases` (positions reclassified) and `crawford` (positions rehashed), plus
`match_stats` (matches recomputed) with `--stats`.

The per-match statistics are the decisions, error, PR, blunders, luck and
dominant analysis provenance of each seat of each match, which the statistics
read instead of rescanning every decision. They are written with the match at
import and dropped by every change that alters them (a match deleted, its seats
swapped, an analysis of one of its positions changed, the blunder threshold
moved); the next read recomputes what is missing. A database migrated from an
older version computes them once, when it is first opened. `--stats` rebuilds
them all: the way back if that bookkeeping were ever wrong.

## Info Command

Display database metadata and statistics.

```bash
./blunderDB info --db database.db [options]
```

**Options:**
- `--db` - Path to the database file (required)
- `--format` - Output format: `text` or `json` (default: text)
- `--estimate` - Answer at once on a very large database: beyond 200,000 rows a count is an estimate (listed in `approximate`), and the blunders are not counted. Default: every row is counted

**Examples:**
```bash
# Display database info
./blunderDB info --db database.db

# Display as JSON (for scripting)
./blunderDB info --db database.db --format json
```

**Example output (text):**
```
Database Information
==================================================
Path: database.db

Metadata:
  Version: 2.3.0
  User: John
  Description: 2025 tournament matches
  Date of Creation: 2025-11-03 14:30:00

Statistics:
  Positions: 1523
  Analyses: 847
  Matches: 12
  Games: 156
  Moves: 3421
```

### Reading where a file came from

`info` never writes: reading a database's origin leaves it untouched. It reads a protected `.dbx` file too, from its cleartext header, without the password.

Nothing is printed for an ordinary database — a file that was never watermarked shows exactly what it always did.

```
Origin:
  Cours de Jean Dupont — 12 mars 2026
  Produced by:  Jean Dupont  (A961-A612-4420-7D68)  ✓ signature verified — marked by you
  Marked on:    2026-03-12
  Note:         Merci de ne pas rediffuser.
```

There is nothing else to show: no recipient, no holder, no history. blunderDB records none of it.

## Edit Command

Edit database metadata (user name, description) and the library's own
thresholds.

```bash
./blunderDB edit --db database.db [options]
```

**Options:**
- `--db` - Path to the database file (required)
- `--user` - Set the user name
- `--description` - Set the description
- `--clear-user` - Clear the user name
- `--clear-description` - Clear the description
- `--error-threshold` - Error threshold, in millipoints: a decision costing at least this much is an error
- `--blunder-threshold` - Blunder threshold, in millipoints: an error costing at least this much is a blunder
- `--format` - Output format: `text` (default) or `json` (`{"changes": [...]}`)

At least one edit option is required.

**Examples:**
```bash
# Set user name
./blunderDB edit --db database.db --user "Jane"

# Set description
./blunderDB edit --db database.db --description "Updated match collection"

# Set both
./blunderDB edit --db database.db --user "Jane" --description "My matches"

# Clear user name
./blunderDB edit --db database.db --clear-user

# Clear description
./blunderDB edit --db database.db --clear-description

# Draw the library's own lines: XG's thresholds (ADR-0046)
./blunderDB edit --db database.db --error-threshold 20 --blunder-threshold 80
```

**Example output:**
```
Database metadata updated:
  - Set user to: Jane
  - Set description to: Updated match collection
```

## Verify Command

Verify database integrity and optionally compare match data against source files.

```bash
./blunderDB verify --db database.db [options]
```

**Options:**
- `--db` - Path to the database file (required)
- `--match` - Match ID to verify (optional — verifies specific match)
- `--mat` - Path to a MAT file to compare against (optional — used with `--match`)
- `--format` - Output format: `text` (default) or `json` (stats, orphans, schema drift, and the match check if any)

When run without `--match`, displays database statistics. When a match ID is specified, verifies the match data. When a MAT file is also provided, cross-references the database positions with the source file.

Every run also checks referential integrity: it counts orphaned rows — games without a match, moves without a game, move analyses without a move, analyses without a position, review-journal entries without a deck or without a position — and prints a `WARNING` line with the total when any exist. A healthy database reports `Orphaned rows: none`. Orphans can be left behind in databases written by versions that did not enforce foreign keys on every connection (issue #157), or before the review journal's own foreign keys existed (issue #185); the rows are unreachable from any match or deck and only take up space. The command still exits 0 when it finds some.

Every run also compares the schema against the reference DDL and lists the tables, columns and indexes the database lacks. Opening a database adds what is missing when it can and only logs what it cannot — typically a `UNIQUE` index that duplicate rows keep it from rebuilding — so this is where that gap becomes visible; a query naming one of those elements fails until the cause is fixed. A healthy database reports `Schema: matches the reference DDL`. Like orphans, drift is a finding, not a failure: the command still exits 0.

Every run also checks the rules the current DDL states but SQLite cannot add to a table that already exists: the range `CHECK` constraints (dice between 0 and 6, non-negative cube and pip counts, 0 to 15 checkers off, review ratings between 1 and 4), the Zobrist hash a row should never be without, and one analysis per position. A database created since schema 2.18.0 enforces them; an older one can still hold rows a new database would refuse, and those are what is counted here, rule by rule. A healthy database reports `Constraints: every row satisfies the current DDL`. One more finding: nothing is repaired and the command still exits 0.

Every run finally recomputes the two denormalised counters, `match.game_count` and `game.move_count`, from the rows they claim to count, and reports how many disagree and by how much at worst. Both are written once, at import, from what the **source file** held, and are what the match list and the game view display; a small gap is usually an import that skipped what it could not map. Nothing is rewritten — overwriting the counter with what was stored would erase the very discrepancy worth looking at. A healthy database reports `Counters: game_count and move_count agree with the rows`.

**Examples:**
```bash
# Verify database overview
./blunderDB verify --db database.db

# Verify a specific match
./blunderDB verify --db database.db --match 1

# Compare match against MAT source file
./blunderDB verify --db database.db --match 1 --mat original.mat
```

**Example output:**
```
Verifying database...

Database Statistics:
  Positions: 1523
  Analyses: 847
  Matches: 12
  Games: 156
  Moves: 3421

Orphaned rows: none

Schema: matches the reference DDL

Constraints: every row satisfies the current DDL (10 rules checked)

Counters: game_count and move_count agree with the rows

Verifying match 1...
  Match: Alice vs Bob
  Database positions: 234
  Comparing with MAT file: original.mat
  MAT file checker moves: 200
  MAT file cube actions: 34
  MAT file total: 234
  Database total positions: 234

Verification complete!
```

## Vacuum Command

Reclaim disk space left behind by deletions (matches, tournaments, purges).
SQLite never shrinks the database file on its own when rows are deleted — this
is the only way to compact it, and it never happens automatically at open,
since the cost is unpredictable on a large database.

```bash
./blunderDB vacuum --db database.db
```

**Options:**
- `--db` - Path to the database file (required)
- `--format` - Output format: `text` (default) or `json` (`{"size_before", "size_after", "reclaimed"}`, in bytes)

The command first runs a WAL checkpoint so the reported "before" size is
honest, then checks that the volume has roughly twice the current file size
free (SQLite rebuilds the whole database before swapping it in — it refuses
with a clear error rather than run out of room partway through), runs
`VACUUM`, and finishes with `ANALYZE` so the query planner's statistics match
the rebuilt file. Before the `VACUUM`, every analysis is rewritten in the
compact binary format at the strongest compression, including those an older
release stored as JSON (see `reencode`).

**Example:**
```bash
./blunderDB vacuum --db database.db
```

**Example output:**
```
Compacting database...
  Before: 128.4 MiB
  After:  41.2 MiB
  Reclaimed: 87.2 MiB
```

## Met Command

List, import or choose the match equity table (MET) gammonNet values a
database's match scores with: the built-in Kazaross-XG2 by default, or an
explicit gnubg `.xml` table. The table belongs to the database, not to the
application. Every analysis gammonNet computes records its table; an analysis
at a match score computed with another table than the current one is shown as
"different MET" and left out of the statistics. Changing the table rewrites no
analysis. The daemon exposes the same operations as `met.list`, `met.import`,
`met.setCurrent` and `met.ofAnalysis`, limited to the caller's tenant.

```bash
./blunderDB met --db database.db --import Rockwell-Kazaross.xml --current
./blunderDB met --db database.db --use 0
```

**Options:**
- `--db` - Path to the database file (required)
- `--import` - Import a gnubg table; a table already held, or equal to Kazaross-XG2, is not added twice
- `--current` - With `--import`: make the imported table current
- `--use` - Make table ID current; `0` returns to Kazaross-XG2
- `--format` - Output format: `text` (default) or `json`

**Example output:**
```
      0  Kazaross-XG2
  *   1  Rockwell/Kazaross 25 point MET
```

## Reencode Command

Rewrite the analyses an older release stored as JSON in the compact binary
format (about half the size, several times faster to read). Old analyses stay
readable without it; `vacuum` performs the same conversion while it compacts.
`reencode` converts without compacting: useful when the disk has no room for
the compacted copy `vacuum` writes beside the file, or on a PostgreSQL server,
which has no `vacuum`. The daemon exposes the same pass as
`maintenance.reencode`, limited to the caller's tenant.

```bash
./blunderDB reencode --db database.db
```

**Options:**
- `--db` - Path to the database file (required)
- `--format` - Output format: `text` (default) or `json` (`{"rewritten"}`, the number of analyses rewritten)

It works in batches of 2,000 analyses, each in its own transaction.
Interrupted, it resumes on the next run where it stopped: analyses already
converted are skipped without being read. An analysis that does not decode is
left as it is and logged. It never runs automatically.

**Example output:**
```
  Analyses rewritten: 15623468
```

## Healthcheck Command

Ask a running `serve` daemon (see the headless chapter of the docs) whether it
is ready: one `GET /readyz`, exit code `0` when the daemon answers 200 (storage
reachable, schema version as expected), `1` otherwise — storage down, stale
schema, or nothing listening at the address. No database file is opened.

```bash
./blunderDB healthcheck [--addr host:port] [--timeout 2s]
```

**Options:**
- `--addr` - Address the daemon listens on (default: `BLUNDERDB_ADDR`, else `:8080`). A listen address without a host (`:8080`) or with a wildcard host (`0.0.0.0`, `[::]`) is probed on the loopback interface.
- `--timeout` - Give up after this long (default: `2s`)

This is what the container image's `HEALTHCHECK` runs — the image is distroless
and ships no `curl` — and the `serve` binary built from `cmd/serve` understands
the same word (`/usr/local/bin/blunderdb healthcheck`). It works just as well
from a shell or a systemd unit.

**Example:**
```bash
./blunderDB serve --db database.db --addr 127.0.0.1:8080 &
./blunderDB healthcheck --addr 127.0.0.1:8080 && echo "daemon ready"
```

**Example output:**
```
ready
```

On failure the reason is printed, so `docker inspect` shows why the container
is unhealthy:

```
Error: healthcheck: http://127.0.0.1:8080/readyz answered 503 Service Unavailable (version_mismatch)
```

## Mcp Command

Serve the database's tools to an AI assistant (Claude Code, Claude Desktop, a
local client) over the Model Context Protocol, on stdin/stdout. blunderDB ships
no language model: the assistant you already use starts this command and calls
its tools — search positions in the command bar's grammar, read a position and
its analysis, explain an error, a player's statistics and recurring errors,
matches, tournaments, collections, a quiz. They only read unless `--write` is
given. The daemon serves the same tools on `POST /mcp` (see the headless
chapter of the docs, and ADR-0059 for the list).

```bash
./blunderDB mcp --db <file> [--write]
```

**Options:**
- `--db` - Path to the database file (required)
- `--write` - Also offer the tools that change the database: save a position, comment one, create and fill a collection. None deletes.

Like `call`, the command migrates an older database's schema when it opens it, even without `--write`.

**Example:** register the database with Claude Code.
```bash
claude mcp add blunderdb -- blunderdb mcp --db /path/to/my.db
```

## Completion Command

Print a shell completion script for the subcommand names to stdout. The
command list embedded in every script is generated from the same table
`blunderdb help` and `main.go`'s dispatch read (`handlers()`), so a new
subcommand is offered by completion the moment it ships — nothing here is
maintained by hand.

```bash
./blunderDB completion <bash|zsh|fish>
```

**Examples:**
```bash
# bash: load for the current shell session
source <(blunderdb completion bash)

# bash: install system-wide (Debian/Ubuntu/Arch layout)
blunderdb completion bash | sudo tee /etc/bash_completion.d/blunderdb > /dev/null

# zsh: install into a directory already on $fpath
blunderdb completion zsh > "${fpath[1]}/_blunderdb"

# fish: load for the current shell session
blunderdb completion fish | source
```

Packages install this automatically: the `.deb`/`.rpm` (nfpm) and the AUR
package generate the three scripts from the packaged binary at build time,
and the Homebrew cask runs `blunderdb completion <shell>` once at install
time via `generate_completions_from_executable`. Nothing is committed to the
repository, so completions can never drift from the subcommand table.

## Common Workflows

### Import Multiple Matches

```bash
# Use batch import (recommended)
./blunderDB import --db mymatches.db --type batch --dir ./matches/

# Or import individual files
./blunderDB import --db mymatches.db --type match --file match1.xg
./blunderDB import --db mymatches.db --type match --file match2.xg
```

### Backup Database

```bash
./blunderDB export --db production.db --type database --file backup-$(date +%Y%m%d).db
```

### Check Database Before and After Import

```bash
# Before
./blunderDB list --db database.db --type stats

# Import
./blunderDB import --db database.db --type match --file newmatch.xg

# After
./blunderDB list --db database.db --type stats
```

### Export Positions for Analysis

```bash
./blunderDB export --db database.db --type positions --file positions.txt
# Process positions.txt with external tools
```

## Error Handling

The CLI provides clear error messages:

```bash
# Missing required flag
./blunderDB import --db database.db --type match
# Error: --file flag is required

# File not found
./blunderDB import --db database.db --type match --file nonexistent.xg
# Error: input file does not exist: nonexistent.xg

# Invalid import type
./blunderDB import --db database.db --type invalid --file test.xg
# Error: unknown import type: invalid (must be 'match', 'position', or 'batch')

# Database errors
./blunderDB list --db /invalid/path/database.db --type stats
# Error: failed to open database: ...
```

## Tips

1. **Database Creation**: Use `create` to make a new database with metadata, or let `import` create one automatically if it doesn't exist.

2. **Match IDs**: After importing a match, note the returned Match ID for future reference (listing, deletion, etc.).

3. **Batch Operations**: Use `import --type batch` to import an entire directory of match files at once.

4. **Data Safety**: Always use `--confirm` flag carefully when deleting data. The delete operation is permanent.

5. **Performance**: For large databases, use `--limit` when listing positions to avoid overwhelming output.

6. **Database Info**: Use `info` and `verify` to inspect database contents and integrity before and after operations.

## Integration with GUI

The CLI and GUI share the same database format, so you can:

1. Import matches via CLI for batch processing
2. Open the same database in GUI for interactive analysis
3. Export from GUI, process via scripts, reimport via CLI

## Exit Codes

- `0` - Success
- `1` - Error occurred

This makes the CLI suitable for use in scripts with error checking:

```bash
if ./blunderDB import --db database.db --type match --file match.xg; then
    echo "Import successful"
else
    echo "Import failed"
    exit 1
fi
```

## Generic `call` dispatcher

In addition to the historical subcommands above, `blunderDB call` exposes
**every** storage operation directly. It dispatches in-process through the exact
same handlers the `serve` daemon serves, so the behaviour is identical to
`POST /v1/<family>.<method>` — useful for scripting and integration testing.

```bash
# List every available method (108+)
blunderDB call --list

# Read-only queries
blunderDB call metadata.counts --db mydb.db
blunderDB call positions.list   --db mydb.db --json '{"limit":10}'
blunderDB call matches.get      --db mydb.db --json '{"id":1}'

# Mutations
blunderDB call positions.save   --db mydb.db --json '{"position":{...}}'
blunderDB call matches.delete   --db mydb.db --json '{"id":42}'

# Maintenance (SQLite backend only; the same code path as `vacuum`)
blunderDB call maintenance.vacuum --db mydb.db
```

Flags:

| Flag | Default | Meaning |
|------|---------|---------|
| `--db <path>` | – | SQLite database file (shorthand for `--backend sqlite --dsn <path>`) |
| `--backend <kind>` | `sqlite` | `sqlite` or `postgres` (or `$BLUNDERDB_BACKEND`) |
| `--dsn <string>` | `$BLUNDERDB_DSN` | backend connection string |
| `--scope <n>` | `1` | tenant, a positive decimal integer (sent as `X-Tenant-ID`; a name such as `alice` is refused; SQLite ignores it for most families) |
| `--json <string>` | `{}` | request body as JSON |
| `--json-file <path>` | – | read the request body from a file |
| `--list` | – | print every `<family>.<method>` and exit |

The JSON response (or NDJSON stream for `*.list` endpoints) is written to
stdout. On an error the process exits non-zero and the `{"error":{…}}` envelope
is printed to stdout so it stays parseable (e.g. with `jq`).

## Migrating a SQLite database into PostgreSQL

`blunderDB migrate` copies a single-user SQLite database into a PostgreSQL
backend under a chosen tenant — the positive decimal integer the reverse-proxy
will send as `X-Tenant-ID` for that user — the path for a desktop user to
"upload" their library into a server deployment.

```bash
blunderDB migrate \
    --from sqlite:///path/to/user.db \
    --to   "postgres://user:pass@host:5432/db?sslmode=disable" \
    --tenant-id 42

# Preview without writing
blunderDB migrate --from sqlite:///path/to/user.db --tenant-id 42 --dry-run
```

It copies **positions, their analyses and comments, matches (games + moves),
tournaments (with match links) and collections (with membership)** under the
tenant scope, remapping primary/foreign keys, inside a single destination
transaction (atomic — a failed run leaves the destination untouched, just
re-run). Progress and the final tally are emitted as NDJSON to stdout.

| Flag | Default | Meaning |
|------|---------|---------|
| `--from <uri>` | – | source SQLite DB (`sqlite:///path` or a bare path) |
| `--to <dsn>` | – | destination PostgreSQL DSN (`postgres://…`) |
| `--tenant-id <n>` | – | destination tenant, a positive decimal integer (required unless `--dry-run`; a name such as `my-tenant` is refused) |
| `--dry-run` | – | count what would be copied without writing |
| `--on-conflict <policy>` | `""` | `""` aborts if the tenant already has data; `skip` merges (positions dedup by Zobrist) |

Not migrated (yet): app-state families — anki decks/cards, the filter library,
search/command history, and the session state. Their per-tenant scoping is in
place (each has its own tenant-scoped table, `session_state` since schema
2.16.0); data migration of the core position library and match history is the
priority.

## Flag reference (generated)

<!-- BEGIN GENERATED CLI REFERENCE (cmd/cli-doc-gen; do not edit by hand, run `go run ./cmd/cli-doc-gen`) -->

Captured verbatim from each subcommand's `--help`. Regenerate with `go run ./cmd/cli-doc-gen` whenever a flag changes; the prose and
examples above are hand-written and this section never rewrites them.

### `blunderdb analyze`

```
Usage: blunderdb analyze [options]

Write a gammonNet analysis for every position that has none —
catching up a library built before this feature existed. A
Position already carrying any analysis (XG, GNUbg, BGBlitz, or a
prior gammonNet run) is left untouched: this only ever fills a
gap (ADR-0013). Interrupted with Ctrl-C, the run is cancelled
cleanly — nothing is lost, and re-running picks up exactly where
it left off, with no journal needed.

--match restricts the sweep to the positions of a single match,
the id `blunderdb list --type matches` prints. Same gap rule and
same guarantees, narrower scope: a match just imported or
transcribed is analysed without sweeping the whole library, and
a correction re-analysed a second time costs only the positions
the correction created.

--stale switches to the other sweep: every position whose stored
analysis is entirely gammonNet's own (never an XG/GNUbg/BGBlitz
one — ADR-0013 protects those unconditionally) but was written at
an older engine version or a different depth than --ply now asks
for. Use it after an engine upgrade, or after raising --ply for a
library already analysed at a shallower depth.

Positions are analysed --jobs at a time, on that many cores: the
positions of a batch are independent, so the result is the same
whatever --jobs says. Use --jobs 1 to leave the machine free.

--compare answers a different question and WRITES NOTHING: on the
positions carrying an analysis somebody else wrote (XG, GNUbg,
BGBlitz), how often does gammonNet name the same best move or the
same cube action, and what would following it have cost on the
imported analysis's own scale? The answer is broken down by game
phase, which is what says where the disagreements sit. Use --limit
to ask the question of a sample rather than of a whole library.

--rollout switches to rollouts: every position --query selects is
rolled out, one after the other on every core, and the rollout is
written beside its analysis — never in its place. The value is a
preset (fast: 216 games truncated at 7; standard: 1296 games
truncated at 11) or custom settings: an optional preset first, then
games=, min-games=, truncation=, jsd=, ply=, candidates=, seed=,
comma-separated. A position already carrying a rollout with the
same settings is skipped, so an interrupted run resumes; the
position in hand when Ctrl-C arrives is dropped whole.

A position gammonNet declines to evaluate (a match score beyond
its MET, a cube state it refuses) is reported separately at the
end, as "refused": not a failure, and not retried to no effect.

Options:
  -candidates int
    	Candidate moves kept per checker decision (default 10)
  -compare
    	Compare gammonNet against the imported analyses instead of writing anything
  -db string
    	Path to the database file (required)
  -format string
    	Output format: text or json (default "text")
  -jobs int
    	Positions analysed in parallel (one CPU each) (default 28)
  -limit int
    	With --compare: stop after this many positions (0 = all)
  -match int
    	Restrict the sweep to one match's positions (0 = the whole library)
  -ply int
    	Search depth (canonical: 2, k=12) (default 2)
  -prune-k int
    	Pruning width (canonical: 12) (default 12)
  -query string
    	With --rollout: the positions to roll out, in the search query language (see search --query-help); empty means every position
  -rollout string
    	Roll out the positions --query selects instead: a preset (fast, standard) or custom settings, e.g. 'standard' or 'games=648,truncation=0,ply=1'
  -stale
    	Re-analyse positions whose gammonNet analysis is outdated, instead of filling gaps

Examples:
  blunderdb analyze --db database.db
  blunderdb analyze --db database.db --jobs 1
  blunderdb analyze --db database.db --match 12
  blunderdb analyze --db database.db --stale --ply 3
  blunderdb analyze --db database.db --format json
  blunderdb analyze --db database.db --compare --limit 500
  blunderdb analyze --db database.db --rollout fast --query 'E>80'
  blunderdb analyze --db database.db --rollout 'standard,ply=1' --query 'c'
```

### `blunderdb anki card`

```
Usage: blunderdb anki card [options]

Suspend, bury or remove one card.

Options:
  -action string
    	suspend, unsuspend, bury or remove (required)
  -db string
    	Path to the database file (required)
  -format string
    	Output format: text or json (default "text")
  -id int
    	Card ID (required)

Examples:
  blunderdb anki card --db database.db --id 12 --action suspend
  blunderdb anki card --db database.db --id 12 --action unsuspend
  blunderdb anki card --db database.db --id 12 --action bury
  blunderdb anki card --db database.db --id 12 --action remove
```

### `blunderdb anki decks`

```
Usage: blunderdb anki decks [options]

List the decks of the database with their counters.

Options:
  -db string
    	Path to the database file (required)
  -format string
    	Output format: text, json or csv (default "text")

Examples:
  blunderdb anki decks --db database.db
  blunderdb anki decks --db database.db --format csv
```

### `blunderdb anki forecast`

```
Usage: blunderdb anki forecast [options]

Cards coming due per calendar day (UTC). Day 0 holds every overdue card.

Options:
  -days int
    	Number of days to project (1-365) (default 30)
  -db string
    	Path to the database file (required)
  -deck int
    	Deck ID (0 = every deck)
  -format string
    	Output format: text, json or csv (default "text")

Examples:
  blunderdb anki forecast --db database.db --deck 2 --days 14
  blunderdb anki forecast --db database.db --days 30 --format csv   # every deck
```

### `blunderdb anki log`

```
Usage: blunderdb anki log [options]

Recorded review events, most recent first.

Options:
  -db string
    	Path to the database file (required)
  -deck int
    	Deck ID (0 = every deck)
  -format string
    	Output format: text or json (default "text")
  -limit int
    	Maximum number of events (default 20)

Examples:
  blunderdb anki log --db database.db
  blunderdb anki log --db database.db --deck 2 --limit 50
  blunderdb anki log --db database.db --format json
```

### `blunderdb anki retention`

```
Usage: blunderdb anki retention [options]

Measured retention of one deck against its target.

Options:
  -db string
    	Path to the database file (required)
  -deck int
    	Deck ID (required)
  -format string
    	Output format: text or json (default "text")

Examples:
  blunderdb anki retention --db database.db --deck 2
  blunderdb anki retention --db database.db --deck 2 --format json
```

### `blunderdb anki stats`

```
Usage: blunderdb anki stats [options]

Review statistics of one deck.

Options:
  -db string
    	Path to the database file (required)
  -deck int
    	Deck ID (required)
  -format string
    	Output format: text or json (default "text")

Examples:
  blunderdb anki stats --db database.db --deck 2
  blunderdb anki stats --db database.db --deck 2 --format json
```

### `blunderdb anki sync`

```
Usage: blunderdb anki sync [options]

Resynchronise a deck with its source (collection or stored search).

Options:
  -db string
    	Path to the database file (required)
  -deck int
    	Deck ID (required)

Examples:
  blunderdb anki sync --db database.db --deck 2
```

### `blunderdb bearoff delete`

```
Usage: blunderdb bearoff delete --ts <domain> [options]

Remove a generated table, along with any paused run and any debris
of an interrupted one. A default domain is regenerated on the next
launch of the application; a wider one is not.

Options:
  -data-dir string
    	Where to look (default: the application's data directory)
  -ts string
    	Domain to delete, as 6x9, or os8 for a one-sided table (required)

Examples:
  blunderdb bearoff delete --ts 6x11
```

### `blunderdb bearoff generate`

```
Usage: blunderdb bearoff generate --ts <domain> [options]

Compute a bearoff table. The result is checked against gnubg's
fingerprint for its domain before it is put in place, so a table
this writes is the table makebearoff writes.

Ctrl-C PAUSES: the state is written beside the table and the next
run on the same domain continues from it rather than starting
over. `bearoff delete` throws a paused run away.

Options:
  -cores int
    	Cores to use (default: every core but one)
  -data-dir string
    	Where to write it (default: the application's data directory)
  -os int
    	One-sided domain to generate, as a point count: 6 … 12
  -quiet
    	No progress line
  -ts string
    	Two-sided domain to generate, as 6x9

Examples:
  blunderdb bearoff generate --ts 6x9
  blunderdb bearoff generate --ts 6x11 --cores 4 --data-dir /srv/bearoff
  blunderdb bearoff generate --os 8       # the EPC beyond the home board
```

### `blunderdb bearoff list`

```
Usage: blunderdb bearoff list [options]

List every domain that can be generated: what it weighs, what it
needs in memory, roughly how long it takes here, and whether this
machine already has it.

Options:
  -cores int
    	Cores the estimate assumes (default: every core but one)
  -data-dir string
    	Where to look (default: the application's data directory)
  -format string
    	Output format: text or json (default "text")

Examples:
  blunderdb bearoff list
  blunderdb bearoff list --format json --cores 8
```

### `blunderdb bearoff verify`

```
Usage: blunderdb bearoff verify <file.bd> [options]

Check a bearoff file against the SHA-256 gnubg produces for its
domain. Three answers: verified (the same bytes as the reference),
unverified (well formed, but no fingerprint is recorded for that
domain), corrupt (the file contradicts its own header).

Options:
  -format string
    	Output format: text or json (default "text")

Examples:
  blunderdb bearoff verify ~/.local/share/blunderDB/gnubg_ts6x6.bd
```

### `blunderdb collection create`

```
Usage: blunderdb collection create [options]

Create an empty collection.

Options:
  -db string
    	Path to the database file (required)
  -description string
    	Collection description
  -name string
    	Collection name (required)

Examples:
  blunderdb collection create --db database.db --name "Blitz openings"
```

### `blunderdb collection delete`

```
Usage: blunderdb collection delete [options]

Delete a collection. Its positions stay in the database.

Options:
  -confirm
    	Confirm deletion without prompting
  -db string
    	Path to the database file (required)
  -id int
    	Collection ID (required)

Examples:
  blunderdb collection delete --db database.db --id 3 --confirm
```

### `blunderdb collection export`

```
Usage: blunderdb collection export [options]

Export one or more collections to a new database file.

Options:
  -analysis
    	Include analyses (default true)
  -comments
    	Include comments (default true)
  -db string
    	Path to the database file (required)
  -id string
    	Collection ID(s) to export, comma-separated (required)
  -out string
    	Path of the database file to write (required)
  -watermark string
    	Mark the exported file with where it comes from
  -watermark-note string
    	Free text attached to the watermark (terms of use, contact)

Examples:
  blunderdb collection export --db database.db --id 3 --out openings.db
  blunderdb collection export --db database.db --id 3,4 --out openings.db --comments=false
  blunderdb collection export --db database.db --id 3 --out cours.db --watermark "Cours du 12 mars"
```

### `blunderdb collection filter`

```
Usage: blunderdb collection filter [options]

Make a collection living: its content becomes the result of a search query, re-evaluated every time it is opened. An empty query turns it back into a hand-made list, keeping the positions it already held.

Options:
  -clear
    	Turn the collection back into a hand-made list
  -db string
    	Path to the database file (required)
  -id int
    	Collection ID (required)
  -query string
    	The search query, in the application's own grammar; empty clears it

Examples:
  blunderdb collection filter --db database.db --id 3 --query "E>80 gt:holding"
```

### `blunderdb collection freeze`

```
Usage: blunderdb collection freeze [options]

Freeze a living collection: the positions its query selects now become its content, and the query is cleared.

Options:
  -db string
    	Path to the database file (required)
  -id int
    	Collection ID (required)

Examples:
  blunderdb collection freeze --db database.db --id 3
```

### `blunderdb collection list`

```
Usage: blunderdb collection list [options]

List the collections of the database.

Options:
  -db string
    	Path to the database file (required)
  -format string
    	Output format: text, json or csv (default "text")

Examples:
  blunderdb collection list --db database.db
  blunderdb collection list --db database.db --format csv
```

### `blunderdb collection pile`

```
Usage: blunderdb collection pile [options]

Put a position on the Pile (the collection of positions to come back to), or take it off when it is there. A position given by XGID that is not in the library is written first, as a position brought in on its own. Without a position, print the Pile.

Options:
  -db string
    	Path to the database file (required)
  -position-id int
    	ID of a stored position
  -xgid string
    	XGID of the position (written to the library if absent)

Examples:
  blunderdb collection pile --db database.db --position-id 42
  blunderdb collection pile --db database.db --xgid "XGID=-b----E-C---eE---c-e----B-:0:0:1:21:0:0:0:0:10"
  blunderdb collection pile --db database.db
```

### `blunderdb collection rename`

```
Usage: blunderdb collection rename [options]

Rename a collection (and optionally change its description).

Options:
  -db string
    	Path to the database file (required)
  -description string
    	New description (empty keeps the current one)
  -id int
    	Collection ID (required)
  -name string
    	New collection name (required)

Examples:
  blunderdb collection rename --db database.db --id 3 --name "Openings"
```

### `blunderdb collection show`

```
Usage: blunderdb collection show [options]

List the positions of one collection.

Options:
  -db string
    	Path to the database file (required)
  -format string
    	Output format: text, json or csv (default "text")
  -id int
    	Collection ID (required)

Examples:
  blunderdb collection show --db database.db --id 3
  blunderdb collection show --db database.db --id 3 --format json
```

### `blunderdb comment add`

```
Usage: blunderdb comment add [options]

Add a comment to a position.

Options:
  -author string
    	Who signs the comment (empty: unsigned)
  -db string
    	Path to the database file (required)
  -position int
    	Position id (required)
  -text string
    	Comment text (required)

Examples:
  blunderdb comment add --db database.db --position 412 --text "Cube too early" --author Alice
```

### `blunderdb comment list`

```
Usage: blunderdb comment list [options]

List the comments of a position, or of the whole database.

Options:
  -author string
    	Only the comments signed by this author (whole name, any case)
  -db string
    	Path to the database file (required)
  -format string
    	Output format: text or json (default "text")
  -position int
    	Only the comments of this position id (0: every position)

Examples:
  blunderdb comment list --db database.db --position 412
  blunderdb comment list --db database.db --author Alice --format json
```

### `blunderdb completion`

```
Usage: blunderdb completion <bash|zsh|fish>

Print a shell completion script for the subcommand names to stdout.

Examples:
  # bash: load for the current shell session
  source <(blunderdb completion bash)

  # bash: install system-wide (Debian/Ubuntu/Arch layout)
  blunderdb completion bash | sudo tee /etc/bash_completion.d/blunderdb > /dev/null

  # zsh: install into a directory already on $fpath
  blunderdb completion zsh > "${fpath[1]}/_blunderdb"

  # fish: load for the current shell session
  blunderdb completion fish | source
```

### `blunderdb create`

```
Usage: blunderdb create [options]

Create a new database with the required schema and optional metadata.

Options:
  -db string
    	Path to the database file to create (required)
  -description string
    	Description of the database
  -force
    	Overwrite existing database if it exists
  -format string
    	Output format: text or json (default "text")
  -user string
    	User name (owner of the database)

Examples:
  # Create a new database
  blunderdb create --db mydb.db

  # Create with metadata
  blunderdb create --db mydb.db --user "John Doe" --description "My backgammon positions"

  # Force overwrite an existing database
  blunderdb create --db mydb.db --force
```

### `blunderdb cubematrix`

```
Usage: blunderdb cubematrix [options] <XGID|OGID>

Show the cube verdict at every score of a match: for each away × away
cell, whether this position is a double and whether it is a take.
The position's own score is ignored — the grid replaces it — but its
cube is kept, so the answer is about this cube, not a centred one.

Every cell is its own search: the engine is match-aware, so a single
search read through different match equities would be wrong exactly
where the score matters. The grid is post-Crawford throughout.

Options:
  -format string
    	Output format: text, json (default "text")
  -jobs int
    	Parallel searches (0 = one per core)
  -match-length int
    	Match length the grid spans (1-25) (default 7)
  -ply int
    	Search depth for every cell (0 or 2) (default 2)
  -prune-k int
    	Candidate moves kept by the prune network (default 12)

Examples:
  blunderdb cubematrix 'XGID=-b----E-C---eE---c-e----B-:0:0:1:00:0:0:0:7:10'
  blunderdb cubematrix --match-length 5 --format json '<XGID>'
  blunderdb cubematrix '11ccccchhhjjjjj:66666888dddddoo:N0N::W::0:0:7:'  # an OGID
```

### `blunderdb delete`

```
Usage: blunderdb delete [options]

Delete data from the database.

Options:
  -confirm
    	Confirm deletion without prompting
  -db string
    	Path to the database file (required)
  -format string
    	Output format: text or json (default "text")
  -id int
    	ID of the item to delete (required)
  -type string
    	Delete type: match (required)

Examples:
  # Delete match with ID 1
  blunderdb delete --db database.db --type match --id 1 --confirm
```

### `blunderdb duel contribute`

```
Usage: blunderdb duel contribute [options]

Contribute to the seed of a Duel created with --combined-seed: once per external Side, before the first roll, which the last contribution starts.

Options:
  -db string
    	Path to the database file (required)
  -format string
    	Output format: text or json (default "text")
  -id int
    	Duel id (required)
  -revision int
    	Refuse unless the Duel is at this revision (default: no check)
  -side int
    	The contributing Side, 1 or 2 (required)
  -value string
    	The contribution, 1 to 64 bytes of text (required)

Examples:
  blunderdb duel contribute --db database.db --id 1 --side 1 --value "my own randomness"
```

### `blunderdb duel create`

```
Usage: blunderdb duel create [options]

Start a Duel and play on to the first Decision of an external Side.

Options:
  -combined-seed
    	Roll nothing before each external Side has contributed to the seed (duel contribute)
  -db string
    	Path to the database file (required)
  -discard-at-end
    	Throw the draft away when the match is won instead of writing the Match
  -format string
    	Output format: text or json (default "text")
  -jacoby
    	With --money: play the Jacoby rule
  -length int
    	Match length in points, 1 to 25
  -money
    	A money session instead of a match
  -name1 string
    	Player 1's name
  -name2 string
    	Player 2's name
  -side1 string
    	Player 1's Side: external, external:<configuration>@<engine> (an external Side declaring its Bot), or bot:<level> (instant, normal, thorough) (default "external")
  -side2 string
    	Player 2's Side: external, external:<configuration>@<engine> (an external Side declaring its Bot), or bot:<level> (instant, normal, thorough); two Bots play the whole match in this call (default "external")
  -start string
    	XGID of the Position the first game begins at (default: the opening position)

Examples:
  blunderdb duel create --db database.db --length 5 --name1 Alice --name2 Bob
  blunderdb duel create --db database.db --money --jacoby
  blunderdb duel create --db database.db --length 7 --side2 bot:normal
  blunderdb duel create --db database.db --length 5 --name1 Alice --side2 external:normal@v1.6.0
  blunderdb duel create --db database.db --length 3 --side1 bot:instant --side2 bot:instant
```

### `blunderdb duel discard`

```
Usage: blunderdb duel discard [options]

Throw the Duel away: nothing of it is written, and the Match is not made.

Options:
  -db string
    	Path to the database file (required)
  -format string
    	Output format: text or json (default "text")
  -id int
    	Duel id (required)
  -revision int
    	Refuse unless the Duel is at this revision (default: no check)

Examples:
  blunderdb duel discard --db database.db --id 1
```

### `blunderdb duel double`

```
Usage: blunderdb duel double [options]

Play one Action, then play on to the next Decision of an external Side. The Side defaults to the one the Duel awaits.

Options:
  -db string
    	Path to the database file (required)
  -format string
    	Output format: text or json (default "text")
  -id int
    	Duel id (required)
  -revision int
    	Refuse the Action unless the Duel is at this revision (default: no check)
  -side int
    	The Side that plays, 1 or 2 (default: the one the Duel awaits)

Examples:
  blunderdb duel double --db database.db --id 1
```

### `blunderdb duel forfeit`

```
Usage: blunderdb duel forfeit [options]

Give the match up: the game in progress goes to the other Side for the points that bring it to the length (at money play, a single at the cube's value), and the Match is written won by the other Side.

Options:
  -db string
    	Path to the database file (required)
  -format string
    	Output format: text or json (default "text")
  -id int
    	Duel id (required)
  -revision int
    	Refuse unless the Duel is at this revision (default: no check)
  -side int
    	The Side that gives the match up, 1 or 2 (required)

Examples:
  blunderdb duel forfeit --db database.db --id 1 --side 1
```

### `blunderdb duel list`

```
Usage: blunderdb duel list [options]

List the Duels in suspense, the most recently played first.

Options:
  -db string
    	Path to the database file (required)
  -format string
    	Output format: text or json (default "text")

Examples:
  blunderdb duel list --db database.db
```

### `blunderdb duel move`

```
Usage: blunderdb duel move [options]

Play one Action, then play on to the next Decision of an external Side. The Side defaults to the one the Duel awaits.

Options:
  -db string
    	Path to the database file (required)
  -format string
    	Output format: text or json (default "text")
  -id int
    	Duel id (required)
  -play string
    	The play in notation, as show lists them (required)
  -revision int
    	Refuse the Action unless the Duel is at this revision (default: no check)
  -side int
    	The Side that plays, 1 or 2 (default: the one the Duel awaits)

Examples:
  blunderdb duel move --db database.db --id 1
```

### `blunderdb duel pass`

```
Usage: blunderdb duel pass [options]

Play one Action, then play on to the next Decision of an external Side. The Side defaults to the one the Duel awaits.

Options:
  -db string
    	Path to the database file (required)
  -format string
    	Output format: text or json (default "text")
  -id int
    	Duel id (required)
  -revision int
    	Refuse the Action unless the Duel is at this revision (default: no check)
  -side int
    	The Side that plays, 1 or 2 (default: the one the Duel awaits)

Examples:
  blunderdb duel pass --db database.db --id 1
```

### `blunderdb duel resign`

```
Usage: blunderdb duel resign [options]

Play one Action, then play on to the next Decision of an external Side. The Side defaults to the one the Duel awaits.

Options:
  -db string
    	Path to the database file (required)
  -format string
    	Output format: text or json (default "text")
  -id int
    	Duel id (required)
  -level int
    	1 single, 2 gammon, 3 backgammon (default 1)
  -revision int
    	Refuse the Action unless the Duel is at this revision (default: no check)
  -side int
    	The Side that plays, 1 or 2 (default: the one the Duel awaits)

Examples:
  blunderdb duel resign --db database.db --id 1
```

### `blunderdb duel roll`

```
Usage: blunderdb duel roll [options]

Play one Action, then play on to the next Decision of an external Side. The Side defaults to the one the Duel awaits.

Options:
  -db string
    	Path to the database file (required)
  -format string
    	Output format: text or json (default "text")
  -id int
    	Duel id (required)
  -revision int
    	Refuse the Action unless the Duel is at this revision (default: no check)
  -side int
    	The Side that plays, 1 or 2 (default: the one the Duel awaits)

Examples:
  blunderdb duel roll --db database.db --id 1
```

### `blunderdb duel show`

```
Usage: blunderdb duel show [options]

Show a Duel in suspense: its score, what it awaits and, when it awaits a play, the legal ones.

Options:
  -db string
    	Path to the database file (required)
  -format string
    	Output format: text or json (default "text")
  -id int
    	Duel id (required)

Examples:
  blunderdb duel show --db database.db --id 1
```

### `blunderdb duel stop`

```
Usage: blunderdb duel stop [options]

Stop a money session and write its Match as it stands: an unfinished game keeps no winner. A match in points is refused: it is written only once won (suspend, forfeit or discard it).

Options:
  -db string
    	Path to the database file (required)
  -format string
    	Output format: text or json (default "text")
  -id int
    	Duel id (required)
  -revision int
    	Refuse unless the Duel is at this revision (default: no check)

Examples:
  blunderdb duel stop --db database.db --id 1
```

### `blunderdb duel take`

```
Usage: blunderdb duel take [options]

Play one Action, then play on to the next Decision of an external Side. The Side defaults to the one the Duel awaits.

Options:
  -db string
    	Path to the database file (required)
  -format string
    	Output format: text or json (default "text")
  -id int
    	Duel id (required)
  -revision int
    	Refuse the Action unless the Duel is at this revision (default: no check)
  -side int
    	The Side that plays, 1 or 2 (default: the one the Duel awaits)

Examples:
  blunderdb duel take --db database.db --id 1
```

### `blunderdb edit`

```
Usage: blunderdb edit [options]

Edit database metadata.

Options:
  -blunder-threshold int
    	Set the blunder threshold, in millipoints (an error costing this much or more is a blunder) (default -1)
  -clear-description
    	Clear description
  -clear-user
    	Clear user name
  -db string
    	Path to the database file (required)
  -description string
    	Set description
  -error-threshold int
    	Set the error threshold, in millipoints (a decision costing this much or more is an error) (default -1)
  -format string
    	Output format: text or json (default "text")
  -user string
    	Set user name

Examples:
  # Set user name
  blunderdb edit --db database.db --user "John Doe"

  # Set description
  blunderdb edit --db database.db --description "My positions collection"

  # Clear user name
  blunderdb edit --db database.db --clear-user

  # Set multiple values
  blunderdb edit --db database.db --user "John" --description "Tournament positions"

  # Draw the library's own lines: XG's thresholds
  blunderdb edit --db database.db --error-threshold 20 --blunder-threshold 80
```

### `blunderdb epc`

```
Usage: blunderdb epc [options] <XGID|OGID>

Compute EPC, win probability and money cube verdict for a position.
Win probability is exact inside the two-sided database domain and
estimated (with its error bound) outside; the cube verdict is only
ever shown when exact.

Options:
  -bearoff-ts string
    	Optional two-sided bearoff database (.bd) widening the TS-06-06 computed on this machine
  -format string
    	Output format: text, json (default "text")

Examples:
  # EPC and race analysis of a bearoff position
  blunderdb epc 'XGID=-BBBB----------------bbbb-:0:0:1:00:0:0:0:0:10'

  # The same position given by its OGID (OpenGammon)
  blunderdb epc 'llmmnnoo:11223344:N0N::W::0:0::'

  # With the downloaded/wider database
  blunderdb epc --bearoff-ts ~/.local/share/blunderdb/gnubg_ts6x11.bd '<XGID>'
```

### `blunderdb events`

```
Usage: blunderdb events alias <add|list|remove|suggest> --db FILE [arguments]

Record the other spellings of a event's name. An import stores the canonical
name where the file writes an alias; the stats, the players table and the
search read every spelling as one. The matches already stored keep the
names their files wrote, and so do the match fingerprints.

Actions:
  add ALIAS CANONICAL  Make ALIAS a spelling of CANONICAL
  list                 List the aliases
  remove ALIAS         Forget an alias
  suggest              Propose the names that differ only by case, accents,
                       punctuation or word order (nothing is recorded)

Options:
  --db FILE        Path to the database file (required)
  --format FORMAT  Output of list and suggest: text or json (default text)

Examples:
  blunderdb events alias add --db base.db "Doe J." "John Doe"
  blunderdb events alias list --db base.db --format json
  blunderdb events alias suggest --db base.db
```

### `blunderdb export`

```
Usage: blunderdb export [options]

Export data from the database.

Options:
  -analysis
    	Include analysis in database export (default: true) (default true)
  -collection-ids string
    	Comma-separated list of collection IDs to export
  -collections
    	Include collections in database export (default: false)
  -comments
    	Include comments in database export (default: true) (default true)
  -db string
    	Path to the database file (required)
  -deck-ids string
    	Comma-separated list of Anki deck IDs to export with their positions, without their review history
  -dir string
    	Output directory for .mat batch export (type=mat, multiple matches)
  -file string
    	Path to the output file (required)
  -filters
    	Include filter library in database export (default: true) (default true)
  -format string
    	Output format: text or json (default "text")
  -match-ids string
    	Comma-separated list of match IDs to export (empty = all)
  -matches
    	Include matches in database export (default: true) (default true)
  -password string
    	Protect the exported file with a password (produces a .dbx container)
  -played-moves
    	Include played moves in analysis (default: true) (default true)
  -tournament-ids string
    	Comma-separated list of tournament IDs to export
  -type string
    	Export type: database, positions, matches, mat (required)
  -watermark string
    	Mark the exported file with where it comes from, e.g. "Cours de Jean Dupont - 12 mars 2026"
  -watermark-note string
    	Free text attached to the watermark (terms of use, contact)

Export Types:
  database   Export entire database (positions, analysis, comments, matches)
  positions  Export positions to text file (JSON format)
  matches    Export only matches to a new database
  mat        Export match(es) as Jellyfish/gnubg .mat transcript(s)

Examples:
  # Export entire database with all matches
  blunderdb export --db database.db --type database --file export.db

  # Export database without matches
  blunderdb export --db database.db --type database --file export.db --matches=false

  # Export without analysis or played moves
  blunderdb export --db database.db --type database --file export.db --analysis=false

  # Export with analysis but without played moves
  blunderdb export --db database.db --type database --file export.db --played-moves=false

  # Export with specific collections
  blunderdb export --db database.db --type database --file export.db --collections --collection-ids=1,2,3

  # Export with specific tournaments
  blunderdb export --db database.db --type database --file export.db --tournament-ids=1,2

  # Mark the exported file with its origin, and protect it with a password
  blunderdb export --db database.db --type database --file cours.db \
      --watermark "Cours de Jean Dupont - 12 mars 2026" --password secret

  # Export positions to text file
  blunderdb export --db database.db --type positions --file positions.txt

  # Export only matches to a new database
  blunderdb export --db database.db --type matches --file matches.db

  # Export one match as a .mat transcript
  blunderdb export --db database.db --type mat --match-ids 5 --file game.mat

  # Export several (or all) matches as .mat files into a directory
  blunderdb export --db database.db --type mat --match-ids 5,9,12 --dir out/
  blunderdb export --db database.db --type mat --dir out/
```

### `blunderdb healthcheck`

```
blunderdb healthcheck — ask a running daemon whether it is ready.

Performs one GET on the daemon's /readyz endpoint and exits 0 when the answer
is 200 (storage reachable, schema version as expected), 1 otherwise: the
storage is down, the schema is stale, or nothing listens at the address. It
is what the container image's HEALTHCHECK runs — the image is distroless and
ships no curl or wget — and works just as well from a shell or a systemd unit.

The address defaults to the one the daemon itself would listen on: --addr, or
BLUNDERDB_ADDR, or :8080. A listen address with no host (":8080") or a
wildcard host (0.0.0.0, [::]) is probed on the loopback interface.

Usage:
  blunderdb healthcheck [flags]

Flags:
  -addr string
    	address the daemon listens on (host:port) (default ":8080")
  -timeout duration
    	give up after this long (default 2s)
```

### `blunderdb identity`

```
Usage: blunderdb identity [options]

Show or move your issuer identity.

The identity is created by itself the first time you issue copies; you never
have to set it up. Copies you have already issued keep verifying whatever you
do here — the name is only a label, and the key is what signs.

The exported file lets anyone holding it sign in your name. Do not share it.

Options:
  -export string
    	Write your identity to a file, to carry to another machine
  -format string
    	Output format: text or json (default "text")
  -import string
    	Install an identity file on this machine
  -name string
    	Change the display name carried by future watermarks
  -passphrase string
    	Passphrase for the exported/imported file (optional)

Examples:
  blunderdb identity
  blunderdb identity --name "Jean Dupont"
  blunderdb identity --export jean.bdbid --passphrase secret
  blunderdb identity --import jean.bdbid --passphrase secret
```

### `blunderdb import`

```
Usage: blunderdb import [options]

Import data into the database.

Options:
  -db string
    	Path to the database file (required)
  -dir string
    	Path to directory for batch import (for batch)
  -fail-on-error
    	Exit non-zero when any item failed to import (position/batch); by default only a total failure (nothing imported, duplicates aside) is an error
  -file string
    	Path to the file to import (for match/position)
  -format string
    	Output format: text or json (default "text")
  -recursive
    	Recursively scan subdirectories for batch import (default true)
  -resume int
    	With --type batch: continue the batch with this id (the id the earlier run printed, or its JSON batch_id); files with the same path, size and mtime as in its journal are skipped unread, those with the same content are read but not parsed
  -skip-duplicates
    	Skip a match already in the database outright; by default its analyses deeper than the stored ones replace them
  -type string
    	Import type: match, position, batch (required)
  -watch
    	With --type batch: keep running and import each match file as it appears in --dir (Ctrl-C to stop)
  -watch-every duration
    	How often --watch looks at the folder (default 10s, floor 2s)

Import Types:
  match     Import a single match file (.xg, .sgf, .mat, .txt, .bgf, .ogxm) or XGP position (.xgp)
  position  Import positions from a text file
  batch     Batch import all match/position files from a directory

Examples:
  # Import XG match file
  blunderdb import --db database.db --type match --file match.xg

  # Import position file
  blunderdb import --db database.db --type position --file positions.txt

  # Batch import all .xg files from a directory (recursive)
  blunderdb import --db database.db --type batch --dir ./matches/

  # Batch import (non-recursive)
  blunderdb import --db database.db --type batch --dir ./matches/ --recursive=false

  # Batch import, machine-readable, failing the run if any file errored
  blunderdb import --db database.db --type batch --dir ./matches/ --format json --fail-on-error

  # Continue batch 12, interrupted earlier: files already journaled are skipped
  blunderdb import --db database.db --type batch --dir ./matches/ --resume 12

  # Import the folder as it stands, then keep importing what appears in it
  blunderdb import --db database.db --type batch --dir ~/XG/Matches
  blunderdb import --db database.db --type batch --dir ~/XG/Matches --watch
```

### `blunderdb info`

```
Usage: blunderdb info [options]

Display database metadata and statistics.

Options:
  -db string
    	Path to the database file (required)
  -estimate
    	Estimate the tables beyond 200000 rows instead of counting them, and skip the blunders there (default: count every row)
  -format string
    	Output format: text, json (default "text")

Examples:
  # Display database info
  blunderdb info --db database.db

  # Answer at once on a very large database (estimated counts)
  blunderdb info --db database.db --estimate

  # Output as JSON
  blunderdb info --db database.db --format json

  # See where a database came from (works on a protected .dbx too)
  blunderdb info --db cours.db
```

### `blunderdb lesson add-step`

```
Usage: blunderdb lesson add-step [options]

Append a step to a lesson. A step may show a collection, a position, both or neither.

Options:
  -collection int
    	Collection the step shows (0: none)
  -db string
    	Path to the database file (required)
  -lesson int
    	Lesson ID (required)
  -position int
    	Position the step shows (0: none)
  -text string
    	Step text
  -text-file string
    	Read the step text from this file (wins over --text)
  -title string
    	Step title

Examples:
  blunderdb lesson add-step --db database.db --lesson 1 --title "Timing" --text-file timing.txt --collection 3
  blunderdb lesson add-step --db database.db --lesson 1 --title "The key position" --position 412
```

### `blunderdb lesson create`

```
Usage: blunderdb lesson create [options]

Create an empty lesson.

Options:
  -db string
    	Path to the database file (required)
  -description string
    	Lesson description
  -name string
    	Lesson name (required)

Examples:
  blunderdb lesson create --db database.db --name "Playing against a prime"
```

### `blunderdb lesson delete`

```
Usage: blunderdb lesson delete [options]

Delete a lesson and its steps; the collections and positions they show stay.

Options:
  -confirm
    	Confirm the deletion (required)
  -db string
    	Path to the database file (required)
  -id int
    	Lesson ID (required)

Examples:
  blunderdb lesson delete --db database.db --id 1 --confirm
```

### `blunderdb lesson done`

```
Usage: blunderdb lesson done [options]

Mark a step done, the reader's own progress (ADR-0069). It is written only by this gesture, in this database; no export carries it.

Options:
  -db string
    	Path to the database file (required)
  -step int
    	Step ID (required)
  -undo
    	Withdraw the mark instead of setting it

Examples:
  blunderdb lesson done --db database.db --step 7
  blunderdb lesson done --db database.db --step 7 --undo
```

### `blunderdb lesson edit`

```
Usage: blunderdb lesson edit [options]

Rename a lesson or change its description; flags not given keep their value.

Options:
  -db string
    	Path to the database file (required)
  -description string
    	New description
  -id int
    	Lesson ID (required)
  -name string
    	New name

Examples:
  blunderdb lesson edit --db database.db --id 1 --name "Primes"
```

### `blunderdb lesson edit-step`

```
Usage: blunderdb lesson edit-step [options]

Change a step; flags not given keep their value, 0 clears a collection or a position.

Options:
  -collection int
    	Collection the step shows (0: none)
  -db string
    	Path to the database file (required)
  -lesson int
    	Lesson ID (required)
  -position int
    	Position the step shows (0: none)
  -step lesson show
    	Step ID, as lesson show prints it (required)
  -text string
    	New text
  -text-file string
    	Read the new text from this file (wins over --text)
  -title string
    	New title

Examples:
  blunderdb lesson edit-step --db database.db --lesson 1 --step 7 --text "Count again."
  blunderdb lesson edit-step --db database.db --lesson 1 --step 7 --position 0
```

### `blunderdb lesson export`

```
Usage: blunderdb lesson export [options]

Export lessons, with the collections and positions their steps show, to a new database file.

Options:
  -analysis
    	Include analyses (default true)
  -comments
    	Include comments (default true)
  -db string
    	Path to the database file (required)
  -id string
    	Lesson ID(s) to export, comma-separated (required)
  -out string
    	Path of the database file to write (required)
  -password string
    	Encrypt the export into a protected .dbx file
  -watermark string
    	Mark the exported file with where it comes from
  -watermark-note string
    	Free text attached to the watermark (terms of use, contact)

Examples:
  blunderdb lesson export --db database.db --id 1 --out lesson.db --watermark "Course of 12 March"
  blunderdb lesson export --db database.db --id 1,2 --out lessons.dbx --password secret
```

### `blunderdb lesson list`

```
Usage: blunderdb lesson list [options]

List the lessons of the database.

Options:
  -db string
    	Path to the database file (required)
  -format string
    	Output format: text or json (default "text")

Examples:
  blunderdb lesson list --db database.db --format json
```

### `blunderdb lesson progress`

```
Usage: blunderdb lesson progress [options]

Show which steps of a lesson are marked done, and the date of the gesture.

Options:
  -db string
    	Path to the database file (required)
  -format string
    	Output format: text or json (default "text")
  -id int
    	Lesson ID (required)

Examples:
  blunderdb lesson progress --db database.db --id 1
  blunderdb lesson progress --db database.db --id 1 --format json
```

### `blunderdb lesson remove-step`

```
Usage: blunderdb lesson remove-step [options]

Remove a step from its lesson.

Options:
  -db string
    	Path to the database file (required)
  -step int
    	Step ID (required)

Examples:
  blunderdb lesson remove-step --db database.db --step 7
```

### `blunderdb lesson reorder`

```
Usage: blunderdb lesson reorder [options]

Set the order of a lesson's steps; --steps names every step once.

Options:
  -db string
    	Path to the database file (required)
  -lesson int
    	Lesson ID (required)
  -steps string
    	Step IDs in the new order, comma-separated (required)

Examples:
  blunderdb lesson reorder --db database.db --lesson 1 --steps 9,7,8
```

### `blunderdb lesson show`

```
Usage: blunderdb lesson show [options]

Show a lesson's steps in reading order.

Options:
  -db string
    	Path to the database file (required)
  -format string
    	Output format: text or json (default "text")
  -id int
    	Lesson ID (required)

Examples:
  blunderdb lesson show --db database.db --id 1
  blunderdb lesson show --db database.db --id 1 --format json
```

### `blunderdb list`

```
Usage: blunderdb list [options]

List database contents.

Options:
  -batch int
    	Show the full report of one import batch instead of the list (imports only)
  -days int
    	Window, in days, for --type study (default 30)
  -db string
    	Path to the database file (required)
  -decision-type string
    	Decision type: all, checker, or cube (stats only) (default "all")
  -engine string
    	Only the decisions analysed by this engine (exact name, as stored)
  -format string
    	Output format: text, json or csv (stats, players and imports only) (default "text")
  -from string
    	Start date filter YYYY-MM-DD (stats only)
  -limit int
    	Maximum number of items to list (default 10)
  -metric string
    	Metric to display: pr or mwc (stats only) (default "pr")
  -min-depth int
    	Only the decisions analysed at least this deep (plies)
  -offset int
    	Number of positions to skip before listing (positions only)
  -player string
    	Filter by player name (stats only)
  -query string
    	With --type matches: keep matches whose players, event, location, round, tournament or date contain this text
  -queue
    	With --type imports --batch <id>: the study queue that follows the report — what to look at now, in order
  -sort string
    	With --type matches: date (default), date_asc, length_asc, length_desc, player1, player1_desc, player2, player2_desc, tournament, tournament_desc, opponent
  -to string
    	End date filter YYYY-MM-DD (stats only)
  -top-blunders int
    	Number of top blunders to show (stats only) (default 10)
  -tournament string
    	Filter by tournament IDs, comma-separated (stats only)
  -type string
    	List type: matches, tournaments, positions, moves, analyses, imports, stats, players, timeerrors, tags, study (required)

Examples:
  # List all matches
  blunderdb list --db database.db --type matches

  # List all tournaments
  blunderdb list --db database.db --type tournaments

  # List first 20 positions
  blunderdb list --db database.db --type positions --limit 20

  # List the recorded imports, then read one's report
  blunderdb list --db database.db --type imports
  blunderdb list --db database.db --type imports --batch 3

  # What to look at now, in the order to look at it
  blunderdb list --db database.db --type imports --batch 3 --queue

  # Show database statistics
  blunderdb list --db database.db --type stats

  # Show stats as JSON
  blunderdb list --db database.db --type stats --format json

  # Show stats in MWC with player filter
  blunderdb list --db database.db --type stats --metric mwc --player "Alice"

  # One statistics row per player, over a competition's dates
  blunderdb list --db database.db --type players --from 2026-03-01 --to 2026-03-08

  # The same table as CSV, for a spreadsheet or a script
  blunderdb list --db database.db --type players --format csv

  # The tag vocabulary of this database, most used first
  blunderdb list --db database.db --type tags

  # Tabular exports, for a notebook or a spreadsheet
  blunderdb list --db database.db --type positions --format csv > positions.csv
  blunderdb list --db database.db --type moves     --format csv > moves.csv
  blunderdb list --db database.db --type analyses  --format csv > analyses.csv
```

### `blunderdb match`

```
Usage: blunderdb match [options]

Display match positions and analysis.

Options:
  -db string
    	Path to the database file (required)
  -format string
    	Output format: json, text, summary (default "json")
  -id int
    	Match ID (required)
  -output string
    	Output file (default: stdout)

Examples:
  # Display match positions in JSON format
  blunderdb match --db database.db --id 1 --format json

  # Display match summary
  blunderdb match --db database.db --id 1 --format summary

  # Save match positions to file
  blunderdb match --db database.db --id 1 --output match.json
```

### `blunderdb mcp`

```
Usage: blunderdb mcp --db <file> [options]

Serve a database's tools to an AI assistant over the Model Context Protocol,
on stdin/stdout. The assistant starts this command itself; for Claude Code:

  claude mcp add blunderdb -- blunderdb mcp --db /path/to/my.db

The tools search positions with the application's query grammar, read a
position and its analysis, explain an error, compute a player's statistics,
list matches, tournaments and collections, and run a quiz. They only read,
unless --write is given: then they may also save a position, comment one,
create and fill a collection.

Options:
  -db string
    	Path to the database file (required)
  -write
    	Also offer the tools that change the database

Examples:
  blunderdb mcp --db my.db
  blunderdb mcp --db my.db --write
```

### `blunderdb met`

```
Usage: blunderdb met [options]

List, import or choose the match equity table (MET) of a database.
Each database has its own table, Kazaross-XG2 by default. gammonNet
values match scores with the current table, and every analysis it
computes records that table. An analysis at a match score valued with
another table is shown as "different MET" and left out of the
statistics. Changing the current table rewrites no analysis.
Only explicit gnubg tables are read (not the parametric zadeh or mec).

Options:
  -current
    	With --import: make the imported table current
  -db string
    	Path to the database file (required)
  -format string
    	Output format: text or json (default "text")
  -import string
    	Import a gnubg match equity table (.xml)
  -use int
    	Make table ID current (0: the built-in Kazaross-XG2) (default -1)

Examples:
  blunderdb met --db database.db
  blunderdb met --db database.db --import Rockwell-Kazaross.xml --current
  blunderdb met --db database.db --use 0
```

### `blunderdb open`

```
Usage: blunderdb open [options]

Open a password-protected copy into an ordinary database file.

You are asked for the password once. The result is a normal blunderDB
database you work with as usual.

Options:
  -db string
    	Path to the protected copy (required)
  -file string
    	Where to write the opened database (default: alongside, with .db)
  -password string
    	The password you were given (required)

Example:
  blunderdb open --db cours.dbx --password secret
```

### `blunderdb players`

```
Usage: blunderdb players alias <add|list|remove|suggest> --db FILE [arguments]

Record the other spellings of a player's name. An import stores the canonical
name where the file writes an alias; the stats, the players table and the
search read every spelling as one. The matches already stored keep the
names their files wrote, and so do the match fingerprints.

Actions:
  add ALIAS CANONICAL  Make ALIAS a spelling of CANONICAL
  list                 List the aliases
  remove ALIAS         Forget an alias
  suggest              Propose the names that differ only by case, accents,
                       punctuation or word order (nothing is recorded)

Options:
  --db FILE        Path to the database file (required)
  --format FORMAT  Output of list and suggest: text or json (default text)

Examples:
  blunderdb players alias add --db base.db "Doe J." "John Doe"
  blunderdb players alias list --db base.db --format json
  blunderdb players alias suggest --db base.db

Also: `players merge --db FILE --into CANONICAL NAME...` and
`players swap --db FILE MATCH_ID` (see their --help).
```

### `blunderdb reencode`

```
Usage: blunderdb reencode [options]

Rewrite the analyses an older release stored as JSON in the compact
binary format, which is about half the size and reads several times
faster. Old analyses stay readable without it; vacuum does the same
conversion. It works in batches: interrupted, it resumes where it
stopped on the next run. It never runs automatically.

Options:
  -db string
    	Path to the database file (required)
  -format string
    	Output format: text or json (default "text")

Examples:
  blunderdb reencode --db database.db
  blunderdb reencode --db database.db --format json
```

### `blunderdb repair`

```
Usage: blunderdb repair [options]

Recompute what the database derives from what it stores:
the scalar columns of every analysis, from the JSON they are
a projection of, the phase of every position, from its board,
and the Crawford sentinel of every away score, from the match
the position came from or the XGID it came in with from
another program. Useful after a fix to how an imported
analysis is read, after a change to how a phase is decided,
and once, for the databases imported before the importers
wrote the sentinel: a post-Crawford position stored as a
Crawford one is read cube-dead. Correcting it changes the
position's Zobrist hash, so such a position is rehashed and
merged with its correct twin when the database holds one.
Nothing runs it automatically.

Options:
  -db string
    	Path to the database file (required)
  -duplicates
    	Only list the pairs of matches whose dice say they are one: same dice under other player names, or a truncated match and its longer version (nothing is merged)
  -format string
    	Output format: text or json (default "text")
  -move-errors
    	Also write the equity error of every played move (stored in the move table) from its position's analysis; resumable, long on a large library
  -stats
    	Also recompute the per-match statistics (PR, decisions, blunders, luck of each seat) from scratch

Examples:
  blunderdb repair --db database.db
  blunderdb repair --db database.db --format json
  blunderdb repair --db database.db --stats
  blunderdb repair --db database.db --move-errors
  blunderdb repair --db database.db --duplicates
```

### `blunderdb rollout`

```
Usage: blunderdb rollout [options] <XGID|OGID>
       blunderdb rollout --db <file> --id <position> [--store] [options]

Roll a position out with gammonNet: its plays when the position has dice,
its cube decision otherwise. Each candidate plays the same dice; the luck
of every roll is taken out of each game (variance reduction); the first two
rolls are stratified; a game stops where the two-sided bearoff table
covers it. The cube is played inside the games (cubeful): trust the
ranking more than the absolute equity.

Equities are money points per unit of the position's cube, or normalised
equity at a match score. Ctrl-C prints what the games finished so far
concluded. With --store, a finished rollout is written on the position as a
second analysis with its own settings, beside the imported or evaluated one;
an interrupted rollout is never stored. Of two rollouts with the same
settings, the one with more games is kept (a tie keeps the newer).

Options:
  -candidates int
    	Plays rolled out when no --move is given, best first at max(--ply, 2) (0 = the preset's)
  -db string
    	Database to read the position from (with --id)
  -format string
    	Output format: text, json (default "text")
  -games int
    	Most games per candidate (0 = the preset's)
  -id int
    	Position of --db to roll out, in place of an XGID or OGID
  -jobs int
    	Games played at once (0 = one per core); never changes the numbers
  -jsd float
    	Stop a candidate when its gap to the best reaches this many standard deviations; 0 never stops early (-1 = the preset's) (default -1)
  -list
    	Print the rollouts stored on the position of --db instead of rolling it out
  -min-games int
    	Games before the JSD rule may stop a candidate (-1 = the preset's) (default -1)
  -move value
    	A play to roll out, in blunderDB notation (repeatable)
  -ply int
    	gammonNet depth of the plays, cube actions and leaves inside the games (-1 = the preset's) (default -1)
  -preset string
    	Starting settings, both at 0 ply: fast (216 games, truncated at 7) or standard (1296 games, truncated at 11); the flags below override it, --ply plays deeper (default "fast")
  -seed uint
    	Dice seed: the same seed gives the same numbers (default 104374970738651)
  -store
    	Write the finished rollout on the position of --db, beside its analysis (never replacing it)
  -truncation int
    	Half-moves per game before the engine values it; 0 plays to the end (-1 = the preset's) (default -1)

Examples:
  blunderdb rollout 'XGID=-b----E-C---eE---c-e----B-:0:0:1:31:0:0:0:0:10'
  blunderdb rollout --move '8/5 6/5' --move '24/23 13/10' '<XGID>'
  blunderdb rollout --preset standard --format json '<XGID>'
  blunderdb rollout --games 648 --truncation 0 --ply 1 '<XGID>'
  blunderdb rollout --db library.db --id 42 --preset standard --store
  blunderdb rollout --db library.db --id 42 --list
```

### `blunderdb search`

```
Usage: blunderdb search [options]

Search for positions in the database using filters.

Options:
  -analysis string
    	Engines and depths of the stored verdict, comma-separated: xg, gnubg, bgblitz, hedgehog, gammonnet, 3ply, 3ply+, book, rollout (ad:)
  -comment-author string
    	Only positions carrying a comment signed by this author (whole name, any case)
  -comment-origin string
    	Only positions carrying a comment from these origins, comma-separated: user, xg, gnubg, bgf, unknown
  -cube int
    	Filter by cube value
  -cube-response string
    	Only one kind of cube decision: double (double/no double) or takepass (take/pass)
  -db string
    	Path to the database file (required)
  -decision string
    	Filter by decision type: checker, cube
  -dice string
    	Filter by dice roll: '5,3' matches both dice (any order); '5' matches positions where 5 was rolled on either die
  -error-min float
    	Minimum equity error (blunders)
  -export string
    	Export results to a new database file
  -flagged
    	Only positions you marked for study in the source tool (eXtreme Gammon flags)
  -format string
    	Output format: table, json, xgid (default "table")
  -game-type blunderdb repair
    	Only positions in these plans of play, comma-separated: race, bearin, crunch, backgame, acepoint, blitz, primevprime, mutualholding, holding, contact (derived label, see blunderdb repair)
  -has-analysis
    	Only positions with analysis
  -has-comment
    	Only positions carrying a comment (whatever its origin — yours or an imported note)
  -individual
    	Only positions imported on their own, not as part of a match
  -limit int
    	Maximum number of results (0 = no limit)
  -match-date string
    	Date of the match: '2024', '2024-01..2024-12', '>2024-06', '<2024-06' (md; not the analysis date)
  -match-ids string
    	Filter by match IDs: comma-separated list e.g. '1,3,5', OR a two-value range e.g. '2,7' (2 through 7), OR a semicolon list e.g. '2;7'
  -match-length int
    	Filter by match length
  -match-lengths string
    	Length of the match the position was met in: '7', '5,9' (5 to 9), '>5', '<9' (ml)
  -move-error-max float
    	Maximum played move error (millipoints)
  -move-error-min float
    	Minimum played move error (millipoints)
  -no-comment
    	Only positions carrying no comment
  -off1-min int
    	Minimum checkers off for player 1
  -off2-min int
    	Minimum checkers off for player 2
  -offset int
    	Skip this many results before the first one returned (paging, with --limit)
  -opponent string
    	With --player, only the matches against this opponent; alone, a name at either seat (op"…")
  -phase blunderdb repair
    	Only positions in these game phases, comma-separated: opening, middlegame, race, bearoff (derived label, see blunderdb repair)
  -pip-max int
    	Maximum pip count difference
  -pip-min int
    	Minimum pip count difference
  -player string
    	Only matches this player sat in (case-insensitive, '*' as a wildcard); same as the pl"…" token
  -position-ids string
    	Filter by position IDs (range '2,7' or explicit list '5;10;15')
  -pr string
    	PR of the whole match for the player who took the decision: '>8', '<5', '4,9' (pr)
  -query string
    	Search with the interface's own query language, e.g. 's cube p>30 E>0.05' (see --query-help); exclusive with the filter flags
  -query-help
    	List the tokens --query understands, and exit
  -round string
    	Only matches of these rounds, comma-separated ('*' as a wildcard; rd:)
  -score1 int
    	Filter by player 1 score (default -1)
  -score2 int
    	Filter by player 2 score (default -1)
  -seat-only
    	With --player: only the decisions that player took (pl!"…")
  -tournament-ids string
    	Filter by tournament IDs: comma-separated list e.g. '1,3,5', OR a two-value range e.g. '2,7' (2 through 7), OR a semicolon list e.g. '2;7'
  -tournament-name string
    	Only matches of tournaments with this name (case-insensitive, '*' as a wildcard; tn"…")
  -winrate-max float
    	Maximum win rate (%)
  -winrate-min float
    	Minimum win rate (%)

Examples:
  # List all positions
  blunderdb search --db database.db

  # Search cube decisions
  blunderdb search --db database.db --decision cube

  # Search positions with errors >= 0.1
  blunderdb search --db database.db --error-min 0.1

  # Search and export to new database
  blunderdb search --db database.db --decision cube --export cubes.db

  # Search bearoff positions
  blunderdb search --db database.db --off1-min 1 --off2-min 1

  # Output as JSON
  blunderdb search --db database.db --format json --limit 10

  # Search in specific matches (2, 5, and 9)
  blunderdb search --db database.db --match-ids 2,5,9

  # Search in a tournament
  blunderdb search --db database.db --tournament-ids 1

  # Search positions where dice were 6-5
  blunderdb search --db database.db --dice 6,5

  # Find the positions you imported yourself, not the ones matches brought in
  blunderdb search --db database.db --individual

  # Search positions where a 6 was rolled on either die
  blunderdb search --db database.db --dice 6

  # Positions flagged for study in XG
  blunderdb search --db database.db --flagged

  # Find every commented position
  blunderdb search --db database.db --has-comment

  # Blunders still waiting to be annotated
  blunderdb search --db database.db --no-comment --error-min 0.1

  # The interface's own query language: cube decisions, 30+ pips behind, 50 millipoints of error
  blunderdb search --db database.db --query 's cube p>30 E>50'

  # One player's own decisions in 7-point matches of 2024, analysed at 3 plies or more
  blunderdb search --db database.db --player Alice --seat-only --match-lengths 7 --match-date 2024 --analysis 3ply+

  # Filters no flag exposes: a move pattern, a comment tag, an analysis date
  blunderdb search --db database.db --query 's m"13/11" t"blunder" pl"Alice" T>2026/01/01'
```

### `blunderdb stats breakdown`

```
Usage: blunderdb stats breakdown --db <file> [options]

The Breakdowns tab of the Stats panel: the PR of the filter split by game
phase, plan of play, comment tag, score (away x away) and cube action, and
the direction in which the cube decisions went wrong.

Options:
  -db string
    	Path to the database file (required)
  -decision-type string
    	Decision type: all, checker, or cube (default "all")
  -format string
    	Output format: text or json (default "text")
  -from string
    	Start date filter YYYY-MM-DD
  -player string
    	Only this player's decisions
  -to string
    	End date filter YYYY-MM-DD
  -tournament string
    	Filter by tournament IDs, comma-separated

Examples:
  blunderdb stats breakdown --db database.db --player "Alice"
  blunderdb stats breakdown --db database.db --format json
```

### `blunderdb stats contrast`

```
Usage: blunderdb stats contrast --db <file> --player <name> --opponent <name> [options]

The positions both players decided, whoever they played against, where one
played well and the other did not: the widest gap first. A player's error on
a position is their worst play of it; "well" means below the library's Error
threshold. Open the positions with --format json (position_id).

Only scored plays are compared, and no import scores them: run
`blunderdb repair --db <file> --move-errors` first. The count of plays still
unscored is printed (unscored_moves in JSON).

Options:
  -db string
    	Path to the database file (required)
  -decision-type string
    	Decision type: all, checker, or cube (default "all")
  -engine string
    	Only the decisions analysed by this engine (exact name, as stored)
  -format string
    	Output format: text or json (default "text")
  -from string
    	Start date filter YYYY-MM-DD (matches)
  -limit int
    	Maximum number of positions shown (text only; 0 = all) (default 20)
  -min-depth int
    	Only the decisions analysed at least this deep (plies)
  -opponent string
    	Second player (required)
  -player string
    	First player (required)
  -to string
    	End date filter YYYY-MM-DD (matches)
  -tournament string
    	Filter the matches by tournament IDs, comma-separated

Examples:
  blunderdb stats contrast --db database.db --player "Alice" --opponent "Bob"
  blunderdb stats contrast --db database.db --player "Alice" --opponent "Bob" --format json
```

### `blunderdb stats h2h`

```
Usage: blunderdb stats h2h --db <file> --player <name> --opponent <name> [options]

The matches the two players played against each other, each one's PR in
each match and over all of them, and the record (decided matches only).

Options:
  -db string
    	Path to the database file (required)
  -decision-type string
    	Decision type: all, checker, or cube (default "all")
  -engine string
    	Only the decisions analysed by this engine (exact name, as stored)
  -format string
    	Output format: text or json (default "text")
  -from string
    	Start date filter YYYY-MM-DD (matches)
  -min-depth int
    	Only the decisions analysed at least this deep (plies)
  -opponent string
    	Second player (required)
  -player string
    	First player (required)
  -to string
    	End date filter YYYY-MM-DD (matches)
  -tournament string
    	Filter the matches by tournament IDs, comma-separated

Examples:
  blunderdb stats h2h --db database.db --player "Alice" --opponent "Bob"
  blunderdb stats h2h --db database.db --player "Alice" --opponent "Bob" --format json
```

### `blunderdb stats plan`

```
Usage: blunderdb stats plan --db <file> [options]

Answer "what should I work on now?": the recurring-error families (plan of play
x theme) ranked by the winning chances studying them would recover (ADR-0077).
A family's recoverable MWC is the sum, over its errors, of the loss minus the
difficulty: what a reference player would have lost in the same positions.
It comes with a 95% interval; a family enters the plan with at least
5 priced errors and an interval above zero, ranked by the interval's lower
bound. The others are listed apart, to confirm. Money-play errors carry no
MWC and are only counted (unpriced).

Options:
  -db string
    	Path to the database file (required)
  -decision-type string
    	Decision type: all, checker, or cube (default "all")
  -deck string
    	Create an Anki deck of this name from the positions of the plan's first families
  -family int
    	With --quiz, --deck or --queue: the rank of one family (1 = first) instead of the first three
  -format string
    	Output format: text or json (default "text")
  -from string
    	Start date filter YYYY-MM-DD
  -limit int
    	Maximum number of families shown (text only; 0 = all) (default 10)
  -player string
    	Only this player's decisions
  -queue
    	List the study queue of the plan's first families, the largest excess first
  -quiz
    	Draw a quiz: position ids picked at random from the plan's first families
  -quiz-size int
    	Number of positions --quiz draws (default 20)
  -to string
    	End date filter YYYY-MM-DD
  -tournament string
    	Filter by tournament IDs, comma-separated

Examples:
  blunderdb stats plan --db database.db --player "Alice"
  blunderdb stats plan --db database.db --player "Alice" --quiz --format json
  blunderdb stats plan --db database.db --family 1 --deck "Plan: first family"
  blunderdb stats plan --db database.db --family 2 --queue
```

### `blunderdb stats progression`

```
Usage: blunderdb stats progression --db <file> [options]

The Progression tab of the Stats panel: the PR of each match in date order,
each tournament's PR, and the rolling PR over the last N decisions.

Options:
  -db string
    	Path to the database file (required)
  -decision-type string
    	Decision type: all, checker, or cube (default "all")
  -format string
    	Output format: text or json (default "text")
  -from string
    	Start date filter YYYY-MM-DD
  -player string
    	Only this player's decisions
  -to string
    	End date filter YYYY-MM-DD
  -tournament string
    	Filter by tournament IDs, comma-separated

Examples:
  blunderdb stats progression --db database.db --player "Alice"
  blunderdb stats progression --db database.db --format json
```

### `blunderdb stats ranking`

```
Usage: blunderdb stats ranking --db <file> [options]

The players ranked by PR, the lowest first, among those with at least
--min-decisions counted decisions: a PR over a few decisions is noise.
Equal PRs share a rank.

Options:
  -db string
    	Path to the database file (required)
  -decision-type string
    	Decision type: all, checker, or cube (default "all")
  -engine string
    	Only the decisions analysed by this engine (exact name, as stored)
  -format string
    	Output format: text or json (default "text")
  -from string
    	Start date filter YYYY-MM-DD (matches)
  -limit int
    	Show only the first N ranks (0: all)
  -min-decisions int
    	Rank only the players with at least this many counted decisions (default 500)
  -min-depth int
    	Only the decisions analysed at least this deep (plies)
  -to string
    	End date filter YYYY-MM-DD (matches)
  -tournament string
    	Filter the matches by tournament IDs, comma-separated

Examples:
  blunderdb stats ranking --db database.db --min-decisions 1000 --limit 20
  blunderdb stats ranking --db database.db --from 2024-01-01 --format json
```

### `blunderdb stats recurring`

```
Usage: blunderdb stats recurring --db <file> [options]

Group the errors of the filter by plan of play and theme, costliest first.
A checker theme is the reason the explanation rules name (gammon, blots,
point, passive); a cube theme is the direction of the cube error. An error
no rule names is listed apart, per plan of play, outside the ranking (JSON:
"Unthemed"): the rules only speak from 60 mp, above the Error threshold.
Cost is the share of the filter's PR the group accounts for; an error is a
counted decision costing at least the library's Error threshold.

--quiz and --deck turn the ranking into study: --quiz draws positions at
random from the three costliest groups (or the one --group names), --deck
makes an Anki deck of all their positions.

Options:
  -db string
    	Path to the database file (required)
  -decision-type string
    	Decision type: all, checker, or cube (default "all")
  -deck string
    	Create an Anki deck of this name from the positions of the worst groups
  -format string
    	Output format: text or json (default "text")
  -from string
    	Start date filter YYYY-MM-DD
  -group int
    	With --quiz or --deck: the rank of one group (1 = costliest) instead of the three costliest
  -limit int
    	Maximum number of groups shown (text only; 0 = all) (default 20)
  -player string
    	Only this player's decisions
  -quiz
    	Draw a quiz: position ids picked at random from the worst groups, for the Decision exercise or quiz_grade
  -quiz-size int
    	Number of positions --quiz draws (default 20)
  -to string
    	End date filter YYYY-MM-DD
  -tournament string
    	Filter by tournament IDs, comma-separated

Examples:
  blunderdb stats recurring --db database.db --player "Alice"
  blunderdb stats recurring --db database.db --quiz --format json
  blunderdb stats recurring --db database.db --group 1 --deck "My worst group"
  blunderdb stats recurring --db database.db --decision-type checker --format json
```

### `blunderdb stats report`

```
Usage: blunderdb stats report --db <file> --html [options]

The HTML report of the Stats panel: the filter's indicators and its ten most
expensive decisions with their diagrams, as one self-contained file (no image,
style sheet or script outside it) that a browser prints to PDF.

Options:
  -db string
    	Path to the database file (required)
  -decision-type string
    	Decision type: all, checker, or cube (default "all")
  -format string
    	Output format: text or json (default "text")
  -from string
    	Start date filter YYYY-MM-DD
  -html
    	Write the report as one self-contained HTML file (the only format)
  -lang string
    	Report language: fr, en, de, el, es, fi, it, ja, ru (default "en")
  -output string
    	File to write (default: standard output)
  -player string
    	Only this player's decisions
  -to string
    	End date filter YYYY-MM-DD
  -tournament string
    	Filter by tournament IDs, comma-separated

Examples:
  blunderdb stats report --db database.db --html --output rapport.html
  blunderdb stats report --db database.db --html --player "Alice" --lang fr
```

### `blunderdb stats training`

```
Usage: blunderdb stats training --db <file> [options]

The Decision quiz PR, the real PR of the matches and the Anki retention,
folded by calendar window so the three can be read side by side. The quiz
PR is on the real PR's scale; the retention is the share of review-state
card reviews rated Hard or better. Each series carries its own count: a
window with no decision has a count of 0, not a value. The filter options
restrict the matches only; the quiz and Anki journals carry no player.

Options:
  -db string
    	Path to the database file (required)
  -decision-type string
    	Decision type of the matches: all, checker, or cube (default "all")
  -format string
    	Output format: text or json (default "text")
  -from string
    	Start date filter YYYY-MM-DD (matches)
  -player string
    	Only this player's matches (the real PR series)
  -to string
    	End date filter YYYY-MM-DD (matches)
  -tournament string
    	Filter the matches by tournament IDs, comma-separated
  -window string
    	Calendar window: week or month (default "week")

Examples:
  blunderdb stats training --db database.db --player "Alice"
  blunderdb stats training --db database.db --window month --format json
```

### `blunderdb stats windows`

```
Usage: blunderdb stats windows --db <file> [options]

The PR over a sliding calendar window, one line per month: each line covers
that month and the ones before it in the window. A window without a counted
decision shows a dash, not a PR.

Options:
  -db string
    	Path to the database file (required)
  -decision-type string
    	Decision type: all, checker, or cube (default "all")
  -engine string
    	Only the decisions analysed by this engine (exact name, as stored)
  -format string
    	Output format: text or json (default "text")
  -from string
    	Start date filter YYYY-MM-DD (matches)
  -min-depth int
    	Only the decisions analysed at least this deep (plies)
  -player string
    	Only this player's decisions
  -to string
    	End date filter YYYY-MM-DD (matches)
  -tournament string
    	Filter the matches by tournament IDs, comma-separated
  -window string
    	Sliding window: month, quarter, or a number of months (default "month")

Examples:
  blunderdb stats windows --db database.db --player "Alice"
  blunderdb stats windows --db database.db --player "Alice" --window quarter --format json
```

### `blunderdb study mark`

```
Usage: blunderdb study mark [options]

Mark a position studied: it leaves the study backlog.

Options:
  -db string
    	Path to the database file (required)
  -id int
    	Position ID (required)

Examples:
  blunderdb study mark --db database.db --id 1234
```

### `blunderdb study queue`

```
Usage: blunderdb study queue [options]

List your unhandled blunders across the whole library, costliest first.
Your name is the database's reference player (see `blunderdb players`).

Options:
  -db string
    	Path to the database file (required)
  -format string
    	Output format: text or json (default "text")
  -limit int
    	Maximum number of positions (default and ceiling: 50)

Examples:
  blunderdb study queue --db database.db
  blunderdb study queue --db database.db --limit 10 --format json
```

### `blunderdb study unmark`

```
Usage: blunderdb study unmark [options]

Withdraw a position's studied mark: it returns to the study backlog.

Options:
  -db string
    	Path to the database file (required)
  -id int
    	Position ID (required)

Examples:
  blunderdb study unmark --db database.db --id 1234
```

### `blunderdb tournament confirm`

```
Usage: blunderdb tournament confirm [options]

Confirm one proposal, by its number in `tournament proposals`, and print the queue that follows.

Options:
  -db string
    	Path to the database file (required)
  -format string
    	Output format: text or json (default "text")
  -id int
    	Tournament ID (required)
  -n int
    	Number of the proposal, as printed by the proposals sub-command (required)

Examples:
  blunderdb tournament confirm --db base.db --id 3 --n 1
```

### `blunderdb tournament export`

```
Usage: blunderdb tournament export [options]

Print the raw event journal of a direction.

Options:
  -db string
    	Path to the database file (required)
  -id int
    	Tournament ID (required)

Examples:
  blunderdb tournament export --db base.db --id 3 > journal.json
```

### `blunderdb tournament hall`

```
Usage: blunderdb tournament hall [options]

Print a Rencontre's tables, every event together, then the proposals of every event in one queue and the matches held for want of a table.

Options:
  -db string
    	Path to the database file (required)
  -format string
    	Output format: text or json (default "text")
  -rencontre int
    	Rencontre ID (required)

Examples:
  blunderdb tournament hall --db base.db --rencontre 1
  blunderdb tournament hall --db base.db --rencontre 1 --format json
```

### `blunderdb tournament list`

```
Usage: blunderdb tournament list [options]

List the directed tournaments of the database.

Options:
  -db string
    	Path to the database file (required)
  -format string
    	Output format: text or json (default "text")

Examples:
  blunderdb tournament list --db base.db
  blunderdb tournament list --db base.db --format json
```

### `blunderdb tournament move`

```
Usage: blunderdb tournament move [options]

Move a running match to another table; a taken table swaps the two matches.

Options:
  -db string
    	Path to the database file (required)
  -format string
    	Output format: text or json (default "text")
  -id int
    	Tournament ID (required)
  -match string
    	Running match ID (required)
  -table int
    	Destination table number (required)

Examples:
  blunderdb tournament move --db base.db --id 3 --match m4 --table 7
  blunderdb tournament move --db base.db --id 3 --match m4 --table 2 --format json
```

### `blunderdb tournament page`

```
Usage: blunderdb tournament page [options]

Write the standalone display page of a direction, or a Rencontre's wall page.

Options:
  -db string
    	Path to the database file (required)
  -id int
    	Tournament ID
  -out string
    	Folder to write the page into (default: standard output)
  -rencontre int
    	Rencontre ID: write its wall page instead of one tournament's

Examples:
  blunderdb tournament page --db base.db --id 3 > affichage.html
  blunderdb tournament page --db base.db --id 3 --out /tmp/affichage
  blunderdb tournament page --db base.db --rencontre 1 --out /tmp/salle
```

### `blunderdb tournament proposals`

```
Usage: blunderdb tournament proposals [options]

Print the engine's proposals for a tournament, numbered for `tournament confirm`.

Options:
  -db string
    	Path to the database file (required)
  -format string
    	Output format: text or json (default "text")
  -id int
    	Tournament ID (required)

Examples:
  blunderdb tournament proposals --db base.db --id 3
  blunderdb tournament proposals --db base.db --id 3 --format json
```

### `blunderdb tournament ranking`

```
Usage: blunderdb tournament ranking [options]

Print a season ranking: the finished tournaments of a Rencontre or a period, scored by place.

Options:
  -db string
    	Path to the database file (required)
  -elo
    	Add a club Elo replayed over the season's matches (FIBS formula, start 1500)
  -format string
    	Output format: csv or json (default "csv")
  -from string
    	First tournament date, YYYY-MM-DD, inclusive
  -participation float
    	Points added for every finished tournament played
  -points string
    	Points by place, comma-separated, winner first (default 25,18,15,12,10,8,6,4,2,1)
  -rencontre int
    	Only the tournaments of this Rencontre
  -season standings
    	Rank over several tournaments (required: a single tournament is standings)
  -to string
    	Last tournament date, YYYY-MM-DD, inclusive

Examples:
  blunderdb tournament ranking --db base.db --season --from 2026-01-01 --to 2026-12-31
  blunderdb tournament ranking --db base.db --season --rencontre 1 --points 10,6,4 --elo --format json
```

### `blunderdb tournament standings`

```
Usage: blunderdb tournament standings [options]

Print the standings of a direction as CSV.

Options:
  -db string
    	Path to the database file (required)
  -id int
    	Tournament ID (required)

Examples:
  blunderdb tournament standings --db base.db --id 3
  blunderdb tournament standings --db base.db --id 3 > classement.csv
```

### `blunderdb tournament tables`

```
Usage: blunderdb tournament tables [options]

Print the table properties — name, room, reserved, kept for — and the rooms of each event.

Options:
  -db string
    	Path to the database file (required)
  -format string
    	Output format: text or json (default "text")
  -rencontre int
    	Rencontre ID
  -tournament int
    	Tournament ID

Examples:
  blunderdb tournament tables --db base.db --rencontre 1
  blunderdb tournament tables --db base.db --tournament 3 --format json
```

### `blunderdb tournament verify`

```
Usage: blunderdb tournament verify [options]

Replay a direction and report any remaining warning. Exits in error if one remains.

Options:
  -db string
    	Path to the database file (required)
  -format string
    	Output format: text or json (default "text")
  -id int
    	Tournament ID (required)

Examples:
  blunderdb tournament verify --db base.db --id 3
  blunderdb tournament verify --db base.db --id 3 --format json
```

### `blunderdb training missed`

```
Usage: blunderdb training missed --db <file> [options]

The positions answered wrong in the Training journal, each once, the most
recently missed first. A question that ran out of time counts as missed.
Only the Decision exercise records the position of each question, so only
its sessions have missed positions.
--deck and --collection turn them into study material.

Options:
  -collection string
    	Make a collection of these positions, with this name
  -db string
    	Path to the database file (required)
  -deck string
    	Make an Anki deck of these positions, with this name
  -exercise string
    	Only this exercise's sessions (decision: the only exercise that records its positions)
  -format string
    	Output format: text or json (default "text")
  -limit int
    	Number of positions (0 = all)
  -session training sessions
    	Only this session (an id from training sessions)

Examples:
  blunderdb training missed --db database.db
  blunderdb training missed --db database.db --session 12 --deck "Missed on Monday"
  blunderdb training missed --db database.db --collection "My misses" --format json
```

### `blunderdb training sessions`

```
Usage: blunderdb training sessions --db <file> [options]

The sessions of the Training journal, most recent first, with their id
(for `training missed --session`). PR is the Decision exercise's only.

Options:
  -db string
    	Path to the database file (required)
  -exercise string
    	Only this exercise (scores, pips, bearoff, evaluation, decision)
  -format string
    	Output format: text or json (default "text")
  -limit int
    	Number of sessions shown (0 = all) (default 20)

Examples:
  blunderdb training sessions --db database.db
  blunderdb training sessions --db database.db --exercise decision --format json
```

### `blunderdb transcribe`

```
Usage: blunderdb transcribe [options]

Replay a transcription and report what the replay finds.

The source is a .mat file, a match of a library, or a
transcription draft of one — exactly one of the three.
A match is read through its own .mat rendering, so what
is replayed is what an export of it would contain.

--check lists the inconsistencies: an illegal play, two
turns in a row for the same player, an impossible cube
action, an action past the end of the match, a play that
does not use its own roll, a play the record does not
carry (gnubg writes it "???"), a game score the games
before it do not give. Each is named with the
action number and the game it belongs to.

An inconsistency is REPORTED, never held against the
input: nothing is refused for one, and the exit status is
0 whatever the replay finds. A non-zero status means the
file, the database or the output failed.

--render writes the transcription back as a .mat file,
which is how the round trip is checked on real files.

Three options write, and only these: --finish writes a
draft's match and releases the draft, --abandon deletes a
draft without a match, --edit opens a draft on an existing
match so that --finish replaces it in place. Editing an
imported match counts what a .mat cannot carry (analyses,
comments) and refuses without --accept-losses. Abandoning a
draft that never produced a match loses everything typed
in it, and asks for --yes.

Options:
  -abandon
    	Abandon the --draft: delete it without a match; a match it was opened from is left untouched
  -accept-losses
    	With --edit on an imported match: accept that the analyses and comments a .mat cannot carry may be lost
  -check
    	List the inconsistencies the replay finds (the default)
  -db string
    	Database holding the match or the draft to replay
  -draft int
    	Transcription draft id to replay (requires --db)
  -edit
    	Open a draft on the --match (or return the one already open on it), for a correction finished with --finish
  -finish
    	Finish the --draft: write its match (or replace the one it was opened from) and release the draft
  -format string
    	Output format: text or json (default "text")
  -mat string
    	Jellyfish/gnubg .mat file to replay
  -match int
    	Match id to replay (requires --db)
  -materialize string
    	JSON document of Actions (header and actions, dice included) to write as a match in one step, or nothing (requires --db)
  -render string
    	Write the transcription back as a .mat file to this path
  -yes
    	With --abandon on a draft that never produced a match: confirm that everything typed in it is lost

Examples:
  # List what a .mat file's replay finds
  blunderdb transcribe --mat match.mat --check

  # Same, machine-readable
  blunderdb transcribe --mat match.mat --check --format json

  # Round trip: render it back and compare
  blunderdb transcribe --mat match.mat --render out.mat

  # Replay a match of the library, or a draft being typed
  blunderdb transcribe --db database.db --match 5 --check
  blunderdb transcribe --db database.db --draft 3 --check

  # Correct a match: open a draft on it, then finish it
  blunderdb transcribe --db database.db --match 5 --edit
  blunderdb transcribe --db database.db --draft 4 --finish

  # Write a match played elsewhere from its Actions; the first illegal Action refuses all
  blunderdb transcribe --db database.db --materialize match.json

  # Drop a draft
  blunderdb transcribe --db database.db --draft 4 --abandon
```

### `blunderdb trash`

```
Usage: blunderdb trash <subcommand> [options]

What was deleted through the trash, and how to put it back.
A delete is still a delete: a JSON snapshot of what disappears is
written first, and nothing else in the database knows the trash
exists — no search filter, no statistic, no retention rule.

Subcommands:
  list                 What is in the trash, most recently deleted first.
  restore --id N       Put entry N back, and drop it from the trash.
  discard --id N       Drop entry N now, without restoring it.
  empty [--older-than D]  Drop everything, or only what is older than D days.
  delete --kind K --id N  Delete an object THROUGH the trash, so it can be undone.

Options:
  --db string          Path to the database file (required)
  --kind string        position, collection, comment (delete); narrows list, which also takes anki_card
  --limit int          Maximum entries listed (default 50)
  --format string      text (default) or json

`blunderdb delete` still deletes outright: a script that deletes a
position expects it gone. Use `trash delete` to keep the undo.

Examples:
  blunderdb trash delete --db base.db --kind position --id 412
  blunderdb trash list --db base.db
  blunderdb trash restore --db base.db --id 3
  blunderdb trash empty --db base.db --older-than 30
```

### `blunderdb vacuum`

```
Usage: blunderdb vacuum [options]

Compact the database file, reclaiming space left behind by
deletions (matches, tournaments, purges). SQLite needs roughly
twice the current file size in free disk space to rebuild it;
blunderdb refuses with a clear error rather than risk running out
of room partway through. This never runs automatically — it is
the only way it happens.

Options:
  -db string
    	Path to the database file (required)
  -format string
    	Output format: text or json (default "text")

Examples:
  blunderdb vacuum --db database.db
  blunderdb vacuum --db database.db --format json
```

### `blunderdb verify`

```
Usage: blunderdb verify [options]

Verify database integrity and imported data.

Options:
  -db string
    	Path to the database file (required)
  -format string
    	Output format: text or json (default "text")
  -mat string
    	MAT file to compare against (optional)
  -match int
    	Match ID to verify (optional)

Examples:
  # Verify database integrity
  blunderdb verify --db database.db

  # Verify match against MAT file
  blunderdb verify --db database.db --match 1 --mat test.mat

  # Machine-readable output
  blunderdb verify --db database.db --format json
```

<!-- END GENERATED CLI REFERENCE -->

## See Also

- `ARCHITECTURE.md` — the current architecture tour (mode dispatch, the
  `database`/`storage`/backends layering, an import's path through the
  parser/ingest/Zobrist pipeline).
- `CLAUDE.md` — working rules and invariants, and pointers to where each
  subsystem's own documentation lives.
- `doc/archive/MATCH_IMPORT_ARCHITECTURE.md`,
  `doc/archive/POSITION_TRACKING_IMPLEMENTATION.md` — historical design notes
  from when match import and position tracking were first built; useful for
  the reasoning behind a decision, but they predate the `storage`/`database`
  split and do not describe current code.
