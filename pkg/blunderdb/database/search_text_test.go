package database

// search_text_test.go — regression tests for the t"tag1;tag2;..." "Search Text"
// filter that matches keywords against position comments.
//
// Previously the keyword parser used strings.Trim(searchText, ` t"'`) and a raw
// split on ';', which had three bugs:
//   - a tag typed with surrounding spaces ("tag1; tag2") never matched;
//   - a stray trailing ';' produced an empty tag that matched every comment;
//   - a leading tag starting with 't' had its 't' eaten by the trim.

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseSearchTextKeywords(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{`t"blitz"`, []string{"blitz"}},
		{`t"blitz;prime"`, []string{"blitz", "prime"}},
		{`t"blitz; prime"`, []string{"blitz", "prime"}},   // space after ';'
		{`t"prime ;blitz"`, []string{"prime", "blitz"}},   // space before ';'
		{`t"timing"`, []string{"timing"}},                 // leading 't' tag preserved
		{`t"timing;take"`, []string{"timing", "take"}},    // both 't' tags preserved
		{`t"blitz;"`, []string{"blitz"}},                  // trailing ';' drops empty tag
		{`t"  blitz  "`, []string{"blitz"}},               // surrounding spaces
		{`t""`, nil},                                      // empty wrapper -> no keywords
		{`blitz`, []string{"blitz"}},                      // unwrapped raw value
	}
	for _, tc := range cases {
		got := parseSearchTextKeywords(tc.in)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("parseSearchTextKeywords(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestSearchTextFilter(t *testing.T) {
	dir := t.TempDir()
	db := NewDatabase()
	if err := db.SetupDatabase(filepath.Join(dir, "search_text.db")); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	defer db.Close()

	pos1 := InitializePosition()
	id1, err := db.SavePosition(&pos1)
	if err != nil {
		t.Fatalf("SavePosition 1: %v", err)
	}
	pos2 := InitializePosition()
	pos2.Dice = [2]int{6, 5}
	id2, err := db.SavePosition(&pos2)
	if err != nil {
		t.Fatalf("SavePosition 2: %v", err)
	}

	if err := db.SaveComment(id1, "tags:blitz,prime,timing"); err != nil {
		t.Fatalf("SaveComment 1: %v", err)
	}
	if err := db.SaveComment(id2, "tags:race,bearoff"); err != nil {
		t.Fatalf("SaveComment 2: %v", err)
	}
	// A position may have several comment entries; the filter must match any of
	// them, not only the first one (regression: loadCommentText read one row).
	if err := db.AddComment(id1, "toto"); err != nil {
		t.Fatalf("AddComment 1b: %v", err)
	}
	if err := db.AddComment(id1, "cocorico"); err != nil {
		t.Fatalf("AddComment 1c: %v", err)
	}

	cases := []struct {
		searchText string
		want       []int64
	}{
		{`t"blitz"`, []int64{id1}},
		{`t"blitz;prime"`, []int64{id1}},
		{`t"race"`, []int64{id2}},
		{`t"timing"`, []int64{id1}},        // leading 't' tag must still match
		{`t"foo; timing"`, []int64{id1}},   // space after ';' must not break the tag
		{`t"blitz; race"`, []int64{id1, id2}},
		{`t"blitz;"`, []int64{id1}},        // trailing ';' must not match everything
		{`t"foo;bar"`, nil},                // no tag present
		{`t"toto"`, []int64{id1}},          // 2nd comment entry of a position
		{`t"cocorico"`, []int64{id1}},      // 3rd comment entry of a position
	}
	for _, tc := range cases {
		positions, err := db.LoadPositionsByFilters(SearchFilters{SearchText: tc.searchText})
		if err != nil {
			t.Fatalf("LoadPositionsByFilters(%q): %v", tc.searchText, err)
		}
		var got []int64
		for _, p := range positions {
			got = append(got, p.ID)
		}
		if !sameIDSet(got, tc.want) {
			t.Errorf("searchText %q: got positions %v, want %v", tc.searchText, got, tc.want)
		}
	}
}

func sameIDSet(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[int64]int, len(a))
	for _, id := range a {
		seen[id]++
	}
	for _, id := range b {
		seen[id]--
	}
	for _, n := range seen {
		if n != 0 {
			return false
		}
	}
	return true
}
