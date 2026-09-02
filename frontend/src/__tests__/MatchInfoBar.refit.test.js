/**
 * MatchInfoBar.refit.test.js
 *
 * #201 (D.1): the match-info bar is inserted above the board when a match is
 * reviewed and removed when it ends, but Board.svelte only re-measures its
 * container on a window 'resize' — so the board stayed 22 px too tall for a
 * whole review. Locks the fix: each flip of the bar's visibility dispatches a
 * synthetic 'resize' on the next animation frame.
 *
 * requestAnimationFrame is stubbed with a manually flushed queue (the recipe
 * of Board.redraw.test.js) so the assertions are deterministic.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import { tick } from 'svelte';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    GetMatchByID: vi.fn(() => Promise.resolve({ id: 7, player1_name: 'Alice', player2_name: 'Bob', match_length: 7 })),
    GetPositionProvenance: vi.fn(() => Promise.resolve([]))
}));

import MatchInfoBar from '../components/MatchInfoBar.svelte';
import { matchContextStore, positionStore } from '../stores/positionStore.js';

const NO_MATCH = { isMatchMode: false, matchID: null, movePositions: [], currentIndex: 0, player1Name: '', player2Name: '' };
const IN_MATCH = { ...NO_MATCH, isMatchMode: true, matchID: 7, player1Name: 'Alice', player2Name: 'Bob' };

let pendingFrames;
function flushFrames() {
    const frames = [...pendingFrames];
    pendingFrames = [];
    for (const cb of frames) cb();
}

let resizes;
const onResize = () => resizes++;

beforeEach(() => {
    pendingFrames = [];
    vi.stubGlobal('requestAnimationFrame', (cb) => pendingFrames.push(cb));
    resizes = 0;
    window.addEventListener('resize', onResize);
    matchContextStore.set({ ...NO_MATCH });
    positionStore.set(null);
});

afterEach(() => {
    cleanup();
    window.removeEventListener('resize', onResize);
    vi.unstubAllGlobals();
});

describe('MatchInfoBar — the board is re-fitted when the bar appears or disappears', () => {
    test('entering and leaving a match each dispatch one resize after a frame', async () => {
        const { container } = render(MatchInfoBar);
        await tick();
        flushFrames(); // the mount-time run, harmless (the board measures itself once more)
        resizes = 0;

        matchContextStore.set({ ...IN_MATCH });
        await tick();
        expect(container.querySelector('[data-testid="match-info-bar"]'), 'the bar is in the DOM').not.toBeNull();
        expect(resizes, 'not before the frame: the layout must have reflowed').toBe(0);
        flushFrames();
        expect(resizes).toBe(1);

        matchContextStore.set({ ...NO_MATCH });
        await tick();
        expect(container.querySelector('[data-testid="match-info-bar"]')).toBeNull();
        flushFrames();
        expect(resizes).toBe(2);
    });

    test('navigating within the match (bar still visible) dispatches nothing', async () => {
        render(MatchInfoBar);
        matchContextStore.set({ ...IN_MATCH, currentIndex: 0 });
        await tick();
        flushFrames();
        resizes = 0;

        matchContextStore.set({ ...IN_MATCH, currentIndex: 1 });
        await tick();
        flushFrames();
        expect(resizes).toBe(0);
    });
});
