/**
 * viewStore.js orchestrates multiple "views" (each with its own position
 * list/current position/analysis/UI state) sharing the same underlying
 * per-feature stores (positionStore, positionsStore, analysisStore, ...):
 * switching views snapshots the outgoing view's state out of those stores and
 * restores the incoming view's state into them.
 *
 * viewStore is a module-level singleton (`export const viewStore = ...`), so
 * each test gets its own fresh instance via vi.resetModules() + a dynamic
 * re-import — otherwise view/position state would leak between tests.
 *
 * This file is also the regression test for a latent bug the fiche-11 audit
 * flagged: viewStore's own default position (used for the very first view,
 * and as deserialize()'s fallback when a saved position id can no longer be
 * found) used to be a bare array of signed numbers — the classic "-2 0 0 5
 * ..." board notation — while positionStore (and Board.svelte, which reads
 * `point.checkers`/`point.color`) expect an array of {checkers, color}
 * objects. Landing on that default silently produced an undrawable board.
 * The fix replaced the ad hoc default with positionStore's own
 * emptyPosition() factory, used consistently everywhere a "no position yet"
 * value is needed.
 */
import { describe, test, expect, beforeEach, vi } from 'vitest';
import { get } from 'svelte/store';

async function freshViewStore() {
    vi.resetModules();
    const viewStoreMod = await import('../stores/viewStore.js');
    const positionStoreMod = await import('../stores/positionStore.js');
    const analysisStoreMod = await import('../stores/analysisStore.js');
    const uiStoreMod = await import('../stores/uiStore.js');
    return {
        viewStore: viewStoreMod.viewStore,
        positionStore: positionStoreMod.positionStore,
        positionsStore: positionStoreMod.positionsStore,
        emptyPosition: positionStoreMod.emptyPosition,
        analysisStore: analysisStoreMod.analysisStore,
        activeTabStore: uiStoreMod.activeTabStore,
        commentTextStore: uiStoreMod.commentTextStore,
        currentPositionIndexStore: uiStoreMod.currentPositionIndexStore
    };
}

// A minimal stand-in for a domain.Position, shaped like the real thing
// (Board.svelte-compatible points), distinguishable by its id.
function fakePosition(id) {
    return {
        id,
        board: {
            points: Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 })),
            bearoff: [15, 15]
        },
        cube: { owner: -1, value: 0 },
        dice: [3, 1],
        score: [-1, -1],
        player_on_roll: 0,
        decision_type: 0,
        has_jacoby: 0,
        has_beaver: 0
    };
}

describe('viewStore — default position shape (regression)', () => {
    let ctx;
    beforeEach(async () => {
        ctx = await freshViewStore();
    });

    test('the initial view (id 1) has a Board.svelte-shaped position, not a bare number array', async () => {
        const views = get(ctx.viewStore.views);
        expect(views).toHaveLength(1);
        const points = views[0].position.board.points;

        expect(Array.isArray(points)).toBe(true);
        expect(points).toHaveLength(26);
        for (const point of points) {
            expect(typeof point).toBe('object');
            expect(point).not.toBeNull();
            expect(typeof point.checkers).toBe('number');
            expect(typeof point.color).toBe('number');
        }
    });

    test("the initial view's position matches positionStore's own emptyPosition()", async () => {
        const views = get(ctx.viewStore.views);
        expect(views[0].position).toEqual(ctx.emptyPosition());
    });

    test('deserialize() falls back to a Board.svelte-shaped position when a saved position id is missing', async () => {
        // Simulate a session save referencing a position that no longer
        // exists (e.g. it was deleted since the last session).
        const savedJSON = JSON.stringify({
            nextViewId: 2,
            activeViewId: 1,
            views: [
                {
                    id: 1,
                    name: '#1',
                    positionIds: [999], // does not exist
                    positionIndex: 0,
                    selectedMove: null,
                    activeTab: 'matches',
                    commentText: '',
                    mode: 'NORMAL'
                }
            ]
        });
        const loadAll = async () => []; // nothing in the database matches id 999

        const ok = await ctx.viewStore.deserialize(savedJSON, loadAll);
        expect(ok).toBe(true);

        const restoredPosition = get(ctx.positionStore);
        const points = restoredPosition.board.points;
        expect(Array.isArray(points)).toBe(true);
        expect(points).toHaveLength(26);
        for (const point of points) {
            expect(typeof point.checkers).toBe('number');
            expect(typeof point.color).toBe('number');
        }
    });
});

describe('viewStore — snapshot/restore across views', () => {
    let ctx;
    beforeEach(async () => {
        ctx = await freshViewStore();
    });

    test('switching views snapshots the outgoing state and restores the incoming state', async () => {
        // Set up view 1 with a distinctive position.
        ctx.positionStore.set(fakePosition(11));
        ctx.positionsStore.set([fakePosition(11)]);
        ctx.commentTextStore.set('note on view 1');

        ctx.viewStore.addView(); // snapshots view 1, creates+activates view 2 (a copy)

        const views = get(ctx.viewStore.views);
        expect(views).toHaveLength(2);
        // addView() clones the current view's state into the new one.
        expect(get(ctx.positionStore).id).toBe(11);
        expect(get(ctx.commentTextStore)).toBe('note on view 1');

        // Diverge view 2's state.
        ctx.positionStore.set(fakePosition(22));
        ctx.commentTextStore.set('note on view 2');

        const view2Id = get(ctx.viewStore.activeViewId);
        ctx.viewStore.switchTo(1);

        expect(get(ctx.positionStore).id).toBe(11);
        expect(get(ctx.commentTextStore)).toBe('note on view 1');

        ctx.viewStore.switchTo(view2Id);
        expect(get(ctx.positionStore).id).toBe(22);
        expect(get(ctx.commentTextStore)).toBe('note on view 2');
    });

    test('switching to the currently active view is a no-op', async () => {
        ctx.positionStore.set(fakePosition(5));
        const before = get(ctx.viewStore.activeViewId);

        ctx.viewStore.switchTo(before);

        expect(get(ctx.viewStore.activeViewId)).toBe(before);
        expect(get(ctx.positionStore).id).toBe(5);
    });

    test('closeView refuses to close the last remaining view', async () => {
        const onlyId = get(ctx.viewStore.activeViewId);
        ctx.viewStore.closeView(onlyId);

        expect(get(ctx.viewStore.views)).toHaveLength(1);
        expect(get(ctx.viewStore.activeViewId)).toBe(onlyId);
    });

    test('closing the active view falls back to the last remaining view', async () => {
        ctx.viewStore.addView();
        const view2Id = get(ctx.viewStore.activeViewId);
        ctx.positionStore.set(fakePosition(2));

        ctx.viewStore.closeView(view2Id);

        expect(get(ctx.viewStore.views)).toHaveLength(1);
        expect(get(ctx.viewStore.activeViewId)).toBe(1);
        // view 1's state (not view 2's) should now be live.
        expect(get(ctx.positionStore).id).not.toBe(2);
    });

    test('closing a background (non-active) view leaves the active view untouched', async () => {
        ctx.viewStore.addView();
        const view2Id = get(ctx.viewStore.activeViewId);
        ctx.positionStore.set(fakePosition(2));

        ctx.viewStore.switchTo(1);
        ctx.viewStore.closeView(view2Id);

        expect(get(ctx.viewStore.views)).toHaveLength(1);
        expect(get(ctx.viewStore.activeViewId)).toBe(1);
    });

    test('renameView updates only the targeted view', async () => {
        ctx.viewStore.addView();
        const view2Id = get(ctx.viewStore.activeViewId);

        ctx.viewStore.renameView(view2Id, 'Search results');

        const views = get(ctx.viewStore.views);
        const renamed = views.find((v) => v.id === view2Id);
        const untouched = views.find((v) => v.id === 1);
        expect(renamed.name).toBe('Search results');
        expect(untouched.name).toBe('#1');
    });

    test('selectNextView / selectPreviousView cycle through views, wrapping at the ends', async () => {
        ctx.viewStore.addView(); // view 2, now active
        ctx.viewStore.addView(); // view 3, now active
        const ids = get(ctx.viewStore.views).map((v) => v.id);
        expect(ids).toEqual([1, 2, 3]);
        expect(get(ctx.viewStore.activeViewId)).toBe(3);

        ctx.viewStore.selectNextView(); // wraps 3 -> 1
        expect(get(ctx.viewStore.activeViewId)).toBe(1);

        ctx.viewStore.selectPreviousView(); // wraps 1 -> 3
        expect(get(ctx.viewStore.activeViewId)).toBe(3);

        ctx.viewStore.selectPreviousView(); // 3 -> 2
        expect(get(ctx.viewStore.activeViewId)).toBe(2);
    });

    test('selectNextView/selectPreviousView are no-ops with a single view', async () => {
        const onlyId = get(ctx.viewStore.activeViewId);
        ctx.viewStore.selectNextView();
        ctx.viewStore.selectPreviousView();
        expect(get(ctx.viewStore.activeViewId)).toBe(onlyId);
    });
});

describe('viewStore — serialize/deserialize round trip', () => {
    let ctx;
    beforeEach(async () => {
        ctx = await freshViewStore();
    });

    test('serialize/deserialize round-trips positions, tab and comment through position ids', async () => {
        const posA = fakePosition(101);
        const posB = fakePosition(102);
        ctx.positionsStore.set([posA, posB]);
        ctx.positionStore.set(posA);
        ctx.currentPositionIndexStore.set(0);
        ctx.activeTabStore.set('analysis');
        ctx.commentTextStore.set('remember this');

        const json = ctx.viewStore.serialize();
        const parsed = JSON.parse(json);
        expect(parsed.views[0].positionIds).toEqual([101, 102]);
        expect(parsed.views[0].commentText).toBe('remember this');

        // Deserialize into the same fresh instance, resolving ids through a
        // fake "database".
        const loadAll = async () => [posA, posB];
        const ok = await ctx.viewStore.deserialize(json, loadAll);
        expect(ok).toBe(true);

        expect(get(ctx.positionStore).id).toBe(101);
        expect(get(ctx.positionsStore).map((p) => p.id)).toEqual([101, 102]);
        expect(get(ctx.commentTextStore)).toBe('remember this');
    });

    test('deserialize returns false and leaves state untouched on malformed JSON', async () => {
        const before = get(ctx.viewStore.views);
        const ok = await ctx.viewStore.deserialize('{not valid json', async () => []);
        expect(ok).toBe(false);
        expect(get(ctx.viewStore.views)).toBe(before);
    });

    test('deserialize returns false when the saved payload has no views', async () => {
        const ok = await ctx.viewStore.deserialize(JSON.stringify({ views: [] }), async () => []);
        expect(ok).toBe(false);
    });

    test('EPC/EDIT transient modes are not restored: deserialize resets them to NORMAL', async () => {
        const posA = fakePosition(1);
        const json = JSON.stringify({
            nextViewId: 2,
            activeViewId: 1,
            views: [
                {
                    id: 1,
                    name: '#1',
                    positionIds: [1],
                    positionIndex: 0,
                    activeTab: 'epc',
                    commentText: '',
                    mode: 'EPC'
                }
            ]
        });
        // statusBarModeStore lives in uiStore; import it fresh alongside the rest.
        const uiStoreMod = await import('../stores/uiStore.js');

        const ok = await ctx.viewStore.deserialize(json, async () => [posA]);
        expect(ok).toBe(true);
        expect(get(uiStoreMod.statusBarModeStore)).toBe('NORMAL');
    });
});
