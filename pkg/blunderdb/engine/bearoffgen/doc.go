// Package bearoffgen generates gnubg bearoff databases, byte for byte.
//
// ADR-0027: the tables are generated on the machine that needs them, neither
// shipped inside the binary nor downloaded (gammonNet cannot replace the exact
// table: its cube verdict does not converge with the ply).
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
//     constant us+them — ordering by `us` alone is wrong from the pair (4,5).

package bearoffgen
