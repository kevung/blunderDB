// Package printsheet builds the printable A4 documents of the library: one
// self-contained HTML file (no style sheet, font, image or script outside it)
// that a browser prints, or saves as PDF. No board diagram is drawn: a
// position is written as its ArkID (ADR-0118), in a monospaced face where I
// and l differ.
//
// # The mechanism
//
// Document wraps documents (a section each, one or more A4 sheets) in the
// shared style sheet and the fitting script. The script sets the body of each
// sheet from 9 pt down to 5.5 pt by quarter points until its content fits;
// when it does not, the block marked data-verso moves to a second sheet that
// repeats the header, and a transcription marked data-flow switches from one
// move a line to running text. A verso still too long at 5.5 pt hands its last
// .pe entries to a further sheet, header repeated, until every one fits; an
// entry taller than a sheet on its own is cut at the last word of its comment
// that fits and goes on over the next sheet: the sheet's edge never hides a
// position or a word. A document marked data-fill (ListSheet) is laid out
// line by line instead: each page takes the .de lines that fit at 5.5 pt,
// header and legend repeated, its back (the data-verso block, same numbers)
// right after it, and every page is numbered. Without the script (a reader
// that runs none) the sheet prints at its default size.
//
// The classes of the style sheet are the vocabulary of every document: .mh
// the header, .lg the legend, .ph the head of a block, .pl a list of
// positions in columns, .pe one entry, .ref a number (black: a blunder;
// .o, framed: a remarkable position), .nt an ArkID, .qrb a reimport code;
// .dl a list of lines in columns, .de one line, .cm a comment.
//
// # The match sheet
//
// BuildMatches writes one sheet per match: header, transcription, and apart
// the positions to review, numbered in the transcription. Everything comes
// from the storage contract, so the desktop app, the CLI (`match sheet`) and
// the daemon (`matches.sheet`) print the same sheet. Errors are graded at the
// library's thresholds (storage.StatsStore.MatchMoveGrades), costs are on the
// one equity scale that leaves the engine (ADR-0019), and nothing is written:
// printing records nothing about who printed (ADR-0007). The reimport code is
// the match's paper archive (package arkqr), drawn and embedded, so the saved
// file imports as an archive page does.
//
// # The position list
//
// BuildPositions prints the whole list a PositionSource names — ids, a query
// or a collection, at most MaxPositions, a longer one refused with
// storage.ErrInvalid — one line a position: number, ArkID, pips of the player
// on roll and of the opponent, then the analysis (best choice, another with
// its loss in thousandths, win and gammon chances), on the line or on the
// back. The desktop app, the CLI (`sheet`) and the daemon (`positions.sheet`)
// print the same list, and nothing is written.
package printsheet
