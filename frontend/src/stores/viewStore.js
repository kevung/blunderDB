import { writable, get } from 'svelte/store';
import { positionStore, positionsStore, matchContextStore } from './positionStore';
import { analysisStore, selectedMoveStore } from './analysisStore';
import { currentPositionIndexStore, activeTabStore, commentTextStore, statusBarModeStore, previousModeStore } from './uiStore';

function createDefaultPosition() {
    return {
        id: 0,
        board: {
            points: [0, -2, 0, 0, 0, 0, 5, 0, 3, 0, 0, 0, -5, 5, 0, 0, 0, -3, 0, -5, 0, 0, 0, 0, 2, 0],
            bearoff: [0, 0]
        },
        cube: { owner: -1, value: 0 },
        dice: [3, 1],
        score: [-1, -1],
        player_on_roll: 1,
        decision_type: '',
        has_jacoby: false,
        has_beaver: false
    };
}

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
        positions: [],
        positionIndex: 0,
        position: createDefaultPosition(),
        analysis: createDefaultAnalysis(),
        selectedMove: null,
        activeTab: 'analysis',
        commentText: '',
        mode: 'NORMAL',
        previousMode: 'NORMAL',
        matchContext: createDefaultMatchContext()
    };
}

let nextViewId = 2;

function createViewStore() {
    const views = writable([createDefaultView(1)]);
    const activeViewId = writable(1);

    function saveCurrentViewState() {
        const currentId = get(activeViewId);
        views.update(vs => vs.map(v => {
            if (v.id === currentId) {
                return {
                    ...v,
                    positions: get(positionsStore),
                    positionIndex: get(currentPositionIndexStore),
                    position: JSON.parse(JSON.stringify(get(positionStore))),
                    analysis: JSON.parse(JSON.stringify(get(analysisStore))),
                    selectedMove: get(selectedMoveStore),
                    activeTab: get(activeTabStore),
                    commentText: get(commentTextStore),
                    mode: get(statusBarModeStore),
                    previousMode: get(previousModeStore),
                    matchContext: JSON.parse(JSON.stringify(get(matchContextStore)))
                };
            }
            return v;
        }));
    }

    function restoreViewState(view) {
        positionsStore.set(view.positions || []);
        positionStore.set(view.position);
        analysisStore.set(view.analysis);
        selectedMoveStore.set(view.selectedMove ?? null);
        activeTabStore.set(view.activeTab || 'analysis');
        commentTextStore.set(view.commentText || '');
        statusBarModeStore.set(view.mode || 'NORMAL');
        previousModeStore.set(view.previousMode || 'NORMAL');
        matchContextStore.set(view.matchContext || createDefaultMatchContext());
        currentPositionIndexStore.set(-1);
        currentPositionIndexStore.set(view.positionIndex || 0);
    }

    function switchTo(viewId) {
        const currentId = get(activeViewId);
        if (viewId === currentId) return;
        saveCurrentViewState();
        const vs = get(views);
        const target = vs.find(v => v.id === viewId);
        if (target) {
            activeViewId.set(viewId);
            restoreViewState(target);
        }
    }

    function addView() {
        const id = nextViewId++;
        const newView = createDefaultView(id);
        saveCurrentViewState();
        views.update(vs => [...vs, newView]);
        activeViewId.set(id);
        restoreViewState(newView);
    }

    function closeView(viewId) {
        const vs = get(views);
        if (vs.length <= 1) return;
        const remaining = vs.filter(v => v.id !== viewId);
        views.set(remaining);
        if (get(activeViewId) === viewId) {
            const next = remaining[remaining.length - 1];
            activeViewId.set(next.id);
            restoreViewState(next);
        }
    }

    function renameView(viewId, newName) {
        views.update(vs => vs.map(v => v.id === viewId ? { ...v, name: newName } : v));
    }

    // Serialize all views for persistence (only stores position IDs, not full objects)
    function serialize() {
        saveCurrentViewState();
        const vs = get(views);
        return JSON.stringify({
            nextViewId,
            activeViewId: get(activeViewId),
            views: vs.map(v => ({
                id: v.id,
                name: v.name,
                positionIds: (v.positions || []).map(p => p.id).filter(id => id != null),
                positionIndex: v.positionIndex || 0,
                selectedMove: v.selectedMove,
                activeTab: v.activeTab || 'analysis',
                commentText: v.commentText || '',
                mode: v.mode || 'NORMAL',
                previousMode: v.previousMode || 'NORMAL'
            }))
        });
    }

    // Restore views from serialized data + a function to load positions by IDs
    async function deserialize(json, loadAllPositionsFn) {
        try {
            const data = JSON.parse(json);
            if (!data || !data.views || data.views.length === 0) return false;

            // Load all positions once for ID lookup
            const allPositions = await loadAllPositionsFn();
            const posMap = new Map(allPositions.map(p => [p.id, p]));

            nextViewId = data.nextViewId || data.views.length + 1;

            const restoredViews = data.views.map(sv => {
                const positions = (sv.positionIds || []).map(id => posMap.get(id)).filter(Boolean);
                return {
                    id: sv.id,
                    name: sv.name,
                    positions,
                    positionIndex: Math.min(sv.positionIndex || 0, Math.max(positions.length - 1, 0)),
                    position: positions[sv.positionIndex] || createDefaultPosition(),
                    analysis: createDefaultAnalysis(),
                    selectedMove: sv.selectedMove ?? null,
                    activeTab: sv.activeTab || 'analysis',
                    commentText: sv.commentText || '',
                    mode: sv.mode || 'NORMAL',
                    previousMode: sv.previousMode || 'NORMAL',
                    matchContext: createDefaultMatchContext()
                };
            });

            views.set(restoredViews);
            const targetId = data.activeViewId || restoredViews[0].id;
            activeViewId.set(targetId);
            const target = restoredViews.find(v => v.id === targetId) || restoredViews[0];
            restoreViewState(target);
            return true;
        } catch (e) {
            console.error('Error deserializing views:', e);
            return false;
        }
    }

    function selectPreviousView() {
        const vs = get(views);
        if (vs.length <= 1) return;
        const currentId = get(activeViewId);
        const idx = vs.findIndex(v => v.id === currentId);
        const prevIdx = idx > 0 ? idx - 1 : vs.length - 1;
        switchTo(vs[prevIdx].id);
    }

    function selectNextView() {
        const vs = get(views);
        if (vs.length <= 1) return;
        const currentId = get(activeViewId);
        const idx = vs.findIndex(v => v.id === currentId);
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
