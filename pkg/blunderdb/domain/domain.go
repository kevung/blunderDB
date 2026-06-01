// Package domain holds the core backgammon domain types shared across the
// blunderDB engine, persistence layer, CLI and GUI. It has no dependencies
// beyond the standard library so it can be imported by any other package.
package domain

import "time"

const (
	NumPoints = 24
	BlackBar  = 25
	WhiteBar  = 0
	None      = -1
	Black     = 0
	White     = 1
	// ExcludeEmpty marks, in an exclusion ("Sauf") structure only, a point that must
	// contain NO checker of any colour. Stored as a board Point{Checkers:1, Color:ExcludeEmpty}.
	ExcludeEmpty = 2
)

const (
	Unlimited    = -1
	PostCrawford = 0
	Crawford     = 1
)

const (
	CheckerAction = iota
	CubeAction
)

const (
	NoDouble = iota
	Double
	ReDouble
	TooGood
	Take
	Pass
	Beaver
)

const (
	DatabaseVersion = "2.8.0"
)

// Anki deck source types
const (
	AnkiSourceCollection = "collection"
	AnkiSourceSearch     = "search"
)

// AnkiDeck represents a spaced repetition deck
type AnkiDeck struct {
	ID               int64   `json:"id"`
	Name             string  `json:"name"`
	Description      string  `json:"description"`
	SourceType       string  `json:"sourceType"`       // "collection" or "search"
	SourceID         int64   `json:"sourceId"`         // collection ID (if source is collection)
	SourceCommand    string  `json:"sourceCommand"`    // search command (if source is search)
	RequestRetention float64 `json:"requestRetention"` // target retention rate (0.7-0.99)
	MaximumInterval  float64 `json:"maximumInterval"`  // max days between reviews
	EnableFuzz       bool    `json:"enableFuzz"`       // add randomness to intervals
	CardCount        int     `json:"cardCount"`        // total cards
	DueCount         int     `json:"dueCount"`         // cards due for review
	NewCount         int     `json:"newCount"`         // new cards not yet reviewed
	CreatedAt        string  `json:"createdAt"`
	UpdatedAt        string  `json:"updatedAt"`
}

// AnkiCard represents a single FSRS card linked to a position
type AnkiCard struct {
	ID            int64   `json:"id"`
	DeckID        int64   `json:"deckId"`
	PositionID    int64   `json:"positionId"`
	Due           string  `json:"due"`
	Stability     float64 `json:"stability"`
	Difficulty    float64 `json:"difficulty"`
	ElapsedDays   int     `json:"elapsedDays"`
	ScheduledDays int     `json:"scheduledDays"`
	Reps          int     `json:"reps"`
	Lapses        int     `json:"lapses"`
	State         int     `json:"state"` // 0=New, 1=Learning, 2=Review, 3=Relearning
	LastReview    string  `json:"lastReview"`
}

// AnkiReviewCard is the card plus position data sent to the frontend for review
type AnkiReviewCard struct {
	Card     AnkiCard `json:"card"`
	Position Position `json:"position"`
}

// AnkiDeckStats holds review statistics
type AnkiDeckStats struct {
	NewCount      int `json:"newCount"`
	LearningCount int `json:"learningCount"`
	ReviewCount   int `json:"reviewCount"`
	DueCount      int `json:"dueCount"`
	TotalCount    int `json:"totalCount"`
}

// Tournament represents a tournament that organizes matches
type Tournament struct {
	ID         int64   `json:"id"`
	Name       string  `json:"name"`
	Date       string  `json:"date"`
	Location   string  `json:"location"`
	SortOrder  int     `json:"sortOrder"`
	CreatedAt  string  `json:"createdAt"`
	UpdatedAt  string  `json:"updatedAt"`
	MatchCount int     `json:"matchCount"`
	Comment    string  `json:"comment"`
	PR         float64 `json:"pr"`
	MWCLoss    float64 `json:"mwc_loss"`
}

// CommentEntry represents a comment for display in the comment wall
type CommentEntry struct {
	ID         int64  `json:"id"`
	PositionID int64  `json:"positionId"`
	Text       string `json:"text"`
	CreatedAt  string `json:"createdAt"`
	ModifiedAt string `json:"modifiedAt"`
}

type Point struct {
	Checkers int `json:"checkers"`
	Color    int `json:"color"`
}

type Cube struct {
	Owner int `json:"owner"`
	Value int `json:"value"`
}

type Board struct {
	Points  [NumPoints + 2]Point `json:"points"`
	Bearoff [2]int               `json:"bearoff"`
}

type Position struct {
	ID           int64  `json:"id"` // Add ID field
	Board        Board  `json:"board"`
	Cube         Cube   `json:"cube"`
	Dice         [2]int `json:"dice"`
	Score        [2]int `json:"score"`
	PlayerOnRoll int    `json:"player_on_roll"`
	DecisionType int    `json:"decision_type"`
	HasJacoby    int    `json:"has_jacoby"` // Add HasJacoby field
	HasBeaver    int    `json:"has_beaver"` // Add HasBeaver field
}

// SearchFilters bundles all filter parameters for LoadPositionsByFilters.
type SearchFilters struct {
	Filter                        Position `json:"filter"`
	ExcludeFilter                 Position `json:"excludeFilter"`
	IncludeCube                   bool     `json:"includeCube"`
	IncludeScore                  bool     `json:"includeScore"`
	PipCountFilter                string   `json:"pipCountFilter"`
	WinRateFilter                 string   `json:"winRateFilter"`
	GammonRateFilter              string   `json:"gammonRateFilter"`
	BackgammonRateFilter          string   `json:"backgammonRateFilter"`
	Player2WinRateFilter          string   `json:"player2WinRateFilter"`
	Player2GammonRateFilter       string   `json:"player2GammonRateFilter"`
	Player2BackgammonRateFilter   string   `json:"player2BackgammonRateFilter"`
	Player1CheckerOffFilter       string   `json:"player1CheckerOffFilter"`
	Player2CheckerOffFilter       string   `json:"player2CheckerOffFilter"`
	Player1BackCheckerFilter      string   `json:"player1BackCheckerFilter"`
	Player2BackCheckerFilter      string   `json:"player2BackCheckerFilter"`
	Player1CheckerInZoneFilter    string   `json:"player1CheckerInZoneFilter"`
	Player2CheckerInZoneFilter    string   `json:"player2CheckerInZoneFilter"`
	SearchText                    string   `json:"searchText"`
	Player1AbsolutePipCountFilter string   `json:"player1AbsolutePipCountFilter"`
	EquityFilter                  string   `json:"equityFilter"`
	DecisionTypeFilter            bool     `json:"decisionTypeFilter"`
	DiceRollFilter                bool     `json:"diceRollFilter"`
	DiceRollMode                  string   `json:"diceRollMode"`
	MovePatternFilter             string   `json:"movePatternFilter"`
	DateFilter                    string   `json:"dateFilter"`
	Player1OutfieldBlotFilter     string   `json:"player1OutfieldBlotFilter"`
	Player2OutfieldBlotFilter     string   `json:"player2OutfieldBlotFilter"`
	Player1JanBlotFilter          string   `json:"player1JanBlotFilter"`
	Player2JanBlotFilter          string   `json:"player2JanBlotFilter"`
	NoContactFilter               bool     `json:"noContactFilter"`
	MirrorFilter                  bool     `json:"mirrorFilter"`
	MoveErrorFilter               string   `json:"moveErrorFilter"`
	MatchIDsFilter                string   `json:"matchIDsFilter"`
	TournamentIDsFilter           string   `json:"tournamentIDsFilter"`
	RestrictToPositionIDs         string   `json:"restrictToPositionIDs"`
}

// ExportOptions bundles all parameters for ExportDatabase.
type ExportOptions struct {
	ExportPath           string            `json:"exportPath"`
	Positions            []Position        `json:"positions"`
	Metadata             map[string]string `json:"metadata"`
	IncludeAnalysis      bool              `json:"includeAnalysis"`
	IncludeComments      bool              `json:"includeComments"`
	IncludeFilterLibrary bool              `json:"includeFilterLibrary"`
	IncludePlayedMoves   bool              `json:"includePlayedMoves"`
	IncludeMatches       bool              `json:"includeMatches"`
	IncludeCollections   bool              `json:"includeCollections"`
	CollectionIDs        []int64           `json:"collectionIDs"`
	MatchIDs             []int64           `json:"matchIDs"`
	TournamentIDs        []int64           `json:"tournamentIDs"`
}

type DoublingCubeAnalysis struct {
	AnalysisDepth             string  `json:"analysisDepth"`
	AnalysisEngine            string  `json:"analysisEngine,omitempty"`
	PlayerWinChances          float64 `json:"playerWinChances"`
	PlayerGammonChances       float64 `json:"playerGammonChances"`
	PlayerBackgammonChances   float64 `json:"playerBackgammonChances"`
	OpponentWinChances        float64 `json:"opponentWinChances"`
	OpponentGammonChances     float64 `json:"opponentGammonChances"`
	OpponentBackgammonChances float64 `json:"opponentBackgammonChances"`
	CubelessNoDoubleEquity    float64 `json:"cubelessNoDoubleEquity"`
	CubelessDoubleEquity      float64 `json:"cubelessDoubleEquity"`
	CubefulNoDoubleEquity     float64 `json:"cubefulNoDoubleEquity"`
	CubefulNoDoubleError      float64 `json:"cubefulNoDoubleError"`
	CubefulDoubleTakeEquity   float64 `json:"cubefulDoubleTakeEquity"`
	CubefulDoubleTakeError    float64 `json:"cubefulDoubleTakeError"`
	CubefulDoublePassEquity   float64 `json:"cubefulDoublePassEquity"`
	CubefulDoublePassError    float64 `json:"cubefulDoublePassError"`
	BestCubeAction            string  `json:"bestCubeAction"`
	WrongPassPercentage       float64 `json:"wrongPassPercentage"`
	WrongTakePercentage       float64 `json:"wrongTakePercentage"`
}

type CheckerMove struct {
	Index                    int      `json:"index"`
	AnalysisDepth            string   `json:"analysisDepth"`
	AnalysisEngine           string   `json:"analysisEngine,omitempty"`
	Move                     string   `json:"move"`
	Equity                   float64  `json:"equity"`
	EquityError              *float64 `json:"equityError,omitempty"`
	PlayerWinChance          float64  `json:"playerWinChance"`
	PlayerGammonChance       float64  `json:"playerGammonChance"`
	PlayerBackgammonChance   float64  `json:"playerBackgammonChance"`
	OpponentWinChance        float64  `json:"opponentWinChance"`
	OpponentGammonChance     float64  `json:"opponentGammonChance"`
	OpponentBackgammonChance float64  `json:"opponentBackgammonChance"`
}

type CheckerAnalysis struct {
	Moves []CheckerMove `json:"moves"`
}

type PositionAnalysis struct {
	PositionID            int                    `json:"positionId"`
	XGID                  string                 `json:"xgid"`
	Player1               string                 `json:"player1"`
	Player2               string                 `json:"player2"`
	AnalysisType          string                 `json:"analysisType"`
	AnalysisEngineVersion string                 `json:"analysisEngineVersion"`
	DoublingCubeAnalysis  *DoublingCubeAnalysis  `json:"doublingCubeAnalysis,omitempty"`
	AllCubeAnalyses       []DoublingCubeAnalysis `json:"allCubeAnalyses,omitempty"`
	CheckerAnalysis       *CheckerAnalysis       `json:"checkerAnalysis,omitempty"`
	PlayedMove            string                 `json:"playedMove,omitempty"`        // Deprecated: Use PlayedMoves instead
	PlayedCubeAction      string                 `json:"playedCubeAction,omitempty"`  // Deprecated: Use PlayedCubeActions instead
	PlayedMoves           []string               `json:"playedMoves,omitempty"`       // All moves played in this position across different matches
	PlayedCubeActions     []string               `json:"playedCubeActions,omitempty"` // All cube actions taken in this position across different matches
	CreationDate          time.Time              `json:"creationDate"`
	LastModifiedDate      time.Time              `json:"lastModifiedDate"`
}

func initializeBoard() Board {
	var board Board

	board.Points[1] = Point{2, White}
	board.Points[12] = Point{5, White}
	board.Points[17] = Point{3, White}
	board.Points[19] = Point{5, White}

	board.Points[24] = Point{2, Black}
	board.Points[13] = Point{5, Black}
	board.Points[8] = Point{3, Black}
	board.Points[6] = Point{5, Black}
	return board
}

func InitializePosition() Position {
	var position Position

	position.Board = initializeBoard()
	position.Cube = Cube{None, 0}
	position.Dice = [2]int{3, 1}
	position.Score = [2]int{7, 7}
	position.PlayerOnRoll = Black
	position.DecisionType = CheckerAction

	return position
}

func (p *Position) MatchesCheckerPosition(filter Position) bool {
	for i := 0; i < len(p.Board.Points); i++ {
		if filter.Board.Points[i].Checkers > 0 {
			if p.Board.Points[i].Color != filter.Board.Points[i].Color || p.Board.Points[i].Checkers < filter.Board.Points[i].Checkers {
				return false
			}
		}
	}
	return true
}

// EffectiveIncludeFilter returns a copy of the include ("At least") filter with
// board points cleared where the exclude ("Sauf") filter contradicts the include.
//
// On a shared point with the same color, the include requires ≥I checkers and the
// exclude rejects ≥E checkers (i.e. keeps ≤E-1). These are compatible when I < E
// (the result is the range [I, E-1] — e.g. include 2, exclude 3 ⇒ exactly 2, a
// made point with no spare) and the include is kept. They contradict when I ≥ E
// (e.g. include 2, exclude 1 ⇒ no position has ≥2 and ≤0); there the exclusion
// wins and the include constraint on that point is dropped, so a closed board on
// 1-6 with a checker excepted on 1 searches 2-6 with point 1 free.
func EffectiveIncludeFilter(include, exclude Position) Position {
	result := include // [26]Point array → value copy, safe to mutate
	for i := range result.Board.Points {
		ep := exclude.Board.Points[i]
		ip := include.Board.Points[i]
		if ep.Checkers <= 0 || ep.Color < 0 || ip.Checkers <= 0 {
			continue
		}
		// A "must be empty" marker always wins; otherwise the exclude wins only when
		// its count is ≤ the include count (same colour) — a genuine contradiction.
		if ep.Color == ExcludeEmpty || (ip.Color == ep.Color && ip.Checkers >= ep.Checkers) {
			result.Board.Points[i] = Point{}
		}
	}
	return result
}

// ContainsAnyCheckerOf reports whether the position contains ANY of the checker
// elements described by filter — i.e. for at least one occupied filter point, the
// position has the same color and at least as many checkers. This is the
// "Sauf"/exclusion predicate: a position is rejected from search results when it
// contains any one of the excluded elements (OR semantics across points).
func (p *Position) ContainsAnyCheckerOf(filter Position) bool {
	for i := 0; i < len(p.Board.Points); i++ {
		fp := filter.Board.Points[i]
		if fp.Checkers <= 0 || fp.Color < 0 {
			continue
		}
		pp := p.Board.Points[i]
		if fp.Color == ExcludeEmpty {
			// "Must be empty" marker: reject if the point holds any checker.
			if pp.Checkers > 0 {
				return true
			}
			continue
		}
		if pp.Color == fp.Color && pp.Checkers >= fp.Checkers {
			return true
		}
	}
	return false
}

// Match-related structures for XG import

type Match struct {
	ID                  int64     `json:"id"`
	Player1Name         string    `json:"player1_name"`
	Player2Name         string    `json:"player2_name"`
	Event               string    `json:"event"`
	Location            string    `json:"location"`
	Round               string    `json:"round"`
	MatchLength         int32     `json:"match_length"`
	MatchDate           time.Time `json:"match_date"`
	ImportDate          time.Time `json:"import_date"`
	FilePath            string    `json:"file_path"`
	GameCount           int       `json:"game_count"`
	TournamentID        *int64    `json:"tournament_id,omitempty"`
	TournamentName      string    `json:"tournament_name"`
	LastVisitedPosition int       `json:"last_visited_position"`
	Comment             string    `json:"comment"`
	TournamentSortOrder int       `json:"tournament_sort_order"`
	PR                  float64   `json:"pr"`
	MWCLoss             float64   `json:"mwc_loss"`
	PR2                 float64   `json:"pr2"`
	MWCLoss2            float64   `json:"mwc_loss2"`
	// MatchHash is the format-specific content hash; CanonicalHash is the
	// format-independent hash used for cross-format duplicate detection. Both
	// are set at import time and used by MatchStore dedup. Empty when unknown.
	MatchHash     string `json:"match_hash,omitempty"`
	CanonicalHash string `json:"canonical_hash,omitempty"`
}

type Game struct {
	ID           int64    `json:"id"`
	MatchID      int64    `json:"match_id"`
	GameNumber   int32    `json:"game_number"`
	InitialScore [2]int32 `json:"initial_score"`
	Winner       int32    `json:"winner"`
	PointsWon    int32    `json:"points_won"`
	MoveCount    int      `json:"move_count"`
}

type Move struct {
	ID          int64    `json:"id"`
	GameID      int64    `json:"game_id"`
	MoveNumber  int32    `json:"move_number"`
	MoveType    string   `json:"move_type"` // "checker" or "cube"
	PositionID  int64    `json:"position_id"`
	Player      int32    `json:"player"`
	Dice        [2]int32 `json:"dice"`
	CheckerMove string   `json:"checker_move,omitempty"`
	CubeAction  string   `json:"cube_action,omitempty"`
}

type MoveAnalysis struct {
	ID                     int64   `json:"id"`
	MoveID                 int64   `json:"move_id"`
	AnalysisType           string  `json:"analysis_type"` // "checker" or "cube"
	Depth                  string  `json:"depth"`
	Equity                 float64 `json:"equity"`
	EquityError            float64 `json:"equity_error"`
	WinRate                float64 `json:"win_rate"`
	GammonRate             float64 `json:"gammon_rate"`
	BackgammonRate         float64 `json:"backgammon_rate"`
	OpponentWinRate        float64 `json:"opponent_win_rate"`
	OpponentGammonRate     float64 `json:"opponent_gammon_rate"`
	OpponentBackgammonRate float64 `json:"opponent_backgammon_rate"`
}

// MatchMovePosition combines position data with match context
type MatchMovePosition struct {
	Position     Position `json:"position"`       // The position (stored from player on roll POV)
	MoveID       int64    `json:"move_id"`        // Move ID
	GameID       int64    `json:"game_id"`        // Game ID
	GameNumber   int32    `json:"game_number"`    // Game number in match
	MoveNumber   int32    `json:"move_number"`    // Move number in game
	MoveType     string   `json:"move_type"`      // Move type: "checker" or "cube"
	PlayerOnRoll int32    `json:"player_on_roll"` // Player who rolled (0=Player1, 1=Player2)
	Player1Name  string   `json:"player1_name"`   // Player 1 name for reference
	Player2Name  string   `json:"player2_name"`   // Player 2 name for reference
	CheckerMove  string   `json:"checker_move"`   // The checker move played in this specific position
	CubeAction   string   `json:"cube_action"`    // The cube action taken in this specific position
}
