// Package duel is the Arbiter of a Duel: a match played inside blunderDB, the
// dice rolled here, the rules enforced, each Side deciding for itself
// (ADR-0072). Finished, it is an ordinary Match.
//
// # What lives where
//
// The rules are pkg/blunderdb/transcript's rule machine: an Action gives the
// next state or a reasoned refusal. This package adds what a match played here
// has and a record does not:
//
//   - the dice, drawn from a seed sealed at creation (dice.go): every roll is
//     [Roll] of the seed and its rank, the seed's [Fingerprint] is published at
//     creation and the seed itself revealed with the Match (rule 8);
//   - the Sides (side.go), asked for each Decision through one interface,
//     whatever stands behind them; this package holds the external one;
//   - what is not a decision, which the Arbiter plays alone: the opening roll,
//     the roll when the cube is not available, the dance, the only play
//     (rule 9);
//   - the Start's decision, which the machine does not read: a Start on a
//     double waiting for its answer, or on a cube decision the rules do not
//     offer, is refused by name (rule 12);
//   - the draft (service.go), written after every Play over
//     storage.DuelStore, resumed at the same point with the same dice to come;
//     several in suspense, one open (rule 10);
//   - the end: the Match written through ingest.WriteMatch, as a Transcription
//     writes it, with its origin (storage.MatchOrigin) — or the draft thrown
//     away. Stopping and keeping writes the games as they stand; nothing
//     invents a result.
//
// A Bot (bot.go) is a delegated Side: gammonNet's stateless playing policy
// (engine/gammonnet, policy.go) at a named level, the Configuration that will
// analyse the match, whose name its player carries in the Match (rules 6, 7).
// Resolve gives it like any Side; it answers at once, so two Bots play a
// whole match in the call that creates the Duel. The policy resigns only
// before a roll, and the Arbiter asks for no decision there when the cube is
// not available: a Bot then plays its certain loss out. A Cadence is not here
// yet: it will time the Decisions settle asks for.
package duel
