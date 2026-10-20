/**
 * Startup milestones as `performance.mark` entries, prefixed `bdb:`. They cost nothing
 * when nobody reads them; `scripts/measure-startup.mjs` and the WebView inspector
 * (Performance → Timings) read them to say where the time to a usable window goes.
 */

/** @param {string} name */
export function markStartup(name) {
    try {
        performance.mark(`bdb:${name}`);
    } catch {
        // No Performance API (an old test environment): the marks are diagnostics only.
    }
}
