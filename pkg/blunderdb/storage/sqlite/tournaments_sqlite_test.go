package sqlite_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// TestTournamentCRUD covers Create, Get, List, Update, UpdateComment, Delete.
func TestTournamentCRUD(t *testing.T) {
	ctx := context.Background()
	s := openMem(t)

	id, err := s.Tournaments().Create(ctx, "", "Worlds 2025", "2025-07-01", "Monaco")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if id == 0 {
		t.Fatal("Create returned id 0")
	}

	got, err := s.Tournaments().Get(ctx, "", id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "Worlds 2025" || got.Date != "2025-07-01" || got.Location != "Monaco" {
		t.Errorf("Get: %+v", got)
	}
	if got.MatchCount != 0 {
		t.Errorf("Get MatchCount: got %d, want 0", got.MatchCount)
	}

	if err := s.Tournaments().Update(ctx, "", id, "Worlds", "2025-07-02", "Nice"); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if err := s.Tournaments().UpdateComment(ctx, "", id, "great event"); err != nil {
		t.Fatalf("UpdateComment: %v", err)
	}
	got, _ = s.Tournaments().Get(ctx, "", id)
	if got.Name != "Worlds" || got.Location != "Nice" || got.Comment != "great event" {
		t.Errorf("after Update: %+v", got)
	}

	if _, err := s.Tournaments().Create(ctx, "", "Other", "2024-01-01", ""); err != nil {
		t.Fatalf("Create second: %v", err)
	}
	n := 0
	for _, err := range s.Tournaments().List(ctx, "") {
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		n++
	}
	if n != 2 {
		t.Errorf("List count: got %d, want 2", n)
	}

	if err := s.Tournaments().Delete(ctx, "", id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Tournaments().Get(ctx, "", id); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("after Delete Get: got %v, want ErrNotFound", err)
	}
}

// TestTournamentMatchMembership covers AddMatch, RemoveMatch, Matches,
// TournamentOf, ReorderMatches and that Delete unlinks rather than deletes
// matches.
func TestTournamentMatchMembership(t *testing.T) {
	ctx := context.Background()
	s := openMem(t)

	tID, err := s.Tournaments().Create(ctx, "", "Cup", "2025-01-01", "")
	if err != nil {
		t.Fatalf("Create tournament: %v", err)
	}

	m1 := domain.Match{Player1Name: "A", Player2Name: "B"}
	id1, _ := s.Matches().Save(ctx, "", &m1)
	m2 := domain.Match{Player1Name: "C", Player2Name: "D"}
	id2, _ := s.Matches().Save(ctx, "", &m2)

	if err := s.Tournaments().AddMatch(ctx, "", tID, id1); err != nil {
		t.Fatalf("AddMatch id1: %v", err)
	}
	if err := s.Tournaments().AddMatch(ctx, "", tID, id2); err != nil {
		t.Fatalf("AddMatch id2: %v", err)
	}

	got, _ := s.Tournaments().Get(ctx, "", tID)
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

	// Reorder so id2 comes first, then verify order.
	if err := s.Tournaments().ReorderMatches(ctx, "", tID, []int64{id2, id1}); err != nil {
		t.Fatalf("ReorderMatches: %v", err)
	}
	var order []int64
	for m, err := range s.Tournaments().Matches(ctx, "", tID) {
		if err != nil {
			t.Fatalf("Matches: %v", err)
		}
		order = append(order, m.ID)
	}
	if len(order) != 2 || order[0] != id2 || order[1] != id1 {
		t.Errorf("Matches order: got %v, want [%d %d]", order, id2, id1)
	}

	if err := s.Tournaments().RemoveMatch(ctx, "", id1); err != nil {
		t.Fatalf("RemoveMatch: %v", err)
	}
	if _, err := s.Tournaments().TournamentOf(ctx, "", id1); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("TournamentOf after RemoveMatch: got %v, want ErrNotFound", err)
	}

	// Deleting the tournament unlinks the remaining match but keeps it.
	if err := s.Tournaments().Delete(ctx, "", tID); err != nil {
		t.Fatalf("Delete tournament: %v", err)
	}
	if _, err := s.Matches().Get(ctx, "", id2); err != nil {
		t.Errorf("match deleted with tournament: %v", err)
	}
}

// TestTournamentSetMatchByName covers find-or-create and detach semantics.
func TestTournamentSetMatchByName(t *testing.T) {
	ctx := context.Background()
	s := openMem(t)

	m := domain.Match{Player1Name: "A", Player2Name: "B"}
	matchID, _ := s.Matches().Save(ctx, "", &m)

	// Unknown name → tournament created and linked.
	if err := s.Tournaments().SetMatchByName(ctx, "", matchID, "Spring Open"); err != nil {
		t.Fatalf("SetMatchByName create: %v", err)
	}
	of, err := s.Tournaments().TournamentOf(ctx, "", matchID)
	if err != nil {
		t.Fatalf("TournamentOf: %v", err)
	}
	if of.Name != "Spring Open" {
		t.Errorf("created tournament name: got %q", of.Name)
	}

	// Same name again → reuse, no duplicate.
	if err := s.Tournaments().SetMatchByName(ctx, "", matchID, "Spring Open"); err != nil {
		t.Fatalf("SetMatchByName reuse: %v", err)
	}
	n := 0
	for _, err := range s.Tournaments().List(ctx, "") {
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		n++
	}
	if n != 1 {
		t.Errorf("tournament count after reuse: got %d, want 1", n)
	}

	// Empty name → detach.
	if err := s.Tournaments().SetMatchByName(ctx, "", matchID, ""); err != nil {
		t.Fatalf("SetMatchByName detach: %v", err)
	}
	if _, err := s.Tournaments().TournamentOf(ctx, "", matchID); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("TournamentOf after detach: got %v, want ErrNotFound", err)
	}
}
