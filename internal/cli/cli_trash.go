package cli

import (
	"flag"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// runTrash is `blunderdb trash`: what was deleted, and how to put it back
// (ADR-0036). The CLI's `delete` deletes outright, since a script expects the
// row gone without a snapshot growing the file; `trash delete` goes through
// the trash.
func (cli *CLI) runTrash(args []string) error {
	if len(args) == 0 {
		cli.printTrashUsage()
		return fmt.Errorf("missing subcommand")
	}
	sub := args[0]
	rest := args[1:]

	fs := flag.NewFlagSet("trash "+sub, flag.ContinueOnError)
	dbPath := fs.String("db", "", "Path to the database file (required)")
	format := fs.String("format", "text", "Output format: text or json")
	id := fs.Int64("id", 0, "Trash entry (restore, discard) or object (delete) id")
	kind := fs.String("kind", "", "Narrow the listing: position, collection, comment, match, anki_card")
	limit := fs.Int("limit", 50, "Maximum entries listed")
	olderThan := fs.Int("older-than", 0, "empty: drop only entries older than this many days (0 = all)")
	fs.Usage = cli.printTrashUsage

	if err := fs.Parse(rest); err != nil {
		return err
	}
	if *dbPath == "" {
		cli.printTrashUsage()
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
			fmt.Fprintf(cli.stdout(), "Restored as id %d.\n", restored.ID)
			for _, w := range restored.Warnings {
				fmt.Fprintf(cli.stdout(), "Warning: %s.\n", w.Message)
			}
			return nil
		}
		return cli.printJSON(struct {
			Restored int64                 `json:"restored"`
			Warnings []domain.TrashWarning `json:"warnings,omitempty"`
		}{restored.ID, restored.Warnings})
	case "discard":
		if *id == 0 {
			return fmt.Errorf("discard: missing required flag: --id")
		}
		if err := cli.db.DiscardFromTrash(*id); err != nil {
			return fmt.Errorf("discard: %w", err)
		}
		if text {
			fmt.Fprintf(cli.stdout(), "Entry %d discarded; it cannot be restored now.\n", *id)
			return nil
		}
		return cli.printJSON(struct {
			Discarded int64 `json:"discarded"`
		}{*id})
	case "empty":
		n, err := cli.db.EmptyTrash(*olderThan)
		if err != nil {
			return fmt.Errorf("empty: %w", err)
		}
		if text {
			fmt.Fprintf(cli.stdout(), "%d entrie(s) dropped.\n", n)
			return nil
		}
		return cli.printJSON(struct {
			Purged int `json:"purged"`
		}{n})
	case "delete":
		return cli.trashDelete(*kind, *id, text)
	default:
		cli.printTrashUsage()
		return fmt.Errorf("unknown subcommand: %s", sub)
	}
}

func (cli *CLI) trashList(kind string, limit int, text bool) error {
	entries, err := cli.db.ListTrash(kind, limit, 0)
	if err != nil {
		return err
	}
	if !text {
		return cli.printJSON(entries)
	}
	if len(entries) == 0 {
		fmt.Fprintln(cli.stdout(), "The trash is empty.")
		return nil
	}
	fmt.Fprintf(cli.stdout(), "%d entrie(s), most recently deleted first:\n\n", len(entries))
	w := tabwriter.NewWriter(cli.stdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tKind\tDeleted\tWhat")
	fmt.Fprintln(w, "--\t----\t-------\t----")
	for _, e := range entries {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", e.ID, e.Kind, e.DeletedAt, e.Label)
	}
	w.Flush()
	fmt.Fprintf(cli.stdout(), "\nEntries older than %d days are dropped by `blunderdb vacuum`.\n", domain.TrashRetentionDays)
	fmt.Fprintln(cli.stdout(), "Use --id with `restore` to put one back, or with `discard` to drop it now.")
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
	case domain.TrashMatch:
		trashID, err = cli.db.TrashMatch(id)
	default:
		return fmt.Errorf("delete: --kind must be one of position, collection, comment, match")
	}
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	if text {
		fmt.Fprintf(cli.stdout(), "Deleted; trash entry %d, restorable for %d days.\n", trashID, domain.TrashRetentionDays)
		return nil
	}
	return cli.printJSON(struct {
		TrashID int64 `json:"trashId"`
	}{trashID})
}

func (cli *CLI) printTrashUsage() {
	fmt.Fprintln(cli.stdout(), "Usage: blunderdb trash <subcommand> [options]")
	fmt.Fprintln(cli.stdout())
	fmt.Fprintln(cli.stdout(), "What was deleted through the trash, and how to put it back.")
	fmt.Fprintln(cli.stdout(), "A delete is still a delete: a JSON snapshot of what disappears is")
	fmt.Fprintln(cli.stdout(), "written first, and nothing else in the database knows the trash")
	fmt.Fprintln(cli.stdout(), "exists — no search filter, no statistic, no retention rule.")
	fmt.Fprintln(cli.stdout())
	fmt.Fprintln(cli.stdout(), "Subcommands:")
	fmt.Fprintln(cli.stdout(), "  list                 What is in the trash, most recently deleted first.")
	fmt.Fprintln(cli.stdout(), "  restore --id N       Put entry N back, and drop it from the trash.")
	fmt.Fprintln(cli.stdout(), "  discard --id N       Drop entry N now, without restoring it.")
	fmt.Fprintln(cli.stdout(), "  empty [--older-than D]  Drop everything, or only what is older than D days.")
	fmt.Fprintln(cli.stdout(), "  delete --kind K --id N  Delete an object THROUGH the trash, so it can be undone.")
	fmt.Fprintln(cli.stdout())
	fmt.Fprintln(cli.stdout(), "Options:")
	fmt.Fprintln(cli.stdout(), "  --db string          Path to the database file (required)")
	fmt.Fprintln(cli.stdout(), "  --kind string        position, collection, comment, match (delete); narrows list, which also takes anki_card")
	fmt.Fprintln(cli.stdout(), "  --id int             Trash entry (restore, discard) or object (delete) id")
	fmt.Fprintln(cli.stdout(), "  --limit int          Maximum entries listed (default 50)")
	fmt.Fprintln(cli.stdout(), "  --older-than int     empty: drop only entries older than this many days (0 = all)")
	fmt.Fprintln(cli.stdout(), "  --format string      text (default) or json")
	fmt.Fprintln(cli.stdout())
	fmt.Fprintln(cli.stdout(), "`blunderdb delete` still deletes outright: a script that deletes a")
	fmt.Fprintln(cli.stdout(), "position expects it gone. Use `trash delete` to keep the undo.")
	fmt.Fprintln(cli.stdout())
	fmt.Fprintln(cli.stdout(), "Examples:")
	fmt.Fprintln(cli.stdout(), "  blunderdb trash delete --db base.db --kind position --id 412")
	fmt.Fprintln(cli.stdout(), "  blunderdb trash delete --db base.db --kind match --id 12")
	fmt.Fprintln(cli.stdout(), "  blunderdb trash list --db base.db")
	fmt.Fprintln(cli.stdout(), "  blunderdb trash restore --db base.db --id 3")
	fmt.Fprintln(cli.stdout(), "  blunderdb trash empty --db base.db --older-than 30")
}
