import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

// Ctrl-V has two destinations, and the mode decides which. On a database
// record it IMPORTS: the pasted position is saved and shown. On a scratch
// board — the search board (EDIT) and, since this change, the Eval panel
// (EPC) — it DROPS THE POSITION ONTO THE BOARD instead, which is the paste
// side of the Ctrl-C that copied it. Before this, pasting in the Eval panel
// silently imported a new record into the database and left the Eval board
// untouched, so there was no way to bring an existing position in to edit it.

const SaveIndividualPosition = vi.fn();
const ClipboardGetText = vi.fn();
const ParsePositionText = vi.fn();

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
    SaveAnalysis: vi.fn(),
    SaveComment: vi.fn(),
    LoadComment: vi.fn(),
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
    ParsePositionText
}));
vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetGammonNetAutoAnalyze: vi.fn(() => Promise.resolve(false)),
    GetGammonNetAnalysisPly: vi.fn(() => Promise.resolve(2)),
    GetGammonNetPruneK: vi.fn(() => Promise.resolve(12))
}));
vi.mock('../../wailsjs/runtime/runtime.js', () => ({ ClipboardGetText }));
vi.mock('../services/databaseService.js', () => ({ setStatusBarMessage: vi.fn() }));
vi.mock('../services/positionService.js', () => ({ loadAllPositions: vi.fn() }));

const { pastePosition } = await import('../services/importService.js');
const { positionStore, clipboardPositionStore } = await import('../stores/positionStore.js');
const { databasePathStore } = await import('../stores/databaseStore.js');
const { statusBarModeStore } = await import('../stores/uiStore.js');

/** A board of all-empty points, the shape every position carries. */
function emptyBoard() {
    return {
        points: Array(26)
            .fill(null)
            .map(() => ({ checkers: 0, color: -1 })),
        bearoff: [0, 0]
    };
}

function makePosition(overrides = {}) {
    return {
        id: 0,
        board: emptyBoard(),
        cube: { owner: -1, value: 0 },
        dice: [0, 0],
        score: [-1, -1],
        player_on_roll: 0,
        decision_type: 0,
        has_jacoby: 0,
        has_beaver: 0,
        ...overrides
    };
}

beforeEach(() => {
    vi.clearAllMocks();
    databasePathStore.set('/tmp/test.db');
    clipboardPositionStore.set(null);
    positionStore.set(makePosition());
    ClipboardGetText.mockResolvedValue('');
    SaveIndividualPosition.mockResolvedValue(0);
    ParsePositionText.mockResolvedValue({ position: makePosition({ dice: [6, 5] }), analysis: {}, comment: '' });
});

describe('pastePosition in the Eval panel (EPC mode)', () => {
    test('drops the copied position onto the board instead of importing it', async () => {
        const copied = makePosition({ dice: [6, 5], score: [2, 3], player_on_roll: 1 });
        copied.board.bearoff = [4, 7];
        copied.board.points[6] = { checkers: 5, color: 0 };
        clipboardPositionStore.set(copied);
        statusBarModeStore.set('EPC');

        await pastePosition();

        const board = get(positionStore);
        expect(board.dice).toEqual([6, 5]);
        expect(board.score).toEqual([2, 3]);
        expect(board.player_on_roll).toBe(1);
        expect(board.board.bearoff).toEqual([4, 7]);
        expect(board.board.points[6]).toEqual({ checkers: 5, color: 0 });
        expect(SaveIndividualPosition, 'a scratch-board paste writes nothing to the database').not.toHaveBeenCalled();
    });

    test('leaves the mode alone — a paste is not a way out of the Eval panel', async () => {
        clipboardPositionStore.set(makePosition({ dice: [3, 1] }));
        statusBarModeStore.set('EPC');

        await pastePosition();

        expect(get(statusBarModeStore)).toBe('EPC');
    });

    test('still imports into the database in NORMAL mode', async () => {
        clipboardPositionStore.set(makePosition({ dice: [6, 5] }));
        statusBarModeStore.set('NORMAL');

        await pastePosition();

        expect(get(positionStore).dice, 'the board is untouched by an import').toEqual([0, 0]);
    });
});
