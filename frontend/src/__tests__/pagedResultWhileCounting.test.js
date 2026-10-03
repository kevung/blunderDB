/**
 * pagedResultWhileCounting.test.js — une action sur « tout le résultat » ne s'arrête pas à la
 * première page d'une recherche encore en train d'être comptée : l'export, son compteur et la
 * recherche dans les résultats lisent le résultat jusqu'au bout.
 */

import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

const TOTAL = 1500;
const all = Array.from({ length: TOTAL }, (_, i) => i + 1);

vi.mock('../../wailsjs/go/gui/App.js', () => ({
    OpenExportDatabaseDialog: vi.fn(() => Promise.resolve('/tmp/export.db')),
    ShowAlert: vi.fn(() => Promise.resolve())
}));

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    SearchPositionIDs: vi.fn((_f, offset = 0, limit = 0) => Promise.resolve(all.slice(offset, limit > 0 ? offset + limit : undefined))),
    CountPositionsByFilters: vi.fn(() => new Promise(() => {})),
    IndexOfPositionByFilters: vi.fn(() => Promise.resolve(-1)),
    LoadPositionsByIDs: vi.fn((ids) => Promise.resolve(ids.map((id) => ({ id })))),
    ExportDatabase: vi.fn(() => Promise.resolve()),
    CollectionCoverage: vi.fn(() => Promise.resolve({})),
    LoadMetadata: vi.fn(() => Promise.resolve({})),
    GetAllMatches: vi.fn(() => Promise.resolve([])),
    GetAllCollections: vi.fn(() => Promise.resolve([])),
    GetAllTournaments: vi.fn(() => Promise.resolve([]))
}));

vi.mock('../services/sessionService.js', () => ({ saveSessionState: vi.fn() }));

import { positionsStore, searchSource, listedIds } from '../stores/positionStore.js';
import { withDisplayedPositionIDs } from '../services/positionService.js';
import { exportDatabase } from '../services/exportService.js';
import { exportPositionCountStore, resetExportState } from '../stores/exportModalStore.js';
import { databasePathStore } from '../stores/databaseStore.js';

beforeEach(async () => {
    resetExportState();
    databasePathStore.set('/tmp/lib.db');
    const source = searchSource({});
    positionsStore.adoptFirstPage(source, await source.window(0, positionsStore.firstPageSize()));
    expect(get(positionsStore).length).toBe(positionsStore.firstPageSize());
});

describe('un résultat dont le compte n’est pas encore revenu', () => {
    test('listedIds le lit jusqu’au bout', async () => {
        expect((await listedIds()).length).toBe(TOTAL);
    });

    test('la recherche dans les résultats reçoit tous ses ids', async () => {
        const ids = await withDisplayedPositionIDs((displayed) => displayed);
        expect(ids?.length).toBe(TOTAL);
    });

    test('le compteur de l’export dit ce que l’export enverra', async () => {
        await exportDatabase();
        expect(get(exportPositionCountStore)).toBe(TOTAL);
    });
});
