package storage

import (
	"context"
	"iter"
)

// Duel is the DRAFT of a match played here, under blunderDB's arbitration
// (ADR-0072 rule 10): written after every Action, resumed where it stopped,
// and gone once it has become a Match or been thrown away. Like a
// Transcription, the play itself is one opaque Document the store never looks
// inside, versioned by its own FormatVersion.
//
// DiceSeed is a column of its own, not part of the Document: every roll of the
// Duel is computed from it, and keeping it apart is what lets the Document be
// shown while the seed stays with the Arbiter until the Match reveals it
// (ADR-0072 rule 8).
type Duel struct {
	ID int64 `json:"id"`
	// CreatedAt and UpdatedAt are the backend's timestamps, as text.
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
	FormatVersion string `json:"format_version"`
	// Label is what a list of Duels in suspense shows: the players.
	Label    string `json:"label"`
	Document string `json:"document"`
	DiceSeed string `json:"-"`
	// Revision counts the row's writes, as Transcription.Revision does, and is
	// the caller's expectation on Save: non-zero and stale reports ErrConflict.
	Revision int64 `json:"revision"`
}

// MatchOrigin is how a Match came to be when it was played here rather than
// imported or transcribed (ADR-0072 rule 10). A Match without one was not
// played here. Nothing in it is invented: StoppedEarly says the match was
// stopped before its end, never who would have won it.
type MatchOrigin struct {
	MatchID int64 `json:"match_id"`
	// Start is the XGID of the Position the first game began at, "" for the
	// default: the opening position at the start of the match. A Match with a
	// Start cannot be written as a .mat (ADR-0072 rule 12).
	Start string `json:"start"`
	// DiceSeed is the revealed seed every roll was computed from.
	DiceSeed     string `json:"dice_seed"`
	StoppedEarly bool   `json:"stopped_early"`
	// OverTime, BotLevel and Cadence are the parts of the origin a Bot and
	// a Cadence give (ADR-0072 rule 10, ADR-0073); empty when the Duel had
	// neither. OverTime is the player (1 or 2) whose reserve ran out first,
	// 0 for none: a fact, whether the Cadence then lost them the match —
	// StoppedEarly — or let them play on. Cadence is the Cadence as the Duel
	// was set, in JSON.
	OverTime int    `json:"over_time"`
	BotLevel string `json:"bot_level"`
	Cadence  string `json:"cadence"`
}

// DuelStore persists the drafts of Duels and the origin of the Matches they
// became.
type DuelStore interface {
	// List streams the scope's Duels in suspense, most recently updated first.
	List(ctx context.Context, scope string) iter.Seq2[*Duel, error]

	// Get returns one Duel, or ErrNotFound.
	Get(ctx context.Context, scope string, id int64) (*Duel, error)

	// Save inserts d when it carries no id — d.DiceSeed is written then and
	// never again — and otherwise rewrites the row's document and label under
	// d.Revision's expectation: ErrNotFound when the row is gone, ErrConflict
	// when d.Revision is non-zero and no longer the row's. d.Revision holds the
	// row's new revision on success.
	Save(ctx context.Context, scope string, d *Duel) (int64, error)

	// Delete removes a Duel, or reports ErrNotFound.
	Delete(ctx context.Context, scope string, id int64) error

	// SetOrigin writes the origin of a Match played here, replacing any. A
	// MatchID that names no match of the scope reports ErrNotFound.
	SetOrigin(ctx context.Context, scope string, o *MatchOrigin) error

	// Origin returns a Match's origin, or ErrNotFound when it was not played
	// here. Deleting the Match deletes its origin.
	Origin(ctx context.Context, scope string, matchID int64) (*MatchOrigin, error)
}
