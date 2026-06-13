// Pure helpers for the Matches panel table: sorting comparators and the date /
// dice formatters. Extracted from MatchPanel.svelte so they can be unit-tested
// without mounting the component (they close over no component state).

/**
 * Comparator with null-handling: nulls sort last, strings compare
 * case-insensitively, numbers numerically.
 */
export function compareValues(a, b) {
    if (a == null && b == null) return 0;
    if (a == null) return 1;
    if (b == null) return -1;
    if (typeof a === 'string') return a.localeCompare(b, undefined, { sensitivity: 'base' });
    return a - b;
}

/**
 * Extract the sortable value for a match table column. Mirrors the column keys
 * used by the Matches panel header.
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
        default:
            return '';
    }
}

/**
 * Sort a copy of `matches` by `column` in `direction` ('asc' | 'desc').
 * Returns `matches` unchanged when no column is given. The Matches panel's
 * `sortedMatches` derived state delegates here.
 */
export function sortMatches(matches, column, direction) {
    if (!column) return matches;
    const sorted = [...matches].sort((a, b) => {
        const cmp = compareValues(getSortValue(a, column), getSortValue(b, column));
        return direction === 'asc' ? cmp : -cmp;
    });
    return sorted;
}

/** Convert a date string to a `yyyy-mm-dd` value for a date <input>; '' if invalid. */
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

/** Format a date string as `yyyy/mm/dd` for display; '-' if empty or invalid. */
export function formatDate(dateStr) {
    if (!dateStr) return '-';
    const date = new Date(dateStr);
    if (isNaN(date.getTime())) return '-';
    const y = date.getFullYear();
    const m = String(date.getMonth() + 1).padStart(2, '0');
    const d = String(date.getDate()).padStart(2, '0');
    return `${y}/${m}/${d}`;
}

/** Compact dice rendering, e.g. [3,1] → "31"; '' when there are no dice. */
export function formatDiceShort(dice) {
    if (!dice || (!dice[0] && !dice[1])) return '';
    return `${dice[0]}${dice[1]}`;
}

// --- Match-detail per-player stats table -------------------------------------

// Cell formatters for the MatchPanel "stats" tab. A player's MatchPlayerDetailStats
// maps to a displayed string; em-dash ("—") marks "not applicable / no data".
export const fmtPR = (val, decisions) => (decisions > 0 ? val.toFixed(2) : '—');
export const fmtEquityError = (v) => (v > 0 ? '-' + v.toFixed(3) : '—');
export const fmtMwcLoss = (v) => (v > 0 ? '-' + (v * 100).toFixed(2) + '%' : '—');
export const fmtErrorsBlunders = (errors, blunders) => `${errors} (${blunders})`;

// MATCH_STAT_ROWS describes the per-player match stats table row by row. Each
// entry is either a section header ({ section }) or a metric ({ label, fmt, … });
// fmt maps one player's stats to its cell. Rendered by MatchPanel; labels are
// i18n keys translated at render time. bullet = leading "•"; sub = indented
// sub-metric; valClass = extra value-cell class.
export const MATCH_STAT_ROWS = [
    { section: 'match.performanceRating' },
    { label: 'match.overallPr', bullet: true, valClass: 'pr-val', fmt: (p) => fmtPR(p.pr, p.total_decisions) },
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
