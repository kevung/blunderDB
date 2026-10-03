/**
 * Le panneau des rollouts : chaque composant est monté, les appels Wails sont simulés.
 * Vérifie le lancement (Rapide / Standard / Libre), la progression venue des événements,
 * l'état retrouvé par RolloutStatus au remontage, le lot (confirmation avec le total) et
 * l'affichage des rollouts stockés à côté de l'analyse.
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
vi.mock('../../wailsjs/runtime/runtime.js', () => ({
    EventsOn: vi.fn((name, cb) => {
        handlers[name] = cb;
        return () => {};
    })
}));
const displayedIDs = vi.fn(() => [3, 5, 8]);
vi.mock('../services/modeMachine.js', async (importOriginal) => ({ ...(await importOriginal()), displayedPositionIDs: () => displayedIDs() }));
vi.mock('../services/confirmService.js', async (importOriginal) => ({ ...(await importOriginal()), confirmAction: (...a) => confirmAction(...a) }));

const { positionStore } = await import('../stores/positionStore.js');
const { databasePathStore } = await import('../stores/databaseStore.js');
const { rolloutStore, rolloutChoiceStore, idleRollout } = await import('../stores/rolloutStore.js');
const RolloutSection = (await import('../components/RolloutSection.svelte')).default;
const RolloutResults = (await import('../components/RolloutResults.svelte')).default;
const { runRolloutCommand } = await import('../services/rolloutService.js');

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
});
afterEach(() => cleanup());

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

describe('RolloutSection', () => {
    test('lists the stored rollouts of the position beside the analysis', async () => {
        loadRollouts.mockResolvedValue([stored()]);
        render(RolloutSection);
        await waitFor(() => expect(screen.getByTestId('rollout-block')).toBeTruthy());
        expect(loadRollouts).toHaveBeenCalledWith(7);
    });

    test('Standard starts the preset on the saved position and asks for storage', async () => {
        render(RolloutSection);
        const start = await screen.findByTestId('rollout-start');
        await waitFor(() => expect(start.disabled).toBe(false));
        await fireEvent.click(start);
        expect(startRollout).toHaveBeenCalledTimes(1);
        const req = startRollout.mock.calls[0][0];
        expect(req.positionId).toBe(7);
        expect(req.store).toBe(true);
        expect(req.settings.max_games).toBe(1296);
    });

    test('Rapide starts the fast preset', async () => {
        render(RolloutSection);
        await fireEvent.click(await screen.findByText('Fast'));
        await fireEvent.click(screen.getByTestId('rollout-start'));
        expect(startRollout.mock.calls[0][0].settings.max_games).toBe(216);
    });

    test('Libre exposes every setting and sends the edited values', async () => {
        render(RolloutSection);
        await fireEvent.click(await screen.findByText('Custom'));
        for (const key of ['truncation', 'min_games', 'max_games', 'jsd_limit', 'ply', 'candidates', 'seed', 'workers']) {
            expect(screen.getByTestId('rollout-' + key)).toBeTruthy();
        }
        const maxGames = screen.getByTestId('rollout-max_games');
        await fireEvent.change(maxGames, { target: { value: '720' } });
        await fireEvent.click(screen.getByTestId('rollout-start'));
        expect(startRollout.mock.calls[0][0].settings.max_games).toBe(720);
        expect(startRollout.mock.calls[0][0].settings.seed).toBe(1);
    });

    test('an unsaved position is rolled out but never stored', async () => {
        positionStore.update((p) => ({ ...p, id: 0 }));
        render(RolloutSection);
        const start = await screen.findByTestId('rollout-start');
        await waitFor(() => expect(start.disabled).toBe(false));
        await fireEvent.click(start);
        const req = startRollout.mock.calls[0][0];
        expect(req.store).toBe(false);
        expect(req.positionId).toBe(0);
        expect(req.position).toBeTruthy();
    });

    test('progress events fill the bar and Cancel stops the job', async () => {
        render(RolloutSection);
        await screen.findByTestId('rollout-start');
        handlers['rollout:progress']({ job: 1, positionId: 7, games: 108, maxGames: 216, candidates: [{ move: '13/7 8/7', equity: 0.1, std_err: 0.01, ci95: 0.02, games: 108, jsd: 0 }] });
        await tick();
        expect(screen.getByTestId('rollout-progress').textContent).toContain('108/216');
        expect(screen.getByTestId('rollout-live')).toBeTruthy();
        await fireEvent.click(screen.getByTestId('rollout-cancel'));
        expect(cancelRollout).toHaveBeenCalled();
        handlers['rollout:cancelled']({ job: 1, positionId: 7, result: null });
        await tick();
        expect(screen.getByTestId('rollout-outcome')).toBeTruthy();
        expect(screen.queryByTestId('rollout-progress')).toBeNull();
    });

    test('the end of a rollout reloads what is stored', async () => {
        render(RolloutSection);
        await screen.findByTestId('rollout-start');
        loadRollouts.mockClear();
        handlers['rollout:done']({ positionId: 7, result: {}, stored: true });
        await waitFor(() => expect(loadRollouts).toHaveBeenCalledWith(7));
    });

    test('a remount finds the running job through RolloutStatus', async () => {
        rolloutStatus.mockResolvedValue({ running: true, kind: 'position', positionId: 7, games: 36, maxGames: 216, done: 0, total: 0 });
        render(RolloutSection);
        await waitFor(() => expect(screen.getByTestId('rollout-progress')).toBeTruthy());
        expect(screen.getByTestId('rollout-cancel')).toBeTruthy();
    });

    test('the batch rolls out exactly the list on screen, after asking with the total', async () => {
        render(RolloutSection);
        const batch = await screen.findByTestId('rollout-batch');
        await waitFor(() => expect(batch.disabled).toBe(false));
        await fireEvent.click(batch);
        await waitFor(() => expect(startIDs).toHaveBeenCalledTimes(1));
        expect(countIDs.mock.calls[0][0]).toEqual([3, 5, 8]);
        expect(confirmAction.mock.calls[0][0]).toContain('42');
        expect(startIDs.mock.calls[0][0]).toEqual([3, 5, 8]);
    });

    test('declining the confirmation starts nothing', async () => {
        confirmAction.mockResolvedValueOnce(false);
        render(RolloutSection);
        const batch = await screen.findByTestId('rollout-batch');
        await waitFor(() => expect(batch.disabled).toBe(false));
        await fireEvent.click(batch);
        await waitFor(() => expect(confirmAction).toHaveBeenCalled());
        expect(startIDs).not.toHaveBeenCalled();
    });

    test('batch events show position count and the end summary', async () => {
        render(RolloutSection);
        await screen.findByTestId('rollout-start');
        handlers['rollout-batch:started']({ job: 1, total: 5 });
        handlers['rollout-batch:progress']({ job: 1, done: 2, total: 5, positionId: 9, games: 36, maxGames: 216 });
        await tick();
        expect(screen.getByTestId('rollout-progress').textContent).toContain('2');
        handlers['rollout-batch:done']({ job: 1, total: 5, rolledOut: 4, refused: 1, failed: 0, signature: 's' });
        await tick();
        expect(screen.getByTestId('rollout-outcome').textContent).toContain('4');
        expect(get(rolloutStore).running).toBe(false);
    });
});

describe('guards', () => {
    test('a seed beyond what JavaScript holds exactly is refused, never rounded', async () => {
        render(RolloutSection);
        await fireEvent.click(await screen.findByText('Custom'));
        await fireEvent.change(screen.getByTestId('rollout-seed'), { target: { value: '18014398509481984' } });
        await fireEvent.click(screen.getByTestId('rollout-start'));
        expect(startRollout).not.toHaveBeenCalled();
        await waitFor(() => expect(screen.getByRole('alert')).toBeTruthy());
    });

    test('an event of a replaced job is ignored', async () => {
        render(RolloutSection);
        await screen.findByTestId('rollout-start');
        handlers['rollout:progress']({ job: 5, positionId: 7, games: 36, maxGames: 216, candidates: [] });
        handlers['rollout:cancelled']({ job: 4, positionId: 7 });
        await tick();
        expect(screen.getByTestId('rollout-progress')).toBeTruthy();
        expect(get(rolloutStore).job).toBe(5);
    });

    test('r stops a rollout of a position but not a batch', async () => {
        const { toggleRollout } = await import('../services/rolloutService.js');
        rolloutStore.set({ ...idleRollout(), running: true, kind: 'batch', total: 3 });
        await toggleRollout();
        expect(cancelRollout).not.toHaveBeenCalled();
        rolloutStore.set({ ...idleRollout(), running: true, kind: 'position', positionId: 7 });
        await toggleRollout();
        expect(cancelRollout).toHaveBeenCalledTimes(1);
    });

    test('a refusal started with r is shown in the panel', async () => {
        const { toggleRollout } = await import('../services/rolloutService.js');
        startRollout.mockRejectedValueOnce(new Error('rollout: no database is open'));
        render(RolloutSection);
        await screen.findByTestId('rollout-start');
        await toggleRollout();
        await waitFor(() => expect(screen.getByRole('alert').textContent).toContain('no database is open'));
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

    test('Escape in a settings field leaves the field and does not close the panel', async () => {
        const onClose = vi.fn();
        const AnalysisPanel = (await import('../components/AnalysisPanel.svelte')).default;
        render(AnalysisPanel, { props: { onClose } });
        await fireEvent.click(await screen.findByText('Custom'));
        const field = screen.getByTestId('rollout-ply');
        field.focus();
        await fireEvent.keyDown(field, { key: 'Escape' });
        expect(document.activeElement).not.toBe(field);
        expect(onClose).not.toHaveBeenCalled();
    });

    test('Escape in a settings field drops what was typed', async () => {
        const AnalysisPanel = (await import('../components/AnalysisPanel.svelte')).default;
        render(AnalysisPanel, { props: { onClose: vi.fn() } });
        await fireEvent.click(await screen.findByText('Custom'));
        const field = screen.getByTestId('rollout-ply');
        const before = field.value;
        field.focus();
        await fireEvent.input(field, { target: { value: '7' } });
        await fireEvent.keyDown(field, { key: 'Escape' });
        expect(field.value).toBe(before);
    });

    test('the end of the job a start replaced does not touch the new one', async () => {
        positionStore.update((p) => ({ ...p, id: 0 }));
        rolloutStore.set({ ...idleRollout(), job: 4, running: true, kind: 'position' });
        startRollout.mockImplementationOnce(async () => {
            handlers['rollout:cancelled']({ job: 4, positionId: 0 });
            handlers['rollout:done']({ job: 5, positionId: 0, stored: false, record: stored({ signature: 'new-job' }) });
            return 5;
        });
        render(RolloutSection);
        const start = await screen.findByTestId('rollout-start');
        rolloutStore.update((s) => ({ ...s, running: false }));
        await waitFor(() => expect(start.disabled).toBe(false));
        await fireEvent.click(start);
        await waitFor(() => expect(screen.getByText('new-job')).toBeTruthy());
    });

    test('an event without a job number is ignored', async () => {
        render(RolloutSection);
        await screen.findByTestId('rollout-start');
        handlers['rollout:progress']({ positionId: 0, games: 5, maxGames: 10 });
        expect(get(rolloutStore).running).toBe(false);
    });

    test('an unsaved board keeps its result only while that board is on screen', async () => {
        positionStore.update((p) => ({ ...p, id: 0 }));
        render(RolloutSection);
        const start = await screen.findByTestId('rollout-start');
        await waitFor(() => expect(start.disabled).toBe(false));
        await fireEvent.click(start);
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
});
