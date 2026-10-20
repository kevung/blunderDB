package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// runTransfer copies part of one library into another (ADR-0102): the
// selection of a database export, merged as the import of that export would
// merge it, without a file in between and without a trace of where the copy
// came from.
func (cli *CLI) runTransfer(args []string) error {
	fs := flag.NewFlagSet("transfer", flag.ContinueOnError)
	from := fs.String("from", "", "Path to the library to copy from (required; the transfer writes nothing in it)")
	to := fs.String("to", "", "Path to the library to copy into (required; created when missing)")
	sel := selectionFlags(fs, "the transfer")
	format := fs.String("format", "text", "Output format: text or json")
	fs.Usage = func() {
		fmt.Println("Usage: blunderdb transfer --from <library.db> --to <library.db> [options]")
		fmt.Println()
		fmt.Println("Copy part of one library into another. The selection is the database export's:")
		fmt.Println("every family by default, each one left out (--matches=false) or narrowed")
		fmt.Println("(--collection-ids=1,2). The copy merges like an import of that export: a position")
		fmt.Println("the destination holds is not added twice, analyses and comments merge, a match,")
		fmt.Println("a collection or a lesson it already holds is completed, not duplicated.")
		fmt.Println("Nothing is written that says where the copy came from, nor anything in the source.")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  # Copy a whole library into another")
		fmt.Println("  blunderdb transfer --from club.db --to mine.db")
		fmt.Println()
		fmt.Println("  # Copy two collections, with their positions, and nothing else")
		fmt.Println("  blunderdb transfer --from club.db --to mine.db --collection-ids=3,7 \\")
		fmt.Println("      --matches=false --tournaments=false --lessons=false --decks=false")
		fmt.Println()
		fmt.Println("  # Copy one match back the other way")
		fmt.Println("  blunderdb transfer --from mine.db --to club.db --match-ids=12 --collections=false --lessons=false --decks=false")
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *from == "" || *to == "" {
		fs.Usage()
		return fmt.Errorf("missing required flag: --from and --to")
	}
	formatLower := strings.ToLower(*format)
	if formatLower != "text" && formatLower != "json" {
		return fmt.Errorf("unknown format: %s (must be 'text' or 'json')", *format)
	}
	text := formatLower == "text"
	opts, err := sel.options()
	if err != nil {
		return err
	}
	// The source is read, never created: a mistyped path would otherwise
	// copy an empty new library and report success.
	if info, err := os.Stat(*from); err != nil || info.IsDir() {
		return fmt.Errorf("source library not found: %s", *from)
	}
	if err := cli.initDatabase(*from); err != nil {
		return err
	}
	if cli.db.IsOpenDatabase(*to) {
		return fmt.Errorf("--from and --to name the same library: %w", ingest.ErrSelfTransfer)
	}
	dst := &CLI{db: NewDatabase()}
	if err := dst.initDatabase(*to); err != nil {
		return err
	}
	defer dst.db.Close()

	var report ingest.TransferReport
	err = withInterruptibleContext(func() {
		if text {
			fmt.Println("\nCancelling...")
		}
	}, func(ctx context.Context) error {
		var err error
		report, err = cli.db.TransferTo(ctx, dst.db, opts)
		return err
	})
	if errors.Is(err, context.Canceled) {
		return fmt.Errorf("transfer cancelled: the destination is unchanged")
	}
	if err != nil {
		return fmt.Errorf("transfer failed: %w", err)
	}
	if !text {
		return printJSON(report)
	}
	fmt.Printf("Transferred %s -> %s\n", *from, *to)
	fmt.Println("Read from the source:")
	printExportReport(report.Exported)
	im := report.Imported
	fmt.Println("Merged into the destination:")
	fmt.Printf("  positions %d, matches %d (already held %d), collections changed %d, lessons %d, Anki decks %d\n",
		im.SavedPositions, im.Matches, im.SkippedDuplicates, im.Collections, im.Lessons, im.Decks)
	if im.Tournaments > 0 || im.Transcriptions > 0 || im.Aliases > 0 {
		fmt.Printf("  tournaments %d, transcriptions %d, aliases %d\n", im.Tournaments, im.Transcriptions, im.Aliases)
	}
	for _, name := range im.LivingCollectionsSkipped {
		fmt.Printf("  collection %q: its namesake here is living, it received no members\n", name)
	}
	for _, name := range im.LivingCollectionsFrozen {
		fmt.Printf("  living collection %q arrived frozen: its query names rows the transfer did not carry\n", name)
	}
	for _, name := range im.LivingCollectionsOnManual {
		fmt.Printf("  living collection %q: its namesake here is a hand-made list, left as it was\n", name)
	}
	return nil
}
