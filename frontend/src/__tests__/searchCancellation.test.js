/**
 * searchCancellation.test.js — une recherche à la fois, qui se montre avant
 * d'être comptée et que Échap interrompt.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { get } from 'svelte/store';

const bindings = vi.hoisted(() => ({
    SaveSearchHistory: vi.fn(() => Promise.resolve()),
    ListPositionIDs: vi.fn(() => Promise.resolve([])),
    LoadPositionsByIDs: vi.fn(() => Promise.resolve([])),
    CountPositionsByFilters: vi.fn(),
    SearchPositionIDs: vi.fn(),
    IndexOfPositionByFilters: vi.fn(() => Promise.resolve(-1)),
    CancelSearch: vi.fn(() => Promise.resolve()),
    LoadAnalysis: vi.fn(() => Promise.resolve(null)),
    LoadComment: vi.fn(() => Promise.resolve('')),
    SaveLastVisitedPosition: vi.fn(() => Promise.resolve())
}));

vi.mock('../../wailsjs/go/database/Database.js', () => bindings);
vi.mock('../services/sessionService.js', () => ({ saveSessionState: vi.fn() }));
vi.mock('../services/confirmService.js', async (importOriginal) => ({
    ...(await importOriginal()),
    confirmAction: vi.fn(() => Promise.resolve(true))
}));

import { loadPositionsByFilters, cancelSearch, isSearching, settleList } from '../services/positionService.js';
import { positionsStore, searchSource } from '../stores/positionStore.js';
import { statusBarTextStore, statusBarModeStore, currentPositionIndexStore } from '../stores/uiStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { handleEscapeCapture } from '../services/escapeService.js';

/** @returns {{ promise: Promise<any>, resolve: (v: any) => void, reject: (e: any) => void }} */
function deferred() {
    let resolve;
    let reject;
    const promise = new Promise((res, rej) => {
        resolve = res;
        reject = rej;
    });
    return { promise, resolve, reject };
}

const flush = async () => {
    for (let i = 0; i < 5; i++) await new Promise((resolve) => setTimeout(resolve, 0));
};
const page = () => Array.from({ length: positionsStore.firstPageSize() }, (_, i) => i + 1);
const key = () => /** @type {any} */ (get(statusBarTextStore))?.i18nKey;

beforeEach(() => {
    vi.resetAllMocks();
    bindings.CancelSearch.mockResolvedValue(undefined);
    bindings.IndexOfPositionByFilters.mockResolvedValue(-1);
    bindings.LoadAnalysis.mockResolvedValue(null);
    bindings.LoadComment.mockResolvedValue('');
    bindings.SaveSearchHistory.mockResolvedValue(undefined);
    bindings.SaveLastVisitedPosition.mockResolvedValue(undefined);
    bindings.LoadPositionsByIDs.mockResolvedValue([]);
    databasePathStore.set('/tmp/lib.db');
    statusBarModeStore.set('NORMAL');
    currentPositionIndexStore.set(-1);
    statusBarTextStore.set('');
});

afterEach(() => {
    cancelSearch();
});

describe('la recherche se montre avant d’être comptée', () => {
    test('la première fenêtre s’affiche, le compte arrive après', async () => {
        const count = deferred();
        bindings.SearchPositionIDs.mockResolvedValue(page());
        bindings.CountPositionsByFilters.mockReturnValue(count.promise);

        const search = loadPositionsByFilters({});
        await flush();
        expect(get(positionsStore).length).toBe(page().length);
        expect(isSearching()).toBe(true);

        count.resolve(5000);
        await search;
        expect(get(positionsStore).length).toBe(5000);
        expect(isSearching()).toBe(false);
    });

    test('une fenêtre courte est la liste entière : pas de compte', async () => {
        bindings.SearchPositionIDs.mockResolvedValue([7, 8]);
        await loadPositionsByFilters({});
        expect(bindings.CountPositionsByFilters).not.toHaveBeenCalled();
        expect(get(positionsStore).length).toBe(2);
    });
});

describe('Échap interrompt la recherche', () => {
    test('pendant le compte : la première page reste, le backend est prévenu, le compte tardif est ignoré', async () => {
        const count = deferred();
        bindings.SearchPositionIDs.mockResolvedValue(page());
        bindings.CountPositionsByFilters.mockReturnValue(count.promise);

        const search = loadPositionsByFilters({});
        await flush();
        handleEscapeCapture(new KeyboardEvent('keydown', { key: 'Escape' }));

        expect(bindings.CancelSearch).toHaveBeenCalledTimes(1);
        expect(isSearching()).toBe(false);
        expect(key()).toBe('status.searchPartial');

        count.resolve(5000);
        await search;
        expect(get(positionsStore).length).toBe(page().length);
        expect(key()).toBe('status.searchPartial');
    });

    test('avant toute fenêtre : rien ne change à l’écran', async () => {
        const window = deferred();
        bindings.SearchPositionIDs.mockReturnValue(window.promise);
        const before = get(positionsStore);

        const search = loadPositionsByFilters({});
        await flush();
        cancelSearch();
        expect(key()).toBe('status.searchCancelled');

        window.resolve(page());
        await search;
        expect(get(positionsStore)).toBe(before);
    });

    test('sans recherche en cours, Échap ne fait rien', () => {
        cancelSearch();
        expect(bindings.CancelSearch).not.toHaveBeenCalled();
    });
});

describe('une seule recherche à la fois', () => {
    test('la réponse d’une recherche remplacée est ignorée', async () => {
        const first = deferred();
        bindings.SearchPositionIDs.mockReturnValueOnce(first.promise).mockResolvedValueOnce([40, 41]);

        const a = loadPositionsByFilters({});
        await flush();
        const b = loadPositionsByFilters({});
        await b;
        expect(bindings.CancelSearch).toHaveBeenCalledTimes(1);
        expect(get(positionsStore).length).toBe(2);

        first.resolve(page());
        await a;
        expect(get(positionsStore).length).toBe(2);
        expect(isSearching()).toBe(false);
    });
});

describe('la recherche remplacée est périmée dès le remplacement', () => {
    test('une fenêtre revenue pendant que CancelSearch attend n’est ni affichée ni comptée', async () => {
        const first = deferred();
        const cancel = deferred();
        bindings.SearchPositionIDs.mockReturnValueOnce(first.promise).mockResolvedValueOnce([40, 41]);
        bindings.CancelSearch.mockReturnValueOnce(cancel.promise);
        const before = get(positionsStore);

        const a = loadPositionsByFilters({});
        await flush();
        const b = loadPositionsByFilters({});
        await flush();
        first.resolve(page());
        await a;
        expect(get(positionsStore)).toBe(before);
        expect(bindings.CountPositionsByFilters).not.toHaveBeenCalled();

        cancel.resolve(undefined);
        await b;
        expect(get(positionsStore).length).toBe(2);
    });

    test('une fenêtre rejetée pendant que CancelSearch attend ne dit pas d’erreur', async () => {
        const first = deferred();
        const cancel = deferred();
        bindings.SearchPositionIDs.mockReturnValueOnce(first.promise).mockResolvedValueOnce([40, 41]);
        bindings.CancelSearch.mockReturnValueOnce(cancel.promise);

        const a = loadPositionsByFilters({});
        await flush();
        const b = loadPositionsByFilters({});
        await flush();
        first.reject(new Error('context canceled'));
        await a;
        expect(key()).not.toBe('status.errorLoadingByFilters');

        cancel.resolve(undefined);
        await b;
    });
});

describe('une liste restaurée sur sa première page se compte comme une recherche', () => {
    const restoreFirstPage = () => {
        const source = searchSource({ searchText: 'w>50' });
        positionsStore.restoreList({ source, length: page().length, provisional: true, firstPage: page() });
        return source;
    };

    test('le compte et le rang arrivent en arrière-plan', async () => {
        const source = restoreFirstPage();
        bindings.CountPositionsByFilters.mockResolvedValue(5000);
        bindings.IndexOfPositionByFilters.mockResolvedValue(4321);

        expect(await settleList({ source, count: true, positionId: 99999 })).toEqual({ index: 4321 });
        expect(get(positionsStore).length).toBe(5000);
        expect(isSearching()).toBe(false);
    });

    test('Échap l’interrompt : le backend est prévenu, la première page reste', async () => {
        const source = restoreFirstPage();
        const count = deferred();
        bindings.CountPositionsByFilters.mockReturnValue(count.promise);

        const settling = settleList({ source, count: true, positionId: 99999 });
        await flush();
        expect(isSearching()).toBe(true);
        handleEscapeCapture(new KeyboardEvent('keydown', { key: 'Escape' }));
        expect(bindings.CancelSearch).toHaveBeenCalledTimes(1);

        count.resolve(5000);
        expect(await settling).toBeNull();
        expect(get(positionsStore).length).toBe(page().length);
        expect(bindings.IndexOfPositionByFilters).not.toHaveBeenCalled();
    });

    test('une recherche en cours garde le backend : rien n’est lancé', async () => {
        const window = deferred();
        bindings.SearchPositionIDs.mockReturnValue(window.promise);
        const search = loadPositionsByFilters({});
        await flush();

        const source = searchSource({ searchText: 'w>50' });
        expect(await settleList({ source, count: true, positionId: 1 })).toBeNull();
        expect(bindings.CountPositionsByFilters).not.toHaveBeenCalled();

        cancelSearch();
        window.resolve([]);
        await search;
    });
});
