package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// runCollectionSuggest proposes reference positions (ADR-0080) and, on
// request, makes a collection or an Anki deck of them.
func (cli *CLI) runCollectionSuggest(args []string) error {
	fs, dbPath := collectionFlagSet("suggest",
		"Propose reference positions to study: among the player's errors, the positions whose lesson\n"+
			"covers the most recoverable match winning chances of their neighbouring errors (same family,\n"+
			fmt.Sprintf("within %d checker-pips; a cube reference holds at one score). A close lesson (second-best\n", storage.ReferenceRadius)+
			"option within half an error) or an unstable verdict (another depth or a rollout disagrees) counts\n"+
			"for half. Positions already commented, carded, collected or marked studied are never proposed,\n"+
			"and no two proposed positions are near-duplicates (ADR-0080). Nothing is written unless\n"+
			"--collection or --deck is given.",
		"blunderdb collection suggest --db database.db --player \"Alice\"",
		"blunderdb collection suggest --db database.db --player \"Alice\" --tournament 4 --size 10",
		"blunderdb collection suggest --db database.db --player \"Alice\" --match 12 --collection \"References: match 12\"",
		"blunderdb collection suggest --db database.db --player \"Alice\" --size 50 --deck \"References\" --format json")
	player := fs.String("player", "", "Only this player's errors")
	tournament := fs.String("tournament", "", "Only these tournaments, comma-separated IDs")
	match := fs.String("match", "", "Only these matches, comma-separated IDs")
	from := fs.String("from", "", "Start date filter YYYY-MM-DD")
	to := fs.String("to", "", "End date filter YYYY-MM-DD")
	decisionType := fs.String("decision-type", "all", "Decision type: all, checker, or cube")
	size := fs.Int("size", storage.ReferenceDefaultSize, fmt.Sprintf("Number of positions proposed (10, 20 or 50; at most %d)", storage.ReferenceMaxSize))
	collection := fs.String("collection", "", "Create a collection of this name holding the proposed positions")
	deck := fs.String("deck", "", "Create an Anki deck of this name from the proposed positions")
	format := fs.String("format", "text", "Output format: text or json")
	if err := cli.collectionOpen(fs, dbPath, args); err != nil {
		return err
	}
	if *size < 1 || *size > storage.ReferenceMaxSize {
		return fmt.Errorf("invalid --size %d: want 1 to %d", *size, storage.ReferenceMaxSize)
	}
	filter, err := buildStatsFilter(*player, *tournament, *from, *to, *decisionType)
	if err != nil {
		return err
	}
	var matchIDs []int64
	if strings.TrimSpace(*match) != "" {
		if matchIDs, err = parseIDList(*match); err != nil {
			return fmt.Errorf("invalid --match: %w", err)
		}
	}

	textOutput := strings.ToLower(*format) != "json"
	var res *storage.ReferenceSuggestions
	err = withInterruptibleContext(func() {
		if textOutput {
			fmt.Println("\nCancelling...")
		}
	}, func(ctx context.Context) error {
		var err error
		res, err = cli.db.SuggestReferencePositionsCtx(ctx, filter, matchIDs, *size)
		return err
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return fmt.Errorf("suggestion cancelled")
		}
		return fmt.Errorf("suggest references: %w", err)
	}

	ids := res.IDs()
	var collectionID, deckID int64
	if (*collection != "" || *deck != "") && len(ids) == 0 {
		return fmt.Errorf("no reference position to keep")
	}
	if name := strings.TrimSpace(*collection); name != "" {
		if collectionID, err = cli.db.CreateCollection(name, ""); err != nil {
			return fmt.Errorf("create collection: %w", err)
		}
		if err := cli.db.AddPositionsToCollection(collectionID, ids); err != nil {
			return fmt.Errorf("fill collection: %w", err)
		}
	}
	if name := strings.TrimSpace(*deck); name != "" {
		if deckID, err = cli.db.CreateStudyDeck(name, ids); err != nil {
			return err
		}
	}

	if !textOutput {
		out := struct {
			*storage.ReferenceSuggestions
			CollectionID int64 `json:"CollectionID,omitempty"`
			DeckID       int64 `json:"DeckID,omitempty"`
		}{res, collectionID, deckID}
		data, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal references: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}
	printReferences(res)
	if collectionID != 0 {
		fmt.Printf("Created collection %d (%s) with %d positions\n", collectionID, strings.TrimSpace(*collection), len(ids))
	}
	if deckID != 0 {
		fmt.Printf("Created deck %d (%s)\n", deckID, strings.TrimSpace(*deck))
	}
	return nil
}

// printReferences writes one line per proposed position, then its reason.
// MWC figures are percentage points of match winning chances.
func printReferences(res *storage.ReferenceSuggestions) {
	fmt.Printf("Reference positions — %d counted decisions, error threshold %d mp, neighbours within %d checker-pips\n",
		res.NumDecisions, res.ThresholdMP, res.Radius)
	fmt.Printf("%d candidate positions weighed, %d already handled.\n\n", res.Candidates, res.Handled)
	if len(res.References) == 0 {
		fmt.Println("No position to propose: no priced, themed error left unhandled.")
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "#\tPOSITION\tPLAN\tKIND\tTHEME\tSCORE\tCOVERS\tMWC COVERED\tREASON")
	for i, r := range res.References {
		fmt.Fprintf(w, "%d\t%d\t%s\t%s\t%s\t%s\t%d errors / %d matches\t%.2f%%\t%s\n", i+1, r.PositionID,
			r.GameType, r.Kind, r.Theme, referenceScore(r), r.Errors, r.Matches, 100*r.Covered, referenceReason(r, res.NumDecisions))
	}
	w.Flush()
}

func referenceScore(r storage.ReferenceSuggestion) string {
	if r.Kind != "cube" {
		return "-"
	}
	return fmt.Sprintf("%da-%da", r.AwayOnRoll, r.AwayOpponent)
}

// referenceReason words a proposal's reason in English, from its components.
func referenceReason(r storage.ReferenceSuggestion, numDecisions int) string {
	parts := []string{fmt.Sprintf("stands for %d of your errors in %d matches", r.Errors, r.Matches)}
	if numDecisions > 0 {
		parts = append(parts, fmt.Sprintf("family: %d errors in %d decisions", r.FamilyErrors, numDecisions))
	}
	parts = append(parts, fmt.Sprintf("own recoverable %.2f%%", 100*r.Excess))
	if r.GapMP >= 0 {
		parts = append(parts, fmt.Sprintf("second best %d mp behind", r.GapMP))
	}
	if r.Close {
		parts = append(parts, "close lesson (counts half)")
	}
	if r.Unstable {
		parts = append(parts, "unstable verdict (counts half)")
	}
	if r.Gain < r.Covered {
		parts = append(parts, fmt.Sprintf("counted %.2f%%", 100*r.Gain))
	}
	if r.RolledOut {
		parts = append(parts, "confirmed by a rollout")
	}
	return strings.Join(parts, "; ")
}
