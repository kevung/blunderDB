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
	from := fs.String("from", "", "Path to the library to copy from (required, at this blunderDB's schema; the transfer writes nothing in it)")
	to := fs.String("to", "", "Path to the library to copy into (required; created when missing)")
	sel := selectionFlags(fs, "the transfer")
	format := fs.String("format", "text", "Output format: text or json")
	fs.Usage = func() {
		fmt.Println("Usage: blunderdb transfer --from <library.db> --to <library.db> [options]")
		fmt.Println()
		fmt.Println("Copy part of one library into another. Every family goes by default; each one can")
		fmt.Println("be left out (--matches=false) or narrowed to some ids (--collection-ids=1,2).")
		fmt.Println("Without ids, every position of the source goes, filed or not. As soon as ids are")
		fmt.Println("named, only the positions the families kept reach go, with their analyses and")
		fmt.Println("comments: a collection's members, a match's moves, a lesson's steps, a deck's cards.")
		fmt.Println("A match named by id brings its tournament (name, date, place), not the tournament's")
		fmt.Println("other matches; --tournaments=false copies it without one. An id the source does")
		fmt.Println("not hold is an error.")
		fmt.Println()
		fmt.Println("The copy merges like an import: a position the destination holds is not added")
		fmt.Println("twice, analyses and comments merge, a match, a collection or a lesson it already")
		fmt.Println("holds is completed, not duplicated. Nothing is written that says where the copy")
		fmt.Println("came from, nor anything in the source: a source at another schema version is")
		fmt.Println("refused (open an older one once with blunderDB to upgrade it).")
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
		fmt.Println("  # Copy one match, with its positions and its tournament, back the other way")
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
	// Opening a library migrates an older one, a write the source must not
	// receive; a newer one holds families this build would drop in silence.
	// Its schema is read without writing first, and either is refused.
	probe, err := openReadOnly(context.Background(), *from)
	if err != nil {
		return fmt.Errorf("source library: %w", err)
	}
	probe.Close()
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
