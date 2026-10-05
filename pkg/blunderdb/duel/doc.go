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
//     invents a result;
//   - time (cadence.go, ADR-0073): the Arbiter stamps when it hands a
//     Decision out and when it receives the Play, never the client. Every
//     Decision a Side takes is timed, with or without a Cadence, and the
//     duration goes on its Action, then on the Move; what the Arbiter plays
//     alone has none. A Cadence adds a reserve and a delay per Side; running
//     out is noted, and loses the match only if the Duel was set so. A Duel
//     in suspense stops the clocks.
//
// A Bot (a delegated Side) is not here yet: it is one more Side the
// SideResolver gives, and the Arbiter already asks every Side the same way.
package duel
