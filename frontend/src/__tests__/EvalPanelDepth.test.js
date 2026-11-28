/**
 * The Eval panel's cache, its own depth and the targeted re-run: a position already evaluated
 * deeper shows that answer again without a search; the depth selector re-runs the position, or
 * only the selected plays, without touching the configuration; each re-run says how far each
 * play moved, and rows from several depths each name theirs.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

const handlers = vi.hoisted(() => /** @type {Record<string, (e: any) => void>} */ ({}));
const at = vi.hoisted(() => (/** @type {string} */ depth, /** @type {[string, number][]} */ rows) => rows.map(([move, equity]) => ({ move, equity, analysisDepth: depth })));

const evaluatePositionImmediate = vi.fn();
const startEvaluationAtRest = vi.fn().mockResolvedValue(undefined);
const startPlaysEvaluationAtRest = vi.fn().mockResolvedValue(undefined);
const saveDisplayPly = vi.fn();

vi.mock('../../wailsjs/go/gui/App.js', () => ({
    EvaluatePositionImmediate: (/** @type {any[]} */ ...a) => evaluatePositionImmediate(...a),
    StartEvaluationAtRest: (/** @type {any[]} */ ...a) => startEvaluationAtRest(...a),
    StartPlaysEvaluationAtRest: (/** @type {any[]} */ ...a) => startPlaysEvaluationAtRest(...a),
    CancelEvaluationAtRest: vi.fn().mockResolvedValue(undefined)
}));

vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetEpcChallenge: vi.fn().mockResolvedValue(false),
    SaveEpcChallenge: vi.fn().mockResolvedValue(undefined),
    GetGammonNetDisplayPly: vi.fn().mockResolvedValue(2),
    SaveGammonNetDisplayPly: (/** @type {any[]} */ ...a) => saveDisplayPly(...a),
    GetGammonNetPruneK: vi.fn().mockResolvedValue(12),
    GetGammonNetCandidates: vi.fn().mockResolvedValue(10)
}));

vi.mock('../../wailsjs/runtime/runtime.js', () => ({
    EventsOn: vi.fn((/** @type {string} */ name, /** @type {(e: any) => void} */ fn) => {
        handlers[name] = fn;
        return () => {};
    }),
    BrowserOpenURL: vi.fn()
}));

import { statusBarModeStore } from '../stores/uiStore.js';
import { positionStore, emptyPosition } from '../stores/positionStore.js';
import { selectedMoveStore } from '../stores/analysisStore.js';
import { evalCache } from '../services/evalCache.js';
import EvalPanel from '../components/EvalPanel.svelte';
import { must } from './helpers/must.js';

const ZERO = at('0-ply', [
    ['8/5 6/5', 0.2],
    ['24/21 13/11', 0.1],
    ['13/10 13/11', 0.0]
]);
const TWO = at('2-ply', [
    ['8/5 6/5', 0.15],
    ['24/21 13/11', 0.12],
    ['13/10 13/11', 0.01]
]);

/** Answers the search the last Start* call started. @param {any} result @param {any} start */
async function answer(result, start) {
    const tag = start.mock.calls.at(-1)[4];
    handlers['gammonnet-eval:done']({ ...result, metId: 0, tag });
    await tick();
}

/** @param {HTMLElement} container */
const rowMoves = (container) => [...container.querySelectorAll('tr[data-move]')].map((r) => r.getAttribute('data-move'));

describe('EvalPanel depth, cache and targeted re-run', () => {
    beforeEach(() => {
        vi.useFakeTimers();
        evalCache.clear();
        evaluatePositionImmediate.mockReset().mockResolvedValue({ moves: ZERO, metId: 0 });
        startEvaluationAtRest.mockClear();
        startPlaysEvaluationAtRest.mockClear();
        saveDisplayPly.mockClear();
        statusBarModeStore.set('EVAL');
        positionStore.set(emptyPosition());
        selectedMoveStore.set(null);
    });

    afterEach(() => {
        vi.runOnlyPendingTimers();
        vi.useRealTimers();
        statusBarModeStore.set('NORMAL');
        selectedMoveStore.set(null);
    });

    async function mountAtRest() {
        const view = render(EvalPanel);
        await vi.advanceTimersByTimeAsync(500);
        expect(startEvaluationAtRest).toHaveBeenCalledTimes(1);
        await answer({ moves: TWO }, startEvaluationAtRest);
        return view;
    }

    test('coming back to a position shows its deeper answer, without a search', async () => {
        const { container } = await mountAtRest();
        positionStore.update((p) => ({ ...p, dice: [6, 5] }));
        await vi.advanceTimersByTimeAsync(500);
        expect(startEvaluationAtRest).toHaveBeenCalledTimes(2);

        positionStore.update((p) => ({ ...p, dice: [3, 1] }));
        await vi.advanceTimersByTimeAsync(500);
        expect(startEvaluationAtRest).toHaveBeenCalledTimes(2);
        expect(container.textContent).toContain('2-ply');
        expect(container.textContent).toContain('0.150');
    });

    test('the depth selector re-runs the position at its depth, leaves the configuration, and shows each change', async () => {
        const { container } = await mountAtRest();
        const select = /** @type {HTMLSelectElement} */ (must(container.querySelector('[data-testid="eval-depth"]')));
        expect(select.value).toBe('2');

        await fireEvent.change(select, { target: { value: '3' } });
        await vi.advanceTimersByTimeAsync(0);
        expect(startEvaluationAtRest).toHaveBeenCalledTimes(2);
        expect(startEvaluationAtRest.mock.calls[1][1]).toBe(3);
        expect(saveDisplayPly).not.toHaveBeenCalled();

        await answer(
            {
                moves: at('3-ply', [
                    ['24/21 13/11', 0.14],
                    ['8/5 6/5', 0.13],
                    ['13/10 13/11', 0.0]
                ])
            },
            startEvaluationAtRest
        );
        expect(rowMoves(container)[0]).toBe('24/21 13/11');
        const deltas = [...container.querySelectorAll('[data-testid="row-delta"]')].map((c) => c.textContent);
        expect(deltas).toEqual(['+0.020', '−0.020', '−0.010']);
        expect(select.value).toBe('3');

        // A new position goes back to the configured depth.
        positionStore.update((p) => ({ ...p, dice: [6, 5] }));
        await tick();
        expect(select.value).toBe('2');
        expect(container.querySelector('[data-testid="row-delta"]')).toBeNull();
    });

    test('with plays selected, only those are re-run; the others keep their rows and depths', async () => {
        const { container } = await mountAtRest();
        const rows = container.querySelectorAll('tr[data-move]');
        await fireEvent.click(rows[1], { ctrlKey: true });
        await fireEvent.click(rows[2], { ctrlKey: true });
        const rerun = must(container.querySelector('[data-testid="eval-rerun"]'));
        expect(rerun.textContent).toContain('2');

        const select = /** @type {HTMLSelectElement} */ (must(container.querySelector('[data-testid="eval-depth"]')));
        await fireEvent.change(select, { target: { value: '4' } });
        await vi.advanceTimersByTimeAsync(0);
        expect(startEvaluationAtRest).toHaveBeenCalledTimes(1);
        expect(startPlaysEvaluationAtRest).toHaveBeenCalledTimes(1);
        const [, ply, , plays] = startPlaysEvaluationAtRest.mock.calls[0];
        expect(ply).toBe(4);
        expect(plays).toEqual(['24/21 13/11', '13/10 13/11']);

        await answer(
            {
                partial: true,
                moves: at('4-ply', [
                    ['24/21 13/11', 0.18],
                    ['13/10 13/11', 0.02]
                ])
            },
            startPlaysEvaluationAtRest
        );
        expect(rowMoves(container)).toEqual(['24/21 13/11', '8/5 6/5', '13/10 13/11']);
        const depths = [...container.querySelectorAll('[data-testid="row-depth"]')].map((c) => c.textContent);
        expect(depths).toEqual(['4-ply', '2-ply', '4-ply']);
        const deltas = [...container.querySelectorAll('[data-testid="row-delta"]')].map((c) => c.textContent);
        expect(deltas).toEqual(['+0.060', '', '+0.010']);
        expect(container.textContent).toContain('2–4-ply');
    });

    test('the board reset icons stay beside the depth selector', async () => {
        const { container } = await mountAtRest();
        expect(container.querySelector('[data-testid="eval-start-position"]')).not.toBeNull();
        expect(container.querySelector('[data-testid="eval-clear-board"]')).not.toBeNull();
        expect(container.querySelector('[data-testid="eval-depth"]')).not.toBeNull();
    });
});
