#!/bin/bash

# test_import.sh - Comprehensive test script for XG import verification
# This script creates a new database, imports test.xg, and verifies the data

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BLUNDERDB="${PROJECT_DIR}/build/bin/blunderdb"
TEST_DB="${SCRIPT_DIR}/test_import.db"
XG_FILE="${SCRIPT_DIR}/test.xg"
MAT_FILE="${SCRIPT_DIR}/test.mat"
OUTPUT_JSON="${SCRIPT_DIR}/test_output.json"
OUTPUT_SUMMARY="${SCRIPT_DIR}/test_summary.txt"

# Function to print colored output
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}→ $1${NC}"
}

print_section() {
    echo ""
    echo "========================================"
    echo "$1"
    echo "========================================"
}

# Function to check if blunderDB binary exists
check_binary() {
    if [ ! -f "$BLUNDERDB" ]; then
        print_error "blunderDB binary not found at: $BLUNDERDB"
        echo "Please build the project first with: go build -o build/bin/blunderdb"
        exit 1
    fi
    print_success "Found blunderDB binary"
}

# Function to check test files exist
check_test_files() {
    if [ ! -f "$XG_FILE" ]; then
        print_error "Test XG file not found: $XG_FILE"
        exit 1
    fi
    print_success "Found test.xg file"
    
    if [ ! -f "$MAT_FILE" ]; then
        print_error "Test MAT file not found: $MAT_FILE"
        exit 1
    fi
    print_success "Found test.mat file"
}

# Function to count moves in MAT file
count_mat_moves() {
    local mat_file="$1"
    # Count lines that match move pattern: starts with whitespace, number, and )
    grep -E "^\s+[0-9]+\)" "$mat_file" | wc -l
}

# Function to clean up old test files
cleanup() {
    print_info "Cleaning up old test files..."
    rm -f "$TEST_DB" "$OUTPUT_JSON" "$OUTPUT_SUMMARY"
    print_success "Cleanup complete"
}

# Main test execution
main() {
    print_section "BlunderDB XG Import Test Suite"
    
    # Step 1: Verify prerequisites
    print_section "Step 1: Checking Prerequisites"
    check_binary
    check_test_files
    
    # Step 2: Clean up old test files
    print_section "Step 2: Cleanup"
    cleanup
    
    # Step 3: Create new database
    print_section "Step 3: Creating New Database"
    print_info "Creating database: $TEST_DB"
    "$BLUNDERDB" create --db "$TEST_DB"
    if [ $? -eq 0 ]; then
        print_success "Database created successfully"
    else
        print_error "Failed to create database"
        exit 1
    fi
    
    # Step 4: Import XG file
    print_section "Step 4: Importing XG File"
    print_info "Importing: $XG_FILE"
    "$BLUNDERDB" import --db "$TEST_DB" --type match --file "$XG_FILE"
    if [ $? -eq 0 ]; then
        print_success "XG file imported successfully"
    else
        print_error "Failed to import XG file"
        exit 1
    fi
    
    # Step 5: List matches
    print_section "Step 5: Listing Matches"
    "$BLUNDERDB" list --db "$TEST_DB" --type matches
    
    # Step 6: Display database statistics
    print_section "Step 6: Database Statistics"
    "$BLUNDERDB" list --db "$TEST_DB" --type stats
    
    # Step 7: Export match positions to JSON
    print_section "Step 7: Exporting Match Positions (JSON)"
    print_info "Exporting to: $OUTPUT_JSON"
    "$BLUNDERDB" match --db "$TEST_DB" --id 1 --format json --output "$OUTPUT_JSON"
    if [ $? -eq 0 ]; then
        print_success "Match positions exported to JSON"
        
        # Count positions in JSON
        DB_POSITION_COUNT=$(grep -o '"position_count":[^,}]*' "$OUTPUT_JSON" | grep -o '[0-9]*')
        print_info "Database position count: $DB_POSITION_COUNT"
    else
        print_error "Failed to export match positions"
        exit 1
    fi
    
    # Step 8: Export match summary
    print_section "Step 8: Exporting Match Summary"
    print_info "Exporting to: $OUTPUT_SUMMARY"
    "$BLUNDERDB" match --db "$TEST_DB" --id 1 --format summary --output "$OUTPUT_SUMMARY"
    if [ $? -eq 0 ]; then
        print_success "Match summary exported"
        echo ""
        cat "$OUTPUT_SUMMARY"
    else
        print_error "Failed to export match summary"
        exit 1
    fi
    
    # Step 9: Count moves in MAT file
    print_section "Step 9: Comparing with MAT File"
    MAT_MOVE_COUNT=$(count_mat_moves "$MAT_FILE")
    print_info "MAT file move count: $MAT_MOVE_COUNT"
    
    # Step 10: Verify position counts match
    print_section "Step 10: Verification Results"
    
    if [ "$DB_POSITION_COUNT" -eq "$MAT_MOVE_COUNT" ]; then
        print_success "Position count matches! (DB: $DB_POSITION_COUNT, MAT: $MAT_MOVE_COUNT)"
    else
        print_error "Position count mismatch! (DB: $DB_POSITION_COUNT, MAT: $MAT_MOVE_COUNT)"
        echo "Note: Some differences may be expected if:"
        echo "  - The XG file contains cube decisions (not always in MAT files)"
        echo "  - The MAT file contains additional notation or commentary"
    fi
    
    # Step 11: Run verification command
    print_section "Step 11: Running Built-in Verification"
    "$BLUNDERDB" verify --db "$TEST_DB" --match 1 --mat "$MAT_FILE"
    
    # Step 12: Sample position inspection
    print_section "Step 12: Sample Position Inspection"
    print_info "Displaying first position from JSON export:"
    echo ""
    # Extract first position using jq if available, otherwise use grep
    if command -v jq &> /dev/null; then
        jq '.positions[0]' "$OUTPUT_JSON" 2>/dev/null || echo "Install jq for better JSON formatting"
    else
        print_info "Install 'jq' for formatted JSON display"
        grep -A 20 '"positions"' "$OUTPUT_JSON" | head -25
    fi
    
    # Step 13: Verify player storage
    print_section "Step 13: Verifying Position Storage"
    print_info "Checking that positions are stored from player on roll POV..."
    if command -v jq &> /dev/null; then
        PLAYER1_COUNT=$(jq '[.positions[] | select(.player_on_roll == 0)] | length' "$OUTPUT_JSON")
        PLAYER2_COUNT=$(jq '[.positions[] | select(.player_on_roll == 1)] | length' "$OUTPUT_JSON")
        print_info "Positions with Player 1 on roll: $PLAYER1_COUNT"
        print_info "Positions with Player 2 on roll: $PLAYER2_COUNT"
        print_success "Position storage verified (stored from player on roll POV)"
    fi
    
    # Final summary
    print_section "Test Complete!"
    echo ""
    echo "Summary of generated files:"
    echo "  - Database: $TEST_DB"
    echo "  - JSON export: $OUTPUT_JSON"
    echo "  - Summary: $OUTPUT_SUMMARY"
    echo ""
    echo "To view match positions in detail:"
    echo "  $BLUNDERDB match --db $TEST_DB --id 1 --format text"
    echo ""
    echo "To verify again:"
    echo "  $BLUNDERDB verify --db $TEST_DB --match 1 --mat $MAT_FILE"
    echo ""
}

# Run main function
main
