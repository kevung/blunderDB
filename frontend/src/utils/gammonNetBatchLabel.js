/**
 * The status-bar chip of a gammonNet batch: its depth when the event names
 * one, so a slow deep batch says why it is slow; the job it waits behind
 * while it is queued.
 * @param {(key: string, params?: object) => string} t
 * @param {{ done: number, total: number, ply?: number, job?: string }} batch
 */
export function gammonNetBatchLabel(t, batch) {
    if (batch.job) return t('eval.batchWaiting', { job: t(`jobs.${batch.job}`) });
    const counts = { done: batch.done, total: batch.total };
    if (!Number.isInteger(batch.ply)) return t('eval.batchProgress', counts);
    return t('eval.batchProgressAt', { ...counts, depth: t('config.gammonnetPly', { n: batch.ply }) });
}
