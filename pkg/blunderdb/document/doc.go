// Package document holds the Relevé of a document — the positions a book, an
// article or a web page shows, in the order it shows them — and its import
// into a Collection (ADR-0117).
//
// A Relevé is a JSON file a person can read and edit. Its header names the
// document (title, author, edition or URL, the SHA-256 of the source file, the
// profile its diagrams are drawn with); each Entry gives one diagram's rank,
// page, chapter, label, position, state (sure, to verify, uncertain), the
// doubts the reading left, the diagram's rectangle on the page and whether a
// person has verified it. The position is an XGID, the encoding the rest of
// the repository already reads and writes (domain.DecodeXGID): it carries the
// checkers, the cube, the player on roll, the dice, the score and the match
// length in one string, so none of them can disagree with another field.
//
// Reading a document produces a Relevé and writes nothing. Only Import writes,
// and only what a Relevé allows: positions, each as a position brought in on
// its own, in one Collection named after the document, with the document's
// title, author and edition as the Collection's description and each
// position's reference ("p. 146, Problem 191") as a comment. No image, no
// rectangle and no text of the author enter the database. An entry neither
// sure nor verified refuses the whole import unless the caller forces it, and
// a refused import writes nothing.
package document
