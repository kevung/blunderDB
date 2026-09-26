/**
 * Lazy loader for chart.js, kept out of the main chunk; registrables are
 * registered here once for the three chart components.
 */
import { logger } from '../../../utils/logger.js';

/** @type {Promise<typeof import('chart.js').Chart | null> | null} */
let pending = null;

/**
 * Resolve the chart.js `Chart` class with every controller, element, scale
 * and plugin the stats charts use already registered.
 *
 * Memoised, never rejects: a failed load logs, resolves to `null` and is
 * forgotten so the next mount retries.
 *
 * @returns {Promise<typeof import('chart.js').Chart | null>}
 */
export function loadChart() {
    if (!pending) {
        pending = import('chart.js')
            .then((m) => {
                m.Chart.register(m.LineController, m.BarController, m.ScatterController, m.LineElement, m.BarElement, m.PointElement, m.LinearScale, m.CategoryScale, m.Tooltip, m.Legend, m.Filler);
                return m.Chart;
            })
            .catch((err) => {
                logger.error('chart.js failed to load:', err);
                pending = null;
                return null;
            });
    }
    return pending;
}
