// Package cli implements the blunderDB command-line interface: one
// `blunderdb <command>` per file (cli_<cmd>.go), sharing the CLI type and its
// *database.Database handle. handlers() in cli.go is the single source of
// truth for what counts as a subcommand — main.go's mode dispatch asks
// IsCommand instead of keeping a second list, and CommandNames() is the
// exported, sorted view cmd/cli-doc-gen walks to keep CLI_USAGE.md's flag
// reference from drifting. Two subcommands (collection, anki) are
// themselves small dispatchers over their own sub-command table, wired the
// same way for the same reason: a sub-command missing from the table is a
// sub-command the parity test (cli_dispatch_test.go) will not find.
//
// Every command follows the same shape: build a flag.FlagSet with a custom
// Usage, parse, open the database, do the one thing the command is for, and
// print `text` or `json` depending on --format where the command supports
// both. See CLI_USAGE.md for the reference and doc/source/cli.rst for the
// user-facing manual.
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
	AnkiForecastDay      = domain.AnkiForecastDay
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
	RawConn           = database.RawConn

	ComputeMatchHash                   = database.ComputeMatchHash
	ComputeGnuBGMatchHash              = database.ComputeGnuBGMatchHash
	ComputeBGFMatchHash                = database.ComputeBGFMatchHash
	ComputeCanonicalMatchHashFromXG    = database.ComputeCanonicalMatchHashFromXG
	ComputeCanonicalMatchHashFromBGF   = database.ComputeCanonicalMatchHashFromBGF
	ComputeCanonicalMatchHashFromGnuBG = database.ComputeCanonicalMatchHashFromGnuBG
	ConvertEMGLossToMWCLoss            = database.ConvertEMGLossToMWCLoss
)
