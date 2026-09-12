/**
 * modeMachine.test.js
 *
 * Les transitions de l'automate de modes (services/modeMachine.js), une par
 * bloc : chaque « bug 1 » / « bug 2 » que les commentaires du code citaient
 * est verrouillé ici.
 *
 * Stratégie (celle de positionService.enterEvalMode.test.js) : stores Svelte
 * réels, partagés avec les modules testés ; seules les liaisons Wails et
 * databaseService (E/S asynchrones) sont mockées.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ListPositionIDs: vi.fn(() => Promise.resolve([])),
    DeletePosition: vi.fn(),
    DeleteAnalysis: vi.fn(),
    UpdatePosition: vi.fn(),
    SaveAnalysis: vi.fn(),
    LoadAnalysis: vi.fn(() => Promise.resolve(null)),
    LoadPositionIDsByFilters: vi.fn(() => Promise.resolve([])),
    ComputeEPCFromPosition: vi.fn(() => Promise.resolve({})),
    SaveLastVisitedPosition: vi.fn(() => Promise.resolve()),
    GetLastVisitedMatch: vi.fn(() => Promise.resolve(null)),
    GetMatchMovePositions: vi.fn(() => Promise.resolve([])),
    SaveEditPosition: vi.fn(),
    SaveExcludePosition: vi.fn(),
    SaveFilter: vi.fn(),
    LoadComment: vi.fn(() => Promise.resolve(''))
}));

vi.mock('../services/databaseService.js', () => ({
    setStatusBarMessage: vi.fn(),
    warningMessageStore: { subscribe: vi.fn(), set: vi.fn(), update: vi.fn() }
}));

vi.mock('../services/sessionService.js', () => ({
    saveSessionState: vi.fn()
}));

import { ListPositionIDs, LoadAnalysis, SaveLastVisitedPosition, GetLastVisitedMatch, GetMatchMovePositions } from '../../wailsjs/go/database/Database.js';
import { setStatusBarMessage } from '../services/databaseService.js';
import { statusBarModeStore, statusBarTextStore, currentPositionIndexStore, activeTabStore, openPanels, openPanel, PANEL } from '../stores/uiStore.js';
import { positionStore, positionsStore, matchContextStore, lastVisitedMatchStore } from '../stores/positionStore.js';
import { analysisStore, selectedMoveStore } from '../stores/analysisStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { activeCollectionStore, collectionPositionsStore, selectedCollectionStore } from '../stores/collectionStore.js';
import { lastSearchStore } from '../stores/searchHistoryStore.js';

import {
    MODE,
    modeState,
    forgetContextBeforeEval,
    enterEditMode,
    exitEditMode,
    enterEvalMode,
    exitEvalMode,
    toggleEvalMode,
    enterTranscribeMode,
    exitTranscribeMode,
    sendPositionToEval,
    toggleMatchMode,
    handleOpenCollection,
    exitCollectionMode
} from '../services/modeMachine.js';
import * as positionService from '../services/positionService.js';

// ── Helpers ───────────────────────────────────────────────────────────────────

function makePosition(id, bearoff = [3, 3]) {
    return {
        id,
        board: { points: Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 })), bearoff },
        cube: { owner: -1, value: 1 },
        dice: [3, 1],
        score: [5, 5],
        player_on_roll: 0,
        decision_type: 0,
        has_jacoby: 0,
        has_beaver: 0
    };
}

function makeMatchContext(currentIndex = 1) {
    return {
        isMatchMode: true,
        matchID: 7,
        movePositions: [
            { position: makePosition(101), move_number: 0, game_number: 1, move_type: 'checker', checker_move: '', cube_action: '' },
            { position: makePosition(102), move_number: 1, game_number: 1, move_type: 'checker', checker_move: '8/5 6/5', cube_action: '' },
            { position: makePosition(103), move_number: 2, game_number: 1, move_type: 'checker', checker_move: '24/21', cube_action: '' }
        ],
        currentIndex,
        player1Name: 'Alice',
        player2Name: 'Bob'
    };
}

const NO_MATCH = { isMatchMode: false, matchID: null, movePositions: [], currentIndex: 0, player1Name: '', player2Name: '' };

/** Bibliothèque de trois positions, la deuxième affichée, mode NORMAL. */
function setLibrary() {
    const lib = [makePosition(1), makePosition(2), makePosition(3)];
    positionsStore.set(lib);
    // A clone, as showPosition() puts one on the board: the cache keeps the record.
    positionStore.set(structuredClone(lib[1]));
    currentPositionIndexStore.set(1);
    statusBarModeStore.set(MODE.NORMAL);
    return lib;
}

/** Partie en cours d'étude : mode MATCH sur le coup `currentIndex`. */
function setMatch(currentIndex = 1) {
    const ctx = makeMatchContext(currentIndex);
    matchContextStore.set(ctx);
    // A clone, as showPosition() puts one on the board: the match keeps its record.
    positionStore.set(structuredClone(ctx.movePositions[currentIndex].position));
    statusBarModeStore.set(MODE.MATCH);
    return ctx;
}

function resetStores() {
    databasePathStore.set('/fake/db.sqlite');
    statusBarModeStore.set(MODE.NORMAL);
    statusBarTextStore.set('');
    activeTabStore.set('analysis');
    currentPositionIndexStore.set(-1);
    positionStore.set(null);
    positionsStore.set([]);
    matchContextStore.set({ ...NO_MATCH });
    analysisStore.set({ checkerAnalysis: { moves: [] } });
    selectedMoveStore.set(null);
    openPanels.set(new Set());
    activeCollectionStore.set(null);
    selectedCollectionStore.set(null);
    collectionPositionsStore.set([]);
    lastSearchStore.set(null);
    lastVisitedMatchStore.set(null);
}

beforeEach(() => {
    vi.clearAllMocks();
    resetStores();
    // Un exit précédent peut avoir laissé un contexte : on repart d'un automate vierge.
    forgetContextBeforeEval();
});

afterEach(() => {
    vi.restoreAllMocks();
});

// ── Ré-exports ────────────────────────────────────────────────────────────────

describe('positionService ré-exporte les transitions', () => {
    test('les callers historiques (App, keyboardService) trouvent les mêmes fonctions', () => {
        expect(positionService.enterEditMode).toBe(enterEditMode);
        expect(positionService.exitEditMode).toBe(exitEditMode);
        expect(positionService.enterEvalMode).toBe(enterEvalMode);
        expect(positionService.exitEvalMode).toBe(exitEvalMode);
        expect(positionService.toggleEvalMode).toBe(toggleEvalMode);
        expect(positionService.sendPositionToEval).toBe(sendPositionToEval);
        expect(positionService.toggleMatchMode).toBe(toggleMatchMode);
        expect(positionService.exitCollectionMode).toBe(exitCollectionMode);
        expect(positionService.handleOpenCollection).toBe(handleOpenCollection);
        expect(positionService.enterTranscribeMode).toBe(enterTranscribeMode);
        expect(positionService.exitTranscribeMode).toBe(exitTranscribeMode);
    });

    test('modeState expose { mode, savedContext } et démarre vide', () => {
        expect(modeState()).toEqual({
            mode: MODE.NORMAL,
            savedContext: { beforeTranscribe: null, beforeEval: null, beforeEdit: null, evalSeed: null, lastEvalBoard: null }
        });
    });
});

// ── NORMAL → EDIT → NORMAL ────────────────────────────────────────────────────

describe('NORMAL → EDIT → NORMAL', () => {
    test('enterEditMode vide le damier, efface le coup sélectionné', async () => {
        setLibrary();
        selectedMoveStore.set({ move: '8/5 6/5' });

        await enterEditMode();

        expect(get(statusBarModeStore)).toBe(MODE.EDIT);
        expect(get(selectedMoveStore)).toBeNull();
        const board = get(positionStore);
        expect(board.board.bearoff).toEqual([15, 15]);
        expect(board.board.points.every((p) => p.checkers === 0)).toBe(true);
        expect(board.score).toEqual([7, 7]);
        expect(modeState().savedContext.beforeEdit, 'pas de partie à restaurer').toBeNull();
    });

    test('enterEditMode sans base ouverte est un no-op', async () => {
        databasePathStore.set('');
        setLibrary();
        await enterEditMode();
        expect(get(statusBarModeStore)).toBe(MODE.NORMAL);
        expect(get(positionStore).board.bearoff).toEqual([3, 3]);
    });

    test('exitEditMode revient en NORMAL et fait repasser l’index par -1 pour forcer le redessin', async () => {
        setLibrary();
        await enterEditMode();
        const seen = [];
        const orig = currentPositionIndexStore.set;
        vi.spyOn(currentPositionIndexStore, 'set').mockImplementation((v) => {
            seen.push(v);
            orig(v);
        });

        await exitEditMode();

        expect(get(statusBarModeStore)).toBe(MODE.NORMAL);
        expect(seen).toEqual([-1, 1]);
        expect(ListPositionIDs, 'la bibliothèque n’est pas rechargée').not.toHaveBeenCalled();
    });

    test('exitEditMode hors du mode EDIT ne touche à rien', async () => {
        setLibrary();
        await exitEditMode();
        expect(get(statusBarModeStore)).toBe(MODE.NORMAL);
        expect(get(currentPositionIndexStore)).toBe(1);
    });
});

// ── MATCH → EDIT → MATCH (bug 2) ──────────────────────────────────────────────

describe('MATCH → EDIT → MATCH (bug 2 : l’onglet recherche ne perd pas la partie étudiée)', () => {
    test('enterEditMode mémorise la partie, persiste le dernier coup visité et ne recharge pas la bibliothèque', async () => {
        setMatch(1);

        await enterEditMode();

        expect(get(statusBarModeStore)).toBe(MODE.EDIT);
        expect(SaveLastVisitedPosition).toHaveBeenCalledWith(7, 1);
        expect(get(matchContextStore).isMatchMode).toBe(false);
        expect(modeState().savedContext.beforeEdit).toMatchObject({ isMatchMode: true, matchID: 7, currentIndex: 1 });
        // C’est loadAllPositions() qui, en résolvant, basculait l’onglet sur
        // 'matches' et écrasait le mode : il ne doit pas être appelé.
        expect(ListPositionIDs).not.toHaveBeenCalled();
        expect(get(activeTabStore)).toBe('analysis');
    });

    test('exitEditMode restaure la partie et réaffiche le coup étudié (analyse rechargée)', async () => {
        setMatch(1);
        await enterEditMode();
        LoadAnalysis.mockResolvedValueOnce({ positionId: 102, checkerAnalysis: { moves: [{ move: '8/5 6/5' }] } });

        await exitEditMode();

        expect(get(statusBarModeStore)).toBe(MODE.MATCH);
        expect(get(matchContextStore)).toMatchObject({ isMatchMode: true, matchID: 7, currentIndex: 1 });
        expect(get(positionStore).id).toBe(102);
        expect(LoadAnalysis).toHaveBeenCalledWith(102);
        expect(get(analysisStore).playedMove).toBe('8/5 6/5');
        expect(get(statusBarTextStore)).toBe('Alice vs Bob');
        expect(modeState().savedContext.beforeEdit, 'consommé').toBeNull();
    });

    test('un instantané de partie périmé n’est jamais restauré dans une session EDIT entrée depuis NORMAL', async () => {
        setMatch(1);
        await enterEditMode();
        await exitEditMode();
        expect(get(statusBarModeStore)).toBe(MODE.MATCH);

        // Sortie de la partie, puis nouvelle session de recherche depuis la bibliothèque.
        matchContextStore.set({ ...NO_MATCH });
        setLibrary();
        await enterEditMode();
        expect(modeState().savedContext.beforeEdit).toBeNull();
        await exitEditMode();

        expect(get(statusBarModeStore)).toBe(MODE.NORMAL);
        expect(get(matchContextStore).isMatchMode).toBe(false);
    });
});

// ── NORMAL → EVAL → NORMAL ─────────────────────────────────────────────────────

describe('NORMAL → EVAL → NORMAL', () => {
    test('enterEvalMode sauvegarde la bibliothèque et pose le bearoff par défaut', () => {
        const lib = setLibrary();

        enterEvalMode();

        expect(get(statusBarModeStore)).toBe(MODE.EVAL);
        expect(get(positionsStore)).toHaveLength(1);
        expect(get(positionStore).board.bearoff).toEqual([0, 15]);
        expect(get(positionStore).id).toBe(0);
        expect(get(currentPositionIndexStore)).toBe(0);
        const { beforeEval } = modeState().savedContext;
        expect(beforeEval.mode).toBe(MODE.NORMAL);
        expect(beforeEval.ids).toEqual(lib.map((p) => p.id));
        expect(beforeEval.position.id).toBe(2);
        expect(beforeEval.positionIndex).toBe(1);
    });

    test('exitEvalMode restaure la liste et l’index, recharge l’analyse, vide le contexte', async () => {
        const lib = setLibrary();
        enterEvalMode();
        LoadAnalysis.mockResolvedValueOnce({ positionId: 2, checkerAnalysis: { moves: [] } });

        await exitEvalMode();

        expect(get(statusBarModeStore)).toBe(MODE.NORMAL);
        expect(get(positionsStore).ids).toEqual(lib.map((p) => p.id));
        expect(get(currentPositionIndexStore)).toBe(1);
        expect(get(positionStore).id).toBe(2);
        expect(LoadAnalysis).toHaveBeenCalledWith(2);
        expect(get(statusBarTextStore)).toBe('');
        expect(modeState().savedContext.beforeEval).toBeNull();
    });

    test('après forgetContextBeforeEval (rechargement de la bibliothèque), exitEvalMode recharge au lieu de restaurer', async () => {
        setLibrary();
        enterEvalMode();
        forgetContextBeforeEval();

        await exitEvalMode();

        expect(get(statusBarModeStore)).toBe(MODE.NORMAL);
        expect(ListPositionIDs).toHaveBeenCalledTimes(1);
    });
});

// ── MATCH → EVAL → MATCH (bug 2, bug 1) ────────────────────────────────────────

describe('MATCH → EVAL → MATCH', () => {
    test('exitEvalMode revient à la partie étudiée, contexte de match restauré (bug 2)', async () => {
        const ctx = setMatch(2);
        enterEvalMode();
        expect(get(statusBarModeStore)).toBe(MODE.EVAL);

        await exitEvalMode();

        expect(get(statusBarModeStore)).toBe(MODE.MATCH);
        expect(get(matchContextStore)).toEqual(ctx);
        expect(get(statusBarTextStore)).toBe('Alice vs Bob');
        expect(get(positionStore).id).toBe(103);
    });

    test('exitEvalMode repasse par showPosition : l’analyse est rechargée alors que l’effet de nav ne redessine plus en MATCH (bug 1)', async () => {
        setMatch(2);
        enterEvalMode();
        LoadAnalysis.mockResolvedValueOnce({ positionId: 103, checkerAnalysis: { moves: [{ move: '24/21' }] } });

        await exitEvalMode();

        expect(LoadAnalysis).toHaveBeenCalledWith(103);
        expect(get(analysisStore).checkerAnalysis.moves).toEqual([{ move: '24/21' }]);
        expect(get(analysisStore).playedMove, 'le coup joué vient du contexte de match').toBe('24/21');
    });

    test('le mode est restauré avant la position (l’effet EVAL ne doit pas voir la position restaurée sous le mode EVAL)', async () => {
        setMatch(1);
        enterEvalMode();
        const order = [];
        for (const [name, store] of Object.entries({ statusBarModeStore, positionStore })) {
            const orig = store.set;
            vi.spyOn(store, 'set').mockImplementation((v) => {
                order.push(name);
                orig(v);
            });
        }

        await exitEvalMode();

        expect(order.indexOf('statusBarModeStore')).toBeLessThan(order.indexOf('positionStore'));
    });
});

// ── EVAL ↔ EDIT ────────────────────────────────────────────────────────────────

describe('EVAL → EDIT : l’automate quitte Eval avant d’entrer en recherche', () => {
    test('enterEditMode depuis EVAL ne rebascule pas l’onglet (l’utilisateur a cliqué « recherche »)', async () => {
        setLibrary();
        activeTabStore.set('eval');
        enterEvalMode();
        activeTabStore.set('search');

        await enterEditMode();

        expect(get(statusBarModeStore)).toBe(MODE.EDIT);
        expect(get(activeTabStore), 'toggleEvalMode aurait renvoyé sur analysis').toBe('search');
        expect(modeState().savedContext.beforeEval, 'contexte EVAL consommé').toBeNull();
    });

    test('MATCH → EVAL → EDIT → MATCH : la partie survit aux deux brouillons', async () => {
        setMatch(1);
        enterEvalMode();

        await enterEditMode();
        expect(get(statusBarModeStore)).toBe(MODE.EDIT);
        expect(modeState().savedContext.beforeEdit).toMatchObject({ isMatchMode: true, matchID: 7, currentIndex: 1 });

        await exitEditMode();
        expect(get(statusBarModeStore)).toBe(MODE.MATCH);
        expect(get(matchContextStore).matchID).toBe(7);
        expect(get(positionStore).id).toBe(102);
    });
});

describe('EDIT → EVAL : l’automate quitte la recherche avant d’entrer dans Eval', () => {
    test('enterEvalMode depuis EDIT ne sauvegarde pas le damier vierge de la recherche', async () => {
        setMatch(1);
        await enterEditMode();
        expect(get(positionStore).board.bearoff).toEqual([15, 15]);

        await enterEvalMode();

        expect(get(statusBarModeStore)).toBe(MODE.EVAL);
        const { beforeEval, beforeEdit } = modeState().savedContext;
        expect(beforeEdit, 'la session EDIT est close').toBeNull();
        expect(beforeEval.mode).toBe(MODE.MATCH);
        expect(beforeEval.matchContext.matchID).toBe(7);
        expect(beforeEval.position.id, 'la position de la partie, pas le damier vierge').toBe(102);
        expect(beforeEval.position.board.bearoff, 'le damier de la partie, pas celui de la requête').toEqual([3, 3]);

        await exitEvalMode();
        expect(get(statusBarModeStore)).toBe(MODE.MATCH);
        expect(get(positionStore).id).toBe(102);
        expect(get(positionStore).board.bearoff).toEqual([3, 3]);
    });

    // #201 : App.svelte enchaîne exitEditMode() SANS await puis enterEvalMode()
    // quand l'onglet passe de « recherche » à « Eval ». Le damier vierge de la
    // recherche est un clone sous l'id de la position de bibliothèque ; le
    // redessin déclenché par l'index est asynchrone. La photo prise par
    // enterEvalMode était donc toujours le damier vierge, et la sortie d'Eval
    // le remettait à l'écran.
    test('depuis la bibliothèque, enchaînement d’App.svelte : la position revient sur le damier avant la photo', async () => {
        const lib = setLibrary();
        await enterEditMode();
        expect(get(positionStore).id).toBe(2);
        expect(get(positionStore).board.bearoff, 'le damier de la requête').toEqual([15, 15]);

        exitEditMode(); // sans await, comme App.svelte
        expect(get(positionStore).board.bearoff, 'restaurée depuis le cache, de façon synchrone').toEqual([3, 3]);
        expect(get(positionStore)).not.toBe(lib[1]);

        await enterEvalMode();
        const { beforeEval } = modeState().savedContext;
        expect(beforeEval.mode).toBe(MODE.NORMAL);
        expect(beforeEval.position.id).toBe(2);
        expect(beforeEval.position.board.bearoff).toEqual([3, 3]);
        expect(beforeEval.positionIndex).toBe(1);

        await exitEvalMode();
        expect(get(statusBarModeStore)).toBe(MODE.NORMAL);
        expect(get(positionStore).id).toBe(2);
        expect(get(positionStore).board.bearoff, 'jamais le damier vierge au retour').toEqual([3, 3]);
    });

    test('depuis la bibliothèque, appel direct : enterEvalMode attend la sortie de la recherche', async () => {
        setLibrary();
        await enterEditMode();

        await enterEvalMode();

        expect(get(statusBarModeStore)).toBe(MODE.EVAL);
        const { beforeEval, beforeEdit } = modeState().savedContext;
        expect(beforeEdit).toBeNull();
        expect(beforeEval.mode).toBe(MODE.NORMAL);
        expect(beforeEval.position.board.bearoff).toEqual([3, 3]);
        expect(beforeEval.ids).toEqual([1, 2, 3]);

        await exitEvalMode();
        expect(get(positionsStore).ids).toEqual([1, 2, 3]);
        expect(get(positionStore).board.bearoff).toEqual([3, 3]);
    });
});

// ── toggleEvalMode ─────────────────────────────────────────────────────────────

describe('toggleEvalMode', () => {
    test('hors EVAL : demande l’onglet Eval (App.svelte relaie vers enterEvalMode)', () => {
        setLibrary();
        toggleEvalMode();
        expect(get(activeTabStore)).toBe('eval');
        expect(get(statusBarModeStore), 'le mode change via l’effet d’onglet, pas ici').toBe(MODE.NORMAL);
    });

    test('en EVAL : sort et revient sur l’onglet analyse', () => {
        setLibrary();
        activeTabStore.set('eval');
        enterEvalMode();
        toggleEvalMode();
        expect(get(statusBarModeStore)).toBe(MODE.NORMAL);
        expect(get(activeTabStore)).toBe('analysis');
    });
});

// ── Entrée en EVAL depuis une position existante ───────────────────────────────

describe('une position existante entre dans le panneau Eval et en ressort', () => {
    test('exitEvalMode rend la position étudiée, pas le brouillon transmis', async () => {
        const lib = setLibrary();
        sendPositionToEval(lib[1]);
        expect(modeState().savedContext.evalSeed.id).toBe(0);
        enterEvalMode();
        expect(modeState().savedContext.evalSeed, 'graine consommée').toBeNull();
        expect(get(positionStore).id).toBe(0);
        expect(get(positionStore).board.bearoff).toEqual([3, 3]);

        get(positionStore).board.bearoff[0] = 9; // édition du brouillon
        await exitEvalMode();

        expect(get(statusBarModeStore)).toBe(MODE.NORMAL);
        expect(get(positionStore).id).toBe(2);
        expect(get(positionStore).board.bearoff, 'l’originale n’a pas bougé').toEqual([3, 3]);
        expect(get(positionsStore).ids).toEqual(lib.map((p) => p.id));
    });

    test('depuis une partie : la graine ouvre Eval et la sortie revient à la partie', async () => {
        const ctx = setMatch(1);
        sendPositionToEval(ctx.movePositions[1].position);
        expect(get(activeTabStore)).toBe('eval');
        enterEvalMode();
        expect(get(positionStore).id).toBe(0);

        await exitEvalMode();
        expect(get(statusBarModeStore)).toBe(MODE.MATCH);
        expect(get(matchContextStore).matchID).toBe(7);
    });
});

// ── MATCH ↔ NORMAL ────────────────────────────────────────────────────────────

describe('toggleMatchMode', () => {
    test('MATCH → NORMAL : persiste le dernier coup, vide le contexte, recharge la bibliothèque', async () => {
        setMatch(2);
        ListPositionIDs.mockResolvedValueOnce([1]);

        await toggleMatchMode();

        expect(SaveLastVisitedPosition).toHaveBeenCalledWith(7, 2);
        expect(get(statusBarModeStore)).toBe(MODE.NORMAL);
        expect(get(matchContextStore).isMatchMode).toBe(false);
        expect(ListPositionIDs).toHaveBeenCalledTimes(1);
    });

    test('MATCH → NORMAL : reste sur la position quittée quand la bibliothèque la contient (#201)', async () => {
        setMatch(1); // le coup étudié est la position 102
        ListPositionIDs.mockResolvedValueOnce([1, 102, 3]);

        await toggleMatchMode();

        expect(get(statusBarModeStore)).toBe(MODE.NORMAL);
        expect(get(positionsStore).ids).toEqual([1, 102, 3]);
        expect(get(currentPositionIndexStore), 'l’index de la position quittée, pas la dernière').toBe(1);
    });

    test('MATCH → NORMAL : sur la dernière position quand la position quittée n’est pas dans la liste', async () => {
        setMatch(1);
        ListPositionIDs.mockResolvedValueOnce([1, 2, 3]);

        await toggleMatchMode();

        expect(get(currentPositionIndexStore)).toBe(2);
    });

    test('NORMAL → MATCH : reprend la dernière partie visitée à son dernier coup', async () => {
        setLibrary();
        const ctx = makeMatchContext();
        GetLastVisitedMatch.mockResolvedValueOnce({ id: 7, player1_name: 'Alice', player2_name: 'Bob', last_visited_position: 2 });
        GetMatchMovePositions.mockResolvedValueOnce(ctx.movePositions);

        await toggleMatchMode();

        expect(get(statusBarModeStore)).toBe(MODE.MATCH);
        expect(get(matchContextStore)).toMatchObject({ isMatchMode: true, matchID: 7, currentIndex: 2, player1Name: 'Alice' });
        expect(get(positionStore).id).toBe(103);
        expect(get(analysisStore).playedMove).toBe('24/21');
        expect(get(lastVisitedMatchStore)).toEqual({ matchID: 7, currentIndex: 2, gameNumber: 1 });
    });

    test('depuis EVAL : le brouillon est abandonné, son contexte oublié', async () => {
        setLibrary();
        enterEvalMode();
        GetLastVisitedMatch.mockResolvedValueOnce({ id: 7, player1_name: 'Alice', player2_name: 'Bob', last_visited_position: 0 });
        GetMatchMovePositions.mockResolvedValueOnce(makeMatchContext().movePositions);

        await toggleMatchMode();

        expect(get(statusBarModeStore)).toBe(MODE.MATCH);
        expect(modeState().savedContext.beforeEval).toBeNull();
    });

    test('sans partie en base : reste en NORMAL', async () => {
        setLibrary();
        await toggleMatchMode();
        expect(get(statusBarModeStore)).toBe(MODE.NORMAL);
        expect(get(matchContextStore).isMatchMode).toBe(false);
    });

    test('GetLastVisitedMatch qui échoue avec "no matches" affiche le message dédié', async () => {
        setLibrary();
        GetLastVisitedMatch.mockRejectedValueOnce(new Error('no matches found'));

        await toggleMatchMode();

        expect(get(statusBarModeStore)).toBe(MODE.NORMAL);
        expect(setStatusBarMessage).toHaveBeenCalledWith({ i18nKey: 'status.noMatchesInDb', i18nParams: null });
    });

    test('GetLastVisitedMatch qui échoue autrement affiche le message générique', async () => {
        setLibrary();
        GetLastVisitedMatch.mockRejectedValueOnce(new Error('bridge disconnected'));

        await toggleMatchMode();

        expect(get(statusBarModeStore)).toBe(MODE.NORMAL);
        expect(setStatusBarMessage).toHaveBeenCalledWith({ i18nKey: 'status.errorEnteringMatchMode', i18nParams: null });
    });
});

// ── COLLECTION → NORMAL ───────────────────────────────────────────────────────

describe('COLLECTION → NORMAL', () => {
    test('handleOpenCollection entre en COLLECTION sur la première position', () => {
        setMatch(1);
        const coll = [makePosition(2), makePosition(3)];

        handleOpenCollection({ name: 'Backgames' }, coll);

        expect(get(statusBarModeStore)).toBe(MODE.COLLECTION);
        expect(get(matchContextStore).isMatchMode).toBe(false);
        expect(get(positionsStore).ids).toEqual([2, 3]);
        expect(get(positionStore).id).toBe(2);
        expect(get(currentPositionIndexStore)).toBe(0);
    });

    test('handleOpenCollection refuse une collection vide', () => {
        setLibrary();
        handleOpenCollection({ name: 'vide' }, []);
        expect(get(statusBarModeStore)).toBe(MODE.NORMAL);
    });

    test('exitCollectionMode revient en NORMAL sur la position consultée, vide les stores de collection et l’état de recherche', async () => {
        const lib = [makePosition(1), makePosition(2), makePosition(3)];
        handleOpenCollection({ name: 'Backgames' }, [lib[1], lib[2]]);
        openPanel(PANEL.COLLECTION);
        activeCollectionStore.set({ id: 4 });
        selectedCollectionStore.set({ id: 4 });
        positionService.setSearchState('s', {}, true);
        lastSearchStore.set({ command: 's' });
        ListPositionIDs.mockResolvedValueOnce(lib.map((p) => p.id));

        await exitCollectionMode();

        expect(get(statusBarModeStore)).toBe(MODE.NORMAL);
        expect(get(openPanels).has(PANEL.COLLECTION)).toBe(false);
        expect(get(activeCollectionStore)).toBeNull();
        expect(get(selectedCollectionStore)).toBeNull();
        expect(get(collectionPositionsStore)).toEqual([]);
        expect(get(positionsStore).ids).toEqual(lib.map((p) => p.id));
        expect(get(currentPositionIndexStore), 'retrouvée par id dans la bibliothèque').toBe(1);
        expect(positionService.getSearchState()).toEqual({ lastSearchCommand: '', lastSearchPosition: null, hasActiveSearch: false });
        expect(get(lastSearchStore)).toBeNull();
    });

    test('exitCollectionMode : ListPositionIDs qui échoue retombe sur loadAllPositions au lieu de rester bloqué', async () => {
        handleOpenCollection({ name: 'Backgames' }, [makePosition(2)]);
        ListPositionIDs.mockRejectedValueOnce(new Error('db locked'));
        ListPositionIDs.mockResolvedValueOnce([1, 2, 3]); // l'appel de repli, dans loadAllPositions

        await exitCollectionMode();

        expect(get(statusBarModeStore)).toBe(MODE.NORMAL);
        expect(get(activeCollectionStore)).toBeNull();
        expect(get(collectionPositionsStore)).toEqual([]);
        expect(ListPositionIDs).toHaveBeenCalledTimes(2);
    });

    test('enterEditMode depuis COLLECTION passe par exitCollectionMode', async () => {
        handleOpenCollection({ name: 'Backgames' }, [makePosition(2)]);
        ListPositionIDs.mockResolvedValueOnce([1, 2]);

        await enterEditMode();

        expect(get(statusBarModeStore)).toBe(MODE.EDIT);
        expect(get(activeCollectionStore)).toBeNull();
        expect(get(positionStore).board.bearoff).toEqual([15, 15]);
    });
});
