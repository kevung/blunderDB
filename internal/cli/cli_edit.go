package cli

import (
	"flag"
	"fmt"
	"strings"
)

// runEdit handles the edit command
func (cli *CLI) runEdit(args []string) error {
	editCmd := flag.NewFlagSet("edit", flag.ContinueOnError)

	// Define flags
	dbPath := editCmd.String("db", "", "Path to the database file (required)")
	user := editCmd.String("user", "", "Set user name")
	description := editCmd.String("description", "", "Set description")
	clearUser := editCmd.Bool("clear-user", false, "Clear user name")
	clearDescription := editCmd.Bool("clear-description", false, "Clear description")
	// The two thresholds are taken in millipoints, the unit `E>x` and
	// --move-error-min already speak; the GUI is where they are entered in
	// equity. -1 means "leave it alone" — 0 is a value the pair refuses, not
	// an absent flag.
	errorThreshold := editCmd.Int("error-threshold", -1,
		"Set the error threshold, in millipoints (a decision costing this much or more is an error)")
	blunderThreshold := editCmd.Int("blunder-threshold", -1,
		"Set the blunder threshold, in millipoints (an error costing this much or more is a blunder)")
	format := editCmd.String("format", "text", "Output format: text or json")

	editCmd.Usage = func() {
		fmt.Println("Usage: blunderdb edit [options]")
		fmt.Println()
		fmt.Println("Edit database metadata.")
		fmt.Println()
		fmt.Println("Options:")
		editCmd.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  # Set user name")
		fmt.Println("  blunderdb edit --db database.db --user \"John Doe\"")
		fmt.Println()
		fmt.Println("  # Set description")
		fmt.Println("  blunderdb edit --db database.db --description \"My positions collection\"")
		fmt.Println()
		fmt.Println("  # Clear user name")
		fmt.Println("  blunderdb edit --db database.db --clear-user")
		fmt.Println()
		fmt.Println("  # Set multiple values")
		fmt.Println("  blunderdb edit --db database.db --user \"John\" --description \"Tournament positions\"")
		fmt.Println()
		fmt.Println("  # Draw the library's own lines: XG's thresholds")
		fmt.Println("  blunderdb edit --db database.db --error-threshold 20 --blunder-threshold 80")
	}

	if err := editCmd.Parse(args); err != nil {
		return err
	}

	// Validate required flags
	if *dbPath == "" {
		editCmd.Usage()
		return fmt.Errorf("missing required flag: --db")
	}

	// Check that at least one edit option is provided
	if *user == "" && *description == "" && !*clearUser && !*clearDescription &&
		*errorThreshold < 0 && *blunderThreshold < 0 {
		editCmd.Usage()
		return fmt.Errorf("no edit options provided")
	}

	formatLower := strings.ToLower(*format)
	if formatLower != "text" && formatLower != "json" {
		return fmt.Errorf("unknown format: %s (must be 'text' or 'json')", *format)
	}
	text := formatLower != "json"

	// Initialize database
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}

	// Build metadata updates
	metadata := make(map[string]string)
	changes := []string{}

	if *clearUser {
		metadata["user"] = ""
		changes = append(changes, "Cleared user")
	} else if *user != "" {
		metadata["user"] = *user
		changes = append(changes, fmt.Sprintf("Set user to: %s", *user))
	}

	if *clearDescription {
		metadata["description"] = ""
		changes = append(changes, "Cleared description")
	} else if *description != "" {
		metadata["description"] = *description
		changes = append(changes, fmt.Sprintf("Set description to: %s", *description))
	}

	// Save metadata
	err := cli.db.SaveMetadata(metadata)
	if err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	// The library's own settings (ADR-0046) are not metadata rows on every
	// backend, so they go through their own accessor. Either threshold can be
	// set alone: the other keeps the value the library already had, and the
	// pair is validated as a pair, so raising one past the other is refused
	// here rather than stored.
	if *errorThreshold >= 0 || *blunderThreshold >= 0 {
		settings, err := cli.db.GetLibrarySettings()
		if err != nil {
			return fmt.Errorf("failed to read library settings: %w", err)
		}
		if *errorThreshold >= 0 {
			settings.ErrorThresholdMP = *errorThreshold
		}
		if *blunderThreshold >= 0 {
			settings.BlunderThresholdMP = *blunderThreshold
		}
		if err := cli.db.SaveLibrarySettings(settings); err != nil {
			return fmt.Errorf("failed to save library settings: %w", err)
		}
		changes = append(changes,
			fmt.Sprintf("Set error threshold to: %d millipoints", settings.ErrorThresholdMP),
			fmt.Sprintf("Set blunder threshold to: %d millipoints", settings.BlunderThresholdMP))
	}

	if !text {
		return printJSON(struct {
			Changes []string `json:"changes"`
		}{Changes: changes})
	}

	fmt.Println("Database metadata updated:")
	for _, change := range changes {
		fmt.Printf("  - %s\n", change)
	}

	return nil
}
