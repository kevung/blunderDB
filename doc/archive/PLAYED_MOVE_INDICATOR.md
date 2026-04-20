# Played Move Indicator Feature

## Overview

When reviewing matches that have been imported from XG files, the analysis panel now displays a discrete indicator (★) next to the move that was actually played in the match. This helps users quickly identify which move was chosen during the actual game versus the other analyzed alternatives.

## Feature Details

### Visual Indicators

1. **Checker Moves**: A small gold star (★) appears before the move notation in the Move column
2. **Cube Actions**: A small gold star (★) appears before the decision text (e.g., "No Double", "Double/Take")
3. **Row Highlighting**: The row with the played move has a light yellow background (#fff3cd) to make it easily distinguishable

### When Does It Appear?

The indicator only appears when:
- Viewing a position that is part of an imported match (from an XG file)
- The position has associated analysis data
- The move/cube action was recorded in the match database

For positions entered manually or analyzed outside of match context, no indicator will be shown.

## Database Changes

### Backend (Go)

1. **Model Updates** (`model.go`):
   - Added `PlayedMove` field to `PositionAnalysis` struct
   - Added `PlayedCubeAction` field to `PositionAnalysis` struct

2. **Database Queries** (`db.go`):
   - Modified `LoadAnalysis()` to query the `move` table and retrieve `checker_move` or `cube_action` for the position
   - The played move information is automatically populated when available

### Frontend (Svelte)

1. **Store Updates** (`analysisStore.js`):
   - Added `playedMove` field to store the actually played checker move
   - Added `playedCubeAction` field to store the actually played cube action

2. **UI Updates** (`AnalysisPanel.svelte`):
   - Added `isPlayedMove()` helper function to identify played moves
   - Added `isPlayedCubeAction()` helper function to identify played cube actions
   - Updated move row template to display star indicator
   - Updated cube decision rows to display star indicator
   - Added CSS styling for played move highlighting

3. **App Updates** (`App.svelte`):
   - Updated analysis loading to pass `playedMove` and `playedCubeAction` to the store

## Testing

### Import a Match
```bash
./build/bin/blunderdb import --db test.db --file test.xg --type match
```

### Launch GUI and Navigate
1. Open the database in the GUI
2. Navigate to a position from the imported match
3. Open the analysis panel (if not already open)
4. Look for the ★ indicator next to one of the moves

### Expected Results

- The move that was actually played in the match will have:
  - A small gold star (★) before the move notation
  - A light yellow background row
  - The indicator remains visible even when the row is selected (mixed color)

## Design Considerations

### Discrete but Noticeable

The design balances being discrete while still being easy to spot:

- **Small Star Icon**: Uses a common symbol (★) that's universally recognized
- **Subtle Color**: Gold/yellow theme (#856404 text, #fff3cd background) that doesn't overwhelm
- **Font Size**: Slightly smaller (11px) than regular text to avoid drawing too much attention
- **Color Preservation**: Works well with the existing blue selection highlight

### Performance

- No additional database queries needed per move
- Single query per position to fetch played move
- Efficient string comparison for move matching

## Future Enhancements

Potential improvements for future versions:

1. Allow users to toggle the indicator on/off in preferences
2. Add different indicator styles (>, •, →) as user preference
3. Show additional metadata about the played move (timestamp, player, etc.) on hover
4. Extend to show different indicators for best move vs played move vs blunders
