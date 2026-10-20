/**
 * Library groups: several libraries open, one group of tabs each. A switch must be instant: the
 * group comes back from its photograph (views, list, position cache, search, counts) with no
 * request but the backend's in-memory exchange; what may be stale is read after it is shown.
 */

import { describe, test, expect, beforeEach, vi } from 'vitest';
import { get } from 'svelte/store';

const calls = vi.hoisted(() => /** @type {string[]} */ ([]));

// Plain doubles: the reflective factories of __mocks__/wails.js import the real binding modules,
// which, mocked here all three, would wait on each other.
vi.mock('../../wailsjs/go/gui/App.js', () => {
    let next = 2;
    return {
        OpenLibraryGroup: vi.fn(async (/** @type {string} */ path) => {
            calls.push('OpenLibraryGroup');
            return { id: next++, path, active: true, readOnly: false };
        }),
        ActivateLibraryGroup: vi.fn(async () => {
            calls.push('ActivateLibraryGroup');
        }),
        CloseLibraryGroup: vi.fn(async () => {
            calls.push('CloseLibraryGroup');
        })
    };
});
vi.mock('../../wailsjs/go/database/Database.js', () => ({
    LoadPositionsByIDs: vi.fn(async (/** @type {number[]} */ ids) => {
        calls.push('LoadPositionsByIDs');
        return ids.map((id) => ({ id, board: {} }));
    }),
    GetLibraryCounts: vi.fn(async () => {
        calls.push('GetLibraryCounts');
        return null;
    })
}));
vi.mock('../../wailsjs/runtime/runtime.js', () => ({ WindowSetTitle: vi.fn(), EventsOn: vi.fn(() => () => {}) }));

import { viewStore } from '../stores/viewStore.js';
import { positionsStore, positionStore } from '../stores/positionStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { searchParamsStore } from '../stores/searchParamsStore.js';
import { currentPositionIndexStore } from '../stores/uiStore.js';
import {
    groupIdsStore,
    activeGroupId,
    parkedGroupsStore,
    tabGroupsStore,
    lastSwitchMsStore,
    openInNewGroup,
    activateGroup,
    closeGroup,
    cycleTab,
    cycleGroup,
    groupName
} from '../stores/libraryGroupStore.js';

/** @param {number[]} ids */
function showLibrary(ids, index = 0) {
    positionsStore.setIds(ids, { reset: true });
    for (const id of ids) positionsStore.upsert({ id, board: {} });
    currentPositionIndexStore.set(index);
    positionStore.set({ id: ids[index], board: {} });
}

beforeEach(async () => {
    // Back to one group showing library A.
    for (const id of get(groupIdsStore)) if (id !== get(activeGroupId)) await closeGroup(id);
    calls.length = 0;
    databasePathStore.set('/libs/a.db');
    searchParamsStore.set({ query: 'cube' });
    showLibrary([11, 12, 13], 2);
});

describe('library groups', () => {
    test('a library opened in a new group leaves the first one waiting, photographed', async () => {
        const before = get(activeGroupId);
        await openInNewGroup('/libs/b.db');
        expect(get(activeGroupId)).not.toBe(before);
        expect(get(parkedGroupsStore).get(before)?.path).toBe('/libs/a.db');
        // The new group starts empty: no id of A leaks into B's list or cache.
        expect(get(positionsStore).length).toBe(0);
        expect(positionsStore.cacheSize).toBe(0);
        expect(get(searchParamsStore)).toBeNull();
    });

    test('the round trip keeps list, position, search and tab without a blocking request', async () => {
        const groupA = get(activeGroupId);
        await openInNewGroup('/libs/b.db');
        const groupB = get(activeGroupId);
        databasePathStore.set('/libs/b.db');
        showLibrary([21, 22], 1);
        viewStore.addView();
        const viewsOfB = get(viewStore.views).length;

        calls.length = 0;
        expect(await activateGroup(groupA)).toBe(true);
        // Only the in-memory exchange went to the backend before the group was on screen.
        expect(calls).toEqual(['ActivateLibraryGroup']);
        expect(get(databasePathStore)).toBe('/libs/a.db');
        expect([0, 1, 2].map((i) => positionsStore.peek(i)?.id)).toEqual([11, 12, 13]);
        expect(get(currentPositionIndexStore)).toBe(2);
        expect(get(positionStore).id).toBe(13);
        expect(get(searchParamsStore)).toEqual({ query: 'cube' });
        expect(get(lastSwitchMsStore)).toBeLessThan(50);

        calls.length = 0;
        expect(await activateGroup(groupB)).toBe(true);
        expect(calls).toEqual(['ActivateLibraryGroup']);
        expect(get(viewStore.views)).toHaveLength(viewsOfB);
        expect(positionsStore.peek(0)?.id).toBe(21);
    });

    test('clicking a tab of another group brings its library and that tab in one gesture', async () => {
        const groupA = get(activeGroupId);
        viewStore.addView();
        const views = get(viewStore.views);
        const firstOfA = views[0].id;
        await openInNewGroup('/libs/b.db');
        expect(await activateGroup(groupA, firstOfA)).toBe(true);
        expect(get(activeGroupId)).toBe(groupA);
        expect(get(viewStore.activeViewId)).toBe(firstOfA);
    });

    test('the tab keys walk across groups, and the library keys from group to group', async () => {
        const groupA = get(activeGroupId);
        await openInNewGroup('/libs/b.db');
        const groupB = get(activeGroupId);
        const bar = get(tabGroupsStore);
        expect(bar.map((g) => g.id)).toEqual([groupA, groupB]);
        expect(bar[0].name).toBe('a');

        // B's only tab is the last of the bar: the next one is A's first.
        await cycleTab(1);
        expect(get(activeGroupId)).toBe(groupA);
        await cycleGroup(1);
        expect(get(activeGroupId)).toBe(groupB);
        await cycleGroup(-1);
        expect(get(activeGroupId)).toBe(groupA);
    });

    test('closing the active group hands the screen to its neighbour', async () => {
        const groupA = get(activeGroupId);
        await openInNewGroup('/libs/b.db');
        const groupB = get(activeGroupId);
        expect(await closeGroup(groupB)).toBe(true);
        expect(get(activeGroupId)).toBe(groupA);
        expect(get(groupIdsStore)).toEqual([groupA]);
        expect(get(parkedGroupsStore).size).toBe(0);
        expect(get(databasePathStore)).toBe('/libs/a.db');
    });

    test('a group is named after its file', () => {
        expect(groupName('/home/me/Matches 2026.db')).toBe('Matches 2026');
        expect(groupName('C:\\libs\\club.db')).toBe('club');
    });
});

describe('the switch on a large library', () => {
    test('a list of 100 000 positions comes back in a few milliseconds', async () => {
        const groupA = get(activeGroupId);
        const ids = Array.from({ length: 100_000 }, (_, i) => i + 1);
        showLibrary(ids, 50_000);
        await openInNewGroup('/libs/b.db');
        const groupB = get(activeGroupId);
        const samples = [];
        for (let i = 0; i < 10; i++) {
            await activateGroup(i % 2 === 0 ? groupA : groupB);
            samples.push(/** @type {number} */ (get(lastSwitchMsStore)));
        }
        samples.sort((x, y) => x - y);
        const median = samples[5];
        // eslint-disable-next-line no-console -- the measure is the point of this test
        console.info(`library group switch, 100 000 positions: median ${median.toFixed(2)} ms`);
        expect(median).toBeLessThan(50);
    });
});
