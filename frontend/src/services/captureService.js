/**
 * captureService.js — a screen capture of a board, read onto the scratch board.
 *
 * The reading is a proposal: it lands on the EDIT board with what it doubted ringed there, and
 * nothing is stored until the user saves the corrected position by the usual gesture.
 */
import { get } from 'svelte/store';
import { ReadCaptureData, ReadCaptureFile } from '../../wailsjs/go/database/Database.js';
import { statusBarModeStore } from '../stores/uiStore.js';
import { duelHoldsBoardStore } from '../stores/duelStore.js';
import { captureDoubtsStore } from '../stores/captureStore.js';
import { tMsg } from '../i18n';
import { logger } from '../utils/logger.js';
import { setStatusBarMessage } from './databaseService.js';
import { applyPositionToBoard } from './importService.js';
import { enterEditMode } from './modeMachine.js';

export const CAPTURE_EXTENSIONS = ['png', 'jpg', 'jpeg', 'gif'];

/** @param {string} path */
export function isCaptureFile(path) {
    const ext = String(path).toLowerCase().split('.').pop() ?? '';
    return CAPTURE_EXTENSIONS.includes(ext);
}

/**
 * Puts a reading on the EDIT board (entering EDIT unless a scratch board is already up).
 * @param {{ position: any, software?: string, doubts?: import('../stores/captureStore.js').CaptureDoubt[], unread?: string[] }} reading
 */
export async function showCaptureOnBoard(reading) {
    const mode = get(statusBarModeStore);
    if (mode !== 'EDIT' && mode !== 'EVAL') await enterEditMode();
    applyPositionToBoard(reading.position);
    const doubts = reading.doubts ?? [];
    captureDoubtsStore.set(doubts);
    const unread = (reading.unread ?? []).join(', ');
    const key = doubts.length > 0 ? 'status.captureOnBoardDoubts' : 'status.captureOnBoard';
    setStatusBarMessage(tMsg(key, { software: reading.software ?? '', count: doubts.length, unread }));
}

/** @param {() => Promise<any>} read */
async function readOntoBoard(read) {
    if (get(duelHoldsBoardStore)) return false;
    try {
        await showCaptureOnBoard(await read());
        return true;
    } catch (error) {
        logger.error('could not read the capture:', error);
        setStatusBarMessage(tMsg('status.captureFailed', { error }));
        return false;
    }
}

/** @param {string} path an image file on disk */
export function importCaptureFile(path) {
    return readOntoBoard(() => ReadCaptureFile(path));
}

/** @param {Blob} blob an image from the clipboard */
export function importCaptureBlob(blob) {
    return readOntoBoard(async () => ReadCaptureData(await blobToBase64(blob)));
}

/** @param {Blob} blob */
async function blobToBase64(blob) {
    const bytes = new Uint8Array(await blob.arrayBuffer());
    let binary = '';
    for (let i = 0; i < bytes.length; i += 0x8000) binary += String.fromCharCode(...bytes.subarray(i, i + 0x8000));
    return btoa(binary);
}

/**
 * The image a paste event carries, if any; null for a text paste.
 * @param {ClipboardEvent} event
 * @returns {File | null}
 */
export function imageOfPaste(event) {
    for (const item of event.clipboardData?.items ?? []) {
        if (item.kind === 'file' && item.type.startsWith('image/')) return item.getAsFile();
    }
    return null;
}

/**
 * Window-level paste handler: an image on the clipboard is read as a capture, in any mode.
 * Text is left to the Ctrl+V keyboard path.
 * @param {ClipboardEvent} event
 */
export function handlePasteEvent(event) {
    const target = /** @type {HTMLElement | null} */ (event.target);
    if (target && (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable)) return;
    const image = imageOfPaste(event);
    if (!image) return;
    event.preventDefault();
    importCaptureBlob(image);
}
