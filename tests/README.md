# BlunderDB Test Suite

This directory contains test files and scripts for verifying the XG import functionality of blunderDB.

## Test Files

- **test.xg**: XG match file containing match positions and analysis data
- **test.mat**: MAT file (text transcription) of the same match showing moves played
- **test_import.sh**: Comprehensive test script for verifying XG import

## Running Tests

### Prerequisites

1. Build blunderDB:
   ```bash
   cd /home/unger/src/blunderDB
   go build -o build/bin/blunderdb
   ```

2. Ensure test files are present:
   - `test.xg` - Binary XG match file
   - `test.mat` - Text MAT file for comparison

### Running the Full Test Suite

```bash
cd tests
./test_import.sh
```

This script will:
1. Check prerequisites (binary and test files)
2. Clean up old test files
3. Create a new database from scratch
4. Import the test.xg file
5. List imported matches
6. Display database statistics
7. Export match positions to JSON
8. Export match summary
9. Compare position count with test.mat
10. Run built-in verification
11. Display sample position data
12. Verify position storage format

### Manual Testing

You can also test individual CLI commands:

#### Create a Database
```bash
./build/bin/blunderdb create --db tests/mytest.db
```

#### Import a Match
```bash
./build/bin/blunderdb import --db tests/mytest.db --type match --file tests/test.xg
```

#### List Matches
```bash
./build/bin/blunderdb list --db tests/mytest.db --type matches
```

#### Display Match Positions (JSON)
```bash
./build/bin/blunderdb match --db tests/mytest.db --id 1 --format json
```

#### Display Match Positions (Text)
```bash
./build/bin/blunderdb match --db tests/mytest.db --id 1 --format text
```

#### Display Match Summary
```bash
./build/bin/blunderdb match --db tests/mytest.db --id 1 --format summary
```

#### Verify Database
```bash
./build/bin/blunderdb verify --db tests/mytest.db
```

#### Verify Match Against MAT File
```bash
./build/bin/blunderdb verify --db tests/mytest.db --match 1 --mat tests/test.mat
```

#### Export Match to File
```bash
./build/bin/blunderdb match --db tests/mytest.db --id 1 --format json --output match.json
```

#### Database Statistics
```bash
./build/bin/blunderdb list --db tests/mytest.db --type stats
```

## Expected Results

### Position Count
The test.mat file contains **178 moves**. The imported database should have the same number of positions (or close to it, depending on whether cube decisions are included).

### Position Storage
Positions are stored in the database from the **player on roll point of view**:
- When Player 1 is on roll: positive checker counts for Player 1
- When Player 2 is on roll: positive checker counts for Player 2

The frontend is responsible for mirroring the display so that:
- Player 1 is always shown on the bottom
- Player 2 is always shown on the top

### Score Verification
Each position includes:
- Match score (e.g., 3-4 in a 7-point match)
- Cube value and owner
- Dice rolled (for checker decisions)
- Player on roll

## Verification Points

The test script verifies:

1. **Position Count**: Database positions should match MAT file moves
2. **Data Integrity**: All positions have valid data (scores, cube, dice)
3. **Player Storage**: Positions stored from player on roll POV
4. **Score Consistency**: Scores progress correctly through the match
5. **Database Schema**: Correct tables and relationships

## Test Output Files

After running the test script, the following files are generated:

- `test_import.db`: SQLite database with imported match
- `test_output.json`: JSON export of all match positions
- `test_summary.txt`: Text summary of the match

## Troubleshooting

### Position Count Mismatch

If the position count doesn't exactly match:
- Check if the XG file includes cube decisions not in the MAT file
- Verify that both files represent the same match
- Some MAT files may have formatting differences

### Build Errors

If the binary doesn't exist:
```bash
cd /home/unger/src/blunderDB
go mod tidy
go build -o build/bin/blunderdb
```

### Test File Issues

Ensure test files are in the correct location:
```bash
ls -la tests/test.xg tests/test.mat
```

## CLI Command Reference

### Create
Create a new database:
```bash
blunderdb create --db <path> [--force]
```

### Import
Import match data:
```bash
blunderdb import --db <path> --type match --file <xg-file>
```

### List
List database contents:
```bash
blunderdb list --db <path> --type <matches|positions|stats> [--limit N]
```

### Match
Display match positions:
```bash
blunderdb match --db <path> --id <match-id> [--format json|text|summary] [--output file]
```

### Verify
Verify database integrity:
```bash
blunderdb verify --db <path> [--match <id>] [--mat <mat-file>]
```

### Delete
Delete a match:
```bash
blunderdb delete --db <path> --type match --id <match-id> [--confirm]
```

## Match Information

The test match (test.xg/test.mat) is:
- **Players**: Kévin Unger vs Maxence Job
- **Event**: HSBT Paris 2023
- **Match Length**: 7 points
- **Games**: 7 games
- **Location**: Paris, Fédération Française de Bridge

This provides a real-world test case with:
- Multiple games
- Cube actions
- Various game situations
- Complete analysis data
