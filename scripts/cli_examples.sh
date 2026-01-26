#!/bin/bash
# CLI Examples - Quick Reference for blunderDB CLI commands

BLUNDERDB="../build/bin/blunderdb"
DB="example.db"
XG_FILE="test.xg"
MAT_FILE="test.mat"

echo "=== BlunderDB CLI Examples ==="
echo ""

# Database Creation
echo "1. Create a new database:"
echo "   $BLUNDERDB create --db $DB"
echo ""

# Import
echo "2. Import an XG match file:"
echo "   $BLUNDERDB import --db $DB --type match --file $XG_FILE"
echo ""

# List Matches
echo "3. List all matches:"
echo "   $BLUNDERDB list --db $DB --type matches"
echo ""

# List Positions
echo "4. List first 20 positions:"
echo "   $BLUNDERDB list --db $DB --type positions --limit 20"
echo ""

# Database Statistics
echo "5. Show database statistics:"
echo "   $BLUNDERDB list --db $DB --type stats"
echo ""

# Display Match (JSON)
echo "6. Display match positions as JSON:"
echo "   $BLUNDERDB match --db $DB --id 1 --format json"
echo ""

# Display Match (Text)
echo "7. Display match positions as text:"
echo "   $BLUNDERDB match --db $DB --id 1 --format text"
echo ""

# Display Match Summary
echo "8. Display match summary:"
echo "   $BLUNDERDB match --db $DB --id 1 --format summary"
echo ""

# Export to File
echo "9. Export match to JSON file:"
echo "   $BLUNDERDB match --db $DB --id 1 --format json --output match.json"
echo ""

# Verify Database
echo "10. Verify database integrity:"
echo "    $BLUNDERDB verify --db $DB"
echo ""

# Verify Match with MAT
echo "11. Verify match against MAT file:"
echo "    $BLUNDERDB verify --db $DB --match 1 --mat $MAT_FILE"
echo ""

# Delete Match
echo "12. Delete a match (with confirmation):"
echo "    $BLUNDERDB delete --db $DB --type match --id 1 --confirm"
echo ""

# Help
echo "13. Show help:"
echo "    $BLUNDERDB help"
echo ""

# Version
echo "14. Show version:"
echo "    $BLUNDERDB version"
echo ""

# Advanced Examples
echo "=== Advanced Examples ==="
echo ""

echo "Create database and import in one go:"
echo "  $BLUNDERDB create --db mydb.db && \\"
echo "  $BLUNDERDB import --db mydb.db --type match --file match.xg"
echo ""

echo "Export all matches to separate JSON files:"
echo "  for id in \$(sqlite3 $DB 'SELECT id FROM match'); do"
echo "    $BLUNDERDB match --db $DB --id \$id --format json --output match_\$id.json"
echo "  done"
echo ""

echo "Query database directly with sqlite3:"
echo "  sqlite3 $DB 'SELECT move_type, COUNT(*) FROM move GROUP BY move_type'"
echo ""

echo "Check match positions count:"
echo "  sqlite3 $DB 'SELECT COUNT(*) FROM move WHERE game_id IN (SELECT id FROM game WHERE match_id = 1)'"
echo ""

# Database Queries
echo "=== Useful Database Queries ==="
echo ""

echo "Get all matches:"
echo "  sqlite3 $DB 'SELECT id, player1_name, player2_name, match_length FROM match'"
echo ""

echo "Get games in a match:"
echo "  sqlite3 $DB 'SELECT game_number, initial_score_1, initial_score_2, winner FROM game WHERE match_id = 1'"
echo ""

echo "Count moves by type:"
echo "  sqlite3 $DB 'SELECT move_type, COUNT(*) FROM move GROUP BY move_type'"
echo ""

echo "Get positions with analysis:"
echo "  sqlite3 $DB 'SELECT p.id, COUNT(a.id) as analysis_count FROM position p LEFT JOIN analysis a ON p.id = a.position_id GROUP BY p.id'"
echo ""

echo "=== Complete Workflow Example ==="
echo ""
echo "# Complete workflow: create, import, verify, export"
echo "$BLUNDERDB create --db test.db"
echo "$BLUNDERDB import --db test.db --type match --file test.xg"
echo "$BLUNDERDB verify --db test.db --match 1 --mat test.mat"
echo "$BLUNDERDB match --db test.db --id 1 --format summary"
echo "$BLUNDERDB match --db test.db --id 1 --format json --output test_match.json"
echo ""
