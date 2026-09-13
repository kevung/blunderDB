/**
 * searchBoardOutsideEdit.test.js — #410
 *
 * Une recherche envoie toujours un plateau, que le backend lit comme une
 * structure « Au moins » dès qu'il porte un pion (sqlshared/search.go,
 * HasBoardFilter). En mode EDIT, c'est le plateau de requête dessiné : c'est
 * voulu. Hors EDIT — `s E>80` tapé sur la bibliothèque, `ss E>80` tapé en
 * collection ou en match — c'était la position affichée, ses trente pions
 * compris : la recherche ne rendait qu'elle (mesuré sur une vraie base :
 * 40 positions sans plateau, 1 avec, TestSearch_DisplayedBoardOutsideEdit).
 *
 * Verrouille la règle : la structure ne vient du plateau qu'en EDIT. Hors EDIT,
 * le plateau part sans pions, le reste de la position (dés, videau, score,
 * trait, type de décision) inchangé ; l'historique et la dernière recherche
 * enregistrent ce plateau-là, si bien que rejouer l'entrée ne ramène pas la
 * structure — sauf un `like`, dont le plateau est la cible (#404).
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
    LoadPositionIDsByFilters: vi.fn(() => Promise.resolve([2])),
    RankPositionIDsByFilters: vi.fn(() => Promise.resolve([])),
    LoadPositionsByIDs: vi.fn((/** @type {number[]} */ ids) => Promise.resolve(ids.map((id) => makePosition(id)))),
    SaveLastVisitedPosition: vi.fn(() => Promise.resolve()),
    SaveSearchHistory: vi.fn(() => Promise.resolve()),
    LoadSearchHistory: vi.fn(() => Promise.resolve([])),
    DeleteSearchHistoryEntry: vi.fn(() => Promise.resolve()),
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

import { LoadPositionIDsByFilters, RankPositionIDsByFilters, SaveSearchHistory } from '../../wailsjs/go/database/Database.js';
import { processCommand, initCommandProcessor } from '../commandProcessor.js';
import { statusBarModeStore, statusBarTextStore, currentPositionIndexStore, activeTabStore } from '../stores/uiStore.js';
import { positionStore, positionsStore, matchContextStore } from '../stores/positionStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { activeCollectionStore } from '../stores/collectionStore.js';
import { searchHistoryStore, lastSearchStore } from '../stores/searchHistoryStore.js';
import { filterLibraryStore } from '../stores/filterLibraryStore.js';
import { searchExcludePositionStore, emptySearchBoardPosition } from '../stores/searchExcludePositionStore.js';
import { MODE, enterEditMode, handleOpenCollection } from '../services/modeMachine.js';
import { loadPositionsByFilters, loadAllPositions } from '../services/positionService.js';
import SearchPanel from '../components/SearchPanel.svelte';

/**
 * A library position as the board shows it: thirty checkers on the points.
 * @param {number} id
 * @param {number} [playerOnRoll]
 * @returns {any}
 */
function makePosition(id, playerOnRoll = 0) {
    const points = Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 }));
    points[6] = { checkers: 5, color: 0 };
    points[8] = { checkers: 3, color: 0 };
    points[13] = { checkers: 5, color: 0 };
    points[24] = { checkers: 2, color: 0 };
    points[19] = { checkers: 5, color: 1 };
    points[17] = { checkers: 3, color: 1 };
    points[12] = { checkers: 5, color: 1 };
    points[1] = { checkers: 2, color: 1 };
    return {
        id,
        board: { points, bearoff: [0, 0] },
        cube: { owner: 1, value: 2 },
        dice: [6, 5],
        score: [3, 5],
        player_on_roll: playerOnRoll,
        decision_type: 0,
        has_jacoby: 0,
        has_beaver: 0
    };
}

const NO_MATCH = { isMatchMode: false, matchID: null, movePositions: [], currentIndex: 0, player1Name: '', player2Name: '' };
/** @param {any} board */
const checkers = (board) => board.points.reduce((/** @type {number} */ n, /** @type {any} */ p) => n + (p.checkers > 0 ? p.checkers : 0), 0);
const flush = async () => {
    for (let i = 0; i < 10; i++) await Promise.resolve();
    await new Promise((r) => setTimeout(r, 0));
};
/**
 * The SearchFilters payload of the last backend search.
 * @returns {any}
 */
const sent = () => vi.mocked(LoadPositionIDsByFilters).mock.calls.at(-1)?.[0];
/** @returns {any[]} */
const history = () => get(searchHistoryStore);

function showLibraryPosition(playerOnRoll = 0) {
    positionsStore.set([makePosition(1, playerOnRoll), makePosition(2, playerOnRoll), makePosition(3, playerOnRoll)]);
    positionStore.set(makePosition(2, playerOnRoll));
    currentPositionIndexStore.set(1);
    statusBarModeStore.set(MODE.NORMAL);
}

beforeEach(() => {
    vi.clearAllMocks();
    databasePathStore.set('/fake/db.sqlite');
    statusBarModeStore.set(MODE.NORMAL);
    statusBarTextStore.set('');
    activeTabStore.set('analysis');
    currentPositionIndexStore.set(-1);
    positionsStore.set([]);
    matchContextStore.set({ ...NO_MATCH });
    activeCollectionStore.set(null);
    lastSearchStore.set(null);
    searchHistoryStore.set([]);
    filterLibraryStore.set([]);
    searchExcludePositionStore.set(emptySearchBoardPosition());
    initCommandProcessor({ onLoadPositionsByFilters: loadPositionsByFilters });
});

afterEach(async () => {
    cleanup();
    await loadAllPositions();
});

describe('hors EDIT, le plateau affiché n’est pas une structure', () => {
    test('bibliothèque : `s E>80` part sans pions, le reste de la position gardé', async () => {
        showLibraryPosition();
        processCommand('s E>80');
        await flush();
        const payload = sent();
        expect(payload.moveErrorFilter).toBe('E>80');
        expect(checkers(payload.filter.board)).toBe(0);
        expect(payload.filter.dice).toEqual([6, 5]);
        expect(payload.filter.cube).toEqual({ owner: 1, value: 2 });
        expect(payload.filter.score).toEqual([3, 5]);
    });

    test('bibliothèque, joueur 2 au trait : sans pions, et toujours mis en miroir', async () => {
        showLibraryPosition(1);
        processCommand('s E>80');
        await flush();
        const payload = sent();
        expect(checkers(payload.filter.board)).toBe(0);
        expect(payload.filter.player_on_roll).toBe(0);
        expect(payload.filter.score).toEqual([5, 3]);
        expect(payload.filter.cube.owner).toBe(0);
    });

    test('collection : `ss E>80` part sans pions', async () => {
        activeCollectionStore.set(/** @type {any} */ ({ id: 4, name: 'Primes' }));
        handleOpenCollection({ id: 4, name: 'Primes' }, [makePosition(10), makePosition(20)]);
        processCommand('ss E>80');
        await flush();
        expect(sent().restrictToPositionIDs).toBe('10,20');
        expect(checkers(sent().filter.board)).toBe(0);
    });

    test('match : `ss E>80` part sans pions', async () => {
        matchContextStore.set(
            /** @type {any} */ ({
                isMatchMode: true,
                matchID: 7,
                movePositions: [101, 102].map((id, i) => ({ position: makePosition(id), move_number: i, game_number: 1, move_type: 'checker' })),
                currentIndex: 1,
                player1Name: 'Alice',
                player2Name: 'Bob'
            })
        );
        positionStore.set(makePosition(102));
        statusBarModeStore.set(MODE.MATCH);
        processCommand('ss E>80');
        await flush();
        expect(sent().restrictToPositionIDs).toBe('101,102');
        expect(checkers(sent().filter.board)).toBe(0);
    });

    test('l’historique et la dernière recherche enregistrent le plateau envoyé', async () => {
        showLibraryPosition();
        processCommand('s E>80');
        await flush();
        expect(checkers(JSON.parse(history()[0].position).board)).toBe(0);
        expect(checkers(JSON.parse(/** @type {any} */ (vi.mocked(SaveSearchHistory).mock.calls[0])[1]).board)).toBe(0);
        expect(checkers(JSON.parse(/** @type {any} */ (get(lastSearchStore)).position).board)).toBe(0);
    });

    test('un `like` garde le plateau affiché dans l’historique : c’est sa cible (#404)', async () => {
        showLibraryPosition();
        processCommand('s like E>80');
        await flush();
        expect(RankPositionIDsByFilters).toHaveBeenCalled();
        expect(checkers(JSON.parse(history()[0].position).board)).toBe(30);
    });
});

describe('en EDIT, le plateau de requête reste la structure', () => {
    test('`s E>80` sur un plateau dessiné envoie ses pions', async () => {
        showLibraryPosition();
        await enterEditMode();
        positionStore.update((p) => {
            p.board.points[6] = { checkers: 2, color: 0 };
            return p;
        });
        processCommand('s E>80');
        await flush();
        expect(checkers(sent().filter.board)).toBe(2);
    });

    test('rejouer une entrée faite hors EDIT ne ramène pas la structure', async () => {
        showLibraryPosition();
        processCommand('s E>80');
        await flush();
        const entry = history()[0];

        showLibraryPosition();
        activeTabStore.set('search');
        await enterEditMode();
        vi.mocked(LoadPositionIDsByFilters).mockClear();
        const utils = render(SearchPanel, { props: { onLoadPositionsByFilters: loadPositionsByFilters, onAddToFilterLibrary: vi.fn() } });
        await flush();
        // After mounting: the panel reloads the history from the database on mount.
        searchHistoryStore.set(/** @type {any} */ ([entry]));
        await tick();
        await fireEvent.click(utils.container.querySelectorAll('.sub-tab-btn')[1]);
        await tick();
        await fireEvent.dblClick(/** @type {Element} */ (utils.container.querySelector('.history-table tbody tr')));
        await flush();
        expect(LoadPositionIDsByFilters).toHaveBeenCalledTimes(1);
        expect(sent().moveErrorFilter).toBe('E>80');
        expect(checkers(sent().filter.board)).toBe(0);
    });
});
