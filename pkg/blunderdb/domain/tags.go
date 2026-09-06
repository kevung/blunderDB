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

// TagCount is one entry of a database's tag vocabulary: the tag, and how many
// POSITIONS carry it (issue #265). Positions, not comments: a tag written
// twice on the same position is one position tagged, and the number a panel
// shows next to a tag has to be the number of positions clicking it will
// yield.
type TagCount struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

// RecommendedTags is the vocabulary blunderDB SUGGESTS while a comment is
// being typed (issue #265). Suggests, and nothing more: a tag is a `#word`
// in the user's own prose, nothing has to be declared, and a tag absent from
// this list is as valid as one on it.
//
// The names come from the backgammon literature — they are the terms the
// research report P5 collected for the game-type classifier, and the ones a
// player reads in a book — rather than from anything blunderDB computes. A
// vocabulary invented here would be a taxonomy nobody agreed to; a vocabulary
// borrowed from the books is a spelling convention, which is all a suggestion
// list should be.
//
// Kept short on purpose. Twenty suggestions in a dropdown is a menu to read;
// a dozen is a habit to acquire.
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
// Delimited, not substring: the tags of the text are extracted first and
// compared whole, so `#prime` does not match `#priming` — the difference the
// free-text comment search (`t"#prime"`) cannot make, and the reason a tag
// search is its own filter rather than a spelling of that one.
//
// An empty want matches everything, so a caller can pass its filter through
// unconditionally.
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
