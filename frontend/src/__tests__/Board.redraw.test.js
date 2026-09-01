/**
 * Board.redraw.test.js
 *
 * Characterization test for fiche-12 (coalesced board redraws). Board.svelte
 * has no rendering test otherwise — it draws through two.js, which needs a
 * real canvas/SVG backend the test environment doesn't provide — so two.js
 * itself is mocked here: every drawBoard() call starts with two.clear()
 * (Board.svelte:~932), so counting clear() calls on the mocked instance is
 * equivalent to counting drawBoard() calls.
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
// makeCircle/makeRectangle/makeLine/makePath on the instance, and sets plain
// properties (fill, stroke, size, …) plus `.translation.set(...)` on whatever
// makeXxx() returns. A bag object with a stub `translation` covers all of it.
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
            twoInstances.push(this);
        }
        appendTo() {
            return this;
        }
        makeGroup() {
            return { add: () => {} };
        }
        makeText() {
            return makeFakeShape();
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
        makePath() {
            return makeFakeShape();
        }
    }
    return { default: FakeTwo };
});

import Board from '../components/Board.svelte';
import { positionStore, emptyPosition } from '../stores/positionStore.js';
import { selectedMoveStore, analysisStore } from '../stores/analysisStore.js';

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
}

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
        // baseline.
        expect(two.clear.mock.calls.length).toBe(0);
        expect(pendingFrames.size).toBe(1);
        flushFrame();
        const baseline = two.clear.mock.calls.length;
        expect(baseline).toBe(1);

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
        expect(two.clear.mock.calls.length).toBe(baseline);

        flushFrame();

        expect(two.clear.mock.calls.length).toBe(baseline + 1);
        expect(pendingFrames.size).toBe(0);
    });

    test('selecting/hovering a move (AnalysisPanel) redraws once per frame, not per keystroke', async () => {
        render(Board);
        await tick();
        await tick();
        flushFrame(); // consume the mount-time coalesced frame

        const two = twoInstances[0];
        const baseline = two.clear.mock.calls.length;

        selectedMoveStore.set('13/7 6/1');
        selectedMoveStore.set('13/7 8/2'); // hovering a different candidate move
        selectedMoveStore.set(null); // mouse leaves the row

        expect(pendingFrames.size).toBe(1);
        expect(two.clear.mock.calls.length).toBe(baseline);

        flushFrame();

        expect(two.clear.mock.calls.length).toBe(baseline + 1);
    });

    test('a burst of resize events redraws at most once per frame', async () => {
        render(Board);
        await tick();
        await tick();
        flushFrame(); // consume the mount-time coalesced frame

        const two = twoInstances[0];
        const baseline = two.clear.mock.calls.length;

        window.dispatchEvent(new Event('resize'));
        window.dispatchEvent(new Event('resize'));
        window.dispatchEvent(new Event('resize'));

        expect(pendingFrames.size).toBe(1);
        expect(two.clear.mock.calls.length).toBe(baseline);

        flushFrame();

        expect(two.clear.mock.calls.length).toBe(baseline + 1);
        expect(pendingFrames.size).toBe(0);
    });

    test('Ctrl-Arrow (board orientation) redraws on the next frame, not inline', async () => {
        render(Board);
        await tick();
        await tick();
        flushFrame(); // consume the mount-time coalesced frame

        const two = twoInstances[0];
        const baseline = two.clear.mock.calls.length;

        window.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowLeft', ctrlKey: true }));
        window.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowRight', ctrlKey: true }));

        expect(two.clear.mock.calls.length).toBe(baseline);
        expect(pendingFrames.size).toBe(1);

        flushFrame();

        expect(two.clear.mock.calls.length).toBe(baseline + 1);
    });

    test('an unmount with a pending frame cancels it (no draw on a detached board)', async () => {
        const { unmount } = render(Board);
        await tick();
        await tick();
        flushFrame(); // consume the mount-time coalesced frame

        const two = twoInstances[0];

        positionStore.update((p) => ({ ...p, id: 99 }));
        expect(pendingFrames.size).toBe(1);

        unmount();
        const drawsAtUnmount = two.clear.mock.calls.length;

        // Flushing the frame after teardown must not draw again.
        flushFrame();
        expect(two.clear.mock.calls.length).toBe(drawsAtUnmount);
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
        render(Board);
        await tick();
        await tick();
        flushFrame();
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
