package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// listTags prints the tag vocabulary of the database: every `#word` written
// in a comment, with the number of positions carrying it (issue #265, fiche
// I.9).
//
// The list a user has actually built, and — when it is empty or short — the
// list blunderDB suggests. A fresh library has no tags of its own, and a
// vocabulary panel that stays blank until somebody guesses the convention
// teaches nothing.
func (cli *CLI) listTags(format string) error {
	tags, err := cli.db.Tags()
	if err != nil {
		return fmt.Errorf("failed to read the tags: %w", err)
	}

	switch format {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(struct {
			Tags        []domain.TagCount `json:"tags"`
			Recommended []string          `json:"recommended"`
		}{Tags: tags, Recommended: domain.RecommendedTags})
	case "csv":
		w := csv.NewWriter(os.Stdout)
		defer w.Flush()
		if err := w.Write([]string{"tag", "positions"}); err != nil {
			return err
		}
		for _, t := range tags {
			if err := w.Write([]string{t.Tag, strconv.Itoa(t.Count)}); err != nil {
				return err
			}
		}
		return nil
	}

	if len(tags) == 0 {
		fmt.Println("No tag used yet.")
		fmt.Println()
		fmt.Println("A tag is a #word inside a comment; nothing has to be declared.")
		fmt.Println("Suggested vocabulary:")
		for _, t := range domain.RecommendedTags {
			fmt.Printf("  %s\n", t)
		}
		return nil
	}

	fmt.Printf("Found %d tag(s):\n\n", len(tags))
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Tag\tPositions")
	fmt.Fprintln(w, "---\t---------")
	for _, t := range tags {
		fmt.Fprintf(w, "%s\t%d\n", t.Tag, t.Count)
	}
	w.Flush()
	fmt.Println()
	fmt.Println("Search a tag with: blunderdb search --db <path> --query '#prime'")
	return nil
}
