/**
 * Counts open what they count: the dashboard's tournaments and matches open their lists, its decisions open
 * the positions, under the filter the figures were computed with; a training period's match
 * decisions open that period's positions.
 */
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent, waitFor } from '@testing-library/svelte';

vi.mock('../components/stats/charts/chartjs.js', () => ({
    loadChart: () =>
        Promise.resolve(
            class FakeChart {
                destroy() {}
            }
        )
}));
vi.mock('../../wailsjs/go/database/Database.js', () => ({ StatsMatchIDs: vi.fn(() => Promise.resolve([11, 12])) }));
vi.mock('../../wailsjs/go/main/Config.js', () => ({ GetStatsFilter: vi.fn(() => Promise.resolve(null)), SaveStatsFilter: vi.fn(() => Promise.resolve()) }));
vi.mock('../stores/databaseStore.js', () => {
    const { writable } = require('svelte/store');
    return { databasePathStore: writable('/db'), databaseLoadedStore: writable(true) };
});
vi.mock('../services/positionLoader.js', () => ({
    loadPositionsFromStatsSelection: vi.fn(() => Promise.resolve()),
    openMatchListInPanel: vi.fn(() => Promise.resolve()),
    openMatchInPanel: vi.fn(),
    openStatsTournaments: vi.fn(() => Promise.resolve())
}));

import { StatsMatchIDs } from '../../wailsjs/go/database/Database.js';
import { loadPositionsFromStatsSelection, openMatchListInPanel, openStatsTournaments } from '../services/positionLoader.js';
import { statsResultFilterStore, patchStatsFilter, _resetStatsFilterForTests } from '../stores/statsStore.js';
import StatsDashboardTab from '../components/stats/StatsDashboardTab.svelte';
import StatsTrainingTab from '../components/stats/StatsTrainingTab.svelte';

const ALL = { Kind: 'all', OnlyWithError: false };
const result = /** @type {any} */ ({ Totals: { NumTournaments: 1, NumMatches: 2, NumDecisions: 40 }, PRGlobal: 5, PRChecker: 5, PRCube: 5 });

beforeEach(() => {
    vi.clearAllMocks();
    _resetStatsFilterForTests();
    statsResultFilterStore.set(null);
});
afterEach(() => cleanup());

/** @param {HTMLElement} container */
const totalsLinks = (container) => /** @type {HTMLElement[]} */ ([...container.querySelectorAll('.stats-totals [data-testid="count-link"]')]);

describe('Dashboard totals', () => {
    test('tournaments open the tournament list, matches the match list, decisions the positions, under the computed filter', async () => {
        const computedUnder = { playerName: 'Alice', tournamentIDs: [], dateFrom: '', dateTo: '', decisionType: -1, matchLength: [], analysisEngine: '', minAnalysisDepth: 0 };
        statsResultFilterStore.set(computedUnder);
        patchStatsFilter({ playerName: 'Bob' });
        const { container } = render(StatsDashboardTab, { props: { result, metric: 'pr' } });
        const [tournaments, matches, decisions] = totalsLinks(container);
        expect([tournaments.textContent, matches.textContent, decisions.textContent]).toEqual(['1', '2', '40']);

        await fireEvent.click(tournaments);
        expect(openStatsTournaments).toHaveBeenCalled();

        await fireEvent.click(matches);
        expect(StatsMatchIDs).toHaveBeenCalledWith(computedUnder);
        await waitFor(() => expect(openMatchListInPanel).toHaveBeenCalledWith([11, 12]));

        await fireEvent.click(decisions);
        expect(loadPositionsFromStatsSelection).toHaveBeenCalledWith(computedUnder, ALL);
    });

    test('a decision type narrows the totals below the match list: the match count stays text', () => {
        patchStatsFilter({ decisionType: 1 });
        const { container } = render(StatsDashboardTab, { props: { result, metric: 'pr' } });
        expect(totalsLinks(container).map((l) => l.textContent)).toEqual(['1', '40']);
    });
});

describe('Training periods', () => {
    test("a period's match decisions open that week's positions", async () => {
        patchStatsFilter({ playerName: 'Alice' });
        const data = {
            Window: 'week',
            Sessions: [],
            Themes: [],
            Periods: [{ Start: '2025-12-29', QuizDecisions: 0, QuizPR: 0, MatchDecisions: 30, MatchPR: 4, AnkiReviews: 0, AnkiRetention: 0 }]
        };
        const { container } = render(StatsTrainingTab, { props: { data } });
        const link = /** @type {HTMLElement} */ (container.querySelector('[data-testid="count-link"]'));
        expect(link.textContent).toBe('30');
        await fireEvent.click(link);
        expect(loadPositionsFromStatsSelection).toHaveBeenCalledWith(expect.objectContaining({ playerName: 'Alice', dateFrom: '2025-12-29', dateTo: '2026-01-04' }), ALL);
    });
});
