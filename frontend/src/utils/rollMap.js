// The roll map's arithmetic: the 21 rows of a position, their spread around the mean, and the
// grid they fill. Pure, so the view only draws.

export const ROLL_COUNT = 21;

/** The map's rows in the engine's order (1-1, 1-2, … 6-6), absent until their event lands.
 * @param {Array<any> | null | undefined} rows */
export function rollRows(rows) {
    return (rows ?? []).filter(Boolean);
}

/** Roll-weighted mean of the equities present.
 * @param {Array<any>} rows */
export function weightedMean(rows) {
    let w = 0;
    let sum = 0;
    for (const r of rows) {
        w += r.weight;
        sum += r.weight * r.equity;
    }
    return w ? sum / w : 0;
}

/**
 * Mean, weighted standard deviation, best and worst of the rows.
 * @param {Array<any>} rows
 * @param {number} [mean] the map's own average, when it has one
 */
export function rollStats(rows, mean) {
    const average = mean ?? weightedMean(rows);
    let w = 0;
    let variance = 0;
    let best = null;
    let worst = null;
    for (const r of rows) {
        w += r.weight;
        variance += r.weight * (r.equity - average) ** 2;
        if (!best || r.equity > best.equity) best = r;
        if (!worst || r.equity < worst.equity) worst = r;
    }
    return { average, stdev: w ? Math.sqrt(variance / w) : 0, best, worst, marketLost: rows.filter((r) => r.marketLost).reduce((n, r) => n + r.weight, 0) };
}

/** @param {any} row */
export const rollLabel = (row) => `${row.die1}-${row.die2}`;

/** @type {Record<string, (a: any, b: any, mean: number) => number>} */
const COMPARE = {
    roll: (a, b) => a.index - b.index,
    move: (a, b) => (a.move ?? '').localeCompare(b.move ?? ''),
    equity: (a, b) => a.equity - b.equity,
    gap: (a, b, mean) => Math.abs(a.equity - mean) - Math.abs(b.equity - mean)
};

/**
 * The rows, each carrying its index, sorted by key; ties fall back on the engine's order.
 * @param {Array<any>} rows
 * @param {'roll' | 'move' | 'equity' | 'gap'} key
 * @param {boolean} descending
 * @param {number} mean
 */
export function sortRows(rows, key, descending, mean) {
    const compare = COMPARE[key] ?? COMPARE.roll;
    const sign = descending ? -1 : 1;
    return [...rows].sort((a, b) => sign * compare(a, b, mean) || a.index - b.index);
}

/**
 * Where an equity sits between the worst (-1) and the best (+1) of the rows, the mean at 0.
 * @param {number} equity
 * @param {number} mean
 * @param {number} spread the largest distance to the mean among the rows
 */
export function deviationScale(equity, mean, spread) {
    return spread > 0 ? Math.max(-1, Math.min(1, (equity - mean) / spread)) : 0;
}

/** @param {number} value @param {number} digits */
export function signed(value, digits = 3) {
    const text = value.toFixed(digits);
    return value > 0 ? `+${text}` : text;
}

/** The roll's cell in the 6×6 grid: both orders of a non-double share a row.
 * @param {number} a @param {number} b */
export const rollIndex = (a, b) => {
    const lo = Math.min(a, b);
    const hi = Math.max(a, b);
    // rows run 1-1, 1-2 … 1-6, 2-2 … 6-6
    let index = 0;
    for (let i = 1; i < lo; i++) index += 7 - i;
    return index + (hi - lo);
};
