package backend

const (
    NumPoints = 24
    BlackBar = 25
    WhiteBar = 0
    None = -1
    Black = 0
    White = 1
)

const (
    Unlimited = -1
    PostCrawford = 0
    Crawford = 1
)

const (
    CheckerAction = iota
    CubeAction
)

type Point struct {
    Checkers int `json:"checkers"`
    Color int `json:"color"`
}

type Cube struct {
    Owner int `json:"owner"`
    Value int `json:"value"`
}

type Board struct {
    Points [NumPoints+2]Point `json:"points"`
    Bearoff [2]int `json:"bearoff"`
}

type Position struct {
    Board Board `json:"board"`
    Cube Cube `json:"cube"`
    Dice [2]int `json:"dice"`
    Score [2]int `json:"score"`
    PlayerOnRoll int `json:"player_on_roll"`
    DecisionType int `json:"decision_type"`
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
