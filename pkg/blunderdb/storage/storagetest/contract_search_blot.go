package storagetest

import (
	"context"
	"errors"
	"math/rand"
	"slices"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testSearchBlotFilters pins the four blot filters to the board: each answers
// the positions whose decoded board passes the filter, in id order, whichever
// way the backend selects them. An unreadable filter is refused, not answered
// by an empty result after reading every position.
func testSearchBlotFilters(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	rng := rand.New(rand.NewSource(7))
	for i := 0; i < 60; i++ {
		p := domain.InitializePosition()
		p.DecisionType = domain.CheckerAction
		p.Dice = [2]int{1 + i%6, 1 + (i/6)%6}
		left := [2]int{15, 15}
		for pt := 1; pt <= 24; pt++ {
			p.Board.Points[pt] = domain.Point{Color: -1}
			color := rng.Intn(3) - 1
			if color < 0 || left[color] == 0 {
				continue
			}
			n := 1
			if rng.Intn(3) == 0 {
				n = min(2+rng.Intn(2), left[color])
			}
			left[color] -= n
			p.Board.Points[pt] = domain.Point{Color: color, Checkers: n}
		}
		if _, err := s.Positions().Save(ctx, "", &p); err != nil {
			t.Fatalf("Save position %d: %v", i, err)
		}
	}
	var all []domain.Position
	for pos, err := range s.Search().Find(ctx, "", domain.SearchFilters{}, storage.ListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		all = append(all, *pos)
	}
	if len(all) < 50 {
		t.Fatalf("%d positions stored, want the 60 boards", len(all))
	}

	type field struct {
		prefix string
		set    func(*domain.SearchFilters, string)
		match  func(*domain.Position, string) bool
	}
	fields := []field{
		{"bo", func(f *domain.SearchFilters, v string) { f.Player1OutfieldBlotFilter = v }, (*domain.Position).MatchesPlayer1OutfieldBlot},
		{"BO", func(f *domain.SearchFilters, v string) { f.Player2OutfieldBlotFilter = v }, (*domain.Position).MatchesPlayer2OutfieldBlot},
		{"bj", func(f *domain.SearchFilters, v string) { f.Player1JanBlotFilter = v }, (*domain.Position).MatchesPlayer1JanBlot},
		{"BJ", func(f *domain.SearchFilters, v string) { f.Player2JanBlotFilter = v }, (*domain.Position).MatchesPlayer2JanBlot},
	}
	nonEmpty := 0
	for _, fd := range fields {
		for _, tail := range []string{">0", ">1", ">2", ">3", ">4", ">5", ">9", "<0", "<1", "<3", "1,2", "3,1", "2,2", "4,6"} {
			filter := fd.prefix + tail
			var f domain.SearchFilters
			fd.set(&f, filter)
			var want []int64
			for i := range all {
				if fd.match(&all[i], filter) {
					want = append(want, all[i].ID)
				}
			}
			got := searchIDs(t, s, f)
			if !slices.Equal(got, want) {
				t.Errorf("%s: got %v, want %v", filter, got, want)
			}
			if len(want) > 0 && len(want) < len(all) {
				nonEmpty++
			}
		}
		for _, bad := range []string{fd.prefix + "3", fd.prefix + ">x", fd.prefix + "1,2,3", fd.prefix, fd.prefix + "<"} {
			var f domain.SearchFilters
			fd.set(&f, bad)
			var err error
			for _, e := range s.Search().Find(ctx, "", f, storage.ListOpts{}) {
				err = e
				break
			}
			if !errors.Is(err, storage.ErrInvalid) {
				t.Errorf("%q: got %v, want storage.ErrInvalid", bad, err)
			}
		}
	}
	if nonEmpty < 20 {
		t.Errorf("only %d filters select some positions and not all: the boards do not exercise the filters", nonEmpty)
	}

	// Two filters at once, one per side.
	both := domain.SearchFilters{Player1OutfieldBlotFilter: "bo>2", Player2JanBlotFilter: "BJ>1"}
	var want []int64
	for i := range all {
		if all[i].MatchesPlayer1OutfieldBlot("bo>2") && all[i].MatchesPlayer2JanBlot("BJ>1") {
			want = append(want, all[i].ID)
		}
	}
	if got := searchIDs(t, s, both); !slices.Equal(got, want) {
		t.Errorf("bo>2 with BJ>1: got %v, want %v", got, want)
	}
}
