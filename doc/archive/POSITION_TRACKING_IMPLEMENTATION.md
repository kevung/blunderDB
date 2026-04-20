# Position Tracking Implementation for Match Import

## Overview

This document describes the implementation of position tracking for moves when importing XG match files into blunderDB. The changes ensure that:

1. **Each position is stored in the database** and linked to its corresponding move
2. **Positions are deduplicated** - identical positions are not stored multiple times
3. **Browsing positions** - When loading a match, you can browse through all positions played

## Database Schema

The existing database schema (version 1.4.0) already includes the necessary tables:

### `match` table
Stores match-level metadata:
- `id`: Primary key
- `player1_name`, `player2_name`: Player names
- `event`, `location`, `round`: Tournament information
- `match_length`: Match length
- `match_date`: When the match was played
- `import_date`: When imported into database
- `file_path`: Source XG file path
- `game_count`: Number of games in match

### `game` table
Stores individual games within a match:
- `id`: Primary key
- `match_id`: Foreign key to match
- `game_number`: Game number in sequence
- `initial_score_1`, `initial_score_2`: Score at start of game
- `winner`: Which player won
- `points_won`: Points awarded
- `move_count`: Number of moves in game

### `move` table
Stores each move with its position:
- `id`: Primary key
- `game_id`: Foreign key to game
- `move_number`: Move sequence number
- `move_type`: "checker" or "cube"
- **`position_id`**: Foreign key to position table ⭐
- `player`: Which player made the move
- `dice_1`, `dice_2`: Dice rolled (for checker moves)
- `checker_move`: Move notation (e.g., "24/23 13/11")
- `cube_action`: Cube action (for cube moves)

### `position` table
Stores unique board positions:
- `id`: Primary key
- `state`: JSON containing complete position state

The position state JSON includes:
```json
{
  "id": 123,
  "board": {
    "points": [...],  // 26 points (0-25)
    "bearoff": [0, 0]  // Checkers borne off
  },
  "cube": {
    "owner": 0,  // -1=none, 0=player1, 1=player2
    "value": 2   // Current cube value
  },
  "dice": [3, 1],
  "score": [0, 0],
  "player_on_roll": 0,  // 0=player1, 1=player2
  "decision_type": 0,   // 0=checker, 1=cube
  "has_jacoby": 0,
  "has_beaver": 0
}
```

## Implementation Changes

### 1. Position Deduplication

**File: `db.go`**

Created `savePositionInTxWithCache()` function that:
- Uses an in-memory hash map to track positions during import
- Normalizes positions (excludes ID) for comparison
- Returns existing position ID if duplicate found
- Creates new position only if unique
- Updates cache with new positions

```go
func (d *Database) savePositionInTxWithCache(tx *sql.Tx, position *Position, 
    positionCache map[string]int64) (int64, error)
```

**Benefits:**
- **Performance**: O(1) lookup instead of O(n) database scans
- **Space efficiency**: Eliminates duplicate positions
- **Consistency**: Same position always has same ID

### 2. Position Attributes

**File: `db.go` - `importMoveWithCache()`**

Enhanced to properly set position attributes:
- `PlayerOnRoll`: Set from move's active player
- `DecisionType`: CheckerAction (0) or CubeAction (1)
- `Dice`: Actual dice rolled (for checker moves) or [0,0] (for cube decisions)

This ensures positions contain complete state information for browsing.

### 3. Match Import Workflow

**File: `db.go` - `ImportXGMatch()`**

The import process:
1. Parse XG file using xgparser library
2. Begin database transaction
3. **Build position cache** from existing positions
4. Insert match metadata
5. For each game:
   - Insert game record
   - For each move:
     - Create position from XG data
     - Set position attributes (dice, player, decision type)
     - **Check cache** for existing position
     - Save position (reuse if exists, create if new)
     - Link move to position
     - Save move analysis if available
6. Commit transaction

**Key optimization:**
```go
// Build position cache once at start
positionCache := make(map[string]int64)
existingRows, _ := tx.Query(`SELECT id, state FROM position`)
// ... populate cache ...

// Reuse cache for all moves
for each move {
    posID, _ := d.savePositionInTxWithCache(tx, pos, positionCache)
    // Cache prevents duplicate positions
}
```

## API Functions

### Import Functions

```go
// Import a match from XG file
func (d *Database) ImportXGMatch(filePath string) (int64, error)
```

### Query Functions

```go
// Get all matches
func (d *Database) GetAllMatches() ([]Match, error)

// Get specific match by ID
func (d *Database) GetMatchByID(matchID int64) (*Match, error)

// Get all games in a match
func (d *Database) GetGamesByMatch(matchID int64) ([]Game, error)

// Get all moves in a game (with position IDs)
func (d *Database) GetMovesByGame(gameID int64) ([]Move, error)

// Load position by ID (existing function)
func (d *Database) LoadPosition(id int) (*Position, error)
```

### Delete Function

```go
// Delete match and cascade to games/moves/analysis
func (d *Database) DeleteMatch(matchID int64) error
```

## Usage Example

### Import a Match

```javascript
import { ImportXGMatch } from './wailsjs/go/main/Database';

// Import match from file
const matchId = await ImportXGMatch('/path/to/match.xg');
console.log(`Imported match ID: ${matchId}`);
```

### Browse Match Positions

```javascript
import { GetAllMatches, GetGamesByMatch, GetMovesByGame, LoadPosition } 
    from './wailsjs/go/main/Database';

// Get all matches
const matches = await GetAllMatches();

// Select a match
const matchId = matches[0].id;

// Get games in match
const games = await GetGamesByMatch(matchId);

// Get moves in first game
const moves = await GetMovesByGame(games[0].id);

// Browse positions
for (const move of moves) {
    const position = await LoadPosition(move.position_id);
    console.log(`Move ${move.move_number}:`, position);
    // Display position on board
}
```

## Data Flow Diagram

```
XG File Import
     │
     ├──> Parse XG (xgparser)
     │
     ├──> Create Match Record
     │         │
     │         ├──> Create Game Records
     │         │         │
     │         │         └──> For each Move:
     │         │               │
     │         │               ├──> Extract Position from XG
     │         │               │
     │         │               ├──> Set Position Attributes
     │         │               │    (dice, player, type)
     │         │               │
     │         │               ├──> Check Position Cache
     │         │               │    │
     │         │               │    ├──> EXISTS? → Reuse ID
     │         │               │    │
     │         │               │    └──> NEW? → Create & Cache
     │         │               │
     │         │               ├──> Create Move Record
     │         │               │    (links to position_id)
     │         │               │
     │         │               └──> Save Analysis (if any)
     │         │
     │         └──> Commit Transaction
     │
     └──> Return Match ID
```

## Benefits

### 1. Storage Efficiency
- Identical positions stored once
- Reduces database size
- Faster queries

### 2. Position Browsing
- Each move linked to its position
- Navigate match chronologically
- View exact board state at any point

### 3. Analysis Integration
- Positions can have analysis attached
- Move-level analysis stored separately
- Both accessible through position_id

### 4. Performance
- In-memory cache during import
- Single transaction for ACID compliance
- Optimized lookups (O(1) vs O(n))

## Testing Recommendations

1. **Import Test**: Import an XG file and verify:
   - Match, game, and move records created
   - Positions linked correctly
   - No duplicate positions

2. **Browsing Test**: Load a match and:
   - Navigate through moves sequentially
   - Verify positions match expected board states
   - Check dice and player_on_roll are correct

3. **Deduplication Test**: Import match with repeated positions:
   - Verify same position reused
   - Count position table rows vs move count

4. **Analysis Test**: Import analyzed match:
   - Verify move_analysis records created
   - Link to correct move_id
   - Accessible through position

## Future Enhancements

### Possible Improvements:

1. **Position Search by Match**
   ```sql
   SELECT DISTINCT p.* FROM position p
   JOIN move m ON p.id = m.position_id
   JOIN game g ON m.game_id = g.id
   WHERE g.match_id = ?
   ```

2. **Match Statistics**
   - Count unique positions per match
   - Identify most common positions
   - Track blunders/errors per game

3. **Position Annotations**
   - Link comments to positions
   - Share annotations across matches
   - Position-specific notes

4. **Export Functionality**
   - Export match to XG format
   - Preserve positions and analysis
   - Round-trip compatibility

## Conclusion

The position tracking implementation provides a robust foundation for match analysis in blunderDB. Positions are efficiently stored, deduplicated, and linked to moves, enabling comprehensive match browsing and analysis workflows.

All positions from imported matches are now available in the position table and can be browsed, searched, filtered, and analyzed using the existing blunderDB interface.
