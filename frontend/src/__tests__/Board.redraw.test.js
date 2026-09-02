/**
 * Board.redraw.test.js
 *
 * Characterization test for fiche-12 (coalesced board redraws) and for the
 * static/dynamic split of the scene. Board.svelte has no rendering test
 * otherwise — it draws through two.js, which needs a real canvas/SVG backend
 * the test environment doesn't provide — so two.js itself is mocked here.
 *
 * Two counters on the mocked instance stand for two different facts:
 *   two.update()  — every drawBoard() ends with it: one call = one paint.
 *   two.clear()   — only rebuildStaticLayers() calls it: one call = the
 *                   static layer (24 triangles, 24 labels, bar, outline)
 *                   was thrown away and recreated.
 * Before the split, drawBoard() itself started with two.clear() and this
 * file counted clear() as "one paint"; a paint no longer implies a rebuild,
 * which is the whole point of the split, so the expectations below
 * distinguish the two.
 *
 * Before scheduleRedraw(), a position navigation touched positionStore,
 * selectedMoveStore and analysisStore in the same tick, and each one called
 * drawBoard() directly — 3+ full two.js scene rebuilds for one navigation.
 * This test pins the fixed behaviour: any number of store writes within a
 * tick collapse into exactly one requestAnimationFrame request and exactly
 * one drawBoard() call when that frame fires. Same for a burst of window
 * 'resize' events.
 *
 * requestAnimationFrame is stubbed with a manually-flushed queue instead of
 * relying on jsdom's own rAF timing, so the assertions are deterministic.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import { tick } from 'svelte';
import { get } from 'svelte/store';

// ── two.js mock ──────────────────────────────────────────────────────────────
// Board.svelte only ever calls appendTo/clear/update/makeGroup/makeText/
// makeCircle/makeRectangle/makeLine/makePath on the instance, sets plain
// properties (fill, stroke, size, …) plus `.translation.set(...)` on whatever
// makeXxx() returns, and add()/remove()/children on groups. A bag object with
// a stub `translation` covers the shapes; groups keep a real children list so
// "empty the dynamic layer" is exercised, not stubbed away.
const twoInstances = [];

vi.mock('two.js', () => {
    function makeFakeShape() {
        return { translation: { set: () => {} } };
    }
    class FakeTwo {
        constructor(params) {
            this.width = params?.width ?? 0;
            this.height = params?.height ?? 0;
            this.renderer = { setSize: vi.fn() };
            this.clear = vi.fn();
            this.update = vi.fn();
            this.makePath = vi.fn(makeFakeShape);
            this.makeText = vi.fn(makeFakeShape);
            this.groups = []; // in creation order: static, dynamic, frame (see Board.svelte)
            twoInstances.push(this);
        }
        appendTo() {
            return this;
        }
        makeGroup() {
            const group = {
                children: [],
                add(shape) {
                    group.children.push(shape);
                },
                remove(shapes) {
                    for (const s of [...shapes]) group.children.splice(group.children.indexOf(s), 1);
                }
            };
            this.groups.push(group);
            return group;
        }
        makeCircle() {
            return makeFakeShape();
        }
        makeRectangle() {
            return makeFakeShape();
        }
        makeLine() {
            return makeFakeShape();
        }
    }
    return { default: FakeTwo };
});

import Board from '../components/Board.svelte';
import { positionStore, emptyPosition } from '../stores/positionStore.js';
import { selectedMoveStore, analysisStore } from '../stores/analysisStore.js';
import { boardColorsStore, DEFAULT_BOARD_COLORS } from '../stores/boardColorsStore.js';

// ── Deterministic requestAnimationFrame ─────────────────────────────────────
// Captures scheduled callbacks instead of really waiting a frame; flushFrame()
// runs (and clears) whatever is pending, mirroring "the browser paints".
// cancelAnimationFrame actually removes the callback (a real browser would),
// so this also exercises Board.svelte's onDestroy cancellation, not just a
// no-op stub that would hide a stale-draw-after-unmount bug.
let pendingFrames = new Map();
let nextFrameId = 0;
function stubRaf() {
    pendingFrames = new Map();
    nextFrameId = 0;
    vi.stubGlobal('requestAnimationFrame', (cb) => {
        const id = ++nextFrameId;
        pendingFrames.set(id, cb);
        return id;
    });
    vi.stubGlobal('cancelAnimationFrame', (id) => {
        pendingFrames.delete(id);
    });
}
function flushFrame() {
    const toRun = Array.from(pendingFrames.values());
    pendingFrames.clear();
    toRun.forEach((cb) => cb());
}

function resetStores() {
    positionStore.set(emptyPosition());
    selectedMoveStore.set(null);
    analysisStore.update((a) => ({ ...a, positionId: null }));
    boardColorsStore.set({ ...DEFAULT_BOARD_COLORS }); // not resetBoardColors(): it persists through Wails
}

/** Mount, consume the mount-time coalesced frame, return the mocked Two. */
async function mountBoard() {
    const rendered = render(Board);
    await tick();
    await tick();
    flushFrame();
    return { ...rendered, two: twoInstances[0] };
}

const paints = (two) => two.update.mock.calls.length;
const rebuilds = (two) => two.clear.mock.calls.length;

describe('Board — redraws are coalesced to one drawBoard() per frame (fiche-12)', () => {
    beforeEach(() => {
        resetStores();
        stubRaf();
    });
    afterEach(() => {
        cleanup();
        twoInstances.length = 0;
        vi.unstubAllGlobals();
    });

    test('a position navigation touching 3 stores draws exactly once, on the next frame', async () => {
        render(Board);
        await tick();
        await tick();

        const two = twoInstances[0];
        expect(two).toBeTruthy();

        // Mount does not draw synchronously: the redraw triggers' subscribe()
        // fire once immediately (svelte/store invokes the callback
        // synchronously on subscription) and that schedules the one coalesced
        // frame the first paint rides on. Flush it so we start from a clean
        // baseline: one paint, which also built the static layer.
        expect(paints(two)).toBe(0);
        expect(pendingFrames.size).toBe(1);
        flushFrame();
        expect(paints(two)).toBe(1);
        expect(rebuilds(two)).toBe(1);

        // Simulate a position navigation the way positionService.showPosition()
        // drives it: position, selected move and analysis all update in the
        // same synchronous pass.
        const next = emptyPosition();
        next.id = 42;
        positionStore.set(next);
        selectedMoveStore.set('24/18');
        analysisStore.update((a) => ({ ...a, positionId: 42 }));

        // Nothing has painted yet — exactly one frame is queued.
        expect(pendingFrames.size).toBe(1);
        expect(paints(two)).toBe(1);

        flushFrame();

        expect(paints(two)).toBe(2);
        expect(pendingFrames.size).toBe(0);
        // Same size, orientation, palette and perspective: the static layer
        // survived the navigation.
        expect(rebuilds(two)).toBe(1);
    });

    test('selecting/hovering a move (AnalysisPanel) redraws once per frame, not per keystroke', async () => {
        const { two } = await mountBoard();
        const baseline = paints(two);

        selectedMoveStore.set('13/7 6/1');
        selectedMoveStore.set('13/7 8/2'); // hovering a different candidate move
        selectedMoveStore.set(null); // mouse leaves the row

        expect(pendingFrames.size).toBe(1);
        expect(paints(two)).toBe(baseline);

        flushFrame();

        expect(paints(two)).toBe(baseline + 1);
    });

    test('a burst of resize events redraws at most once per frame', async () => {
        const { two } = await mountBoard();
        const baseline = paints(two);

        window.dispatchEvent(new Event('resize'));
        window.dispatchEvent(new Event('resize'));
        window.dispatchEvent(new Event('resize'));

        expect(pendingFrames.size).toBe(1);
        expect(paints(two)).toBe(baseline);

        flushFrame();

        expect(paints(two)).toBe(baseline + 1);
        expect(pendingFrames.size).toBe(0);
    });

    test('Ctrl-Arrow (board orientation) redraws on the next frame, not inline', async () => {
        const { two } = await mountBoard();
        const baseline = paints(two);

        window.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowLeft', ctrlKey: true }));
        window.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowRight', ctrlKey: true }));

        expect(paints(two)).toBe(baseline);
        expect(pendingFrames.size).toBe(1);

        flushFrame();

        expect(paints(two)).toBe(baseline + 1);
    });

    test('an unmount with a pending frame cancels it (no draw on a detached board)', async () => {
        const { two, unmount } = await mountBoard();

        positionStore.update((p) => ({ ...p, id: 99 }));
        expect(pendingFrames.size).toBe(1);

        unmount();
        const drawsAtUnmount = paints(two);

        // Flushing the frame after teardown must not draw again.
        flushFrame();
        expect(paints(two)).toBe(drawsAtUnmount);
    });
});

// The scene is two two.js groups: a static layer (24 triangles, 24 labels,
// bar, outline) rebuilt only when the geometry, the palette or the label side
// changes, and a dynamic layer emptied and refilled on every paint. A hover
// in the analysis panel — the most frequent redraw trigger — must touch the
// dynamic layer only.
describe('Board — the static layer survives a redraw', () => {
    beforeEach(() => {
        resetStores();
        stubRaf();
    });
    afterEach(() => {
        cleanup();
        twoInstances.length = 0;
        vi.unstubAllGlobals();
    });

    test('hovering a move repaints without rebuilding the static layer', async () => {
        const { two } = await mountBoard();
        expect(rebuilds(two)).toBe(1);
        const [staticLayer] = two.groups;
        const pathsBefore = two.makePath.mock.calls.length;
        const textsBefore = two.makeText.mock.calls.length;

        selectedMoveStore.set('13/7 6/1');
        flushFrame();

        expect(paints(two)).toBe(2);
        expect(rebuilds(two)).toBe(1); // no two.clear(): triangles and labels kept
        expect(two.groups).toHaveLength(3); // same three groups, none recreated
        expect(staticLayer.children).toHaveLength(49);
        // Nothing triangle-shaped was created: a static rebuild would have added
        // 24 triangles and 24 labels. (The arrowheads themselves are makePath
        // calls too, but jsdom gives the board no size, every slot collapses to
        // one point and a zero-length arrow is skipped — boardScene.test.js
        // covers the arrows with real dimensions.)
        expect(two.makePath.mock.calls.length - pathsBefore).toBe(0);
        expect(two.makeText.mock.calls.length - textsBefore).toBeLessThan(24);
    });

    test('the dynamic layer is emptied before it is refilled (no pile-up across redraws)', async () => {
        const { two } = await mountBoard();
        const [staticLayer, dynamicLayer, frameLayer] = two.groups;
        expect(two.groups).toHaveLength(3);
        expect(staticLayer.children).toHaveLength(24 + 24 + 1); // triangles, labels, bar
        expect(frameLayer.children).toHaveLength(1);

        const position = emptyPosition();
        position.id = 1;
        position.board.points[6] = { checkers: 5, color: 0 };
        positionStore.set(position);
        flushFrame();
        const filled = dynamicLayer.children.length;
        expect(filled).toBeGreaterThan(5); // the five checkers, at least

        // The same position painted twice more: same group, same size.
        positionStore.set({ ...position });
        flushFrame();
        positionStore.set({ ...position });
        flushFrame();
        expect(two.groups).toHaveLength(3);
        expect(dynamicLayer.children).toHaveLength(filled);
        expect(staticLayer.children).toHaveLength(49);
        expect(rebuilds(two)).toBe(1);
    });

    test.each([
        ['a resize', () => window.dispatchEvent(new Event('resize'))],
        ['an orientation change', () => window.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowLeft', ctrlKey: true }))],
        ['a palette change', () => boardColorsStore.update((c) => ({ ...c, point1: '#123456' }))]
    ])('%s rebuilds the static layer once', async (_label, trigger) => {
        const { two } = await mountBoard();
        expect(rebuilds(two)).toBe(1);

        trigger();
        await tick(); // the palette goes through an $effect, which flushes in a microtask
        flushFrame();

        expect(paints(two)).toBe(2);
        expect(rebuilds(two)).toBe(2);
    });

    test("switching to player 2's perspective renumbers the labels, hence rebuilds", async () => {
        const { two } = await mountBoard();
        const { statusBarModeStore } = await import('../stores/uiStore.js');
        statusBarModeStore.set('EDIT'); // the display position is shown as-is
        flushFrame();
        const before = rebuilds(two);

        positionStore.update((p) => ({ ...p, player_on_roll: 1 }));
        flushFrame();
        expect(rebuilds(two)).toBe(before + 1);

        // Same perspective again: no further rebuild.
        positionStore.update((p) => ({ ...p, dice: [6, 6] }));
        flushFrame();
        expect(rebuilds(two)).toBe(before + 1);
        statusBarModeStore.set('NORMAL');
    });
});

// The positionStore subscription in Board.svelte carries one business rule
// next to the redraw: the move picked in the analysis panel is dropped only on
// a *real* navigation, i.e. when the position id changes. Every other tick of
// the store (a board edit, an analysis refresh re-setting the same position)
// must leave it alone — otherwise the arrows the user asked for vanish on the
// first unrelated store write.
describe('Board — selectedMoveStore is reset only when the position id changes', () => {
    beforeEach(() => {
        resetStores();
        stubRaf();
    });
    afterEach(() => {
        cleanup();
        twoInstances.length = 0;
        vi.unstubAllGlobals();
    });

    async function mountOnPosition(id) {
        await mountBoard();
        const position = emptyPosition();
        position.id = id;
        positionStore.set(position);
        flushFrame();
    }

    test('the same id written again keeps the selected move', async () => {
        await mountOnPosition(42);
        selectedMoveStore.set('24/18 13/7');
        expect(get(selectedMoveStore)).toBe('24/18 13/7');

        // A board edit or an analysis refresh: new object, same position.
        positionStore.update((p) => ({ ...p }));
        positionStore.set({ ...get(positionStore), id: 42 });
        flushFrame();

        expect(get(selectedMoveStore)).toBe('24/18 13/7');
    });

    test('a different id clears the selected move', async () => {
        await mountOnPosition(42);
        selectedMoveStore.set('24/18 13/7');

        positionStore.set({ ...get(positionStore), id: 43 });

        expect(get(selectedMoveStore)).toBeNull();
    });
});
