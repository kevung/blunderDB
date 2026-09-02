/**
 * sessionService.test.js
 *
 * La restauration de session tourne au démarrage, juste après l'ouverture de la
 * base : un échec silencieux ouvre l'app sur un état faux (liste vide, index
 * hors bornes, recherche « active » sans positions). Ces tests figent les
 * quatre issues : pas de session → bibliothèque entière ; session complète →
 * positions et index restaurés dans les stores ; base ou binding défaillant →
 * repli sur la bibliothèque sans exception ; sauvegarde à la fermeture.
 */

import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

const { bindings, searchState, positionService, setStatusBarMessage } = vi.hoisted(() => {
    const searchState = { lastSearchCommand: '', lastSearchPosition: null, hasActiveSearch: false };
    return {
        searchState,
        bindings: { SaveSessionState: vi.fn(), LoadSessionState: vi.fn(), ListPositionIDs: vi.fn() },
        positionService: {
            getSearchState: () => ({ ...searchState }),
            setSearchState: vi.fn((next) => Object.assign(searchState, next)),
            loadAllPositions: vi.fn(() => Promise.resolve())
        },
        setStatusBarMessage: vi.fn()
    };
});

vi.mock('../../wailsjs/go/database/Database.js', () => bindings);
vi.mock('../services/positionService.js', () => positionService);
vi.mock('../services/databaseService.js', () => ({ setStatusBarMessage }));

import { saveSessionState, restoreSessionState } from '../services/sessionService.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { positionsStore } from '../stores/positionStore.js';
import { currentPositionIndexStore } from '../stores/uiStore.js';
import { lastSearchStore } from '../stores/searchHistoryStore.js';

const pos = (id) => ({ id, board: { points: [], bearoff: [0, 0] }, dice: [1, 2], score: [0, 0] });
const library = [pos(1), pos(2), pos(3)];
const searchBoard = { board: { points: [{ checkers: 2, color: 0 }] } };

beforeEach(() => {
    vi.clearAllMocks();
    Object.assign(searchState, { lastSearchCommand: '', lastSearchPosition: null, hasActiveSearch: false });
    databasePathStore.set('/tmp/lib.db');
    positionsStore.set([]);
    currentPositionIndexStore.set(-1);
    lastSearchStore.set({ command: 'stale', position: '{}' });
    bindings.ListPositionIDs.mockResolvedValue(library.map((p) => p.id));
    bindings.SaveSessionState.mockResolvedValue(undefined);
});

describe('restoreSessionState', () => {
    test('sans session enregistrée : état de recherche vierge et bibliothèque entière', async () => {
        bindings.LoadSessionState.mockResolvedValue(null);

        await restoreSessionState();

        expect(positionService.setSearchState).toHaveBeenCalledWith({ lastSearchCommand: '', lastSearchPosition: null, hasActiveSearch: false });
        expect(get(lastSearchStore)).toBeNull();
        expect(positionService.loadAllPositions).toHaveBeenCalledTimes(1);
        expect(setStatusBarMessage).not.toHaveBeenCalled();
    });

    test('session avec recherche active : positions, ordre et index restaurés dans les stores', async () => {
        bindings.LoadSessionState.mockResolvedValue({
            hasActiveSearch: true,
            lastSearchCommand: 's p>10',
            lastSearchPosition: JSON.stringify(searchBoard),
            lastPositionIds: [3, 1],
            lastPositionIndex: 1
        });

        await restoreSessionState();

        expect(get(positionsStore).ids).toEqual([3, 1]);
        expect(get(currentPositionIndexStore)).toBe(1);
        expect(searchState).toEqual({ lastSearchCommand: 's p>10', lastSearchPosition: searchBoard, hasActiveSearch: true });
        expect(setStatusBarMessage).toHaveBeenCalledWith({ i18nKey: 'status.sessionRestored', i18nParams: { count: 2, index: 2 } });
        expect(positionService.loadAllPositions).not.toHaveBeenCalled();
    });

    test('index hors bornes : ramené dans la liste restaurée', async () => {
        bindings.LoadSessionState.mockResolvedValue({ hasActiveSearch: true, lastPositionIds: [2, 3], lastPositionIndex: 9 });

        await restoreSessionState();

        expect(get(currentPositionIndexStore)).toBe(1);
    });

    test('session avec vues : les onglets sont restaurés et le message le dit', async () => {
        const viewsJSON = JSON.stringify({ nextViewId: 2, activeViewId: 1, views: [{ id: 1, name: 'Vue 1', positionIds: [2, 3], positionIndex: 1 }] });
        bindings.LoadSessionState.mockResolvedValue({ viewsJSON, hasActiveSearch: true, lastSearchCommand: 's', lastPositionIds: [2, 3] });

        await restoreSessionState();

        expect(get(positionsStore).ids).toEqual([2, 3]);
        expect(get(currentPositionIndexStore)).toBe(1);
        expect(setStatusBarMessage).toHaveBeenCalledWith({ i18nKey: 'status.sessionRestoredViews', i18nParams: null });
        expect(positionService.loadAllPositions).not.toHaveBeenCalled();
    });

    test('positions disparues de la base : repli sur la bibliothèque, sans exception', async () => {
        bindings.LoadSessionState.mockResolvedValue({ hasActiveSearch: true, lastPositionIds: [42, 43], lastPositionIndex: 0 });
        bindings.ListPositionIDs.mockResolvedValue([]);

        await expect(restoreSessionState()).resolves.toBeUndefined();

        expect(get(positionsStore).ids).toEqual([]);
        expect(positionService.loadAllPositions).toHaveBeenCalledTimes(1);
    });

    test('binding en erreur : repli sur la bibliothèque, sans exception', async () => {
        bindings.LoadSessionState.mockRejectedValue(new Error('database is locked'));

        await expect(restoreSessionState()).resolves.toBeUndefined();

        expect(positionService.loadAllPositions).toHaveBeenCalledTimes(1);
    });

    test('session illisible (JSON corrompu) : repli sur la bibliothèque, sans exception', async () => {
        bindings.LoadSessionState.mockResolvedValue({ hasActiveSearch: true, lastPositionIds: [1], lastSearchPosition: '{not json' });

        await expect(restoreSessionState()).resolves.toBeUndefined();

        expect(positionService.loadAllPositions).toHaveBeenCalledTimes(1);
    });
});

describe('saveSessionState (à la fermeture)', () => {
    test('persiste la liste courante, l’index, la recherche et les vues', async () => {
        positionsStore.set([pos(3), pos(1)]);
        currentPositionIndexStore.set(1);
        Object.assign(searchState, { lastSearchCommand: 's p>10', lastSearchPosition: searchBoard, hasActiveSearch: true });

        await saveSessionState();

        expect(bindings.SaveSessionState).toHaveBeenCalledTimes(1);
        const saved = bindings.SaveSessionState.mock.calls[0][0];
        expect(saved).toMatchObject({ lastPositionIds: [3, 1], lastPositionIndex: 1, hasActiveSearch: true, lastSearchCommand: 's p>10' });
        expect(JSON.parse(saved.lastSearchPosition)).toEqual(searchBoard);
        expect(JSON.parse(saved.viewsJSON).views[0].positionIds).toEqual([3, 1]);
    });

    test('sans base ouverte : rien n’est écrit', async () => {
        databasePathStore.set('');

        await saveSessionState();

        expect(bindings.SaveSessionState).not.toHaveBeenCalled();
    });

    test('binding en erreur : ne lève pas', async () => {
        bindings.SaveSessionState.mockRejectedValue(new Error('disk full'));

        await expect(saveSessionState()).resolves.toBeUndefined();
    });
});
