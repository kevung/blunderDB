# blunderDB CLI Documentation

The blunderDB application supports both GUI and command-line interface (CLI) modes in a **single binary**. The CLI provides powerful tools for batch operations, automation, and scripting.

## Building

Build the blunderDB binary using Wails (the `webkit2_41` tag matches
webkit2gtk-4.1 on Arch and ubuntu-latest; drop it for webkit2gtk-4.0 on
ubuntu-22.04 — see `CLAUDE.md`):

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

- `create` - Create a new database with optional metadata
- `import` - Import data into the database (match, position, batch)
- `export` - Export data from the database
- `identity` - Show or move your issuer identity
- `open` - Open a password-protected copy into an ordinary database
- `search` - Search positions with filters
- `list` - List database contents
- `match` - Display match positions and analysis
- `collection` - Manage collections (list, show, create, rename, delete, export)
- `anki` - Spaced-repetition decks (decks, stats, forecast, sync)
- `epc` - EPC, win probability and money cube verdict for a bearoff position
- `analyze` - Write a gammonNet analysis for every position missing one
- `info` - Display database metadata
- `edit` - Edit database metadata
- `verify` - Verify database integrity
- `vacuum` - Compact the database file, reclaiming freed space
- `delete` - Delete data from the database
- `healthcheck` - Probe a running `serve` daemon's `/readyz`; exit 0 when it is ready
- `completion` - Print a shell completion script (bash, zsh, fish)
- `help` - Show help message
- `version` - Show version information

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

Import match files (.xg, .sgf, .mat, .txt, .bgf) or XGP position files (.xgp) into a database.

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

Supported file types: `.xg`, `.xgp`, `.sgf`, `.mat`, `.txt`, `.bgf`.

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

**Reading the output:** a `—` (an empty field in CSV) marks a figure that was never measured, which is not the same as zero. Luck in particular is only available for matches imported since database schema 2.15.0, and only from formats that carry it (XG, gnuBG — not BGF or Jellyfish `.mat`); re-import the source files to obtain it. The `luck_rolls` column says how many rolls the average covers.

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

## EPC Command

Compute the Effective Pip Count, the win probability and the money cube
verdict for a bearoff position given as an XGID. Pure computation: no
database file is involved.

```bash
./blunderDB epc [options] '<XGID>'
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

# Re-analyse everything gammonNet wrote at an outdated version or depth,
# at 3-ply
./blunderDB analyze --db database.db --stale --ply 3
```

## Info Command

Display database metadata and statistics.

```bash
./blunderDB info --db database.db [options]
```

**Options:**
- `--db` - Path to the database file (required)
- `--format` - Output format: `text` or `json` (default: text)

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

Edit database metadata (user name and description).

```bash
./blunderDB edit --db database.db [options]
```

**Options:**
- `--db` - Path to the database file (required)
- `--user` - Set the user name
- `--description` - Set the description
- `--clear-user` - Clear the user name
- `--clear-description` - Clear the description
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
the rebuilt file.

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

--stale switches to the other sweep: every position whose stored
analysis is entirely gammonNet's own (never an XG/GNUbg/BGBlitz
one — ADR-0013 protects those unconditionally) but was written at
an older engine version or a different depth than --ply now asks
for. Use it after an engine upgrade, or after raising --ply for a
library already analysed at a shallower depth.

Positions are analysed --jobs at a time, on that many cores: the
positions of a batch are independent, so the result is the same
whatever --jobs says. Use --jobs 1 to leave the machine free.

A position gammonNet declines to evaluate (a match score beyond
its MET, a cube state it refuses) is reported separately at the
end, as "refused": not a failure, and not retried to no effect.

Options:
  -candidates int
    	Candidate moves kept per checker decision (default 10)
  -db string
    	Path to the database file (required)
  -format string
    	Output format: text or json (default "text")
  -jobs int
    	Positions analysed in parallel (one CPU each) (default 16)
  -ply int
    	Search depth (canonical: 2, k=12) (default 2)
  -prune-k int
    	Pruning width (canonical: 12) (default 12)
  -stale
    	Re-analyse positions whose gammonNet analysis is outdated, instead of filling gaps

Examples:
  blunderdb analyze --db database.db
  blunderdb analyze --db database.db --jobs 1
  blunderdb analyze --db database.db --stale --ply 3
  blunderdb analyze --db database.db --format json
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

### `blunderdb edit`

```
Usage: blunderdb edit [options]

Edit database metadata.

Options:
  -clear-description
    	Clear description
  -clear-user
    	Clear user name
  -db string
    	Path to the database file (required)
  -description string
    	Set description
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
```

### `blunderdb epc`

```
Usage: blunderdb epc [options] <XGID>

Compute EPC, win probability and money cube verdict for a position.
Win probability is exact inside the two-sided database domain and
estimated (with its error bound) outside; the cube verdict is only
ever shown when exact.

Options:
  -bearoff-ts string
    	Optional two-sided bearoff database (.bd) widening the embedded TS-06-06
  -format string
    	Output format: text, json (default "text")

Examples:
  # EPC and race analysis of a bearoff position
  blunderdb epc 'XGID=-BBBB----------------bbbb-:0:0:1:00:0:0:0:0:10'

  # With the downloaded/wider database
  blunderdb epc --bearoff-ts ~/.local/share/blunderdb/gnubg_ts6x11.bd '<XGID>'
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
    	Exit non-zero when any item failed to import (position/batch); by default only a total failure (nothing imported) is an error
  -file string
    	Path to the file to import (for match/position)
  -format string
    	Output format: text or json (default "text")
  -recursive
    	Recursively scan subdirectories for batch import (default true)
  -type string
    	Import type: match, position, batch (required)

Import Types:
  match     Import a single match file (.xg, .sgf, .mat, .txt, .bgf) or XGP position (.xgp)
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
```

### `blunderdb info`

```
Usage: blunderdb info [options]

Display database metadata and statistics.

Options:
  -db string
    	Path to the database file (required)
  -format string
    	Output format: text, json (default "text")

Examples:
  # Display database info
  blunderdb info --db database.db

  # Output as JSON
  blunderdb info --db database.db --format json

  # See where a database came from (works on a protected .dbx too)
  blunderdb info --db cours.db
```

### `blunderdb list`

```
Usage: blunderdb list [options]

List database contents.

Options:
  -db string
    	Path to the database file (required)
  -decision-type string
    	Decision type: all, checker, or cube (stats only) (default "all")
  -format string
    	Output format: text, json or csv (stats and players only) (default "text")
  -from string
    	Start date filter YYYY-MM-DD (stats only)
  -limit int
    	Maximum number of items to list (default 10)
  -metric string
    	Metric to display: pr or mwc (stats only) (default "pr")
  -player string
    	Filter by player name (stats only)
  -to string
    	End date filter YYYY-MM-DD (stats only)
  -top-blunders int
    	Number of top blunders to show (stats only) (default 10)
  -tournament string
    	Filter by tournament IDs, comma-separated (stats only)
  -type string
    	List type: matches, tournaments, positions, stats, players (required)

Examples:
  # List all matches
  blunderdb list --db database.db --type matches

  # List all tournaments
  blunderdb list --db database.db --type tournaments

  # List first 20 positions
  blunderdb list --db database.db --type positions --limit 20

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

### `blunderdb repair`

```
Usage: blunderdb repair [options]

Recompute the scalar columns of every analysis from the JSON
they are a projection of. The analyses themselves are left
untouched: this repairs what was derived from them, and is
useful after a fix to how an imported analysis is read.
Nothing runs it automatically.

Options:
  -db string
    	Path to the database file (required)
  -format string
    	Output format: text or json (default "text")

Examples:
  blunderdb repair --db database.db
  blunderdb repair --db database.db --format json
```

### `blunderdb search`

```
Usage: blunderdb search [options]

Search for positions in the database using filters.

Options:
  -cube int
    	Filter by cube value
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
  -has-analysis
    	Only positions with analysis
  -has-comment
    	Only positions carrying a comment (whatever its origin — yours or an imported note)
  -individual
    	Only positions imported on their own, not as part of a match
  -limit int
    	Maximum number of results (0 = no limit)
  -match-ids string
    	Filter by match IDs: comma-separated list e.g. '1,3,5', OR a two-value range e.g. '2,7' (2 through 7), OR a semicolon list e.g. '2;7'
  -match-length int
    	Filter by match length
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
  -pip-max int
    	Maximum pip count difference
  -pip-min int
    	Minimum pip count difference
  -position-ids string
    	Filter by position IDs (range '2,7' or explicit list '5;10;15')
  -query string
    	Search with the interface's own query language, e.g. 's cube p>30 E>0.05' (see --query-help); exclusive with the filter flags
  -query-help
    	List the tokens --query understands, and exit
  -score1 int
    	Filter by player 1 score (default -1)
  -score2 int
    	Filter by player 2 score (default -1)
  -tournament-ids string
    	Filter by tournament IDs: comma-separated list e.g. '1,3,5', OR a two-value range e.g. '2,7' (2 through 7), OR a semicolon list e.g. '2;7'
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

  # Filters no flag exposes: a move pattern, a comment tag, a player, a date
  blunderdb search --db database.db --query 's m"13/11" t"blunder" pl"Alice" T>2026/01/01'
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
