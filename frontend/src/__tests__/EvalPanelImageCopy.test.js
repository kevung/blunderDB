/**
 * In EVAL mode analysisStore describes another position: the image copy
 * (C-X C-X) reads the panel's live evaluation through evalAnalysisStore.
 */
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import { get } from 'svelte/store';

const evaluatePositionImmediate = vi.fn();

vi.mock('../../wailsjs/go/gui/App.js', () => ({
    EvaluatePositionImmediate: (...args) => evaluatePositionImmediate(...args),
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
import { evalAnalysisStore } from '../stores/analysisStore.js';
import { analysisStrip } from '../services/clipboardService.js';
import EvalPanel from '../components/EvalPanel.svelte';

const move = { index: 0, move: '13/7 8/5', analysisDepth: '0-ply', equity: 0.12, playerWinChance: 55, opponentWinChance: 45 };
const cube = {
    analysisDepth: '0-ply',
    playerWinChances: 70,
    playerGammonChances: 20,
    playerBackgammonChances: 1,
    opponentWinChances: 30,
    opponentGammonChances: 5,
    opponentBackgammonChances: 0,
    cubefulNoDoubleEquity: 0.8,
    cubefulDoubleTakeEquity: 1.1,
    cubefulDoublePassEquity: 1.0
};

describe('EvalPanel hands its evaluation to the image copy', () => {
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

    test('checker decision: the candidate moves', async () => {
        evaluatePositionImmediate.mockResolvedValue({ moves: [move] });
        positionStore.set({ ...emptyPosition(), dice: [3, 3] });
        render(EvalPanel);
        await vi.advanceTimersByTimeAsync(0);

        const analysis = get(evalAnalysisStore);
        expect(analysis.analysisType).toBe('CheckerMove');
        expect(analysis.checkerAnalysis.moves).toEqual([move]);
        expect(analysisStrip(analysis)).toEqual({ kind: 'checker', rows: 2 });
    });

    test('cube decision: the record and the panel decision', async () => {
        evaluatePositionImmediate.mockResolvedValue({ cube, cubeVerdict: 'double_take' });
        positionStore.set({ ...emptyPosition(), dice: [0, 0] });
        render(EvalPanel);
        await vi.advanceTimersByTimeAsync(0);

        const analysis = get(evalAnalysisStore);
        expect(analysis.analysisType).toBe('DoublingCube');
        expect(analysis.doublingCubeAnalysis.playerWinChances).toBe(70);
        expect(analysis.decision.state).not.toBe('pending');
        expect(analysisStrip(analysis)).toEqual({ kind: 'cube', rows: 6 });
    });

    test('leaving the panel withdraws the evaluation', async () => {
        evaluatePositionImmediate.mockResolvedValue({ moves: [move] });
        positionStore.set({ ...emptyPosition(), dice: [3, 3] });
        render(EvalPanel);
        await vi.advanceTimersByTimeAsync(0);
        statusBarModeStore.set('NORMAL');
        await vi.advanceTimersByTimeAsync(0);

        expect(get(evalAnalysisStore)).toBeNull();
    });
});
