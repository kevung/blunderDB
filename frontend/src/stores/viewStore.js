import { writable, get } from 'svelte/store';
import { positionStore, positionsStore, matchContextStore, emptyPosition } from './positionStore';
import { indexInList, listLength } from './positionList.js';
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
        matchContext: createDefaultMatchContext()
    };
}

let nextViewId = 2;

function createViewStore() {
    const views = writable([createDefaultView(1)]);
    const activeViewId = writable(1);

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
                        matchContext: JSON.parse(JSON.stringify(get(matchContextStore)))
                    };
                }
                return v;
            })
        );
    }

    function restoreViewState(view) {
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

    function switchTo(viewId) {
        const currentId = get(activeViewId);
        if (viewId === currentId) return;
        saveCurrentViewState();
        const vs = get(views);
        const target = vs.find((v) => v.id === viewId);
        if (target) {
            activeViewId.set(viewId);
            restoreViewState(target);
        }
    }

    function addView() {
        saveCurrentViewState();
        const id = nextViewId++;
        const currentId = get(activeViewId);
        const vs = get(views);
        const current = vs.find((v) => v.id === currentId);
        const newView = {
            ...JSON.parse(JSON.stringify({ ...current, list: null })),
            // A paged list's source is functions, which JSON drops: the list is shared, not copied.
            list: current.list,
            id,
            name: `#${id}`
        };
        views.update((vs) => [...vs, newView]);
        activeViewId.set(id);
        restoreViewState(newView);
    }

    function closeView(viewId) {
        const vs = get(views);
        if (vs.length <= 1) return;
        const remaining = vs.filter((v) => v.id !== viewId);
        views.set(remaining);
        if (get(activeViewId) === viewId) {
            const next = remaining[remaining.length - 1];
            activeViewId.set(next.id);
            restoreViewState(next);
        }
    }

    function renameView(viewId, newName) {
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
                    mode,
                    previousMode: v.previousMode || 'NORMAL'
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

            const restoredViews = [];
            for (const sv of data.views) {
                const origin = sv.origin || LIBRARY_ORIGIN;
                const list = (await resolveListFn(origin)) || { ids: [] };
                let positionIndex = sv.positionId != null ? await indexInList(list, sv.positionId) : -1;
                if (positionIndex < 0) positionIndex = Math.min(sv.positionIndex || 0, Math.max(listLength(list) - 1, 0));
                restoredViews.push({
                    id: sv.id,
                    name: sv.name,
                    list,
                    positionId: sv.positionId ?? null,
                    origin,
                    positionIndex,
                    // No board yet: from the cache, or the index effect (getPosition).
                    position: null,
                    analysis: createDefaultAnalysis(),
                    selectedMove: sv.selectedMove ?? null,
                    // Old names (`epc` tab, `EPC` mode) reopen as Eval.
                    activeTab: normalizeTabId(sv.activeTab) || 'analysis',
                    commentText: sv.commentText || '',
                    mode: (sv.mode === 'EPC' ? 'EVAL' : sv.mode) || 'NORMAL',
                    previousMode: sv.previousMode || 'NORMAL',
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
        deserialize
    };
}

export const viewStore = createViewStore();
