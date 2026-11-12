package searchquery

import (
	"regexp"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// Reads names what the answer of a filter depends on, which decides whether a
// stored query can leave a library without the provenance of its positions.
type Reads int

const (
	// ReadsPosition: the board, its derived columns, its analysis, its flag.
	// All of it travels, so the filter selects the same rows at the
	// destination; its token leaves, rewritten by [Format].
	ReadsPosition Reads = iota + 1
	// ReadsComment: the comments of the position. The filter selects the same
	// rows only where the comments travel too.
	ReadsComment
	// ReadsPlayed: the move that was played and its clock, which never leave
	// without the provenance. The filter would select other rows there.
	ReadsPlayed
	// ReadsProvenance: a player, a match, a tournament, an author, or a text
	// free to name any of them.
	ReadsProvenance
)

// FieldReads classifies every field of [FieldTokens]. The parity test fails
// when a field is in one map and not in the other: a new filter cannot be
// added without deciding, in writing, what it reads.
var FieldReads = map[string]Reads{
	"IncludeCube":                   ReadsPosition,
	"IncludeScore":                  ReadsPosition,
	"DecisionTypeFilter":            ReadsPosition,
	"DiceRollFilter":                ReadsPosition,
	"DiceRollMode":                  ReadsPosition,
	"ExceptDiceFilter":              ReadsPosition,
	"NoContactFilter":               ReadsPosition,
	"MirrorFilter":                  ReadsPosition,
	"FlaggedFilter":                 ReadsPosition,
	"GamePhaseFilter":               ReadsPosition,
	"GameTypeFilter":                ReadsPosition,
	"LikeFilter":                    ReadsPosition,
	"LikeTargetID":                  ReadsPosition,
	"LikeMaxDistance":               ReadsPosition,
	"LikeWidened":                   ReadsPosition,
	"PipCountFilter":                ReadsPosition,
	"Player1AbsolutePipCountFilter": ReadsPosition,
	"WinRateFilter":                 ReadsPosition,
	"GammonRateFilter":              ReadsPosition,
	"BackgammonRateFilter":          ReadsPosition,
	"Player2WinRateFilter":          ReadsPosition,
	"Player2GammonRateFilter":       ReadsPosition,
	"Player2BackgammonRateFilter":   ReadsPosition,
	"Player1CheckerOffFilter":       ReadsPosition,
	"Player2CheckerOffFilter":       ReadsPosition,
	"Player1BackCheckerFilter":      ReadsPosition,
	"Player2BackCheckerFilter":      ReadsPosition,
	"Player1CheckerInZoneFilter":    ReadsPosition,
	"Player2CheckerInZoneFilter":    ReadsPosition,
	"Player1OutfieldBlotFilter":     ReadsPosition,
	"Player2OutfieldBlotFilter":     ReadsPosition,
	"Player1JanBlotFilter":          ReadsPosition,
	"Player2JanBlotFilter":          ReadsPosition,
	"EquityFilter":                  ReadsPosition,
	"DateFilter":                    ReadsPosition,
	"MovePatternFilter":             ReadsPosition,
	"AwayFilter":                    ReadsPosition,
	"OpponentAwayFilter":            ReadsPosition,
	"CrawfordFilter":                ReadsPosition,
	"AnalysisProvenanceFilter":      ReadsPosition,
	"PositionIDsFilter":             ReadsPosition,

	"CommentFilter":       ReadsComment,
	"CommentOriginFilter": ReadsComment,
	"TagFilter":           ReadsComment,

	// The error and the clock are those of the played move; the margin and
	// the take-or-pass side of a cube decision are told apart by it.
	"MoveErrorFilter":    ReadsPlayed,
	"DecisionTimeFilter": ReadsPlayed,
	"MarginFilter":       ReadsPlayed,
	"CubeResponseFilter": ReadsPlayed,

	"SearchText":                 ReadsProvenance,
	"PlayerFilter":               ReadsProvenance,
	"OpponentFilter":             ReadsProvenance,
	"TournamentNameFilter":       ReadsProvenance,
	"CommentAuthorFilter":        ReadsProvenance,
	"RoundFilter":                ReadsProvenance,
	"MatchLengthFilter":          ReadsProvenance,
	"MatchDateFilter":            ReadsProvenance,
	"PlayerPRFilter":             ReadsProvenance,
	"EncounterFilter":            ReadsProvenance,
	"MatchIDsFilter":             ReadsProvenance,
	"TournamentIDsFilter":        ReadsProvenance,
	"IndividuallyImportedFilter": ReadsProvenance,
}

const number = `-?\d+(?:\.\d+)?`

var (
	// A prefix rule of Parse claims any word that starts with its letter
	// (`wanda` is a win-rate filter): only this shape is a range.
	rangeRe   = regexp.MustCompile(`^[A-Za-z]{1,2}(?:[<>]` + number + `|` + number + `(?:,` + number + `)?)$`)
	dayRe     = `\d{4}/\d{2}/\d{2}`
	dateRe    = regexp.MustCompile(`^T(?:[<>]` + dayRe + `|` + dayRe + `,` + dayRe + `)$`)
	patternRe = regexp.MustCompile(`^m["'](?:[0-9/*(); ,-]|bar|off|nd|dt|dp)+["']$`)
	depthRe   = regexp.MustCompile(`^(?:\d+ply\+?|book|rollout)$`)
)

// Anonymous returns the form of query that may leave a library without the
// provenance of its positions — [Format] of what [Parse] read, so no text the
// parser did not understand travels — or false when the query must not leave
// at all and what it selects has to be evaluated at the source instead. That
// is the case when it reads something that stays behind (a player, a match, a
// free text, the played move; a comment when comments is false), when it
// holds a token the parser does not know or ignores, or when a value is not
// of the closed shape its filter reads: a word the storage layer would ignore
// is still a word.
func Anonymous(query string, comments bool) (string, bool) {
	f, diags := Parse(query)
	if len(diags) > 0 {
		return "", false
	}
	// Parse keeps the first quoted value of each kind and drops the others
	// without a word.
	quoted := 0
	for _, v := range []string{f.PlayerFilter, f.OpponentFilter, f.TournamentNameFilter, f.CommentAuthorFilter, f.MovePatternFilter, f.SearchText} {
		if v != "" {
			quoted++
		}
	}
	if len(quotedRe.FindAllString(query, -1)) != quoted {
		return "", false
	}

	for _, stays := range []string{
		f.SearchText, f.PlayerFilter, f.OpponentFilter, f.TournamentNameFilter, f.CommentAuthorFilter,
		f.RoundFilter, f.MatchLengthFilter, f.MatchDateFilter, f.PlayerPRFilter, f.EncounterFilter,
		f.MatchIDsFilter, f.TournamentIDsFilter,
		f.MoveErrorFilter, f.DecisionTimeFilter, f.MarginFilter, f.CubeResponseFilter,
	} {
		if stays != "" {
			return "", false
		}
	}
	if f.IndividuallyImportedFilter {
		return "", false
	}
	if !comments && (f.CommentFilter != "" || f.CommentOriginFilter != "" || f.TagFilter != "") {
		return "", false
	}

	for _, r := range []string{
		f.PipCountFilter, f.Player1AbsolutePipCountFilter, f.WinRateFilter, f.GammonRateFilter, f.BackgammonRateFilter,
		f.Player2WinRateFilter, f.Player2GammonRateFilter, f.Player2BackgammonRateFilter,
		f.Player1CheckerOffFilter, f.Player2CheckerOffFilter, f.Player1BackCheckerFilter, f.Player2BackCheckerFilter,
		f.Player1CheckerInZoneFilter, f.Player2CheckerInZoneFilter, f.Player1OutfieldBlotFilter, f.Player2OutfieldBlotFilter,
		f.Player1JanBlotFilter, f.Player2JanBlotFilter, f.EquityFilter,
	} {
		if r != "" && !rangeRe.MatchString(r) {
			return "", false
		}
	}
	if f.DateFilter != "" && !dateRe.MatchString(f.DateFilter) {
		return "", false
	}
	if f.MovePatternFilter != "" && !patternRe.MatchString(strings.ToLower(f.MovePatternFilter)) {
		return "", false
	}
	known := func(list string, ok func(string) bool) bool {
		for _, v := range domain.SplitFilterList(list) {
			if !ok(v) {
				return false
			}
		}
		return true
	}
	closed := known(f.GamePhaseFilter, func(v string) bool { _, ok := domain.ParseGamePhase(v); return ok }) &&
		known(f.GameTypeFilter, func(v string) bool { _, ok := domain.ParseGameType(v); return ok }) &&
		known(f.CommentOriginFilter, func(v string) bool {
			for _, o := range domain.CommentOrigins {
				if v == string(o) {
					return true
				}
			}
			return false
		}) &&
		// An engine is a free word (`ad:gammonnet` reads a label by prefix);
		// a depth is one of a closed set.
		known(f.AnalysisProvenanceFilter, depthRe.MatchString)
	if !closed {
		return "", false
	}
	return Format(f), true
}
