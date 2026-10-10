import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent, waitFor } from '@testing-library/svelte';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    GetAllPlayerNames: vi.fn(() => Promise.resolve([{ Name: 'Alice' }, { Name: 'Bob' }])),
    GetAllTournaments: vi.fn(() => Promise.resolve([])),
    GetStatsAnalysisEngines: vi.fn().mockResolvedValue(['XG']),
    GetStatsDateRange: vi.fn(() => Promise.resolve({ DateFrom: '', DateTo: '' })),
    HeadToHead: vi.fn(() =>
        Promise.resolve({
            player_a: 'Alice',
            player_b: 'Bob',
            wins_a: 2,
            wins_b: 1,
            pr_a: 4.5,
            pr_b: 6.25,
            matches: [{ id: 7, date: '2025-01-02', match_length: 7, outcome: 1, pr_a: 4, pr_b: 6 }]
        })
    ),
    PRByWindow: vi.fn(() => Promise.resolve([{ from: '2025-01-01', to: '2025-03-31', num_matches: 3, num_decisions: 420, pr: 5.12 }])),
    PlayerRanking: vi.fn(() => Promise.resolve([{ rank: 1, name: 'Alice', matches: 3, wins: 2, losses: 1, decisions: 600, pr: 4.4, pr_checker: 4.1, pr_cube: 6.2 }]))
}));

vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetStatsFilter: vi.fn(() => Promise.resolve(null)),
    SaveStatsFilter: vi.fn(() => Promise.resolve())
}));

import { HeadToHead, PRByWindow, PlayerRanking } from '../../wailsjs/go/database/Database.js';
import StatsCorpusTab from '../components/stats/StatsCorpusTab.svelte';
import StatsFilterBar from '../components/stats/StatsFilterBar.svelte';
import { statsFilterStore } from '../stores/statsStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { must } from './helpers/must.js';

const filter = { playerName: '', tournamentIDs: [], dateFrom: '', dateTo: '', decisionType: -1, matchLength: [], analysisEngine: 'gnubg', minAnalysisDepth: 2 };

beforeEach(() => vi.clearAllMocks());
afterEach(() => {
    cleanup();
    databasePathStore.set('');
});

describe('StatsCorpusTab', () => {
    test('head-to-head asks for the two players under the current filter and lists their matches', async () => {
        const { container, getByLabelText, findByText } = render(StatsCorpusTab, { props: { filter } });
        await fireEvent.input(getByLabelText('Player A'), { target: { value: 'Alice' } });
        await fireEvent.input(getByLabelText('Player B'), { target: { value: 'Bob' } });
        await fireEvent.click(must(container.querySelector('section:nth-of-type(1) button')));
        expect(HeadToHead).toHaveBeenCalledWith('Alice', 'Bob', filter);
        expect(await findByText('2025-01-02')).toBeTruthy();
        expect(container.textContent).toContain('Alice 2 – 1 Bob');
    });

    test('calendar windows pass the chosen width in months', async () => {
        const { container, getByLabelText, findByText } = render(StatsCorpusTab, { props: { filter } });
        await fireEvent.change(getByLabelText('Months per window'), { target: { value: '6' } });
        await fireEvent.click(must(container.querySelector('section:nth-of-type(2) button')));
        expect(PRByWindow).toHaveBeenCalledWith(filter, 6);
        expect(await findByText('5.12')).toBeTruthy();
    });

    test('ranking passes the decision floor and shows the ranked rows', async () => {
        const { container, getByLabelText, findByText } = render(StatsCorpusTab, { props: { filter } });
        await fireEvent.input(getByLabelText('Min. decisions'), { target: { value: '100' } });
        await fireEvent.click(must(container.querySelector('section:nth-of-type(3) button')));
        expect(PlayerRanking).toHaveBeenCalledWith(filter, 100);
        expect(await findByText('2–1')).toBeTruthy();
    });
});

describe('StatsFilterBar — provenance', () => {
    test('engine and minimum depth reach the stats filter', async () => {
        databasePathStore.set('/tmp/x.db');
        const { findByLabelText } = render(StatsFilterBar);
        const engine = await findByLabelText('Engine');
        await waitFor(() => expect(engine.querySelectorAll('option').length).toBe(2));
        await fireEvent.change(engine, { target: { value: 'XG' } });
        await waitFor(() => expect(get(statsFilterStore).analysisEngine).toBe('XG'));
        const depth = await findByLabelText('Min. depth');
        await fireEvent.change(depth, { target: { value: '3' } });
        await waitFor(() => expect(get(statsFilterStore).minAnalysisDepth).toBe(3));
    });
});
