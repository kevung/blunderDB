/**
 * A write on the position on the board names it by id: once another library came on screen
 * between the read of that id (the menu built, the confirmation asked) and the write, the write is
 * not sent, where it would name another row, or none.
 */

import { describe, test, expect, vi, beforeEach } from 'vitest';

const bindings = vi.hoisted(() => {
    const pos = (/** @type {number} */ id) => ({ id, board: { points: [], bearoff: [0, 0] }, cube: { owner: -1, value: 0 }, dice: [1, 2], score: [3, 3], player_on_roll: 0, decision_type: 0 });
    return {
        ListPositionIDs: vi.fn(() => Promise.resolve([])),
        CountPositions: vi.fn(() => Promise.resolve(0)),
        IndexOfPosition: vi.fn(() => Promise.resolve(-1)),
        LoadPositionsByIDs: vi.fn((/** @type {number[]} */ ids) => Promise.resolve(ids.map(pos))),
        TrashPosition: vi.fn(() => Promise.resolve()),
        LoadAnalysis: vi.fn(() => Promise.resolve(null)),
        ComputeEPCFromPosition: vi.fn(() => Promise.resolve({})),
        SaveLastVisitedPosition: vi.fn(() => Promise.resolve()),
        GetLastVisitedMatch: vi.fn(() => Promise.resolve(null)),
        GetMatchMovePositions: vi.fn(() => Promise.resolve([])),
        LoadComment: vi.fn(() => Promise.resolve('')),
        SetPositionStudied: vi.fn(() => Promise.resolve()),
        IsPositionStudied: vi.fn(() => Promise.resolve(false)),
        SyncAnkiDeckWithPositions: vi.fn(() => Promise.resolve()),
        GetAnkiDeckStats: vi.fn(() => Promise.resolve(null))
    };
});

/** @type {{ resolve: (v: boolean) => void }} */
const confirmation = vi.hoisted(() => ({ resolve: () => {} }));

vi.mock('../../wailsjs/go/database/Database.js', () => bindings);
vi.mock('../services/databaseService.js', () => ({
    setStatusBarMessage: vi.fn(),
    warningMessageStore: { subscribe: vi.fn(), set: vi.fn(), update: vi.fn() }
}));
vi.mock('../services/sessionService.js', () => ({ saveSessionState: vi.fn() }));
vi.mock('../services/confirmService.js', () => ({
    confirmAction: vi.fn(() => new Promise((resolve) => (confirmation.resolve = resolve))),
    chooseAction: vi.fn(),
    confirmModalStore: { subscribe: (/** @type {(v: null) => void} */ run) => (run(null), () => {}) }
}));

import { currentPositionIndexStore, statusBarModeStore } from '../stores/uiStore.js';
import { positionsStore } from '../stores/positionStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { libraryReplaced } from '../stores/groupActivityStore.js';
import { useLibrary } from '../__mocks__/wails.js';
import { deletePosition, loadAllPositions } from '../services/positionService.js';
import { positionContext } from '../services/positionActions.js';

/** Another library comes on screen: the switch's own steps, without its machinery. */
function anotherLibrary() {
    libraryReplaced();
    databasePathStore.set('/libs/b.db');
}

beforeEach(async () => {
    vi.clearAllMocks();
    databasePathStore.set('/libs/a.db');
    statusBarModeStore.set('NORMAL');
    positionsStore.setIds([], { reset: true });
    currentPositionIndexStore.set(-1);
});

describe('deleting the position on the board', () => {
    test('is trashed when the library is still the one it was confirmed in', async () => {
        useLibrary(bindings, [1, 2, 3]);
        await loadAllPositions();
        await positionsStore.getPosition(2);
        const deleting = deletePosition();
        await new Promise((r) => setTimeout(r, 0));
        confirmation.resolve(true);
        await deleting;
        expect(bindings.TrashPosition).toHaveBeenCalledWith(3);
    });

    test('is not sent when another library came on screen during the confirmation', async () => {
        useLibrary(bindings, [1, 2, 3]);
        await loadAllPositions();
        await positionsStore.getPosition(2);
        const deleting = deletePosition();
        await new Promise((r) => setTimeout(r, 0));
        anotherLibrary();
        confirmation.resolve(true);
        await deleting;
        expect(bindings.TrashPosition).not.toHaveBeenCalled();
    });
});

describe('the menu of the position on the board', () => {
    test('marks it studied in the library the menu was built in', async () => {
        await positionContext({ positionId: 7 }).do.markStudied();
        expect(bindings.SetPositionStudied).toHaveBeenCalledWith(7, true);
    });

    test('does not mark it studied once another library came on screen', async () => {
        const menu = positionContext({ positionId: 7 });
        anotherLibrary();
        await menu.do.markStudied();
        expect(bindings.SetPositionStudied).not.toHaveBeenCalled();
    });

    test('does not add it to a deck once another library came on screen', async () => {
        const menu = positionContext({ positionId: 7 });
        anotherLibrary();
        await menu.do.addToAnkiDeck({ id: 4, name: 'deck' });
        expect(bindings.SyncAnkiDeckWithPositions).not.toHaveBeenCalled();
    });
});
