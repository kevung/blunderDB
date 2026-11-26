// Package arkqr is the paper archive of matches (ADR-0119): a compact codec
// of a whole match, the QR codes that carry it, the printable A4 page they
// are laid out on, and the reading back of that page or of its scan.
//
// # The codec
//
// A match is archived as its transcription document (package transcript):
// header, Actions, durations and timecodes, plus the match comment and the
// notes — comments and study mark — of the positions it was played from.
// Analyses are not archived: they are computed again after an import.
//
// An integer arithmetic coder drives three models. The dice have their exact
// law. The play is coded as its rank among the boards the legal plays leave,
// under a small softmax with integer weights over integer features (model.go),
// one regime for contact and one for the race. Which Action follows which,
// the side that acts and the resignation level go through static tables or
// adaptive bits. Whatever the models do not predict — an unknown play, a
// board entered by hand, impossible dice, a board the replay reaches another
// way — is spelled out, so every document a Replay reads round-trips.
//
// A legal play is written back with the steps this package's own generator
// finds for the board it leaves: two spellings of one play are one archive.
//
// # Determinism
//
// The format does not depend on the engine: the legal plays come from this
// package's own generator (board.go), ordered by the board they leave, and no
// floating point enters the coder. The first byte of a coded match is
// FormatVersion; the weights and tables are frozen for it.
//
// # Codes and pages
//
// A Frame is one QR code (level M): one match when it fits, else one part of
// it, or the parity of five data frames, which rebuilds any one of them that
// cannot be read. Every frame names its archive, its rank and the archive's
// size, and ends with a CRC-32; its text is Base45, the QR alphanumeric
// alphabet. Page lays the codes out on A4 sheets in four quadrants with a
// white gutter on both fold lines, and embeds the frames so the HTML file
// imports without reading a code; ScanImage reads them from a PNG or JPEG.
//
// # Position lists
//
// A printed position list carries its positions in frames of their own kind
// (KindPositions, a first byte of its own): EncodePositions codes a tranche
// of Tranche positions as their ArkIDs say them, in about twelve bytes each,
// and PositionFrames makes it an archive of its own, its parity included, so
// one printed page reads back without the others. Assemble returns matches
// and tranches apart; a match archive is read exactly as before.
package arkqr
