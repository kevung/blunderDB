// Library groups (ADR-0101): the desktop keeps several libraries open, one group of tabs each.
// The backend's bound database always carries the active library; the others wait open in Go.
// A switch must be instant: what the screen needs of a group (its views, its list, the position
// cache, the search, the counts, the scroll) is photographed when it is left and put back when it
// returns, without a request; what may be stale is refreshed once the group is on screen.

import { writable, derived, get } from 'svelte/store';
import { OpenLibraryGroup, ActivateLibraryGroup, CloseLibraryGroup } from '../../wailsjs/go/gui/App.js';
import { WindowSetTitle } from '../../wailsjs/runtime/runtime.js';
import { viewStore } from './viewStore.js';
import { positionsStore } from './positionStore.js';
import { databasePathStore } from './databaseStore.js';
import { libraryCountsStore, refreshLibraryCounts } from './libraryCountsStore.js';
import { searchParamsStore, searchEmptyStore } from './searchParamsStore.js';
import { matchPanelRefreshTriggerStore } from './uiStore.js';
import { logger } from '../utils/logger.js';

/**
 * @typedef {{ views: any[], activeViewId: number }} ViewsPhoto
 * @typedef {{ path: string, views: ViewsPhoto, list: import('./positionList.js').LibraryState,
 *   search: any, searchEmpty: boolean, counts: any, scroll: number }} GroupPhoto
 */

/** The groups in tab-bar order: ids only, the active one's path is databasePathStore. */
export const groupIdsStore = writable(/** @type {number[]} */ ([1]));
export const activeGroupId = writable(1);
/** Photographs of the groups waiting, by id. */
export const parkedGroupsStore = writable(/** @type {Map<number, GroupPhoto>} */ (new Map()));
/** How long the last switch took to put the group on screen, in ms (shown nowhere; tests, e2e). */
export const lastSwitchMsStore = writable(/** @type {number | null} */ (null));

const PALETTE_SIZE = 6;
/** @param {number} id */
export const groupColorIndex = (id) => ((id - 1) % PALETTE_SIZE) + 1;

/** @param {string} path */
export function groupName(path) {
    const file = (path || '').split(/[\\/]/).pop() || '';
    return file.replace(/\.db$/i, '');
}

/**
 * The tab bar's model: each group with its name, colour and views; the active one's views live.
 */
export const tabGroupsStore = derived(
    [groupIdsStore, activeGroupId, parkedGroupsStore, viewStore.views, viewStore.activeViewId, databasePathStore],
    ([$ids, $active, $parked, $views, $activeView, $path]) =>
        $ids.map((id) => {
            const active = id === $active;
            const photo = $parked.get(id);
            const path = active ? $path : (photo?.path ?? '');
            return {
                id,
                active,
                path,
                name: groupName(path),
                color: groupColorIndex(id),
                views: active ? $views : (photo?.views.views ?? []),
                activeViewId: active ? $activeView : (photo?.views.activeViewId ?? null)
            };
        })
);

function scrollBox() {
    return typeof document === 'undefined' ? null : document.querySelector('[data-testid="tab-content"]');
}

/**
 * Photograph the group on screen and hand its list to it; `incoming` is the list state of the
 * group coming in, null for a library just opened.
 * @param {import('./positionList.js').LibraryState | null} incoming
 * @returns {GroupPhoto}
 */
function photograph(incoming) {
    const views = viewStore.takeGroup();
    return {
        path: get(databasePathStore),
        views,
        list: positionsStore.exchangeLibrary(incoming),
        search: get(searchParamsStore),
        searchEmpty: get(searchEmptyStore),
        counts: get(libraryCountsStore),
        scroll: scrollBox()?.scrollTop ?? 0
    };
}

/** @param {string} path */
function showTitle(path) {
    WindowSetTitle(path ? `blunderDB - ${path.split(/[\\/]/).pop()}` : 'blunderDB');
}

/** What may have changed while the group waited, read once it is on screen. */
function refreshAfterShow() {
    refreshLibraryCounts();
    matchPanelRefreshTriggerStore.update((n) => n + 1);
}

/**
 * Open path in a new group and make it active; used by openDatabaseByPath, which runs the usual
 * after-open steps (session, counts, title). With no library open, the active group takes it.
 * @param {string} path
 */
export async function openInNewGroup(path) {
    if (viewStore.viewsLocked()) throw new Error('views locked');
    const leaving = get(activeGroupId);
    const group = await OpenLibraryGroup(path);
    if (!group || group.id === leaving) return;
    const photo = photograph(null);
    parkedGroupsStore.update((m) => new Map(m).set(leaving, photo));
    viewStore.giveGroup(null);
    searchParamsStore.set(null);
    searchEmptyStore.set(false);
    libraryCountsStore.set(null);
    groupIdsStore.update((ids) => [...ids, group.id]);
    activeGroupId.set(group.id);
}

/**
 * Bring group id on screen, on view viewId if given. The backend exchanges two open handles
 * (no I/O); the screen comes back from the group's photograph; counts and the match list are
 * refreshed afterwards.
 * @param {number} id
 * @param {number | null} [viewId]
 * @returns {Promise<boolean>} false when refused (a Duel holds the board, a job writes the library)
 */
export async function activateGroup(id, viewId = null) {
    const leaving = get(activeGroupId);
    if (id === leaving) {
        if (viewId != null) viewStore.switchTo(viewId);
        return true;
    }
    const target = get(parkedGroupsStore).get(id);
    if (!target || viewStore.viewsLocked()) return false;
    const started = performance.now();
    try {
        await ActivateLibraryGroup(id);
    } catch (error) {
        logger.error('Error switching library group:', error);
        const { setStatusBarMessage, libraryGroupError } = await import('../services/databaseService.js');
        setStatusBarMessage(libraryGroupError(error));
        return false;
    }
    const photo = photograph(target.list);
    parkedGroupsStore.update((m) => {
        const next = new Map(m);
        next.delete(id);
        next.set(leaving, photo);
        return next;
    });
    activeGroupId.set(id);
    databasePathStore.set(target.path);
    searchParamsStore.set(target.search);
    searchEmptyStore.set(target.searchEmpty);
    libraryCountsStore.set(target.counts);
    const views = viewId != null && target.views.views.some((v) => v.id === viewId) ? { ...target.views, activeViewId: viewId } : target.views;
    viewStore.giveGroup(views);
    showTitle(target.path);
    lastSwitchMsStore.set(performance.now() - started);
    requestAnimationFrame(() => {
        const box = scrollBox();
        if (box) box.scrollTop = target.scroll;
        refreshAfterShow();
    });
    return true;
}

/**
 * Close group id and its library. The active group hands the screen to its neighbour first; the
 * last group is never closed here (the ordinary close of a library is).
 * @param {number} id
 */
export async function closeGroup(id) {
    const ids = get(groupIdsStore);
    if (ids.length <= 1 || !ids.includes(id)) return false;
    if (id === get(activeGroupId)) {
        const at = ids.indexOf(id);
        const neighbour = ids[at + 1] ?? ids[at - 1];
        if (!(await activateGroup(neighbour))) return false;
    }
    try {
        await CloseLibraryGroup(id);
    } catch (error) {
        logger.error('Error closing library group:', error);
        return false;
    }
    parkedGroupsStore.update((m) => {
        const next = new Map(m);
        next.delete(id);
        return next;
    });
    groupIdsStore.update((list) => list.filter((g) => g !== id));
    return true;
}

/** @param {number} step +1 next, -1 previous */
export function cycleGroup(step) {
    const ids = get(groupIdsStore);
    if (ids.length <= 1) return Promise.resolve(false);
    const at = ids.indexOf(get(activeGroupId));
    return activateGroup(ids[(at + step + ids.length) % ids.length]);
}

/**
 * The next or previous tab across every group: past a group's last tab comes the next group's
 * first, so one gesture walks the whole bar.
 * @param {number} step +1 next, -1 previous
 */
export function cycleTab(step) {
    const flat = get(tabGroupsStore).flatMap((g) => g.views.map((v) => ({ group: g.id, view: v.id })));
    if (flat.length <= 1) return Promise.resolve(false);
    const group = get(activeGroupId);
    const view = get(viewStore.activeViewId);
    const at = flat.findIndex((t) => t.group === group && t.view === view);
    const next = flat[(at + step + flat.length) % flat.length];
    if (next.group === group) {
        viewStore.switchTo(next.view);
        return Promise.resolve(true);
    }
    return activateGroup(next.group, next.view);
}
