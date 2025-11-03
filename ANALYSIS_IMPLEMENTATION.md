# Analysis Import Implementation

## Overview

This document describes the implementation of XG match analysis import into the position's `analysis` table for UI compatibility.

## Problem Statement

When importing XG matches, analysis was being saved to the `move_analysis` table (linked to moves), but the UI reads from the `analysis` table (linked to positions). This caused the analysis panel to appear empty when viewing imported matches.

## Solution

Added two new functions to convert XG analysis format to `PositionAnalysis` format and save to the position's `analysis` table:

### 1. `saveCheckerAnalysisToPositionInTx`

Converts XG checker move analysis to `PositionAnalysis` format.

**Input**: `[]xgparser.CheckerAnalysis`  
**Output**: Saves to `analysis` table linked to position

**Key Conversions**:
- Converts XG move format `[8]int8` to readable move notation using `convertXGMoveToString`
- Converts `float32` values to `float64` for consistency
- Calculates equity errors relative to best move
- Converts win rates from decimals to percentages (multiply by 100)
- Preserves all analysis alternatives (not just the best move)

**Example**:
```go
// XG Analysis:
{
    Move: [8]int8{24, 23, 13, 11, 0, 0, 0, 0},
    Equity: 0.452,
    Player1WinRate: 0.623,
    Player1GammonRate: 0.145,
    ...
}

// Converted to PositionAnalysis:
{
    AnalysisType: "CheckerMove",
    CheckerAnalysis: {
        Moves: [{
            Move: "24/23 13/11",
            Equity: 0.452,
            PlayerWinChance: 62.3,      // Converted to percentage
            PlayerGammonChance: 14.5,   // Converted to percentage
            ...
        }]
    }
}
```

### 2. `saveCubeAnalysisToPositionInTx`

Converts XG cube decision analysis to `PositionAnalysis` format.

**Input**: `*xgparser.CubeAnalysis`  
**Output**: Saves to `analysis` table linked to position

**Key Conversions**:
- Converts all equity values from `float32` to `float64`
- Converts win/gammon/backgammon rates to percentages
- Calculates equity errors for each decision relative to best action
- Determines best cube action (No Double, Double/Take, or Double/Pass)
- Maps XG's `WrongPassTakePercent` to `WrongPassPercentage`

**Example**:
```go
// XG Analysis:
{
    Player1WinRate: 0.714,
    CubefulNoDouble: 0.623,
    CubefulDoubleTake: 0.845,
    CubefulDoublePass: 1.000,
    ...
}

// Converted to PositionAnalysis:
{
    AnalysisType: "DoublingCube",
    DoublingCubeAnalysis: {
        BestCubeAction: "Double, Pass",
        PlayerWinChances: 71.4,              // Converted to percentage
        CubefulNoDoubleEquity: 0.623,
        CubefulNoDoubleError: 0.377,         // 1.000 - 0.623
        CubefulDoubleTakeEquity: 0.845,
        CubefulDoubleTakeError: 0.155,       // 1.000 - 0.845
        CubefulDoublePassEquity: 1.000,
        CubefulDoublePassError: 0.000,       // Best action
        ...
    }
}
```

### 3. `determineBestCubeAction`

Helper function to determine the best cube action based on equity values.

**Logic**:
```
if CubefulDoublePass has highest equity:
    return "Double, Pass"
else if CubefulDoubleTake has highest equity:
    return "Double, Take"
else:
    return "No Double"
```

### 4. `saveAnalysisInTx`

Saves or updates `PositionAnalysis` in the `analysis` table within a transaction.

**Features**:
- Checks for existing analysis for the position
- Preserves creation date when updating
- Stores analysis as JSON in the `data` column
- Uses transactions for ACID compliance

## Integration Points

The analysis saving is integrated into `importMoveWithCache`:

```go
// For checker moves:
if len(move.CheckerMove.Analysis) > 0 {
    // Save to move_analysis table (existing)
    for _, analysis := range move.CheckerMove.Analysis {
        err = d.saveMoveAnalysisInTx(tx, moveID, &analysis)
        ...
    }
    
    // NEW: Also save to position analysis table (for UI compatibility)
    err = d.saveCheckerAnalysisToPositionInTx(tx, positionID, move.CheckerMove.Analysis)
    ...
}

// For cube decisions:
if move.CubeMove.Analysis != nil {
    // Save to move_analysis table (existing)
    err = d.saveCubeAnalysisInTx(tx, moveID, move.CubeMove.Analysis)
    ...
    
    // NEW: Also save to position analysis table (for UI compatibility)
    err = d.saveCubeAnalysisToPositionInTx(tx, positionID, move.CubeMove.Analysis)
    ...
}
```

## Data Flow

```
XG Match File
     ↓
ImportXGMatch()
     ↓
importMoveWithCache() - for each move
     ├─→ Save position (with deduplication)
     ├─→ Save move
     ├─→ Save to move_analysis table
     └─→ NEW: Save to position analysis table
             ├─→ saveCheckerAnalysisToPositionInTx() OR
             └─→ saveCubeAnalysisToPositionInTx()
                     ↓
                 saveAnalysisInTx()
                     ↓
                 analysis table (position_id, data)
```

## Database Schema

The `analysis` table structure:
```sql
CREATE TABLE analysis (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    position_id INTEGER,
    data TEXT,  -- JSON-serialized PositionAnalysis
    FOREIGN KEY (position_id) REFERENCES position(id)
)
```

## Testing

To verify the implementation:

1. **Import a match**:
   ```bash
   # Run the application
   ./build/bin/blunderdb
   
   # Import an XG match file with analysis
   # (use the UI's import functionality)
   ```

2. **Check the database**:
   ```sql
   -- Count analyses
   SELECT COUNT(*) FROM analysis;
   
   -- View sample analysis
   SELECT a.id, a.position_id, p.decision_type, 
          json_extract(a.data, '$.analysis_type') as type,
          json_extract(a.data, '$.analysis_engine_version') as engine
   FROM analysis a
   JOIN position p ON a.position_id = p.id
   LIMIT 10;
   
   -- View checker analysis details
   SELECT a.id,
          json_extract(a.data, '$.checker_analysis.moves[0].move') as best_move,
          json_extract(a.data, '$.checker_analysis.moves[0].equity') as equity,
          json_extract(a.data, '$.checker_analysis.moves[0].player_win_chance') as win_pct
   FROM analysis a
   WHERE json_extract(a.data, '$.analysis_type') = 'CheckerMove'
   LIMIT 5;
   
   -- View cube analysis details
   SELECT a.id,
          json_extract(a.data, '$.doubling_cube_analysis.best_cube_action') as best_action,
          json_extract(a.data, '$.doubling_cube_analysis.cubeful_no_double_equity') as no_double,
          json_extract(a.data, '$.doubling_cube_analysis.cubeful_double_take_equity') as double_take
   FROM analysis a
   WHERE json_extract(a.data, '$.analysis_type') = 'DoublingCube'
   LIMIT 5;
   ```

3. **Verify in UI**:
   - Open the imported match
   - Navigate to positions
   - Open the analysis panel
   - Verify that checker move analysis or doubling cube analysis appears

## Type Conversions Reference

| XG Type | Our Type | Conversion |
|---------|----------|------------|
| `float32` | `float64` | `float64(value)` |
| `[8]int8` | `string` | `convertXGMoveToString()` |
| `int16` | `string` | `fmt.Sprintf("%d", value)` |
| `int32` | `string` | `fmt.Sprintf("%d", value)` |
| Win rates (0-1) | Percentages | `value * 100.0` |

## Performance Considerations

- Analysis is saved in the same transaction as the move, ensuring atomicity
- Position deduplication prevents duplicate analysis for the same position
- JSON serialization provides flexibility for future schema changes
- No additional database queries (uses existing transaction)

## Future Enhancements

Possible improvements:
1. Add support for more analysis engines (GNU Backgammon, eXtreme Gammon 3)
2. Implement analysis comparison between different engines
3. Add analysis quality metrics
4. Support partial analysis updates
5. Add analysis caching for frequently accessed positions

## Related Files

- `db.go`: Main implementation
- `model.go`: Data structures (`PositionAnalysis`, `CheckerAnalysis`, `DoublingCubeAnalysis`)
- `POSITION_TRACKING_IMPLEMENTATION.md`: Position deduplication documentation
