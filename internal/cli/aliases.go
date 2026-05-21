package cli

// Domain and persistence symbols are re-exported here so the CLI command
// files (moved out of package main during the headless refactor) keep
// compiling against the unqualified names they were written with.

import (
	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// ── domain constants ─────────────────────────────────────────────────────────

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

// ── domain types ─────────────────────────────────────────────────────────────

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

// ── persistence types ────────────────────────────────────────────────────────

type (
	Database               = database.Database
	BlunderEntry           = database.BlunderEntry
	Collection             = database.Collection
	CollectionPosition     = database.CollectionPosition
	CubeActionStats        = database.CubeActionStats
	ErrorBucket            = database.ErrorBucket
	MatchDetailStats       = database.MatchDetailStats
	MatchPlayerDetailStats = database.MatchPlayerDetailStats
	MatchStats             = database.MatchStats
	PlayerFrequency        = database.PlayerFrequency
	RawCubeAction          = database.RawCubeAction
	SearchHistory          = database.SearchHistory
	SelectionSpec          = database.SelectionSpec
	SessionState           = database.SessionState
	StatsDateRange         = database.StatsDateRange
	StatsFilter            = database.StatsFilter
	StatsResult            = database.StatsResult
	StatsTotals            = database.StatsTotals
	TournamentStats        = database.TournamentStats
)

// ── persistence functions and values ─────────────────────────────────────────

var (
	NewDatabase       = database.NewDatabase
	DeleteFile        = database.DeleteFile
	ErrDuplicateMatch = database.ErrDuplicateMatch

	ComputeMatchHash                   = database.ComputeMatchHash
	ComputeGnuBGMatchHash              = database.ComputeGnuBGMatchHash
	ComputeBGFMatchHash                = database.ComputeBGFMatchHash
	ComputeCanonicalMatchHashFromXG    = database.ComputeCanonicalMatchHashFromXG
	ComputeCanonicalMatchHashFromBGF   = database.ComputeCanonicalMatchHashFromBGF
	ComputeCanonicalMatchHashFromGnuBG = database.ComputeCanonicalMatchHashFromGnuBG
	ConvertEMGLossToMWCLoss            = database.ConvertEMGLossToMWCLoss
)
