// @ts-nocheck -- vi.mock factories and partial positions, checked by the assertions
/**
 * The « Ailleurs » view and the in-place swap: one read per board through ElsewhereInGroups, the
 * swap showing the pinned board or the same board held elsewhere, a hold returning on release and
 * a tap on the next press, and nothing read when no other group is open.
 */
import { test, expect, vi, beforeEach } from 'vitest';
import { get, writable } from 'svelte/store';

const ElsewhereInGroups = vi.fn();
vi.mock('../../wailsjs/go/gui/App.js', () => ({ ElsewhereInGroups: (...args) => ElsewhereInGroups(...args) }));
const activateGroup = vi.fn(async () => true);
vi.mock('../stores/libraryGroupStore.js', () => ({
    groupIdsStore: writable([1, 2]),
    activeGroupId: writable(2),
    activateGroup: (...args) => activateGroup(...args)
}));
const findIndex = vi.fn(async () => 4);
vi.mock('../stores/positionStore.js', () => ({ positionStore: writable({ id: 9, board: { points: [] } }), positionsStore: { findIndex: (...a) => findIndex(...a) } }));
const loadAllPositions = vi.fn(async () => {});
vi.mock('../services/positionService.js', () => ({ loadAllPositions: (...a) => loadAllPositions(...a) }));

const { databasePathStore } = await import('../stores/databaseStore.js');
const { currentPositionIndexStore, statusBarTextStore } = await import('../stores/uiStore.js');
const { groupIdsStore } = await import('../stores/libraryGroupStore.js');
const { positionStore } = await import('../stores/positionStore.js');
const { elsewhereStore, elsewherePinStore, elsewherePeekStore, idleElsewhere } = await import('../stores/elsewhereStore.js');
const svc = await import('../services/elsewhereService.js');

const report = (group, positionId) => ({ group, path: `/libs/g${group}.db`, positionId, otherMet: false, comments: [], plays: [], matchesTotal: 0, playsTruncated: false, neighbours: [] });

beforeEach(() => {
    vi.clearAllMocks();
    databasePathStore.set('/libs/g2.db');
    groupIdsStore.set([1, 2]);
    positionStore.set({ id: 9, board: { points: [] } });
    elsewhereStore.set(idleElsewhere());
    elsewherePinStore.set(null);
    svc.endSwap();
    ElsewhereInGroups.mockResolvedValue({ zobrist: 7, judge: { settings: {}, metDigest: '' }, groups: [report(1, 31)] });
});

test('the board is read once, without the id that names a row of this library only', async () => {
    await svc.loadElsewhere();
    await svc.loadElsewhere(get(positionStore));
    expect(ElsewhereInGroups).toHaveBeenCalledTimes(1);
    expect(ElsewhereInGroups.mock.calls[0][1]).toBe(svc.ELSEWHERE_NEIGHBOURS);
    expect(svc.boardKey({ id: 3, a: 1 })).toBe(svc.boardKey({ id: 4, a: 1 }));
    expect(get(elsewhereStore).view.groups[0].positionId).toBe(31);
});

test('with one group open nothing is read and the swap says so', async () => {
    groupIdsStore.set([2]);
    expect(await svc.loadElsewhere()).toBeNull();
    expect(await svc.startSwap()).toBe(false);
    expect(ElsewhereInGroups).not.toHaveBeenCalled();
    expect(get(statusBarTextStore)).toEqual({ i18nKey: 'elsewhere.noOtherLibrary', i18nParams: null });
});

test('the swap shows the same board in the group that holds it, the pin when there is one', async () => {
    expect(await svc.startSwap()).toBe(true);
    const peek = get(elsewherePeekStore);
    expect(peek.group).toBe(1);
    expect(peek.pinned).toBe(false);
    expect(peek.position).toEqual(get(positionStore));
    svc.endSwap();

    const neighbour = { id: 40, board: { points: [1] } };
    svc.togglePin({ group: 1, path: '/libs/g1.db', positionId: 40, position: neighbour, distance: 2 });
    await svc.startSwap();
    expect(get(elsewherePeekStore)).toMatchObject({ group: 1, pinned: true, position: neighbour });
    svc.togglePin({ group: 1, path: '/libs/g1.db', positionId: 40, position: neighbour, distance: 2 });
    expect(get(elsewherePinStore)).toBeNull();
});

test('a board no other group holds shows nothing', async () => {
    ElsewhereInGroups.mockResolvedValue({ zobrist: 7, judge: {}, groups: [report(1, 0)] });
    expect(await svc.startSwap()).toBe(false);
    expect(get(elsewherePeekStore)).toBeNull();
    expect(get(statusBarTextStore)).toEqual({ i18nKey: 'elsewhere.notHeld', i18nParams: null });
});

test('a tap keeps the swap until the next press; a hold returns on release', async () => {
    const now = vi.spyOn(performance, 'now');
    now.mockReturnValue(1000);
    svc.swapKeyDown(/** @type {any} */ ({ repeat: false }));
    await vi.waitFor(() => expect(get(elsewherePeekStore)).not.toBeNull());
    now.mockReturnValue(1100);
    svc.swapKeyUp();
    expect(get(elsewherePeekStore)).not.toBeNull();
    svc.swapKeyDown(/** @type {any} */ ({ repeat: false }));
    expect(get(elsewherePeekStore)).toBeNull();

    now.mockReturnValue(2000);
    svc.swapKeyDown(/** @type {any} */ ({ repeat: false }));
    await vi.waitFor(() => expect(get(elsewherePeekStore)).not.toBeNull());
    svc.swapKeyDown(/** @type {any} */ ({ repeat: true }));
    expect(get(elsewherePeekStore)).not.toBeNull();
    now.mockReturnValue(2000 + svc.SWAP_HOLD_MS);
    svc.swapKeyUp();
    expect(get(elsewherePeekStore)).toBeNull();
    now.mockRestore();
});

test('a release before the read lands shows nothing', async () => {
    /** @type {(v: any) => void} */
    let land = () => {};
    ElsewhereInGroups.mockReturnValue(new Promise((resolve) => (land = resolve)));
    const shown = svc.startSwap();
    svc.endSwap();
    land({ zobrist: 7, judge: {}, groups: [report(1, 31)] });
    expect(await shown).toBe(false);
    expect(get(elsewherePeekStore)).toBeNull();
});

test('opening a result activates its group on that position', async () => {
    await svc.openElsewhere(1, 31);
    expect(activateGroup).toHaveBeenCalledWith(1);
    expect(findIndex).toHaveBeenCalledWith(31);
    expect(get(currentPositionIndexStore)).toBe(4);

    findIndex.mockResolvedValueOnce(-1);
    await svc.openElsewhere(1, 32);
    expect(loadAllPositions).toHaveBeenCalledWith({ focusId: 32 });
});

test('an analysis read elsewhere is shown by its best play', () => {
    expect(svc.analysisSummary(null).kind).toBe('none');
    expect(svc.analysisSummary({ analysisEngineVersion: 'GNU 1.07', checkerAnalysis: { moves: [{ move: '8/5 6/5' }] } })).toEqual({ kind: 'checker', best: '8/5 6/5', engine: 'GNU 1.07' });
    expect(svc.analysisSummary({ doublingCubeAnalysis: { bestCubeAction: 'No double' } }).kind).toBe('cube');
    expect(svc.formatCost(84)).toBe('0.084');
    expect(svc.formatCost(null)).toBe('');
});
