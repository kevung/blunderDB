package storagetest

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testTrainingStatsAndStudy runs the shared study logic on both backends: the
// training series reads the three journals into calendar windows, and the
// positions of "my worst groups" become a quiz draw or a search deck.
func testTrainingStatsAndStudy(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	if _, err := s.Stats().DateRange(ctx, ""); errors.Is(err, storage.ErrInternal) {
		t.Skip("Stats not implemented on this backend")
	}
	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7,
		MatchDate: time.Date(2025, 6, 4, 0, 0, 0, 0, time.UTC)} // a Wednesday
	matchID, err := s.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	gameID, err := s.Matches().CreateGame(ctx, "", &domain.Game{MatchID: matchID, GameNumber: 1})
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	gammonID := statsCheckerDecision(t, s, gameID, 0, 1, "24/18 13/11", 0.200, 8)
	if _, err := s.Training().Save(ctx, "", storage.TrainingSession{Exercise: "decision", SeedSource: "library", NumbersAsked: 4}); err != nil {
		t.Fatalf("Save training session: %v", err)
	}

	filter := storage.StatsFilter{DecisionType: -1, PlayerName: "Alice"}
	got, err := storage.ComputeTrainingStats(ctx, s, "", filter, storage.TrainingWindowWeek)
	if err != nil {
		t.Fatalf("ComputeTrainingStats: %v", err)
	}
	var matchDecisions, quizDecisions int
	var monday string
	for _, p := range got.Periods {
		matchDecisions += p.MatchDecisions
		quizDecisions += p.QuizDecisions
		if p.MatchDecisions > 0 {
			monday = p.Start
		}
	}
	if matchDecisions == 0 || monday != "2025-06-02" {
		t.Errorf("match decisions = %d in week %q, want some in 2025-06-02", matchDecisions, monday)
	}
	if quizDecisions != 4 || len(got.Sessions) != 1 {
		t.Errorf("quiz decisions = %d, sessions = %d, want 4 and 1", quizDecisions, len(got.Sessions))
	}

	// The targeted series gives the windows the full Compute gave, for every
	// filter: the provenance ones and the decision types included.
	sessions, err := s.Training().Sessions(ctx, "", "decision", 0)
	if err != nil {
		t.Fatalf("Sessions: %v", err)
	}
	for _, f := range []storage.StatsFilter{
		filter,
		{DecisionType: 0},
		{DecisionType: 1, PlayerName: "Alice"},
		{DecisionType: -1, MinAnalysisDepth: 2},
		{DecisionType: -1, AnalysisEngine: "GNU"},
	} {
		got, err := storage.ComputeTrainingStats(ctx, s, "", f, storage.TrainingWindowMonth)
		if err != nil {
			t.Fatalf("ComputeTrainingStats %+v: %v", f, err)
		}
		full, err := s.Stats().Compute(ctx, "", f)
		if err != nil {
			t.Fatalf("Compute %+v: %v", f, err)
		}
		want := storage.BuildTrainingStats(storage.TrainingWindowMonth, sessions, full.PerMatch, nil)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("filter %+v:\n targeted %+v\n full     %+v", f, got, want)
		}
	}

	ids, err := storage.StudyIDs(ctx, s, "", filter, 0, 0)
	if err != nil {
		t.Fatalf("StudyIDs: %v", err)
	}
	if len(ids) != 1 || ids[0] != gammonID {
		t.Fatalf("StudyIDs = %v, want [%d]", ids, gammonID)
	}
	deckID, err := storage.CreateStudyDeck(ctx, s, "", "Worst", ids)
	if err != nil || deckID == 0 {
		t.Fatalf("CreateStudyDeck = %d, %v", deckID, err)
	}
	var inDeck int
	for _, err := range s.Anki().DeckPositions(ctx, "", deckID) {
		if err != nil {
			t.Fatalf("DeckPositions: %v", err)
		}
		inDeck++
	}
	if inDeck != 1 {
		t.Errorf("deck holds %d positions, want 1", inDeck)
	}
}
