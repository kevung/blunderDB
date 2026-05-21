package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

// runMatch handles the match command
func (cli *CLI) runMatch(args []string) error {
	matchCmd := flag.NewFlagSet("match", flag.ExitOnError)

	// Define flags
	dbPath := matchCmd.String("db", "", "Path to the database file (required)")
	matchID := matchCmd.Int64("id", 0, "Match ID (required)")
	format := matchCmd.String("format", "json", "Output format: json, text, summary")
	output := matchCmd.String("output", "", "Output file (default: stdout)")

	matchCmd.Usage = func() {
		fmt.Println("Usage: blunderdb match [options]")
		fmt.Println()
		fmt.Println("Display match positions and analysis.")
		fmt.Println()
		fmt.Println("Options:")
		matchCmd.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  # Display match positions in JSON format")
		fmt.Println("  blunderdb match --db database.db --id 1 --format json")
		fmt.Println()
		fmt.Println("  # Display match summary")
		fmt.Println("  blunderdb match --db database.db --id 1 --format summary")
		fmt.Println()
		fmt.Println("  # Save match positions to file")
		fmt.Println("  blunderdb match --db database.db --id 1 --output match.json")
	}

	if err := matchCmd.Parse(args); err != nil {
		return err
	}

	// Validate required flags
	if *dbPath == "" {
		matchCmd.Usage()
		return fmt.Errorf("missing required flag: --db")
	}

	if *matchID == 0 {
		matchCmd.Usage()
		return fmt.Errorf("missing required flag: --id")
	}

	// Initialize database
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}

	// Get match info
	match, err := cli.db.GetMatchByID(*matchID)
	if err != nil {
		return fmt.Errorf("failed to get match: %v", err)
	}

	// Get match positions
	positions, err := cli.db.GetMatchMovePositions(*matchID)
	if err != nil {
		return fmt.Errorf("failed to get match positions: %v", err)
	}

	// Format output based on requested format
	var outputData string
	switch strings.ToLower(*format) {
	case "json":
		outputData, err = cli.formatMatchJSON(match, positions)
	case "text":
		outputData, err = cli.formatMatchText(match, positions)
	case "summary":
		outputData, err = cli.formatMatchSummary(match, positions)
	default:
		return fmt.Errorf("unknown format: %s (must be 'json', 'text', or 'summary')", *format)
	}

	if err != nil {
		return fmt.Errorf("failed to format output: %v", err)
	}

	// Output results
	if *output != "" {
		err := os.WriteFile(*output, []byte(outputData), 0644)
		if err != nil {
			return fmt.Errorf("failed to write output file: %v", err)
		}
		fmt.Printf("Match data written to: %s\n", *output)
	} else {
		fmt.Println(outputData)
	}

	return nil
}

// formatMatchJSON formats match data as JSON
func (cli *CLI) formatMatchJSON(match *Match, positions []MatchMovePosition) (string, error) {
	output := map[string]interface{}{
		"match":          match,
		"positions":      positions,
		"position_count": len(positions),
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}

// formatMatchText formats match data as text
func (cli *CLI) formatMatchText(match *Match, positions []MatchMovePosition) (string, error) {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Match ID: %d\n", match.ID))
	sb.WriteString(fmt.Sprintf("Players: %s vs %s\n", match.Player1Name, match.Player2Name))
	if match.Event != "" {
		sb.WriteString(fmt.Sprintf("Event: %s\n", match.Event))
	}
	if match.Location != "" {
		sb.WriteString(fmt.Sprintf("Location: %s\n", match.Location))
	}
	sb.WriteString(fmt.Sprintf("Match Length: %d\n", match.MatchLength))
	sb.WriteString(fmt.Sprintf("Total Positions: %d\n\n", len(positions)))

	for i, movePos := range positions {
		sb.WriteString(fmt.Sprintf("Position %d:\n", i+1))
		sb.WriteString(fmt.Sprintf("  Game: %d, Move: %d\n", movePos.GameNumber, movePos.MoveNumber))

		// Handle XG player encoding: -1 = Player1 (X), 1 = Player2 (O)
		var playerName string
		if movePos.PlayerOnRoll == -1 {
			playerName = match.Player1Name
		} else if movePos.PlayerOnRoll == 1 {
			playerName = match.Player2Name
		} else {
			playerName = "Unknown"
		}
		sb.WriteString(fmt.Sprintf("  Player on roll: %d (%s)\n", movePos.PlayerOnRoll, playerName))
		sb.WriteString(fmt.Sprintf("  Score: %d-%d\n", movePos.Position.Score[0], movePos.Position.Score[1]))
		sb.WriteString(fmt.Sprintf("  Cube: %d (owner: %d)\n", movePos.Position.Cube.Value, movePos.Position.Cube.Owner))
		if movePos.Position.Dice[0] != 0 {
			sb.WriteString(fmt.Sprintf("  Dice: %d-%d\n", movePos.Position.Dice[0], movePos.Position.Dice[1]))
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// formatMatchSummary formats match data as a summary
func (cli *CLI) formatMatchSummary(match *Match, positions []MatchMovePosition) (string, error) {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Match: %s vs %s\n", match.Player1Name, match.Player2Name))
	if match.Event != "" {
		sb.WriteString(fmt.Sprintf("Event: %s\n", match.Event))
	}
	sb.WriteString(fmt.Sprintf("Match Length: %d points\n", match.MatchLength))
	sb.WriteString(fmt.Sprintf("Games: %d\n", match.GameCount))
	sb.WriteString(fmt.Sprintf("Total Positions: %d\n\n", len(positions)))

	// Count positions by game
	gamePositions := make(map[int32]int)
	for _, pos := range positions {
		gamePositions[pos.GameNumber]++
	}

	sb.WriteString("Positions per game:\n")
	for gameNum := int32(1); gameNum <= int32(match.GameCount); gameNum++ {
		count := gamePositions[gameNum]
		sb.WriteString(fmt.Sprintf("  Game %d: %d positions\n", gameNum, count))
	}

	return sb.String(), nil
}
