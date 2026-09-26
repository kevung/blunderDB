package domain

import (
	"regexp"
	"sort"
	"strings"
)

// Tags. A tag is not a table: it is a `#word` inside a comment, undeclared.
// So a tag is found by reading comment text, never by a GROUP BY; the per-tag
// breakdown reads the selected positions' tags once and tallies in Go.

// tagPattern is a '#' followed by at least one character that is not
// whitespace and not another '#'. Deliberately permissive: a tag may carry
// digits, accents, hyphens — "#back-game", "#2-away", "#préparation" — since
// the vocabulary is the user's.
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

// TagCount is one entry of a database's tag vocabulary: the tag, and how many
// POSITIONS carry it — the number clicking the tag will yield, so a tag
// written twice on one position counts once.
type TagCount struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

// RecommendedTags is the vocabulary blunderDB SUGGESTS while a comment is
// being typed; a tag absent from it is as valid as one on it. The names are
// the backgammon literature's (research report P5), a spelling convention
// rather than a taxonomy, and the list is kept short on purpose.
var RecommendedTags = []string{
	"#ace-point",
	"#backgame",
	"#blitz",
	"#containment",
	"#crunch",
	"#cube",
	"#holding",
	"#prime",
	"#priming",
	"#race",
	"#timing",
}

// MatchesAllTags reports whether text carries every tag in want.
//
// Delimited, not substring: `#prime` does not match `#priming`, which the
// free-text search (`t"#prime"`) cannot tell apart. An empty want matches
// everything.
func MatchesAllTags(text string, want []string) bool {
	if len(want) == 0 {
		return true
	}
	have := map[string]bool{}
	for _, t := range ExtractTags(text) {
		have[t] = true
	}
	for _, t := range want {
		if !have[NormalizeTag(t)] {
			return false
		}
	}
	return true
}

// NormalizeTag puts a tag in the shape ExtractTags produces: lower-cased, with
// a leading '#'. It is what a filter's tags go through before comparison, so
// "Prime", "#Prime" and "#prime" are one tag.
func NormalizeTag(tag string) string {
	tag = strings.ToLower(strings.TrimSpace(tag))
	if tag == "" {
		return ""
	}
	if !strings.HasPrefix(tag, "#") {
		tag = "#" + tag
	}
	return tag
}

// ParseTagFilter splits a SearchFilters.TagFilter into its normalised tags.
// The separator is ';', as everywhere else in the grammar, and every tag must
// be present for a position to match (see the field's own documentation).
func ParseTagFilter(filter string) []string {
	var out []string
	for _, part := range strings.Split(filter, ";") {
		if t := NormalizeTag(part); t != "" && t != "#" {
			out = append(out, t)
		}
	}
	return out
}
