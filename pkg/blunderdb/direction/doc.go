// Package direction runs a Tournament from inside blunderDB: it is the only place in the tree
// that knows the Nicomaque engine.
//
// # What a Direction is
//
// A Tournament can be created BEFORE its Matches exist and run from here. Everything the
// director decides while running it is its Direction: format, entries, every match launched,
// result, correction, withdrawal, draw and phase change, in order (ADR-0047, CONTEXT.md
// § "Directing a tournament").
//
// # Two rules shape this package
//
//  1. The log is APPEND-ONLY. A wrong result is corrected by a later correction event, a match
//     launched by mistake by a later cancellation. Apply writes an event only once the engine has
//     accepted it, inside the caller's transaction; the state in memory is never persisted, so
//     a crash leaves a log that replays to exactly what the user last saw.
//
//  2. The derived state is NEVER stored. Standings, brackets, pairings and the next thing to do
//     are replayed from the events at every open, so a power cut costs nothing and a correction
//     leaves no stale intermediate state.
//
// # Replay cost
//
// A 64-player two-life tournament (318 events) replays in ~32 ms (BenchmarkOpen), under the
// 50 ms budget, so Open replays unconditionally: a cache would be a second source of truth.
// Past that budget, the answer is an incremental replay.
//
// # Codes, not sentences
//
// Nothing this package returns is displayed as it stands: the engine emits codes (labels,
// ranking notes, warnings, wait reasons) that live in the database, and translation belongs to
// the frontend (or labeler.go for the standalone page). A test enforces it.
//
// # What is NOT here
//
// No SQL: persistence goes through the Store interface, implemented by the desktop wrapper and
// by the storage backends. No display strings. No knowledge of Wails, of the board, or of the
// analysis engine.
package direction
