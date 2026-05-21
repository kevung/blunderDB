package cli

import (
	"flag"
	"fmt"
)

// runEdit handles the edit command
func (cli *CLI) runEdit(args []string) error {
	editCmd := flag.NewFlagSet("edit", flag.ExitOnError)

	// Define flags
	dbPath := editCmd.String("db", "", "Path to the database file (required)")
	user := editCmd.String("user", "", "Set user name")
	description := editCmd.String("description", "", "Set description")
	clearUser := editCmd.Bool("clear-user", false, "Clear user name")
	clearDescription := editCmd.Bool("clear-description", false, "Clear description")

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
	if *user == "" && *description == "" && !*clearUser && !*clearDescription {
		editCmd.Usage()
		return fmt.Errorf("no edit options provided")
	}

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
		return fmt.Errorf("failed to save metadata: %v", err)
	}

	fmt.Println("Database metadata updated:")
	for _, change := range changes {
		fmt.Printf("  - %s\n", change)
	}

	return nil
}
