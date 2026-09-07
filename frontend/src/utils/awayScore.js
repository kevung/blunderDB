// awayScore.js — reading the away score blunderDB stores.
//
// The away score carries the Crawford rule INSIDE the number (CONTEXT.md,
// « Away score »): `-1` is money play, `0` is "one point to go, Crawford
// behind us", `1` is "one point to go, and this IS the Crawford game", `n ≥ 2`
// is n points. So 0 and 1 describe the same distance to victory and different
// rules — and any code turning an away score into a distance owes both
// sentinels a reading. Subtracting a stored 0 from a match length says "has
// already won", which is how a match-equity lookup ends up refusing a real
// position (the Go twin is domain.PointsAway).

// pointsAway is the DISTANCE an away score means: one point for either
// sentinel, the number itself otherwise. Money play (-1) has no distance and
// comes back unchanged, for the caller to recognise as it already must.
export function pointsAway(away) {
    return away === 0 ? 1 : away;
}
