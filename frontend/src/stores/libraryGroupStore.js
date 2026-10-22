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
import { databasePathStore, libraryChanged, libraryTitle } from './databaseStore.js';
import { libraryCountsStore, refreshLibraryCounts } from './libraryCountsStore.js';
import { matchPanelRefreshTriggerStore, statusBarTextStore, dbMutationCounterStore } from './uiStore.js';
import { trainingSessionStore } from './trainingTabStore.js';
import { bumpTranscriptionLibrary } from './transcriptionStore.js';
import { matchListStore } from './matchListStore.js';
import { directionSessionState, forgetDirection, restoreDirection } from './directionStore.js';
import { takeScopedState, putScopedState } from './libraryScope.js';
import { forgetTransferGroup } from './transferBasketStore.js';
import { groupIdsStore, activeGroupId, takeActivity, importForegroundStore as importForegroundStoreRef, groupSwitchingStore, holdingSwitch } from './groupActivityStore.js';

export { groupIdsStore, activeGroupId, groupActivityStore, importForegroundStore, groupOnScreen, noteGroupJob, groupSwitchingStore } from './groupActivityStore.js';
import { tMsg, translate } from '../i18n';
import { moveItem } from '../utils/moveItem.js';
import { logger } from '../utils/logger.js';

/**
 * @typedef {{ views: any[], activeViewId: number, nextNumber?: number }} ViewsPhoto
 * @typedef {{ path: string, views: ViewsPhoto, list: import('./positionList.js').LibraryState,
 *   counts: any, scroll: ScrollMark[], scoped: Record<string, any>,
 *   direction: { openId: number | null, tabs: Record<string, string> } }} GroupPhoto
 */

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

/**
 * @typedef {{ tag: string, cls: string, nth: number, top: number, left: number }} ScrollMark
 *   an element of the panel scrolled away from its origin, found again by tag, class and rank
 */

function panelRoot() {
    return typeof document === 'undefined' ? null : document.querySelector('[data-testid="tab-content"]');
}

/**
 * Where each scrolling box of the panel stands. The panel's frame does not scroll: its lists
 * and columns do (a Matches list, the Search filters), each its own box, so every box is read.
 * @returns {ScrollMark[]}
 */
function scrollMarks() {
    const root = panelRoot();
    if (!root) return [];
    /** @type {ScrollMark[]} */
    const marks = [];
    /** @type {Map<string, number>} */
    const ranks = new Map();
    for (const el of [root, ...root.querySelectorAll('*')]) {
        const key = `${el.tagName}|${el.className}`;
        const nth = ranks.get(key) ?? 0;
        ranks.set(key, nth + 1);
        if (el.scrollTop > 0 || el.scrollLeft > 0) marks.push({ tag: el.tagName, cls: String(el.className), nth, top: el.scrollTop, left: el.scrollLeft });
    }
    return marks;
}

/** @param {ScrollMark} mark */
function findMarked(mark) {
    const root = panelRoot();
    if (!root) return null;
    const same = [root, ...root.querySelectorAll(mark.tag)].filter((el) => el.tagName === mark.tag && String(el.className) === mark.cls);
    return same[mark.nth] ?? null;
}

/** How long a box may take to grow back to its length (a list read again) before giving up. */
const SCROLL_PATIENCE_MS = 1500;

/**
 * Put the panel's boxes back where they stood. A list read again after the switch is short at
 * first: its box is scrolled again each frame until it reaches its place, unless the user moved
 * it meanwhile.
 * @param {ScrollMark[]} marks
 */
function restoreScroll(marks) {
    if (!marks.length) return;
    const until = performance.now() + SCROLL_PATIENCE_MS;
    /** @type {Map<ScrollMark, number>} */
    const lastSet = new Map();
    const step = () => {
        let pending = false;
        for (const mark of marks) {
            const el = findMarked(mark);
            if (!el) {
                pending = true;
                continue;
            }
            const was = lastSet.get(mark);
            if (was !== undefined && Math.abs(el.scrollTop - was) > 1) continue;
            if (Math.abs(el.scrollTop - mark.top) <= 1 && Math.abs(el.scrollLeft - mark.left) <= 1) continue;
            el.scrollTop = mark.top;
            el.scrollLeft = mark.left;
            lastSet.set(mark, el.scrollTop);
            if (Math.abs(el.scrollTop - mark.top) > 1) pending = true;
        }
        if (pending && performance.now() < until) requestAnimationFrame(step);
    };
    step();
}

/**
 * Photograph the group on screen and hand its list to it; `incoming` is the list state of the
 * group coming in, null for a library just opened. The group's own state (a draft, a review,
 * the open Direction) goes into the photograph; what the next library starts from is installed
 * by putScopedState, after this, so a reset never reaches the photograph.
 * @param {import('./positionList.js').LibraryState | null} incoming
 * @returns {GroupPhoto}
 */
function photograph(incoming) {
    const views = viewStore.takeGroup();
    const photo = {
        path: get(databasePathStore),
        views,
        list: positionsStore.exchangeLibrary(incoming),
        counts: get(libraryCountsStore),
        scroll: scrollMarks(),
        scoped: takeScopedState(),
        direction: directionSessionState()
    };
    // Its ids name rows of the library left; its pending session save would land in the next.
    forgetDirection();
    // A transcription answer still in flight belongs to the library left.
    bumpTranscriptionLibrary();
    return photo;
}

/** @param {string} path @param {boolean} readOnly */
function showTitle(path, readOnly) {
    WindowSetTitle(libraryTitle(path, readOnly, translate('commands.readOnlySuffix')));
}

/**
 * Why the group on screen cannot be left now, as a status-bar message; null when it can.
 * A running Duel holds the board, and a training session's clock and journal follow no group.
 */
export function switchRefusal() {
    if (viewStore.viewsLocked()) return tMsg('libraryGroups.duel');
    if (get(trainingSessionStore)) return tMsg('libraryGroups.training');
    if (get(importForegroundStoreRef)) return tMsg('libraryGroups.busyImport');
    return null;
}

/** A switch refused before anything moved; libraryGroupError shows its status message. */
export class LibraryGroupRefused extends Error {
    name = 'LibraryGroupRefused';
    /** @param {import('../i18n').StatusMessage} status */
    constructor(status) {
        super(status.i18nKey);
        this.status = status;
    }
}

/**
 * What may have changed while the group waited, read once it is on screen. The match list came
 * back with its photograph: it is read again only if it never was, if the library's match
 * count moved, or if a background job wrote the library meanwhile (stale), since reading it
 * again repaints every row.
 * @param {any} countsBefore the counts photographed with the group
 * @param {boolean} [stale] a job of the group wrote its library while it was off screen
 */
async function refreshAfterShow(countsBefore, stale = false) {
    const group = get(activeGroupId);
    await refreshLibraryCounts();
    if (get(activeGroupId) !== group) return;
    const list = get(matchListStore);
    if (stale || !list.loaded || get(libraryCountsStore)?.matches !== countsBefore?.matches) matchPanelRefreshTriggerStore.update((n) => n + 1);
    // What reads the library on a write (statistics, the panels) reads it again.
    if (stale) dbMutationCounterStore.update((n) => n + 1);
}

/**
 * Open path in a new group and make it active; used by openDatabaseByPath, which runs the usual
 * after-open steps (session, counts, title). With no library open, the active group takes it.
 * @param {string} path
 */
export async function openInNewGroup(path) {
    const refusal = switchRefusal();
    if (refusal) throw new LibraryGroupRefused(refusal);
    if (get(groupSwitchingStore)) throw new LibraryGroupRefused(tMsg('libraryGroups.switching'));
    await holdingSwitch(() => openGroup(path));
}

/**
 * openInNewGroup under the switch lock.
 * @param {string} path
 */
async function openGroup(path) {
    const leaving = get(activeGroupId);
    const group = await OpenLibraryGroup(path);
    if (!group || group.id === leaving) return;
    const photo = photograph(null);
    parkedGroupsStore.update((m) => new Map(m).set(leaving, photo));
    putScopedState(null);
    viewStore.giveGroup(null);
    libraryCountsStore.set(null);
    groupIdsStore.update((ids) => [...ids, group.id]);
    activeGroupId.set(group.id);
    libraryChanged();
}

/**
 * Bring group id on screen, on view viewId if given. The backend exchanges two open handles
 * (no I/O); the screen comes back from the group's photograph; counts and the match list are
 * refreshed afterwards.
 * @param {number} id
 * @param {number | null} [viewId]
 * @returns {Promise<boolean>} false when refused (a Duel holds the board, a training session
 *   runs, a job writes the library); the status bar says why
 */
export async function activateGroup(id, viewId = null) {
    // A key held down repeats: the switches after the first wait for none, they are dropped.
    const done = await holdingSwitch(() => switchGroup(id, viewId));
    return done ?? false;
}

/**
 * activateGroup under the switch lock.
 * @param {number} id
 * @param {number | null} viewId
 */
async function switchGroup(id, viewId) {
    const leaving = get(activeGroupId);
    if (id === leaving) {
        if (viewId != null) viewStore.switchTo(viewId);
        return true;
    }
    const target = get(parkedGroupsStore).get(id);
    if (!target) return false;
    const refusal = switchRefusal();
    if (refusal) {
        statusBarTextStore.set(refusal);
        return false;
    }
    const started = performance.now();
    try {
        await ActivateLibraryGroup(id);
    } catch (error) {
        logger.error('Error switching library group:', error);
        const { libraryGroupError } = await import('../services/databaseService.js');
        statusBarTextStore.set(libraryGroupError(error, { switching: groupName(target.path) }));
        return false;
    }
    const photo = photograph(target.list);
    parkedGroupsStore.update((m) => {
        const next = new Map(m);
        next.delete(id);
        next.set(leaving, photo);
        return next;
    });
    const { activity, stale } = takeActivity(id);
    // The path first: what follows the path (the match list drops another library's ids) acts on
    // the state left, and the group's own state is put back over it.
    databasePathStore.set(target.path);
    putScopedState(target.scoped);
    activeGroupId.set(id);
    libraryCountsStore.set(target.counts);
    const views = viewId != null && target.views.views.some((v) => v.id === viewId) ? { ...target.views, activeViewId: viewId } : target.views;
    viewStore.giveGroup(views);
    showTitle(target.path, target.scoped.databaseReadOnlyStore ?? false);
    libraryChanged();
    lastSwitchMsStore.set(performance.now() - started);
    // What a job finished while the group was away is said once it is back.
    if (activity?.state === 'done' && activity.message) statusBarTextStore.set(activity.message);
    requestAnimationFrame(() => {
        restoreScroll(target.scroll);
        refreshAfterShow(target.counts, stale).catch((error) => logger.error('Error refreshing a library group:', error));
        restoreDirection(target.direction).catch((error) => logger.error('Error reopening the Direction of a library group:', error));
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
    const closing = get(tabGroupsStore).find((g) => g.id === id)?.name ?? '';
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
    takeActivity(id);
    forgetTransferGroup(id);
    // Says what happened, and replaces whatever an earlier refusal left there.
    statusBarTextStore.set(tMsg('libraryGroups.closed', { name: closing, shown: groupName(get(databasePathStore)) }));
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
        if (viewStore.viewsLocked()) {
            statusBarTextStore.set(tMsg('libraryGroups.duelViews'));
            return Promise.resolve(false);
        }
        viewStore.switchTo(next.view);
        return Promise.resolve(true);
    }
    return activateGroup(next.group, next.view);
}

/**
 * Put a view before or after another of the same group, on screen or waiting. A view never
 * leaves its group: the tab bar offers no drop elsewhere.
 * @param {number} groupId
 * @param {number} viewId
 * @param {number} targetId
 * @param {boolean} after
 */
export function moveGroupView(groupId, viewId, targetId, after) {
    if (groupId === get(activeGroupId)) {
        viewStore.moveView(viewId, targetId, after);
        return;
    }
    parkedGroupsStore.update((m) => {
        const photo = m.get(groupId);
        if (!photo) return m;
        const moved = moveItem(photo.views.views, (v) => v.id, viewId, targetId, after);
        if (moved === photo.views.views) return m;
        return new Map(m).set(groupId, { ...photo, views: { ...photo.views, views: moved } });
    });
}

/**
 * Put a group, with all its views, before or after another.
 * @param {number} groupId
 * @param {number} targetId
 * @param {boolean} after
 */
export function moveGroup(groupId, targetId, after) {
    groupIdsStore.update((ids) => moveItem(ids, (id) => id, groupId, targetId, after));
}
