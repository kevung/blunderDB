import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

// A watched folder imports without moving the user: mode, active search, tab,
// current position and board stay where they were.

const ImportXGMatch = vi.fn();
const ImportXGPPosition = vi.fn();
const loadAllPositions = vi.fn();

vi.mock('../../wailsjs/go/gui/App.js', () => ({
    OpenImportDatabaseDialog: vi.fn(),
    OpenPositionFilesDialog: vi.fn(),
    OpenPositionFolderDialog: vi.fn(),
    CollectImportableFiles: vi.fn(),
    ReadFileContent: vi.fn(),
    ShowAlert: vi.fn(),
    ShowQuestionDialog: vi.fn(),
    IsDirectory: vi.fn(),
    StartGammonNetBatch: vi.fn()
}));
vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetGammonNetAutoAnalyze: vi.fn(async () => false),
    GetGammonNetAnalysisPly: vi.fn(),
    GetGammonNetPruneK: vi.fn()
}));
vi.mock('../../wailsjs/go/database/Database.js', () => ({
    SaveIndividualPosition: vi.fn(),
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
    ParsePositionText: vi.fn(),
    BeginImportBatch: vi.fn(async () => 1),
    FinishImportBatch: vi.fn(),
    ImportReport: vi.fn(async () => null),
    RefreshSearchStatistics: vi.fn(),
    CountPositionsWithoutAnalysis: vi.fn(async () => 0)
}));
vi.mock('../../wailsjs/runtime/runtime.js', () => ({ ClipboardGetText: vi.fn() }));
vi.mock('../services/databaseService.js', () => ({ setStatusBarMessage: vi.fn() }));
vi.mock('../services/positionService.js', () => ({ loadAllPositions }));

const { importWatchedFiles } = await import('../services/importService.js');
const { activeTabStore, currentPositionIndexStore, statusBarModeStore } = await import('../stores/uiStore.js');
const { positionStore, matchContextStore } = await import('../stores/positionStore.js');

beforeEach(() => {
    vi.clearAllMocks();
    activeTabStore.set('analysis');
    currentPositionIndexStore.set(5);
    statusBarModeStore.set('EDIT');
    matchContextStore.set({ isMatchMode: true, matchID: 9, movePositions: [1, 2], currentIndex: 1, player1Name: 'a', player2Name: 'b' });
    positionStore.set({ id: 77, board: {} });
});

describe('importWatchedFiles', () => {
    test('a match and a position leave mode, tab, index, match context and board alone', async () => {
        ImportXGMatch.mockResolvedValue(3);
        ImportXGPPosition.mockResolvedValue(42);

        const r = await importWatchedFiles(['/w/a.xg', '/w/b.xgp']);

        expect(r).toEqual({ succeeded: 2, skipped: 0, failed: 0 });
        expect(loadAllPositions).not.toHaveBeenCalled();
        expect(get(activeTabStore)).toBe('analysis');
        expect(get(currentPositionIndexStore)).toBe(5);
        expect(get(statusBarModeStore)).toBe('EDIT');
        expect(get(matchContextStore).matchID).toBe(9);
        expect(get(positionStore).id).toBe(77);
    });
});
