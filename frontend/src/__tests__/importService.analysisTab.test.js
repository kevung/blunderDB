import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

// Importing a *position* must land on the board with the analysis tab open;
// importing a *match* keeps opening the match list. The trap is that
// loadAllPositions() unconditionally selects the matches tab, so every position
// import path has to re-select 'analysis' after its reload — that is what these
// tests lock.

const ImportXGPPosition = vi.fn();
const ImportXGMatch = vi.fn();

vi.mock('../../wailsjs/go/gui/App.js', () => ({
    OpenImportDatabaseDialog: vi.fn(),
    OpenPositionFilesDialog: vi.fn(),
    OpenPositionFolderDialog: vi.fn(),
    CollectImportableFiles: vi.fn(),
    ReadFileContent: vi.fn(),
    ShowAlert: vi.fn(),
    ShowQuestionDialog: vi.fn(),
    IsDirectory: vi.fn()
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
    ParsePositionText: vi.fn()
}));
vi.mock('../../wailsjs/runtime/runtime.js', () => ({ ClipboardGetText: vi.fn() }));
vi.mock('../services/databaseService.js', () => ({ setStatusBarMessage: vi.fn() }));

// Stand-in for the real loadAllPositions: refills the position list and forces
// the matches tab, exactly like positionService.js does.
vi.mock('../services/positionService.js', () => ({
    loadAllPositions: vi.fn(async () => {
        const { positionsStore } = await import('../stores/positionStore.js');
        const { activeTabStore, currentPositionIndexStore } = await import('../stores/uiStore.js');
        positionsStore.set([{ id: 7 }, { id: 42 }]);
        currentPositionIndexStore.set(1);
        activeTabStore.set('matches');
    })
}));

const { importSingleFile } = await import('../services/importService.js');
const { activeTabStore, currentPositionIndexStore, openPanels, PANEL } = await import('../stores/uiStore.js');
const { positionsStore } = await import('../stores/positionStore.js');

beforeEach(() => {
    vi.clearAllMocks();
    positionsStore.set([]);
    activeTabStore.set('matches');
    currentPositionIndexStore.set(0);
    openPanels.set(new Set());
});

describe('importSingleFile — panel shown after the import', () => {
    test('position (.xgp) → analysis tab, board on the imported position', async () => {
        ImportXGPPosition.mockResolvedValue(42);

        const outcome = await importSingleFile('/tmp/blunder.xgp');

        expect(outcome).toEqual({ type: 'position', id: 42 });
        expect(get(activeTabStore)).toBe('analysis');
        expect(get(positionsStore)[get(currentPositionIndexStore)].id).toBe(42);
        expect(get(openPanels).has(PANEL.MATCH)).toBe(false);
    });

    test('match (.xg) → match tab, unchanged', async () => {
        ImportXGMatch.mockResolvedValue(3);

        const outcome = await importSingleFile('/tmp/game.xg');

        expect(outcome).toEqual({ type: 'match', id: 3 });
        expect(get(activeTabStore)).toBe('matches');
        expect(get(openPanels).has(PANEL.MATCH)).toBe(true);
    });

    test('failed position import leaves the view alone', async () => {
        ImportXGPPosition.mockRejectedValue(new Error('boom'));

        const outcome = await importSingleFile('/tmp/broken.xgp');

        expect(outcome).toBeNull();
        expect(get(activeTabStore)).toBe('matches');
    });
});
