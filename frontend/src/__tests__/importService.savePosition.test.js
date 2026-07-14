import { describe, test, expect, vi, beforeEach } from 'vitest';

// Saving the position on the board goes through one backend call,
// SaveIndividualPosition, which deduplicates on the Zobrist hash and records
// that the user brought this position in on their own (docs/adr/0002).
//
// The trap these tests lock: the old code asked PositionExists first and, when
// the position was already stored, never called the writer at all. That skipped
// the provenance flag in precisely the case it exists for — import a match, then
// save one of its positions from the board — so the position stayed invisible to
// the very filter meant to find it.

const SaveIndividualPosition = vi.fn();
const SaveAnalysis = vi.fn();
const SaveComment = vi.fn();
const LoadComment = vi.fn();

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
    SaveIndividualPosition,
    SaveAnalysis,
    SaveComment,
    LoadComment,
    AnalyzeImportDatabase: vi.fn(),
    CommitImportDatabase: vi.fn(),
    CancelImport: vi.fn(),
    ImportXGMatch: vi.fn(),
    ImportGnuBGMatch: vi.fn(),
    ImportGnuBGMatchFromText: vi.fn(),
    ImportBGFMatch: vi.fn(),
    ImportBGFPosition: vi.fn(),
    ImportBGFPositionFromText: vi.fn(),
    ImportXGPPosition: vi.fn(),
    ParsePositionText: vi.fn()
}));
vi.mock('../../wailsjs/runtime/runtime.js', () => ({ ClipboardGetText: vi.fn() }));
vi.mock('../services/databaseService.js', () => ({ setStatusBarMessage: vi.fn() }));
vi.mock('../services/positionService.js', () => ({ loadAllPositions: vi.fn() }));

const { savePositionAndAnalysis } = await import('../services/importService.js');

describe('savePositionAndAnalysis', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        LoadComment.mockResolvedValue('');
        SaveAnalysis.mockResolvedValue(undefined);
        SaveComment.mockResolvedValue(undefined);
    });

    test('a brand-new position is written and its id returned', async () => {
        SaveIndividualPosition.mockResolvedValue({ id: 7, existed: false });

        const id = await savePositionAndAnalysis({ board: {} }, {}, 'saved');

        expect(SaveIndividualPosition).toHaveBeenCalledTimes(1);
        expect(id).toBe(7);
        expect(SaveAnalysis).toHaveBeenCalledWith(7, expect.anything());
    });

    test('an already-stored position is still written, so provenance is recorded', async () => {
        SaveIndividualPosition.mockResolvedValue({ id: 42, existed: true });

        const id = await savePositionAndAnalysis({ board: {} }, {}, 'saved');

        // The write happens even though the position was already there — this is
        // the regression that made the individually-imported filter miss it.
        expect(SaveIndividualPosition).toHaveBeenCalledTimes(1);
        expect(id).toBe(42);
    });

    test('a comment on an existing position is merged, not overwritten', async () => {
        SaveIndividualPosition.mockResolvedValue({ id: 42, existed: true });
        LoadComment.mockResolvedValue('first thought');

        await savePositionAndAnalysis({ board: {} }, { comment: 'second thought' }, 'saved');

        expect(SaveComment).toHaveBeenCalledWith(42, 'first thought\n\nsecond thought');
    });

    test('a comment already present is not appended twice', async () => {
        SaveIndividualPosition.mockResolvedValue({ id: 42, existed: true });
        LoadComment.mockResolvedValue('first thought');

        await savePositionAndAnalysis({ board: {} }, { comment: 'first thought' }, 'saved');

        expect(SaveComment).toHaveBeenCalledWith(42, 'first thought');
    });

    test('a failed write returns null and does not touch the analysis', async () => {
        SaveIndividualPosition.mockRejectedValue(new Error('disk on fire'));

        const id = await savePositionAndAnalysis({ board: {} }, {}, 'saved');

        expect(id).toBeNull();
        expect(SaveAnalysis).not.toHaveBeenCalled();
    });
});
