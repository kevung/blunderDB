// Package document holds the Relevé of a document — the positions a book, an
// article or a web page shows, in the order it shows them — and its import
// into a Collection (ADR-0117).
//
// A Relevé is a JSON file a person can read and edit. Its header names the
// document (title, author, edition or URL, the SHA-256 of the source file, the
// profile its diagrams are drawn with, its default context); each Entry gives
// one diagram's rank, page, chapter, label, position, two certainties — its
// checkers (sure, to verify, uncertain; complete when fifteen a side were read
// with no doubt) and its context (read, default, to verify, uncertain) — the
// doubts the reading left, the diagram's rectangle on the page and whether a
// person has verified it. Its state follows from the two (Releve.StateOf): sure
// only when the checkers are and the context is certified, read or under the
// document's defaults once a person confirmed them (Defaults.Confirmed). The position is an XGID, the encoding the rest of
// the repository already reads and writes (domain.DecodeXGID): it carries the
// checkers, the cube, the player on roll, the dice, the score and the match
// length in one string, so none of them can disagree with another field.
//
// Reading a document produces a Relevé and writes nothing. AnalyzePDF reads
// the diagrams a PDF draws as text with eXtreme Gammon's board font (XG's PDF
// prints): each diagram is decoded exactly from its glyphs (xgboard.go holds
// the table) and is sure only when the print's own cross-checks hold —
// fifteen checkers a side, the printed pip counts of both players — and
// nothing in it lies beyond what the table was proven on. A PDF without a
// page drawn with that font is a scanned book: each page is composed from its
// images with its text layer (pdf.Scan) and read by package structure; a page
// that cannot be composed is passed over and named (AnalyzeOptions.Warn), and
// a PDF with no page made of images is refused (ErrNoGlyphDiagram). Package
// pdf is the reader of the file itself: pure Go, it distrusts the PDF and is
// bounded (see its doc). AnalyzeImages reads page images (a folder of them,
// or one: AnalyzeFile) with the same package structure, which finds diagrams
// by their structure and learns the document's style from the document. The
// printed pip counts that confirm its entries are read in the text layer
// when there is one and in the image itself; when the two disagree, the
// entry's checkers are not confirmed. The context is read field by field
// (structure.Context): the cube by its value in its square or in the text,
// coherent with its place; the dice counted in the image and confirmed by the
// text, the player on roll by their colour and their half of the board; the
// score by the panel of the text layer, each score assigned by the pips
// printed beside it. An entry whose context is read whole is "read"; what is
// not read takes StructureDefaults (cube centred, no dice, the bottom player
// on roll, money), the document's default context, unconfirmed, and carries
// ContextDoubt; a cube, dice or score printed and not certified — or a cube
// the top player owns with no dice drawn — leave its context to verify even
// once the defaults are confirmed (Releve.CheckersConfirmed,
// Summary.CheckersConfirmed). A person verifies such an entry's cube, dice,
// player on roll and score against the diagram and marks it verified.
//
// Only Import writes, and only what a Relevé allows: positions, each as a
// position brought in on its own, in one Collection named after the document,
// with the document's title, author and edition as the Collection's
// description and each position's reference ("p. 146, Problem 191") as a
// comment. No image, no rectangle and no text of the author enter the
// database. An entry neither sure nor verified refuses the whole import
// unless the caller forces it, or accepts complete entries whose context is
// certified and whose direction of play is not an unconfirmed supposition
// (Defaults.Direction, ImportOptions.AcceptComplete); an entry with no position is
// refused even then, and a refused import writes nothing (one transaction). The daemon
// imports a Relevé; it never reads a document.
//
// A reader of another kind of document plugs in by returning a Relevé that
// Validate accepts, and by what it puts in it: ranks in the document's
// order, a legal XGID or none (then the entry is uncertain), and a state
// that means what State says. Sure is claimed only on a cross-check the
// reader did not itself produce — the print's pip counts, never an engine's
// opinion of the position (ADR-0108 rule 4) — and every doubt is named.
package document
