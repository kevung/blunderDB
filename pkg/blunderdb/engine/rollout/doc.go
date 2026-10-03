// Package rollout plays a position out to settle what a search cannot: two
// plays a few thousandths apart, or a cube decision the cube model is unsure
// of. It is written on top of gammonnet's Searcher and is meant to move
// upstream into gammonNet; ADR-0059 fixes its choices.
//
// A rollout is its own Configuration (EngineVersion): the gammonNet
// Configuration that plays the games, plus the procedure around it, which
// is the recipe of docs/recherche/P8-rollouts.md, always on:
//
//   - variance reduction: the luck of every roll, measured one ply deep, is
//     taken out of each game's result (worker.play);
//   - common dice: every candidate plays game n with the same rolls;
//   - quasi-random dice: the first two rolls are stratified over 36 and
//     36×36 games (gameDice);
//   - the game stops exactly where the two-sided bearoff table covers it;
//   - each game's dice are a function of (seed, game number), and results
//     are summed in game order: the numbers do not depend on the number of
//     workers or on their scheduling.
//
// Rollouts are cubeful: inside a game the cube is offered, taken or passed
// by gammonNet's cube decision at the rollout's ply, and a truncated game is
// valued by the cube model. Both lean on the model, which is why every
// Result carries CubefulBias: trust the ranking more than the absolute
// equity.
//
// Equities leave on the single output scale (ADR-0019): money points per
// unit of the position's cube, or normalised equity at a match score.
// Inside, a game is counted in money points or in the root's match winning
// chance; both are affine in the output, so the mean, the standard error and
// the JSD convert once, at the end.
package rollout
