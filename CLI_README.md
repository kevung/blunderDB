# blunderDB Unified Binary

blunderDB now supports both GUI and CLI modes in a **single executable**.

## Quick Start

### GUI Mode (default)
```bash
./blunderdb
```

### CLI Mode
```bash
# Create a new database
./blunderdb create --db matches.db --user "John"

# Import an XG match or XGP position
./blunderdb import --db matches.db --type match --file game.xg

# Batch import a directory
./blunderdb import --db matches.db --type batch --dir ./matches/

# List statistics  
./blunderdb list --db matches.db --type stats

# Search positions
./blunderdb search --db matches.db --decision cube --error-min 0.1

# Display match data
./blunderdb match --db matches.db --id 1 --format summary

# View database info
./blunderdb info --db matches.db

# Edit database metadata
./blunderdb edit --db matches.db --user "Jane"

# Verify database integrity
./blunderdb verify --db matches.db

# Show help
./blunderdb help
```

## How It Works

The binary automatically detects the mode:
- If the first argument is a CLI command (`create`, `import`, `export`, `list`, `search`, `match`, `info`, `edit`, `verify`, `delete`, `help`, `version`), it runs in **headless CLI mode**
- Otherwise, it launches the **GUI application**

## Building

```bash
wails build
```

Binary location: `build/bin/blunderdb`

## Benefits

✅ Single binary to distribute  
✅ No separate CLI tool needed  
✅ Shared codebase for both modes  
✅ Consistent database format  
✅ Easy automation with CLI  
✅ Full-featured GUI when needed  

## Full Documentation

See [CLI_USAGE.md](CLI_USAGE.md) for complete CLI documentation with all commands, options, and examples.
