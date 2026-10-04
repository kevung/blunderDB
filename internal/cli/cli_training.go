package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// runTraining dispatches `blunderdb training <sub-command>`. The journal is
// written by whoever ran the exercise (the GUI, or a client over
// /v1/training.save); the CLI reads it back and turns the missed questions
// into study material.
func (cli *CLI) runTraining(args []string) error {
	if len(args) < 1 {
		cli.printTrainingUsage()
		return fmt.Errorf("missing training sub-command")
	}
	sub := strings.ToLower(args[0])
	if sub == "--help" || sub == "-h" || sub == "help" {
		cli.printTrainingUsage()
		return nil
	}
	run, ok := cli.trainingHandlers()[sub]
	if !ok {
		cli.printTrainingUsage()
		return fmt.Errorf("unknown training sub-command: %s", args[0])
	}
	return run(args[1:])
}

// trainingHandlers returns the sub-command table of `blunderdb training`.
func (cli *CLI) trainingHandlers() map[string]func([]string) error {
	return map[string]func([]string) error{
		"sessions": cli.runTrainingSessions,
		"missed":   cli.runTrainingMissed,
	}
}

// TrainingSubcommands returns the sub-commands of `blunderdb training`,
// sorted — the exported view cmd/cli-doc-gen walks.
func (cli *CLI) TrainingSubcommands() []string {
	h := cli.trainingHandlers()
	names := make([]string, 0, len(h))
	for name := range h {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (cli *CLI) printTrainingUsage() {
	fmt.Println("Usage: blunderdb training <sub-command> [options]")
	fmt.Println()
	fmt.Println("The Training journal: the sessions recorded by the Training tab.")
	fmt.Println()
	fmt.Println("Sub-commands:")
	fmt.Println("  sessions  The recorded sessions, most recent first")
	fmt.Println("  missed    The positions answered wrong, as a list, a deck or a collection")
}

func (cli *CLI) runTrainingSessions(args []string) error {
	fs := flag.NewFlagSet("training sessions", flag.ContinueOnError)
	dbPath := fs.String("db", "", "Path to the database file (required)")
	exercise := fs.String("exercise", "", "Only this exercise (scores, pips, bearoff, evaluation, decision)")
	limit := fs.Int("limit", 20, "Number of sessions shown (0 = all)")
	format := fs.String("format", "text", "Output format: text or json")
	fs.Usage = func() {
		fmt.Println("Usage: blunderdb training sessions --db <file> [options]")
		fmt.Println()
		fmt.Println("The sessions of the Training journal, most recent first, with their id")
		fmt.Println("(for `training missed --session`). PR is the Decision exercise's only.")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  blunderdb training sessions --db database.db")
		fmt.Println("  blunderdb training sessions --db database.db --exercise decision --format json")
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dbPath == "" {
		fs.Usage()
		return fmt.Errorf("missing required flag: --db")
	}
	if *limit < 0 {
		return fmt.Errorf("invalid --limit %d: want 0 (all) or more", *limit)
	}
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}
	sessions, err := cli.db.LoadTrainingSessions(*exercise, *limit)
	if err != nil {
		return fmt.Errorf("training sessions: %w", err)
	}
	if strings.ToLower(*format) == "json" {
		if sessions == nil {
			sessions = []storage.TrainingSession{}
		}
		data, err := json.MarshalIndent(sessions, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal training sessions: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}
	if len(sessions) == 0 {
		fmt.Println("No training session recorded.")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDATE\tEXERCISE\tASKED\tFAULTS\tMEDIAN\tPR")
	for _, s := range sessions {
		pr := "-"
		if s.Exercise == "decision" {
			pr = fmt.Sprintf("%.2f", s.PR)
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%d\t%d\t%.1fs\t%s\n", s.ID, s.CreatedAt, s.Exercise, s.NumbersAsked, s.Faults,
			float64(s.MedianMs)/1000, pr)
	}
	return w.Flush()
}

func (cli *CLI) runTrainingMissed(args []string) error {
	fs := flag.NewFlagSet("training missed", flag.ContinueOnError)
	dbPath := fs.String("db", "", "Path to the database file (required)")
	exercise := fs.String("exercise", "", "Only this exercise's sessions (decision, evaluation)")
	session := fs.Int64("session", 0, "Only this session (an id from `training sessions`)")
	limit := fs.Int("limit", 0, "Number of positions (0 = all)")
	deckName := fs.String("deck", "", "Make an Anki deck of these positions, with this name")
	collectionName := fs.String("collection", "", "Make a collection of these positions, with this name")
	format := fs.String("format", "text", "Output format: text or json")
	fs.Usage = func() {
		fmt.Println("Usage: blunderdb training missed --db <file> [options]")
		fmt.Println()
		fmt.Println("The positions answered wrong in the Training journal, each once, the most")
		fmt.Println("recently missed first. A question that ran out of time counts as missed.")
		fmt.Println("--deck and --collection turn them into study material.")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  blunderdb training missed --db database.db")
		fmt.Println("  blunderdb training missed --db database.db --session 12 --deck \"Missed on Monday\"")
		fmt.Println("  blunderdb training missed --db database.db --collection \"My misses\" --format json")
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dbPath == "" {
		fs.Usage()
		return fmt.Errorf("missing required flag: --db")
	}
	if *limit < 0 || *session < 0 {
		return fmt.Errorf("invalid --limit or --session: want 0 (no bound) or more")
	}
	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}
	ids, err := cli.db.LoadTrainingMissed(storage.TrainingMissedFilter{Exercise: *exercise, SessionID: *session, Limit: *limit})
	if err != nil {
		return fmt.Errorf("training missed: %w", err)
	}
	if (*deckName != "" || *collectionName != "") && len(ids) == 0 {
		return fmt.Errorf("no missed position to make a deck or a collection from")
	}
	var deckID, collectionID int64
	if *deckName != "" {
		if deckID, err = cli.db.CreateStudyDeck(*deckName, ids); err != nil {
			return err
		}
	}
	if *collectionName != "" {
		if collectionID, err = cli.db.CreateCollection(*collectionName, ""); err != nil {
			return err
		}
		if err := cli.db.AddPositionsToCollection(collectionID, ids); err != nil {
			return err
		}
	}
	if strings.ToLower(*format) == "json" {
		if ids == nil {
			ids = []int64{}
		}
		out := struct {
			PositionIDs  []int64 `json:"PositionIDs"`
			DeckID       int64   `json:"DeckID,omitempty"`
			CollectionID int64   `json:"CollectionID,omitempty"`
		}{ids, deckID, collectionID}
		data, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal missed positions: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}
	if len(ids) == 0 {
		fmt.Println("No missed position.")
		return nil
	}
	fmt.Printf("Missed — %d positions: %s\n", len(ids), joinIDs(ids))
	if deckID != 0 {
		fmt.Printf("Created deck %d (%s)\n", deckID, *deckName)
	}
	if collectionID != 0 {
		fmt.Printf("Created collection %d (%s)\n", collectionID, *collectionName)
	}
	return nil
}
