// Package engine holds what blunderDB computes ABOUT a position, as opposed
// to what it stores about one (pkg/blunderdb/storage) or what a position IS
// (pkg/blunderdb/domain, which this package imports and which never imports
// back).
//
// It is deliberately a flat package of independent files rather than a
// layered one: nothing here calls anything else here except the two codecs,
// which read the derived quantities they store.
//
//	zobrist.go       the position identity — one 64-bit key per position,
//	                 per tenant, the key SavePosition dedups on (ADR-0001).
//	                 Its random stream is FROZEN: a change to the order in
//	                 which keys are drawn rehashes every database ever
//	                 written, which is why retired keys are drawn and
//	                 discarded rather than removed (ADR-0028).
//	bitboards.go     the four 26-bit occupancy and point masks the search
//	                 filters on, computed once and stored as columns.
//	epc.go           effective pip count, from the embedded one-sided
//	                 gnubg OS-06-15 database (gnubg_os6.bd): the roll
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
//	positioncodec.go the v2 storage shape of a position: the compact
//	                 28-integer board plus every derived scalar column, and
//	                 the reconstruction that reads them back.
//	analysiscodec.go the same for an analysis: the zstd blob and its shared
//	                 dictionary (analysis_dict.bin, ADR-0030), plus the
//	                 scalar columns the statistics and the SQL filters read
//	                 — rates ×100, equities ×1000, and the one canonical
//	                 reading of a cube label (CanonicalCubeAction).
//
// # The two subpackages are the two evaluators
//
//	engine/race/      exact and estimated race analysis: the two-sided
//	                  bearoff reader, the calibrated win-probability
//	                  correction outside the table, and money cube verdicts
//	                  that are READ or convolved, never estimated
//	                  (ADR-0009, ADR-0012).
//	engine/gammonnet/ the neural evaluator — a Go port of gammonNet's
//	                  encoding, network, search and cube model, about 5 000
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
