package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/capture"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// runCapture handles the capture command: the position a screen capture of
// another program's board shows, with what the reading could not settle.
// Saving is a choice: an uncertain or incomplete reading is refused unless
// --force says the user has checked it.
func (cli *CLI) runCapture(args []string) error {
	fs := flag.NewFlagSet("capture", flag.ContinueOnError)
	format := fs.String("format", "text", "Output format: text, json, xgid")
	dbPath := fs.String("db", "", "Database to save the position into (with --save)")
	save := fs.Bool("save", false, "Save the position into --db as an individual position")
	force := fs.Bool("force", false, "Save even when the reading has doubts or leaves the score or the turn unread")
	fs.Usage = func() {
		fmt.Println("Usage: blunderdb capture [options] <image>")
		fmt.Println()
		fmt.Println("Read the position a screen capture of an eXtreme Gammon board shows")
		fmt.Println("(PNG, JPEG or GIF, at any size): checkers, bar, borne-off checkers,")
		fmt.Println("dice, cube and player on roll. The score is not read. Whatever the")
		fmt.Println("reading cannot settle is listed as a doubt, never guessed.")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  blunderdb capture screenshot.png")
		fmt.Println("  blunderdb capture --format xgid screenshot.png")
		fmt.Println("  blunderdb capture --db my.db --save screenshot.png")
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("expected exactly one image")
	}
	data, err := os.ReadFile(fs.Arg(0))
	if err != nil {
		return err
	}
	rd, err := capture.Decode(data)
	if err != nil {
		return err
	}

	switch *format {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(rd); err != nil {
			return err
		}
	case "xgid":
		fmt.Println("XGID=" + domain.EncodeXGID(&rd.Position))
	default:
		fmt.Printf("Software: %s (fit %.2f)\n", rd.Software, rd.Fit)
		fmt.Printf("XGID=%s\n", domain.EncodeXGID(&rd.Position))
		fmt.Printf("Not read: %s\n", strings.Join(rd.Unread, ", "))
		for _, d := range rd.Doubts {
			if d.Field == "point" || d.Field == "bar" {
				fmt.Printf("Doubt: %s %d — %s\n", d.Field, d.Point, d.Reason)
			} else {
				fmt.Printf("Doubt: %s — %s\n", d.Field, d.Reason)
			}
		}
	}

	if !*save {
		return nil
	}
	if *dbPath == "" {
		return fmt.Errorf("--save needs --db")
	}
	// An unread score is saved as money play and an unread turn as a guess:
	// both are wrong values, not blanks, so the user must have set them.
	if (len(rd.Doubts) > 0 || len(rd.Unread) > 0) && !*force {
		return fmt.Errorf("the reading has %d doubt(s) and does not show: %s; check and set them, then save with --force", len(rd.Doubts), strings.Join(rd.Unread, ", "))
	}
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}
	res, err := cli.db.SaveIndividualPosition(&rd.Position)
	if err != nil {
		return err
	}
	if *format != "json" {
		fmt.Printf("Saved as position %d (already there: %v)\n", res.ID, res.Existed)
	}
	return nil
}
