// @ts-nocheck -- vi.mock factories and partial photographs, checked by the assertions
/**
 * The split screen: refused below its minimum width, closed when the window shrinks, the panel
 * beside the boards only in a wide window. Each side holds one library and keeps it whatever is
 * activated; a pane is brought live through activateGroup, a refusal shakes it; two views of one
 * library trade places; the split is remembered, by file, for the next session.
 */
import { test, expect, vi, beforeEach } from 'vitest';
import { get, writable } from 'svelte/store';

const activateGroup = vi.fn(async () => true);
vi.mock('../stores/libraryGroupStore.js', () => ({
    groupIdsStore: writable([1, 2]),
    activeGroupId: writable(1),
    parkedGroupsStore: writable(new Map()),
    activateGroup: (...args) => activateGroup(...args)
}));
const findIndex = vi.fn(async () => 4);
vi.mock('../stores/positionStore.js', () => ({
    positionStore: writable({ id: 7 }),
    matchContextStore: writable({ isMatchMode: false }),
    positionsStore: { idAt: () => 7, findIndex: (...args) => findIndex(...args) }
}));

const { statusBarTextStore, currentPositionIndexStore } = await import('../stores/uiStore.js');
const { parkedGroupsStore, activeGroupId, groupIdsStore } = await import('../stores/libraryGroupStore.js');
const { splitStore, splitShakeStore, splitFrozenStore } = await import('../stores/splitStore.js');
const svc = await import('../services/splitService.js');

const off = { on: false, left: null, right: null, sameSide: 'left' };

beforeEach(() => {
    splitStore.set(off);
    statusBarTextStore.set('');
    activeGroupId.set(1);
    groupIdsStore.set([1, 2]);
    activateGroup.mockReset();
    activateGroup.mockImplementation(async () => true);
    localStorage.clear();
});

test('below the minimum width the split is refused; opened, the active library takes the left', () => {
    expect(svc.toggleSplit(svc.SPLIT_MIN_WIDTH - 1)).toBe(false);
    expect(get(splitStore).on).toBe(false);
    expect(get(statusBarTextStore)).toMatchObject({ i18nKey: 'split.tooNarrow' });
    activeGroupId.set(2);
    expect(svc.toggleSplit(2560)).toBe(true);
    expect(get(splitStore)).toMatchObject({ on: true, left: 2, right: 1 });
    expect(svc.toggleSplit(2560)).toBe(false);
});

test('a window shrinking under the minimum closes the split; the panel goes under below 1600 px', () => {
    svc.toggleSplit(1366);
    svc.closeSplitIfNarrow(1366);
    expect(get(splitStore).on).toBe(true);
    svc.closeSplitIfNarrow(1000);
    expect(get(splitStore).on).toBe(false);
    expect(get(statusBarTextStore)).toMatchObject({ i18nKey: 'split.closedNarrow' });
    expect(svc.splitPanelBeside(1599)).toBe(false);
    expect(svc.splitPanelBeside(1600)).toBe(true);
});

test('each side keeps its library: activating the other pane moves the focus, not the panes', () => {
    const state = { on: true, left: 1, right: 2, sameSide: 'left' };
    expect(svc.liveSide(state, svc.paneGroups(state, [1, 2], 1), 1)).toBe('left');
    expect(svc.followActivation(state, 1, 2, [1, 2])).toBe(state);
    expect(svc.liveSide(state, svc.paneGroups(state, [1, 2], 2), 2)).toBe('right');
    // A group neither pane holds takes the live side; the frozen one stays.
    expect(svc.followActivation(state, 2, 3, [1, 2, 3])).toEqual({ ...state, right: 3 });
    // One library on both sides, a library just opened takes the live side.
    const same = { on: true, left: 1, right: 1, sameSide: 'right' };
    expect(svc.followActivation(same, 1, 2, [1, 2])).toEqual({ ...same, right: 2 });
    // A side whose group closed takes the one left.
    expect(svc.paneGroups(state, [1], 1)).toEqual({ left: 1, right: 1 });
});

test('a pane comes live through activateGroup; a refusal shakes it and leaves the focus', async () => {
    splitStore.set({ on: true, left: 1, right: 2, sameSide: 'left' });
    expect(await svc.activatePane('left')).toBe(true);
    expect(activateGroup).not.toHaveBeenCalled();
    activateGroup.mockImplementation(async () => false);
    const before = get(splitShakeStore).n;
    expect(await svc.activatePane('right')).toBe(false);
    expect(activateGroup).toHaveBeenCalledWith(2);
    expect(get(splitShakeStore)).toEqual({ side: 'right', n: before + 1 });
    expect(get(splitStore)).toMatchObject({ left: 1, right: 2 });
});

test('two views of one library trade places: the frozen position comes live, the live one freezes', async () => {
    groupIdsStore.set([1]);
    splitStore.set({ on: true, left: 1, right: 1, sameSide: 'left' });
    splitFrozenStore.set({ positionId: 12, refreeze: null });
    currentPositionIndexStore.set(0);
    expect(await svc.activatePane('right')).toBe(true);
    expect(findIndex).toHaveBeenCalledWith(12);
    expect(get(currentPositionIndexStore)).toBe(4);
    expect(get(splitStore).sameSide).toBe('right');
    expect(get(splitFrozenStore).refreeze).toMatchObject({ positionId: 7 });
    expect(activateGroup).not.toHaveBeenCalled();
    findIndex.mockImplementationOnce(async () => -1);
    expect(await svc.activatePane('left')).toBe(false);
    expect(get(statusBarTextStore)).toMatchObject({ i18nKey: 'split.sameLibraryNotInList' });
});

test('the frozen pane starts where its group was left: a match on its move, else the position', () => {
    const photo = (view) => ({ path: '/b.db', views: { activeViewId: 4, views: [{ id: 3 }, { id: 4, ...view }] } });
    expect(svc.viewerStart(photo({ positionId: 12, matchContext: { isMatchMode: false } }))).toEqual({ source: { kind: 'library', id: 0 }, positionId: 12, index: 0 });
    expect(svc.viewerStart(photo({ positionId: 12, matchContext: { isMatchMode: true, matchID: 5, currentIndex: 9 } }))).toEqual({ source: { kind: 'match', id: 5 }, positionId: null, index: 9 });
    parkedGroupsStore.set(new Map([[2, photo({ positionId: 3 })]]));
    expect(svc.viewerStartOf(2).positionId).toBe(3);
    expect(svc.viewerStartOf(1).positionId).toBe(7);
});

test('the split is remembered by file and put back on its sides, the active library kept', async () => {
    const paths = { 1: '/a.db', 2: '/b.db' };
    svc.rememberSplit({ on: true, left: 2, right: 1, sameSide: 'left' }, (g) => paths[g] ?? '');
    expect(svc.rememberedSplit()).toEqual({ on: true, left: '/b.db', right: '/a.db' });
    groupIdsStore.set([1]);
    const openOther = vi.fn(async () => {
        groupIdsStore.set([1, 5]);
        return 5;
    });
    expect(await svc.restoreSplit('/a.db', openOther, 2560)).toBe(true);
    expect(openOther).toHaveBeenCalledWith('/b.db');
    expect(get(splitStore)).toMatchObject({ on: true, left: 5, right: 1 });
    expect(svc.splitRemembering()).toBe(true);
    // Another library reopened: nothing is put back.
    splitStore.set(off);
    expect(await svc.restoreSplit('/c.db', openOther, 2560)).toBe(false);
    expect(await svc.restoreSplit('/a.db', openOther, 1000)).toBe(false);
});
