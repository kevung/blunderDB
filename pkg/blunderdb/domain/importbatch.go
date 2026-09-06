package domain

// Import batches and their report (issue #257, fiche I.1).
//
// An import used to end on nothing: the progress bar reached the end and the
// window went back to what it was. The user had no way to tell what had just
// entered the database, what had been skipped as a duplicate, or which of the
// positions that came in were worth looking at.
//
// The unit of account is the BATCH — one import the user launched, whatever it
// read: one file, a folder, a paste. Matches point back at their batch, which
// is what lets the report speak about *this import* rather than about the
// database.

// ImportBatch is one import the user launched.
type ImportBatch struct {
	ID int64 `json:"id"`
	// StartedAt and FinishedAt are ISO-8601 timestamps as the backend spells
	// them; FinishedAt is empty while the import is still running or when it
	// was cancelled.
	StartedAt  string `json:"startedAt"`
	FinishedAt string `json:"finishedAt"`
	// Source is what was imported, shown to the user verbatim: a file path, a
	// folder, or a short label for something with no path of its own.
	Source string `json:"source"`
	// Format is the import format ("xg", "gnubg", "bgf", "mat", "db",
	// "position"), or "mixed" for a folder that held several.
	Format string `json:"format"`
	// Report is what the batch found. It is stored as JSON in one column
	// rather than in columns of its own: the report gains figures over time,
	// and each one would otherwise be a schema bump.
	Report ImportReport `json:"report"`
}

// ImportReport is what an import found, in the shape the end-of-import panel
// shows it. Every count is of THIS batch, never of the database.
//
// Counts are what the import itself observed; the rest is measured afterwards
// over the batch's matches, so a report can be recomputed from a batch id
// alone and does not have to be trusted to have been written correctly.
type ImportReport struct {
	// MatchesImported is how many matches entered the database. Skipped are
	// the exact same-format duplicates nothing was written for; Enriched are
	// the cross-format duplicates whose analyses and comments were merged into
	// what was already stored.
	MatchesImported int `json:"matchesImported"`
	MatchesSkipped  int `json:"matchesSkipped"`
	MatchesEnriched int `json:"matchesEnriched"`

	// FilesFailed counts the files the batch could not read at all, and
	// Failures names the first few of them with the reason. A batch that
	// imported nine files out of ten is a success that must still say so.
	FilesFailed int             `json:"filesFailed"`
	Failures    []ImportFailure `json:"failures,omitempty"`

	// PositionsSaved counts the positions the batch wrote — new rows only:
	// deduplication means a position already stored is not saved again.
	PositionsSaved int `json:"positionsSaved"`
	// PositionsFlagged counts the batch's positions the source tool had marked
	// for study (docs/adr/0006) — the first thing a user wants to look at.
	PositionsFlagged int `json:"positionsFlagged"`
	// PositionsWithoutAnalysis counts the batch's positions no engine has
	// judged. It is what the panel's "analyse these now" button acts on.
	PositionsWithoutAnalysis int `json:"positionsWithoutAnalysis"`

	// Decisions and PR are the batch's own performance: how many decisions
	// were scored and their Performance Rating — the same figure the
	// statistics show, over this import alone. Both are 0 when the batch
	// carried no analysis, which the panel must show as "no analysis" rather
	// than as a PR of zero.
	Decisions int     `json:"decisions"`
	PR        float64 `json:"pr"`
	// Player names whose decisions the PR is about, empty when it scores both
	// seats. The panel and the CLI say which: a PR mixing two players is a
	// fact about the import, but only if it is labelled as one.
	Player string `json:"player,omitempty"`

	// WorstDecisions are the batch's five most expensive mistakes, worst
	// first: enough to answer "what did I just do wrong?" without opening the
	// statistics.
	WorstDecisions []ImportBlunder `json:"worstDecisions,omitempty"`
}

// ImportFailure names one file the batch could not read.
type ImportFailure struct {
	Source string `json:"source"`
	Reason string `json:"reason"`
}

// ImportBlunder is one of a batch's worst decisions.
type ImportBlunder struct {
	PositionID int64 `json:"positionId"`
	MatchID    int64 `json:"matchId"`
	// Label is what to show for the match ("Alice — Bob, 7 pts"), built by the
	// query rather than by the panel so the CLI prints the same thing.
	Label string `json:"label"`
	// ErrorMP is the cost of the played move or cube action in millipoints,
	// positive.
	ErrorMP int `json:"errorMp"`
	// IsCube tells a cube error from a checker one; the panel shows a
	// different icon and the CLI a different word.
	IsCube bool `json:"isCube"`
}

// MaxImportBlunders is how many of a batch's worst decisions the report
// carries. Five: enough to see a pattern, few enough to read at a glance
// without the panel becoming a second statistics tab.
const MaxImportBlunders = 5

// MaxImportFailures is how many failing files a report names individually.
// The count is exact; the list is a sample, because a folder of a thousand
// unreadable files must not produce a thousand-line report.
const MaxImportFailures = 10
