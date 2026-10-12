/**
 * Narrowing a Stats filter to a span of days, for the counts of a window or a period: the span
 * is intersected with the filter's own dates, never widened beyond them.
 */

/**
 * @template {{ dateFrom?: string, dateTo?: string }} F
 * @param {F} filter
 * @param {string} from - first day, YYYY-MM-DD
 * @param {string} to   - last day, YYYY-MM-DD (inclusive)
 * @returns {F}
 */
export function narrowToDays(filter, from, to) {
    return {
        ...filter,
        dateFrom: filter?.dateFrom && filter.dateFrom > from ? filter.dateFrom : from,
        dateTo: filter?.dateTo && filter.dateTo < to ? filter.dateTo : to
    };
}

/** Last day of a month, YYYY-MM-DD. */
function lastDayOfMonth(/** @type {number} */ y, /** @type {number} */ m) {
    const day = new Date(Date.UTC(y, m, 0)).getUTCDate();
    return `${y}-${String(m).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
}

/**
 * The filter narrowed to a corpus window, `from` and `to` given as YYYY-MM (both months
 * inclusive).
 *
 * @template {{ dateFrom?: string, dateTo?: string }} F
 * @param {F} filter
 * @param {{ from: string, to: string }} w
 */
export function narrowToMonths(filter, w) {
    const [y, m] = String(w.to).split('-').map(Number);
    return narrowToDays(filter, `${w.from}-01`, lastDayOfMonth(y, m));
}

/**
 * The filter narrowed to a training period: a week from its Monday, or a calendar month from
 * its first day.
 *
 * @template {{ dateFrom?: string, dateTo?: string }} F
 * @param {F} filter
 * @param {string} start - the period's first day, YYYY-MM-DD
 * @param {'week' | 'month' | string} kind
 */
export function narrowToPeriod(filter, start, kind) {
    const [y, m, d] = start.split('-').map(Number);
    if (kind === 'month') return narrowToDays(filter, start, lastDayOfMonth(y, m));
    const end = new Date(Date.UTC(y, m - 1, d + 6)).toISOString().slice(0, 10);
    return narrowToDays(filter, start, end);
}
