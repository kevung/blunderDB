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
        bindings: { SaveSessionState: vi.fn(), LoadSessionState: vi.fn(), ListPositionIDs: vi.fn(), LoadPositionIDsByFilters: vi.fn(), RankPositionIDsByFilters: vi.fn() },
        positionService: {
            getSearchState: () => ({ ...searchState }),
            setSearchState: vi.fn((next) => Object.assign(searchState, next)),
            loadAllPositions: vi.fn(() => Promise.resolve())
        },
        setStatusBarMessage: vi.fn()
    };
});

vi.mock('../../wailsjs/go/database/Database.js', () => bindings);
vi.mock('../../wailsjs/go/main/Config.js', () => ({ GetLikeLimit: vi.fn().mockResolvedValue(10) }));
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

    const viewsOf = (view) => JSON.stringify({ nextViewId: 2, activeViewId: 1, views: [{ id: 1, name: '#1', positionIndex: 0, ...view }] });

    test('session de recherche : la requête est rejouée, la position courante retrouvée par id', async () => {
        const payload = { searchText: 'p>10' };
        bindings.LoadPositionIDsByFilters.mockResolvedValue([3, 2, 1]);
        bindings.LoadSessionState.mockResolvedValue({
            hasActiveSearch: true,
            lastSearchCommand: 's p>10',
            lastSearchPosition: JSON.stringify(searchBoard),
            viewsJSON: viewsOf({ origin: { kind: 'search', payload }, positionId: 2, positionIndex: 0 })
        });

        await restoreSessionState();

        expect(bindings.LoadPositionIDsByFilters).toHaveBeenCalledWith(payload);
        expect(get(positionsStore).ids).toEqual([3, 2, 1]);
        expect(get(currentPositionIndexStore)).toBe(1);
        expect(searchState).toEqual({ lastSearchCommand: 's p>10', lastSearchPosition: searchBoard, hasActiveSearch: true });
        expect(setStatusBarMessage).toHaveBeenCalledWith({ i18nKey: 'status.sessionRestoredViews', i18nParams: null });
        expect(positionService.loadAllPositions).not.toHaveBeenCalled();
    });

    test('recherche devenue vide : repli sur la bibliothèque', async () => {
        bindings.LoadPositionIDsByFilters.mockResolvedValue([]);
        bindings.LoadSessionState.mockResolvedValue({ viewsJSON: viewsOf({ origin: { kind: 'search', payload: {} }, positionId: 2 }) });

        await restoreSessionState();

        expect(get(positionsStore).ids).toEqual([1, 2, 3]);
        expect(get(currentPositionIndexStore)).toBe(1);
    });

    test('vue de bibliothèque : position retrouvée par id', async () => {
        bindings.LoadSessionState.mockResolvedValue({ viewsJSON: viewsOf({ origin: { kind: 'library' }, positionId: 3 }) });

        await restoreSessionState();

        expect(get(positionsStore).ids).toEqual([1, 2, 3]);
        expect(get(currentPositionIndexStore)).toBe(2);
        expect(bindings.LoadPositionIDsByFilters).not.toHaveBeenCalled();
    });

    test('id disparu : l’index enregistré, ramené dans les bornes', async () => {
        bindings.LoadSessionState.mockResolvedValue({ viewsJSON: viewsOf({ positionId: 99, positionIndex: 9 }) });

        await restoreSessionState();

        expect(get(currentPositionIndexStore)).toBe(2);
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
    test('persiste un descripteur : taille indépendante du nombre de positions', async () => {
        Object.assign(searchState, { lastSearchCommand: 's p>10', lastSearchPosition: searchBoard, hasActiveSearch: true });
        const sizes = [];
        for (const n of [10, 100000]) {
            vi.clearAllMocks();
            positionsStore.setIds(Array.from({ length: n }, (_, i) => i + 1));
            currentPositionIndexStore.set(3);
            await saveSessionState();
            const saved = bindings.SaveSessionState.mock.calls[0][0];
            expect(saved).toMatchObject({ lastPositionIds: [], lastPositionIndex: 3, hasActiveSearch: true, lastSearchCommand: 's p>10' });
            expect(JSON.parse(saved.lastSearchPosition)).toEqual(searchBoard);
            expect(JSON.parse(saved.viewsJSON).views[0].positionId).toBe(4);
            sizes.push(JSON.stringify(saved).length);
        }
        expect(sizes[1]).toBe(sizes[0]);
        expect(sizes[1]).toBeLessThan(4000);
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
