# blunderDB CLI Documentation

The blunderDB application supports both GUI and command-line interface (CLI) modes in a **single binary**. The CLI provides powerful tools for batch operations, automation, and scripting.

## Building

Build the blunderDB binary using Wails:

```bash
wails build
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

### Batch Import

Import all match files from a directory at once:

```bash
./blunderDB import --db database.db --type batch --dir ./matches/
```

**Options:**
- `--dir` - Path to the directory to scan (required for batch)
- `--recursive` - Recursively scan subdirectories (default: true)

Supported file types: `.xg`, `.xgp`, `.sgf`, `.mat`, `.txt`, `.bgf`.

**Examples:**
```bash
# Batch import all files recursively
./blunderDB import --db database.db --type batch --dir ./matches/

# Batch import (non-recursive)
./blunderDB import --db database.db --type batch --dir ./matches/ --recursive=false
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

```bash
./blunderDB analyze --db database.db [options]
```

**Options:**
- `--ply` - Search depth (default: 2, the canonical parameter)
- `--prune-k` - Pruning width (default: 12, the canonical parameter)
- `--candidates` - Candidate moves kept per checker decision (default: 10)
- `--jobs` - Positions analysed in parallel (default: the number of CPUs)

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

**Example:**
```bash
./blunderDB analyze --db database.db
# Analyzing 1204 position(s) with gammonNet (2-ply, k=12, 16 job(s))...
#   1/1204 (0%)
#   61/1204 (5%)
#   ...
#   1204/1204 (100%)
# Done.

# One core only, on a machine that has other work to do
./blunderDB analyze --db database.db --jobs 1
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

When run without `--match`, displays database statistics. When a match ID is specified, verifies the match data. When a MAT file is also provided, cross-references the database positions with the source file.

Every run also checks referential integrity: it counts orphaned rows — games without a match, moves without a game, move analyses without a move, analyses without a position — and prints a `WARNING` line with the total when any exist. A healthy database reports `Orphaned rows: none`. Orphans can be left behind in databases written by versions that did not enforce foreign keys on every connection (issue #157); they are unreachable from any match and only take up space. The command still exits 0 when it finds some.

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
| `--scope <string>` | `default` | tenant scope (sent as `X-Tenant-ID`; SQLite ignores it for most families) |
| `--json <string>` | `{}` | request body as JSON |
| `--json-file <path>` | – | read the request body from a file |
| `--list` | – | print every `<family>.<method>` and exit |

The JSON response (or NDJSON stream for `*.list` endpoints) is written to
stdout. On an error the process exits non-zero and the `{"error":{…}}` envelope
is printed to stdout so it stays parseable (e.g. with `jq`).

## Migrating a SQLite database into PostgreSQL

`blunderDB migrate` copies a single-user SQLite database into a PostgreSQL
backend under a chosen tenant scope — the path for a desktop user to "upload"
their library into a server deployment.

```bash
blunderDB migrate \
    --from sqlite:///path/to/user.db \
    --to   "postgres://user:pass@host:5432/db?sslmode=disable" \
    --tenant-id my-tenant

# Preview without writing
blunderDB migrate --from sqlite:///path/to/user.db --tenant-id my-tenant --dry-run
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
| `--tenant-id <scope>` | – | destination tenant scope (required unless `--dry-run`) |
| `--dry-run` | – | count what would be copied without writing |
| `--on-conflict <policy>` | `""` | `""` aborts if the tenant already has data; `skip` merges (positions dedup by Zobrist) |

Not migrated (yet): app-state families — anki decks/cards, the filter library,
search/command history, and session metadata. Their per-tenant scoping is
formalised by the session-scope phase; data migration of the core position
library and match history is the priority.

## See Also

- Main blunderDB documentation
- `doc/archive/MATCH_IMPORT_ARCHITECTURE.md` - Technical details about match import
- `doc/archive/POSITION_TRACKING_IMPLEMENTATION.md` - Position data structures
