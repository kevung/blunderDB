package searchquery

// parity_test.go — every field of domain.SearchFilters must be reachable from
// a query, or be listed as unreachable with a reason, so adding a filter forces
// a decision about how a user asks for it.

import (
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

func TestEveryFilterFieldIsAccountedFor(t *testing.T) {
	t.Parallel()
	typ := reflect.TypeOf(domain.SearchFilters{})
	for i := 0; i < typ.NumField(); i++ {
		name := typ.Field(i).Name
		_, hasToken := FieldTokens[name]
		_, unrepresentable := Unrepresentable[name]
		switch {
		case hasToken && unrepresentable:
			t.Errorf("%s is listed both as a token and as unrepresentable", name)
		case !hasToken && !unrepresentable:
			t.Errorf("domain.SearchFilters.%s is reachable from no query token and is not listed in Unrepresentable.\n"+
				"Add its token to FieldTokens (and a corpus case in testdata/search_query_corpus.json), or say in Unrepresentable why a user cannot type it.", name)
		}
	}
	// The reverse direction: a token naming a field that no longer exists.
	for name := range FieldTokens {
		if _, ok := typ.FieldByName(name); !ok {
			t.Errorf("FieldTokens names %q, which domain.SearchFilters no longer has", name)
		}
	}
	for name := range Unrepresentable {
		if _, ok := typ.FieldByName(name); !ok {
			t.Errorf("Unrepresentable names %q, which domain.SearchFilters no longer has", name)
		}
	}
}

// TestRoundTripGeneratedFilters is the property the corpus cannot state: over
// randomly built filter sets, Parse(Format(f)) == f. It catches the token
// collisions a hand-written corpus misses — a `bo3,5` read back as a
// backgammon-rate filter, an `o5` that loses its expansion.
func TestRoundTripGeneratedFilters(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewSource(20260904))
	for i := 0; i < 2000; i++ {
		want := randomFilters(rng)
		command := Format(want)
		got, diags := Parse(command)
		for _, d := range diags {
			if d.Kind == DiagUnknown {
				t.Fatalf("Format produced a token Parse rejects: %q in %q", d.Token, command)
			}
		}
		if !reflect.DeepEqual(want, got) {
			t.Fatalf("round trip lost information\n  command %q\n  want    %+v\n  got     %+v", command, want, got)
		}
	}
}

// randomFilters builds a filter set out of values the grammar can express. The
// numeric filters take one of the three shapes the tokens allow; the six count
// filters are given the expanded range form Parse normalises to, since that is
// what a parsed filter set holds.
func randomFilters(rng *rand.Rand) domain.SearchFilters {
	var f domain.SearchFilters
	maybe := func() bool { return rng.Intn(2) == 0 }
	// bound renders `x>n`, `x<n` or `xa,b` for a prefix.
	bound := func(prefix string) string {
		if !maybe() {
			return ""
		}
		switch rng.Intn(3) {
		case 0:
			return prefix + ">" + strconv.Itoa(rng.Intn(100))
		case 1:
			return prefix + "<" + strconv.Itoa(rng.Intn(100))
		default:
			lo := rng.Intn(50)
			return prefix + strconv.Itoa(lo) + "," + strconv.Itoa(lo+rng.Intn(50))
		}
	}
	// count renders what Parse leaves behind for o/O/k/K/z/Z: either a bound,
	// or the expanded `o5,5` form a bare `o5` becomes.
	count := func(prefix string) string {
		if !maybe() {
			return ""
		}
		if rng.Intn(2) == 0 {
			n := strconv.Itoa(rng.Intn(15))
			return prefix + n + "," + n
		}
		return bound(prefix)
	}

	f.IncludeCube = maybe()
	f.IncludeScore = maybe()
	f.DecisionTypeFilter = maybe()
	f.NoContactFilter = maybe()
	f.MirrorFilter = maybe()
	f.IndividuallyImportedFilter = maybe()
	f.FlaggedFilter = maybe()
	f.DiceRollFilter = maybe()
	f.DiceRollMode = "both"
	if f.DiceRollFilter && maybe() {
		f.DiceRollMode = "first"
	}
	if maybe() {
		var rolls []string
		for n := rng.Intn(3) + 1; n > 0; n-- {
			rolls = append(rolls, strconv.Itoa(rng.Intn(6)+1)+strconv.Itoa(rng.Intn(6)+1))
		}
		f.ExceptDiceFilter = strings.Join(rolls, ";")
	}
	switch rng.Intn(3) {
	case 0:
		f.CommentFilter = "has"
	case 1:
		f.CommentFilter = "none"
	}
	f.PipCountFilter = bound("p")
	f.Player1AbsolutePipCountFilter = bound("P")
	f.WinRateFilter = bound("w")
	f.GammonRateFilter = bound("g")
	f.BackgammonRateFilter = bound("b")
	f.Player2WinRateFilter = bound("W")
	f.Player2GammonRateFilter = bound("G")
	f.Player2BackgammonRateFilter = bound("B")
	f.Player1CheckerOffFilter = count("o")
	f.Player2CheckerOffFilter = count("O")
	f.Player1BackCheckerFilter = count("k")
	f.Player2BackCheckerFilter = count("K")
	f.Player1CheckerInZoneFilter = count("z")
	f.Player2CheckerInZoneFilter = count("Z")
	f.Player1OutfieldBlotFilter = bound("bo")
	f.Player2OutfieldBlotFilter = bound("BO")
	f.Player1JanBlotFilter = bound("bj")
	f.Player2JanBlotFilter = bound("BJ")
	f.EquityFilter = bound("e")
	if maybe() {
		f.MoveErrorFilter = "E>" + strconv.Itoa(rng.Intn(200))
	}
	f.DecisionTimeFilter = bound("tm")
	if maybe() {
		f.DateFilter = "T>2026/0" + strconv.Itoa(rng.Intn(9)+1) + "/01"
	}
	if maybe() {
		f.SearchText = `t"` + pick(rng, "blunder", "big win", "cube;tag") + `"`
	}
	if maybe() {
		f.MovePatternFilter = `m"` + pick(rng, "13/11", "24/18 13/11", "nd") + `"`
	}
	if maybe() {
		f.PlayerFilter = `pl"` + pick(rng, "Alice", "Kévin Unger") + `"`
	}
	if maybe() {
		f.PlayerFilter = pick(rng, `pl!"Alice"`, `pl!"Ali*"`)
	}
	if maybe() {
		f.OpponentFilter = `op"` + pick(rng, "Bob", "Bob Smith", "B*") + `"`
	}
	if maybe() {
		f.TournamentNameFilter = `tn"` + pick(rng, "Open Cup", "open*") + `"`
	}
	if maybe() {
		f.RoundFilter = pick(rng, "3", "3;Final*", "1;2;4")
	}
	if maybe() {
		f.MatchLengthFilter = pick(rng, "ml:7", "ml:5,9", "ml>5", "ml<9")
	}
	if maybe() {
		f.AwayFilter = pick(rng, "aw:4", "aw:2,5", "aw>3", "aw<5", "aw:1")
	}
	if maybe() {
		f.MarginFilter = pick(rng, "mg>100", "mg<20", "mg20,60", "mg<12.5")
	}
	if maybe() {
		f.OpponentAwayFilter = pick(rng, "AW:4", "AW:2,5", "AW>3", "AW<5", "AW:1")
	}
	if maybe() {
		f.CrawfordFilter = pick(rng, domain.CrawfordGame, domain.CrawfordAfter, domain.CrawfordNone)
	}
	if maybe() {
		f.MatchDateFilter = pick(rng, "md:2024-01..2024-12", "md:2024", "md>2024-06", "md<2024/06/30")
	}
	if maybe() {
		f.PlayerPRFilter = pick(rng, "pr>8", "pr<5.5", "pr4,9")
	}
	if maybe() {
		f.AnalysisProvenanceFilter = pick(rng, "xg", "gnubg;3ply", "4ply+;rollout;book")
	}
	f.CubeResponseFilter = pick(rng, "", "double", "takepass")
	f.MatchIDsFilter = idList(rng, 3)
	f.TournamentIDsFilter = idList(rng, 2)
	f.PositionIDsFilter = idList(rng, 4)
	return f
}

func idList(rng *rand.Rand, max int) string {
	if rng.Intn(2) == 0 {
		return ""
	}
	var ids []string
	for n := rng.Intn(max) + 1; n > 0; n-- {
		ids = append(ids, strconv.Itoa(rng.Intn(900)+1))
	}
	return strings.Join(ids, ";")
}

func pick(rng *rand.Rand, options ...string) string { return options[rng.Intn(len(options))] }

// TestFieldReads_ClassifiesEveryToken: a filter the grammar can express says
// what it reads, or it could leave a library without provenance unexamined.
func TestFieldReads_ClassifiesEveryToken(t *testing.T) {
	for name := range FieldTokens {
		if _, ok := FieldReads[name]; !ok {
			t.Errorf("FieldTokens names %q, which FieldReads does not classify: say what the filter reads, and make Anonymous act on it", name)
		}
	}
	for name := range FieldReads {
		if _, ok := FieldTokens[name]; !ok {
			t.Errorf("FieldReads names %q, which no token expresses", name)
		}
	}
}

// TestAnonymous_IsWhatTheParserWrites: a query that may leave is a fixed
// point of Parse and Format, and reads nothing that stays behind.
func TestAnonymous_IsWhatTheParserWrites(t *testing.T) {
	for _, q := range []string{"s w>50", "s c o5 nc", `s m"13/11;bar/22"`, "s ph:race gt:holding ad:book", "s #Prime co"} {
		got, ok := Anonymous(q, true, map[string]bool{"#prime": true})
		if !ok {
			t.Errorf("%q may not leave; want it rewritten", q)
			continue
		}
		f, diags := Parse(got)
		if len(diags) > 0 || Format(f) != got {
			t.Errorf("%q leaves as %q, which Parse and Format write back as %q (%v)", q, got, Format(f), diags)
		}
	}
	if _, ok := Anonymous("s #prime", false, map[string]bool{"#prime": true}); ok {
		t.Error("a tag may leave without the comments it is read from")
	}
}

// TestAnonymous_RefusesWhatStaysBehind: every field FieldReads does not class
// as read from the position keeps its query at the source, and so does a
// value that is not of the shape its filter reads.
func TestAnonymous_RefusesWhatStaysBehind(t *testing.T) {
	tags := map[string]bool{"#prime": true}
	for _, q := range []string{
		// A ranking against a stored row leaves out the matches it was met in.
		"s like42", "s like42<12*", "s w>50 like5",
		// A word in the place of a number.
		"s id1Zanzibar", "s id1 id3Zanzibar", "s bZ<-11", "s wZ>50", "s BOx>1", "s eZ>0",
		// A tag no comment that leaves bears.
		"s #zanzibar", "s #prime #zanzibar",
	} {
		if got, ok := Anonymous(q, true, tags); ok {
			t.Errorf("%q leaves, as %q", q, got)
		}
	}
	for _, q := range []string{"s like", "s like<12*", "s id1 id3", "s b<11", "s p-5,5", "s #prime"} {
		if _, ok := Anonymous(q, true, tags); !ok {
			t.Errorf("%q does not leave", q)
		}
	}
	// Every field classed as staying behind is refused once it is set, the
	// comments' fields when the comments stay: the table is what refuses.
	for name, reads := range FieldReads {
		if reads == ReadsPosition {
			continue
		}
		var f domain.SearchFilters
		field := reflect.ValueOf(&f).Elem().FieldByName(name)
		switch field.Kind() {
		case reflect.Bool:
			field.SetBool(true)
		case reflect.Int64:
			field.SetInt(7)
		default:
			continue // a string field: its token is in the corpus of the export test
		}
		if Format(f) == Format(domain.SearchFilters{}) {
			continue // written only beside another field
		}
		if got, ok := Anonymous("s "+Format(f), false, nil); ok {
			t.Errorf("%s is classed as staying behind, and %q leaves as %q", name, Format(f), got)
		}
	}
}

// TestAnonymous_RefusesAnUnclassifiedField: a filter the grammar can set and
// the table does not classify keeps its query at the source, so a filter added
// tomorrow without a line in FieldReads cannot leave by default.
func TestAnonymous_RefusesAnUnclassifiedField(t *testing.T) {
	for q, name := range map[string]string{"s p-5,5": "PipCountFilter", "s w>50": "WinRateFilter", "s like": "LikeFilter"} {
		if _, ok := anonymous(q, true, nil, FieldReads); !ok {
			t.Fatalf("%q does not leave with every field classified", q)
		}
		unclassified := make(map[string]Reads, len(FieldReads))
		for k, v := range FieldReads {
			if k != name {
				unclassified[k] = v
			}
		}
		if got, ok := anonymous(q, true, nil, unclassified); ok {
			t.Errorf("%q leaves as %q though %s is not classified", q, got, name)
		}
	}
}
