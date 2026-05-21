package engine

import "github.com/kevung/blunderdb/pkg/blunderdb/domain"

// Test-only shorthands so the engine test files can use the unqualified
// domain type/constant names they were originally written against.
type (
	Position = domain.Position
	Board    = domain.Board
	Point    = domain.Point
	Cube     = domain.Cube
)

const (
	Black         = domain.Black
	White         = domain.White
	None          = domain.None
	BlackBar      = domain.BlackBar
	WhiteBar      = domain.WhiteBar
	CheckerAction = domain.CheckerAction
	CubeAction    = domain.CubeAction
)
