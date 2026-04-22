/* eslint-disable no-console */
/**
 * Logger utility for blunderDB.
 *
 * Perf instrumentation — activation :
 *   VITE_PERF_THRESHOLD_MS=0  npm run dev   → logge tous les appels (même < 1 ms)
 *   VITE_PERF_THRESHOLD_MS=50 npm run dev   → logge uniquement les appels > 50 ms
 *   VITE_PERF_THRESHOLD_MS=-1               → désactive même en dev (défaut prod)
 * Inactif en prod (import.meta.env.DEV = false) et si seuil négatif.
 */
const isDev = import.meta.env.DEV;

export const logger = {
    log: (...args) => isDev && console.log(...args),
    warn: (...args) => isDev && console.warn(...args),
    error: (...args) => console.error(...args),
    debug: (...args) => isDev && console.debug(...args),
    perf,
};

/**
 * Mesure la durée d'exécution de `fn` avec `performance.measure`.
 * Logge uniquement si la durée dépasse VITE_PERF_THRESHOLD_MS (défaut : 16 ms).
 * Sans effet en prod ou si le seuil est négatif.
 *
 * @template T
 * @param {string} label
 * @param {() => T} fn
 * @returns {T}
 */
function perf(label, fn) {
    const threshold = Number(import.meta.env.VITE_PERF_THRESHOLD_MS ?? 16);
    if (!import.meta.env.DEV || threshold < 0) return fn();
    const mark = `perf-${label}-${Math.random().toString(36).slice(2, 7)}`;
    performance.mark(`${mark}-start`);
    const result = fn();
    const finish = () => {
        performance.mark(`${mark}-end`);
        const measure = performance.measure(label, `${mark}-start`, `${mark}-end`);
        if (measure.duration >= threshold) {
            console.log(`[perf] ${label} ${measure.duration.toFixed(2)}ms`);
        }
        performance.clearMarks(`${mark}-start`);
        performance.clearMarks(`${mark}-end`);
        performance.clearMeasures(label);
    };
    if (result && typeof result.then === 'function') {
        return result.finally(finish);
    }
    finish();
    return result;
}
