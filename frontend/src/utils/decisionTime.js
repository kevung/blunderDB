// Decision times of a match (ADR-0073). A duration the database does not know
// is null or absent and reads as an empty cell, never as zero.

/**
 * @param {number | null | undefined} ms
 * @returns {string} "" when unknown, "5.2 s" below a minute, "1:05" above.
 */
export function fmtDuration(ms) {
    if (ms === null || ms === undefined || !Number.isFinite(ms)) return '';
    if (ms < 60000) return (ms / 1000).toFixed(1) + ' s';
    const total = Math.round(ms / 1000);
    return `${Math.floor(total / 60)}:${String(total % 60).padStart(2, '0')}`;
}

/**
 * A duration as a column of the Transcript shows it: whole seconds, "12 s" below a
 * minute, "1:05" above; "" when unknown.
 *
 * @param {number | null | undefined} ms
 */
export function fmtSeconds(ms) {
    if (ms === null || ms === undefined || !Number.isFinite(ms)) return '';
    const total = Math.round(ms / 1000);
    if (total < 60) return `${total} s`;
    return `${Math.floor(total / 60)}:${String(total % 60).padStart(2, '0')}`;
}

/**
 * A mean over `count` known durations; empty when there is none.
 *
 * @param {number} totalMS
 * @param {number} count
 */
export function fmtMean(totalMS, count) {
    return count > 0 ? fmtDuration(totalMS / count) : '';
}

/**
 * The time a Move took over its decisions, null when neither is known.
 *
 * @param {{ decision_ms?: number | null, cube_decision_ms?: number | null }} mp
 * @returns {number | null}
 */
export function moveTotalMS(mp) {
    const d = mp?.decision_ms ?? null;
    const c = mp?.cube_decision_ms ?? null;
    if (d === null && c === null) return null;
    return (d ?? 0) + (c ?? 0);
}

/** @param {readonly { mp: any }[]} moves */
export function hasAnyDuration(moves) {
    return moves.some(({ mp }) => moveTotalMS(mp) !== null);
}

/**
 * Rows ordered by duration, the unknown ones last whichever the direction.
 * Stable; `''` keeps the match order.
 *
 * @template {{ mp: any }} T
 * @param {readonly T[]} moves
 * @param {'' | 'asc' | 'desc'} direction
 * @returns {T[]}
 */
export function sortByDuration(moves, direction) {
    if (!direction) return [...moves];
    const sign = direction === 'asc' ? 1 : -1;
    return [...moves].sort((a, b) => {
        const x = moveTotalMS(a.mp);
        const y = moveTotalMS(b.mp);
        if (x === null && y === null) return 0;
        if (x === null) return 1;
        if (y === null) return -1;
        return sign * (x - y);
    });
}

/**
 * The bars of the time-over-the-match chart: one per Move with a known time,
 * at its place in the match.
 *
 * @param {readonly { player_on_roll: number, decision_ms?: number | null, cube_decision_ms?: number | null }[]} movePositions
 * @returns {{ index: number, player: 0 | 1, ms: number }[]}
 */
export function timeBars(movePositions) {
    /** @type {{ index: number, player: 0 | 1, ms: number }[]} */
    const bars = [];
    movePositions.forEach((mp, index) => {
        const ms = moveTotalMS(mp);
        if (ms !== null) bars.push({ index, player: mp.player_on_roll === 1 ? 1 : 0, ms });
    });
    return bars;
}

/**
 * The clock of every Move: the time its player has used since the start of the
 * match, that Move included. Unknown durations add nothing: a match that kept
 * durations kept them for every decision a player made, so a move without one
 * was played by the Arbiter and cost its player no time. Indexed like
 * `movePositions`, which must be in match order.
 *
 * @param {readonly { player_on_roll: number, decision_ms?: number | null, cube_decision_ms?: number | null }[]} movePositions
 * @returns {number[]}
 */
export function cumulativeClocks(movePositions) {
    const used = [0, 0];
    return movePositions.map((mp) => {
        const p = mp.player_on_roll === 1 ? 1 : 0;
        used[p] += moveTotalMS(mp) ?? 0;
        return used[p];
    });
}

/**
 * A clock reading in fixed m:ss, h:mm:ss past an hour; empty when unknown.
 *
 * @param {number | null | undefined} ms
 */
export function fmtClock(ms) {
    if (ms === null || ms === undefined || !Number.isFinite(ms)) return '';
    const total = Math.round(ms / 1000);
    const s = String(total % 60).padStart(2, '0');
    if (total < 3600) return `${Math.floor(total / 60)}:${s}`;
    return `${Math.floor(total / 3600)}:${String(Math.floor((total % 3600) / 60)).padStart(2, '0')}:${s}`;
}

/**
 * The band of a decision-time histogram a duration falls in: band i holds the
 * durations under `bounds[i]` and at or above `bounds[i - 1]`, the last band
 * everything longer — the banding of the time summary's histogram.
 *
 * @param {readonly number[]} bounds ascending, in ms
 * @param {number} ms
 */
export function timeBand(bounds, ms) {
    const i = bounds.findIndex((b) => ms < b);
    return i < 0 ? bounds.length : i;
}

/**
 * A band's label in whole seconds: "<2 s", "2–5 s", ">120 s".
 *
 * @param {readonly number[]} bounds
 * @param {number} i
 */
export function bandLabel(bounds, i) {
    const s = (/** @type {number} */ ms) => String(Math.round(ms / 1000));
    if (i === 0) return `<${s(bounds[0])} s`;
    if (i >= bounds.length) return `>${s(bounds[bounds.length - 1])} s`;
    return `${s(bounds[i - 1])}–${s(bounds[i])} s`;
}

/**
 * The decisions of each band, in match order: a Move enters the band of its
 * play and the band of its cube decision, once per band. Indexed like
 * `movePositions`.
 *
 * @param {readonly { player_on_roll: number, decision_ms?: number | null, cube_decision_ms?: number | null }[]} movePositions
 * @param {readonly number[]} bounds
 * @returns {{ index: number, player: 0 | 1 }[][]}
 */
export function bandMoves(movePositions, bounds) {
    /** @type {{ index: number, player: 0 | 1 }[][]} */
    const bands = Array.from({ length: bounds.length + 1 }, () => []);
    movePositions.forEach((mp, index) => {
        const player = mp.player_on_roll === 1 ? 1 : 0;
        const seen = new Set();
        for (const ms of [mp.decision_ms, mp.cube_decision_ms]) {
            if (ms === null || ms === undefined || !Number.isFinite(ms)) continue;
            const b = timeBand(bounds, ms);
            if (seen.has(b)) continue;
            seen.add(b);
            bands[b].push({ index, player });
        }
    });
    return bands;
}
