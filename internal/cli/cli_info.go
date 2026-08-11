package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/issuance"
)

// runInfo handles the info command
func (cli *CLI) runInfo(args []string) error {
	infoCmd := flag.NewFlagSet("info", flag.ExitOnError)

	// Define flags
	dbPath := infoCmd.String("db", "", "Path to the database file (required)")
	format := infoCmd.String("format", "text", "Output format: text, json")

	infoCmd.Usage = func() {
		fmt.Println("Usage: blunderdb info [options]")
		fmt.Println()
		fmt.Println("Display database metadata and statistics.")
		fmt.Println()
		fmt.Println("Options:")
		infoCmd.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  # Display database info")
		fmt.Println("  blunderdb info --db database.db")
		fmt.Println()
		fmt.Println("  # Output as JSON")
		fmt.Println("  blunderdb info --db database.db --format json")
		fmt.Println()
		fmt.Println("  # See where a database came from (works on a protected .dbx too)")
		fmt.Println("  blunderdb info --db cours.db")
	}

	if err := infoCmd.Parse(args); err != nil {
		return err
	}

	// Validate required flags
	if *dbPath == "" {
		infoCmd.Usage()
		return fmt.Errorf("missing required flag: --db")
	}

	// A protected copy is not a database yet: its header is readable without the password,
	// which is exactly what makes a copy found in the wild identifiable. Report it and stop
	// rather than failing to open a file that was never meant to be opened directly.
	if issuance.IsContainer(*dbPath) {
		return reportContainer(*dbPath, strings.ToLower(*format) == "json")
	}

	// Reading a database's origin never writes to it — nothing anywhere records that a file
	// was opened, read or inspected (ADR-0007).
	iss, issErr := database.InspectIssuance(*dbPath)
	if issErr != nil {
		iss = domain.IssuanceInfo{}
	}

	// Initialize database
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}

	// Get metadata
	metadata, err := cli.db.LoadMetadata()
	if err != nil {
		metadata = make(map[string]string)
	}

	// Get stats
	stats, err := cli.db.GetDatabaseStats()
	if err != nil {
		return fmt.Errorf("failed to get database stats: %w", err)
	}

	// Format output
	if strings.ToLower(*format) == "json" {
		output := map[string]interface{}{
			"path":     *dbPath,
			"metadata": metadata,
			"stats":    stats,
			"issuance": iss,
		}
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		fmt.Println("Database Information")
		fmt.Println(strings.Repeat("=", 50))
		fmt.Printf("Path: %s\n\n", *dbPath)

		fmt.Println("Metadata:")
		if v, ok := metadata["database_version"]; ok {
			fmt.Printf("  Version: %s\n", v)
		}
		if v, ok := metadata["user"]; ok && v != "" {
			fmt.Printf("  User: %s\n", v)
		}
		if v, ok := metadata["description"]; ok && v != "" {
			fmt.Printf("  Description: %s\n", v)
		}
		if v, ok := metadata["dateOfCreation"]; ok && v != "" {
			fmt.Printf("  Date of Creation: %s\n", v)
		}

		fmt.Println("\nStatistics:")
		if posCount, ok := stats["position_count"].(int64); ok {
			fmt.Printf("  Positions: %d\n", posCount)
		}
		if analysisCount, ok := stats["analysis_count"].(int64); ok {
			fmt.Printf("  Analyses: %d\n", analysisCount)
		}
		if matchCount, ok := stats["match_count"].(int64); ok {
			fmt.Printf("  Matches: %d\n", matchCount)
		}
		if gameCount, ok := stats["game_count"].(int64); ok {
			fmt.Printf("  Games: %d\n", gameCount)
		}
		if moveCount, ok := stats["move_count"].(int64); ok {
			fmt.Printf("  Moves: %d\n", moveCount)
		}

		printIssuance(iss)
	}

	return nil
}

// printIssuance reports where a database says it comes from. It prints nothing at all for
// the overwhelmingly common case — a database that was never watermarked — and there is
// nothing else to print: no recipient, no holder, no history. See ADR-0007.
func printIssuance(iss domain.IssuanceInfo) {
	if iss.Watermark == nil {
		return
	}
	fmt.Println("\nOrigin:")
	printWatermark(iss.Watermark, "  ")
}

func printWatermark(w *domain.WatermarkInfo, indent string) {
	fmt.Printf("%s%s\n", indent, w.Origin)
	fmt.Printf("%sProduced by:  %s  (%s)  %s\n", indent, w.IssuerName, w.IssuerFingerprint, verdict(w))
	fmt.Printf("%sMarked on:    %s\n", indent, shortDate(w.IssuedAt))
	if w.Note != "" {
		fmt.Printf("%sNote:         %s\n", indent, w.Note)
	}
}

// verdict states what verification concluded. A watermark proves the file was marked by the
// holder of that key and has not been altered; matching it against a published fingerprint
// is what ties the key to a person.
func verdict(w *domain.WatermarkInfo) string {
	switch {
	case !w.SignatureValid:
		return "!! SIGNATURE INVALID — altered or forged"
	case w.IssuedByYou:
		return "✓ signature verified — marked by you"
	default:
		return "✓ signature verified"
	}
}

func shortDate(stamp string) string {
	if len(stamp) >= 10 {
		return stamp[:10]
	}
	return stamp
}

// reportContainer describes a protected copy from its cleartext header alone.
func reportContainer(path string, asJSON bool) error {
	iss, err := database.InspectIssuance(path)
	if err != nil {
		return fmt.Errorf("cannot read the protected file: %w", err)
	}
	if asJSON {
		jsonData, err := json.MarshalIndent(map[string]interface{}{
			"path": path, "protected": true, "issuance": iss,
		}, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format JSON: %w", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}
	fmt.Println("Protected blunderDB file")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("Path: %s\n\n", path)
	fmt.Println("The database itself is encrypted. Its origin, below, is readable without the")
	fmt.Println("password.")
	fmt.Println()
	if iss.Watermark == nil {
		fmt.Println("This file carries no watermark.")
		return nil
	}
	printWatermark(iss.Watermark, "  ")
	return nil
}
