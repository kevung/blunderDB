// Package direction runs a Tournament from inside blunderDB: it is the only place in the tree
// that knows the Nicomaque engine.
//
// # What a Direction is
//
// A Tournament used to be an afterthought — a label put on Matches that came in from files. It
// can also be created BEFORE its Matches exist and run from here, and everything the director
// decides while running it is its Direction: the format, the entries, every match launched,
// every result, correction, withdrawal, draw and phase change, in the order they happened
// (ADR-0047, CONTEXT.md § "Directing a tournament").
//
// # Two rules shape this package
//
//  1. The log is APPEND-ONLY. A wrong result is corrected by a later correction event, a match
//     launched by mistake by a later cancellation. Apply writes the event BEFORE applying it,
//     inside the caller's transaction, so a crash between the two leaves a log that replays to
//     exactly what the user last saw.
//
//  2. The derived state is NEVER stored. Standings, brackets, pairings and the next thing to do
//     are replayed from the events at every open. That is what makes a power cut cost nothing,
//     and what keeps a correction from leaving a stale intermediate state behind.
//
// # Replaying costs what it costs, and it was measured
//
// A full 64-player two-life tournament — 125 matches, 318 events — replays in about 32 ms on a
// loaded laptop (BenchmarkOpen, 2026-09-07). That is under the 50 ms budget the design was
// given, and it is why Open replays unconditionally instead of caching a state: a cache would
// be a second source of truth for exactly the thing this package refuses to have two of. If a
// real tournament ever pushes past that budget, the answer is an incremental replay, and the
// new measure belongs right here.
//
// # Codes, not sentences
//
// Nothing this package returns is meant to be displayed as it stands. The engine emits codes
// (labels, ranking notes, warnings, wait reasons) precisely because blunderDB speaks nine
// languages and these values live in the database; the translation belongs to the frontend.
// A test enforces it.
//
// # What is NOT here
//
// No SQL: persistence goes through the Store interface, implemented by the desktop wrapper and
// by the storage backends. No display strings. No knowledge of Wails, of the board, or of the
// analysis engine.
package direction
