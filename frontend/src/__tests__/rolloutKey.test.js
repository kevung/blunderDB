import { describe, test, expect, vi, afterEach, beforeEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

const db = vi.hoisted(() => ({ LoadAnalysis: vi.fn(() => Promise.resolve(null)), LoadRollouts: vi.fn(() => Promise.resolve([])) }));
vi.mock('../../wailsjs/go/database/Database.js', () => db);

const roll = vi.hoisted(() => ({ toggleRollout: vi.fn() }));
vi.mock('../services/rolloutService.js', async (orig) => ({ ...(await orig()), toggleRollout: roll.toggleRollout }));

import { analysisStore } from '../stores/analysisStore.js';
import { positionStore, emptyPosition } from '../stores/positionStore.js';
import AnalysisPanel from '../components/AnalysisPanel.svelte';

// `r` is the random position everywhere; the rollout is Shift+R.
describe('AnalysisPanel — the rollout key', () => {
    beforeEach(() => {
        roll.toggleRollout.mockClear();
        positionStore.set({ ...emptyPosition(), id: 7, dice: [3, 1] });
        analysisStore.set(/** @type {any} */ ({ checkerAnalysis: { moves: [{ move: '8/5 6/5', equity: 0.1 }] }, analysisType: 'CheckerMove' }));
    });
    afterEach(() => cleanup());

    test('r leaves the event to the global dispatcher (random position)', async () => {
        const { container } = render(AnalysisPanel);
        await tick();
        const panel = /** @type {HTMLElement} */ (container.querySelector('#analysisPanel'));
        const ev = new KeyboardEvent('keydown', { key: 'r', bubbles: true, cancelable: true });
        panel.dispatchEvent(ev);
        expect(roll.toggleRollout).not.toHaveBeenCalled();
        expect(ev.defaultPrevented).toBe(false);
    });

    test('Shift+R starts the rollout', async () => {
        const { container } = render(AnalysisPanel);
        await tick();
        const panel = /** @type {HTMLElement} */ (container.querySelector('#analysisPanel'));
        await fireEvent.keyDown(panel, { key: 'R', shiftKey: true });
        expect(roll.toggleRollout).toHaveBeenCalledTimes(1);
    });
});
