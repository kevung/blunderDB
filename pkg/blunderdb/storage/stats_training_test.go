package storage

import (
	"math"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

func TestBuildTrainingStats_WeeksAlignQuizMatchesAndAnki(t *testing.T) {
	sessions := []TrainingSession{
		{ID: 1, Exercise: "decision", CreatedAt: "2026-10-05 09:00:00", NumbersAsked: 10, PR: 8},
		{ID: 2, Exercise: "decision", CreatedAt: "2026-10-11 20:00:00", NumbersAsked: 30, PR: 4},
		{ID: 3, Exercise: "pips", CreatedAt: "2026-10-06 09:00:00", NumbersAsked: 5, PR: 0},
		{ID: 4, Exercise: "decision", CreatedAt: "2026-10-12 09:00:00", NumbersAsked: 10, PR: 2},
	}
	matches := []MatchStats{
		{Date: "2026-10-07", PR: 6, NumDecisions: 100},
		{Date: "2026-10-12", PR: 3, NumDecisions: 50},
	}
	reviews := []domain.AnkiReviewLog{
		{State: 2, Rating: 3, ReviewedAt: "2026-10-06T10:00:00Z"},
		{State: 2, Rating: 1, ReviewedAt: "2026-10-07 10:00:00"},
		{State: 0, Rating: 3, ReviewedAt: "2026-10-07 11:00:00"},
	}
	got := BuildTrainingStats(TrainingWindowWeek, sessions, matches, reviews)

	if len(got.Periods) != 2 || got.Periods[0].Start != "2026-10-05" || got.Periods[1].Start != "2026-10-12" {
		t.Fatalf("periods = %+v", got.Periods)
	}
	w1 := got.Periods[0]
	// Sunday 11 October still belongs to the week of Monday the 5th.
	if w1.QuizSessions != 2 || w1.QuizDecisions != 40 || math.Abs(w1.QuizPR-5) > 1e-9 {
		t.Errorf("week 1 quiz = %+v, want 2 sessions, 40 decisions, PR 5", w1)
	}
	if w1.MatchDecisions != 100 || w1.MatchPR != 6 {
		t.Errorf("week 1 matches = %+v", w1)
	}
	if w1.AnkiReviews != 2 || w1.AnkiPassed != 1 || w1.AnkiRetention != 0.5 {
		t.Errorf("week 1 anki = %+v: only review-state cards count", w1)
	}
	if len(got.Sessions) != 3 {
		t.Errorf("only Decision sessions are listed: got %d", len(got.Sessions))
	}
}

func TestBuildTrainingStats_MonthAndUnreadableDates(t *testing.T) {
	got := BuildTrainingStats(TrainingWindowMonth,
		[]TrainingSession{{ID: 1, Exercise: "decision", CreatedAt: "bad", NumbersAsked: 3, PR: 1}, {ID: 2, Exercise: "decision", CreatedAt: "2026-10-31 23:59:59", NumbersAsked: 3, PR: 1}},
		nil, nil)
	if len(got.Periods) != 1 || got.Periods[0].Start != "2026-10-01" || len(got.Sessions) != 1 {
		t.Fatalf("got %+v", got)
	}
	empty := BuildTrainingStats(TrainingWindowWeek, nil, nil, nil)
	if empty.Periods == nil || empty.Sessions == nil {
		t.Error("empty slices must marshal as [], not null")
	}
}
