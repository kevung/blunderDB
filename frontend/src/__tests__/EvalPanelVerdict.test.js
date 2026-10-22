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

vi.mock('../../wailsjs/go/gui/App.js', () => ({
    EvaluatePositionImmediate: (/** @type {any[]} */ ...args) => evaluatePositionImmediate(...args),
    StartEvaluationAtRest: vi.fn().mockResolvedValue(undefined),
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
    EventsOn: vi.fn(() => () => {})
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
});
