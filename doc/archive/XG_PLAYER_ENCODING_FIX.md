# XG Player Encoding Fix

## Issues Found

### 1. Player On Roll Always Showing Player 2
**Problem:** Dice were always displayed on Player 2's side (top), regardless of who was actually on roll.

**Root Cause:** XG format uses player encoding of -1 and 1 (not 0 and 1):
- XG: -1 = Player 1 (bottom, black)
- XG: 1 = Player 2 (top, white)

But blunderDB uses:
- blunderDB: 0 = Player 1 (bottom, black)
- blunderDB: 1 = Player 2 (top, white)

The import code was directly casting `move.CheckerMove.ActivePlayer` to int without converting the encoding, resulting in `player_on_roll` being set to -1 instead of 0 for Player 1.

### 2. Checker Color Display
The colors were actually being assigned correctly during import. Any visual issue with "all checkers appearing black" was likely a rendering or perception issue, not a data problem.

## Fixes Applied

### 1. Added Conversion Function
**File:** `db.go`

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

### 2. Updated Import Functions
Fixed `PlayerOnRoll` assignment in two functions:

#### `importMoveWithCache()`
```go
// Before:
pos.PlayerOnRoll = int(move.CheckerMove.ActivePlayer)  // Wrong: -1 or 1
pos.PlayerOnRoll = int(move.CubeMove.ActivePlayer)     // Wrong: -1 or 1

// After:
pos.PlayerOnRoll = convertXGPlayerToBlunderDB(move.CheckerMove.ActivePlayer)  // Correct: 0 or 1
pos.PlayerOnRoll = convertXGPlayerToBlunderDB(move.CubeMove.ActivePlayer)     // Correct: 0 or 1
```

#### `importMoveWithCacheAndRawCube()`
Same fix applied to this function as well.

## Testing

### Before Fix
```sql
SELECT move_number, player, player_on_roll FROM...
0|1|1     ← Player 2 on roll (correct)
2|-1|-1   ← Player 1 on roll (WRONG! Should be 0)
4|1|1     ← Player 2 on roll (correct)
6|-1|-1   ← Player 1 on roll (WRONG! Should be 0)
```

### After Fix
```sql
SELECT move_number, player, player_on_roll FROM...
0|1|1     ← Player 2 on roll ✓
2|-1|0    ← Player 1 on roll ✓
4|1|1     ← Player 2 on roll ✓
6|-1|0    ← Player 1 on roll ✓
```

## Impact

- **Dice Display:** Dice now correctly alternate between Player 1 (bottom) and Player 2 (top) based on who is on roll
- **Match Navigation:** The board correctly shows which player is making the decision at each move
- **Analysis Display:** Analysis is now shown for the correct player
- **All Existing Databases:** Need to be re-imported to get correct player_on_roll values

## Files Modified

- `db.go`: Added `convertXGPlayerToBlunderDB()` function and updated all `PlayerOnRoll` assignments

## How to Fix Existing Databases

1. Re-import all XG match files:
   ```bash
   ./blunderdb import --db your_database.db --file match.xg --type match
   ```

2. Or, if you want to fix an existing database, you would need to run an UPDATE query to convert player_on_roll values from -1 to 0, but this requires extracting and updating JSON which is complex in SQLite.

## Related Documentation

- XG Player Encoding: -1 = Player 1, 1 = Player 2
- See `tests/debug_xg_test.go` for XG encoding examples
- See `cli.go` line 962 for original XG player encoding comment
