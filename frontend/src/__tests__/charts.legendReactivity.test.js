/**
 * charts.legendReactivity.test.js — #205
 *
 * BarChart/LineChart/ScatterChart each built their chart.js `options` from a
 * plain top-level `const baseOptions = { … legend: { display: datasets?.length
 * > 1 } … }` — a real bug, not just a compiler nag (`state_referenced_locally`):
 * that object is evaluated once, at the component's first run, so a chart
 * created with a single series kept its legend hidden forever, even after a
 * second series appeared later (the exact case the `> 1` check exists for).
 * `baseOptions` now lives inside the redraw effect, where `datasets` is read
 * live. This locks the fix for all three chart components.
 */

import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';

let lastConfig = null;
class FakeChart {
    constructor(_canvas, config) {
        lastConfig = config;
    }
    destroy() {}
}

vi.mock('../components/stats/charts/chartjs.js', () => ({
    loadChart: () => Promise.resolve(FakeChart)
}));

import BarChart from '../components/stats/charts/BarChart.svelte';
import LineChart from '../components/stats/charts/LineChart.svelte';
import ScatterChart from '../components/stats/charts/ScatterChart.svelte';

afterEach(() => {
    cleanup();
    lastConfig = null;
});

/** Poll until the fake Chart has been constructed at least `times` times. */
async function waitForChart() {
    for (let i = 0; i < 50; i++) {
        if (lastConfig) return;
        await new Promise((r) => setTimeout(r, 0));
    }
    throw new Error('chart was never constructed');
}

const CASES = [
    ['BarChart', BarChart, { labels: ['a'], datasets: [{ data: [1] }] }, { labels: ['a'], datasets: [{ data: [1] }, { data: [2] }] }],
    ['LineChart', LineChart, { labels: ['a'], datasets: [{ data: [1] }] }, { labels: ['a'], datasets: [{ data: [1] }, { data: [2] }] }],
    ['ScatterChart', ScatterChart, { datasets: [{ data: [{ x: 1, y: 1 }] }] }, { datasets: [{ data: [{ x: 1, y: 1 }] }, { data: [{ x: 2, y: 2 }] }] }]
];

describe.each(CASES)('%s — legend reflects the current dataset count', (_name, Component, oneSeries, twoSeries) => {
    test('legend is hidden with one series, then appears once a second series is added', async () => {
        const { rerender } = render(Component, { props: oneSeries });
        await waitForChart();
        expect(lastConfig.options.plugins.legend.display).toBe(false);

        lastConfig = null;
        await rerender(twoSeries);
        await waitForChart();
        expect(lastConfig.options.plugins.legend.display).toBe(true);
    });
});
