package cli

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// runAnki handles the anki command: the read-only and maintenance side of the
// spaced-repetition decks the GUI's Anki panel reviews and the daemon serves
// under /v1/anki.*. A script can list the decks with their due/new counters,
// read a deck's statistics, project the cards coming due, and resynchronise a
// deck with its source. Reviewing a card is a GUI gesture (it needs the
// board) and stays out of the CLI on purpose.
func (cli *CLI) runAnki(args []string) error {
	if len(args) < 1 {
		cli.printAnkiUsage()
		return fmt.Errorf("missing anki sub-command")
	}
	sub := strings.ToLower(args[0])
	if sub == "--help" || sub == "-h" || sub == "help" {
		cli.printAnkiUsage()
		return nil
	}
	run, ok := cli.ankiHandlers()[sub]
	if !ok {
		cli.printAnkiUsage()
		return fmt.Errorf("unknown anki sub-command: %s", args[0])
	}
	return run(args[1:])
}

// ankiHandlers returns the sub-command table of `blunderdb anki`. It is a
// map, like handlers(), so the parity test can walk it.
func (cli *CLI) ankiHandlers() map[string]func([]string) error {
	return map[string]func([]string) error{
		"decks":     cli.runAnkiDecks,
		"stats":     cli.runAnkiStats,
		"forecast":  cli.runAnkiForecast,
		"sync":      cli.runAnkiSync,
		"retention": cli.runAnkiRetention,
		"card":      cli.runAnkiCard,
		"log":       cli.runAnkiLog,
	}
}

// AnkiSubcommands returns the sub-commands of `blunderdb anki`, sorted — the
// exported view cmd/cli-doc-gen walks.
func (cli *CLI) AnkiSubcommands() []string {
	h := cli.ankiHandlers()
	names := make([]string, 0, len(h))
	for name := range h {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (cli *CLI) printAnkiUsage() {
	fmt.Println("Usage: blunderdb anki <sub-command> [options]")
	fmt.Println()
	fmt.Println("Inspect and maintain spaced-repetition (FSRS) decks. Reviewing cards")
	fmt.Println("is done in the GUI's Anki panel.")
	fmt.Println()
	fmt.Println("Sub-commands:")
	fmt.Println("  decks     List decks with their card, due and new counters")
	fmt.Println("  stats     Review statistics of one deck (new, learning, review, due)")
	fmt.Println("  forecast  Cards coming due per day over the next N days")
	fmt.Println("  sync      Resynchronise a deck with its collection or stored search")
	fmt.Println("  retention Measured retention of one deck against its target")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  blunderdb anki decks --db database.db")
	fmt.Println("  blunderdb anki stats --db database.db --deck 2 --format json")
	fmt.Println("  blunderdb anki forecast --db database.db --deck 2 --days 14")
	fmt.Println("  blunderdb anki sync --db database.db --deck 2")
	fmt.Println("  blunderdb anki retention --db database.db --deck 2")
	fmt.Println()
	fmt.Println("Use 'blunderdb anki <sub-command> --help' for the options of a sub-command.")
}

// ankiFlagSet builds the FlagSet shared by every sub-command.
func ankiFlagSet(sub, summary string, examples ...string) (*flag.FlagSet, *string) {
	fs := flag.NewFlagSet("anki "+sub, flag.ContinueOnError)
	dbPath := fs.String("db", "", "Path to the database file (required)")
	fs.Usage = func() {
		fmt.Printf("Usage: blunderdb anki %s [options]\n\n%s\n\nOptions:\n", sub, summary)
		fs.PrintDefaults()
		if len(examples) > 0 {
			fmt.Println()
			fmt.Println("Examples:")
			for _, ex := range examples {
				fmt.Println("  " + ex)
			}
		}
	}
	return fs, dbPath
}

// ankiDeck finds one deck by id, the way the GUI keeps the deck it displays:
// GetAllAnkiDecks is the only Database call that returns a deck with its
// counters, and a deck list is small.
func (cli *CLI) ankiDeck(id int64) (*AnkiDeck, error) {
	decks, err := cli.db.GetAllAnkiDecks()
	if err != nil {
		return nil, fmt.Errorf("failed to get decks: %w", err)
	}
	for i := range decks {
		if decks[i].ID == id {
			return &decks[i], nil
		}
	}
	return nil, fmt.Errorf("deck with ID %d not found", id)
}

// ── decks ────────────────────────────────────────────────────────────────────

func (cli *CLI) runAnkiDecks(args []string) error {
	fs, dbPath := ankiFlagSet("decks", "List the decks of the database with their counters.",
		"blunderdb anki decks --db database.db",
		"blunderdb anki decks --db database.db --format csv")
	format := fs.String("format", "text", "Output format: text, json or csv")
	if err := cli.collectionOpen(fs, dbPath, args); err != nil {
		return err
	}
	decks, err := cli.db.GetAllAnkiDecks()
	if err != nil {
		return fmt.Errorf("failed to get decks: %w", err)
	}

	switch strings.ToLower(*format) {
	case "json":
		if decks == nil {
			decks = []AnkiDeck{}
		}
		return printJSON(decks)
	case "csv":
		w := csv.NewWriter(os.Stdout)
		defer w.Flush()
		if err := w.Write([]string{"id", "name", "source", "cards", "due", "new", "request_retention"}); err != nil {
			return fmt.Errorf("write csv header: %w", err)
		}
		for _, d := range decks {
			if err := w.Write([]string{
				strconv.FormatInt(d.ID, 10), d.Name, ankiSourceLabel(d),
				strconv.Itoa(d.CardCount), strconv.Itoa(d.DueCount), strconv.Itoa(d.NewCount),
				strconv.FormatFloat(d.RequestRetention, 'f', 2, 64),
			}); err != nil {
				return fmt.Errorf("write csv row: %w", err)
			}
		}
		return w.Error()
	case "text":
		if len(decks) == 0 {
			fmt.Println("No decks found in database")
			return nil
		}
		fmt.Printf("Found %d deck(s):\n\n", len(decks))
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tName\tSource\tCards\tDue\tNew")
		fmt.Fprintln(w, "--\t----\t------\t-----\t---\t---")
		for _, d := range decks {
			fmt.Fprintf(w, "%d\t%s\t%s\t%d\t%d\t%d\n", d.ID, d.Name, ankiSourceLabel(d), d.CardCount, d.DueCount, d.NewCount)
		}
		return w.Flush()
	default:
		return fmt.Errorf("unknown format: %s (must be 'text', 'json', or 'csv')", *format)
	}
}

// ankiSourceLabel names where a deck draws its cards from.
func ankiSourceLabel(d AnkiDeck) string {
	switch d.SourceType {
	case AnkiSourceCollection:
		return fmt.Sprintf("collection %d", d.SourceID)
	case AnkiSourceSearch:
		return "search"
	default:
		return d.SourceType
	}
}

// ── stats ────────────────────────────────────────────────────────────────────

func (cli *CLI) runAnkiStats(args []string) error {
	fs, dbPath := ankiFlagSet("stats", "Review statistics of one deck.",
		"blunderdb anki stats --db database.db --deck 2",
		"blunderdb anki stats --db database.db --deck 2 --format json")
	deckID := fs.Int64("deck", 0, "Deck ID (required)")
	format := fs.String("format", "text", "Output format: text or json")
	if err := cli.collectionOpen(fs, dbPath, args); err != nil {
		return err
	}
	if *deckID == 0 {
		fs.Usage()
		return fmt.Errorf("missing required flag: --deck")
	}
	deck, err := cli.ankiDeck(*deckID)
	if err != nil {
		return err
	}
	stats, err := cli.db.GetAnkiDeckStats(*deckID)
	if err != nil {
		return fmt.Errorf("failed to get deck stats: %w", err)
	}

	switch strings.ToLower(*format) {
	case "json":
		return printJSON(struct {
			Deck  *AnkiDeck     `json:"deck"`
			Stats AnkiDeckStats `json:"stats"`
		}{deck, stats})
	case "text":
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "Deck %d:\t%s\n", deck.ID, deck.Name)
		fmt.Fprintf(w, "  Source:\t%s\n", ankiSourceLabel(*deck))
		fmt.Fprintf(w, "  Request retention:\t%.2f\n", deck.RequestRetention)
		fmt.Fprintf(w, "  Maximum interval:\t%.0f days\n", deck.MaximumInterval)
		fmt.Fprintln(w)
		fmt.Fprintf(w, "  Total cards:\t%d\n", stats.TotalCount)
		fmt.Fprintf(w, "  New:\t%d\n", stats.NewCount)
		fmt.Fprintf(w, "  Learning:\t%d\n", stats.LearningCount)
		fmt.Fprintf(w, "  Review due:\t%d\n", stats.ReviewCount)
		fmt.Fprintf(w, "  Due now:\t%d\n", stats.DueCount)
		return w.Flush()
	default:
		return fmt.Errorf("unknown format: %s (must be 'text' or 'json')", *format)
	}
}

// ── forecast ─────────────────────────────────────────────────────────────────

func (cli *CLI) runAnkiForecast(args []string) error {
	fs, dbPath := ankiFlagSet("forecast", "Cards coming due per calendar day (UTC). Day 0 holds every overdue card.",
		"blunderdb anki forecast --db database.db --deck 2 --days 14",
		"blunderdb anki forecast --db database.db --days 30 --format csv   # every deck")
	deckID := fs.Int64("deck", 0, "Deck ID (0 = every deck)")
	days := fs.Int("days", 30, "Number of days to project (1-365)")
	format := fs.String("format", "text", "Output format: text, json or csv")
	if err := cli.collectionOpen(fs, dbPath, args); err != nil {
		return err
	}
	if *deckID != 0 {
		if _, err := cli.ankiDeck(*deckID); err != nil {
			return err
		}
	}
	forecast, err := cli.db.GetAnkiForecast(*deckID, *days)
	if err != nil {
		return fmt.Errorf("failed to compute forecast: %w", err)
	}

	switch strings.ToLower(*format) {
	case "json":
		if forecast == nil {
			forecast = []AnkiForecastDay{}
		}
		return printJSON(forecast)
	case "csv":
		w := csv.NewWriter(os.Stdout)
		defer w.Flush()
		if err := w.Write([]string{"day", "due"}); err != nil {
			return fmt.Errorf("write csv header: %w", err)
		}
		for _, d := range forecast {
			if err := w.Write([]string{d.Day, strconv.Itoa(d.Due)}); err != nil {
				return fmt.Errorf("write csv row: %w", err)
			}
		}
		return w.Error()
	case "text":
		total := 0
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "Day\tDue")
		fmt.Fprintln(w, "---\t---")
		for _, d := range forecast {
			total += d.Due
			fmt.Fprintf(w, "%s\t%d\n", d.Day, d.Due)
		}
		if err := w.Flush(); err != nil {
			return err
		}
		fmt.Printf("\n%d card(s) due over %d day(s)\n", total, len(forecast))
		return nil
	default:
		return fmt.Errorf("unknown format: %s (must be 'text', 'json', or 'csv')", *format)
	}
}

// ── sync ─────────────────────────────────────────────────────────────────────

// runAnkiSync adds a card for every position of the deck's source that has
// none yet; existing cards keep their scheduling state. A collection deck
// re-reads its collection (Database.SyncAnkiDeck, exactly the GUI's call). A
// search deck stores the search as JSON — {"command","position","ids"} — and
// the GUI re-runs the command in its own command processor before calling
// SyncAnkiDeckWithPositions with the merged ids; that grammar lives in the
// frontend, so the CLI resynchronises from the stored ids and says so.
func (cli *CLI) runAnkiSync(args []string) error {
	fs, dbPath := ankiFlagSet("sync", "Resynchronise a deck with its source (collection or stored search).",
		"blunderdb anki sync --db database.db --deck 2")
	deckID := fs.Int64("deck", 0, "Deck ID (required)")
	if err := cli.collectionOpen(fs, dbPath, args); err != nil {
		return err
	}
	if *deckID == 0 {
		fs.Usage()
		return fmt.Errorf("missing required flag: --deck")
	}
	deck, err := cli.ankiDeck(*deckID)
	if err != nil {
		return err
	}

	if deck.SourceType == AnkiSourceSearch {
		if ids, ok := ankiSearchStoredIDs(deck.SourceCommand); ok {
			if err := cli.db.SyncAnkiDeckWithPositions(deck.ID, ids); err != nil {
				return fmt.Errorf("failed to sync deck: %w", err)
			}
			fmt.Fprintln(os.Stderr, "Note: a search deck is resynchronised from the position ids stored with it; the GUI re-runs the search itself.")
		} else if err := cli.db.SyncAnkiDeck(deck.ID); err != nil {
			return fmt.Errorf("failed to sync deck: %w", err)
		}
	} else if err := cli.db.SyncAnkiDeck(deck.ID); err != nil {
		return fmt.Errorf("failed to sync deck: %w", err)
	}

	after, err := cli.ankiDeck(deck.ID)
	if err != nil {
		return err
	}
	fmt.Printf("Synced deck %d (%s): %d card(s), %d added\n", after.ID, after.Name, after.CardCount, after.CardCount-deck.CardCount)
	return nil
}

// ankiSearchStoredIDs reads the position ids of a JSON search source. ok is
// false for the legacy comma-separated form, which SyncAnkiDeck parses itself.
func ankiSearchStoredIDs(sourceCommand string) ([]int64, bool) {
	var stored struct {
		IDs []int64 `json:"ids"`
	}
	if err := json.Unmarshal([]byte(sourceCommand), &stored); err != nil {
		return nil, false
	}
	return stored.IDs, true
}

// runAnkiRetention reports the retention a deck's review log measures, read
// against the target its owner chose.
//
// It reports and never writes (ADR-0026 rule 5). Under
// domain.AnkiRetentionMinSample review-state reviews the measurement is named
// as unavailable rather than printed: a pass rate over three reviews reads as
// fact while being noise.
func (cli *CLI) runAnkiRetention(args []string) error {
	fs, dbPath := ankiFlagSet("retention", "Measured retention of one deck against its target.",
		"blunderdb anki retention --db database.db --deck 2",
		"blunderdb anki retention --db database.db --deck 2 --format json")
	deckID := fs.Int64("deck", 0, "Deck ID (required)")
	format := fs.String("format", "text", "Output format: text or json")
	if err := cli.collectionOpen(fs, dbPath, args); err != nil {
		return err
	}
	if *deckID == 0 {
		fs.Usage()
		return fmt.Errorf("missing required flag: --deck")
	}
	deck, err := cli.ankiDeck(*deckID)
	if err != nil {
		return err
	}
	ret, err := cli.db.GetAnkiDeckRetention(*deckID)
	if err != nil {
		return fmt.Errorf("failed to measure deck retention: %w", err)
	}

	switch strings.ToLower(*format) {
	case "json":
		return printJSON(struct {
			Deck      *AnkiDeck             `json:"deck"`
			Retention *domain.AnkiRetention `json:"retention"`
		}{deck, ret})
	case "text":
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "Deck %d:\t%s\n", deck.ID, deck.Name)
		fmt.Fprintf(w, "  Target retention:\t%.0f%%\n", ret.TargetRetention*100)
		if ret.SampleSize < domain.AnkiRetentionMinSample {
			fmt.Fprintf(w, "  Measured retention:\tnot enough reviews (%d of %d needed)\n",
				ret.SampleSize, domain.AnkiRetentionMinSample)
		} else {
			fmt.Fprintf(w, "  Measured retention:\t%.0f%% over %d reviews\n",
				ret.ObservedRetention*100, ret.SampleSize)
		}
		return w.Flush()
	default:
		return fmt.Errorf("unknown format: %s (use text or json)", *format)
	}
}

// runAnkiCard is the `anki card` sub-command: the three gestures on a single
// card — suspend, bury, remove — which the daemon had served since it existed
// while the CLI and the GUI had no way to reach them (G.14, #242).
//
// One sub-command with an action rather than three sibling sub-commands: they
// share their argument (a card id), their confirmation and their output, and a
// caller thinks "do something to this card", not "run the bury program".
func (cli *CLI) runAnkiCard(args []string) error {
	fs, dbPath := ankiFlagSet("card", "Suspend, bury or remove one card.",
		"blunderdb anki card --db database.db --id 12 --action suspend",
		"blunderdb anki card --db database.db --id 12 --action unsuspend",
		"blunderdb anki card --db database.db --id 12 --action bury",
		"blunderdb anki card --db database.db --id 12 --action remove")
	cardID := fs.Int64("id", 0, "Card ID (required)")
	action := fs.String("action", "", "suspend, unsuspend, bury or remove (required)")
	format := fs.String("format", "text", "Output format: text or json")
	if err := cli.collectionOpen(fs, dbPath, args); err != nil {
		return err
	}
	if *cardID == 0 {
		fs.Usage()
		return fmt.Errorf("missing required flag: --id")
	}

	var err error
	switch strings.ToLower(*action) {
	case "suspend":
		err = cli.db.SetAnkiCardSuspended(*cardID, true)
	case "unsuspend":
		err = cli.db.SetAnkiCardSuspended(*cardID, false)
	case "bury":
		err = cli.db.BuryAnkiCard(*cardID)
	case "remove":
		err = cli.db.RemoveAnkiCard(*cardID)
	case "":
		fs.Usage()
		return fmt.Errorf("missing required flag: --action")
	default:
		return fmt.Errorf("unknown action: %s (use suspend, unsuspend, bury or remove)", *action)
	}
	if err != nil {
		return fmt.Errorf("failed to %s card %d: %w", strings.ToLower(*action), *cardID, err)
	}

	switch strings.ToLower(*format) {
	case "json":
		return printJSON(struct {
			CardID int64  `json:"cardId"`
			Action string `json:"action"`
			OK     bool   `json:"ok"`
		}{*cardID, strings.ToLower(*action), true})
	case "text":
		fmt.Printf("Card %d: %sd\n", *cardID, strings.ToLower(*action))
		return nil
	default:
		return fmt.Errorf("unknown format: %s (use text or json)", *format)
	}
}

// runAnkiLog is the `anki log` sub-command: the recorded review events, most
// recent first. The log is what the scheduler was told, as opposed to what it
// currently plans — the only place a grade entered by mistake is visible at
// all, ADR-0026 keeping the schedule itself out of reach.
func (cli *CLI) runAnkiLog(args []string) error {
	fs, dbPath := ankiFlagSet("log", "Recorded review events, most recent first.",
		"blunderdb anki log --db database.db",
		"blunderdb anki log --db database.db --deck 2 --limit 50",
		"blunderdb anki log --db database.db --format json")
	deckID := fs.Int64("deck", 0, "Deck ID (0 = every deck)")
	limit := fs.Int("limit", 20, "Maximum number of events")
	format := fs.String("format", "text", "Output format: text or json")
	if err := cli.collectionOpen(fs, dbPath, args); err != nil {
		return err
	}
	entries, err := cli.db.GetAnkiReviewLog(*deckID, *limit)
	if err != nil {
		return fmt.Errorf("failed to read the review log: %w", err)
	}

	switch strings.ToLower(*format) {
	case "json":
		return printJSON(entries)
	case "text":
		if len(entries) == 0 {
			fmt.Println("No review recorded.")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "REVIEWED AT\tDECK\tCARD\tPOSITION\tGRADE\tINTERVAL")
		for _, e := range entries {
			fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%s\t%dd\n",
				e.ReviewedAt, e.DeckID, e.CardID, e.PositionID, ankiGradeLabel(e.Rating), e.ScheduledDays)
		}
		return w.Flush()
	default:
		return fmt.Errorf("unknown format: %s (use text or json)", *format)
	}
}

// ankiGradeLabel names an FSRS rating the way the review panel does.
func ankiGradeLabel(rating int) string {
	switch rating {
	case 1:
		return "Again"
	case 2:
		return "Hard"
	case 3:
		return "Good"
	case 4:
		return "Easy"
	default:
		return fmt.Sprintf("?%d", rating)
	}
}
