# blunderDB Unified Binary

blunderDB now supports both GUI and CLI modes in a **single executable**.

## Quick Start

### GUI Mode (default)
```bash
./blunderDB
```

### CLI Mode
```bash
# Create a new database
./blunderDB create --db matches.db --user "John"

# Import an XG match or XGP position
./blunderDB import --db matches.db --type match --file game.xg

# Batch import a directory
./blunderDB import --db matches.db --type batch --dir ./matches/

# List statistics  
./blunderDB list --db matches.db --type stats

# Search positions
./blunderDB search --db matches.db --decision cube --error-min 0.1

# Display match data
./blunderDB match --db matches.db --id 1 --format summary

# View database info
./blunderDB info --db matches.db

# Edit database metadata
./blunderDB edit --db matches.db --user "Jane"

# Verify database integrity
./blunderDB verify --db matches.db

# Show help
./blunderDB help
```

## How It Works

The binary automatically detects the mode:
- If the first argument is a CLI command (`create`, `import`, `export`, `list`, `search`, `match`, `info`, `edit`, `verify`, `delete`, `help`, `version`), it runs in **headless CLI mode**
- Otherwise, it launches the **GUI application**

## Building

```bash
wails build
```

Binary location: `build/bin/blunderDB`

## Benefits

✅ Single binary to distribute  
✅ No separate CLI tool needed  
✅ Shared codebase for both modes  
✅ Consistent database format  
✅ Easy automation with CLI  
✅ Full-featured GUI when needed  

## Full Documentation

See [CLI_USAGE.md](CLI_USAGE.md) for complete CLI documentation with all commands, options, and examples.
