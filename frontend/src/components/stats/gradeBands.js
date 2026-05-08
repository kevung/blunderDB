/**
 * Backgammon skill-grade horizontal bands for progression charts.
 *
 * Thresholds follow XG / gnuBG PR conventions.
 * Colours are very low-alpha fills so they remain visible but non-invasive.
 */

export const GRADE_BANDS = [
    { label: 'World Class', min: 0, max: 2, color: 'rgba(46, 125, 50,  0.10)' },
    { label: 'Expert', min: 2, max: 4, color: 'rgba(100, 160, 50, 0.09)' },
    { label: 'Advanced', min: 4, max: 6, color: 'rgba(200, 160, 30, 0.09)' },
    { label: 'Intermediate', min: 6, max: 9, color: 'rgba(230, 120, 20, 0.09)' },
    { label: 'Casual', min: 9, max: 12, color: 'rgba(200,  60, 30, 0.09)' },
    { label: 'Beginner', min: 12, max: Infinity, color: 'rgba(183,  28, 28, 0.10)' }
];

/**
 * Return the grade label for a given PR value.
 * @param {number} pr
 * @returns {string}
 */
export function gradeForPR(pr) {
    for (const band of GRADE_BANDS) {
        if (pr < band.max) return band.label;
    }
    return 'Beginner';
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
