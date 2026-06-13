// Shared table-header sort-toggle logic, factored out of the per-panel
// handleSort functions (MatchPanel, TournamentPanel, AnalysisPanel) which had
// each reimplemented the same click-to-cycle behaviour.

/**
 * Compute the next sort state when a sortable column header is clicked.
 *
 * Clicking the currently-sorted column cycles its direction: ascending →
 * descending, then (when `tristate`) back to unsorted (`column: null`), else
 * back to ascending. Clicking a different column selects it with `defaultDir`.
 *
 * @param {string|null} curColumn   currently sorted column (null = unsorted)
 * @param {'asc'|'desc'} curDirection current direction
 * @param {string} column           the clicked column
 * @param {{tristate?: boolean, defaultDir?: 'asc'|'desc'}} [opts]
 *   tristate  — a third click on the same column clears the sort (default false)
 *   defaultDir — direction when switching to a different column (default 'asc')
 * @returns {{column: string|null, direction: 'asc'|'desc'}}
 */
export function nextSort(curColumn, curDirection, column, opts = {}) {
    const { tristate = false, defaultDir = 'asc' } = opts;
    if (curColumn === column) {
        if (curDirection === 'asc') return { column, direction: 'desc' };
        if (tristate) return { column: null, direction: 'asc' };
        return { column, direction: 'asc' };
    }
    return { column, direction: defaultDir };
}
