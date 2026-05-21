package main

// The persistence layer now lives in
// github.com/kevung/blunderdb/pkg/blunderdb/database. The names the GUI/CLI
// code (and the Wails bindings) depend on are re-exported here so package
// main keeps compiling unchanged while the headless refactor is in progress.

import "github.com/kevung/blunderdb/pkg/blunderdb/database"

type (
	Database    = database.Database
	StatsFilter = database.StatsFilter
	StatsResult = database.StatsResult
)

var (
	NewDatabase       = database.NewDatabase
	DeleteFile        = database.DeleteFile
	ErrDuplicateMatch = database.ErrDuplicateMatch
)
