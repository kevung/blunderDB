package searchquery

// corpus_test.go — the shared-corpus gate. testdata/search_query_corpus.json is
// written once and read twice: by this package and by
// frontend/src/__tests__/searchQueryCorpus.test.js. A case added on either side
// fails the other until both grammars agree, which is the whole point (#186,
// #203) — the JS grammar and this one were meant to be one, and only a shared
// fixture keeps them so.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// TestMain runs the package from the repository root: the corpus is referenced
// by its repo-relative path, exactly as the JS side references it.
func TestMain(m *testing.M) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("cannot determine test file location")
	}
	if err := os.Chdir(filepath.Join(filepath.Dir(thisFile), "..", "..", "..")); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

// corpusField maps a corpus key to the domain.SearchFilters field it asserts.
// The two names differ where the frontend's own field name differs from the
// backend's (mirrorPositionFilter/MirrorFilter), and `excludeStructure` maps to
// no field at all — it is the `x` token, which this grammar reports as a
// no-effect diagnostic rather than a filter (see the package doc).
var corpusField = map[string]string{
	"includeCube":                   "IncludeCube",
	"includeScore":                  "IncludeScore",
	"decisionTypeFilter":            "DecisionTypeFilter",
	"noContactFilter":               "NoContactFilter",
	"mirrorPositionFilter":          "MirrorFilter",
	"individuallyImportedFilter":    "IndividuallyImportedFilter",
	"flaggedFilter":                 "FlaggedFilter",
	"gamePhaseFilter":               "GamePhaseFilter",
	"tagFilter":                     "TagFilter",
	"commentOriginFilter":           "CommentOriginFilter",
	"diceRollFilter":                "DiceRollFilter",
	"diceRollMode":                  "DiceRollMode",
	"exceptDiceFilter":              "ExceptDiceFilter",
	"commentFilter":                 "CommentFilter",
	"pipCountFilter":                "PipCountFilter",
	"winRateFilter":                 "WinRateFilter",
	"gammonRateFilter":              "GammonRateFilter",
	"backgammonRateFilter":          "BackgammonRateFilter",
	"player2WinRateFilter":          "Player2WinRateFilter",
	"player2GammonRateFilter":       "Player2GammonRateFilter",
	"player2BackgammonRateFilter":   "Player2BackgammonRateFilter",
	"player1CheckerOffFilter":       "Player1CheckerOffFilter",
	"player2CheckerOffFilter":       "Player2CheckerOffFilter",
	"player1BackCheckerFilter":      "Player1BackCheckerFilter",
	"player2BackCheckerFilter":      "Player2BackCheckerFilter",
	"player1CheckerInZoneFilter":    "Player1CheckerInZoneFilter",
	"player2CheckerInZoneFilter":    "Player2CheckerInZoneFilter",
	"player1AbsolutePipCountFilter": "Player1AbsolutePipCountFilter",
	"equityFilter":                  "EquityFilter",
	"dateFilter":                    "DateFilter",
	"movePatternFilter":             "MovePatternFilter",
	"searchText":                    "SearchText",
	"player1OutfieldBlotFilter":     "Player1OutfieldBlotFilter",
	"player2OutfieldBlotFilter":     "Player2OutfieldBlotFilter",
	"player1JanBlotFilter":          "Player1JanBlotFilter",
	"player2JanBlotFilter":          "Player2JanBlotFilter",
	"moveErrorFilter":               "MoveErrorFilter",
	"matchIDsFilter":                "MatchIDsFilter",
	"tournamentIDsFilter":           "TournamentIDsFilter",
	"playerFilter":                  "PlayerFilter",
	"positionIDsFilter":             "PositionIDsFilter",
}

type corpusCase struct {
	Command  string         `json:"command"`
	Expected map[string]any `json:"expected"`
}

func loadCorpus(t *testing.T) []corpusCase {
	t.Helper()
	raw, err := os.ReadFile("testdata/search_query_corpus.json")
	if err != nil {
		t.Fatalf("read corpus: %v", err)
	}
	var doc struct {
		Cases []corpusCase `json:"cases"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse corpus: %v", err)
	}
	if len(doc.Cases) == 0 {
		t.Fatal("corpus is empty")
	}
	return doc.Cases
}

func TestParseMatchesSharedCorpus(t *testing.T) {
	t.Parallel()
	for _, tc := range loadCorpus(t) {
		t.Run(tc.Command, func(t *testing.T) {
			t.Parallel()
			got, _ := Parse(tc.Command)
			v := reflect.ValueOf(got)
			for key, want := range tc.Expected {
				if key == "excludeStructure" {
					assertExcludeStructure(t, tc.Command, want)
					continue
				}
				field, ok := corpusField[key]
				if !ok {
					t.Fatalf("corpus key %q has no Go field mapping: add it to corpusField, or explain in the package doc why the Go grammar cannot express it", key)
				}
				fv := v.FieldByName(field)
				if !fv.IsValid() {
					t.Fatalf("domain.SearchFilters has no field %q (mapped from corpus key %q)", field, key)
				}
				assertField(t, key, fv, want)
			}
		})
	}
}

// assertExcludeStructure checks the `x` token, which carries no filter here: it
// switches on a board the interface holds, so Parse reports it as a no-effect
// diagnostic instead of silently dropping it.
func assertExcludeStructure(t *testing.T, command string, want any) {
	t.Helper()
	if want != true {
		return
	}
	_, diags := Parse(command)
	for _, d := range diags {
		if d.Token == "x" && d.Kind == DiagNoEffect {
			return
		}
	}
	t.Errorf("`x` in %q produced no no-effect diagnostic; diagnostics were %v", command, diags)
}

func assertField(t *testing.T, key string, fv reflect.Value, want any) {
	t.Helper()
	switch fv.Kind() {
	case reflect.Bool:
		wantBool, ok := want.(bool)
		if !ok {
			t.Fatalf("corpus key %q expects %T, field is bool", key, want)
		}
		if fv.Bool() != wantBool {
			t.Errorf("%s = %v, want %v", key, fv.Bool(), wantBool)
		}
	case reflect.String:
		wantStr, ok := want.(string)
		if !ok {
			t.Fatalf("corpus key %q expects %T, field is string", key, want)
		}
		if fv.String() != wantStr {
			t.Errorf("%s = %q, want %q", key, fv.String(), wantStr)
		}
	default:
		t.Fatalf("corpus key %q maps to an unsupported field kind %s", key, fv.Kind())
	}
}

// TestParseLeavesUnmentionedFieldsAtZero pins the corpus's own convention: a
// case lists only the fields it is about, and every other field must keep its
// zero value. Without this, a rule that fired on the wrong token would go
// unnoticed as long as the token it stole from was not in `expected`.
func TestParseLeavesUnmentionedFieldsAtZero(t *testing.T) {
	t.Parallel()
	for _, tc := range loadCorpus(t) {
		t.Run(tc.Command, func(t *testing.T) {
			t.Parallel()
			got, _ := Parse(tc.Command)
			v := reflect.ValueOf(got)
			mentioned := map[string]bool{}
			for key := range tc.Expected {
				if field, ok := corpusField[key]; ok {
					mentioned[field] = true
				}
			}
			// DiceRollMode defaults to "both" whether or not dice are filtered,
			// mirroring the JS; it is never "unset".
			mentioned["DiceRollMode"] = true
			for i := 0; i < v.NumField(); i++ {
				name := v.Type().Field(i).Name
				if mentioned[name] {
					continue
				}
				fv := v.Field(i)
				if fv.Kind() != reflect.Bool && fv.Kind() != reflect.String {
					continue
				}
				if !fv.IsZero() {
					t.Errorf("%q set %s to %v, but the corpus case does not mention it", tc.Command, name, fv.Interface())
				}
			}
		})
	}
}

// TestFormatRoundTripsCorpus checks the other direction on real cases: a parsed
// command, formatted and parsed again, yields the same filters. Format's token
// order is canonical, so this also proves two equal filter sets format alike.
func TestFormatRoundTripsCorpus(t *testing.T) {
	t.Parallel()
	for _, tc := range loadCorpus(t) {
		t.Run(tc.Command, func(t *testing.T) {
			t.Parallel()
			first, _ := Parse(tc.Command)
			formatted := Format(first)
			second, diags := Parse(formatted)
			for _, d := range diags {
				if d.Kind == DiagUnknown {
					t.Errorf("Format(%q) = %q, which Parse cannot read back: %v", tc.Command, formatted, d)
				}
			}
			if !reflect.DeepEqual(first, second) {
				t.Errorf("round trip changed the filters\n  command   %q\n  formatted %q\n  before    %+v\n  after     %+v", tc.Command, formatted, first, second)
			}
			if again := Format(second); again != formatted {
				t.Errorf("Format is not idempotent: %q then %q", formatted, again)
			}
		})
	}
}

var _ = domain.SearchFilters{}
