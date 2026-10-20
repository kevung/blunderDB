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
const titles = vi.hoisted(() => /** @type {string[]} */ ([]));
vi.mock('../../wailsjs/runtime/runtime.js', () => ({
    WindowSetTitle: vi.fn((/** @type {string} */ title) => titles.push(title)),
    EventsOn: vi.fn(() => () => {})
}));

import { viewStore } from '../stores/viewStore.js';
import { positionsStore, positionStore } from '../stores/positionStore.js';
import { databasePathStore, databaseReadOnlyStore, libraryChangedStore } from '../stores/databaseStore.js';
import { collectionsStore, selectedCollectionStore } from '../stores/collectionStore.js';
import { tournamentsStore } from '../stores/tournamentStore.js';
import { transcriptionStore, transcriptionInfoStore } from '../stores/transcriptionStore.js';
import { lessonStore, lessonStepIndexStore } from '../stores/lessonStore.js';
import { ankiReviewCardStore } from '../stores/ankiStore.js';
import { studyQueueStore, studyQueueActiveStore } from '../stores/studyQueueStore.js';
import { duelStore } from '../stores/duelStore.js';
import { trainingSessionStore } from '../stores/trainingTabStore.js';
import { statusBarTextStore } from '../stores/uiStore.js';
import { searchParamsStore, attachSearchForm } from '../stores/searchParamsStore.js';
import { matchListStore } from '../stores/matchListStore.js';
import { matchPanelRefreshTriggerStore } from '../stores/uiStore.js';
import { ActivateLibraryGroup } from '../../wailsjs/go/gui/App.js';
import { currentPositionIndexStore } from '../stores/uiStore.js';
import {
    groupIdsStore,
    activeGroupId,
    parkedGroupsStore,
    tabGroupsStore,
    lastSwitchMsStore,
    openInNewGroup,
    activateGroup,
    LibraryGroupRefused,
    closeGroup,
    cycleTab,
    cycleGroup,
    groupName,
    noteGroupJob,
    groupActivityStore,
    importForegroundStore
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
    titles.length = 0;
    duelStore.set(null);
    trainingSessionStore.set(null);
    transcriptionStore.set(null);
    collectionsStore.set([]);
    databaseReadOnlyStore.set(false);
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

describe('what a group holds of its library', () => {
    test('lists read from A are emptied for B and read again, on a counter that moves', async () => {
        const groupA = get(activeGroupId);
        collectionsStore.set([{ id: 1, name: 'A collection' }]);
        selectedCollectionStore.set({ id: 1, name: 'A collection' });
        tournamentsStore.set([{ id: 7, name: 'A tournament' }]);
        const before = get(libraryChangedStore);

        await openInNewGroup('/libs/b.db');
        expect(get(collectionsStore)).toEqual([]);
        expect(get(selectedCollectionStore)).toBeNull();
        expect(get(tournamentsStore)).toEqual([]);
        expect(get(libraryChangedStore)).toBe(before + 1);

        collectionsStore.set([{ id: 2, name: 'B collection' }]);
        expect(await activateGroup(groupA)).toBe(true);
        // A's lists are read again from A, never shown from B's.
        expect(get(collectionsStore)).toEqual([]);
        expect(get(libraryChangedStore)).toBe(before + 2);
    });

    test('a draft, a Lesson, a review and a study queue survive the round trip, each in its group', async () => {
        const groupA = get(activeGroupId);
        const draftA = { id: 5, entries: ['8/5 6/5'] };
        transcriptionStore.set(draftA);
        transcriptionInfoStore.set({ name: 'draft A' });
        lessonStore.set({ id: 3 });
        lessonStepIndexStore.set(2);
        ankiReviewCardStore.set({ id: 9 });
        studyQueueStore.set([41, 42]);
        studyQueueActiveStore.set(true);

        await openInNewGroup('/libs/b.db');
        const groupB = get(activeGroupId);
        // B starts from nothing: no draft of A to type into.
        expect(get(transcriptionStore)).toBeNull();
        expect(get(lessonStore)).toBeNull();
        expect(get(ankiReviewCardStore)).toBeNull();
        expect(get(studyQueueActiveStore)).toBe(false);
        const draftB = { id: 5, entries: [] };
        transcriptionStore.set(draftB);

        expect(await activateGroup(groupA)).toBe(true);
        expect(get(transcriptionStore)).toBe(draftA);
        expect(get(transcriptionInfoStore)).toEqual({ name: 'draft A' });
        expect(get(lessonStepIndexStore)).toBe(2);
        expect(get(ankiReviewCardStore)).toEqual({ id: 9 });
        expect(get(studyQueueStore)).toEqual([41, 42]);

        expect(await activateGroup(groupB)).toBe(true);
        expect(get(transcriptionStore)).toBe(draftB);
        expect(get(lessonStore)).toBeNull();
    });

    test('a running Duel refuses the switch, and says so', async () => {
        const groupA = get(activeGroupId);
        await openInNewGroup('/libs/b.db');
        duelStore.set({ state: { ended: false } });
        calls.length = 0;
        expect(await activateGroup(groupA)).toBe(false);
        expect(calls).toEqual([]);
        expect(get(statusBarTextStore)).toEqual({ i18nKey: 'libraryGroups.duel', i18nParams: null });
        await expect(openInNewGroup('/libs/c.db')).rejects.toBeInstanceOf(LibraryGroupRefused);
        duelStore.set(null);
    });

    test('a training session refuses the switch, and says so', async () => {
        const groupA = get(activeGroupId);
        await openInNewGroup('/libs/b.db');
        trainingSessionStore.set({ kind: 'decision' });
        expect(await activateGroup(groupA)).toBe(false);
        expect(get(statusBarTextStore)).toEqual({ i18nKey: 'libraryGroups.training', i18nParams: null });
        trainingSessionStore.set(null);
    });

    test('the title keeps the read-only mark of the group brought back', async () => {
        const groupA = get(activeGroupId);
        databaseReadOnlyStore.set(true);
        await openInNewGroup('/libs/b.db');
        expect(get(databaseReadOnlyStore)).toBe(false);
        titles.length = 0;
        expect(await activateGroup(groupA)).toBe(true);
        expect(get(databaseReadOnlyStore)).toBe(true);
        expect(titles.at(-1)).toMatch(/^blunderDB - a\.db .+/);
    });
});

describe('the Search criteria and the Matches list of a group', () => {
    test('criteria typed and never run stay in their library, and B starts blank', async () => {
        // The mounted panel: its fields live outside the store until a search runs.
        let fields = { matchIDsSelected: [7, 8], playerName: 'Ann' };
        const detach = attachSearchForm({
            save: () => searchParamsStore.set({ ...fields }),
            restore: () => {
                fields = get(searchParamsStore) ?? { matchIDsSelected: [], playerName: '' };
            }
        });
        try {
            const groupA = get(activeGroupId);
            await openInNewGroup('/libs/b.db');
            const groupB = get(activeGroupId);
            // A's match ids name rows of A: B's form is blank.
            expect(fields).toEqual({ matchIDsSelected: [], playerName: '' });
            fields = { matchIDsSelected: [3], playerName: 'Bob' };
            await activateGroup(groupA);
            expect(fields).toEqual({ matchIDsSelected: [7, 8], playerName: 'Ann' });
            await activateGroup(groupB);
            expect(fields).toEqual({ matchIDsSelected: [3], playerName: 'Bob' });
        } finally {
            detach();
        }
    });

    test('a return puts the Matches list back and reads it again only if never read', async () => {
        const groupA = get(activeGroupId);
        const rows = [{ id: 1 }, { id: 2 }];
        matchListStore.set({ rows, total: 2, text: '', ids: null, column: null, direction: 'asc', loading: false, loaded: true });
        await openInNewGroup('/libs/b.db');
        const groupB = get(activeGroupId);
        expect(get(matchListStore).rows).toEqual([]);

        const bumps = get(matchPanelRefreshTriggerStore);
        await activateGroup(groupA);
        expect(get(matchListStore).rows).toBe(rows);
        await new Promise((r) => requestAnimationFrame(() => setTimeout(r, 20)));
        // Same counts, a list already read: nothing repaints it.
        expect(get(matchPanelRefreshTriggerStore)).toBe(bumps);

        await activateGroup(groupB);
        await new Promise((r) => requestAnimationFrame(() => setTimeout(r, 20)));
        // B's list was never read: it is now.
        expect(get(matchPanelRefreshTriggerStore)).toBe(bumps + 1);
    });
});

describe('what the status bar says', () => {
    test('closing a group names it and the library shown, over an older refusal', async () => {
        await openInNewGroup('/libs/b.db');
        databasePathStore.set('/libs/b.db');
        statusBarTextStore.set({ i18nKey: 'libraryGroups.training', i18nParams: null });
        await closeGroup(get(activeGroupId));
        expect(get(statusBarTextStore)).toEqual({ i18nKey: 'libraryGroups.closed', i18nParams: { name: 'b', shown: 'a' } });
    });

    test('a job of a group off screen badges its header; its end is said, and its list read again, on return', async () => {
        const groupA = get(activeGroupId);
        await openInNewGroup('/libs/b.db');
        expect(noteGroupJob(groupA, 'running', 'import')).toBe(false);
        expect(get(groupActivityStore).get(groupA)?.state).toBe('running');
        const message = { i18nKey: 'status.importDone', i18nParams: { succeeded: 1, skipped: 0, failed: 0 } };
        noteGroupJob(groupA, 'done', 'import', message);
        noteGroupJob(groupA, 'running', 'batch');
        expect(get(groupActivityStore).get(groupA)).toEqual({ state: 'done', job: 'import', message });
        // A job of the group on screen shows as before: no badge.
        expect(noteGroupJob(get(activeGroupId), 'running', 'batch')).toBe(true);
        expect(get(groupActivityStore).has(get(activeGroupId))).toBe(false);

        const refreshes = get(matchPanelRefreshTriggerStore);
        expect(await activateGroup(groupA)).toBe(true);
        expect(get(groupActivityStore).has(groupA)).toBe(false);
        expect(get(statusBarTextStore)).toEqual(message);
        await new Promise((r) => requestAnimationFrame(() => setTimeout(r, 0)));
        await Promise.resolve();
        expect(get(matchPanelRefreshTriggerStore)).toBeGreaterThan(refreshes);
    });

    test('an import running file by file holds the group on screen', async () => {
        const groupA = get(activeGroupId);
        await openInNewGroup('/libs/b.db');
        importForegroundStore.set(true);
        try {
            expect(await activateGroup(groupA)).toBe(false);
            expect(get(statusBarTextStore)).toEqual({ i18nKey: 'libraryGroups.busyImport', i18nParams: null });
        } finally {
            importForegroundStore.set(false);
        }
    });

    test('a busy job is named; another failure says the switch failed, not an open', async () => {
        const groupA = get(activeGroupId);
        await openInNewGroup('/libs/b.db');
        vi.mocked(ActivateLibraryGroup).mockRejectedValueOnce('database is busy: rollout');
        expect(await activateGroup(groupA)).toBe(false);
        expect(get(statusBarTextStore)).toEqual({ i18nKey: 'libraryGroups.busyRollout', i18nParams: null });
        vi.mocked(ActivateLibraryGroup).mockRejectedValueOnce('database is busy: something-new');
        await activateGroup(groupA);
        expect(get(statusBarTextStore)).toEqual({ i18nKey: 'libraryGroups.busy', i18nParams: { job: 'something-new' } });
        vi.mocked(ActivateLibraryGroup).mockRejectedValueOnce(new Error('no library group 1'));
        await activateGroup(groupA);
        expect(get(statusBarTextStore)).toEqual({ i18nKey: 'libraryGroups.switchFailed', i18nParams: { name: 'a' } });
    });

    test('a library opened in a new group numbers its views from #1', async () => {
        viewStore.addView();
        await openInNewGroup('/libs/b.db');
        expect(get(viewStore.views).map((v) => v.name)).toEqual(['#1']);
        viewStore.addView();
        expect(get(viewStore.views).map((v) => v.name)).toEqual(['#1', '#2']);
    });

    test('during a Duel the tab keys say why they do nothing', async () => {
        viewStore.addView();
        duelStore.set({ state: { ended: false } });
        statusBarTextStore.set(null);
        expect(await cycleTab(1)).toBe(false);
        expect(get(statusBarTextStore)).toEqual({ i18nKey: 'libraryGroups.duelViews', i18nParams: null });
        duelStore.set(null);
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
