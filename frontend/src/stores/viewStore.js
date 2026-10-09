import { writable, get } from 'svelte/store';
import { duelHoldsBoardStore } from './duelStore.js';
import { positionStore, positionsStore, matchContextStore, emptyPosition } from './positionStore';
import { indexInList, listLength, isSettled } from './positionList.js';
import { analysisStore, selectedMoveStore } from './analysisStore';
import { currentPositionIndexStore, activeTabStore, commentTextStore, statusBarModeStore } from './uiStore';
import { listOriginStore, LIBRARY_ORIGIN } from './listOriginStore';
import { logger } from '../utils/logger.js';
import { normalizeTabId } from '../services/tabOrder.js';

function createDefaultAnalysis() {
    return {
        positionId: 0,
        xgid: '',
        player1: '',
        player2: '',
        analysisType: '',
        analysisEngineVersion: '',
        checkerAnalysis: { moves: [] },
        doublingCubeAnalysis: null,
        allCubeAnalyses: [],
        playedMoves: [],
        playedCubeActions: [],
        creationDate: '',
        lastModifiedDate: ''
    };
}

function createDefaultMatchContext() {
    return { isMatchMode: false, matchID: null, movePositions: [], currentIndex: 0, player1Name: '', player2Name: '' };
}

function createDefaultView(id) {
    return {
        id,
        name: `#${id}`,
        list: /** @type {import('./positionList.js').ListSnapshot | null} */ (null),
        positionId: /** @type {number | null} */ (null),
        origin: LIBRARY_ORIGIN,
        positionIndex: 0,
        position: emptyPosition(),
        analysis: createDefaultAnalysis(),
        selectedMove: null,
        activeTab: 'matches',
        commentText: '',
        mode: 'NORMAL',
        matchContext: createDefaultMatchContext(),
        // The view's Eval slots (modeMachine's takeEvalContext): its way back and its last
        // scratch board. Not persisted: a scratch board lives as long as the session.
        evalContext: /** @type {EvalContext | null} */ (null)
    };
}

/** @typedef {{ beforeEval: any, lastEvalBoard: any, evalSeed: any }} EvalContext */

/**
 * A copy a second view can own. The way back's list is shared, not copied: a paged list's
 * source is functions; it is a snapshot nobody mutates.
 *
 * @param {EvalContext | null | undefined} context
 * @returns {EvalContext | null}
 */
function copyEvalContext(context) {
    if (!context) return null;
    const clone = (/** @type {any} */ value) => (value ? JSON.parse(JSON.stringify(value)) : null);
    const before = context.beforeEval;
    return {
        beforeEval: before ? { ...clone({ ...before, list: null }), list: before.list } : null,
        lastEvalBoard: clone(context.lastEvalBoard),
        evalSeed: clone(context.evalSeed)
    };
}

let nextViewId = 2;

function createViewStore() {
    const views = writable([createDefaultView(1)]);
    const activeViewId = writable(1);
    // Counts a paged list put back before its length was known and ranks a position in it
    // (positionService, which imports this store: handed in rather than imported). Resolves to
    // `{ index }` once done (index -1 when not asked for or absent), null when it did not run.
    /** @typedef {(options: { source: import('./positionList.js').IdSource, count: boolean, positionId: number | null }) => Promise<{ index: number } | null>} ListSettler */
    /** @type {ListSettler} */
    let settleList = async () => null;

    /** @param {ListSettler} fn */
    function setListSettler(fn) {
        settleList = fn;
    }

    // The Eval slots live in modeMachine (which imports this store): handed in, as the settler.
    /** @type {{ take: () => EvalContext | null, give: (context: EvalContext | null) => void }} */
    let evalContextKeeper = { take: () => null, give: () => {} };

    /** @param {{ take: () => EvalContext | null, give: (context: EvalContext | null) => void }} keeper */
    function setEvalContextKeeper(keeper) {
        evalContextKeeper = keeper;
    }

    // A view whose list or position is still to be found asks for it once it is on screen; only
    // the list on screen is counted, so restoring several views scans for one. The count lands
    // on the view's source whichever view is shown by then (positionList's settled lengths); the
    // rank lands on the view itself, shown or left, unless the user moved in it meanwhile. A
    // settling that did not run (a search owned the backend) leaves the rank pending, asked for
    // again the next time the view is shown or that search ends.
    /** @param {number} viewId */
    async function settle(viewId) {
        const view = get(views).find((v) => v.id === viewId);
        const list = view?.list;
        if (!view || !list || !('source' in list)) return;
        const positionId = view.pendingPositionId ?? null;
        const count = !isSettled(list);
        if (positionId == null && !count) return;
        const indexAtStart = view.positionIndex || 0;
        const shownIndex = get(activeViewId) === viewId ? get(currentPositionIndexStore) : null;
        const result = await settleList({ source: list.source, count, positionId }).catch(() => null);
        if (!result) return;
        const index = result.index;
        const shown = get(activeViewId) === viewId;
        if (shown) {
            const unmoved = get(currentPositionIndexStore) === (shownIndex ?? indexAtStart);
            if (index >= 0 && unmoved && positionsStore.isSource(list.source)) currentPositionIndexStore.set(index);
            views.update((vs) => vs.map((v) => (v.id === viewId ? { ...v, pendingPositionId: null } : v)));
            return;
        }
        views.update((vs) =>
            vs.map((v) => {
                if (v.id !== viewId) return v;
                const moved = v.positionIndex !== (shownIndex ?? indexAtStart);
                if (index < 0 || moved) return { ...v, pendingPositionId: null };
                // Left before its rank came back: the board saved with it was the fallback's.
                return { ...v, pendingPositionId: null, positionIndex: index, positionId, position: null };
            })
        );
    }

    /** Settle the view on screen: a search that held the backend has ended. */
    function settleActive() {
        const view = get(views).find((v) => v.id === get(activeViewId));
        // Only while the view still shows its own list: the search may have replaced it.
        if (view?.list && 'source' in view.list && positionsStore.isSource(view.list.source)) settle(view.id);
    }

    function saveCurrentViewState() {
        const currentId = get(activeViewId);
        views.update((vs) =>
            vs.map((v) => {
                if (v.id === currentId) {
                    return {
                        ...v,
                        list: positionsStore.snapshotList(),
                        positionId: positionsStore.idAt(get(currentPositionIndexStore)) ?? null,
                        origin: get(listOriginStore),
                        positionIndex: get(currentPositionIndexStore),
                        position: JSON.parse(JSON.stringify(get(positionStore))),
                        analysis: JSON.parse(JSON.stringify(get(analysisStore))),
                        selectedMove: get(selectedMoveStore),
                        activeTab: get(activeTabStore),
                        commentText: get(commentTextStore),
                        mode: get(statusBarModeStore),
                        matchContext: JSON.parse(JSON.stringify(get(matchContextStore))),
                        evalContext: evalContextKeeper.take()
                    };
                }
                return v;
            })
        );
    }

    function restoreViewState(view) {
        evalContextKeeper.give(view.evalContext ?? null);
        // A view left in Eval this session resumes there, its scratch board and its way back
        // intact: the tab handler's entry would photograph the scratch board as the way back.
        // Eval is set before the board lands, as enterEvalMode does. Every scratch board has
        // id 0 in the shared cache, so the list is the board itself, never the snapshot.
        if (view.mode === 'EVAL' && view.evalContext?.beforeEval && view.position) {
            const board = JSON.parse(JSON.stringify(view.position));
            statusBarModeStore.set('EVAL');
            positionsStore.set([board]);
            listOriginStore.set(view.origin || LIBRARY_ORIGIN);
            positionStore.set(board);
            analysisStore.set(view.analysis);
            selectedMoveStore.set(view.selectedMove ?? null);
            activeTabStore.set(view.activeTab || 'eval');
            commentTextStore.set(view.commentText || '');
            matchContextStore.set(view.matchContext || createDefaultMatchContext());
            currentPositionIndexStore.set(-1);
            currentPositionIndexStore.set(0);
            return;
        }
        // The position cache is shared by every view (keyed by id): only the list moves.
        positionsStore.restoreList(view.list);
        listOriginStore.set(view.origin || LIBRARY_ORIGIN);
        // A view restored from disk has no board: cache, else the index effect fetches it.
        const cached = view.position ? null : positionsStore.peek(view.positionIndex || 0);
        positionStore.set(view.position ?? (cached ? JSON.parse(JSON.stringify(cached)) : emptyPosition()));
        analysisStore.set(view.analysis);
        selectedMoveStore.set(view.selectedMove ?? null);
        // EVAL and EDIT follow the active tab: restore as NORMAL, the tab handler re-enters them
        // once the DB is open.
        const mode = view.mode || 'NORMAL';
        statusBarModeStore.set(mode === 'EVAL' || mode === 'EDIT' ? 'NORMAL' : mode);
        activeTabStore.set(view.activeTab || 'matches');
        commentTextStore.set(view.commentText || '');
        matchContextStore.set(view.matchContext || createDefaultMatchContext());
        currentPositionIndexStore.set(-1);
        currentPositionIndexStore.set(view.positionIndex || 0);
    }

    // A Duel holds the board: a view restore would put a library position under it.
    const viewsLocked = () => get(duelHoldsBoardStore);

    function switchTo(/** @type {number} */ viewId) {
        if (viewsLocked()) return;
        const currentId = get(activeViewId);
        if (viewId === currentId) return;
        saveCurrentViewState();
        const vs = get(views);
        const target = vs.find((v) => v.id === viewId);
        if (target) {
            activeViewId.set(viewId);
            restoreViewState(target);
            settle(target.id);
        }
    }

    /**
     * Open a copy of the view on screen and show it.
     *
     * @param {{ name?: (id: number, originId: number) => string }} [options] names the new tab
     *        (default `#id`), from its id and the id of the view it was copied from
     * @returns {number | null} the new view's id, null while views are locked
     */
    function addView(options = {}) {
        if (viewsLocked()) return null;
        saveCurrentViewState();
        const currentId = get(activeViewId);
        const current = get(views).find((v) => v.id === currentId);
        if (!current) return null;
        const id = nextViewId++;
        const newView = {
            ...JSON.parse(JSON.stringify({ ...current, list: null, evalContext: null })),
            // A paged list's source is functions, which JSON drops: the list is shared, not copied.
            list: current.list,
            evalContext: copyEvalContext(current.evalContext),
            id,
            name: options.name ? options.name(id, currentId) : `#${id}`
        };
        views.update((vs) => [...vs, newView]);
        activeViewId.set(id);
        restoreViewState(newView);
        return id;
    }

    function closeView(/** @type {number} */ viewId) {
        if (viewsLocked()) return;
        const vs = get(views);
        if (vs.length <= 1) return;
        const remaining = vs.filter((v) => v.id !== viewId);
        views.set(remaining);
        if (get(activeViewId) === viewId) {
            const next = remaining[remaining.length - 1];
            activeViewId.set(next.id);
            restoreViewState(next);
            settle(next.id);
        }
    }

    function renameView(/** @type {number} */ viewId, newName) {
        views.update((vs) => vs.map((v) => (v.id === viewId ? { ...v, name: newName } : v)));
    }

    // Serialize all views for persistence: each view's definition (name, where its list comes
    // from, the id of the position it shows), never the list itself, so the payload does not
    // grow with the library.
    function serialize() {
        saveCurrentViewState();
        const vs = get(views);
        return JSON.stringify({
            nextViewId,
            activeViewId: get(activeViewId),
            views: vs.map((v) => {
                const mode = v.mode || 'NORMAL';
                // A match or a collection is not replayable from the library: reopen on the library.
                const replayable = mode !== 'MATCH' && mode !== 'COLLECTION';
                return {
                    id: v.id,
                    name: v.name,
                    origin: replayable && v.origin ? v.origin : LIBRARY_ORIGIN,
                    positionId: v.positionId ?? null,
                    positionIndex: v.positionIndex || 0,
                    selectedMove: v.selectedMove,
                    activeTab: v.activeTab || 'analysis',
                    commentText: v.commentText || '',
                    mode: replayable ? mode : 'NORMAL',
                    previousMode: replayable ? v.previousMode || 'NORMAL' : 'NORMAL'
                };
            })
        });
    }

    // Restore views: each list is rebuilt by replaying its origin (resolveListFn(origin) → a
    // positionList snapshot), and the current position is found again by id, the saved index
    // being the fallback.
    async function deserialize(json, resolveListFn) {
        try {
            const data = JSON.parse(json);
            if (!data || !data.views || data.views.length === 0) return false;

            nextViewId = data.nextViewId || data.views.length + 1;

            // Side by side: each replay reads at most a first window, as a search does.
            const lists = await Promise.all(data.views.map(async (sv) => (await resolveListFn(sv.origin || LIBRARY_ORIGIN)) || { ids: [] }));
            const restoredViews = [];
            for (const [i, sv] of data.views.entries()) {
                const origin = sv.origin || LIBRARY_ORIGIN;
                const list = lists[i];
                // A list known only by its first page is not ranked here: the rank of a position
                // past it is a scan, run once the view is on screen (settle).
                let positionIndex = -1;
                let pendingPositionId = null;
                if (sv.positionId != null) {
                    if (isSettled(list)) positionIndex = await indexInList(list, sv.positionId);
                    else {
                        positionIndex = Array.isArray(list.firstPage) ? list.firstPage.indexOf(sv.positionId) : -1;
                        if (positionIndex < 0) pendingPositionId = sv.positionId;
                    }
                }
                if (positionIndex < 0) positionIndex = Math.min(sv.positionIndex || 0, Math.max(listLength(list) - 1, 0));
                // A match or a collection reopens on the library (serialize): as the library.
                const savedMode = (sv.mode === 'EPC' ? 'EVAL' : sv.mode) || 'NORMAL';
                const mode = savedMode === 'MATCH' || savedMode === 'COLLECTION' ? 'NORMAL' : savedMode;
                const previousMode = sv.previousMode === 'MATCH' || sv.previousMode === 'COLLECTION' ? 'NORMAL' : sv.previousMode || 'NORMAL';
                restoredViews.push({
                    id: sv.id,
                    name: sv.name,
                    list,
                    positionId: sv.positionId ?? null,
                    pendingPositionId,
                    origin,
                    positionIndex,
                    // No board yet: from the cache, or the index effect (getPosition).
                    position: null,
                    analysis: createDefaultAnalysis(),
                    selectedMove: sv.selectedMove ?? null,
                    // Old names (`epc` tab, `EPC` mode) reopen as Eval.
                    activeTab: normalizeTabId(sv.activeTab) || 'analysis',
                    commentText: sv.commentText || '',
                    mode,
                    previousMode,
                    matchContext: createDefaultMatchContext()
                });
            }

            views.set(restoredViews);
            const targetId = data.activeViewId || restoredViews[0].id;
            activeViewId.set(targetId);
            const target = restoredViews.find((v) => v.id === targetId) || restoredViews[0];
            // A paged list reads the page of its current position first, so the board can come
            // from the cache as it does for a list held whole.
            positionsStore.restoreList(target.list);
            await positionsStore.ensureIds(target.positionIndex, target.positionIndex);
            restoreViewState(target);
            settle(target.id);
            return true;
        } catch (e) {
            logger.error('Error deserializing views:', e);
            return false;
        }
    }

    function selectPreviousView() {
        const vs = get(views);
        if (vs.length <= 1) return;
        const currentId = get(activeViewId);
        const idx = vs.findIndex((v) => v.id === currentId);
        const prevIdx = idx > 0 ? idx - 1 : vs.length - 1;
        switchTo(vs[prevIdx].id);
    }

    function selectNextView() {
        const vs = get(views);
        if (vs.length <= 1) return;
        const currentId = get(activeViewId);
        const idx = vs.findIndex((v) => v.id === currentId);
        const nextIdx = idx < vs.length - 1 ? idx + 1 : 0;
        switchTo(vs[nextIdx].id);
    }

    return {
        views,
        activeViewId,
        switchTo,
        addView,
        closeView,
        renameView,
        selectPreviousView,
        selectNextView,
        saveCurrentViewState,
        serialize,
        deserialize,
        setListSettler,
        setEvalContextKeeper,
        settleActive
    };
}

export const viewStore = createViewStore();
