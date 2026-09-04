package searchquery

// parity_test.go — the fiche's fourth bullet (#186): every field of
// domain.SearchFilters must be reachable from a query, or be listed as
// unreachable with a reason. The point is not the count; it is that adding a
// filter to the domain forces a decision, in writing, about how a user is
// supposed to ask for it. Before this, twenty-one of the forty-five fields
// existed only in the frontend's command bar.

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
