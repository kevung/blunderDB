/**
 * pinnedFilters.test.js — #287, filtres favoris épinglés.
 *
 * Un filtre épinglé est à un geste : une pastille en haut du panneau de
 * recherche, ALT-n partout ailleurs. Ce test suit les deux chemins jusqu'au
 * backend et exige que le filtre lancé par ALT-n pose la même question que le
 * double-clic dans la bibliothèque — structure comprise, même hors du mode
 * ÉDITION où le plateau à l'écran n'en est pas une (#410).
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
    SetFilterPinned: vi.fn(() => Promise.resolve()),
    DeleteFilter: vi.fn(() => Promise.resolve()),
    LoadEditPosition: vi.fn(() => Promise.resolve(null)),
    LoadExcludePosition: vi.fn(() => Promise.resolve(null)),
    LoadPositionIDsByFilters: vi.fn(() => Promise.resolve([])),
    RankPositionIDsByFilters: vi.fn(() => Promise.resolve([])),
    LoadPositionsByIDs: vi.fn(() => Promise.resolve([])),
    ListPositionIDs: vi.fn(() => Promise.resolve([])),
    SaveComment: vi.fn(() => Promise.resolve())
}));

vi.mock('../../wailsjs/go/database/Database.js', async (importOriginal) => ({ ...(await importOriginal()), ...bindings }));
vi.mock('../../wailsjs/go/main/Config.js', async (importOriginal) => ({
    ...(await importOriginal()),
    GetLikeLimit: vi.fn(() => Promise.resolve(10)),
    GetLikeMaxDistance: vi.fn(() => Promise.resolve(0))
}));
vi.mock('../services/databaseService.js', () => ({
    setStatusBarMessage: vi.fn(),
    newDatabase: vi.fn(),
    openDatabase: vi.fn(),
    exitApp: vi.fn(),
    warningMessageStore: { subscribe: vi.fn(), set: vi.fn(), update: vi.fn() }
}));
vi.mock('../services/sessionService.js', () => ({ saveSessionState: vi.fn() }));
vi.mock('../services/confirmService.js', () => ({ confirmAction: vi.fn(() => Promise.resolve(true)) }));

import SearchPanel from '../components/SearchPanel.svelte';
import { handleKeyDown, pinnedFilterDigit, isAlwaysGlobal } from '../services/keyboardService.js';
import { runPinnedFilter, pinnedFilters } from '../services/filterLibraryService.js';
import { loadPositionsByFilters } from '../services/positionService.js';
import { filterLibraryStore } from '../stores/filterLibraryStore.js';
import { positionStore, emptyPosition } from '../stores/positionStore.js';
import { statusBarModeStore, statusBarTextStore, currentPositionIndexStore, activeTabStore } from '../stores/uiStore.js';
import { ankiViewModeStore, ankiReviewActionStore } from '../stores/ankiStore.js';
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

const LIBRARY = [
    { id: 1, name: 'prime', command: 's E>80', pinned: true },
    { id: 2, name: 'jamais', command: 's nc', pinned: false },
    { id: 3, name: 'cube', command: 's D', pinned: true }
];

function useLibrary(lib = LIBRARY, board = drawnBoard()) {
    bindings.LoadFilters.mockResolvedValue(lib);
    bindings.LoadEditPosition.mockResolvedValue(board ? JSON.stringify(board) : null);
}

function altDigit(n) {
    return new KeyboardEvent('keydown', { code: `Digit${n}`, key: String(n), altKey: true, cancelable: true, bubbles: true });
}

// Laisse les promesses du service s'écouler (LoadFilters, LoadEditPosition…).
async function settle() {
    for (let i = 0; i < 10; i++) await Promise.resolve();
    await tick();
}

beforeEach(() => {
    vi.clearAllMocks();
    databasePathStore.set('/tmp/lib.db');
    statusBarModeStore.set('NORMAL');
    statusBarTextStore.set('');
    currentPositionIndexStore.set(-1);
    activeTabStore.set('analysis');
    ankiViewModeStore.set('decks');
    positionStore.set(emptyPosition());
});

afterEach(() => {
    cleanup();
    filterLibraryStore.set([]);
});

describe('ALT-n lance le n-ième filtre épinglé', () => {
    test('le rang suit l’ordre de la bibliothèque, pas celui des épingles', () => {
        expect(pinnedFilters(LIBRARY).map((f) => f.name)).toEqual(['prime', 'cube']);
    });

    test('ALT-2 lance le deuxième épinglé, avec son nom pour la structure', async () => {
        useLibrary();
        const event = altDigit(2);
        handleKeyDown(event);
        await settle();
        expect(event.defaultPrevented).toBe(true);
        expect(bindings.LoadEditPosition).toHaveBeenCalledWith('cube');
        expect(bindings.LoadPositionIDsByFilters).toHaveBeenCalledTimes(1);
    });

    test('un rang sans épingle le dit, sans rien chercher', async () => {
        useLibrary();
        await runPinnedFilter(3);
        expect(bindings.LoadPositionIDsByFilters).not.toHaveBeenCalled();
        expect(get(statusBarTextStore)).toEqual(tMsg('search.noPinnedAt', { n: 3 }));
    });

    test('hors des modes NORMAL et ÉDITION, la recherche est refusée', async () => {
        useLibrary();
        statusBarModeStore.set('MATCH');
        await runPinnedFilter(1);
        expect(bindings.LoadPositionIDsByFilters).not.toHaveBeenCalled();
        expect(get(statusBarTextStore)).toEqual(tMsg('commands.searchRequiresMode'));
    });

    test('pendant une révision Anki, ALT-1 ne note pas la carte et ne cherche rien', async () => {
        useLibrary();
        activeTabStore.set('anki');
        ankiViewModeStore.set('review');
        ankiReviewActionStore.set(null);
        handleKeyDown(altDigit(1));
        await settle();
        expect(get(ankiReviewActionStore)).toBe(null);
        expect(bindings.LoadFilters).not.toHaveBeenCalled();
    });

    test('ALT-chiffre est global ; le chiffre seul, CTRL-chiffre et le pavé numérique ne le sont pas pour lui', () => {
        expect(pinnedFilterDigit(altDigit(4))).toBe(4);
        expect(isAlwaysGlobal(altDigit(4))).toBe(true);
        expect(pinnedFilterDigit(new KeyboardEvent('keydown', { code: 'Digit4' }))).toBe(0);
        expect(pinnedFilterDigit(new KeyboardEvent('keydown', { code: 'Digit4', altKey: true, ctrlKey: true }))).toBe(0);
        expect(pinnedFilterDigit(new KeyboardEvent('keydown', { code: 'Numpad4', altKey: true }))).toBe(0);
        expect(pinnedFilterDigit(new KeyboardEvent('keydown', { code: 'Digit0', altKey: true }))).toBe(0);
    });
});

describe('ALT-n pose la question du double-clic', () => {
    test('en mode NORMAL, la structure enregistrée voyage comme au panneau', async () => {
        useLibrary();

        // Le double-clic dans la bibliothèque, là où vit le panneau : ÉDITION.
        statusBarModeStore.set('EDIT');
        filterLibraryStore.set(LIBRARY);
        let pending;
        const { container } = render(SearchPanel, {
            props: {
                onLoadPositionsByFilters: (o) => (pending = loadPositionsByFilters(o)),
                onAddToFilterLibrary: vi.fn()
            }
        });
        await tick();
        await fireEvent.click(container.querySelectorAll('.sub-tab-btn')[2]);
        await tick();
        await fireEvent.dblClick(container.querySelector('.saved-item'));
        for (let i = 0; i < 10 && !pending; i++) await tick();
        await pending;
        expect(bindings.LoadPositionIDsByFilters).toHaveBeenCalledTimes(1);
        const fromPanel = bindings.LoadPositionIDsByFilters.mock.calls[0];
        cleanup();
        vi.clearAllMocks();
        useLibrary();

        // ALT-1 en feuilletant : le plateau à l'écran est une autre position.
        statusBarModeStore.set('NORMAL');
        positionStore.set(emptyPosition());
        await runPinnedFilter(1);
        expect(bindings.LoadPositionIDsByFilters).toHaveBeenCalledTimes(1);
        const fromAlt = bindings.LoadPositionIDsByFilters.mock.calls[0];
        expect(fromAlt).toEqual(fromPanel);
        expect(fromAlt[0].filter.board.points[6].checkers).toBe(5);
    });
});

describe('le panneau de recherche', () => {
    async function mount() {
        const utils = render(SearchPanel, { props: { onLoadPositionsByFilters: vi.fn(), onAddToFilterLibrary: vi.fn() } });
        await tick();
        return utils;
    }

    test('sans épingle, pas de barre', async () => {
        filterLibraryStore.set(LIBRARY.map((f) => ({ ...f, pinned: false })));
        const { container } = await mount();
        expect(container.querySelector('[data-testid="pinned-bar"]')).toBeNull();
    });

    test('les épinglés sont en haut, numérotés comme ALT-n, sur chaque sous-onglet', async () => {
        filterLibraryStore.set(LIBRARY);
        const { container } = await mount();
        for (const index of [0, 1, 2]) {
            await fireEvent.click(container.querySelectorAll('.sub-tab-btn')[index]);
            await tick();
            const chips = [...container.querySelectorAll('.pinned-chip')].map((c) => c.textContent.replace(/\s+/g, ' ').trim());
            expect(chips).toEqual(['1prime', '2cube']);
        }
    });

    test('l’étoile d’une ligne épingle, puis la bibliothèque est relue', async () => {
        const lib = LIBRARY.map((f) => ({ ...f, pinned: false }));
        filterLibraryStore.set(lib);
        bindings.LoadFilters.mockResolvedValue([{ ...lib[0], pinned: true }, lib[1], lib[2]]);
        const { container } = await mount();
        await fireEvent.click(container.querySelectorAll('.sub-tab-btn')[2]);
        await tick();
        const star = container.querySelector('.saved-item .pin-btn');
        expect(star.getAttribute('aria-pressed')).toBe('false');
        await fireEvent.click(star);
        await settle();
        expect(bindings.SetFilterPinned).toHaveBeenCalledWith(1, true);
        expect(container.querySelector('.saved-item .pin-btn').getAttribute('aria-pressed')).toBe('true');
        expect(container.querySelectorAll('.pinned-chip')).toHaveLength(1);
    });
});
