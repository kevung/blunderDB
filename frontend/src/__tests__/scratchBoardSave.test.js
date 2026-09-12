/**
 * scratchBoardSave.test.js — saving the Search panel's scratch board (#400).
 *
 * A scratch board has no identity: nothing known about the position it was
 * copied from travels with it (CONTEXT.md, *Scratch board*). Yet while the
 * Search panel is open, analysisStore still describes the position studied
 * BEFORE entering it (modeMachine.js header), and the save used to start from
 * that store: the played moves, cube analyses and player names of another
 * position went to SaveAnalysis, which merges them into the new one.
 *
 * The same path also left the panel: a new position reloaded the library
 * (mode NORMAL, Matches tab), a known one set the index against whatever list
 * was on screen. The board stays a scratch board after it is saved.
 */

import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

const db = vi.hoisted(() => ({
    SaveIndividualPosition: vi.fn(),
    SaveAnalysis: vi.fn(() => Promise.resolve()),
    SaveComment: vi.fn(() => Promise.resolve()),
    LoadComment: vi.fn(() => Promise.resolve('')),
    LoadAnalysis: vi.fn(() => Promise.resolve(null)),
    ListPositionIDs: vi.fn(() => Promise.resolve([])),
    CountPositionsWithoutAnalysis: vi.fn(() => Promise.resolve(0)),
    SaveLastVisitedPosition: vi.fn(() => Promise.resolve()),
    GetLastVisitedMatch: vi.fn(() => Promise.resolve(null)),
    GetMatchMovePositions: vi.fn(() => Promise.resolve([])),
    LoadPositionsByIDs: vi.fn(() => Promise.resolve([]))
}));
const config = vi.hoisted(() => ({
    GetGammonNetAutoAnalyze: vi.fn(() => Promise.resolve(false)),
    GetGammonNetAnalysisPly: vi.fn(() => Promise.resolve(2)),
    GetGammonNetPruneK: vi.fn(() => Promise.resolve(12))
}));
const app = vi.hoisted(() => ({ StartGammonNetBatch: vi.fn(() => Promise.resolve()) }));
const status = vi.hoisted(() => ({ setStatusBarMessage: vi.fn() }));

vi.mock('../../wailsjs/go/database/Database.js', () => db);
vi.mock('../../wailsjs/go/main/Config.js', () => config);
vi.mock('../../wailsjs/go/gui/App.js', () => app);
vi.mock('../../wailsjs/runtime/runtime.js', () => ({ ClipboardGetText: vi.fn() }));
vi.mock('../services/databaseService.js', () => ({
    setStatusBarMessage: status.setStatusBarMessage,
    warningMessageStore: { subscribe: vi.fn(), set: vi.fn(), update: vi.fn() }
}));
vi.mock('../services/sessionService.js', () => ({ saveSessionState: vi.fn() }));

import { statusBarModeStore, currentPositionIndexStore, activeTabStore } from '../stores/uiStore.js';
import { positionStore, positionsStore, matchContextStore } from '../stores/positionStore.js';
import { analysisStore, emptyAnalysis } from '../stores/analysisStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { lastSearchStore } from '../stores/searchHistoryStore.js';
import { activeCollectionStore } from '../stores/collectionStore.js';
import { saveCurrentPosition, enterEditMode, exitEditMode, enterEPCMode, exitEPCMode, setSearchState } from '../services/positionService.js';
import { joinLibraryBehindScratchBoard } from '../services/modeMachine.js';

function emptyPoints() {
    return Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 }));
}

function libraryPosition(id) {
    const points = emptyPoints();
    points[6] = { checkers: 5, color: 0 };
    points[19] = { checkers: 5, color: 1 };
    return {
        id,
        board: { points, bearoff: [10, 10] },
        cube: { owner: -1, value: 0 },
        dice: [3, 1],
        score: [5, 5],
        player_on_roll: 0,
        decision_type: 0,
        has_jacoby: 0,
        has_beaver: 0
    };
}

/** What the user draws on the Search panel: a valid board, nothing like the studied one. */
function drawBoard(position) {
    const points = emptyPoints();
    points[1] = { checkers: 2, color: 0 };
    points[24] = { checkers: 2, color: 1 };
    position.board.points = points;
    position.board.bearoff = [13, 13];
    position.dice = [6, 5];
    return position;
}

/** The analysis of the position studied before: an XG match position. */
function studiedAnalysis(positionId) {
    return {
        ...emptyAnalysis(),
        positionId,
        xgid: 'XGID=-a----E-C---eE---c-e----B-:0:0:1:52:0:0:0:7:10',
        player1: 'Alice',
        player2: 'Bob',
        analysisType: 'CheckerMove',
        analysisEngineVersion: 'XG 2.19',
        checkerAnalysis: { moves: [{ index: 0, move: '13/8 13/11', equity: 0.1 }] },
        allCubeAnalyses: [{ analysisEngine: 'XG', bestCubeAction: 'No double' }],
        playedMoves: ['13/8 13/11'],
        playedCubeActions: ['No double']
    };
}

/** The last status-bar message, as the tMsg() descriptor it was given. */
function lastStatus() {
    const calls = status.setStatusBarMessage.mock.calls;
    return calls.length ? calls[calls.length - 1][0] : null;
}

/** A library of four positions, the third one studied, then the Search tab. */
async function openSearchFromLibrary() {
    const ids = [11, 12, 13, 14];
    const byId = new Map(ids.map((id) => [id, libraryPosition(id)]));
    positionsStore.setLoader(async (wanted) => wanted.map((id) => byId.get(id)).filter(Boolean));
    positionsStore.setIds(ids, { reset: true });
    await positionsStore.getPosition(2);
    currentPositionIndexStore.set(2);
    positionStore.set(libraryPosition(13));
    analysisStore.set(studiedAnalysis(13));
    activeTabStore.set('search');
    await enterEditMode();
    positionStore.update(drawBoard);
}

beforeEach(() => {
    vi.clearAllMocks();
    databasePathStore.set('/tmp/test.db');
    statusBarModeStore.set('NORMAL');
    matchContextStore.set({ isMatchMode: false, matchID: null, movePositions: [], currentIndex: 0, player1Name: '', player2Name: '' });
    activeCollectionStore.set(null);
    lastSearchStore.set(null);
    setSearchState('', null, false);
    analysisStore.set(emptyAnalysis());
    db.SaveIndividualPosition.mockResolvedValue({ id: 99, existed: false });
    db.ListPositionIDs.mockResolvedValue([11, 12, 13, 14, 99]);
});

describe('saving the Search scratch board', () => {
    test('nothing of the position studied before reaches SaveAnalysis, and the store is left alone', async () => {
        await openSearchFromLibrary();
        const before = JSON.parse(JSON.stringify(get(analysisStore)));

        await saveCurrentPosition();

        expect(db.SaveAnalysis).toHaveBeenCalledTimes(1);
        const [id, sent] = db.SaveAnalysis.mock.calls[0];
        expect(id).toBe(99);
        expect(sent.playedMoves ?? []).toEqual([]);
        expect(sent.playedCubeActions ?? []).toEqual([]);
        expect(sent.allCubeAnalyses ?? []).toEqual([]);
        expect(sent.checkerAnalysis?.moves ?? []).toEqual([]);
        expect(sent.player1 ?? '').toBe('');
        expect(sent.player2 ?? '').toBe('');
        expect(sent.analysisType ?? '').toBe('');
        // The xgid is the drawn board's, not the studied one's.
        expect(sent.xgid).toBeTruthy();
        expect(sent.xgid).not.toBe(before.xgid);
        expect(sent).not.toBe(get(analysisStore));
        expect(get(analysisStore)).toEqual(before);
    });

    test('a new position: the panel stays open on the same board, and the number is announced', async () => {
        await openSearchFromLibrary();
        const board = JSON.parse(JSON.stringify(get(positionStore)));

        await saveCurrentPosition();

        expect(get(statusBarModeStore)).toBe('EDIT');
        expect(get(activeTabStore)).toBe('search');
        expect(get(positionStore)).toEqual(board);
        expect(get(currentPositionIndexStore)).toBe(2);
        expect(lastStatus()).toEqual({ i18nKey: 'status.scratchBoardSaved', i18nParams: { id: 99 } });
    });

    test('a known position: the panel stays open on the same board, and its number is announced', async () => {
        db.SaveIndividualPosition.mockResolvedValue({ id: 1234, existed: true });
        await openSearchFromLibrary();
        const board = JSON.parse(JSON.stringify(get(positionStore)));

        await saveCurrentPosition();

        expect(get(statusBarModeStore)).toBe('EDIT');
        expect(get(activeTabStore)).toBe('search');
        expect(get(positionStore)).toEqual(board);
        expect(get(currentPositionIndexStore)).toBe(2);
        expect(get(positionsStore).ids).toEqual([11, 12, 13, 14]);
        expect(lastStatus()).toEqual({ i18nKey: 'status.scratchBoardAlreadyStored', i18nParams: { id: 1234 } });
    });

    test('entered from the library without a search, the new position joins the end of the list', async () => {
        await openSearchFromLibrary();

        await saveCurrentPosition();
        expect(get(positionsStore).ids).toEqual([11, 12, 13, 14, 99]);

        await exitEditMode();
        expect(get(statusBarModeStore)).toBe('NORMAL');
        expect(get(currentPositionIndexStore)).toBe(2);
        expect(get(positionStore).id).toBe(13);
    });

    test('entered over an active search, leaving the panel finds the same results and position', async () => {
        await openSearchFromLibrary();
        // The list on screen is a search result, not the library.
        setSearchState('s cube', null, true);
        lastSearchStore.set({ command: 's cube', position: '{}' });

        await saveCurrentPosition();
        expect(get(positionsStore).ids).toEqual([11, 12, 13, 14]);

        await exitEditMode();
        expect(get(positionsStore).ids).toEqual([11, 12, 13, 14]);
        expect(get(currentPositionIndexStore)).toBe(2);
        expect(get(positionStore).id).toBe(13);
    });

    test('entered from a match, leaving the panel returns to the same match position', async () => {
        const movePositions = [{ position: libraryPosition(21) }, { position: libraryPosition(22) }];
        positionsStore.setIds([21, 22], { reset: true });
        currentPositionIndexStore.set(1);
        positionStore.set(libraryPosition(22));
        analysisStore.set(studiedAnalysis(22));
        matchContextStore.set({ isMatchMode: true, matchID: 5, movePositions, currentIndex: 1, player1Name: 'Alice', player2Name: 'Bob' });
        statusBarModeStore.set('MATCH');
        activeTabStore.set('search');
        await enterEditMode();
        positionStore.update(drawBoard);

        await saveCurrentPosition();
        expect(get(statusBarModeStore)).toBe('EDIT');
        expect(get(positionsStore).ids).toEqual([21, 22]);

        await exitEditMode();
        expect(get(statusBarModeStore)).toBe('MATCH');
        expect(get(matchContextStore).matchID).toBe(5);
        expect(get(matchContextStore).currentIndex).toBe(1);
        expect(get(positionsStore).ids).toEqual([21, 22]);
    });

    test('entered over another list shown in NORMAL mode (a deck, a statistics selection), it is left alone', async () => {
        await openSearchFromLibrary();
        // Same mode, no search flag — but the list is two positions, not the library.
        positionsStore.setIds([12, 13]);
        currentPositionIndexStore.set(1);

        await saveCurrentPosition();

        expect(get(positionsStore).ids).toEqual([12, 13]);
        expect(get(currentPositionIndexStore)).toBe(1);
    });

    test('the board keeps no id: it does not become a library record', async () => {
        await openSearchFromLibrary();
        const idBefore = get(positionStore).id;
        let sentId;
        db.SaveIndividualPosition.mockImplementation(async (position) => {
            sentId = position.id;
            return { id: 99, existed: false };
        });

        await saveCurrentPosition();

        expect(get(positionStore).id).toBe(idBefore);
        expect(sentId).toBe(0);
    });

    test('a known position gets its provenance flag and nothing else: its stored analysis is not overwritten', async () => {
        db.SaveIndividualPosition.mockResolvedValue({ id: 1234, existed: true });
        await openSearchFromLibrary();

        await saveCurrentPosition();

        expect(db.SaveIndividualPosition).toHaveBeenCalledTimes(1);
        expect(db.SaveAnalysis).not.toHaveBeenCalled();
        expect(db.SaveComment).not.toHaveBeenCalled();
    });

    test('the gammonNet auto-analysis runs after the save when it is enabled', async () => {
        config.GetGammonNetAutoAnalyze.mockResolvedValue(true);
        db.CountPositionsWithoutAnalysis.mockResolvedValue(1);
        await openSearchFromLibrary();

        await saveCurrentPosition();

        expect(app.StartGammonNetBatch).toHaveBeenCalledWith(2, 12, 0);
    });

    test('a refused board is not written, and the refusal is said', async () => {
        await openSearchFromLibrary();
        positionStore.update((p) => {
            p.board.points = emptyPoints();
            p.board.points[24] = { checkers: 2, color: 1 };
            return p;
        });

        await saveCurrentPosition();

        expect(db.SaveIndividualPosition).not.toHaveBeenCalled();
        expect(status.setStatusBarMessage).toHaveBeenCalledTimes(1);
    });
});

describe('the list behind the Eval board (#399 will save from there)', () => {
    test('entered from the library, the saved id joins the list the exit puts back', async () => {
        const ids = [11, 12, 13, 14];
        const byId = new Map(ids.map((id) => [id, libraryPosition(id)]));
        positionsStore.setLoader(async (wanted) => wanted.map((id) => byId.get(id)).filter(Boolean));
        positionsStore.setIds(ids, { reset: true });
        await positionsStore.getPosition(2);
        currentPositionIndexStore.set(2);
        positionStore.set(libraryPosition(13));
        await enterEPCMode();

        expect(await joinLibraryBehindScratchBoard(99)).toBe(true);
        await exitEPCMode();

        expect(get(positionsStore).ids).toEqual([11, 12, 13, 14, 99]);
        expect(get(currentPositionIndexStore)).toBe(2);
    });

    test('entered from a match, nothing joins', async () => {
        positionsStore.setIds([21, 22], { reset: true });
        currentPositionIndexStore.set(1);
        positionStore.set(libraryPosition(22));
        matchContextStore.set({
            isMatchMode: true,
            matchID: 5,
            movePositions: [{ position: libraryPosition(21) }, { position: libraryPosition(22) }],
            currentIndex: 1,
            player1Name: 'A',
            player2Name: 'B'
        });
        statusBarModeStore.set('MATCH');
        await enterEPCMode();

        expect(await joinLibraryBehindScratchBoard(99)).toBe(false);
    });
});
