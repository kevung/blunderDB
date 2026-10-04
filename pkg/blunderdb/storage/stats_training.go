package storage

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// Training statistics: what the Decision quiz and the Anki reviews say about
// the player's progress, on the same calendar as the Performance Rating of the
// real matches. Nothing here is stored: the three series are read from the
// journals that already exist (training_session, anki_review_log, the matches)
// and folded by calendar window, so a window holds no duplicate of any row.

// Calendar windows a TrainingStats can be folded into.
const (
	TrainingWindowWeek  = "week"
	TrainingWindowMonth = "month"
)

// TrainingQuizSession is one Decision session of the journal: its PR is on the
// scale of the real PR, so the two can be laid side by side.
type TrainingQuizSession struct {
	ID        int64   `json:"ID"`
	CreatedAt string  `json:"CreatedAt"`
	Decisions int     `json:"Decisions"`
	PR        float64 `json:"PR"`
}

// TrainingPeriod is one calendar window. A series with nothing in the window
// has a zero count and a zero value: the count says whether the value means
// anything.
type TrainingPeriod struct {
	// Start is the first day of the window, ISO "YYYY-MM-DD" (a Monday for a
	// week, the 1st for a month).
	Start string `json:"Start"`
	// QuizSessions and QuizDecisions are the Decision sessions of the window
	// and the decisions they judged; QuizPR is their PR weighted by decisions.
	QuizSessions  int     `json:"QuizSessions"`
	QuizDecisions int     `json:"QuizDecisions"`
	QuizPR        float64 `json:"QuizPR"`
	// MatchDecisions and MatchPR are the real matches of the window, the
	// filter's own PR, weighted by decisions.
	MatchDecisions int     `json:"MatchDecisions"`
	MatchPR        float64 `json:"MatchPR"`
	// AnkiReviews counts the reviews of review-state cards; AnkiPassed those
	// rated Hard or better; AnkiRetention is their ratio, the measure of
	// AnkiStore.Retention on the same window.
	AnkiReviews   int     `json:"AnkiReviews"`
	AnkiPassed    int     `json:"AnkiPassed"`
	AnkiRetention float64 `json:"AnkiRetention"`
}

// TrainingStats is the training side of the progression.
type TrainingStats struct {
	Window   string                `json:"Window"`
	Sessions []TrainingQuizSession `json:"Sessions"`
	Periods  []TrainingPeriod      `json:"Periods"`
}

// ComputeTrainingStats reads the three journals and folds them by window. The
// filter restricts the real matches only: the quiz and the Anki journals are
// the player's own and carry no player.
func ComputeTrainingStats(ctx context.Context, st Stores, scope string, filter StatsFilter, window string) (*TrainingStats, error) {
	if window != TrainingWindowWeek && window != TrainingWindowMonth {
		return nil, fmt.Errorf("unknown window %q: want %q or %q", window, TrainingWindowWeek, TrainingWindowMonth)
	}
	sessions, err := st.Training().Sessions(ctx, scope, "decision", 0)
	if err != nil {
		return nil, err
	}
	// The match side needs only each match's PR: MatchSeries answers it
	// without the other passes of Compute.
	perMatch, err := st.Stats().MatchSeries(ctx, scope, filter)
	if err != nil {
		return nil, err
	}
	var reviews []domain.AnkiReviewLog
	for l, err := range st.Anki().ReviewLog(ctx, scope, 0, 0) {
		if err != nil {
			return nil, err
		}
		reviews = append(reviews, *l)
	}
	return BuildTrainingStats(window, sessions, perMatch, reviews), nil
}

// BuildTrainingStats is the pure fold: sessions of the Decision exercise,
// matches and review events in, windows out, oldest first. A row whose date
// cannot be read is left out rather than put in a window it does not belong to.
func BuildTrainingStats(window string, sessions []TrainingSession, matches []MatchStats, reviews []domain.AnkiReviewLog) *TrainingStats {
	out := &TrainingStats{Window: window, Sessions: []TrainingQuizSession{}, Periods: []TrainingPeriod{}}
	periods := map[string]*TrainingPeriod{}
	at := func(date string) *TrainingPeriod {
		day, ok := isoDay(date)
		if !ok {
			return nil
		}
		start := windowStart(day, window).Format("2006-01-02")
		p := periods[start]
		if p == nil {
			p = &TrainingPeriod{Start: start}
			periods[start] = p
		}
		return p
	}
	var quizSum, matchSum = map[string]float64{}, map[string]float64{}
	for _, s := range sessions {
		if s.Exercise != "decision" || s.NumbersAsked == 0 {
			continue
		}
		p := at(s.CreatedAt)
		if p == nil {
			continue
		}
		out.Sessions = append(out.Sessions, TrainingQuizSession{ID: s.ID, CreatedAt: s.CreatedAt, Decisions: s.NumbersAsked, PR: s.PR})
		p.QuizSessions++
		p.QuizDecisions += s.NumbersAsked
		quizSum[p.Start] += s.PR * float64(s.NumbersAsked)
	}
	for _, m := range matches {
		if m.NumDecisions == 0 {
			continue
		}
		if p := at(m.Date); p != nil {
			p.MatchDecisions += m.NumDecisions
			matchSum[p.Start] += m.PR * float64(m.NumDecisions)
		}
	}
	for _, r := range reviews {
		if r.State != 2 {
			continue
		}
		if p := at(r.ReviewedAt); p != nil {
			p.AnkiReviews++
			if r.Rating >= 2 {
				p.AnkiPassed++
			}
		}
	}
	for start, p := range periods {
		if p.QuizDecisions > 0 {
			p.QuizPR = quizSum[start] / float64(p.QuizDecisions)
		}
		if p.MatchDecisions > 0 {
			p.MatchPR = matchSum[start] / float64(p.MatchDecisions)
		}
		if p.AnkiReviews > 0 {
			p.AnkiRetention = float64(p.AnkiPassed) / float64(p.AnkiReviews)
		}
		out.Periods = append(out.Periods, *p)
	}
	sort.Slice(out.Periods, func(i, j int) bool { return out.Periods[i].Start < out.Periods[j].Start })
	sort.Slice(out.Sessions, func(i, j int) bool {
		if out.Sessions[i].CreatedAt != out.Sessions[j].CreatedAt {
			return out.Sessions[i].CreatedAt < out.Sessions[j].CreatedAt
		}
		return out.Sessions[i].ID < out.Sessions[j].ID
	})
	return out
}

// isoDay reads the date part of a stored timestamp ("2026-10-03",
// "2026-10-03 07:00:00" or RFC 3339): both backends write one of them. The day
// is the one written, in UTC, never shifted into a local zone.
func isoDay(s string) (time.Time, bool) {
	if len(s) < 10 {
		return time.Time{}, false
	}
	t, err := time.Parse("2006-01-02", s[:10])
	return t, err == nil
}

// windowStart is the first day of the week (Monday) or month holding day.
func windowStart(day time.Time, window string) time.Time {
	if window == TrainingWindowMonth {
		return time.Date(day.Year(), day.Month(), 1, 0, 0, 0, 0, time.UTC)
	}
	back := (int(day.Weekday()) + 6) % 7
	return day.AddDate(0, 0, -back)
}
