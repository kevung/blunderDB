# blunderDB Unified Binary

blunderDB now supports both GUI and CLI modes in a **single executable**.

## Quick Start

### GUI Mode (default)
```bash
./blunderdb
```

### CLI Mode
```bash
# Import an XG match
./blunderdb import --db matches.db --type match --file game.xg

# List statistics  
./blunderdb list --db matches.db --type stats

# Show help
./blunderdb help
```

## How It Works

The binary automatically detects the mode:
- If the first argument is a CLI command (`import`, `export`, `list`, `delete`, `help`, `version`), it runs in **headless CLI mode**
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
