// What a table shows of its rows: kept by the per-column filters, then ordered by the sort.
// Pure, so PanelTable and its tests share one definition (ADR-0090).

import { compareValues } from './matchTable.js';

/**
 * @typedef {object} ViewColumn
 * @property {string} key
 * @property {(row: any) => any} [value]  the sorted and filtered value; null = not measured
 * @property {'number' | 'text'} [filter] the kind of per-column filter, none when absent
 */

/**
 * @typedef {{ test: (value: number) => boolean } | { error: true } | null} NumericFilter
 */

const NUM = '(-?\\d+(?:[.,]\\d+)?)';
const COMPARISON = new RegExp(`^(>=|<=|>|<|≥|≤|=)?\\s*${NUM}$`);
const RANGE = new RegExp(`^${NUM}\\s*(?:-|–|\\.\\.)\\s*${NUM}$`);

/** @param {string} s */
const toNumber = (s) => Number(s.replace(',', '.'));

/**
 * Reads a numeric column filter: `>=500`, `≥500`, `>500`, `<2.5`, `<=2`, `3-7` (inclusive), `5`.
 * Blank text is no filter (`null`); text it cannot read is `{ error: true }`, which filters nothing.
 *
 * @param {string | null | undefined} text
 * @returns {NumericFilter}
 */
export function parseNumericFilter(text) {
    const s = String(text ?? '').trim();
    if (!s) return null;
    const range = RANGE.exec(s);
    if (range) {
        const a = toNumber(range[1]);
        const b = toNumber(range[2]);
        const lo = Math.min(a, b);
        const hi = Math.max(a, b);
        return { test: (v) => v >= lo && v <= hi };
    }
    const cmp = COMPARISON.exec(s);
    if (!cmp) return { error: true };
    const n = toNumber(cmp[2]);
    switch (cmp[1]) {
        case '>=':
        case '≥':
            return { test: (v) => v >= n };
        case '<=':
        case '≤':
            return { test: (v) => v <= n };
        case '>':
            return { test: (v) => v > n };
        case '<':
            return { test: (v) => v < n };
        default: {
            // A bare number matches at the precision typed: `4.2` keeps 4.15 < v < 4.25.
            const half = 0.5 * 10 ** -(cmp[2].split(/[.,]/)[1]?.length ?? 0);
            return { test: (v) => Math.abs(v - n) < half };
        }
    }
}

/**
 * @param {ViewColumn} col
 * @param {any} row
 */
function valueOf(col, row) {
    const v = col.value ? col.value(row) : row?.[col.key];
    return v === undefined || (typeof v === 'number' && Number.isNaN(v)) ? null : v;
}

/**
 * Whether a column's filter text cannot be read; the field shows it, the rows ignore it.
 *
 * @param {ViewColumn | undefined} col
 * @param {string | undefined} text
 */
export function filterIsInvalid(col, text) {
    const f = col?.filter === 'number' ? parseNumericFilter(text) : null;
    return !!f && 'error' in f;
}

/**
 * The rows a table shows: those every column filter keeps, ordered by `sort`. A row whose
 * value is null (not measured) fails a numeric filter and sorts last in both directions;
 * equal values fall back to the first text-filtered column, ascending.
 *
 * @template T
 * @param {T[]} rows
 * @param {ViewColumn[]} columns
 * @param {{ sort?: { column: string | null, direction: 'asc' | 'desc' } | null, filters?: Record<string, string> }} view
 * @returns {T[]}
 */
export function applyColumnView(rows, columns, { sort = null, filters = {} } = {}) {
    /** @type {Array<(row: T) => boolean>} */
    const keep = [];
    for (const col of columns) {
        const text = filters[col.key];
        if (!col.filter || !text || !text.trim()) continue;
        if (col.filter === 'number') {
            const f = parseNumericFilter(text);
            if (!f || 'error' in f) continue;
            keep.push((row) => {
                const v = valueOf(col, row);
                return v !== null && f.test(Number(v));
            });
        } else {
            const needle = text.trim().toLocaleLowerCase();
            keep.push((row) =>
                String(valueOf(col, row) ?? '')
                    .toLocaleLowerCase()
                    .includes(needle)
            );
        }
    }
    const kept = keep.length ? rows.filter((row) => keep.every((k) => k(row))) : [...rows];

    const sortCol = sort?.column ? columns.find((c) => c.key === sort.column) : undefined;
    if (!sortCol) return kept;
    const sign = sort?.direction === 'desc' ? -1 : 1;
    const tie = columns.find((c) => c.filter === 'text' && c !== sortCol);
    return kept
        .map((row, i) => ({ row, i, v: valueOf(sortCol, row) }))
        .sort((a, b) => {
            if (a.v === null || b.v === null) {
                if (a.v === null && b.v === null) return a.i - b.i;
                return a.v === null ? 1 : -1;
            }
            const cmp = compareValues(a.v, b.v);
            if (cmp !== 0) return sign * cmp;
            if (tie) {
                const t = compareValues(valueOf(tie, a.row), valueOf(tie, b.row));
                if (t !== 0) return t;
            }
            return a.i - b.i;
        })
        .map((e) => e.row);
}
