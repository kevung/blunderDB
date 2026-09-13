/**
 * subSearchDisplayedList.test.js — #410
 *
 * `ss` cherche dans la liste affichée, quel que soit le chemin :
 *
 *   - en collection, tapé directement ou après TAB : les positions de la collection ;
 *   - en match, tapé directement ou après TAB : les positions du match (pas la
 *     bibliothèque restée derrière le damier de requête) ;
 *   - la case « Rechercher dans les résultats actuels » du panneau suit la même règle ;
 *   - `s` reste refusé en collection et en match, en disant pourquoi ;
 *   - quitter les résultats d'une sous-recherche ramène à la collection entière,
 *     ou au match sur le coup étudié, sur la même position.
 *
 * Stores Svelte réels, machine à modes réelle ; seules les liaisons Wails et les
 * E/S de session sont mockées (stratégie de modeMachine.test.js).
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { get } from 'svelte/store';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

vi.mock('../../wailsjs/go/database/Database.js', async (importOriginal) => ({
    ...(await importOriginal()),
    ListPositionIDs: vi.fn(() => Promise.resolve([1, 2, 3])),
    LoadAnalysis: vi.fn(() => Promise.resolve(null)),
    LoadComment: vi.fn(() => Promise.resolve('')),
    LoadPositionIDsByFilters: vi.fn(() => Promise.resolve([])),
    LoadPositionsByIDs: vi.fn((/** @type {number[]} */ ids) => Promise.resolve(ids.map((id) => makePosition(id)))),
    SaveLastVisitedPosition: vi.fn(() => Promise.resolve()),
    SaveSearchHistory: vi.fn(() => Promise.resolve()),
    LoadSearchHistory: vi.fn(() => Promise.resolve([])),
    LoadFilters: vi.fn(() => Promise.resolve([])),
    LoadEditPosition: vi.fn(() => Promise.resolve(null)),
    LoadExcludePosition: vi.fn(() => Promise.resolve(null)),
    GetMatchMovePositions: vi.fn(() => Promise.resolve([]))
}));

vi.mock('../../wailsjs/go/main/Config.js', async (importOriginal) => ({
    ...(await importOriginal()),
    GetLikeLimit: vi.fn(() => Promise.resolve(0)),
    GetLikeMaxDistance: vi.fn(() => Promise.resolve(0))
}));

vi.mock('../services/sessionService.js', () => ({ saveSessionState: vi.fn() }));

vi.mock('../services/importService.js', () => ({
    importDatabase: vi.fn(),
    importPosition: vi.fn(),
    importFolder: vi.fn(),
    pastePosition: vi.fn()
}));

import { LoadPositionIDsByFilters } from '../../wailsjs/go/database/Database.js';
import { processCommand, initCommandProcessor } from '../commandProcessor.js';
import { translate, resolveStatusMessage } from '../i18n';
import { statusBarModeStore, statusBarTextStore, currentPositionIndexStore, activeTabStore } from '../stores/uiStore.js';
import { positionStore, positionsStore, matchContextStore } from '../stores/positionStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { activeCollectionStore, collectionPositionsStore, selectedCollectionStore } from '../stores/collectionStore.js';
import { lastSearchStore } from '../stores/searchHistoryStore.js';
import { MODE, enterEditMode, exitEditMode, handleOpenCollection, leaveSubSearchResults, displayedPositionIDs } from '../services/modeMachine.js';
import { loadPositionsByFilters, loadAllPositions } from '../services/positionService.js';
import { handleKeyDown } from '../services/keyboardService.js';
import SearchPanel from '../components/SearchPanel.svelte';

// ── Helpers ───────────────────────────────────────────────────────────────────

/**
 * @param {number} id
 * @returns {any}
 */
function makePosition(id) {
    const points = Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 }));
    points[6] = { checkers: 5, color: 0 };
    return {
        id,
        board: { points, bearoff: [0, 0] },
        cube: { owner: -1, value: 0 },
        dice: [3, 1],
        score: [-1, -1],
        player_on_roll: 0,
        decision_type: 0,
        has_jacoby: 0,
        has_beaver: 0
    };
}

const NO_MATCH = { isMatchMode: false, matchID: null, movePositions: [], currentIndex: 0, player1Name: '', player2Name: '' };
/** @type {any} */
const COLLECTION = { id: 4, name: 'Primes' };

const statusText = () => resolveStatusMessage(get(statusBarTextStore), translate);
const flush = async () => {
    for (let i = 0; i < 10; i++) await Promise.resolve();
    await new Promise((r) => setTimeout(r, 0));
};

/** La bibliothèque derrière : trois positions, en NORMAL. */
function setLibrary() {
    positionsStore.set([makePosition(1), makePosition(2), makePosition(3)]);
    positionStore.set(makePosition(2));
    currentPositionIndexStore.set(1);
    statusBarModeStore.set(MODE.NORMAL);
}

/** Une collection ouverte sur sa troisième position (id 30). */
function openCollection() {
    activeCollectionStore.set(COLLECTION);
    handleOpenCollection(COLLECTION, [makePosition(10), makePosition(20), makePosition(30)]);
    currentPositionIndexStore.set(2);
    positionStore.set(makePosition(30));
}

/** Un match étudié sur son troisième coup (id 103) ; la bibliothèque reste derrière. */
function openMatch() {
    setLibrary();
    /** @type {any} */
    const ctx = {
        isMatchMode: true,
        matchID: 7,
        movePositions: [101, 102, 103, 102].map((id, i) => ({ position: makePosition(id), move_number: i, game_number: 1, move_type: 'checker' })),
        currentIndex: 2,
        player1Name: 'Alice',
        player2Name: 'Bob'
    };
    matchContextStore.set(ctx);
    positionStore.set(makePosition(103));
    statusBarModeStore.set(MODE.MATCH);
    return ctx;
}

/** @type {import('vitest').Mock} */
let onLoadPositionsByFilters;

beforeEach(() => {
    vi.clearAllMocks();
    databasePathStore.set('/fake/db.sqlite');
    statusBarModeStore.set(MODE.NORMAL);
    statusBarTextStore.set('');
    activeTabStore.set('analysis');
    currentPositionIndexStore.set(-1);
    positionStore.set(makePosition(0));
    positionsStore.set([]);
    matchContextStore.set({ ...NO_MATCH });
    activeCollectionStore.set(null);
    selectedCollectionStore.set(null);
    collectionPositionsStore.set([]);
    lastSearchStore.set(null);
    onLoadPositionsByFilters = vi.fn();
    initCommandProcessor({ onLoadPositionsByFilters: (/** @type {any} */ opts) => onLoadPositionsByFilters(opts) });
});

afterEach(async () => {
    cleanup();
    // Rien ne doit survivre d'un test à l'autre : la bibliothèque rechargée
    // oublie tout instantané de sous-recherche.
    await loadAllPositions();
    document.body.innerHTML = '';
});

// ── La liste affichée ─────────────────────────────────────────────────────────

describe('ss cherche dans la liste affichée (#410)', () => {
    test('collection, tapé directement : les identifiants de la collection', () => {
        openCollection();
        processCommand('ss E>80');
        expect(onLoadPositionsByFilters).toHaveBeenCalledTimes(1);
        expect(onLoadPositionsByFilters.mock.calls[0][0].restrictToPositionIDs).toBe('10,20,30');
    });

    test('collection, après TAB : les identifiants de la collection', async () => {
        openCollection();
        await enterEditMode();
        processCommand('ss E>80');
        expect(onLoadPositionsByFilters.mock.calls[0][0].restrictToPositionIDs).toBe('10,20,30');
    });

    test('match, tapé directement : les positions du match, chacune une fois', () => {
        openMatch();
        processCommand('ss E>80');
        expect(onLoadPositionsByFilters).toHaveBeenCalledTimes(1);
        expect(onLoadPositionsByFilters.mock.calls[0][0].restrictToPositionIDs).toBe('101,102,103');
    });

    test('match, après TAB : le match, pas la bibliothèque restée derrière', async () => {
        openMatch();
        await enterEditMode();
        processCommand('ss E>80');
        expect(onLoadPositionsByFilters.mock.calls[0][0].restrictToPositionIDs).toBe('101,102,103');
    });

    test('bibliothèque ou résultats : la liste de positionsStore, comme avant', () => {
        setLibrary();
        processCommand('ss E>80');
        expect(onLoadPositionsByFilters.mock.calls[0][0].restrictToPositionIDs).toBe('1,2,3');
    });

    test('displayedPositionIDs est la source unique : elle suit le mode', async () => {
        openMatch();
        expect(displayedPositionIDs()).toEqual([101, 102, 103]);
        await enterEditMode();
        expect(displayedPositionIDs()).toEqual([101, 102, 103]);
        await exitEditMode();
        expect(displayedPositionIDs()).toEqual([101, 102, 103]);
    });
});

describe('s reste refusé en collection et en match, en disant pourquoi', () => {
    test('en collection', () => {
        openCollection();
        processCommand('s E>80');
        expect(onLoadPositionsByFilters).not.toHaveBeenCalled();
        expect(statusText()).toMatch(/collection/i);
        expect(statusText()).toMatch(/\bss\b/);
    });

    test('en match', () => {
        openMatch();
        processCommand('s E>80');
        expect(onLoadPositionsByFilters).not.toHaveBeenCalled();
        expect(statusText()).toMatch(/match/i);
        expect(statusText()).toMatch(/\bss\b/);
    });
});

describe('la case « Rechercher dans les résultats actuels » suit la même règle', () => {
    async function searchFromPanel() {
        const utils = render(SearchPanel, { props: { onLoadPositionsByFilters, onAddToFilterLibrary: vi.fn() } });
        await tick();
        // Checked, not toggled: the panel restores the box from the last search's state.
        const box = /** @type {HTMLInputElement} */ (utils.container.querySelector('.search-in-results input[type="checkbox"]'));
        if (!box.checked) await fireEvent.click(box);
        await fireEvent.click(/** @type {Element} */ (utils.container.querySelector('.btn-search')));
        await tick();
        expect(onLoadPositionsByFilters).toHaveBeenCalledTimes(1);
        return onLoadPositionsByFilters.mock.calls[0][0];
    }

    test('entré depuis un match : les positions du match', async () => {
        openMatch();
        activeTabStore.set('search');
        await enterEditMode();
        expect((await searchFromPanel()).restrictToPositionIDs).toBe('101,102,103');
    });

    test('entré depuis une collection : les positions de la collection', async () => {
        openCollection();
        activeTabStore.set('search');
        await enterEditMode();
        expect((await searchFromPanel()).restrictToPositionIDs).toBe('10,20,30');
    });
});

// ── Sortie des résultats ──────────────────────────────────────────────────────

describe('quitter les résultats d’une sous-recherche ramène au mode d’origine', () => {
    beforeEach(() => {
        initCommandProcessor({ onLoadPositionsByFilters: loadPositionsByFilters });
    });

    /**
     * @param {string} command
     * @param {number[]} resultIds
     */
    async function subSearch(command, resultIds) {
        vi.mocked(LoadPositionIDsByFilters).mockResolvedValueOnce(resultIds);
        processCommand(command);
        await flush();
        expect(get(statusBarModeStore)).toBe(MODE.NORMAL);
        expect(get(positionsStore).ids).toEqual(resultIds);
    }

    function expectCollectionBack() {
        expect(get(statusBarModeStore)).toBe(MODE.COLLECTION);
        expect(get(activeCollectionStore)).toEqual(COLLECTION);
        expect(get(positionsStore).ids).toEqual([10, 20, 30]);
        expect(get(currentPositionIndexStore)).toBe(2);
        expect(get(positionStore).id).toBe(30);
    }

    function expectMatchBack() {
        expect(get(statusBarModeStore)).toBe(MODE.MATCH);
        const ctx = get(matchContextStore);
        expect(ctx.isMatchMode).toBe(true);
        expect(ctx.matchID).toBe(7);
        expect(ctx.currentIndex).toBe(2);
        expect(get(positionStore).id).toBe(103);
    }

    test('collection, directement : la collection entière, sur la même position', async () => {
        openCollection();
        await subSearch('ss E>80', [20]);
        expect(await leaveSubSearchResults()).toBe(true);
        expectCollectionBack();
    });

    test('collection, après TAB : la position de la collection, pas le damier de requête', async () => {
        openCollection();
        await enterEditMode();
        await subSearch('ss E>80', [10, 20]);
        expect(await leaveSubSearchResults()).toBe(true);
        expectCollectionBack();
        expect(get(positionStore).board.points[6].checkers).toBe(5);
    });

    test('match, directement : le match sur le coup étudié', async () => {
        openMatch();
        await subSearch('ss E>80', [102]);
        expect(await leaveSubSearchResults()).toBe(true);
        expectMatchBack();
    });

    test('match, après TAB : le match sur le coup étudié', async () => {
        openMatch();
        await enterEditMode();
        await subSearch('ss E>80', [101, 103]);
        expect(await leaveSubSearchResults()).toBe(true);
        expectMatchBack();
    });

    test('une sous-recherche dans les résultats garde l’origine : on revient à la collection', async () => {
        openCollection();
        await subSearch('ss E>80', [20, 30]);
        await subSearch('ss p<100', [30]);
        expect(await leaveSubSearchResults()).toBe(true);
        expectCollectionBack();
    });

    test('une recherche dans toute la bibliothèque oublie l’origine', async () => {
        openCollection();
        await subSearch('ss E>80', [20]);
        await subSearch('s p<100', [1, 2]);
        expect(await leaveSubSearchResults()).toBe(false);
        expect(get(statusBarModeStore)).toBe(MODE.NORMAL);
    });

    test('une liste remplacée par un autre geste n’est plus celle des résultats : rien ne se passe', async () => {
        openMatch();
        await subSearch('ss E>80', [102]);
        positionsStore.setIds([1, 2, 3]);
        expect(await leaveSubSearchResults()).toBe(false);
        expect(get(statusBarModeStore)).toBe(MODE.NORMAL);
    });

    test('après un rechargement de la bibliothèque, il n’y a plus rien à quitter', async () => {
        openCollection();
        await subSearch('ss E>80', [20]);
        await loadAllPositions();
        expect(await leaveSubSearchResults()).toBe(false);
    });

    test('sans sous-recherche, rien à quitter', async () => {
        setLibrary();
        expect(await leaveSubSearchResults()).toBe(false);
        expect(get(statusBarModeStore)).toBe(MODE.NORMAL);
    });

    test('la barre d’état dit comment revenir', async () => {
        openCollection();
        await subSearch('ss E>80', [20]);
        expect(statusText()).toMatch(/Esc/);
    });

    describe('Échap sur le plateau', () => {
        const press = () => {
            const event = new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, cancelable: true });
            handleKeyDown(event);
        };

        test('revient à la collection', async () => {
            openCollection();
            await subSearch('ss E>80', [20]);
            /** @type {HTMLElement} */ (document.activeElement)?.blur?.();
            press();
            await flush();
            expectCollectionBack();
        });

        test('focus dans un panneau : Échap appartient au panneau, on reste dans les résultats', async () => {
            openMatch();
            await subSearch('ss E>80', [102]);
            const wrapper = document.createElement('div');
            wrapper.className = 'panel-wrapper';
            const inner = document.createElement('section');
            inner.className = 'analysis-panel';
            inner.tabIndex = -1;
            wrapper.appendChild(inner);
            document.body.appendChild(wrapper);
            inner.focus();
            press();
            await flush();
            expect(get(statusBarModeStore)).toBe(MODE.NORMAL);
            expect(get(positionsStore).ids).toEqual([102]);
        });
    });
});
