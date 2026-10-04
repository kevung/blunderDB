package domain

// Import batches and their report. The unit of account is the BATCH — one
// import the user launched, whatever it read (a file, a folder, a paste).
// Matches point back at their batch, so the report speaks about *this import*
// rather than about the database.

import "strings"

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
	// MatchesDeepened counts the skipped duplicates that still brought a
	// deeper analysis than the stored one, and AnalysesDeepened the positions
	// whose analysis they replaced: "duplicate, nothing deeper" is
	// MatchesSkipped - MatchesDeepened.
	MatchesDeepened  int `json:"matchesDeepened,omitempty"`
	AnalysesDeepened int `json:"analysesDeepened,omitempty"`
	// AnalysesDropped counts the decisions the files analysed with a value
	// that is not a finite number: imported, but without that analysis.
	AnalysesDropped int `json:"analysesDropped,omitempty"`
	// ProbableDuplicates lists the matches this batch wrote whose dice are
	// those of a match already stored under other player names: signalled,
	// never merged (DuplicateSuspect).
	ProbableDuplicates []DuplicateSuspect `json:"probableDuplicates,omitempty"`

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

// The post-import study queue: an ordered list of the batch's positions worth
// a second look, walked through once. It is not a saved object and nothing
// records that a position was seen — what the user does with it (a comment, a
// collection, an Anki card) is the record, the restraint ADR-0006 states about
// flags.

// StudyQueueReason says why a position is in the queue. Three reasons, in the
// order they are offered.
type StudyQueueReason string

const (
	// StudyBlunder is a decision that cost something. The costliest first:
	// this is what the user came for.
	StudyBlunder StudyQueueReason = "blunder"
	// StudyFlagged is a position the SOURCE TOOL marked for study (ADR-0006).
	StudyFlagged StudyQueueReason = "flagged"
	// StudyClose is a cube decision the engine judged close — one where the
	// right answer was not obvious even though nothing was lost.
	StudyClose StudyQueueReason = "close"
)

// StudyQueueEntry is one position of the queue.
type StudyQueueEntry struct {
	PositionID int64            `json:"positionId"`
	MatchID    int64            `json:"matchId"`
	Reason     StudyQueueReason `json:"reason"`
	// Label is the match, spelt the way the report spells it.
	Label string `json:"label"`
	// ErrorMP is the cost in millipoints for a blunder, 0 for the other two
	// reasons — a flagged position may have been played perfectly.
	ErrorMP int `json:"errorMp"`
	// IsCube tells a cube decision from a checker one.
	IsCube bool `json:"isCube"`
}

// MaxStudyQueue bounds the queue: a queue nobody finishes is a queue nobody
// starts.
const MaxStudyQueue = 50

// The queue's error threshold is the library's own (storage.LibrarySettings,
// ADR-0046).

// Kinds of DuplicateSuspect.
const (
	// DuplicateSameDice: both matches have the same length, initial score
	// and dice in every game, under other player names.
	DuplicateSameDice = "same_dice"
	// DuplicateLonger: MatchID's dice continue OtherID's — the same match,
	// truncated in OtherID and completed in MatchID.
	DuplicateLonger = "longer"
)

// DuplicateSuspect pairs two stored matches the dice say are probably one.
// It is a signal for the user, who decides — by an alias, by deleting one —
// and nothing ever merges them on its own.
type DuplicateSuspect struct {
	Kind         string `json:"kind"`
	MatchID      int64  `json:"matchId"`
	OtherID      int64  `json:"otherId"`
	Players      string `json:"players"`
	OtherPlayers string `json:"otherPlayers"`
	// Pairings, for a same-dice suspect, are the ways of reading MatchID's
	// players as OtherID's: each one the aliases that would make the two
	// matches name the same players (SameDicePairings).
	Pairings []AliasPairing `json:"pairings,omitempty"`
}

// PlayerAlias is one spelling to record as a player's other one.
type PlayerAlias struct {
	Alias     string `json:"alias"`
	Canonical string `json:"canonical"`
}

// AliasPairing is one reading of which player of a match is which of the
// other's: the aliases to record for it, one or two.
type AliasPairing struct {
	Aliases []PlayerAlias `json:"aliases"`
}

// NewSameDiceSuspect is the same-dice suspect of match (players p1, p2)
// against other (o1, o2), with its alias pairings.
func NewSameDiceSuspect(matchID, otherID int64, p1, p2, o1, o2 string) DuplicateSuspect {
	return DuplicateSuspect{Kind: DuplicateSameDice, MatchID: matchID, OtherID: otherID,
		Players: p1 + " – " + p2, OtherPlayers: o1 + " – " + o2,
		Pairings: SameDicePairings(p1, p2, o1, o2)}
}

// SameDicePairings lists the readings of p1, p2 (the suspect's spellings,
// the aliases) as o1, o2 (the other match's, the canonical names): seat for
// seat, then crosswise. A reading in which a name already appears on both
// sides is anchored by it — the other name can only be the other player —
// so when one reading is anchored, only the anchored ones are proposed; a
// player paired with his own spelling needs no alias and is left out.
func SameDicePairings(p1, p2, o1, o2 string) []AliasPairing {
	same := func(a, b string) bool {
		return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
	}
	type reading struct {
		pairs    [2][2]string
		anchored bool
	}
	readings := []reading{
		{pairs: [2][2]string{{p1, o1}, {p2, o2}}},
		{pairs: [2][2]string{{p1, o2}, {p2, o1}}},
	}
	anyAnchored := false
	for i := range readings {
		for _, pr := range readings[i].pairs {
			if same(pr[0], pr[1]) {
				readings[i].anchored = true
			}
		}
		anyAnchored = anyAnchored || readings[i].anchored
	}
	var out []AliasPairing
	seen := map[string]bool{}
	for _, r := range readings {
		if anyAnchored && !r.anchored {
			continue
		}
		var p AliasPairing
		key := ""
		for _, pr := range r.pairs {
			if same(pr[0], pr[1]) || strings.TrimSpace(pr[0]) == "" || strings.TrimSpace(pr[1]) == "" {
				continue
			}
			p.Aliases = append(p.Aliases, PlayerAlias{Alias: pr[0], Canonical: pr[1]})
			key += pr[0] + "\x00" + pr[1] + "\x00"
		}
		if len(p.Aliases) == 0 || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, p)
	}
	return out
}

// Journal outcomes: what one file of a batch became.
const (
	// JournalNew is a file that gave a new match (or a position).
	JournalNew = "new"
	// JournalDuplicate is a file whose match is already stored; MatchID is
	// the match that covers it.
	JournalDuplicate = "duplicate"
	// JournalEnriched is a file whose analyses and comments were merged into
	// a match already stored.
	JournalEnriched = "enriched"
	// JournalError is a file the batch could not read or write; Error says why.
	JournalError = "error"
)

// ImportFileEntry is one line of a batch's journal: a file, and what it gave.
// It is import data, never a reading mark.
type ImportFileEntry struct {
	// Path is the file as the import met it.
	Path string `json:"path"`
	Size int64  `json:"size"`
	// MTime is the modification time in UTC, "YYYY-MM-DD HH:MM:SS", empty
	// when unknown.
	MTime  string `json:"mtime,omitempty"`
	SHA256 string `json:"sha256,omitempty"`
	// Outcome is one of the Journal* constants.
	Outcome string `json:"outcome"`
	// MatchID is the new match, the enriched one, or the match that covers a
	// duplicate; 0 when the file gave none.
	MatchID int64  `json:"matchId,omitempty"`
	Error   string `json:"error,omitempty"`
}
