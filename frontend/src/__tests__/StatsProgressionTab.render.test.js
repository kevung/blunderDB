/**
 * StatsProgressionTab.render.test.js
 *
 * StatsProgressionTab.test.js (D.13, #214) never rendered
 * `StatsProgressionTab.svelte` — its "dataset construction"/"empty state"/
 * "single tournament fallback" describe blocks reimplemented the component's
 * own helpers as local copies (`buildTourDatasets`, `truncateLabel`, …) and
 * asserted against those copies, and its "drill-down" blocks called
 * `positionLoader.js` directly, already covered by `positionLoader.test.js`.
 * None of that ever touched the real file, which is why it showed 0%
 * coverage. The genuinely independent parts (gradeBands.js) moved to
 * `gradeBands.test.js`; this file renders the actual component (chart.js
 * mocked out, the same way `StatsErrorsTab.render.test.js` does) so the real
 * template and its click/drill-down wiring are what gets exercised.
 */

import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, screen, fireEvent } from '@testing-library/svelte';

let lastLineConfig = null;
let lastScatterConfig = null;

class FakeChart {
    constructor(canvas, config) {
        if (config.type === 'line') lastLineConfig = config;
        else lastScatterConfig = config;
    }
    destroy() {}
}

vi.mock('../components/stats/charts/chartjs.js', () => ({
    loadChart: () => Promise.resolve(FakeChart)
}));

vi.mock('../services/positionLoader.js', () => ({
    loadPositionsFromTournament: vi.fn(),
    loadPositionsFromMatch: vi.fn(),
    openTournamentInPanel: vi.fn(),
    openMatchInPanel: vi.fn()
}));

import StatsProgressionTab from '../components/stats/StatsProgressionTab.svelte';
import { loadPositionsFromTournament, openTournamentInPanel, openMatchInPanel } from '../services/positionLoader.js';

afterEach(() => {
    cleanup();
    lastLineConfig = null;
    lastScatterConfig = null;
    vi.clearAllMocks();
});

/** Poll until a chart has been constructed. */
async function waitFor(getConfig) {
    for (let i = 0; i < 50; i++) {
        if (getConfig()) return;
        await new Promise((r) => setTimeout(r, 0));
    }
    throw new Error('chart was never constructed');
}

const TOURNAMENTS = [
    { ID: 1, Name: 'Open de Paris', Date: '2025-01-10', PR: 3.5, MWC: 0.021, NumDecisions: 120 },
    { ID: 2, Name: 'Monte Carlo BG', Date: '2025-04-05', PR: 2.8, MWC: 0.017, NumDecisions: 95 },
    { ID: 3, Name: 'World Cup 2025', Date: '2025-09-20', PR: 4.1, MWC: 0.025, NumDecisions: 200 }
];

const MATCHES = [
    { ID: 10, Date: '2025-01-15T12:00:00Z', PlayerName: 'Alice', PR: 3.2, MWC: 0.019, NumDecisions: 40 },
    { ID: 11, Date: '2025-02-20T15:30:00Z', PlayerName: 'Bob', PR: 5.0, MWC: 0.031, NumDecisions: 28 }
];

const RESULT = { PerTournament: TOURNAMENTS, PerMatch: MATCHES };

describe('StatsProgressionTab — empty state', () => {
    test('no result: shows the empty-state message, no chart built', async () => {
        render(StatsProgressionTab, { props: { result: null, metric: 'pr' } });
        expect(screen.getByText(/no data for the filtered period/i)).toBeTruthy();
        await new Promise((r) => setTimeout(r, 0));
        expect(lastLineConfig).toBeNull();
    });

    test('empty tournaments and matches: also treated as empty', async () => {
        render(StatsProgressionTab, { props: { result: { PerTournament: [], PerMatch: [] }, metric: 'pr' } });
        expect(screen.getByText(/no data for the filtered period/i)).toBeTruthy();
    });
});

describe('StatsProgressionTab — tournament line chart', () => {
    test('renders the real PR values and truncated labels', async () => {
        render(StatsProgressionTab, { props: { result: RESULT, metric: 'pr' } });
        await waitFor(() => lastLineConfig);

        expect(lastLineConfig.data.labels).toEqual(['Open de Paris', 'Monte Carlo BG', 'World Cup 2025']);
        expect(lastLineConfig.data.datasets[0].data).toEqual([3.5, 2.8, 4.1]);
        expect(lastLineConfig.data.datasets[0].label).toBe('PR');
    });

    test('switches to MWC values and label when metric=mwc', async () => {
        render(StatsProgressionTab, { props: { result: RESULT, metric: 'mwc' } });
        await waitFor(() => lastLineConfig);

        expect(lastLineConfig.data.datasets[0].data).toEqual([0.021, 0.017, 0.025]);
        expect(lastLineConfig.data.datasets[0].label).toBe('MWC loss');
    });

    test('clicking a tournament point opens a context menu whose actions drill down', async () => {
        render(StatsProgressionTab, { props: { result: RESULT, metric: 'pr' } });
        await waitFor(() => lastLineConfig);

        lastLineConfig.options.onClick({ native: { clientX: 10, clientY: 20 } }, [{ datasetIndex: 0, index: 1 }]);
        await new Promise((r) => setTimeout(r, 0));

        const openPositionsBtn = screen.getByText('Open positions');
        await fireEvent.click(openPositionsBtn);
        expect(loadPositionsFromTournament).toHaveBeenCalledWith(2);

        // Re-open the menu and use the other action this time.
        lastLineConfig.options.onClick({ native: { clientX: 10, clientY: 20 } }, [{ datasetIndex: 0, index: 0 }]);
        await new Promise((r) => setTimeout(r, 0));
        await fireEvent.click(screen.getByText('Open tournament'));
        expect(openTournamentInPanel).toHaveBeenCalledWith(1);
    });
});

describe('StatsProgressionTab — single tournament fallback', () => {
    const single = { PerTournament: [TOURNAMENTS[0]], PerMatch: [] };

    test('shows a card instead of a chart, with the PR value and grade', async () => {
        render(StatsProgressionTab, { props: { result: single, metric: 'pr' } });
        await new Promise((r) => setTimeout(r, 0));

        expect(screen.getByText('3.50')).toBeTruthy();
        expect(screen.getByText('Open de Paris')).toBeTruthy();
        expect(screen.getByText('Expert')).toBeTruthy();
        expect(lastLineConfig).toBeNull();
    });

    test('the card buttons drill down to the tournament', async () => {
        render(StatsProgressionTab, { props: { result: single, metric: 'pr' } });

        await fireEvent.click(screen.getByText('Open tournament'));
        expect(openTournamentInPanel).toHaveBeenCalledWith(1);

        await fireEvent.click(screen.getByText('Open positions'));
        expect(loadPositionsFromTournament).toHaveBeenCalledWith(1);
    });
});

describe('StatsProgressionTab — match scatter chart', () => {
    test('renders one point per match with clamped radii and the real PR values', async () => {
        render(StatsProgressionTab, { props: { result: RESULT, metric: 'pr' } });
        await waitFor(() => lastScatterConfig);

        expect(lastScatterConfig.data.datasets[0].data.map((p) => p.y)).toEqual([3.2, 5.0]);
        for (const r of lastScatterConfig.data.datasets[0].pointRadius) {
            expect(r).toBeGreaterThanOrEqual(4);
            expect(r).toBeLessThanOrEqual(12);
        }
    });

    test('clicking a match point opens a context menu whose actions drill down', async () => {
        render(StatsProgressionTab, { props: { result: RESULT, metric: 'pr' } });
        await waitFor(() => lastScatterConfig);

        lastScatterConfig.options.onClick({ native: { clientX: 5, clientY: 5 } }, [{ datasetIndex: 0, index: 0 }]);
        await new Promise((r) => setTimeout(r, 0));

        await fireEvent.click(screen.getByText('Open match'));
        expect(openMatchInPanel).toHaveBeenCalledWith(10);
    });
});

describe('StatsProgressionTab — grade legend', () => {
    test('renders all 6 grade bands', async () => {
        render(StatsProgressionTab, { props: { result: RESULT, metric: 'pr' } });
        const pills = document.querySelectorAll('.grade-pill');
        expect(pills).toHaveLength(6);
        expect(pills[0].textContent).toContain('World Class');
    });
});
