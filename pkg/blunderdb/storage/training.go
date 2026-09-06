package storage

import "context"

// The Training journal (ADR-0040 rule 6): what the user asked themselves,
// kept in the library's own tables so it travels with the file.
//
// Two tables and not a JSON key of `metadata`, because the per-number detail
// is the whole point — "tp4 last roll: 6 faults in 9" — and a blob that grows
// by one entry per revealed number is a register a table settles. No cap: the
// old fifty-session bound existed to keep a metadata value small.
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
}
