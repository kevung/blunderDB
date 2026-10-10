// What the match panel's repair shows of a RepairMatch report
// (ingest.RepairReport), and the message key of its refusals.

/**
 * @typedef {{ moves: number, positioned: number, scored: number }} RepairCounts
 * @typedef {{ before: RepairCounts, after: RepairCounts, changed_moves: number }} RepairReport
 */

/** @param {RepairCounts} c */
function percent(c) {
    return c && c.moves ? Math.round((100 * c.scored) / c.moves) : 0;
}

/**
 * Whether the stored match differs from its source file (RepairReport.Differs).
 * @param {RepairReport} r
 */
export function repairDiffers(r) {
    if (!r) return false;
    const b = r.before || {},
        a = r.after || {};
    return r.changed_moves > 0 || b.moves !== a.moves || b.positioned !== a.positioned || b.scored !== a.scored;
}

/**
 * The figures the confirmation and the status bar interpolate.
 * @param {RepairReport} r
 */
export function repairCounts(r) {
    return { changed: r.changed_moves, before: percent(r.before), after: percent(r.after) };
}

/**
 * The i18n key for what RepairMatch rejected with: its two refusals are
 * recognised by their message (ingest.ErrRepairNoSource / ErrRepairSourceChanged).
 * @param {unknown} error
 */
export function repairMatchErrorKey(error) {
    const msg = String(error && typeof error === 'object' && 'message' in error ? error.message : error);
    if (msg.includes('source file of the match cannot be read')) return 'match.repairNoSource';
    if (msg.includes('source file no longer holds this match')) return 'match.repairSourceChanged';
    return 'match.errorRepairing';
}
