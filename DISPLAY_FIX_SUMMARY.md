# Match Display Fix Summary

## Issues Fixed

### Issue 1: Player On Roll Encoding
**Problem:** Dice were always displayed on Player 2's side (top) because `player_on_roll` was being set incorrectly.

**Root Cause:** XG format uses player encoding of -1 and 1:
- XG: -1 = Player 1, 1 = Player 2
- blunderDB: 0 = Player 1, 1 = Player 2

The import code was not converting between these encodings, so Player 1's `player_on_roll` was -1 instead of 0.

**Fix:** Added `convertXGPlayerToBlunderDB()` function to properly convert player encoding during import.

### Issue 2: Board Mirroring
**Problem:** After match import, the board was being mirrored when Player 2 was on roll, which was incorrect.

**Requirement:** 
- **Player 1 should always be displayed at the bottom** (black checkers, moving 24→1)
- **Player 2 should always be displayed at the top** (white checkers, moving 1→24)

**Fix:** Removed the mirroring logic in `Board.svelte` since positions are already stored from Player 1's perspective.

## Fixes Applied

### 1. db.go - Added Player Encoding Conversion
**File:** `/home/unger/src/blunderDB/db.go`

Added conversion function:
```go
// convertXGPlayerToBlunderDB converts XG player encoding to blunderDB encoding
// XG: -1 = Player 1, 1 = Player 2
// blunderDB: 0 = Player 1, 1 = Player 2
func convertXGPlayerToBlunderDB(xgPlayer int32) int {
	if xgPlayer == -1 {
		return 0 // Player 1
	}
	return 1 // Player 2
}
```

Updated all PlayerOnRoll assignments in:
- `importMoveWithCache()` - for checker and cube moves
- `importMoveWithCacheAndRawCube()` - for checker and cube moves

**Before:**
```go
pos.PlayerOnRoll = int(move.CheckerMove.ActivePlayer)  // Wrong: -1 or 1
```

**After:**
```go
pos.3layerOnRoll = convertXGPlayerToBlunderDB(move.CheckerMove.ActivePlayer)  // Correct: 0 or 1
```

### 2. Board.svelte - Removed Mirroring Logic
**File:** `/home/unger/src/blunderDB/frontend/src/components/Board.svelte`

Changed `getDisplayPosition()` from:
```javascript
function getDisplayPosition() {
    const position = get(positionStore);
    const matchCtx = get(matchContextStore);
    
    if (matchCtx && matchCtx.isMatchMode && position.player_on_roll === 1) {
        return mirrorPosition(position);
    }
    
    return position;
}
```

To:
```javascript
function getDisplayPosition() {
    const position = get(positionStore);
    return position;
}
```

**Reason:** Positions are already stored from Player 1's perspective in the database. No transformation is needed for display.

### 2. db.go - Fixed Bar Index Mapping
**File:** `/home/unger/src/blunderDB/db.go`

Fixed the XG to blunderDB bar index conversion in `createPositionFromXG()`:

**For activePlayer == 0 (Player 1):**
- XG index 24 (active player's bar) → blunderDB index 25 (Player 1's bar)
- XG index 25 (opponent's bar) → blunderDB index 0 (Player 2's bar)

**For activePlayer == 1 (Player 2):**
- XG index 24 (active player's bar) → blunderDB index 0 (Player 2's bar)
- XG index 25 (opponent's bar) → blunderDB index 25 (Player 1's bar)

### 4. Updated Documentation
**File:** `/home/unger/src/blunderDB/MATCH_MODE_DISPLAY_IMPLEMENTATION.md`

Corrected the documentation to reflect:
- No mirroring is applied in match mode
- Player 1 is always at the bottom
- Correct bar index mapping (index 0 = Player 2's bar, index 25 = Player 1's bar)

## Data Model Reference

### Player Color Mapping
- **Color 0 = Player 1** (black checkers, bottom, moves 24→1)
- **Color 1 = Player 2** (white checkers, top, moves 1→24)

### Point Indices
- **Indices 1-24:** Regular board points
- **Index 0:** Player 2's bar (white)
- **Index 25:** Player 1's bar (black)

### Bearoff
- `bearoff[0]`: Player 1's checkers borne off
- `bearoff[1]`: Player 2's checkers borne off

## Expected Behavior Example

## Expected Behavior After Fixes

### Test Match Example
```
Game 1
Kévin Unger : 0                       Maxence Job : 0
 1) 51: 24/23 13/8                    61: 13/7 8/7
**File:** `testdata/test.txt`
```
Game 1
Kévin Unger : 0                       Maxence Job : 0
 1) 51: 24/23 13/8                    61: 13/7 8/7
```

Where:
- Player 1 = Maxence Job (always at bottom)
- Player 2 = Kévin Unger (always at top, plays first in this game)

**Display after import:**
1. **Move 1 - Player 2 (Kévin Unger) on roll:**
   - Board shows Player 1 (Maxence Job) at bottom with black checkers
   - Board shows Player 2 (Kévin Unger) at top with white checkers  
   - **Dice displayed at TOP** (Player 2's side) showing 5-1
   - Player 2's move (51: 24/23 13/8) moves white checkers from Player 2's perspective

2. **Move 1 - Player 1 (Maxence Job) on roll:**
   - Board shows Player 1 (Maxence Job) at bottom with black checkers
   - Board shows Player 2 (Kévin Unger) at top with white checkers
   - **Dice displayed at BOTTOM** (Player 1's side) showing 6-1  
   - Player 1's move (61: 13/7 8/7) moves black checkers from Player 1's perspective

### Testing Checklist
1. Import a match file: `./build/bin/blunderdb import --db test.db --file testdata/test.xg --type match`
2. Open the GUI and load the match
3. Verify Player 1 is always displayed at the bottom
4. Verify dice alternate between top and bottom based on who is on roll
5. Navigate through moves with j/k keys
6. Confirm the board perspective never changes
7. Verify checker colors are correct (Player 1=black, Player 2=white)bottom
4. Navigate through moves with j/k keys
5. Confirm the board perspective never changes

## Related Files Modified
- `/home/unger/src/blunderDB/frontend/src/components/Board.svelte`
- `/home/unger/src/blunderDB/db.go`
- `/home/unger/src/blunderDB/MATCH_MODE_DISPLAY_IMPLEMENTATION.md`
