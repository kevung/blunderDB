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
	infoCmd := flag.NewFlagSet("info", flag.ContinueOnError)

	// Define flags
	dbPath := infoCmd.String("db", "", "Path to the database file (required)")
	format := infoCmd.String("format", "text", "Output format: text, json")
	estimate := infoCmd.Bool("estimate", false, "Estimate the tables beyond 200000 rows instead of counting them, and skip the blunders there unless their cached count still holds (default: count every row)")

	infoCmd.Usage = func() {
		fmt.Fprintln(cli.stdout(), "Usage: blunderdb info [options]")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Display database metadata and statistics.")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Options:")
		infoCmd.PrintDefaults()
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "Examples:")
		fmt.Fprintln(cli.stdout(), "  # Display database info")
		fmt.Fprintln(cli.stdout(), "  blunderdb info --db database.db")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "  # Answer at once on a very large database (estimated counts)")
		fmt.Fprintln(cli.stdout(), "  blunderdb info --db database.db --estimate")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "  # Output as JSON")
		fmt.Fprintln(cli.stdout(), "  blunderdb info --db database.db --format json")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "  # See where a database came from (works on a protected .dbx too)")
		fmt.Fprintln(cli.stdout(), "  blunderdb info --db cours.db")
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
		return cli.reportContainer(*dbPath, strings.ToLower(*format) == "json")
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
	statsFn := cli.db.GetDatabaseStats
	if *estimate {
		statsFn = cli.db.GetDatabaseStatsEstimate
	}
	stats, err := statsFn()
	if err != nil {
		return fmt.Errorf("failed to get database stats: %w", err)
	}

	// The library's own thresholds (ADR-0046). They are printed because the
	// blunder count just above depends on them: a number of blunders without
	// the line it was drawn at cannot be compared with anybody else's.
	settings, err := cli.db.GetLibrarySettings()
	if err != nil {
		return fmt.Errorf("failed to read library settings: %w", err)
	}

	// Format output
	if strings.ToLower(*format) == "json" {
		output := map[string]interface{}{
			"path":             *dbPath,
			"metadata":         metadata,
			"stats":            stats,
			"library_settings": settings,
			"issuance":         iss,
		}
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format JSON: %w", err)
		}
		fmt.Fprintln(cli.stdout(), string(jsonData))
	} else {
		fmt.Fprintln(cli.stdout(), "Database Information")
		fmt.Fprintln(cli.stdout(), strings.Repeat("=", 50))
		fmt.Fprintf(cli.stdout(), "Path: %s\n\n", *dbPath)

		fmt.Fprintln(cli.stdout(), "Metadata:")
		if v, ok := metadata["database_version"]; ok {
			fmt.Fprintf(cli.stdout(), "  Version: %s\n", v)
		}
		if v, ok := metadata["user"]; ok && v != "" {
			fmt.Fprintf(cli.stdout(), "  User: %s\n", v)
		}
		if v, ok := metadata["description"]; ok && v != "" {
			fmt.Fprintf(cli.stdout(), "  Description: %s\n", v)
		}
		if v, ok := metadata["dateOfCreation"]; ok && v != "" {
			fmt.Fprintf(cli.stdout(), "  Date of Creation: %s\n", v)
		}

		approx := map[string]bool{}
		if names, ok := stats["approximate"].([]string); ok {
			for _, n := range names {
				approx[n] = true
			}
		}
		count := func(label, key, name string) {
			if n, ok := stats[key].(int64); ok {
				mark := ""
				if approx[name] {
					mark = "≈ "
				}
				fmt.Fprintf(cli.stdout(), "  %s: %s%d\n", label, mark, n)
			}
		}

		fmt.Fprintln(cli.stdout(), "\nStatistics:")
		count("Positions", "position_count", "positions")
		count("Analyses", "analysis_count", "analyses")
		count("Matches", "match_count", "matches")
		count("Games", "game_count", "games")
		count("Moves", "move_count", "moves")
		if _, ok := stats["blunder_count"]; ok {
			count("Blunders", "blunder_count", "blunders")
		} else {
			fmt.Fprintln(cli.stdout(), "  Blunders: not counted on a large database (omit --estimate)")
		}

		fmt.Fprintln(cli.stdout(), "\nThresholds (millipoints):")
		fmt.Fprintf(cli.stdout(), "  Error: %d\n", settings.ErrorThresholdMP)
		fmt.Fprintf(cli.stdout(), "  Blunder: %d\n", settings.BlunderThresholdMP)

		cli.printIssuance(iss)
	}

	return nil
}

// printIssuance reports where a database says it comes from. It prints nothing at all for
// the overwhelmingly common case — a database that was never watermarked — and there is
// nothing else to print: no recipient, no holder, no history. See ADR-0007.
func (cli *CLI) printIssuance(iss domain.IssuanceInfo) {
	if iss.Watermark == nil {
		return
	}
	fmt.Fprintln(cli.stdout(), "\nOrigin:")
	cli.printWatermark(iss.Watermark, "  ")
}

func (cli *CLI) printWatermark(w *domain.WatermarkInfo, indent string) {
	fmt.Fprintf(cli.stdout(), "%s%s\n", indent, w.Origin)
	fmt.Fprintf(cli.stdout(), "%sProduced by:  %s  (%s)  %s\n", indent, w.IssuerName, w.IssuerFingerprint, verdict(w))
	fmt.Fprintf(cli.stdout(), "%sMarked on:    %s\n", indent, shortDate(w.IssuedAt))
	if w.Note != "" {
		fmt.Fprintf(cli.stdout(), "%sNote:         %s\n", indent, w.Note)
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
func (cli *CLI) reportContainer(path string, asJSON bool) error {
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
		fmt.Fprintln(cli.stdout(), string(jsonData))
		return nil
	}
	fmt.Fprintln(cli.stdout(), "Protected blunderDB file")
	fmt.Fprintln(cli.stdout(), strings.Repeat("=", 50))
	fmt.Fprintf(cli.stdout(), "Path: %s\n\n", path)
	fmt.Fprintln(cli.stdout(), "The database itself is encrypted. Its origin, below, is readable without the")
	fmt.Fprintln(cli.stdout(), "password.")
	fmt.Fprintln(cli.stdout())
	if iss.Watermark == nil {
		fmt.Fprintln(cli.stdout(), "This file carries no watermark.")
		return nil
	}
	cli.printWatermark(iss.Watermark, "  ")
	return nil
}
