package cli

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/arkqr"
	"github.com/kevung/blunderdb/pkg/blunderdb/printsheet"
)

// runSheet is `sheet`: the printable position list of the interface's Print
// list action, through the same PositionSheetHTML as the daemon's
// positions.sheet.
func (cli *CLI) runSheet(args []string) error {
	def := printsheet.DefaultPositionsOptions()
	fs := flag.NewFlagSet("sheet", flag.ContinueOnError)
	dbPath := fs.String("db", "", "Path to the database file (required)")
	query := fs.String("query", "", "Print the result of this search, in the command bar's grammar (see search --query-help)")
	collection := fs.Int64("collection", 0, "Print this collection; a living one is resolved by its query")
	ids := fs.String("ids", "", "Print these comma-separated position ids, in this order")
	output := fs.String("output", "", "HTML file to write (default: standard output)")
	lang := fs.String("lang", "en", "Sheet language: "+strings.Join(printsheet.Languages(), ", "))
	title := fs.String("title", "", "Title of every page (default: the query, the collection's name)")
	analysis := fs.Bool("analysis", def.Analysis, "Write each position's short analysis on its line")
	verso := fs.Bool("verso", def.Verso, "Print the analyses overleaf, under the same numbers")
	comments := fs.Bool("comments", def.Comments, "Print the positions' comments, whole (overleaf with --verso)")
	expanded := fs.Bool("expanded", def.Expanded, "Write the ArkIDs in their developed form")
	code := fs.Bool("reimport-code", def.ReimportCode, "Draw the positions' reimport QR codes, a block every "+strconv.Itoa(arkqr.Tranche)+" positions, and embed them in the file")
	fs.Usage = func() {
		out := cli.stdout()
		fmt.Fprintln(out, "Usage: blunderdb sheet --db FILE (--query Q | --collection ID | --ids IDS) [options]")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Write a printable list of positions as one self-contained HTML file a browser")
		fmt.Fprintln(out, "prints or saves as PDF: one line a position — number, ArkID, pips of the player")
		fmt.Fprintln(out, "on roll and of the opponent — then its analysis: best play in bold, another and")
		fmt.Fprintln(out, "its loss in thousandths, the chances of winning and of a gammon; for a cube")
		fmt.Fprintln(out, "decision, the right action and what the others lose. The whole list is printed,")
		fmt.Fprintln(out, "on as many numbered pages as it takes, header and legend on each; a comment")
		fmt.Fprintln(out, "longer than a page goes on over the next: nothing is ever cut off. With --verso")
		fmt.Fprintln(out, "each page of positions is followed by its answers. At most "+strconv.Itoa(printsheet.MaxPositions)+" positions;")
		fmt.Fprintln(out, "a longer list is refused. Nothing is written to the library. With --reimport-code")
		fmt.Fprintln(out, "the positions are coded "+strconv.Itoa(arkqr.Tranche)+" at a time, each block of QR codes printed on the")
		fmt.Fprintln(out, "page of its first position and read back alone: `archive import` takes the saved")
		fmt.Fprintln(out, "file, a scan or a photograph and saves the positions, without analyses or comments.")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Options:")
		fs.PrintDefaults()
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Examples:")
		fmt.Fprintln(out, "  blunderdb sheet --db database.db --query 'E>80' --lang fr --output blunders.html")
		fmt.Fprintln(out, "  blunderdb sheet --db database.db --collection 3 --verso --comments=false")
		fmt.Fprintln(out, "  blunderdb sheet --db database.db --query 'E>80' --reimport-code --output blunders.html")
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dbPath == "" {
		fs.Usage()
		return fmt.Errorf("missing required flag: --db")
	}
	src := printsheet.PositionSource{Query: *query, Collection: *collection}
	if *ids != "" {
		list, err := parseIDList(*ids)
		if err != nil {
			return err
		}
		src.IDs = list
	}
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}
	page, err := cli.db.PositionSheetHTML(src, printsheet.PositionsOptions{
		Language: *lang, Title: *title, Analysis: *analysis, Verso: *verso, Comments: *comments, Expanded: *expanded, ReimportCode: *code,
	})
	if err != nil {
		return fmt.Errorf("sheet: %w", err)
	}
	if *output == "" {
		fmt.Fprint(cli.stdout(), page)
		return nil
	}
	if err := os.WriteFile(*output, []byte(page), 0o644); err != nil {
		return fmt.Errorf("write sheet: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Position list written to %s\n", *output)
	return nil
}
