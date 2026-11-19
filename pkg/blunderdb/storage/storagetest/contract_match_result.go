// Contract case for the stored result of a match: written with the match,
// read back by Get and List, kept by a header restated without it, turned
// with the seats.
package storagetest

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func testMatchResult(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ms := s.Matches()
	won := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7,
		FinalScore: &[2]int32{7, 4}, Winner: domain.WinnerPlayer1}
	wonID, err := ms.Save(ctx, "", &won)
	if err != nil {
		t.Fatal(err)
	}
	bare := domain.Match{Player1Name: "Carol", Player2Name: "Dan", MatchLength: 5}
	bareID, err := ms.Save(ctx, "", &bare)
	if err != nil {
		t.Fatal(err)
	}

	results := func() map[int64]domain.Match {
		out := map[int64]domain.Match{}
		for m, err := range ms.List(ctx, "", storage.MatchListOpts{}) {
			if err != nil {
				t.Fatal(err)
			}
			out[m.ID] = *m
		}
		return out
	}
	listed := results()
	if m := listed[wonID]; m.FinalScore == nil || *m.FinalScore != [2]int32{7, 4} || m.Winner != domain.WinnerPlayer1 {
		t.Errorf("listed result = %v, winner %d; want 7-4 won by player 1", m.FinalScore, m.Winner)
	}
	if m := listed[bareID]; m.FinalScore != nil || m.Winner != domain.WinnerUnfinished {
		t.Errorf("match without games: result %v, winner %d; want none", m.FinalScore, m.Winner)
	}

	header := won
	header.FinalScore, header.Winner = nil, domain.WinnerUnfinished
	header.Event = "Restated"
	if err := ms.ReplaceHeader(ctx, "", wonID, &header); err != nil {
		t.Fatal(err)
	}
	if got, err := ms.Get(ctx, "", wonID); err != nil || got.FinalScore == nil || *got.FinalScore != [2]int32{7, 4} || got.Winner != domain.WinnerPlayer1 {
		t.Errorf("a header without result: %v; want the stored 7-4 kept", got)
	}
	header.FinalScore, header.Winner = &[2]int32{6, 7}, domain.WinnerPlayer2
	if err := ms.ReplaceHeader(ctx, "", wonID, &header); err != nil {
		t.Fatal(err)
	}
	if got, err := ms.Get(ctx, "", wonID); err != nil || got.FinalScore == nil || *got.FinalScore != [2]int32{6, 7} || got.Winner != domain.WinnerPlayer2 {
		t.Errorf("a header with result: %v; want 6-7 won by player 2", got)
	}

	if err := ms.SwapPlayers(ctx, "", wonID); err != nil {
		t.Fatal(err)
	}
	if got, err := ms.Get(ctx, "", wonID); err != nil || got.FinalScore == nil || *got.FinalScore != [2]int32{7, 6} || got.Winner != domain.WinnerPlayer1 {
		t.Errorf("swapped seats: %v; want 7-6 won by player 1", got)
	}
}
