// Package searchquery is the Go side of blunderDB's search grammar: the one
// that turns `s cube p>30 E>0.05` into a domain.SearchFilters, and back.
//
// # Why it exists
//
// The grammar was born in the frontend and lived there alone. `blunderdb
// search` offered 24 flags against 45 filter fields, and `/v1` offered none of
// the token forms at all: board patterns, mirror search, move patterns, dates,
// equity, comment text, excluded dice, zones and blots were reachable from the
// command bar and from nowhere else. Anything scripted had to rebuild the
// struct by hand and hope it matched what the GUI would have produced.
//
// This package is the second reader of one grammar, not a second grammar.
// [Parse] is a line-for-line port of `parseSearchTokens` in
// frontend/src/services/searchFilterService.js, and both are held to
// testdata/search_query_corpus.json — the same arrangement testdata/
// xgid_corpus.json imposes on the two XGID decoders. A case added on either
// side fails the other until both agree.
//
// # What the fields hold
//
// Deliberately, the string filters keep the token whole — SearchText is
// `t"blunder"`, not `blunder`; MovePatternFilter is `m"13/11"`. That is the
// shape the storage backends already parse (searchfilter.ParseSearchTextKeywords,
// searchfilter.AnalysisMatchesMovePattern, searchfilter.PlayerName), and the
// shape the frontend has always sent. Format is the exact inverse, so
// Parse(Format(f)) == f for every filter this grammar can express.
//
// # Why the frontend still has its own parser
//
// The fiche left the choice open: have the JS call into Go through a binding,
// or keep a JS parser generated from the same corpus. Neither: the JS keeps the
// hand-written parseSearchTokens it already had, and the corpus is the contract
// between them.
//
// A binding was the tempting answer and is the wrong one. The command bar
// parses on every keystroke to drive autocompletion, and a Wails round trip per
// keystroke trades a working feature for an architectural preference. Worse,
// the frontend's own test suites (vitest, Playwright) run with no Go process at
// all — the grammar would become untestable exactly where it is used. Generating
// the JS from the corpus fails for the same reason plus one: the corpus states
// what the grammar must do on 32 cases, not what it does on everything else.
//
// So the two implementations stay, and the corpus keeps them honest. That is
// the same arrangement testdata/xgid_corpus.json has held for the two XGID
// decoders, and it has caught real drift.
//
// # What it cannot express
//
// Four fields of domain.SearchFilters have no token, by nature rather than by
// omission, and searchquery_parity_test.go pins the list:
//
//   - Filter and ExcludeFilter are board positions — the checkers themselves,
//     which the user draws rather than types.
//   - RestrictToPositionIDs is an internal narrowing (search inside a result
//     set), not a user-facing filter.
//   - Sort is set by the caller, not by the query text; adding a token here
//     alone would fork the grammar the corpus exists to keep single.
package searchquery

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// DiagKind classifies what [Parse] wants to say about a token.
type DiagKind string

const (
	// DiagUnknown marks a token no rule claimed. The token is ignored — a
	// search is a filter, and an unreadable filter that silently narrows
	// nothing is friendlier than a refused query — but the caller can surface
	// it, and `blunderdb search --query` does.
	DiagUnknown DiagKind = "unknown"
	// DiagNoEffect marks a token that is understood but cannot act here: `x`
	// switches on the exclusion structure, and the structure itself is a board
	// the GUI holds, so a text query carries the switch and nothing to switch on.
	DiagNoEffect DiagKind = "no-effect"
)

// A Diag reports one token [Parse] could not turn into a filter.
type Diag struct {
	Kind    DiagKind
	Token   string
	Message string
}

func (d Diag) String() string { return fmt.Sprintf("%s: %s", d.Token, d.Message) }

// Quoted filter values — pl"…" (player), m"…" (move pattern) and t"…" (comment
// text) — may contain spaces, so they are lifted out of the command before it
// is split. Without that, `t"big win"` tears into `t"big` and `win"`, and the
// loose `win"` is then read as a win-rate filter: the query silently returns
// the wrong rows. Both quote styles are accepted, as in the JS.
var (
	quotedRe    = regexp.MustCompile(`(?:pl|m|t)["'][^"']*["']`)
	movePatRe   = regexp.MustCompile(`m["'][^"']*["']`)
	searchTxtRe = regexp.MustCompile(`t["'][^"']*["']`)
	playerRe    = regexp.MustCompile(`pl["'][^"']*["']`)
	exceptDice  = regexp.MustCompile(`^xD[1-6][1-6]$`)
	maRe        = regexp.MustCompile(`^ma\d`)
	tnRe        = regexp.MustCompile(`^tn\d`)
	idRe        = regexp.MustCompile(`^id\d`)
	// Closed vocabularies, so the token names its value rather than quoting it:
	// `ph:race`, `co:user`. Repeatable, and joined the way the storage layer
	// expects — the same shape as the id lists above.
	phaseRe   = regexp.MustCompile(`^ph:[a-z]+$`)
	originRe  = regexp.MustCompile(`^co:[a-z]+$`)
	moveErrRe = regexp.MustCompile(`^E\d`)
)

// StripQuoted blanks out the quoted regions of a command so a whitespace split
// cannot tear a multi-word value apart. Exported because the same treatment is
// needed by anything that wants the command's bare tokens.
func StripQuoted(s string) string { return quotedRe.ReplaceAllString(s, " ") }

// Tokenize splits a search command into its filter tokens, dropping the leading
// verb (`s` or `ss`) and the quoted regions. A command that is only the verb
// yields no tokens.
func Tokenize(command string) []string {
	cmd := strings.TrimSpace(command)
	switch cmd {
	case "", "s", "ss":
		return nil
	}
	if rest, ok := strings.CutPrefix(cmd, "ss "); ok {
		cmd = rest
	} else if rest, ok := strings.CutPrefix(cmd, "s "); ok {
		cmd = rest
	}
	return strings.Fields(StripQuoted(cmd))
}

// Parse reads a search command — `s cube p>30`, `ss t"blunder"`, or the bare
// token list — into the filters it denotes, plus a diagnostic per token no rule
// claimed. It never fails: an unreadable token is reported and skipped.
//
// It is a port of parseSearchTokens (searchFilterService.js) and is held to the
// same corpus; see the package doc.
func Parse(command string) (domain.SearchFilters, []Diag) {
	tokens := Tokenize(command)
	var f domain.SearchFilters
	claimed := make([]bool, len(tokens))

	has := func(want string) bool {
		found := false
		for i, tok := range tokens {
			if tok == want {
				claimed[i] = true
				found = true
			}
		}
		return found
	}
	// first returns the first token satisfying pred, marking it claimed. The
	// JS uses Array.prototype.find, so a repeated numeric filter keeps its
	// first occurrence; repeating one is a user error either way.
	first := func(pred func(string) bool) string {
		for i, tok := range tokens {
			if pred(tok) {
				claimed[i] = true
				return tok
			}
		}
		return ""
	}
	all := func(pred func(string) bool) []string {
		var out []string
		for i, tok := range tokens {
			if pred(tok) {
				claimed[i] = true
				out = append(out, tok)
			}
		}
		return out
	}
	prefix := func(p string) func(string) bool {
		return func(s string) bool { return strings.HasPrefix(s, p) }
	}

	// Board-level toggles. `cube`/`score` keep their historical abbreviations.
	f.IncludeCube = has("cube") || has("cu") || has("c") || has("cub")
	f.IncludeScore = has("score") || has("sco") || has("sc") || has("s")
	f.NoContactFilter = has("nc")
	f.DecisionTypeFilter = has("d")
	// `D` matches the board's dice, `D1` only the first die.
	dBoth, dFirst := has("D"), has("D1")
	f.DiceRollFilter = dBoth || dFirst
	f.DiceRollMode = "both"
	if dFirst {
		f.DiceRollMode = "first"
	}
	f.MirrorFilter = has("M")
	f.IndividuallyImportedFilter = has("i")
	f.FlaggedFilter = has("fl")
	// Comment presence. Asking for both is contradictory rather than ambiguous;
	// "none" wins and the search comes back empty, which is the honest answer.
	switch {
	case has("xco"):
		f.CommentFilter = "none"
	case has("co"):
		f.CommentFilter = "has"
	}

	// `xD65` excludes the 6-5 roll (order-insensitive) and repeats; the values
	// travel to the backend as one ";"-separated string.
	var excluded []string
	for _, tok := range all(func(s string) bool { return exceptDice.MatchString(s) }) {
		excluded = append(excluded, tok[2:])
	}
	f.ExceptDiceFilter = strings.Join(excluded, ";")

	// Closed vocabularies, claimed BEFORE the numeric ranges: `ph:race` starts
	// with `p` and would otherwise be swallowed by the pip-count filter, the
	// way `pl"…"` once was.
	f.GamePhaseFilter = joinValues(all(func(s string) bool { return phaseRe.MatchString(s) }), 3)
	f.CommentOriginFilter = joinValues(all(func(s string) bool { return originRe.MatchString(s) }), 3)

	// Numeric ranges. Each reads `x>n`, `x<n` or `xa,b`; the six count filters
	// below additionally accept a bare `x5`, which means exactly five.
	f.PipCountFilter = first(func(s string) bool {
		return strings.HasPrefix(s, "p") && !strings.HasPrefix(s, "pl") && !strings.HasPrefix(s, "ph")
	})
	f.WinRateFilter = first(prefix("w"))
	f.GammonRateFilter = first(prefix("g"))
	f.BackgammonRateFilter = first(func(s string) bool {
		return strings.HasPrefix(s, "b") && !strings.HasPrefix(s, "bo") && !strings.HasPrefix(s, "bj")
	})
	f.Player2WinRateFilter = first(prefix("W"))
	f.Player2GammonRateFilter = first(prefix("G"))
	f.Player2BackgammonRateFilter = first(func(s string) bool {
		return strings.HasPrefix(s, "B") && !strings.HasPrefix(s, "BO") && !strings.HasPrefix(s, "BJ")
	})
	f.Player1CheckerOffFilter = exact(first(prefix("o")))
	f.Player2CheckerOffFilter = exact(first(prefix("O")))
	f.Player1BackCheckerFilter = exact(first(prefix("k")))
	f.Player2BackCheckerFilter = exact(first(prefix("K")))
	f.Player1CheckerInZoneFilter = exact(first(prefix("z")))
	f.Player2CheckerInZoneFilter = exact(first(prefix("Z")))
	f.Player1AbsolutePipCountFilter = first(prefix("P"))
	f.EquityFilter = first(prefix("e"))
	f.DateFilter = first(prefix("T"))
	f.Player1OutfieldBlotFilter = first(prefix("bo"))
	f.Player2OutfieldBlotFilter = first(prefix("BO"))
	f.Player1JanBlotFilter = first(prefix("bj"))
	f.Player2JanBlotFilter = first(prefix("BJ"))
	f.MoveErrorFilter = first(func(s string) bool {
		return strings.HasPrefix(s, "E>") || strings.HasPrefix(s, "E<") || moveErrRe.MatchString(s)
	})

	// Repeatable id lists, joined the way the storage layer expects.
	f.MatchIDsFilter = joinValues(all(func(s string) bool { return maRe.MatchString(s) }), 2)
	f.TournamentIDsFilter = joinValues(all(func(s string) bool { return tnRe.MatchString(s) }), 2)
	f.PositionIDsFilter = joinValues(all(func(s string) bool { return idRe.MatchString(s) }), 2)

	// Quoted values are read off the raw command, not the split tokens, so a
	// name or a comment with spaces survives. The token is kept whole, wrapper
	// included: that is what the storage layer unwraps.
	raw := strings.TrimSpace(command)
	f.MovePatternFilter = movePatRe.FindString(raw)
	f.SearchText = searchTxtRe.FindString(raw)
	f.PlayerFilter = playerRe.FindString(raw)

	var diags []Diag
	for i, tok := range tokens {
		if claimed[i] {
			continue
		}
		if tok == "x" {
			diags = append(diags, Diag{
				Kind:  DiagNoEffect,
				Token: tok,
				Message: "the exclusion structure is a board, not a token: `x` only switches on " +
					"a pattern the interface holds, so a text query has nothing to exclude",
			})
			continue
		}
		diags = append(diags, Diag{Kind: DiagUnknown, Token: tok, Message: "unknown filter token"})
	}
	return f, diags
}

// exact expands the bare form of a count filter (`o5`) into the range the
// backend compares against (`o5,5`). A token that already carries a bound or a
// range is left alone, and so is the empty string.
func exact(tok string) string {
	if tok == "" || strings.ContainsAny(tok, ",<>") {
		return tok
	}
	return tok + "," + tok[1:]
}

// joinValues strips the prefix of each token and joins the values with ";",
// which is how the storage layer reads an id list.
func joinValues(tokens []string, prefixLen int) string {
	if len(tokens) == 0 {
		return ""
	}
	values := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		values = append(values, tok[prefixLen:])
	}
	return strings.Join(values, ";")
}

// Format renders filters back into a search command. It is the inverse of
// [Parse] over everything the grammar can express (see the package doc for the
// four fields it cannot), so Parse(Format(f)) == f — the property
// searchquery_roundtrip_test.go checks against generated filter sets.
//
// The token order is fixed rather than incidental: flags first, then the
// numeric ranges in the order the search panel lists them, then the quoted
// values and the id lists. Two equal filter sets therefore format identically,
// which is what lets a saved search be compared, deduplicated, and diffed.
func Format(f domain.SearchFilters) string {
	parts := []string{"s"}
	add := func(tok string) {
		if tok != "" {
			parts = append(parts, tok)
		}
	}
	flag := func(on bool, tok string) {
		if on {
			parts = append(parts, tok)
		}
	}

	flag(f.IncludeCube, "cube")
	flag(f.IncludeScore, "score")
	flag(f.DecisionTypeFilter, "d")
	if f.DiceRollFilter {
		if f.DiceRollMode == "first" {
			add("D1")
		} else {
			add("D")
		}
	}
	for _, roll := range strings.Split(f.ExceptDiceFilter, ";") {
		if roll != "" {
			add("xD" + roll)
		}
	}
	flag(f.NoContactFilter, "nc")
	flag(f.MirrorFilter, "M")
	flag(f.IndividuallyImportedFilter, "i")
	flag(f.FlaggedFilter, "fl")
	switch f.CommentFilter {
	case "has":
		add("co")
	case "none":
		add("xco")
	}

	add(f.PipCountFilter)
	add(f.Player1AbsolutePipCountFilter)
	add(f.WinRateFilter)
	add(f.GammonRateFilter)
	add(f.BackgammonRateFilter)
	add(f.Player2WinRateFilter)
	add(f.Player2GammonRateFilter)
	add(f.Player2BackgammonRateFilter)
	add(f.Player1CheckerOffFilter)
	add(f.Player2CheckerOffFilter)
	add(f.Player1BackCheckerFilter)
	add(f.Player2BackCheckerFilter)
	add(f.Player1CheckerInZoneFilter)
	add(f.Player2CheckerInZoneFilter)
	add(f.Player1OutfieldBlotFilter)
	add(f.Player2OutfieldBlotFilter)
	add(f.Player1JanBlotFilter)
	add(f.Player2JanBlotFilter)
	add(f.EquityFilter)
	add(f.MoveErrorFilter)
	add(f.DateFilter)

	add(f.SearchText)
	add(f.MovePatternFilter)
	add(f.PlayerFilter)

	addList(&parts, "ma", f.MatchIDsFilter)
	addList(&parts, "tn", f.TournamentIDsFilter)
	addList(&parts, "id", f.PositionIDsFilter)
	addList(&parts, "ph:", f.GamePhaseFilter)
	addList(&parts, "co:", f.CommentOriginFilter)

	return strings.Join(parts, " ")
}

// addList renders a ";"-separated id list back as one token per id, the form
// Parse reads. A list joined into a single token (`ma1;2`) would parse back to
// the same string, but the per-id form is what the interface produces and what
// the corpus pins.
func addList(parts *[]string, prefix, list string) {
	for _, v := range strings.Split(list, ";") {
		if v != "" {
			*parts = append(*parts, prefix+v)
		}
	}
}

// FieldTokens maps each domain.SearchFilters field this grammar can express to
// the token prefix that expresses it. searchquery_parity_test.go walks
// domain.SearchFilters by reflection and fails when a field appears here
// neither as a token nor in the unrepresentable list — so a new filter field
// cannot be added to the domain without deciding, in writing, how a user is
// meant to ask for it.
var FieldTokens = map[string]string{
	"IncludeCube":                   "cube",
	"IncludeScore":                  "score",
	"DecisionTypeFilter":            "d",
	"DiceRollFilter":                "D",
	"DiceRollMode":                  "D1",
	"ExceptDiceFilter":              "xD",
	"NoContactFilter":               "nc",
	"MirrorFilter":                  "M",
	"IndividuallyImportedFilter":    "i",
	"FlaggedFilter":                 "fl",
	"CommentFilter":                 "co",
	"CommentOriginFilter":           "co:",
	"GamePhaseFilter":               "ph:",
	"PipCountFilter":                "p",
	"Player1AbsolutePipCountFilter": "P",
	"WinRateFilter":                 "w",
	"GammonRateFilter":              "g",
	"BackgammonRateFilter":          "b",
	"Player2WinRateFilter":          "W",
	"Player2GammonRateFilter":       "G",
	"Player2BackgammonRateFilter":   "B",
	"Player1CheckerOffFilter":       "o",
	"Player2CheckerOffFilter":       "O",
	"Player1BackCheckerFilter":      "k",
	"Player2BackCheckerFilter":      "K",
	"Player1CheckerInZoneFilter":    "z",
	"Player2CheckerInZoneFilter":    "Z",
	"Player1OutfieldBlotFilter":     "bo",
	"Player2OutfieldBlotFilter":     "BO",
	"Player1JanBlotFilter":          "bj",
	"Player2JanBlotFilter":          "BJ",
	"EquityFilter":                  "e",
	"MoveErrorFilter":               "E",
	"DateFilter":                    "T",
	"SearchText":                    `t"…"`,
	"MovePatternFilter":             `m"…"`,
	"PlayerFilter":                  `pl"…"`,
	"MatchIDsFilter":                "ma",
	"TournamentIDsFilter":           "tn",
	"PositionIDsFilter":             "id",
	"CubeResponseFilter":            "", // set by the GUI's cube panel, no token yet
}

// Unrepresentable lists the domain.SearchFilters fields no token can carry, and
// why. See the package doc.
var Unrepresentable = map[string]string{
	"Filter":                "a board position: the checkers are drawn, not typed",
	"ExcludeFilter":         "a board position: the exclusion structure is drawn, not typed",
	"RestrictToPositionIDs": "an internal narrowing (search within a result set), not a user filter",
	"Sort":                  "set by the caller; a Go-only token would fork the grammar",
}

// Tokens returns every token prefix the grammar understands, sorted. Used by
// the CLI's --query help and by the parity test.
func Tokens() []string {
	out := make([]string, 0, len(FieldTokens))
	for _, tok := range FieldTokens {
		if tok != "" {
			out = append(out, tok)
		}
	}
	sort.Strings(out)
	return out
}
