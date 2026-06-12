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
