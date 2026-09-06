package storage

import "context"

// StatsFilter defines the filtering criteria for a stats computation.
type StatsFilter struct {
	PlayerName string
	// PlayerAliases are the other spellings the same person signed under. A
	// player's name is typed by hand into every file, so one person routinely
	// appears as "Kévin Unger", "Kévin UNGER" and "UNGER Kevin" — in a real
	// 151-match corpus, 12 % of the matches sat under the two minor spellings.
	// Matching a single string computes on part of the data and nothing looks
	// wrong, which is the worst kind of defect.
	//
	// The filter keeps the decisions of ANY of PlayerName + PlayerAliases: the
	// field is purely additive, so filling both is not a trap. Merging the names
	// in place (MergePlayers) is the other answer, and the wrong one as soon as
	// the database is shared — it would rewrite other people's matches.
	PlayerAliases []string
	TournamentIDs []int64
	DateFrom      string // ISO "YYYY-MM-DD"
	DateTo        string // ISO "YYYY-MM-DD"
	DecisionType  int    // -1=all, 0=checker, 1=cube
	MatchLength   []int
}

// StatsDateRange is the span of match dates present in the database.
type StatsDateRange struct {
	DateFrom string `json:"DateFrom"` // ISO "YYYY-MM-DD", empty if no matches
	DateTo   string `json:"DateTo"`   // ISO "YYYY-MM-DD", empty if no matches
}

// StatsTotals holds high-level counts for a stats result.
type StatsTotals struct {
	NumPositions   int `json:"NumPositions"`
	NumMatches     int `json:"NumMatches"`
	NumTournaments int `json:"NumTournaments"`
	NumDecisions   int `json:"NumDecisions"`
}

// TournamentStats holds aggregated stats for a single tournament.
type TournamentStats struct {
	ID           int64   `json:"ID"`
	Name         string  `json:"Name"`
	Date         string  `json:"Date"`
	PR           float64 `json:"PR"`
	MWC          float64 `json:"MWC"`
	NumDecisions int     `json:"NumDecisions"`
}

// MatchStats holds aggregated stats for a single match.
type MatchStats struct {
	ID           int64   `json:"ID"`
	Date         string  `json:"Date"`
	PlayerName   string  `json:"PlayerName"`
	PR           float64 `json:"PR"`
	MWC          float64 `json:"MWC"`
	NumDecisions int     `json:"NumDecisions"`
}

// CubeActionStats holds aggregated stats grouped by cube action.
type CubeActionStats struct {
	Action       string  `json:"Action"`
	PR           float64 `json:"PR"`
	MWC          float64 `json:"MWC"`
	NumDecisions int     `json:"NumDecisions"`
	BlunderCount int     `json:"BlunderCount"`
}

// ErrorBucket groups decisions by magnitude of error.
type ErrorBucket struct {
	MinMP int `json:"MinMP"`
	MaxMP int `json:"MaxMP"`
	Count int `json:"Count"`
}

// BlunderEntry identifies a single bad decision.
type BlunderEntry struct {
	PositionID   int64   `json:"PositionID"`
	MatchID      int64   `json:"MatchID"`
	TournamentID int64   `json:"TournamentID"`
	ErrorMP      int64   `json:"ErrorMP"`
	MWCLoss      float64 `json:"MWCLoss"`
	Description  string  `json:"Description"`
	DecisionType int     `json:"DecisionType"` // 0=checker, 1=cube
	MatchDate    string  `json:"MatchDate"`
	PlayerNames  string  `json:"PlayerNames"`
}

// StatsResult contains all computed statistics for a given filter.
type StatsResult struct {
	Totals              StatsTotals       `json:"Totals"`
	PRGlobal            float64           `json:"PRGlobal"`
	PRChecker           float64           `json:"PRChecker"`
	PRCube              float64           `json:"PRCube"`
	PRRolling           map[int]float64   `json:"PRRolling"`
	MWCGlobal           float64           `json:"MWCGlobal"`
	MWCChecker          float64           `json:"MWCChecker"`
	MWCCube             float64           `json:"MWCCube"`
	MWCRolling          map[int]float64   `json:"MWCRolling"`
	MWCAvailable        bool              `json:"MWCAvailable"`
	SnowieGlobal        float64           `json:"SnowieGlobal"`
	PerTournament       []TournamentStats `json:"PerTournament"`
	PerMatch            []MatchStats      `json:"PerMatch"`
	CubeActionBreakdown []CubeActionStats `json:"CubeActionBreakdown"`
	// CubeDirections says in WHICH direction the cube decisions went wrong,
	// where CubeActionBreakdown says how much they cost. See
	// stats_cubedirections.go.
	CubeDirections CubeDirections `json:"CubeDirections"`
	ErrorHistogram []ErrorBucket  `json:"ErrorHistogram"`
	TopBlunders    []BlunderEntry `json:"TopBlunders"`

	// The three breakdowns of issue #266 (fiche I.10). Each is the SAME
	// figures as the global ones, restricted to a slice of the same
	// selection — never a second notion of what counts as a decision, which
	// would be a second PR.
	//
	// PerPhase splits by the position's derived game phase (ADR-0035), which
	// is what answers "my PR in the race versus in contact".
	PerPhase []PhaseStats `json:"PerPhase"`
	// PerTag splits by the tags in the position's comments. A position
	// carrying two tags appears in both rows: a tag is a label, not a
	// partition, so the rows deliberately do not sum to the total.
	PerTag []TagStats `json:"PerTag"`
	// PerScore is the away × away matrix — Crawford, post-Crawford, DMP and
	// everything between. Cells whose sample is too small to read are still
	// returned WITH their count, so the caller greys them out rather than
	// being handed a figure with no idea how much is behind it.
	PerScore []ScoreCellStats `json:"PerScore"`
}

// PhaseStats is one row of the per-phase breakdown (#266).
type PhaseStats struct {
	// Phase is the stable token of domain.GamePhase ("opening", "race", …).
	Phase        string  `json:"Phase"`
	PR           float64 `json:"PR"`
	NumDecisions int     `json:"NumDecisions"`
	BlunderCount int     `json:"BlunderCount"`
}

// TagStats is one row of the per-tag breakdown (#266). Tag carries the "#".
type TagStats struct {
	Tag          string  `json:"Tag"`
	PR           float64 `json:"PR"`
	NumDecisions int     `json:"NumDecisions"`
	BlunderCount int     `json:"BlunderCount"`
}

// ScoreCellStats is one cell of the away × away matrix (#266).
//
// Away is "points still needed to win": 1 is DMP-adjacent, and the cell is
// identified by the pair (mover's away, opponent's away) from the reference
// player's side. Crawford is a separate flag rather than a fourth dimension:
// a Crawford game at 1-away/3-away is a different decision from a
// post-Crawford one at the same score, and folding them would average two
// different games.
type ScoreCellStats struct {
	MoverAway    int     `json:"MoverAway"`
	OpponentAway int     `json:"OpponentAway"`
	PR           float64 `json:"PR"`
	NumDecisions int     `json:"NumDecisions"`
	BlunderCount int     `json:"BlunderCount"`
}

// MinCellDecisions is the sample below which a matrix cell is not worth
// reading. It is not enforced here — the cell is returned with its count, and
// the caller greys it — because "too small to read" is a display decision and
// hiding the count would make it unauditable.
const MinCellDecisions = 10

// SelectionSpec selects a subset of positions out of a stats result, e.g. the
// decisions behind a histogram bucket or a tournament row.
type SelectionSpec struct {
	Kind string // "all","checker","cube","cube_action","cube_direction","error_bucket","tournament","match","last_n","position","top_blunders"
	// CubeAction matches analysis.best_cube_action VERBATIM, in whatever
	// spelling the importer wrote ("No Double", "Double, Take"…) — not the
	// canonical form.
	CubeAction string
	// CubeCell, for Kind "cube_direction", names one cell of the cube matrix:
	// one of the CubeCell* constants in stats_cubedirections.go.
	CubeCell      string
	BucketMinMP   int // inclusive
	BucketMaxMP   int // exclusive; -1 = +∞
	TournamentID  int64
	MatchID       int64
	LastN         int
	PositionID    int64
	OnlyWithError bool
}

// PlayerFrequency is a player name and how many matches they appear in.
type PlayerFrequency struct {
	Name  string
	Count int
}

// PlayerRow is one line of the players table: everything the panel shows about
// a player, computed over the matches the filter retains.
//
// A row is keyed by a player NAME exactly as it appears in the matches, not by
// a person: the same human signing two ways is two rows. Merging spellings is a
// user gesture (merge players), never something this computes.
type PlayerRow struct {
	Name string `json:"name"`

	// Matches counts the matches the player appears in; Wins and Losses count
	// those with a decided outcome, so Wins+Losses can be less than Matches —
	// an unfinished match counts for neither.
	Matches int `json:"matches"`
	Wins    int `json:"wins"`
	Losses  int `json:"losses"`

	// Decisions is the number of counted decisions (statsCountedExpr), the
	// denominator behind PR. It is the reader's measure of how much the rates
	// beside it are worth: a PR over twelve decisions is noise, and the panel
	// shows the count rather than hiding the row.
	Decisions        int `json:"decisions"`
	CheckerDecisions int `json:"checker_decisions"`
	CubeDecisions    int `json:"cube_decisions"`

	PR        float64 `json:"pr"`
	PRChecker float64 `json:"pr_checker"`
	PRCube    float64 `json:"pr_cube"`

	// SnowieER divides this player's errors by BOTH players' checker moves in
	// their matches (gnuBG formatgs.c:415-424).
	SnowieER float64 `json:"snowie_er"`

	Errors   int `json:"errors"`
	Blunders int `json:"blunders"`

	// LuckMPSum is the total luck in signed millipoints and LuckRolls the
	// number of rolls it was measured over — never the number of rolls played.
	// Averaging over rolls whose luck is unknown would quietly pull every
	// player towards zero, so the caller divides by LuckRolls and shows nothing
	// at all when it is zero. See ADR-0010.
	LuckMPSum int64 `json:"luck_mp_sum"`
	LuckRolls int   `json:"luck_rolls"`
}

// LuckRateMP is the average luck per measured roll, in signed millipoints, and
// false when this player has no luck data at all — which is not the same as an
// average of zero, and must not be displayed as one.
func (r PlayerRow) LuckRateMP() (float64, bool) {
	if r.LuckRolls == 0 {
		return 0, false
	}
	return float64(r.LuckMPSum) / float64(r.LuckRolls), true
}

// MatchPlayerDetailStats holds one player's detailed stats for a single match.
type MatchPlayerDetailStats struct {
	TotalDecisions   int     `json:"total_decisions"`
	TotalErrors      int     `json:"total_errors"`
	TotalBlunders    int     `json:"total_blunders"`
	TotalEquityError float64 `json:"total_equity_error"`
	PR               float64 `json:"pr"`
	MWCLoss          float64 `json:"mwc_loss"`

	CheckerDecisions   int     `json:"checker_decisions"`
	CheckerErrors      int     `json:"checker_errors"`
	CheckerBlunders    int     `json:"checker_blunders"`
	CheckerEquityError float64 `json:"checker_equity_error"`
	PRChecker          float64 `json:"pr_checker"`
	CheckerMWCLoss     float64 `json:"checker_mwc_loss"`

	DoubleDecisions   int     `json:"double_decisions"`
	DoubleErrors      int     `json:"double_errors"`
	DoubleBlunders    int     `json:"double_blunders"`
	DoubleEquityError float64 `json:"double_equity_error"`
	DoubleMWCLoss     float64 `json:"double_mwc_loss"`

	TakeDecisions   int     `json:"take_decisions"`
	TakeErrors      int     `json:"take_errors"`
	TakeBlunders    int     `json:"take_blunders"`
	TakeEquityError float64 `json:"take_equity_error"`
	TakeMWCLoss     float64 `json:"take_mwc_loss"`

	PRCube      float64 `json:"pr_cube"`
	CubeMWCLoss float64 `json:"cube_mwc_loss"`

	SnowieER float64 `json:"snowie_er"`
}

// MatchDetailStats holds per-player statistics for a single match.
type MatchDetailStats struct {
	MatchID int64                  `json:"match_id"`
	Player1 MatchPlayerDetailStats `json:"player1"`
	Player2 MatchPlayerDetailStats `json:"player2"`
}

// MatchBadge is the per-player PR/MWC summary shown on each match-list row.
// PR/MWCLoss are player 1's, PR2/MWCLoss2 player 2's. It is the list-row
// projection of MatchDetailStats (badge.PR == detail.Player1.PR for a match).
type MatchBadge struct {
	PR       float64 `json:"pr"`
	MWCLoss  float64 `json:"mwc_loss"`
	PR2      float64 `json:"pr2"`
	MWCLoss2 float64 `json:"mwc_loss2"`
}

// TournamentBadge is the PR/MWC shown on each tournament-list row. Unlike a
// match (two fixed players), a tournament groups matches against varying
// opponents, so a pooled PR would blend the reference player's decisions with
// every opponent's. Instead the badge reports the reference player's own PR:
// RefPlayer is the person appearing in the most of the tournament's matches
// (see PickReferencePlayer), and PR/MWCLoss cover only that player's decisions.
type TournamentBadge struct {
	PR        float64 `json:"pr"`
	MWCLoss   float64 `json:"mwc_loss"`
	RefPlayer string  `json:"ref_player"`
}

// TournamentPlayerAcc accumulates one player's counted decisions within a single
// tournament. Backends fill one per (tournament, player) and hand the per-player
// map to PickReferencePlayer. Matches holds the distinct match IDs in which the
// player made a counted decision (used to rank frequency).
type TournamentPlayerAcc struct {
	SumErr  int64
	Cnt     int
	MWC     float64
	Matches map[int64]struct{}
}

// PickReferencePlayer selects a tournament's reference player and returns its
// badge. The reference player is the person present in the most of the
// tournament's matches; ties break on the most counted decisions, then on the
// lexicographically smallest non-empty name (deterministic across backends).
// An empty map yields the zero badge.
func PickReferencePlayer(players map[string]*TournamentPlayerAcc) TournamentBadge {
	var best *TournamentPlayerAcc
	var bestName string
	for name, pa := range players {
		if best == nil || refPlayerBetter(name, pa, bestName, best) {
			best, bestName = pa, name
		}
	}
	if best == nil {
		return TournamentBadge{}
	}
	pr := 0.0
	if best.Cnt > 0 {
		pr = 500 * float64(best.SumErr) / 1000 / float64(best.Cnt)
	}
	return TournamentBadge{PR: pr, MWCLoss: best.MWC, RefPlayer: bestName}
}

// refPlayerBetter reports whether candidate (name, pa) outranks the current best
// (bestName, best) as reference player, applying the tie-break order documented
// on PickReferencePlayer.
func refPlayerBetter(name string, pa *TournamentPlayerAcc, bestName string, best *TournamentPlayerAcc) bool {
	if len(pa.Matches) != len(best.Matches) {
		return len(pa.Matches) > len(best.Matches)
	}
	if pa.Cnt != best.Cnt {
		return pa.Cnt > best.Cnt
	}
	// Prefer a named player over an empty/unknown name, then order by name.
	if (name == "") != (bestName == "") {
		return bestName == ""
	}
	return name < bestName
}

// StatsStore computes aggregate statistics over stored decisions.
type StatsStore interface {
	// DateRange returns the span of match dates present in the database.
	DateRange(ctx context.Context, scope string) (StatsDateRange, error)

	// Compute aggregates statistics for the decisions matching filter.
	Compute(ctx context.Context, scope string, filter StatsFilter) (*StatsResult, error)

	// PositionIDsBySelection returns the position ids behind a selection of a
	// previously computed stats result.
	PositionIDsBySelection(ctx context.Context, scope string, filter StatsFilter, sel SelectionSpec) ([]int64, error)

	// PositionIDsByTournament returns the position ids of a tournament.
	PositionIDsByTournament(ctx context.Context, scope string, tournamentID int64) ([]int64, error)

	// PositionIDsByMatch returns the position ids of a match.
	PositionIDsByMatch(ctx context.Context, scope string, matchID int64) ([]int64, error)

	// PlayerNames returns every player name ranked by match frequency.
	PlayerNames(ctx context.Context, scope string) ([]PlayerFrequency, error)

	// PlayerTable computes one row per player over the matches the filter
	// retains, for the players table of the Stats panel.
	//
	// It honours the filter's date range, tournaments and match lengths only.
	// PlayerName/PlayerAliases and DecisionType are ignored by design: the
	// table is about every player at once, and it splits checker from cube in
	// its own columns rather than through a global filter that would leave them
	// inconsistent with each other.
	//
	// Rows come back sorted by PR ascending — best player first — with players
	// having no counted decision last. Nothing is hidden: a player with three
	// decisions is listed, with the decision count that says what their PR is
	// worth.
	PlayerTable(ctx context.Context, scope string, filter StatsFilter) ([]PlayerRow, error)

	// MatchDetail computes per-player statistics for a single match.
	MatchDetail(ctx context.Context, scope string, matchID int64) (*MatchDetailStats, error)

	// MatchBadges returns the per-player PR/MWC badge for the given matches,
	// keyed by match id. A nil/empty matchIDs computes badges for every match in
	// scope (a whole-database scan); pass the ids of the page being displayed to
	// bound the work. Matches with no counted decisions are absent from the map
	// (their badge stays zero-valued).
	MatchBadges(ctx context.Context, scope string, matchIDs []int64) (map[int64]MatchBadge, error)

	// TournamentBadges returns the aggregate PR/MWC badge for every tournament in
	// scope, keyed by tournament id. Tournaments with no counted decisions are
	// absent from the map.
	TournamentBadges(ctx context.Context, scope string) (map[int64]TournamentBadge, error)
}
