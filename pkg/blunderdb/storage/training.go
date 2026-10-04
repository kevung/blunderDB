package storage

import "context"

// The Training journal (ADR-0040 rule 6): what the user asked themselves,
// kept in the library's own tables so it travels with the file.
//
// Two tables rather than a metadata JSON key: the per-number detail grows
// without bound.
//
// A session is written once, at « Terminer », with its items; nothing is
// written by « Quitter ». There is no update and no delete: the journal is a
// record of what happened, and a record that can be edited measures nothing.

// TrainingItem is one Number of one Question, as the journal keeps it.
//
// Deviation is the SIGNED error of an *entered* Number (the size of the
// mistake is the lesson there). A declared Number — a pip count, a table cell
// — has none, and neither has a Question that ran out of time: HasDeviation
// says which, so a missing answer never enters a mean as a zero error.
type TrainingItem struct {
	// NumberType is the kind of number, not the face that was asked:
	// "tp4.last", "gv2", "pips.bottom", … The same cell of the same table
	// seen from either side is one weakness, not two (ADR-0040 rule 4).
	NumberType   string  `json:"numberType"`
	Wrong        bool    `json:"wrong"`
	HasDeviation bool    `json:"hasDeviation"`
	Deviation    float64 `json:"deviation"`
	// PositionID is the library position a decision question was asked on,
	// nil for the number exercises and for a position the question did not
	// come from. Answer is what was played or chosen, as the quiz renders it
	// ('' when nothing was), and ErrorMp its cost in millipoints of
	// normalised equity — nil when nothing was judged (out of time), which is
	// not the same as a correct answer's 0.
	PositionID *int64 `json:"positionId,omitempty"`
	Answer     string `json:"answer,omitempty"`
	ErrorMp    *int   `json:"errorMp,omitempty"`
}

// TrainingSession is one row of the journal, with its items on the way in and
// without them on the way out (the summaries read the aggregates, never the
// item list).
type TrainingSession struct {
	ID       int64  `json:"id"`
	Exercise string `json:"exercise"`
	// SeedSource is "pool", "board" or "library" — where the Questions came
	// from (ADR-0041 rule 2).
	SeedSource    string  `json:"seedSource"`
	CreatedAt     string  `json:"createdAt"`
	NumbersAsked  int     `json:"numbersAsked"`
	Faults        int     `json:"faults"`
	Deviations    int     `json:"deviations"`
	MeanDeviation float64 `json:"meanDeviation"`
	MedianMs      int     `json:"medianMs"`
	// PR is the session's performance rating, for the Decision exercise only;
	// zero everywhere else, where no equity error exists to rate.
	PR    float64        `json:"pr"`
	Items []TrainingItem `json:"items,omitempty"`
}

// TrainingNumberStat is the per-Number-type detail one Exercise's summary
// unfolds into.
type TrainingNumberStat struct {
	NumberType    string  `json:"numberType"`
	Asked         int     `json:"asked"`
	Faults        int     `json:"faults"`
	Deviations    int     `json:"deviations"`
	MeanDeviation float64 `json:"meanDeviation"`
}

// TrainingStore persists the Training journal. It is deliberately narrow:
// append a session, read sessions back, read the per-number aggregate. The
// summary itself (fault rate, median of medians, trend over the last ten) is
// computed where it is shown, from these rows.
type TrainingStore interface {
	// Save appends a session and its items atomically, and returns the
	// session's id.
	Save(ctx context.Context, scope string, session TrainingSession) (int64, error)
	// Sessions returns the recorded sessions, most recent first. An empty
	// exercise means every exercise; a limit of zero means no bound.
	Sessions(ctx context.Context, scope, exercise string, limit int) ([]TrainingSession, error)
	// NumberStats aggregates the items of one exercise by number type,
	// most-asked first.
	NumberStats(ctx context.Context, scope, exercise string) ([]TrainingNumberStat, error)
	// Missed returns the positions of the items answered wrong, each once,
	// the most recently missed first: of one session when sessionID is above
	// zero, of every session of the exercise otherwise (an empty exercise
	// means every exercise). A limit of zero means no bound. A position since
	// deleted is not returned.
	Missed(ctx context.Context, scope string, filter TrainingMissedFilter) ([]int64, error)
	// DecisionErrors returns the judged decision questions of the Decision
	// exercise that still reach a position, oldest first: the date of their
	// session, the position's plan of play and the cost of the answer.
	DecisionErrors(ctx context.Context, scope string) ([]TrainingDecisionError, error)
}

// TrainingDecisionError is one judged decision question, seen from the
// theme of its position.
type TrainingDecisionError struct {
	CreatedAt string
	// GameType is the position's domain.GameType code.
	GameType int
	// ErrorMp is the cost of the answer in millipoints of normalised equity.
	ErrorMp int
}

// TrainingMissedFilter selects the missed questions Missed returns.
type TrainingMissedFilter struct {
	Exercise  string `json:"exercise"`
	SessionID int64  `json:"sessionId"`
	Limit     int    `json:"limit"`
}
