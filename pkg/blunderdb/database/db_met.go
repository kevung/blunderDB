package database

import (
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// The GNUbg Match Equity Table machinery (Kazaross-XG2 + Zadeh fallback) lives
// in package engine (engine/met.go) so both the SQLite Storage backend and this
// wrapper can convert equities to MWC. This file only re-exports the helper the
// database package's callers still reference by its unqualified name.

// ConvertEMGLossToMWCLoss is re-exported from package engine so the database
// package (and its callers, e.g. the CLI) keep referencing the unqualified
// name. New code should call engine.ConvertEMGLossToMWCLoss directly.
var ConvertEMGLossToMWCLoss = engine.ConvertEMGLossToMWCLoss
