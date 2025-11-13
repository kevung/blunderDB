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

In match mode, positions are **transformed for display** only:
- If Player 2 (player_on_roll = 1) is on roll, the position is mirrored
- This ensures Player 1 is always displayed at the bottom
- The actual stored position remains unchanged

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

Added display transformation logic:

```javascript
// Helper function to get the position to display
function getDisplayPosition() {
    const position = get(positionStore);
    const matchCtx = get(matchContextStore);
    
    // In match mode, if Player 2 is on roll, mirror for display
    if (matchCtx && matchCtx.isMatchMode && position.player_on_roll === 1) {
        return mirrorPosition(position);
    }
    
    return position;
}

// Mirror a position (swap players)
function mirrorPosition(pos) {
    const mirrored = JSON.parse(JSON.stringify(pos));
    
    // Mirror the board points
    for (let i = 0; i < 26; i++) {
        mirrored.board.points[25 - i] = {
            color: tempPoints[i].color === -1 ? -1 : 1 - tempPoints[i].color,
            checkers: tempPoints[i].checkers
        };
    }
    
    // Swap bearoff, player on roll, scores, cube owner
    [mirrored.board.bearoff[0], mirrored.board.bearoff[1]] = 
        [mirrored.board.bearoff[1], mirrored.board.bearoff[0]];
    mirrored.player_on_roll = 1 - mirrored.player_on_roll;
    [mirrored.score[0], mirrored.score[1]] = [mirrored.score[1], mirrored.score[0]];
    if (mirrored.cube.owner !== -1) {
        mirrored.cube.owner = 1 - mirrored.cube.owner;
    }
    
    return mirrored;
}
```

Updated `drawBoard()` to use `getDisplayPosition()`:
```javascript
export function drawBoard() {
    // ...
    const position = getDisplayPosition(); // Use transformed position
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
5. Board automatically redraws
6. Applies mirroring if needed
```

## Key Design Decisions

### 1. Storage vs Display Separation

**Storage**: Positions stored from player on roll POV
- Maintains compatibility with parsers
- Preserves original game notation
- No data transformation at import time

**Display**: Transformed only when rendering
- Clean separation of concerns
- Original data remains intact
- Easy to toggle display modes

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
- Player: Kévin Unger (Player 2, at BOTTOM)
- Move: 51: 13/8 24/23

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

- The mirroring is ONLY for display - stored data never changes
- Player numbers: 0 = Player 1, 1 = Player 2
- Point numbering: 1-24 (backgammon standard)
- Bar: point 0 (Player 1) and point 25 (Player 2)
- Bearoff: separate array, not part of points

## Compatibility

This implementation maintains full compatibility with:
- Existing position storage format
- Search and filter functions
- Analysis display
- Comment system
- All other database features

Only the display layer is affected when in match mode.
