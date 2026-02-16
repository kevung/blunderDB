package main

import "time"

const (
	NumPoints = 24
	BlackBar  = 25
	WhiteBar  = 0
	None      = -1
	Black     = 0
	White     = 1
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
	DatabaseVersion = "1.6.0"
)

// Tournament represents a tournament that organizes matches
type Tournament struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Date       string `json:"date"`
	Location   string `json:"location"`
	SortOrder  int    `json:"sortOrder"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
	MatchCount int    `json:"matchCount"`
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

type DoublingCubeAnalysis struct {
	AnalysisDepth             string  `json:"analysisDepth"`
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
	PositionID            int                   `json:"positionId"`
	XGID                  string                `json:"xgid"`
	Player1               string                `json:"player1"`
	Player2               string                `json:"player2"`
	AnalysisType          string                `json:"analysisType"`
	AnalysisEngineVersion string                `json:"analysisEngineVersion"`
	DoublingCubeAnalysis  *DoublingCubeAnalysis `json:"doublingCubeAnalysis,omitempty"`
	CheckerAnalysis       *CheckerAnalysis      `json:"checkerAnalysis,omitempty"`
	PlayedMove            string                `json:"playedMove,omitempty"`        // Deprecated: Use PlayedMoves instead
	PlayedCubeAction      string                `json:"playedCubeAction,omitempty"`  // Deprecated: Use PlayedCubeActions instead
	PlayedMoves           []string              `json:"playedMoves,omitempty"`       // All moves played in this position across different matches
	PlayedCubeActions     []string              `json:"playedCubeActions,omitempty"` // All cube actions taken in this position across different matches
	CreationDate          time.Time             `json:"creationDate"`
	LastModifiedDate      time.Time             `json:"lastModifiedDate"`
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

// Match-related structures for XG import

type Match struct {
	ID           int64     `json:"id"`
	Player1Name  string    `json:"player1_name"`
	Player2Name  string    `json:"player2_name"`
	Event        string    `json:"event"`
	Location     string    `json:"location"`
	Round        string    `json:"round"`
	MatchLength  int32     `json:"match_length"`
	MatchDate    time.Time `json:"match_date"`
	ImportDate   time.Time `json:"import_date"`
	FilePath     string    `json:"file_path"`
	GameCount    int       `json:"game_count"`
	TournamentID *int64    `json:"tournament_id,omitempty"`
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
