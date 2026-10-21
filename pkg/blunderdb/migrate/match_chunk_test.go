package migrate

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// TestRunCopiesMatchesAcrossChunks copies more matches than one chunk holds
// and checks each keeps its own games and moves and each tournament its order.
func TestRunCopiesMatchesAcrossChunks(t *testing.T) {
	CheckCopiesMatchesAcrossChunks(t, openMemory(t))
}

// CheckCopiesMatchesAcrossChunks is TestRunCopiesMatchesAcrossChunks into dst,
// an empty destination of any backend. One tournament holds matches all at
// sort order 0 (the order then falls to the most recent first), the other an
// order set by ReorderMatches against the insertion order.
func CheckCopiesMatchesAcrossChunks(t *testing.T, dst storage.Storage) {
	t.Helper()
	ctx := context.Background()
	src := openMemory(t)
	ms := src.Matches()

	tiedID, err := src.Tournaments().Create(ctx, "", "Tied", "", "")
	if err != nil {
		t.Fatal(err)
	}
	orderedID, err := src.Tournaments().Create(ctx, "", "Ordered", "", "")
	if err != nil {
		t.Fatal(err)
	}
	var ordered []int64
	n := matchChunk + 5
	for i := range n {
		mt := domain.Match{Player1Name: fmt.Sprintf("p%03d", i), Player2Name: "x", MatchLength: 5,
			MatchDate: time.Date(2020, 1, 1+i%28, 0, 0, 0, 0, time.UTC)}
		if i%7 == 0 {
			mt.TournamentID = &tiedID
		}
		id, err := ms.Save(ctx, "", &mt)
		if err != nil {
			t.Fatalf("Save match %d: %v", i, err)
		}
		if i%5 == 1 {
			if err := src.Tournaments().AddMatch(ctx, "", orderedID, id); err != nil {
				t.Fatal(err)
			}
			ordered = append(ordered, id)
		}
		for gn := range 1 + i%3 {
			gid, err := ms.CreateGame(ctx, "", &domain.Game{MatchID: id, GameNumber: int32(gn + 1), PointsWon: int32(i)})
			if err != nil {
				t.Fatal(err)
			}
			for mn := range gn + 1 {
				if _, err := ms.CreateMove(ctx, "", &domain.Move{GameID: gid, MoveNumber: int32(mn + 1),
					MoveType: "checker", CheckerMove: fmt.Sprintf("%d.%d.%d", i, gn, mn)}); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	slices.Reverse(ordered)
	if err := src.Tournaments().ReorderMatches(ctx, "", orderedID, ordered); err != nil {
		t.Fatal(err)
	}

	if _, err := Run(ctx, src, dst, "", Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if got, want := matchShape(t, dst), matchShape(t, src); !slices.Equal(got, want) {
		t.Errorf("games and moves differ after the copy:\n got %v\nwant %v", got, want)
	}
	for _, name := range []string{"Tied", "Ordered"} {
		if got, want := tournamentOrder(t, dst, name), tournamentOrder(t, src, name); !slices.Equal(got, want) {
			t.Errorf("tournament %s lists %v, source %v", name, got, want)
		}
	}
}

// matchShape lists, match by match in name order, each game's number and
// points and each move's text: what a misplaced game or move would change.
func matchShape(t *testing.T, s storage.Storage) []string {
	t.Helper()
	ctx := context.Background()
	// Each stream is drained before the next opens: a :memory: SQLite holds
	// one connection.
	var matches []*domain.Match
	for mt, err := range s.Matches().List(ctx, "", storage.MatchListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		matches = append(matches, mt)
	}
	var out []string
	for _, mt := range matches {
		line := mt.Player1Name
		var games []*domain.Game
		for g, err := range s.Matches().Games(ctx, "", mt.ID) {
			if err != nil {
				t.Fatal(err)
			}
			games = append(games, g)
		}
		for _, g := range games {
			line += fmt.Sprintf(" g%d/%d[", g.GameNumber, g.PointsWon)
			for mv, err := range s.Matches().Moves(ctx, "", g.ID) {
				if err != nil {
					t.Fatal(err)
				}
				line += fmt.Sprintf(" %d:%s", mv.MoveNumber, mv.CheckerMove)
			}
			line += "]"
		}
		out = append(out, line)
	}
	slices.Sort(out)
	return out
}

func tournamentOrder(t *testing.T, s storage.Storage, name string) []string {
	t.Helper()
	ctx := context.Background()
	id := int64(0)
	for tr, err := range s.Tournaments().List(ctx, "", storage.ListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		if tr.Name == name {
			id = tr.ID
		}
	}
	if id != 0 {
		var out []string
		for mt, err := range s.Tournaments().Matches(ctx, "", id) {
			if err != nil {
				t.Fatal(err)
			}
			out = append(out, mt.Player1Name)
		}
		return out
	}
	t.Fatalf("no tournament %s", name)
	return nil
}
