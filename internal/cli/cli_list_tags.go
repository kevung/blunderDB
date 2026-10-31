package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"text/tabwriter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// listTags prints the tag vocabulary of the database: every `#word` written
// in a comment, with the number of positions carrying it.
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
		enc := json.NewEncoder(cli.stdout())
		enc.SetIndent("", "  ")
		return enc.Encode(struct {
			Tags        []domain.TagCount `json:"tags"`
			Recommended []string          `json:"recommended"`
		}{Tags: tags, Recommended: domain.RecommendedTags})
	case "csv":
		w := csv.NewWriter(cli.stdout())
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
		fmt.Fprintln(cli.stdout(), "No tag used yet.")
		fmt.Fprintln(cli.stdout())
		fmt.Fprintln(cli.stdout(), "A tag is a #word inside a comment; nothing has to be declared.")
		fmt.Fprintln(cli.stdout(), "Suggested vocabulary:")
		for _, t := range domain.RecommendedTags {
			fmt.Fprintf(cli.stdout(), "  %s\n", t)
		}
		return nil
	}

	fmt.Fprintf(cli.stdout(), "Found %d tag(s):\n\n", len(tags))
	w := tabwriter.NewWriter(cli.stdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Tag\tPositions")
	fmt.Fprintln(w, "---\t---------")
	for _, t := range tags {
		fmt.Fprintf(w, "%s\t%d\n", t.Tag, t.Count)
	}
	w.Flush()
	fmt.Fprintln(cli.stdout())
	fmt.Fprintln(cli.stdout(), "Search a tag with: blunderdb search --db <path> --query '#prime'")
	return nil
}
