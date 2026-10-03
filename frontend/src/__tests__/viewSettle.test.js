/**
 * viewSettle.test.js — une vue rouverte sur sa première page se compte et retrouve sa position
 * une fois affichée, même si on la ferme, la quitte ou en montre une autre pendant le compte.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { get } from 'svelte/store';

const bindings = vi.hoisted(() => ({
    SaveSearchHistory: vi.fn(() => Promise.resolve()),
    ListPositionIDs: vi.fn(() => Promise.resolve([])),
    CountPositions: vi.fn(() => Promise.resolve(0)),
    IndexOfPosition: vi.fn(() => Promise.resolve(-1)),
    LoadPositionsByIDs: vi.fn((ids) => Promise.resolve(ids.map((id) => ({ id })))),
    CountPositionsByFilters: vi.fn(),
    SearchPositionIDs: vi.fn(),
    IndexOfPositionByFilters: vi.fn(),
    CancelSearch: vi.fn(() => Promise.resolve()),
    LoadAnalysis: vi.fn(() => Promise.resolve(null)),
    LoadComment: vi.fn(() => Promise.resolve('')),
    SaveLastVisitedPosition: vi.fn(() => Promise.resolve())
}));

vi.mock('../../wailsjs/go/database/Database.js', () => bindings);
vi.mock('../services/sessionService.js', () => ({ saveSessionState: vi.fn() }));
vi.mock('../services/confirmService.js', () => ({ confirmAction: vi.fn(() => Promise.resolve(true)) }));

import { cancelSearch, loadPositionsByFilters } from '../services/positionService.js';
import { viewStore } from '../stores/viewStore.js';
import { positionsStore, searchSource } from '../stores/positionStore.js';
import { currentPositionIndexStore } from '../stores/uiStore.js';
import { databasePathStore } from '../stores/databaseStore.js';

/** @returns {{ promise: Promise<any>, resolve: (v: any) => void }} */
function deferred() {
    let resolve = (/** @type {any} */ _v) => {};
    const promise = new Promise((res) => (resolve = res));
    return { promise, resolve };
}

const flush = async () => {
    for (let i = 0; i < 5; i++) await new Promise((resolve) => setTimeout(resolve, 0));
};
const size = () => positionsStore.firstPageSize();
const firstPage = (base) => Array.from({ length: size() }, (_, i) => base + i);

// Two views, each a search result known by its first page, each showing a position past it.
const SESSION = JSON.stringify({
    nextViewId: 3,
    activeViewId: 1,
    views: [
        { id: 1, name: '#1', origin: { kind: 'search', payload: { searchText: 'a' } }, positionId: 1, positionIndex: 0 },
        { id: 2, name: '#2', origin: { kind: 'search', payload: { searchText: 'b' } }, positionId: 2, positionIndex: 0 }
    ]
});
const BASE = { a: 10_000, b: 20_000 };
/** @param {any} origin */
const resolveList = async (origin) => {
    const source = searchSource(origin.payload);
    return { source, length: size(), provisional: true, firstPage: firstPage(BASE[origin.payload.searchText]) };
};

/** @type {Record<string, ReturnType<typeof deferred>>} */
let counts;
/** @type {Record<string, ReturnType<typeof deferred>>} */
let ranks;

beforeEach(() => {
    vi.clearAllMocks();
    databasePathStore.set('/tmp/lib.db');
    counts = { a: deferred(), b: deferred() };
    ranks = { a: deferred(), b: deferred() };
    bindings.CountPositionsByFilters.mockImplementation((p) => counts[p.searchText].promise);
    bindings.IndexOfPositionByFilters.mockImplementation((p) => ranks[p.searchText].promise);
    bindings.SearchPositionIDs.mockImplementation(async (p, offset, limit) => firstPage(BASE[p.searchText]).slice(offset, offset + limit));
});

afterEach(() => cancelSearch());

describe('une vue rouverte sur sa première page', () => {
    test('fermée pendant son compte, la vue qui la remplace se compte et retrouve sa position', async () => {
        expect(await viewStore.deserialize(SESSION, resolveList)).toBe(true);
        await flush();
        viewStore.closeView(1);
        await flush();
        expect(bindings.CountPositionsByFilters).toHaveBeenCalledWith({ searchText: 'b' });

        counts.b.resolve(5000);
        ranks.b.resolve(4200);
        await flush();
        expect(get(positionsStore).length).toBe(5000);
        expect(get(currentPositionIndexStore)).toBe(4200);
    });

    test('quittée pendant son compte, elle retrouve longueur et position au retour', async () => {
        expect(await viewStore.deserialize(SESSION, resolveList)).toBe(true);
        await flush();
        viewStore.switchTo(2);
        await flush();
        // La vue montrée se compte, sans attendre celle qu'on a quittée.
        expect(bindings.CountPositionsByFilters).toHaveBeenCalledWith({ searchText: 'b' });
        counts.b.resolve(5000);
        ranks.b.resolve(4200);
        await flush();
        expect(get(positionsStore).length).toBe(5000);
        expect(get(currentPositionIndexStore)).toBe(4200);

        counts.a.resolve(3000);
        ranks.a.resolve(2500);
        await flush();
        viewStore.switchTo(1);
        await flush();
        expect(get(positionsStore).length).toBe(3000);
        expect(get(currentPositionIndexStore)).toBe(2500);
    });

    test('montrée pendant une recherche, elle se compte quand la recherche finit', async () => {
        expect(await viewStore.deserialize(SESSION, resolveList)).toBe(true);
        await flush();
        const window = deferred();
        bindings.SearchPositionIDs.mockReturnValueOnce(window.promise);
        const search = loadPositionsByFilters({});
        await flush();
        viewStore.switchTo(2);
        await flush();
        expect(bindings.CountPositionsByFilters).not.toHaveBeenCalledWith({ searchText: 'b' });

        window.resolve([]);
        await search;
        await flush();
        expect(bindings.CountPositionsByFilters).toHaveBeenCalledWith({ searchText: 'b' });
        counts.b.resolve(5000);
        ranks.b.resolve(4200);
        await flush();
        expect(get(currentPositionIndexStore)).toBe(4200);
    });
});
