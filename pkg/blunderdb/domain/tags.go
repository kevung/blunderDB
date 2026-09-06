package domain

import (
	"regexp"
	"sort"
	"strings"
)

// Tags (issue #265, #266).
//
// A tag is not a table. It is a `#word` inside a comment, which is how the
// `#blitz #prime` command has always written one and how the search has always
// found one — the tag vocabulary lives in the user's own prose, and nothing
// forces them to declare a tag before using it.
//
// That has a consequence the statistics have to live with: a tag can only be
// found by reading comment text, never by a GROUP BY. The per-tag breakdown
// therefore reads the tags of the selected positions once and tallies in Go.

// tagPattern is a '#' followed by at least one character that is not
// whitespace and not another '#'. Deliberately permissive: a tag may carry
// digits, accents, hyphens — "#back-game", "#2-away", "#préparation" — because
// the vocabulary is the user's and refusing their spelling would be inventing
// a rule they never agreed to.
var tagPattern = regexp.MustCompile(`#[^\s#]+`)

// ExtractTags returns the tags of a comment, lower-cased, deduplicated, in
// alphabetical order. The leading '#' is kept: it is what makes a tag legible
// as a tag wherever it is shown, and what the search token already carries.
//
// Trailing punctuation is trimmed — "#blitz." and "#blitz," are the same tag
// written at the end of a sentence — but not a hyphen or an underscore, which
// occur inside real tags.
func ExtractTags(text string) []string {
	if text == "" {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, m := range tagPattern.FindAllString(text, -1) {
		tag := strings.ToLower(strings.TrimRight(m, ".,;:!?)]}\"'"))
		if tag == "#" || seen[tag] {
			continue
		}
		seen[tag] = true
		out = append(out, tag)
	}
	sort.Strings(out)
	return out
}
