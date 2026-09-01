package searchfilter

import (
	"strings"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

func TestParseFilterIDList(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    []int64
		wantErr bool
	}{
		{"empty", "", nil, false},
		{"single", "5", []int64{5}, false},
		{"range", "2,7", []int64{2, 3, 4, 5, 6, 7}, false},
		{"three-item comma list", "1,3,5", []int64{1, 3, 5}, false},
		{"semicolon list", "2;5;9", []int64{2, 5, 9}, false},
		{"mixed comma and semicolon", "1,3;4,5", []int64{1, 3, 4, 5}, false},
		{"invalid token", "1,abc,3", nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseFilterIDList(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseFilterIDList(%q): expected error, got %v", tc.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseFilterIDList(%q): unexpected error: %v", tc.input, err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("ParseFilterIDList(%q) = %v, want %v", tc.input, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("ParseFilterIDList(%q) = %v, want %v", tc.input, got, tc.want)
				}
			}
		})
	}
}

func TestParseIntFilterExpr(t *testing.T) {
	cases := []struct {
		input          string
		min, max       int
		hasMin, hasMax bool
	}{
		{"p>5", 5, 0, true, false},
		{"p<12", 0, 12, false, true},
		{"p7", 7, 7, true, true},
		{"p3,9", 3, 9, true, true},
		{"p9,3", 3, 9, true, true},
		{"p", 0, 0, false, false},
		{"pabc", 0, 0, false, false},
		{"q5", 0, 0, false, false},
	}
	for _, tc := range cases {
		mn, mx, hasMin, hasMax := ParseIntFilterExpr(tc.input, "p")
		if mn != tc.min || mx != tc.max || hasMin != tc.hasMin || hasMax != tc.hasMax {
			t.Errorf("ParseIntFilterExpr(%q) = (%d, %d, %v, %v), want (%d, %d, %v, %v)",
				tc.input, mn, mx, hasMin, hasMax, tc.min, tc.max, tc.hasMin, tc.hasMax)
		}
	}
}

func TestAppendIntRangeSQL(t *testing.T) {
	cases := []struct {
		name           string
		min, max       int
		hasMin, hasMax bool
		wantSQL        string
		wantArgs       []any
	}{
		{"none", 0, 0, false, false, "", nil},
		{"exact", 4, 4, true, true, " AND c = ?", []any{4}},
		{"between", 2, 6, true, true, " AND c BETWEEN ? AND ?", []any{2, 6}},
		{"min", 2, 0, true, false, " AND c >= ?", []any{2}},
		{"max", 0, 6, false, true, " AND c <= ?", []any{6}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var where strings.Builder
			var args []any
			AppendIntRangeSQL("c", tc.min, tc.max, tc.hasMin, tc.hasMax, &where, &args)
			if where.String() != tc.wantSQL {
				t.Fatalf("sql = %q, want %q", where.String(), tc.wantSQL)
			}
			if len(args) != len(tc.wantArgs) {
				t.Fatalf("args = %v, want %v", args, tc.wantArgs)
			}
			for i := range args {
				if args[i] != tc.wantArgs[i] {
					t.Fatalf("args = %v, want %v", args, tc.wantArgs)
				}
			}
		})
	}
}

func TestParseSearchTextKeywords(t *testing.T) {
	cases := []struct {
		input string
		want  []string
	}{
		{`t"Prime; Blitz ;"`, []string{"prime", "blitz"}},
		{`t'x'`, []string{"x"}},
		{`"a;b"`, []string{"a", "b"}},
		{`t""`, nil},
		{`   `, nil},
	}
	for _, tc := range cases {
		got := ParseSearchTextKeywords(tc.input)
		if len(got) != len(tc.want) {
			t.Fatalf("ParseSearchTextKeywords(%q) = %v, want %v", tc.input, got, tc.want)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("ParseSearchTextKeywords(%q) = %v, want %v", tc.input, got, tc.want)
			}
		}
	}
}

func TestMatchesMoveError(t *testing.T) {
	cases := []struct {
		millipoints float64
		filter      string
		want        bool
	}{
		{50, "E>20", true},
		{20, "E>20", true},
		{10, "E>20", false},
		{10, "E<20", true},
		{30, "E<20", false},
		{15, "E10,20", true},
		{15, "E20,10", true},
		{25, "E10,20", false},
		{25, "E10", false},
		{25, "E>abc", false},
		{25, "x>1", false},
	}
	for _, tc := range cases {
		if got := MatchesMoveError(tc.millipoints, tc.filter); got != tc.want {
			t.Errorf("MatchesMoveError(%v, %q) = %v, want %v", tc.millipoints, tc.filter, got, tc.want)
		}
	}
}

func TestMatchesDateFilter(t *testing.T) {
	created := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	ana := &domain.PositionAnalysis{CreationDate: created}
	cases := []struct {
		filter string
		want   bool
	}{
		{"T>2024/06/15", true},
		{"T>2024/06/16", false},
		{"T<2024/06/15", true},
		{"T<2024/06/14", false},
		{"T2024/06/01,2024/06/30", true},
		{"T2024/06/30,2024/06/01", true},
		{"T2024/07/01,2024/07/31", false},
		{"T2024/06/01", false},
		{"Tnope", false},
	}
	for _, tc := range cases {
		if got := MatchesDateFilter(ana, tc.filter); got != tc.want {
			t.Errorf("MatchesDateFilter(%q) = %v, want %v", tc.filter, got, tc.want)
		}
	}
	if MatchesDateFilter(nil, "T>2024/06/15") {
		t.Error("MatchesDateFilter(nil) must be false")
	}
}
