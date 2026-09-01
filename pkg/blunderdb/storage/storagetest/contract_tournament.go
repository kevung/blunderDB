// Contract cases for tournaments: attaching and detaching matches.
// The table that runs them lives in contract.go.
package storagetest

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func testTournamentAddRemoveMatch(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	tID, err := s.Tournaments().Create(ctx, "", "Cup", "2025-01-01", "Paris")
	if err != nil {
		t.Fatalf("Create tournament: %v", err)
	}

	m1 := domain.Match{Player1Name: "A", Player2Name: "B"}
	id1, err := s.Matches().Save(ctx, "", &m1)
	if err != nil {
		t.Fatalf("Save match 1: %v", err)
	}
	m2 := domain.Match{Player1Name: "C", Player2Name: "D"}
	id2, err := s.Matches().Save(ctx, "", &m2)
	if err != nil {
		t.Fatalf("Save match 2: %v", err)
	}

	if err := s.Tournaments().AddMatch(ctx, "", tID, id1); err != nil {
		t.Fatalf("AddMatch 1: %v", err)
	}
	if err := s.Tournaments().AddMatch(ctx, "", tID, id2); err != nil {
		t.Fatalf("AddMatch 2: %v", err)
	}

	got, err := s.Tournaments().Get(ctx, "", tID)
	if err != nil {
		t.Fatalf("Get tournament: %v", err)
	}
	if got.MatchCount != 2 {
		t.Errorf("MatchCount after AddMatch: got %d, want 2", got.MatchCount)
	}

	of, err := s.Tournaments().TournamentOf(ctx, "", id1)
	if err != nil {
		t.Fatalf("TournamentOf: %v", err)
	}
	if of.ID != tID {
		t.Errorf("TournamentOf: got %d, want %d", of.ID, tID)
	}

	var matchIDs []int64
	for m, err := range s.Tournaments().Matches(ctx, "", tID) {
		if err != nil {
			t.Fatalf("Matches: %v", err)
		}
		matchIDs = append(matchIDs, m.ID)
	}
	if len(matchIDs) != 2 {
		t.Fatalf("Matches count: got %d, want 2", len(matchIDs))
	}

	if err := s.Tournaments().RemoveMatch(ctx, "", id1); err != nil {
		t.Fatalf("RemoveMatch: %v", err)
	}
	if _, err := s.Tournaments().TournamentOf(ctx, "", id1); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("TournamentOf after RemoveMatch: got %v, want ErrNotFound", err)
	}
	got, err = s.Tournaments().Get(ctx, "", tID)
	if err != nil {
		t.Fatalf("Get tournament after remove: %v", err)
	}
	if got.MatchCount != 1 {
		t.Errorf("MatchCount after RemoveMatch: got %d, want 1", got.MatchCount)
	}
}
