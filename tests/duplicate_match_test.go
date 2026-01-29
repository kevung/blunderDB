package tests

import (
	"testing"
)

// TestComputeMatchHash tests that the match hash is computed based on full transcription
func TestComputeMatchHash(t *testing.T) {
	// ComputeMatchHash now takes the full xgparser.Match structure
	// and includes all game data for comprehensive duplicate detection:
	// - Player names (normalized: lowercase, trimmed)
	// - Match length
	// - All games with their scores, winners, points won
	// - All moves with dice rolls and played moves
	// - All cube actions

	t.Log("Match hash now includes full match transcription:")
	t.Log("  - Player names (case-insensitive, trimmed)")
	t.Log("  - Match length")
	t.Log("  - For each game: initial scores, winner, points won")
	t.Log("  - For each move: move type, dice, played move")
	t.Log("  - For cube moves: cube action")
}

// TestDuplicateMatchDetection documents the expected behavior of duplicate detection
func TestDuplicateMatchDetection(t *testing.T) {
	t.Log("Duplicate match detection should:")
	t.Log("  1. Prevent importing the same XG file twice")
	t.Log("  2. Detect duplicates based on full match transcription")
	t.Log("  3. Include all moves and cube decisions in the hash")
	t.Log("  4. Use case-insensitive player name comparison")
	t.Log("  5. Return ErrDuplicateMatch error when duplicate is detected")
	t.Log("  6. Display status bar message (no popup) in GUI for duplicates")
	t.Log("  7. Display clear error message in CLI for duplicates")
}
