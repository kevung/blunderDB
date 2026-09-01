/**
 * Lazy loader for chart.js.
 *
 * chart.js only serves the Statistics panel, yet a static import would put it
 * in the single main chunk every user downloads at startup. The three chart
 * components share this loader so the library is fetched once, on first use,
 * as its own chunk; the registrables they need are registered here, once,
 * instead of in each component.
 */
import { logger } from '../../../utils/logger.js';

/** @type {Promise<typeof import('chart.js').Chart | null> | null} */
let pending = null;

/**
 * Resolve the chart.js `Chart` class with every controller, element, scale
 * and plugin the stats charts use already registered.
 *
 * The promise is memoised and never rejects: a failed load is logged,
 * resolves to `null` (the caller leaves its canvas blank) and is forgotten,
 * so the next mount retries instead of staying broken for the session.
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
