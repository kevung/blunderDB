package storage

import (
	"context"
	"sort"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// StatsFilter defines the filtering criteria for a stats computation.
type StatsFilter struct {
	PlayerName string
	// PlayerAliases are the other spellings the same person signed under
	// (names are typed by hand, so one person appears several ways). The
	// filter keeps the decisions of ANY of PlayerName + PlayerAliases. Merging
	// names in place (MergePlayers) would rewrite a shared database's matches.
	PlayerAliases []string
	TournamentIDs []int64
	DateFrom      string // ISO "YYYY-MM-DD"
	DateTo        string // ISO "YYYY-MM-DD"
	DecisionType  int    // -1=all, 0=checker, 1=cube
	MatchLength   []int
	// Provenance (ADR-0013): keep only the decisions analysed by this engine
	// (analysis.analysis_engine, exact) and at least this deep
	// (analysis.analysis_depth). Empty and 0 keep every analysis.
	AnalysisEngine   string `json:"AnalysisEngine,omitempty"`
	MinAnalysisDepth int    `json:"MinAnalysisDepth,omitempty"`
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
	// MWC7 pools the row's (match, player) units (ADR-0075).
	MWC7 domain.MWC7 `json:"MWC7"`
}

// MatchStats holds aggregated stats for a single match.
type MatchStats struct {
	ID           int64   `json:"ID"`
	Date         string  `json:"Date"`
	PlayerName   string  `json:"PlayerName"`
	PR           float64 `json:"PR"`
	MWC          float64 `json:"MWC"`
	NumDecisions int     `json:"NumDecisions"`
	// MWC7 pools the row's (match, player) units (ADR-0075).
	MWC7 domain.MWC7 `json:"MWC7"`
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
	Totals   StatsTotals `json:"Totals"`
	PRGlobal float64     `json:"PRGlobal"`
	// PRInterval is PRGlobal's 95 % interval over the selection's matches
	// (ADR-0078).
	PRInterval   domain.Interval `json:"PRInterval"`
	PRChecker    float64         `json:"PRChecker"`
	PRCube       float64         `json:"PRCube"`
	PRRolling    map[int]float64 `json:"PRRolling"`
	MWCGlobal    float64         `json:"MWCGlobal"`
	MWCChecker   float64         `json:"MWCChecker"`
	MWCCube      float64         `json:"MWCCube"`
	MWCRolling   map[int]float64 `json:"MWCRolling"`
	MWCAvailable bool            `json:"MWCAvailable"`
	// MWC7 pools every (match, player) unit of the selection
	// (ADR-0075); unavailable when the selection holds no match play.
	MWC7 domain.MWC7 `json:"MWC7"`
	// MWC7Checker and MWC7Cube split MWC7 by decision type over the same
	// units, so the two add up to it.
	MWC7Checker         domain.MWC7       `json:"MWC7Checker"`
	MWC7Cube            domain.MWC7       `json:"MWC7Cube"`
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

	// The breakdowns below are the SAME figures as the global ones, restricted
	// to a slice of the selection — never a second notion of what counts as a
	// decision.
	//
	// PerPhase splits by the position's derived game phase (ADR-0035).
	PerPhase []PhaseStats `json:"PerPhase"`
	// PerGameType splits by the position's derived plan of play.
	PerGameType []GameTypeStats `json:"PerGameType"`
	// PerTag splits by the tags in the position's comments. A position
	// carrying two tags appears in both rows: a tag is a label, not a
	// partition, so the rows deliberately do not sum to the total.
	PerTag []TagStats `json:"PerTag"`
	// PerScore is the away × away matrix — Crawford, post-Crawford, DMP and
	// everything between. Every cell is returned WITH its count and its
	// interval; one without an interval (a single match behind it) is greyed
	// by the caller rather than hidden, so the omission stays auditable.
	PerScore []ScoreCellStats `json:"PerScore"`
}

// PhaseStats is one row of the per-phase breakdown.
type PhaseStats struct {
	// Phase is the stable token of domain.GamePhase ("opening", "race", …).
	Phase string  `json:"Phase"`
	PR    float64 `json:"PR"`
	// PRInterval is PR's 95 % interval over the cell's matches (ADR-0078).
	PRInterval   domain.Interval `json:"PRInterval"`
	NumDecisions int             `json:"NumDecisions"`
	BlunderCount int             `json:"BlunderCount"`
}

// GameTypeStats is one row of the per-game-type breakdown.
type GameTypeStats struct {
	// GameType is the stable token of domain.GameType ("holding", "blitz", …).
	GameType string  `json:"GameType"`
	PR       float64 `json:"PR"`
	// PRInterval is PR's 95 % interval over the cell's matches (ADR-0078).
	PRInterval   domain.Interval `json:"PRInterval"`
	NumDecisions int             `json:"NumDecisions"`
	BlunderCount int             `json:"BlunderCount"`
}

// TagStats is one row of the per-tag breakdown. Tag carries the "#".
type TagStats struct {
	Tag string  `json:"Tag"`
	PR  float64 `json:"PR"`
	// PRInterval is PR's 95 % interval over the cell's matches (ADR-0078).
	PRInterval   domain.Interval `json:"PRInterval"`
	NumDecisions int             `json:"NumDecisions"`
	BlunderCount int             `json:"BlunderCount"`
}

// ScoreCellStats is one cell of the away × away matrix, keyed by (mover's
// away, opponent's away) from the reference player's side. Post-Crawford is read as one away
// (domain.PointsAway), as in ScoreBias; money play is the cell with Money set
// and both aways at 0.
type ScoreCellStats struct {
	Money        bool    `json:"Money"`
	MoverAway    int     `json:"MoverAway"`
	OpponentAway int     `json:"OpponentAway"`
	PR           float64 `json:"PR"`
	// PRInterval is PR's 95 % interval over the cell's matches (ADR-0078).
	PRInterval   domain.Interval `json:"PRInterval"`
	NumDecisions int             `json:"NumDecisions"`
	BlunderCount int             `json:"BlunderCount"`
}

// SelectionSpec selects a subset of positions out of a stats result, e.g. the
// decisions behind a histogram bucket or a tournament row.
type SelectionSpec struct {
	Kind string // "all","checker","cube","cube_action","cube_direction","error_bucket","tournament","match","last_n","position","top_blunders","breakdown"
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
	// Breakdown and BreakdownKey, for Kind "breakdown", name one row of the
	// breakdowns: Breakdown is one of the Breakdown* dimensions, BreakdownKey
	// the row's key as BreakdownPositionCounts lists it. OnlyBlunders keeps
	// the positions where a decision of the selection is a blunder.
	Breakdown    string
	BreakdownKey string
	OnlyBlunders bool
}

// The dimensions of the breakdowns a SelectionSpec of Kind "breakdown" names.
const (
	BreakdownPhase    = "phase"     // key: the GamePhase token
	BreakdownGameType = "game_type" // key: the GameType token
	BreakdownTag      = "tag"       // key: the tag, "#" included
	BreakdownScore    = "score"     // key: "money", or "<mover away>-<opponent away>"
)

// BreakdownPositionCount is what a breakdown row holds counted in distinct
// positions, the unit a drill-down loads: a position reached by several
// decisions counts once, where NumDecisions counts each decision.
type BreakdownPositionCount struct {
	Positions int `json:"Positions"`
	// Blunders counts the positions where at least one decision of the
	// selection is a blunder.
	Blunders int `json:"Blunders"`
}

// BreakdownPositionCounts maps a Breakdown* dimension to its rows' counts, by
// row key.
type BreakdownPositionCounts map[string]map[string]BreakdownPositionCount

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

	// MWC7 pools the player's (match, seat) units (ADR-0075); unavailable
	// for a player with no match play.
	MWC7 domain.MWC7 `json:"mwc7"`
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
	// MWC7 is MWCLoss rescaled to seven points (ADR-0075).
	MWC7 domain.MWC7 `json:"mwc7"`

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

// The two grades a Move of a match can carry in its Transcript, drawn at the
// library's thresholds (ADR-0046) as the statistics are. A play under the
// error threshold carries no grade.
const (
	MoveGradeError   = "error"
	MoveGradeBlunder = "blunder"
)

// MoveGrade is one Move of a match scored by its Position's analysis: the
// cost of THAT play, in millipoints, and the grade the library's thresholds
// give it. A Position deduplicated across a match (an opening reached twice)
// is scored once per Move, each by its own play — not by the denormalised
// column, which holds only the first. A Move whose play the analysis does not
// score (no analysis, a move absent from the candidates) has no MoveGrade.
type MoveGrade struct {
	MoveID  int64  `json:"move_id"`
	ErrorMP int    `json:"error_mp"`
	Grade   string `json:"grade"` // "", MoveGradeError or MoveGradeBlunder
}

// DecisionLoss is one Move of a match with the match winning chances its play
// cost, as a fraction (0.0123 is 1.23 %), the unit of Match.MWCLoss. Summed
// per player, the losses are that player's Match.MWCLoss / MWCLoss2. MWCLoss
// is nil for a Move that carries no figure (no analysis, a decision that does
// not count toward the statistics, a position the match equity table cannot
// price, money play): unscored is not a loss of zero.
type DecisionLoss struct {
	MoveID       int64    `json:"move_id"`
	GameNumber   int      `json:"game_number"`
	MoveNumber   int      `json:"move_number"`
	Player       int      `json:"player"`        // 0 is player 1, 1 is player 2
	DecisionType string   `json:"decision_type"` // "checker" or "cube"
	MWCLoss      *float64 `json:"mwc_loss"`
	// Difficulty is the loss a reference player expects in the same position
	// (ADR-0076), in the unit of MWCLoss; nil wherever MWCLoss is, or where the
	// analysis gives no option costs.
	Difficulty *float64 `json:"difficulty"`
	// Avoidable marks an error (the library's threshold) the reference player
	// would rarely make: Difficulty at most AvoidableShare of MWCLoss.
	Avoidable bool `json:"avoidable"`
	// ErrorMP is the counted decision's error in millipoints, nil outside
	// the counted set; Error says it reaches the library's error threshold.
	ErrorMP *int64 `json:"error_mp"`
	Error   bool   `json:"error"`
	// Luck is the roll's luck for the player who rolled, converted to MWC at
	// the position's score and cube like a loss (ADR-0078); nil when unknown,
	// on a cube row, or at money.
	Luck *float64 `json:"luck"`
	// Rolled marks a checker play that has its position: a roll whose luck
	// the analysis could have measured. A checker row without one never
	// carries luck, so it is not a roll the luck coverage misses.
	Rolled bool `json:"rolled"`
	// DurationMS is the time taken over the decision (ADR-0073), nil when
	// unknown.
	DurationMS *int64 `json:"duration_ms"`
	// Away is the position's stored away scores, player 1's first, the
	// Crawford rule inside them (CONTEXT.md, « Away score »).
	Away [2]int `json:"away"`
}

// PlayerTimeSummary is what one player's recorded decision times add up to
// (ADR-0073). Durations are milliseconds; a Move whose duration is unknown is
// counted in Unknown and in no total or mean, so an unknown is never a zero:
// a mean is meaningful only when its count is not zero.
type PlayerTimeSummary struct {
	TotalMS int64 `json:"total_ms"`
	// CheckerCount and CheckerTotalMS are the checker plays with a known time;
	// CubeCount and CubeTotalMS the cube decisions with one, whether taken
	// before a roll (the Move carries it) or as a double, a take or a pass.
	CheckerCount   int   `json:"checker_count"`
	CheckerTotalMS int64 `json:"checker_total_ms"`
	CubeCount      int   `json:"cube_count"`
	CubeTotalMS    int64 `json:"cube_total_ms"`
	Unknown        int   `json:"unknown"`
	// OverTime is true for the player whose reserve ran out, as the Duel
	// recorded it in the Match's origin. Only the first to run out is
	// recorded: the other reads false even if its reserve ran out later.
	OverTime bool `json:"over_time"`
}

// MatchTimeSummary is a Match's decision times, per player (index 0 is
// player 1). HasCadence says the Match was played here under a Cadence, so a
// false OverTime means "did not run out first" and not "not measured".
type MatchTimeSummary struct {
	HasCadence bool                 `json:"has_cadence"`
	Players    [2]PlayerTimeSummary `json:"players"`
}

// MatchTurn is one recorded Move of a Match as its clock sees it: the player
// (index 0 is player 1), and the durations kept, nil when unknown. A cube row
// (double, take, pass) carries its own decision in PlayMS; a checker row carries
// the cube decision before the roll in CubeMS and the play in PlayMS.
type MatchTurn struct {
	Player int    `json:"player"`
	Cube   bool   `json:"cube"`
	CubeMS *int64 `json:"cube_ms"`
	PlayMS *int64 `json:"play_ms"`
}

// MatchTurns are a Match's Moves in match order, with the score its first game
// started from, which is what a clock with a reserve per point counts from.
type MatchTurns struct {
	Score [2]int32    `json:"score"`
	Turns []MatchTurn `json:"turns"`
}

// TimeBucketBounds are the upper bounds, in milliseconds, of the first three
// duration bands of TimeErrorRow.Bucket: under 5 s, 5-15 s, 15-30 s; the
// fourth is everything longer.
var TimeBucketBounds = [3]int64{5000, 15000, 30000}

// TimeErrorRow is one player's decisions of one duration band: how many, how
// many of them the analysis scores, the mean equity they gave up and how many
// were blunders at the library's threshold. Only decisions with a known
// duration are counted: an unknown one belongs to no band. The error is scored
// from the Position's current analysis, not read from move.error_mp; a
// decision it cannot score (unanalysed, or a play absent from the candidates)
// is in Decisions and in neither Scored, MeanErrorMP nor Blunders.
type TimeErrorRow struct {
	Player      string  `json:"player"`
	Bucket      int     `json:"bucket"`
	Decisions   int     `json:"decisions"`
	Scored      int     `json:"scored"`
	MeanErrorMP float64 `json:"mean_error_mp"`
	Blunders    int     `json:"blunders"`
}

// MatchBadge is the per-player PR/MWC summary shown on each match-list row.
// PR/MWCLoss are player 1's, PR2/MWCLoss2 player 2's. It is the list-row
// projection of MatchDetailStats (badge.PR == detail.Player1.PR for a match).
type MatchBadge struct {
	PR       float64 `json:"pr"`
	MWCLoss  float64 `json:"mwc_loss"`
	PR2      float64 `json:"pr2"`
	MWCLoss2 float64 `json:"mwc_loss2"`
	// MWC7/MWC7P2 are MWCLoss/MWCLoss2 rescaled to seven points
	// (ADR-0075), with the interval of the match's games.
	MWC7   domain.MWC7 `json:"mwc7"`
	MWC7P2 domain.MWC7 `json:"mwc7_p2"`
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
	// MWC7 pools the reference player's matches of the tournament
	// (ADR-0075).
	MWC7 domain.MWC7 `json:"mwc7"`
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
	// MatchMWC and MatchLength split MWC by match, for the 7-point MWC loss; a
	// backend that leaves them nil gives the badge none.
	MatchMWC    map[int64]float64
	MatchLength map[int64]int
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
	ids := make([]int64, 0, len(best.MatchMWC))
	for id := range best.MatchMWC {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	var elo domain.MWC7Pool
	for _, id := range ids {
		elo.AddLoss(best.MatchMWC[id], best.MatchLength[id])
	}
	return TournamentBadge{PR: pr, MWCLoss: best.MWC, RefPlayer: bestName, MWC7: elo.Result()}
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

	// BreakdownPositionCounts counts, for every row of the breakdowns, the
	// positions a "breakdown" selection of that row returns: the figure and
	// the drill-down are one computation, so they cannot disagree.
	BreakdownPositionCounts(ctx context.Context, scope string, filter StatsFilter) (BreakdownPositionCounts, error)

	// PositionIDsByTournament returns the position ids of a tournament.
	PositionIDsByTournament(ctx context.Context, scope string, tournamentID int64) ([]int64, error)

	// PositionIDsByMatch returns the position ids of a match.
	PositionIDsByMatch(ctx context.Context, scope string, matchID int64) ([]int64, error)

	// AnalysisEngines returns the distinct engine labels the analyses of the
	// scope carry (the values StatsFilter.AnalysisEngine matches exactly),
	// sorted. Unlabelled analyses are left out.
	AnalysisEngines(ctx context.Context, scope string) ([]string, error)

	// PlayerNames returns every player name ranked by match frequency.
	PlayerNames(ctx context.Context, scope string) ([]PlayerFrequency, error)

	// PlayerTable computes one row per player over the matches the filter
	// retains, for the players table of the Stats panel.
	//
	// It honours the filter's date range, tournaments and match lengths only;
	// PlayerName/PlayerAliases and DecisionType are ignored by design (every
	// player at once, checker and cube in their own columns).
	//
	// Rows come back sorted by PR ascending, players with no counted decision
	// last. Nothing is hidden: the decision count says what a PR is worth.
	PlayerTable(ctx context.Context, scope string, filter StatsFilter) ([]PlayerRow, error)

	// MatchDetail computes per-player statistics for a single match.
	MatchDetail(ctx context.Context, scope string, matchID int64) (*MatchDetailStats, error)

	// MatchMoveGrades scores every Move of a match by its own play and grades
	// it at the library's thresholds — what the Match panel's Transcript
	// colours its rows with. Moves come back in Transcript order; a
	// Move the analysis does not score is absent.
	MatchMoveGrades(ctx context.Context, scope string, matchID int64) ([]MoveGrade, error)

	// MatchDecisionLosses lists every Move of a Match, in Transcript order,
	// with the winning chances it cost. It reads the rows the statistics read
	// and converts through the same function, so a player's losses add up to
	// the Match's MWCLoss (MWCLoss2 for player 2).
	MatchDecisionLosses(ctx context.Context, scope string, matchID int64) ([]DecisionLoss, error)

	// MatchReview is a match's study summary (ADR-0078): PR and L7 with
	// their intervals over the games, the errors worth revisiting, the
	// luck-adjusted result and the hasty/deliberate split of the errors.
	MatchReview(ctx context.Context, scope string, matchID int64) (MatchReview, error)

	// TournamentReview is one player's review of a tournament (ADR-0081):
	// L7 and PR against the player's usual level, by round, by decision rank,
	// at pressure scores and by pace, and the tournament's error families.
	// An empty player is the one who played the most of its matches. A
	// tournament of another tenant, or none, is ErrNotFound.
	TournamentReview(ctx context.Context, scope string, tournamentID int64, player string) (TournamentReview, error)

	// MatchTimeSummary adds up the decision times of a Match per player, and
	// counts what overran the Cadence it was played under (match_origin). A
	// Match with no recorded time gives a summary of unknowns.
	MatchTimeSummary(ctx context.Context, scope string, matchID int64) (MatchTimeSummary, error)

	// MatchTurns returns the Moves of a Match in match order with their
	// durations, the order GetMatchMovePositions gives them in.
	MatchTurns(ctx context.Context, scope string, matchID int64) (MatchTurns, error)

	// TimeErrors crosses the time a player took over a decision with the error
	// it cost: one row per player and duration band that holds a decision, the
	// players named as the Matches name them, the bands in order.
	TimeErrors(ctx context.Context, scope string) ([]TimeErrorRow, error)

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

	// RecurringErrors groups the filter's errors (counted decisions costing
	// at least the library's Error threshold) by plan of play and theme,
	// heaviest summed cost first. See GroupRecurringErrors.
	RecurringErrors(ctx context.Context, scope string, filter StatsFilter) (*RecurringErrors, error)

	// StudyPlan ranks the same families by the winning chances studying them
	// would recover, apart from those short of evidence (ADR-0077). See
	// BuildStudyPlan.
	StudyPlan(ctx context.Context, scope string, filter StatsFilter) (*StudyPlan, error)

	// SuggestReferences proposes the reference positions of a filter: those
	// whose lesson covers the most recoverable MWC of their neighbouring
	// errors, near-duplicates and handled positions left out (ADR-0080). See
	// SuggestReferences.
	SuggestReferences(ctx context.Context, scope string, req ReferenceRequest) (*ReferenceSuggestions, error)

	// StudyEffect measures each studied family's loss rate in real play
	// before and after the day it was first studied (ADR-0079). See
	// BuildStudyEffect.
	StudyEffect(ctx context.Context, scope string, filter StatsFilter) (*StudyEffect, error)

	// DirectionalBiases tallies which way the filter's decisions err: takes
	// against passes, premature against missed doubles by score, bolder
	// against more cautious plays (ADR-0079). See BuildDirectionalBiases.
	DirectionalBiases(ctx context.Context, scope string, filter StatsFilter) (*DirectionalBiases, error)

	// MatchStats returns the stored per-match, per-seat tallies of the given
	// matches (every match in scope when matchIDs is empty), two rows per
	// match, ordered by match then seat. A match whose rows are missing — a
	// fresh import, an invalidated match — is recomputed first, so the rows
	// always agree with the decisions they summarise.
	MatchStats(ctx context.Context, scope string, matchIDs []int64) ([]MatchStatsRow, error)

	// MatchSeries is Compute's PerMatch (ID, Date, PR, NumDecisions; MWC
	// left zero) read from match_stats: the PR of the filter's players in
	// each match, oldest first, matches without a counted decision absent.
	// It honours every field of the filter, like Compute: a provenance
	// filter, which the table cannot apply (it keeps one provenance per seat,
	// not per decision), and a connection that refuses writes are answered
	// by Compute's direct per-match pass over the decisions.
	MatchSeries(ctx context.Context, scope string, filter StatsFilter) ([]MatchStats, error)

	// HeadToHead is the record of two players against each other: the
	// matches where one sat against the other (aliases of neither are
	// folded), each player's PR in each of them and over all of them, and
	// the outcomes (MatchOutcome). The filter's player fields are ignored;
	// its tournaments, dates, match lengths and decision type apply. A
	// provenance filter is refused with ErrInvalid. A connection that refuses
	// writes computes the seat rows from the decisions instead of the table.
	HeadToHead(ctx context.Context, scope, playerA, playerB string, filter StatsFilter) (*HeadToHead, error)

	// PlayerContrast lists the positions both players decided (as the one who
	// took the decision; aliases folded) where one played well and the other
	// did not, at the library's Error threshold. The filter's player fields
	// are ignored; its tournaments, dates, match lengths, decision type and
	// provenance apply. Two players that are one person are ErrInvalid.
	PlayerContrast(ctx context.Context, scope, playerA, playerB string, filter StatsFilter) (*PlayerContrast, error)

	// PRByWindow is the PR of the filter's players over a sliding calendar
	// window of `months` months (1 a month, 3 a quarter), one point per
	// month from the first month with a counted decision to the last, each
	// point covering that month and the months-1 before it. Read from
	// match_stats (or, read-only, from the decisions as HeadToHead does); a
	// provenance filter is refused with ErrInvalid.
	PRByWindow(ctx context.Context, scope string, filter StatsFilter, months int) ([]WindowStats, error)

	// RefreshMatchStats recomputes the rows of matchIDs now — what an import
	// does for the match it has just written, inside its transaction.
	RefreshMatchStats(ctx context.Context, scope string, matchIDs []int64) error

	// FillMatchStats computes the rows of every match in scope that has none,
	// in committed batches: an interrupted run resumes where it stopped.
	// progress, when not nil, is told done/total after each batch. It returns
	// the number of matches computed.
	FillMatchStats(ctx context.Context, scope string, progress func(done, total int)) (int, error)

	// RebuildMatchStats discards every stored row of the scope and computes
	// them all again (repair --stats). It returns the number of matches.
	RebuildMatchStats(ctx context.Context, scope string, progress func(done, total int)) (int, error)
}

// MatchStatsRow is one seat of one match as the match_stats table holds it:
// the same counted decisions, error column and blunder threshold as Compute,
// so a PR read here equals the PR Compute derives for that seat's decisions.
// Seat 1 is the match's player1, seat 2 its player2; the table stores no
// name, so a renamed player keeps his rows. AnalysisEngine / AnalysisDepth
// are the provenance most of the seat's counted decisions carry ("" / 0 when
// none does).
type MatchStatsRow struct {
	MatchID          int64   `json:"match_id"`
	Seat             int     `json:"seat"`
	Decisions        int     `json:"decisions"`
	CheckerDecisions int     `json:"checker_decisions"`
	CubeDecisions    int     `json:"cube_decisions"`
	ErrorMP          int64   `json:"error_mp"`
	PR               float64 `json:"pr"` // 0 when Decisions is 0 (NULL in the table)
	LuckMP           int64   `json:"luck_mp"`
	LuckRolls        int     `json:"luck_rolls"`
	Blunders         int     `json:"blunders"`
	AnalysisEngine   string  `json:"analysis_engine"`
	AnalysisDepth    int     `json:"analysis_depth"`
	CheckerErrorMP   int64   `json:"checker_error_mp"`
	CubeErrorMP      int64   `json:"cube_error_mp"`
	Errors           int     `json:"errors"`
	// The Snowie rate's parts, outside the counted predicate: the seat's
	// error over every analysed decision, its analysed checker decisions,
	// and its checker decisions analysed or not.
	SnowieErrorMP int64 `json:"snowie_error_mp"`
	SnowieMoves   int   `json:"snowie_moves"`
	CheckerMoves  int   `json:"checker_moves"`
}

// HeadToHead is the record of two players against each other (StatsStore.HeadToHead).
type HeadToHead struct {
	PlayerA    string            `json:"player_a"`
	PlayerB    string            `json:"player_b"`
	Matches    []HeadToHeadMatch `json:"matches"`
	WinsA      int               `json:"wins_a"`
	WinsB      int               `json:"wins_b"`
	DecisionsA int               `json:"decisions_a"`
	DecisionsB int               `json:"decisions_b"`
	PRA        float64           `json:"pr_a"`
	PRB        float64           `json:"pr_b"`
}

// HeadToHeadMatch is one match of a HeadToHead, A's and B's sides whatever
// their seats. Outcome is MatchOutcome from A's side: 1 A won, -1 B won, 0
// undecided.
type HeadToHeadMatch struct {
	ID          int64   `json:"id"`
	Date        string  `json:"date"`
	MatchLength int     `json:"match_length"`
	Outcome     int     `json:"outcome"`
	DecisionsA  int     `json:"decisions_a"`
	DecisionsB  int     `json:"decisions_b"`
	PRA         float64 `json:"pr_a"`
	PRB         float64 `json:"pr_b"`
}

// WindowStats is one point of StatsStore.PRByWindow: the months From..To
// ("YYYY-MM", both included). A window without a counted decision has
// NumDecisions 0 and PR 0, which the reader shows as no value.
type WindowStats struct {
	From         string  `json:"from"`
	To           string  `json:"to"`
	NumMatches   int     `json:"num_matches"`
	NumDecisions int     `json:"num_decisions"`
	PR           float64 `json:"pr"`
}

// RankedPlayer is a PlayerRow with its rank in RankPlayers.
type RankedPlayer struct {
	Rank int `json:"rank"`
	PlayerRow
}

// RankPlayers ranks the players with at least minDecisions counted
// decisions by PR, the lowest first; equal PRs share a rank (1, 1, 3) and
// are listed by name. A PR over few decisions is noise, which is what the
// floor keeps out of the ranking.
func RankPlayers(rows []PlayerRow, minDecisions int) []RankedPlayer {
	out := make([]RankedPlayer, 0, len(rows))
	for _, r := range rows {
		if r.Decisions > 0 && r.Decisions >= minDecisions {
			out = append(out, RankedPlayer{PlayerRow: r})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].PR != out[j].PR {
			return out[i].PR < out[j].PR
		}
		return out[i].Name < out[j].Name
	})
	for i := range out {
		if i > 0 && out[i].PR == out[i-1].PR {
			out[i].Rank = out[i-1].Rank
		} else {
			out[i].Rank = i + 1
		}
	}
	return out
}
