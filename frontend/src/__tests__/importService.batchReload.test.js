import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

// A batch import (folder, multi-select, drag-drop of several files) must
// reload the position table ONCE, after its last file. Each reload pulls every
// row through the Wails IPC, and the old code paid it per position: the .txt
// path went through savePositionAndAnalysis, which reloaded on every new row,
// and the batch then reloaded again at the end — N+1 full reloads for N files.
// These tests count the reloads for a batch and pin them to one.

const SaveIndividualPosition = vi.fn();
const ImportXGPPosition = vi.fn();
const ImportXGMatch = vi.fn();
const ReadFileContent = vi.fn();
const ParsePositionText = vi.fn();
const CollectImportableFiles = vi.fn();
const OpenPositionFolderDialog = vi.fn();
const loadAllPositions = vi.fn();

vi.mock('../../wailsjs/go/gui/App.js', () => ({
    OpenImportDatabaseDialog: vi.fn(),
    OpenPositionFilesDialog: vi.fn(),
    OpenPositionFolderDialog,
    CollectImportableFiles,
    ReadFileContent,
    ShowAlert: vi.fn(),
    ShowQuestionDialog: vi.fn(),
    IsDirectory: vi.fn(),
    StartGammonNetBatch: vi.fn()
}));
vi.mock('../../wailsjs/go/database/Database.js', () => ({
    SaveIndividualPosition,
    SaveAnalysis: vi.fn(),
    LoadComment: vi.fn(),
    SaveComment: vi.fn(),
    AnalyzeImportDatabase: vi.fn(),
    CommitImportDatabase: vi.fn(),
    CancelImport: vi.fn(),
    ImportXGMatch,
    ImportGnuBGMatch: vi.fn(),
    ImportGnuBGMatchFromText: vi.fn(),
    ImportBGFMatch: vi.fn(),
    ImportBGFPosition: vi.fn(),
    ImportBGFPositionFromText: vi.fn(),
    ImportXGPPosition,
    ParsePositionText,
    RefreshSearchStatistics: vi.fn(),
    CountPositionsWithoutAnalysis: vi.fn()
}));
vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetGammonNetAutoAnalyze: vi.fn(async () => false),
    GetGammonNetAnalysisPly: vi.fn(),
    GetGammonNetPruneK: vi.fn()
}));
vi.mock('../../wailsjs/runtime/runtime.js', () => ({ ClipboardGetText: vi.fn() }));
vi.mock('../services/databaseService.js', () => ({ setStatusBarMessage: vi.fn() }));
vi.mock('../services/positionService.js', () => ({ loadAllPositions }));

const { importMultipleFiles, importFolder } = await import('../services/importService.js');
const { positionsStore, matchContextStore } = await import('../stores/positionStore.js');
const { databasePathStore } = await import('../stores/databaseStore.js');
const { activeTabStore, currentPositionIndexStore, statusBarModeStore } = await import('../stores/uiStore.js');
const { fileImportResultsStore, fileImportModeStore } = await import('../stores/importModalStore.js');

// The rows the fake backend holds; the fake loadAllPositions hands them to the
// store exactly like positionService.js does (and lands on the matches tab).
let rows = [];
let nextID = 1;

function txtFile(name) {
    return `/import/${name}.txt`;
}

beforeEach(() => {
    vi.clearAllMocks();
    rows = [];
    nextID = 1;
    databasePathStore.set('/tmp/test.db');
    positionsStore.set([]);
    activeTabStore.set('matches');
    currentPositionIndexStore.set(-1);
    statusBarModeStore.set('NORMAL');

    ReadFileContent.mockImplementation(async (path) => ({ content: `XGID=${path}` }));
    ParsePositionText.mockImplementation(async (content) => ({
        position: { board: { points: [], bearoff: [0, 0] }, xgid: content },
        analysis: {},
        comment: ''
    }));
    SaveIndividualPosition.mockImplementation(async (positionData) => {
        const id = nextID++;
        rows.push({ id, xgid: positionData.xgid });
        return { id, existed: false };
    });
    ImportXGPPosition.mockImplementation(async () => {
        const id = nextID++;
        rows.push({ id });
        return id;
    });
    loadAllPositions.mockImplementation(async () => {
        positionsStore.set([...rows]);
        currentPositionIndexStore.set(rows.length - 1);
        activeTabStore.set('matches');
        statusBarModeStore.set('NORMAL');
        matchContextStore.set({ isMatchMode: false, matchID: null, movePositions: [], currentIndex: 0, player1Name: '', player2Name: '' });
    });
});

describe('importMultipleFiles — one reload per batch', () => {
    test('N position files reload the table exactly once', async () => {
        const files = ['a', 'b', 'c', 'd', 'e'].map(txtFile);

        const outcome = await importMultipleFiles(files);

        expect(SaveIndividualPosition).toHaveBeenCalledTimes(5);
        expect(loadAllPositions).toHaveBeenCalledTimes(1);
        // Visible outcome unchanged: last position shown, analysis tab, counters.
        expect(outcome).toEqual({ type: 'position', id: 5 });
        expect(get(positionsStore)[get(currentPositionIndexStore)].id).toBe(5);
        expect(get(activeTabStore)).toBe('analysis');
        expect(get(fileImportResultsStore)).toEqual({ succeeded: 5, failed: 0, skipped: 0, errors: [] });
        expect(get(fileImportModeStore)).toBe('completed');
    });

    test('a batch mixing .txt and .xgp positions still reloads once', async () => {
        const files = [txtFile('a'), '/import/b.xgp', txtFile('c')];

        const outcome = await importMultipleFiles(files);

        expect(loadAllPositions).toHaveBeenCalledTimes(1);
        expect(outcome).toEqual({ type: 'position', id: 3 });
    });

    test('a partially failed batch reloads once and reports the failure', async () => {
        ImportXGPPosition.mockRejectedValueOnce(new Error('corrupt file'));
        const files = [txtFile('a'), '/import/broken.xgp', txtFile('c')];

        const outcome = await importMultipleFiles(files);

        expect(loadAllPositions).toHaveBeenCalledTimes(1);
        const results = get(fileImportResultsStore);
        expect(results.succeeded).toBe(2);
        expect(results.failed).toBe(1);
        expect(results.errors).toEqual([{ file: '/import/broken.xgp', message: 'corrupt file' }]);
        expect(outcome).toEqual({ type: 'position', id: 2 });
    });

    test('a batch with a match reloads once and leaves the match list open', async () => {
        ImportXGMatch.mockResolvedValue(9);
        const files = [txtFile('a'), '/import/game.xg'];

        const outcome = await importMultipleFiles(files);

        expect(loadAllPositions).toHaveBeenCalledTimes(1);
        expect(outcome).toBeNull();
        expect(get(activeTabStore)).toBe('matches');
    });
});

describe('importFolder — the reload is not repeated on leaving match mode', () => {
    test('a folder import started from match mode reloads once', async () => {
        OpenPositionFolderDialog.mockResolvedValue('/import');
        CollectImportableFiles.mockResolvedValue(['a', 'b', 'c'].map(txtFile));
        statusBarModeStore.set('MATCH');
        matchContextStore.set({ isMatchMode: true, matchID: 4, movePositions: [], currentIndex: 2, player1Name: 'A', player2Name: 'B' });

        await importFolder();

        expect(loadAllPositions).toHaveBeenCalledTimes(1);
        expect(get(matchContextStore).isMatchMode).toBe(false);
        expect(get(statusBarModeStore)).toBe('NORMAL');
        expect(get(activeTabStore)).toBe('analysis');
    });

    test('a cancelled folder dialog from match mode still leaves match mode', async () => {
        OpenPositionFolderDialog.mockResolvedValue('');
        statusBarModeStore.set('MATCH');
        matchContextStore.set({ isMatchMode: true, matchID: 4, movePositions: [], currentIndex: 2, player1Name: 'A', player2Name: 'B' });

        await importFolder();

        expect(loadAllPositions).toHaveBeenCalledTimes(1);
        expect(get(matchContextStore).isMatchMode).toBe(false);
        expect(get(statusBarModeStore)).toBe('NORMAL');
    });
});
