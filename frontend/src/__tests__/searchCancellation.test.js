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
vi.mock('../services/confirmService.js', () => ({ confirmAction: vi.fn(() => Promise.resolve(true)) }));

import { loadPositionsByFilters, cancelSearch, isSearching } from '../services/positionService.js';
import { positionsStore } from '../stores/positionStore.js';
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
