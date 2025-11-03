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

When you provide CLI commands (`import`, `export`, `list`, `delete`, `help`, `version`) as the first argument, it automatically runs in headless CLI mode without displaying the frontend.

### Basic Syntax

```bash
./blunderdb <command> [options]
```

### Available Commands

- `import` - Import data into the database
- `export` - Export data from the database  
- `list` - List database contents
- `delete` - Delete data from the database
- `help` - Show help message
- `version` - Show version information

## Import Command

Import XG match files or position files into a database.

### Import XG Match

```bash
./blunderdb import --db database.db --type match --file match.xg
```

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
- `--type` - Export type: `database` or `positions` (required)
- `--file` - Path to the output file (required)

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

## Common Workflows

### Import Multiple Matches

```bash
#!/bin/bash
for xg_file in matches/*.xg; do
    ./blunderdb import --db mymatches.db --type match --file "$xg_file"
done
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
# Error: unknown import type: invalid (must be 'match' or 'position')

# Database errors
./blunderdb list --db /invalid/path/database.db --type stats
# Error: failed to open database: ...
```

## Tips

1. **Database Creation**: If the database file doesn't exist, it will be created automatically during import or export operations.

2. **Match IDs**: After importing a match, note the returned Match ID for future reference (listing, deletion, etc.).

3. **Batch Operations**: Use shell scripts to automate importing multiple files or performing regular backups.

4. **Data Safety**: Always use `--confirm` flag carefully when deleting data. The delete operation is permanent.

5. **Performance**: For large databases, use `--limit` when listing positions to avoid overwhelming output.

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
