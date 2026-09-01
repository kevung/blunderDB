import { describe, test, expect, vi, beforeEach } from 'vitest';

// Ctrl-C used to go through navigator.clipboard.writeText(), which requires
// transient user activation — and a keydown carrying a modifier is not an
// activation-triggering event, so the shortcut rejected with NotAllowedError
// while the toolbar button (a click, hence activated) worked. copyPosition now
// writes through the Go backend, the same clipboard the paste side reads.

const ClipboardSetText = vi.fn();

vi.mock('../../wailsjs/go/gui/App.js', () => ({ CopyImageToClipboard: vi.fn() }));
vi.mock('../../wailsjs/runtime/runtime.js', () => ({ ClipboardSetText }));
vi.mock('../services/databaseService.js', () => ({ setStatusBarMessage: vi.fn() }));
vi.mock('../services/positionService.js', () => ({ generateXGID: () => 'XGID-STUB' }));

const { copyPosition } = await import('../services/clipboardService.js');
const { databasePathStore } = await import('../stores/databaseStore.js');
const { positionStore } = await import('../stores/positionStore.js');
const { analysisStore } = await import('../stores/analysisStore.js');
const { statusBarModeStore } = await import('../stores/uiStore.js');

describe('copyPosition', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        databasePathStore.set('/tmp/test.db');
        statusBarModeStore.set('NORMAL');
        positionStore.update((p) => ({
            ...p,
            board: { points: [], bearoff: [0, 0] },
            cube: { owner: -1, value: 0 },
            dice: [3, 1],
            score: [-1, -1],
            player_on_roll: 0,
            decision_type: 0
        }));
    });

    test('writes through the backend clipboard, not the WebView one', async () => {
        ClipboardSetText.mockResolvedValue(true);
        const writeText = vi.fn().mockResolvedValue(undefined);
        vi.stubGlobal('navigator', { clipboard: { writeText } });

        copyPosition();
        await vi.waitFor(() => expect(ClipboardSetText).toHaveBeenCalledTimes(1));

        expect(writeText).not.toHaveBeenCalled();
        expect(ClipboardSetText.mock.calls[0][0]).toContain('XGID=');
    });

    test('falls back to the WebView clipboard when the backend one fails', async () => {
        ClipboardSetText.mockRejectedValue(new Error('no runtime'));
        const writeText = vi.fn().mockResolvedValue(undefined);
        vi.stubGlobal('navigator', { clipboard: { writeText } });

        copyPosition();
        await vi.waitFor(() => expect(writeText).toHaveBeenCalledTimes(1));

        expect(writeText.mock.calls[0][0]).toContain('XGID=');
    });

    test('the checker block keeps the best move parseable even without an equity error', async () => {
        ClipboardSetText.mockResolvedValue(true);
        vi.stubGlobal('navigator', { clipboard: { writeText: vi.fn() } });
        analysisStore.update((a) => ({
            ...a,
            analysisType: 'CheckerMove',
            checkerAnalysis: {
                moves: [
                    { index: 0, move: '24/23 13/10', analysisDepth: '4-ply', equity: 0.123, equityError: undefined },
                    { index: 1, move: '24/21 13/12', analysisDepth: '4-ply', equity: 0.1, equityError: 0.023 }
                ]
            }
        }));

        copyPosition();
        await vi.waitFor(() => expect(ClipboardSetText).toHaveBeenCalledTimes(1));

        const text = ClipboardSetText.mock.calls[0][0];
        // The best move deliberately carries no "Equity Error:" line — the Go
        // parser must stay tolerant of that (parser_internal_checker_test.go).
        expect(text).toContain('Move 0: 24/23 13/10');
        expect(text).not.toContain('Equity Error: undefined');
        expect(text).toContain('Equity Error: 0.023');
    });
});

// Entering the Eval panel (EPC) or the search board (EDIT) replaces the board
// but leaves analysisStore holding the record last opened. Copying there used
// to emit THAT record's xgid and analysis — so a position built in Eval was
// pasted into XG, or into another blunderDB, as an entirely different one.
describe('copyPosition on a scratch board', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        databasePathStore.set('/tmp/test.db');
        ClipboardSetText.mockResolvedValue(true);
        vi.stubGlobal('navigator', { clipboard: { writeText: vi.fn() } });
        positionStore.update((p) => ({
            ...p,
            board: { points: [], bearoff: [0, 0] },
            cube: { owner: -1, value: 0 },
            dice: [6, 5],
            score: [-1, -1],
            player_on_roll: 0,
            decision_type: 0
        }));
        // A stored record left behind by the position visited before Eval.
        analysisStore.update((a) => ({
            ...a,
            xgid: 'XGID-FROM-A-DIFFERENT-POSITION',
            analysisType: 'CheckerMove',
            checkerAnalysis: { moves: [{ index: 0, move: '24/23 13/10', analysisDepth: '4-ply', equity: 0.123 }] }
        }));
    });

    test.each([['EPC'], ['EDIT']])('%s copies the board it shows, not the record left in the store', async (mode) => {
        statusBarModeStore.set(mode);

        copyPosition();
        await vi.waitFor(() => expect(ClipboardSetText).toHaveBeenCalledTimes(1));

        const text = ClipboardSetText.mock.calls[0][0];
        expect(text).toContain('XGID=XGID-STUB');
        expect(text).not.toContain('XGID-FROM-A-DIFFERENT-POSITION');
        expect(text, 'no analysis belonging to another position').not.toContain('24/23 13/10');
        expect(text, 'the board itself still travels').toContain('Dice: 6, 5');
    });

    test('NORMAL mode still carries the stored record', async () => {
        statusBarModeStore.set('NORMAL');

        copyPosition();
        await vi.waitFor(() => expect(ClipboardSetText).toHaveBeenCalledTimes(1));

        const text = ClipboardSetText.mock.calls[0][0];
        expect(text).toContain('XGID=XGID-FROM-A-DIFFERENT-POSITION');
        expect(text).toContain('24/23 13/10');
    });
});
