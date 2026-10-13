import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

const ReadCaptureFile = vi.fn();
const ReadCaptureData = vi.fn();
const SaveIndividualPosition = vi.fn();
const enterEditMode = vi.fn(async () => {});
const applyPositionToBoard = vi.fn();
const setStatusBarMessage = vi.fn();

vi.mock('../../wailsjs/go/database/Database.js', () => ({ ReadCaptureFile, ReadCaptureData, SaveIndividualPosition }));
vi.mock('../services/modeMachine.js', () => ({ enterEditMode }));
vi.mock('../services/importService.js', () => ({ applyPositionToBoard }));
vi.mock('../stores/duelStore.js', async () => {
    const { writable } = await import('svelte/store');
    return { duelHoldsBoardStore: writable(false) };
});
vi.mock('../services/databaseService.js', () => ({ setStatusBarMessage }));

const { showCaptureOnBoard, importCaptureFile, importCaptureBlob, imageOfPaste, isCaptureFile } = await import('../services/captureService.js');
const { statusBarModeStore } = await import('../stores/uiStore.js');
const { captureDoubtsStore } = await import('../stores/captureStore.js');
const { duelHoldsBoardStore } = await import('../stores/duelStore.js');
const { translate, resolveStatusMessage } = await import('../i18n');

/** The last status line, as the status bar shows it (English). */
const shownStatus = () => resolveStatusMessage(setStatusBarMessage.mock.lastCall?.[0], translate);

const reading = {
    position: { board: { points: [] } },
    software: 'xg',
    unread: ['score'],
    doubts: [{ field: 'point', point: 7, reason: 'unclear' }]
};

beforeEach(() => {
    vi.clearAllMocks();
    statusBarModeStore.set('NORMAL');
    captureDoubtsStore.set([]);
    duelHoldsBoardStore.set(false);
});

describe('a capture read onto the board', () => {
    test('enters EDIT, fills the board and keeps the doubts for the board to ring', async () => {
        ReadCaptureFile.mockResolvedValue(reading);
        expect(await importCaptureFile('/tmp/a.png')).toBe(true);
        expect(enterEditMode).toHaveBeenCalledOnce();
        expect(applyPositionToBoard).toHaveBeenCalledWith(reading.position);
        expect(get(captureDoubtsStore)).toEqual(reading.doubts);
    });

    test('the EDIT board is not re-entered', async () => {
        statusBarModeStore.set('EDIT');
        await showCaptureOnBoard(reading);
        expect(enterEditMode).not.toHaveBeenCalled();
        expect(applyPositionToBoard).toHaveBeenCalledOnce();
    });

    test('EVAL does not draw the doubts: the reading moves to EDIT', async () => {
        statusBarModeStore.set('EVAL');
        await showCaptureOnBoard(reading);
        expect(enterEditMode).toHaveBeenCalledOnce();
    });

    test('a doubt the board cannot ring is named, not counted as ringed', async () => {
        statusBarModeStore.set('EDIT');
        await showCaptureOnBoard({ ...reading, unread: ['score', 'turn'], doubts: [{ field: 'orientation', point: 0, reason: 'x' }] });
        expect(shownStatus()).toBe("Capture read onto the board (xg). To check, not ringed: the board's direction. Check it, set the score, the player on roll, then save.");
    });

    test('the ringed doubts are counted apart from the others', async () => {
        statusBarModeStore.set('EDIT');
        await showCaptureOnBoard({ ...reading, doubts: [...reading.doubts, { field: 'dice', point: 0, reason: 'x' }] });
        expect(shownStatus()).toBe('Capture read onto the board (xg); 1 doubtful spot(s) ringed. To check, not ringed: the dice. Check it, set the score, then save.');
    });

    test('images dropped beyond the first are named after the reading, not over it', async () => {
        ReadCaptureFile.mockResolvedValue(reading);
        await importCaptureFile('/tmp/a.png', 2);
        expect(setStatusBarMessage).toHaveBeenCalledOnce();
        expect(shownStatus()).toBe('Capture read onto the board (xg); 1 doubtful spot(s) ringed. Check it, set the score, then save. 3 images dropped: only the first was read.');
    });

    test('nothing is saved: the reading only reaches the board', async () => {
        ReadCaptureData.mockResolvedValue(reading);
        await importCaptureBlob(new Blob([new Uint8Array([1, 2, 3])], { type: 'image/png' }));
        expect(ReadCaptureData).toHaveBeenCalledWith('AQID');
        expect(SaveIndividualPosition).not.toHaveBeenCalled();
    });

    test('a refused image leaves the board alone and says why', async () => {
        ReadCaptureFile.mockRejectedValue('no backgammon board recognised');
        expect(await importCaptureFile('/tmp/a.png')).toBe(false);
        expect(applyPositionToBoard).not.toHaveBeenCalled();
        expect(enterEditMode).not.toHaveBeenCalled();
        expect(setStatusBarMessage).toHaveBeenCalledOnce();
    });

    test('a Duel keeps its board', async () => {
        duelHoldsBoardStore.set(true);
        expect(await importCaptureFile('/tmp/a.png')).toBe(false);
        expect(ReadCaptureFile).not.toHaveBeenCalled();
    });
});

describe('what counts as a capture', () => {
    test('image extensions, any case', () => {
        expect(['a.PNG', 'b.jpg', 'c.jpeg', 'd.gif'].every(isCaptureFile)).toBe(true);
        expect(['a.xg', 'b.txt', 'c'].some(isCaptureFile)).toBe(false);
    });

    test('a paste carries an image only when a file item says so', () => {
        const file = new File(['x'], 'p.png', { type: 'image/png' });
        const image = {
            clipboardData: {
                items: [
                    { kind: 'string', type: 'text/plain' },
                    { kind: 'file', type: 'image/png', getAsFile: () => file }
                ]
            }
        };
        expect(imageOfPaste(/** @type {any} */ (image))).toBe(file);
        expect(imageOfPaste(/** @type {any} */ ({ clipboardData: { items: [{ kind: 'string', type: 'text/plain' }] } }))).toBeNull();
    });
});
