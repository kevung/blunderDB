/**
 * positionService.evalBoardMemory.test.js
 *
 * Reported: leaving the Eval panel and coming back loses the position and its
 * evaluation. enterEPCMode always opened on the default bearoff, so whatever
 * the user had built on the scratch board was thrown away on the way out.
 *
 * The panel now reopens on the board it was last left on. Three rules go with
 * that: a position sent from the library still wins, the remembered board is a
 * board and not a library record (id 0, no analysis riding along — a stale
 * analysis describing another position is the scratch-board bug), and the very
 * first opening still gets the default bearoff.
 */

import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    LoadAllPositions: vi.fn(() => Promise.resolve([])),
    DeletePosition: vi.fn(),
    DeleteAnalysis: vi.fn(),
    UpdatePosition: vi.fn(),
    SaveAnalysis: vi.fn(),
    LoadAnalysis: vi.fn(() => Promise.resolve(null)),
    LoadPositionIDsByFilters: vi.fn(() => Promise.resolve([])),
    ComputeEPCFromPosition: vi.fn(() => Promise.resolve({})),
    SaveLastVisitedPosition: vi.fn(),
    GetLastVisitedMatch: vi.fn(() => Promise.resolve(null)),
    GetMatchMovePositions: vi.fn(() => Promise.resolve([])),
    ListPositionIDs: vi.fn(() => Promise.resolve([])),
    SaveEditPosition: vi.fn(),
    SaveFilter: vi.fn(),
    LoadComment: vi.fn(() => Promise.resolve(''))
}));

vi.mock('../services/databaseService.js', () => ({
    setStatusBarMessage: vi.fn(),
    warningMessageStore: { subscribe: vi.fn(), set: vi.fn(), update: vi.fn() }
}));

import { statusBarModeStore, statusBarTextStore, currentPositionIndexStore } from '../stores/uiStore.js';
import { positionStore, positionsStore } from '../stores/positionStore.js';
import { epcDataStore } from '../stores/epcStore.js';
import { enterEPCMode, exitEPCMode, sendPositionToEval } from '../services/positionService.js';

function emptyPoints() {
    return Array(26)
        .fill(null)
        .map(() => ({ checkers: 0, color: -1 }));
}

/** A board the user could have built inside the panel: two checkers on the 4-point. */
function scratchBoard() {
    const points = emptyPoints();
    points[4] = { checkers: 2, color: 0 };
    return {
        id: 0,
        board: { points, bearoff: [0, 13] },
        cube: { owner: -1, value: 0 },
        dice: [0, 0],
        score: [0, 0],
        player_on_roll: 0,
        decision_type: 0,
        has_jacoby: 0,
        has_beaver: 0
    };
}

function libraryPosition(id = 42) {
    const points = emptyPoints();
    points[6] = { checkers: 5, color: 0 };
    return { ...scratchBoard(), id, board: { points, bearoff: [0, 10] } };
}

/** How many checkers the board carries, as a cheap identity for these tests. */
function checkerSignature(position) {
    return position.board.points
        .map((p, i) => (p.checkers ? `${i}:${p.checkers}` : ''))
        .filter(Boolean)
        .join(',');
}

beforeEach(() => {
    statusBarModeStore.set('NORMAL');
    statusBarTextStore.set('');
    currentPositionIndexStore.set(0);
    positionStore.set(null);
    positionsStore.set([]);
    epcDataStore.set({ bottomEPC: null, topEPC: null, race: null, error: null });
    vi.clearAllMocks();
});

describe('the Eval panel reopens on the board it was left on', () => {
    test('the first opening gets the default bearoff', async () => {
        await enterEPCMode();
        expect(get(statusBarModeStore)).toBe('EPC');
        // The default bearoff: 15 checkers spread over the six home points.
        expect(checkerSignature(get(positionStore))).toBe('1:2,2:2,3:2,4:3,5:3,6:3');
    });

    test('a board built in the panel is still there on the way back', async () => {
        await enterEPCMode();

        // The user rearranges the board.
        const built = scratchBoard();
        positionStore.set(built);
        const signature = checkerSignature(built);

        await exitEPCMode();
        expect(get(statusBarModeStore)).toBe('NORMAL');

        await enterEPCMode();
        expect(checkerSignature(get(positionStore)), 'the panel found its board again').toBe(signature);
    });

    test('the remembered board is a board, never a library record', async () => {
        await enterEPCMode();
        positionStore.set({ ...scratchBoard(), id: 77 });
        await exitEPCMode();
        await enterEPCMode();
        expect(get(positionStore).id, 'no library id rides along').toBe(0);
    });

    test('a position sent from the library wins over the remembered board', async () => {
        await enterEPCMode();
        positionStore.set(scratchBoard());
        await exitEPCMode();

        const fromLibrary = libraryPosition();
        await sendPositionToEval(fromLibrary);
        // sendPositionToEval hands the position off through the tab effect;
        // enterEPCMode is what consumes it.
        if (get(statusBarModeStore) !== 'EPC') await enterEPCMode();

        expect(checkerSignature(get(positionStore))).toBe(checkerSignature(fromLibrary));
    });
});
