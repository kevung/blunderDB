// Display of a 95 % interval (domain.Interval, ADR-0078): the bootstrap band of a PR over the
// games of a match or the matches of a selection. Without one (a single unit) nothing is shown,
// and the caller greys the figure.

import { formatNumber } from './format.js';

const TWO_DECIMALS = { minimumFractionDigits: 2, maximumFractionDigits: 2 };

/**
 * Whether the interval exists.
 * @param {{ available?: boolean } | null | undefined} iv
 * @returns {boolean}
 */
export function hasInterval(iv) {
    return !!iv && iv.available === true;
}

/**
 * A PR interval, e.g. "[3.10–6.42]"; '' without one.
 * @param {{ available?: boolean, low?: number, high?: number } | null | undefined} iv
 * @returns {string}
 */
export function fmtPRInterval(iv) {
    if (!hasInterval(iv)) return '';
    return `[${formatNumber(/** @type {number} */ (iv?.low), TWO_DECIMALS)}–${formatNumber(/** @type {number} */ (iv?.high), TWO_DECIMALS)}]`;
}
