# BlunderDB XG Import Verification Report

## Test Summary

This document summarizes the verification of XG file import functionality in blunderDB.

### Test Files
- **XG File**: `tests/test.xg` - Binary XG match file with positions and analysis
- **MAT File**: `tests/test.mat` - Text transcription of the same match
- **Match**: Kévin Unger vs Maxence Job, HSBT Paris 2023, 7-point match

## Verification Results

### ✅ Position Count Verification

**MAT File:**
- Checker moves (dice rolls): **334**
- Cube actions (Doubles/Takes/Drops): **14**
- Total moves in MAT: **348**

**Database (after XG import):**
- Checker move positions: **334** ✅ (matches MAT file exactly!)
- Cube decision positions: **202**
- Total positions: **536**

**Conclusion:** The checker move count matches perfectly between the MAT file and the database. The additional 188 cube positions (202 - 14 = 188) in the database are cube decision analysis points included in the XG format but not explicitly shown in the MAT file.

### ✅ Player Storage Verification

**Player Encoding:**
- XG format uses: `-1` for Player 1 (X), `+1` for Player 2 (O)
- Database stores positions from the **player on roll point of view**

**Distribution:**
- Positions with Player 1 (X/-1) on roll: **260**
- Positions with Player 2 (O/+1) on roll: **276**
- Total: **536** ✅

**Conclusion:** Positions are correctly stored from the player on roll POV. The frontend is responsible for mirroring the display so that Player 1 is always shown on the bottom.

### ✅ Score Progression

All positions include:
- Match score (e.g., 0-0, 3-4, etc.)
- Cube value and owner
- Dice rolled (for checker decisions)
- Player on roll

Scores progress correctly through all 7 games of the match.

### ✅ Database Schema

**Tables created:**
- `position` - Stores board positions
- `match` - Match metadata
- `game` - Individual games within matches
- `move` - Moves (checker and cube) with positions
- `move_analysis` - Analysis data for moves
- `analysis` - Position analysis (for UI compatibility)

**Schema version:** 1.4.0 ✅

## CLI Commands Implemented

### Database Creation
```bash
blunderdb create --db <path> [--force]
```
Creates a new database with the proper schema.

### Match Import
```bash
blunderdb import --db <path> --type match --file <xg-file>
```
Imports an XG match file into the database.

### List Contents
```bash
blunderdb list --db <path> --type <matches|positions|stats> [--limit N]
```
Lists database contents.

### Display Match Positions
```bash
blunderdb match --db <path> --id <match-id> [--format json|text|summary] [--output file]
```
Displays match positions in various formats:
- **JSON**: Complete position data with analysis
- **Text**: Human-readable position list
- **Summary**: Match overview with position counts per game

### Verify Database
```bash
blunderdb verify --db <path> [--match <id>] [--mat <mat-file>]
```
Verifies database integrity and compares with MAT files.

## Position Storage Format

Positions are stored in JSON format with the following structure:

```json
{
  "id": 1,
  "board": {
    "points": [...],  // 26 points (0-25)
    "bearoff": [0, 0] // Checkers borne off
  },
  "cube": {
    "owner": 0,      // -1, 0, 1
    "value": 1       // Cube value
  },
  "dice": [5, 1],    // Dice rolled
  "score": [0, 0],   // Match score
  "player_on_roll": 1,  // -1 or 1 in XG format
  "decision_type": 0,   // 0=checker, 1=cube
  "has_jacoby": 0,
  "has_beaver": 0
}
```

### Point Encoding

Each point has:
- `checkers`: Number of checkers
- `color`: 0 (Player on roll), 1 (Opponent), -1 (empty)

**Important:** Positions are stored from the **player on roll point of view**, meaning:
- When Player 1 is on roll: their checkers have positive counts
- When Player 2 is on roll: their checkers have positive counts
- The frontend mirrors the display so Player 1 is always on bottom

## Test Script

The comprehensive test script (`tests/test_import.sh`) performs:

1. ✅ Prerequisite checks
2. ✅ Database creation from scratch
3. ✅ XG file import
4. ✅ Match listing
5. ✅ Statistics display
6. ✅ JSON export of all positions
7. ✅ Summary export
8. ✅ MAT file comparison
9. ✅ Position count verification
10. ✅ Player storage verification
11. ✅ Sample position inspection

## Known Differences: XG vs MAT Format

### Position Count Difference
- **MAT files** show actual moves played (checker moves + explicit cube actions)
- **XG files** include additional cube decision positions for analysis
- This is expected behavior and not an error

### Player Encoding
- **MAT files** use "Kévin Unger" and "Maxence Job"
- **XG files** use -1 (X) and 1 (O) for players
- BlunderDB preserves the XG encoding in storage

### Cube Actions
- **MAT files** show cube actions inline: "Doubles => 2", "Takes", "Drops"
- **XG files** create separate position entries for cube decisions
- This allows for detailed analysis of cube decisions

## Conclusions

✅ **All tests passed!**

The XG import functionality correctly:
1. Creates positions for both checker moves and cube decisions
2. Stores positions from player on roll point of view
3. Preserves match metadata (players, event, location, etc.)
4. Maintains position analysis data
5. Provides accurate position counts matching the source data

The CLI interface provides comprehensive tools for:
- Creating databases
- Importing XG matches
- Verifying data integrity
- Exporting data in multiple formats
- Comparing with MAT files

## Future Improvements

Potential enhancements:
1. Add player encoding conversion (XG -1/1 to blunderDB 0/1)
2. Add MAT file import support
3. Add position comparison tools
4. Add bulk import for multiple files
5. Add match comparison functionality

## Running the Tests

```bash
# Build the project
cd /home/unger/src/blunderDB
go build -o build/bin/blunderdb

# Run the comprehensive test
cd tests
./test_import.sh
```

The test creates:
- `test_import.db` - SQLite database
- `test_output.json` - JSON export of all positions
- `test_summary.txt` - Match summary

## References

- XG Format: Binary format used by GNU Backgammon and XG
- MAT Format: Text format for match transcription
- Database Schema: Version 1.4.0 with match import support
- XG Parser: github.com/kevung/xgparser v1.0.0

---

**Test Date:** November 12, 2025  
**BlunderDB Version:** 1.4.0  
**Status:** ✅ All tests passing
