/**
 * AnkiPanel.exportApkg.test.js — a deck row's export button asks where to
 * write the package, then exports that deck in the interface language; a
 * cancelled dialog exports nothing.
 */
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    CreateAnkiDeck: vi.fn(() => Promise.resolve(1)),
    GetAllAnkiDecks: vi.fn(() => Promise.resolve([])),
    UpdateAnkiDeck: vi.fn(() => Promise.resolve()),
    UpdateAnkiDeckParams: vi.fn(() => Promise.resolve()),
    DeleteAnkiDeck: vi.fn(() => Promise.resolve()),
    SyncAnkiDeck: vi.fn(() => Promise.resolve()),
    SyncAnkiDeckWithPositions: vi.fn(() => Promise.resolve()),
    GetAnkiDeckStats: vi.fn(() => Promise.resolve({ newCount: 2, learningCount: 0, reviewCount: 1, totalCount: 3, dueCount: 2 })),
    CountAnkiDeckPositions: vi.fn(() => Promise.resolve(1)),
    ListAnkiDeckPositionIDs: vi.fn((/** @type {number} */ _deck, offset = 0, limit = 0) => Promise.resolve([10].slice(offset, limit > 0 ? offset + limit : undefined))),
    IndexOfAnkiDeckPosition: vi.fn((/** @type {number} */ _deck, /** @type {number} */ id) => Promise.resolve([10].indexOf(id))),
    CountAnkiDeckFilteredPositions: vi.fn(() => Promise.resolve(1)),
    ListAnkiDeckFilteredPositionIDs: vi.fn(() => Promise.resolve([10])),
    IndexOfAnkiDeckFilteredPosition: vi.fn(() => Promise.resolve(0)),
    GetNextAnkiCard: vi.fn(() => Promise.resolve(null)),
    GetRandomAnkiCard: vi.fn(() => Promise.resolve(null)),
    ReviewAnkiCard: vi.fn(() => Promise.resolve(null)),
    ResetAnkiDeck: vi.fn(() => Promise.resolve()),
    GetAllCollections: vi.fn(() => Promise.resolve([{ id: 7, name: 'Openings', positionCount: 3 }])),
    LoadPositionIDsByFilters: vi.fn(() => Promise.resolve([])),
    LoadCommandHistory: vi.fn(() => Promise.resolve([])),
    SaveCommand: vi.fn(() => Promise.resolve()),
    ExportAnkiPackage: vi.fn(() => Promise.resolve({ name: 'Alpha', notes: 3, total: 3, truncated: false }))
}));

vi.mock('../../wailsjs/go/gui/App.js', () => ({
    OpenExportApkgDialog: vi.fn(() => Promise.resolve('/tmp/Alpha.apkg'))
}));

import * as db from '../../wailsjs/go/database/Database.js';
import { OpenExportApkgDialog } from '../../wailsjs/go/gui/App.js';
import AnkiPanel from '../components/AnkiPanel.svelte';
import { ankiDecksStore, selectedAnkiDeckStore, ankiViewModeStore } from '../stores/ankiStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { statusBarTextStore } from '../stores/uiStore.js';
import { language, tMsg } from '../i18n';

const DECKS = [{ id: 1, name: 'Alpha', description: '', sourceType: 'collection', sourceId: 7, cardCount: 3, newCount: 2, dueCount: 2 }];

beforeEach(() => {
    vi.clearAllMocks();
    databasePathStore.set('/tmp/test.db');
    vi.mocked(db.GetAllAnkiDecks).mockResolvedValue(/** @type {any} */ (DECKS));
    ankiDecksStore.set(DECKS);
    selectedAnkiDeckStore.set(null);
    ankiViewModeStore.set('list');
});

afterEach(() => {
    cleanup();
    databasePathStore.set('');
});

async function settle() {
    for (let i = 0; i < 6; i++) await tick();
}

describe('AnkiPanel .apkg export', () => {
    test('exports the deck to the chosen path, in the interface language', async () => {
        const { getByTestId } = render(AnkiPanel);
        await settle();
        await fireEvent.click(getByTestId('anki-export-apkg'));
        await settle();
        expect(OpenExportApkgDialog).toHaveBeenCalledWith('Alpha.apkg');
        expect(db.ExportAnkiPackage).toHaveBeenCalledWith(1, 0, get(language), '/tmp/Alpha.apkg');
        expect(get(statusBarTextStore)).toEqual(tMsg('anki.apkgExported', { name: 'Alpha', count: 3 }));
        // The button acts on its row; it does not open the deck.
        expect(get(selectedAnkiDeckStore)).toBeNull();
    });

    test('a cancelled dialog exports nothing', async () => {
        vi.mocked(OpenExportApkgDialog).mockResolvedValueOnce('');
        const { getByTestId } = render(AnkiPanel);
        await settle();
        await fireEvent.click(getByTestId('anki-export-apkg'));
        await settle();
        expect(db.ExportAnkiPackage).not.toHaveBeenCalled();
    });
});
