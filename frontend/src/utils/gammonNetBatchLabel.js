/**
 * The status-bar chip of a gammonNet batch: its depth when the event names
 * one, so a slow deep batch says why it is slow.
 * @param {(key: string, params?: object) => string} t
 * @param {{ done: number, total: number, ply?: number }} batch
 */
export function gammonNetBatchLabel(t, batch) {
    const counts = { done: batch.done, total: batch.total };
    if (!Number.isInteger(batch.ply)) return t('eval.batchProgress', counts);
    return t('eval.batchProgressAt', { ...counts, depth: t('config.gammonnetPly', { n: batch.ply }) });
}
