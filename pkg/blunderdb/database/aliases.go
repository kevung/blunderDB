// Package database is the legacy SQLite-only persistence wrapper the Wails
// GUI and the CLI run against — the *Database type, historically the sole
// backend before the storage contract (pkg/blunderdb/storage) was extracted
// for the serve daemon's PostgreSQL side. It delegates the actual reads and
// writes to storage/sqlite, and adds nothing on top except the RWMutex the
// GUI's concurrent goroutines share (mu) and the aliasing this file provides.
//
// Schema DDL lives in db_schema.go, migrations in db_migration.go, and
// everything else is split by domain into per-feature db_*.go files (one
// each for matches, positions, collections, anki decks, stats, and so on).
// A schema change needs a DatabaseVersion bump (pkg/blunderdb/domain) and a
// migration step registered in both this package and storage/sqlite — see
// the schema-change invariant in CLAUDE.md.
package database

// Domain types and constants are re-exported from
// github.com/kevung/blunderdb/pkg/blunderdb/domain so the persistence code
// (moved here from package main during the headless refactor) keeps
// compiling against the unqualified names it was written with.

import "github.com/kevung/blunderdb/pkg/blunderdb/domain"

const (
	NumPoints = domain.NumPoints
	BlackBar  = domain.BlackBar
	WhiteBar  = domain.WhiteBar
	None      = domain.None
	Black     = domain.Black
	White     = domain.White

	ExcludeEmpty = domain.ExcludeEmpty

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

	IssuanceInfo  = domain.IssuanceInfo
	WatermarkInfo = domain.WatermarkInfo
)

// InitializePosition returns the standard starting position.
func InitializePosition() Position {
	return domain.InitializePosition()
}
