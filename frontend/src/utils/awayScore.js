// awayScore.js — reading the stored away score, which carries the Crawford rule INSIDE the number
// (CONTEXT.md, « Away score »): -1 money, 0 one point to go post-Crawford, 1 one point to go in
// the Crawford game, n ≥ 2 n points. Turning it into a distance must read both sentinels: a raw 0
// subtracted from a match length says "already won" (Go twin: domain.PointsAway).

// pointsAway is the DISTANCE an away score means: one point for either
// sentinel, the number itself otherwise. Money play (-1) has no distance and
// comes back unchanged, for the caller to recognise as it already must.
export function pointsAway(away) {
    return away === 0 ? 1 : away;
}
