package cli

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/printsheet"
)

// runMatchSheet is `match sheet`: the printable match sheet of the Match
// panel's Print action, through the same MatchSheetHTML as the daemon's
// matches.sheet.
func (cli *CLI) runMatchSheet(args []string) error {
	def := printsheet.DefaultMatchOptions()
	fs := flag.NewFlagSet("match sheet", flag.ContinueOnError)
	dbPath := fs.String("db", "", "Path to the database file (required)")
	ids := fs.String("match", "", "Comma-separated match ids, one sheet each, at most "+strconv.Itoa(printsheet.MaxMatches)+" (required)")
	output := fs.String("output", "", "HTML file to write (default: standard output)")
	lang := fs.String("lang", "en", "Sheet language: "+strings.Join(printsheet.Languages(), ", "))
	player := fs.String("player", "", "Player the sheet is printed for (default: the Main player, else player 1)")
	opp := fs.Bool("opponent-blunders", def.OpponentBlunders, "Number and detail the opponent's blunders too")
	analysis := fs.Bool("analysis", def.Analysis, "Write the short analysis of each listed position")
	comments := fs.Bool("comments", def.Comments, "Print the match's and the positions' comments, whole")
	expanded := fs.Bool("expanded", def.Expanded, "Write the ArkIDs in their developed form")
	code := fs.Bool("reimport-code", def.ReimportCode, "Draw the match's reimport QR code and embed it in the file")
	fs.Usage = func() {
		out := cli.stdout()
		fmt.Fprintln(out, "Usage: blunderdb match sheet --db FILE --match IDS [options]")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Write the printable sheet of matches, one A4 sheet a match, as one self-contained")
		fmt.Fprintln(out, "HTML file a browser prints or saves as PDF: header, transcription with ? and ??")
		fmt.Fprintln(out, "at the library's thresholds, and apart the positions to review written in ArkID,")
		fmt.Fprintln(out, "numbered in the transcription. The body shrinks from 9 to 5.5 pt to fit one page;")
		fmt.Fprintln(out, "the positions go overleaf when they still do not, and on further sheets when the")
		fmt.Fprintln(out, "overleaf is full; a comment longer than a sheet goes on over the next: nothing is")
		fmt.Fprintln(out, "ever cut off. At most "+strconv.Itoa(printsheet.MaxMatches)+" matches a file.")
		fmt.Fprintln(out, "Nothing is written to the library.")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Options:")
		fs.PrintDefaults()
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Examples:")
		fmt.Fprintln(out, "  blunderdb match sheet --db database.db --match 12 --lang fr --output feuille.html")
		fmt.Fprintln(out, "  blunderdb match sheet --db database.db --match 12,15 --opponent-blunders --reimport-code=false")
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dbPath == "" || *ids == "" {
		fs.Usage()
		return fmt.Errorf("missing required flag: --db and --match")
	}
	matchIDs, err := parseIDList(*ids)
	if err != nil {
		return err
	}
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}
	page, err := cli.db.MatchSheetHTML(matchIDs, printsheet.MatchOptions{
		Language: *lang, Player: *player, OpponentBlunders: *opp, Analysis: *analysis,
		Comments: *comments, Expanded: *expanded, ReimportCode: *code,
	})
	if err != nil {
		return fmt.Errorf("match sheet: %w", err)
	}
	if *output == "" {
		fmt.Fprint(cli.stdout(), page)
		return nil
	}
	if err := os.WriteFile(*output, []byte(page), 0o644); err != nil {
		return fmt.Errorf("write sheet: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Match sheet written to %s\n", *output)
	return nil
}
