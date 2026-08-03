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
		fmt.Println("  # Identify a copy that came back to you (never writes to the file)")
		fmt.Println("  blunderdb info --db suspect.db")
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

	// This command is the forensic entry point and must stay non-mutating: it never records
	// a holder, so examining a suspect file cannot write the examiner's own machine into the
	// evidence.
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
		return fmt.Errorf("failed to get database stats: %v", err)
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
			return fmt.Errorf("failed to format JSON: %v", err)
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

// printIssuance reports how a database was handed out and where its contents came from. It
// prints nothing at all for the overwhelmingly common case — a database that was never
// issued and never imported an issued copy — which is what "discreet" means here.
func printIssuance(iss domain.IssuanceInfo) {
	if iss.Watermark != nil {
		fmt.Println("\nIssued copy:")
		printWatermark(iss.Watermark, "  ")
		if len(iss.Holders) == 0 {
			fmt.Println("  Holders:          none recorded yet")
		} else {
			label := "machines"
			if len(iss.Holders) == 1 {
				label = "machine"
			}
			fmt.Printf("  Holders:          %d distinct %s\n", len(iss.Holders), label)
			for _, h := range iss.Holders {
				fmt.Printf("    %s  %s → %s  (%d openings)\n",
					h.Fingerprint, shortDate(h.FirstSeen), shortDate(h.LastSeen), h.Openings)
			}
			if !iss.ChainIntact {
				fmt.Println("    !! the holder registry has been altered — an entry was removed or reordered")
			}
		}
	}

	if len(iss.Lineage) > 0 {
		fmt.Println("\nContains material from issued copies:")
		for _, w := range iss.Lineage {
			printWatermark(&w, "  ")
			fmt.Println()
		}
	}

	if len(iss.Issued) > 0 {
		fmt.Printf("\nCopies issued from this database (%d):\n", len(iss.Issued))
		for _, r := range iss.Issued {
			who := r.Recipient
			if who == "" {
				who = "(collective)"
			}
			fmt.Printf("  %3d/%-3d  %-24s  %s  %s\n", r.Number, r.Total, who, shortDate(r.IssuedAt), r.FileName)
		}
		fmt.Println("  (this register never travels inside an issued copy)")
	}
}

func printWatermark(w *domain.WatermarkInfo, indent string) {
	fmt.Printf("%sDistribution:     %s\n", indent, w.Distribution)
	fmt.Printf("%sIssued by:        %s  (%s)  %s\n", indent, w.IssuerName, w.IssuerFingerprint, verdict(w))
	if w.Nominative {
		fmt.Printf("%sIssued to:        %s", indent, w.Recipient)
		if w.Total > 0 {
			fmt.Printf("   — copy %d of %d", w.Number, w.Total)
		}
		fmt.Println()
	} else {
		fmt.Printf("%sIssued to:        the distribution as a whole (no recipient named)\n", indent)
	}
	fmt.Printf("%sIssued on:        %s\n", indent, shortDate(w.IssuedAt))
}

// verdict states what verification concluded. "issued by you" is the case that closes the
// loop locally: the person examining a copy that came back is normally the person who signed
// it, so no fingerprint ever had to be published.
func verdict(w *domain.WatermarkInfo) string {
	switch {
	case !w.SignatureValid:
		return "!! SIGNATURE INVALID — altered or forged"
	case w.IssuedByYou:
		return "✓ signature verified — issued by you"
	default:
		return "✓ signature verified — issued by another key"
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
		return fmt.Errorf("cannot read the protected file: %v", err)
	}
	if asJSON {
		jsonData, err := json.MarshalIndent(map[string]interface{}{
			"path": path, "protected": true, "issuance": iss,
		}, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format JSON: %v", err)
		}
		fmt.Println(string(jsonData))
		return nil
	}
	fmt.Println("Protected blunderDB copy")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("Path: %s\n\n", path)
	fmt.Println("The database itself is encrypted; the watermark below is readable without")
	fmt.Println("the password, so a copy found in the wild stays identifiable.")
	fmt.Println()
	if iss.Watermark == nil {
		fmt.Println("This file carries no watermark.")
		return nil
	}
	printWatermark(iss.Watermark, "  ")
	return nil
}
