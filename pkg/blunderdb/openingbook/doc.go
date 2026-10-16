// Package openingbook is the book of replies to the opening move at a match
// score: for each of the 315 reply positions (15 opening plays × 21 reply
// rolls) and each score of a fixed grid, the candidate plays rolled out by
// pkg/blunderdb/engine/rollout, their equity and its standard error.
//
// The book is data, not schema: a gzipped JSON Lines file embedded in the
// binary (book.jsonl.gz), one line per reply position and score (Line). It
// is computed by blunderDB and reproducible from the seed and the signature
// each line carries; no external table is embedded (ADR-0094).
//
// Equities are on the single output scale (ADR-0019): normalised equity at
// the score, ±1 = the centred cube. Rollouts are cubeful and lean on the cube
// model, so the book shows gaps to the best play, never absolute equities:
// Line.Rank says how far each play sits behind the best, with the 95 %
// interval of that gap, and which plays cannot be told apart from it.
//
// Generate produces the lines job by job, appending each finished one to a
// file: a run stopped by Ctrl-C or a reboot resumes without redoing a
// finished job. cmd/openingbook-gen drives it and compiles the result into
// the embedded file.
package openingbook
