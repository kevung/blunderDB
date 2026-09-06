package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// runTrash is `blunderdb trash`: what was deleted, and how to put it back
// (issue #285, ADR-0036).
//
// The CLI's own `delete` still deletes outright — a script that deletes a
// position expects it gone, and quietly leaving a snapshot behind would grow a
// file nobody asked to grow. Deleting THROUGH the trash is what
// `trash delete` is for, and the difference is the point.
func (cli *CLI) runTrash(args []string) error {
	if len(args) == 0 {
		printTrashUsage()
		return fmt.Errorf("missing subcommand")
	}
	sub := args[0]
	rest := args[1:]

	fs := flag.NewFlagSet("trash "+sub, flag.ContinueOnError)
	dbPath := fs.String("db", "", "Path to the database file (required)")
	format := fs.String("format", "text", "Output format: text or json")
	id := fs.Int64("id", 0, "Trash entry (restore, discard) or object (delete) id")
	kind := fs.String("kind", "", "Narrow the listing: position, collection, comment, anki_card")
	limit := fs.Int("limit", 50, "Maximum entries listed")
	olderThan := fs.Int("older-than", 0, "empty: drop only entries older than this many days (0 = all)")
	fs.Usage = printTrashUsage

	if err := fs.Parse(rest); err != nil {
		return err
	}
	if *dbPath == "" {
		printTrashUsage()
		return fmt.Errorf("missing required flag: --db")
	}
	formatLower := strings.ToLower(*format)
	if formatLower != "text" && formatLower != "json" {
		return fmt.Errorf("unknown format: %s (must be 'text' or 'json')", *format)
	}
	text := formatLower != "json"

	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}

	switch sub {
	case "list":
		return cli.trashList(*kind, *limit, text)
	case "restore":
		if *id == 0 {
			return fmt.Errorf("restore: missing required flag: --id")
		}
		restored, err := cli.db.RestoreFromTrash(*id)
		if err != nil {
			return fmt.Errorf("restore: %w", err)
		}
		if text {
			fmt.Printf("Restored as id %d.\n", restored)
			return nil
		}
		return printJSON(struct {
			Restored int64 `json:"restored"`
		}{restored})
	case "discard":
		if *id == 0 {
			return fmt.Errorf("discard: missing required flag: --id")
		}
		if err := cli.db.DiscardFromTrash(*id); err != nil {
			return fmt.Errorf("discard: %w", err)
		}
		if text {
			fmt.Printf("Entry %d discarded; it cannot be restored now.\n", *id)
			return nil
		}
		return printJSON(struct {
			Discarded int64 `json:"discarded"`
		}{*id})
	case "empty":
		n, err := cli.db.EmptyTrash(*olderThan)
		if err != nil {
			return fmt.Errorf("empty: %w", err)
		}
		if text {
			fmt.Printf("%d entrie(s) dropped.\n", n)
			return nil
		}
		return printJSON(struct {
			Purged int `json:"purged"`
		}{n})
	case "delete":
		return cli.trashDelete(*kind, *id, text)
	default:
		printTrashUsage()
		return fmt.Errorf("unknown subcommand: %s", sub)
	}
}

func (cli *CLI) trashList(kind string, limit int, text bool) error {
	entries, err := cli.db.ListTrash(kind, limit, 0)
	if err != nil {
		return err
	}
	if !text {
		return printJSON(entries)
	}
	if len(entries) == 0 {
		fmt.Println("The trash is empty.")
		return nil
	}
	fmt.Printf("%d entrie(s), most recently deleted first:\n\n", len(entries))
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tKind\tDeleted\tWhat")
	fmt.Fprintln(w, "--\t----\t-------\t----")
	for _, e := range entries {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", e.ID, e.Kind, e.DeletedAt, e.Label)
	}
	w.Flush()
	fmt.Printf("\nEntries older than %d days are dropped by `blunderdb vacuum`.\n", domain.TrashRetentionDays)
	fmt.Println("Use --id with `restore` to put one back, or with `discard` to drop it now.")
	return nil
}

// trashDelete deletes an object through the trash, so the gesture can be
// undone. --kind says what --id names.
func (cli *CLI) trashDelete(kind string, id int64, text bool) error {
	if id == 0 {
		return fmt.Errorf("delete: missing required flag: --id")
	}
	var (
		trashID int64
		err     error
	)
	switch domain.TrashKind(kind) {
	case domain.TrashPosition:
		trashID, err = cli.db.TrashPosition(id)
	case domain.TrashCollection:
		trashID, err = cli.db.TrashCollection(id)
	case domain.TrashComment:
		trashID, err = cli.db.TrashCommentEntry(id)
	default:
		return fmt.Errorf("delete: --kind must be one of position, collection, comment")
	}
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	if text {
		fmt.Printf("Deleted; trash entry %d, restorable for %d days.\n", trashID, domain.TrashRetentionDays)
		return nil
	}
	return printJSON(struct {
		TrashID int64 `json:"trashId"`
	}{trashID})
}

func printTrashUsage() {
	fmt.Println("Usage: blunderdb trash <subcommand> [options]")
	fmt.Println()
	fmt.Println("What was deleted through the trash, and how to put it back.")
	fmt.Println("A delete is still a delete: a JSON snapshot of what disappears is")
	fmt.Println("written first, and nothing else in the database knows the trash")
	fmt.Println("exists — no search filter, no statistic, no retention rule.")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  list                 What is in the trash, most recently deleted first.")
	fmt.Println("  restore --id N       Put entry N back, and drop it from the trash.")
	fmt.Println("  discard --id N       Drop entry N now, without restoring it.")
	fmt.Println("  empty [--older-than D]  Drop everything, or only what is older than D days.")
	fmt.Println("  delete --kind K --id N  Delete an object THROUGH the trash, so it can be undone.")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --db string          Path to the database file (required)")
	fmt.Println("  --kind string        position, collection, comment (delete); also narrows list")
	fmt.Println("  --limit int          Maximum entries listed (default 50)")
	fmt.Println("  --format string      text (default) or json")
	fmt.Println()
	fmt.Println("`blunderdb delete` still deletes outright: a script that deletes a")
	fmt.Println("position expects it gone. Use `trash delete` to keep the undo.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  blunderdb trash delete --db base.db --kind position --id 412")
	fmt.Println("  blunderdb trash list --db base.db")
	fmt.Println("  blunderdb trash restore --db base.db --id 3")
	fmt.Println("  blunderdb trash empty --db base.db --older-than 30")
}
