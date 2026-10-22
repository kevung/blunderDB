/**
 * EvalPanelVerdict.test.js — the verdict CTRL-S stores beside the board is the
 * engine's own output for the board on screen (evalVerdictStore), never the
 * panel's display shapes.
 */
import { must } from './helpers/must.js';
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import { get } from 'svelte/store';

const evaluatePositionImmediate = vi.fn();
const startEvaluationAtRest = vi.fn().mockResolvedValue(undefined);
/** @type {Record<string, (payload: any) => void>} */
const listeners = {};

vi.mock('../../wailsjs/go/gui/App.js', () => ({
    EvaluatePositionImmediate: (/** @type {any[]} */ ...args) => evaluatePositionImmediate(...args),
    StartEvaluationAtRest: (/** @type {any[]} */ ...args) => startEvaluationAtRest(...args),
    CancelEvaluationAtRest: vi.fn().mockResolvedValue(undefined)
}));

vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetEpcChallenge: vi.fn().mockResolvedValue(false),
    SaveEpcChallenge: vi.fn().mockResolvedValue(undefined),
    GetGammonNetDisplayPly: vi.fn().mockResolvedValue(2),
    GetGammonNetPruneK: vi.fn().mockResolvedValue(12),
    GetGammonNetCandidates: vi.fn().mockResolvedValue(10)
}));

vi.mock('../../wailsjs/runtime/runtime.js', () => ({
    EventsOn: vi.fn((/** @type {string} */ name, /** @type {(payload: any) => void} */ fn) => {
        listeners[name] = fn;
        return () => {};
    })
}));

import { statusBarModeStore } from '../stores/uiStore.js';
import { positionStore, emptyPosition } from '../stores/positionStore.js';
import { evalVerdictStore } from '../stores/analysisStore.js';
import EvalPanel from '../components/EvalPanel.svelte';

const move = { index: 0, move: '13/7 8/5', analysisDepth: '2-ply', equity: 0.12 };
const cube = { analysisDepth: '2-ply', cubefulNoDoubleEquity: 0.8, cubefulDoubleTakeEquity: 1.1, cubefulDoublePassEquity: 1.0 };

describe('EvalPanel exposes its verdict for CTRL-S', () => {
    beforeEach(() => {
        vi.useFakeTimers();
        statusBarModeStore.set('EVAL');
    });

    afterEach(() => {
        cleanup();
        vi.runOnlyPendingTimers();
        vi.useRealTimers();
        statusBarModeStore.set('NORMAL');
    });

    test('dice set: the moves, tagged with the board they describe', async () => {
        evaluatePositionImmediate.mockResolvedValue({ moves: [move] });
        positionStore.set({ ...emptyPosition(), dice: [3, 3] });
        render(EvalPanel);
        await vi.advanceTimersByTimeAsync(0);

        const verdict = must(get(evalVerdictStore));
        expect(verdict.signature).toBe(JSON.stringify(get(positionStore)));
        expect(verdict.analysis).toEqual({ analysisType: 'CheckerMove', checkerAnalysis: { moves: [move] } });
        expect(verdict.metId).toBe(0);
    });

    test('no dice: the cube record as the engine gave it, without the panel facts', async () => {
        evaluatePositionImmediate.mockResolvedValue({ cube, cubeVerdict: 'double_take', preRoll: { playerWinChance: 0.7 } });
        positionStore.set({ ...emptyPosition(), dice: [0, 0] });
        render(EvalPanel);
        await vi.advanceTimersByTimeAsync(0);

        expect(must(get(evalVerdictStore)).analysis).toEqual({ analysisType: 'DoublingCube', doublingCubeAnalysis: cube });
    });

    test('a declined position and a left panel leave nothing to store', async () => {
        evaluatePositionImmediate.mockResolvedValue({ refused: true });
        positionStore.set({ ...emptyPosition(), dice: [0, 0] });
        render(EvalPanel);
        await vi.advanceTimersByTimeAsync(0);
        expect(get(evalVerdictStore)).toBeNull();

        evaluatePositionImmediate.mockResolvedValue({ moves: [move] });
        positionStore.set({ ...emptyPosition(), dice: [3, 3] });
        await vi.advanceTimersByTimeAsync(0);
        expect(get(evalVerdictStore)).not.toBeNull();
        statusBarModeStore.set('NORMAL');
        await vi.advanceTimersByTimeAsync(0);
        expect(get(evalVerdictStore)).toBeNull();
    });

    test('the metId of the result travels with the verdict', async () => {
        evaluatePositionImmediate.mockResolvedValue({ moves: [move], metId: 5 });
        positionStore.set({ ...emptyPosition(), dice: [3, 3] });
        render(EvalPanel);
        await vi.advanceTimersByTimeAsync(0);

        expect(must(get(evalVerdictStore)).metId).toBe(5);
    });

    test('a late "done" from a search a newer gesture superseded is dropped', async () => {
        const stale = { ...move, move: '24/18 13/10' };
        const fresh = { ...move, move: '8/2 6/3' };
        evaluatePositionImmediate.mockResolvedValue({ moves: [move] });
        positionStore.set({ ...emptyPosition(), dice: [3, 3] });
        render(EvalPanel);
        await vi.advanceTimersByTimeAsync(600); // the at-rest search of board A starts
        expect(startEvaluationAtRest).toHaveBeenCalled();
        const staleTag = must(startEvaluationAtRest.mock.calls.at(-1))[4];

        evaluatePositionImmediate.mockResolvedValue({ moves: [fresh] });
        positionStore.set({ ...emptyPosition(), dice: [6, 5] }); // board B
        await vi.advanceTimersByTimeAsync(0);
        const boardB = JSON.stringify(get(positionStore));

        listeners['gammonnet-eval:done']({ moves: [stale], tag: staleTag });
        await vi.advanceTimersByTimeAsync(0);

        let verdict = must(get(evalVerdictStore));
        expect(verdict.signature).toBe(boardB);
        expect(verdict.analysis.checkerAnalysis?.moves).toEqual([fresh]);

        await vi.advanceTimersByTimeAsync(600); // board B's own at-rest search
        const freshTag = must(startEvaluationAtRest.mock.calls.at(-1))[4];
        expect(freshTag).not.toBe(staleTag);
        const deeper = { ...fresh, analysisDepth: '3-ply' };
        listeners['gammonnet-eval:done']({ moves: [deeper], tag: freshTag });
        await vi.advanceTimersByTimeAsync(0);

        verdict = must(get(evalVerdictStore));
        expect(verdict.signature).toBe(boardB);
        expect(verdict.analysis.checkerAnalysis?.moves).toEqual([deeper]);
    });
});
