package main

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
	DatabaseVersion = "1.0.0"
)

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
