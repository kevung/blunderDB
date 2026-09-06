// Contract cases for the tag vocabulary and the delimited tag search.
// The table that runs them lives in contract.go.
package storagetest

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testTagVocabulary pins what a tag is worth counting: POSITIONS, not
// comments (#265). A tag written twice on one position is one position
// tagged, and the number shown beside a tag in the panel has to be the number
// of positions clicking it will yield — otherwise the panel promises more than
// the search delivers.
func testTagVocabulary(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	cs := s.Comments()

	first := statsDecisionPos(t, 0)
	firstID, err := s.Positions().Save(ctx, "", &first)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	second := statsDecisionPos(t, 1)
	secondID, err := s.Positions().Save(ctx, "", &second)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}

	// Two comments on the same position, both carrying #prime: one position.
	if _, err := cs.Add(ctx, "", firstID, "trop passif ici #prime"); err != nil {
		if errors.Is(err, storage.ErrInternal) {
			t.Skip("Comments not implemented on this backend")
		}
		t.Fatalf("Add comment: %v", err)
	}
	if _, err := cs.Add(ctx, "", firstID, "et encore #prime #cube"); err != nil {
		t.Fatalf("Add comment: %v", err)
	}
	if _, err := cs.Add(ctx, "", secondID, "ici c'est du #priming, pas du prime"); err != nil {
		t.Fatalf("Add comment: %v", err)
	}

	tags, err := cs.Tags(ctx, "")
	if err != nil {
		t.Fatalf("Tags: %v", err)
	}
	got := map[string]int{}
	for _, tc := range tags {
		got[tc.Tag] = tc.Count
	}
	for tag, want := range map[string]int{"#prime": 1, "#cube": 1, "#priming": 1} {
		if got[tag] != want {
			t.Errorf("tag %s counted %d position(s), want %d — got %v", tag, got[tag], want, got)
		}
	}

	// Most used first: the panel is read to find the tags one actually uses.
	for i := 1; i < len(tags); i++ {
		if tags[i-1].Count < tags[i].Count {
			t.Fatalf("tags are not ordered by count: %v", tags)
		}
	}
}

// testTagSearchIsDelimited is the reason a tag filter exists at all: the
// free-text comment search is a substring search, so `t"#prime"` cannot tell
// #prime from #priming. The tag filter extracts the comment's tags and
// compares them whole (#265).
func testTagSearchIsDelimited(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	primed := statsDecisionPos(t, 2)
	primedID, err := s.Positions().Save(ctx, "", &primed)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	priming := statsDecisionPos(t, 3)
	primingID, err := s.Positions().Save(ctx, "", &priming)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	both := statsDecisionPos(t, 4)
	bothID, err := s.Positions().Save(ctx, "", &both)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}

	if _, err := s.Comments().Add(ctx, "", primedID, "#prime"); err != nil {
		if errors.Is(err, storage.ErrInternal) {
			t.Skip("Comments not implemented on this backend")
		}
		t.Fatalf("Add comment: %v", err)
	}
	if _, err := s.Comments().Add(ctx, "", primingID, "#priming"); err != nil {
		t.Fatalf("Add comment: %v", err)
	}
	if _, err := s.Comments().Add(ctx, "", bothID, "#prime #cube"); err != nil {
		t.Fatalf("Add comment: %v", err)
	}

	find := func(filter string) map[int64]bool {
		t.Helper()
		out := map[int64]bool{}
		for pos, err := range s.Search().Find(ctx, "", domain.SearchFilters{TagFilter: filter}, storage.ListOpts{}) {
			if err != nil {
				t.Fatalf("Find(%q): %v", filter, err)
			}
			out[pos.ID] = true
		}
		return out
	}

	if got := find("#prime"); !got[primedID] || !got[bothID] || got[primingID] {
		t.Errorf("#prime matched %v; want the two #prime positions and NOT the #priming one", got)
	}
	if got := find("#priming"); !got[primingID] || got[primedID] || got[bothID] {
		t.Errorf("#priming matched %v; want only the #priming position", got)
	}
	// Two tags narrow together: a position has many tags, so naming two means
	// "both" — unlike the phase and origin lists, where naming two can only
	// mean "either".
	if got := find("#prime;#cube"); !got[bothID] || got[primedID] {
		t.Errorf("#prime #cube matched %v; want only the position carrying both", got)
	}
	// Case and a missing '#' are spellings of the same tag, not other tags.
	if got := find("Prime"); !got[primedID] || !got[bothID] {
		t.Errorf("the tag filter should normalise \"Prime\" to \"#prime\"; matched %v", got)
	}
}
