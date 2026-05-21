package main

// The domain types and constants now live in
// github.com/kevung/blunderdb/pkg/blunderdb/domain. This file re-exports them
// into package main so the existing GUI/CLI code (and the Wails bindings)
// keep compiling unchanged while the headless refactor is in progress.

import "github.com/kevung/blunderdb/pkg/blunderdb/domain"

const (
	NumPoints = domain.NumPoints
	BlackBar  = domain.BlackBar
	WhiteBar  = domain.WhiteBar
	None      = domain.None
	Black     = domain.Black
	White     = domain.White

	Unlimited    = domain.Unlimited
	PostCrawford = domain.PostCrawford
	Crawford     = domain.Crawford

	CheckerAction = domain.CheckerAction
	CubeAction    = domain.CubeAction

	NoDouble = domain.NoDouble
	Double   = domain.Double
	ReDouble = domain.ReDouble
	TooGood  = domain.TooGood
	Take     = domain.Take
	Pass     = domain.Pass
	Beaver   = domain.Beaver

	DatabaseVersion = domain.DatabaseVersion

	AnkiSourceCollection = domain.AnkiSourceCollection
	AnkiSourceSearch     = domain.AnkiSourceSearch
)

type (
	AnkiDeck             = domain.AnkiDeck
	AnkiCard             = domain.AnkiCard
	AnkiReviewCard       = domain.AnkiReviewCard
	AnkiDeckStats        = domain.AnkiDeckStats
	Tournament           = domain.Tournament
	CommentEntry         = domain.CommentEntry
	Point                = domain.Point
	Cube                 = domain.Cube
	Board                = domain.Board
	Position             = domain.Position
	SearchFilters        = domain.SearchFilters
	ExportOptions        = domain.ExportOptions
	DoublingCubeAnalysis = domain.DoublingCubeAnalysis
	CheckerMove          = domain.CheckerMove
	CheckerAnalysis      = domain.CheckerAnalysis
	PositionAnalysis     = domain.PositionAnalysis
	Match                = domain.Match
	Game                 = domain.Game
	Move                 = domain.Move
	MoveAnalysis         = domain.MoveAnalysis
	MatchMovePosition    = domain.MatchMovePosition
)

// InitializePosition returns the standard starting position.
func InitializePosition() Position {
	return domain.InitializePosition()
}
