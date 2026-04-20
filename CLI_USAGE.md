# blunderDB CLI Documentation

The blunderDB application supports both GUI and command-line interface (CLI) modes in a **single binary**. The CLI provides powerful tools for batch operations, automation, and scripting.

## Building

Build the blunderDB binary using Wails:

```bash
wails build
```

The binary will be located at `build/bin/blunderdb`.

## Usage

The same binary works for both GUI and CLI modes:
- **GUI Mode**: Run without arguments: `./blunderdb` 
- **CLI Mode**: Provide CLI commands as arguments: `./blunderdb import --db database.db ...`

When you provide a CLI command as the first argument, it automatically runs in headless CLI mode without displaying the frontend.

### Basic Syntax

```bash
./blunderdb <command> [options]
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

Use `blunderdb <command> --help` for more information about a command.

## Create Command

Create a new blunderDB database file with optional metadata.

```bash
./blunderdb create --db <path> [options]
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
./blunderdb create --db mymatches.db

# Create with metadata
./blunderdb create --db mymatches.db --user "John" --description "2025 tournament matches"

# Overwrite existing database
./blunderdb create --db mymatches.db --force
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
./blunderdb import --db database.db --type match --file match.xg
```

### Import XGP Position

```bash
./blunderdb import --db database.db --type match --file position.xgp
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
./blunderdb import --db mymatches.db --type match --file test.xg

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
./blunderdb import --db database.db --type position --file positions.txt
```

**Position file format:**
Each line should be a JSON-serialized Position object.

### Batch Import

Import all match files from a directory at once:

```bash
./blunderdb import --db database.db --type batch --dir ./matches/
```

**Options:**
- `--dir` - Path to the directory to scan (required for batch)
- `--recursive` - Recursively scan subdirectories (default: true)

Supported file types: `.xg`, `.xgp`, `.sgf`, `.mat`, `.txt`, `.bgf`.

**Examples:**
```bash
# Batch import all files recursively
./blunderdb import --db database.db --type batch --dir ./matches/

# Batch import (non-recursive)
./blunderdb import --db database.db --type batch --dir ./matches/ --recursive=false
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
./blunderdb export --db database.db --type database --file export.db
```

This creates a complete copy of the database including all positions, analyses, matches, and metadata.

### Export Positions

Export all positions to a JSON text file:

```bash
./blunderdb export --db database.db --type positions --file positions.txt
```

Each position is exported as a JSON object on a separate line.

**Options:**
- `--db` - Path to the source database file (required)
- `--type` - Export type: `database`, `positions`, or `matches` (required)
- `--file` - Path to the output file (required)
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
./blunderdb export --db database.db --type database --file export.db --matches=false
```

This creates a copy of the database with positions, analyses, and comments, but without match data.

### Export Matches Only

Export only match data (with linked positions) to a new database:

```bash
./blunderdb export --db database.db --type matches --file matches.db
```

This creates a new database containing only the match structure and linked positions.

## Search Command

Search for positions in the database using filters.

```bash
./blunderdb search --db database.db [options]
```

**Options:**
- `--db` - Path to the database file (required)
- `--export` - Export results to a new database file
- `--limit` - Maximum number of results (0 = no limit)
- `--format` - Output format: `table`, `json`, `xgid` (default: table)
- `--decision` - Filter by decision type: `checker`, `cube`
- `--pip-min` / `--pip-max` - Pip count difference range
- `--winrate-min` / `--winrate-max` - Win rate range (%)
- `--cube` - Filter by cube value
- `--score1` / `--score2` - Filter by player scores
- `--match-length` - Filter by match length
- `--error-min` - Minimum equity error
- `--move-error-min` / `--move-error-max` - Played move error range (millipoints)
- `--has-analysis` - Only positions with analysis
- `--off1-min` / `--off2-min` - Minimum checkers off for player 1/2
- `--match-ids` - Filter by match IDs (comma-separated, e.g. `1,3,5` or range `2,7`)
- `--tournament-ids` - Filter by tournament IDs (comma-separated, e.g. `1,3` or range `1,5`)

### Examples

```bash
# Search cube decisions
./blunderdb search --db database.db --decision cube

# Search positions with errors >= 0.1
./blunderdb search --db database.db --error-min 0.1

# Search in specific matches
./blunderdb search --db database.db --match-ids 2,5

# Search in a tournament
./blunderdb search --db database.db --tournament-ids 1

# Search and export to new database
./blunderdb search --db database.db --decision cube --export cubes.db

# Output as JSON
./blunderdb search --db database.db --format json --limit 10
```

## List Command

Display database contents and statistics.

### List Matches

```bash
./blunderdb list --db database.db --type matches
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

### List Positions

```bash
./blunderdb list --db database.db --type positions --limit 20
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
./blunderdb list --db database.db --type stats
```

Displays comprehensive database statistics:
- Number of positions
- Number of analyses
- Number of matches
- Number of games
- Number of moves

**Example output:**
```
Database Statistics:

  Positions: 1523
  Analyses: 847
  Matches: 12
  Games: 156
  Moves: 3421
```

## Delete Command

Remove data from the database.

### Delete Match

```bash
./blunderdb delete --db database.db --type match --id 1 --confirm
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
./blunderdb delete --db database.db --type match --id 1

# Output:
# Match ID: 1
#   Players: Player1 vs Player2
#   Event: Tournament
#   Games: 15
#
# Are you sure you want to delete this match? (yes/no): yes
# Successfully deleted match ID 1

# Delete without prompt
./blunderdb delete --db database.db --type match --id 1 --confirm
```

## Match Command

Display match positions and analysis data.

```bash
./blunderdb match --db database.db --id <match_id> [options]
```

**Options:**
- `--db` - Path to the database file (required)
- `--id` - Match ID to display (required)
- `--format` - Output format: `json`, `text`, or `summary` (default: json)
- `--output` - Output file path (default: stdout)

**Examples:**
```bash
# Display match as JSON
./blunderdb match --db database.db --id 1

# Display match summary
./blunderdb match --db database.db --id 1 --format summary

# Save match data to a file
./blunderdb match --db database.db --id 1 --format text --output match1.txt
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
./blunderdb info --db database.db [options]
```

**Options:**
- `--db` - Path to the database file (required)
- `--format` - Output format: `text` or `json` (default: text)

**Examples:**
```bash
# Display database info
./blunderdb info --db database.db

# Display as JSON (for scripting)
./blunderdb info --db database.db --format json
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
./blunderdb edit --db database.db [options]
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
./blunderdb edit --db database.db --user "Jane"

# Set description
./blunderdb edit --db database.db --description "Updated match collection"

# Set both
./blunderdb edit --db database.db --user "Jane" --description "My matches"

# Clear user name
./blunderdb edit --db database.db --clear-user

# Clear description
./blunderdb edit --db database.db --clear-description
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
./blunderdb verify --db database.db [options]
```

**Options:**
- `--db` - Path to the database file (required)
- `--match` - Match ID to verify (optional — verifies specific match)
- `--mat` - Path to a MAT file to compare against (optional — used with `--match`)

When run without `--match`, displays database statistics. When a match ID is specified, verifies the match data. When a MAT file is also provided, cross-references the database positions with the source file.

**Examples:**
```bash
# Verify database overview
./blunderdb verify --db database.db

# Verify a specific match
./blunderdb verify --db database.db --match 1

# Compare match against MAT source file
./blunderdb verify --db database.db --match 1 --mat original.mat
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
./blunderdb import --db mymatches.db --type batch --dir ./matches/

# Or import individual files
./blunderdb import --db mymatches.db --type match --file match1.xg
./blunderdb import --db mymatches.db --type match --file match2.xg
```

### Backup Database

```bash
./blunderdb export --db production.db --type database --file backup-$(date +%Y%m%d).db
```

### Check Database Before and After Import

```bash
# Before
./blunderdb list --db database.db --type stats

# Import
./blunderdb import --db database.db --type match --file newmatch.xg

# After
./blunderdb list --db database.db --type stats
```

### Export Positions for Analysis

```bash
./blunderdb export --db database.db --type positions --file positions.txt
# Process positions.txt with external tools
```

## Error Handling

The CLI provides clear error messages:

```bash
# Missing required flag
./blunderdb import --db database.db --type match
# Error: --file flag is required

# File not found
./blunderdb import --db database.db --type match --file nonexistent.xg
# Error: input file does not exist: nonexistent.xg

# Invalid import type
./blunderdb import --db database.db --type invalid --file test.xg
# Error: unknown import type: invalid (must be 'match', 'position', or 'batch')

# Database errors
./blunderdb list --db /invalid/path/database.db --type stats
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
if ./blunderdb import --db database.db --type match --file match.xg; then
    echo "Import successful"
else
    echo "Import failed"
    exit 1
fi
```

## See Also

- Main blunderDB documentation
- `MATCH_IMPORT_ARCHITECTURE.md` - Technical details about match import
- `POSITION_TRACKING_IMPLEMENTATION.md` - Position data structures
