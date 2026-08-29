/**
 * #125's non-regression criterion: the effect that drives the Eval panel's
 * progressive evaluation (0-ply at the gesture, 2-ply at rest) must depend
 * ONLY on the position (plus whether the panel is even shown) — never on
 * evalMoves/evalCubeAnalysis, which the same effect writes. Reading a $state
 * an effect just wrote is the fcde0243 class of bug
 * (effect_update_depth_exceeded, StatsFilterBar.svelte): once Svelte gives
 * up, nothing on the page updates again. See StatsFilterBarNoDb.test.js for
 * the precedent this test follows.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render } from '@testing-library/svelte';

const evaluatePositionImmediate = vi.fn().mockResolvedValue({ moves: [], cube: null });
const startEvaluationAtRest = vi.fn().mockResolvedValue(undefined);
const cancelEvaluationAtRest = vi.fn().mockResolvedValue(undefined);

vi.mock('../../wailsjs/go/gui/App.js', () => ({
    EvaluatePositionImmediate: (...args) => evaluatePositionImmediate(...args),
    StartEvaluationAtRest: (...args) => startEvaluationAtRest(...args),
    CancelEvaluationAtRest: (...args) => cancelEvaluationAtRest(...args)
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
import EPCPanel from '../components/EPCPanel.svelte';

describe('EPCPanel eval-escalation effect', () => {
    /** @type {string[]} */
    let svelteErrors;
    let restore;

    beforeEach(() => {
        vi.useFakeTimers();
        evaluatePositionImmediate.mockClear();
        startEvaluationAtRest.mockClear();
        cancelEvaluationAtRest.mockClear();
        statusBarModeStore.set('EPC');
        positionStore.set(emptyPosition());
        svelteErrors = [];
        // effect_update_depth_exceeded reaches console.error inside Svelte's
        // flush, not the test body — same technique as StatsFilterBarNoDb.
        const spy = vi.spyOn(console, 'error').mockImplementation((...args) => {
            svelteErrors.push(args.map(String).join(' '));
        });
        restore = () => spy.mockRestore();
    });

    afterEach(async () => {
        vi.runOnlyPendingTimers();
        vi.useRealTimers();
        statusBarModeStore.set('NORMAL');
        restore();
    });

    test('mounting does not loop the escalation effect', async () => {
        render(EPCPanel);
        await vi.advanceTimersByTimeAsync(0);

        expect(svelteErrors.join('\n')).not.toMatch(/effect_update_depth_exceeded/);
        expect(evaluatePositionImmediate).toHaveBeenCalledTimes(1);
    });

    test('rapid position edits re-run 0-ply each time without looping, and debounce the 2-ply search', async () => {
        render(EPCPanel);
        await vi.advanceTimersByTimeAsync(0);

        for (let i = 0; i < 5; i++) {
            positionStore.update((p) => ({ ...p, dice: [(i % 6) + 1, (i % 6) + 1] }));
            await vi.advanceTimersByTimeAsync(50); // well under the 500ms rest delay
        }

        expect(svelteErrors.join('\n')).not.toMatch(/effect_update_depth_exceeded/);
        // Six distinct positions (initial + 5 edits) → six 0-ply calls, but the
        // 2-ply search never got 500ms of rest, so it never started.
        expect(evaluatePositionImmediate).toHaveBeenCalledTimes(6);
        expect(startEvaluationAtRest).not.toHaveBeenCalled();
        // Every edit after the first cancels whatever was in flight.
        expect(cancelEvaluationAtRest).toHaveBeenCalledTimes(6);
    });

    test('500ms of rest starts exactly one 2-ply search at the configured depth', async () => {
        render(EPCPanel);
        await vi.advanceTimersByTimeAsync(0);
        evaluatePositionImmediate.mockClear();
        cancelEvaluationAtRest.mockClear();

        positionStore.update((p) => ({ ...p, dice: [6, 5] }));
        await vi.advanceTimersByTimeAsync(500);

        expect(svelteErrors.join('\n')).not.toMatch(/effect_update_depth_exceeded/);
        expect(startEvaluationAtRest).toHaveBeenCalledTimes(1);
        expect(startEvaluationAtRest.mock.calls[0][1]).toBe(2); // GetGammonNetDisplayPly() mocked to 2
    });
});
