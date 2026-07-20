package storage

import "context"

// StatsFilter defines the filtering criteria for a stats computation.
type StatsFilter struct {
	PlayerName    string
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
	ErrorHistogram      []ErrorBucket     `json:"ErrorHistogram"`
	TopBlunders         []BlunderEntry    `json:"TopBlunders"`
}

// SelectionSpec selects a subset of positions out of a stats result, e.g. the
// decisions behind a histogram bucket or a tournament row.
type SelectionSpec struct {
	Kind          string // "all","checker","cube","cube_action","error_bucket","tournament","match","last_n","position","top_blunders"
	CubeAction    string // "NoDouble" | "DoubleTake" | "DoublePass" | "TooGood"
	BucketMinMP   int    // inclusive
	BucketMaxMP   int    // exclusive; -1 = +∞
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
