/**
 * positionService.pagination.test.js
 *
 * The library is loaded as ids only (ListPositionIDs), never as the full
 * position array; the positions the board shows are fetched by window
 * through LoadPositionsByIDs, one call per half-window while browsing.
 */

import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

const bindings = vi.hoisted(() => {
    const pos = (id) => ({ id, board: { points: [], bearoff: [0, 0] }, cube: { owner: -1, value: 0 }, dice: [1, 2], score: [3, 3], player_on_roll: 0, decision_type: 0 });
    return {
        ListPositionIDs: vi.fn(() => Promise.resolve([])),
        LoadPositionsByIDs: vi.fn((ids) => Promise.resolve(ids.map(pos))),
        TrashPosition: vi.fn(() => Promise.resolve()),
        DeleteAnalysis: vi.fn(),
        UpdatePosition: vi.fn(),
        SaveAnalysis: vi.fn(),
        LoadAnalysis: vi.fn(() => Promise.resolve(null)),
        LoadPositionIDsByFilters: vi.fn(() => Promise.resolve([])),
        ComputeEPCFromPosition: vi.fn(() => Promise.resolve({})),
        SaveLastVisitedPosition: vi.fn(() => Promise.resolve()),
        GetLastVisitedMatch: vi.fn(() => Promise.resolve(null)),
        GetMatchMovePositions: vi.fn(() => Promise.resolve([])),
        SaveEditPosition: vi.fn(),
        SaveExcludePosition: vi.fn(),
        SaveFilter: vi.fn(),
        LoadComment: vi.fn(() => Promise.resolve(''))
    };
});

vi.mock('../../wailsjs/go/database/Database.js', () => bindings);
vi.mock('../services/databaseService.js', () => ({
    setStatusBarMessage: vi.fn(),
    warningMessageStore: { subscribe: vi.fn(), set: vi.fn(), update: vi.fn() }
}));
vi.mock('../services/sessionService.js', () => ({ saveSessionState: vi.fn() }));
vi.mock('../services/confirmService.js', () => ({ confirmAction: vi.fn(() => Promise.resolve(true)) }));

import { statusBarModeStore, currentPositionIndexStore } from '../stores/uiStore.js';
import { positionsStore } from '../stores/positionStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { loadAllPositions, nextPosition, previousPosition, firstPosition, lastPosition, deletePosition } from '../services/positionService.js';

const range = (n) => Array.from({ length: n }, (_, i) => i + 1);
// Prefetches are fire-and-forget and reach the binding through a dynamic
// import: give the event loop a turn before counting calls.
const flush = () => new Promise((resolve) => setTimeout(resolve, 0));

beforeEach(() => {
    vi.clearAllMocks();
    databasePathStore.set('/tmp/lib.db');
    statusBarModeStore.set('NORMAL');
    positionsStore.setIds([], { reset: true });
    currentPositionIndexStore.set(-1);
});

describe('loadAllPositions', () => {
    test('loads the ids only and lands on the last one; no position travels', async () => {
        bindings.ListPositionIDs.mockResolvedValue(range(300));

        await loadAllPositions();

        expect(bindings.ListPositionIDs).toHaveBeenCalledTimes(1);
        expect(bindings.LoadPositionsByIDs).not.toHaveBeenCalled();
        expect(get(positionsStore)).toEqual({ ids: range(300), length: 300 });
        expect(get(currentPositionIndexStore)).toBe(299);
        expect(get(statusBarModeStore)).toBe('NORMAL');
    });

    test('an empty library leaves the index at -1', async () => {
        bindings.ListPositionIDs.mockResolvedValue([]);
        await loadAllPositions();
        expect(get(positionsStore).length).toBe(0);
        expect(get(currentPositionIndexStore)).toBe(-1);
    });

    test('a reload drops the window cache (positions may have been edited)', async () => {
        bindings.ListPositionIDs.mockResolvedValue(range(10));
        await loadAllPositions();
        await positionsStore.getPosition(9);
        expect(positionsStore.peek(9)).toBeTruthy();

        await loadAllPositions();
        expect(positionsStore.peek(9)).toBeUndefined();
    });
});

describe('browsing fetches windows, one call per half-window', () => {
    test('showing the last position fetches its window; walking back stays inside it', async () => {
        bindings.ListPositionIDs.mockResolvedValue(range(300));
        await loadAllPositions();

        // What the index effect does for the shown index.
        const shown = await positionsStore.getPosition(get(currentPositionIndexStore));
        expect(shown.id).toBe(300);
        expect(bindings.LoadPositionsByIDs).toHaveBeenCalledTimes(1);
        expect(bindings.LoadPositionsByIDs.mock.calls[0][0]).toEqual(range(300).slice(249));

        for (let step = 0; step < 10; step++) {
            await previousPosition();
            const p = await positionsStore.getPosition(get(currentPositionIndexStore));
            expect(p.id).toBe(299 - step);
        }
        await flush();
        expect(bindings.LoadPositionsByIDs, 'ten steps inside the window: no new call').toHaveBeenCalledTimes(1);

        for (let step = 10; step < 30; step++) {
            await previousPosition();
            await positionsStore.getPosition(get(currentPositionIndexStore));
        }
        await flush();
        expect(bindings.LoadPositionsByIDs, 'one prefetch for the next half-window').toHaveBeenCalledTimes(2);
        expect(bindings.LoadPositionsByIDs.mock.calls[1][0]).toEqual(range(300).slice(223, 249));
    });

    test('first / next / last move the index within the id list', async () => {
        bindings.ListPositionIDs.mockResolvedValue(range(5));
        await loadAllPositions();
        expect(get(currentPositionIndexStore)).toBe(4);
        await nextPosition();
        expect(get(currentPositionIndexStore)).toBe(4);
        await firstPosition();
        expect(get(currentPositionIndexStore)).toBe(0);
        await previousPosition();
        expect(get(currentPositionIndexStore)).toBe(0);
        await nextPosition();
        expect(get(currentPositionIndexStore)).toBe(1);
        await lastPosition();
        expect(get(currentPositionIndexStore)).toBe(4);
    });
});

describe('deletePosition', () => {
    test('deletes the id at the current index and reloads the ids', async () => {
        bindings.ListPositionIDs.mockResolvedValueOnce([10, 20, 30]).mockResolvedValueOnce([10, 30]);
        await loadAllPositions();
        currentPositionIndexStore.set(1);

        await deletePosition();

        expect(bindings.TrashPosition).toHaveBeenCalledWith(20);
        expect(get(positionsStore).ids).toEqual([10, 30]);
    });
});
