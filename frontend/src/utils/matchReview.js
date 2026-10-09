// Readings of a match's study summary (storage.MatchReview, ADR-0078) shared by
// its at-a-glance summary and its folded details.

/** A signed MWC fraction as a percentage, e.g. "+52.0 %". @param {number} v */
export const signedPct = (v) => (v >= 0 ? '+' : '−') + Math.abs(v * 100).toFixed(1) + ' %';

/**
 * Which of dice and play decided the result, from the player's side.
 * @param {{ result: number, adjusted: number }} luck
 */
export function luckVerdict(luck) {
    const won = luck.result > 0;
    const skill = luck.adjusted > 0;
    if (won) return skill ? 'matchReview.wonByPlay' : 'matchReview.wonByDice';
    return skill ? 'matchReview.lostByDice' : 'matchReview.lostByPlay';
}

/**
 * What the split of the errors by time calls for.
 * @param {{ hasty: number, deliberate: number }} pace
 */
export function paceHint(pace) {
    if (pace.hasty > pace.deliberate) return 'matchReview.paceDiscipline';
    if (pace.deliberate > pace.hasty) return 'matchReview.paceKnowledge';
    return '';
}

/** Whether the review has anything to say. @param {any} review */
export const hasReview = (review) => !!review?.players?.some((/** @type {any} */ p) => p.decisions > 0);
