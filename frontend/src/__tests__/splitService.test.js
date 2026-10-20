// @ts-nocheck -- vi.mock factories and partial photographs, checked by the assertions
/**
 * The split screen: refused below its minimum width (the swap offered instead), closed when the
 * window shrinks, the viewer on the first other group, the two panes trading roles and sides when
 * the viewer's group is activated, and the viewer starting where its group was left.
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
vi.mock('../stores/positionStore.js', () => ({
    positionStore: writable({ id: 7 }),
    matchContextStore: writable({ isMatchMode: false }),
    positionsStore: { idAt: () => 7 }
}));

const { statusBarTextStore } = await import('../stores/uiStore.js');
const { parkedGroupsStore, activeGroupId, groupIdsStore } = await import('../stores/libraryGroupStore.js');
const { splitStore } = await import('../stores/splitStore.js');
const svc = await import('../services/splitService.js');

beforeEach(() => {
    splitStore.set({ on: false, group: null, side: 'right' });
    statusBarTextStore.set('');
    activeGroupId.set(1);
    groupIdsStore.set([1, 2]);
    activateGroup.mockClear();
});

test('below the minimum width the split is refused and the swap offered', () => {
    expect(svc.toggleSplit(svc.SPLIT_MIN_WIDTH - 1)).toBe(false);
    expect(get(splitStore).on).toBe(false);
    expect(get(statusBarTextStore)).toMatchObject({ i18nKey: 'split.tooNarrow' });
    expect(svc.toggleSplit(2560)).toBe(true);
    expect(get(splitStore).on).toBe(true);
    expect(svc.toggleSplit(2560)).toBe(false);
});

test('a window shrinking under the minimum closes the split', () => {
    svc.toggleSplit(1366);
    svc.closeSplitIfNarrow(1366);
    expect(get(splitStore).on).toBe(true);
    svc.closeSplitIfNarrow(1000);
    expect(get(splitStore).on).toBe(false);
    expect(get(statusBarTextStore)).toMatchObject({ i18nKey: 'split.closedNarrow' });
});

test('the viewer shows the first other group, else the active one', () => {
    expect(svc.viewerGroup(null, [1, 2, 3], 2)).toBe(1);
    expect(svc.viewerGroup(3, [1, 2, 3], 2)).toBe(3);
    expect(svc.viewerGroup(9, [1], 1)).toBe(1);
});

test("activating the viewer's group trades the panes, each library keeping its side", () => {
    const state = { on: true, group: null, side: 'right' };
    const next = svc.followActivation(state, 1, 2, [1, 2]);
    expect(next).toEqual({ on: true, group: 1, side: 'left' });
    // A third group coming on screen leaves the viewer where it is.
    const third = { on: true, group: 2, side: 'right' };
    expect(svc.followActivation(third, 1, 3, [1, 2, 3])).toBe(third);
});

test('clicking the viewer activates its group, not the one on screen', async () => {
    splitStore.set({ on: true, group: null, side: 'right' });
    await svc.activateViewer();
    expect(activateGroup).toHaveBeenCalledWith(2);
    groupIdsStore.set([1]);
    activateGroup.mockClear();
    await svc.activateViewer();
    expect(activateGroup).not.toHaveBeenCalled();
});

test('the viewer starts where its group was left: a match on its move, else the position', () => {
    const photo = (view) => ({ path: '/b.db', views: { activeViewId: 4, views: [{ id: 3 }, { id: 4, ...view }] } });
    expect(svc.viewerStart(photo({ positionId: 12, matchContext: { isMatchMode: false } }))).toEqual({ source: { kind: 'library', id: 0 }, positionId: 12, index: 0 });
    expect(svc.viewerStart(photo({ positionId: 12, matchContext: { isMatchMode: true, matchID: 5, currentIndex: 9 } }))).toEqual({ source: { kind: 'match', id: 5 }, positionId: null, index: 9 });
    parkedGroupsStore.set(new Map([[2, photo({ positionId: 3 })]]));
    expect(svc.viewerStartOf(2).positionId).toBe(3);
    expect(svc.viewerStartOf(1).positionId).toBe(7);
});
