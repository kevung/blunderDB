# Match Mode Display Implementation

## Overview

This document describes the implementation for displaying match positions in blunderDB, ensuring that Player 1 is always displayed at the bottom of the board in match mode, regardless of who is on roll.

## Problem Statement

When importing match files (.xg, .sgf, .bgf), all positions need to be:
1. **Stored in the database** from the player on roll's point of view (as provided by parsers)
2. **Displayed in match mode** with Player 1 always at the bottom

The test files (test.xg, test.mat, test.txt) should show the same first positions in match mode as described in the MAT/TXT format.

## Architecture

### Storage Layer (Database)

Positions are stored **as-is** from the parser's perspective:
- Each position is stored from the player on roll's point of view
- This is the native format from xgparser, gnubgparser, and bgfparser
- No transformation at storage time

### Display Layer (Frontend)

In match mode, positions are displayed **exactly as stored**:
- Player 1 is always displayed at the bottom (black checkers, moves 24→1)
- Player 2 is always displayed at the top (white checkers, moves 1→24)
- No transformation is applied - the stored position IS the display position
- This provides a consistent perspective throughout the match

## Implementation Details

### 1. Backend Changes (`db.go`)

#### New Structure: `MatchMovePosition`
```go
type MatchMovePosition struct {
    Position     Position // The position (stored from player on roll POV)
    MoveID       int64    
    GameID       int64    
    GameNumber   int32    
    MoveNumber   int32    
    PlayerOnRoll int32    // Player who rolled (0=Player1, 1=Player2)
    Player1Name  string   
    Player2Name  string   
}
```

#### New Function: `GetMatchMovePositions`
```go
func (d *Database) GetMatchMovePositions(matchID int64) ([]MatchMovePosition, error)
```

This function:
- Returns all move positions for a match in chronological order (by game number, then move number)
- Includes metadata about which player is on roll
- Does NOT transform the position (stored as-is)

### 2. Frontend Changes

#### A. Position Store (`positionStore.js`)

Added `matchContextStore`:
```javascript
export const matchContextStore = writable({
    isMatchMode: false,      // Whether we're in match mode
    matchID: null,           // Current match ID
    movePositions: [],       // Array of MatchMovePosition objects
    currentIndex: 0,         // Current position index
    player1Name: '',         // Player 1 name
    player2Name: '',         // Player 2 name
});
```

#### B. Match Panel (`MatchPanel.svelte`)

Updated `loadMatchPositions()` to:
1. Call `GetMatchMovePositions()` to load all positions in order
2. Set the match context store with loaded data
3. Load the first position into positionStore
4. Switch to MATCH mode

```javascript
async function loadMatchPositions(match) {
    const movePositions = await GetMatchMovePositions(match.id);
    
    matchContextStore.set({
        isMatchMode: true,
        matchID: match.id,
        movePositions: movePositions,
        currentIndex: 0,
        player1Name: match.player1_name,
        player2Name: match.player2_name,
    });

    const firstMovePos = movePositions[0];
    positionStore.set(firstMovePos.position);
    statusBarModeStore.set('MATCH');
}
```

#### C. Board Component (`Board.svelte`)

Simplified display logic - no transformation needed:

```javascript
// Helper function to get the position to display
// In match mode, positions are already stored from Player 1's perspective
// (Player 1 at bottom with black checkers, Player 2 at top with white checkers)
// No mirroring needed - just return the position as-is
function getDisplayPosition() {
    const position = get(positionStore);
    return position;
}
```

The `mirrorPosition()` function is kept for potential future use but is not currently used in match mode.

Updated `drawBoard()` to use `getDisplayPosition()`:
```javascript
export function drawBoard() {
    // ...
    const position = getDisplayPosition(); // Returns position as-is
    // ... rest of drawing code uses this position
}
```

#### D. Navigation (`App.svelte`)

Updated `previousPosition()` and `nextPosition()` to handle match mode:

```javascript
function previousPosition() {
    // ... mode checks ...
    
    if ($statusBarModeStore === 'MATCH' && $matchContextStore.isMatchMode) {
        const matchCtx = $matchContextStore;
        if (matchCtx.currentIndex > 0) {
            const newIndex = matchCtx.currentIndex - 1;
            matchContextStore.update(ctx => ({ ...ctx, currentIndex: newIndex }));
            const movePos = matchCtx.movePositions[newIndex];
            positionStore.set(movePos.position);
            statusBarTextStore.set(`Match: ${matchCtx.player1Name} vs ${matchCtx.player2Name} | Game ${movePos.game_number}, Move ${movePos.move_number + 1}`);
        }
    } else {
        // Normal mode navigation
    }
}
```

## Data Flow

### Match Import
```
1. User imports .xg file
2. xgparser extracts positions (player on roll POV)
3. Positions stored in database AS-IS
4. Move table links to positions with player info
```

### Match Display
```
1. User double-clicks match in Match Panel
2. GetMatchMovePositions() returns chronological list
3. First position loaded into positionStore
4. Board.svelte checks match mode
5. If Player 2 on roll → mirror for display
6. Board drawn with Player 1 at bottom
```

### Navigation
```
1. User presses j/k or arrows
2. App.svelte checks if in MATCH mode
3. Updates matchContextStore.currentIndex
4. Loads new position into positionStore
5. Board displays position as-is
6. Player 1 always at bottom
```

## Key Design Decisions

### 1. Storage vs Display Separation

**Storage**: Positions stored from player on roll POV
- Maintains compatibility with parsers
- Preserves original game notation
- No data transformation at import time

**Display**: Transformed only when rendering
- Clean separation of concerns
- Original data displays position as-is
5. Player 1 always at bottom, Player 2 always at top
6. No transformation applied
### 2. Mirror Function Location

Placed in `Board.svelte` rather than utility:
- Only used for display rendering
- Has access to match context
- Keeps transformation logic with rendering

### 3. Match Context Store

Separate from positionStore:
- Maintains match-specific state
- Doesn't pollute position data
- Easy to exit match mode

## Testing

Test with the provided files:
1. Import `test.xg`
2. Open Match Panel (Ctrl+M)
3. Double-click the match
4. Verify first position matches `test.mat` or `test.txt`
5. Navigate with j/k keys
6. Verify Player 1 always at bottom

Expected first position (Game 1, Move 1):
- Player on roll: Kévin Unger (Player 2)
- Player 2 displayed at TOP (white checkers, moves 1→24)
- Player 1 displayed at BOTTOM (black checkers, moves 24→1)
- Move: 51: 24/23 13/8 (from Player 2's perspective, moving from their back points)

## Future Enhancements

1. **Analysis Display**: Show move analysis in match mode
2. **Game Boundaries**: Visual indicators between games
3. **Match Statistics**: Aggregate stats across games
4. **Position Comparison**: Compare with database positions
5. **Export Match**: Export back to match file format

## Related Files

- `model.go`: MatchMovePosition structure
- `db.go`: GetMatchMovePositions function
- `positionStore.js`: matchContextStore
- `MatchPanel.svelte`: Match loading
- `Board.svelte`: Display transformation
- `App.svelte`: Match navigation

## Notes

- No transformation is applied in match mode - positions are displayed exactly as stored
- Player numbers: 0 = Player 1 (bottom, black), 1 = Player 2 (top, white)
- Point numbering: 1-24 (backgammon standard)
- Player 1's checkers move from point 24 to point 1 (bearing off)
- Player 2's checkers move from point 1 to point 24 (bearing off)
- Bar: index 0 = Player 2's bar (white), index 25 = Player 1's bar (black)
- Bearoff: separate array [Player 1's bearoff, Player 2's bearoff]

## Compatibility

This implementation maintains full compatibility with:
- Existing position storage format
- Search and filter functions
- Analysis display
- Comment system
- All other database features

Only the display layer is affected when in match mode.
