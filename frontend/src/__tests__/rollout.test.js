/**
 * Les rollouts du panneau Analyse : chaque composant est monté, les appels Wails sont simulés.
 * Vérifie la sélection de coups (Ctrl/Maj+clic), le menu contextuel qui lance ou annule, le
 * réglage gardé dans la configuration, la barre de progression, le résultat porté par la ligne
 * du coup, le refus d'une base en lecture seule et la commande (dont le lot sur la liste).
 */
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, cleanup, fireEvent, waitFor } from '@testing-library/svelte';
import { tick } from 'svelte';
import { get } from 'svelte/store';

const FAST = { truncation: 7, min_games: 108, max_games: 216, jsd_limit: 3, ply: 0, candidates: 5, seed: 1, workers: 0 };
const STANDARD = { truncation: 11, min_games: 324, max_games: 1296, jsd_limit: 3, ply: 0, candidates: 5, seed: 1, workers: 0 };

const handlers = {};
const startRollout = vi.fn(() => Promise.resolve());
const startIDs = vi.fn(() => Promise.resolve());
const cancelRollout = vi.fn();
const countIDs = vi.fn(() => Promise.resolve(42));
const rolloutStatus = vi.fn(() => Promise.resolve({ running: false }));
const loadRollouts = vi.fn(() => Promise.resolve([]));
const confirmAction = vi.fn(() => Promise.resolve(true));

vi.mock('../../wailsjs/go/gui/App.js', () => ({
    StartRollout: (...a) => startRollout(...a),
    StartRolloutIDs: (...a) => startIDs(...a),
    CancelRollout: (...a) => cancelRollout(...a),
    CountRolloutIDs: (...a) => countIDs(...a),
    RolloutStatus: (...a) => rolloutStatus(...a),
    RolloutPresets: () =>
        Promise.resolve([
            { name: 'fast', settings: FAST },
            { name: 'standard', settings: STANDARD }
        ])
}));
vi.mock('../../wailsjs/go/database/Database.js', () => ({ LoadRollouts: (...a) => loadRollouts(...a) }));
const saveChoice = vi.fn(() => Promise.resolve());
const getChoice = vi.fn(() => Promise.resolve({ preset: '', custom: null }));
vi.mock('../../wailsjs/go/main/Config.js', () => ({ GetRolloutChoice: (...a) => getChoice(...a), SaveRolloutChoice: (...a) => saveChoice(...a) }));
vi.mock('../../wailsjs/runtime/runtime.js', () => ({
    EventsOn: vi.fn((name, cb) => {
        handlers[name] = cb;
        return () => {};
    })
}));
const displayedIDs = vi.fn(() => [3, 5, 8]);
vi.mock('../services/modeMachine.js', async (importOriginal) => ({ ...(await importOriginal()), withDisplayedPositionIDs: (run) => run(displayedIDs()) }));
vi.mock('../services/confirmService.js', async (importOriginal) => ({ ...(await importOriginal()), confirmAction: (...a) => confirmAction(...a) }));

const { positionStore } = await import('../stores/positionStore.js');
const { databasePathStore } = await import('../stores/databaseStore.js');
const { rolloutStore, rolloutChoiceStore, idleRollout } = await import('../stores/rolloutStore.js');
const { analysisStore, selectedMoveStore } = await import('../stores/analysisStore.js');
const { statusBarTextStore } = await import('../stores/uiStore.js');
const RolloutResults = (await import('../components/RolloutResults.svelte')).default;
const RolloutSettings = (await import('../components/RolloutSettings.svelte')).default;
const RolloutStrip = (await import('../components/RolloutStrip.svelte')).default;
const AnalysisPanel = (await import('../components/AnalysisPanel.svelte')).default;
const { runRolloutCommand, toggleRollout, initRolloutChoice } = await import('../services/rolloutService.js');
const { canonicalMove } = await import('../utils/moveNotation.js');

const stored = (over = {}) => ({
    analysisEngine: 'blunderDB rollout v1 / gammonNet v1.2.1',
    analysisDepth: 'Rollout 216 games (0-ply, truncated 7)',
    signature: 'sig-A',
    kind: 'moves',
    settings: { jsdLimit: 3 },
    games: 216,
    stop: 'jsd',
    cubefulBias: true,
    exactBearoff: false,
    candidates: [
        { move: '13/7 8/7', equity: 0.123, stdErr: 0.004, ci95: 0.008, games: 216, jsd: 0 },
        { move: '24/18 13/11', equity: 0.05, stdErr: 0.005, ci95: 0.01, games: 108, jsd: 3.4 }
    ],
    ...over
});

beforeEach(() => {
    vi.clearAllMocks();
    rolloutStore.set(idleRollout());
    rolloutChoiceStore.set({ preset: 'standard', custom: null });
    databasePathStore.set('/tmp/x.db');
    positionStore.update((p) => ({ ...p, id: 7 }));
    displayedIDs.mockReturnValue([3, 5, 8]);
    loadRollouts.mockResolvedValue([]);
    rolloutStatus.mockResolvedValue({ running: false });
    analysisStore.set({ analysisType: '', checkerAnalysis: null, doublingCubeAnalysis: null });
    selectedMoveStore.set(null);
});
afterEach(() => cleanup());

const MOVES = [
    { move: '13/7* 8/7', equity: 0.13, equityError: 0, analysisDepth: '3', analysisEngine: 'XG' },
    { move: '24/18 13/11', equity: 0.05, equityError: -0.08, analysisDepth: '3', analysisEngine: 'XG' },
    { move: '13/11 13/7', equity: 0.01, equityError: -0.12, analysisDepth: '3', analysisEngine: 'XG' }
];

function withCheckerAnalysis() {
    analysisStore.set({ analysisType: 'CheckerMove', checkerAnalysis: { moves: MOVES }, doublingCubeAnalysis: null });
}

/** The row of a play in the candidate moves table. */
const row = (move) => /** @type {HTMLElement} */ (document.querySelector(`tr[data-move="${move}"]`));

async function renderPanel() {
    const view = render(AnalysisPanel, { props: { onClose: vi.fn() } });
    await tick();
    return view;
}

describe('canonicalMove', () => {
    test.each([
        ['13/7', '13/8 8/7'],
        ['24/22(2)', '24/23(2) 23/22(2)'],
        ['6/3 3/1', '6/3* 3/1'],
        ['8/5 6/5', '6/5 8/5'],
        ['bar/22 13/11', 'bar/24 24/22 13/11'],
        ['6/off', '6/2 2/off']
    ])('%s and %s are one play', (a, b) => {
        expect(canonicalMove(a)).toBe(canonicalMove(b));
    });
    test('different plays stay different', () => {
        expect(canonicalMove('13/7')).not.toBe(canonicalMove('13/8 6/5'));
    });
});

describe('RolloutResults', () => {
    test('shows equity, 95 % CI and JSD per candidate, and the Configuration', () => {
        render(RolloutResults, { props: { rollouts: [stored()], isMoney: true } });
        expect(screen.getByText('+0.123')).toBeTruthy();
        expect(screen.getAllByText(/Equity \((money|match)\)/).length).toBeGreaterThan(0);
        expect(screen.getByText('±0.008')).toBeTruthy();
        expect(screen.getByText('3.4')).toBeTruthy();
        expect(screen.getByText('sig-A')).toBeTruthy();
        expect(screen.getByText('blunderDB rollout v1 / gammonNet v1.2.1')).toBeTruthy();
        expect(screen.getByTestId('rollout-bias')).toBeTruthy();
    });

    test('a cube rollout names its actions and the verdict', () => {
        const cube = stored({ kind: 'cube', bestCubeAction: 'No double', jsdDouble: 4.2, jsdTake: 0, candidates: [{ move: 'No double', equity: 0.5, stdErr: 0.01, ci95: 0.02, games: 100, jsd: 0 }] });
        render(RolloutResults, { props: { rollouts: [cube] } });
        expect(screen.getByText('No Double')).toBeTruthy();
        expect(screen.getByTestId('rollout-cube-verdict')).toBeTruthy();
    });
});

describe('RolloutSettings', () => {
    test('Fast is chosen and persisted', async () => {
        render(RolloutSettings);
        await fireEvent.click(await screen.findByTestId('rollout-preset-fast'));
        expect(get(rolloutChoiceStore).preset).toBe('fast');
        expect(saveChoice).toHaveBeenCalledWith({ preset: 'fast', custom: null });
    });

    test('Custom exposes every setting and persists the edited values', async () => {
        render(RolloutSettings);
        await waitFor(() => expect(screen.getByText(/1296 games|1296/)).toBeTruthy());
        await fireEvent.click(screen.getByTestId('rollout-preset-custom'));
        for (const key of ['truncation', 'min_games', 'max_games', 'jsd_limit', 'ply', 'candidates', 'seed', 'workers']) {
            expect(screen.getByTestId('rollout-' + key)).toBeTruthy();
        }
        await fireEvent.change(screen.getByTestId('rollout-max_games'), { target: { value: '720' } });
        const last = saveChoice.mock.calls.at(-1)[0];
        expect(last.preset).toBe('custom');
        expect(last.custom.max_games).toBe(720);
        expect(last.custom.seed).toBe(1);
    });

    test('a seed beyond what JavaScript holds exactly is shown, not saved', async () => {
        const { container } = render(RolloutSettings);
        await waitFor(() => expect(screen.getByTestId('rollout-preset-custom')).toBeTruthy());
        await fireEvent.click(screen.getByTestId('rollout-preset-custom'));
        saveChoice.mockClear();
        await fireEvent.change(await screen.findByTestId('rollout-seed'), { target: { value: '18014398509481984' } });
        expect(saveChoice).not.toHaveBeenCalled();
        expect(container.querySelector('.note.error')?.textContent).toMatch(/seed/i);
    });

    test('the persisted choice is read back at start', async () => {
        getChoice.mockResolvedValueOnce({ preset: 'fast', custom: null });
        await initRolloutChoice();
        expect(get(rolloutChoiceStore).preset).toBe('fast');
    });
});

describe('RolloutStrip', () => {
    test('is there only while a rollout runs, with its games and Cancel', async () => {
        render(RolloutStrip, { props: { positionId: 7 } });
        expect(screen.queryByTestId('rollout-progress')).toBeNull();
        const { ensureRolloutEvents } = await import('../services/rolloutService.js');
        ensureRolloutEvents();
        handlers['rollout:progress']({ job: 1, positionId: 7, games: 108, maxGames: 216, candidates: [] });
        await tick();
        expect(screen.getByTestId('rollout-progress').textContent).toContain('108/216');
        await fireEvent.click(screen.getByTestId('rollout-cancel'));
        expect(cancelRollout).toHaveBeenCalled();
        handlers['rollout:cancelled']({ job: 1, positionId: 7, result: null });
        await tick();
        expect(screen.queryByTestId('rollout-progress')).toBeNull();
    });

    test('a batch shows its position count; its end is said in the status bar', async () => {
        render(RolloutStrip, { props: { positionId: 7 } });
        handlers['rollout-batch:started']({ job: 1, total: 5 });
        handlers['rollout-batch:progress']({ job: 1, done: 2, total: 5, positionId: 9, games: 36, maxGames: 216 });
        await tick();
        expect(screen.getByTestId('rollout-progress').textContent).toContain('2');
        handlers['rollout-batch:done']({ job: 1, total: 5, rolledOut: 4, refused: 1, failed: 0, signature: 's' });
        await tick();
        expect(get(statusBarTextStore)).toEqual({ i18nKey: 'rollout.batchDone', i18nParams: { rolledOut: 4, total: 5, refused: 1, failed: 0 } });
        expect(screen.queryByTestId('rollout-progress')).toBeNull();
    });

    test('an error is shown, then goes', async () => {
        rolloutStore.set({ ...idleRollout(), error: 'boom' });
        render(RolloutStrip, { props: { positionId: 7, errorMs: 20 } });
        expect(screen.getByTestId('rollout-error').textContent).toContain('boom');
        await waitFor(() => expect(screen.queryByTestId('rollout-error')).toBeNull());
    });
});

describe('the candidate moves table', () => {
    test('a plain click keeps the single selection', async () => {
        withCheckerAnalysis();
        await renderPanel();
        await fireEvent.click(row('24/18 13/11'));
        expect(get(selectedMoveStore)).toBe('24/18 13/11');
        await fireEvent.click(row('24/18 13/11'));
        expect(get(selectedMoveStore)).toBe(null);
    });

    test('Ctrl+click picks plays, the menu rolls them out with the chosen preset', async () => {
        withCheckerAnalysis();
        await renderPanel();
        await fireEvent.click(row('13/7* 8/7'));
        await fireEvent.click(row('13/11 13/7'), { ctrlKey: true });
        expect(row('13/7* 8/7').classList.contains('selected')).toBe(true);
        expect(row('13/11 13/7').classList.contains('selected')).toBe(true);
        expect(row('24/18 13/11').classList.contains('selected')).toBe(false);
        await fireEvent.contextMenu(row('13/11 13/7'), { clientX: 10, clientY: 10 });
        const item = await screen.findByText(/Rollout \(Standard\)/);
        await fireEvent.click(item);
        await waitFor(() => expect(startRollout).toHaveBeenCalledTimes(1));
        const req = startRollout.mock.calls[0][0];
        expect(req.moves).toEqual(['13/7* 8/7', '13/11 13/7']);
        expect(req.positionId).toBe(7);
        expect(req.store).toBe(true);
        expect(req.settings.max_games).toBe(1296);
    });

    test('Shift+click extends the selection over the rows on screen', async () => {
        withCheckerAnalysis();
        await renderPanel();
        await fireEvent.click(row('13/7* 8/7'));
        await fireEvent.click(row('13/11 13/7'), { shiftKey: true });
        for (const m of MOVES) expect(row(m.move).classList.contains('selected')).toBe(true);
    });

    test('a right-click on a row outside the selection makes it the selection', async () => {
        withCheckerAnalysis();
        rolloutChoiceStore.set({ preset: 'fast', custom: null });
        await renderPanel();
        await fireEvent.click(row('13/7* 8/7'));
        await fireEvent.contextMenu(row('24/18 13/11'), { clientX: 10, clientY: 10 });
        expect(get(selectedMoveStore)).toBe('24/18 13/11');
        await fireEvent.click(await screen.findByText('Rollout (Fast)'));
        await waitFor(() => expect(startRollout).toHaveBeenCalledTimes(1));
        expect(startRollout.mock.calls[0][0].moves).toEqual(['24/18 13/11']);
        expect(startRollout.mock.calls[0][0].settings.max_games).toBe(216);
    });

    test('while a rollout runs, the menu cancels it', async () => {
        withCheckerAnalysis();
        rolloutStore.set({ ...idleRollout(), job: 1, running: true, kind: 'position', positionId: 7 });
        rolloutStatus.mockResolvedValue({ running: true, job: 1, kind: 'position', positionId: 7, games: 36, maxGames: 216 });
        await renderPanel();
        await fireEvent.contextMenu(row('13/7* 8/7'), { clientX: 10, clientY: 10 });
        await fireEvent.click(await screen.findByText('Cancel the rollout'));
        expect(cancelRollout).toHaveBeenCalled();
        expect(startRollout).not.toHaveBeenCalled();
    });

    test('r rolls out the selection; Escape cancels the rollout running', async () => {
        withCheckerAnalysis();
        const { container } = await renderPanel();
        await fireEvent.click(row('24/18 13/11'));
        rolloutStatus.mockResolvedValue({ running: true, job: 1, kind: 'position', positionId: 7, games: 0, maxGames: 1296 });
        const panel = /** @type {HTMLElement} */ (container.querySelector('#analysisPanel'));
        await fireEvent.keyDown(panel, { key: 'r' });
        await waitFor(() => expect(startRollout).toHaveBeenCalledTimes(1));
        expect(startRollout.mock.calls[0][0].moves).toEqual(['24/18 13/11']);
        await waitFor(() => expect(get(rolloutStore).running).toBe(true));
        await fireEvent.keyDown(panel, { key: 'Escape' });
        expect(cancelRollout).toHaveBeenCalled();
    });

    test('a stored rollout is read in the row of its play, in any notation', async () => {
        withCheckerAnalysis();
        loadRollouts.mockResolvedValue([stored()]);
        await renderPanel();
        // Its two plays: one in XG's dialect on screen, one written alike.
        await waitFor(() => expect(screen.getAllByTestId('rollout-cell').length).toBe(2));
        const cell = screen.getAllByTestId('rollout-cell')[0];
        expect(row('13/7* 8/7').contains(cell)).toBe(true);
        expect(cell.textContent).toContain('+0.123');
        expect(cell.textContent).toContain('±0.008');
        expect(cell.getAttribute('title')).toContain('sig-A');
        expect(loadRollouts).toHaveBeenCalledWith(7);
    });

    test('no rollout, no column', async () => {
        withCheckerAnalysis();
        await renderPanel();
        await tick();
        expect(screen.queryByTestId('rollout-cell')).toBeNull();
        expect(screen.queryByTestId('rollout-section')).toBeNull();
    });

    test('a remount finds the running job through RolloutStatus', async () => {
        rolloutStatus.mockResolvedValue({ running: true, job: 2, kind: 'position', positionId: 7, games: 36, maxGames: 216, done: 0, total: 0 });
        await renderPanel();
        await waitFor(() => expect(screen.getByTestId('rollout-progress')).toBeTruthy());
    });
});

describe('guards', () => {
    test('a library another instance holds refuses the rollout, said in the reader language', async () => {
        startRollout.mockRejectedValueOnce(new Error('database is read-only: another blunderDB instance holds it'));
        const err = await toggleRollout(['13/7* 8/7']);
        expect(err).toContain('read-only');
        expect(err).toContain('another blunderDB instance');
        expect(get(rolloutStore).error).toBe(err);
        expect(get(rolloutStore).running).toBe(false);
        expect(JSON.stringify(get(statusBarTextStore))).toContain('read-only');
    });

    test('an event of a replaced job is ignored', async () => {
        const { ensureRolloutEvents } = await import('../services/rolloutService.js');
        ensureRolloutEvents();
        handlers['rollout:progress']({ job: 5, positionId: 7, games: 36, maxGames: 216, candidates: [] });
        handlers['rollout:cancelled']({ job: 4, positionId: 7 });
        expect(get(rolloutStore).running).toBe(true);
        expect(get(rolloutStore).job).toBe(5);
    });

    test('an event without a job number is ignored', () => {
        handlers['rollout:progress']({ positionId: 0, games: 5, maxGames: 10 });
        expect(get(rolloutStore).running).toBe(false);
    });

    test('the end of a rollout reloads what is stored', async () => {
        await renderPanel();
        await tick();
        loadRollouts.mockClear();
        handlers['rollout:done']({ job: 1, positionId: 7, result: {}, stored: true });
        await waitFor(() => expect(loadRollouts).toHaveBeenCalledWith(7));
    });

    test('an unsaved board keeps its result only while that board is on screen', async () => {
        positionStore.update((p) => ({ ...p, id: 0 }));
        await renderPanel();
        await toggleRollout();
        expect(startRollout.mock.calls[0][0].store).toBe(false);
        handlers['rollout:done']({ job: 1, positionId: 0, stored: false, record: stored({ signature: 'unsaved-sig' }) });
        await waitFor(() => expect(screen.getByText('unsaved-sig')).toBeTruthy());
        positionStore.update((p) => ({ ...p, dice: [6, 6] }));
        await waitFor(() => expect(screen.queryByText('unsaved-sig')).toBeNull());
    });
});

describe('command', () => {
    test('rollout fast starts the fast preset; stop cancels', async () => {
        await runRolloutCommand('fast');
        expect(startRollout.mock.calls[0][0].settings.max_games).toBe(216);
        await runRolloutCommand('stop');
        expect(cancelRollout).toHaveBeenCalled();
    });

    test('rollout search rolls out exactly the list on screen, after asking with the total', async () => {
        await runRolloutCommand('search');
        await waitFor(() => expect(startIDs).toHaveBeenCalledTimes(1));
        expect(countIDs.mock.calls[0][0]).toEqual([3, 5, 8]);
        expect(confirmAction.mock.calls[0][0]).toContain('42');
        expect(startIDs.mock.calls[0][0]).toEqual([3, 5, 8]);
        expect(startIDs.mock.calls[0][1].max_games).toBe(1296);
    });

    test('declining the confirmation starts nothing', async () => {
        confirmAction.mockResolvedValueOnce(false);
        await runRolloutCommand('search fast');
        await waitFor(() => expect(confirmAction).toHaveBeenCalled());
        expect(startIDs).not.toHaveBeenCalled();
    });

    test('a command asks before replacing a running job', async () => {
        rolloutStore.set({ ...idleRollout(), running: true, kind: 'batch', total: 3 });
        confirmAction.mockResolvedValueOnce(false);
        await runRolloutCommand('fast');
        expect(startRollout).not.toHaveBeenCalled();
        confirmAction.mockResolvedValueOnce(true);
        await runRolloutCommand('fast');
        expect(startRollout).toHaveBeenCalledTimes(1);
    });
});
