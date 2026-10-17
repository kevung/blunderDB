package cli

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/kevung/blunderdb/pkg/blunderdb/openingbook"
)

// runOpeningBook dispatches `blunderdb openingbook <sub-command>`: the
// exercise « which play at this score? » over the opening book, drawn and
// judged by the same functions the Training tab and the daemon call. No
// database: the book is embedded in the binary.
func (cli *CLI) runOpeningBook(args []string) error {
	if len(args) < 1 {
		cli.printOpeningBookUsage()
		return fmt.Errorf("missing openingbook sub-command")
	}
	sub := strings.ToLower(args[0])
	if sub == "--help" || sub == "-h" || sub == "help" {
		cli.printOpeningBookUsage()
		return nil
	}
	run, ok := cli.openingBookHandlers()[sub]
	if !ok {
		cli.printOpeningBookUsage()
		return fmt.Errorf("unknown openingbook sub-command: %s", args[0])
	}
	return run(args[1:])
}

func (cli *CLI) openingBookHandlers() map[string]func([]string) error {
	return map[string]func([]string) error{
		"status":   cli.runOpeningBookStatus,
		"question": cli.runOpeningBookQuestion,
		"answer":   cli.runOpeningBookAnswer,
	}
}

// OpeningBookSubcommands returns the sub-commands of `blunderdb openingbook`,
// sorted — the exported view cmd/cli-doc-gen walks.
func (cli *CLI) OpeningBookSubcommands() []string {
	h := cli.openingBookHandlers()
	names := make([]string, 0, len(h))
	for name := range h {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (cli *CLI) printOpeningBookUsage() {
	fmt.Println("Usage: blunderdb openingbook <sub-command> [options]")
	fmt.Println()
	fmt.Println("The exercise « which play at this score? »: the opening play (1st roll) or")
	fmt.Println("the reply after an opening play tied with the best (2nd roll) at a match score,")
	fmt.Println("judged against the opening book embedded in blunderDB.")
	fmt.Println("Every play tied with the best is a right answer.")
	fmt.Println()
	fmt.Println("Sub-commands:")
	fmt.Println("  status    How many lines the book holds, and at which scores")
	fmt.Println("  question  Draw a position to play at a score (--roll first|second)")
	fmt.Println("  answer    Judge a play and show every candidate against the best")
}

// openingBookFlags are the flags every sub-command shares.
func openingBookFlags(fs *flag.FlagSet) (book, format *string) {
	book = fs.String("book", "", "Book file (.jsonl or .jsonl.gz) instead of the embedded one")
	format = fs.String("format", "text", "Output format: text, json")
	return
}

func loadOpeningBook(path string) (*openingbook.Book, error) {
	if path != "" {
		return openingbook.Open(path)
	}
	return openingbook.Default()
}

const openingBookEmpty = "The opening book is empty: it has not been computed yet, so there is nothing to ask."

func (cli *CLI) runOpeningBookStatus(args []string) error {
	fs := flag.NewFlagSet("openingbook status", flag.ContinueOnError)
	bookPath, format := openingBookFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	b, err := loadOpeningBook(*bookPath)
	if err != nil {
		return err
	}
	st := b.Status()
	if *format == "json" {
		if st.Scores == nil {
			st.Scores = []string{}
		}
		return printJSON(st)
	}
	if st.Lines == 0 {
		fmt.Println(openingBookEmpty)
		return nil
	}
	fmt.Printf("%d lines at %d scores: %s\n", st.Lines, len(st.Scores), strings.Join(st.Scores, ", "))
	return nil
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (cli *CLI) runOpeningBookQuestion(args []string) error {
	fs := flag.NewFlagSet("openingbook question", flag.ContinueOnError)
	bookPath, format := openingBookFlags(fs)
	roll := fs.String("roll", openingbook.RollFirst, "Which roll to ask: first (the opening play) or second (the reply after an opening play tied with the best)")
	scores := fs.String("score", "", "Scores to draw from, comma-separated (replier-opener away: 4-2, 1-2 Crawford, money)")
	openings := fs.String("opening", "", "Opening rolls to draw from, comma-separated (31,65)")
	replies := fs.String("reply", "", "Reply rolls to draw from, comma-separated (21,66); --roll second only")
	seed := fs.Uint64("seed", 0, "Seed for a reproducible draw (0 = random)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	b, err := loadOpeningBook(*bookPath)
	if err != nil {
		return err
	}
	q, err := b.Draw(openingbook.QuestionRequest{
		Roll: *roll, Scores: splitList(*scores), Openings: splitList(*openings), Replies: splitList(*replies), Seed: *seed,
	})
	if err != nil {
		return err
	}
	if *format == "json" {
		return printJSON(q)
	}
	if q.Empty {
		if b.Len() == 0 {
			fmt.Println(openingBookEmpty)
		} else {
			fmt.Println("No line of the opening book matches these options.")
		}
		return nil
	}
	at := fmt.Sprintf("At %s (the opener %s-away, the replier %s-away)", scoreLabel(q.Score), awayOf(q.Score, 1), awayOf(q.Score, 0))
	if q.Roll == openingbook.RollSecond {
		fmt.Printf("%s, after the opening %s played %s: reply %s.\n", at, q.Opening, q.OpeningPlay, q.Reply)
		fmt.Printf("XGID: %s\n", q.XGID)
		fmt.Printf("Answer: blunderdb openingbook answer --opening %s --reply %s --score %s --after '%s' '<reply>'\n", q.Opening, q.Reply, q.Score, q.OpeningPlay)
		return nil
	}
	fmt.Printf("%s: play the opening %s.\n", at, q.Opening)
	fmt.Printf("XGID: %s\n", q.XGID)
	fmt.Printf("Answer: blunderdb openingbook answer --opening %s --score %s '<opening play>'\n", q.Opening, q.Score)
	return nil
}

func scoreLabel(key string) string {
	if key == "money" {
		return "money play"
	}
	return "score " + key
}

func awayOf(key string, i int) string {
	parts := strings.SplitN(key, "-", 2)
	if len(parts) != 2 {
		return "∞"
	}
	return parts[i]
}

func (cli *CLI) runOpeningBookAnswer(args []string) error {
	fs := flag.NewFlagSet("openingbook answer", flag.ContinueOnError)
	bookPath, format := openingBookFlags(fs)
	opening := fs.String("opening", "", "Opening roll of the question (31)")
	reply := fs.String("reply", "", "Reply roll of a second-roll question (21)")
	score := fs.String("score", "", "Score of the question (4-2, 1-2, money)")
	after := fs.String("after", "", "The opening play the reply follows, as the second-roll question names it; empty judges the opening play")
	fs.Usage = func() {
		fmt.Println("Usage: blunderdb openingbook answer --opening <roll> --score <score> [--reply <roll> --after <opening play>] [options] <play>")
		fmt.Println()
		fmt.Println("Without --after, judges the opening play (1st roll); with --reply and --after,")
		fmt.Println("judges the reply played after that opening play (2nd roll).")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Example:")
		fmt.Println("  blunderdb openingbook answer --opening 31 --score 4-2 '8/5 6/5'")
		fmt.Println("  blunderdb openingbook answer --opening 31 --reply 21 --score 4-2 --after '8/5 6/5' '13/11 6/5'")
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("expected exactly one play, quoted")
	}
	b, err := loadOpeningBook(*bookPath)
	if err != nil {
		return err
	}
	v, err := b.Grade(openingbook.Answer{Opening: *opening, OpeningPlay: *after, Reply: *reply, Score: *score, Notation: fs.Arg(0)})
	if err != nil {
		return err
	}
	if *format == "json" {
		return printJSON(v)
	}
	switch {
	case !v.Legal:
		fmt.Printf("%s is not a legal play of the roll.\n", v.Notation)
	case v.Right:
		fmt.Printf("Right: %s is tied with the best.\n", v.Notation)
	case v.Played < 0:
		fmt.Printf("Wrong: %s is not among the book's candidates.\n", v.Notation)
	default:
		fmt.Printf("Wrong: %s is %.3f behind the best.\n", v.Notation, v.Ranked[v.Played].Gap)
	}
	fmt.Println()
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "\tPlay\tGap\t95 % CI\t")
	for i, r := range v.Ranked {
		mark := " "
		if i == v.Played {
			mark = "*"
		}
		tied := ""
		if r.Tied {
			tied = "tied"
		}
		ci := "—"
		if i > 0 {
			ci = fmt.Sprintf("±%.3f", r.CI95)
		}
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%.3f\t%s\t%s\n", mark, r.Move, r.Gap, ci, tied)
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	fmt.Println()
	fmt.Println("Gap: normalised equity at the score (±1 = the centred cube) behind the best play.")
	fmt.Printf("Rollout: %d games; %s\n", v.Games, v.Signature)
	return nil
}
