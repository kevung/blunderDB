/**
 * MaintenanceSettings.svelte: the Maintenance tab of the settings. It mounts, shows the free
 * counts of MaintenanceStatus, computes a costly count on demand, opens what a count counts
 * (matches for stats.matches, positions otherwise), starts a pass in the background and
 * disables every action on a read-only library.
 */
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

const { calls, fwd } = vi.hoisted(() => {
    const STATUS = [
        { kind: 'analysis.missing', tier: 'seconds', unit: 'positions', listable: true, runnable: false, applicable: true },
        { kind: 'stats.matches', tier: 'free', count: 3, unit: 'matches', listable: true, runnable: true, applicable: true },
        { kind: 'analysis.columns', tier: 'none', listable: false, runnable: true, applicable: true, last: { kind: 'analysis.columns', startedAt: 1, finishedAt: 2, changed: 5, outcome: 'done' } },
        { kind: 'trash', tier: 'free', count: 0, unit: 'entries', listable: false, runnable: false, applicable: true },
        { kind: 'vacuum', tier: 'free', unit: 'bytes', listable: false, runnable: false, applicable: false }
    ];

    const calls = {
        MaintenanceStatus: vi.fn(() => Promise.resolve(STATUS)),
        CountMaintenance: vi.fn(() => Promise.resolve(42)),
        MaintenanceIDs: vi.fn((/** @type {string} */ kind) => Promise.resolve(kind === 'stats.matches' ? [7, 8, 9] : [1, 2])),
        StaleGammonNetIDs: vi.fn(() => Promise.resolve([4])),
        IsReadOnly: vi.fn(() => Promise.resolve(false)),
        Vacuum: vi.fn(() => Promise.resolve({})),
        StartMaintenance: vi.fn(() => Promise.resolve()),
        StartGammonNetBatch: vi.fn(() => Promise.resolve()),
        StartGammonNetStaleBatch: vi.fn(() => Promise.resolve()),
        openPositionIDs: vi.fn(),
        openMatchListInPanel: vi.fn(() => Promise.resolve())
    };
    /** @param {keyof typeof calls} name */
    const fwd =
        (name) =>
        (/** @type {any[]} */ ...a) =>
            /** @type {any} */ (calls[name])(...a);
    return { calls, fwd };
});
vi.mock('../../wailsjs/go/database/Database.js', () => ({
    MaintenanceStatus: fwd('MaintenanceStatus'),
    CountMaintenance: fwd('CountMaintenance'),
    MaintenanceIDs: fwd('MaintenanceIDs'),
    StaleGammonNetIDs: fwd('StaleGammonNetIDs'),
    IsReadOnly: fwd('IsReadOnly'),
    Vacuum: fwd('Vacuum')
}));
vi.mock('../../wailsjs/go/gui/App.js', () => ({
    StartMaintenance: fwd('StartMaintenance'),
    StartGammonNetBatch: fwd('StartGammonNetBatch'),
    StartGammonNetStaleBatch: fwd('StartGammonNetStaleBatch')
}));
vi.mock('../services/positionLoader.js', () => ({
    openPositionIDs: fwd('openPositionIDs'),
    openMatchListInPanel: fwd('openMatchListInPanel')
}));

import MaintenanceSettings from '../components/MaintenanceSettings.svelte';
import { databasePathStore } from '../stores/databaseStore.js';
import { maintenanceStore } from '../stores/maintenanceStore.js';
import { statusBarTextStore } from '../stores/uiStore';
import { get } from 'svelte/store';

async function settle() {
    for (let i = 0; i < 6; i++) await tick();
    await new Promise((r) => setTimeout(r, 0));
    for (let i = 0; i < 4; i++) await tick();
}

/** @param {HTMLElement} c @param {string} id */
const byId = (c, id) => /** @type {HTMLElement} */ (c.querySelector(`[data-testid="${id}"]`));

beforeEach(() => {
    Object.values(calls).forEach((f) => f.mockClear());
    calls.IsReadOnly.mockImplementation(() => Promise.resolve(false));
    databasePathStore.set('/tmp/lib.db');
    maintenanceStore.set(null);
});
afterEach(() => cleanup());

describe('MaintenanceSettings', () => {
    test('mounts with one line per applicable operation, plus the stale verdicts', async () => {
        const { container } = render(MaintenanceSettings, { props: { ply: 2 } });
        await settle();
        expect(calls.MaintenanceStatus).toHaveBeenCalled();
        expect(byId(container, 'maintenance-count-stats.matches').textContent).toContain('3');
        expect(byId(container, 'maintenance-count-analysis.stale')).toBeTruthy();
        expect(byId(container, 'maintenance-count-vacuum')).toBeNull();
        // Counting the stale verdicts is a scan: never on mount.
        expect(calls.StaleGammonNetIDs).not.toHaveBeenCalled();
    });

    test('a costly count is computed on demand, then opens the positions it counts', async () => {
        const onLeave = vi.fn();
        const { container } = render(MaintenanceSettings, { props: { onLeave } });
        await settle();
        await fireEvent.click(byId(container, 'maintenance-countbtn-analysis.missing'));
        await settle();
        expect(calls.CountMaintenance).toHaveBeenCalledWith('analysis.missing');
        await fireEvent.click(byId(container, 'maintenance-open-analysis.missing'));
        await settle();
        expect(calls.MaintenanceIDs).toHaveBeenCalledWith('analysis.missing');
        expect(calls.openPositionIDs).toHaveBeenCalledWith([1, 2]);
        expect(onLeave).toHaveBeenCalled();
    });

    test('the stats.matches count opens matches, not positions', async () => {
        const { container } = render(MaintenanceSettings);
        await settle();
        await fireEvent.click(byId(container, 'maintenance-open-stats.matches'));
        await settle();
        expect(calls.openMatchListInPanel).toHaveBeenCalledWith([7, 8, 9]);
        expect(calls.openPositionIDs).not.toHaveBeenCalled();
    });

    test('the stale count keeps its ids and runs at the configured ply', async () => {
        const { container } = render(MaintenanceSettings, { props: { ply: 3, pruneK: 8, candidates: 10 } });
        await settle();
        await fireEvent.click(byId(container, 'maintenance-countbtn-analysis.stale'));
        await settle();
        expect(calls.StaleGammonNetIDs).toHaveBeenCalledWith(3);
        await fireEvent.click(byId(container, 'maintenance-open-analysis.stale'));
        await settle();
        expect(calls.openPositionIDs).toHaveBeenCalledWith([4]);
        await fireEvent.click(byId(container, 'maintenance-run-analysis.stale'));
        expect(calls.StartGammonNetStaleBatch).toHaveBeenCalledWith(3, 8, 10);
    });

    test('a run starts in the background and holds the other actions', async () => {
        const { container } = render(MaintenanceSettings);
        await settle();
        await fireEvent.click(byId(container, 'maintenance-run-analysis.columns'));
        await settle();
        expect(calls.StartMaintenance).toHaveBeenCalledWith('analysis.columns');
        expect(/** @type {HTMLButtonElement} */ (byId(container, 'maintenance-run-stats.matches')).disabled).toBe(true);
    });

    test('a list that cannot be read says so and stays on the tab', async () => {
        calls.MaintenanceIDs.mockImplementationOnce(() => Promise.reject(new Error('boom')));
        const onLeave = vi.fn();
        const { container } = render(MaintenanceSettings, { props: { onLeave } });
        await settle();
        await fireEvent.click(byId(container, 'maintenance-open-stats.matches'));
        await settle();
        expect(onLeave).not.toHaveBeenCalled();
        expect(JSON.stringify(get(statusBarTextStore))).toContain('boom');
    });

    test('a read-only library disables every action', async () => {
        calls.IsReadOnly.mockImplementation(() => Promise.resolve(true));
        const { container } = render(MaintenanceSettings);
        await settle();
        for (const kind of ['analysis.missing', 'stats.matches', 'analysis.columns']) {
            expect(/** @type {HTMLButtonElement} */ (byId(container, `maintenance-run-${kind}`)).disabled).toBe(true);
        }
    });
});
