package database

import (
	"context"
)

// StatsDateRange holds the earliest and latest match dates in the database.
type StatsDateRange struct {
	DateFrom string `json:"DateFrom"` // ISO "YYYY-MM-DD", empty if no matches
	DateTo   string `json:"DateTo"`   // ISO "YYYY-MM-DD", empty if no matches
}

// GetStatsDateRange returns the minimum and maximum match dates present in the
// database. Both fields are empty when no matches with a date exist.
// GetStatsDateRange delegates to the storage StatsStore. legacyGetStatsDateRange
// keeps the original SQL as the parity-test reference.
func (d *Database) GetStatsDateRange() StatsDateRange {
	d.mu.RLock()
	defer d.mu.RUnlock()
	r, _ := d.store.Stats().DateRange(context.Background(), "")
	return fromStorageDateRange(r)
}

// StatsFilter defines the filtering criteria for ComputeStats.
type StatsFilter struct {
	PlayerName    string
	TournamentIDs []int64
	DateFrom      string // ISO "YYYY-MM-DD"
	DateTo        string // ISO "YYYY-MM-DD"
	DecisionType  int    // -1=all, 0=checker, 1=cube
	MatchLength   []int
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
	MatchDate    string  `json:"MatchDate"`    // ISO date string, may be empty
	PlayerNames  string  `json:"PlayerNames"`  // "Player1 vs Player2"
}

// StatsResult contains all computed statistics for the given filter.
type StatsResult struct {
	Totals              StatsTotals       `json:"Totals"`
	PRGlobal            float64           `json:"PRGlobal"`
	PRChecker           float64           `json:"PRChecker"`
	PRCube              float64           `json:"PRCube"`
	PRRolling           map[int]float64   `json:"PRRolling"`    // keyed by N: 5,10,50,100,250,500,1000
	MWCGlobal           float64           `json:"MWCGlobal"`    // sum of MWC losses across all match-play decisions
	MWCChecker          float64           `json:"MWCChecker"`   // MWC loss from checker play errors
	MWCCube             float64           `json:"MWCCube"`      // MWC loss from cube action errors
	MWCRolling          map[int]float64   `json:"MWCRolling"`   // rolling MWC loss over N most-recent decisions (same keys as PRRolling)
	MWCAvailable        bool              `json:"MWCAvailable"` // true if at least one match-play decision contributed
	SnowieGlobal        float64           `json:"SnowieGlobal"` // Snowie ER: 500×Σerr / (total checker moves, both players, forced included)
	PerTournament       []TournamentStats `json:"PerTournament"`
	PerMatch            []MatchStats      `json:"PerMatch"`
	CubeActionBreakdown []CubeActionStats `json:"CubeActionBreakdown"`
	ErrorHistogram      []ErrorBucket     `json:"ErrorHistogram"`
	TopBlunders         []BlunderEntry    `json:"TopBlunders"`
}

// ComputeStats aggregates performance metrics for the given filter.
// ComputeStats delegates to the storage StatsStore (the single production
// implementation, shared with the headless server). legacyComputeStats keeps
// the original SQL as the parity-test reference.
func (d *Database) ComputeStats(filter StatsFilter) (*StatsResult, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	r, err := d.store.Stats().Compute(context.Background(), "", toStorageStatsFilter(filter))
	if err != nil {
		return nil, err
	}
	return fromStorageStatsResult(r), nil
}

// SelectionSpec describes what the user selected in the stats panel (a click on
// a bar, point, or row). The frontend passes this to GetPositionIDsByStatsSelection
// to obtain the matching position IDs for navigation.
type SelectionSpec struct {
	Kind string // "all", "checker", "cube", "cube_action",
	// "error_bucket", "tournament", "match",
	// "last_n", "position", "top_blunders"
	CubeAction    string // "NoDouble" | "DoubleTake" | "DoublePass" | "TooGood"
	BucketMinMP   int    // inclusive
	BucketMaxMP   int    // exclusive; -1 = +∞
	TournamentID  int64
	MatchID       int64
	LastN         int
	PositionID    int64
	OnlyWithError bool // for "cube_action", "checker", "cube" → error > 0
}

// GetPositionIDsByStatsSelection resolves a user selection made in the Stats
// panel into a deduplicated list of position IDs. The StatsFilter is always
// applied so that the IDs correspond exactly to what is displayed in the panel
// (invariant: "ce qu'on clique = ce qu'on voit").
// GetPositionIDsByStatsSelection delegates to the storage StatsStore.
func (d *Database) GetPositionIDsByStatsSelection(filter StatsFilter, sel SelectionSpec) ([]int64, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.store.Stats().PositionIDsBySelection(context.Background(), "", toStorageStatsFilter(filter), toStorageSelectionSpec(sel))
}

// GetPositionIDsByTournament returns all position IDs belonging to the given
// tournament, regardless of any stats filter. Used when the user explicitly
// reopens a tournament (Open tournament action).
// GetPositionIDsByTournament delegates to the storage StatsStore.
func (d *Database) GetPositionIDsByTournament(tournamentID int64) ([]int64, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.store.Stats().PositionIDsByTournament(context.Background(), "", tournamentID)
}

// GetPositionIDsByMatch returns all position IDs belonging to the given match,
// regardless of any stats filter. Used when the user explicitly reopens a match.
// GetPositionIDsByMatch delegates to the storage StatsStore.
func (d *Database) GetPositionIDsByMatch(matchID int64) ([]int64, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.store.Stats().PositionIDsByMatch(context.Background(), "", matchID)
}

// PlayerFrequency pairs a player name with the number of matches in which they appear.
type PlayerFrequency struct {
	Name  string
	Count int
}

// GetAllPlayerNames returns all player names found in the match table, ranked by
// the total number of matches (player1 + player2 appearances) descending.
// Names that are equal in frequency are sorted alphabetically.
// GetAllPlayerNames delegates to the storage StatsStore.
func (d *Database) GetAllPlayerNames() ([]PlayerFrequency, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	p, err := d.store.Stats().PlayerNames(context.Background(), "")
	if err != nil {
		return nil, err
	}
	return fromStoragePlayerFreq(p), nil
}

// applyMatchBadges fills the per-player PR/MWC badge fields of every match in
// the slice from the storage backend. Callers already hold the read lock.
func (d *Database) applyMatchBadges(matches []Match) error {
	if len(matches) == 0 {
		return nil
	}
	badges, err := d.store.Stats().MatchBadges(context.Background(), "")
	if err != nil {
		return err
	}
	for i := range matches {
		if b, ok := badges[matches[i].ID]; ok {
			matches[i].PR = b.PR
			matches[i].MWCLoss = b.MWCLoss
			matches[i].PR2 = b.PR2
			matches[i].MWCLoss2 = b.MWCLoss2
		}
	}
	return nil
}

// applyTournamentBadges fills the aggregate PR/MWC badge fields of every
// tournament in the slice from the storage backend.
func (d *Database) applyTournamentBadges(tournaments []Tournament) error {
	if len(tournaments) == 0 {
		return nil
	}
	badges, err := d.store.Stats().TournamentBadges(context.Background(), "")
	if err != nil {
		return err
	}
	for i := range tournaments {
		if b, ok := badges[tournaments[i].ID]; ok {
			tournaments[i].PR = b.PR
			tournaments[i].MWCLoss = b.MWCLoss
		}
	}
	return nil
}

// MatchPlayerDetailStats holds per-player statistics for a single match.
type MatchPlayerDetailStats struct {
	// Overall
	TotalDecisions   int     `json:"total_decisions"`
	TotalErrors      int     `json:"total_errors"`
	TotalBlunders    int     `json:"total_blunders"`
	TotalEquityError float64 `json:"total_equity_error"` // sum of errors in EMG
	PR               float64 `json:"pr"`
	MWCLoss          float64 `json:"mwc_loss"`

	// Checker play
	CheckerDecisions   int     `json:"checker_decisions"`
	CheckerErrors      int     `json:"checker_errors"`
	CheckerBlunders    int     `json:"checker_blunders"`
	CheckerEquityError float64 `json:"checker_equity_error"` // in EMG
	PRChecker          float64 `json:"pr_checker"`
	CheckerMWCLoss     float64 `json:"checker_mwc_loss"`

	// Cube play: doubling decisions
	DoubleDecisions   int     `json:"double_decisions"`
	DoubleErrors      int     `json:"double_errors"`
	DoubleBlunders    int     `json:"double_blunders"`
	DoubleEquityError float64 `json:"double_equity_error"` // in EMG
	DoubleMWCLoss     float64 `json:"double_mwc_loss"`

	// Cube play: take/pass decisions
	TakeDecisions   int     `json:"take_decisions"`
	TakeErrors      int     `json:"take_errors"`
	TakeBlunders    int     `json:"take_blunders"`
	TakeEquityError float64 `json:"take_equity_error"` // in EMG
	TakeMWCLoss     float64 `json:"take_mwc_loss"`

	// Combined cube PR
	PRCube      float64 `json:"pr_cube"`
	CubeMWCLoss float64 `json:"cube_mwc_loss"`

	// Snowie Error Rate: 500 × Σerr(player) / (anTotalMoves[P1]+anTotalMoves[P2]).
	// Denominator = total checker moves for both players (forced included, cube excluded).
	// This is asymmetric per player (gnuBG formatgs.c:415-424 convention).
	SnowieER float64 `json:"snowie_er"`
}

// MatchDetailStats holds per-player statistics for a single match.
type MatchDetailStats struct {
	MatchID int64                  `json:"match_id"`
	Player1 MatchPlayerDetailStats `json:"player1"`
	Player2 MatchPlayerDetailStats `json:"player2"`
}

// GetMatchDetailStats computes per-player statistics for the given match.
// GetMatchDetailStats delegates to the storage StatsStore. legacyGetMatchDetailStats
// keeps the original SQL as the parity-test reference (pinned against eXtreme
// Gammon reference values in TestStatsParity).
func (d *Database) GetMatchDetailStats(matchID int64) (*MatchDetailStats, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	m, err := d.store.Stats().MatchDetail(context.Background(), "", matchID)
	if err != nil {
		return nil, err
	}
	return fromStorageMatchDetail(m), nil
}
