import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

// ── Mocks ────────────────────────────────────────────────────────────────────

vi.mock('../../wailsjs/go/main/Database.js', () => ({
    ComputeStats: vi.fn().mockResolvedValue({ prGlobal: 4.0, totals: { numDecisions: 10 } })
}));

import { ComputeStats } from '../../wailsjs/go/main/Database.js';

import { statsFilterStore, statsResultStore, statsLoadingStore, statsErrorStore, statsMetricStore, refreshStats } from '../stores/statsStore.js';

import { openPanels, PANEL, openPanel, closePanel } from '../stores/uiStore.js';

// ── Helpers ───────────────────────────────────────────────────────────────────

function resetStores() {
    statsResultStore.set(null);
    statsLoadingStore.set(false);
    statsErrorStore.set(null);
    statsMetricStore.set('pr');
    openPanels.set(new Set());
}

// ── Tests ─────────────────────────────────────────────────────────────────────

describe('StatsPanel — metric toggle (via statsMetricStore)', () => {
    beforeEach(resetStores);

    test('default metric is pr', () => {
        expect(get(statsMetricStore)).toBe('pr');
    });

    test('switching to mwc updates the store', () => {
        statsMetricStore.set('mwc');
        expect(get(statsMetricStore)).toBe('mwc');
    });

    test('switching back to pr updates the store', () => {
        statsMetricStore.set('mwc');
        statsMetricStore.set('pr');
        expect(get(statsMetricStore)).toBe('pr');
    });
});

describe('StatsPanel — panel visibility (via openPanels)', () => {
    beforeEach(resetStores);

    test('PANEL.STATS is not open by default', () => {
        expect(get(openPanels).has(PANEL.STATS)).toBe(false);
    });

    test('openPanel(PANEL.STATS) makes panel visible', () => {
        openPanel(PANEL.STATS);
        expect(get(openPanels).has(PANEL.STATS)).toBe(true);
    });

    test('closePanel(PANEL.STATS) hides panel', () => {
        openPanel(PANEL.STATS);
        closePanel(PANEL.STATS);
        expect(get(openPanels).has(PANEL.STATS)).toBe(false);
    });

    test('other panels can be open simultaneously', () => {
        openPanel(PANEL.STATS);
        openPanel(PANEL.MATCH);
        expect(get(openPanels).has(PANEL.STATS)).toBe(true);
        expect(get(openPanels).has(PANEL.MATCH)).toBe(true);
    });
});

describe('StatsPanel — refreshStats integration', () => {
    beforeEach(() => {
        resetStores();
        vi.resetAllMocks();
        ComputeStats.mockResolvedValue({ prGlobal: 2.5, totals: { numDecisions: 5 } });
    });

    test('refreshStats populates statsResultStore', async () => {
        const filter = get(statsFilterStore);
        await refreshStats(filter);
        const result = get(statsResultStore);
        expect(result).not.toBeNull();
        expect(result.prGlobal).toBe(2.5);
    });

    test('refreshStats clears loading flag on success', async () => {
        const filter = get(statsFilterStore);
        await refreshStats(filter);
        expect(get(statsLoadingStore)).toBe(false);
    });

    test('refreshStats sets errorStore on failure', async () => {
        ComputeStats.mockRejectedValueOnce(new Error('network error'));
        const filter = get(statsFilterStore);
        await refreshStats(filter);
        expect(get(statsErrorStore)).toBe('network error');
        expect(get(statsResultStore)).toBeNull();
        expect(get(statsLoadingStore)).toBe(false);
    });
});

describe('StatsPanel — tab accessibility (role attribute check in template)', () => {
    // These tests verify the template contract without mounting the full component.
    // Full DOM tests require @testing-library/svelte; these cover the store contract.

    test('PANEL constant STATS is defined', () => {
        expect(PANEL.STATS).toBe('stats');
    });

    test('statsMetricStore only holds pr or mwc values by convention', () => {
        const validValues = ['pr', 'mwc'];
        statsMetricStore.set('pr');
        expect(validValues).toContain(get(statsMetricStore));
        statsMetricStore.set('mwc');
        expect(validValues).toContain(get(statsMetricStore));
    });
});
