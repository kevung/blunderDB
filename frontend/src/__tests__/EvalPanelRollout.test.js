/**
 * Les rollouts du panneau Eval : mêmes gestes que le panneau Analyse (sélection, menu contextuel,
 * `r`, Échap, barre de progression, colonne Rollout), mais le plateau brouillon n'est jamais
 * enregistré : le rollout tourne en mémoire, sans base ni droit d'écriture, et son résultat ne
 * s'affiche que tant que ce plateau est à l'écran.
 */
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, cleanup, fireEvent, waitFor } from '@testing-library/svelte';
import { get } from 'svelte/store';

const MOVES = vi.hoisted(() => [
    { move: '8/5 6/5', equity: 0.1, equityError: 0 },
    { move: '24/21 13/11', equity: 0.05, equityError: -0.05 },
    { move: '13/10 13/11', equity: 0.0, equityError: -0.1 }
]);
const FAST = { truncation: 7, min_games: 108, max_games: 216, jsd_limit: 3, ply: 0, candidates: 5, seed: 1, workers: 0 };
const STANDARD = { truncation: 11, min_games: 324, max_games: 1296, jsd_limit: 3, ply: 0, candidates: 5, seed: 1, workers: 0 };

/** @type {Record<string, (e: any) => void>} */
const handlers = {};
/** @type {import('vitest').Mock<(...a: any[]) => Promise<any>>} */
const startRollout = vi.fn(() => Promise.resolve(1));
/** @type {import('vitest').Mock<(...a: any[]) => any>} */
const cancelRollout = vi.fn();
/** @type {import('vitest').Mock<(...a: any[]) => Promise<any>>} */
const rolloutStatus = vi.fn(() => Promise.resolve({ running: false }));
/** @type {import('vitest').Mock<(...a: any[]) => Promise<any>>} */
const evaluate = vi.fn(() => Promise.resolve({ moves: MOVES, preRoll: null }));

vi.mock('../../wailsjs/go/gui/App.js', () => ({
    EvaluatePositionImmediate: (/** @type {any[]} */ ...a) => evaluate(...a),
    StartEvaluationAtRest: vi.fn().mockResolvedValue(undefined),
    CancelEvaluationAtRest: vi.fn().mockResolvedValue(undefined),
    StartRollout: (/** @type {any[]} */ ...a) => startRollout(...a),
    StartRolloutIDs: vi.fn(),
    CountRolloutIDs: vi.fn(),
    CancelRollout: (/** @type {any[]} */ ...a) => cancelRollout(...a),
    RolloutStatus: (/** @type {any[]} */ ...a) => rolloutStatus(...a),
    RolloutPresets: () =>
        Promise.resolve([
            { name: 'fast', settings: FAST },
            { name: 'standard', settings: STANDARD }
        ])
}));
vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetEpcChallenge: vi.fn().mockResolvedValue(false),
    SaveEpcChallenge: vi.fn().mockResolvedValue(undefined),
    GetGammonNetDisplayPly: vi.fn().mockResolvedValue(2),
    GetGammonNetPruneK: vi.fn().mockResolvedValue(12),
    GetGammonNetCandidates: vi.fn().mockResolvedValue(10),
    GetRolloutChoice: vi.fn().mockResolvedValue(null),
    SaveRolloutChoice: vi.fn().mockResolvedValue(undefined)
}));
vi.mock('../../wailsjs/runtime/runtime.js', () => ({
    EventsOn: vi.fn((/** @type {string} */ name, /** @type {(e: any) => void} */ cb) => {
        handlers[name] = cb;
        return () => {};
    }),
    BrowserOpenURL: vi.fn()
}));
/** @type {import('vitest').Mock<(...a: any[]) => any>} */
const copyImage = vi.fn();
vi.mock('../services/clipboardService.js', async (importOriginal) => ({ ...(await importOriginal()), copyBoardWithAnalysisImage: (/** @type {any[]} */ ...a) => copyImage(...a) }));

const { statusBarModeStore } = await import('../stores/uiStore.js');
const { positionStore, emptyPosition } = await import('../stores/positionStore.js');
const { databasePathStore } = await import('../stores/databaseStore.js');
const { selectedMoveStore } = await import('../stores/analysisStore.js');
const { rolloutStore, rolloutChoiceStore, idleRollout } = await import('../stores/rolloutStore.js');
const EvalPanel = (await import('../components/EvalPanel.svelte')).default;

const row = (/** @type {string} */ move) => /** @type {HTMLElement} */ (document.querySelector(`tr[data-move="${move}"]`));
const record = (/** @type {number} */ equity) => ({
    analysisDepth: 'Rollout 216 games',
    signature: 'eval-sig',
    kind: 'moves',
    games: 216,
    stop: 'jsd',
    candidates: [{ move: '24/21 13/11', equity, stdErr: 0.004, ci95: 0.008, games: 216, jsd: 0 }]
});

async function mount() {
    const view = render(EvalPanel);
    await waitFor(() => expect(row('8/5 6/5')).toBeTruthy());
    return view;
}

beforeEach(() => {
    vi.clearAllMocks();
    statusBarModeStore.set('EVAL');
    // A board seeded from the library keeps its id: it is still rolled out as a scratch board.
    positionStore.set({ ...emptyPosition(), id: 42 });
    databasePathStore.set('');
    rolloutStore.set(idleRollout());
    rolloutChoiceStore.set({ preset: 'fast', custom: null });
    selectedMoveStore.set(null);
});
afterEach(() => {
    cleanup();
    statusBarModeStore.set('NORMAL');
    selectedMoveStore.set(null);
});

describe('EvalPanel rollouts', () => {
    test('the menu of a picked play rolls it out in memory, without a database', async () => {
        await mount();
        await fireEvent.click(row('8/5 6/5'));
        await fireEvent.click(row('13/10 13/11'), { ctrlKey: true });
        expect(row('13/10 13/11').classList.contains('selected')).toBe(true);
        await fireEvent.contextMenu(row('13/10 13/11'), { clientX: 5, clientY: 5 });
        expect(screen.getByText('Copy position and analysis')).toBeTruthy();
        expect(screen.getByText('Copy position and selected moves')).toBeTruthy();
        await fireEvent.click(screen.getByText(/Rollout \(Fast\)/));
        await waitFor(() => expect(startRollout).toHaveBeenCalledTimes(1));
        const req = startRollout.mock.calls[0][0];
        expect(req.positionId).toBe(0);
        expect(req.store).toBe(false);
        expect(req.position.dice).toEqual(get(positionStore).dice);
        expect(req.moves).toEqual(['8/5 6/5', '13/10 13/11']);
    });

    test('the copy entries copy the whole evaluation, or the plays selected', async () => {
        await mount();
        await fireEvent.click(row('24/21 13/11'));
        await fireEvent.contextMenu(row('24/21 13/11'), { clientX: 5, clientY: 5 });
        await fireEvent.click(screen.getByText('Copy position and selected moves'));
        expect(copyImage).toHaveBeenLastCalledWith({ moves: ['24/21 13/11'] });
        await fireEvent.contextMenu(row('24/21 13/11'), { clientX: 5, clientY: 5 });
        await fireEvent.click(screen.getByText('Copy position and analysis'));
        expect(copyImage).toHaveBeenLastCalledWith({ moves: [] });
    });

    test('a cube decision offers the rollout and the whole copy only', async () => {
        positionStore.set({ ...emptyPosition(), dice: [0, 0] });
        evaluate.mockResolvedValueOnce({ cube: null, cubeVerdict: '', preRoll: null });
        const { container } = render(EvalPanel);
        await waitFor(() => expect(container.querySelector('.eval-content')).toBeTruthy());
        await fireEvent.contextMenu(/** @type {HTMLElement} */ (container.querySelector('.eval-content')), { clientX: 5, clientY: 5 });
        expect(screen.getByText(/Rollout \(Fast\)/)).toBeTruthy();
        expect(screen.getByText('Copy position and analysis')).toBeTruthy();
        expect(screen.queryByText('Copy position and selected moves')).toBeNull();
    });

    test('r starts, the strip follows, Escape cancels', async () => {
        const { container } = await mount();
        const panel = /** @type {HTMLElement} */ (container.querySelector('.eval-panel'));
        await fireEvent.keyDown(panel, { key: 'r' });
        await waitFor(() => expect(startRollout).toHaveBeenCalledTimes(1));
        expect(startRollout.mock.calls[0][0].moves).toEqual([]);
        handlers['rollout:progress']({ job: 1, positionId: 0, games: 108, maxGames: 216, candidates: [] });
        await waitFor(() => expect(screen.getByTestId('rollout-progress')).toBeTruthy());
        expect(screen.getByTestId('rollout-progress').textContent).not.toContain('another position');
        await fireEvent.keyDown(panel, { key: 'Escape' });
        expect(cancelRollout).toHaveBeenCalled();
    });

    test('the result is read in its row, and goes when the board changes', async () => {
        await mount();
        await fireEvent.keyDown(/** @type {HTMLElement} */ (document.querySelector('.eval-panel')), { key: 'r' });
        await waitFor(() => expect(startRollout).toHaveBeenCalledTimes(1));
        handlers['rollout:done']({ job: 1, positionId: 0, stored: false, record: record(0.321) });
        await waitFor(() => expect(screen.getByTestId('rollout-cell').textContent).toContain('0.321'));
        positionStore.update((p) => ({ ...p, dice: [6, 6] }));
        await waitFor(() => expect(screen.queryByTestId('rollout-cell')).toBeNull());
    });

    test('a rollout of a board since edited shows nothing on the new one', async () => {
        await mount();
        await fireEvent.keyDown(/** @type {HTMLElement} */ (document.querySelector('.eval-panel')), { key: 'r' });
        await waitFor(() => expect(startRollout).toHaveBeenCalledTimes(1));
        positionStore.update((p) => ({ ...p, dice: [5, 2] }));
        handlers['rollout:progress']({ job: 1, positionId: 0, games: 36, maxGames: 216, candidates: record(0.5).candidates });
        await waitFor(() => expect(screen.getByTestId('rollout-progress').textContent).toContain('another position'));
        expect(screen.queryByTestId('rollout-cell')).toBeNull();
        handlers['rollout:done']({ job: 1, positionId: 0, stored: false, record: record(0.5) });
        await waitFor(() => expect(screen.queryByTestId('rollout-progress')).toBeNull());
        expect(screen.queryByTestId('rollout-cell')).toBeNull();
    });
});
