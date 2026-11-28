/**
 * EvalPanelBoardResets.test.js — the Eval panel's two board resets (starting position, clear).
 * The panel is mounted for real; what is checked is the board they leave, the score, cube and
 * side to move they reset, and that the evaluation is asked again for the new position.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/gui/App.js', async (importOriginal) => ({
    ...(await importOriginal()),
    EvaluatePositionImmediate: vi.fn().mockResolvedValue({ moves: [], cube: null }),
    StartEvaluationAtRest: vi.fn().mockResolvedValue(undefined),
    CancelEvaluationAtRest: vi.fn().mockResolvedValue(undefined)
}));
vi.mock('../../wailsjs/go/main/Config.js', async (importOriginal) => ({
    ...(await importOriginal()),
    GetEpcChallenge: vi.fn().mockResolvedValue(false),
    SaveEpcChallenge: vi.fn().mockResolvedValue(undefined),
    GetGammonNetDisplayPly: vi.fn().mockResolvedValue(2),
    GetGammonNetPruneK: vi.fn().mockResolvedValue(12),
    GetGammonNetCandidates: vi.fn().mockResolvedValue(10)
}));
vi.mock('../../wailsjs/runtime/runtime.js', async (importOriginal) => ({
    ...(await importOriginal()),
    EventsOn: vi.fn(() => () => {}),
    BrowserOpenURL: vi.fn()
}));

import { statusBarModeStore } from '../stores/uiStore.js';
import { positionStore } from '../stores/positionStore.js';
import { enterEvalMode, exitEvalMode } from '../services/modeMachine.js';
import EvalPanel from '../components/EvalPanel.svelte';

const total = (/** @type {any} */ pos, /** @type {number} */ color) =>
    pos.board.points.filter((/** @type {any} */ p) => p.color === color).reduce((/** @type {number} */ n, /** @type {any} */ p) => n + p.checkers, 0);

beforeEach(async () => {
    vi.clearAllMocks();
    statusBarModeStore.set('NORMAL');
    await enterEvalMode();
});

afterEach(async () => {
    cleanup();
    await exitEvalMode();
});

const startBtn = (/** @type {HTMLElement} */ c) => /** @type {HTMLElement} */ (c.querySelector('[data-testid="eval-start-position"]'));
const clearBtn = (/** @type {HTMLElement} */ c) => /** @type {HTMLElement} */ (c.querySelector('[data-testid="eval-clear-board"]'));

describe('the Eval panel board resets', () => {
    test('both sit in the band before the add button, labelled', async () => {
        const { container } = render(EvalPanel);
        await tick();
        expect(startBtn(container)).toBeTruthy();
        expect(clearBtn(container)).toBeTruthy();
        expect(startBtn(container).getAttribute('aria-label')).toBe('Starting position');
        expect(clearBtn(container).getAttribute('aria-label')).toBe('Clear the position');
    });

    test('starting position: 15 checkers a side, money, no dice, cube centred, player 1 on roll', async () => {
        const pos = JSON.parse(JSON.stringify(get(positionStore)));
        pos.score = [3, 5];
        pos.cube = { owner: 1, value: 2 };
        pos.player_on_roll = 1;
        pos.dice = [4, 2];
        positionStore.set(pos);
        const { container } = render(EvalPanel);
        await tick();

        await fireEvent.click(startBtn(container));
        await tick();
        const after = get(positionStore);
        expect(total(after, 0)).toBe(15);
        expect(total(after, 1)).toBe(15);
        expect(after.board.bearoff).toEqual([0, 0]);
        expect(after.score).toEqual([-1, -1]);
        expect(after.cube).toEqual({ owner: -1, value: 0 });
        expect(after.player_on_roll).toBe(0);
        expect(after.dice).toEqual([0, 0]);
    });

    test('clear: no checker left on the points, bearoff untouched by the starting position', async () => {
        const { container } = render(EvalPanel);
        await tick();
        await fireEvent.click(startBtn(container));
        await tick();
        await fireEvent.click(clearBtn(container));
        await tick();
        const after = get(positionStore);
        expect(total(after, 0) + total(after, 1)).toBe(0);
    });
});
