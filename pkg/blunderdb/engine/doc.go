// Package engine holds what blunderDB computes ABOUT a position, as opposed
// to what it stores about one (pkg/blunderdb/storage) or what a position IS
// (pkg/blunderdb/domain, which this package imports and which never imports
// back).
//
// It is deliberately a flat package of independent files rather than a
// layered one: the codecs read the derived quantities they store, quiz.go and
// explain.go compare moves through the codec's NormalizeMove, met.go reads
// met_table.go, and nothing else here calls anything else here.
//
//	zobrist.go       the position identity — one 64-bit key per position,
//	                 per tenant, the key SavePosition dedups on (ADR-0001).
//	                 Its random stream is FROZEN: a change to the order in
//	                 which keys are drawn rehashes every database ever
//	                 written, which is why retired keys are drawn and
//	                 discarded rather than removed (ADR-0028).
//	bitboards.go     the four 26-bit occupancy and point masks the search
//	                 filters on, computed once and stored as columns.
//	epc.go           effective pip count, from a generated one-sided
//	                 gnubg table (ADR-0027): the roll
//	                 distribution of a bearoff position, its mean, and the
//	                 wastage the two imply.
//	bearoff_export.go the combinatorial indexing every gnubg bearoff
//	                 database uses, exported for engine/race, which
//	                 addresses TWO-sided tables with the same scheme.
//	met.go           the match equity table: Kazaross-XG2 (XG's own default,
//	                 met_kazaross_xg2.json, checksum-pinned) extended by a
//	                 Zadeh fallback beyond it, up to MaxScore (64) away.
//	                 GnuBGGetME is the one entry point, and the cube model
//	                 of engine/gammonnet branches onto it rather than
//	                 re-porting gammonNet's gn_met.c.
//	met_table.go     MET, a table imported from a gnubg .xml file
//	                 (ADR-0068); a nil *MET reads the built-in one.
//	positioncodec.go the v2 storage shape of a position: the compact
//	                 28-integer board plus every derived scalar column, and
//	                 the reconstruction that reads them back.
//	analysiscodec.go the same for an analysis: the zstd blob and its shared
//	                 dictionary (analysis_dict.bin, ADR-0030), plus the
//	                 scalar columns the statistics and the SQL filters read
//	                 — rates ×100, equities ×1000, and the one canonical
//	                 reading of a cube label (CanonicalCubeAction).
//	analysisbin.go   the binary payload inside that blob (ADR-0070): every
//	                 field of a PositionAnalysis, floats bit for bit.
//	gamephase.go     ClassifyGamePhase and ClassifyGameType: labels derived
//	gametype.go      from the board alone, stored in indexed columns.
//	movenotation.go  comparing one move written by two engines.
//	similarity.go    the distance behind "positions like this one".
//	quiz.go          grading an answer against the stored analysis.
//	explain.go       the error theme of a blunder, as a code, never a
//	                 sentence.
//
// # The two subpackages are the two evaluators
//
// (Three more stand beside them and evaluate nothing of their own:
// engine/bearoffgen generates the tables race reads, ADR-0027; engine/training
// makes the Training questions that need gammonNet — a seed played out by the
// engine, and its truth — and lives above both evaluators because race may not
// import gammonnet, ADR-0041; engine/rollout plays positions out on top of
// gammonnet's Searcher, ADR-0060.)
//
//	engine/race/      exact and estimated race analysis: the two-sided
//	                  bearoff reader, the calibrated win-probability
//	                  correction outside the table, and money cube verdicts
//	                  that are READ or convolved, never estimated
//	                  (ADR-0009, ADR-0012).
//	engine/gammonnet/ the neural evaluator — a Go port of gammonNet's
//	                  encoding, network, search and cube model, about 5 500
//	                  lines, the largest thing in this tree (ADR-0011).
//	                  It has its own package doc; read it, and cube.go's
//	                  header, before changing anything there. Its arithmetic
//	                  is a contract, not an implementation detail
//	                  (ADR-0024).
//
// # The rule these files share
//
// A derived value is computed HERE and stored, never recomputed at read time
// by a caller that happens to need it — that is what makes a scalar column
// trustworthy, and it is why a function that stops writing one breaks
// nothing visible until a search silently returns fewer rows.
package engine
