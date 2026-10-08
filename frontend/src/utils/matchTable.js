// Pure helpers for the Matches panel table: sorting comparators and the date /
// dice formatters. Extracted from MatchPanel.svelte so they can be unit-tested
// without mounting the component (they close over no component state).

import { fmtMwc7Full, mwc7Percent, mwc7Tooltip } from './mwc7.js';

/** @typedef {Partial<import('../../wailsjs/go/models').domain.Match>} Match */
/** @typedef {import('../../wailsjs/go/models').database.MatchPlayerDetailStats} MatchPlayerStats */
/** @typedef {import('../../wailsjs/go/models').storage.MoveGrade} MoveGrade */
/** @typedef {'player1' | 'player2' | 'date' | 'length' | 'tournament' | 'pr' | 'mwc' | 'mwc7'} MatchColumn */
/** @typedef {'asc' | 'desc'} SortDirection */

/**
 * Comparator with null-handling: nulls sort last, strings compare
 * case-insensitively, numbers numerically.
 * @param {string | number | null | undefined} a
 * @param {string | number | null | undefined} b
 * @returns {number}
 */
export function compareValues(a, b) {
    if (a == null && b == null) return 0;
    if (a == null) return 1;
    if (b == null) return -1;
    if (typeof a === 'string') return a.localeCompare(String(b), undefined, { sensitivity: 'base' });
    return Number(a) - Number(b);
}

/**
 * Extract the sortable value for a Matches panel column key.
 * @param {Match} match
 * @param {string} column
 * @returns {string | number}
 */
export function getSortValue(match, column) {
    switch (column) {
        case 'player1':
            return match.player1_name || '';
        case 'player2':
            return match.player2_name || '';
        case 'date':
            return match.match_date || '';
        case 'length':
            return match.match_length || 0;
        case 'tournament':
            return match.tournament_name || match.event || '';
        case 'pr':
            return match.pr || 0;
        case 'mwc':
            return match.mwc_loss || 0;
        case 'mwc7':
            return mwc7Percent(match.mwc7) ?? /** @type {any} */ (null);
        default:
            return '';
    }
}

/**
 * Sort a copy of `matches` by `column` in `direction` ('asc' | 'desc'); unchanged without a column.
 * @param {Match[]} matches
 * @param {string | null | undefined} column
 * @param {string} direction
 * @returns {Match[]}
 */
export function sortMatches(matches, column, direction) {
    if (!column) return matches;
    const sorted = [...matches].sort((a, b) => {
        const cmp = compareValues(getSortValue(a, column), getSortValue(b, column));
        return direction === 'asc' ? cmp : -cmp;
    });
    return sorted;
}

/**
 * Convert a date string to a `yyyy-mm-dd` value for a date <input>; '' if invalid.
 * @param {string | number | Date | null | undefined} dateStr
 * @returns {string}
 */
export function toDateInputValue(dateStr) {
    if (!dateStr) return '';
    try {
        const date = new Date(dateStr);
        if (isNaN(date.getTime())) return '';
        return date.toISOString().split('T')[0];
    } catch {
        return '';
    }
}

/**
 * Format a date string as `yyyy/mm/dd` for display; '-' if empty or invalid.
 * @param {string | number | Date | null | undefined} dateStr
 * @returns {string}
 */
export function formatDate(dateStr) {
    if (!dateStr) return '-';
    const date = new Date(dateStr);
    if (isNaN(date.getTime())) return '-';
    const y = date.getFullYear();
    const m = String(date.getMonth() + 1).padStart(2, '0');
    const d = String(date.getDate()).padStart(2, '0');
    return `${y}/${m}/${d}`;
}

/**
 * Compact dice rendering, e.g. [3,1] → "31"; '' when there are no dice.
 * @param {number[] | null | undefined} dice
 * @returns {string}
 */
export function formatDiceShort(dice) {
    if (!dice || (!dice[0] && !dice[1])) return '';
    return `${dice[0]}${dice[1]}`;
}

// --- Match-detail per-player stats table -------------------------------------

// Cell formatters for the MatchPanel "stats" tab. A player's MatchPlayerDetailStats
// maps to a displayed string; em-dash ("—") marks "not applicable / no data".
/** @type {(val: number, decisions: number) => string} */
export const fmtPR = (val, decisions) => (decisions > 0 ? val.toFixed(2) : '—');
/** @type {(v: number) => string} */
export const fmtEquityError = (v) => (v > 0 ? '-' + v.toFixed(3) : '—');
/** @type {(v: number) => string} */
export const fmtMwcLoss = (v) => (v > 0 ? '-' + (v * 100).toFixed(2) + '%' : '—');
/** @type {(errors: number, blunders: number) => string} */
export const fmtErrorsBlunders = (errors, blunders) => `${errors} (${blunders})`;

// The per-player match stats table, row by row: a section header ({ section }) or a metric
// ({ label, fmt, … }; fmt maps one player's stats to its cell). Labels are i18n keys.
// bullet = leading "•"; sub = indented sub-metric; valClass = extra value-cell class.
/**
 * @typedef {{ section: string, title?: undefined } | { label: string, bullet?: boolean, sub?: boolean, valClass?: string, fmt: (p: MatchPlayerStats) => string, title?: (p: MatchPlayerStats) => string }} MatchStatRow
 */

/** @type {MatchStatRow[]} */
export const MATCH_STAT_ROWS = [
    { section: 'match.performanceRating' },
    { label: 'match.overallPr', bullet: true, valClass: 'pr-val', fmt: (p) => fmtPR(p.pr, p.total_decisions) },
    { label: 'mwc7.name', bullet: true, fmt: (p) => fmtMwc7Full(p.mwc7), title: (p) => mwc7Tooltip(p.mwc7) },
    { label: 'match.checkerPlayPr', bullet: true, fmt: (p) => fmtPR(p.pr_checker, p.checker_decisions) },
    { label: 'match.cubePlayPr', bullet: true, fmt: (p) => fmtPR(p.pr_cube, p.double_decisions + p.take_decisions) },

    { section: 'match.totalErrors' },
    { label: 'match.errorsBlunders', bullet: true, fmt: (p) => fmtErrorsBlunders(p.total_errors, p.total_blunders) },
    { label: 'match.equityErrorEmg', sub: true, fmt: (p) => fmtEquityError(p.total_equity_error) },
    { label: 'match.mwcLoss', sub: true, fmt: (p) => fmtMwcLoss(p.mwc_loss) },
    { label: 'match.decisions', sub: true, fmt: (p) => String(p.total_decisions) },

    { section: 'match.checkerPlay' },
    { label: 'match.checkerErrorsBlunders', bullet: true, fmt: (p) => fmtErrorsBlunders(p.checker_errors, p.checker_blunders) },
    { label: 'match.equityErrorEmg', sub: true, fmt: (p) => fmtEquityError(p.checker_equity_error) },
    { label: 'match.mwcLoss', sub: true, fmt: (p) => fmtMwcLoss(p.checker_mwc_loss) },
    { label: 'match.unforcedMoves', sub: true, fmt: (p) => String(p.checker_decisions) },

    { section: 'match.cubePlay' },
    { label: 'match.doublesBlunders', bullet: true, fmt: (p) => fmtErrorsBlunders(p.double_errors, p.double_blunders) },
    { label: 'match.equityErrorEmg', sub: true, fmt: (p) => fmtEquityError(p.double_equity_error) },
    { label: 'match.mwcLoss', sub: true, fmt: (p) => fmtMwcLoss(p.double_mwc_loss) },
    { label: 'match.cubeDecisions', sub: true, fmt: (p) => String(p.double_decisions) },
    { label: 'match.takesBlunders', bullet: true, fmt: (p) => fmtErrorsBlunders(p.take_errors, p.take_blunders) },
    { label: 'match.equityErrorEmg', sub: true, fmt: (p) => fmtEquityError(p.take_equity_error) },
    { label: 'match.mwcLoss', sub: true, fmt: (p) => fmtMwcLoss(p.take_mwc_loss) },
    { label: 'match.takeDecisions', sub: true, fmt: (p) => String(p.take_decisions) }
];

// Transcript marks for a Move's grade (ADR-0046): `?` Error, `??` Blunder. The grade comes from
// the backend (GetMatchMoveGrades), so the Transcript cannot draw a line the statistics do not.
export const GRADE_MARKS = { error: '?', blunder: '??' };

/**
 * Index a match's MoveGrade list by move id, keeping only graded Moves.
 * @param {MoveGrade[] | null | undefined} grades
 * @returns {Map<number, MoveGrade>}
 */
export function indexMoveGrades(grades) {
    /** @type {Map<number, MoveGrade>} */
    const byMove = new Map();
    for (const g of grades || []) {
        if (g && GRADE_MARKS[/** @type {'error' | 'blunder'} */ (g.grade)]) byMove.set(g.move_id, g);
    }
    return byMove;
}

/**
 * Count the marks of one game's moves, as they appear in its rows.
 * @param {{ grade?: { grade?: string } | null }[]} moves
 * @returns {{ errors: number, blunders: number }}
 */
export function countGrades(moves) {
    let errors = 0;
    let blunders = 0;
    for (const { grade } of moves) {
        if (grade?.grade === 'blunder') blunders++;
        else if (grade?.grade === 'error') errors++;
    }
    return { errors, blunders };
}

/**
 * A play's cost in equity, the unit every table shows (millipoints stored).
 * @param {number | string} mp
 * @returns {string}
 */
export const fmtGradeCost = (mp) => (Number(mp) / 1000).toFixed(3);
