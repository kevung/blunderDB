// Package bearoffgen generates gnubg bearoff databases, byte for byte.
//
// ADR-0027: the tables are generated on the machine that needs them, not
// shipped inside the binary nor downloaded. The two embedded files were a
// third of the compressed delivery (gnubg_ts0.bd 6.8 MB, gnubg_os6.bd 1.4 MB),
// and the measurement that opened the question — could gammonNet's evaluated
// regime replace the exact table? — came back no: the win probability
// converges with the ply, the cube verdict does not (worst decision 0.45, and
// double/take turning into no-double on last-roll positions).
//
// Exactness here means identity with gnubg, not "close enough". The generator
// is a port of makebearoff.c, and its test is a byte-for-byte comparison with
// the file gnubg itself produced. Two consequences run through the code:
//
//   - no floating point, anywhere. Equities are int32 arithmetic over int16
//     storage, exactly as makebearoff.c does with `short int`, down to the
//     one's-complement negation (^x, not -x: they differ by one) and the
//     truncating /36.
//   - the sweep order is part of the contract. A position at (us, them) reads
//     row `them` and columns j < us, so the file is filled along diagonals of
//     constant us+them — ordering by `us` alone gives a wrong answer, first
//     seen at the pair (4,5).
package bearoffgen
