# XG Match Import Architecture

## Overview

This document describes the architecture for importing eXtremeGammon (.xg) match files into blunderDB. The implementation allows users to import complete match data including positions, moves, games, and analysis information.

## Database Schema (Version 1.4.0)

### New Tables

#### `match` Table
Stores match-level metadata.

```sql
CREATE TABLE match (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    player1_name TEXT,
    player2_name TEXT,
    event TEXT,
    location TEXT,
    round TEXT,
    match_length INTEGER,
    match_date DATETIME,
    import_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    file_path TEXT,
    game_count INTEGER DEFAULT 0
);
```

#### `game` Table
Stores individual games within a match.

```sql
CREATE TABLE game (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    match_id INTEGER,
    game_number INTEGER,
    initial_score_1 INTEGER,
    initial_score_2 INTEGER,
    winner INTEGER,
    points_won INTEGER,
    move_count INTEGER DEFAULT 0,
    FOREIGN KEY(match_id) REFERENCES match(id) ON DELETE CASCADE
);
```

#### `move` Table
Stores individual moves within games.

```sql
CREATE TABLE move (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    game_id INTEGER,
    move_number INTEGER,
    move_type TEXT,           -- "checker" or "cube"
    position_id INTEGER,
    player INTEGER,
    dice_1 INTEGER,
    dice_2 INTEGER,
    checker_move TEXT,
    cube_action TEXT,
    FOREIGN KEY(game_id) REFERENCES game(id) ON DELETE CASCADE,
    FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE SET NULL
);
```

#### `move_analysis` Table
Stores analysis data for each move.

```sql
CREATE TABLE move_analysis (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    move_id INTEGER,
    analysis_type TEXT,       -- "checker" or "cube"
    depth TEXT,
    equity REAL,
    equity_error REAL,
    win_rate REAL,
    gammon_rate REAL,
    backgammon_rate REAL,
    opponent_win_rate REAL,
    opponent_gammon_rate REAL,
    opponent_backgammon_rate REAL,
    FOREIGN KEY(move_id) REFERENCES move(id) ON DELETE CASCADE
);
```

## Data Flow

### Import Process

1. **File Selection**: User selects an .xg file through a file dialog
2. **Parsing**: The `xgparser` library parses the file into structured data
3. **Transaction Begin**: A database transaction starts (ACID compliance)
4. **Match Import**:
   - Insert match metadata
   - For each game in the match:
     - Insert game data
     - For each move in the game:
       - Create and save position
       - Insert move data
       - Save analysis data (if available)
5. **Transaction Commit**: All data is committed atomically

### Key Components

#### Backend (Go)

- **`ImportXGMatch(filePath string) (int64, error)`**: Main import function
- **`importGame(tx, matchID, game)`**: Imports a single game
- **`importMove(tx, gameID, moveNumber, move, game)`**: Imports a single move
- **`createPositionFromXG(xgPos, game, moveNum)`**: Converts XG position format to blunderDB format
- **`GetAllMatches()`**: Returns all imported matches
- **`GetGamesByMatch(matchID)`**: Returns games for a match
- **`GetMovesByGame(gameID)`**: Returns moves for a game
- **`DeleteMatch(matchID)`**: Deletes a match and all associated data

#### Frontend (Svelte)

Components to be implemented:
- **MatchPanel.svelte**: Displays list of imported matches
- **MATCH Mode**: Navigation mode for reviewing match positions sequentially

## XG Parser Integration

### Library: github.com/kevung/xgparser

The xgparser library provides lightweight parsing of .xg files.

#### Key Structures

```go
type Match struct {
    Metadata MatchMetadata
    Games    []Game
}

type MatchMetadata struct {
    Player1Name   string
    Player2Name   string
    Event         string
    Location      string
    Round         string
    DateTime      string
    MatchLength   int32
}

type Game struct {
    GameNumber   int32
    InitialScore [2]int32
    Moves        []Move
    Winner       int32
    PointsWon    int32
}

type Move struct {
    MoveType    string  // "checker" or "cube"
    CheckerMove *CheckerMove
    CubeMove    *CubeMove
}
```

### Position Format Conversion

XG uses a different position format than blunderDB:

**XG Format**:
- Index 0-23: Points (0 = player's 1-point)
- Index 24: Player's bar
- Index 25: Opponent's bar
- Positive values: Player's checkers
- Negative values: Opponent's checkers

**blunderDB Format**:
- Index 0-25: Points + bars
- Each point has Color (0/1/-1) and Checkers count
- Bearoff array tracks checkers borne off

## MATCH Mode

### Purpose
A new navigation mode for sequentially reviewing all positions in an imported match.

### Features

1. **Sequential Navigation**: Navigate through match positions in chronological order
2. **Status Bar Info**: Display current game number, move number, score
3. **Analysis Display**: Show move analysis for each position
4. **Game Boundaries**: Visual indication of game transitions

### UI Elements

- **Match Panel**: List of imported matches with metadata
- **Status Bar**: Game/move indicators in MATCH mode
- **Navigation**: j/k or arrow keys to move through positions
- **Match Info**: Player names, event, score

## Implementation Status

### Completed ✅

1. Database schema (v1.4.0) with match, game, move, move_analysis tables
2. Auto-migration from v1.3.0 to v1.4.0
3. Basic match management functions (GetAllMatches, GetMatchByID, etc.)
4. XG parser dependency added to go.mod

### In Progress 🚧

1. Complete ImportXGMatch implementation (field mapping issues to resolve)
2. Frontend file dialog for XG files
3. Match Panel component
4. MATCH mode implementation

### To Do 📋

1. Fix XG position→blunderDB position conversion
2. Test import with test.xg file
3. Implement Match Panel UI
4. Add MATCH mode to status bar
5. Update navigation for sequential match review
6. Add keyboard shortcuts for match operations

## Testing Plan

### Test File: test.xg

Located at `/home/unger/src/blunderDB/test.xg`

#### Test Steps

1. **Import Test**:
   ```
   - Open blunderDB
   - Select File → Import XG Match
   - Choose test.xg
   - Verify import success message
   ```

2. **Data Verification**:
   ```sql
   SELECT * FROM match;
   SELECT * FROM game WHERE match_id = 1;
   SELECT * FROM move WHERE game_id = 1 LIMIT 10;
   SELECT * FROM move_analysis LIMIT 10;
   ```

3. **Navigation Test**:
   ```
   - Open Match Panel
   - Double-click on imported match
   - Verify first position loads
   - Press 'j' to navigate to next position
   - Verify game/move info in status bar
   ```

4. **Analysis Test**:
   ```
   - Navigate to a checker move
   - Open Analysis Panel (Ctrl+L)
   - Verify analysis data displays correctly
   ```

## Known Issues

### Field Name Mismatches

The xgparser library uses different field names than initially assumed:

**Position fields**:
- `Cube` (not `CubeValue`)
- `CubePos` (not `CubeOwner`)
- No `ActivePlayer` field (comes from Move context)

**Analysis fields**:
- `Player1WinRate` (not `WinRate`)
- `Player1GammonRate` (not `GammonRate`)
- `Player2GammonRate` (not `OpponentGammonRate`)

### Resolution Needed

Update `createPositionFromXG` and analysis functions to use correct field names from xgparser library.

## Future Enhancements

1. **Match Statistics**: Aggregate stats across games
2. **Error Rate Analysis**: Calculate and display error rates
3. **Match Comparison**: Compare multiple matches
4. **Export Match**: Export match data back to .xg format
5. **Match Annotations**: Add notes to specific games/moves

## API Reference

### Backend Functions

```go
// Import XG file and return match ID
func (d *Database) ImportXGMatch(filePath string) (int64, error)

// Get all matches
func (d *Database) GetAllMatches() ([]Match, error)

// Get specific match
func (d *Database) GetMatchByID(matchID int64) (*Match, error)

// Get games in match
func (d *Database) GetGamesByMatch(matchID int64) ([]Game, error)

// Get moves in game
func (d *Database) GetMovesByGame(gameID int64) ([]Move, error)

// Delete match and all associated data
func (d *Database) DeleteMatch(matchID int64) error
```

### Frontend Bindings (To Be Implemented)

```javascript
import { ImportXGMatch, GetAllMatches, GetMatchByID } from '../wailsjs/go/main/Database.js';

// Import match
const matchID = await ImportXGMatch("/path/to/match.xg");

// Load matches
const matches = await GetAllMatches();

// Load specific match
const match = await GetMatchByID(matchID);
```

## Architecture Decisions

### Why Separate Tables?

Instead of storing everything in the existing `position` table:

**Advantages**:
- Clear separation of match structure vs isolated positions
- Efficient queries for match-specific data
- Maintains referential integrity with foreign keys
- Allows match-level operations (delete entire match)
- Preserves chronological order of moves

**Trade-offs**:
- Slightly more complex schema
- More tables to maintain
- Positions duplicated if same position appears multiple times

### Why Link Moves to Positions?

Each move creates a position entry:

**Advantages**:
- Positions can be searched/filtered independently
- Existing position analysis features work
- Can compare match positions with database positions
- Maintains compatibility with existing features

**Design**:
- `move.position_id` links to `position.id`
- `ON DELETE SET NULL` prevents cascade deletion
- Positions exist independently of matches

### Transaction Management

All match import operations use database transactions:

**Benefits**:
- ACID compliance (Atomic, Consistent, Isolated, Durable)
- All-or-nothing import (no partial match data)
- Data integrity guaranteed
- Can rollback on error

## Maintenance

### Database Migrations

Automatic migration from v1.3.0 to v1.4.0:
- Checks for missing tables
- Creates new tables if needed
- Updates version metadata
- Non-destructive (existing data preserved)

### Cleanup

```sql
-- Remove orphaned positions (optional)
DELETE FROM position 
WHERE id NOT IN (SELECT position_id FROM move WHERE position_id IS NOT NULL)
AND id NOT IN (/* other position sources */);

-- Verify match integrity
SELECT m.id, m.player1_name, COUNT(g.id) as games
FROM match m
LEFT JOIN game g ON g.match_id = m.id
GROUP BY m.id;
```

## Performance Considerations

1. **Batch Imports**: Use transactions for atomic commits
2. **Indexing**: Consider adding indexes on frequently queried fields
3. **Position Deduplication**: Future optimization to reduce storage
4. **Analysis Caching**: Store computed statistics

## Security

1. **File Validation**: Verify .xg file format before parsing
2. **SQL Injection**: Use parameterized queries (already implemented)
3. **Transaction Isolation**: Prevent concurrent modification conflicts
4. **File Path Security**: Validate file paths before access

## Conclusion

The XG match import feature integrates the xgparser library with blunderDB's existing position management system. The architecture maintains data integrity through foreign key relationships and transactions while providing a clear separation between match-specific data and general position data.

The MATCH mode will provide a streamlined workflow for reviewing match positions sequentially, complementing the existing database search capabilities.
