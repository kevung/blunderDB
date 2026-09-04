/**
 * StatsErrorsTab.render.test.js
 *
 * StatsErrorsTab.test.js (D.13, #214) mirrors the component's private
 * `bucketLabel`/`blunderRate`/dataset-selection functions as local copies and
 * asserts against those copies — never against `StatsErrorsTab.svelte`
 * itself. That is why the component showed 0% coverage despite 47 tests
 * bearing its name: none of them import or render it, so a bug in the real
 * file (or a divergence introduced by editing one side and not the other)
 * would never fail a test. This file renders the actual component (chart.js
 * itself mocked out, the same way `charts.legendReactivity.test.js` does)
 * so the real template and helpers are what gets exercised.
 */

import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, screen } from '@testing-library/svelte';

let lastCubeConfig = null;
let lastCompConfig = null;
let lastHistConfig = null;
let constructionOrder = 0;

class FakeChart {
    constructor(canvas, config) {
        constructionOrder += 1;
        // The three BarChart/Histogram instances mount in the same order the
        // markup declares them (cube breakdown, checker-vs-cube, histogram);
        // capture each into its own slot by construction order.
        if (constructionOrder % 3 === 1) lastCubeConfig = config;
        else if (constructionOrder % 3 === 2) lastCompConfig = config;
        else lastHistConfig = config;
    }
    destroy() {}
}

vi.mock('../components/stats/charts/chartjs.js', () => ({
    loadChart: () => Promise.resolve(FakeChart)
}));

vi.mock('../services/positionLoader.js', () => ({
    loadPositionsFromStatsSelection: vi.fn()
}));

import StatsErrorsTab from '../components/stats/StatsErrorsTab.svelte';
import { loadPositionsFromStatsSelection } from '../services/positionLoader.js';

afterEach(() => {
    cleanup();
    lastCubeConfig = null;
    lastCompConfig = null;
    lastHistConfig = null;
    constructionOrder = 0;
    vi.clearAllMocks();
});

/** Poll until all three charts have been constructed. */
async function waitForCharts() {
    for (let i = 0; i < 50; i++) {
        if (lastCubeConfig && lastCompConfig && lastHistConfig) return;
        await new Promise((r) => setTimeout(r, 0));
    }
    throw new Error('charts were never constructed');
}

const SAMPLE_RESULT = {
    Totals: { NumDecisions: 350 },
    PRChecker: 3.2,
    PRCube: 5.5,
    MWCChecker: 0.018,
    MWCCube: 0.032,
    CubeActionBreakdown: [
        { Action: 'NoDouble', PR: 3.5, MWC: 0.021, NumDecisions: 80, BlunderCount: 5 },
        { Action: 'DoubleTake', PR: 4.2, MWC: 0.028, NumDecisions: 60, BlunderCount: 8 }
    ],
    ErrorHistogram: [
        { MinMP: 0, MaxMP: 5, Count: 120 },
        { MinMP: 100, MaxMP: -1, Count: 4 }
    ]
};

describe('StatsErrorsTab — empty state', () => {
    test('no result: shows the empty-state message, no charts built', async () => {
        render(StatsErrorsTab, { props: { result: null, metric: 'pr' } });
        expect(screen.getByText(/no.*decision/i)).toBeTruthy();
        await new Promise((r) => setTimeout(r, 0));
        expect(lastCubeConfig).toBeNull();
    });

    test('zero decisions: also treated as empty', async () => {
        render(StatsErrorsTab, { props: { result: { Totals: { NumDecisions: 0 } }, metric: 'pr' } });
        expect(screen.getByText(/no.*decision/i)).toBeTruthy();
    });
});

describe('StatsErrorsTab — populated result', () => {
    test('renders the cube-action breakdown with the real bucket/PR values (PR metric)', async () => {
        render(StatsErrorsTab, { props: { result: SAMPLE_RESULT, metric: 'pr' } });
        await waitForCharts();

        expect(lastCubeConfig.data.labels).toEqual(['NoDouble', 'DoubleTake']);
        expect(lastCubeConfig.data.datasets[0].data).toEqual([3.5, 4.2]);
        expect(lastCompConfig.data.datasets[0].data).toEqual([3.2, 5.5]);
        expect(lastHistConfig.data.labels).toEqual(['0.000–0.005', '≥0.100']);
        expect(lastHistConfig.data.datasets[0].data).toEqual([120, 4]);
    });

    test('switches to MWC values when metric=mwc', async () => {
        render(StatsErrorsTab, { props: { result: SAMPLE_RESULT, metric: 'mwc' } });
        await waitForCharts();

        expect(lastCubeConfig.data.datasets[0].data).toEqual([0.021, 0.028]);
        expect(lastCompConfig.data.datasets[0].data).toEqual([0.018, 0.032]);
    });

    test('clicking a cube-action bar loads the filtered positions for that action', async () => {
        render(StatsErrorsTab, { props: { result: SAMPLE_RESULT, metric: 'pr' } });
        await waitForCharts();

        lastCubeConfig.options.onClick({}, [{ datasetIndex: 0, index: 1 }]);

        expect(loadPositionsFromStatsSelection).toHaveBeenCalledWith(expect.anything(), {
            Kind: 'cube_action',
            CubeAction: 'DoubleTake',
            OnlyWithError: true
        });
    });

    test('clicking the checker/cube comparison bar distinguishes the two sides', async () => {
        render(StatsErrorsTab, { props: { result: SAMPLE_RESULT, metric: 'pr' } });
        await waitForCharts();

        lastCompConfig.options.onClick({}, [{ datasetIndex: 0, index: 0 }]);
        expect(loadPositionsFromStatsSelection).toHaveBeenCalledWith(expect.anything(), { Kind: 'checker', OnlyWithError: true });

        lastCompConfig.options.onClick({}, [{ datasetIndex: 0, index: 1 }]);
        expect(loadPositionsFromStatsSelection).toHaveBeenCalledWith(expect.anything(), { Kind: 'cube', OnlyWithError: true });
    });

    test('clicking a histogram bucket loads positions in that error range', async () => {
        render(StatsErrorsTab, { props: { result: SAMPLE_RESULT, metric: 'pr' } });
        await waitForCharts();

        lastHistConfig.options.onClick({}, [{ datasetIndex: 0, index: 1 }]);

        expect(loadPositionsFromStatsSelection).toHaveBeenCalledWith(expect.anything(), {
            Kind: 'error_bucket',
            BucketMinMP: 100,
            BucketMaxMP: -1
        });
    });

    test('the tooltip reports the real blunder rate, including a zero-decision bucket', async () => {
        const result = {
            ...SAMPLE_RESULT,
            CubeActionBreakdown: [
                { Action: 'NoDouble', PR: 3.5, MWC: 0.021, NumDecisions: 80, BlunderCount: 5 },
                { Action: 'DoublePass', PR: 0, MWC: 0, NumDecisions: 0, BlunderCount: 0 }
            ]
        };
        render(StatsErrorsTab, { props: { result, metric: 'pr' } });
        await waitForCharts();

        const afterBody = lastCubeConfig.options.plugins.tooltip.callbacks.afterBody;
        expect(afterBody([{ dataIndex: 0 }])).toEqual(['Decisions: 80', 'Blunders: 5 (6.3%)']);
        expect(afterBody([{ dataIndex: 1 }])).toEqual(['Decisions: 0', 'Blunders: 0 (0.0%)']);
    });

    test('histogram bucket labels cover every boundary shape, not just the sample two', async () => {
        const result = {
            ...SAMPLE_RESULT,
            ErrorHistogram: [
                { MinMP: 0, MaxMP: 5, Count: 1 },
                { MinMP: 5, MaxMP: 10, Count: 1 },
                { MinMP: 25, MaxMP: 50, Count: 1 },
                { MinMP: 100, MaxMP: -1, Count: 1 }
            ]
        };
        render(StatsErrorsTab, { props: { result, metric: 'pr' } });
        await waitForCharts();

        expect(lastHistConfig.data.labels).toEqual(['0.000–0.005', '0.005–0.010', '0.025–0.050', '≥0.100']);
    });

    test('no cube decisions: the breakdown chart is skipped, not rendered on empty data', async () => {
        const result = { ...SAMPLE_RESULT, CubeActionBreakdown: [] };
        render(StatsErrorsTab, { props: { result, metric: 'pr' } });
        // Only the comparison chart (always shown) constructs; give the effects
        // a tick and confirm the cube-breakdown slot never got a config. Both
        // the breakdown AND the directions sections fall back to this same
        // empty-subsection message when there is no cube data at all.
        await new Promise((r) => setTimeout(r, 0));
        expect(screen.getAllByText(/no cube decisions/i)).toHaveLength(2);
    });

    test('all-zero histogram: the empty-subsection message is shown instead of the chart', async () => {
        const result = { ...SAMPLE_RESULT, ErrorHistogram: SAMPLE_RESULT.ErrorHistogram.map((b) => ({ ...b, Count: 0 })) };
        render(StatsErrorsTab, { props: { result, metric: 'pr' } });
        expect(screen.getByText(/no errors/i)).toBeTruthy();
    });
});

describe('StatsErrorsTab — cube error directions', () => {
    const withDirections = {
        ...SAMPLE_RESULT,
        CubeDirections: {
            Offer: { Right: 10, Missed: 2, MissedMP: 500, Premature: 1, PrematureMP: 200 },
            Answer: { Right: 8, WrongPass: 3, WrongPassMP: 900, WrongTake: 0, WrongTakeMP: 0 }
        }
    };

    test('renders one button per non-empty cell and none for the empty one', async () => {
        render(StatsErrorsTab, { props: { result: withDirections, metric: 'pr' } });

        const missedBtn = screen.getByText('2').closest('button');
        expect(missedBtn.disabled).toBe(false);

        const wrongTakeBtn = screen.getByText('0').closest('button');
        expect(wrongTakeBtn.disabled).toBe(true);
    });

    test('clicking a non-empty direction cell loads positions for it, an empty cell is inert', async () => {
        render(StatsErrorsTab, { props: { result: withDirections, metric: 'pr' } });

        await screen.getByText('2').closest('button').click();
        expect(loadPositionsFromStatsSelection).toHaveBeenCalledWith(expect.anything(), { Kind: 'cube_direction', CubeCell: 'offer_missed' });

        loadPositionsFromStatsSelection.mockClear();
        screen.getByText('0').closest('button').click();
        expect(loadPositionsFromStatsSelection).not.toHaveBeenCalled();
    });

    test('all directions at zero: shows the empty message instead of the table', () => {
        const allZero = {
            ...SAMPLE_RESULT,
            CubeDirections: {
                Offer: { Right: 0, Missed: 0, MissedMP: 0, Premature: 0, PrematureMP: 0 },
                Answer: { Right: 0, WrongPass: 0, WrongPassMP: 0, WrongTake: 0, WrongTakeMP: 0 }
            }
        };
        render(StatsErrorsTab, { props: { result: allZero, metric: 'pr' } });
        expect(screen.getByText(/no cube decisions/i)).toBeTruthy();
    });
});
