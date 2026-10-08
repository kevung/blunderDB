// Display of the 7-point MWC loss (domain.MWC7): the share of match-winning chances lost against
// a perfect player, rescaled to a 7-point match. One formatter for every panel, so a figure never
// reads differently from one place to the next. `available: false` (money play) never shows a number.

import { formatNumber } from './format.js';
import { translate } from '../i18n/index.js';

/** @typedef {import('../../wailsjs/go/models').domain.MWC7} MWC7 */

const DASH = '—';
const ONE_DECIMAL = { minimumFractionDigits: 1, maximumFractionDigits: 1 };

/**
 * Whether there is a figure to show.
 * @param {Partial<MWC7> | null | undefined} m
 * @returns {boolean}
 */
export function hasMwc7(m) {
    return !!m && m.available === true && Number.isFinite(m.loss);
}

/**
 * The loss as a percentage with one decimal, e.g. "12.3 %"; "—" when unavailable.
 * @param {Partial<MWC7> | null | undefined} m
 * @returns {string}
 */
export function fmtMwc7(m) {
    if (!hasMwc7(m)) return DASH;
    return `${formatNumber(/** @type {number} */ (m?.loss) * 100, ONE_DECIMAL)} %`;
}

/**
 * The 95 % interval of the loss, e.g. "[8.1–16.5]"; '' without one.
 * @param {Partial<MWC7> | null | undefined} m
 * @returns {string}
 */
export function fmtMwc7Interval(m) {
    if (!hasMwc7(m) || !m?.has_interval) return '';
    return `[${formatNumber(/** @type {number} */ (m.low) * 100, ONE_DECIMAL)}–${formatNumber(/** @type {number} */ (m.high) * 100, ONE_DECIMAL)}]`;
}

/**
 * Headline and interval on one line, e.g. "12.3 % [8.1–16.5]".
 * @param {Partial<MWC7> | null | undefined} m
 * @returns {string}
 */
export function fmtMwc7Full(m) {
    const interval = fmtMwc7Interval(m);
    return interval ? `${fmtMwc7(m)} ${interval}` : fmtMwc7(m);
}

/**
 * The explanatory tooltip; for an unavailable figure, why there is none.
 * @param {Partial<MWC7> | null | undefined} m
 * @param {(key: string, params?: Record<string, unknown> | null) => string} [tfn] `$t` in a component, so it follows the language
 * @returns {string}
 */
export function mwc7Tooltip(m, tfn = translate) {
    if (!hasMwc7(m)) return tfn('mwc7.unavailable');
    const prefix = m?.elo_floored ? '≤' : '≈';
    const elo = `${prefix} ${formatNumber(Math.round(/** @type {number} */ (m?.elo)))}`;
    return tfn('mwc7.tooltip', { elo });
}

/**
 * The loss as a chart value in percent, or null when there is no figure (the point is skipped).
 * @param {Partial<MWC7> | null | undefined} m
 * @returns {number | null}
 */
export function mwc7Percent(m) {
    return hasMwc7(m) ? /** @type {number} */ (m?.loss) * 100 : null;
}
