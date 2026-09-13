/**
 * SearchPanel.replayLike.test.js — #404
 *
 * L'ADR-0043 range `like` dans la grammaire pour que tout ce qui relit une
 * requête la comprenne, l'historique compris. Or le rejeu (double-clic dans
 * l'historique ou dans la bibliothèque de filtres, executeSearch) ne
 * transmettait aucun champ du classement : `s like42` rejoué partait en
 * recherche ordinaire, non classée, sans rien dire.
 *
 * Ce test suit le rejeu jusqu'à l'appel du backend, sans raccourci :
 * double-clic → executeSearch → loadPositionsByFilters →
 * RankPositionIDsByFilters, et exige la même charge utile que la même
 * commande tapée dans la barre.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import { get } from 'svelte/store';

const bindings = vi.hoisted(() => ({
    SaveSearchHistory: vi.fn(() => Promise.resolve()),
    LoadSearchHistory: vi.fn(() => Promise.resolve([])),
    DeleteSearchHistoryEntry: vi.fn(() => Promise.resolve()),
    LoadFilters: vi.fn(() => Promise.resolve([])),
    DeleteFilter: vi.fn(() => Promise.resolve()),
    LoadEditPosition: vi.fn(() => Promise.resolve(null)),
    LoadExcludePosition: vi.fn(() => Promise.resolve(null)),
    LoadPositionIDsByFilters: vi.fn(() => Promise.resolve([])),
    RankPositionIDsByFilters: vi.fn(() => Promise.resolve([])),
    LoadPositionsByIDs: vi.fn(() => Promise.resolve([])),
    ListPositionIDs: vi.fn(() => Promise.resolve([])),
    SaveComment: vi.fn(() => Promise.resolve()),
    ClearCommandHistory: vi.fn(() => Promise.resolve())
}));

vi.mock('../../wailsjs/go/database/Database.js', async (importOriginal) => ({ ...(await importOriginal()), ...bindings }));
vi.mock('../../wailsjs/go/main/Config.js', async (importOriginal) => ({
    ...(await importOriginal()),
    GetLikeLimit: vi.fn(() => Promise.resolve(10)),
    GetLikeMaxDistance: vi.fn(() => Promise.resolve(0))
}));
vi.mock('../services/databaseService.js', () => ({
    setStatusBarMessage: vi.fn(),
    warningMessageStore: { subscribe: vi.fn(), set: vi.fn(), update: vi.fn() }
}));
vi.mock('../services/sessionService.js', () => ({ saveSessionState: vi.fn() }));
vi.mock('../services/confirmService.js', () => ({ confirmAction: vi.fn(() => Promise.resolve(true)) }));

import SearchPanel from '../components/SearchPanel.svelte';
import { processCommand, initCommandProcessor } from '../commandProcessor.js';
import { loadPositionsByFilters } from '../services/positionService.js';
import { searchHistoryStore } from '../stores/searchHistoryStore.js';
import { filterLibraryStore } from '../stores/filterLibraryStore.js';
import { positionStore, emptyPosition } from '../stores/positionStore.js';
import { statusBarModeStore, statusBarTextStore, currentPositionIndexStore } from '../stores/uiStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { tMsg } from '../i18n';

// Un plateau dessiné, reconnaissable : cinq pions sur le 6, trois sur le 8.
function drawnBoard() {
    const p = emptyPosition();
    p.board.points[6] = { checkers: 5, color: 0 };
    p.board.points[8] = { checkers: 3, color: 0 };
    p.board.bearoff = [7, 15];
    return p;
}

let pending;
function trackedLoad(opts) {
    pending = loadPositionsByFilters(opts);
    return pending;
}

// Ce que le backend reçoit pour une commande tapée dans la barre.
async function typed(command, board) {
    positionStore.set(board);
    pending = undefined;
    initCommandProcessor({ onLoadPositionsByFilters: trackedLoad });
    processCommand(command);
    expect(pending, `${command} never reached loadPositionsByFilters`).toBeDefined();
    await pending;
    const calls = bindings.RankPositionIDsByFilters.mock.calls;
    expect(calls, `${command} typed was not ranked`).toHaveLength(1);
    const sent = calls[0];
    vi.clearAllMocks();
    searchHistoryStore.set([]);
    return sent;
}

async function mountOn(subTab) {
    pending = undefined;
    const utils = render(SearchPanel, { props: { onLoadPositionsByFilters: trackedLoad, onAddToFilterLibrary: vi.fn() } });
    await tick();
    const index = { history: 1, saved: 2 }[subTab];
    await fireEvent.click(utils.container.querySelectorAll('.sub-tab-btn')[index]);
    await tick();
    return utils;
}

// Rejoue une entrée d'historique ; l'écran porte un AUTRE plateau, pour
// prouver que c'est celui de l'entrée qui voyage.
async function replayHistory(command, board) {
    const entry = { timestamp: 1700000000000, command, position: board ? JSON.stringify(board) : null, excludePosition: null };
    searchHistoryStore.set([entry]);
    bindings.LoadSearchHistory.mockResolvedValue([entry]);
    positionStore.set(emptyPosition());
    const { container } = await mountOn('history');
    const row = container.querySelector('.history-table tbody tr');
    expect(row).not.toBeNull();
    await fireEvent.dblClick(row);
    await tick();
    if (pending) await pending;
}

async function replaySaved(command, board) {
    const filter = { id: 1, name: 'voisines', command };
    filterLibraryStore.set([filter]);
    bindings.LoadFilters.mockResolvedValue([filter]);
    bindings.LoadEditPosition.mockResolvedValue(board ? JSON.stringify(board) : null);
    positionStore.set(emptyPosition());
    const { container } = await mountOn('saved');
    const item = container.querySelector('.saved-item');
    expect(item).not.toBeNull();
    await fireEvent.dblClick(item);
    // executeSavedFilter attend LoadEditPosition et LoadExcludePosition.
    for (let i = 0; i < 5 && !pending; i++) await tick();
    if (pending) await pending;
}

function ranked() {
    expect(bindings.LoadPositionIDsByFilters, 'the replay ran an unranked search').not.toHaveBeenCalled();
    expect(bindings.RankPositionIDsByFilters).toHaveBeenCalledTimes(1);
    return bindings.RankPositionIDsByFilters.mock.calls[0];
}

beforeEach(() => {
    vi.clearAllMocks();
    databasePathStore.set('/tmp/lib.db');
    // Le panneau de recherche vit en mode ÉDITION (App.svelte l'y met).
    statusBarModeStore.set('EDIT');
    currentPositionIndexStore.set(-1);
});

afterEach(() => {
    cleanup();
    searchHistoryStore.set([]);
    filterLibraryStore.set([]);
});

describe('le rejeu porte le classement like (#404)', () => {
    for (const command of ['s like42', 's like42*', 's like42 E>80']) {
        test(`${command} rejoué depuis l'historique envoie ce que la barre envoie`, async () => {
            const board = drawnBoard();
            const expected = await typed(command, board);
            await replayHistory(command, board);
            expect(ranked()).toEqual(expected);
        });

        test(`${command} rejoué depuis la bibliothèque envoie ce que la barre envoie`, async () => {
            const board = drawnBoard();
            const expected = await typed(command, board);
            await replaySaved(command, board);
            expect(ranked()).toEqual(expected);
        });
    }

    test('like42* élargit la classe et E>80 restreint, au rejeu comme à la frappe', async () => {
        await replayHistory('s like42* E>80', drawnBoard());
        const [payload] = ranked();
        expect(payload.likeFilter).toBe(true);
        expect(payload.likeTargetId).toBe(42);
        expect(payload.likeWidened).toBe(true);
        expect(payload.moveErrorFilter).toBe('E>80');
    });

    test('s like sur un plateau dessiné reclasse contre le plateau conservé avec l’entrée', async () => {
        const board = drawnBoard();
        const expected = await typed('s like', board);
        expect(expected[0].likeTargetBoard.board.points[6].checkers).toBe(5);

        await replayHistory('s like', board);
        const sent = ranked();
        expect(sent).toEqual(expected);
        expect(sent[0].likeTargetId).toBe(0);
        expect(sent[0].likeTargetBoard.board.points[6].checkers).toBe(5);
    });

    test('s like sur un plateau dessiné, depuis la bibliothèque, reclasse contre le plateau enregistré', async () => {
        const board = drawnBoard();
        const expected = await typed('s like', board);
        await replaySaved('s like', board);
        expect(ranked()).toEqual(expected);
    });

    test('un like nu dont l’entrée n’a pas gardé de plateau est refusé, pas relancé contre un autre', async () => {
        await replayHistory('s like', null);
        expect(bindings.RankPositionIDsByFilters).not.toHaveBeenCalled();
        expect(bindings.LoadPositionIDsByFilters).not.toHaveBeenCalled();
        expect(get(statusBarTextStore)).toEqual(tMsg('similar.noPosition'));
    });
});
