/**
 * searchViewOrigin.test.js — une recherche reste attachée à la vue qui l'a lancée.
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

import { loadPositionsByFilters, cancelSearch, isSearching } from '../services/positionService.js';
import { positionsStore } from '../stores/positionStore.js';
import { statusBarTextStore, statusBarModeStore, currentPositionIndexStore } from '../stores/uiStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { isSettled } from '../stores/positionList.js';
import { viewStore } from '../stores/viewStore.js';

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
const key = () => /** @type {any} */ (get(statusBarTextStore))?.i18nKey;
const page = () => Array.from({ length: positionsStore.firstPageSize() }, (_, i) => i + 1);

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

describe('une recherche reste attachée à sa vue', () => {
    test('la première fenêtre qui revient dans une autre vue n’est pas montrée', async () => {
        const win = deferred();
        bindings.SearchPositionIDs.mockReturnValue(win.promise);

        const search = loadPositionsByFilters({});
        await flush();
        viewStore.addView();
        const shown = get(positionsStore);

        win.resolve(page());
        await search;
        expect(get(positionsStore)).toBe(shown);
        expect(isSearching()).toBe(false);
    });

    test('dans la vue d’origine, la fenêtre s’affiche', async () => {
        bindings.SearchPositionIDs.mockResolvedValue([7, 8]);
        await loadPositionsByFilters({});
        expect(get(positionsStore).length).toBe(2);
    });

    test('le compte d’une liste déjà montrée s’enregistre même si la vue change', async () => {
        const count = deferred();
        bindings.SearchPositionIDs.mockResolvedValue(page());
        bindings.CountPositionsByFilters.mockReturnValue(count.promise);

        const search = loadPositionsByFilters({});
        await flush();
        const shown = positionsStore.snapshotList();
        expect(isSettled(shown)).toBe(false);
        viewStore.addView();

        count.resolve(5000);
        await search;
        expect(isSettled(shown)).toBe(true);
    });

    test('un changement de vue pendant l’annulation d’une recherche relancée rend la ligne d’état', async () => {
        const first = deferred();
        const cancel = deferred();
        bindings.SearchPositionIDs.mockReturnValueOnce(first.promise).mockResolvedValue(page());
        bindings.CancelSearch.mockReturnValueOnce(cancel.promise);

        const one = loadPositionsByFilters({});
        await flush();
        const two = loadPositionsByFilters({});
        await flush();
        viewStore.addView();

        cancel.resolve(undefined);
        first.resolve(page());
        await Promise.all([one, two]);
        expect(isSearching()).toBe(false);
        expect(key()).not.toBe('status.searching');
    });
});
