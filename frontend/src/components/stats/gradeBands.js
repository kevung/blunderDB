/**
 * Backgammon skill-grade horizontal bands for progression charts.
 *
 * Thresholds follow XG / gnuBG PR conventions.
 * Colours are very low-alpha fills so they remain visible but non-invasive.
 */

// `label` is the English name (and the test oracle); `key` is the i18n key
// under `stats.grade.*`, which is what the panel renders — the manual
// documents the same six bands in every language, so the interface must
// not be the one place where they stay in English. The upper bound of a
// band is excluded: a PR of exactly 4 is Advanced, not Expert.
export const GRADE_BANDS = [
    { key: 'worldClass', label: 'World Class', min: 0, max: 2, color: 'rgba(46, 125, 50,  0.10)' },
    { key: 'expert', label: 'Expert', min: 2, max: 4, color: 'rgba(100, 160, 50, 0.09)' },
    { key: 'advanced', label: 'Advanced', min: 4, max: 6, color: 'rgba(200, 160, 30, 0.09)' },
    { key: 'intermediate', label: 'Intermediate', min: 6, max: 9, color: 'rgba(230, 120, 20, 0.09)' },
    { key: 'casual', label: 'Casual', min: 9, max: 12, color: 'rgba(200,  60, 30, 0.09)' },
    { key: 'beginner', label: 'Beginner', min: 12, max: Infinity, color: 'rgba(183,  28, 28, 0.10)' }
];

/**
 * Return the grade band for a given PR value.
 * @param {number} pr
 * @returns {(typeof GRADE_BANDS)[number]}
 */
export function bandForPR(pr) {
    for (const band of GRADE_BANDS) {
        if (pr < band.max) return band;
    }
    return GRADE_BANDS[GRADE_BANDS.length - 1];
}

/**
 * Return the English grade label for a given PR value.
 * @param {number} pr
 * @returns {string}
 */
export function gradeForPR(pr) {
    return bandForPR(pr).label;
}

/**
 * Build a Chart.js per-chart plugin that draws horizontal grade-band
 * backgrounds behind the dataset layer.
 *
 * In Chart.js a *per-chart* plugin is an object passed to the `plugins`
 * array in the chart config (not via `Chart.register`).
 *
 * @param {typeof GRADE_BANDS} bands
 * @returns {import('chart.js').Plugin}
 */
export function makeGradeBandPlugin(bands) {
    return {
        id: 'gradeBands',
        beforeDraw(chart) {
            const { ctx, chartArea, scales } = chart;
            const y = scales.y;
            if (!y || !chartArea) return;

            const { top, bottom, left, right } = chartArea;
            const width = right - left;

            ctx.save();
            // Clip drawing to the inner chart area
            ctx.beginPath();
            ctx.rect(left, top, width, bottom - top);
            ctx.clip();

            for (const band of bands) {
                // In Chart.js linear scale (non-inverted):
                //   large value → small pixel y  (visually near top)
                //   small value → large pixel y  (visually near bottom)
                // So band.max → pxTop (smaller y), band.min → pxBot (larger y)
                const pxTop =
                    band.max === Infinity
                        ? top - 1 // extend past chart top to cover any high value
                        : y.getPixelForValue(band.max);
                const pxBot = y.getPixelForValue(band.min);

                // Clamp to visible chart area
                const drawTop = Math.max(top, Math.min(pxTop, pxBot));
                const drawBot = Math.min(bottom, Math.max(pxTop, pxBot));
                if (drawBot <= drawTop) continue; // band off-screen

                ctx.fillStyle = band.color;
                ctx.fillRect(left, drawTop, width, drawBot - drawTop);
            }

            ctx.restore();
        }
    };
}
