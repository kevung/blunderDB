// What the match panel's repair shows of a RepairMatch report
// (ingest.RepairReport), and the message key of its refusals.

/**
 * @typedef {{ moves: number, positioned: number, scored: number }} RepairCounts
 * @typedef {{ before: RepairCounts, after: RepairCounts, changed_moves: number, kept_comments?: number }} RepairReport
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
    return { changed: r.changed_moves, before: percent(r.before), after: percent(r.after), kept: r.kept_comments || 0 };
}

/**
 * The i18n key of the status after a repair: it names the comments left on
 * a position something else still holds, when there are any.
 * @param {RepairReport} r
 */
export function repairDoneKey(r) {
    return r && r.kept_comments ? 'match.repairedKept' : 'match.repaired';
}

/**
 * The i18n key for what RepairMatch rejected with: its refusals are
 * recognised by their message (ingest.ErrRepairNoSource / ErrRepairSourceChanged /
 * ErrRepairSeatsUnclear).
 * @param {unknown} error
 */
export function repairMatchErrorKey(error) {
    const msg = String(error && typeof error === 'object' && 'message' in error ? error.message : error);
    if (msg.includes('source file of the match cannot be read')) return 'match.repairNoSource';
    if (msg.includes('source file no longer holds this match')) return 'match.repairSourceChanged';
    if (msg.includes('cannot be matched with its source file')) return 'match.repairSeatsUnclear';
    return 'match.errorRepairing';
}
