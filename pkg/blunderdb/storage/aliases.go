package storage

import (
	"context"
	"sort"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// AliasKind says which names an alias renames: a player's or an event's.
type AliasKind string

const (
	AliasPlayer AliasKind = "player"
	AliasEvent  AliasKind = "event"
)

// Valid reports whether k is one of the two kinds.
func (k AliasKind) Valid() bool { return k == AliasPlayer || k == AliasEvent }

// Alias is one other spelling of a name: the import stores Canonical where a
// file writes Alias, and the stats, the players table and the search read
// both as one.
type Alias struct {
	Alias     string `json:"alias"`
	Canonical string `json:"canonical"`
}

// AliasStore keeps the aliases of players and events. The table stays flat:
// a canonical name is never itself an alias, so resolving is one lookup.
//
// A match keeps the names its file wrote in its fingerprints (match_hash,
// canonical_hash): the alias is applied after them, to the stored names
// only. Hashing the canonical names would make a file imported before its
// alias existed hash differently on its next import and escape the
// duplicate check.
type AliasStore interface {
	// Set makes alias a spelling of canonical. A canonical that is itself an
	// alias is followed to its own canonical, and the aliases that pointed at
	// alias move to canonical, so the table stays flat. Blank names, an alias
	// equal to its canonical and an invalid kind are ErrInvalid.
	Set(ctx context.Context, scope string, kind AliasKind, alias, canonical string) error

	// Remove forgets alias; removed is false when it was not one.
	Remove(ctx context.Context, scope string, kind AliasKind, alias string) (removed bool, err error)

	// List returns the aliases of kind ordered by canonical, then alias.
	List(ctx context.Context, scope string, kind AliasKind) ([]Alias, error)

	// Suggest runs SuggestAliases over the names the scope's matches write
	// (players at either seat, or events), weighted by how many matches
	// carry each. Nothing is recorded.
	Suggest(ctx context.Context, scope string, kind AliasKind) ([]AliasSuggestion, error)
}

// AliasMap resolves names through a list of aliases.
type AliasMap map[string]string

// NewAliasMap indexes aliases by their alias spelling.
func NewAliasMap(aliases []Alias) AliasMap {
	m := make(AliasMap, len(aliases))
	for _, a := range aliases {
		m[a.Alias] = a.Canonical
	}
	return m
}

// Canonical returns the name to store and group under for name: its
// canonical when name is an alias, name itself (trimmed) otherwise.
func (m AliasMap) Canonical(name string) string {
	name = strings.TrimSpace(name)
	if c, ok := m[name]; ok {
		return c
	}
	return name
}

// Group returns every spelling of the person or event names designates: the
// canonical of each name and all the aliases that point to it, without
// blanks or repeats, in a stable order.
func (m AliasMap) Group(names ...string) []string {
	canon := map[string]struct{}{}
	for _, n := range names {
		if n = strings.TrimSpace(n); n != "" {
			canon[m.Canonical(n)] = struct{}{}
		}
	}
	seen := map[string]struct{}{}
	var out []string
	add := func(s string) {
		if _, dup := seen[s]; !dup && s != "" {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	for _, n := range names {
		add(strings.TrimSpace(n))
	}
	keys := make([]string, 0, len(canon))
	for c := range canon {
		keys = append(keys, c)
	}
	sort.Strings(keys)
	aliases := make([]string, 0, len(m))
	for a := range m {
		aliases = append(aliases, a)
	}
	sort.Strings(aliases)
	for _, c := range keys {
		add(c)
		for _, a := range aliases {
			if m[a] == c {
				add(a)
			}
		}
	}
	return out
}

// AliasSuggestion is a set of names that differ only by case, accents,
// punctuation or the order of their words: probably one person, offered to
// the user and never applied without them.
type AliasSuggestion struct {
	// Canonical is the most frequent spelling, the one proposed to keep.
	Canonical string   `json:"canonical"`
	Aliases   []string `json:"aliases"`
}

// SuggestAliases groups names that fold to the same key (case, accents,
// punctuation and word order ignored). counts gives each name's weight; the
// heaviest spelling of a group is its proposed canonical, ties broken
// alphabetically. A name that is already an alias is left out: the user has
// decided for it.
func SuggestAliases(counts map[string]int, existing AliasMap) []AliasSuggestion {
	groups := map[string][]string{}
	for name := range counts {
		if _, aliased := existing[name]; aliased {
			continue
		}
		if k := foldName(name); k != "" {
			groups[k] = append(groups[k], name)
		}
	}
	var out []AliasSuggestion
	for _, names := range groups {
		if len(names) < 2 {
			continue
		}
		sort.Slice(names, func(i, j int) bool {
			if counts[names[i]] != counts[names[j]] {
				return counts[names[i]] > counts[names[j]]
			}
			return names[i] < names[j]
		})
		out = append(out, AliasSuggestion{Canonical: names[0], Aliases: names[1:]})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Canonical < out[j].Canonical })
	return out
}

var stripMarks = transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)

// foldName is the key SuggestAliases groups by: lower case, no accents, the
// words sorted, punctuation dropped.
func foldName(name string) string {
	s, _, err := transform.String(stripMarks, name)
	if err != nil {
		s = name
	}
	words := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	sort.Strings(words)
	return strings.Join(words, " ")
}
