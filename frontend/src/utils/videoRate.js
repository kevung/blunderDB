/**
 * The playback speeds the video keys step through. One list for every source: a file plays
 * them all, YouTube only those its player reports (getAvailablePlaybackRates), so the list is
 * filtered by what the source accepts rather than kept per player.
 */
// Quarter steps from 0.25× to 4×: fine enough near normal speed, fast enough to run through the moves.
export const VIDEO_RATES = Object.freeze(Array.from({ length: 16 }, (_, i) => (i + 1) * 0.25));

/**
 * The next speed one step slower (direction -1) or faster (+1), among the speeds `accepted`
 * also offers. The ends hold: past the slowest or the fastest the speed stays where it is.
 *
 * @param {number} current the speed now applied
 * @param {-1 | 1} direction
 * @param {readonly number[]} [accepted] the speeds the source accepts; all of them by default
 * @returns {number}
 */
export function stepRate(current, direction, accepted = VIDEO_RATES) {
    const usable = VIDEO_RATES.filter((rate) => accepted.some((a) => Math.abs(a - rate) < 1e-6));
    if (direction > 0) return usable.find((rate) => rate > current + 1e-6) ?? current;
    return [...usable].reverse().find((rate) => rate < current - 1e-6) ?? current;
}

/**
 * @param {KeyboardEvent} event
 * @returns {-1 | 0 | 1} the step `[` (slower) or `]` (faster) asks for, read by the character typed:
 *   AltGr, which types them on AZERTY, arrives as Alt (and Ctrl+Alt on Windows) and is no reason to refuse.
 */
export function rateKeyDirection(event) {
    if (event.metaKey) return 0;
    if (event.key === '[') return -1;
    if (event.key === ']') return 1;
    return 0;
}
