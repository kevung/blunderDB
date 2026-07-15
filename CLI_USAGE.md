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
- `search` - Search positions with filters
- `list` - List database contents
- `match` - Display match positions and analysis
- `info` - Display database metadata
- `edit` - Edit database metadata
- `verify` - Verify database integrity
- `delete` - Delete data from the database
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
- `--match-ids` - Filter by match IDs: comma-separated list e.g. `1,3,5`, OR a two-value range e.g. `2,7` (2 through 7), OR a semicolon list e.g. `2;7`
- `--tournament-ids` - Filter by tournament IDs: comma-separated list e.g. `1,3,5`, OR a two-value range e.g. `2,7` (2 through 7), OR a semicolon list e.g. `2;7`

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
