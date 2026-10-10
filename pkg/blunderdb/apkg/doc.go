// Package apkg writes a study deck or a collection as an Anki package (.apkg),
// to review it in Anki on a phone without the serve daemon.
//
// The package is the legacy layout every Anki client imports — desktop since
// 2.0, AnkiDroid, AnkiMobile: a zip holding collection.anki2 (collection
// schema 11), a "media" JSON index and the media files named by their index
// (ADR-0087). One position is one note with one card: the front shows the
// board (report.Diagram, the SVG the CLI and the daemon already draw), the
// score, the cube and the dice; the back the right move or cube decision, its
// equity and the error of what was played.
//
// A note's id and guid derive from the position's Zobrist hash, the note
// type's id is a constant and the deck's id derives from its name: a
// re-export imported over the first one updates its notes instead of
// duplicating them.
//
// Export reads and never writes (ADR-0007). The deck's description carries
// the producer's metadata through issuance.CarriedMetadataKeys, nothing else.
package apkg
