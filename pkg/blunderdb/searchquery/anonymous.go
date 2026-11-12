package searchquery

import (
	"reflect"
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
	// free to name any of them — or a row whose answer is narrowed by the
	// matches it was met in.
	ReadsProvenance
)

// FieldReads classifies every field of [FieldTokens]. The parity test fails
// when a field is in one map and not in the other: a new filter cannot be
// added without deciding, in writing, what it reads. [Anonymous] refuses a
// query by this table and no other list.
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

	// A ranking against a stored position leaves out the matches that
	// position was met in; against a drawn board it reads the board alone.
	"LikeTargetID": ReadsProvenance,

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

// rangeFields are the filters whose value is a numeric range written after
// the filter's own token.
var rangeFields = []string{
	"PipCountFilter", "Player1AbsolutePipCountFilter", "WinRateFilter", "GammonRateFilter", "BackgammonRateFilter",
	"Player2WinRateFilter", "Player2GammonRateFilter", "Player2BackgammonRateFilter",
	"Player1CheckerOffFilter", "Player2CheckerOffFilter", "Player1BackCheckerFilter", "Player2BackCheckerFilter",
	"Player1CheckerInZoneFilter", "Player2CheckerInZoneFilter", "Player1OutfieldBlotFilter", "Player2OutfieldBlotFilter",
	"Player1JanBlotFilter", "Player2JanBlotFilter", "EquityFilter",
}

// rangeShapes holds, for each of rangeFields, the only shape its value may
// leave in. A prefix rule of Parse claims any word that starts with its
// letter (`wanda` is a win-rate filter, `bZ<-11` a backgammon rate): the
// shape demands the filter's exact token, then a range.
var rangeShapes = func() map[string]*regexp.Regexp {
	out := make(map[string]*regexp.Regexp, len(rangeFields))
	for _, name := range rangeFields {
		out[name] = regexp.MustCompile(`^` + regexp.QuoteMeta(FieldTokens[name]) + `(?:[<>]` + number + `|` + number + `(?:,` + number + `)?)$`)
	}
	return out
}()

var (
	dayRe     = `\d{4}/\d{2}/\d{2}`
	dateRe    = regexp.MustCompile(`^T(?:[<>]` + dayRe + `|` + dayRe + `,` + dayRe + `)$`)
	patternRe = regexp.MustCompile(`^m["'](?:[0-9/*(); ,-]|bar|off|nd|dt|dp)+["']$`)
	depthRe   = regexp.MustCompile(`^(?:\d+ply\+?|book|rollout)$`)
	// Parse claims as an id list any word that starts with `id` and a digit.
	idListRe = regexp.MustCompile(`^\d+(?:[-,;]\d+)*$`)
)

// Anonymous returns the form of query that may leave a library without the
// provenance of its positions — [Format] of what [Parse] read, so no text the
// parser did not understand travels — or false when the query must not leave
// at all and what it selects has to be evaluated at the source instead. That
// is the case when it reads something that stays behind (a player, a match, a
// free text, the played move, the matches of a ranking's stored target; a
// comment when comments is false), when it holds a token the parser does not
// know or ignores, or when a value is not of the closed shape its filter
// reads: a word the storage layer would ignore is still a word. A tag is a
// word too: it leaves only when tags, the tags of the comments that leave,
// holds it — a tag no carried comment bears selects nothing there and tells
// what the sender was looking for.
func Anonymous(query string, comments bool, tags map[string]bool) (string, bool) {
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

	v := reflect.ValueOf(f)
	for name, reads := range FieldReads {
		if v.FieldByName(name).IsZero() {
			continue
		}
		switch reads {
		case ReadsPlayed, ReadsProvenance:
			return "", false
		case ReadsComment:
			if !comments {
				return "", false
			}
		}
	}
	for _, tag := range domain.ParseTagFilter(f.TagFilter) {
		if !tags[tag] {
			return "", false
		}
	}

	for name, shape := range rangeShapes {
		if r := v.FieldByName(name).String(); r != "" && !shape.MatchString(r) {
			return "", false
		}
	}
	if f.PositionIDsFilter != "" && !idListRe.MatchString(f.PositionIDsFilter) {
		return "", false
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
